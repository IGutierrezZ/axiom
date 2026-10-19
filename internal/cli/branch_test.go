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
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
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

// sealBranchChange sella el kickoff de un cambio con los roles dados, como lo hace
// `axiom sdd kickoff seal`.
func sealBranchChange(t *testing.T, master, change string, roles ...string) {
	t.Helper()
	changeRoot := filepath.Join(master, "openspec", "changes", change)
	if err := os.MkdirAll(changeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	var sealed []kickoff.KickoffRole
	for _, role := range roles {
		sealed = append(sealed, kickoff.KickoffRole{Role: role, GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"})
	}
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema:    kickoff.KickoffSchemaV1,
		Change:    change,
		SealedBy:  "prueba",
		Config:    kickoff.FlowConfig{FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit", HandoffPolicy: kickoff.HandoffNone, Roles: sealed},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	})
	if err != nil || !ok {
		t.Fatalf("sellar el kickoff de %s: %v (publicado: %v)", change, err, ok)
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

// Lo que las pruebas leen del `axiom.branch-result/v1` de `status`.
type branchBaseJSON struct {
	Ref     string `json:"ref"`
	SHA     string `json:"sha"`
	Source  string `json:"source"`
	Fetched bool   `json:"fetched"`
}

type branchBranchJSON struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	KindLabel  string `json:"kindLabel"`
	Role       string `json:"role"`
	Exists     bool   `json:"exists"`
	Worktree   string `json:"worktree"`
	Dirty      bool   `json:"dirty"`
	ComparedTo string `json:"comparedTo"`
	Ahead      *int   `json:"ahead"`
	Behind     *int   `json:"behind"`
	PRTarget   string `json:"prTarget"`
}

type branchRepositoryJSON struct {
	Name              string             `json:"name"`
	GitRoot           string             `json:"gitRoot"`
	CommonDir         string             `json:"commonDir"`
	Role              string             `json:"role"`
	RoleLabel         string             `json:"roleLabel"`
	Roles             []string           `json:"roles"`
	IntegrationBranch string             `json:"integrationBranch"`
	CurrentBranch     string             `json:"currentBranch"`
	Detached          bool               `json:"detached"`
	Base              *branchBaseJSON    `json:"base"`
	Branches          []branchBranchJSON `json:"branches"`
}

type branchSkippedJSON struct {
	Path    string `json:"path"`
	Role    string `json:"role"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type branchResultJSON struct {
	Schema        string                 `json:"schema"`
	Operation     string                 `json:"operation"`
	Status        string                 `json:"status"`
	StatusLabel   string                 `json:"statusLabel"`
	Code          string                 `json:"code"`
	Message       string                 `json:"message"`
	Continuation  string                 `json:"continuation"`
	Details       map[string]string      `json:"details"`
	Change        string                 `json:"change"`
	Lane          string                 `json:"lane"`
	Topology      string                 `json:"topology"`
	WorkspaceRoot string                 `json:"workspaceRoot"`
	PlanningHome  string                 `json:"planningHome"`
	OddDocument   string                 `json:"oddDocument"`
	Repositories  []branchRepositoryJSON `json:"repositories"`
	Skipped       []branchSkippedJSON    `json:"skipped"`
	Warnings      []string               `json:"warnings"`
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

// REQ-25.2: `status --change x --json --cwd <ruta>` imprime un `axiom.branch-result/v1`
// con el cambio, el carril y la topología y, por repositorio, su nombre, papel, raíz
// git, git common dir, rama actual (por ReadHead), rama de integración y base, con las
// ramas del plan.
func TestBranchStatusPrintsTheResultAsJSON(t *testing.T) {
	repo := newBranchRepository(t)
	branchTestGit(t, repo, "branch", "feat/x")

	code, stdout, stderr := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)
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
	if result.Change != "x" || result.Lane != "sdd" || result.Topology != "monorepo-embedded" {
		t.Errorf("cambio %q, carril %q, topología %q; se esperaba x, sdd y monorepo-embedded", result.Change, result.Lane, result.Topology)
	}
	// El workspace del cambio está en su worktree, aún sin crear: <raíz de worktrees>/<cambio>/<repo>.
	wantWorkspace := filepath.Join(filepath.Dir(repo), filepath.Base(repo)+"-wt", "x", filepath.Base(repo))
	if result.WorkspaceRoot != wantWorkspace || result.PlanningHome != filepath.Join(wantWorkspace, "openspec") {
		t.Errorf("workspaceRoot %q, planningHome %q; se esperaba %q y su carpeta openspec", result.WorkspaceRoot, result.PlanningHome, wantWorkspace)
	}

	repository := onlyBranchRepository(t, result)
	wantBranchDirectory(t, "gitRoot", repository.GitRoot, repo)
	wantBranchDirectory(t, "commonDir", repository.CommonDir, filepath.Join(repo, ".git"))
	if repository.Name != filepath.Base(repo) || repository.Role != "change_branch" || repository.RoleLabel != "repositorio con rama de cambio" {
		t.Errorf("nombre %q, role %q / %q", repository.Name, repository.Role, repository.RoleLabel)
	}
	if repository.CurrentBranch != "main" || repository.Detached || repository.IntegrationBranch != "main" {
		t.Errorf("currentBranch %q, detached %v, integración %q; se esperaba main, false y main", repository.CurrentBranch, repository.Detached, repository.IntegrationBranch)
	}
	if repository.Base == nil || repository.Base.Ref != "main" || repository.Base.Source != "local" || repository.Base.Fetched || len(repository.Base.SHA) != 40 {
		t.Errorf("base = %+v, se esperaba la rama local main sin fetch", repository.Base)
	}
	if len(repository.Branches) != 1 {
		t.Fatalf("ramas = %+v, se esperaba solo feat/x", repository.Branches)
	}
	feature := repository.Branches[0]
	if feature.Name != "feat/x" || feature.Kind != "change" || feature.KindLabel != "rama del cambio" || !feature.Exists || feature.PRTarget != "main" {
		t.Errorf("rama = %+v", feature)
	}
	if feature.Ahead == nil || *feature.Ahead != 0 || feature.Behind == nil || *feature.Behind != 0 || feature.ComparedTo != "main" {
		t.Errorf("la rama recién creada no tiene adelanto ni atraso respecto de main: %+v", feature)
	}
}

// REQ-25.15 y REQ-25.2 (escenarios «Cambio no iniciado» y «Los valores del JSON son
// tokens en inglés con etiqueta en castellano»): un cambio sin ramas es `not_started`
// con la etiqueta «no iniciado» y código de salida 0.
func TestBranchStatusOfAChangeWithoutBranchesIsNotStarted(t *testing.T) {
	repo := newBranchRepository(t)

	code, stdout, stderr := runBranchCLI(t, "status", "--change", "nuevo", "--json", "--cwd", repo)

	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	result := decodeBranchResult(t, stdout)
	if result.Status != "not_started" || result.StatusLabel != "no iniciado" || result.Code != "" {
		t.Errorf("status %q / %q, code %q; se esperaba not_started / no iniciado sin código", result.Status, result.StatusLabel, result.Code)
	}
	repository := onlyBranchRepository(t, result)
	if len(repository.Branches) != 1 || repository.Branches[0].Name != "feat/nuevo" || repository.Branches[0].Exists {
		t.Errorf("ramas = %+v, se esperaba la rama planificada feat/nuevo sin crear", repository.Branches)
	}

	_, text, _ := runBranchCLI(t, "status", "--change", "nuevo", "--cwd", repo)
	if !strings.Contains(text, "Estado: no iniciado") {
		t.Errorf("el texto debe decir «Estado: no iniciado»:\n%s", text)
	}
}

// La salida es estable: dos ejecuciones sobre el mismo estado dan los mismos bytes.
func TestBranchStatusOutputIsDeterministic(t *testing.T) {
	repo := newBranchRepository(t)
	_, first, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)
	_, second, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)
	if first != second || first == "" {
		t.Errorf("la salida cambió entre ejecuciones:\n%s\n---\n%s", first, second)
	}
	if !strings.HasSuffix(first, "\n") {
		t.Errorf("la salida JSON debe terminar en un salto de línea: %q", first)
	}
}

// Desde un worktree enlazado, la raíz es la del worktree y el common dir el del
// checkout principal; la rama actual es la del worktree.
func TestBranchStatusInALinkedWorktree(t *testing.T) {
	main := newBranchRepository(t)
	linked := filepath.Join(t.TempDir(), "enlazado")
	branchTestGit(t, main, "worktree", "add", "-q", "-b", "inc/mi-cambio", linked, "HEAD")

	code, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", linked)
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

	code, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)
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
	branchTestGit(t, repo, "branch", "feat/x")
	if err := os.WriteFile(filepath.Join(repo, ".git", "HEAD"), []byte("ref: refs/heads/.invalid\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)
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
		_, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)
		wantBranchDirectory(t, "gitRoot", onlyBranchRepository(t, decodeBranchResult(t, stdout)).GitRoot, repo)
	})
	t.Run("sin --cwd rige el directorio del proceso", func(t *testing.T) {
		t.Chdir(sub)
		_, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json")
		wantBranchDirectory(t, "gitRoot", onlyBranchRepository(t, decodeBranchResult(t, stdout)).GitRoot, repo)
	})
	t.Run("un --cwd relativo se resuelve contra el proceso y la salida es absoluta", func(t *testing.T) {
		t.Chdir(repo)
		_, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", filepath.Join("app", "web"))
		wantBranchDirectory(t, "gitRoot", onlyBranchRepository(t, decodeBranchResult(t, stdout)).GitRoot, repo)
	})
	t.Run("una subcarpeta resuelve a la raíz del repositorio", func(t *testing.T) {
		_, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", sub)
		wantBranchDirectory(t, "gitRoot", onlyBranchRepository(t, decodeBranchResult(t, stdout)).GitRoot, repo)
	})
}

// REQ-25.2: fuera de git, `not_a_git_repository` con código 1, token con etiqueta y
// una continuación en castellano con la ruta citada.
func TestBranchStatusOutsideGitIsRefusedWithATypedCode(t *testing.T) {
	isolateBranchGit(t)
	outside := branchOutsideGit(t)

	code, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", outside)
	if code != 1 {
		t.Errorf("código de salida = %d, se esperaba 1", code)
	}
	result := decodeBranchResult(t, stdout)
	if result.Status != "refused" || result.StatusLabel != "rechazado" || result.Code != "not_a_git_repository" {
		t.Errorf("status %q / %q, code %q; se esperaba refused / rechazado / not_a_git_repository", result.Status, result.StatusLabel, result.Code)
	}
	if len(result.Repositories) != 0 || result.Change != "" {
		t.Errorf("un rechazo no lleva datos del cambio: %+v", result)
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

	code, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", missing)
	result := decodeBranchResult(t, stdout)
	if code != 1 || result.Code != "not_a_git_repository" || !strings.Contains(result.Message, "no existe") {
		t.Errorf("código %d, resultado %+v; se esperaba 1, not_a_git_repository y un mensaje que diga que no existe", code, result)
	}
}

// REQ-25.3 y REQ-25.11: un nombre de cambio inválido es `invalid_name`, con código de
// salida 1, antes de leer nada con él.
func TestBranchStatusRefusesAnInvalidChangeNameAsInvalidName(t *testing.T) {
	repo := newBranchRepository(t)

	for _, invalid := range []string{"a b:c", "CON", "../fuera"} {
		t.Run(invalid, func(t *testing.T) {
			code, stdout, _ := runBranchCLI(t, "status", "--change", invalid, "--json", "--cwd", repo)
			if code != 1 {
				t.Errorf("código de salida = %d, se esperaba 1", code)
			}
			result := decodeBranchResult(t, stdout)
			if result.Status != "refused" || result.Code != "invalid_name" || result.Details["field"] != "--change" || result.Details["value"] != invalid {
				t.Errorf("status %q, code %q, details %v; se esperaba refused / invalid_name con el campo --change", result.Status, result.Code, result.Details)
			}
			for _, want := range []string{"axiom branch status --change <cambio> --cwd", pathquote.Quote(repo)} {
				if !strings.Contains(result.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
				}
			}
		})
	}
}

// Sin --json la salida es texto en castellano, con los mismos códigos de salida.
func TestBranchStatusPrintsSpanishTextWithoutJSON(t *testing.T) {
	repo := newBranchRepository(t)
	branchTestGit(t, repo, "branch", "feat/x")

	code, stdout, stderr := runBranchCLI(t, "status", "--change", "x", "--cwd", repo)
	if code != 0 || stderr != "" {
		t.Fatalf("código %d, stderr %q; se esperaba 0 y stderr vacío", code, stderr)
	}
	for _, want := range []string{"Cambio: x (carril sdd)", "Estado: listo", "Repositorio git:", "Directorio común de git:", "Rama actual: main", "feat/x (rama del cambio)", "PR hacia main"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("la salida de texto debe contener %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, `"schema"`) {
		t.Errorf("sin --json la salida no es JSON:\n%s", stdout)
	}

	outside := branchOutsideGit(t)
	code, stdout, stderr = runBranchCLI(t, "status", "--change", "x", "--cwd", outside)
	if code != 1 || stdout != "" {
		t.Errorf("fuera de git: código %d, stdout %q; se esperaba 1 y stdout vacío", code, stdout)
	}
	for _, want := range []string{"not_a_git_repository", "axiom branch status --cwd"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("el rechazo en texto debe contener %q:\n%s", want, stderr)
		}
	}
}

// Los errores de uso de `status` terminan con 2 y nombran la continuación: el cambio
// es obligatorio en esta fase y el carril solo admite sdd u odd.
func TestBranchStatusUsageErrorsExitWithCode2(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"sin --change", []string{"status"}, "--change"},
		{"sin --change y con --json", []string{"status", "--json"}, "--change"},
		{"--change vacío", []string{"status", "--change", ""}, "--change"},
		{"--change solo con espacios", []string{"status", "--change", "  "}, "--change"},
		{"--change sin valor", []string{"status", "--change"}, "change"},
		{"carril desconocido", []string{"status", "--change", "x", "--lane", "xyz"}, "--lane 'xyz'"},
		{"bandera desconocida", []string{"status", "--change", "x", "--no-existe"}, "no-existe"},
		{"argumento posicional", []string{"status", "--change", "x", "sobra"}, "'sobra'"},
		{"--cwd sin valor", []string{"status", "--change", "x", "--cwd"}, "cwd"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runBranchCLI(t, tc.args...)
			if code != 2 || stdout != "" {
				t.Errorf("código %d, stdout %q; se esperaba 2 y stdout vacío", code, stdout)
			}
			if !strings.Contains(stderr, "axiom branch status --help") || !strings.Contains(stderr, tc.want) {
				t.Errorf("stderr %q debe nombrar %q y `axiom branch status --help`", stderr, tc.want)
			}
		})
	}

	t.Run("status --help describe las banderas", func(t *testing.T) {
		code, stdout, _ := runBranchCLI(t, "status", "--help")
		if code != 0 {
			t.Errorf("código %d, se esperaba 0", code)
		}
		for _, want := range []string{"--change", "--lane", "--cwd", "--json", "no iniciado"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("la ayuda de status debe documentar %s:\n%s", want, stdout)
			}
		}
	})
}

// REQ-25.13: el carril ODD se elige con `--lane odd` y el resultado nombra el documento
// `odd/tasks/<feature>.md`.
func TestBranchStatusOddLane(t *testing.T) {
	repo := newBranchRepository(t)

	code, stdout, _ := runBranchCLI(t, "status", "--change", "mi-feature", "--lane", "odd", "--json", "--cwd", repo)

	if code != 0 {
		t.Fatalf("código de salida = %d, se esperaba 0\n%s", code, stdout)
	}
	result := decodeBranchResult(t, stdout)
	if result.Lane != "odd" || result.Change != "mi-feature" {
		t.Errorf("carril %q, cambio %q; se esperaba odd y mi-feature", result.Lane, result.Change)
	}
	if want := filepath.Join("mi-feature", filepath.Base(repo), "odd", "tasks", "mi-feature.md"); !strings.HasSuffix(result.OddDocument, want) {
		t.Errorf("oddDocument = %q, se esperaba una ruta que termine en %q", result.OddDocument, want)
	}
	if repository := onlyBranchRepository(t, result); len(repository.Branches) != 1 || repository.Branches[0].Name != "feat/mi-feature" {
		t.Errorf("ramas = %+v, se esperaba solo feat/mi-feature", repository.Branches)
	}
}

// REQ-25.4: con dos roles sellados, el JSON lleva la rama de seguimiento y una rama por
// rol, con su tipo y su etiqueta y el destino propuesto de su PR, y el aviso de
// publicar la de seguimiento.
func TestBranchStatusWithTwoSealedRolesShowsTheRoleBranches(t *testing.T) {
	repo := newBranchRepository(t)
	configWithTwoRoles := strings.Replace(branchConfigBase, "roles:\n", "roles:\n  qa:\n    name: QA\n    repositories:\n      - path: .\n", 1)
	if err := os.WriteFile(filepath.Join(repo, "axiom.yaml"), []byte(configWithTwoRoles), 0o644); err != nil {
		t.Fatal(err)
	}
	sealBranchChange(t, repo, "x", "core", "qa")

	code, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)

	if code != 0 {
		t.Fatalf("código de salida = %d, se esperaba 0\n%s", code, stdout)
	}
	result := decodeBranchResult(t, stdout)
	repository := onlyBranchRepository(t, result)
	if strings.Join(repository.Roles, ",") != "core,qa" {
		t.Errorf("roles = %v, se esperaba core y qa", repository.Roles)
	}
	type shape struct{ kind, kindLabel, role, prTarget string }
	want := map[string]shape{
		"feat/x":      {"change", "rama del cambio", "", "main"},
		"feat/x-core": {"role", "rama de rol", "core", "feat/x"},
		"feat/x-qa":   {"role", "rama de rol", "qa", "feat/x"},
	}
	if len(repository.Branches) != len(want) {
		t.Fatalf("ramas = %+v, se esperaban %d", repository.Branches, len(want))
	}
	for _, described := range repository.Branches {
		expected, known := want[described.Name]
		if !known || described.Kind != expected.kind || described.KindLabel != expected.kindLabel || described.Role != expected.role || described.PRTarget != expected.prTarget {
			t.Errorf("rama %+v, se esperaba %+v", described, expected)
		}
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "Publica la rama de seguimiento feat/x") {
		t.Errorf("avisos = %v, se esperaba el de publicar la rama de seguimiento", result.Warnings)
	}
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
// ninguna operación remota, ni ninguna fuera de la lista cerrada de lecturas.
func TestBranchStatusLaunchesNoRemoteGitOperation(t *testing.T) {
	repo := newBranchRepository(t)
	branchTestGit(t, repo, "remote", "add", "origin", "https://ejemplo.invalid/repo.git")
	branchTestGit(t, repo, "branch", "feat/x")

	var seen [][]string
	gitcommon.Observe = func(dir string, args []string) {
		seen = append(seen, append([]string{dir}, args...))
	}
	t.Cleanup(func() { gitcommon.Observe = nil })

	for _, args := range [][]string{
		{"status", "--change", "x", "--json", "--cwd", repo},
		{"status", "--change", "x", "--cwd", repo},
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
		// El HEAD se lee de ficheros (REQ-25.21) y solo se lanzan las lecturas de la
		// lista cerrada de `internal/branch/gitcli`.
		switch invocation[1] {
		case "rev-parse", "worktree", "status", "rev-list":
		default:
			t.Errorf("status lanzó %q, fuera de la lista cerrada de lecturas: %v", invocation[1], invocation)
		}
	}
}

// REQ-25.2 (G-4): cada token del JSON lleva su etiqueta en castellano en un campo
// hermano, tanto en un resultado listo como en un rechazo.
func TestBranchStatusPairsEveryTokenWithItsLabel(t *testing.T) {
	repo := newBranchRepository(t)
	branchTestGit(t, repo, "branch", "feat/x")
	_, ready, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)
	_, refused, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", branchOutsideGit(t))

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

	// El papel del repositorio y el tipo de la rama también llevan su etiqueta.
	var generic map[string]any
	if err := json.Unmarshal([]byte(ready), &generic); err != nil {
		t.Fatal(err)
	}
	repository := generic["repositories"].([]any)[0].(map[string]any)
	feature := repository["branches"].([]any)[0].(map[string]any)
	if repository["roleLabel"] == "" || repository["roleLabel"] == nil || feature["kindLabel"] == "" || feature["kindLabel"] == nil {
		t.Errorf("role y kind llevan su etiqueta: %v / %v", repository, feature)
	}
}

// REQ-25.3: sin configuración rige `main`; el bloque `git:` lo sustituye; una entrada
// de `repositories:` prevalece sobre `git:` solo en su repositorio.
func TestBranchStatusShowsTheIntegrationBranchPerRepository(t *testing.T) {
	statusOf := func(t *testing.T, cwd string) []branchRepositoryJSON {
		t.Helper()
		code, stdout, stderr := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", cwd)
		if code != 0 || stderr != "" {
			t.Fatalf("código %d, stderr %q; se esperaba 0\n%s", code, stderr, stdout)
		}
		return decodeBranchResult(t, stdout).Repositories
	}
	integrationOf := func(t *testing.T, cwd string) string {
		t.Helper()
		repositories := statusOf(t, cwd)
		if len(repositories) != 1 {
			t.Fatalf("se esperaba un único repositorio, hay %d: %+v", len(repositories), repositories)
		}
		return repositories[0].IntegrationBranch
	}

	t.Run("sin axiom.yaml rige main", func(t *testing.T) {
		repo := newBranchRepository(t)
		if got := integrationOf(t, repo); got != "main" {
			t.Errorf("integrationBranch = %q, se esperaba main", got)
		}
	})
	t.Run("axiom.yaml sin bloques rige main", func(t *testing.T) {
		repo := newBranchRepository(t)
		writeBranchConfig(t, repo, "")
		if got := integrationOf(t, repo); got != "main" {
			t.Errorf("integrationBranch = %q, se esperaba main", got)
		}
	})
	t.Run("el bloque git sustituye el valor por defecto", func(t *testing.T) {
		repo := newBranchRepository(t)
		writeBranchConfig(t, repo, "git:\n  integration_branch: develop\n")
		if got := integrationOf(t, repo); got != "develop" {
			t.Errorf("integrationBranch = %q, se esperaba develop", got)
		}
	})
	t.Run("una entrada de repositories para el repositorio de la carpeta maestra prevalece", func(t *testing.T) {
		repo := newBranchRepository(t)
		writeBranchConfig(t, repo, "git:\n  integration_branch: develop\nrepositories:\n  - path: .\n    integration_branch: release\n")
		if got := integrationOf(t, repo); got != "release" {
			t.Errorf("integrationBranch = %q, se esperaba release", got)
		}
	})
	t.Run("cada repositorio del plan toma su propia política", func(t *testing.T) {
		master := newBranchRepository(t)
		config := `workspace:
  name: ejemplo
  topology: monorepo-embedded
  specs_repository: .
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
  frontend:
    name: Frontend
    repositories:
      - path: repo-frontend
git:
  integration_branch: main
repositories:
  - path: repo-backend
    integration_branch: release
`
		if err := os.WriteFile(filepath.Join(master, "axiom.yaml"), []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"repo-backend", "repo-frontend"} {
			dir := filepath.Join(master, name)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			initBranchRepository(t, dir)
		}
		branchTestGit(t, filepath.Join(master, "repo-backend"), "branch", "release")
		sealBranchChange(t, master, "x", "fullstack")

		got := map[string]string{}
		for _, repository := range statusOf(t, master) {
			got[repository.Name] = repository.IntegrationBranch
		}
		want := map[string]string{filepath.Base(master): "main", "repo-backend": "release", "repo-frontend": "main"}
		if len(got) != len(want) {
			t.Fatalf("repositorios = %v, se esperaban %v", got, want)
		}
		for name, integration := range want {
			if got[name] != integration {
				t.Errorf("%s: integrationBranch = %q, se esperaba %q", name, got[name], integration)
			}
		}
	})
	t.Run("un repositorio fuera de la carpeta maestra solo hereda la política del workspace", func(t *testing.T) {
		repo := newBranchRepository(t)
		master := filepath.Join(repo, "app")
		if err := os.MkdirAll(master, 0o755); err != nil {
			t.Fatal(err)
		}
		writeBranchConfig(t, master, "git:\n  integration_branch: develop\nrepositories:\n  - path: .\n    integration_branch: release\n")
		if got := integrationOf(t, master); got != "develop" {
			t.Errorf("integrationBranch = %q; la entrada «.» es de la carpeta maestra, no del repositorio que la contiene (se esperaba develop)", got)
		}
	})
	t.Run("sin --json la rama de integración sale en el texto", func(t *testing.T) {
		repo := newBranchRepository(t)
		writeBranchConfig(t, repo, "git:\n  integration_branch: develop\n")
		_, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--cwd", repo)
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

			code, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)
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

	code, stdout, _ := runBranchCLI(t, "status", "--change", "x", "--json", "--cwd", repo)
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
