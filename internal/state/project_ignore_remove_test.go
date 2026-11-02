package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RemoveProjectStateIgnored deshace lo que añadió EnsureProjectStateIgnored y deja
// el contenido previo del fichero.
func TestRemoveProjectStateIgnoredRestoresTheExcludeFile(t *testing.T) {
	gitAvailable(t)
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	exclude := filepath.Join(root, ".git", "info", "exclude")
	if err := os.WriteFile(exclude, []byte("*.secreto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureProjectStateIgnored(root); err != nil {
		t.Fatal(err)
	}

	path, removed, err := RemoveProjectStateIgnored(root)
	if err != nil {
		t.Fatalf("RemoveProjectStateIgnored() = %v", err)
	}
	if path != exclude || len(removed) != 2 {
		t.Fatalf("path=%q removed=%v, want %q y las dos líneas", path, removed, exclude)
	}
	if got := string(readBytes(t, exclude)); got != "*.secreto\n" {
		t.Errorf("info/exclude = %q, want el contenido previo", got)
	}
	if gitIgnored(t, root, ".axiom/state.json") {
		t.Error("git sigue ignorando el estado tras retirarlo")
	}

	// Segunda llamada: nada que retirar.
	path, removed, err = RemoveProjectStateIgnored(root)
	if err != nil || path != "" || len(removed) != 0 {
		t.Fatalf("la segunda llamada debe ser un no-op: path=%q removed=%v err=%v", path, removed, err)
	}
}

// La cabecera se queda mientras otro proyecto del mismo repositorio use sus patrones.
func TestRemoveProjectStateIgnoredKeepsTheEntriesOfOtherProjectsInTheRepository(t *testing.T) {
	gitAvailable(t)
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	first := filepath.Join(repo, "servicios", "api")
	second := filepath.Join(repo, "servicios", "web")
	for _, dir := range []string{first, second} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := EnsureProjectStateIgnored(dir); err != nil {
			t.Fatal(err)
		}
	}

	if _, _, err := RemoveProjectStateIgnored(first); err != nil {
		t.Fatal(err)
	}

	if gitIgnored(t, repo, "servicios/api/.axiom/state.json") {
		t.Error("el estado del primer proyecto debe dejar de ignorarse")
	}
	if !gitIgnored(t, repo, "servicios/web/.axiom/state.json") {
		t.Error("el estado del segundo proyecto debe seguir ignorado")
	}
	if got := string(readBytes(t, filepath.Join(repo, ".git", "info", "exclude"))); !strings.Contains(got, excludeHeader) {
		t.Errorf("la cabecera debe quedarse mientras queden patrones de Axiom:\n%s", got)
	}

	if _, _, err := RemoveProjectStateIgnored(second); err != nil {
		t.Fatal(err)
	}
	if got := string(readBytes(t, filepath.Join(repo, ".git", "info", "exclude"))); strings.Contains(got, excludeHeader) || strings.Contains(got, ".axiom") {
		t.Errorf("sin proyectos que los usen, debe desaparecer todo lo de Axiom:\n%s", got)
	}
}

func TestRemoveProjectStateIgnoredOutsideGitDoesNothing(t *testing.T) {
	path, removed, err := RemoveProjectStateIgnored(t.TempDir())
	if err != nil || path != "" || len(removed) != 0 {
		t.Fatalf("fuera de git no hay nada que retirar: path=%q removed=%v err=%v", path, removed, err)
	}
}
