package azuredevops_test

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

var portConfig = map[string]any{"organization": "contoso", "project": "Fabrikam", "team": "Fabrikam Team"}

// newPort devuelve el plugin apuntado al servidor y la petición base del puerto con el PAT ficticio.
func newPort(srv *adotest.Server) (*azuredevops.Plugin, integrations.PortRequest, *patSource) {
	src := &patSource{pat: testPAT}
	return azuredevops.New(srv.Options()), integrations.PortRequest{Config: portConfig, Secrets: src}, src
}

func relation(srv *adotest.Server, rel string, id int) map[string]any {
	return map[string]any{"rel": rel, "url": srv.URL() + "/contoso/_apis/wit/workItems/" + strconv.Itoa(id)}
}

func seedStory(srv *adotest.Server) {
	srv.SeedWorkItem(42, 5, map[string]any{
		"System.WorkItemType": "User Story",
		"System.Title":        "Alta de usuarios",
		"System.State":        "Active",
		"System.Tags":         "uno; Dos ;; tres",
		"System.Description":  "<p>Hola</p>",
		"Microsoft.VSTS.Common.AcceptanceCriteria": "<ul><li>a</li></ul>",
		"Microsoft.VSTS.TCM.ReproSteps":            "<p>pasos</p>",
	},
		relation(srv, "System.LinkTypes.Hierarchy-Reverse", 10),
		relation(srv, "System.LinkTypes.Related", 11),
		map[string]any{"rel": "AttachedFile", "url": srv.URL() + "/contoso/Fabrikam/_apis/wit/attachments/g1",
			"attributes": map[string]any{"name": "pantalla.png", "resourceSize": 2048}},
		map[string]any{"rel": "Hyperlink", "url": "https://example.test/x"},
		map[string]any{"rel": "ArtifactLink", "url": "vstfs:///Git/Commit/x"},
	)
	srv.SeedWorkItem(10, 1, map[string]any{"System.WorkItemType": "Feature", "System.Title": "Padre"})
	srv.SeedWorkItem(11, 1, map[string]any{"System.WorkItemType": "Bug", "System.Title": "Hermano"})
}

func TestReadItemFullTranslation(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	seedStory(srv)
	p, req, _ := newPort(srv)

	item, err := p.ReadItem(context.Background(), integrations.ItemRequest{PortRequest: req, ID: "42"})
	if err != nil {
		t.Fatalf("ReadItem: %v", err)
	}
	wantURL := srv.URL() + "/contoso/Fabrikam/_workitems/edit/42"
	if item.ID != "42" || item.Type != "User Story" || item.Title != "Alta de usuarios" || item.State != "Active" || item.URL != wantURL {
		t.Fatalf("cabecera = %+v (URL esperada %s)", item, wantURL)
	}
	if item.Description != "<p>Hola</p>" || item.ReproSteps != "<p>pasos</p>" || item.AcceptanceCriteria != "<ul><li>a</li></ul>" {
		t.Fatalf("campos HTML = %+v", item)
	}
	if len(item.Tags) != 3 || item.Tags[0] != "uno" || item.Tags[1] != "Dos" || item.Tags[2] != "tres" {
		t.Fatalf("etiquetas = %q", item.Tags)
	}
	if item.Parent == nil || *item.Parent != (integrations.ItemRelation{ID: "10", Type: "Feature", Title: "Padre"}) {
		t.Fatalf("padre = %+v", item.Parent)
	}
	if len(item.Related) != 1 || item.Related[0] != (integrations.ItemRelation{ID: "11", Type: "Bug", Title: "Hermano"}) {
		t.Fatalf("relacionados = %+v", item.Related)
	}
	if len(item.Attachments) != 1 || item.Attachments[0].Name != "pantalla.png" || item.Attachments[0].Size != 2048 ||
		item.Attachments[0].URL != srv.URL()+"/contoso/Fabrikam/_apis/wit/attachments/g1" {
		t.Fatalf("adjuntos = %+v", item.Attachments)
	}
	if len(item.Warnings) != 0 {
		t.Fatalf("avisos = %v", item.Warnings)
	}
	// Una lectura del WI y un único lote para padre y Related.
	if got := srv.Count("GET", "/contoso/Fabrikam/_apis/wit/workitems/42"); got != 1 {
		t.Errorf("lecturas del WI = %d, esperaba 1", got)
	}
	if got := srv.Count("POST", "/contoso/Fabrikam/_apis/wit/workitemsbatch"); got != 1 {
		t.Errorf("lotes = %d, esperaba 1", got)
	}
}

func TestStateCategoryFromStates(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedStates("User Story", adotest.State{Name: "New", Category: "Proposed"}, adotest.State{Name: "Closed", Category: "InProgress"}, adotest.State{Name: "Hecho", Category: "Completed"})
	p, req, _ := newPort(srv)

	for state, want := range map[string]string{"Closed": "InProgress", "hecho": "Completed", "New": "Proposed"} {
		got, err := p.StateCategory(context.Background(), integrations.StateRequest{PortRequest: req, Type: "User Story", State: state})
		if err != nil || !got.Known || got.Category != want {
			t.Errorf("StateCategory(%q) = %+v, %v; esperaba %s", state, got, err, want)
		}
	}
	if got := srv.Count("GET", "/contoso/Fabrikam/_apis/wit/workitemtypes/User Story/states"); got != 3 {
		t.Errorf("consultas de estados = %d", got)
	}
}

func TestStateCategoryUsesConfiguredTypeSpelling(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedStates("User Story", adotest.State{Name: "New", Category: "Proposed"})
	p, req, _ := newPort(srv)

	// Una grafía hostil en el tipo se escapa como un solo segmento y no compone rutas de la API.
	_, _ = p.StateCategory(context.Background(), integrations.StateRequest{PortRequest: req, Type: "../../x/User Story", State: "New"})
	reqs := srv.Requests()
	if len(reqs) > 1 {
		t.Fatalf("peticiones = %d", len(reqs))
	}
	for _, r := range reqs {
		if !strings.HasPrefix(r.EscapedPath, "/contoso/Fabrikam/_apis/wit/workitemtypes/") || strings.Count(r.EscapedPath, "/") != 7 {
			t.Errorf("la ruta debe contener el tipo como un único segmento escapado: %s", r.EscapedPath)
		}
	}
}

func TestStateCategoryUnknownState(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedStates("Bug", adotest.State{Name: "New", Category: "Proposed"})
	p, req, _ := newPort(srv)

	got, err := p.StateCategory(context.Background(), integrations.StateRequest{PortRequest: req, Type: "Bug", State: "Raro"})
	if err != nil || got.Known || got.Category != "" {
		t.Fatalf("un estado desconocido es Known=false sin error: %+v, %v", got, err)
	}
}

func TestReadItemNotFoundOnlyForItem(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 1, map[string]any{"System.WorkItemType": "Bug", "System.Title": "t"},
		relation(srv, "System.LinkTypes.Related", 99)) // el relacionado no existe: lote 404
	p, req, _ := newPort(srv)

	_, err := p.ReadItem(context.Background(), integrations.ItemRequest{PortRequest: req, ID: "7"})
	if !errors.Is(err, integrations.ErrItemNotFound) {
		t.Fatalf("el 404 del propio WI es ErrItemNotFound: %v", err)
	}
	item, err := p.ReadItem(context.Background(), integrations.ItemRequest{PortRequest: req, ID: "42"})
	if err != nil {
		t.Fatalf("el 404 del lote de relaciones no es un error: %v", err)
	}
	if len(item.Related) != 1 || item.Related[0].ID != "99" || item.Related[0].Title != "" || len(item.Warnings) == 0 {
		t.Fatalf("las relaciones quedan con su identificador y un aviso: %+v", item)
	}

	srv.Reply("GET", "/contoso/Fabrikam/_apis/wit/workitems/5", adotest.Response{Status: http.StatusBadRequest, JSON: map[string]any{"message": "x"}})
	_, err = p.ReadItem(context.Background(), integrations.ItemRequest{PortRequest: req, ID: "5"})
	if err == nil || errors.Is(err, integrations.ErrItemNotFound) {
		t.Fatalf("un 400 no es ErrItemNotFound: %v", err)
	}
}

func TestReadItemUnknownRelationAndMissingFields(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 1, map[string]any{"System.WorkItemType": "Task"},
		map[string]any{"rel": "System.LinkTypes.Dependency-Forward", "url": srv.URL() + "/contoso/_apis/wit/workItems/3"},
		map[string]any{"rel": "System.LinkTypes.Related", "url": "no es una url ::"},
	)
	p, req, _ := newPort(srv)

	item, err := p.ReadItem(context.Background(), integrations.ItemRequest{PortRequest: req, ID: "42"})
	if err != nil {
		t.Fatalf("ReadItem: %v", err)
	}
	if item.Title != "" || item.State != "" || len(item.Tags) != 0 || item.Parent != nil || len(item.Related) != 0 || len(item.Attachments) != 0 {
		t.Fatalf("lo desconocido o ausente se ignora: %+v", item)
	}
	if got := srv.Count("POST", "/contoso/Fabrikam/_apis/wit/workitemsbatch"); got != 0 {
		t.Errorf("sin relaciones útiles no hay lote: %d", got)
	}
}

func TestReadItemForeignRelationNotRequested(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 1, map[string]any{"System.WorkItemType": "Task", "System.Title": "t"},
		map[string]any{"rel": "System.LinkTypes.Hierarchy-Reverse", "url": "http://evil.example/contoso/_apis/wit/workItems/5"},
		map[string]any{"rel": "System.LinkTypes.Related", "url": srv.URL() + "/otra/_apis/wit/workItems/6"},
	)
	p, req, _ := newPort(srv)

	item, err := p.ReadItem(context.Background(), integrations.ItemRequest{PortRequest: req, ID: "42"})
	if err != nil {
		t.Fatalf("ReadItem: %v", err)
	}
	if item.Parent != nil || len(item.Related) != 0 {
		t.Fatalf("las relaciones de otro host u organización se ignoran: %+v", item)
	}
	if len(srv.Requests()) != 1 {
		t.Fatalf("solo la lectura del WI: %d peticiones", len(srv.Requests()))
	}
}

func TestReadItemRedactsPAT(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", "/contoso/Fabrikam/_apis/wit/workitems/42", adotest.Response{Status: http.StatusBadRequest,
		JSON: map[string]any{"message": "no vale " + testPAT}})
	p, req, _ := newPort(srv)

	_, err := p.ReadItem(context.Background(), integrations.ItemRequest{PortRequest: req, ID: "42"})
	if err == nil || strings.Contains(err.Error(), testPAT) {
		t.Fatalf("el error no debe llevar el PAT: %v", err)
	}
}
