package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readManualDeInicio lee el manual real de docs/ con una ruta relativa al
// paquete (mismo patrón que internal/absorptionledger/ledger_test.go): los pins
// de documentación leen el texto que se publica, no una copia.
func readManualDeInicio(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "manual-de-inicio.md"))
	if err != nil {
		t.Fatalf("no se pudo leer docs/manual-de-inicio.md: %v", err)
	}
	return string(raw)
}

func assertDocContains(t *testing.T, doc string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(doc, want) {
			t.Errorf("docs/manual-de-inicio.md no contiene %q", want)
		}
	}
}

// REQ-24.12: la receta del espacio canónico multirepo se documenta, con el
// puntero, la configuración a la que apunta y el montaje manual.
func TestManualDeInicioDocumentsTheCanonicalWorkspaceRecipe(t *testing.T) {
	assertDocContains(t, readManualDeInicio(t),
		"### Espacio Canónico Multirepo (Montaje Manual)",
		"se monta a mano",
		"carpeta maestra no versionada",
		"`.axiom-workspace`",
		"`<specs_repository>/axiom.yaml`",
		"config: repo-specs/axiom.yaml",
		"`<maestra>/repo-specs/openspec`",
		"`axiom workspace validate --path <maestra>`",
		"`axiom sdd-attempt grant`",
	)
}

// REQ-24.12 y REQ-21.16: la evidencia pr_merged se verifica contra el
// repositorio de especificaciones y la de los repositorios de código es otra.
func TestManualDeInicioDocumentsPRMergedEvidence(t *testing.T) {
	assertDocContains(t, readManualDeInicio(t),
		"`axiom sdd gate record --evidence-kind pr_merged`",
		"se verifica contra el grafo git del repositorio de especificaciones (`workspaceRoot`)",
		"con las evidencias `deployment` o `attestation`",
	)
}

// REQ-24.21: la frontera entre axiom.yaml y openspec/config.yaml se documenta.
func TestOpenSpecConfigDocumentsTheConfigurationBoundary(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "openspec-config.md"))
	if err != nil {
		t.Fatalf("no se pudo leer docs/openspec-config.md: %v", err)
	}
	doc := string(raw)
	for _, want := range []string{
		"## Frontera de configuración con `axiom.yaml`",
		"`baseline` y `technical_context` viven solo en `axiom.yaml`",
		"Una clave con ese nombre en `openspec/config.yaml` no tiene ningún efecto",
		"Las reglas de conducta (`rules.<fase>`, `strict_tdd`, `artifact_store`) permanecen en `openspec/config.yaml`",
		"`governance.context` no cambia",
		"debe ser `openspec` o `hybrid`",
		"`axiom workspace validate --path <carpeta maestra>`",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("docs/openspec-config.md no contiene %q", want)
		}
	}
}

// m6 de design §13: el cambio de resolución desde un subdirectorio sin
// marcadores propios y el aviso de usar el binario nuevo en todo el equipo.
func TestManualDeInicioDocumentsTheResolutionChangeFromASubdirectory(t *testing.T) {
	assertDocContains(t, readManualDeInicio(t),
		"desde un subdirectorio sin marcadores propios resolvían `<subdirectorio>/openspec`",
		"Ahora resuelven el planning home del marcador más cercano",
		"todo el equipo debe usar el binario nuevo",
	)
}
