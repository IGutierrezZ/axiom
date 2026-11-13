package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/gitcli"
)

// folderWriter registra las llamadas a RemoveEmptyDir y devuelve err; el resto lo hace git real.
type folderWriter struct {
	gitWriter
	removed []string
	err     error
}

func (f *folderWriter) RemoveEmptyDir(_ context.Context, path string) error {
	f.removed = append(f.removed, path)
	return f.err
}

func doneWithFolderWriter(t *testing.T, repo string, err error) (*folderWriter, branch.Result) {
	t.Helper()
	writer := &folderWriter{gitWriter: gitcli.New(), err: err}
	engine := Default()
	engine.writer = writer
	return writer, engine.Done(context.Background(), doneRequest(repo))
}

// Tras retirar el worktree, `done` pide retirar su carpeta contenedora `<raíz>/<cambio>` cuando
// el nombre coincide con el que planifica el motor.
func TestDoneRemovesTheEmptyFolderThatHeldTheWorktree(t *testing.T) {
	repo := embeddedCoreProject(t)
	wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"}))
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	mergeInto(t, repo, "feat/x")

	writer, result := doneWithFolderWriter(t, repo, nil)

	wantReady(t, result)
	if want := filepath.Dir(worktree); len(writer.removed) != 1 || !Default().samePlace(writer.removed[0], want) {
		t.Errorf("RemoveEmptyDir = %v, se esperaba solo %s", writer.removed, want)
	}
}

// Un worktree colocado a mano en una carpeta con otro nombre no se toca, y un fallo del retiro de
// la carpeta no cambia el resultado: solo añade un aviso.
func TestDoneLeavesAFolderWithAnotherNameAndIgnoresAFailedFolderRemoval(t *testing.T) {
	repo := embeddedCoreProject(t)
	wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"}))
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	manual := filepath.Join(t.TempDir(), "colocado-a-mano", filepath.Base(repo))
	if err := os.MkdirAll(filepath.Dir(manual), 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "worktree", "move", worktree, manual)
	mergeInto(t, repo, "feat/x")

	writer, result := doneWithFolderWriter(t, repo, nil)

	wantReady(t, result)
	if len(writer.removed) != 0 {
		t.Errorf("RemoveEmptyDir = %v, una carpeta con otro nombre no se toca", writer.removed)
	}
	if _, err := os.Stat(filepath.Dir(manual)); err != nil {
		t.Errorf("la carpeta colocada a mano debía seguir ahí: %v", err)
	}
}

func TestDoneOutcomeIsUnchangedWhenTheFolderCannotBeRemoved(t *testing.T) {
	for name, tc := range map[string]struct {
		err         error
		wantWarning bool
	}{
		"no vacía":    {err: gitcli.ErrDirNotEmpty},
		"inexistente": {err: os.ErrNotExist},
		"otro fallo":  {err: errors.New("acceso denegado"), wantWarning: true},
	} {
		t.Run(name, func(t *testing.T) {
			repo := embeddedCoreProject(t)
			wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"}))
			mergeInto(t, repo, "feat/x")

			writer, result := doneWithFolderWriter(t, repo, tc.err)

			wantReady(t, result)
			wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeRemoved, branch.LayerAncestry, "")
			if len(writer.removed) != 1 {
				t.Errorf("RemoveEmptyDir se llamó %d veces, se esperaba 1", len(writer.removed))
			}
			if got := len(result.Warnings) > 0; got != tc.wantWarning {
				t.Errorf("avisos = %v, ¿aviso esperado? %v", result.Warnings, tc.wantWarning)
			}
			if branchExists(t, repo, "feat/x") {
				t.Error("la rama debía eliminarse igualmente")
			}
		})
	}
}
