package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// `done` y la promoción de ramas (C-02-a, REQ-25.49, REQ-25.16, REQ-25.17, REQ-25.50, design D7): en
// modo `branch` la rama del cambio solo se elimina cuando la tiene `master`, el destino final; la
// evidencia `pr_merged` acredita la PR hacia `develop` y nunca la segunda. En `release` y en las
// demás ramas el destino sigue siendo la integración. Con git real y sin red.

// wantKept comprueba que `done` conservó feat/x con `merge_not_confirmed` contra el destino dado.
func wantKept(t *testing.T, repo string, result branch.Result, target string) {
	t.Helper()
	wantDoneStatus(t, result, branch.StatusRefused, branch.CodeMergeNotConfirmed)
	wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeKept, "", branch.CodeMergeNotConfirmed)
	if result.Details["target"] != target {
		t.Errorf("destino del rechazo = %q, se esperaba %q (%v)", result.Details["target"], target, result.Details)
	}
	if !branchExists(t, repo, "feat/x") {
		t.Error("sin fusión confirmada la rama se conserva")
	}
}

func TestDonePromotionBranchModeConfirmsAgainstTheMainBranch(t *testing.T) {
	t.Run("un merge commit solo en develop no basta", func(t *testing.T) {
		requireMergeTree(t)
		repo, _, _ := twoStageProject(t, "")
		changeFromMaster(t, repo)
		mergeCommitInto(t, repo, "develop")

		result := doneOf(t, doneRequest(repo))

		wantKept(t, repo, result, "master")
	})

	t.Run("fusionada en master se elimina por ascendencia", func(t *testing.T) {
		repo, _, _ := twoStageProject(t, "")
		changeFromMaster(t, repo)
		mergeCommitInto(t, repo, "develop")
		mergeCommitInto(t, repo, "master")
		seen := observe(t)

		result := doneOf(t, doneRequest(repo))

		wantReady(t, result)
		wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeRemoved, branch.LayerAncestry, "")
		if branchExists(t, repo, "feat/x") {
			t.Error("fusionada en master la rama se elimina")
		}
		wantNoRemote(t, *seen)
	})

	t.Run("un squash real en las dos etapas se confirma por árbol equivalente", func(t *testing.T) {
		requireMergeTree(t)
		repo, _, _ := twoStageProject(t, "")
		changeFromMaster(t, repo)
		squashInto(t, repo, "develop")
		squashInto(t, repo, "master")

		result := doneOf(t, doneRequest(repo))

		wantReady(t, result)
		wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeRemoved, branch.LayerEquivalentTree, "")
	})

	t.Run("un squash solo en develop se conserva", func(t *testing.T) {
		requireMergeTree(t)
		repo, _, _ := twoStageProject(t, "")
		changeFromMaster(t, repo)
		squashInto(t, repo, "develop")

		wantKept(t, repo, doneOf(t, doneRequest(repo)), "master")
	})
}

// REQ-25.49: `pr_merged` verificada en `gates.yaml` acredita la PR hacia la integración, no la
// segunda: en modo `branch` no confirma sin la fusión en `master`; `--force` deja la rama por
// `forced` con su punta.
func TestDonePromotionBranchModeNeverConfirmsByPRMergedAndForceStillWorks(t *testing.T) {
	repo, _, _ := twoStageProject(t, "")
	mergeTreeSupport(t, false)
	changeFromMaster(t, repo)
	mergeCommitInto(t, repo, "develop")
	recordIntegration(t, filepath.Join(repo, "openspec", "changes", "x"), kickoff.DecisionApproved, true)

	kept := doneOf(t, doneRequest(repo))

	wantKept(t, repo, kept, "master")
	tip := runGit(t, repo, "rev-parse", "feat/x")

	request := doneRequest(repo)
	request.Force = true
	forced := doneOf(t, request)

	wantReady(t, forced)
	described := branchByName(t, onlyRepository(t, forced), "feat/x")
	wantOutcome(t, described, branch.OutcomeRemoved, branch.LayerForced, "")
	if described.TipSha != tip {
		t.Errorf("--force deja la punta %q, se esperaba %q", described.TipSha, tip)
	}
}

// REQ-25.49: en `release` el destino sigue siendo la integración y la capa `pr_merged` sigue
// valiendo.
func TestDonePromotionReleaseModeConfirmsAgainstTheIntegration(t *testing.T) {
	t.Run("fusionada en develop se elimina", func(t *testing.T) {
		repo, _, _ := twoStageProject(t, "  promotion: release\n")
		changeFromMaster(t, repo)
		mergeCommitInto(t, repo, "develop")

		result := doneOf(t, doneRequest(repo))

		wantReady(t, result)
		wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeRemoved, branch.LayerAncestry, "")
	})

	t.Run("sin fusionar el destino es develop y pr_merged confirma", func(t *testing.T) {
		repo, _, _ := twoStageProject(t, "  promotion: release\n")
		mergeTreeSupport(t, false)
		changeFromMaster(t, repo)
		squashInto(t, repo, "develop")
		recordIntegration(t, filepath.Join(repo, "openspec", "changes", "x"), kickoff.DecisionApproved, true)

		result := doneOf(t, doneRequest(repo))

		wantReady(t, result)
		wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeRemoved, branch.LayerPRMerged, "")
	})
}

// REQ-25.49 y REQ-25.50: la rama de archivado sigue confirmándose contra la integración, también en
// modo `branch`, y sin la capa `pr_merged`.
func TestDonePromotionArchiveBranchStillConfirmsAgainstTheIntegration(t *testing.T) {
	repo, _, _ := twoStageProject(t, "")
	runGit(t, repo, "switch", "-q", "-c", "feat/x-archive", "develop")
	commitFile(t, repo, "archivado.txt", "archivado")
	runGit(t, repo, "switch", "-q", "develop")
	mergeInto(t, repo, "feat/x-archive")

	result := doneOf(t, doneRequest(repo))

	wantReady(t, result)
	wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x-archive"), branch.OutcomeRemoved, branch.LayerAncestry, "")
	if branchExists(t, repo, "feat/x-archive") {
		t.Error("la rama de archivado fusionada en develop se elimina")
	}
}

// REQ-25.49: con el registro borrado y el modo `ask` sin respuesta no se sabe cómo se promociona:
// se toma el destino más exigente, `master`, y la rama se conserva aunque esté en `develop`.
func TestDonePromotionAskPendingWithoutRecordKeepsTheBranchUntilMaster(t *testing.T) {
	repo, _, _ := twoStageProject(t, "  promotion: ask\n")
	changeFromMaster(t, repo)
	mergeCommitInto(t, repo, "develop")

	kept := doneOf(t, doneRequest(repo))

	wantKept(t, repo, kept, "master")

	mergeCommitInto(t, repo, "master")
	removed := doneOf(t, doneRequest(repo))
	wantReady(t, removed)
	wantOutcome(t, branchByName(t, onlyRepository(t, removed), "feat/x"), branch.OutcomeRemoved, branch.LayerAncestry, "")
}

// REQ-25.49 y REQ-25.17: `done` ejecutado desde el worktree del propio cambio resuelve el modo del
// registro y confirma contra `master`: retira el worktree y elimina la rama.
func TestDonePromotionFromTheChangeWorktreeConfirmsAgainstTheMainBranch(t *testing.T) {
	repo, _, _ := twoStageProject(t, "")
	changeFromMaster(t, repo)
	mergeCommitInto(t, repo, "develop")
	mergeCommitInto(t, repo, "master")
	worktree := filepath.Join(t.TempDir(), "x")
	runGit(t, repo, "worktree", "add", "-q", worktree, "feat/x")
	record := registeredRecord(repo)
	record.Repositories[0].Promotion = branch.PromotionBranch
	record.Repositories[0].PromotionSource = branch.PromotionSourceDefault
	record.Repositories[0].Branches = []branch.RecordBranch{{Name: "feat/x", Kind: branch.KindChange, Worktree: worktree}}
	registry := writeRegistry(t, commonDir(repo), record)

	result := doneOf(t, doneRequest(worktree))

	wantReady(t, result)
	wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeRemoved, branch.LayerAncestry, "")
	if _, err := os.Stat(worktree); err == nil {
		t.Error("el worktree del cambio se retira")
	}
	if _, err := os.Stat(registry); err == nil {
		t.Error("sin ramas el registro se borra")
	}
}
