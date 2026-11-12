package isoevidence

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeTree escribe los ficheros (ruta con `/` relativa a la raíz) en un árbol
// temporal y devuelve su raíz.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// linkDir crea un enlace simbólico a un directorio. En Windows sin privilegio
// para enlaces simbólicos recurre a una unión de directorios (`mklink /J`),
// que no lo necesita y que `os.Root` también debe tratar como salida del árbol.
func linkDir(target, link string) error {
	err := os.Symlink(target, link)
	if err == nil || runtime.GOOS != "windows" {
		return err
	}
	if out, jerr := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); jerr != nil {
		return fmt.Errorf("%w; unión de directorios: %v: %s", err, jerr, out)
	}
	return nil
}

func newResolver(t *testing.T, root string) ClosableTreeResolver {
	t.Helper()
	r, err := NewTreeResolver(root)
	if err != nil {
		t.Fatalf("NewTreeResolver: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

// TestTreeResolverRejectsEscapes es la prueba RED de amenazas (design §8): toda
// referencia que sale del árbol se rechaza con el resultado «fuera del árbol».
func TestTreeResolverRejectsEscapes(t *testing.T) {
	outside := writeTree(t, map[string]string{"secreto_test.go": "func TestX(t *testing.T) {}\n"})
	root := writeTree(t, map[string]string{"dentro_test.go": "func TestX(t *testing.T) {}\n"})
	r := newResolver(t, root)

	cases := []struct {
		name string
		path string
	}{
		{"unidad de Windows", `C:\x`},
		{"unidad de Windows con barra", "C:/x_test.go"},
		{"ruta UNC", `\\srv\x`},
		{"salida con punto punto", "../x_test.go"},
		{"salida anidada", "a/../../x_test.go"},
		{"ruta absoluta con barra", "/etc/passwd"},
		{"ruta absoluta del sistema", filepath.ToSlash(filepath.Join(outside, "secreto_test.go"))},
		{"barra invertida", `dentro\x_test.go`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := r.ResolvePathSymbol(tc.path, "Sym")
			if err != nil {
				t.Fatalf("la ruta que sale del árbol debía rechazarse con un resultado, no con un error: %v", err)
			}
			if got != OutcomeOutsideTree {
				t.Fatalf("resultado = %q, quería %q", got, OutcomeOutsideTree)
			}
		})
	}

	t.Run("enlace simbólico a un directorio fuera del árbol", func(t *testing.T) {
		link := filepath.Join(root, "enlace")
		if err := linkDir(outside, link); err != nil {
			t.Skipf("el sistema no permite crear enlaces simbólicos: %v", err)
		}
		got, err := r.ResolvePathSymbol("enlace/secreto_test.go", "TestX")
		if err != nil {
			t.Fatalf("el enlace que sale del árbol debía rechazarse con un resultado: %v", err)
		}
		if got != OutcomeOutsideTree {
			t.Fatalf("resultado = %q, quería %q", got, OutcomeOutsideTree)
		}
	})

	t.Run("enlace simbólico a un fichero fuera del árbol", func(t *testing.T) {
		link := filepath.Join(root, "enlace_fichero_test.go")
		if err := os.Symlink(filepath.Join(outside, "secreto_test.go"), link); err != nil {
			t.Skipf("el sistema no permite crear enlaces simbólicos: %v", err)
		}
		got, err := r.ResolvePathSymbol("enlace_fichero_test.go", "TestX")
		if err != nil {
			t.Fatalf("el enlace que sale del árbol debía rechazarse con un resultado: %v", err)
		}
		if got != OutcomeOutsideTree {
			t.Fatalf("resultado = %q, quería %q", got, OutcomeOutsideTree)
		}
	})
}

// TestTreeResolverNeverExecutes: las referencias son datos. Un fichero de
// aspecto ejecutable (`README.sh`) solo se lee como texto: resolver nunca
// ejecuta nada ni crea ficheros.
func TestTreeResolverNeverExecutes(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "centinela")
	script := "#!/bin/sh\ntouch " + filepath.ToSlash(sentinel) + "\n# Sym\n"
	root := writeTree(t, map[string]string{"README.sh": script})
	r := newResolver(t, root)

	got, err := r.ResolvePathSymbol("README.sh", "Sym")
	if err != nil {
		t.Fatal(err)
	}
	if got != OutcomeResolved {
		t.Fatalf("la referencia a README.sh debía resolver solo por texto, y dio %q", got)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("el centinela existe (err=%v): se ejecutó algo durante la resolución", err)
	}
	after, err := os.ReadFile(filepath.Join(root, "README.sh"))
	if err != nil || string(after) != script {
		t.Fatalf("la resolución modificó el fichero referenciado: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("la resolución dejó ficheros nuevos en el árbol: %v %v", entries, err)
	}
}

func TestTreeResolverPathSymbol(t *testing.T) {
	big := strings.Repeat("x ", maxReferenceFileBytes/2)
	root := writeTree(t, map[string]string{
		"internal/pagos/cobro_test.go": "package pagos\n\nfunc TestCobraIVA(t *testing.T) {}\n",
		"e2e/checkout.spec.ts":         "test('x', async () => { await pagaConTarjeta(); });\n",
		"tests/test_pago.py":           "def test_cobra_iva():\n    pass\n",
		"solo_subcadena.ts":            "function pagaConTarjeta() {}\n",
		"con_dolar.js":                 "const $paga = 1;\nconst otro_paga = 2;\n",
		"grande.txt":                   big + "SimboloTardio\n",
		"grande_temprano.txt":          "SimboloTemprano\n" + big,
		"vacio.txt":                    "",
	})
	r := newResolver(t, root)

	cases := []struct {
		name   string
		path   string
		symbol string
		want   ResolveOutcome
	}{
		{"forma general en Go", "internal/pagos/cobro_test.go", "TestCobraIVA", OutcomeResolved},
		{"TypeScript", "e2e/checkout.spec.ts", "pagaConTarjeta", OutcomeResolved},
		{"Python", "tests/test_pago.py", "test_cobra_iva", OutcomeResolved},
		{"fichero inexistente", "e2e/otro.spec.ts", "pagaConTarjeta", OutcomeFileMissing},
		{"un directorio no es un fichero", "e2e", "pagaConTarjeta", OutcomeFileMissing},
		{"símbolo ausente", "e2e/checkout.spec.ts", "noExiste", OutcomeSymbolMissing},
		{"subcadena de otro identificador: prefijo", "solo_subcadena.ts", "paga", OutcomeSymbolMissing},
		{"subcadena de otro identificador: sufijo", "solo_subcadena.ts", "ConTarjeta", OutcomeSymbolMissing},
		{"el carácter dólar forma parte del identificador", "con_dolar.js", "paga", OutcomeSymbolMissing},
		{"el identificador completo con dólar resuelve", "con_dolar.js", "$paga", OutcomeResolved},
		{"el guion bajo forma parte del identificador", "con_dolar.js", "paga", OutcomeSymbolMissing},
		{"símbolo vacío", "e2e/checkout.spec.ts", "", OutcomeSymbolMissing},
		{"fichero vacío", "vacio.txt", "X", OutcomeSymbolMissing},
		{"símbolo más allá del límite de lectura", "grande.txt", "SimboloTardio", OutcomeSymbolMissing},
		{"símbolo dentro del límite de lectura", "grande_temprano.txt", "SimboloTemprano", OutcomeResolved},
		{"ruta con ./ al principio", "./e2e/checkout.spec.ts", "pagaConTarjeta", OutcomeResolved},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := r.ResolvePathSymbol(tc.path, tc.symbol)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if got != tc.want {
				t.Fatalf("resultado = %q, quería %q", got, tc.want)
			}
		})
	}
}

func TestResolveGoTest(t *testing.T) {
	root := writeTree(t, map[string]string{
		"internal/pagos/cobro_test.go":    "package pagos\n\nimport \"testing\"\n\nfunc TestCobraIVA(t *testing.T) {}\n\nfunc TestCobroExento(t *testing.T) {}\n",
		"internal/otro/metodo_test.go":    "package otro\n\ntype S struct{}\n\nfunc (s *S) TestComoMetodo() {}\n",
		"internal/otro/helper.go":         "package otro\n\nfunc TestEnFicheroDeProduccion() {}\n",
		"vendor/lib/lib_test.go":          "package lib\n\nfunc TestEnVendor(t *testing.T) {}\n",
		"node_modules/x/x_test.go":        "package x\n\nfunc TestEnNodeModules(t *testing.T) {}\n",
		"internal/p/testdata/d_test.go":   "package d\n\nfunc TestEnTestdata(t *testing.T) {}\n",
		".git/hooks/h_test.go":            "package h\n\nfunc TestEnGit(t *testing.T) {}\n",
		"internal/roto/roto_test.go":      "package roto\n\nfunc TestRoto(t *testing.T {\n",
		"internal/lit/comentario_test.go": "package lit\n\n// func TestSoloEnComentario(t *testing.T) {}\nvar s = `func TestEnCadena(t *testing.T) {}`\n",
		"internal/pref/p_test.go":         "package pref\n\nfunc TestXY(t *testing.T) {}\n\nfunc Test(t *testing.T) {}\n\nfunc Test_Guion(t *testing.T) {}\n",
	})
	r := newResolver(t, root)

	cases := []struct {
		name string
		test string
		want bool
	}{
		{"declarada en un fichero de prueba", "TestCobraIVA", true},
		{"otra declarada en el mismo fichero", "TestCobroExento", true},
		{"inexistente", "TestInexistente", false},
		{"un método no es una prueba de nivel superior", "TestComoMetodo", false},
		{"un fichero que no es de prueba no cuenta", "TestEnFicheroDeProduccion", false},
		{"vendor se omite", "TestEnVendor", false},
		{"node_modules se omite", "TestEnNodeModules", false},
		{"testdata se omite", "TestEnTestdata", false},
		{".git se omite", "TestEnGit", false},
		{"un fichero que no compila se omite", "TestRoto", false},
		{"un comentario no declara nada", "TestSoloEnComentario", false},
		{"una cadena no declara nada", "TestEnCadena", false},
		{"subcadena de otro identificador", "TestX", false},
		{"identificador más largo que la referencia", "TestXY", true},
		{"solo Test sin sufijo", "Test", true},
		{"Test con guion bajo", "Test_Guion", true},
		{"minúscula tras Test no es un nombre de prueba", "Testimonial", false},
		{"sin el prefijo Test", "CobraIVA", false},
		{"vacío", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := r.ResolveGoTest(tc.test)
			if err != nil {
				t.Fatalf("error inesperado: %v", err)
			}
			if got != tc.want {
				t.Fatalf("ResolveGoTest(%q) = %v, quería %v", tc.test, got, tc.want)
			}
		})
	}
}

// TestResolveGoTestBuildsIndexOnce comprueba que el recorrido del árbol se hace
// una sola vez por validación (design D11).
func TestResolveGoTestBuildsIndexOnce(t *testing.T) {
	root := writeTree(t, map[string]string{
		"a_test.go": "package a\n\nfunc TestUno(t *testing.T) {}\n",
	})
	r := newResolver(t, root)
	if ok, err := r.ResolveGoTest("TestUno"); err != nil || !ok {
		t.Fatalf("TestUno debía resolver: %v %v", ok, err)
	}
	if err := os.WriteFile(filepath.Join(root, "b_test.go"), []byte("package a\n\nfunc TestDos(t *testing.T) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := r.ResolveGoTest("TestDos"); err != nil || ok {
		t.Fatalf("el índice se construyó una sola vez: TestDos, creada después, no debía verse (%v, %v)", ok, err)
	}
}

func TestNewTreeResolverRejectsMissingRoot(t *testing.T) {
	if _, err := NewTreeResolver(filepath.Join(t.TempDir(), "no-existe")); err == nil {
		t.Fatal("una raíz inexistente debía devolver error")
	}
}
