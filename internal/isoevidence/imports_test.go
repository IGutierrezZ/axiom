package isoevidence

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPackageImports fija la arquitectura del paquete hoja: ningún fichero de
// producción importa `os/exec` ni un paquete interno de Axiom.
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
				t.Errorf("%s importa os/exec: el paquete no ejecuta órdenes", file)
			}
			if strings.Contains(path, "/internal/") || strings.HasPrefix(path, "internal/") {
				t.Errorf("%s importa el paquete interno %s: isoevidence es un paquete hoja", file, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no se encontró ningún fichero de producción que comprobar")
	}
}
