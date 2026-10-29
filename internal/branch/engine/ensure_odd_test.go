package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// Carril ODD de `ensure` (REQ-25.13, design D-04): la misma política que SDD sin roles, sin
// kickoff y sin archivado. En un monorepo prepara `<prefijo>/<feature>` y su worktree; en un
// multirepo trabaja in situ en el repositorio de specs y solo incluye los repositorios de código
// que se piden con `--repo`, cada uno con solo la rama de la feature. Con git real, entorno
// aislado y sin red.

func ensureOdd(t *testing.T, cwd, feature string, request EnsureRequest) branch.Result {
	t.Helper()
	request.CWD, request.Change, request.Lane = cwd, feature, branch.LaneODD
	return ensureWith(t, hubWorktree, request)
}

// wantOddDocument comprueba que el documento ODD es `<root>/odd/tasks/<feature>.md`: el fichero no
// existe, así que se compara por su carpeta raíz.
func wantOddDocument(t *testing.T, result branch.Result, root, feature string) {
	t.Helper()
	if filepath.Base(result.OddDocument) != feature+".md" || filepath.Base(filepath.Dir(result.OddDocument)) != "tasks" || filepath.Base(filepath.Dir(filepath.Dir(result.OddDocument))) != "odd" {
		t.Errorf("oddDocument = %q, se esperaba <raíz>/odd/tasks/%s.md", result.OddDocument, feature)
		return
	}
	wantSameDirectory(t, "raíz del documento ODD", filepath.Dir(filepath.Dir(filepath.Dir(result.OddDocument))), root)
}

func wantNoOpenSpecChanges(t *testing.T, root string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, "openspec", "changes")); err == nil {
		t.Errorf("el carril ODD no crea kickoff ni nada bajo openspec/changes en %s", root)
	}
}

// REQ-25.13, escenarios «ODD en monorepo prepara rama y worktree» y «ODD no crea
// `kickoff.yaml`»: `feat/mi-feature` con su worktree, y el JSON indica
// `<worktree>/odd/tasks/mi-feature.md`; no se crea ningún fichero bajo `openspec/changes/` ni en
// el checkout ni en el worktree, y el registro es el del carril odd.
func TestEnsureOddLaneInAMonorepoPreparesTheBranchAndItsWorktree(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)

	result := ensureOdd(t, repo, "mi-feature", EnsureRequest{})

	wantReady(t, result)
	if result.Lane != branch.LaneODD || result.Change != "mi-feature" {
		t.Errorf("carril %q, cambio %q", result.Lane, result.Change)
	}
	repository := onlyRepository(t, result)
	feature := branchByName(t, repository, "feat/mi-feature")
	worktree := plannedWorktree(t, repo, "mi-feature", filepath.Base(repo))
	if len(repository.Branches) != 1 || !feature.Exists || feature.Kind != branch.KindChange {
		t.Fatalf("ramas = %+v, se esperaba solo feat/mi-feature creada", repository.Branches)
	}
	wantSameDirectory(t, "worktree de la feature", feature.Worktree, worktree)
	if got := runGit(t, worktree, "symbolic-ref", "--short", "HEAD"); got != "feat/mi-feature" {
		t.Errorf("el worktree tiene %s, se esperaba feat/mi-feature", got)
	}
	wantOddDocument(t, result, worktree, "mi-feature")
	wantNoOpenSpecChanges(t, repo)
	wantNoOpenSpecChanges(t, worktree)
	if got := loadRecordAt(t, repo, branch.LaneODD, "mi-feature"); got.Lane != branch.LaneODD {
		t.Errorf("carril del registro = %q, se esperaba odd", got.Lane)
	}
	wantNoRegistryFile(t, repo, "mi-feature")
}

// REQ-25.13, escenario «ODD en multirepo trabaja in situ en el repositorio de specs»: no se crea
// rama ni worktree en el repositorio de specs, los repositorios de código no entran al plan sin
// `--repo` aunque haya roles declarados y el documento ODD está en el repositorio de specs.
func TestEnsureOddLaneInAMultirepoWorksInPlaceInTheSpecsRepository(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	specsState := inPlaceState(t, specs)
	backendState := inPlaceState(t, filepath.Join(master, "repo-backend"))
	seen := observe(t)

	result := ensureOdd(t, master, "mi-feature", EnsureRequest{})

	wantReady(t, result)
	wantNoWriteIn(t, *seen, specs)
	wantNoWriteIn(t, *seen, filepath.Join(master, "repo-backend"))
	if len(result.Repositories) != 1 || result.Repositories[0].Role != branch.RepoSpecsInPlace || len(result.Repositories[0].Branches) != 0 {
		t.Fatalf("repositorios = %+v, se esperaba solo el de specs, in situ y sin ramas", result.Repositories)
	}
	wantOddDocument(t, result, specs, "mi-feature")
	wantSameDirectory(t, "workspaceRoot", result.WorkspaceRoot, specs)
	if got := inPlaceState(t, specs); got != specsState {
		t.Errorf("el repositorio de specs no cambia:\n%s\n---\n%s", specsState, got)
	}
	if got := inPlaceState(t, filepath.Join(master, "repo-backend")); got != backendState {
		t.Errorf("repo-backend no entra al plan sin --repo:\n%s\n---\n%s", backendState, got)
	}
	wantNoOpenSpecChanges(t, specs)
}

// REQ-25.13, escenario «Un repositorio de código se incluye con `--repo`»: el repositorio indicado
// recibe solo `<prefijo>/<feature>`, sin ramas de rol aunque varios roles lo declaren; el que no
// se indica no entra; repetir sin `--repo` recompone el mismo estado desde el registro (mismo
// JSON, ninguna escritura); y una ruta que no existe se omite con `not_a_git_repository`.
func TestEnsureOddLaneRepoFlagIncludesCodeRepositoriesWithOnlyTheFeatureBranch(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}, "qa": {"repo-backend"}, "frontend": {"repo-frontend"}}, "repo-backend", "repo-frontend")
	backend, frontend := filepath.Join(master, "repo-backend"), filepath.Join(master, "repo-frontend")
	frontendState := inPlaceState(t, frontend)

	first := ensureOdd(t, master, "mi-feature", EnsureRequest{Mode: branch.ModeInPlace, Repos: []string{"repo-backend"}})

	wantReady(t, first)
	var names, roles []string
	for _, repository := range first.Repositories {
		names, roles = append(names, repository.Name), append(roles, string(repository.Role))
	}
	if want := []string{"repo-specs", "repo-backend"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("repositorios = %v, se esperaba %v", names, want)
	}
	if want := []string{"specs_in_place", "code"}; !reflect.DeepEqual(roles, want) {
		t.Errorf("papeles = %v, se esperaba %v", roles, want)
	}
	included := first.Repositories[1]
	if len(included.Branches) != 1 || included.Branches[0].Name != "feat/mi-feature" || !included.Branches[0].Exists || len(included.Roles) != 0 {
		t.Errorf("repo-backend = %+v, se esperaba solo feat/mi-feature, sin roles", included)
	}
	if got := runGit(t, backend, "branch", "--list", "feat/*"); strings.Count(got, "feat/") != 1 {
		t.Errorf("repo-backend solo tiene la rama de la feature, hay:\n%s", got)
	}
	if got := inPlaceState(t, frontend); got != frontendState {
		t.Errorf("repo-frontend no se pidió con --repo y no cambia:\n%s\n---\n%s", frontendState, got)
	}
	wantOddDocument(t, first, specs, "mi-feature")
	wantNoOpenSpecChanges(t, specs)

	t.Run("repetir sin --repo recompone el mismo estado desde el registro", func(t *testing.T) {
		seen := observe(t)

		again := ensureOdd(t, master, "mi-feature", EnsureRequest{})

		wantReady(t, again)
		if got, want := resultJSON(t, again), resultJSON(t, first); got != want {
			t.Errorf("repetir debe dar el mismo JSON:\n%s\n---\n%s", want, got)
		}
		wantNoWrites(t, *seen, "repetir no escribe nada")
	})
	t.Run("una ruta que no existe se omite", func(t *testing.T) {
		result := ensureOdd(t, master, "otra", EnsureRequest{Mode: branch.ModeInPlace, Repos: []string{"no-existe"}})

		wantReady(t, result)
		if len(result.Repositories) != 1 || len(result.Skipped) != 1 || result.Skipped[0].Code != branch.CodeNotAGitRepository || !strings.Contains(result.Skipped[0].Message, "no-existe") {
			t.Errorf("repositorios %+v, omitidos %+v; se esperaba solo el de specs y no-existe omitido", result.Repositories, result.Skipped)
		}
	})
}

// REQ-25.13 y REQ-25.8: en un monorepo, `--repo` incluye otro repositorio con su propio worktree
// y solo la rama de la feature; el repositorio del planning home no se repite si se indica a sí
// mismo (`.`).
func TestEnsureOddLaneRepoFlagInAMonorepoIncludesTheOtherRepositoryOnce(t *testing.T) {
	t.Parallel()
	requireGit(t)
	parent := t.TempDir()
	repo, nested := filepath.Join(parent, "proyecto"), filepath.Join(parent, "servicio")
	for _, dir := range []string{repo, nested} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		initRepository(t, dir)
	}
	writeFile(t, filepath.Join(repo, "axiom.yaml"), configWithRoles(map[string][]string{"core": {"."}}, ""))
	runGit(t, repo, "add", "axiom.yaml")
	commit(t, repo, "configuración")

	result := ensureOdd(t, repo, "mi-feature", EnsureRequest{Mode: branch.ModeInPlace, Repos: []string{".", "../servicio"}})

	wantReady(t, result)
	var names []string
	for _, repository := range result.Repositories {
		names = append(names, repository.Name)
	}
	if want := []string{filepath.Base(repo), "servicio"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("repositorios = %v, se esperaba %v (sin repetir el del planning home)", names, want)
	}
	for _, repository := range result.Repositories {
		if len(repository.Branches) != 1 || repository.Branches[0].Name != "feat/mi-feature" || !repository.Branches[0].Exists {
			t.Errorf("%s: ramas %+v, se esperaba solo feat/mi-feature", repository.Name, repository.Branches)
		}
	}
	if got := currentBranch(t, nested); got != "feat/mi-feature" {
		t.Errorf("servicio está en %q, se esperaba feat/mi-feature", got)
	}
}
