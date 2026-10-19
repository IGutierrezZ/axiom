package engine

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/gitcli"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de `ensure` con git real, entorno aislado y sin red (REQ-25.7, REQ-25.8,
// REQ-25.10, REQ-25.11, matriz de amenazas «Selección de repositorio git», «Estado del
// commit» y «Estado del push»). Cubren el modo `worktree` con la base sin red; el registro
// local con escritura está en ensure_registry_test.go y `--fetch`, en ensure_fetch_test.go.

func ensureOf(t *testing.T, cwd, change string) branch.Result {
	t.Helper()
	return Default().Ensure(context.Background(), EnsureRequest{CWD: cwd, Change: change, Lane: branch.LaneSDD})
}

func resultJSON(t *testing.T, result branch.Result) string {
	t.Helper()
	var out bytes.Buffer
	if err := result.WriteJSON(&out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// repositoryState es lo que `ensure` puede cambiar de un repositorio: sus ramas con el
// commit al que apuntan y los worktrees que git conoce.
func repositoryState(t *testing.T, repo string) string {
	t.Helper()
	return runGit(t, repo, "for-each-ref", "--format=%(refname) %(objectname)", "refs/heads") +
		"\n--\n" + runGit(t, repo, "worktree", "list", "--porcelain")
}

// checkoutState es lo que `ensure` no debe tocar del checkout principal en modo
// `worktree` (REQ-25.8): HEAD, rama actual, índice y estado del árbol de trabajo.
func checkoutState(t *testing.T, repo string) string {
	t.Helper()
	return strings.Join([]string{
		runGit(t, repo, "rev-parse", "HEAD"),
		runGit(t, repo, "symbolic-ref", "HEAD"),
		runGit(t, repo, "ls-files", "--stage"),
		runGit(t, repo, "status", "--porcelain=v1", "--untracked-files=all"),
	}, "\n--\n")
}

func isWriteInvocation(invocation []string) bool {
	return invocation[1] == "branch" || (invocation[1] == "worktree" && len(invocation) > 2 && invocation[2] == "add")
}

func wantNothingCreated(t *testing.T, repo, wantState string, change string) {
	t.Helper()
	if got := repositoryState(t, repo); got != wantState {
		t.Errorf("el rechazo no debe crear nada:\nantes:\n%s\ndespués:\n%s", wantState, got)
	}
	if _, err := os.Stat(branch.DefaultWorktreeRoot(canonicalDirectory(t, repo))); err == nil {
		t.Errorf("el rechazo no debe crear la raíz de worktrees para %s", change)
	}
}

func embeddedCoreProject(t *testing.T) string {
	t.Helper()
	return newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
}

// REQ-25.7, REQ-25.8, REQ-25.11 y REQ-25.10: en un monorepo de un rol, `ensure` crea la
// rama `<prefijo>/<cambio>` desde la base local y un worktree en
// `<raíz>/<cambio>/<repo>`; el checkout principal queda como estaba y el resultado
// describe el estado final, con la base y el modo.
func TestEnsureCreatesTheChangeBranchAndItsWorktree(t *testing.T) {
	repo := embeddedCoreProject(t)
	head := runGit(t, repo, "rev-parse", "HEAD")
	// El checkout principal tiene trabajo sin confirmar: no se toca (REQ-25.8).
	writeFile(t, filepath.Join(repo, "axiom.yaml"), configWithRoles(map[string][]string{"core": {"."}}, "# cambio local\n"))
	writeFile(t, filepath.Join(repo, "nuevo.txt"), "sin seguimiento\n")
	runGit(t, repo, "add", "nuevo.txt")
	writeFile(t, filepath.Join(repo, "suelto.txt"), "sin añadir\n")
	before := checkoutState(t, repo)

	result := ensureOf(t, repo, "x")

	wantReady(t, result)
	if result.Operation != "ensure" || result.Change != "x" || result.Lane != branch.LaneSDD || result.Topology != workspace.TopologyMonorepoEmbedded {
		t.Errorf("operación %q, cambio %q, carril %q, topología %q", result.Operation, result.Change, result.Lane, result.Topology)
	}
	if result.Mode != branch.ModeWorktree || result.ModeLabel == "" || result.ModeSource != branch.ModeSourceDefault || result.ModeSourceLabel == "" {
		t.Errorf("modo %q (%q), origen %q (%q); se esperaba worktree por defecto, cada token con su etiqueta", result.Mode, result.ModeLabel, result.ModeSource, result.ModeSourceLabel)
	}
	wantSameDirectory(t, "worktreeRoot", result.WorktreeRoot, branch.DefaultWorktreeRoot(canonicalDirectory(t, repo)))

	repository := onlyRepository(t, result)
	if repository.Base == nil || *repository.Base != (branch.BaseResult{Ref: "main", SHA: head, Source: "local", Fetched: false}) {
		t.Errorf("base = %+v, se esperaba la rama local main en %s sin fetch", repository.Base, head)
	}
	feature := branchByName(t, repository, "feat/x")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	if !feature.Exists || feature.Kind != branch.KindChange || feature.PRTarget != "main" || feature.Dirty {
		t.Errorf("rama del cambio = %+v", feature)
	}
	wantSameDirectory(t, "worktree de feat/x", feature.Worktree, worktree)
	if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); got != head {
		t.Errorf("feat/x apunta a %s, se esperaba la base %s", got, head)
	}
	if got := runGit(t, worktree, "symbolic-ref", "--short", "HEAD"); got != "feat/x" {
		t.Errorf("el worktree tiene %s, se esperaba feat/x", got)
	}
	if _, err := os.Stat(filepath.Join(worktree, "axiom.yaml")); err != nil {
		t.Errorf("el worktree debe tener los ficheros de la rama: %v", err)
	}
	if after := checkoutState(t, repo); after != before {
		t.Errorf("el checkout principal cambió:\nantes:\n%s\ndespués:\n%s", before, after)
	}
}

// REQ-25.7, escenario «Segunda ejecución sin cambios»: el mismo JSON byte a byte, las mismas
// ramas y los mismos worktrees, y la segunda ejecución no lanza ninguna escritura.
func TestEnsureIsIdempotentByteForByte(t *testing.T) {
	repo := embeddedCoreProject(t)
	first := resultJSON(t, ensureOf(t, repo, "x"))
	state := repositoryState(t, repo)
	seen := observe(t)

	second := resultJSON(t, ensureOf(t, repo, "x"))

	if first != second {
		t.Errorf("la segunda ejecución cambió el JSON:\n%s\n---\n%s", first, second)
	}
	if !strings.HasSuffix(first, "\n") || strings.Contains(first, "created") || strings.Contains(first, "changed") {
		t.Errorf("el resultado describe el estado final y no las acciones: %s", first)
	}
	if got := repositoryState(t, repo); got != state {
		t.Errorf("la segunda ejecución cambió el repositorio:\n%s\n---\n%s", state, got)
	}
	if len(*seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación: la prueba sería vacua")
	}
	for _, invocation := range *seen {
		if isWriteInvocation(invocation) {
			t.Errorf("la segunda ejecución no tiene nada que crear y lanzó %v", invocation)
		}
	}
}

// REQ-25.7, escenario «La rama existente se reutiliza sin moverla»: una rama con commits
// propios y sin worktree recibe su worktree y no cambia de commit.
func TestEnsureReusesAnExistingBranchWithoutMovingIt(t *testing.T) {
	repo := embeddedCoreProject(t)
	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	commit(t, repo, "propio de feat/x")
	owned := runGit(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "switch", "-q", "main")
	commits(t, repo, "en main", 2)

	result := ensureOf(t, repo, "x")

	wantReady(t, result)
	if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); got != owned {
		t.Errorf("feat/x se movió de %s a %s", owned, got)
	}
	feature := branchByName(t, onlyRepository(t, result), "feat/x")
	wantSameDirectory(t, "worktree de feat/x", feature.Worktree, plannedWorktree(t, repo, "x", filepath.Base(repo)))
	if got := runGit(t, feature.Worktree, "rev-parse", "HEAD"); got != owned {
		t.Errorf("el worktree está en %s, se esperaba el commit propio %s", got, owned)
	}
}

// REQ-25.7, escenario «Rama desplegada en un worktree desconocido»: `branch_checked_out_elsewhere`
// con la ruta existente y sin crear nada, sea un worktree ajeno o el propio checkout principal.
func TestEnsureRefusesABranchCheckedOutInAnUnknownWorktree(t *testing.T) {
	t.Run("en un worktree ajeno", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		elsewhere := filepath.Join(t.TempDir(), "ajeno")
		runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", elsewhere, "main")
		state := repositoryState(t, repo)

		result := ensureOf(t, repo, "x")

		wantRefusal(t, result, branch.CodeBranchCheckedOutElsewhere, branch.StatusRefused)
		if !strings.Contains(result.Message, elsewhere) && !strings.Contains(result.Message, canonicalDirectory(t, elsewhere)) {
			t.Errorf("el mensaje %q debe nombrar el worktree existente %s", result.Message, elsewhere)
		}
		wantSameDirectory(t, "details.worktree", result.Details["worktree"], elsewhere)
		for _, want := range []string{"git -C", "worktree remove", "axiom branch ensure --change x --cwd", pathquote.Quote(repo)} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
			}
		}
		wantNothingCreated(t, repo, state, "x")
	})
	t.Run("en el checkout principal", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		runGit(t, repo, "switch", "-q", "-c", "feat/x")
		state := repositoryState(t, repo)

		result := ensureOf(t, repo, "x")

		wantRefusal(t, result, branch.CodeBranchCheckedOutElsewhere, branch.StatusRefused)
		for _, want := range []string{"git -C", "switch", "axiom branch ensure --change x --cwd"} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q (el principal no se retira: se cambia de rama)", result.Continuation, want)
			}
		}
		wantNothingCreated(t, repo, state, "x")
	})
	t.Run("una vista desacoplada de RDD no cuenta", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		runGit(t, repo, "worktree", "add", "-q", "--detach", filepath.Join(t.TempDir(), "vista"), "main")

		wantReady(t, ensureOf(t, repo, "x"))
	})
}

// REQ-25.7, escenario «Ejecución previa al sellado y posterior al sellado»: antes de sellar
// solo se crea `feat/x`; con dos roles sellados se completan, sin alterar lo creado,
// `feat/x-core` y `feat/x-qa` desde ella, cada una con su worktree.
func TestEnsureCompletesIncrementallyBeforeAndAfterSealing(t *testing.T) {
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))
	name := filepath.Base(repo)

	first := ensureOf(t, repo, "x")

	wantReady(t, first)
	if branches := onlyRepository(t, first).Branches; len(branches) != 1 || branches[0].Name != "feat/x" {
		t.Fatalf("antes del sellado solo existe feat/x: %+v", branches)
	}
	commit(t, plannedWorktree(t, repo, "x", name), "trabajo en la rama de seguimiento")
	tracking := runGit(t, repo, "rev-parse", "refs/heads/feat/x")
	sealChange(t, repo, "x", "core", "qa")

	second := ensureOf(t, repo, "x")

	wantReady(t, second)
	repository := onlyRepository(t, second)
	if len(repository.Branches) != 3 {
		t.Fatalf("ramas = %+v, se esperaban feat/x, feat/x-core y feat/x-qa", repository.Branches)
	}
	if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); got != tracking {
		t.Errorf("feat/x se movió de %s a %s al completar las ramas de rol", tracking, got)
	}
	for _, role := range []string{"core", "qa"} {
		described := branchByName(t, repository, "feat/x-"+role)
		if described.Kind != branch.KindRole || described.Role != role || described.PRTarget != "feat/x" || !described.Exists {
			t.Errorf("rama de rol %s = %+v", role, described)
		}
		if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x-"+role); got != tracking {
			t.Errorf("feat/x-%s apunta a %s, se esperaba la rama de seguimiento %s", role, got, tracking)
		}
		wantSameDirectory(t, "worktree de "+role, described.Worktree, plannedWorktree(t, repo, "x-"+role, name))
	}
	wantSameDirectory(t, "worktree de feat/x", branchByName(t, repository, "feat/x").Worktree, plannedWorktree(t, repo, "x", name))
}

// REQ-25.7, escenario «Se rechaza un nombre de cambio que no es válido»: `invalid_name`
// antes de crear nada.
func TestEnsureRefusesAnInvalidChangeNameBeforeCreatingAnything(t *testing.T) {
	repo := embeddedCoreProject(t)
	state := repositoryState(t, repo)

	for _, invalid := range []string{"a b:c", "../fuera", "CON"} {
		t.Run(invalid, func(t *testing.T) {
			result := ensureOf(t, repo, invalid)

			wantRefusal(t, result, branch.CodeInvalidName, branch.StatusRefused)
			for _, want := range []string{"axiom branch ensure --change <cambio> --cwd", pathquote.Quote(repo)} {
				if !strings.Contains(result.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
				}
			}
			wantNothingCreated(t, repo, state, invalid)
		})
	}
}

// remoteWithMain crea un remoto desnudo local, lo registra como `origin` en repo y publica
// main; es preparación de la prueba, no del motor.
func remoteWithMain(t *testing.T, repo string) string {
	t.Helper()
	remote := t.TempDir()
	runGit(t, remote, "init", "-q", "--bare", "-b", "main")
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-q", "origin", "main")
	return remote
}

// REQ-25.10: con `origin/main` y `main` en commits distintos, la base es la referencia
// remota ya traída a local y la rama nace de ella; sin remoto, la rama local.
func TestEnsureTakesTheBaseFromTheRemoteTrackingReferenceIfItExists(t *testing.T) {
	t.Run("referencia remota local", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		remoteWithMain(t, repo)
		remoteSHA := runGit(t, repo, "rev-parse", "refs/remotes/origin/main")
		commits(t, repo, "solo local", 2)
		if runGit(t, repo, "rev-parse", "main") == remoteSHA {
			t.Fatal("la preparación debía dejar main distinta de origin/main")
		}

		result := ensureOf(t, repo, "x")

		wantReady(t, result)
		base := onlyRepository(t, result).Base
		if base == nil || *base != (branch.BaseResult{Ref: "origin/main", SHA: remoteSHA, Source: "remote-tracking", Fetched: false}) {
			t.Errorf("base = %+v, se esperaba origin/main en %s (remote-tracking, sin fetch)", base, remoteSHA)
		}
		if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); got != remoteSHA {
			t.Errorf("feat/x apunta a %s, se esperaba la base remota %s", got, remoteSHA)
		}
	})
	t.Run("sin remoto, la rama local", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		head := runGit(t, repo, "rev-parse", "main")

		base := onlyRepository(t, ensureOf(t, repo, "x")).Base

		if base == nil || *base != (branch.BaseResult{Ref: "main", SHA: head, Source: "local", Fetched: false}) {
			t.Errorf("base = %+v, se esperaba la rama local main en %s", base, head)
		}
	})
}

// REQ-25.10, escenario «Sin `--fetch` no hay operaciones remotas», y matriz de amenazas
// «Estado del push»: con un remoto configurado y un observador en el único punto que ejecuta
// git, `ensure` sin `--fetch` no lanza ninguna operación remota (ni `fetch`: la lista
// cerrada de subcomandos no lo incluye) y solo escribe con `git branch` y `git worktree
// add`. Con `--fetch` la lista añade `fetch` y nada más (ensure_fetch_test.go).
func TestEnsureLaunchesNoRemoteOperationWithoutFetch(t *testing.T) {
	repo := embeddedCoreProject(t)
	runGit(t, repo, "remote", "add", "origin", "https://ejemplo.invalid/repo.git")
	seen := observe(t)

	wantReady(t, ensureOf(t, repo, "x"))

	if len(*seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación: la prueba sería vacua")
	}
	wrote := 0
	for _, invocation := range *seen {
		if remoteOperation(invocation[1]) {
			t.Errorf("ensure lanzó la operación remota git %s: %v", invocation[1], invocation)
		}
		switch invocation[1] {
		case "rev-parse", "worktree", "status", "rev-list", "ls-tree", "branch":
		default:
			t.Errorf("ensure lanzó %q, fuera de la lista cerrada: %v", invocation[1], invocation)
		}
		if isWriteInvocation(invocation) {
			wrote++
		}
	}
	if wrote != 2 {
		t.Errorf("se esperaban dos escrituras (`git branch` y `git worktree add`), el observador vio %d: %v", wrote, *seen)
	}
}

// REQ-25.10, escenario «Rama de integración inexistente»: `base_not_found` nombrando la rama
// esperada, sin crear nada; también en un repositorio sin commits (matriz «Estado del commit»).
func TestEnsureRefusesAMissingIntegrationBranch(t *testing.T) {
	t.Run("rama de integración inexistente", func(t *testing.T) {
		repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, "git:\n  integration_branch: develop\n"))
		state := repositoryState(t, repo)

		result := ensureOf(t, repo, "x")

		wantRefusal(t, result, branch.CodeBaseNotFound, branch.StatusRefused)
		if !strings.Contains(result.Message, "develop") {
			t.Errorf("el mensaje %q debe nombrar develop", result.Message)
		}
		for _, want := range []string{"git -C", pathquote.Quote(repo), "axiom branch ensure --change x --cwd"} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
			}
		}
		wantNothingCreated(t, repo, state, "x")
	})
	t.Run("repositorio sin commits", func(t *testing.T) {
		requireGit(t)
		repo := t.TempDir()
		runGit(t, repo, "init", "-q", "-b", "main")
		writeFile(t, filepath.Join(repo, "axiom.yaml"), configWithRoles(map[string][]string{"core": {"."}}, ""))

		result := ensureOf(t, repo, "x")

		wantRefusal(t, result, branch.CodeBaseNotFound, branch.StatusRefused)
		if entries := runGit(t, repo, "branch", "--list"); entries != "" {
			t.Errorf("no se crea ninguna rama: %q", entries)
		}
	})
}

// failingWriter hace fallar las escrituras de git que se piden y, en las demás, delega en
// git real: fuerza los fallos que un repositorio real no produce.
type failingWriter struct {
	real         gitWriter
	failBranch   bool
	failWorktree bool
}

func (f failingWriter) CreateBranch(ctx context.Context, dir, name, startPoint string) error {
	if f.failBranch {
		return errors.New("fallo simulado de git al crear la rama")
	}
	return f.real.CreateBranch(ctx, dir, name, startPoint)
}

func (f failingWriter) AddWorktree(ctx context.Context, dir, path, branchName string) error {
	if f.failWorktree {
		return errors.New("fallo simulado de git al crear el worktree")
	}
	return f.real.AddWorktree(ctx, dir, path, branchName)
}

func (f failingWriter) Fetch(ctx context.Context, dir, remote string) error {
	return f.real.Fetch(ctx, dir, remote)
}

// REQ-25.2 y D-05: un fallo de git durante la aplicación es `git_failed`, que nombra la
// operación y resume el error; repetir `ensure` completa lo pendiente.
func TestEnsureReportsGitFailedNamingTheOperationAndRepeatingCompletes(t *testing.T) {
	flaky := func(w failingWriter) *Engine {
		engine := newEngine(runtime.GOOS)
		w.real = gitcli.New()
		engine.writer = w
		return engine
	}
	request := func(repo string) EnsureRequest {
		return EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD}
	}

	t.Run("al crear la rama", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		state := repositoryState(t, repo)

		result := flaky(failingWriter{failBranch: true}).Ensure(context.Background(), request(repo))

		wantRefusal(t, result, branch.CodeGitFailed, branch.StatusError)
		if !strings.Contains(result.Details["operation"], "feat/x") || !strings.Contains(result.Message, result.Details["operation"]) || !strings.Contains(result.Message, "fallo simulado") {
			t.Errorf("el mensaje %q debe nombrar la operación %q y resumir el error", result.Message, result.Details["operation"])
		}
		for _, want := range []string{"axiom branch ensure --change x --cwd", pathquote.Quote(repo)} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q (repetir completa lo pendiente)", result.Continuation, want)
			}
		}
		wantNothingCreated(t, repo, state, "x")
	})
	t.Run("al crear el worktree, con la rama ya creada", func(t *testing.T) {
		repo := embeddedCoreProject(t)

		result := flaky(failingWriter{failWorktree: true}).Ensure(context.Background(), request(repo))

		wantRefusal(t, result, branch.CodeGitFailed, branch.StatusError)
		if !strings.Contains(result.Details["operation"], "worktree") || !strings.Contains(result.Details["operation"], "feat/x") {
			t.Errorf("la operación %q debe nombrar el worktree y la rama", result.Details["operation"])
		}
		if runGit(t, repo, "branch", "--list", "feat/x") == "" {
			t.Fatal("la rama ya creada se conserva: repetir completa el worktree")
		}

		wantReady(t, ensureOf(t, repo, "x"))
		if runGit(t, plannedWorktree(t, repo, "x", filepath.Base(repo)), "symbolic-ref", "--short", "HEAD") != "feat/x" {
			t.Error("repetir ensure debía crear el worktree que faltaba")
		}
	})
	t.Run("git no se puede ejecutar", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		t.Setenv("PATH", t.TempDir())

		result := ensureOf(t, repo, "x")

		wantRefusal(t, result, branch.CodeGitFailed, branch.StatusError)
		if result.Details["operation"] == "" || result.Details["summary"] == "" {
			t.Errorf("git_failed nombra la operación y resume el error: %v", result.Details)
		}
		for _, want := range []string{"axiom branch ensure --change x --cwd", pathquote.Quote(repo)} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
			}
		}
	})
}

// REQ-25.2: un axiom.yaml ilegible o con un valor de política inválido es `config_invalid`
// y no se crea nada.
func TestEnsureRefusesAnInvalidConfigurationAsConfigInvalid(t *testing.T) {
	for name, content := range map[string]string{
		"prefijo inválido": configWithRoles(map[string][]string{"core": {"."}}, "git:\n  branch_prefix: \"mi prefijo//\"\n"),
		"YAML ilegible":    "workspace: [sin cerrar\n",
	} {
		t.Run(name, func(t *testing.T) {
			repo := newProject(t, content)
			state := repositoryState(t, repo)

			result := ensureOf(t, repo, "x")

			wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
			for _, want := range []string{"axiom workspace validate --path", pathquote.Quote(repo)} {
				if !strings.Contains(result.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
				}
			}
			wantNothingCreated(t, repo, state, "x")
		})
	}
}

// Matriz de amenazas «Selección de repositorio git»: las variables con las que un proceso
// padre fija el repositorio no cambian la selección, y el directorio del proceso no
// interviene.
func TestEnsureSelectsTheRepositoryFromTheDirectoryNotFromTheEnvironment(t *testing.T) {
	repo := embeddedCoreProject(t)
	other := newRepository(t)
	otherState := repositoryState(t, other)

	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)
	t.Setenv("GIT_COMMON_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(other, ".git", "index"))
	t.Chdir(other)
	result := ensureOf(t, repo, "x")
	for _, name := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"} {
		_ = os.Unsetenv(name)
	}

	wantReady(t, result)
	wantSameDirectory(t, "gitRoot", onlyRepository(t, result).GitRoot, repo)
	if runGit(t, repo, "branch", "--list", "feat/x") == "" {
		t.Error("feat/x debía crearse en el repositorio de --cwd")
	}
	if got := repositoryState(t, other); got != otherState {
		t.Errorf("el repositorio ajeno cambió:\n%s\n---\n%s", otherState, got)
	}
}

// Matriz de amenazas «Selección de repositorio git»: toda creación se hace desde el checkout
// principal, también cuando `--cwd` es un worktree enlazado.
func TestEnsureRunsEveryWriteFromTheMainCheckout(t *testing.T) {
	main := embeddedCoreProject(t)
	linked := filepath.Join(t.TempDir(), "enlazado")
	runGit(t, main, "worktree", "add", "-q", "-b", "inc/otro", linked, "main")
	seen := observe(t)

	for name, cwd := range map[string]string{"desde el principal": main, "desde un enlazado": linked} {
		t.Run(name, func(t *testing.T) {
			*seen = nil
			wantReady(t, ensureOf(t, cwd, "x"+strings.ReplaceAll(name, " ", "")))

			writes := 0
			for _, invocation := range *seen {
				if !isWriteInvocation(invocation) {
					continue
				}
				writes++
				if !pathidentity.SameDirectory(invocation[0], main) {
					t.Errorf("la escritura %v no se lanzó desde el checkout principal %s", invocation[1:], main)
				}
			}
			if writes != 2 {
				t.Errorf("se esperaban dos escrituras, el observador vio %d: %v", writes, *seen)
			}
		})
	}
}

// Matriz de amenazas «Selección de repositorio git»: una ruta de rol con `..` se resuelve a
// la raíz git real que la contiene, nunca por texto, y una que no es un repositorio se
// omite con `not_a_git_repository` sin impedir el resto del plan (REQ-25.6).
func TestEnsureResolvesRolePathsToTheirRealGitRoot(t *testing.T) {
	requireGit(t)
	parent := t.TempDir()
	master := filepath.Join(parent, "proyecto")
	sibling := filepath.Join(parent, "hermano")
	plain := filepath.Join(parent, "sin-git")
	for _, dir := range []string{master, sibling} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		initRepository(t, dir)
	}
	writeFile(t, filepath.Join(plain, "nota.txt"), "no es un repositorio\n")
	for current := plain; ; {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			t.Skipf("el directorio temporal %s está dentro de un repositorio git", plain)
		}
		if next := filepath.Dir(current); next != current {
			current = next
			continue
		}
		break
	}
	writeFile(t, filepath.Join(master, "axiom.yaml"), configWithRoles(map[string][]string{
		"core":  {"."},
		"fuera": {"app/../../hermano/"},
		"docs":  {"../sin-git", "docs/inexistente"},
	}, ""))
	runGit(t, master, "add", "axiom.yaml")
	commit(t, master, "configuración")
	sealChange(t, master, "x", "fullstack")

	result := ensureOf(t, master, "x")

	wantReady(t, result)
	if len(result.Repositories) != 2 {
		t.Fatalf("repositorios = %+v, se esperaban el del proyecto y el hermano", result.Repositories)
	}
	hermano := result.Repositories[1]
	wantSameDirectory(t, "raíz git del hermano", hermano.GitRoot, sibling)
	worktree := plannedWorktree(t, master, "x", "hermano")
	wantSameDirectory(t, "worktree del hermano", branchByName(t, hermano, "feat/x").Worktree, worktree)
	if pathidentity.Contains(sibling, worktree) || pathidentity.Contains(master, worktree) {
		t.Errorf("el worktree %s no puede quedar dentro de ningún repositorio del plan", worktree)
	}
	codes := map[string]string{}
	for _, skipped := range result.Skipped {
		codes[filepath.Base(skipped.Path)] = string(skipped.Code)
		if !strings.Contains(skipped.Message, skipped.Path) || !strings.Contains(skipped.Message, skipped.Role) {
			t.Errorf("el mensaje %q debe nombrar la ruta y el rol", skipped.Message)
		}
	}
	if len(result.Skipped) != 2 || codes["sin-git"] != "not_a_git_repository" || codes["inexistente"] != "not_a_git_repository" {
		t.Errorf("omitidos = %+v, se esperaban sin-git e inexistente con not_a_git_repository", result.Skipped)
	}
}

// REQ-25.5, REQ-25.11 y matriz «Selección de repositorio git»: en un multirepo el
// repositorio de specs no recibe ramas ni worktrees y los repositorios de código crean los
// suyos fuera de la carpeta maestra, en la raíz de worktrees.
func TestEnsureCreatesTheWorktreesOfAMultirepoOutsideTheMasterFolder(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}, "frontend": {"repo-frontend"}}, "repo-backend", "repo-frontend")
	sealChangeAt(t, specs, "x", "backend", "frontend")
	specsState := repositoryState(t, specs)

	result := ensureOf(t, master, "x")

	wantReady(t, result)
	if got := repositoryState(t, specs); got != specsState {
		t.Errorf("el repositorio de specs se trabaja in situ y no cambia:\n%s\n---\n%s", specsState, got)
	}
	for _, name := range []string{"repo-backend", "repo-frontend"} {
		code := filepath.Join(master, name)
		worktree := plannedWorktree(t, master, "x", name)
		if got := runGit(t, worktree, "symbolic-ref", "--short", "HEAD"); got != "feat/x" {
			t.Errorf("%s: el worktree tiene %s, se esperaba feat/x", name, got)
		}
		if pathidentity.Contains(master, worktree) {
			t.Errorf("el worktree %s no puede quedar dentro de la carpeta maestra", worktree)
		}
		if got := runGit(t, code, "symbolic-ref", "--short", "HEAD"); got != "main" {
			t.Errorf("%s: el checkout principal pasó a %s", name, got)
		}
	}
}

// D-05, fase 3: la comprobación sin efectos cubre TODOS los repositorios antes de mutar
// ninguno. Si el segundo no tiene base, el primero tampoco se crea.
func TestEnsureChecksEveryRepositoryBeforeMutatingAny(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}, "frontend": {"repo-frontend"}}, "repo-backend", "repo-frontend")
	sealChangeAt(t, specs, "x", "backend", "frontend")
	runGit(t, filepath.Join(master, "repo-frontend"), "branch", "-m", "main", "tronco")
	backend := filepath.Join(master, "repo-backend")
	state := repositoryState(t, backend)

	result := ensureOf(t, master, "x")

	wantRefusal(t, result, branch.CodeBaseNotFound, branch.StatusRefused)
	if !strings.Contains(result.Message, "repo-frontend") {
		t.Errorf("el mensaje %q debe nombrar el repositorio sin base", result.Message)
	}
	if got := repositoryState(t, backend); got != state {
		t.Errorf("repo-backend no debía cambiar si repo-frontend no tiene base:\n%s\n---\n%s", state, got)
	}
	if _, err := os.Stat(branch.DefaultWorktreeRoot(canonicalDirectory(t, master))); err == nil {
		t.Error("no debía crearse ni la raíz de worktrees")
	}
}

// El plan reutiliza el worktree que ya está en la ruta planificada con la rama esperada,
// aunque lo creara otra herramienta, y no lanza ninguna escritura.
func TestEnsureReusesAWorktreeAlreadyAtThePlannedPath(t *testing.T) {
	repo := embeddedCoreProject(t)
	planned := plannedWorktree(t, repo, "x", filepath.Base(repo))
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", planned, "main")
	state := repositoryState(t, repo)
	seen := observe(t)

	result := ensureOf(t, repo, "x")

	wantReady(t, result)
	wantSameDirectory(t, "worktree", branchByName(t, onlyRepository(t, result), "feat/x").Worktree, planned)
	if got := repositoryState(t, repo); got != state {
		t.Errorf("no había nada que crear:\n%s\n---\n%s", state, got)
	}
	for _, invocation := range *seen {
		if isWriteInvocation(invocation) {
			t.Errorf("no había nada que crear y se lanzó %v", invocation)
		}
	}
}

// D-05, fase 3: `plan_conflict` cuando la ruta planificada está ocupada, antes de crear nada:
// por un worktree con otra rama, por un worktree cuyo directorio ya no existe, o por un
// directorio con contenido.
func TestEnsureRefusesAPlannedPathThatIsNotFree(t *testing.T) {
	t.Run("un worktree con otra rama", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		planned := plannedWorktree(t, repo, "x", filepath.Base(repo))
		runGit(t, repo, "worktree", "add", "-q", "-b", "otra", planned, "main")
		state := repositoryState(t, repo)

		result := ensureOf(t, repo, "x")

		wantRefusal(t, result, branch.CodePlanConflict, branch.StatusRefused)
		if !strings.Contains(result.Message, "otra") || !strings.Contains(result.Message, "feat/x") {
			t.Errorf("el mensaje %q debe nombrar las dos ramas", result.Message)
		}
		if got := repositoryState(t, repo); got != state {
			t.Errorf("el rechazo no debe crear nada:\n%s\n---\n%s", state, got)
		}
	})
	t.Run("un worktree cuyo directorio ya no existe", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		planned := plannedWorktree(t, repo, "x", filepath.Base(repo))
		runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", planned, "main")
		if err := os.RemoveAll(planned); err != nil {
			t.Fatal(err)
		}

		result := ensureOf(t, repo, "x")

		wantRefusal(t, result, branch.CodePlanConflict, branch.StatusRefused)
		for _, want := range []string{"git -C", "worktree prune", "axiom branch ensure --change x --cwd"} {
			if !strings.Contains(result.Continuation, want) {
				t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
			}
		}
	})
	t.Run("un directorio con contenido", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		planned := plannedWorktree(t, repo, "x", filepath.Base(repo))
		writeFile(t, filepath.Join(planned, "ajeno.txt"), "no es un worktree\n")
		state := repositoryState(t, repo)

		result := ensureOf(t, repo, "x")

		wantRefusal(t, result, branch.CodePlanConflict, branch.StatusRefused)
		if !strings.Contains(result.Message, planned) && !strings.Contains(result.Message, canonicalDirectory(t, planned)) {
			t.Errorf("el mensaje %q debe nombrar la ruta ocupada %s", result.Message, planned)
		}
		if got := repositoryState(t, repo); got != state {
			t.Errorf("el rechazo no debe crear nada:\n%s\n---\n%s", state, got)
		}
	})
	t.Run("un directorio vacío sí se admite", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		planned := plannedWorktree(t, repo, "x", filepath.Base(repo))
		if err := os.MkdirAll(planned, 0o755); err != nil {
			t.Fatal(err)
		}

		wantReady(t, ensureOf(t, repo, "x"))
	})
}

// REQ-25.11 y D-04: el límite de Windows se comprueba antes de crear nada.
func TestEnsureAppliesTheWindowsPathLimitBeforeCreatingAnything(t *testing.T) {
	repo := embeddedCoreProject(t)
	state := repositoryState(t, repo)
	change := strings.Repeat("c", 200)

	result := newEngine("windows").Ensure(context.Background(), EnsureRequest{CWD: repo, Change: change, Lane: branch.LaneSDD})

	wantRefusal(t, result, branch.CodePathTooLong, branch.StatusRefused)
	wantNothingCreated(t, repo, state, change)
}
