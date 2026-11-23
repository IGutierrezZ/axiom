package azuredevops_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

const connectionDataRoute = "/contoso/_apis/connectionData"

func connectionDataOK(id, custom, provider string) adotest.Response {
	return adotest.Response{JSON: map[string]any{"authenticatedUser": map[string]any{
		"id": id, "customDisplayName": custom, "providerDisplayName": provider,
	}}}
}

// identityServer sirve connectionData con la respuesta dada y los miembros abc-123 y def-456.
func identityServer(t *testing.T, connection adotest.Response) *adotest.Server {
	t.Helper()
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", connectionDataRoute, connection)
	srv.Reply("GET", membersRoute, adotest.Response{JSON: map[string]any{"value": []map[string]any{
		member("abc-123", "Ana"), member("def-456", "Berto"),
	}}})
	return srv
}

func resolve(ctx context.Context, srv *adotest.Server, user string, src integrations.SecretSource) (integrations.TrackerIdentity, error) {
	plugin := azuredevops.New(srv.Options())
	return plugin.ResolveIdentity(ctx, integrations.IdentityRequest{
		PortRequest: integrations.PortRequest{Config: discoverCfg(), Secrets: src},
		User:        user,
	})
}

func assertSelection(t *testing.T, err error, wantUser string) *integrations.IdentitySelectionError {
	t.Helper()
	var selection *integrations.IdentitySelectionError
	if !errors.Is(err, integrations.ErrIdentitySelectionRequired) || !errors.As(err, &selection) {
		t.Fatalf("se esperaba IdentitySelectionError, obtuvo %v", err)
	}
	if selection.User != wantUser || selection.Account != "contoso" || selection.Plugin != "azure-devops" {
		t.Errorf("selección = %+v", selection)
	}
	if len(selection.Candidates) != 2 || selection.Candidates[0] != (integrations.Member{ID: "abc-123", DisplayName: "Ana"}) {
		t.Errorf("candidatos = %+v", selection.Candidates)
	}
	return selection
}

func TestResolveIdentityFromConnectionDataDoesNotQueryMembers(t *testing.T) {
	srv := identityServer(t, connectionDataOK("guid-1", "Ana Pérez", "ana"))

	got, err := resolve(context.Background(), srv, "", &patSource{pat: testPAT})

	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	want := integrations.TrackerIdentity{Account: "contoso", ID: "guid-1", DisplayName: "Ana Pérez", Source: "connection_data"}
	if got != want {
		t.Fatalf("identidad = %+v, quería %+v", got, want)
	}
	if n := srv.Count("GET", membersRoute); n != 0 {
		t.Errorf("con connectionData utilizable no se consultan los miembros (%d consultas)", n)
	}
}

func TestResolveIdentityFallsBackToTheProviderDisplayName(t *testing.T) {
	srv := identityServer(t, connectionDataOK("guid-1", "", "ana"))

	got, err := resolve(context.Background(), srv, "", &patSource{pat: testPAT})

	if err != nil || got.DisplayName != "ana" {
		t.Fatalf("identidad = %+v, %v", got, err)
	}
}

func TestResolveIdentityListsCandidatesWhenConnectionDataIsUnusable(t *testing.T) {
	tests := map[string]adotest.Response{
		"401":                        {Status: 401},
		"authenticatedUser vacío":    {JSON: map[string]any{"authenticatedUser": map[string]any{}}},
		"sin authenticatedUser":      {JSON: map[string]any{}},
		"usuario sin nombre visible": {JSON: map[string]any{"authenticatedUser": map[string]any{"id": "guid-1"}}},
	}
	for name, response := range tests {
		t.Run(name, func(t *testing.T) {
			srv := identityServer(t, response)

			got, err := resolve(context.Background(), srv, "", &patSource{pat: testPAT})

			selection := assertSelection(t, err, "")
			if selection.Reason == "" {
				t.Error("el motivo de la caída debe informarse")
			}
			if got != (integrations.TrackerIdentity{}) {
				t.Errorf("no se devuelve ninguna identidad: %+v", got)
			}
		})
	}
}

func TestResolveIdentityWithUserSkipsConnectionData(t *testing.T) {
	srv := identityServer(t, connectionDataOK("guid-1", "Ana", "ana"))

	got, err := resolve(context.Background(), srv, "ABC-123", &patSource{pat: testPAT}) // sin distinguir mayúsculas

	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	want := integrations.TrackerIdentity{Account: "contoso", ID: "abc-123", DisplayName: "Ana", Source: "member_selection"}
	if got != want {
		t.Fatalf("identidad = %+v, quería %+v", got, want)
	}
	if n := srv.Count("GET", connectionDataRoute); n != 0 {
		t.Errorf("con --user no se consulta connectionData (%d consultas)", n)
	}
}

func TestResolveIdentityWithUnknownUserNamesTheUserAndCandidates(t *testing.T) {
	srv := identityServer(t, connectionDataOK("guid-1", "Ana", "ana"))

	_, err := resolve(context.Background(), srv, "zzz", &patSource{pat: testPAT})

	assertSelection(t, err, "zzz")
	if !strings.Contains(err.Error(), "zzz") {
		t.Errorf("el error debe nombrar el usuario: %q", err)
	}
	if n := srv.Count("GET", connectionDataRoute); n != 0 {
		t.Errorf("con --user no se consulta connectionData (%d consultas)", n)
	}
}

func TestResolveIdentityReportsTheMemberErrorWhenBothFail(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", connectionDataRoute, adotest.Response{Status: 401})
	srv.Reply("GET", membersRoute, adotest.Response{Status: 403})

	_, err := resolve(context.Background(), srv, "", &patSource{pat: testPAT})

	var auth *azuredevops.AuthError
	if !errors.Is(err, azuredevops.ErrAuth) || !errors.As(err, &auth) || auth.Scope != "vso.project" {
		t.Fatalf("se esperaba el AuthError de los miembros con vso.project, obtuvo %v", err)
	}
	if !strings.Contains(err.Error(), "connectionData") {
		t.Errorf("el error debe anotar que connectionData tampoco sirvió: %q", err)
	}
}

func TestResolveIdentityPropagatesContextCancellation(t *testing.T) {
	srv := identityServer(t, connectionDataOK("guid-1", "Ana", "ana"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := resolve(ctx, srv, "", &patSource{pat: testPAT})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("la cancelación se propaga, obtuvo %v", err)
	}
	if errors.Is(err, integrations.ErrIdentitySelectionRequired) {
		t.Error("una cancelación no es una petición de --user")
	}
	if len(srv.Requests()) != 0 {
		t.Errorf("con el contexto cancelado no se envía nada (%d peticiones)", len(srv.Requests()))
	}
}

func TestResolveIdentityMissingPATIsReturnedWithoutFallingBackToMembers(t *testing.T) {
	srv := identityServer(t, connectionDataOK("guid-1", "Ana", "ana"))
	missing := &secrets.MissingError{Ref: secrets.Ref{Plugin: "azure-devops", Name: "pat"}}

	_, err := resolve(context.Background(), srv, "", &patSource{err: missing})

	if !errors.Is(err, secrets.ErrNotFound) {
		t.Fatalf("PAT ausente = %v", err)
	}
	if len(srv.Requests()) != 0 {
		t.Errorf("sin PAT no hay peticiones (%d)", len(srv.Requests()))
	}
}

func TestResolveIdentityRejectsAnIncompleteConfig(t *testing.T) {
	srv := identityServer(t, connectionDataOK("guid-1", "Ana", "ana"))
	plugin := azuredevops.New(srv.Options())

	_, err := plugin.ResolveIdentity(context.Background(), integrations.IdentityRequest{
		PortRequest: integrations.PortRequest{Config: map[string]any{"organization": "contoso"}, Secrets: &patSource{pat: testPAT}},
	})

	if !errors.Is(err, azuredevops.ErrConfig) || len(srv.Requests()) != 0 {
		t.Fatalf("configuración incompleta = %v", err)
	}
}
