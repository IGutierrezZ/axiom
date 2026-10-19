package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// isolateBranchGit deja a git sin acceso a la configuración del usuario, con un
// autor fijo y sin preguntas ni red (tasks.md, regla 9). Omite la prueba con -short
// o sin git: ejecuta procesos externos (skill go-testing).
func isolateBranchGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("la prueba ejecuta git: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está disponible: %v", err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Autor de Prueba")
	t.Setenv("GIT_AUTHOR_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Autor de Prueba")
	t.Setenv("GIT_COMMITTER_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
}

func branchTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
}

// initBranchRepository convierte dir en un repositorio real, en la rama main y con
// un commit. El entorno de git ya debe estar aislado (isolateBranchGit).
func initBranchRepository(t *testing.T, dir string) {
	t.Helper()
	branchTestGit(t, dir, "init", "-q", "-b", "main")
	branchTestGit(t, dir, "commit", "-q", "--allow-empty", "-m", "inicial")
}

// newBranchRepository crea un repositorio real, en la rama main y con un commit.
func newBranchRepository(t *testing.T) string {
	t.Helper()
	isolateBranchGit(t)
	dir := t.TempDir()
	initBranchRepository(t, dir)
	return dir
}

// branchConfigBase es un axiom.yaml mínimo y válido; los bloques de política de ramas
// se le añaden detrás.
const branchConfigBase = `workspace:
  name: ejemplo
  topology: monorepo-embedded
  specs_repository: .
roles:
  core:
    name: Core
    repositories:
      - path: .
`

func writeBranchConfig(t *testing.T, dir, blocks string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "axiom.yaml"), []byte(branchConfigBase+blocks), 0o644); err != nil {
		t.Fatal(err)
	}
}

// branchOutsideGit devuelve un directorio que no pertenece a ningún repositorio git,
// u omite la prueba si el temporal de la máquina está dentro de uno.
func branchOutsideGit(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for current := dir; ; {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			t.Skipf("el directorio temporal %s está dentro de un repositorio git", dir)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return dir
		}
		current = parent
	}
}

func runBranchCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = RunBranch(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// branchRepositoryJSON y branchResultJSON son lo que las pruebas leen del
// `axiom.branch-result/v1` de `status`.
type branchRepositoryJSON struct {
	GitRoot           string `json:"gitRoot"`
	CommonDir         string `json:"commonDir"`
	IntegrationBranch string `json:"integrationBranch"`
	CurrentBranch     string `json:"currentBranch"`
	Detached          bool   `json:"detached"`
}

type branchResultJSON struct {
	Schema       string                 `json:"schema"`
	Operation    string                 `json:"operation"`
	Status       string                 `json:"status"`
	StatusLabel  string                 `json:"statusLabel"`
	Code         string                 `json:"code"`
	Message      string                 `json:"message"`
	Continuation string                 `json:"continuation"`
	Details      map[string]string      `json:"details"`
	Repositories []branchRepositoryJSON `json:"repositories"`
	Warnings     []string               `json:"warnings"`
}

func decodeBranchResult(t *testing.T, stdout string) branchResultJSON {
	t.Helper()
	var result branchResultJSON
	decoder := json.NewDecoder(strings.NewReader(stdout))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("la salida no es un axiom.branch-result/v1 reconocible: %v\n%s", err, stdout)
	}
	return result
}

// onlyBranchRepository devuelve el único repositorio de un resultado de `status`.
func onlyBranchRepository(t *testing.T, result branchResultJSON) branchRepositoryJSON {
	t.Helper()
	if len(result.Repositories) != 1 {
		t.Fatalf("se esperaba un único repositorio, hay %d: %+v", len(result.Repositories), result)
	}
	return result.Repositories[0]
}

func wantBranchDirectory(t *testing.T, what, got, want string) {
	t.Helper()
	if !filepath.IsAbs(got) {
		t.Errorf("%s = %q, se esperaba una ruta absoluta", what, got)
		return
	}
	if !pathidentity.SameDirectory(got, want) {
		t.Errorf("%s = %q, se esperaba el directorio %q", what, got, want)
	}
}

var branchHelpVerb = regexp.MustCompile(`(?m)^  ([a-z]+)\s`)

func helpVerbs(help string) []string {
	var verbs []string
	for _, match := range branchHelpVerb.FindAllStringSubmatch(help, -1) {
		verbs = append(verbs, match[1])
	}
	return verbs
}

// REQ-25.1 y decisión T3: la ayuda lista solo los verbos implementados (por ahora
// `status`); ni `push`, ni `pr`, ni el retirado `check`.
func TestBranchWithoutSubcommandShowsTheHelpOfTheImplementedVerbs(t *testing.T) {
	t.Run("sin subcomando es un error de uso con la ayuda", func(t *testing.T) {
		code, stdout, _ := runBranchCLI(t)
		if code != 2 {
			t.Errorf("código de salida = %d, se esperaba 2 (uso)", code)
		}
		if verbs := helpVerbs(stdout); len(verbs) != 1 || verbs[0] != "status" {
			t.Errorf("la ayuda lista %v, se esperaba solo [status]:\n%s", verbs, stdout)
		}
	})

	for _, flagName := range []string{"--help", "-h", "help"} {
		t.Run(flagName+" muestra la ayuda y termina con 0", func(t *testing.T) {
			code, stdout, stderr := runBranchCLI(t, flagName)
			if code != 0 || stderr != "" {
				t.Errorf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
			}
			if verbs := helpVerbs(stdout); len(verbs) != 1 || verbs[0] != "status" {
				t.Errorf("la ayuda lista %v, se esperaba solo [status]:\n%s", verbs, stdout)
			}
			for _, retired := range []string{"push", "pr", "check"} {
				if strings.Contains(" "+strings.Join(strings.Fields(stdout), " ")+" ", " "+retired+" ") {
					t.Errorf("la ayuda no puede ofrecer %q:\n%s", retired, stdout)
				}
			}
		})
	}
}

// REQ-25.1, escenario «Un subcomando no listado no existe».
func TestBranchRejectsUnrecognisedSubcommandsWithCode2(t *testing.T) {
	for _, subcommand := range []string{"push", "pr", "check", "inexistente"} {
		t.Run(subcommand, func(t *testing.T) {
			code, stdout, stderr := runBranchCLI(t, subcommand)
			if code != 2 {
				t.Errorf("código de salida = %d, se esperaba 2", code)
			}
			if stdout != "" {
				t.Errorf("un uso erróneo no escribe en stdout, escribió %q", stdout)
			}
			for _, want := range []string{"'" + subcommand + "'", "no reconocido", "axiom branch --help"} {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr %q debe contener %q (nombra la continuación)", stderr, want)
				}
			}
		})
	}
}

// REQ-25.2: `status --json --cwd <ruta>` imprime un `axiom.branch-result/v1` con, por
// repositorio, la raíz git, el git common dir, la rama actual (por ReadHead) y la rama
// de integración de la política.
func TestBranchStatusPrintsTheResultAsJSON(t *testing.T) {
	repo := newBranchRepository(t)

	code, stdout, stderr := runBranchCLI(t, "status", "--json", "--cwd", repo)
	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	result := decodeBranchResult(t, stdout)
	if result.Schema != "axiom.branch-result/v1" || result.Operation != "status" {
		t.Errorf("schema %q, operation %q; se esperaba axiom.branch-result/v1 y status", result.Schema, result.Operation)
	}
	if result.Status != "ready" || result.StatusLabel != "listo" {
		t.Errorf("status %q / statusLabel %q; se esperaba el token ready con la etiqueta listo", result.Status, result.StatusLabel)
	}
	if result.Code != "" || result.Message != "" || result.Continuation != "" {
		t.Errorf("un resultado listo no lleva código, mensaje ni continuación: %+v", result)
	}
	repository := onlyBranchRepository(t, result)
	wantBranchDirectory(t, "gitRoot", repository.GitRoot, repo)
	wantBranchDirectory(t, "commonDir", repository.CommonDir, filepath.Join(repo, ".git"))
	if repository.CurrentBranch != "main" || repository.Detached {
		t.Errorf("currentBranch %q, detached %v; se esperaba main y false", repository.CurrentBranch, repository.Detached)
	}
}

// La salida es estable: dos ejecuciones sobre el mismo estado dan los mismos bytes.
func TestBranchStatusOutputIsDeterministic(t *testing.T) {
	repo := newBranchRepository(t)
	_, first, _ := runBranchCLI(t, "status", "--json", "--cwd", repo)
	_, second, _ := runBranchCLI(t, "status", "--json", "--cwd", repo)
	if first != second || first == "" {
		t.Errorf("la salida cambió entre ejecuciones:\n%s\n---\n%s", first, second)
	}
	if !strings.HasSuffix(first, "\n") {
		t.Errorf("la salida JSON debe terminar en un salto de línea: %q", first)
	}
}

// Desde un worktree enlazado, la raíz es la del worktree y el common dir el del
// checkout principal; la rama es la del worktree.
func TestBranchStatusInALinkedWorktree(t *testing.T) {
	main := newBranchRepository(t)
	linked := filepath.Join(t.TempDir(), "enlazado")
	branchTestGit(t, main, "worktree", "add", "-q", "-b", "inc/mi-cambio", linked, "HEAD")

	code, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", linked)
	if code != 0 {
		t.Fatalf("código de salida = %d, se esperaba 0\n%s", code, stdout)
	}
	result := decodeBranchResult(t, stdout)
	repository := onlyBranchRepository(t, result)
	wantBranchDirectory(t, "gitRoot", repository.GitRoot, linked)
	wantBranchDirectory(t, "commonDir", repository.CommonDir, filepath.Join(main, ".git"))
	if repository.CurrentBranch != "inc/mi-cambio" {
		t.Errorf("currentBranch = %q, se esperaba inc/mi-cambio", repository.CurrentBranch)
	}
}

func TestBranchStatusWithADetachedHead(t *testing.T) {
	repo := newBranchRepository(t)
	branchTestGit(t, repo, "checkout", "-q", "--detach")

	code, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", repo)
	if code != 0 {
		t.Fatalf("código de salida = %d, se esperaba 0\n%s", code, stdout)
	}
	result := decodeBranchResult(t, stdout)
	repository := onlyBranchRepository(t, result)
	if repository.CurrentBranch != "" || !repository.Detached {
		t.Errorf("currentBranch %q, detached %v; se esperaba una rama vacía y HEAD desacoplado", repository.CurrentBranch, repository.Detached)
	}
}

// Un HEAD que no se puede leer sin git (reftable) no es un fallo: el resultado sigue
// listo, sin rama y con un aviso.
func TestBranchStatusWarnsWhenHeadHasAnUnknownForm(t *testing.T) {
	repo := newBranchRepository(t)
	if err := os.WriteFile(filepath.Join(repo, ".git", "HEAD"), []byte("ref: refs/heads/.invalid\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", repo)
	result := decodeBranchResult(t, stdout)
	if code != 0 || result.Status != "ready" {
		t.Fatalf("código %d, status %q; se esperaba 0 y ready", code, result.Status)
	}
	repository := onlyBranchRepository(t, result)
	if repository.CurrentBranch != "" || repository.Detached || len(result.Warnings) != 1 {
		t.Errorf("currentBranch %q, detached %v, avisos %v; se esperaba sin rama y un aviso", repository.CurrentBranch, repository.Detached, result.Warnings)
	}
}

// REQ-25.2, escenario «`--cwd` fija el directorio de partida»: el resultado es el
// del `--cwd`, no el del directorio de trabajo del proceso; sin `--cwd` rige este.
func TestBranchStatusCwdFixesTheStartingDirectory(t *testing.T) {
	repo := newBranchRepository(t)
	other := newBranchRepository(t)
	sub := filepath.Join(repo, "app", "web")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("con --cwd manda el argumento", func(t *testing.T) {
		t.Chdir(other)
		_, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", repo)
		wantBranchDirectory(t, "gitRoot", onlyBranchRepository(t, decodeBranchResult(t, stdout)).GitRoot, repo)
	})
	t.Run("sin --cwd rige el directorio del proceso", func(t *testing.T) {
		t.Chdir(sub)
		_, stdout, _ := runBranchCLI(t, "status", "--json")
		wantBranchDirectory(t, "gitRoot", onlyBranchRepository(t, decodeBranchResult(t, stdout)).GitRoot, repo)
	})
	t.Run("un --cwd relativo se resuelve contra el proceso y la salida es absoluta", func(t *testing.T) {
		t.Chdir(repo)
		_, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", filepath.Join("app", "web"))
		wantBranchDirectory(t, "gitRoot", onlyBranchRepository(t, decodeBranchResult(t, stdout)).GitRoot, repo)
	})
	t.Run("una subcarpeta resuelve a la raíz del repositorio", func(t *testing.T) {
		_, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", sub)
		wantBranchDirectory(t, "gitRoot", onlyBranchRepository(t, decodeBranchResult(t, stdout)).GitRoot, repo)
	})
}

// REQ-25.2: fuera de git, `not_a_git_repository` con código 1, token con etiqueta y
// una continuación en castellano con la ruta citada.
func TestBranchStatusOutsideGitIsRefusedWithATypedCode(t *testing.T) {
	isolateBranchGit(t)
	outside := branchOutsideGit(t)

	code, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", outside)
	if code != 1 {
		t.Errorf("código de salida = %d, se esperaba 1", code)
	}
	result := decodeBranchResult(t, stdout)
	if result.Status != "refused" || result.StatusLabel != "rechazado" || result.Code != "not_a_git_repository" {
		t.Errorf("status %q / %q, code %q; se esperaba refused / rechazado / not_a_git_repository", result.Status, result.StatusLabel, result.Code)
	}
	if len(result.Repositories) != 0 {
		t.Errorf("un rechazo no lleva datos del repositorio: %+v", result)
	}
	if !strings.Contains(result.Message, outside) {
		t.Errorf("el mensaje %q debe nombrar el directorio %s", result.Message, outside)
	}
	for _, want := range []string{"axiom branch status --cwd", pathquote.Quote(outside)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
}

func TestBranchStatusOfAMissingDirectoryIsRefusedWithTheSameCode(t *testing.T) {
	isolateBranchGit(t)
	missing := filepath.Join(branchOutsideGit(t), "no", "existe")

	code, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", missing)
	result := decodeBranchResult(t, stdout)
	if code != 1 || result.Code != "not_a_git_repository" || !strings.Contains(result.Message, "no existe") {
		t.Errorf("código %d, resultado %+v; se esperaba 1, not_a_git_repository y un mensaje que diga que no existe", code, result)
	}
}

// Sin --json la salida es texto en castellano, con los mismos códigos de salida.
func TestBranchStatusPrintsSpanishTextWithoutJSON(t *testing.T) {
	repo := newBranchRepository(t)

	code, stdout, stderr := runBranchCLI(t, "status", "--cwd", repo)
	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{"Repositorio git:", "Directorio común de git:", "Rama actual: main"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la salida de texto debe contener %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, `"schema"`) {
		t.Errorf("sin --json la salida no es JSON:\n%s", stdout)
	}

	outside := branchOutsideGit(t)
	code, stdout, stderr = runBranchCLI(t, "status", "--cwd", outside)
	if code != 1 || stdout != "" {
		t.Errorf("fuera de git: código %d, stdout %q; se esperaba 1 y stdout vacío", code, stdout)
	}
	for _, want := range []string{"not_a_git_repository", "axiom branch status --cwd"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("el rechazo en texto debe contener %q:\n%s", want, stderr)
		}
	}
}

// Los errores de uso de `status` terminan con 2 y nombran la continuación.
func TestBranchStatusUsageErrorsExitWithCode2(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"bandera desconocida", []string{"status", "--no-existe"}},
		{"argumento posicional", []string{"status", "sobra"}},
		{"--cwd sin valor", []string{"status", "--cwd"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runBranchCLI(t, tc.args...)
			if code != 2 || stdout != "" {
				t.Errorf("código %d, stdout %q; se esperaba 2 y stdout vacío", code, stdout)
			}
			if !strings.Contains(stderr, "axiom branch status --help") {
				t.Errorf("stderr %q debe nombrar `axiom branch status --help`", stderr)
			}
		})
	}

	t.Run("status --help describe las banderas", func(t *testing.T) {
		code, stdout, _ := runBranchCLI(t, "status", "--help")
		if code != 0 {
			t.Errorf("código %d, se esperaba 0", code)
		}
		for _, want := range []string{"--cwd", "--json"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("la ayuda de status debe documentar %s:\n%s", want, stdout)
			}
		}
	})
}

// branchRemoteOperation dice si una invocación de git es de las que `status` no
// puede lanzar (REQ-25.1): fetch, pull, push o ls-remote.
func branchRemoteOperation(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "-c" || arg == "-C":
			i++
		case strings.HasPrefix(arg, "-"):
		case arg == "fetch" || arg == "pull" || arg == "push" || arg == "ls-remote":
			return arg, true
		default:
			return arg, false
		}
	}
	return "", false
}

// REQ-25.1, escenario «Ningún subcomando publica ni abre PR»: con un remoto
// configurado y un observador en el único punto que ejecuta git, `status` no lanza
// ninguna operación remota.
func TestBranchStatusLaunchesNoRemoteGitOperation(t *testing.T) {
	repo := newBranchRepository(t)
	branchTestGit(t, repo, "remote", "add", "origin", "https://ejemplo.invalid/repo.git")

	var seen [][]string
	gitcommon.Observe = func(dir string, args []string) {
		seen = append(seen, append([]string{dir}, args...))
	}
	t.Cleanup(func() { gitcommon.Observe = nil })

	for _, args := range [][]string{
		{"status", "--json", "--cwd", repo},
		{"status", "--cwd", repo},
	} {
		if code, _, _ := runBranchCLI(t, args...); code != 0 {
			t.Fatalf("status %v terminó con el código %d", args, code)
		}
	}

	if len(seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación: la prueba sería vacua")
	}
	for _, invocation := range seen {
		if subcommand, remote := branchRemoteOperation(invocation[1:]); remote {
			t.Errorf("status lanzó la operación remota git %s: %v", subcommand, invocation)
		}
		// El HEAD se lee de ficheros (REQ-25.21): ninguna invocación lo consulta.
		if len(invocation) > 1 && invocation[1] != "rev-parse" {
			t.Errorf("status solo consulta rutas con rev-parse, lanzó %v", invocation)
		}
	}
}

// REQ-25.2 (G-4): cada token del JSON lleva su etiqueta en castellano en un campo
// hermano, tanto en un resultado listo como en un rechazo.
func TestBranchStatusPairsEveryTokenWithItsLabel(t *testing.T) {
	repo := newBranchRepository(t)
	_, ready, _ := runBranchCLI(t, "status", "--json", "--cwd", repo)
	_, refused, _ := runBranchCLI(t, "status", "--json", "--cwd", branchOutsideGit(t))

	for name, tc := range map[string]struct {
		stdout, status, label string
	}{
		"listo":     {ready, "ready", "listo"},
		"rechazado": {refused, "refused", "rechazado"},
	} {
		t.Run(name, func(t *testing.T) {
			var generic map[string]any
			if err := json.Unmarshal([]byte(tc.stdout), &generic); err != nil {
				t.Fatalf("la salida no es JSON: %v\n%s", err, tc.stdout)
			}
			if generic["status"] != tc.status || generic["statusLabel"] != tc.label {
				t.Errorf("status %v / statusLabel %v; se esperaba %s / %s", generic["status"], generic["statusLabel"], tc.status, tc.label)
			}
		})
	}
}

// REQ-25.3: sin configuración rige `main`; el bloque `git:` lo sustituye; una entrada
// de `repositories:` prevalece sobre `git:` solo en su repositorio.
func TestBranchStatusShowsTheIntegrationBranchPerRepository(t *testing.T) {
	statusOf := func(t *testing.T, cwd string) branchRepositoryJSON {
		t.Helper()
		code, stdout, stderr := runBranchCLI(t, "status", "--json", "--cwd", cwd)
		if code != 0 || stderr != "" {
			t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
		}
		return onlyBranchRepository(t, decodeBranchResult(t, stdout))
	}

	t.Run("sin axiom.yaml rige main", func(t *testing.T) {
		repo := newBranchRepository(t)
		if got := statusOf(t, repo).IntegrationBranch; got != "main" {
			t.Errorf("integrationBranch = %q, se esperaba main", got)
		}
	})
	t.Run("axiom.yaml sin bloques rige main", func(t *testing.T) {
		repo := newBranchRepository(t)
		writeBranchConfig(t, repo, "")
		if got := statusOf(t, repo).IntegrationBranch; got != "main" {
			t.Errorf("integrationBranch = %q, se esperaba main", got)
		}
	})
	t.Run("el bloque git sustituye el valor por defecto", func(t *testing.T) {
		repo := newBranchRepository(t)
		writeBranchConfig(t, repo, "git:\n  integration_branch: develop\n")
		if got := statusOf(t, repo).IntegrationBranch; got != "develop" {
			t.Errorf("integrationBranch = %q, se esperaba develop", got)
		}
	})
	t.Run("una entrada de repositories para el repositorio de la carpeta maestra prevalece", func(t *testing.T) {
		repo := newBranchRepository(t)
		writeBranchConfig(t, repo, "git:\n  integration_branch: develop\nrepositories:\n  - path: .\n    integration_branch: release\n")
		if got := statusOf(t, repo).IntegrationBranch; got != "release" {
			t.Errorf("integrationBranch = %q, se esperaba release", got)
		}
	})
	t.Run("cada repositorio de la carpeta maestra toma su propia política", func(t *testing.T) {
		isolateBranchGit(t)
		master := t.TempDir()
		writeBranchConfig(t, master, "git:\n  integration_branch: main\nrepositories:\n  - path: repo-backend\n    integration_branch: release\n")
		for _, name := range []string{"repo-backend", "repo-frontend"} {
			dir := filepath.Join(master, name)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			initBranchRepository(t, dir)
		}
		if got := statusOf(t, filepath.Join(master, "repo-backend")).IntegrationBranch; got != "release" {
			t.Errorf("repo-backend: integrationBranch = %q, se esperaba release", got)
		}
		if got := statusOf(t, filepath.Join(master, "repo-frontend")).IntegrationBranch; got != "main" {
			t.Errorf("repo-frontend: integrationBranch = %q, se esperaba main", got)
		}
	})
	t.Run("un repositorio fuera de la carpeta maestra solo hereda la política del workspace", func(t *testing.T) {
		repo := newBranchRepository(t)
		master := filepath.Join(repo, "app")
		if err := os.MkdirAll(master, 0o755); err != nil {
			t.Fatal(err)
		}
		writeBranchConfig(t, master, "git:\n  integration_branch: develop\nrepositories:\n  - path: .\n    integration_branch: release\n")
		if got := statusOf(t, master).IntegrationBranch; got != "develop" {
			t.Errorf("integrationBranch = %q; la entrada «.» es de la carpeta maestra, no del repositorio que la contiene (se esperaba develop)", got)
		}
	})
	t.Run("sin --json la rama de integración sale en el texto", func(t *testing.T) {
		repo := newBranchRepository(t)
		writeBranchConfig(t, repo, "git:\n  integration_branch: develop\n")
		_, stdout, _ := runBranchCLI(t, "status", "--cwd", repo)
		if !strings.Contains(stdout, "Rama de integración: develop") {
			t.Errorf("el texto debe mostrar la rama de integración:\n%s", stdout)
		}
	})
}

// REQ-25.3: un valor inválido o una configuración ilegible en axiom.yaml es
// `config_invalid`, con código de salida 1, la causa en el mensaje y la continuación
// ejecutable con la carpeta maestra citada.
func TestBranchStatusRefusesAnInvalidConfigurationAsConfigInvalid(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCause string
	}{
		{"prefijo inválido", branchConfigBase + "git:\n  branch_prefix: \"mi prefijo//\"\n", "branch_prefix"},
		{"rama de integración inválida", branchConfigBase + "git:\n  integration_branch: \"a b\"\n", "integration_branch"},
		{"repositories con ruta fuera de la carpeta maestra", branchConfigBase + "repositories:\n  - path: ../fuera\n", "repositories[0]"},
		{"YAML ilegible", "workspace: [sin cerrar\n", "axiom.yaml"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newBranchRepository(t)
			if err := os.WriteFile(filepath.Join(repo, "axiom.yaml"), []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}

			code, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", repo)
			if code != 1 {
				t.Errorf("código de salida = %d, se esperaba 1", code)
			}
			result := decodeBranchResult(t, stdout)
			if result.Status != "refused" || result.StatusLabel != "rechazado" || result.Code != "config_invalid" {
				t.Errorf("status %q / %q, code %q; se esperaba refused / rechazado / config_invalid", result.Status, result.StatusLabel, result.Code)
			}
			if !strings.Contains(result.Message, tc.wantCause) {
				t.Errorf("el mensaje %q debe nombrar la causa %q", result.Message, tc.wantCause)
			}
			for _, want := range []string{"axiom workspace validate --path", pathquote.Quote(repo)} {
				if !strings.Contains(result.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
				}
			}
			if len(result.Repositories) != 0 {
				t.Errorf("un rechazo no lleva repositorios: %+v", result.Repositories)
			}
		})
	}
}

// REQ-25.2: un fallo de git que ningún otro código cubre es `git_failed` y nombra la
// operación y resume el error. Aquí git no se puede ejecutar: el PATH no lo contiene.
func TestBranchStatusReportsGitFailedWhenGitCannotRun(t *testing.T) {
	repo := newBranchRepository(t)
	t.Setenv("PATH", t.TempDir())

	code, stdout, _ := runBranchCLI(t, "status", "--json", "--cwd", repo)
	if code != 1 {
		t.Errorf("código de salida = %d, se esperaba 1", code)
	}
	result := decodeBranchResult(t, stdout)
	if result.Status != "error" || result.StatusLabel != "error" || result.Code != "git_failed" {
		t.Errorf("status %q / %q, code %q; se esperaba error / error / git_failed", result.Status, result.StatusLabel, result.Code)
	}
	if result.Details["operation"] == "" || !strings.Contains(result.Message, result.Details["operation"]) {
		t.Errorf("el mensaje %q debe nombrar la operación %q", result.Message, result.Details["operation"])
	}
	if result.Details["summary"] == "" || !strings.Contains(result.Message, result.Details["summary"]) {
		t.Errorf("el mensaje %q debe resumir el error %q", result.Message, result.Details["summary"])
	}
	for _, want := range []string{"axiom branch status --cwd", pathquote.Quote(repo)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
}
