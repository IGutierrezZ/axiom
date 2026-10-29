package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Excepciones por cambio de `ensure` (REQ-25.12, REQ-25.3, design D-04 y D-05): `--prefix`,
// `--branch` y `--base` se registran por cambio, una excepción posterior contradictoria es
// `plan_conflict`, la repetición idéntica es idempotente y un nombre inválido de la invocación
// es `invalid_name` sin efectos. Con git real, entorno aislado y sin red. Salvo la prueba de la
// ruta del worktree, usan `in_place`, que no crea worktrees (lo más caro en Windows).

// ensureException pide `--mode in_place` con las excepciones dadas y un hub que no pregunta.
func ensureException(t *testing.T, cwd, change string, exceptions branch.Exceptions) branch.Result {
	t.Helper()
	return ensureWith(t, hubWorktree, EnsureRequest{CWD: cwd, Change: change, Mode: branch.ModeInPlace, Exceptions: exceptions})
}

// commitAll confirma todo lo que haya en el árbol de trabajo: el kickoff sellado del fixture
// quedaría sin seguimiento y ensuciaría el árbol de `in_place`.
func commitAll(t *testing.T, repo, message string) {
	t.Helper()
	runGit(t, repo, "add", "-A")
	commit(t, repo, message)
}

// REQ-25.12, escenario «`--branch` fija el nombre de la rama del cambio»: la rama del cambio es
// la indicada y no `feat/x`; el worktree sigue componiéndose con el nombre del cambio, no con el
// de la rama (`<raíz>/<cambio>/<repo>`), y la excepción queda registrada.
func TestEnsureExceptionBranchFixesTheChangeBranchName(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)

	result := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x", Exceptions: branch.Exceptions{Branch: "inc/inc-25-x"}})

	wantReady(t, result)
	repository := onlyRepository(t, result)
	if len(repository.Branches) != 1 {
		t.Fatalf("ramas = %+v, se esperaba solo la del cambio", repository.Branches)
	}
	feature := branchByName(t, repository, "inc/inc-25-x")
	if !feature.Exists || feature.Kind != branch.KindChange || feature.PRTarget != "main" {
		t.Errorf("rama del cambio = %+v", feature)
	}
	wantSameDirectory(t, "worktree de la rama", feature.Worktree, plannedWorktree(t, repo, "x", filepath.Base(repo)))
	if got := runGit(t, repo, "branch", "--list", "feat/x"); got != "" {
		t.Errorf("no debía crearse feat/x, hay %q", got)
	}
	if record := loadRecord(t, repo, "x"); record.Exceptions != (branch.Exceptions{Branch: "inc/inc-25-x"}) {
		t.Errorf("excepciones registradas = %+v, se esperaba solo la rama", record.Exceptions)
	}
}

// REQ-25.12, escenario «`--branch` con ramas de rol deriva los nombres»: la rama de seguimiento
// es la indicada y las de rol son `<rama>-<rol>`.
func TestEnsureExceptionBranchWithRoleBranchesDerivesTheRoleNames(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))
	sealChange(t, repo, "x", "core", "qa")
	commitAll(t, repo, "kickoff")

	result := ensureException(t, repo, "x", branch.Exceptions{Branch: "inc/x"})

	wantReady(t, result)
	repository := onlyRepository(t, result)
	var names []string
	for _, planned := range repository.Branches {
		names = append(names, planned.Name)
	}
	wantNamesEqual(t, "ramas", names, "inc/x", "inc/x-core", "inc/x-qa")
	if got := runGit(t, repo, "branch", "--list", "feat/*"); got != "" {
		t.Errorf("no debía crearse ninguna rama feat/*, hay %q", got)
	}
	if got := currentBranch(t, repo); got != "inc/x" {
		t.Errorf("el repositorio está en %q, se esperaba la rama de seguimiento inc/x", got)
	}
	if got := runGit(t, repo, "rev-parse", "inc/x-core"); got != runGit(t, repo, "rev-parse", "inc/x") {
		t.Errorf("la rama de rol sale de la de seguimiento: %s", got)
	}
}

func wantNamesEqual(t *testing.T, what string, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("%s = %v, se esperaba %v", what, got, want)
	}
}

// REQ-25.12, escenario «`--base` fija la base», y REQ-25.10 (`base_not_found` nombra la
// referencia fijada): `base.ref` es la indicada, la rama del cambio sale de ella y una base que
// no existe es `base_not_found` sin crear nada.
func TestEnsureExceptionBaseFixesTheBaseOfTheChange(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	runGit(t, repo, "switch", "-q", "-c", "release/2")
	commit(t, repo, "versión 2")
	release := runGit(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "switch", "-q", "main")

	t.Run("la base fijada sustituye a la de integración", func(t *testing.T) {
		result := ensureException(t, repo, "x", branch.Exceptions{Base: "release/2"})

		wantReady(t, result)
		repository := onlyRepository(t, result)
		if repository.Base == nil || *repository.Base != (branch.BaseResult{Ref: "release/2", SHA: release, Source: "local"}) {
			t.Errorf("base = %+v, se esperaba release/2 en %s", repository.Base, release)
		}
		if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); got != release {
			t.Errorf("feat/x apunta a %s, se esperaba la base fijada %s", got, release)
		}
	})
	t.Run("una base inexistente es base_not_found y no crea nada", func(t *testing.T) {
		state := inPlaceState(t, repo)

		result := ensureException(t, repo, "y", branch.Exceptions{Base: "no-existe"})

		wantRefusal(t, result, branch.CodeBaseNotFound, branch.StatusRefused)
		if !strings.Contains(result.Message, "no-existe") {
			t.Errorf("el mensaje %q debe nombrar la base fijada", result.Message)
		}
		if got := inPlaceState(t, repo); got != state {
			t.Errorf("el rechazo no debe cambiar el repositorio:\n%s\n---\n%s", state, got)
		}
		wantNoRegistryFile(t, repo, "y")
	})
}

// REQ-25.12, escenarios «Una excepción posterior contradictoria se rechaza» y «Una repetición
// idéntica de la excepción es idempotente»: contradecir lo materializado (cualquiera de las tres
// excepciones) es `plan_conflict` con una continuación ejecutable que repite la excepción
// registrada, sin escribir nada; repetirla o no indicarla da el mismo JSON byte a byte.
func TestEnsureExceptionALaterContradictionIsPlanConflictAndARepeatIsIdempotent(t *testing.T) {
	repo := embeddedCoreProject(t)
	first := ensureException(t, repo, "x", branch.Exceptions{Branch: "inc/x"})
	wantReady(t, first)
	want := resultJSON(t, first)
	state := inPlaceState(t, repo)
	recorded := loadRecord(t, repo, "x")

	t.Run("la repetición idéntica es idempotente y no escribe", func(t *testing.T) {
		seen := observe(t)

		again := ensureException(t, repo, "x", branch.Exceptions{Branch: "inc/x"})

		wantReady(t, again)
		if got := resultJSON(t, again); got != want {
			t.Errorf("repetir la excepción debe dar el mismo JSON:\n%s\n---\n%s", want, got)
		}
		wantNoWrites(t, *seen, "repetir la misma excepción no escribe nada")
	})
	t.Run("sin indicar la excepción rige la registrada", func(t *testing.T) {
		again := ensureException(t, repo, "x", branch.Exceptions{})

		wantReady(t, again)
		if got := resultJSON(t, again); got != want {
			t.Errorf("sin excepción debe regir la registrada:\n%s\n---\n%s", want, got)
		}
	})
	for name, tc := range map[string]struct {
		exceptions branch.Exceptions
		mentions   []string
	}{
		"otra rama":    {branch.Exceptions{Branch: "feat/x"}, []string{"--branch", "inc/x", "feat/x"}},
		"otro prefijo": {branch.Exceptions{Prefix: "doc"}, []string{"--prefix", "doc"}},
		"otra base":    {branch.Exceptions{Base: "main"}, []string{"--base", "main"}},
	} {
		t.Run("contradicción: "+name, func(t *testing.T) {
			seen := observe(t)

			result := ensureException(t, repo, "x", tc.exceptions)

			wantRefusal(t, result, branch.CodePlanConflict, branch.StatusRefused)
			for _, want := range tc.mentions {
				if !strings.Contains(result.Message, want) {
					t.Errorf("el mensaje %q debe contener %q", result.Message, want)
				}
			}
			for _, want := range []string{"axiom branch ensure --change x", "--branch inc/x", "--cwd " + pathquote.Quote(repo)} {
				if !strings.Contains(result.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
				}
			}
			wantNoWrites(t, *seen, "una excepción contradictoria no escribe nada")
			if got := inPlaceState(t, repo); got != state {
				t.Errorf("el rechazo no debe cambiar el repositorio:\n%s\n---\n%s", state, got)
			}
			if got := loadRecord(t, repo, "x"); got.Exceptions != recorded.Exceptions {
				t.Errorf("el registro cambió: %+v, se esperaba %+v", got.Exceptions, recorded.Exceptions)
			}
		})
	}
}

// REQ-25.12 y REQ-25.14: mientras el cambio no tiene ramas materializadas (un multirepo antes del
// sellado solo tiene el repositorio de specs, sin ramas), la excepción aún no contradice nada: se
// registra y una nueva la sustituye; cuando el sellado añade el repositorio de código, la rama
// registrada se aplica sin repetir la excepción.
func TestEnsureExceptionIsReplacedWhileNothingIsMaterializedYet(t *testing.T) {
	t.Parallel()
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	request := func(exceptions branch.Exceptions) EnsureRequest {
		return EnsureRequest{CWD: master, Change: "x", Mode: branch.ModeInPlace, Exceptions: exceptions}
	}

	wantReady(t, ensureWith(t, hubWorktree, request(branch.Exceptions{Branch: "inc/x"})))
	if got := loadRecordAt(t, specs, branch.LaneSDD, "x"); got.Exceptions.Branch != "inc/x" {
		t.Fatalf("excepciones registradas = %+v, se esperaba inc/x", got.Exceptions)
	}
	wantReady(t, ensureWith(t, hubWorktree, request(branch.Exceptions{Branch: "inc/y"})))
	if got := loadRecordAt(t, specs, branch.LaneSDD, "x"); got.Exceptions.Branch != "inc/y" {
		t.Fatalf("excepciones registradas = %+v, sin ramas materializadas la nueva sustituye a la anterior", got.Exceptions)
	}

	sealChangeAt(t, specs, "x", "backend")
	wantReady(t, ensureWith(t, hubWorktree, request(branch.Exceptions{})))
	backend := filepath.Join(master, "repo-backend")
	if got := runGit(t, backend, "branch", "--list", "inc/y"); !strings.Contains(got, "inc/y") {
		t.Errorf("repo-backend debe tener la rama registrada inc/y, hay %q", got)
	}
	if got := runGit(t, backend, "branch", "--list", "feat/*", "inc/x"); got != "" {
		t.Errorf("no debían crearse ni feat/x ni inc/x, hay %q", got)
	}
}

// loadRecordAt carga el registro de un cambio del carril dado, que vive en el git common dir del
// repositorio del planning home.
func loadRecordAt(t *testing.T, planningRepo string, lane branch.Lane, change string) *branch.Record {
	t.Helper()
	record, err := registry.New().Load(commonDir(planningRepo), lane, change)
	if err != nil || record == nil {
		t.Fatalf("el registro de %s (%s) debía existir: %+v, %v", change, lane, record, err)
	}
	return record
}

// REQ-25.3 y REQ-25.12: un `--prefix`, un `--branch` o un `--base` inválidos de la invocación
// son `invalid_name` sin efectos (ni ramas, ni worktrees, ni registro, ni su bloqueo), con el
// campo y el valor en los detalles; un prefijo inválido de `axiom.yaml` sigue siendo
// `config_invalid`.
func TestEnsureExceptionInvalidNamesAreInvalidNameWithoutEffects(t *testing.T) {
	repo := embeddedCoreProject(t)
	state := inPlaceState(t, repo)

	for name, tc := range map[string]struct {
		exceptions branch.Exceptions
		field      string
		value      string
	}{
		"prefijo con espacios":         {branch.Exceptions{Prefix: "mi prefijo"}, "--prefix", "mi prefijo"},
		"prefijo con barra final":      {branch.Exceptions{Prefix: "feat/"}, "--prefix", "feat/"},
		"rama con puntos consecutivos": {branch.Exceptions{Branch: "inc/x..y"}, "--branch", "inc/x..y"},
		"rama que empieza por guion":   {branch.Exceptions{Branch: "-inc"}, "--branch", "-inc"},
		"base que empieza por guion":   {branch.Exceptions{Base: "--detach"}, "--base", "--detach"},
	} {
		t.Run(name, func(t *testing.T) {
			seen := observe(t)

			result := ensureException(t, repo, "x", tc.exceptions)

			wantRefusal(t, result, branch.CodeInvalidName, branch.StatusRefused)
			if result.Details["field"] != tc.field || result.Details["value"] != tc.value {
				t.Errorf("details = %v, se esperaba el campo %s con el valor %q", result.Details, tc.field, tc.value)
			}
			if !strings.Contains(result.Continuation, "axiom branch ensure --change x") || !strings.Contains(result.Continuation, pathquote.Quote(repo)) {
				t.Errorf("la continuación %q debe ser la orden de ensure con el cambio y --cwd", result.Continuation)
			}
			wantNoWrites(t, *seen, "un nombre inválido no escribe nada")
			if got := inPlaceState(t, repo); got != state {
				t.Errorf("el rechazo no debe cambiar el repositorio:\n%s\n---\n%s", state, got)
			}
			if _, err := os.Stat(filepath.Join(commonDir(repo), "axiom")); err == nil {
				t.Error("un nombre inválido se rechaza antes de bloquear el cambio: no debe existir la carpeta del registro")
			}
		})
	}

	t.Run("un prefijo inválido de axiom.yaml es config_invalid", func(t *testing.T) {
		configured := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, "git:\n  branch_prefix: \"mi prefijo//\"\n"))
		configuredState := inPlaceState(t, configured)

		result := ensureException(t, configured, "x", branch.Exceptions{Base: "main"})

		wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
		if got := inPlaceState(t, configured); got != configuredState {
			t.Errorf("el rechazo no debe cambiar el repositorio:\n%s\n---\n%s", configuredState, got)
		}
	})
}
