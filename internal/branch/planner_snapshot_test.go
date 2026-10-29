package branch

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// Pruebas del plan con instantánea (design D-04 y D-12, decisión T2): con la instantánea del
// kickoff, sus nombres de rama prevalecen sobre la política; sin ella, o sin entrada para el
// repositorio, rige la política. Tablas sin git ni disco, como las del resto del planificador.

// snapshotFor es una instantánea de `x` con prefijo `inc` para el repositorio de ruta path.
func snapshotFor(path string, branches ...kickoff.GitSnapshotBranch) *kickoff.GitSnapshot {
	return &kickoff.GitSnapshot{
		IntegrationBranch: "main",
		BranchPrefix:      "inc",
		ChangeBranch:      "inc/x",
		Repositories: []kickoff.GitSnapshotRepo{{
			Path: path, Name: "proyecto", Role: "change_branch", IntegrationBranch: "main", Base: "main", Branches: branches,
		}},
	}
}

var incBranches = []kickoff.GitSnapshotBranch{
	{Name: "inc/x", Kind: "change"},
	{Name: "inc/x-core", Kind: "role", Role: "core"},
	{Name: "inc/x-qa", Kind: "role", Role: "qa"},
	{Name: "inc/x-archive", Kind: "archive"},
}

// D-12: la política actual dice `feat`, pero el cambio se selló con `inc`: el plan conserva los
// nombres de la instantánea, la rama de seguimiento y las de rol.
func TestPlanTakesTheBranchNamesFromTheSnapshotOverThePolicy(t *testing.T) {
	in := embeddedInput("x", "core", "qa")
	in.PlanningRepo = planningFacts("core", "qa")
	in.Snapshot = snapshotFor(".", incBranches...)

	plan := mustPlan(t, in)

	repo := plan.Repositories[0]
	wantNames(t, "ramas", branchNames(repo), "inc/x", "inc/x-core", "inc/x-qa")
	if repo.Branches[1].From != "inc/x" || repo.Branches[1].PRTarget != "inc/x" || repo.Branches[0].PRTarget != "main" {
		t.Errorf("el seguimiento y los PR derivan de los nombres de la instantánea: %+v", repo.Branches)
	}
}

// Una instantánea sin ramas de rol (un único rol, o un cambio sellado antes de que el repositorio
// declarase el segundo) no impide que el rol salga: se compone con el nombre del cambio.
func TestPlanComposesTheRoleBranchesTheSnapshotDoesNotName(t *testing.T) {
	in := embeddedInput("x", "core", "qa")
	in.PlanningRepo = planningFacts("core", "qa")
	in.Snapshot = snapshotFor(".", kickoff.GitSnapshotBranch{Name: "inc/x", Kind: "change"})

	wantNames(t, "ramas", branchNames(mustPlan(t, in).Repositories[0]), "inc/x", "inc/x-core", "inc/x-qa")
}

// Sin instantánea, o sin entrada para el repositorio, rige la política.
func TestPlanFallsBackToThePolicyWithoutASnapshotEntryForTheRepository(t *testing.T) {
	for name, snapshot := range map[string]*kickoff.GitSnapshot{
		"sin instantánea":                 nil,
		"entrada de otro repositorio":     snapshotFor("otro", incBranches...),
		"entrada sin rama del cambio":     snapshotFor("."),
		"repositorio fuera de la maestra": snapshotFor("../fuera", incBranches...),
	} {
		t.Run(name, func(t *testing.T) {
			in := embeddedInput("x", "core")
			in.PlanningRepo = planningFacts("core")
			in.Snapshot = snapshot
			wantNames(t, "ramas", branchNames(mustPlan(t, in).Repositories[0]), "feat/x")
		})
	}
}

// El repositorio fuera de la carpeta maestra (RelPath vacío) no tiene entrada propia: rige la
// política aunque la instantánea tenga una ruta que lo nombre.
func TestPlanNeverMatchesARepositoryOutsideTheMasterFolderWithTheSnapshot(t *testing.T) {
	in := embeddedInput("x", "core")
	facts := planningFacts("core")
	facts.RelPath = ""
	in.PlanningRepo = facts
	in.Snapshot = snapshotFor("", incBranches...)

	wantNames(t, "ramas", branchNames(mustPlan(t, in).Repositories[0]), "feat/x")
}
