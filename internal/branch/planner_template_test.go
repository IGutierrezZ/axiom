package branch

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas del plan con plantilla de nombre de rama (C-07-f, design D-04, D-05, D-07, D-11 y
// D-15): tablas sin git ni disco, como el resto del planificador.

// templatePolicy construye la política de un workspace con plantillas en las tres capas.
func templatePolicy(git string, repos map[string]string, roles map[string]string) Policy {
	cfg := &workspace.WorkspaceConfig{Git: &workspace.GitConfig{BranchTemplate: git}}
	for path, raw := range repos {
		cfg.Repositories = append(cfg.Repositories, workspace.RepositoryPolicy{Path: path, BranchTemplate: raw})
	}
	if len(roles) > 0 {
		cfg.Roles = map[string]workspace.RoleConfig{}
		for id, raw := range roles {
			cfg.Roles[id] = workspace.RoleConfig{BranchTemplate: raw}
		}
	}
	return ResolvePolicy(cfg)
}

// multirepoTemplateInput es un multirepo sellado con los roles dados sobre un repositorio de código.
func multirepoTemplateInput(policy Policy, roles ...string) PlanInput {
	in := embeddedInput("us-2525-login", roles...)
	in.Topology = workspace.TopologyMultirepo
	in.CodeRepos = []RepoFacts{codeFacts("repo-backend", roles...)}
	in.Policy = policy
	// Los tests simulan el plan del propio sellado: aún sin instantánea rige la política.
	in.Sealing = true
	return in
}

func TestPlanTemplateRendersTheChangeBranchOfTheOpenspecRepository(t *testing.T) {
	in := embeddedInput("us-2525-login")
	in.Policy = templatePolicy("{kind}/{ref}-{slug}", nil, nil)
	in.TemplateValues = TemplateValues{Kind: "fix", Ref: "2525"}

	repo := mustPlan(t, in).Repositories[0]

	wantNames(t, "ramas", branchNames(repo), "fix/2525-login")
	if repo.Branches[0].Template != "{kind}/{ref}-{slug}" {
		t.Errorf("Template = %q, se esperaba la plantilla efectiva", repo.Branches[0].Template)
	}
}

func TestPlanTemplateWithPrefixAndChangeEqualsTheDefaultName(t *testing.T) {
	in := embeddedInput("x")
	in.Policy = templatePolicy("{prefix}/{change}", nil, nil)

	wantNames(t, "ramas", branchNames(mustPlan(t, in).Repositories[0]), "feat/x")
}

func TestPlanTemplatePrefixExceptionOnlyFeedsThePrefixVariable(t *testing.T) {
	in := embeddedInput("x")
	in.Policy = templatePolicy("{prefix}/{change}", nil, nil)
	in.Exceptions.Prefix = "inc"
	wantNames(t, "con {prefix}", branchNames(mustPlan(t, in).Repositories[0]), "inc/x")

	in.Policy = templatePolicy("work/{change}", nil, nil)
	wantNames(t, "sin {prefix}", branchNames(mustPlan(t, in).Repositories[0]), "work/x")
}

// D-04: la plantilla efectiva es rol > repositorio > git:, y el repositorio solo cuenta si declara.
func TestPlanTemplateAttributionByRole(t *testing.T) {
	policy := templatePolicy("git/{change}", map[string]string{"repo-backend": "repo/{change}"}, map[string]string{"backend": "role/{role}/{change}"})
	in := multirepoTemplateInput(policy, "backend")
	repo := mustPlan(t, in).Repositories[1]
	wantNames(t, "rol > repositorio", branchNames(repo), "role/backend/us-2525-login")

	policy = templatePolicy("git/{change}", map[string]string{"repo-backend": "repo/{change}"}, nil)
	in = multirepoTemplateInput(policy, "backend")
	wantNames(t, "repositorio > git", branchNames(mustPlan(t, in).Repositories[1]), "repo/us-2525-login")

	policy = templatePolicy("git/{change}", nil, nil)
	in = multirepoTemplateInput(policy, "backend")
	wantNames(t, "git", branchNames(mustPlan(t, in).Repositories[1]), "git/us-2525-login")

	// fullstack es un rol del kickoff sin plantilla de rol: rige el repositorio o git:.
	in = multirepoTemplateInput(policy, "fullstack")
	wantNames(t, "fullstack", branchNames(mustPlan(t, in).Repositories[1]), "git/us-2525-login")
}

// D-04: en un repositorio compartido la rama de seguimiento no tiene rol; las de rol usan solo
// roles.<id>.branch_template o `<seguimiento>-<rol>`.
func TestPlanTemplateSharedRepositoryTrackingAndRoleBranches(t *testing.T) {
	policy := templatePolicy("feature/{change}", nil, map[string]string{"api": "role/{role}-{change}"})
	in := multirepoTemplateInput(policy, "api", "jobs")
	repo := mustPlan(t, in).Repositories[1]

	wantNames(t, "ramas", branchNames(repo), "feature/us-2525-login", "role/api-us-2525-login", "feature/us-2525-login-jobs")
	if repo.Branches[1].From != "feature/us-2525-login" || repo.Branches[2].From != "feature/us-2525-login" {
		t.Errorf("las ramas de rol salen del seguimiento resuelto: %+v", repo.Branches)
	}
}

// R2, R3 y la regla de la rama con openspec/: {role} y {task_ref} no aplican.
func TestPlanTemplateNotApplicable(t *testing.T) {
	t.Run("shared_branch", func(t *testing.T) {
		in := multirepoTemplateInput(templatePolicy("{role}/{change}", nil, nil), "api", "jobs")
		refusal := mustRefuse(t, in, CodeBranchTemplateNotApplicable)
		if refusal.Details["reason"] != TemplateReasonSharedBranch {
			t.Errorf("reason = %q", refusal.Details["reason"])
		}
	})
	t.Run("openspec_branch", func(t *testing.T) {
		in := embeddedInput("x")
		in.Policy = templatePolicy("feature/{task_ref}", nil, nil)
		refusal := mustRefuse(t, in, CodeBranchTemplateNotApplicable)
		if refusal.Details["reason"] != TemplateReasonOpenspecBranch {
			t.Errorf("reason = %q", refusal.Details["reason"])
		}
	})
	t.Run("odd_lane", func(t *testing.T) {
		in := embeddedInput("x")
		in.Lane = LaneODD
		in.Policy = templatePolicy("{role}/{change}", nil, nil)
		refusal := mustRefuse(t, in, CodeBranchTemplateNotApplicable)
		if refusal.Details["reason"] != TemplateReasonODDLane {
			t.Errorf("reason = %q", refusal.Details["reason"])
		}
	})
	t.Run("sin plantilla no se evalúa nada", func(t *testing.T) {
		in := multirepoTemplateInput(ResolvePolicy(nil), "api", "jobs")
		wantNames(t, "ramas", branchNames(mustPlan(t, in).Repositories[1]), "feat/us-2525-login", "feat/us-2525-login-api", "feat/us-2525-login-jobs")
	})
}

// D-07: con {task_ref} la rama queda pendiente hasta que entra su rol.
func TestPlanTemplatePendingUntilTheRoleEnters(t *testing.T) {
	policy := templatePolicy("feature/TASK-{task_ref}", map[string]string{"repo-backend": "feature/TASK-{task_ref}"}, nil)
	values := TemplateValues{TaskRefs: map[string]string{"backend": "1235"}}

	in := multirepoTemplateInput(policy, "backend")
	in.TemplateValues = values
	repo := mustPlan(t, in).Repositories[1]
	if len(repo.Branches) != 0 || len(repo.Pending) != 1 {
		t.Fatalf("ramas %v, pendientes %+v; se esperaba una pendiente", branchNames(repo), repo.Pending)
	}
	pending := repo.Pending[0]
	if pending.Kind != KindChange || pending.RoleID != "backend" || pending.Template != "feature/TASK-{task_ref}" || pending.Prefix != "feat" || pending.Candidate != "feature/TASK-1235" {
		t.Errorf("pendiente = %+v", pending)
	}
	if pending.Key != "repositories[repo-backend].branch_template" {
		t.Errorf("Key = %q", pending.Key)
	}

	in.EnteringRole = "BACKEND"
	repo = mustPlan(t, in).Repositories[1]
	wantNames(t, "ramas", branchNames(repo), "feature/TASK-1235")
	if len(repo.Pending) != 0 {
		t.Errorf("el rol que entra ya no está pendiente: %+v", repo.Pending)
	}

	in.EnteringRole = "frontend"
	repo = mustPlan(t, in).Repositories[1]
	if len(repo.Branches) != 0 || len(repo.Pending) != 1 {
		t.Errorf("otro rol no hace entrar a backend: ramas %v pendientes %+v", branchNames(repo), repo.Pending)
	}
}

func TestPlanTemplateBranchExceptionNamesAPendingChangeBranch(t *testing.T) {
	in := multirepoTemplateInput(templatePolicy("feature/TASK-{task_ref}", nil, nil), "backend")
	in.Exceptions.Branch = "hotfix/manual"

	repo := mustPlan(t, in).Repositories[1]

	wantNames(t, "ramas", branchNames(repo), "hotfix/manual")
	if len(repo.Pending) != 0 {
		t.Errorf("--branch saca la rama de pendiente: %+v", repo.Pending)
	}
}

func TestPlanTemplateRoleBranchesWithTaskRefArePendingAndBranchNeverNamesThem(t *testing.T) {
	policy := templatePolicy("feature/{change}", nil, map[string]string{"api": "feature/TASK-{task_ref}", "jobs": "feature/JOB-{task_ref}"})
	in := multirepoTemplateInput(policy, "api", "jobs")
	in.TemplateValues = TemplateValues{TaskRefs: map[string]string{"api": "7", "jobs": "8"}}
	in.Exceptions.Branch = "hotfix/shared"

	repo := mustPlan(t, in).Repositories[1]

	wantNames(t, "ramas", branchNames(repo), "hotfix/shared")
	if len(repo.Pending) != 2 || repo.Pending[0].Kind != KindRole || repo.Pending[0].RoleID != "api" || repo.Pending[1].RoleID != "jobs" {
		t.Fatalf("pendientes = %+v; se esperaban api y jobs", repo.Pending)
	}

	in.Exceptions.Branch = ""
	in.EnteringRole = "jobs"
	repo = mustPlan(t, in).Repositories[1]
	wantNames(t, "entra jobs", branchNames(repo), "feature/us-2525-login", "feature/JOB-8")
	if len(repo.Pending) != 1 || repo.Pending[0].RoleID != "api" {
		t.Errorf("la entrada de jobs no toca a api: %+v", repo.Pending)
	}
}

func TestPlanTemplateUnresolvedVariables(t *testing.T) {
	t.Run("ref sin valor", func(t *testing.T) {
		in := embeddedInput("x")
		in.Policy = templatePolicy("{ref}/{change}", nil, nil)
		refusal := mustRefuse(t, in, CodeBranchTemplateUnresolved)
		if refusal.Details["variable"] != "ref" {
			t.Errorf("variable = %q", refusal.Details["variable"])
		}
	})
	t.Run("task_ref del rol que entra sin Task", func(t *testing.T) {
		in := multirepoTemplateInput(templatePolicy("feature/TASK-{task_ref}", nil, nil), "backend")
		in.EnteringRole = "backend"
		refusal := mustRefuse(t, in, CodeBranchTemplateUnresolved)
		if refusal.Details["variable"] != "task_ref" || refusal.Details["role"] != "backend" {
			t.Errorf("details = %v", refusal.Details)
		}
	})
	t.Run("status lo deja pendiente con su rechazo", func(t *testing.T) {
		in := embeddedInput("x")
		in.Policy = templatePolicy("{ref}/{change}", nil, nil)
		in.Describe = true
		repo := mustPlan(t, in).Repositories[0]
		if len(repo.Branches) != 0 || len(repo.Pending) != 1 || repo.Pending[0].Refusal == nil || repo.Pending[0].Refusal.Code != CodeBranchTemplateUnresolved {
			t.Errorf("ramas %v, pendientes %+v", branchNames(repo), repo.Pending)
		}
	})
	t.Run("not_applicable sigue rechazando en status", func(t *testing.T) {
		in := embeddedInput("x")
		in.Policy = templatePolicy("{task_ref}", nil, nil)
		in.Describe = true
		mustRefuse(t, in, CodeBranchTemplateNotApplicable)
	})
}

// D-11 (G4): con integrations.yaml ilegible solo falla lo que hay que renderizar con un valor no
// congelado.
func TestPlanTemplateLinksUnreadableFailsClosedOnlyWhenNeeded(t *testing.T) {
	broken := errors.New("esquema ajeno")
	for _, raw := range []string{"{ref}/{change}", "{slug}", "{kind}/{change}"} {
		in := embeddedInput("x")
		in.Policy = templatePolicy(raw, nil, nil)
		in.TemplateValues = TemplateValues{Ref: "9", LinksErr: broken}
		if refusal := mustRefuse(t, in, CodeIntegrationsLinksInvalid); refusal.Details["operation"] != "ensure" {
			t.Errorf("%s: details = %v", raw, refusal.Details)
		}
	}
	// Con los valores congelados no falla.
	in := embeddedInput("x")
	in.Policy = templatePolicy("{kind}/{ref}-{slug}", nil, nil)
	in.TemplateValues = TemplateValues{Kind: "fix", Ref: "9", KindFrozen: true, RefFrozen: true, LinksErr: broken}
	wantNames(t, "congelado", branchNames(mustPlan(t, in).Repositories[0]), "fix/9-x")
	// Una plantilla que no usa nada de eso no se ve afectada.
	in = embeddedInput("x")
	in.Policy = templatePolicy("{prefix}/{change}", nil, nil)
	in.TemplateValues = TemplateValues{LinksErr: broken}
	wantNames(t, "sin variables", branchNames(mustPlan(t, in).Repositories[0]), "feat/x")
}

// Matriz de amenazas (composición de argumentos): un valor hostil nunca se sanea ni crea nada.
func TestPlanTemplateRejectsUnsafeValuesWithoutSanitising(t *testing.T) {
	for _, value := range []string{"-D", "--force", "T#12", "a b", "..", "x.lock", "CON"} {
		in := multirepoTemplateInput(templatePolicy("feature/TASK-{task_ref}", nil, nil), "backend")
		in.EnteringRole = "backend"
		in.TemplateValues = TemplateValues{TaskRefs: map[string]string{"backend": value}}
		refusal := mustRefuse(t, in, CodeInvalidName)
		if refusal.Details["field"] != "task_ref" || refusal.Details["value"] != value {
			t.Errorf("%q: details = %v", value, refusal.Details)
		}
	}
}

// D-05: --branch > instantánea > congelado > plantilla > defecto.
func TestPlanTemplatePrecedenceOfTheChangeBranchName(t *testing.T) {
	policy := templatePolicy("tpl/{change}", nil, nil)
	frozen := []FrozenBranch{{Kind: KindChange, Name: "frozen/x", Template: "old/{change}"}}
	named := &kickoff.GitSnapshot{Repositories: []kickoff.GitSnapshotRepo{{Path: ".", Branches: []kickoff.GitSnapshotBranch{{Name: "snap/x", Kind: "change", Template: "s/{change}"}}}}}

	in := embeddedInput("x")
	in.Policy = policy
	in.PlanningRepo.Frozen = frozen
	wantNames(t, "congelado > plantilla", branchNames(mustPlan(t, in).Repositories[0]), "frozen/x")
	if got := mustPlan(t, in).Repositories[0].Branches[0].Template; got != "old/{change}" {
		t.Errorf("Template del congelado = %q", got)
	}

	in.HasKickoff, in.SealedRoles, in.Snapshot = true, []string{"fullstack"}, named
	wantNames(t, "instantánea > congelado", branchNames(mustPlan(t, in).Repositories[0]), "snap/x")

	in.Exceptions.Branch = "cli/x"
	wantNames(t, "--branch > instantánea", branchNames(mustPlan(t, in).Repositories[0]), "cli/x")
}

// D5: un cambio sellado sin plantilla en su instantánea nunca aplica la de la política; uno sellado
// con una entrada pendiente usa la de la instantánea, no la de la política actual.
func TestPlanTemplateSealedChangeUsesOnlyTheSnapshotTemplates(t *testing.T) {
	policy := templatePolicy("tpl/{change}", map[string]string{"repo-backend": "tpl/{change}"}, nil)
	snapshot := func(pending ...kickoff.GitSnapshotPendingBranch) *kickoff.GitSnapshot {
		return &kickoff.GitSnapshot{Repositories: []kickoff.GitSnapshotRepo{{Path: "repo-backend", PendingBranches: pending}}}
	}

	in := multirepoTemplateInput(policy, "backend")
	in.Sealing = false
	in.Snapshot = snapshot()
	wantNames(t, "sin plantilla sellada", branchNames(mustPlan(t, in).Repositories[1]), "feat/us-2525-login")

	in = multirepoTemplateInput(policy, "backend")
	in.Sealing = false
	in.Snapshot = snapshot(kickoff.GitSnapshotPendingBranch{Kind: "change", Role: "backend", Template: "sealed/{prefix}/{task_ref}", Prefix: "inc"})
	in.TemplateValues = TemplateValues{TaskRefs: map[string]string{"backend": "5"}}
	in.Exceptions.Prefix = ""
	repo := mustPlan(t, in).Repositories[1]
	if len(repo.Pending) != 1 || repo.Pending[0].Template != "sealed/{prefix}/{task_ref}" || repo.Pending[0].Prefix != "inc" || repo.Pending[0].Candidate != "sealed/inc/5" {
		t.Fatalf("pendientes = %+v", repo.Pending)
	}

	in.EnteringRole = "backend"
	wantNames(t, "entra", branchNames(mustPlan(t, in).Repositories[1]), "sealed/inc/5")
}

// Sellando, el plan usa la política y deja los pendientes para congelarlos.
func TestPlanTemplateSealingUsesThePolicy(t *testing.T) {
	in := multirepoTemplateInput(templatePolicy("feature/TASK-{task_ref}", nil, nil), "backend")
	in.Sealing = true
	repo := mustPlan(t, in).Repositories[1]
	if len(repo.Pending) != 1 || repo.Pending[0].Prefix != "feat" {
		t.Errorf("pendientes = %+v", repo.Pending)
	}
}

func TestPlanTemplateCollisions(t *testing.T) {
	t.Run("dos roles con el mismo task_ref", func(t *testing.T) {
		policy := templatePolicy("feature/{change}", nil, map[string]string{"api": "feature/TASK-{task_ref}", "jobs": "feature/TASK-{task_ref}"})
		in := multirepoTemplateInput(policy, "api", "jobs")
		in.TemplateValues = TemplateValues{TaskRefs: map[string]string{"api": "7", "jobs": "7"}}
		in.CodeRepos[0].Frozen = []FrozenBranch{{Kind: KindRole, RoleID: "api", Name: "feature/TASK-7", Template: "feature/TASK-{task_ref}"}}
		in.EnteringRole = "jobs"
		refusal := mustRefuse(t, in, CodeInvalidName)
		if !strings.Contains(refusal.Details["roles"], "api") || !strings.Contains(refusal.Details["roles"], "jobs") {
			t.Errorf("details = %v", refusal.Details)
		}
	})
	t.Run("rama de integración", func(t *testing.T) {
		in := embeddedInput("main")
		in.Policy = templatePolicy("{change}", nil, nil)
		refusal := mustRefuse(t, in, CodeInvalidName)
		if refusal.Details["reserved"] != "integration_branch" {
			t.Errorf("details = %v", refusal.Details)
		}
	})
	t.Run("main_branch", func(t *testing.T) {
		cfg := &workspace.WorkspaceConfig{Git: &workspace.GitConfig{BranchTemplate: "{change}", MainBranch: "release"}}
		in := embeddedInput("release")
		in.Policy = ResolvePolicy(cfg)
		refusal := mustRefuse(t, in, CodeInvalidName)
		if refusal.Details["reserved"] != "main_branch" {
			t.Errorf("details = %v", refusal.Details)
		}
	})
}

// El archivado con plantilla ignora --prefix y parte de la rama del cambio resuelta.
func TestPlanArchiveWithTemplateIgnoresThePrefixException(t *testing.T) {
	in := embeddedInput("x", "fullstack")
	in.Policy = templatePolicy("{kind}/{change}", nil, nil)
	in.Snapshot = &kickoff.GitSnapshot{Repositories: []kickoff.GitSnapshotRepo{{Path: ".", Branches: []kickoff.GitSnapshotBranch{{Name: "feature/x", Kind: "change", Template: "{kind}/{change}"}}}}}
	in.Archive = true
	in.ChangeBranch = "feature/x"
	in.Exceptions.Prefix = "inc"

	wantNames(t, "archivado", branchNames(mustPlan(t, in).Repositories[0]), "feature/x-archive")

	in.Snapshot = nil
	in.Policy = ResolvePolicy(nil)
	in.ChangeBranch = ""
	wantNames(t, "sin plantilla sigue usando --prefix", branchNames(mustPlan(t, in).Repositories[0]), "inc/x-archive")
}

// Un ODD y un repositorio incluido reciben la plantilla del repositorio o de git:.
func TestPlanTemplateOddLane(t *testing.T) {
	in := embeddedInput("x")
	in.Lane = LaneODD
	in.Policy = templatePolicy("{kind}/{ref}-{slug}", nil, nil)
	in.TemplateValues = TemplateValues{Kind: "feature", Ref: "12"}
	in.IncludedRepos = []RepoFacts{codeFacts("repo-api")}
	plan := mustPlan(t, in)
	wantNames(t, "ramas", branchNames(plan.Repositories[0]), "feature/12-x")
	wantNames(t, "incluido", branchNames(plan.Repositories[1]), "feature/12-x")
	if got := plan.Repositories[0].Branches[0].Worktree; got != filepath.Join(testWorktreeRoot, "x", "axiom") {
		t.Errorf("worktree = %q", got)
	}
}
