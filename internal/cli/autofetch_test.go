package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
)

// Pruebas de `axiom autofetch` (C-05-d; RF-HUB-13, RF-HUB-07, RF-BRA-39, RF-BRA-40, RF-BRA-01)
// en proceso con runAutofetch, el hub inyectado y repositorios reales con remotos locales: no hay
// red. Tocan el observador global de gitcommon y variables de paquete, así que ninguna es paralela.

// autofetchHubPath es la ruta de un hub de prueba que todavía no existe.
func autofetchHubPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "hub", "workspaces.json")
}

// autofetchRepo es un monorepo embebido real con el axiom.yaml confirmado en main.
func autofetchRepo(t *testing.T) string {
	t.Helper()
	repo := newBranchRepository(t)
	writeBranchConfig(t, repo, "")
	branchTestGit(t, repo, "add", "axiom.yaml")
	branchTestGit(t, repo, "commit", "-q", "-m", "configuración")
	return repo
}

// autofetchRegister registra el proyecto en el hub de prueba.
func autofetchRegister(t *testing.T, hubPath, project string) {
	t.Helper()
	manager, err := hub.NewManager(hubPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Register(project, "ejemplo", ""); err != nil {
		t.Fatal(err)
	}
}

// autofetchCLI ejecuta runAutofetch con el hub y la entrada dados.
func autofetchCLI(t *testing.T, hubPath, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = runAutofetch(args, strings.NewReader(stdin), &out, &errOut, hubPath)
	return code, out.String(), errOut.String()
}

// autofetchObserve instala el observador del único punto que ejecuta git.
func autofetchObserve(t *testing.T) *[][]string {
	t.Helper()
	var seen [][]string
	gitcommon.Observe = func(dir string, args []string) {
		seen = append(seen, append([]string{dir}, args...))
	}
	t.Cleanup(func() { gitcommon.Observe = nil })
	return &seen
}

// autofetchPreferenceOf decodifica la salida --json de enable, disable y status.
func autofetchPreferenceOf(t *testing.T, stdout string) autofetchPreference {
	t.Helper()
	var preference autofetchPreference
	if err := json.Unmarshal([]byte(stdout), &preference); err != nil {
		t.Fatalf("la salida no es JSON (%v):\n%s", err, stdout)
	}
	if preference.Schema != AutofetchPreferenceSchema {
		t.Fatalf("esquema %q, se esperaba %q", preference.Schema, AutofetchPreferenceSchema)
	}
	return preference
}

// autofetchResultOf decodifica la salida --json de run.
func autofetchResultOf(t *testing.T, stdout string) branch.AutofetchResult {
	t.Helper()
	var result branch.AutofetchResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("la salida no es JSON (%v):\n%s", err, stdout)
	}
	if result.Schema != branch.AutofetchResultSchema {
		t.Fatalf("esquema %q, se esperaba %q", result.Schema, branch.AutofetchResultSchema)
	}
	return result
}

// autofetchRemote crea un remoto desnudo con main, lo añade como origin y publica main.
func autofetchRemote(t *testing.T, repo string) string {
	t.Helper()
	remote := t.TempDir()
	branchTestGit(t, remote, "init", "-q", "--bare", "-b", "main")
	branchTestGit(t, repo, "remote", "add", "origin", remote)
	branchTestGit(t, repo, "push", "-q", "origin", "main")
	return remote
}

// autofetchAdvanceRemote añade un commit a main del remoto desde un clon aparte.
func autofetchAdvanceRemote(t *testing.T, remote string) {
	t.Helper()
	scratch := t.TempDir()
	clone := filepath.Join(scratch, "clon")
	branchTestGit(t, scratch, "clone", "-q", remote, clone)
	branchTestGit(t, clone, "commit", "-q", "--allow-empty", "-m", "adelanto en el remoto")
	branchTestGit(t, clone, "push", "-q", "origin", "main")
}

// autofetchDirtyProject es un proyecto con el interruptor encendido cuyo origin/main avanzó y cuyo
// checkout de integración tiene el árbol sucio: `run` no puede avanzar y avisa.
func autofetchDirtyProject(t *testing.T) (repo, hubPath string) {
	t.Helper()
	repo = autofetchRepo(t)
	autofetchAdvanceRemote(t, autofetchRemote(t, repo))
	file := filepath.Join(repo, "axiom.yaml")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, append(data, []byte("# cambio local sin confirmar\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	hubPath = autofetchHubPath(t)
	autofetchRegister(t, hubPath, repo)
	if code, _, stderr := autofetchCLI(t, hubPath, "", "enable", "--cwd", repo); code != 0 {
		t.Fatalf("enable terminó con %d: %s", code, stderr)
	}
	return repo, hubPath
}

func autofetchFileState(path string) (data []byte, exists bool) {
	data, err := os.ReadFile(path)
	return data, err == nil
}

// RF-BRA-01: `axiom autofetch` es un verbo de primer nivel y `axiom branch` conserva sus cinco
// verbos, sin `autofetch` en su ayuda.
func TestAutofetchIsATopLevelVerbAndBranchKeepsItsFiveVerbs(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := RunBranch([]string{"--help"}, &out, &errOut); code != 0 {
		t.Fatalf("branch --help terminó con %d", code)
	}
	if strings.Contains(out.String(), "autofetch") {
		t.Errorf("la ayuda de branch no debe mencionar autofetch:\n%s", out.String())
	}
	if code, stdout, _ := autofetchCLI(t, "", "", "--help"); code != 0 {
		t.Fatalf("autofetch --help terminó con %d", code)
	} else {
		for _, verb := range []string{"enable", "disable", "status", "run"} {
			if !strings.Contains(stdout, "\n  "+verb+" ") {
				t.Errorf("la ayuda de autofetch debe listar %q:\n%s", verb, stdout)
			}
		}
	}
	for _, verb := range []string{"enable", "disable", "status", "run"} {
		if code, stdout, _ := autofetchCLI(t, "", "", verb, "--help"); code != 0 || !strings.Contains(stdout, "axiom autofetch "+verb) {
			t.Errorf("%s --help: código %d, salida %q", verb, code, stdout)
		}
	}
	if code, _, _ := autofetchCLI(t, "", ""); code != branch.ExitUsage {
		t.Errorf("sin subcomando se esperaba el código 2, salió %d", code)
	}
	if code, _, stderr := autofetchCLI(t, "", "", "fetch"); code != branch.ExitUsage || !strings.Contains(stderr, "no reconocido") {
		t.Errorf("subcomando desconocido: código %d, stderr %q", code, stderr)
	}
}

// RF-HUB-07 y RF-HUB-13, tabla de D-03: proyecto no registrado y hub ilegible, orden por orden.
func TestAutofetchUnregisteredAndUnreadableHubTable(t *testing.T) {
	for _, hubKind := range []string{"no registrado", "ilegible"} {
		t.Run(hubKind, func(t *testing.T) {
			repo := autofetchRepo(t)
			hubPath := autofetchHubPath(t)
			unreadable := hubKind == "ilegible"
			if unreadable {
				if err := os.MkdirAll(filepath.Dir(hubPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(hubPath, []byte("{esto no es json"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			before, existed := autofetchFileState(hubPath)
			seen := autofetchObserve(t)
			wantWarning := "axiom project add"
			if unreadable {
				wantWarning = "no se pudo leer el hub"
			}

			code, stdout, stderr := autofetchCLI(t, hubPath, "", "status", "--json", "--cwd", repo)
			status := autofetchPreferenceOf(t, stdout)
			if code != 0 || status.Autofetch != "disabled" || status.Registered || len(status.Warnings) != 1 || !strings.Contains(status.Warnings[0], wantWarning) {
				t.Errorf("status: código %d, %+v, stderr %q", code, status, stderr)
			}

			code, stdout, _ = autofetchCLI(t, hubPath, "", "run", "--json", "--cwd", repo)
			run := autofetchResultOf(t, stdout)
			if code != 0 || run.Autofetch != "disabled" || run.Fetch != branch.AutofetchFetchNotAttempted || len(run.Warnings) != 1 || !strings.Contains(run.Warnings[0], wantWarning) {
				t.Errorf("run --json: código %d, %+v", code, run)
			}

			code, stdout, _ = autofetchCLI(t, hubPath, "", "run", "--cwd", repo)
			if code != 0 || !strings.Contains(stdout, "Autofetch: apagado") || !strings.Contains(stdout, wantWarning) {
				t.Errorf("run manual: código %d, salida %q", code, stdout)
			}

			code, stdout, stderr = autofetchCLI(t, hubPath, "", "run", "--agent", "claude-code", "--cwd", repo)
			if code != 0 || stdout != "" || stderr != "" {
				t.Errorf("run --agent: código %d, stdout %q, stderr %q; se esperaba silencio y 0", code, stdout, stderr)
			}

			code, stdout, stderr = autofetchCLI(t, hubPath, "", "enable", "--cwd", repo)
			if code != 1 || stdout != "" || !strings.Contains(stderr, "Error:") {
				t.Errorf("enable: código %d, stdout %q, stderr %q; se esperaba el rechazo con 1", code, stdout, stderr)
			}
			if !unreadable && !strings.Contains(stderr, "axiom project add \"") {
				t.Errorf("enable sin registrar debe indicar cómo registrarlo: %q", stderr)
			}

			code, stdout, stderr = autofetchCLI(t, hubPath, "", "disable", "--json", "--cwd", repo)
			if unreadable {
				if code != 1 || !strings.Contains(stderr, "Error:") {
					t.Errorf("disable con hub ilegible: código %d, stderr %q; se esperaba el rechazo con 1", code, stderr)
				}
			} else {
				disable := autofetchPreferenceOf(t, stdout)
				if code != 0 || disable.Registered || disable.Changed || disable.Autofetch != "disabled" || !strings.Contains(disable.Message, "No hay nada que apagar") {
					t.Errorf("disable sin registrar: código %d, %+v", code, disable)
				}
			}

			after, exists := autofetchFileState(hubPath)
			if exists != existed || !bytes.Equal(before, after) {
				t.Errorf("el hub no debe crearse ni modificarse (existía %v, existe %v)", existed, exists)
			}
			if len(*seen) != 0 {
				t.Errorf("con el interruptor apagado no se lanza git: %v", *seen)
			}
		})
	}
}

// RF-HUB-13: enable, status y disable persisten la preferencia en el hub, son idempotentes y no
// escriben axiom.yaml ni lanzan git.
func TestAutofetchEnableDisableStatusAreIdempotentAndWriteOnlyTheHub(t *testing.T) {
	repo := autofetchRepo(t)
	hubPath := autofetchHubPath(t)
	autofetchRegister(t, hubPath, repo)
	config := filepath.Join(repo, "axiom.yaml")
	configBefore, _ := os.ReadFile(config)
	seen := autofetchObserve(t)

	statusOf := func() autofetchPreference {
		t.Helper()
		code, stdout, stderr := autofetchCLI(t, hubPath, "", "status", "--json", "--cwd", repo)
		if code != 0 {
			t.Fatalf("status terminó con %d: %s", code, stderr)
		}
		return autofetchPreferenceOf(t, stdout)
	}
	if status := statusOf(); status.Autofetch != "disabled" || !status.Registered || len(status.Warnings) != 0 || status.Operation != "autofetch status" {
		t.Fatalf("estado inicial %+v, se esperaba disabled, registrado y sin avisos", status)
	}

	code, stdout, stderr := autofetchCLI(t, hubPath, "", "enable", "--json", "--cwd", repo)
	enable := autofetchPreferenceOf(t, stdout)
	if code != 0 || enable.Autofetch != "enabled" || !enable.Registered || !enable.Changed || enable.Operation != "autofetch enable" {
		t.Fatalf("enable: código %d, %+v, stderr %q", code, enable, stderr)
	}
	hubAfterEnable, _ := autofetchFileState(hubPath)
	if !strings.Contains(string(hubAfterEnable), `"autofetch": true`) {
		t.Errorf("el hub debe contener autofetch: true:\n%s", hubAfterEnable)
	}
	if status := statusOf(); status.Autofetch != "enabled" {
		t.Errorf("tras enable, status = %+v", status)
	}

	code, stdout, _ = autofetchCLI(t, hubPath, "", "enable", "--json", "--cwd", repo)
	again := autofetchPreferenceOf(t, stdout)
	hubAfterRepeat, _ := autofetchFileState(hubPath)
	if code != 0 || again.Changed || again.Autofetch != "enabled" || !bytes.Equal(hubAfterEnable, hubAfterRepeat) {
		t.Errorf("enable repetido: código %d, %+v; el hub debe quedar byte a byte igual", code, again)
	}

	code, stdout, _ = autofetchCLI(t, hubPath, "", "disable", "--json", "--cwd", repo)
	disable := autofetchPreferenceOf(t, stdout)
	hubAfterDisable, _ := autofetchFileState(hubPath)
	if code != 0 || !disable.Changed || disable.Autofetch != "disabled" || !disable.Registered || strings.Contains(string(hubAfterDisable), "autofetch") {
		t.Errorf("disable: código %d, %+v; la entrada debe quedar sin la clave:\n%s", code, disable, hubAfterDisable)
	}
	code, stdout, _ = autofetchCLI(t, hubPath, "", "disable", "--cwd", repo)
	hubAfterRepeat, _ = autofetchFileState(hubPath)
	if code != 0 || !strings.Contains(stdout, "ya estaba apagado") || !bytes.Equal(hubAfterDisable, hubAfterRepeat) {
		t.Errorf("disable repetido: código %d, salida %q; el hub debe quedar byte a byte igual", code, stdout)
	}

	// Texto en castellano y --agent aceptado sin cambiar nada.
	code, stdout, _ = autofetchCLI(t, hubPath, "", "enable", "--agent", "codex", "--cwd", repo)
	if code != 0 || !strings.Contains(stdout, "Autofetch encendido") {
		t.Errorf("enable --agent: código %d, salida %q", code, stdout)
	}

	if configAfter, _ := os.ReadFile(config); !bytes.Equal(configBefore, configAfter) {
		t.Error("enable y disable nunca escriben axiom.yaml")
	}
	if len(*seen) != 0 {
		t.Errorf("enable, disable y status no lanzan git: %v", *seen)
	}
}

// RF-HUB-13 y D-03: la identidad es la carpeta maestra del checkout principal, también desde un
// worktree de feature enlazado, un subdirectorio y, en multirepo, un repositorio de código con
// puntero `.axiom-workspace`.
func TestAutofetchResolvesTheProjectFromWorktreeSubdirectoryAndPointer(t *testing.T) {
	t.Run("worktree de feature y subdirectorio", func(t *testing.T) {
		repo := autofetchRepo(t)
		hubPath := autofetchHubPath(t)
		autofetchRegister(t, hubPath, repo)
		feature := filepath.Join(t.TempDir(), "feature")
		branchTestGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", feature)
		sub := filepath.Join(repo, "docs", "interno")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}

		if code, _, stderr := autofetchCLI(t, hubPath, "", "enable", "--cwd", feature); code != 0 {
			t.Fatalf("enable desde el worktree terminó con %d: %s", code, stderr)
		}
		if enabled, registered, err := mustLookup(t, hubPath, repo); err != nil || !registered || !enabled {
			t.Fatalf("la entrada de la maestra debe quedar encendida: enabled=%v registered=%v err=%v", enabled, registered, err)
		}
		for _, cwd := range []string{repo, feature, sub} {
			code, stdout, stderr := autofetchCLI(t, hubPath, "", "status", "--json", "--cwd", cwd)
			if status := autofetchPreferenceOf(t, stdout); code != 0 || status.Autofetch != "enabled" || !status.Registered {
				t.Errorf("status desde %s: código %d, %+v, stderr %q", cwd, code, status, stderr)
			}
		}
		if code, _, stderr := autofetchCLI(t, hubPath, "", "disable", "--cwd", sub); code != 0 {
			t.Fatalf("disable desde el subdirectorio terminó con %d: %s", code, stderr)
		}
		if enabled, _, _ := mustLookup(t, hubPath, repo); enabled {
			t.Error("disable desde un subdirectorio debe apagar la entrada de la maestra")
		}
	})

	t.Run("multirepo con puntero", func(t *testing.T) {
		isolateBranchGit(t)
		master := t.TempDir()
		for _, dir := range []string{"repo-specs", "repo-core"} {
			if err := os.MkdirAll(filepath.Join(master, dir), 0o755); err != nil {
				t.Fatal(err)
			}
			initBranchRepository(t, filepath.Join(master, dir))
		}
		config := "workspace:\n  name: ejemplo\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  core:\n    name: core\n    repositories:\n      - path: \"repo-core\"\n"
		if err := os.WriteFile(filepath.Join(master, "repo-specs", "axiom.yaml"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(master, ".axiom-workspace"), []byte("config: repo-specs/axiom.yaml\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		hubPath := autofetchHubPath(t)
		manager, err := hub.NewManager(hubPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := manager.Register(master, "ejemplo", "multirepo"); err != nil {
			t.Fatal(err)
		}
		code := filepath.Join(master, "repo-core")

		if exit, _, stderr := autofetchCLI(t, hubPath, "", "enable", "--cwd", code); exit != 0 {
			t.Fatalf("enable desde el repositorio de código terminó con %d: %s", exit, stderr)
		}
		if enabled, registered, err := mustLookup(t, hubPath, master); err != nil || !registered || !enabled {
			t.Fatalf("la entrada de la maestra debe quedar encendida: enabled=%v registered=%v err=%v", enabled, registered, err)
		}
		exit, stdout, stderr := autofetchCLI(t, hubPath, "", "status", "--json", "--cwd", code)
		if status := autofetchPreferenceOf(t, stdout); exit != 0 || status.Autofetch != "enabled" {
			t.Errorf("status desde el repositorio de código: código %d, %+v, stderr %q", exit, status, stderr)
		}
	})
}

func mustLookup(t *testing.T, hubPath, project string) (enabled, registered bool, err error) {
	t.Helper()
	manager, err := hub.NewManager(hubPath)
	if err != nil {
		t.Fatal(err)
	}
	return manager.LookupAutofetch(project)
}

// enable y disable fuera de un proyecto Axiom se rechazan con 1 y sin escribir el hub.
func TestAutofetchPreferenceOutsideAnAxiomProjectIsRejected(t *testing.T) {
	outside := t.TempDir()
	hubPath := autofetchHubPath(t)
	seen := autofetchObserve(t)
	for _, verb := range []string{"enable", "disable", "status"} {
		code, stdout, stderr := autofetchCLI(t, hubPath, "", verb, "--cwd", outside)
		if code != 1 || stdout != "" || !strings.Contains(stderr, "Error:") {
			t.Errorf("%s fuera de un proyecto: código %d, stdout %q, stderr %q", verb, code, stdout, stderr)
		}
	}
	if _, exists := autofetchFileState(hubPath); exists || len(*seen) != 0 {
		t.Errorf("no se crea el hub ni se lanza git (hub=%v, git=%v)", exists, *seen)
	}
}

// RF-BRA-39: `--cwd` relativa es un error de uso (2) en CLI; el modo hook termina con 0.
func TestAutofetchRelativeCwdIsAUsageError(t *testing.T) {
	hubPath := autofetchHubPath(t)
	for _, verb := range []string{"run", "enable", "disable", "status"} {
		code, stdout, stderr := autofetchCLI(t, hubPath, "", verb, "--cwd", "relativa")
		if code != branch.ExitUsage || stdout != "" || !strings.Contains(stderr, "ruta absoluta") {
			t.Errorf("%s con --cwd relativa: código %d, stdout %q, stderr %q", verb, code, stdout, stderr)
		}
	}
	if code, stdout, stderr := autofetchCLI(t, hubPath, "", "run", "--bandera-inexistente"); code != branch.ExitUsage || stdout != "" || stderr == "" {
		t.Errorf("bandera desconocida: código %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	if code, _, _ := autofetchCLI(t, hubPath, "", "status", "sobra"); code != branch.ExitUsage {
		t.Errorf("argumento sobrante: código %d, se esperaba 2", code)
	}
	code, stdout, stderr := autofetchCLI(t, hubPath, "", "run", "--agent", "codex", "--cwd", "relativa")
	if code != 0 || stdout != "" || stderr != "" {
		t.Errorf("modo hook con --cwd relativa: código %d, stdout %q, stderr %q; se esperaba silencio y 0", code, stdout, stderr)
	}
}

// RF-BRA-39: `run` a mano con el interruptor encendido, en JSON y en texto, sin avisos cuando todo
// está al día.
func TestAutofetchRunOutputInJSONAndText(t *testing.T) {
	repo := autofetchRepo(t)
	autofetchRemote(t, repo)
	hubPath := autofetchHubPath(t)
	autofetchRegister(t, hubPath, repo)
	if code, _, stderr := autofetchCLI(t, hubPath, "", "enable", "--cwd", repo); code != 0 {
		t.Fatalf("enable terminó con %d: %s", code, stderr)
	}

	code, stdout, stderr := autofetchCLI(t, hubPath, "", "run", "--json", "--cwd", repo)
	result := autofetchResultOf(t, stdout)
	if code != 0 || result.Autofetch != "enabled" || result.Fetch != branch.AutofetchFetchDone || result.Notices == nil || len(result.Notices) != 0 || result.Integration == nil || result.Integration.State != "up_to_date" {
		t.Fatalf("run --json: código %d, %+v, stderr %q", code, result, stderr)
	}
	if !strings.Contains(stdout, `"notices": []`) {
		t.Errorf("los avisos vacíos van como []:\n%s", stdout)
	}

	code, stdout, _ = autofetchCLI(t, hubPath, "", "run", "--cwd", repo)
	for _, want := range []string{"Autofetch: encendido", "Fetch: omitido (dentro del intervalo mínimo)", "Integración main: up_to_date"} {
		if code != 0 || !strings.Contains(stdout, want) {
			t.Errorf("run en texto: código %d; falta %q en:\n%s", code, want, stdout)
		}
	}
}

// RF-BRA-40: el hook, sin avisos, no escribe nada y termina con 0.
func TestAutofetchHookWithoutNoticesIsSilent(t *testing.T) {
	repo := autofetchRepo(t)
	autofetchRemote(t, repo)
	hubPath := autofetchHubPath(t)
	autofetchRegister(t, hubPath, repo)
	if code, _, stderr := autofetchCLI(t, hubPath, "", "enable", "--cwd", repo); code != 0 {
		t.Fatalf("enable terminó con %d: %s", code, stderr)
	}
	for _, agent := range []string{"claude-code", "codex"} {
		code, stdout, stderr := autofetchCLI(t, hubPath, "", "run", "--agent", agent, "--cwd", repo)
		if code != 0 || stdout != "" || stderr != "" {
			t.Errorf("hook %s sin avisos: código %d, stdout %q, stderr %q", agent, code, stdout, stderr)
		}
	}
}

// RF-BRA-40: con avisos, el hook escribe el JSON `SessionStart` con el contexto adicional; el
// directorio sale de --cwd o del campo cwd de la carga de la entrada estándar.
func TestAutofetchHookWithNoticesEmitsSessionStartContext(t *testing.T) {
	check := func(name string, stdin string, args ...string) {
		t.Helper()
		freshRepo, freshHub := autofetchDirtyProject(t)
		replaced := make([]string, len(args))
		for i, arg := range args {
			replaced[i] = strings.ReplaceAll(arg, "<repo>", freshRepo)
		}
		code, out, errOut := autofetchCLI(t, freshHub, strings.ReplaceAll(stdin, "<repo>", strings.ReplaceAll(freshRepo, `\`, `\\`)), replaced...)
		if code != 0 || errOut != "" {
			t.Fatalf("%s: código %d, stderr %q", name, code, errOut)
		}
		var payload struct {
			HookSpecificOutput struct {
				HookEventName     string `json:"hookEventName"`
				AdditionalContext string `json:"additionalContext"`
			} `json:"hookSpecificOutput"`
		}
		if err := json.Unmarshal([]byte(out), &payload); err != nil {
			t.Fatalf("%s: la salida no es JSON (%v):\n%s", name, err, out)
		}
		hook := payload.HookSpecificOutput
		if hook.HookEventName != "SessionStart" || !strings.HasPrefix(hook.AdditionalContext, "Axiom autofetch:\n- ") || !strings.Contains(hook.AdditionalContext, "Orden: ") {
			t.Errorf("%s: contexto adicional inesperado: %+v", name, hook)
		}
	}
	check("--cwd", "", "run", "--agent", "claude-code", "--cwd", "<repo>")
	check("cwd por la entrada estándar", `{"session_id":"s","cwd":"<repo>","hook_event_name":"SessionStart"}`, "run", "--agent", "codex")
}

// RF-BRA-40: con el interruptor apagado, fuera de un proyecto Axiom o con una carga ilegible, el
// hook no escribe nada, termina con 0 y no lanza git por esas causas.
func TestAutofetchHookIsInertAndFailsOpen(t *testing.T) {
	t.Run("interruptor apagado y fuera de proyecto", func(t *testing.T) {
		repo := autofetchRepo(t)
		autofetchRemote(t, repo)
		hubPath := autofetchHubPath(t)
		autofetchRegister(t, hubPath, repo)
		seen := autofetchObserve(t)
		for name, cwd := range map[string]string{"apagado": repo, "fuera de proyecto": t.TempDir()} {
			code, stdout, stderr := autofetchCLI(t, hubPath, "", "run", "--agent", "claude-code", "--cwd", cwd)
			if code != 0 || stdout != "" || stderr != "" {
				t.Errorf("%s: código %d, stdout %q, stderr %q", name, code, stdout, stderr)
			}
		}
		if len(*seen) != 0 {
			t.Errorf("el hook inerte no lanza git: %v", *seen)
		}
	})

	t.Run("cargas ilegibles, demasiado grandes y banderas inválidas", func(t *testing.T) {
		repo, hubPath := autofetchDirtyProject(t)
		oversized := `{"cwd":` + `"` + strings.ReplaceAll(repo, `\`, `\\`) + `","relleno":"` + strings.Repeat("x", autofetchStdinLimit) + `"}`
		cases := map[string]struct {
			stdin string
			args  []string
		}{
			"carga ilegible":           {"esto no es json", []string{"run", "--agent", "claude-code"}},
			"carga vacía":              {"", []string{"run", "--agent", "codex"}},
			"carga de más de 256 KiB":  {oversized, []string{"run", "--agent", "claude-code"}},
			"cwd relativo en la carga": {`{"cwd":"relativo"}`, []string{"run", "--agent", "claude-code"}},
			"runtime desconocido":      {"", []string{"run", "--agent", "otro", "--cwd", repo}},
			"bandera desconocida":      {"", []string{"run", "--agent", "codex", "--sorpresa"}},
			"argumento sobrante":       {"", []string{"run", "--agent", "codex", "sobra"}},
		}
		for name, tc := range cases {
			code, stdout, stderr := autofetchCLI(t, hubPath, tc.stdin, tc.args...)
			if code != 0 || stdout != "" || stderr != "" {
				t.Errorf("%s: código %d, stdout %q, stderr %q; se esperaba silencio y 0", name, code, stdout, stderr)
			}
		}
	})
}

// RF-BRA-40: el hook espera como mucho su presupuesto, también con un remoto que no responde:
// termina con 0 dentro de la cota y avisa del tiempo agotado.
func TestAutofetchHookRespectsTheBudgetWithASlowRemote(t *testing.T) {
	repo := autofetchRepo(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("sin socket local: %v", err)
	}
	var held []net.Conn
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			held = append(held, conn)
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		<-done
		for _, conn := range held {
			_ = conn.Close()
		}
	})
	branchTestGit(t, repo, "remote", "add", "origin", "http://"+listener.Addr().String()+"/lento.git")
	hubPath := autofetchHubPath(t)
	autofetchRegister(t, hubPath, repo)
	if code, _, stderr := autofetchCLI(t, hubPath, "", "enable", "--cwd", repo); code != 0 {
		t.Fatalf("enable terminó con %d: %s", code, stderr)
	}
	previousContext := newAutofetchContext
	newAutofetchContext = func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 600*time.Millisecond)
	}
	t.Cleanup(func() { newAutofetchContext = previousContext })

	started := time.Now()
	code, stdout, stderr := autofetchCLI(t, hubPath, "", "run", "--agent", "claude-code", "--cwd", repo)
	elapsed := time.Since(started)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; el hook termina siempre con 0", code, stderr)
	}
	if elapsed > 5*time.Second {
		t.Errorf("el hook tardó %v con un presupuesto de 600 ms", elapsed)
	}
	if !strings.Contains(stdout, "SessionStart") || !strings.Contains(stdout, "agotó el tiempo") {
		t.Errorf("se esperaba el aviso de tiempo agotado en el contexto del hook:\n%s", stdout)
	}
}

// RF-BRA-39: `run` en modo CLI con el mismo remoto lento también termina con 0 y devuelve el
// resultado con el fetch en `timeout`.
func TestAutofetchRunWithAnUnreachableRemoteReportsAndExitsZero(t *testing.T) {
	repo := autofetchRepo(t)
	branchTestGit(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "no-existe.git"))
	hubPath := autofetchHubPath(t)
	autofetchRegister(t, hubPath, repo)
	if code, _, stderr := autofetchCLI(t, hubPath, "", "enable", "--cwd", repo); code != 0 {
		t.Fatalf("enable terminó con %d: %s", code, stderr)
	}

	code, stdout, _ := autofetchCLI(t, hubPath, "", "run", "--json", "--cwd", repo)
	result := autofetchResultOf(t, stdout)
	kinds := make([]string, 0, len(result.Notices))
	for _, notice := range result.Notices {
		kinds = append(kinds, notice.Kind)
	}
	if code != 0 || result.Fetch != branch.AutofetchFetchFailed || !slices.Contains(kinds, branch.AutofetchNoticeFetchFailed) {
		t.Errorf("código %d, fetch %q, avisos %v; se esperaba fetch fallido con aviso y salida 0", code, result.Fetch, kinds)
	}
}
