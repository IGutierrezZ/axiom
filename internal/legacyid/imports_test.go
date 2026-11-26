package legacyid

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLegacyIDPackageImports fija por arquitectura que el modelo del mapa no ejecuta órdenes,
// no usa git y no depende de reqtrace (la interfaz LegacyResolver la implementa de forma
// estructural).
func TestLegacyIDPackageImports(t *testing.T) {
	forbidden := map[string]bool{
		"os/exec": true,
		"github.com/IGutierrezZ/axiom/v3/internal/gitcommon": true,
		"github.com/IGutierrezZ/axiom/v3/internal/reqtrace":  true,
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		checked++
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			if path := strings.Trim(spec.Path.Value, `"`); forbidden[path] {
				t.Errorf("%s importa %s: legacyid no ejecuta órdenes, no usa git y no depende de reqtrace", file, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no se encontró ningún fichero de producción que comprobar")
	}
}
