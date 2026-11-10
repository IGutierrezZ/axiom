package sddpublish

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

// Ayudas de las pruebas con git real: repositorios y remotos desnudos temporales, un entorno de
// git aislado (sin configuración global ni del sistema) y sin red. Las pruebas ejecutan git
// directamente, sin pasar por el adaptador, para que el oráculo no dependa de lo que se prueba.

// isolateGit aísla git de la configuración del usuario y fija la identidad de las pruebas. Los
// hooks de ejemplo no existen porque `git init` los copia de una plantilla que se vacía.
func isolateGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no está disponible en el PATH")
	}
	home := t.TempDir()
	empty := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	template := filepath.Join(home, "template")
	if err := os.MkdirAll(template, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"HOME":                home,
		"USERPROFILE":         home,
		"XDG_CONFIG_HOME":     home,
		"GIT_CONFIG_GLOBAL":   empty,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_TEMPLATE_DIR":    template,
		"GIT_AUTHOR_NAME":     "Prueba",
		"GIT_AUTHOR_EMAIL":    "prueba@example.invalid",
		"GIT_COMMITTER_NAME":  "Prueba",
		"GIT_COMMITTER_EMAIL": "prueba@example.invalid",
		"GIT_TERMINAL_PROMPT": "0",
		"LC_ALL":              "C",
	} {
		t.Setenv(name, value)
	}
	for _, name := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE"} {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}
}

// runGit ejecuta git en dir y devuelve su salida sin el salto final; falla la prueba si git falla.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := tryGit(dir, args...)
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return out
}

// tryGit ejecuta git en dir y devuelve su salida combinada y el error.
func tryGit(dir string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	return strings.TrimRight(string(output), "\r\n"), err
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newRepo crea un repositorio con un commit inicial en main y devuelve su raíz.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "core.autocrlf", "false")
	writeFile(t, filepath.Join(dir, "README.md"), "inicial\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", "inicial")
	return dir
}

// newBare crea un remoto desnudo local vacío.
func newBare(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "--bare", "-b", "main")
	return dir
}

// newRepoWithRemote crea un repositorio con `origin` apuntando a un remoto desnudo que ya tiene
// main publicada.
func newRepoWithRemote(t *testing.T) (repo, bare string) {
	t.Helper()
	repo, bare = newRepo(t), newBare(t)
	runGit(t, repo, "remote", "add", "origin", bare)
	runGit(t, repo, "push", "-q", "origin", "main")
	return repo, bare
}

// observe instala un observador de las invocaciones de git y lo retira al terminar la prueba.
func observe(t *testing.T) *[][]string {
	t.Helper()
	var seen [][]string
	gitcommon.Observe = func(dir string, args []string) {
		seen = append(seen, append([]string{dir}, args...))
	}
	t.Cleanup(func() { gitcommon.Observe = nil })
	return &seen
}
