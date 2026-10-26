package branch

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// REQ-25.2 (G-4) y design §4.3: lo que `done` hizo con cada rama es un token estable en inglés con
// su etiqueta en castellano, y el JSON de una rama que `status` describe no gana ningún campo de
// `done`.
func TestDoneOutcomeTokensHaveSpanishLabelsAndStayOutOfStatusBranches(t *testing.T) {
	for token, want := range map[DoneOutcome]string{OutcomeRemoved: "eliminada", OutcomeKept: "conservada", OutcomeAbsent: "ya no existía"} {
		if got := Label(string(token)); got != want {
			t.Errorf("Label(%q) = %q, se esperaba %q", token, got, want)
		}
	}

	var out bytes.Buffer
	result := NewResult("status")
	result.Repositories = []RepositoryResult{{Branches: []BranchResult{{Name: "feat/x", Kind: KindChange, KindLabel: "rama del cambio", PRTarget: "main"}}}}
	if err := result.WriteJSON(&out); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"outcome", "confirmedBy", "discarded", "tipSha", "continuation"} {
		if strings.Contains(out.String(), `"`+field) {
			t.Errorf("una rama de `status` no lleva el campo %q de `done`:\n%s", field, out.String())
		}
	}
}

// design D-09: `partial` lleva el código y la continuación de la primera rama conservada, sale con 1
// y, en texto, la salida describe cada rama en stdout y deja el error en stderr.
func TestPartialDoneResultExitsWithOneAndWritesEveryBranchBeforeTheError(t *testing.T) {
	removed := BranchResult{Name: "feat/x-core", Kind: KindRole, KindLabel: Label(string(KindRole)), Worktree: `C:\wt\x-core\repo`}
	removed.SetOutcome(OutcomeRemoved)
	removed.SetConfirmedBy(LayerForced)
	removed.TipSha = "0123456789abcdef0123456789abcdef01234567"
	removed.Discarded = []string{"a.txt", "b.txt"}
	kept := BranchResult{Name: "feat/x", Kind: KindChange, KindLabel: Label(string(KindChange)), Exists: true}
	kept.Keep(&Refusal{Code: CodeMergeNotConfirmed, Message: "no está fusionada", Continuation: "axiom branch done --change x"})

	result := NewResult("done")
	result.Change, result.Lane = "x", LaneSDD
	result.Repositories = []RepositoryResult{{Name: "repo", RoleLabel: "repositorio con rama de cambio", GitRoot: `C:\repo`, CommonDir: `C:\repo\.git`, IntegrationBranch: "main", Branches: []BranchResult{removed, kept}}}
	partial := result.Partial(&Refusal{Code: CodeMergeNotConfirmed, Message: "no está fusionada", Continuation: "axiom branch done --change x"})

	if partial.Status != StatusPartial || partial.StatusLabel != "parcial" || partial.Code != CodeMergeNotConfirmed || partial.ExitCode() != ExitRefused {
		t.Fatalf("status %q (%q), código %q, salida %d", partial.Status, partial.StatusLabel, partial.Code, partial.ExitCode())
	}
	var stdout, stderr bytes.Buffer
	partial.WriteText(&stdout, &stderr)
	for _, want := range []string{
		"Estado: parcial",
		"feat/x-core (rama de rol): eliminada, fusión confirmada por forzado, worktree " + `C:\wt\x-core\repo`,
		"descartados 2 cambios sin confirmar (a.txt, b.txt)", "punta 0123456789ab para recuperarla",
		"feat/x (rama del cambio): conservada, merge_not_confirmed: no está fusionada",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout debe contener %q:\n%s", want, stdout.String())
		}
	}
	if want := "Error [merge_not_confirmed]: no está fusionada\nContinuación: axiom branch done --change x\n"; stderr.String() != want {
		t.Errorf("stderr = %q, se esperaba %q", stderr.String(), want)
	}
}

// Regla 11 de tasks.md: toda continuación de `done` es una orden ejecutable con las rutas citadas y
// el carril cuando no es el de por defecto; `--force` solo se ofrece donde puede resolver el
// rechazo (no con el árbol sucio de `in_place`).
func TestDoneRefusalsNameExecutableContinuations(t *testing.T) {
	repo := RepoPlan{Name: "repo", GitRoot: `C:\mi repo`, IntegrationBranch: "main"}
	planned := BranchPlan{Name: "feat/x", Kind: KindChange, PRTarget: "main"}
	cwd := `C:\mi repo`

	if got, want := DoneInvocation("x", cwd, LaneSDD), "axiom branch done --change x --cwd "+pathquote.Quote(cwd); got != want {
		t.Errorf("DoneInvocation = %q, se esperaba %q", got, want)
	}
	if got := DoneInvocation("x", cwd, LaneODD, "--force"); !strings.Contains(got, "--lane odd") || !strings.HasSuffix(got, " --force") {
		t.Errorf("DoneInvocation con carril odd y --force = %q", got)
	}

	notMerged := MergeNotConfirmed("x", cwd, LaneSDD, repo, planned, []string{"origin/main", "main"})
	if notMerged.Code != CodeMergeNotConfirmed || !strings.Contains(notMerged.Message, "origin/main ni en main") ||
		!strings.Contains(notMerged.Continuation, DoneInvocation("x", cwd, LaneSDD, "--force")) || !strings.Contains(notMerged.Continuation, "log --oneline main..feat/x") {
		t.Errorf("merge_not_confirmed = %+v", notMerged)
	}
	if noTarget := MergeNotConfirmed("x", cwd, LaneSDD, repo, planned, nil); !strings.Contains(noTarget.Message, "no existe main") {
		t.Errorf("sin destino, el mensaje debe decirlo: %q", noTarget.Message)
	}

	paths := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"}
	dirty := DoneDirtyWorktree("x", cwd, LaneSDD, "repo", "feat/x", `C:\wt\x\repo`, paths, false)
	if dirty.Code != CodeDirtyWorktree || !strings.Contains(dirty.Message, "y 2 más") || !strings.Contains(dirty.Continuation, "--force") || strings.Count(dirty.Details["paths"], "\n") != len(paths)-1 {
		t.Errorf("dirty_worktree = %+v", dirty)
	}
	if inPlace := DoneDirtyWorktree("x", cwd, LaneSDD, "repo", "feat/x", cwd, paths, true); strings.Contains(inPlace.Continuation, DoneInvocation("x", cwd, LaneSDD, "--force")) {
		t.Errorf("en in_place no se ofrece --force: %q", inPlace.Continuation)
	}

	failed := DoneGitFailed("x", cwd, LaneSDD, cwd, "retirar el worktree", "comprueba el directorio", errors.New("denegado"))
	if failed.Code != CodeGitFailed || !strings.Contains(failed.Message, "retirar el worktree") || !strings.Contains(failed.Message, "denegado") ||
		!strings.HasPrefix(failed.Continuation, "comprueba el directorio y repite `"+DoneInvocation("x", cwd, LaneSDD)+"`") {
		t.Errorf("git_failed = %+v", failed)
	}
	fetch := DoneFetchFailed("x", cwd, LaneSDD, repo, "origin", errors.New("sin red"))
	if fetch.Code != CodeFetchFailed || !strings.Contains(fetch.Message, "no se ha eliminado ninguna rama") || !strings.Contains(fetch.Continuation, DoneInvocation("x", cwd, LaneSDD, "--fetch")) {
		t.Errorf("fetch_failed = %+v", fetch)
	}
	if elsewhere := DoneCheckedOutInMainCheckout("x", cwd, LaneSDD, repo, "feat/x", cwd); elsewhere.Code != CodeBranchCheckedOutElsewhere || !strings.Contains(elsewhere.Continuation, "switch main") {
		t.Errorf("branch_checked_out_elsewhere = %+v", elsewhere)
	}
}
