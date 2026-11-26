package techcontext

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/testenv"
)

// scratchRoot es la carpeta temporal de todo el binario de pruebas: aloja el HOME
// aislado de git y las plantillas de árbol.
var scratchRoot string

// TestMain deja a los procesos git de las pruebas sin acceso a la configuración
// del usuario, con un autor fijo y sin red. Se fija una vez para todo el binario
// de pruebas (testenv.IsolateGit), de modo que las pruebas con git puedan
// ejecutarse en paralelo. Siguen en serie las que cambian de directorio (t.Chdir)
// o sustituyen una variable global (now con fixedClock, renameFile).
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	root, cleanup, err := testenv.IsolateGit("techcontext")
	if err != nil {
		fmt.Fprintf(os.Stderr, "techcontext: %v\n", err)
		return 1
	}
	defer cleanup()
	scratchRoot = root
	return m.Run()
}

// byteVariant es una de las tres formas de bytes en las que se prueba cada
// documento: las variantes CRLF y CRLF con BOM se generan a partir de la versión
// LF, de modo que las pruebas no dependen de .gitattributes.
type byteVariant struct {
	name string
	bom  bool
	crlf bool
}

var byteVariants = []byteVariant{
	{name: "LF"},
	{name: "CRLF", crlf: true},
	{name: "CRLF+BOM", crlf: true, bom: true},
}

// apply convierte un texto en LF a la variante de bytes. Un texto que ya lleva
// BOM lo conserva en todas las variantes y no recibe otro.
func (v byteVariant) apply(lf string) []byte {
	data := []byte(lf)
	if v.crlf {
		data = bytes.ReplaceAll(data, []byte("\n"), []byte("\r\n"))
	}
	if v.bom && !bytes.HasPrefix(data, utf8BOM) {
		data = append(append([]byte{}, utf8BOM...), data...)
	}
	return data
}

// forEachVariant ejecuta fn como subprueba por cada variante de bytes.
func forEachVariant(t *testing.T, fn func(t *testing.T, v byteVariant)) {
	t.Helper()
	for _, v := range byteVariants {
		t.Run(v.name, func(t *testing.T) { fn(t, v) })
	}
}

// treeTemplate es un árbol de ficheros (repositorios git incluidos) que se
// construye una sola vez por ejecución y se copia a la carpeta de cada prueba, en
// lugar de repetir `git init`, `git add` y `git commit` en cada una. Cada prueba
// recibe su copia independiente: lo que haga en ella no alcanza a las demás.
type treeTemplate struct {
	name string
	once sync.Once
	dir  string
}

// materialize copia la plantilla en dst, construyéndola antes con build si es la
// primera vez. build recibe una carpeta vacía y la deja como el estado inicial de
// la prueba: es el mismo código que montaría el árbol en cada prueba.
func (tt *treeTemplate) materialize(t testing.TB, dst string, build func(t testing.TB, dir string)) {
	t.Helper()
	tt.once.Do(func() {
		dir := filepath.Join(scratchRoot, "plantillas", tt.name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		build(t, dir)
		tt.dir = dir // solo si build terminó: un Fatal en build deja la plantilla sin dir
	})
	if tt.dir == "" {
		t.Fatalf("la plantilla %q no se pudo construir", tt.name)
	}
	copyTree(t, tt.dir, dst)
}

// copyTree copia el árbol src bajo dst, ficheros ocultos y `.git` incluidos. Solo
// admite directorios y ficheros regulares: los árboles de plantilla no llevan
// enlaces simbólicos.
func copyTree(t testing.TB, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s no es un fichero regular", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Los objetos de git son de solo lectura: la copia debe poder borrarse y
		// reescribirse como cualquier otro fichero de la prueba.
		return os.WriteFile(target, data, info.Mode().Perm()|0o200)
	})
	if err != nil {
		t.Fatalf("copiar la plantilla %s a %s: %v", src, dst, err)
	}
}
