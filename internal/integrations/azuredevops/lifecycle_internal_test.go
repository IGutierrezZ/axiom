package azuredevops

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const lifecycleFixture = "workspace:\n  name: test-workspace\n  topology: monorepo-embedded\nroles:\n  core:\n    name: core\n    repositories:\n      - path: .\n" +
	"integrations:\n  azure-devops:\n    organization: contoso\n    project: Fabrikam\n    team: Fabrikam Team\n"

// Escenario «Valores por defecto de las claves nuevas» de RF-ADO-02: un bloque sin las claves del ciclo
// de vida carga con los valores por defecto del manifiesto y lifecycleConfigFrom los entrega tipados.
func TestADOLifecycleConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "axiom.yaml"), []byte(lifecycleFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	loc, err := workspace.Locate(dir)
	if err != nil || loc.Config == nil || loc.Config.Integrations == nil {
		t.Fatalf("Locate: %v, config = %+v", err, loc.Config)
	}
	var values map[string]any
	for _, p := range loc.Config.Integrations.Plugins {
		if p.ID == "azure-devops" {
			values = p.Values
		}
	}
	if values == nil {
		t.Fatal("el bloque azure-devops no se cargó")
	}

	got := lifecycleConfigFrom(values)

	want := lifecycleConfig{
		TaskStateStart:         "Active",
		TaskStateApplyApproved: "Tech Review",
		TaskStateClose:         "Tech Review", // derivado de task_state_apply_approved
		WorkItemStateArchive:   "",            // sin valor: el cierre no transiciona el work item
		ReqTagPrefix:           "req:",
		ReqTagMax:              50,
		WorkdayHours:           8,
		Workdays:               []string{"mon", "tue", "wed", "thu", "fri"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lifecycleConfigFrom = %+v\nesperaba %+v", got, want)
	}
}

func TestLifecycleConfigFromExplicitValues(t *testing.T) {
	got := lifecycleConfigFrom(map[string]any{
		"task_state_start":          "In Progress",
		"task_state_apply_approved": "Review",
		"task_state_close":          "Closed",
		"work_item_state_archive":   "Released",
		"req_tag_prefix":            "rq:",
		"req_tag_max":               int64(10),
		"workday_hours":             float64(6),
		"workdays":                  []any{"mon", "wed"},
	})

	want := lifecycleConfig{
		TaskStateStart: "In Progress", TaskStateApplyApproved: "Review", TaskStateClose: "Closed",
		WorkItemStateArchive: "Released", ReqTagPrefix: "rq:", ReqTagMax: 10, WorkdayHours: 6, Workdays: []string{"mon", "wed"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lifecycleConfigFrom = %+v\nesperaba %+v", got, want)
	}
}
