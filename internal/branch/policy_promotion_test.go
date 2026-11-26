package branch

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RF-BRA-03: `main_branch` y `promotion` entran a la política con la herencia
// actual. Sin ellas, la política es idéntica a la de antes.
func TestPolicyPromotionDefaultsToNothingDeclared(t *testing.T) {
	for name, policy := range map[string]Policy{
		"sin configuración": ResolvePolicy(nil),
		"sin bloques":       loadPolicy(t, ""),
		"solo integración":  loadPolicy(t, "git:\n  integration_branch: develop\n"),
	} {
		t.Run(name, func(t *testing.T) {
			if policy.MainBranch != "" || policy.Promotion != "" {
				t.Errorf("política = %+v, main_branch y promotion deben quedar vacíos", policy)
			}
			got := policy.For("app")
			if got.MainBranch != "" || got.Promotion != "" || got.PromotionSource != "" {
				t.Errorf("For(app) = %+v, no debe llevar promoción", got)
			}
		})
	}
}

// RF-BRA-03 y RF-BRA-28: un repositorio hereda `promotion` del bloque `git:` salvo que
// su entrada la declare, y `PromotionSource` dice de qué capa sale.
func TestPolicyPromotionInheritsFieldByField(t *testing.T) {
	policy := loadPolicy(t, `git:
  integration_branch: develop
  main_branch: master
  promotion: release
repositories:
  - path: web
    promotion: ask
  - path: api
    branch_prefix: ui
  - path: tools
    promotion: branch
`)
	tests := []struct {
		repo      string
		promotion Promotion
		source    string
	}{
		{"web", PromotionAsk, PromotionSourceRepository},
		{"api", PromotionRelease, PromotionSourceGit},
		{"tools", PromotionBranch, PromotionSourceRepository},
		{"otro", PromotionRelease, PromotionSourceGit}, // sin entrada: rige el bloque
		{".", PromotionRelease, PromotionSourceGit},
	}
	for _, tc := range tests {
		t.Run(tc.repo, func(t *testing.T) {
			got := policy.For(tc.repo)
			if got.Promotion != tc.promotion || got.PromotionSource != tc.source {
				t.Errorf("For(%s) = {%q, %q}, se esperaba {%q, %q}", tc.repo, got.Promotion, got.PromotionSource, tc.promotion, tc.source)
			}
			if got.MainBranch != "master" {
				t.Errorf("For(%s).MainBranch = %q, siempre sale del workspace (master)", tc.repo, got.MainBranch)
			}
		})
	}
	wantRepoPolicy(t, "For(api)", policy.For("api"), "develop", "ui")
}

// `main_branch` solo existe en el workspace: una entrada no la sustituye aunque se
// construya a mano una `RepoPolicy` con ella.
func TestPolicyMainBranchComesOnlyFromTheWorkspace(t *testing.T) {
	policy := Policy{
		IntegrationBranch: "develop",
		BranchPrefix:      "feat",
		MainBranch:        "master",
		PerRepository:     map[string]RepoPolicy{"web": {MainBranch: "otra", IntegrationBranch: "rel"}},
	}
	if got := policy.For("web"); got.MainBranch != "master" || got.IntegrationBranch != "rel" {
		t.Errorf("For(web) = %+v, main_branch debe seguir siendo master", got)
	}
}

// Un repositorio fuera de la carpeta maestra (RelPath vacía) solo hereda la política
// del workspace, también en `main_branch` y `promotion`.
func TestRepoPolicyOutsideTheMasterFolderInheritsThePromotion(t *testing.T) {
	policy := Policy{IntegrationBranch: "develop", BranchPrefix: "feat", MainBranch: "master", Promotion: PromotionBranch,
		PerRepository: map[string]RepoPolicy{"web": {Promotion: PromotionRelease}}}

	got := repoPolicy(policy, RepoFacts{})
	want := RepoPolicy{IntegrationBranch: "develop", BranchPrefix: "feat", MainBranch: "master", Promotion: PromotionBranch, PromotionSource: PromotionSourceGit}
	if got != want {
		t.Errorf("repoPolicy fuera de la carpeta maestra = %+v, se esperaba %+v", got, want)
	}

	inside := repoPolicy(policy, RepoFacts{RelPath: "web"})
	if inside.Promotion != PromotionRelease || inside.PromotionSource != PromotionSourceRepository {
		t.Errorf("repoPolicy dentro = %+v, debe usar la entrada", inside)
	}
}

// ResolvePolicy copia lo declarado en `git:` y `repositories:` sin inventar defectos.
func TestResolvePolicyCopiesThePromotionKeys(t *testing.T) {
	policy := ResolvePolicy(&workspace.WorkspaceConfig{
		Git:          &workspace.GitConfig{IntegrationBranch: "develop", MainBranch: "master", Promotion: "ask"},
		Repositories: []workspace.RepositoryPolicy{{Path: "web", Promotion: "branch"}},
	})
	if policy.MainBranch != "master" || policy.Promotion != PromotionAsk {
		t.Errorf("política = %+v", policy)
	}
	if got := policy.PerRepository["web"].Promotion; got != PromotionBranch {
		t.Errorf("PerRepository[web].Promotion = %q, se esperaba branch", got)
	}
}
