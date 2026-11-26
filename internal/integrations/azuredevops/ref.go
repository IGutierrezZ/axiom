package azuredevops

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ErrItemRef indica que --from no es un identificador de work item ni una URL aceptada. El error
// nunca repite el valor recibido.
var ErrItemRef = errors.New("la referencia no es un identificador numérico ni la URL de un work item del host, la organización y el proyecto configurados")

var itemIDPattern = regexp.MustCompile(`^[1-9][0-9]{0,9}$`)

// ParseItemRef interpreta --from sin red (D-29). Acepta un identificador (int32 positivo sin ceros a
// la izquierda) o una URL del host y la organización configurados con las formas
// /{org}[/{project}]/_workitems/edit/{id}, /{org}[/{project}]/_apis/wit/workitems/{id} o cualquier
// ruta bajo /{org}/ con un único parámetro workitem={id}. El fragmento y la barra final se ignoran.
// Si la ruta nombra un proyecto, debe ser el configurado.
func (p *Plugin) ParseItemRef(cfg map[string]any, raw string) (string, error) {
	target, err := TargetFromConfig(cfg)
	if err != nil {
		return "", err
	}
	endpoint := p.opts.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	base, err := url.Parse(endpoint)
	if err != nil || base.Host == "" {
		return "", fmt.Errorf("%w: la URL base no es válida", ErrConfig)
	}
	return parseItemRef(base, target, raw)
}

func parseItemRef(base *url.URL, target Target, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if validItemID(raw) {
		return raw, nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", ErrItemRef
	}
	switch {
	case !strings.EqualFold(u.Scheme, base.Scheme), u.User != nil, !strings.EqualFold(u.Host, base.Host):
		return "", ErrItemRef
	}

	// Segmentos de la ruta ya decodificados; la barra final no cuenta. El prefijo de ruta del
	// endpoint (vacío en producción) se quita antes de la organización.
	path := strings.Trim(u.Path, "/")
	if prefix := strings.Trim(base.Path, "/"); prefix != "" {
		trimmed, ok := strings.CutPrefix(path, prefix)
		if !ok {
			return "", ErrItemRef
		}
		path = strings.TrimPrefix(trimmed, "/")
	}
	segments := strings.Split(path, "/")
	if len(segments) < 2 || !strings.EqualFold(segments[0], target.Organization) {
		return "", ErrItemRef
	}
	rest := segments[1:]
	if !strings.HasPrefix(rest[0], "_") {
		if !strings.EqualFold(rest[0], target.Project) {
			return "", ErrItemRef
		}
		rest = rest[1:]
	}

	id, ok := idFromPath(rest)
	if !ok {
		if workitem := u.Query()["workitem"]; len(workitem) == 1 {
			id, ok = workitem[0], true
		}
	}
	if !ok || !validItemID(id) {
		return "", ErrItemRef
	}
	return id, nil
}

// idFromPath devuelve el identificador de las formas _workitems/edit/{id} y _apis/wit/workitems/{id}.
func idFromPath(rest []string) (string, bool) {
	switch {
	case len(rest) == 3 && rest[0] == "_workitems" && rest[1] == "edit":
		return rest[2], true
	case len(rest) == 4 && rest[0] == "_apis" && rest[1] == "wit" && rest[2] == "workitems":
		return rest[3], true
	}
	return "", false
}

func validItemID(s string) bool {
	if !itemIDPattern.MatchString(s) {
		return false
	}
	n, err := strconv.ParseInt(s, 10, 64)
	return err == nil && n <= math.MaxInt32
}
