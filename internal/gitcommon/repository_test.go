package gitcommon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// RepositoryRoot da la raíz canónica tanto desde la raíz como desde una
// subcarpeta, y fuera de git conserva ErrNotGitRepository.
func TestRepositoryRootFromRootSubfolderAndOutside(t *testing.T) {
	requireGit(t)
	ctx := context.Background()
	main := newRepository(t)
	subfolder := mkdirAll(t, filepath.Join(main, "app", "web"))

	for _, dir := range []string{main, subfolder} {
		root, err := RepositoryRoot(ctx, dir)
		if err != nil {
			t.Fatalf("RepositoryRoot(%q): %v", dir, err)
		}
		wantSameDirectory(t, "RepositoryRoot", root, main)
		if resolved, err := filepath.EvalSymlinks(root); err != nil || resolved != root {
			t.Errorf("RepositoryRoot(%q) = %q, no es canónica (EvalSymlinks = %q, %v)", dir, root, resolved, err)
		}
	}

	if _, err := RepositoryRoot(ctx, ""); err == nil {
		t.Error("RepositoryRoot(\"\") no devolvió error")
	}
	outside := t.TempDir()
	if insideGit(outside) {
		t.Skipf("el directorio temporal %s está dentro de un repositorio git", outside)
	}
	if _, err := RepositoryRoot(ctx, outside); !errors.Is(err, ErrNotGitRepository) {
		t.Errorf("RepositoryRoot fuera de git = %v, se esperaba ErrNotGitRepository", err)
	}
}

// IsAncestor distingue «es antecesor» (código 0) de «no lo es» (código 1) y
// trata cualquier otro código como error.
func TestIsAncestorAnswersYesNoAndFailsOnUnknownRevision(t *testing.T) {
	requireGit(t)
	ctx := context.Background()
	repo := newRepository(t)
	first := gitOutput(t, repo, nil, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(repo, "segundo.txt"), []byte("2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "--", "segundo.txt")
	runGit(t, repo, "commit", "-q", "-m", "segundo")
	second := gitOutput(t, repo, nil, "rev-parse", "HEAD")

	if ok, err := IsAncestor(ctx, repo, first, second); err != nil || !ok {
		t.Errorf("IsAncestor(primero, segundo) = %v, %v; se esperaba true", ok, err)
	}
	if ok, err := IsAncestor(ctx, repo, second, first); err != nil || ok {
		t.Errorf("IsAncestor(segundo, primero) = %v, %v; se esperaba false", ok, err)
	}
	if _, err := IsAncestor(ctx, repo, "0000000000000000000000000000000000000001", second); err == nil {
		t.Error("IsAncestor con una revisión inexistente no devolvió error")
	}
}
