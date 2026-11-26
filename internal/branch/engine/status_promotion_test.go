package engine

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// `status` y la siguiente PR de la promoción de ramas (C-02-a, RF-BRA-30, RF-BRA-16, design D8): en
// modo `branch` la rama del cambio sale de `master` y su primera PR va a `develop`; tras confirmar
// con git local que está fusionada en `develop`, `prTarget` pasa a `master` y `promotionStage` a
// `main`. Con git real y sin red.

// changeFromMaster crea `feat/x` desde `master` con un cambio propio y devuelve el repositorio a
// `develop`, que es donde se integra la primera PR.
func changeFromMaster(t *testing.T, repo string) {
	t.Helper()
	runGit(t, repo, "switch", "-q", "-c", "feat/x", "master")
	commitFile(t, repo, "cambio.txt", "cambio de x")
	runGit(t, repo, "switch", "-q", "develop")
}

// squashInto integra `feat/x` con un squash en target y lo deja con un commit posterior, como tras
// el PR de un cambio; devuelve el repositorio a `develop`.
func squashInto(t *testing.T, repo, target string) {
	t.Helper()
	runGit(t, repo, "switch", "-q", target)
	runGit(t, repo, "merge", "--squash", "feat/x")
	commit(t, repo, "squash de x en "+target)
	commitFile(t, repo, "posterior-en-"+target+".txt", "avanza "+target)
	runGit(t, repo, "switch", "-q", "develop")
}

// mergeCommitInto integra `feat/x` con un merge commit en target; devuelve el repositorio a `develop`.
func mergeCommitInto(t *testing.T, repo, target string) {
	t.Helper()
	runGit(t, repo, "switch", "-q", target)
	mergeInto(t, repo, "feat/x")
	runGit(t, repo, "switch", "-q", "develop")
}

func wantStage(t *testing.T, result branch.Result, stage, prTarget string) branch.BranchResult {
	t.Helper()
	wantReached(t, result)
	described := branchByName(t, onlyRepository(t, result), "feat/x")
	if described.PromotionStage != stage || described.PRTarget != prTarget {
		t.Errorf("etapa %q con prTarget %q; se esperaba %q con %q", described.PromotionStage, described.PRTarget, stage, prTarget)
	}
	if stage != "" && described.PromotionStageLabel == "" {
		t.Errorf("la etapa lleva su etiqueta: %+v", described)
	}
	return described
}

func TestStatusPromotionTargetsTheIntegrationBeforeTheMergeAndTheMainBranchAfterIt(t *testing.T) {
	t.Run("sin fusionar la siguiente PR va a develop", func(t *testing.T) {
		repo, _, _ := twoStageProject(t, "")
		changeFromMaster(t, repo)
		seen := observe(t)

		result := status(t, repo, "x", branch.LaneSDD)

		wantStage(t, result, branch.PromotionStageIntegration, "develop")
		wantNoRemote(t, *seen)
		wantPromotion(t, onlyRepository(t, result), branch.PromotionBranch, branch.PromotionSourceDefault)
	})

	t.Run("fusionada en develop con un merge commit pasa a master", func(t *testing.T) {
		repo, _, _ := twoStageProject(t, "")
		changeFromMaster(t, repo)
		mergeCommitInto(t, repo, "develop")
		seen := observe(t)

		result := status(t, repo, "x", branch.LaneSDD)

		wantStage(t, result, branch.PromotionStageMain, "master")
		wantNoRemote(t, *seen)
	})

	t.Run("fusionada en develop con un squash pasa a master", func(t *testing.T) {
		requireMergeTree(t)
		t.Parallel()
		repo, _, _ := twoStageProject(t, "")
		changeFromMaster(t, repo)
		squashInto(t, repo, "develop")

		result := status(t, repo, "x", branch.LaneSDD)

		wantStage(t, result, branch.PromotionStageMain, "master")
		if len(result.Warnings) != 0 {
			t.Errorf("con soporte de merge-tree no hay avisos: %v", result.Warnings)
		}
	})

	t.Run("una rama recién creada desde master no parece fusionada en develop", func(t *testing.T) {
		t.Parallel()
		repo, _, _ := twoStageProject(t, "")
		runGit(t, repo, "branch", "feat/x", "master")
		if got := runGit(t, repo, "merge-base", "--is-ancestor", "feat/x", "develop"); got != "" {
			t.Fatalf("la preparación debía dejar a master como ascendiente de develop: %s", got)
		}

		result := status(t, repo, "x", branch.LaneSDD)

		wantStage(t, result, branch.PromotionStageIntegration, "develop")
	})
}

// RF-BRA-30: sin la capa del árbol equivalente (git anterior a la 2.38) un squash no se puede
// comprobar: la etapa es `integration` y se avisa, en lugar de fingir certeza. La ascendencia sigue
// valiendo sin esa capa.
func TestStatusPromotionWithoutTheTreeLayerDoesNotPretendCertainty(t *testing.T) {
	t.Run("un squash no concluye y avisa", func(t *testing.T) {
		repo, _, _ := twoStageProject(t, "")
		mergeTreeSupport(t, false)
		changeFromMaster(t, repo)
		squashInto(t, repo, "develop")

		result := status(t, repo, "x", branch.LaneSDD)

		wantStage(t, result, branch.PromotionStageIntegration, "develop")
		wantWarning(t, result, "2.38")
		wantWarning(t, result, "master")
	})

	t.Run("un merge commit sigue confirmándose por ascendencia", func(t *testing.T) {
		repo, _, _ := twoStageProject(t, "")
		mergeTreeSupport(t, false)
		changeFromMaster(t, repo)
		mergeCommitInto(t, repo, "develop")

		result := status(t, repo, "x", branch.LaneSDD)

		wantStage(t, result, branch.PromotionStageMain, "master")
		if len(result.Warnings) != 0 {
			t.Errorf("la ascendencia basta sin avisos: %v", result.Warnings)
		}
	})
}

// RF-BRA-30: con `ask` pendiente (sin registro) `status` no calcula etapa y avisa de que el modo
// está pendiente; en `release` la rama sale de la integración y no hay etapa.
func TestStatusPromotionAskPendingWarnsWithoutAStageAndReleaseHasNone(t *testing.T) {
	t.Run("ask pendiente avisa sin etapa", func(t *testing.T) {
		t.Parallel()
		repo, _, _ := twoStageProject(t, "  promotion: ask\n")
		changeFromMaster(t, repo)
		mergeCommitInto(t, repo, "develop")

		result := status(t, repo, "x", branch.LaneSDD)

		described := wantStage(t, result, "", "develop")
		if described.PromotionStageLabel != "" {
			t.Errorf("sin etapa tampoco hay etiqueta: %+v", described)
		}
		wantWarning(t, result, "pendiente")
		if repository := onlyRepository(t, result); repository.Promotion != string(branch.PromotionAsk) {
			t.Errorf("promoción = %q, se esperaba el modo pendiente ask", repository.Promotion)
		}
	})

	t.Run("release no tiene etapa", func(t *testing.T) {
		t.Parallel()
		repo, _, _ := twoStageProject(t, "  promotion: release\n")
		changeFromMaster(t, repo)
		mergeCommitInto(t, repo, "develop")

		wantStage(t, status(t, repo, "x", branch.LaneSDD), "", "develop")
	})
}

// RF-BRA-30 y RF-BRA-15: desde un worktree enlazado `status` resuelve el registro y la etapa igual
// que desde el checkout principal.
func TestStatusPromotionFromALinkedWorktreeSeesTheRecordedMode(t *testing.T) {
	t.Parallel()
	repo, _, _ := twoStageProject(t, "")
	changeFromMaster(t, repo)
	mergeCommitInto(t, repo, "develop")
	record := registeredRecord(repo)
	record.Repositories[0].Promotion = branch.PromotionBranch
	record.Repositories[0].PromotionSource = branch.PromotionSourceAnswer
	record.Repositories[0].Branches = []branch.RecordBranch{{Name: "feat/x", Kind: branch.KindChange}}
	writeRegistry(t, commonDir(repo), record)
	linked := filepath.Join(t.TempDir(), "enlazado")
	runGit(t, repo, "worktree", "add", "-q", "-b", "inc/otra", linked, "master")

	result := status(t, linked, "x", branch.LaneSDD)

	wantStage(t, result, branch.PromotionStageMain, "master")
	wantPromotion(t, onlyRepository(t, result), branch.PromotionBranch, branch.PromotionSourceAnswer)
	var text strings.Builder
	result.WriteText(&text, &text)
	if !strings.Contains(text.String(), "siguiente PR hacia master") {
		t.Errorf("el texto debe decir «siguiente PR hacia master»:\n%s", text.String())
	}
}
