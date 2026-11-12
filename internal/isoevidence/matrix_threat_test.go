package isoevidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMatrixNeverExecutesCommands es la prueba RED de amenazas de design §8
// («rutas tipo documentación»): los comandos de la matriz y los ficheros
// referenciados son datos. Una fila con un comando que crearía un fichero
// centinela y una referencia a un `README.sh`: tras validar, el centinela no
// existe y la referencia resuelve solo por texto.
func TestMatrixNeverExecutesCommands(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "centinela")
	script := "#!/bin/sh\ntouch " + filepath.ToSlash(sentinel) + "\n# Sym\n"
	root := writeTree(t, map[string]string{"README.sh": script})
	resolver := newResolver(t, root)

	report := "## Matriz de verificación\n\n" + matrixHeader +
		"| REQ-1.1 | Esc | e2e | README.sh:Sym | `touch " + filepath.ToSlash(sentinel) + " && rm -rf /` | 0 | | | | pass | |\n"
	got := ValidateMatrix(ParseMatrix(report), []ScenarioKey{{"REQ-1.1", "Esc"}}, false, resolver)

	if got.State != MatrixValid || !got.Successful {
		t.Fatalf("la fila debía validar por texto: %+v (%v)", got, reasons(got.Findings))
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("el centinela existe (err=%v): se ejecutó el comando de la matriz", err)
	}
	after, err := os.ReadFile(filepath.Join(root, "README.sh"))
	if err != nil || string(after) != script {
		t.Fatalf("la validación modificó el fichero referenciado: %v", err)
	}
}

// TestMatrixRejectsEscapingReferences recorre las referencias adversarias de
// design §8 de extremo a extremo: ParseMatrix y ValidateMatrix con el resolutor
// real, y el motivo nombra «fuera del árbol».
func TestMatrixRejectsEscapingReferences(t *testing.T) {
	root := writeTree(t, map[string]string{"dentro_test.go": "func TestX(t *testing.T) {}\n"})
	resolver := newResolver(t, root)

	for _, ref := range []string{`C:\x:Sym`, `\\srv\x:Sym`, "../x_test.go:TestX", "/etc/passwd:root"} {
		t.Run(ref, func(t *testing.T) {
			// Dentro de una celda de tabla la referencia va entre comillas invertidas.
			report := "## Matriz de verificación\n\n" + matrixHeader +
				"| REQ-1.1 | Esc | unitario | `" + ref + "` | go test | 0 | | | | pass | |\n"
			got := ValidateMatrix(ParseMatrix(report), []ScenarioKey{{"REQ-1.1", "Esc"}}, false, resolver)
			if got.State != MatrixInvalid || len(got.Findings) != 1 {
				t.Fatalf("resultado = %+v (%v)", got, reasons(got.Findings))
			}
			if !strings.Contains(got.Findings[0].Reason(), "fuera del árbol") {
				t.Errorf("el motivo no dice «fuera del árbol»: %q", got.Findings[0].Reason())
			}
		})
	}
}
