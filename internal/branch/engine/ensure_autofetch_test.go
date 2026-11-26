package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// Pruebas del fetch implícito de `ensure` con el autofetch encendido (RF-BRA-10, RF-BRA-13,
// RF-HUB-07, design D-04) con git real y remotos desnudos locales o rutas inexistentes: no hay
// red. Se usa `in_place` porque `git worktree add` cuesta segundos en Windows y aquí no es lo
// que se comprueba.

var (
	autofetchOn  = fakePrefs{mode: branch.ModeInPlace, source: branch.ModeSourceHub, autofetch: true}
	autofetchOff = fakePrefs{mode: branch.ModeInPlace, source: branch.ModeSourceHub}
)

// timeoutWriter es un escritor cuyo fetch con tiempo máximo se agota, y anota el tiempo pedido.
type timeoutWriter struct {
	gitWriter
	asked *time.Duration
}

func (w timeoutWriter) FetchWithin(_ context.Context, _, _ string, timeout time.Duration) error {
	*w.asked = timeout
	return fmt.Errorf("`git fetch origin`: %w", gitcommon.ErrTimeout)
}

func unreachableOrigin(t *testing.T, repo string) {
	t.Helper()
	runGit(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "no-existe.git"))
}

func autofetchWarnings(result branch.Result) []string {
	var found []string
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "autofetch") {
			found = append(found, warning)
		}
	}
	return found
}

// RF-BRA-10, escenario «fetch implícito»: con el interruptor encendido y sin `--fetch`, un único
// `git fetch origin` (sin `--prune`) desde el checkout principal, antes de resolver la base y de
// toda escritura; la base alcanza el commit nuevo con `fetched=true` y la rama local `main` no
// se mueve.
func TestEnsureAutofetchFetchesImplicitlyBeforeResolvingTheBase(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	localMain := runGit(t, repo, "rev-parse", "refs/heads/main")
	advanced := advanceRemote(t, remote)
	seen := observe(t)

	result := ensureWith(t, autofetchOn, EnsureRequest{CWD: repo, Change: "x"})

	wantReady(t, result)
	base := onlyRepository(t, result).Base
	if base == nil || *base != (branch.BaseResult{Ref: "origin/main", SHA: advanced, Source: "remote-tracking", Fetched: true}) {
		t.Errorf("base = %+v, se esperaba origin/main en %s con fetched=true", base, advanced)
	}
	if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); got != advanced {
		t.Errorf("feat/x apunta a %s, se esperaba el commit traído %s", got, advanced)
	}
	if got := runGit(t, repo, "rev-parse", "refs/heads/main"); got != localMain {
		t.Errorf("la rama local main se movió de %s a %s: el fetch implícito solo toca refs/remotes", localMain, got)
	}
	fetches := fetchInvocations(*seen)
	if len(fetches) != 1 || !reflect.DeepEqual(fetches[0][1:], []string{"fetch", "origin"}) || !pathidentity.SameDirectory(fetches[0][0], repo) {
		t.Fatalf("se esperaba un único `git fetch origin` en %s sin --prune: %v", repo, fetches)
	}
	fetchAt := indexOfInvocation(*seen, func(invocation []string) bool { return invocation[1] == "fetch" })
	writeAt := indexOfInvocation(*seen, isWriteInvocation)
	if fetchAt < 0 || writeAt < 0 || fetchAt > writeAt {
		t.Errorf("el fetch (%d) va antes de toda escritura (%d): %v", fetchAt, writeAt, *seen)
	}
	for _, invocation := range *seen {
		if invocation[1] == "push" || invocation[1] == "pull" || invocation[1] == "ls-remote" || invocation[1] == "merge" {
			t.Errorf("ninguna operación remota ni avance distinta de `fetch origin`: %v", invocation)
		}
	}
	if warnings := autofetchWarnings(result); len(warnings) != 0 {
		t.Errorf("un fetch correcto no avisa: %v", warnings)
	}
}

// RF-BRA-10: con `--base <ref>` el fetch se hace igual y la base sigue siendo la sustituida.
func TestEnsureAutofetchWithBaseExceptionStillFetches(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	advanceRemote(t, remote)
	runGit(t, repo, "branch", "otra")
	seen := observe(t)

	result := ensureWith(t, autofetchOn, EnsureRequest{CWD: repo, Change: "x", Exceptions: branch.Exceptions{Base: "otra"}})

	wantReady(t, result)
	if fetches := fetchInvocations(*seen); len(fetches) != 1 {
		t.Errorf("con --base el fetch se hace igual: %v", fetches)
	}
}

// RF-BRA-10: sin `origin` el repositorio se omite sin aviso y `fetched` queda en falso.
func TestEnsureAutofetchWithoutOriginSkipsSilently(t *testing.T) {
	repo := embeddedCoreProject(t)
	seen := observe(t)

	result := ensureWith(t, autofetchOn, EnsureRequest{CWD: repo, Change: "x"})

	wantReady(t, result)
	if base := onlyRepository(t, result).Base; base == nil || base.Fetched {
		t.Errorf("base = %+v, se esperaba fetched=false", base)
	}
	if fetches := fetchInvocations(*seen); len(fetches) != 0 {
		t.Errorf("sin origin no se lanza fetch: %v", fetches)
	}
	if warnings := autofetchWarnings(result); len(warnings) != 0 {
		t.Errorf("sin origin no hay aviso: %v", warnings)
	}
}

// RF-BRA-10, escenario «fetch implícito fallido»: aviso `el autofetch no pudo traer origin…`,
// `fetched=false` y la ejecución sigue con la base local; con un tiempo agotado, igual.
func TestEnsureAutofetchFailureDegradesToAWarning(t *testing.T) {
	t.Run("remoto inalcanzable", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		unreachableOrigin(t, repo)

		result := ensureWith(t, autofetchOn, EnsureRequest{CWD: repo, Change: "x"})

		wantReady(t, result)
		if base := onlyRepository(t, result).Base; base == nil || base.Fetched || base.Source != "local" {
			t.Errorf("base = %+v, se esperaba la base local con fetched=false", base)
		}
		warnings := autofetchWarnings(result)
		if len(warnings) != 1 || !strings.Contains(warnings[0], "el autofetch no pudo traer origin en ") || !strings.HasSuffix(warnings[0], "se sigue con la base local") {
			t.Errorf("avisos = %v, se esperaba el del autofetch fallido", warnings)
		}
		if runGit(t, repo, "branch", "--list", "feat/x") == "" {
			t.Error("la ejecución sigue y crea la rama")
		}
	})
	t.Run("tiempo agotado a los 8 s", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		remoteWithMain(t, repo)
		var asked time.Duration
		engine := Default()
		engine.prefs = autofetchOn
		engine.writer = timeoutWriter{gitWriter: engine.writer, asked: &asked}

		result := engine.Ensure(context.Background(), EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD})

		wantReady(t, result)
		if asked != 8*time.Second || autofetchFetchTimeout != 8*time.Second {
			t.Errorf("tiempo pedido %v (constante %v), se esperaban 8 s", asked, autofetchFetchTimeout)
		}
		warnings := autofetchWarnings(result)
		if len(warnings) != 1 || !strings.Contains(warnings[0], gitcommon.ErrTimeout.Error()) {
			t.Errorf("avisos = %v, se esperaba el aviso con el tiempo agotado", warnings)
		}
		if base := onlyRepository(t, result).Base; base == nil || base.Fetched {
			t.Errorf("base = %+v, se esperaba fetched=false", base)
		}
	})
}

// RF-BRA-10: `--fetch` explícito conserva su contrato con el interruptor encendido: un fallo es
// `fetch_failed` y no crea nada.
func TestEnsureAutofetchExplicitFetchStillRefusesOnFailure(t *testing.T) {
	repo := embeddedCoreProject(t)
	unreachableOrigin(t, repo)
	state := repositoryState(t, repo)

	result := ensureWith(t, autofetchOn, EnsureRequest{CWD: repo, Change: "x", Fetch: true})

	wantFetchFailed(t, result, repo)
	wantNothingCreated(t, repo, state, "x")
}

// D-04: un `ensure` que termina en `decision_required` no hace fetch.
func TestEnsureAutofetchDecisionRequiredDoesNotFetch(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	seen := observe(t)
	prefs := fakePrefs{mode: branch.ModeAsk, source: branch.ModeSourceHub, autofetch: true}

	result := ensureWith(t, prefs, EnsureRequest{CWD: repo, Change: "x"})

	if result.Status != branch.StatusDecisionRequired {
		t.Fatalf("estado %q (%s), se esperaba decision_required", result.Status, result.Message)
	}
	if fetches := fetchInvocations(*seen); len(fetches) != 0 {
		t.Errorf("decision_required no toca la red: %v", fetches)
	}
}

// RF-BRA-01: `status` no lee el interruptor ni trae nada.
func TestEnsureAutofetchStatusNeverFetches(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	seen := observe(t)
	engine := Default()
	engine.prefs = autofetchOn

	engine.Status(context.Background(), repo, "x", branch.LaneSDD)

	if fetches := fetchInvocations(*seen); len(fetches) != 0 {
		t.Errorf("status no trae nada: %v", fetches)
	}
}

// RF-BRA-10: el sello del intervalo no interviene: dos `ensure` seguidos traen los dos.
func TestEnsureAutofetchIgnoresTheInterval(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	seen := observe(t)

	wantReady(t, ensureWith(t, autofetchOn, EnsureRequest{CWD: repo, Change: "x"}))
	wantReady(t, ensureWith(t, autofetchOn, EnsureRequest{CWD: repo, Change: "x"}))

	if fetches := fetchInvocations(*seen); len(fetches) != 2 {
		t.Errorf("cada `ensure` trae, sin intervalo: %v", fetches)
	}
}

// RF-HUB-07: con el interruptor apagado, sin puerto, con el proyecto sin registrar o con el hub
// ilegible, `ensure` no trae nada, no añade avisos del autofetch y la salida es la misma.
func TestEnsureAutofetchOffBehavesExactlyAsBefore(t *testing.T) {
	baseline := func(t *testing.T) (string, string, *[][]string) {
		repo := embeddedCoreProject(t)
		remoteWithMain(t, repo)
		seen := observe(t)
		return repo, runGit(t, repo, "rev-parse", "refs/remotes/origin/main"), seen
	}
	t.Run("interruptor apagado", func(t *testing.T) {
		repo, stale, seen := baseline(t)

		result := ensureWith(t, autofetchOff, EnsureRequest{CWD: repo, Change: "x"})

		wantReady(t, result)
		if fetches := fetchInvocations(*seen); len(fetches) != 0 {
			t.Errorf("apagado no trae: %v", fetches)
		}
		if base := onlyRepository(t, result).Base; base == nil || base.SHA != stale || base.Fetched {
			t.Errorf("base = %+v, se esperaba la referencia previa sin fetch", base)
		}
		if strings.Contains(resultJSON(t, result), `"fetched": true`) {
			t.Error("apagado no informa fetched=true")
		}
	})
	t.Run("sin puerto de preferencias", func(t *testing.T) {
		repo, _, seen := baseline(t)
		engine := Default()
		engine.prefs = nil

		result := engine.Ensure(context.Background(), EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD})

		wantReady(t, result)
		if fetches := fetchInvocations(*seen); len(fetches) != 0 {
			t.Errorf("prefs == nil equivale a apagado: %v", fetches)
		}
	})
	t.Run("proyecto sin registrar y hub ilegible", func(t *testing.T) {
		hubDir := t.TempDir()
		unregistered := filepath.Join(hubDir, "vacio.json")
		unreadable := filepath.Join(hubDir, "roto.json")
		if err := os.WriteFile(unreadable, []byte("{no es json"), 0o644); err != nil {
			t.Fatal(err)
		}
		for name, hubPath := range map[string]string{"sin registrar": unregistered, "ilegible": unreadable} {
			repo, _, seen := baseline(t)

			result := DefaultWithHubPath(hubPath).Ensure(context.Background(), EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD, Mode: branch.ModeInPlace})

			wantReady(t, result)
			if fetches := fetchInvocations(*seen); len(fetches) != 0 {
				t.Errorf("%s: se comporta como apagado, sin fetch: %v", name, fetches)
			}
			if warnings := autofetchWarnings(result); len(warnings) != 0 {
				t.Errorf("%s: `ensure` no añade avisos del autofetch: %v", name, warnings)
			}
		}
	})
}

// RF-BRA-10, repositorio de specs in situ: con el fetch implícito se trae también (sin base ni
// avance) e informa `fetched`; los de código informan en `base.fetched`. Con `--fetch` explícito
// no se trae. Un fallo del de specs avisa y deja `fetched=false`.
func TestEnsureAutofetchFetchesTheSpecsRepositoryInPlace(t *testing.T) {
	newProject := func(t *testing.T) (master, specs, backend string) {
		master, specs = newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
		sealChangeAt(t, specs, "x", "backend")
		backend = filepath.Join(master, "repo-backend")
		remoteWithMain(t, backend)
		return master, specs, backend
	}
	specsResult := func(t *testing.T, result branch.Result) branch.RepositoryResult {
		t.Helper()
		for _, repository := range result.Repositories {
			if repository.Role == branch.RepoSpecsInPlace {
				return repository
			}
		}
		t.Fatalf("el resultado no incluye el repositorio de specs: %+v", result.Repositories)
		return branch.RepositoryResult{}
	}
	t.Run("trae specs y código", func(t *testing.T) {
		master, specs, backend := newProject(t)
		remoteWithMain(t, specs)
		seen := observe(t)

		result := ensureWith(t, autofetchOn, EnsureRequest{CWD: master, Change: "x"})

		wantReady(t, result)
		fetches := fetchInvocations(*seen)
		if len(fetches) != 2 {
			t.Fatalf("se esperaba un fetch por repositorio (specs y código): %v", fetches)
		}
		for _, repo := range []string{specs, backend} {
			if indexOfInvocation(fetches, func(invocation []string) bool { return pathidentity.SameDirectory(invocation[0], repo) }) < 0 {
				t.Errorf("%s no recibió su fetch: %v", repo, fetches)
			}
		}
		if got := specsResult(t, result); got.Fetched == nil || !*got.Fetched || got.Base != nil {
			t.Errorf("specs: fetched=%v base=%+v, se esperaba fetched=true y sin base", got.Fetched, got.Base)
		}
	})
	t.Run("specs sin origin", func(t *testing.T) {
		master, _, _ := newProject(t)

		result := ensureWith(t, autofetchOn, EnsureRequest{CWD: master, Change: "x"})

		wantReady(t, result)
		if got := specsResult(t, result); got.Fetched == nil || *got.Fetched {
			t.Errorf("specs: fetched=%v, se esperaba false sin origin", got.Fetched)
		}
		if warnings := autofetchWarnings(result); len(warnings) != 0 {
			t.Errorf("sin origin no hay aviso: %v", warnings)
		}
	})
	t.Run("fallo en specs", func(t *testing.T) {
		master, specs, _ := newProject(t)
		unreachableOrigin(t, specs)

		result := ensureWith(t, autofetchOn, EnsureRequest{CWD: master, Change: "x"})

		wantReady(t, result)
		if got := specsResult(t, result); got.Fetched == nil || *got.Fetched {
			t.Errorf("specs: fetched=%v, se esperaba false", got.Fetched)
		}
		if warnings := autofetchWarnings(result); len(warnings) != 1 || !strings.Contains(warnings[0], "el autofetch no pudo traer origin en ") {
			t.Errorf("avisos = %v, se esperaba el del repositorio de specs", warnings)
		}
	})
	t.Run("apagado no informa fetched", func(t *testing.T) {
		master, specs, _ := newProject(t)
		remoteWithMain(t, specs)

		result := ensureWith(t, autofetchOff, EnsureRequest{CWD: master, Change: "x"})

		wantReady(t, result)
		if got := specsResult(t, result); got.Fetched != nil {
			t.Errorf("specs: fetched=%v, con el interruptor apagado no se informa", *got.Fetched)
		}
	})
	t.Run("--fetch explícito no trae specs", func(t *testing.T) {
		master, specs, _ := newProject(t)
		remoteWithMain(t, specs)
		seen := observe(t)

		result := ensureWith(t, autofetchOn, EnsureRequest{CWD: master, Change: "x", Fetch: true})

		wantReady(t, result)
		if fetches := fetchInvocations(*seen); len(fetches) != 1 {
			t.Errorf("--fetch solo trae el repositorio de código: %v", fetches)
		}
		if got := specsResult(t, result); got.Fetched != nil {
			t.Errorf("specs: fetched=%v, con --fetch no se informa", *got.Fetched)
		}
	})
}

// RF-BRA-13: `ensure --archive` hereda el fetch implícito: alcanza el cambio fusionado en el
// remoto sin `--fetch`, y sus rechazos no sugieren `--fetch`.
func TestEnsureAutofetchArchive(t *testing.T) {
	t.Run("alcanza el cambio fusionado sin --fetch", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		remote := remoteWithMain(t, repo)
		advanced := publishChange(t, remote, "x")

		result := ensureWith(t, autofetchOn, archiveRequest(repo))

		wantReady(t, result)
		if base := onlyRepository(t, result).Base; base == nil || base.SHA != advanced || !base.Fetched {
			t.Errorf("base = %+v, se esperaba origin/main en %s con fetched=true", base, advanced)
		}
	})
	t.Run("rechazo tras un fetch correcto", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		remoteWithMain(t, repo)

		result := ensureWith(t, autofetchOn, archiveRequest(repo))

		wantRefusal(t, result, branch.CodeChangeNotIntegrated, branch.StatusRefused)
		if !strings.Contains(result.Message, "ni tras el fetch del autofetch") || strings.Contains(result.Message, "--fetch") {
			t.Errorf("mensaje %q, se esperaba «tras el fetch del autofetch» sin --fetch", result.Message)
		}
		if strings.Contains(result.Continuation, "--fetch") || !strings.Contains(result.Continuation, "--archive") {
			t.Errorf("continuación %q, no debe sugerir --fetch y sí repetir --archive", result.Continuation)
		}
	})
	t.Run("rechazo tras un fetch fallido", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		unreachableOrigin(t, repo)

		result := ensureWith(t, autofetchOn, archiveRequest(repo))

		wantRefusal(t, result, branch.CodeChangeNotIntegrated, branch.StatusRefused)
		if strings.Contains(result.Message, "--fetch") {
			t.Errorf("mensaje %q, no debe nombrar --fetch", result.Message)
		}
		if !strings.Contains(result.Continuation, "el autofetch no pudo traer el remoto; comprueba la red y repite `axiom branch ensure --cwd ") || strings.Contains(result.Continuation, "--fetch") {
			t.Errorf("continuación %q, se esperaba el aviso del autofetch fallido sin --fetch", result.Continuation)
		}
	})
	t.Run("apagado sigue sugiriendo --fetch", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		remoteWithMain(t, repo)

		result := ensureWith(t, autofetchOff, archiveRequest(repo))

		wantRefusal(t, result, branch.CodeChangeNotIntegrated, branch.StatusRefused)
		if !strings.Contains(result.Continuation, "--archive --fetch") {
			t.Errorf("continuación %q, apagado conserva la sugerencia de --fetch", result.Continuation)
		}
	})
}
