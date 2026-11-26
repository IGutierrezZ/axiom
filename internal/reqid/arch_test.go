package reqid_test

import (
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// idFormat reconoce, en el texto de un literal, `REQ-` seguido de algo que no es una cifra:
// la forma de un identificador (`REQ-n.m`, `REQ-<n>.<m>`, `REQ-x`, un `fmt` con `REQ-%d`).
// Las citas de un requisito concreto (`REQ-21.18`) llevan cifra y no son forma.
var idFormat = regexp.MustCompile(`REQ-([^0-9]|$)`)

// stringLiterals devuelve el valor de los literales de texto de un fichero Go. Los
// comentarios quedan fuera por construcción: el escáner los descarta sin la opción
// ScanComments.
func stringLiterals(t testing.TB, name string, src []byte) []string {
	t.Helper()
	fset := token.NewFileSet()
	file := fset.AddFile(name, fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, src, func(pos token.Position, msg string) { t.Fatalf("%s: %s", pos, msg) }, 0)
	var out []string
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			return out
		}
		if tok == token.STRING {
			out = append(out, lit)
		}
	}
}

// formatLiterals devuelve los literales de src que contienen la forma de un identificador.
func formatLiterals(t testing.TB, name string, src []byte) []string {
	t.Helper()
	var found []string
	for _, lit := range stringLiterals(t, name, src) {
		if idFormat.MatchString(lit) {
			found = append(found, lit)
		}
	}
	return found
}

// TestNoRequirementIDFormatOutsideReqid fija RF-RID-04: ninguna forma de identificador de
// requisito fuera de internal/reqid. Recorre los literales de texto de los `.go` de
// producción de internal/ y cmd/ (sin `_test.go` ni internal/reqid) y falla nombrando el
// fichero si alguno contiene `REQ-` seguido de algo que no sea una cifra.
func TestNoRequirementIDFormatOutsideReqid(t *testing.T) {
	t.Run("detector", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			src  string
			want bool
		}{
			{"letra", `package p; var s = "REQ-x"`, true},
			{"ángulo", `package p; var s = "REQ-<n>.<m>"`, true},
			{"guion", `package p; var s = "REQ--1"`, true},
			{"literal crudo", "package p; var s = `(REQ-n.m)`", true},
			{"formato", `package p; var s = "REQ-%d.%d"`, true},
			{"final", `package p; var s = "REQ-"`, true},
			{"cita concreta", `package p; var s = "(REQ-21.18)"`, false},
			{"comentario", "package p\n// REQ-n.m\n/* REQ-<n>.<m> */\nvar s = 1", false},
		} {
			got := len(formatLiterals(t, "x.go", []byte(tc.src))) > 0
			if got != tc.want {
				t.Errorf("%s: detectado = %v, quería %v", tc.name, got, tc.want)
			}
		}
	})

	root := filepath.Join("..", "..")
	var offenders []string
	scanned := 0
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			slash := filepath.ToSlash(path)
			if entry.IsDir() {
				if strings.HasSuffix(slash, "/internal/reqid") || entry.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			scanned++
			if literals := formatLiterals(t, path, src); len(literals) > 0 {
				offenders = append(offenders, filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator)))+": "+strings.Join(literals, " | "))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if scanned < 100 {
		t.Fatalf("solo se analizaron %d ficheros de producción: la ruta del árbol no es la esperada", scanned)
	}
	sort.Strings(offenders)
	for _, offender := range offenders {
		t.Errorf("forma de identificador fuera de internal/reqid: %s", offender)
	}
}

// TestProvisionalLiteralOnlyInReqid fija REQ-47.1: la forma del identificador provisional
// (el literal `prov:`) solo existe en el código de producción de internal/reqid; el resto
// del repositorio la usa a través de su API.
func TestProvisionalLiteralOnlyInReqid(t *testing.T) {
	t.Run("detector", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			src  string
			want bool
		}{
			{"literal", `package p; var s = "prov:30:1"`, true},
			{"dentro de un formato", `package p; var s = "(prov:%s:%d)"`, true},
			{"literal crudo", "package p; var s = `prov:`", true},
			{"comentario", "package p\n// prov:30:1\nvar s = 1", false},
			{"otra palabra", `package p; var s = "provider"`, false},
		} {
			got := false
			for _, lit := range stringLiterals(t, "x.go", []byte(tc.src)) {
				got = got || strings.Contains(lit, "prov:")
			}
			if got != tc.want {
				t.Errorf("%s: detectado = %v, quería %v", tc.name, got, tc.want)
			}
		}
	})

	root := filepath.Join("..", "..")
	var offenders []string
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			slash := filepath.ToSlash(path)
			if entry.IsDir() {
				if strings.HasSuffix(slash, "/internal/reqid") || entry.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, lit := range stringLiterals(t, path, src) {
				if strings.Contains(lit, "prov:") {
					offenders = append(offenders, filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator)))+": "+lit)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(offenders)
	for _, offender := range offenders {
		t.Errorf("forma de provisional fuera de internal/reqid: %s", offender)
	}
}
