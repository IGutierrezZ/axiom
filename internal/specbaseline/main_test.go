package specbaseline

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// scratchRoot es la carpeta temporal de todo el binario de pruebas: aloja el HOME
// aislado de git y las plantillas de repositorio. TestMain la crea antes de la
// primera prueba y la borra al terminar.
var scratchRoot string

// TestMain deja a los procesos git de las pruebas sin acceso a la configuración del
// usuario, con un autor fijo y sin red (D-15, design §8). Se fija una sola vez para
// todo el binario, en lugar de con t.Setenv en cada prueba: t.Setenv impide
// t.Parallel, y estas variables son comunes a todas las pruebas con git. Las pocas
// pruebas que necesitan otro entorno (las variables que redirigen git) siguen con
// t.Setenv y, por tanto, sin t.Parallel.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	root, err := os.MkdirTemp("", "specbaseline-pruebas-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "specbaseline: crear la carpeta temporal de las pruebas: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(root) }()
	scratchRoot = root

	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "specbaseline: crear el HOME aislado: %v\n", err)
		return 1
	}
	for key, value := range map[string]string{
		"HOME":                home,
		"USERPROFILE":         home,
		"GIT_CONFIG_GLOBAL":   filepath.Join(home, "gitconfig"),
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME":     "Autor de Prueba",
		"GIT_AUTHOR_EMAIL":    "autor@ejemplo.invalid",
		"GIT_COMMITTER_NAME":  "Autor de Prueba",
		"GIT_COMMITTER_EMAIL": "autor@ejemplo.invalid",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		if err := os.Setenv(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "specbaseline: fijar %s: %v\n", key, err)
			return 1
		}
	}
	return m.Run()
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
