package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

// REQ-29.9: `axiom sdd trace` deriva la traza del cambio con git y es de solo
// lectura. Los nombres de las pruebas empiezan por TestSDDTrace para entrar en el
// mismo patrón de `-run`.

// traceGit ejecuta git en dir con un entorno aislado de la configuración del
// desarrollador y devuelve la salida sin el salto de línea final.
func traceGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "commit.gpgsign=false", "-c", "core.autocrlf=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull,
		"GIT_AUTHOR_NAME=Prueba", "GIT_AUTHOR_EMAIL=prueba@example.test",
		"GIT_COMMITTER_NAME=Prueba", "GIT_COMMITTER_EMAIL=prueba@example.test")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// traceRepo crea un repositorio git con una confirmación inicial en `main`.
func traceRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	traceGit(t, dir, "init", "-q", "-b", "main")
	writeCLIFixture(t, filepath.Join(dir, "README.md"), "inicial\n")
	traceGit(t, dir, "add", "-A")
	traceGit(t, dir, "commit", "-q", "-m", "inicial")
	return dir
}

// observeGit anota cada invocación de git que pasa por gitcommon.Run, y la
// retira al terminar la prueba.
func observeGit(t *testing.T) *[][]string {
	t.Helper()
	var calls [][]string
	gitcommon.Observe = func(dir string, args []string) {
		calls = append(calls, append([]string{dir}, args...))
	}
	t.Cleanup(func() { gitcommon.Observe = nil })
	return &calls
}

// REQ-29.9 y design §8 (selección del repositorio git): los datos no seguros se
// rechazan antes de llamar a git, y git se ejecuta siempre en el repositorio de
// `--cwd`, nunca en el del proceso.
func TestSDDTraceRejectsUnsafeInput(t *testing.T) {
	repo := traceRepo(t)
	writeSDDAxiomYAML(t, repo, "")
	writeCLIFixture(t, filepath.Join(repo, "openspec", "changes", ".gitkeep"), "")

	t.Run("nombres de cambio y referencias no seguros", func(t *testing.T) {
		calls := observeGit(t)
		for _, change := range []string{"", "../x", "a:(x)", "x*y", "a..b", "-x", "a/b", `a\b`, "x y", "archive"} {
			var out bytes.Buffer
			if err := RunSDDTrace([]string{"--cwd", repo, "--change", change}, &out); err == nil {
				t.Errorf("--change %q debió rechazarse", change)
			}
		}
		for _, ref := range []string{"../x", "a..b", "-x", "a b", "x^", "x~1", "x:y", "x@{1}", "x\\y"} {
			var out bytes.Buffer
			if err := RunSDDTrace([]string{"--cwd", repo, "--change", "demo", "--integration-branch", ref}, &out); err == nil {
				t.Errorf("--integration-branch %q debió rechazarse", ref)
			}
		}
		if len(*calls) != 0 {
			t.Fatalf("se llamó a git antes de rechazar la entrada: %v", *calls)
		}
	})

	t.Run("git se ejecuta en el repositorio de --cwd y no en el del proceso", func(t *testing.T) {
		foreign := traceRepo(t)
		t.Chdir(foreign)
		calls := observeGit(t)

		var out bytes.Buffer
		if err := RunSDDTrace([]string{"--cwd", repo, "--change", "demo", "--json"}, &out); err != nil {
			t.Fatalf("RunSDDTrace() error = %v\n%s", err, out.String())
		}
		if len(*calls) == 0 {
			t.Fatal("la traza no llamó a git")
		}
		want, err := filepath.EvalSymlinks(repo)
		if err != nil {
			t.Fatal(err)
		}
		for _, call := range *calls {
			dir, evalErr := filepath.EvalSymlinks(call[0])
			relative, relErr := filepath.Rel(want, dir)
			if evalErr != nil || relErr != nil || !filepath.IsAbs(call[0]) || strings.HasPrefix(relative, "..") {
				t.Errorf("git se ejecutó en %q (git %v), se esperaba el repositorio de --cwd %q", call[0], call[1:], want)
			}
		}
	})
}
