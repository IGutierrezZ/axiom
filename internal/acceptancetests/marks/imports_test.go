package marks

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPackageImports fija el núcleo puro: solo internal/acceptancetests, yaml.v3 y la
// biblioteca estándar; sin os/exec y sin E/S de ficheros.
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
			switch {
			case path == "os/exec", path == "os", path == "io/ioutil", path == "net", path == "net/http":
				t.Errorf("%s importa %s", file, path)
			case strings.Contains(path, "/internal/"):
				if path != "github.com/IGutierrezZ/axiom/v3/internal/acceptancetests" {
					t.Errorf("%s importa el paquete interno %s", file, path)
				}
			case strings.Contains(path, ".") && path != "gopkg.in/yaml.v3":
				t.Errorf("%s importa el paquete externo %s", file, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no hay ficheros de producción que comprobar")
	}
}
