package engine

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// Pruebas del ciclo base de `axiom autofetch run` (RF-BRA-39, RF-HUB-07, design D-01, D-03, D-06
// y D-09) con git real y remotos desnudos locales o rutas inexistentes: no hay red. Tocan
// variables de paquete (intervalo, presupuesto) y el observador global de gitcommon, así que
// ninguna es paralela.

// autofetchStatePrefs es un doble del puerto con el estado del interruptor y el aviso a medida.
type autofetchStatePrefs struct {
	fakePrefs
	enabled bool
	state   branch.AutofetchPrefState
	warning string
}

func (p autofetchStatePrefs) Autofetch(string) (bool, branch.AutofetchPrefState, string) {
	return p.enabled, p.state, p.warning
}

// panicWriter es un escritor cuyo fetch con tiempo máximo entra en pánico.
type panicWriter struct{ gitWriter }

func (panicWriter) FetchWithin(context.Context, string, string, time.Duration) error {
	panic("fallo simulado del fetch")
}

// blockingWriter es un escritor cuyo fetch espera a que el contexto termine, y anota el tiempo
// máximo que se le pidió.
type blockingWriter struct {
	gitWriter
	asked *time.Duration
}

func (w blockingWriter) FetchWithin(ctx context.Context, _, _ string, timeout time.Duration) error {
	*w.asked = timeout
	<-ctx.Done()
	return ctx.Err()
}

// autofetchRun ejecuta un ciclo con el reloj dado (cero: el del sistema).
func autofetchRun(t *testing.T, prefs branch.Prefs, cwd string, clock time.Time) branch.AutofetchResult {
	t.Helper()
	engine := Default()
	engine.prefs = prefs
	if !clock.IsZero() {
		engine.now = func() time.Time { return clock }
	}
	return engine.Autofetch(context.Background(), AutofetchRequest{CWD: cwd})
}

func stampOf(commonDir string) string {
	return filepath.Join(commonDir, "axiom", "autofetch", "v1", "state.json")
}

func noRemoteOperations(t *testing.T, seen [][]string) {
	t.Helper()
	for _, invocation := range seen {
		if slices.Contains([]string{"push", "pull", "ls-remote", "commit"}, invocation[1]) || (invocation[1] == "merge" && !slices.Contains(invocation, "--ff-only")) || slices.Contains(invocation, "--prune") {
			t.Errorf("el ciclo base solo trae origin y avanza con merge --ff-only, sin publicar ni podar: %v", invocation)
		}
	}
}

// RF-BRA-39 y RF-HUB-07: con el interruptor apagado (o sin puerto) no se lanza ningún proceso de
// git ni se escribe nada, y el resultado es `disabled` con la lista de avisos vacía.
func TestAutofetchCycleDisabledRunsNoGitAndWritesNothing(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	seen := observe(t)

	for name, prefs := range map[string]branch.Prefs{"apagado": autofetchOff, "sin puerto": nil} {
		result := autofetchRun(t, prefs, repo, time.Time{})
		if result.Autofetch != branch.AutofetchStateDisabled || result.Fetch != branch.AutofetchFetchNotAttempted || result.Schema != branch.AutofetchResultSchema || result.Operation != "autofetch run" {
			t.Errorf("%s: resultado %+v, se esperaba disabled sin red intentada", name, result)
		}
		if result.Notices == nil || len(result.Notices) != 0 || len(result.Warnings) != 0 {
			t.Errorf("%s: avisos %v / %v, se esperaba [] y ninguno", name, result.Notices, result.Warnings)
		}
	}
	if len(*seen) != 0 {
		t.Errorf("apagado no lanza procesos de git: %v", *seen)
	}
	if _, err := os.Stat(stampOf(filepath.Join(repo, ".git"))); err == nil {
		t.Error("apagado no escribe el sello")
	}
}

// RF-BRA-39: con el interruptor encendido, un único `git fetch origin` (sin `--prune`) desde el
// checkout principal; solo cambian las referencias remotas, y el sello queda escrito.
func TestAutofetchCycleFetchesOriginAndWritesTheStamp(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	localMain := runGit(t, repo, "rev-parse", "refs/heads/main")
	advanced := advanceRemote(t, remote)
	seen := observe(t)
	clock := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)

	result := autofetchRun(t, autofetchOn, repo, clock)

	if result.Autofetch != branch.AutofetchStateEnabled || result.Fetch != branch.AutofetchFetchDone || result.LastFetch != "2026-11-26T10:00:00Z" {
		t.Fatalf("resultado %+v, se esperaba enabled con fetch done y su sello", result)
	}
	if !pathidentity.SameDirectory(result.Repository, repo) || !pathidentity.SameDirectory(result.Project, repo) {
		t.Errorf("repositorio %q y proyecto %q, se esperaba %s", result.Repository, result.Project, repo)
	}
	if got := runGit(t, repo, "rev-parse", "refs/remotes/origin/main"); got != advanced {
		t.Errorf("origin/main = %s, se esperaba el commit traído %s", got, advanced)
	}
	if got := runGit(t, repo, "rev-parse", "refs/heads/main"); got != advanced {
		t.Errorf("la rama local main está en %s (era %s), se esperaba el avance por fast-forward a %s", got, localMain, advanced)
	}
	fetches := fetchInvocations(*seen)
	if len(fetches) != 1 || !reflect.DeepEqual(fetches[0][1:], []string{"fetch", "origin"}) || !pathidentity.SameDirectory(fetches[0][0], repo) {
		t.Errorf("se esperaba un único `git fetch origin` en %s: %v", repo, fetches)
	}
	noRemoteOperations(t, *seen)
	if payload, err := os.ReadFile(stampOf(filepath.Join(repo, ".git"))); err != nil || !strings.Contains(string(payload), `"lastFetch":"2026-11-26T10:00:00Z"`) {
		t.Errorf("sello = %q (%v), se esperaba lastFetch en RFC 3339 UTC", payload, err)
	}
}

// RF-BRA-39, intervalo mínimo: dentro de él se omite la red, pasado el tiempo se repite; el
// sello solo avanza con un fetch correcto.
func TestAutofetchCycleIntervalSkipsTheNetworkAndRepeatsAfterwards(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	first := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	if result := autofetchRun(t, autofetchOn, repo, first); result.Fetch != branch.AutofetchFetchDone {
		t.Fatalf("el primer ciclo debía traer: %+v", result)
	}
	seen := observe(t)

	inside := autofetchRun(t, autofetchOn, repo, first.Add(autofetchInterval-time.Second))
	if inside.Fetch != branch.AutofetchFetchSkippedInterval || inside.LastFetch != "2026-11-26T10:00:00Z" || len(inside.Notices) != 0 {
		t.Errorf("dentro del intervalo: %+v, se esperaba skipped_interval con el sello previo", inside)
	}
	if fetches := fetchInvocations(*seen); len(fetches) != 0 {
		t.Errorf("dentro del intervalo no hay red: %v", fetches)
	}

	after := autofetchRun(t, autofetchOn, repo, first.Add(autofetchInterval))
	if after.Fetch != branch.AutofetchFetchDone || after.LastFetch != "2026-11-26T10:05:00Z" {
		t.Errorf("pasado el intervalo: %+v, se esperaba un fetch nuevo con el sello nuevo", after)
	}
	if fetches := fetchInvocations(*seen); len(fetches) != 1 {
		t.Errorf("pasado el intervalo se trae una vez: %v", fetches)
	}
}

// RF-BRA-39 y D-06: un sello ausente, ilegible, con una fecha ilegible o futura cuenta como
// «sin fetch previo»; no se puede escribir, es un aviso y el ciclo sigue.
func TestAutofetchStampAbsentUnreadableOrFutureMeansNoPreviousFetch(t *testing.T) {
	now := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	for name, content := range map[string]string{
		"ausente":        "",
		"ilegible":       "no es json",
		"fecha ilegible": `{"lastFetch":"ayer"}`,
		"futuro":         `{"lastFetch":"2026-11-26T11:00:00Z"}`,
	} {
		t.Run(name, func(t *testing.T) {
			repo := embeddedCoreProject(t)
			remoteWithMain(t, repo)
			if content != "" {
				writeFile(t, stampOf(filepath.Join(repo, ".git")), content)
			}
			seen := observe(t)

			result := autofetchRun(t, autofetchOn, repo, now)

			if result.Fetch != branch.AutofetchFetchDone || result.LastFetch != "2026-11-26T10:00:00Z" || len(fetchInvocations(*seen)) != 1 {
				t.Errorf("resultado %+v, se esperaba un fetch correcto con el sello nuevo", result)
			}
		})
	}
}

func TestAutofetchStampWriteFailureIsAWarningAndTheCycleContinues(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	// Un fichero donde debería estar la carpeta del sello impide crearlo; el resto de
	// `.git/axiom` (el registro de cambios) sigue legible en todas las plataformas.
	writeFile(t, filepath.Join(repo, ".git", "axiom", "autofetch"), "no es una carpeta")

	result := autofetchRun(t, autofetchOn, repo, time.Time{})

	if result.Fetch != branch.AutofetchFetchDone || len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "sello") {
		t.Errorf("resultado %+v, se esperaba fetch done con un aviso del sello", result)
	}
}

// RF-BRA-39: sin `origin` no hay red ni aviso ni sello.
func TestAutofetchCycleWithoutOriginIsNoOriginWithoutNoticeOrStamp(t *testing.T) {
	repo := embeddedCoreProject(t)
	seen := observe(t)

	result := autofetchRun(t, autofetchOn, repo, time.Time{})

	if result.Fetch != branch.AutofetchFetchNoOrigin || len(result.Notices) != 0 || len(result.Warnings) != 0 {
		t.Errorf("resultado %+v, se esperaba no_origin sin avisos", result)
	}
	if fetches := fetchInvocations(*seen); len(fetches) != 0 {
		t.Errorf("sin origin no se lanza fetch: %v", fetches)
	}
	if _, err := os.Stat(stampOf(filepath.Join(repo, ".git"))); err == nil {
		t.Error("sin fetch correcto no hay sello")
	}
}

// RF-BRA-39, remoto inalcanzable: aviso `fetch_failed`, sin sello y sin cambios.
func TestAutofetchCycleUnreachableRemoteIsANoticeWithoutStamp(t *testing.T) {
	repo := embeddedCoreProject(t)
	unreachableOrigin(t, repo)

	result := autofetchRun(t, autofetchOn, repo, time.Time{})

	if result.Fetch != branch.AutofetchFetchFailed || len(result.Notices) != 1 || result.Notices[0].Kind != branch.AutofetchNoticeFetchFailed || result.Integration != nil {
		t.Fatalf("resultado %+v, se esperaba failed con un aviso fetch_failed y sin integración", result)
	}
	if !strings.Contains(result.Notices[0].Message, "autofetch") || result.LastFetch != "" {
		t.Errorf("aviso %q y sello %q", result.Notices[0].Message, result.LastFetch)
	}
	if _, err := os.Stat(stampOf(filepath.Join(repo, ".git"))); err == nil {
		t.Error("un fetch fallido no escribe el sello")
	}
}

// RF-BRA-39: un fetch que agota su tiempo (doble de `ErrTimeout`) es un aviso `fetch_timeout`, con
// el tiempo máximo de 8 s por fetch, sin sello y sin avance.
func TestAutofetchCycleFetchTimeoutIsANoticeWithoutStamp(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	var asked time.Duration
	engine := Default()
	engine.prefs = autofetchOn
	engine.writer = timeoutWriter{gitWriter: engine.writer, asked: &asked}
	localMain := runGit(t, repo, "rev-parse", "refs/heads/main")

	result := engine.Autofetch(context.Background(), AutofetchRequest{CWD: repo})

	if result.Fetch != branch.AutofetchFetchTimeout || len(result.Notices) != 1 || result.Notices[0].Kind != branch.AutofetchNoticeFetchTimeout || result.Integration != nil {
		t.Fatalf("resultado %+v, se esperaba timeout con un aviso fetch_timeout", result)
	}
	if asked != 8*time.Second {
		t.Errorf("tiempo máximo pedido al fetch = %s, se esperaba 8s", asked)
	}
	if got := runGit(t, repo, "rev-parse", "refs/heads/main"); got != localMain {
		t.Errorf("sin fetch no hay avance: main pasó de %s a %s", localMain, got)
	}
	if _, err := os.Stat(stampOf(filepath.Join(repo, ".git"))); err == nil {
		t.Error("un fetch agotado no escribe el sello")
	}
}

// Un pánico dentro del ciclo se devuelve como resultado, nunca se propaga.
func TestAutofetchBudgetRecoversAPanicAsAResult(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	engine := Default()
	engine.prefs = autofetchOn
	engine.writer = panicWriter{gitWriter: engine.writer}

	result := engine.Autofetch(context.Background(), AutofetchRequest{CWD: repo})

	if result.Autofetch != branch.AutofetchStateEnabled || len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "fallo simulado del fetch") {
		t.Errorf("resultado %+v, se esperaba el pánico como aviso del resultado parcial", result)
	}
	if result.Notices == nil {
		t.Error("los avisos nunca son nulos")
	}
}

// D-09: el presupuesto global acota el ciclo. Con un fetch que no termina, el ciclo vuelve dentro
// de la cota con `timeout`; y al fetch se le pide como máximo lo que queda del presupuesto.
func TestAutofetchBudgetBoundsTheWholeCycle(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	t.Cleanup(func(budget time.Duration) func() { return func() { autofetchRunBudget = budget } }(autofetchRunBudget))
	autofetchRunBudget = 150 * time.Millisecond
	var asked time.Duration
	engine := Default()
	engine.prefs = autofetchOn
	engine.writer = blockingWriter{gitWriter: engine.writer, asked: &asked}

	started := time.Now()
	result := engine.Autofetch(context.Background(), AutofetchRequest{CWD: repo})

	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Errorf("el ciclo tardó %s con un presupuesto de 150ms", elapsed)
	}
	if result.Fetch != branch.AutofetchFetchTimeout || len(result.Notices) != 1 || result.Notices[0].Kind != branch.AutofetchNoticeFetchTimeout {
		t.Errorf("resultado %+v, se esperaba timeout por presupuesto", result)
	}
	if asked <= 0 || asked > 150*time.Millisecond {
		t.Errorf("al fetch se le pidió %s, se esperaba como mucho lo que queda del presupuesto (150ms)", asked)
	}
}

func TestAutofetchBudgetKeepsTheFetchLimitWhenItFitsAndDeclaresTheReserve(t *testing.T) {
	if autofetchRunBudget != 10*time.Second || autofetchFetchTimeout != 8*time.Second || autofetchInterval != 5*time.Minute || autofetchAdvanceReserve != time.Second {
		t.Errorf("cotas %s/%s/%s/%s, se esperaban 10s/8s/5m/1s", autofetchRunBudget, autofetchFetchTimeout, autofetchInterval, autofetchAdvanceReserve)
	}
	ctx, cancel := context.WithTimeout(context.Background(), autofetchRunBudget)
	defer cancel()
	if got := fetchTimeoutWithin(ctx); got != autofetchFetchTimeout {
		t.Errorf("con presupuesto de sobra el fetch conserva su límite: %s", got)
	}
	expired, cancelExpired := context.WithTimeout(context.Background(), -time.Second)
	defer cancelExpired()
	if got := fetchTimeoutWithin(expired); got <= 0 || got > time.Millisecond {
		t.Errorf("con el plazo vencido el fetch se corta de inmediato: %s", got)
	}
}

// D-03: proyecto no registrado y hub ilegible dan apagado con el aviso del puerto y sin red.
func TestAutofetchProjectUnregisteredOrUnreadableHubIsOffWithTheWarning(t *testing.T) {
	repo := embeddedCoreProject(t)
	remoteWithMain(t, repo)
	for name, prefs := range map[string]autofetchStatePrefs{
		"no registrado": {state: branch.AutofetchUnregistered, warning: "el proyecto no está registrado en el hub"},
		"hub ilegible":  {state: branch.AutofetchUnreadable, warning: "no se pudo leer el hub"},
	} {
		t.Run(name, func(t *testing.T) {
			seen := observe(t)

			result := autofetchRun(t, prefs, repo, time.Time{})

			if result.Autofetch != branch.AutofetchStateDisabled || result.Fetch != branch.AutofetchFetchNotAttempted || !reflect.DeepEqual(result.Warnings, []string{prefs.warning}) {
				t.Errorf("resultado %+v, se esperaba apagado con el aviso del puerto", result)
			}
			if len(*seen) != 0 {
				t.Errorf("sin interruptor no hay git: %v", *seen)
			}
		})
	}
}

// D-03: la identidad se resuelve solo con ficheros: desde un subdirectorio, con `GIT_DIR` en el
// entorno y fuera de un proyecto Axiom no se lanza ningún proceso de git.
func TestAutofetchProjectResolvesByFilesFromASubdirectoryWithoutGit(t *testing.T) {
	repo := embeddedCoreProject(t)
	sub := filepath.Join(repo, "internal", "pkg")
	writeFile(t, filepath.Join(sub, "x.go"), "package pkg\n")
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "no-existe"))
	seen := observe(t)

	project, refusal := Default().AutofetchProject(sub)

	if refusal != nil {
		t.Fatalf("rechazo inesperado: %+v", refusal)
	}
	if !pathidentity.SameDirectory(project.Master, repo) || !pathidentity.SameDirectory(project.Checkout, repo) || !pathidentity.SameDirectory(project.GitRoot, repo) {
		t.Errorf("proyecto %+v, se esperaba la carpeta %s", project, repo)
	}
	if len(*seen) != 0 {
		t.Errorf("la identidad por ficheros no lanza git: %v", *seen)
	}

	outside := t.TempDir()
	if _, refusal := Default().AutofetchProject(outside); refusal == nil {
		t.Error("fuera de un proyecto Axiom se esperaba un rechazo")
	}
	if result := autofetchRun(t, autofetchOn, outside, time.Time{}); result.Autofetch != branch.AutofetchStateDisabled || len(result.Warnings) != 1 || len(*seen) != 0 {
		t.Errorf("fuera de un proyecto: %+v y %v", result, *seen)
	}
}

// D-03: desde un worktree de feature enlazado la maestra es la del checkout principal, el fetch va
// al checkout principal y el worktree no se toca.
func TestAutofetchProjectFromALinkedFeatureWorktreeFetchesTheMainCheckout(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	feature := filepath.Join(t.TempDir(), "feat-x")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", feature)
	featureHead := runGit(t, feature, "rev-parse", "HEAD")
	advanced := advanceRemote(t, remote)
	seen := observe(t)

	result := autofetchRun(t, autofetchOn, feature, time.Time{})

	if result.Fetch != branch.AutofetchFetchDone || !pathidentity.SameDirectory(result.Project, repo) || !pathidentity.SameDirectory(result.Repository, repo) {
		t.Fatalf("resultado %+v, se esperaba el proyecto y el repositorio del checkout principal %s", result, repo)
	}
	fetches := fetchInvocations(*seen)
	if len(fetches) != 1 || !pathidentity.SameDirectory(fetches[0][0], repo) {
		t.Errorf("el fetch va al checkout principal %s: %v", repo, fetches)
	}
	if got := runGit(t, repo, "rev-parse", "refs/remotes/origin/main"); got != advanced {
		t.Errorf("origin/main = %s, se esperaba %s", got, advanced)
	}
	if got := runGit(t, feature, "rev-parse", "HEAD"); got != featureHead {
		t.Errorf("el worktree de feature se movió de %s a %s", featureHead, got)
	}
	if _, err := os.Stat(stampOf(filepath.Join(repo, ".git"))); err != nil {
		t.Errorf("el sello vive bajo el git common dir del checkout principal: %v", err)
	}
}

// RF-BRA-39, multirepo: solo se trae el repositorio de specs, también desde un repositorio de
// código y con el puntero `.axiom-workspace`; el remoto del repositorio de código no se toca.
func TestAutofetchProjectMultirepoFetchesOnlyTheSpecsRepository(t *testing.T) {
	for _, pointer := range []bool{false, true} {
		name := "axiom.yaml en la maestra"
		if pointer {
			name = "puntero .axiom-workspace"
		}
		t.Run(name, func(t *testing.T) {
			master, specs := newMultirepo(t, map[string][]string{"core": {"repo-core"}}, "repo-core")
			code := filepath.Join(master, "repo-core")
			if pointer {
				config, err := os.ReadFile(filepath.Join(master, "axiom.yaml"))
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(specs, "axiom.yaml"), string(config))
				writeFile(t, filepath.Join(master, ".axiom-workspace"), "config: repo-specs/axiom.yaml\n")
				if err := os.Remove(filepath.Join(master, "axiom.yaml")); err != nil {
					t.Fatal(err)
				}
			}
			specsRemote := remoteWithMain(t, specs)
			codeRemote := remoteWithMain(t, code)
			specsAdvanced := advanceRemote(t, specsRemote)
			codeBefore := runGit(t, code, "rev-parse", "refs/remotes/origin/main")
			advanceRemote(t, codeRemote)
			seen := observe(t)

			result := autofetchRun(t, autofetchOn, code, time.Time{})

			if result.Fetch != branch.AutofetchFetchDone || !pathidentity.SameDirectory(result.Repository, specs) || !pathidentity.SameDirectory(result.Project, master) {
				t.Fatalf("resultado %+v, se esperaba traer el repositorio de specs %s", result, specs)
			}
			if got := runGit(t, specs, "rev-parse", "refs/remotes/origin/main"); got != specsAdvanced {
				t.Errorf("origin/main de specs = %s, se esperaba %s", got, specsAdvanced)
			}
			if got := runGit(t, code, "rev-parse", "refs/remotes/origin/main"); got != codeBefore {
				t.Errorf("el repositorio de código no se trae: origin/main pasó de %s a %s", codeBefore, got)
			}
			for _, invocation := range *seen {
				if pathidentity.SameDirectory(invocation[0], code) && slices.Contains([]string{"fetch", "merge", "push", "pull"}, invocation[1]) {
					t.Errorf("ninguna orden remota ni de avance en el repositorio de código: %v", invocation)
				}
			}
			noRemoteOperations(t, *seen)
		})
	}
}
