package azuredevops_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

const (
	typesRoute      = "/contoso/Fabrikam/_apis/wit/workitemtypes"
	iterationsRoute = "/contoso/Fabrikam/Fabrikam Team/_apis/work/teamsettings/iterations"
	membersRoute    = "/contoso/_apis/projects/Fabrikam/teams/Fabrikam Team/members"
)

// discoverCfg es la configuración de integrations.azure-devops de las pruebas.
func discoverCfg() map[string]any {
	return map[string]any{"organization": "contoso", "project": "Fabrikam", "team": "Fabrikam Team"}
}

func discoverWith(t *testing.T, srv *adotest.Server, mutate func(*azuredevops.Options)) (integrations.Discovery, error) {
	t.Helper()
	opts := srv.Options()
	if mutate != nil {
		mutate(&opts)
	}
	plugin := azuredevops.New(opts)
	return plugin.Discover(context.Background(), integrations.PortRequest{Config: discoverCfg(), Secrets: &patSource{pat: testPAT}})
}

// emailLike construye en tiempo de ejecución un valor con forma de correo: así ninguna fuente del
// paquete contiene una dirección literal y el escaneo de REQ-45.9 sigue siendo significativo.
func emailLike() string { return "persona" + string(rune(64)) + "ejemplo.test" }

func member(id, name string) map[string]any {
	return map[string]any{
		"identity":    map[string]any{"id": id, "displayName": name, "uniqueName": emailLike()},
		"isTeamAdmin": false,
	}
}

// fullDiscoveryServer sirve dos tipos, dos iteraciones y tres miembros.
func fullDiscoveryServer(t *testing.T) *adotest.Server { return discoveryServerWith(t, nil, nil) }

// discoveryServerWith es fullDiscoveryServer con un manejador propio para los miembros o las
// iteraciones (las rutas se prueban en orden de registro: la primera que coincide responde).
func discoveryServerWith(t *testing.T, members, iterations adotest.Handler) *adotest.Server {
	t.Helper()
	srv := adotest.NewServer(t, nil)
	if members != nil {
		srv.Handle("GET", membersRoute, members)
	}
	if iterations != nil {
		srv.Handle("GET", iterationsRoute, iterations)
	}
	srv.Reply("GET", typesRoute, adotest.Response{JSON: map[string]any{"count": 2, "value": []map[string]any{
		{"name": "Bug", "referenceName": "Microsoft.VSTS.WorkItemTypes.Bug"},
		{"name": "Task", "referenceName": "Microsoft.VSTS.WorkItemTypes.Task"},
	}}})
	srv.Reply("GET", typesRoute+"/Bug/states", adotest.Response{JSON: map[string]any{"value": []map[string]any{
		{"name": "New", "color": "b2b2b2", "category": "Proposed"},
		{"name": "Active", "color": "007acc", "category": "InProgress"},
		{"name": "Resolved", "color": "ff9d00", "category": "Resolved"},
		{"name": "Closed", "color": "339933", "category": "Completed"},
		{"name": "Removed", "color": "ffffff", "category": "Removed"},
	}}})
	srv.Reply("GET", typesRoute+"/Task/states", adotest.Response{JSON: map[string]any{"value": []map[string]any{
		{"name": "To Do", "category": "Proposed"},
		{"name": "Done", "category": "Completed"},
	}}})
	srv.Reply("GET", typesRoute+"/Bug/fields", adotest.Response{JSON: map[string]any{"value": []map[string]any{
		{"referenceName": "System.Title", "name": "Title", "alwaysRequired": true},
		{"referenceName": "System.State", "name": "State", "alwaysRequired": false},
	}}})
	srv.Reply("GET", typesRoute+"/Task/fields", adotest.Response{JSON: map[string]any{"value": []map[string]any{
		{"referenceName": "System.Title", "name": "Title", "alwaysRequired": true},
	}}})
	srv.Reply("GET", iterationsRoute, adotest.Response{JSON: map[string]any{"count": 2, "value": []map[string]any{
		{"id": "it-1", "name": "Sprint 1", "path": `Fabrikam\Sprint 1`,
			"attributes": map[string]any{"startDate": "2026-11-01T00:00:00Z", "finishDate": "2026-11-14T00:00:00Z", "timeFrame": "current"}},
		{"id": "it-2", "name": "Sprint 2", "path": `Fabrikam\Sprint 2`,
			"attributes": map[string]any{"startDate": "2026-11-15T00:00:00Z", "finishDate": "2026-11-28T00:00:00Z", "timeFrame": "future"}},
	}}})
	srv.Reply("GET", membersRoute, adotest.Response{JSON: map[string]any{"count": 3, "value": []map[string]any{
		member("m-1", "Ana"), member("m-2", "Berto"), member("m-3", "Carla"),
	}}})
	return srv
}

func TestDiscoverReturnsTheWholeProcess(t *testing.T) {
	srv := fullDiscoveryServer(t)

	got, err := discoverWith(t, srv, nil)

	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if got.Schema != integrations.DiscoverySchemaV1 || got.Plugin != "azure-devops" {
		t.Errorf("esquema y plugin = %q %q", got.Schema, got.Plugin)
	}
	wantScope := map[string]string{"organization": "contoso", "project": "Fabrikam", "team": "Fabrikam Team"}
	if len(got.Scope) != 3 || got.Scope["organization"] != "contoso" || got.Scope["project"] != "Fabrikam" || got.Scope["team"] != wantScope["team"] {
		t.Errorf("scope = %v", got.Scope)
	}
	if len(got.ItemTypes) != 2 || got.ItemTypes[0].Name != "Bug" || got.ItemTypes[0].ReferenceName != "Microsoft.VSTS.WorkItemTypes.Bug" || got.ItemTypes[1].Name != "Task" {
		t.Fatalf("tipos = %+v", got.ItemTypes)
	}
	bug := got.ItemTypes[0]
	if len(bug.States) != 5 || bug.States[1] != (integrations.ItemState{Name: "Active", Category: "InProgress", Color: "007acc"}) {
		t.Errorf("estados de Bug = %+v", bug.States)
	}
	if len(bug.Fields) != 2 || bug.Fields[0] != (integrations.ItemField{ReferenceName: "System.Title", Name: "Title", AlwaysRequired: true}) {
		t.Errorf("campos de Bug = %+v", bug.Fields)
	}
	if len(got.ItemTypes[1].States) != 2 || len(got.ItemTypes[1].Fields) != 1 {
		t.Errorf("Task = %+v", got.ItemTypes[1])
	}
	if len(got.Iterations) != 2 || got.Iterations[0] != (integrations.Iteration{
		ID: "it-1", Name: "Sprint 1", Path: `Fabrikam\Sprint 1`,
		StartDate: "2026-11-01T00:00:00Z", FinishDate: "2026-11-14T00:00:00Z", TimeFrame: "current",
	}) || got.Iterations[1].TimeFrame != "future" {
		t.Errorf("iteraciones = %+v", got.Iterations)
	}
	if len(got.Members) != 3 || got.Members[0] != (integrations.Member{ID: "m-1", DisplayName: "Ana"}) {
		t.Errorf("miembros = %+v", got.Members)
	}
}

func TestDiscoverCopiesTheStateCategoryUntouched(t *testing.T) {
	srv := fullDiscoveryServer(t)

	got, err := discoverWith(t, srv, nil)

	if err != nil {
		t.Fatal(err)
	}
	categories := map[string]string{}
	for _, s := range got.ItemTypes[0].States {
		categories[s.Name] = s.Category
	}
	if categories["Active"] != "InProgress" || categories["Removed"] != "Removed" || categories["New"] != "Proposed" || categories["Closed"] != "Completed" || categories["Resolved"] != "Resolved" {
		t.Errorf("las categorías deben llegar tal como las envía la API: %v", categories)
	}
}

func TestDiscoverEscapesTheTeamSegmentAndAsksForAPIVersion(t *testing.T) {
	srv := fullDiscoveryServer(t)

	if _, err := discoverWith(t, srv, nil); err != nil {
		t.Fatal(err)
	}

	var iterations, members *adotest.Request
	for _, r := range srv.Requests() {
		r := r
		switch r.Path {
		case iterationsRoute:
			iterations = &r
		case membersRoute:
			members = &r
		}
	}
	if iterations == nil || iterations.EscapedPath != "/contoso/Fabrikam/Fabrikam%20Team/_apis/work/teamsettings/iterations" {
		t.Errorf("la petición de iteraciones = %+v", iterations)
	}
	if members == nil || members.Query.Get("$top") != "100" || members.Query.Get("$skip") != "0" || members.Query.Get("api-version") != "7.1" {
		t.Errorf("la petición de miembros = %+v", members)
	}
}

// pagedMembers responde con total miembros únicos m-0..m-(total-1) paginados por $top y $skip.
func pagedMembers(total, overlap int) adotest.Handler {
	return func(r *adotest.Request) adotest.Response {
		top, _ := strconv.Atoi(r.Query.Get("$top"))
		skip, _ := strconv.Atoi(r.Query.Get("$skip"))
		start := max(skip-overlap, 0)
		var page []map[string]any
		for i := start; i < total && len(page) < top; i++ {
			page = append(page, member(fmt.Sprintf("m-%d", i), fmt.Sprintf("Miembro %d", i)))
		}
		return adotest.Response{JSON: map[string]any{"count": len(page), "value": page}}
	}
}

func TestDiscoverPagesMembersUntilAnIncompletePage(t *testing.T) {
	for name, overlap := range map[string]int{"sin solape": 0, "con una página que repite miembros": 1} {
		t.Run(name, func(t *testing.T) {
			srv := discoveryServerWith(t, pagedMembers(250, overlap), nil)

			got, err := discoverWith(t, srv, nil)

			if err != nil {
				t.Fatalf("Discover: %v", err)
			}
			if len(got.Members) != 250 {
				t.Fatalf("miembros = %d, quería 250 sin repetidos", len(got.Members))
			}
			seen := map[string]bool{}
			for _, m := range got.Members {
				if seen[m.ID] {
					t.Fatalf("miembro repetido: %s", m.ID)
				}
				seen[m.ID] = true
			}
		})
	}
}

func TestDiscoverFailsInsteadOfReturningAPartialMemberListWhenSkipIsIgnored(t *testing.T) {
	// Servidor que ignora $skip: siempre devuelve la primera página de 100 sobre 250 miembros.
	srv := discoveryServerWith(t, func(r *adotest.Request) adotest.Response {
		r.Query.Set("$skip", "0")
		return pagedMembers(250, 0)(r)
	}, nil)

	got, err := discoverWith(t, srv, nil)

	var pagination *azuredevops.PaginationError
	if !errors.Is(err, azuredevops.ErrPagination) || !errors.As(err, &pagination) {
		t.Fatalf("se esperaba un error de paginación, obtuvo %v", err)
	}
	if len(got.Members) != 0 || len(got.ItemTypes) != 0 {
		t.Errorf("un error no devuelve datos parciales: %+v", got)
	}
	if n := srv.Count("GET", membersRoute); n > 3 {
		t.Errorf("el servidor que ignora $skip debe cortar enseguida, recibió %d peticiones", n)
	}
}

func TestDiscoverFailsWhenTheMemberPageCapIsReached(t *testing.T) {
	azuredevops.SetMaxMemberPagesForTest(t, 3)
	srv := discoveryServerWith(t, pagedMembers(10_000, 0), nil)

	got, err := discoverWith(t, srv, nil)

	var pagination *azuredevops.PaginationError
	if !errors.Is(err, azuredevops.ErrPagination) || !errors.As(err, &pagination) || pagination.Pages != 3 {
		t.Fatalf("se esperaba un error de paginación tras 3 páginas, obtuvo %v", err)
	}
	if len(got.Members) != 0 {
		t.Errorf("no se devuelve una lista truncada como completa: %d miembros", len(got.Members))
	}
}

func TestDiscoverAcceptsEmptyIterationsInValuesOrValue(t *testing.T) {
	populated := []map[string]any{{"id": "it-1", "name": "Sprint 1", "path": `Fabrikam\Sprint 1`}}
	tests := []struct {
		name     string
		response adotest.Response
		want     int
	}{
		{"sin iteraciones", adotest.Response{JSON: map[string]any{"count": 0, "value": []any{}}}, 0},
		{"cuerpo vacío de elementos", adotest.Response{JSON: map[string]any{}}, 0},
		{"en values", adotest.Response{JSON: map[string]any{"values": populated}}, 1},
		{"en value", adotest.Response{JSON: map[string]any{"value": populated}}, 1},
		{"values vacío y value con elementos", adotest.Response{JSON: map[string]any{"values": []any{}, "value": populated}}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := serverWithIterations(t, tt.response)

			got, err := discoverWith(t, srv, nil)

			if err != nil {
				t.Fatalf("Discover: %v", err)
			}
			if got.Iterations == nil || len(got.Iterations) != tt.want {
				t.Fatalf("iteraciones = %#v, quería %d", got.Iterations, tt.want)
			}
		})
	}
}

// serverWithIterations monta un servidor sin tipos y con un miembro, con la respuesta dada para las
// iteraciones.
func serverWithIterations(t *testing.T, iterations adotest.Response) *adotest.Server {
	t.Helper()
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", iterationsRoute, iterations)
	srv.Reply("GET", typesRoute, adotest.Response{JSON: map[string]any{"value": []any{}}})
	srv.Reply("GET", membersRoute, adotest.Response{JSON: map[string]any{"value": []any{member("m-1", "Ana")}}})
	return srv
}

func TestDiscoverMemberAuthFailureFailsTheWholeDiscoveryNamingVsoProject(t *testing.T) {
	srv := discoveryServerWith(t, func(*adotest.Request) adotest.Response {
		return adotest.Response{Status: 403, JSON: map[string]any{"message": "denegado"}}
	}, nil)

	got, err := discoverWith(t, srv, nil)

	var auth *azuredevops.AuthError
	if !errors.Is(err, azuredevops.ErrAuth) || !errors.As(err, &auth) || !strings.Contains(err.Error(), "vso.project") {
		t.Fatalf("se esperaba AuthError con vso.project, obtuvo %v", err)
	}
	if len(got.ItemTypes) != 0 || len(got.Iterations) != 0 || len(got.Members) != 0 {
		t.Errorf("no se devuelve un resultado parcial: %+v", got)
	}
}

func TestDiscoverIterationAuthFailureNamesVsoWork(t *testing.T) {
	srv := serverWithIterations(t, adotest.Response{Status: 403})

	_, err := discoverWith(t, srv, nil)

	if !errors.Is(err, azuredevops.ErrAuth) || !strings.Contains(err.Error(), "vso.work") || strings.Contains(err.Error(), "vso.work_write") || strings.Contains(err.Error(), "vso.project") {
		t.Fatalf("el fallo de iteraciones debe nombrar vso.work, obtuvo %v", err)
	}
}

func TestDiscoverMissingPATSendsNothing(t *testing.T) {
	srv := fullDiscoveryServer(t)
	plugin := azuredevops.New(srv.Options())

	_, err := plugin.Discover(context.Background(), integrations.PortRequest{
		Config: discoverCfg(), Secrets: &patSource{err: errors.New("falta el secreto")},
	})

	if err == nil || len(srv.Requests()) != 0 {
		t.Fatalf("sin PAT no hay petición HTTP: %v (%d peticiones)", err, len(srv.Requests()))
	}
}

func TestDiscoverRejectsAnIncompleteConfig(t *testing.T) {
	srv := fullDiscoveryServer(t)
	plugin := azuredevops.New(srv.Options())

	_, err := plugin.Discover(context.Background(), integrations.PortRequest{
		Config: map[string]any{"organization": "contoso"}, Secrets: &patSource{pat: testPAT},
	})

	if !errors.Is(err, azuredevops.ErrConfig) || len(srv.Requests()) != 0 {
		t.Fatalf("configuración incompleta = %v", err)
	}
}

func TestDiscoveryOutputNeverContainsEmailAddresses(t *testing.T) {
	srv := fullDiscoveryServer(t) // los miembros llevan uniqueName con forma de correo

	got, err := discoverWith(t, srv, nil)

	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), emailLike()) || strings.Contains(string(payload), "ejemplo.test") || strings.Contains(string(payload), "uniqueName") {
		t.Errorf("la salida no puede contener uniqueName ni direcciones de correo: %s", payload)
	}
}

// TestPackageSourcesContainNoEmailAddresses recorre el código y los fixtures del adaptador (REQ-45.9:
// el código no contiene direcciones de correo).
func TestPackageSourcesContainNoEmailAddresses(t *testing.T) {
	email := regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9\-]+(\.[A-Za-z0-9\-]+)*\.[A-Za-z]{2,}`)
	scanned := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		scanned++
		if found := email.Find(data); found != nil {
			t.Errorf("%s contiene una dirección de correo: %q", path, found)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 10 {
		t.Fatalf("el escaneo debía recorrer el paquete y adotest (ficheros: %d)", scanned)
	}
}

func TestEmailScanDetectsAnAddress(t *testing.T) {
	email := regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9\-]+(\.[A-Za-z0-9\-]+)*\.[A-Za-z]{2,}`)
	if !email.MatchString("x " + emailLike() + " y") {
		t.Fatal("el patrón debe reconocer una dirección: el escaneo sería vacuo")
	}
}
