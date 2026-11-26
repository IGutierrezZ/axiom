package acceptancetests

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var allowedInternal = map[string]bool{
	"github.com/IGutierrezZ/axiom/v3/internal/reqid":       true,
	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence": true,
}

// TestPackageImports fija el núcleo puro: solo internal/reqid, internal/isoevidence y la
// biblioteca estándar; nunca os/exec.
func TestPackageImports(t *testing.T) {
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
			path := strings.Trim(spec.Path.Value, `"`)
			if path == "os/exec" {
				t.Errorf("%s importa os/exec", file)
			}
			if strings.Contains(path, "/internal/") && !allowedInternal[path] {
				t.Errorf("%s importa el paquete interno %s", file, path)
			}
			if strings.Contains(path, ".") && !strings.Contains(path, "/internal/") {
				t.Errorf("%s importa el paquete externo %s", file, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no hay ficheros de producción que comprobar")
	}
}
