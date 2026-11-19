package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// REQ-43.31 (sin cambiar REQ-25.9 ni REQ-25.17): la cola de integraciones de la carpeta archivada es
// un fichero sin seguimiento en el árbol de la rama de archivado, así que `done` se niega con
// `dirty_worktree` nombrándola mientras esté pendiente y procede cuando se vacía (el fichero se
// borra al quedar sin operaciones).
func TestDoneOnTheArchiveBranchRefusesWithAPendingIntegrationsQueueAndProceedsOnceItIsEmpty(t *testing.T) {
	repo := embeddedCoreProject(t)
	wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"}))
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	writeFile(t, filepath.Join(worktree, "cambio.txt"), "trabajo del cambio\n")
	runGit(t, worktree, "add", "-A")
	commit(t, worktree, "trabajo del cambio")
	mergeInto(t, repo, "feat/x")
	runGit(t, repo, "branch", "feat/x-archive", "main")
	archive := archiveWorktree(t, repo)
	runGit(t, repo, "worktree", "add", "-q", archive, "feat/x-archive")
	// La carpeta archivada está versionada (como tras un publish --archive), así que git lista la cola sola.
	writeFile(t, filepath.Join(archive, "openspec", "changes", "archive", "2026-11-19-x", "proposal.md"), "propuesta\n")
	runGit(t, archive, "add", "-A")
	commit(t, archive, "archivado")
	mergeInto(t, repo, "feat/x-archive")
	queue := filepath.Join(archive, "openspec", "changes", "archive", "2026-11-19-x", ".integrations-outbox.yaml")
	writeFile(t, queue, "schema: axiom.integrations-outbox/v1\nchange: x\n")

	result := doneOf(t, doneRequest(repo))

	archived := branchByName(t, onlyRepository(t, result), "feat/x-archive")
	wantOutcome(t, archived, branch.OutcomeKept, "", branch.CodeDirtyWorktree)
	if !strings.Contains(archived.Message, ".integrations-outbox.yaml") {
		t.Errorf("el rechazo nombra la cola pendiente: %q", archived.Message)
	}
	if !branchExists(t, repo, "feat/x-archive") {
		t.Fatal("el rechazo conserva la rama de archivado")
	}
	if _, err := os.Stat(queue); err != nil {
		t.Fatalf("el rechazo no pierde la cola: %v", err)
	}

	// Con la cola vacía (borrada) done procede.
	if err := os.Remove(queue); err != nil {
		t.Fatal(err)
	}
	second := doneOf(t, doneRequest(repo))
	wantReady(t, second)
	wantOutcome(t, branchByName(t, onlyRepository(t, second), "feat/x-archive"), branch.OutcomeRemoved, branch.LayerAncestry, "")
	if branchExists(t, repo, "feat/x-archive") {
		t.Fatal("la rama de archivado se elimina cuando no queda cola")
	}
}
