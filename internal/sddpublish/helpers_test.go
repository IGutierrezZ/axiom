package sddpublish

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/testenv"
)

// Ayudas de las pruebas con git real: repositorios y remotos desnudos temporales, un entorno de
// git aislado (sin configuración global ni del sistema) y sin red. Las pruebas ejecutan git
// directamente, sin pasar por el adaptador, para que el oráculo no dependa de lo que se prueba.
//
// El entorno se fija una sola vez en TestMain (testenv.IsolateGit), de modo que las pruebas
// pueden marcarse con t.Parallel(). Siguen en serie las que fijan entorno propio (t.Setenv),
// sustituyen una variable global (pushTimeout) o instalan el observador global de git (observe).

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	_, cleanup, err := testenv.IsolateGit("sddpublish")
	if err != nil {
		fmt.Fprintf(os.Stderr, "sddpublish: %v\n", err)
		return 1
	}
	defer cleanup()
	// Las pruebas leen mensajes de git («non-fast-forward»): sin traducir.
	if err := os.Setenv("LC_ALL", "C"); err != nil {
		fmt.Fprintf(os.Stderr, "sddpublish: fijar LC_ALL: %v\n", err)
		return 1
	}
	return m.Run()
}

// isolateGit omite la prueba si git no está disponible. El entorno de git lo fijó TestMain.
func isolateGit(t *testing.T) {
	t.Helper()
	testenv.RequireGit(t)
}

// runGit ejecuta git en dir y devuelve su salida sin el salto final; falla la prueba si git falla.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	return testenv.Git(t, dir, args...)
}

// tryGit ejecuta git en dir y devuelve su salida combinada y el error.
func tryGit(dir string, args ...string) (string, error) {
	return testenv.TryGit(dir, args...)
}

func writeFile(t testing.TB, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Plantillas de los repositorios de las pruebas: se construyen una vez por binario y cada prueba
// recibe una copia, en lugar de repetir los procesos git de `init`, `commit` y `push`.
var (
	repoTemplate = testenv.NewRepoTemplate("repo", func(t testing.TB, dir string) {
		buildRepo(t, dir)
	})
	bareTemplate = testenv.NewRepoTemplate("bare", func(t testing.TB, dir string) {
		buildBare(t, dir)
	})
	repoWithRemoteTemplate = testenv.NewRepoTemplate("repo-remoto", func(t testing.TB, dir string) {
		repo, bare := filepath.Join(dir, "repo"), filepath.Join(dir, "bare")
		buildRepo(t, repo)
		buildBare(t, bare)
		testenv.Git(t, repo, "remote", "add", "origin", filepath.ToSlash(bare))
		testenv.Git(t, repo, "push", "-q", "origin", "main")
	})
)

// buildRepo crea en dir un repositorio con un commit inicial en main.
func buildRepo(t testing.TB, dir string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, "README.md"), "inicial\n")
	testenv.Git(t, dir, "init", "-q", "-b", "main")
	testenv.Git(t, dir, "config", "core.autocrlf", "false")
	testenv.Git(t, dir, "add", "-A")
	testenv.Git(t, dir, "commit", "-q", "-m", "inicial")
}

// buildBare crea en dir un remoto desnudo vacío.
func buildBare(t testing.TB, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	testenv.Git(t, dir, "init", "-q", "--bare", "-b", "main")
}

// newRepo devuelve un repositorio con un commit inicial en main.
func newRepo(t *testing.T) string {
	t.Helper()
	return repoTemplate.Copy(t)
}

// newBare devuelve un remoto desnudo local vacío.
func newBare(t *testing.T) string {
	t.Helper()
	return bareTemplate.Copy(t)
}

// newRepoWithRemote devuelve un repositorio con `origin` apuntando a un remoto desnudo que ya
// tiene main publicada.
func newRepoWithRemote(t *testing.T) (repo, bare string) {
	t.Helper()
	dir := repoWithRemoteTemplate.Copy(t)
	return filepath.Join(dir, "repo"), filepath.Join(dir, "bare")
}

// observe instala un observador de las invocaciones de git y lo retira al terminar la prueba.
// gitcommon.Observe es global: una prueba que observa no puede marcarse con t.Parallel().
func observe(t *testing.T) *[][]string {
	t.Helper()
	var seen [][]string
	gitcommon.Observe = func(dir string, args []string) {
		seen = append(seen, append([]string{dir}, args...))
	}
	t.Cleanup(func() { gitcommon.Observe = nil })
	return &seen
}
