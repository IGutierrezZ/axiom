package azuredevops

import (
	"context"
	"net/url"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

// Motivos cerrados de ForeignURLError.
const (
	foreignScheme       = "scheme"
	foreignUserinfo     = "userinfo"
	foreignHost         = "host"
	foreignOrganization = "organization"
	foreignPath         = "path"
)

// validateResourceURL comprueba, sin red y antes de resolver ningún secreto, que raw puede recibir el
// PAT: el esquema es el del endpoint (https en producción), sin userinfo, con el host del endpoint (sin
// distinguir mayúsculas), con una ruta bajo /{organización}/ (sin distinguir mayúsculas) y sin
// segmentos «.» o «..». Devuelve la URL analizada.
func (c *Client) validateResourceURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, &ForeignURLError{Reason: foreignPath}
	}
	switch {
	case !strings.EqualFold(u.Scheme, c.endpoint.Scheme):
		return nil, &ForeignURLError{Reason: foreignScheme}
	case u.User != nil:
		return nil, &ForeignURLError{Reason: foreignUserinfo}
	case u.Host == "" || !strings.EqualFold(u.Host, c.endpoint.Host):
		return nil, &ForeignURLError{Reason: foreignHost}
	}
	path := u.Path
	if prefix := strings.TrimRight(c.endpoint.Path, "/"); prefix != "" {
		trimmed, ok := strings.CutPrefix(path, prefix)
		if !ok {
			return nil, &ForeignURLError{Reason: foreignPath}
		}
		path = trimmed
	}
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(segments) < 2 || !strings.EqualFold(segments[0], c.target.Organization) {
		return nil, &ForeignURLError{Reason: foreignOrganization}
	}
	for _, s := range segments {
		if s == "." || s == ".." {
			return nil, &ForeignURLError{Reason: foreignPath}
		}
	}
	return u, nil
}

// Download descarga un recurso binario (un adjunto o una imagen incrustada) con un máximo de limit
// bytes. Antes de resolver el PAT valida la URL (validateResourceURL); si no cumple, *ForeignURLError
// sin red. Reutiliza el punto de salida do: autenticación, ritmo, reintentos de 429 y 503, redirecciones
// sin seguir y saneado del PAT. Un cuerpo mayor que limit es *TooLargeError.
func (c *Client) Download(ctx context.Context, rawURL string, limit int64) (integrations.Attachment, error) {
	if limit <= 0 {
		limit = maxResponseBytes
	}
	u, err := c.validateResourceURL(rawURL)
	if err != nil {
		return integrations.Attachment{}, err
	}
	var out binaryResult
	if err := c.do(ctx, request{
		method:      "GET",
		absoluteURL: u.String(),
		scope:       scopeWorkWrite,
		operation:   "descargar un adjunto",
		retryable:   true,
		bodyLimit:   limit,
		binary:      true,
	}, &out); err != nil {
		return integrations.Attachment{}, err
	}
	return integrations.Attachment{Data: out.Data, ContentType: out.ContentType}, nil
}
