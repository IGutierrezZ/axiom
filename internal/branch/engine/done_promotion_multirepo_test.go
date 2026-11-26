package engine

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// `done` en un multirepo de dos etapas (C-02-a, RF-BRA-31, W2 del informe de verificación): cada
// repositorio de código se evalúa con su propio modo de promoción y el de specs, que se trabaja in
// situ y no recibe ramas del cambio, queda omitido. Con git real y sin red.

// repositoryNamed busca un repositorio por nombre en el resultado de `done`.
func repositoryNamed(result branch.Result, name string) (branch.RepositoryResult, bool) {
	for _, repository := range result.Repositories {
		if repository.Name == name {
			return repository, true
		}
	}
	return branch.RepositoryResult{}, false
}

func TestDoneMultirepoEvaluatesEachRepositoryWithItsOwnPromotionMode(t *testing.T) {
	requireMergeTree(t)
	roles := map[string][]string{"backend": {"repo-a"}, "frontend": {"repo-b"}}
	master, specs := newMultirepo(t, roles, "repo-a", "repo-b")
	repoA, repoB := filepath.Join(master, "repo-a"), filepath.Join(master, "repo-b")
	for _, dir := range []string{specs, repoA, repoB} {
		twoStageBranches(t, dir)
	}
	// repo-a toma el modo por defecto (`branch`); repo-b declara `release` en `repositories[]`.
	writeFile(t, filepath.Join(master, "axiom.yaml"), configFor("multirepo", "repo-specs", roles,
		twoStageGit+"repositories:\n  - path: \"repo-b\"\n    promotion: release\n"))
	sealChangeAt(t, specs, "x", "backend", "frontend")

	created := ensurePromoted(t, EnsureRequest{CWD: master, Change: "x"})

	wantReady(t, created)
	byName := map[string]branch.RepositoryResult{}
	for _, repository := range created.Repositories {
		byName[repository.Name] = repository
	}
	wantPromotion(t, byName["repo-a"], branch.PromotionBranch, branch.PromotionSourceDefault)
	wantPromotion(t, byName["repo-b"], branch.PromotionRelease, branch.PromotionSourceRepository)

	// Las dos ramas feat/x se fusionan en develop; solo la de repo-a es de modo `branch`.
	for _, dir := range []string{repoA, repoB} {
		commitFile(t, dir, "cambio.txt", "cambio de x")
		runGit(t, dir, "switch", "-q", "develop")
		mergeInto(t, dir, "feat/x")
	}

	first := doneOf(t, doneRequest(master))

	wantDoneStatus(t, first, branch.StatusPartial, branch.CodeMergeNotConfirmed)
	if _, listed := repositoryNamed(first, "repo-specs"); listed {
		t.Errorf("`done` omite el repositorio de specs: %+v", first.Repositories)
	}
	kept, ok := repositoryNamed(first, "repo-a")
	if !ok {
		t.Fatalf("`done` no describe repo-a: %+v", first.Repositories)
	}
	keptBranch := branchByName(t, kept, "feat/x")
	wantOutcome(t, keptBranch, branch.OutcomeKept, "", branch.CodeMergeNotConfirmed)
	if !strings.Contains(keptBranch.Message, "master") {
		t.Errorf("el rechazo de repo-a nombra master como destino: %q", keptBranch.Message)
	}
	removed, ok := repositoryNamed(first, "repo-b")
	if !ok {
		t.Fatalf("`done` no describe repo-b: %+v", first.Repositories)
	}
	wantOutcome(t, branchByName(t, removed, "feat/x"), branch.OutcomeRemoved, branch.LayerAncestry, "")
	if !branchExists(t, repoA, "feat/x") || branchExists(t, repoB, "feat/x") {
		t.Errorf("repo-a conserva feat/x hasta master y repo-b la elimina al fusionar en develop (a=%v, b=%v)",
			branchExists(t, repoA, "feat/x"), branchExists(t, repoB, "feat/x"))
	}

	// Al fusionar en master, repo-a también se elimina; repo-b ya no tiene ramas y el de specs sigue omitido.
	runGit(t, repoA, "switch", "-q", "master")
	mergeInto(t, repoA, "feat/x")
	runGit(t, repoA, "switch", "-q", "develop")

	second := doneOf(t, doneRequest(master))

	wantReady(t, second)
	if _, listed := repositoryNamed(second, "repo-specs"); listed {
		t.Errorf("`done` omite el repositorio de specs: %+v", second.Repositories)
	}
	finished, ok := repositoryNamed(second, "repo-a")
	if !ok {
		t.Fatalf("`done` no describe repo-a: %+v", second.Repositories)
	}
	wantOutcome(t, branchByName(t, finished, "feat/x"), branch.OutcomeRemoved, branch.LayerAncestry, "")
	if branchExists(t, repoA, "feat/x") {
		t.Error("fusionada en master, la rama de repo-a se elimina")
	}
}
