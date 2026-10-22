package gitcommon

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// design D-08 (identidad del proyecto): el checkout principal de un repositorio sale del git
// common dir sin lanzar procesos, y solo cuando ese directorio se llama `.git`.

func wantMainWorktree(t *testing.T, commonDir, want string) {
	t.Helper()
	main, ok := MainWorktreeFromFiles(commonDir)
	if !ok {
		t.Fatalf("MainWorktreeFromFiles(%q) no resolvió un checkout principal, se esperaba %q", commonDir, want)
	}
	if !pathidentity.SameDirectory(main, want) {
		t.Errorf("MainWorktreeFromFiles(%q) = %q, se esperaba el directorio %q", commonDir, main, want)
	}
}

func TestMainWorktreeFromFilesResolvesTheMainCheckoutOfARealRepository(t *testing.T) {
	requireGit(t)
	ctx := context.Background()

	t.Run("desde el propio checkout principal", func(t *testing.T) {
		repo := newRepository(t)
		common, err := CommonDir(ctx, repo)
		if err != nil {
			t.Fatal(err)
		}
		wantMainWorktree(t, common, repo)
	})

	t.Run("desde un worktree enlazado devuelve el checkout principal, no el enlazado", func(t *testing.T) {
		repo := newRepository(t)
		linked := filepath.Join(t.TempDir(), "enlazado")
		runGit(t, repo, "worktree", "add", "-q", "-b", "inc/x", linked, "HEAD")
		common, err := CommonDir(ctx, linked)
		if err != nil {
			t.Fatal(err)
		}
		wantMainWorktree(t, common, repo)
		if main, _ := MainWorktreeFromFiles(common); pathidentity.SameDirectory(main, linked) {
			t.Errorf("MainWorktreeFromFiles devolvió el worktree enlazado %q", main)
		}
	})

	t.Run("la ruta con segmentos redundantes o separador final se limpia", func(t *testing.T) {
		repo := newRepository(t)
		wantMainWorktree(t, filepath.Join(repo, ".git")+string(filepath.Separator), repo)
		wantMainWorktree(t, filepath.Join(repo, "sub", "..", ".git"), repo)
	})
}

func TestMainWorktreeFromFilesDoesNotResolveACommonDirThatIsNotCalledDotGit(t *testing.T) {
	requireGit(t)
	ctx := context.Background()

	t.Run("con --separate-git-dir", func(t *testing.T) {
		work := t.TempDir()
		separate := filepath.Join(t.TempDir(), "almacen")
		runGit(t, work, "init", "-q", "-b", "main", "--separate-git-dir", separate)
		common, err := CommonDir(ctx, work)
		if err != nil {
			t.Fatal(err)
		}
		if main, ok := MainWorktreeFromFiles(common); ok {
			t.Errorf("MainWorktreeFromFiles(%q) = (%q, true); un common dir que no se llama .git no se resuelve", common, main)
		}
	})

	t.Run("con un repositorio desnudo", func(t *testing.T) {
		bare := filepath.Join(t.TempDir(), "remoto.git")
		runGit(t, t.TempDir(), "init", "-q", "--bare", "-b", "main", bare)
		if main, ok := MainWorktreeFromFiles(bare); ok {
			t.Errorf("MainWorktreeFromFiles(%q) = (%q, true); un repositorio desnudo no tiene checkout principal", bare, main)
		}
	})

	t.Run("con una ruta vacía o sin padre", func(t *testing.T) {
		for _, input := range []string{"", ".", string(filepath.Separator)} {
			if main, ok := MainWorktreeFromFiles(input); ok {
				t.Errorf("MainWorktreeFromFiles(%q) = (%q, true), se esperaba que no se resolviera", input, main)
			}
		}
	})
}
