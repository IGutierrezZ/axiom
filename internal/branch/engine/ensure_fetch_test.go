package engine

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/gitcli"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Pruebas de `ensure --fetch` (REQ-25.10, supuesto S-8, design D-06 y matriz de amenazas
// «Estado del push»): `git fetch origin` es la única operación remota, se lanza solo con
// `--fetch`, una vez por repositorio con ramas del plan, durante la comprobación y antes de
// resolver la base; un fallo no deja ninguna rama, worktree ni registro. Los remotos son
// repositorios desnudos locales o rutas que no existen: no hay red.

func ensureWithFetch(t *testing.T, cwd, change string) branch.Result {
	t.Helper()
	return Default().Ensure(context.Background(), EnsureRequest{CWD: cwd, Change: change, Lane: branch.LaneSDD, Fetch: true})
}

// advanceRemote publica en el remoto desnudo dos commits nuevos de main desde otro clon,
// sin que repo los vea, y devuelve el commit al que apunta main en el remoto.
func advanceRemote(t *testing.T, remote string) string {
	t.Helper()
	scratch := t.TempDir()
	clone := filepath.Join(scratch, "clon")
	runGit(t, scratch, "clone", "-q", remote, clone)
	commits(t, clone, "adelanto en el remoto", 2)
	runGit(t, clone, "push", "-q", "origin", "main")
	return runGit(t, clone, "rev-parse", "HEAD")
}

func fetchInvocations(seen [][]string) [][]string {
	var fetches [][]string
	for _, invocation := range seen {
		if invocation[1] == "fetch" {
			fetches = append(fetches, invocation)
		}
	}
	return fetches
}

func indexOfInvocation(seen [][]string, match func(invocation []string) bool) int {
	for i, invocation := range seen {
		if match(invocation) {
			return i
		}
	}
	return -1
}

// REQ-25.10, escenarios «`--fetch` es explícito» y «La base es la referencia remota local»:
// con `--fetch`, un único `git fetch origin` desde el checkout principal, antes de resolver la
// base y de cualquier escritura; contra un remoto que avanzó, la base alcanza el commit nuevo,
// la rama nace de él y `base.fetched` es `true`. Sin `--fetch`, la base sigue siendo la
// referencia que ya había en local.
func TestEnsureFetchRunsASingleFetchBeforeResolvingTheBase(t *testing.T) {
	t.Run("con --fetch la base alcanza el commit nuevo", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		remote := remoteWithMain(t, repo)
		stale := runGit(t, repo, "rev-parse", "refs/remotes/origin/main")
		advanced := advanceRemote(t, remote)
		if advanced == stale {
			t.Fatal("la preparación debía adelantar el remoto")
		}
		seen := observe(t)

		result := ensureWithFetch(t, repo, "x")

		wantReady(t, result)
		base := onlyRepository(t, result).Base
		if base == nil || *base != (branch.BaseResult{Ref: "origin/main", SHA: advanced, Source: "remote-tracking", Fetched: true}) {
			t.Errorf("base = %+v, se esperaba origin/main en %s con fetched=true", base, advanced)
		}
		if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); got != advanced {
			t.Errorf("feat/x apunta a %s, se esperaba el commit traído %s", got, advanced)
		}
		fetches := fetchInvocations(*seen)
		if len(fetches) != 1 {
			t.Fatalf("se esperaba un único fetch, el observador vio %d: %v", len(fetches), *seen)
		}
		if !reflect.DeepEqual(fetches[0][1:], []string{"fetch", "origin"}) || !pathidentity.SameDirectory(fetches[0][0], repo) {
			t.Errorf("el fetch fue %v en %s; se esperaba `git fetch origin` desde el checkout principal %s (sin --prune)", fetches[0][1:], fetches[0][0], repo)
		}
		fetchAt := indexOfInvocation(*seen, func(invocation []string) bool { return invocation[1] == "fetch" })
		baseAt := indexOfInvocation(*seen, func(invocation []string) bool {
			return invocation[1] == "rev-parse" && strings.Contains(strings.Join(invocation, " "), "refs/remotes/origin/main")
		})
		writeAt := indexOfInvocation(*seen, isWriteInvocation)
		if baseAt < 0 || writeAt < 0 || fetchAt > baseAt || fetchAt > writeAt {
			t.Errorf("el fetch (%d) debe ir antes de resolver la base (%d) y de toda escritura (%d): %v", fetchAt, baseAt, writeAt, *seen)
		}
	})
	t.Run("sin --fetch la base no avanza", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		remote := remoteWithMain(t, repo)
		stale := runGit(t, repo, "rev-parse", "refs/remotes/origin/main")
		advanceRemote(t, remote)
		seen := observe(t)

		result := ensureOf(t, repo, "x")

		wantReady(t, result)
		base := onlyRepository(t, result).Base
		if base == nil || *base != (branch.BaseResult{Ref: "origin/main", SHA: stale, Source: "remote-tracking", Fetched: false}) {
			t.Errorf("base = %+v, se esperaba origin/main sin avanzar (%s) y fetched=false", base, stale)
		}
		if fetches := fetchInvocations(*seen); len(fetches) != 0 {
			t.Errorf("sin --fetch no hay ningún fetch: %v", fetches)
		}
	})
	t.Run("repetir con --fetch da el mismo estado final", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		remoteWithMain(t, repo)

		first := resultJSON(t, ensureWithFetch(t, repo, "x"))
		state := repositoryState(t, repo)
		second := resultJSON(t, ensureWithFetch(t, repo, "x"))

		if first != second || !strings.Contains(first, `"fetched": true`) {
			t.Errorf("el resultado con --fetch es estable y marca fetched=true:\n%s\n---\n%s", first, second)
		}
		if got := repositoryState(t, repo); got != state {
			t.Errorf("la repetición no debía cambiar el repositorio:\n%s\n---\n%s", state, got)
		}
	})
}

// D-06 y matriz «Estado del push»: con `--fetch` la lista cerrada de subcomandos de git añade
// `fetch` y nada más; ninguna orden remota distinta de `fetch origin`.
func TestEnsureFetchLaunchesNoRemoteOperationOtherThanFetchOrigin(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	seen := observe(t)

	wantReady(t, ensureWithFetch(t, repo, "x"))

	for _, invocation := range *seen {
		switch invocation[1] {
		case "rev-parse", "worktree", "status", "rev-list", "ls-tree", "branch":
		case "fetch":
			if !reflect.DeepEqual(invocation[1:], []string{"fetch", "origin"}) {
				t.Errorf("el único fetch permitido es `git fetch origin`: %v", invocation[1:])
			}
		default:
			t.Errorf("ensure --fetch lanzó %q, fuera de la lista cerrada: %v", invocation[1], invocation)
		}
		if invocation[1] != "fetch" && remoteOperation(invocation[1]) {
			t.Errorf("ensure --fetch lanzó la operación remota %s: %v", invocation[1], invocation)
		}
	}
}

// D-06: `--fetch` se ejecuta en cada repositorio del plan que tiene ramas (un fetch por
// repositorio, desde su checkout principal) y nunca en el repositorio de specs, que se
// trabaja in situ.
func TestEnsureFetchRunsOneFetchPerCodeRepositoryOfAMultirepo(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}, "frontend": {"repo-frontend"}}, "repo-backend", "repo-frontend")
	sealChangeAt(t, specs, "x", "backend", "frontend")
	backend, frontend := filepath.Join(master, "repo-backend"), filepath.Join(master, "repo-frontend")
	remoteWithMain(t, backend)
	remoteWithMain(t, frontend)
	seen := observe(t)

	result := ensureWithFetch(t, master, "x")

	wantReady(t, result)
	fetches := fetchInvocations(*seen)
	if len(fetches) != 2 {
		t.Fatalf("se esperaba un fetch por repositorio de código, el observador vio %d: %v", len(fetches), fetches)
	}
	for _, repo := range []string{backend, frontend} {
		count := 0
		for _, fetch := range fetches {
			if pathidentity.SameDirectory(fetch[0], repo) {
				count++
			}
		}
		if count != 1 {
			t.Errorf("%s recibió %d fetch, se esperaba uno", repo, count)
		}
	}
	for _, repository := range result.Repositories {
		switch {
		case repository.Role == branch.RepoSpecsInPlace:
			if repository.Base != nil {
				t.Errorf("el repositorio de specs no tiene base: %+v", repository.Base)
			}
		case repository.Base == nil || !repository.Base.Fetched:
			t.Errorf("%s: base = %+v, se esperaba fetched=true", repository.Name, repository.Base)
		}
	}
}

// REQ-25.10, escenario «`--fetch` fallido no deja efectos», y supuesto S-8: si el fetch falla
// termina con `fetch_failed`, nombrando el repositorio y el remoto, sin crear rama, worktree,
// raíz de worktrees ni registro; también cuando el fallo es del segundo repositorio, porque el
// fetch es parte de la comprobación sin efectos de todo el plan.
func TestEnsureFetchFailedLeavesNoEffects(t *testing.T) {
	t.Parallel()
	t.Run("remoto inalcanzable", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		runGit(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "no-existe.git"))
		state := repositoryState(t, repo)

		result := ensureWithFetch(t, repo, "x")

		wantFetchFailed(t, result, repo)
		wantNothingCreated(t, repo, state, "x")
		wantNoRegistryFile(t, repo, "x")
	})
	t.Run("sin remoto origin", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		state := repositoryState(t, repo)

		result := ensureWithFetch(t, repo, "x")

		wantFetchFailed(t, result, repo)
		wantNothingCreated(t, repo, state, "x")
		wantNoRegistryFile(t, repo, "x")
	})
	t.Run("falla el segundo repositorio del multirepo", func(t *testing.T) {
		master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}, "frontend": {"repo-frontend"}}, "repo-backend", "repo-frontend")
		sealChangeAt(t, specs, "x", "backend", "frontend")
		backend, frontend := filepath.Join(master, "repo-backend"), filepath.Join(master, "repo-frontend")
		remoteWithMain(t, backend)
		runGit(t, frontend, "remote", "add", "origin", filepath.Join(t.TempDir(), "no-existe.git"))
		backendState, frontendState := repositoryState(t, backend), repositoryState(t, frontend)

		result := ensureWithFetch(t, master, "x")

		wantFetchFailed(t, result, frontend)
		for repo, state := range map[string]string{backend: backendState, frontend: frontendState} {
			if got := repositoryState(t, repo); got != state {
				t.Errorf("%s no debía cambiar si el fetch de otro repositorio falla:\n%s\n---\n%s", repo, state, got)
			}
		}
		if _, err := os.Stat(branch.DefaultWorktreeRoot(canonicalDirectory(t, master))); err == nil {
			t.Error("no debía crearse ni la raíz de worktrees")
		}
		wantNoRegistryFile(t, specs, "x")
	})
}

// wantFetchFailed comprueba el rechazo `fetch_failed`: nombra el repositorio y el remoto, el
// resumen de git, y una continuación con las dos salidas (reintentar con `--fetch` o seguir
// sin red).
func wantFetchFailed(t *testing.T, result branch.Result, repo string) {
	t.Helper()
	wantRefusal(t, result, branch.CodeFetchFailed, branch.StatusRefused)
	if !strings.Contains(result.Message, "origin") || !strings.Contains(result.Message, filepath.Base(repo)) {
		t.Errorf("el mensaje %q debe nombrar el remoto origin y el repositorio %s", result.Message, repo)
	}
	if result.Details["remote"] != "origin" || result.Details["summary"] == "" || result.Details["repository"] == "" {
		t.Errorf("detalles = %v, se esperaban remote, repository y summary", result.Details)
	}
	for _, want := range []string{"remote -v", "axiom branch ensure --change x --fetch --cwd", "axiom branch ensure --change x --cwd"} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
	if !strings.Contains(result.Continuation, pathquote.Quote(result.Details["repository"])) {
		t.Errorf("la continuación %q debe citar el repositorio %s", result.Continuation, result.Details["repository"])
	}
}

// Matriz de amenazas «Estado del push» y REQ-25.1: el puerto de git no tiene ningún método
// que publique (`push`) y `Fetch` es la única operación remota; ni el adaptador real ni las
// interfaces locales del motor añaden ninguna.
func TestTheGitPortHasNoPushMethodAndFetchIsTheOnlyRemoteOperation(t *testing.T) {
	t.Parallel()
	remote := func(name string) bool {
		lower := strings.ToLower(name)
		for _, word := range []string{"push", "pull", "lsremote", "publish", "upload"} {
			if strings.Contains(lower, word) {
				return true
			}
		}
		return false
	}
	for name, typ := range map[string]reflect.Type{
		"branch.Git": reflect.TypeOf((*branch.Git)(nil)).Elem(),
		"gitcli.Git": reflect.TypeOf(gitcli.New()),
		"gitWriter":  reflect.TypeOf((*gitWriter)(nil)).Elem(),
		"gitReader":  reflect.TypeOf((*gitReader)(nil)).Elem(),
	} {
		var fetch bool
		for i := 0; i < typ.NumMethod(); i++ {
			method := typ.Method(i).Name
			if remote(method) {
				t.Errorf("%s tiene el método %s: `axiom branch` nunca publica", name, method)
			}
			fetch = fetch || method == "Fetch"
		}
		if name == "branch.Git" && !fetch {
			t.Error("el puerto branch.Git debe tener Fetch, la única operación remota")
		}
	}
}
