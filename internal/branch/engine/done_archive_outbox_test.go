package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// RF-INT-22 (sin cambiar RF-BRA-09 ni RF-BRA-18): la cola de integraciones de la carpeta archivada es
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

// RF-BRA-18: el cerrojo vacío que deja `gate record` (`.gates.lock`) es estado local sin datos; no
// cuenta como árbol sucio, así que `done` retira la rama de archivado aunque siga ahí. Cualquier
// otro fichero sin confirmar sigue bloqueando, y el rechazo no nombra el cerrojo.
func TestDoneOnTheArchiveBranchIgnoresTheLocalGatesLock(t *testing.T) {
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
	archived := filepath.Join(archive, "openspec", "changes", "archive", "2026-11-22-x")
	writeFile(t, filepath.Join(archived, "proposal.md"), "propuesta\n")
	runGit(t, archive, "add", "-A")
	commit(t, archive, "archivado")
	mergeInto(t, repo, "feat/x-archive")
	writeFile(t, filepath.Join(archived, ".gates.lock"), "")
	other := filepath.Join(archived, "notas.md")
	writeFile(t, other, "sin confirmar\n")

	refused := doneOf(t, doneRequest(repo))
	kept := branchByName(t, onlyRepository(t, refused), "feat/x-archive")
	wantOutcome(t, kept, branch.OutcomeKept, "", branch.CodeDirtyWorktree)
	if !strings.Contains(kept.Message, "notas.md") || strings.Contains(kept.Message, ".gates.lock") {
		t.Errorf("el rechazo nombra solo el fichero con datos: %q", kept.Message)
	}

	if err := os.Remove(other); err != nil {
		t.Fatal(err)
	}
	result := doneOf(t, doneRequest(repo))
	wantReady(t, result)
	wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x-archive"), branch.OutcomeRemoved, branch.LayerAncestry, "")
	if branchExists(t, repo, "feat/x-archive") {
		t.Fatal("la rama de archivado se elimina aunque quede el cerrojo local")
	}
	if _, err := os.Stat(archive); !os.IsNotExist(err) {
		t.Fatalf("el worktree de archivado se retira del disco: %v", err)
	}
}
