package adotest_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

func doJSON(t *testing.T, srv *adotest.Server, method, path, contentType, body string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL()+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := srv.HTTPClient().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestServerCreatesTaskWithRelationAndLinksTheParent(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 1, map[string]any{"System.WorkItemType": "User Story"})
	body := `[{"op":"add","path":"/fields/System.Title","value":"t"},` +
		`{"op":"add","path":"/relations/-","value":{"rel":"System.LinkTypes.Hierarchy-Reverse","url":"` + srv.URL() + `/contoso/_apis/wit/workItems/42"}}]`

	status, out := doJSON(t, srv, "POST", "/contoso/Fabrikam/_apis/wit/workitems/$Task", "application/json-patch+json", body)

	if status != 200 || out["id"] != 43.0 || out["rev"] != 1.0 {
		t.Fatalf("respuesta = %d %v", status, out)
	}
	parent, _ := srv.WorkItem(42)
	if len(parent.Relations) != 1 || parent.Relations[0]["rel"] != "System.LinkTypes.Hierarchy-Forward" ||
		!strings.HasSuffix(parent.Relations[0]["url"].(string), "/contoso/_apis/wit/workItems/43") {
		t.Fatalf("relaciones del padre = %v", parent.Relations)
	}
	if srv.Created() != 1 {
		t.Fatalf("Created = %d", srv.Created())
	}
}

func TestServerRejectCreateRelationsAnswers400WithoutCreating(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 1, map[string]any{"System.WorkItemType": "User Story"})
	srv.RejectCreateRelations(true)
	withRel := `[{"op":"add","path":"/relations/-","value":{"rel":"System.LinkTypes.Hierarchy-Reverse","url":"x/workItems/42"}}]`
	withoutRel := `[{"op":"add","path":"/fields/System.Title","value":"t"}]`

	if status, _ := doJSON(t, srv, "POST", "/o/p/_apis/wit/workitems/$Task", "application/json-patch+json", withRel); status != 400 {
		t.Fatalf("con relación: %d", status)
	}
	if srv.Created() != 0 {
		t.Fatal("el 400 no crea")
	}
	if status, _ := doJSON(t, srv, "POST", "/o/p/_apis/wit/workitems/$Task", "application/json-patch+json", withoutRel); status != 200 {
		t.Fatalf("sin relación: %d", status)
	}
}

func TestServerWIQLFiltersByTypeAndTagAndCanFail(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(1, 1, map[string]any{"System.WorkItemType": "Task", "System.Tags": "a; Axiom-Task:x:y"})
	srv.SeedWorkItem(2, 1, map[string]any{"System.WorkItemType": "Bug", "System.Tags": "axiom-task:x:y"})
	srv.SeedWorkItem(3, 1, map[string]any{"System.WorkItemType": "Task", "System.Tags": "otra"})
	query := `{"query":"SELECT [System.Id] FROM WorkItems WHERE [System.WorkItemType] = 'Task' AND [System.Tags] CONTAINS 'axiom-task:x:y'"}`

	status, out := doJSON(t, srv, "POST", "/o/p/_apis/wit/wiql", "application/json", query)

	items, _ := out["workItems"].([]any)
	if status != 200 || len(items) != 1 || items[0].(map[string]any)["id"] != 1.0 {
		t.Fatalf("WIQL = %d %v", status, out)
	}
	srv.FailWIQL(500)
	if status, _ := doJSON(t, srv, "POST", "/o/p/_apis/wit/wiql", "application/json", query); status != 500 {
		t.Fatalf("WIQL caído = %d", status)
	}
}

func TestServerSeedTypeFieldsAndMembers(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedTypeFields("Task", "Microsoft.VSTS.Scheduling.OriginalEstimate")
	srv.SeedMembers(adotest.Member{ID: "m1", DisplayName: "Ana", UniqueName: "ana.unique.fixture"}, adotest.Member{ID: "m2", DisplayName: "Luis"})

	status, fields := doJSON(t, srv, "GET", "/o/p/_apis/wit/workitemtypes/Task/fields", "", "")
	if status != 200 || fields["count"] != 1.0 {
		t.Fatalf("campos = %d %v", status, fields)
	}
	status, page := doJSON(t, srv, "GET", "/o/_apis/projects/p/teams/t/members?$top=1&$skip=1", "", "")
	value, _ := page["value"].([]any)
	if status != 200 || len(value) != 1 || value[0].(map[string]any)["identity"].(map[string]any)["id"] != "m2" {
		t.Fatalf("miembros = %d %v", status, page)
	}
}
