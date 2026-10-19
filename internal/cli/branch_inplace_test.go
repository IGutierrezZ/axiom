package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// `axiom branch ensure --mode in_place` (INC-25, S4.7, REQ-25.8, REQ-25.9): la ayuda, el
// rechazo `dirty_worktree` con código de salida 1 y el cambio de rama sin worktree, sobre
// repositorios reales. El comportamiento del motor se prueba en internal/branch/engine.

func TestBranchEnsureHelpDocumentsInPlaceAndDirtyWorktree(t *testing.T) {
	code, stdout, stderr := runBranchCLI(t, "ensure", "--help")

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{"in_place", "git switch", "dirty_worktree", "árbol limpio", "ficheros sin\nseguimiento"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la ayuda de ensure debe documentar %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "todavía no está disponible") {
		t.Errorf("la ayuda ya no debe anunciar que in_place no está disponible:\n%s", stdout)
	}
}

// REQ-25.9 (harness S4.8): con el árbol sucio, `--mode in_place` sale con 1 y `dirty_worktree`
// sin modificar nada (rama, índice y árbol, y sin rama ni registro); en texto, el código y la
// continuación por stderr. Con el árbol limpio, sale con 0, hace `git switch` sin worktree,
// registra el modo y repetir da el mismo JSON.
func TestBranchEnsureInPlaceRefusesADirtyTreeAndSwitchesACleanOne(t *testing.T) {
	repo := newBranchRepository(t)
	loose := filepath.Join(repo, "suelto.txt")
	if err := os.WriteFile(loose, []byte("suelto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := branchGitOutput(t, repo, "status", "--porcelain=v1", "--untracked-files=all") + branchGitOutput(t, repo, "rev-parse", "HEAD")

	code, stdout, stderr := runBranchCLI(t, "ensure", "--change", "x", "--mode", "in_place", "--json", "--cwd", repo)

	if code != 1 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 1 y stderr vacío con --json\n%s", code, stderr, stdout)
	}
	refused := decodeBranchResult(t, stdout)
	if refused.Status != "refused" || refused.Code != "dirty_worktree" || !strings.Contains(refused.Message, "suelto.txt") || refused.Details["paths"] != "suelto.txt" {
		t.Errorf("status %q, code %q, mensaje %q, details %v", refused.Status, refused.Code, refused.Message, refused.Details)
	}
	for _, want := range []string{"axiom branch ensure --change x --mode in_place --cwd " + pathquote.Quote(repo), "--mode worktree"} {
		if !strings.Contains(refused.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", refused.Continuation, want)
		}
	}
	if len(refused.Repositories) != 0 || refused.Registry != nil {
		t.Errorf("un rechazo no describe repositorios ni registro: %+v", refused)
	}
	if got := branchGitOutput(t, repo, "branch", "--show-current"); got != "main" {
		t.Errorf("el repositorio está en %q, se esperaba main", got)
	}
	if after := branchGitOutput(t, repo, "status", "--porcelain=v1", "--untracked-files=all") + branchGitOutput(t, repo, "rev-parse", "HEAD"); after != before {
		t.Errorf("el rechazo cambió el repositorio:\n%s\n---\n%s", before, after)
	}
	if got := branchGitOutput(t, repo, "branch", "--list", "feat/x"); got != "" {
		t.Errorf("no debía crearse la rama, hay %q", got)
	}

	code, stdout, stderr = runBranchCLI(t, "ensure", "--change", "x", "--mode", "in_place", "--cwd", repo)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "dirty_worktree") || !strings.Contains(stderr, "Continuación:") || !strings.Contains(stderr, "suelto.txt") {
		t.Errorf("en texto: código %d, stdout %q, stderr %q; se esperaba 1, stdout vacío y el código con los ficheros y la continuación por stderr", code, stdout, stderr)
	}

	if err := os.Remove(loose); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runBranchCLI(t, "ensure", "--change", "x", "--mode", "in_place", "--json", "--cwd", repo)
	if code != 0 || stderr != "" {
		t.Fatalf("con el árbol limpio: código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
	}
	ready := decodeBranchResult(t, stdout)
	if ready.Status != "ready" || ready.Mode != "in_place" || ready.ModeSource != "flag" || ready.WorktreeRoot != "" {
		t.Errorf("status %q, modo %q, origen %q, raíz de worktrees %q; se esperaba ready en in_place con origen flag y sin raíz de worktrees", ready.Status, ready.Mode, ready.ModeSource, ready.WorktreeRoot)
	}
	if got := branchGitOutput(t, repo, "branch", "--show-current"); got != "feat/x" {
		t.Errorf("el repositorio está en %q, se esperaba feat/x", got)
	}
	if got := branchGitOutput(t, repo, "worktree", "list", "--porcelain"); strings.Count(got, "worktree ") != 1 {
		t.Errorf("in_place no crea ningún worktree:\n%s", got)
	}

	code, again, _ := runBranchCLI(t, "ensure", "--change", "x", "--mode", "in_place", "--json", "--cwd", repo)
	if code != 0 || again != stdout {
		t.Errorf("repetir debe dar el mismo JSON byte a byte: código %d\n%s\n---\n%s", code, stdout, again)
	}
}
