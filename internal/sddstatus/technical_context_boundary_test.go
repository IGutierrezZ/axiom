package sddstatus

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// REQ-30.32 (conducta): `technical_context` vive solo en `axiom.yaml`. Una clave
// con ese nombre en `openspec/config.yaml` no tiene efecto: el status no la
// proyecta y el archivado no la lee; y `governance.context` convive con la
// declaración de `axiom.yaml` sin confundirse con ella.

const boundaryOpenspecConfig = "schema: spec-driven\ntechnical_context:\n  maintained:\n    paths: [context/]\n"

func TestTechnicalContextBoundaryOpenspecConfigDoesNotDeclareIt(t *testing.T) {
	for name, axiomYAML := range map[string]string{
		"axiom.yaml sin technical_context": declEmbeddedBase,
		"sin axiom.yaml":                   "",
	} {
		t.Run(name, func(t *testing.T) {
			ws := workspacetest.Embedded(t, ".")
			if axiomYAML == "" {
				if err := os.Remove(filepath.Join(ws.Master, "axiom.yaml")); err != nil {
					t.Fatal(err)
				}
			} else {
				workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), axiomYAML)
			}
			workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), boundaryOpenspecConfig)
			seedReadyChange(t, ws.Master, "demo", planningHomeTasks)

			status := resolveFrom(t, ws.Master, "demo")
			if status.TechnicalContext != nil {
				t.Fatalf("status.TechnicalContext = %+v: la clave de openspec/config.yaml no declara contexto técnico", status.TechnicalContext)
			}
			fields := statusJSON(t, status)
			if _, present := fields["technicalContext"]; present {
				t.Error("`axiom sdd status --json` no debe proyectar technicalContext desde openspec/config.yaml")
			}
			if len(status.Notes) != 0 {
				t.Errorf("notes = %v: ignorar la clave no genera avisos", status.Notes)
			}
		})
	}
}

func TestTechnicalContextBoundaryGovernanceContextCoexistsWithTheDeclaration(t *testing.T) {
	ws := embeddedWithDeclaration(t, "governance:\n  context: docs/contexto.md\n"+declBothModes)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), boundaryOpenspecConfig)

	loc, err := workspace.Locate(ws.Master)
	if err != nil {
		t.Fatal(err)
	}
	if loc.Config == nil || loc.Config.Governance.Context != "docs/contexto.md" {
		t.Fatalf("governance.context = %+v: debe conservarse junto a technical_context", loc.Config)
	}
	status := resolveFrom(t, ws.Master, "demo")
	if status.TechnicalContext == nil || status.TechnicalContext.Maintained == nil || status.TechnicalContext.Reference == nil {
		t.Fatalf("technicalContext = %+v: la declaración de axiom.yaml se proyecta", status.TechnicalContext)
	}
	// La clave de openspec/config.yaml no se mezcla con la declaración de axiom.yaml.
	want := []string{filepath.Join(ws.Master, "context")}
	if !reflect.DeepEqual(status.TechnicalContext.Maintained.Paths, want) {
		t.Errorf("maintained.paths = %v, se esperaba %v (de axiom.yaml, no de openspec/config.yaml)", status.TechnicalContext.Maintained.Paths, want)
	}
	if status.TechnicalContext.Maintained.Entry != filepath.Join(ws.Master, "context", "TECHNICAL_CONTEXT.md") {
		t.Errorf("maintained.entry = %q", status.TechnicalContext.Maintained.Entry)
	}
}
