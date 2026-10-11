package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// REQ-24.10: `archive list` lee el catálogo de <planning home>/specs desde la
// carpeta maestra y desde un repositorio de código, donde no hay openspec/ propio.
func TestArchiveListReadsThePlanningHomeCatalog(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "specs", "facturacion", "spec.md"),
		"# Facturación\n\n### Requirement: Regla de facturación (REQ-1)\nTexto.\n")

	for name, start := range map[string]string{
		"carpeta maestra":       ws.Master,
		"repositorio de código": filepath.Join(ws.Backend, "src"),
	} {
		t.Run(name, func(t *testing.T) {
			output, code := runAxiom(t, "archive", "list", "--path", start)
			if code != 0 {
				t.Fatalf("archive list salió con %d:\n%s", code, output)
			}
			if !strings.Contains(output, "facturacion") {
				t.Errorf("la salida debe listar el dominio del planning home:\n%s", output)
			}
			if strings.Contains(output, "[AVISO]") {
				t.Errorf("sin divergencia no hay avisos:\n%s", output)
			}
		})
	}
	assertNoOpenSpecOutsideSpecsRepository(t, ws)
}

// REQ-24.10 y REQ-3.1: con la ubicación antigua divergente, `archive list` sigue
// leyendo solo el planning home e imprime el aviso (Q14), aunque el catálogo esté
// vacío.
func TestArchiveListPrintsTheLegacyLocationWarning(t *testing.T) {
	ws := workspacetest.Embedded(t, "specs")
	legacy := filepath.Join(ws.Master, "specs", "specs")
	workspacetest.WriteFile(t, filepath.Join(legacy, "antiguo", "spec.md"), "# Antiguo\n")

	output, code := runAxiom(t, "archive", "list", "--path", ws.Master)
	if code != 0 {
		t.Fatalf("archive list salió con %d:\n%s", code, output)
	}
	if !strings.Contains(output, "[AVISO]") {
		t.Fatalf("la salida debe llevar el aviso de la ubicación divergente:\n%s", output)
	}
	for _, want := range []string{legacy, filepath.Join(ws.Home, "specs")} {
		if !strings.Contains(output, want) {
			t.Errorf("el aviso debe nombrar %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "antiguo") {
		t.Errorf("no debe listar el dominio de la ubicación antigua:\n%s", output)
	}
}

// REQ-24.10: `runUI` reconoce el puntero al decidir si el directorio está
// configurado, para no sustituir la carpeta maestra del canónico por el
// workspace activo del Hub.
func TestHasWorkspaceConfigRecognisesThePointer(t *testing.T) {
	canonical := workspacetest.Canonical(t)
	embedded := workspacetest.Embedded(t, "")
	empty := t.TempDir()

	for _, tt := range []struct {
		name string
		dir  string
		want bool
	}{
		{"carpeta maestra con puntero y sin axiom.yaml", canonical.Master, true},
		{"proyecto con axiom.yaml propio", embedded.Master, true},
		{"repositorio de código sin configuración", canonical.Backend, false},
		{"directorio vacío", empty, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasWorkspaceConfig(tt.dir); got != tt.want {
				t.Errorf("hasWorkspaceConfig(%s) = %v, se esperaba %v", tt.dir, got, tt.want)
			}
		})
	}
	if !fileExists(filepath.Join(canonical.Master, workspace.WorkspacePointerFilename)) ||
		fileExists(filepath.Join(canonical.Master, "axiom.yaml")) {
		t.Fatal("la fixture canónica debe tener puntero y ningún axiom.yaml en la carpeta maestra")
	}
}
