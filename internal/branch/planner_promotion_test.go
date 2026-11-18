package branch

import (
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// twoStagePolicy es la política de un proyecto con rama principal `master` e integración
// `develop`.
func twoStagePolicy(t *testing.T, extra string) Policy {
	t.Helper()
	return loadPolicy(t, "git:\n  integration_branch: develop\n  main_branch: master\n"+extra)
}

func multirepoTwoStageInput(t *testing.T) PlanInput {
	t.Helper()
	in := embeddedInput("x", "backend", "frontend")
	in.Topology = workspace.TopologyMultirepo
	in.CodeRepos = []RepoFacts{codeFacts("repo-backend", "backend"), codeFacts("repo-frontend", "frontend")}
	in.Policy = twoStagePolicy(t, "")
	return in
}

// REQ-25.46 (D6): la base efectiva depende del modo; IntegrationBranch y PRTarget no cambian.
func TestPlanBaseBranchFollowsThePromotionMode(t *testing.T) {
	for _, tc := range []struct {
		name     string
		resolved ResolvedPromotion
		base     string
	}{
		{"modo branch sale de la rama principal", ResolvedPromotion{Mode: PromotionBranch, Source: PromotionSourceAnswer}, "master"},
		{"modo release sale de la integración", ResolvedPromotion{Mode: PromotionRelease, Source: PromotionSourceGit}, "develop"},
		{"modo pendiente sale de la integración", ResolvedPromotion{Mode: PromotionAsk}, "develop"},
		{"sin resolver sale de la integración", ResolvedPromotion{}, "develop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := embeddedInput("x", "fullstack")
			in.Policy = twoStagePolicy(t, "")
			in.Promotions = map[string]ResolvedPromotion{testMaster: tc.resolved}

			repo := mustPlan(t, in).Repositories[0]

			if repo.MainBranch != "master" || !repo.TwoStage() {
				t.Errorf("MainBranch = %q, TwoStage = %v; se esperaba master y dos etapas", repo.MainBranch, repo.TwoStage())
			}
			if repo.BaseBranch != tc.base || repo.Base() != tc.base {
				t.Errorf("BaseBranch = %q, Base() = %q; se esperaba %q", repo.BaseBranch, repo.Base(), tc.base)
			}
			if repo.Promotion != tc.resolved.Mode || repo.PromotionSource != tc.resolved.Source {
				t.Errorf("modo %q (%q), se esperaba %q (%q)", repo.Promotion, repo.PromotionSource, tc.resolved.Mode, tc.resolved.Source)
			}
			if repo.IntegrationBranch != "develop" {
				t.Errorf("IntegrationBranch = %q: la promoción no lo redefine", repo.IntegrationBranch)
			}
			if got := repo.Branches[0].PRTarget; got != "develop" {
				t.Errorf("PRTarget de la rama del cambio = %q, la primera PR sigue yendo a la integración", got)
			}
		})
	}
}

// El plan sin `Promotions` y sin `main_branch` es idéntico al que no conoce la promoción: ni
// un campo nuevo se rellena.
func TestPlanWithoutMainBranchIsUnchanged(t *testing.T) {
	in := embeddedInput("x", "fullstack")
	plain := mustPlan(t, in)

	in.Promotions = map[string]ResolvedPromotion{testMaster: {Mode: PromotionBranch, Source: PromotionSourceFlag}}
	withMap := mustPlan(t, in)

	if !reflect.DeepEqual(plain, withMap) {
		t.Errorf("un repositorio de una etapa ignora Promotions:\n%+v\n%+v", plain, withMap)
	}
	repo := plain.Repositories[0]
	if repo.MainBranch != "" || repo.Promotion != "" || repo.PromotionSource != "" || repo.BaseBranch != "" || repo.TwoStage() {
		t.Errorf("sin main_branch no hay campos de promoción: %+v", repo)
	}
	if repo.Base() != "main" {
		t.Errorf("Base() = %q, se esperaba main", repo.Base())
	}
}

// Con `main_branch` igual a la integración tampoco hay segunda etapa.
func TestPlanWithMainBranchEqualToIntegrationIsSingleStage(t *testing.T) {
	in := embeddedInput("x", "fullstack")
	in.Policy = loadPolicy(t, "git:\n  main_branch: main\n")
	in.Promotions = map[string]ResolvedPromotion{testMaster: {Mode: PromotionBranch}}

	repo := mustPlan(t, in).Repositories[0]

	if repo.MainBranch != "" || repo.TwoStage() || repo.BaseBranch != "" || repo.Promotion != "" {
		t.Errorf("repositorio de una etapa con campos de promoción: %+v", repo)
	}
}

// REQ-25.50: el repositorio de specs de un multirepo y las ramas de rol no cambian.
func TestPlanPromotionLeavesSpecsRepositoryAndRoleBranchesUntouched(t *testing.T) {
	in := multirepoTwoStageInput(t)
	in.CodeRepos = []RepoFacts{codeFacts("repo-backend", "backend", "api"), codeFacts("repo-frontend", "frontend")}
	in.SealedRoles = []string{"backend", "api", "frontend"}
	backend, frontend := in.CodeRepos[0].GitRoot, in.CodeRepos[1].GitRoot
	in.Promotions = map[string]ResolvedPromotion{
		testMaster: {Mode: PromotionBranch}, // el repositorio de specs: se ignora
		backend:    {Mode: PromotionBranch, Source: PromotionSourceAnswer},
		frontend:   {Mode: PromotionRelease, Source: PromotionSourceAnswer},
	}

	plan := mustPlan(t, in)

	specs := plan.Repositories[0]
	if specs.Role != RepoSpecsInPlace || specs.MainBranch != "" || specs.Promotion != "" || specs.BaseBranch != "" {
		t.Errorf("el repositorio de specs está exento de la promoción: %+v", specs)
	}
	backendPlan, frontendPlan := plan.Repositories[1], plan.Repositories[2]
	if backendPlan.Base() != "master" || frontendPlan.Base() != "develop" {
		t.Errorf("bases = %q y %q, se esperaban master y develop", backendPlan.Base(), frontendPlan.Base())
	}
	for _, branch := range backendPlan.Branches {
		switch branch.Kind {
		case KindChange:
			if branch.PRTarget != "develop" {
				t.Errorf("la rama del cambio apunta a %q, se esperaba develop", branch.PRTarget)
			}
		case KindRole:
			if branch.PRTarget != backendPlan.Branches[0].Name || branch.From != backendPlan.Branches[0].Name {
				t.Errorf("la rama de rol %s sale de y apunta a la de seguimiento: %+v", branch.Name, branch)
			}
		}
	}
}

// El plan de archivado nunca lleva modo de promoción.
func TestPlanArchiveHasNoPromotion(t *testing.T) {
	in := embeddedInput("x", "fullstack")
	in.Policy = twoStagePolicy(t, "")
	in.Promotions = map[string]ResolvedPromotion{testMaster: {Mode: PromotionBranch}}
	in.Archive = true

	plan := mustPlan(t, in)

	for _, repo := range plan.Repositories {
		if repo.Promotion != "" || repo.BaseBranch != "" || repo.MainBranch != "" {
			t.Errorf("el plan de archivado no usa la promoción: %+v", repo)
		}
	}
}

// BaseRef (`--base`) sigue siendo independiente del modo: el plan solo lo propaga.
func TestPlanPromotionKeepsBaseRef(t *testing.T) {
	in := embeddedInput("x", "fullstack")
	in.Policy = twoStagePolicy(t, "")
	in.Exceptions = Exceptions{Base: "origin/release/1"}
	in.Promotions = map[string]ResolvedPromotion{testMaster: {Mode: PromotionBranch}}

	repo := mustPlan(t, in).Repositories[0]

	if repo.BaseRef != "origin/release/1" || repo.BaseBranch != "master" {
		t.Errorf("BaseRef = %q, BaseBranch = %q; BaseRef prevalece en el motor y el plan conserva la base calculada", repo.BaseRef, repo.BaseBranch)
	}
}

// La política por repositorio llega al plan: la entrada declara su propia promoción, que el motor
// resuelve; el plan solo ve el resultado indexado por la raíz git.
func TestPlanPromotionsAreIndexedByGitRoot(t *testing.T) {
	in := multirepoTwoStageInput(t)
	backend := in.CodeRepos[0].GitRoot
	in.Promotions = map[string]ResolvedPromotion{backend: {Mode: PromotionBranch, Source: PromotionSourceRepository}}

	plan := mustPlan(t, in)

	if got := plan.Repositories[1]; got.Name != "repo-backend" || got.Promotion != PromotionBranch || got.PromotionSource != PromotionSourceRepository {
		t.Errorf("repo-backend = %+v", got)
	}
	if got := plan.Repositories[2]; got.Name != "repo-frontend" || got.Promotion != "" || got.Base() != "develop" || !got.TwoStage() {
		t.Errorf("repo-frontend sin modo resuelto se planifica sin modo y con la integración como base: %+v", got)
	}
}
