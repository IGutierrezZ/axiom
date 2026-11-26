package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readManualDeInicio lee el manual real de docs/ con una ruta relativa al
// paquete: los pins de documentación leen el texto que se publica, no una copia.
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

// readRepoFile lee un fichero versionado del repositorio con una ruta relativa a
// la raíz (los pins de scripts y documentos leen el texto que se publica).
func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", rel, err)
	}
	return string(raw)
}

// RF-TOP-20: el manual y docs/openspec-config.md documentan el bloque
// `acceptance_tests:` y el verbo de comprobación.
func TestDocsDocumentAcceptanceTests(t *testing.T) {
	assertDocContains(t, readManualDeInicio(t),
		"### Batería de Pruebas de Aceptación (`acceptance_tests:`)",
		"acceptance_tests:\n  enabled: true\n  environments: [local, dev, pre]",
		"**Solo multirepo.**",
		"axiom sdd tests check",
	)
	config := readRepoFile(t, "docs/openspec-config.md")
	for _, want := range []string{"`acceptance_tests` also lives only in `axiom.yaml`", "axiom sdd tests check"} {
		if !strings.Contains(config, want) {
			t.Errorf("docs/openspec-config.md no contiene %q", want)
		}
	}
}
