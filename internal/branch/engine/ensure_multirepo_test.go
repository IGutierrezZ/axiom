package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// `ensure` en un multirepo (REQ-25.5, REQ-25.6, REQ-25.11, design D-04 y D-05): el repositorio de
// specs se trabaja in situ (nunca recibe ramas, worktrees ni checkout) y cada repositorio de
// código recibe `<prefijo>/<cambio>` desde su propia integración. Con git real, entorno aislado y
// sin red. Salvo la prueba de las rutas del JSON, usan `in_place`, que no crea worktrees.

// wantNoWriteIn comprueba que ninguna escritura de ramas ni de worktrees se lanzó con dir dentro
// de repo.
func wantNoWriteIn(t *testing.T, seen [][]string, repo string) {
	t.Helper()
	for _, invocation := range seen {
		if isWriteInvocation(invocation) && pathidentity.Contains(repo, invocation[0]) {
			t.Errorf("el repositorio %s se trabaja in situ y no admite escrituras, pero se lanzó %v", repo, invocation)
		}
	}
}

// detachableState es el estado de un repositorio que `ensure` no puede cambiar, también con HEAD
// desacoplado (inPlaceState lee la rama simbólica de HEAD, que entonces no existe).
func detachableState(t *testing.T, repo string) string {
	t.Helper()
	return repositoryState(t, repo) + "\n==\n" + strings.Join([]string{
		runGit(t, repo, "rev-parse", "HEAD"),
		runGit(t, repo, "ls-files", "--stage"),
		runGit(t, repo, "status", "--porcelain=v1", "--untracked-files=all"),
	}, "\n--\n")
}

func wantAbsolute(t *testing.T, what, path string) {
	t.Helper()
	if !filepath.IsAbs(path) {
		t.Errorf("%s = %q, se esperaba una ruta absoluta", what, path)
	}
}

// REQ-25.5 (escenario «El repositorio de specs de un multirepo no recibe ramas ni worktrees»),
// REQ-25.11 y el JSON con rutas absolutas: el de specs conserva `main` desplegada sin ramas ni
// worktrees y el resultado lo lista in situ con su raíz git absoluta y su integración; el de
// código crea su worktree en la raíz de worktrees, fuera de la carpeta maestra; el registro
// guarda ambos repositorios.
func TestEnsureMultirepoWorksTheSpecsRepositoryInPlaceAndListsEveryRepositoryWithAbsolutePaths(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	sealChangeAt(t, specs, "x", "backend")
	specsState := inPlaceState(t, specs)
	seen := observe(t)

	result := ensureOf(t, master, "x")

	wantReady(t, result)
	wantNoWriteIn(t, *seen, specs)
	if got := inPlaceState(t, specs); got != specsState {
		t.Errorf("el repositorio de specs se trabaja in situ y no cambia:\n%s\n---\n%s", specsState, got)
	}
	if got := runGit(t, specs, "branch", "--list"); !strings.Contains(got, "main") || strings.Count(got, "\n") != 0 {
		t.Errorf("el repositorio de specs solo tiene main, hay:\n%s", got)
	}
	if got := runGit(t, specs, "worktree", "list", "--porcelain"); strings.Count(got, "worktree ") != 1 {
		t.Errorf("el repositorio de specs no tiene worktrees:\n%s", got)
	}

	if len(result.Repositories) != 2 {
		t.Fatalf("repositorios = %+v, se esperaban el de specs y el de código", result.Repositories)
	}
	specsResult, backend := result.Repositories[0], result.Repositories[1]
	if specsResult.Role != branch.RepoSpecsInPlace || specsResult.RoleLabel == "" || specsResult.IntegrationBranch != "main" || specsResult.CurrentBranch != "main" || len(specsResult.Branches) != 0 {
		t.Errorf("repositorio de specs = %+v, se esperaba in situ, en main y sin ramas", specsResult)
	}
	wantSameDirectory(t, "gitRoot del repositorio de specs", specsResult.GitRoot, specs)
	wantAbsolute(t, "commonDir del repositorio de specs", specsResult.CommonDir)
	wantSameDirectory(t, "workspaceRoot", result.WorkspaceRoot, specs)

	worktree := plannedWorktree(t, master, "x", "repo-backend")
	if backend.Role != branch.RepoCode {
		t.Errorf("papel de repo-backend = %q, se esperaba code", backend.Role)
	}
	wantSameDirectory(t, "gitRoot de repo-backend", backend.GitRoot, filepath.Join(master, "repo-backend"))
	wantAbsolute(t, "commonDir de repo-backend", backend.CommonDir)
	created := branchByName(t, backend, "feat/x")
	wantSameDirectory(t, "worktree de feat/x", created.Worktree, worktree)
	if pathidentity.Contains(master, created.Worktree) {
		t.Errorf("el worktree %s debe quedar fuera de la carpeta maestra %s", created.Worktree, master)
	}
	wantSameDirectory(t, "worktreeRoot", result.WorktreeRoot, branch.DefaultWorktreeRoot(canonicalDirectory(t, master)))

	record := loadRecordAt(t, specs, branch.LaneSDD, "x")
	if len(record.Repositories) != 2 || record.Repositories[0].Role != branch.RepoSpecsInPlace || len(record.Repositories[0].Branches) != 0 {
		t.Errorf("registro = %+v, se esperaban el repositorio de specs sin ramas y el de código", record.Repositories)
	}
	wantAbsolute(t, "planningRepo del registro", record.PlanningRepo)
}

// REQ-25.5 (escenarios «Repositorio de specs fuera de su rama de integración» y «Repositorio de
// specs en una rama de rol del cambio»): `ensure` no hace checkout en el repositorio de specs;
// fuera de su integración (una rama ajena, una rama de rol del cambio o HEAD desacoplado) rechaza
// con `specs_repo_not_on_integration`, nombra la rama actual y la integración y ofrece una
// continuación ejecutable, sin crear ramas ni registro en ningún repositorio.
func TestEnsureMultirepoRefusesASpecsRepositoryOutsideItsIntegrationBranch(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	sealChangeAt(t, specs, "x", "backend")
	backend := filepath.Join(master, "repo-backend")
	runGit(t, specs, "branch", "feat/otro")
	runGit(t, specs, "branch", "feat/x-backend")
	backendState := inPlaceState(t, backend)

	for name, tc := range map[string]struct {
		checkout string
		detach   bool
		current  string // lo que debe nombrar el mensaje
		branch   string // detalle `branch`: vacío con HEAD desacoplado
		lane     branch.Lane
	}{
		"otra rama":              {checkout: "feat/otro", current: "feat/otro", branch: "feat/otro", lane: branch.LaneSDD},
		"rama de rol del cambio": {checkout: "feat/x-backend", current: "feat/x-backend", branch: "feat/x-backend", lane: branch.LaneSDD},
		"HEAD desacoplado":       {checkout: "main", detach: true, current: "desacoplado", lane: branch.LaneSDD},
		"carril odd":             {checkout: "feat/otro", current: "feat/otro", branch: "feat/otro", lane: branch.LaneODD},
	} {
		t.Run(name, func(t *testing.T) {
			args := []string{"switch", "-q", tc.checkout}
			if tc.detach {
				args = []string{"switch", "-q", "--detach", tc.checkout}
			}
			runGit(t, specs, args...)
			t.Cleanup(func() { runGit(t, specs, "switch", "-q", "main") })
			specsState := detachableState(t, specs)
			request := EnsureRequest{CWD: master, Change: "x", Lane: tc.lane}
			wantLane := ""
			if tc.lane == branch.LaneODD {
				wantLane = " --lane odd"
			}
			seen := observe(t)

			result := ensureWith(t, hubWorktree, request)

			wantRefusal(t, result, branch.CodeSpecsRepoNotOnIntegration, branch.StatusRefused)
			for _, want := range []string{tc.current, "main", "repo-specs"} {
				if !strings.Contains(result.Message, want) {
					t.Errorf("el mensaje %q debe contener %q", result.Message, want)
				}
			}
			for _, want := range []string{
				"git -C " + pathquote.Quote(canonicalDirectory(t, specs)) + " switch main",
				"axiom branch ensure --change " + request.Change + wantLane + " --cwd " + pathquote.Quote(master),
			} {
				if !strings.Contains(result.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
				}
			}
			if result.Details["branch"] != tc.branch || result.Details["expected"] != "main" || !pathidentity.SameDirectory(result.Details["repository"], specs) {
				t.Errorf("details = %v, se esperaban el repositorio de specs, la rama actual %q y la esperada main", result.Details, tc.branch)
			}
			wantNoWrites(t, *seen, "el rechazo no escribe nada")
			if got := detachableState(t, specs); got != specsState {
				t.Errorf("no se hace checkout en el repositorio de specs:\n%s\n---\n%s", specsState, got)
			}
			if got := inPlaceState(t, backend); got != backendState {
				t.Errorf("ningún repositorio de código cambia:\n%s\n---\n%s", backendState, got)
			}
			if _, err := os.Stat(filepath.Join(commonDir(specs), "axiom", "branch", "v1", string(request.Lane), request.Change+".json")); err == nil {
				t.Error("no debía escribirse el registro")
			}
		})
	}
}

// REQ-25.5 (repositorios de código) y REQ-25.3: cada repositorio de código con un rol recibe
// `<prefijo>/<cambio>` desde su propia integración, con su propia política (`repositories:`); el
// de specs no recibe nada y su suciedad no cuenta en `in_place` (el kickoff sellado queda sin
// seguimiento en él), mientras que la de un repositorio de código sí: se comprueba antes de
// cambiar de rama en ninguno.
func TestEnsureMultirepoGivesEveryCodeRepositoryTheChangeBranchFromItsOwnIntegration(t *testing.T) {
	t.Parallel()
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}, "frontend": {"repo-frontend"}}, "repo-backend", "repo-frontend")
	writeFile(t, filepath.Join(master, "axiom.yaml"), configFor("multirepo", "repo-specs", map[string][]string{"backend": {"repo-backend"}, "frontend": {"repo-frontend"}},
		"git:\n  integration_branch: main\nrepositories:\n  - path: repo-backend\n    integration_branch: develop\n    branch_prefix: back\n"))
	backend, frontend := filepath.Join(master, "repo-backend"), filepath.Join(master, "repo-frontend")
	runGit(t, backend, "switch", "-q", "-c", "develop")
	commit(t, backend, "desarrollo")
	develop := runGit(t, backend, "rev-parse", "HEAD")
	runGit(t, backend, "switch", "-q", "main")
	mainOfFrontend := runGit(t, frontend, "rev-parse", "HEAD")
	sealChangeAt(t, specs, "x", "backend", "frontend")
	specsState := inPlaceState(t, specs)
	backendState := inPlaceState(t, backend)

	t.Run("un repositorio de código sucio impide cambiar de rama en todos", func(t *testing.T) {
		writeFile(t, filepath.Join(frontend, "suelto.txt"), "sin añadir\n")
		t.Cleanup(func() { resetTree(t, frontend) })

		result := ensureInPlace(t, master, "x")

		wantRefusal(t, result, branch.CodeDirtyWorktree, branch.StatusRefused)
		if !strings.Contains(result.Message, "repo-frontend") || !strings.Contains(result.Message, "suelto.txt") {
			t.Errorf("el mensaje %q debe nombrar el repositorio sucio y su fichero", result.Message)
		}
		if got := inPlaceState(t, backend); got != backendState {
			t.Errorf("repo-backend no debía cambiar si repo-frontend está sucio:\n%s\n---\n%s", backendState, got)
		}
	})
	t.Run("con los repositorios de código limpios cada uno parte de su integración", func(t *testing.T) {
		result := ensureInPlace(t, master, "x")

		wantReady(t, result)
		if got := currentBranch(t, backend); got != "back/x" {
			t.Errorf("repo-backend está en %q, se esperaba back/x", got)
		}
		if got := runGit(t, backend, "rev-parse", "refs/heads/back/x"); got != develop {
			t.Errorf("back/x apunta a %s, se esperaba develop (%s)", got, develop)
		}
		if got := currentBranch(t, frontend); got != "feat/x" {
			t.Errorf("repo-frontend está en %q, se esperaba feat/x", got)
		}
		if got := runGit(t, frontend, "rev-parse", "refs/heads/feat/x"); got != mainOfFrontend {
			t.Errorf("feat/x apunta a %s, se esperaba main (%s)", got, mainOfFrontend)
		}
		if got := inPlaceState(t, specs); got != specsState {
			t.Errorf("el repositorio de specs no cambia:\n%s\n---\n%s", specsState, got)
		}
		var roles []string
		for _, repository := range result.Repositories {
			roles = append(roles, string(repository.Role))
		}
		if want := []string{"specs_in_place", "code", "code"}; !reflect.DeepEqual(roles, want) {
			t.Errorf("papeles = %v, se esperaba %v", roles, want)
		}
		if base := result.Repositories[1].Base; base == nil || base.Ref != "develop" || base.SHA != develop {
			t.Errorf("base de repo-backend = %+v, se esperaba develop", base)
		}
	})
}

// REQ-25.6 (escenario «Dos roles sobre rutas del mismo repositorio se deduplican») y REQ-25.4:
// dos roles sobre rutas del mismo repositorio de código dan un único repositorio con la rama de
// seguimiento y una rama por rol; solo la de seguimiento se despliega en `in_place` (R26).
func TestEnsureMultirepoSharedRepositoryOfTwoRolesGetsTheTrackingAndTheRoleBranches(t *testing.T) {
	t.Parallel()
	master, specs := newMultirepo(t, map[string][]string{"core": {"repo-shared"}, "qa": {"repo-shared/pruebas"}}, "repo-shared")
	if err := os.MkdirAll(filepath.Join(master, "repo-shared", "pruebas"), 0o755); err != nil {
		t.Fatal(err)
	}
	sealChangeAt(t, specs, "x", "core", "qa")

	result := ensureInPlace(t, master, "x")

	wantReady(t, result)
	if len(result.Repositories) != 2 {
		t.Fatalf("repositorios = %+v, se esperaban el de specs y uno compartido", result.Repositories)
	}
	shared := result.Repositories[1]
	if shared.Name != "repo-shared" || !reflect.DeepEqual(shared.Roles, []string{"core", "qa"}) {
		t.Errorf("repositorio compartido = %+v, se esperaban los roles core y qa", shared)
	}
	var names []string
	for _, planned := range shared.Branches {
		names = append(names, planned.Name)
		if !planned.Exists {
			t.Errorf("la rama %s debía crearse", planned.Name)
		}
	}
	wantNamesEqual(t, "ramas", names, "feat/x", "feat/x-core", "feat/x-qa")
	if got := currentBranch(t, filepath.Join(master, "repo-shared")); got != "feat/x" {
		t.Errorf("el repositorio compartido está en %q, se esperaba la rama de seguimiento", got)
	}
	if len(result.Repositories[0].Branches) != 0 {
		t.Errorf("el repositorio de specs no recibe ramas: %+v", result.Repositories[0].Branches)
	}
}

// REQ-25.12 (`--repo` acota los repositorios del plan SDD) y REQ-25.6: con `--repo` solo entran al
// plan los repositorios de código indicados; uno que no pertenece al plan se avisa y se ignora, y
// una ruta que no existe o no es un repositorio git se omite con `not_a_git_repository`.
func TestEnsureMultirepoRepoFlagNarrowsTheCodeRepositoriesOfTheSDDPlan(t *testing.T) {
	t.Parallel()
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}, "frontend": {"repo-frontend"}}, "repo-backend", "repo-frontend", "repo-otro")
	sealChangeAt(t, specs, "x", "backend", "frontend")
	sealChangeAt(t, specs, "y", "backend", "frontend")
	frontend := filepath.Join(master, "repo-frontend")
	frontendState := inPlaceState(t, frontend)

	t.Run("solo entran los repositorios indicados que pertenecen al plan", func(t *testing.T) {
		result := ensureWith(t, hubWorktree, EnsureRequest{CWD: master, Change: "x", Mode: branch.ModeInPlace, Repos: []string{"repo-backend", "repo-otro"}})

		wantReady(t, result)
		var names []string
		for _, repository := range result.Repositories {
			names = append(names, repository.Name)
		}
		if want := []string{"repo-specs", "repo-backend"}; !reflect.DeepEqual(names, want) {
			t.Fatalf("repositorios = %v, se esperaba %v", names, want)
		}
		if got := inPlaceState(t, frontend); got != frontendState {
			t.Errorf("repo-frontend queda fuera del plan y no cambia:\n%s\n---\n%s", frontendState, got)
		}
		if warnings := strings.Join(result.Warnings, "\n"); !strings.Contains(warnings, "repo-otro") || !strings.Contains(warnings, "--repo") {
			t.Errorf("avisos = %v, debe avisar de que repo-otro no pertenece al plan", result.Warnings)
		}
	})
	t.Run("una ruta que no existe se omite con not_a_git_repository", func(t *testing.T) {
		result := ensureWith(t, hubWorktree, EnsureRequest{CWD: master, Change: "y", Mode: branch.ModeInPlace, Repos: []string{"no-existe"}})

		wantReady(t, result)
		if len(result.Repositories) != 1 || result.Repositories[0].Name != "repo-specs" {
			t.Errorf("repositorios = %+v, se esperaba solo el de specs", result.Repositories)
		}
		if len(result.Skipped) != 1 || result.Skipped[0].Code != branch.CodeNotAGitRepository || !strings.Contains(result.Skipped[0].Message, "no-existe") || !strings.Contains(result.Skipped[0].Message, "--repo") {
			t.Errorf("omitidos = %+v, se esperaba la ruta no-existe con not_a_git_repository", result.Skipped)
		}
	})
}
