package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Ayudantes de git y de línea de comando compartidos por las pruebas de la CLI.
// Antes vivían en las pruebas de `axiom sdd integrate`, retirado con la baseline
// ajena (REQ-30.25); los demás verbos los siguen usando.

// isolateIntegrateGit aísla git del entorno del usuario: HOME temporal, sin
// configuración global ni del sistema y con autor fijo.
func isolateIntegrateGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("la prueba ejecuta git: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está disponible: %v", err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Autor de Prueba")
	t.Setenv("GIT_AUTHOR_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Autor de Prueba")
	t.Setenv("GIT_COMMITTER_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
}

// integrateGit ejecuta git en dir y devuelve su salida sin el salto final.
func integrateGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
	return strings.TrimRight(string(output), "\r\n")
}

// integrateInitRepo inicializa un repositorio con la rama main.
func integrateInitRepo(t *testing.T, dir string) {
	t.Helper()
	integrateGit(t, dir, "init", "-q")
	integrateGit(t, dir, "symbolic-ref", "HEAD", "refs/heads/main")
	integrateGit(t, dir, "config", "core.autocrlf", "false")
}

// integrateCommit confirma todo el árbol de dir con message.
func integrateCommit(t *testing.T, dir, message string) {
	t.Helper()
	integrateGit(t, dir, "add", "-A")
	integrateGit(t, dir, "commit", "-q", "-m", message)
}

// splitCommandLine separa una línea de comando con las comillas dobles que emite
// pathquote: las comillas agrupan y no hay escapes.
func splitCommandLine(line string) []string {
	var words []string
	var current strings.Builder
	inQuotes, started := false, false
	for _, r := range line {
		switch {
		case r == '"':
			inQuotes, started = !inQuotes, true
		case r == ' ' && !inQuotes:
			if started {
				words = append(words, current.String())
				current.Reset()
				started = false
			}
		default:
			current.WriteRune(r)
			started = true
		}
	}
	if started {
		words = append(words, current.String())
	}
	return words
}

func TestSplitCommandLine(t *testing.T) {
	got := splitCommandLine(`axiom archive context prepare --path "C:\mi carpeta\repo" --change inc-1`)
	want := []string{"axiom", "archive", "context", "prepare", "--path", `C:\mi carpeta\repo`, "--change", "inc-1"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("palabras = %q, quería %q", got, want)
	}
}
