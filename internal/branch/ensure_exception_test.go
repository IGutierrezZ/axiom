package branch

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Rechazos y validación de las excepciones por cambio y del repositorio de specs de un multirepo
// (REQ-25.3, REQ-25.5, REQ-25.12, design D-04 y D-05): el conflicto de excepciones, el
// repositorio de specs fuera de su integración y los nombres inválidos de la invocación, cada uno
// con la continuación como orden ejecutable. Pruebas puras, sin git.

func TestExceptionConflictNamesTheContradictedExceptionsAndRepeatsTheRegistered(t *testing.T) {
	materialized := Exceptions{Branch: "inc/x"}

	refusal := ExceptionConflict("ensure", "x", modeTestCWD, LaneSDD, materialized, Exceptions{Branch: "feat/x", Prefix: "doc", Base: "main"})

	if refusal.Code != CodePlanConflict || refusal.Decision != nil {
		t.Fatalf("rechazo = %+v, se esperaba plan_conflict sin decisión", refusal)
	}
	for _, want := range []string{"cambio x", "--branch", "inc/x", "feat/x", "--prefix", "doc", "--base", "main", "no se ha tocado nada"} {
		if !strings.Contains(refusal.Message, want) {
			t.Errorf("el mensaje %q debe contener %q", refusal.Message, want)
		}
	}
	want := "axiom branch ensure --change x --branch inc/x --cwd " + pathquote.Quote(modeTestCWD)
	if !strings.Contains(refusal.Continuation, want) {
		t.Errorf("la continuación %q debe repetir las excepciones registradas con %q", refusal.Continuation, want)
	}
	if refusal.Details["change"] != "x" || refusal.Details["fields"] != "prefix,branch,base" {
		t.Errorf("details = %v, se esperaban el cambio y los campos en conflicto", refusal.Details)
	}
}

func TestExceptionConflictOnlyNamesTheFieldsThatDiffer(t *testing.T) {
	materialized := Exceptions{Prefix: "inc", Base: "release/2"}

	refusal := ExceptionConflict("ensure", "mi-feature", modeTestCWD, LaneODD, materialized, Exceptions{Prefix: "inc", Base: "main"})

	if strings.Contains(refusal.Message, "--prefix") || !strings.Contains(refusal.Message, "--base") || refusal.Details["fields"] != "base" {
		t.Errorf("mensaje %q, details %v; solo difiere --base", refusal.Message, refusal.Details)
	}
	want := "axiom branch ensure --change mi-feature --lane odd --prefix inc --base release/2 --cwd " + pathquote.Quote(modeTestCWD)
	if !strings.Contains(refusal.Continuation, want) {
		t.Errorf("la continuación %q debe contener %q (con el carril odd y todas las registradas)", refusal.Continuation, want)
	}
}

func TestExceptionConflictWithNothingRegisteredOffersTheBareInvocation(t *testing.T) {
	refusal := ExceptionConflict("ensure", "x", modeTestCWD, LaneSDD, Exceptions{}, Exceptions{Branch: "inc/x"})

	if want := "axiom branch ensure --change x --cwd " + pathquote.Quote(modeTestCWD); !strings.Contains(refusal.Continuation, want) || strings.Contains(refusal.Continuation, "--branch") {
		t.Errorf("la continuación %q debe ser la orden sin excepciones %q", refusal.Continuation, want)
	}
	if !strings.Contains(refusal.Message, "política") {
		t.Errorf("el mensaje %q debe decir que lo registrado es la política", refusal.Message)
	}
}

var multirepoTestSpecs = RepoPlan{
	Name:              "repo-specs",
	GitRoot:           filepath.Join(modeTestCWD, "repo-specs"),
	IntegrationBranch: "main",
	Role:              RepoSpecsInPlace,
}

func TestMultirepoSpecsRepoNotOnIntegrationNamesTheBranchesAndAnExecutableContinuation(t *testing.T) {
	refusal := SpecsRepoNotOnIntegration("ensure", "x", modeTestCWD, LaneSDD, multirepoTestSpecs, "feat/otro", false)

	if refusal.Code != CodeSpecsRepoNotOnIntegration {
		t.Fatalf("código = %s, se esperaba specs_repo_not_on_integration", refusal.Code)
	}
	for _, want := range []string{"repo-specs", "feat/otro", "main", "no se hace checkout"} {
		if !strings.Contains(refusal.Message, want) {
			t.Errorf("el mensaje %q debe contener %q", refusal.Message, want)
		}
	}
	for _, want := range []string{
		"git -C " + pathquote.Quote(multirepoTestSpecs.GitRoot) + " switch main",
		"axiom branch ensure --change x --cwd " + pathquote.Quote(modeTestCWD),
	} {
		if !strings.Contains(refusal.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", refusal.Continuation, want)
		}
	}
	if refusal.Details["repository"] != multirepoTestSpecs.GitRoot || refusal.Details["branch"] != "feat/otro" || refusal.Details["expected"] != "main" {
		t.Errorf("details = %v", refusal.Details)
	}
}

func TestMultirepoSpecsRepoNotOnIntegrationWithADetachedHeadAndTheOddLane(t *testing.T) {
	refusal := SpecsRepoNotOnIntegration("ensure", "mi-feature", modeTestCWD, LaneODD, multirepoTestSpecs, "", true)

	if !strings.Contains(refusal.Message, "HEAD desacoplado") || refusal.Details["branch"] != "" {
		t.Errorf("mensaje %q, details %v; se esperaba HEAD desacoplado sin rama", refusal.Message, refusal.Details)
	}
	if want := "axiom branch ensure --change mi-feature --lane odd --cwd " + pathquote.Quote(modeTestCWD); !strings.Contains(refusal.Continuation, want) {
		t.Errorf("la continuación %q debe contener %q (con el carril odd)", refusal.Continuation, want)
	}
}

func TestValidateExceptionsRejectsInvalidInvocationNamesAsInvalidName(t *testing.T) {
	for name, tc := range map[string]struct {
		exceptions Exceptions
		field      string
	}{
		"prefijo":        {Exceptions{Prefix: "mi prefijo"}, "--prefix"},
		"rama":           {Exceptions{Branch: "inc/x..y"}, "--branch"},
		"base con guion": {Exceptions{Base: "-x"}, "--base"},
	} {
		t.Run(name, func(t *testing.T) {
			refusal := ValidateExceptions("ensure", modeTestCWD, "x", tc.exceptions)

			if refusal == nil || refusal.Code != CodeInvalidName || refusal.Details["field"] != tc.field || refusal.Continuation == "" {
				t.Fatalf("rechazo = %+v, se esperaba invalid_name sobre %s con continuación", refusal, tc.field)
			}
		})
	}
	t.Run("sin excepciones o con valores válidos no se rechaza", func(t *testing.T) {
		for _, exceptions := range []Exceptions{{}, {Prefix: "equipo/inc", Branch: "inc/inc-25-x", Base: "release/2"}} {
			if refusal := ValidateExceptions("ensure", modeTestCWD, "x", exceptions); refusal != nil {
				t.Errorf("%+v no debía rechazarse: %+v", exceptions, refusal)
			}
		}
	})
}

// REQ-25.13: el carril ODD no divide en roles pero incluye los repositorios que se piden con
// `--repo`: cada uno recibe solo la rama de la feature, con su propia integración y su worktree.
// Los repositorios que declaran los roles del kickoff no entran.
func TestPlanOddLaneIncludesTheRequestedRepositoriesWithOnlyTheFeatureBranch(t *testing.T) {
	in := embeddedInput("mi-feature", "core", "qa")
	in.Lane = LaneODD
	in.PlanningRepo = planningFacts("core", "qa")
	in.CodeRepos = []RepoFacts{codeFacts("repo-declarado", "core")}
	in.IncludedRepos = []RepoFacts{codeFacts("repo-zeta"), codeFacts("repo-alfa")}

	plan := mustPlan(t, in)

	wantNames(t, "repositorios", repositoryNames(plan), "axiom", "repo-alfa", "repo-zeta")
	for _, repo := range plan.Repositories {
		wantNames(t, repo.Name+": ramas", branchNames(repo), "feat/mi-feature")
		if len(repo.Roles) != 0 || repo.Branches[0].Kind != KindChange || repo.Branches[0].PRTarget != "main" {
			t.Errorf("%s = %+v, se esperaba solo la rama del cambio hacia main, sin roles", repo.Name, repo)
		}
	}
	if want := filepath.Join(testWorktreeRoot, "mi-feature", "repo-alfa"); plan.Repositories[1].Branches[0].Worktree != want {
		t.Errorf("worktree de repo-alfa = %q, se esperaba %q", plan.Repositories[1].Branches[0].Worktree, want)
	}

	t.Run("fuera del carril ODD los repositorios incluidos no cuentan", func(t *testing.T) {
		sdd := embeddedInput("x", "core")
		sdd.PlanningRepo = planningFacts("core")
		sdd.IncludedRepos = []RepoFacts{codeFacts("repo-alfa")}

		wantNames(t, "repositorios", repositoryNames(mustPlan(t, sdd)), "axiom")
	})
	t.Run("en multirepo el repositorio de specs sigue in situ y los incluidos son de código", func(t *testing.T) {
		multirepo := embeddedInput("x")
		multirepo.Topology = workspace.TopologyMultirepo
		multirepo.Lane = LaneODD
		multirepo.Change = "mi-feature"
		multirepo.IncludedRepos = []RepoFacts{codeFacts("repo-backend")}

		plan := mustPlan(t, multirepo)

		wantNames(t, "repositorios", repositoryNames(plan), "axiom", "repo-backend")
		if plan.Repositories[0].Role != RepoSpecsInPlace || len(plan.Repositories[0].Branches) != 0 || plan.Repositories[1].Role != RepoCode {
			t.Errorf("repositorios = %+v, se esperaba specs in situ y uno de código", plan.Repositories)
		}
		if want := filepath.Join(testMaster, "odd", "tasks", "mi-feature.md"); plan.OddDocument != want {
			t.Errorf("oddDocument = %q, se esperaba %q", plan.OddDocument, want)
		}
	})
}
