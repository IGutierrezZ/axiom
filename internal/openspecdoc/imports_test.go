package openspecdoc

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const reqidImport = "github.com/IGutierrezZ/axiom/v3/internal/reqid"

// RF-RID-04: openspecdoc sigue siendo un paquete hoja; sus ficheros de producción solo
// importan la biblioteca estándar e internal/reqid.
func TestOpenspecdocImportsOnlyStandardLibraryAndReqid(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			first, _, _ := strings.Cut(path, "/")
			if strings.Contains(first, ".") && path != reqidImport {
				t.Errorf("%s importa %q: solo se admiten la biblioteca estándar e internal/reqid", name, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("ningún fichero de producción analizado")
	}
}
