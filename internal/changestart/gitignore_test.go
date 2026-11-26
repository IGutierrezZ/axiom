package changestart

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// gitInTemp ejecuta git en dir con un entorno aislado (sin configuración global ni del sistema).
func gitInTemp(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// D-16 vía 1 (prov:74:18): el .gitignore versionado de este repositorio ignora context/, los temporales
// de su copia, el contexto archivado y el contexto ODD. Se prueba con `git check-ignore` en un
// repositorio temporal que lleva una copia del .gitignore real.
func TestRepoGitignoreIgnoresContext(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no está disponible")
	}
	_, file, _, _ := runtime.Caller(0)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", ".gitignore"))
	if err != nil {
		t.Fatalf("leer el .gitignore del repositorio: %v", err)
	}
	repo := t.TempDir()
	if out, err := gitInTemp(t, repo, "init", "-q"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	ignored := []string{
		"openspec/changes/x/context/work-item.md",
		"openspec/changes/x/context/attachments/tasks.md",
		"openspec/changes/x/.context-tmp/abc/work-item.md",
		"openspec/changes/archive/2026-11-26-x/context/work-item.json",
		"odd/context/x/work-item.md",
		"odd/context/.tmp/x/abc/work-item.md",
	}
	for _, rel := range ignored {
		if _, err := gitInTemp(t, repo, "check-ignore", "-q", "--", rel); err != nil {
			t.Errorf("git no ignora %s", rel)
		}
	}
	for _, rel := range []string{"openspec/changes/x/proposal.md", "openspec/changes/x/tasks.md", "odd/tasks/x.md"} {
		if _, err := gitInTemp(t, repo, "check-ignore", "-q", "--", rel); err == nil {
			t.Errorf("git ignora %s, que es documentación", rel)
		}
	}
}
