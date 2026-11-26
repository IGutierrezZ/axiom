package branch

import (
	"strings"
	"testing"
)

// Pruebas del veredicto con ramas pendientes de plantilla (C-07-f, design D-10, G2): un repositorio
// que espera a su rol es `not_applicable` con la continuación `--role`.

func pendingInput(current string, pending ...string) VerdictInput {
	in := monorepoInput(current)
	in.Role = RepoCode
	in.Consultant = ConsultantApply
	in.Expected = ExpectedNames{Integration: "main", PendingRoles: pending}
	return in
}

func TestDecideARepositoryAwaitingItsRoleIsNotApplicableWithTheRoleContinuation(t *testing.T) {
	t.Parallel()

	got := Decide(pendingInput("main", "backend"))

	if got.Kind != VerdictNotApplicable {
		t.Fatalf("veredicto %q, se esperaba not_applicable", got.Kind)
	}
	if got.Details["reason"] != PendingReasonAwaitingRole || got.Details["roles"] != "backend" {
		t.Errorf("Details = %v", got.Details)
	}
	want := "axiom branch ensure --cwd " + `"` + verdictMaster + `"` + " --change x --role backend"
	if got.Continuation != want {
		t.Errorf("continuación = %q, se esperaba %q", got.Continuation, want)
	}
	if got.Detail == "" || got.Warning != "" {
		t.Errorf("el veredicto explica la exención sin aviso: %+v", got)
	}
}

func TestDecideARepositoryAwaitingSeveralRolesUsesTheRolePlaceholder(t *testing.T) {
	t.Parallel()

	got := Decide(pendingInput("main", "api", "jobs"))

	if got.Kind != VerdictNotApplicable {
		t.Fatalf("veredicto %q, se esperaba not_applicable", got.Kind)
	}
	if got.Details["roles"] != "api,jobs" || !strings.HasSuffix(got.Continuation, "--role <rol>") {
		t.Errorf("veredicto = %+v", got)
	}
}

// Una rama que no espera a ningún rol sigue siendo wrong_branch, y sin pendientes no hay Details.
func TestDecideWithoutPendingRolesKeepsTheOrdinaryVerdicts(t *testing.T) {
	t.Parallel()

	got := Decide(monorepoInput("main"))

	wantVerdict(t, got, VerdictWrongBranch)
	if got.Details != nil {
		t.Errorf("Details = %v, se esperaba nil", got.Details)
	}
}

// Si otro rol ya entró, el veredicto se calcula sobre sus ramas y los pendientes se informan aparte.
func TestDecideWithARoleInsideReportsThePendingRolesWithoutChangingTheKind(t *testing.T) {
	t.Parallel()
	in := monorepoInput("feat/x-core")
	in.Expected.Roles = []string{"feat/x-core"}
	in.Expected.PendingRoles = []string{"qa"}

	got := Decide(in)

	wantVerdict(t, got, VerdictCorrect)
	if got.Details["pendingRoles"] != "qa" || got.Details["reason"] != "" {
		t.Errorf("Details = %v", got.Details)
	}

	in.Current = "main"
	got = Decide(in)
	wantVerdict(t, got, VerdictWrongBranch)
	if got.Details["pendingRoles"] != "qa" {
		t.Errorf("Details = %v", got.Details)
	}
}

// Un candidato conocido hace del repositorio uno con rama del cambio: se comprueba como siempre.
func TestDecideWithAKnownCandidateChecksTheBranch(t *testing.T) {
	t.Parallel()
	in := monorepoInput("feature/TASK-1235")
	in.Expected.Change = "feature/TASK-1235"

	wantVerdict(t, Decide(in), VerdictCorrect)
}

func TestFrozenFromKeepsOnlyTemplatedBranches(t *testing.T) {
	t.Parallel()
	got := FrozenFrom([]RecordBranch{
		{Name: "feat/x", Kind: KindChange},
		{Name: "feature/TASK-1", Kind: KindChange, RoleID: "ignorado", Template: "feature/TASK-{task_ref}"},
		{Name: "feature/TASK-2", Kind: KindRole, RoleID: "qa", Template: "feature/TASK-{task_ref}"},
	})

	if len(got) != 2 || got[0].RoleID != "" || got[1].RoleID != "qa" || got[0].Template == "" {
		t.Errorf("congeladas = %+v", got)
	}
}
