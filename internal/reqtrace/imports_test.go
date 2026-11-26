package reqtrace

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRequirementTracePackageImports fija prov:trace:10 por arquitectura: ningún fichero de
// producción importa `os`, `os/exec` ni internal/gitcommon, así que el dominio solo puede
// leer a través de io/fs.FS y de isoevidence.History.
func TestRequirementTracePackageImports(t *testing.T) {
	forbidden := map[string]bool{
		"os":      true,
		"os/exec": true,
		"github.com/IGutierrezZ/axiom/v3/internal/gitcommon": true,
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
				t.Errorf("%s importa %s: reqtrace es de solo lectura y no toca el sistema de ficheros ni ejecuta órdenes", file, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no se encontró ningún fichero de producción que comprobar")
	}
}
