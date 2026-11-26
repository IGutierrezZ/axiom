package engine

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
)

// Pruebas del registro con nombres congelados, la recuperación sin registro y la guarda con
// plantilla (C-07-f, design D-09, D-10 y D-14). Las que instalan el doble de integraciones tocan
// estado de paquete y no llaman a t.Parallel().

const taskTemplateBlock = "git:\n  integration_branch: main\n  branch_prefix: inc\n" +
	"repositories:\n  - path: repo-backend\n    branch_template: \"feature/TASK-{task_ref}\"\n" +
	"integrations:\n  fake:\n    project: p\n"

// sealedTaskProject monta un multirepo con el repositorio de código `repo-backend` cuya rama del
// cambio se nombra con `{task_ref}`, el kickoff sellado con su instantánea y el doble instalado.
func sealedTaskProject(t *testing.T) (master, specs, backend string) {
	t.Helper()
	integrationstest.Install(t, integrationstest.NewFake())
	roles := map[string][]string{"backend": {"repo-backend"}}
	master, specs = newMultirepo(t, roles, "repo-backend")
	writeFile(t, filepath.Join(master, "axiom.yaml"), configFor("multirepo", "repo-specs", roles, taskTemplateBlock))
	sealWithSnapshot(t, specs, "x", snapshotOf(t, master, "x", "backend"), "backend")
	return master, specs, filepath.Join(master, "repo-backend")
}

func bindTask(t *testing.T, specs, role, id string) {
	t.Helper()
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "integrations.yaml"),
		"schema: axiom.integrations/v1\nchange: x\nlinks:\n"+
			"  - {plugin: fake, kind: task, id: \""+id+"\", role: "+role+", source: tasks.bind, recorded_at: 2026-01-01T00:00:00Z}\n")
}

func ensureSDD(t *testing.T, engine *Engine, master string) branch.Result {
	t.Helper()
	engine.prefs = hubWorktree
	return engine.Ensure(context.Background(), EnsureRequest{CWD: master, Change: "x", Lane: branch.LaneSDD, Mode: branch.ModeWorktree})
}

func recordedBackend(t *testing.T, specs string) branch.RecordRepo {
	t.Helper()
	record, err := registry.New().Load(commonDir(specs), branch.LaneSDD, "x")
	if err != nil || record == nil {
		t.Fatalf("registro de x: %+v, %v", record, err)
	}
	for _, repo := range record.Repositories {
		if repo.Name == "repo-backend" {
			return repo
		}
	}
	t.Fatalf("el registro no lista repo-backend: %+v", record.Repositories)
	return branch.RecordRepo{}
}

// RF-BRA-15 y D-09: el registro guarda la plantilla de la rama y, si la plantilla cambia después,
// la rama registrada manda: no se crea otra.
func TestEnsureFreezesTheTemplatedNameInTheRegistry(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, gitTemplate("{kind}/{change}")))

	first := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"})

	wantReady(t, first)
	record := loadRecord(t, repo, "x")
	if got := record.Repositories[0].Branches[0]; got.Name != "feature/x" || got.Template != "{kind}/{change}" {
		t.Fatalf("rama registrada = %+v", got)
	}

	writeFile(t, filepath.Join(repo, "axiom.yaml"), configWithRoles(map[string][]string{"core": {"."}}, gitTemplate("otra/{change}")))
	second := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"})

	wantReady(t, second)
	branchByName(t, onlyRepository(t, second), "feature/x")
	if listed := runGit(t, repo, "branch", "--list", "otra/*"); listed != "" {
		t.Errorf("la plantilla nueva no debe crear otra rama: %q", listed)
	}
}

// Sin plantilla el registro no lleva la clave `template`.
func TestEnsureWithoutTemplateRecordsNoTemplate(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)

	wantReady(t, ensureOf(t, repo, "x"))

	for _, recorded := range loadRecord(t, repo, "x").Repositories[0].Branches {
		if recorded.Template != "" {
			t.Errorf("rama %+v: sin plantilla no se registra ninguna", recorded)
		}
	}
}

// D-09 (S3): con el registro borrado, la rama pendiente cuyo candidato ya existe se recupera y se
// vuelve a congelar con su plantilla.
func TestEnsureRecoversAPendingBranchThatAlreadyExistsWithoutRegistry(t *testing.T) {
	master, specs, backend := sealedTaskProject(t)
	wantReady(t, ensureSDD(t, Default(), master))
	if listed := runGit(t, backend, "branch", "--list", "feature/*"); listed != "" {
		t.Fatalf("la rama pendiente no se crea sin su Task: %q", listed)
	}
	bindTask(t, specs, "backend", "1235")
	runGit(t, backend, "branch", "feature/TASK-1235")

	result := ensureSDD(t, Default(), master)

	wantReady(t, result)
	got := recordedBackend(t, specs)
	if len(got.Branches) != 1 || got.Branches[0].Name != "feature/TASK-1235" || got.Branches[0].Template != "feature/TASK-{task_ref}" || got.Branches[0].Kind != branch.KindChange {
		t.Errorf("rama recuperada = %+v", got.Branches)
	}
}

// Sin la rama candidata, la rama sigue pendiente aunque ya haya Task: no se crea ni se registra.
func TestEnsureLeavesThePendingBranchWhenTheCandidateDoesNotExist(t *testing.T) {
	master, specs, backend := sealedTaskProject(t)
	bindTask(t, specs, "backend", "1235")

	wantReady(t, ensureSDD(t, Default(), master))

	if listed := runGit(t, backend, "branch", "--list", "feature/*"); listed != "" {
		t.Errorf("no debe crearse la rama pendiente: %q", listed)
	}
	if got := recordedBackend(t, specs); len(got.Branches) != 0 {
		t.Errorf("no debe registrarse la rama pendiente: %+v", got.Branches)
	}
}

// erroringBranchExists hace fallar BranchExists de un nombre dado.
type erroringBranchExists struct {
	gitReader
	name string
}

func (e erroringBranchExists) BranchExists(ctx context.Context, dir, name string) (bool, error) {
	if name == e.name {
		return false, errors.New("fallo simulado de BranchExists")
	}
	return e.gitReader.BranchExists(ctx, dir, name)
}

// G5: un error de BranchExists distinto de «no existe» se propaga y no se trata como ausencia.
func TestEnsurePropagatesABranchExistsFailureDuringRecovery(t *testing.T) {
	master, specs, backend := sealedTaskProject(t)
	bindTask(t, specs, "backend", "1235")
	engine := Default()
	engine.git = erroringBranchExists{gitReader: engine.git, name: "feature/TASK-1235"}

	result := ensureSDD(t, engine, master)

	wantRefusal(t, result, branch.CodeGitFailed, branch.StatusError)
	if !strings.Contains(result.Message, "feature/TASK-1235") {
		t.Errorf("el mensaje debe nombrar la rama: %q", result.Message)
	}
	if listed := runGit(t, backend, "branch", "--list", "feature/*"); listed != "" {
		t.Errorf("no debe crearse nada: %q", listed)
	}
}

// D-09: dos roles del mismo repositorio con el mismo candidato son `invalid_name` y no se crea nada.
func TestEnsureRefusesTwoRolesWithTheSameCandidate(t *testing.T) {
	integrationstest.Install(t, integrationstest.NewFake())
	roles := map[string][]string{"api": {"repo-backend"}, "jobs": {"repo-backend"}}
	master, specs := newMultirepo(t, roles, "repo-backend")
	writeFile(t, filepath.Join(master, "axiom.yaml"), "workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\n"+
		"git:\n  integration_branch: main\n  branch_prefix: inc\n"+
		"roles:\n"+
		"  api:\n    name: api\n    branch_template: \"feat/TASK-{task_ref}\"\n    repositories:\n      - path: repo-backend\n"+
		"  jobs:\n    name: jobs\n    branch_template: \"feat/TASK-{task_ref}\"\n    repositories:\n      - path: repo-backend\n"+
		"integrations:\n  fake:\n    project: p\n")
	sealWithSnapshot(t, specs, "x", snapshotOf(t, master, "x", "api", "jobs"), "api", "jobs")
	writeFile(t, filepath.Join(specs, "openspec", "changes", "x", "integrations.yaml"),
		"schema: axiom.integrations/v1\nchange: x\nlinks:\n"+
			"  - {plugin: fake, kind: task, id: \"7\", role: api, source: tasks.bind, recorded_at: 2026-01-01T00:00:00Z}\n"+
			"  - {plugin: fake, kind: task, id: \"7\", role: jobs, source: tasks.bind, recorded_at: 2026-01-01T00:00:00Z}\n")

	result := ensureSDD(t, Default(), master)

	wantRefusal(t, result, branch.CodeInvalidName, branch.StatusRefused)
	if !strings.Contains(result.Message, "api") || !strings.Contains(result.Message, "jobs") || !strings.Contains(result.Message, "feat/TASK-7") {
		t.Errorf("el mensaje debe nombrar los dos roles y el candidato: %q", result.Message)
	}
	wantNoRegistryFile(t, specs, "x")
}

// G1: la guarda calcula el nombre esperado como el plan, también sin registro: `feature/TASK-1235`
// es correcta; otra rama, no; y un repositorio sin Task espera a su rol.
func TestVerdictForRendersTheTemplateWithoutRegistry(t *testing.T) {
	_, specs, backend := sealedTaskProject(t)
	query := VerdictQuery{Path: backend, Change: "x", Consultant: branch.ConsultantApply}

	awaiting := verdictFor(t, query)

	if awaiting.Kind != branch.VerdictNotApplicable || awaiting.Details["reason"] != branch.PendingReasonAwaitingRole || awaiting.Details["roles"] != "backend" {
		t.Fatalf("sin Task el veredicto es awaiting_role: %+v", awaiting)
	}
	if !strings.Contains(awaiting.Continuation, "--role backend") {
		t.Errorf("continuación = %q", awaiting.Continuation)
	}

	bindTask(t, specs, "backend", "1235")
	runGit(t, backend, "checkout", "-q", "-b", "feature/TASK-1235")
	wantKind(t, verdictFor(t, query), branch.VerdictCorrect)

	runGit(t, backend, "checkout", "-q", "main")
	wrong := verdictFor(t, query)
	wantKind(t, wrong, branch.VerdictWrongBranch)
	if wrong.Expected[0] != "feature/TASK-1235" {
		t.Errorf("esperada = %v", wrong.Expected)
	}
}

// ODD: `{ref}` sale del staging; sin él, la guarda falla cerrado con el rechazo tipado.
func TestVerdictForTheODDLaneRendersFromTheStagingOrFailsClosed(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, gitTemplate("feature/TASK-{ref}")))
	query := VerdictQuery{Path: repo, Change: "mi-feature", Lane: branch.LaneODD, Consultant: branch.ConsultantWrite, Context: branch.ContextOrdinary}

	_, err := Default().VerdictFor(context.Background(), query)

	var refusal *branch.RefusalError
	if !errors.As(err, &refusal) || refusal.Code != branch.CodeBranchTemplateUnresolved {
		t.Fatalf("error = %v, se esperaba branch_template_unresolved", err)
	}

	stageWorkItem(t, repo, "mi-feature", "1235", "")
	runGit(t, repo, "checkout", "-q", "-b", "feature/TASK-1235")
	wantKind(t, verdictFor(t, query), branch.VerdictCorrect)
}

// S9: con plantilla, un ArchivePrefix registrado no cambia la rama de archivado esperada.
func TestExpectedNamesIgnoresTheArchivePrefixWithATemplate(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, extra, want string
	}{
		{"con plantilla", gitTemplate("{kind}/{change}"), "feature/x-archive"},
		{"sin plantilla", "git:\n  integration_branch: main\n  branch_prefix: feat\n", "zz/x-archive"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, test.extra))
			loc := locationOf(t, repo)
			record := &branch.Record{ArchivePrefix: "zz"}

			got, refusal := expectedNames(VerdictQuery{Change: "x", Lane: branch.LaneSDD}, loc, record, nil, guardRepository{
				root: repo, common: commonDir(repo), isPlanning: true,
				planning: branch.RepoFacts{GitRoot: repo, CommonDir: commonDir(repo), RelPath: "."},
			})

			if refusal != nil || got.Archive != test.want {
				t.Errorf("archivo = %q (%v), se esperaba %q", got.Archive, refusal, test.want)
			}
		})
	}
}

// S9: `registeredChangeBranch` no compone `<prefijo>/<cambio>` con plantilla (lo renderiza el
// planificador) y sin plantilla conserva `--prefix`.
func TestRegisteredChangeBranchWithAndWithoutTemplate(t *testing.T) {
	t.Parallel()
	facts := func(extra string) changeFacts {
		repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, extra))
		loc := locationOf(t, repo)
		return changeFacts{input: branch.PlanInput{
			Change: "x", Lane: branch.LaneSDD, Topology: topologyOf(loc),
			PlanningRepo: branch.RepoFacts{GitRoot: repo, CommonDir: commonDir(repo), RelPath: "."},
			Policy:       branch.ResolvePolicy(loc.Config),
			Exceptions:   branch.Exceptions{Prefix: "zz"},
		}}
	}

	if got := registeredChangeBranch(facts(gitTemplate("{kind}/{change}"))); got != "" {
		t.Errorf("con plantilla = %q, se esperaba vacío", got)
	}
	if got := registeredChangeBranch(facts("git:\n  integration_branch: main\n")); got != "zz/x" {
		t.Errorf("sin plantilla = %q, se esperaba zz/x", got)
	}

	withBranch := facts(gitTemplate("{kind}/{change}"))
	withBranch.input.Exceptions.Branch = "mia"
	if got := registeredChangeBranch(withBranch); got != "mia" {
		t.Errorf("con --branch = %q, se esperaba mia", got)
	}
}
