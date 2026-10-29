package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"gopkg.in/yaml.v3"
)

// Pruebas de `Engine.Snapshot` (REQ-25.36, design D-12 y §4.5) con git real, entorno aislado y
// sin red: la instantánea contiene entradas de política (integración, prefijo, base y los
// nombres de rama derivados del cambio, también la de archivado), nunca resultados: ni SHAs ni
// rutas absolutas. Calcularla no escribe nada.

func locationOf(t *testing.T, dir string) workspace.Location {
	t.Helper()
	loc, err := workspace.Locate(dir)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func snapshotOf(t *testing.T, dir, change string, roles ...string) *kickoff.GitSnapshot {
	t.Helper()
	snapshot, err := Default().Snapshot(context.Background(), locationOf(t, dir), change, roles)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snapshot == nil {
		t.Fatal("Snapshot devolvió nil sin error")
	}
	return snapshot
}

// sealWithSnapshot sella el kickoff de un cambio bajo repo con la instantánea dada, como lo
// hace `axiom sdd kickoff seal` desde una rama del cambio.
func sealWithSnapshot(t *testing.T, repo, change string, snapshot *kickoff.GitSnapshot, roles ...string) {
	t.Helper()
	changeRoot := filepath.Join(repo, "openspec", "changes", change)
	var sealed []kickoff.KickoffRole
	for _, role := range roles {
		sealed = append(sealed, kickoff.KickoffRole{Role: role, GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"})
	}
	writeFile(t, filepath.Join(changeRoot, "proposal.md"), "# propuesta\n")
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema:    kickoff.KickoffSchemaV1,
		Change:    change,
		SealedBy:  "prueba",
		Config:    kickoff.FlowConfig{FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit", HandoffPolicy: kickoff.HandoffNone, Roles: sealed},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		Git:       snapshot,
	})
	if err != nil || !ok {
		t.Fatalf("sellar %s con instantánea: %v (publicado: %v)", change, err, ok)
	}
}

// REQ-25.36, escenario «El sellado escribe la instantánea»: integración `main`, prefijo `feat`,
// la base `main` y los nombres `feat/x`, `feat/x-core`, `feat/x-qa` y la rama de archivado
// `feat/x-archive`.
func TestSnapshotNamesTheDerivedBranchesOfAMonorepoWithTwoRoles(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, "git:\n  integration_branch: main\n  branch_prefix: feat\n"))

	got := snapshotOf(t, repo, "x", "core", "qa")

	want := &kickoff.GitSnapshot{
		IntegrationBranch: "main",
		BranchPrefix:      "feat",
		ChangeBranch:      "feat/x",
		Repositories: []kickoff.GitSnapshotRepo{{
			Path:              ".",
			Name:              filepath.Base(repo),
			Role:              "change_branch",
			IntegrationBranch: "main",
			Base:              "main",
			Branches: []kickoff.GitSnapshotBranch{
				{Name: "feat/x", Kind: "change"},
				{Name: "feat/x-core", Kind: "role", Role: "core"},
				{Name: "feat/x-qa", Kind: "role", Role: "qa"},
				{Name: "feat/x-archive", Kind: "archive"},
			},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("instantánea = %+v, se esperaba %+v", got, want)
	}
	k := kickoff.Kickoff{Git: got}
	data, err := yaml.Marshal(k)
	if err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`\b[0-9a-f]{40}\b`).Match(data) || strings.Contains(string(data), filepath.ToSlash(repo)) || strings.Contains(string(data), repo) {
		t.Errorf("la instantánea no lleva SHAs ni rutas absolutas:\n%s", data)
	}
}

// Con un único rol solo existe la rama del cambio (REQ-25.4) y, además, la de archivado; la
// política del repositorio (prefijo e integración) es la de axiom.yaml.
func TestSnapshotOfASingleRoleHasOnlyTheChangeAndArchiveBranches(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, "git:\n  integration_branch: develop\n  branch_prefix: inc\n"))

	got := snapshotOf(t, repo, "x", "core")

	if got.IntegrationBranch != "develop" || got.BranchPrefix != "inc" || got.ChangeBranch != "inc/x" || got.Exceptions != nil {
		t.Errorf("política = %+v, se esperaba develop, inc e inc/x sin excepciones", got)
	}
	repository := got.Repositories[0]
	if repository.Base != "develop" || repository.IntegrationBranch != "develop" {
		t.Errorf("base %q e integración %q, se esperaba develop", repository.Base, repository.IntegrationBranch)
	}
	want := []kickoff.GitSnapshotBranch{{Name: "inc/x", Kind: "change"}, {Name: "inc/x-archive", Kind: "archive"}}
	if !reflect.DeepEqual(repository.Branches, want) {
		t.Errorf("ramas = %+v, se esperaba %+v", repository.Branches, want)
	}
}

// REQ-25.36, escenario «La instantánea refleja las excepciones por cambio»: tras
// `ensure --change x --branch inc/x` la instantánea nombra `inc/x` y deriva de ella las ramas de
// rol y la de archivado; las excepciones quedan registradas en el bloque.
func TestSnapshotReflectsTheRegisteredExceptions(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))
	wantReady(t, ensureException(t, repo, "x", branch.Exceptions{Branch: "inc/x", Base: "main"}))

	got := snapshotOf(t, repo, "x", "core", "qa")

	if got.ChangeBranch != "inc/x" || !reflect.DeepEqual(got.Exceptions, &kickoff.GitExceptions{Branch: "inc/x", Base: "main"}) {
		t.Errorf("rama del cambio %q y excepciones %+v, se esperaba inc/x con rama y base", got.ChangeBranch, got.Exceptions)
	}
	var names []string
	for _, planned := range got.Repositories[0].Branches {
		names = append(names, planned.Name)
	}
	wantNamesEqual(t, "ramas", names, "inc/x", "inc/x-core", "inc/x-qa", "inc/x-archive")
	if got.Repositories[0].Base != "main" {
		t.Errorf("base = %q, se esperaba la fijada con --base: main", got.Repositories[0].Base)
	}
}

// REQ-25.36 y REQ-25.5: en un multirepo el repositorio de specs entra sin ramas (se trabaja in
// situ), cada repositorio de código lleva la rama del cambio desde su propia integración, y no hay
// rama de archivado. Las rutas son relativas a la carpeta maestra.
func TestSnapshotOfAMultirepoListsTheSpecsRepositoryWithoutBranches(t *testing.T) {
	t.Parallel()
	master, _ := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}, "frontend": {"repo-frontend"}}, "repo-backend", "repo-frontend")

	got := snapshotOf(t, master, "x", "backend", "frontend")

	if got.ChangeBranch != "feat/x" || got.Exceptions != nil {
		t.Errorf("rama del cambio %q y excepciones %+v", got.ChangeBranch, got.Exceptions)
	}
	if len(got.Repositories) != 3 {
		t.Fatalf("repositorios = %+v, se esperaban specs, backend y frontend", got.Repositories)
	}
	specs, backend, frontend := got.Repositories[0], got.Repositories[1], got.Repositories[2]
	if specs.Path != "repo-specs" || specs.Role != "specs_in_place" || len(specs.Branches) != 0 || specs.Base != "main" {
		t.Errorf("repositorio de specs = %+v, se esperaba in situ, sin ramas y con base main", specs)
	}
	for _, code := range []kickoff.GitSnapshotRepo{backend, frontend} {
		if code.Role != "code" || code.IntegrationBranch != "main" || code.Base != "main" ||
			!reflect.DeepEqual(code.Branches, []kickoff.GitSnapshotBranch{{Name: "feat/x", Kind: "change"}}) {
			t.Errorf("repositorio de código = %+v, se esperaba la rama del cambio sin archivado", code)
		}
	}
	if backend.Path != "repo-backend" || frontend.Path != "repo-frontend" {
		t.Errorf("rutas %q y %q, se esperaban las relativas a la carpeta maestra", backend.Path, frontend.Path)
	}
}

// REQ-25.36: la política por repositorio de `repositories:` entra en su entrada.
func TestSnapshotUsesThePerRepositoryPolicy(t *testing.T) {
	t.Parallel()
	master, _ := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	writeFile(t, filepath.Join(master, "axiom.yaml"), configFor("multirepo", "repo-specs", map[string][]string{"backend": {"repo-backend"}},
		"git:\n  integration_branch: main\nrepositories:\n  - path: repo-backend\n    integration_branch: develop\n    branch_prefix: back\n"))
	runGit(t, filepath.Join(master, "repo-backend"), "branch", "develop")

	got := snapshotOf(t, master, "x", "backend")

	backend := got.Repositories[1]
	if backend.IntegrationBranch != "develop" || backend.Base != "develop" || backend.Branches[0].Name != "back/x" {
		t.Errorf("repositorio de backend = %+v, se esperaba develop y back/x", backend)
	}
}

// D-12: calcular la instantánea no escribe nada: ni ramas ni worktrees ni registro, y solo
// lanza lecturas de git.
func TestSnapshotWritesNothing(t *testing.T) {
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))
	state := repositoryState(t, repo)
	seen := observe(t)

	snapshotOf(t, repo, "x", "core", "qa")

	wantNoWrites(t, *seen, "calcular la instantánea es de solo lectura")
	if got := repositoryState(t, repo); got != state {
		t.Errorf("la instantánea cambió el repositorio:\n%s\n---\n%s", state, got)
	}
	if record, err := Default().registry.Load(commonDir(repo), branch.LaneSDD, "x"); err != nil || record != nil {
		t.Errorf("la instantánea no escribe el registro: %+v, %v", record, err)
	}
}

// REQ-25.36 y D-04: un nombre inválido no llega a la instantánea: se rechaza con el código
// tipado del planificador (aquí, un rol `archive` que colisiona con la rama de archivado).
func TestSnapshotRefusesAnInvalidNameInsteadOfWritingIt(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "archive": {"."}}, ""))

	_, err := Default().Snapshot(context.Background(), locationOf(t, repo), "x", []string{"core", "archive"})

	var refusal *branch.RefusalError
	if !errors.As(err, &refusal) || refusal.Code != branch.CodeInvalidName || refusal.Continuation == "" {
		t.Fatalf("error = %v, se esperaba un rechazo tipado invalid_name con continuación", err)
	}
}

// D-12: con un kickoff cuya instantánea nombra `feat/x`, un cambio de prefijo en la política
// posterior al sellado no cambia los nombres esperados: la instantánea prevalece sobre la
// política, en el veredicto y en el plan de `status`.
func TestTheSnapshotPrevailsOverALaterPolicyChange(t *testing.T) {
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, "git:\n  branch_prefix: inc\n"))
	sealWithSnapshot(t, repo, "x", &kickoff.GitSnapshot{
		IntegrationBranch: "main",
		BranchPrefix:      "feat",
		ChangeBranch:      "feat/x",
		Repositories: []kickoff.GitSnapshotRepo{{
			Path: ".", Name: filepath.Base(repo), Role: "change_branch", IntegrationBranch: "main", Base: "main",
			Branches: []kickoff.GitSnapshotBranch{
				{Name: "feat/x", Kind: "change"},
				{Name: "feat/x-core", Kind: "role", Role: "core"},
				{Name: "feat/x-qa", Kind: "role", Role: "qa"},
				{Name: "feat/x-archive", Kind: "archive"},
			},
		}},
	}, "core", "qa")
	commitAll(t, repo, "kickoff con instantánea")

	t.Run("status planifica con los nombres de la instantánea", func(t *testing.T) {
		result := status(t, repo, "x", branch.LaneSDD)
		wantReached(t, result)
		var names []string
		for _, planned := range onlyRepository(t, result).Branches {
			names = append(names, planned.Name)
		}
		wantNamesEqual(t, "ramas", names, "feat/x", "feat/x-core", "feat/x-qa")
	})

	t.Run("el veredicto espera los nombres de la instantánea", func(t *testing.T) {
		runGit(t, repo, "switch", "-q", "-c", "inc/x")
		wrong := verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"})
		wantKind(t, wrong, branch.VerdictWrongBranch)
		wantNamesEqual(t, "ramas esperadas", wrong.Expected, "feat/x", "feat/x-core", "feat/x-qa")
		for _, correct := range []string{"feat/x", "feat/x-qa"} {
			runGit(t, repo, "switch", "-q", "-c", correct)
			wantKind(t, verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"}), branch.VerdictCorrect)
			runGit(t, repo, "switch", "-q", "main")
		}
	})
}

// D-10: la rama de archivado sale del registro si `ensure --archive` recibió `--prefix`; si no,
// de la instantánea; si no, de `<rama del cambio>-archive`.
func TestVerdictForTheArchiveBranchPrefersTheRegistryThenTheSnapshot(t *testing.T) {
	repo := embeddedCoreProject(t)
	sealWithSnapshot(t, repo, "x", &kickoff.GitSnapshot{
		IntegrationBranch: "main", BranchPrefix: "feat", ChangeBranch: "feat/x",
		Repositories: []kickoff.GitSnapshotRepo{{
			Path: ".", Name: filepath.Base(repo), Role: "change_branch", IntegrationBranch: "main", Base: "main",
			Branches: []kickoff.GitSnapshotBranch{{Name: "feat/x", Kind: "change"}, {Name: "feat/x-archive", Kind: "archive"}},
		}},
	}, "core")
	commitAll(t, repo, "kickoff con instantánea")
	archive := VerdictQuery{Path: changeRootOf(repo), Change: "x", Context: branch.ContextArchive}

	runGit(t, repo, "switch", "-q", "-c", "feat/x-archive")
	wantKind(t, verdictFor(t, archive), branch.VerdictCorrect)

	record := registeredRecord(repo, branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange, InPlace: true})
	record.ArchivePrefix = "doc"
	writeRegistry(t, commonDir(repo), record)
	wrong := verdictFor(t, archive)
	wantKind(t, wrong, branch.VerdictWrongBranch)
	wantNamesEqual(t, "rama de archivado esperada", wrong.Expected, "doc/x-archive")
}

// REQ-25.37: un kickoff con instantánea ya no está exento: desde `main` es `wrong_branch`;
// uno retro-sellado (sin instantánea) sigue exento.
func TestVerdictForAChangeWithASnapshotIsNoLongerExempt(t *testing.T) {
	repo := embeddedCoreProject(t)
	sealWithSnapshot(t, repo, "x", snapshotOf(t, repo, "x", "core"), "core")
	sealChange(t, repo, "y", "core")
	commitAll(t, repo, "kickoffs")

	wantKind(t, verdictFor(t, VerdictQuery{Path: changeRootOf(repo), Change: "x"}), branch.VerdictWrongBranch)
	wantKind(t, verdictFor(t, VerdictQuery{Path: filepath.Join(repo, "openspec", "changes", "y"), Change: "y"}), branch.VerdictNotApplicable)
}

// REQ-25.18 (multirepo) con instantánea: el repositorio de specs sigue exigiendo su integración y
// un repositorio de código sin entrada en la instantánea se evalúa con los nombres de la política.
func TestVerdictForAMultirepoWithASnapshotKeepsTheSpecsIntegrationRule(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	// Como en un multirepo real, el repositorio de specs lleva su propia configuración: desde dentro
	// de su `openspec/` el marcador más cercano es el suyo.
	writeFile(t, filepath.Join(specs, "axiom.yaml"), configFor("multirepo", "repo-specs", map[string][]string{"backend": {"repo-backend"}}, ""))
	backend := filepath.Join(master, "repo-backend")
	snapshot := snapshotOf(t, master, "x", "backend")
	sealWithSnapshot(t, specs, "x", snapshot, "backend")
	specsChange := filepath.Join(specs, "openspec", "changes", "x")

	wantKind(t, verdictFor(t, VerdictQuery{Path: specsChange, Change: "x"}), branch.VerdictCorrect)
	runGit(t, specs, "switch", "-q", "-c", "feat/x-backend")
	wantKind(t, verdictFor(t, VerdictQuery{Path: specsChange, Change: "x"}), branch.VerdictWrongBranch)

	applying := verdictFor(t, VerdictQuery{Path: backend, Change: "x", Consultant: branch.ConsultantApply})
	wantKind(t, applying, branch.VerdictWrongBranch)
	wantNamesEqual(t, "ramas esperadas del backend", applying.Expected, "feat/x")
}

// D-25 (paso 3): la rama del cambio de la que deriva la de archivado sale del registro, de la
// instantánea o de la política, en ese orden.
func TestRegisteredChangeBranchFollowsRegistrySnapshotThenPolicy(t *testing.T) {
	t.Parallel()
	gitDir := filepath.Join(t.TempDir(), ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	snapshot := &kickoff.GitSnapshot{
		ChangeBranch: "inc/x",
		Repositories: []kickoff.GitSnapshotRepo{{Path: ".", Name: "p", Role: "change_branch", IntegrationBranch: "main", Base: "main",
			Branches: []kickoff.GitSnapshotBranch{{Name: "inc/x", Kind: "change"}}}},
	}
	facts := func(snapshot *kickoff.GitSnapshot, record *branch.Record) changeFacts {
		return changeFacts{
			record: record,
			input: branch.PlanInput{
				Change:       "x",
				PlanningRepo: branch.RepoFacts{GitRoot: filepath.Dir(gitDir), CommonDir: gitDir, RelPath: "."},
				Snapshot:     snapshot,
			},
		}
	}
	recorded := &branch.Record{Repositories: []branch.RecordRepo{{CommonDir: gitDir, Branches: []branch.RecordBranch{{Name: "reg/x", Kind: branch.KindChange}}}}}

	if got := registeredChangeBranch(facts(snapshot, recorded)); got != "reg/x" {
		t.Errorf("con registro y con instantánea = %q, se esperaba el del registro reg/x", got)
	}
	if got := registeredChangeBranch(facts(snapshot, nil)); got != "inc/x" {
		t.Errorf("solo con instantánea = %q, se esperaba inc/x", got)
	}
	if got := registeredChangeBranch(facts(nil, nil)); got != "" {
		t.Errorf("sin registro ni instantánea = %q, se esperaba vacío (la compone la política)", got)
	}
}
