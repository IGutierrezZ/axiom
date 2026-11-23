package azuredevops

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

// Orígenes de la identidad (integrations.TrackerIdentity.Source).
const (
	sourceConnectionData  = "connection_data"
	sourceMemberSelection = "member_selection"
)

var _ integrations.IdentityResolver = (*Plugin)(nil)

// connectionDataResponse es la parte que se usa de GET {org}/_apis/connectionData. El contrato con un
// PAT real no está verificado: solo lo cubren pruebas con httptest.
type connectionDataResponse struct {
	AuthenticatedUser struct {
		ID                  string `json:"id"`
		CustomDisplayName   string `json:"customDisplayName"`
		ProviderDisplayName string `json:"providerDisplayName"`
	} `json:"authenticatedUser"`
}

// ResolveIdentity implementa el puerto de identidad (integrations.IdentityResolver).
//
//   - Con User: busca el id (sin distinguir mayúsculas, son GUID) entre los miembros del equipo. No
//     consulta connectionData. Si no coincide, devuelve un *IdentitySelectionError con los candidatos.
//   - Sin User: consulta connectionData. Si trae un usuario utilizable (id y nombre visible), esa es
//     la identidad. Si falla o no es utilizable, lista los miembros y devuelve un
//     *IdentitySelectionError que pide repetir con --user. Nunca hay pregunta interactiva.
//
// La cancelación del contexto se propaga y nunca se toma por una petición de --user.
func (p *Plugin) ResolveIdentity(ctx context.Context, req integrations.IdentityRequest) (integrations.TrackerIdentity, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.TrackerIdentity{}, err
	}
	account := client.target.Organization

	if req.User != "" {
		members, err := client.listMembers(ctx)
		if err != nil {
			return integrations.TrackerIdentity{}, err
		}
		for _, m := range members {
			if strings.EqualFold(m.ID, strings.TrimSpace(req.User)) {
				return integrations.TrackerIdentity{Account: account, ID: m.ID, DisplayName: m.DisplayName, Source: sourceMemberSelection}, nil
			}
		}
		return integrations.TrackerIdentity{}, &integrations.IdentitySelectionError{
			Plugin: manifest.AzureDevOpsID, Account: account, User: req.User,
			Reason:     "el usuario indicado no coincide con el id de ningún miembro del equipo",
			Candidates: members,
		}
	}

	identity, reason, err := client.connectionData(ctx)
	if err != nil {
		return integrations.TrackerIdentity{}, err
	}
	if reason == "" {
		return identity, nil
	}

	members, err := client.listMembers(ctx)
	if err != nil {
		// El fallo de los miembros es el que la persona puede corregir; se anota que connectionData
		// tampoco sirvió y se conserva el tipo (errors.Is y errors.As).
		return integrations.TrackerIdentity{}, fmt.Errorf("%w (connectionData tampoco sirvió: %s)", err, reason)
	}
	return integrations.TrackerIdentity{}, &integrations.IdentitySelectionError{
		Plugin: manifest.AzureDevOpsID, Account: account, Reason: reason, Candidates: members,
	}
}

// connectionData consulta GET {org}/_apis/connectionData. Devuelve la identidad si es utilizable; si
// no lo es, devuelve el motivo y ningún error. Solo devuelve error ante la cancelación del contexto o
// un fallo que ninguna alternativa arregla (PAT ausente).
func (c *Client) connectionData(ctx context.Context) (integrations.TrackerIdentity, string, error) {
	var resp connectionDataResponse
	err := c.do(ctx, request{
		method:    "GET",
		segments:  []string{"_apis", "connectionData"},
		operation: "leer connectionData",
		retryable: true,
	}, &resp)
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return integrations.TrackerIdentity{}, "", err
		}
		if errors.Is(err, secrets.ErrNotFound) {
			return integrations.TrackerIdentity{}, "", err
		}
		var auth *AuthError
		if errors.As(err, &auth) {
			return integrations.TrackerIdentity{}, fmt.Sprintf("connectionData rechazó la autenticación (HTTP %d)", auth.Status), nil
		}
		return integrations.TrackerIdentity{}, "connectionData falló: " + err.Error(), nil
	}

	user := resp.AuthenticatedUser
	name := user.CustomDisplayName
	if name == "" {
		name = user.ProviderDisplayName
	}
	if user.ID == "" || name == "" {
		return integrations.TrackerIdentity{}, "connectionData no devolvió un usuario autenticado con identificador y nombre visible", nil
	}
	return integrations.TrackerIdentity{Account: c.target.Organization, ID: user.ID, DisplayName: name, Source: sourceConnectionData}, "", nil
}
