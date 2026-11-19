package manifest

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestLeafImportsOnlyStandardLibrary fija que manifest es una hoja pura: sus ficheros que no son de
// prueba solo importan la biblioteca estándar, así que internal/workspace puede importarlo sin ciclos
// y manifest nunca depende de internal/workspace ni de internal/integrations.
func TestLeafImportsOnlyStandardLibrary(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	checked := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("leer %s: %v", name, err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsear %s: %v", name, err)
		}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("importación ilegible en %s: %v", name, err)
			}
			first := strings.SplitN(path, "/", 2)[0]
			if strings.Contains(first, ".") {
				t.Errorf("%s importa %q, que no es de la biblioteca estándar", name, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("la prueba de hoja no encontró ficheros que comprobar")
	}
}
