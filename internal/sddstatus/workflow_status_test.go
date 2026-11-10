package sddstatus

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// workflowOf lee la proyección `workflow` de un status en sus dos contratos, v1
// (el struct serializado) y v2 (ProjectStatusV2), y exige que coincidan.
func workflowOf(t *testing.T, status Status) WorkflowProjection {
	t.Helper()

	rawV1, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	var v1 map[string]json.RawMessage
	if err := json.Unmarshal(rawV1, &v1); err != nil {
		t.Fatal(err)
	}
	fromV1 := decodeField[WorkflowProjection](t, v1, "workflow")
	fromV2 := decodeField[WorkflowProjection](t, statusJSON(t, status), "workflow")
	if fromV1 != fromV2 {
		t.Fatalf("v1 = %+v y v2 = %+v deben proyectar lo mismo", fromV1, fromV2)
	}
	return fromV1
}

// REQ-27.2: las cuatro filas de la tabla, con y sin cambio seleccionado, en v1 y v2.
func TestWorkflowStatusProjectsLaneAndSource(t *testing.T) {
	tests := []struct {
		name       string
		config     string // vacío: sin axiom.yaml
		wantLane   string
		wantSource string
	}{
		{"sin axiom.yaml", "", "ask", "no_workspace"},
		{"axiom.yaml sin bloque", declEmbeddedBase, "sdd", "default"},
		{"bloque sin lane", declEmbeddedBase + "workflow: {}\n", "sdd", "default"},
		{"lane odd", declEmbeddedBase + "workflow:\n  lane: odd\n", "odd", "declared"},
		{"lane sdd", declEmbeddedBase + "workflow:\n  lane: sdd\n", "sdd", "declared"},
		{"lane ask", declEmbeddedBase + "workflow:\n  lane: ask\n", "ask", "declared"},
		{"lane inválido", declEmbeddedBase + "workflow:\n  lane: SDD\n", "ask", "invalid"},
		{"bloque que no es un mapeo", declEmbeddedBase + "workflow: sdd\n", "ask", "invalid"},
		{"axiom.yaml ilegible", "workspace: [sin cerrar\n", "ask", "invalid"},
		// Fallo cerrado: un bloque válido con la base rota sigue siendo invalid.
		{"bloque válido con base rota", "workspace: {name: x, topology: monorepo-embedded}\nworkflow:\n  lane: odd\n", "ask", "invalid"},
	}
	for _, tt := range tests {
		for _, change := range []string{"demo", ""} {
			name := tt.name + " con cambio"
			if change == "" {
				name = tt.name + " sin cambio"
			}
			t.Run(name, func(t *testing.T) {
				root := t.TempDir() // sin axiom.yaml: solo openspec/
				if tt.config != "" {
					ws := workspacetest.Embedded(t, ".")
					workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), tt.config)
					root = ws.Master
				}
				seedReadyChange(t, root, "demo", planningHomeTasks)

				got := workflowOf(t, resolveFrom(t, root, change))
				if got.Lane != tt.wantLane || got.Source != tt.wantSource {
					t.Errorf("workflow = %+v, se esperaba {%s %s}", got, tt.wantLane, tt.wantSource)
				}
			})
		}
	}
}

// REQ-27.2: el campo es informativo; no añade notas ni bloqueos ni cambia la ruta
// recomendada ni las dependencias respecto de un axiom.yaml sin el bloque.
func TestWorkflowStatusDoesNotChangeTheRouteOrAddNotes(t *testing.T) {
	baseline := embeddedWithDeclaration(t, "")
	reference := resolveFrom(t, baseline.Master, "demo")

	for name, block := range map[string]string{
		"lane odd":      "workflow:\n  lane: odd\n",
		"lane ask":      "workflow:\n  lane: ask\n",
		"lane inválido": "workflow:\n  lane: nope\n",
	} {
		t.Run(name, func(t *testing.T) {
			ws := embeddedWithDeclaration(t, block)
			status := resolveFrom(t, ws.Master, "demo")

			if status.NextRecommended != reference.NextRecommended {
				t.Errorf("nextRecommended = %q, se esperaba %q", status.NextRecommended, reference.NextRecommended)
			}
			if !reflect.DeepEqual(status.Dependencies, reference.Dependencies) {
				t.Errorf("dependencies = %+v, se esperaba %+v", status.Dependencies, reference.Dependencies)
			}
			if len(status.Notes) != len(reference.Notes) {
				t.Errorf("notes = %v, el carril no añade avisos (%v)", status.Notes, reference.Notes)
			}
			if len(status.BlockedReasons) != len(reference.BlockedReasons) {
				t.Errorf("blockedReasons = %v, el carril no añade bloqueos (%v)", status.BlockedReasons, reference.BlockedReasons)
			}
		})
	}
}

// REQ-27.2: un proyecto solo de openspec/ proyecta ask/no_workspace sin alterar el
// resto del status; y la raíz explícita lo proyecta igual que el directorio.
func TestWorkflowStatusWithoutAxiomYAMLAndWithExplicitRoot(t *testing.T) {
	bare := t.TempDir()
	seedReadyChange(t, bare, "demo", planningHomeTasks)
	got := workflowOf(t, resolveFrom(t, bare, "demo"))
	if got != (WorkflowProjection{Lane: "ask", Source: "no_workspace"}) {
		t.Errorf("sin axiom.yaml = %+v", got)
	}

	ws := embeddedWithDeclaration(t, "workflow:\n  lane: odd\n")
	status, err := Resolve(ResolveOptions{WorkspaceRoot: ws.Master, ChangeName: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if got := workflowOf(t, status); got != (WorkflowProjection{Lane: "odd", Source: "declared"}) {
		t.Errorf("raíz explícita = %+v", got)
	}
}
