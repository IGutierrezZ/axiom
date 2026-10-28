package dashboard

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Escritores guardados del dashboard y `ensure` al crear un cambio (INC-25 S8-c4; REQ-25.19
// escenarios 4 y 5, REQ-25.32 escenarios 1 y 4; decisión T4). Pruebas con git real, entorno
// aislado y sin red: un monorepo embebido en un directorio temporal, con las preferencias de
// worktree en el hub del HOME de pruebas y la raíz de worktrees también temporal. Ninguna crea
// worktrees del repositorio real.

// requireGit omite la prueba con -short o sin git, y aísla git de la configuración del desarrollador.
func requireGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("prueba con git real: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está instalado: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Autor de Prueba")
	t.Setenv("GIT_AUTHOR_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Autor de Prueba")
	t.Setenv("GIT_COMMITTER_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
}

// runGit ejecuta git en dir y devuelve su salida sin el salto de línea final.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
	return strings.TrimRight(string(output), "\r\n")
}

// setWorktreePrefs registra project en el hub del HOME de pruebas con el modo de trabajo y la raíz
// de worktrees indicados (REQ-25.29).
func setWorktreePrefs(t *testing.T, project, mode, root string) {
	t.Helper()
	manager, err := hub.NewManager("")
	if err != nil {
		t.Fatalf("hub: %v", err)
	}
	if _, err := manager.Register(project, "", ""); err != nil {
		t.Fatalf("registrar %s en el hub: %v", project, err)
	}
	if _, err := manager.SetWorktreePrefs(project, hub.WorktreePrefs{Mode: mode, Root: root}); err != nil {
		t.Fatalf("preferencias de worktree de %s: %v", project, err)
	}
}

// commitGitRepo convierte root en un repositorio git en `main` con un commit base de sus ficheros.
func commitGitRepo(t *testing.T, root string) {
	t.Helper()
	requireGit(t)
	initGitFixture(t, root)
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-q", "--allow-empty", "-m", "base")
}

// gitProject convierte root en un repositorio git en `main` con un commit base de sus ficheros y le
// da las preferencias de worktree de mode, con la raíz de worktrees en un directorio temporal.
// Devuelve esa raíz.
func gitProject(t *testing.T, root, mode string) string {
	t.Helper()
	commitGitRepo(t, root)
	worktrees := filepath.Join(t.TempDir(), "worktrees")
	setWorktreePrefs(t, root, mode, worktrees)
	return worktrees
}

// embeddedProject es un monorepo embebido (axiom.yaml y openspec/changes) con git en `main`.
func embeddedProject(t *testing.T, mode string) workspacetest.Workspace {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	gitProject(t, ws.Master, mode)
	return ws
}

// worktreeOfBranch devuelve la ruta del worktree donde está desplegada la rama, o falla.
func worktreeOfBranch(t *testing.T, repo, name string) string {
	t.Helper()
	var current string
	for _, line := range strings.Split(runGit(t, repo, "worktree", "list", "--porcelain"), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "worktree "):
			current = filepath.FromSlash(strings.TrimPrefix(line, "worktree "))
		case line == "branch refs/heads/"+name:
			return current
		}
	}
	t.Fatalf("la rama %s no está desplegada en ningún worktree de %s", name, repo)
	return ""
}

func branchExists(t *testing.T, repo, name string) bool {
	t.Helper()
	return runGit(t, repo, "branch", "--list", name) != ""
}

func wantAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s no debía existir (%v)", path, err)
	}
}

func wantFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s debía existir: %v", path, err)
	}
	return string(data)
}

func wantSameDirectory(t *testing.T, label, got, want string) {
	t.Helper()
	if !pathidentity.SameDirectory(got, want) {
		t.Errorf("%s = %q, se esperaba %q", label, got, want)
	}
}

// wantRefusal exige que err sea el rechazo tipado con el código dado y una continuación ejecutable.
func wantRefusal(t *testing.T, err error, code branch.Code) *branch.RefusalError {
	t.Helper()
	var refusal *branch.RefusalError
	if !errors.As(err, &refusal) {
		t.Fatalf("err = %v, se esperaba un *branch.RefusalError %s", err, code)
	}
	if refusal.Code != code {
		t.Fatalf("código = %s, se esperaba %s: %v", refusal.Code, code, err)
	}
	if !strings.Contains(refusal.Continuation, "axiom ") {
		t.Errorf("la continuación debe ser una orden de axiom: %q", refusal.Continuation)
	}
	return refusal
}

func postIncrement(t *testing.T, svc *Service, body CreateIncrementRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	NewServer(svc).Router().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/increments", bytes.NewReader(payload)))
	return rr
}

// REQ-25.32 escenario 1: la creación ejecuta `ensure` (rama y worktree de `feat/x`) y escribe
// `proposal.md` en el `openspec/changes/x/` del worktree; el checkout principal no recibe nada.
func TestCreateIncrementEnsuresThenWritesInTheWorktree(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)

	rr := postIncrement(t, svc, CreateIncrementRequest{Name: "x", Intent: "Propósito X"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("POST /api/increments = %d: %s", rr.Code, rr.Body.String())
	}
	var res CreateIncrementResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("respuesta no válida: %v", err)
	}

	if !branchExists(t, ws.Master, "feat/x") {
		t.Fatal("ensure debía crear la rama feat/x")
	}
	worktree := worktreeOfBranch(t, ws.Master, "feat/x")
	if pathidentity.SameDirectory(worktree, ws.Master) {
		t.Fatalf("feat/x no debía desplegarse en el checkout principal %s", ws.Master)
	}
	changeDir := filepath.Join(worktree, "openspec", "changes", "x")
	wantSameDirectory(t, "carpeta de proposal.md", filepath.Dir(res.Path), changeDir)
	if body := wantFile(t, filepath.Join(changeDir, "proposal.md")); !strings.Contains(body, "Propósito X") {
		t.Errorf("proposal.md del worktree sin el propósito:\n%s", body)
	}

	wantAbsent(t, filepath.Join(ws.Home, "changes", "x"))
	if got := runGit(t, ws.Master, "branch", "--show-current"); got != "main" {
		t.Errorf("el checkout principal debía seguir en main, está en %q", got)
	}
	if status := runGit(t, ws.Master, "status", "--porcelain"); status != "" {
		t.Errorf("el checkout principal debía quedar limpio:\n%s", status)
	}
}

// REQ-25.32 escenario 4: un rechazo de `ensure` (árbol sucio en `in_place`) no deja carpeta, ni rama,
// ni worktree, y llega con su código tipado y su continuación.
func TestCreateIncrementRejectedEnsureLeavesNoFolder(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeInPlace)
	if err := os.WriteFile(filepath.Join(ws.Master, "stray.txt"), []byte("sin confirmar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(ws.Master)

	_, err := svc.CreateIncrement(CreateIncrementRequest{Name: "x"})
	wantRefusal(t, err, branch.CodeDirtyWorktree)

	// REQ-25.32: por HTTP el rechazo es un 409 con `{code, error, continuation}`.
	rr := postIncrement(t, svc, CreateIncrementRequest{Name: "x"})
	body := wantConflict(t, rr)
	if body["code"] != string(branch.CodeDirtyWorktree) {
		t.Errorf("code = %v, se esperaba %s: %v", body["code"], branch.CodeDirtyWorktree, body)
	}
	if msg, _ := body["error"].(string); msg == "" {
		t.Errorf("error debía describir el rechazo: %v", body)
	}
	if next, _ := body["continuation"].(string); !strings.Contains(next, "axiom ") {
		t.Errorf("continuation debía ser una orden de axiom: %v", body)
	}
	if _, present := body["decision"]; present {
		t.Errorf("un rechazo que no es decision_required no lleva decisión: %v", body)
	}
	if _, present := body["status"]; present {
		t.Errorf("un rechazo tipado no lleva status: %v", body)
	}

	wantAbsent(t, filepath.Join(ws.Home, "changes", "x"))
	if branchExists(t, ws.Master, "feat/x") {
		t.Error("un rechazo de ensure no debía dejar la rama feat/x")
	}
	if got := runGit(t, ws.Master, "branch", "--show-current"); got != "main" {
		t.Errorf("el checkout debía seguir en main, está en %q", got)
	}
}

// wantConflict exige un 409 con cuerpo JSON y lo devuelve decodificado.
func wantConflict(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if rr.Code != http.StatusConflict {
		t.Fatalf("POST /api/increments = %d, se esperaba 409: %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("el 409 debía llevar un cuerpo JSON: %v\n%s", err, rr.Body.String())
	}
	return body
}

// REQ-25.32 escenario 2: modo `ask` sin respuesta. El servicio devuelve `decision_required` con su
// decisión y el HTTP, un 409 con `{status, statusLabel, code, decision}`; no se crea nada.
func TestCreateIncrementInAskModeReturnsTheDecisionWithoutWriting(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeAsk)
	svc := NewService(ws.Master)

	_, err := svc.CreateIncrement(CreateIncrementRequest{Name: "x"})
	refusal := wantRefusal(t, err, branch.CodeDecisionRequired)
	if refusal.Decision == nil || len(refusal.Decision.Choices) != 2 {
		t.Errorf("decision_required debía llevar la decisión con dos respuestas: %+v", refusal.Decision)
	}

	body := wantConflict(t, postIncrement(t, svc, CreateIncrementRequest{Name: "x"}))
	if body["status"] != string(branch.StatusDecisionRequired) || body["statusLabel"] != branch.Label(string(branch.StatusDecisionRequired)) {
		t.Errorf("status/statusLabel = %v / %v: %v", body["status"], body["statusLabel"], body)
	}
	if body["code"] != string(branch.CodeDecisionRequired) {
		t.Errorf("code = %v, se esperaba %s", body["code"], branch.CodeDecisionRequired)
	}
	decision, _ := body["decision"].(map[string]any)
	choices, _ := decision["choices"].([]any)
	if decision["question"] == "" || len(choices) != 2 {
		t.Fatalf("decision debía llevar la pregunta y dos respuestas: %v", body["decision"])
	}
	answers := map[string]bool{}
	for _, raw := range choices {
		choice, _ := raw.(map[string]any)
		answer, _ := choice["answer"].(string)
		answers[answer] = true
	}
	if !answers["worktree"] || !answers["in_place"] {
		t.Errorf("las respuestas debían ser worktree e in_place: %v", answers)
	}

	wantAbsent(t, filepath.Join(ws.Home, "changes", "x"))
	if branchExists(t, ws.Master, "feat/x") {
		t.Error("decision_required no debía crear la rama")
	}
	if status := runGit(t, ws.Master, "status", "--porcelain"); status != "" {
		t.Errorf("decision_required no debía tocar el checkout:\n%s", status)
	}
}

// REQ-25.32 escenario 3: repetir la creación con `mode` completa la creación (`in_place` hace
// checkout de `feat/x` y escribe allí) y la respuesta es solo de ese cambio: otro cambio en modo
// `ask` vuelve a devolver `decision_required`.
func TestCreateIncrementRepeatedWithModeCompletesTheCreation(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeAsk)
	svc := NewService(ws.Master)
	wantConflict(t, postIncrement(t, svc, CreateIncrementRequest{Name: "x"}))
	wantAbsent(t, filepath.Join(ws.Home, "changes", "x"))

	rr := postIncrement(t, svc, CreateIncrementRequest{Name: "x", Intent: "Propósito X", Mode: "in_place"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("la repetición con mode debía crear el cambio: %d %s", rr.Code, rr.Body.String())
	}
	if got := runGit(t, ws.Master, "branch", "--show-current"); got != "feat/x" {
		t.Errorf("in_place debía hacer checkout de feat/x en el repositorio, está en %q", got)
	}
	if body := wantFile(t, filepath.Join(ws.Home, "changes", "x", "proposal.md")); !strings.Contains(body, "Propósito X") {
		t.Errorf("proposal.md sin el propósito:\n%s", body)
	}

	// La respuesta se registró para x, no para el proyecto: y sigue preguntando.
	runGit(t, ws.Master, "add", "-A")
	runGit(t, ws.Master, "commit", "-q", "-m", "x")
	runGit(t, ws.Master, "switch", "-q", "main")
	body := wantConflict(t, postIncrement(t, svc, CreateIncrementRequest{Name: "y"}))
	if body["code"] != string(branch.CodeDecisionRequired) {
		t.Errorf("otro cambio en modo ask debía volver a preguntar: %v", body)
	}
}

// REQ-25.32: un `mode` que `ensure` no admite es un rechazo (409) y no crea nada.
func TestCreateIncrementRefusesAnUnknownMode(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)

	body := wantConflict(t, postIncrement(t, svc, CreateIncrementRequest{Name: "x", Mode: "nonsense"}))

	if code, _ := body["code"].(string); code == "" {
		t.Errorf("el rechazo debía llevar un código tipado: %v", body)
	}
	wantAbsent(t, filepath.Join(ws.Home, "changes", "x"))
	if branchExists(t, ws.Master, "feat/x") {
		t.Error("un mode inválido no debía crear la rama")
	}
}

// REQ-25.32 escenario 5: en un multirepo el repositorio de specs se trabaja in situ, en su rama de
// integración: la creación escribe allí por HTTP, sin worktree ni rama de cambio.
func TestCreateIncrementInMultirepoWritesInSituInTheSpecsRepository(t *testing.T) {
	svc, ws := canonicalService(t, startAtMaster)
	commitGitRepo(t, ws.Specs)

	rr := postIncrement(t, svc, CreateIncrementRequest{Name: "x", Intent: "Propósito X"})

	if rr.Code != http.StatusCreated {
		t.Fatalf("POST /api/increments = %d: %s", rr.Code, rr.Body.String())
	}
	if body := wantFile(t, filepath.Join(ws.Home, "changes", "x", "proposal.md")); !strings.Contains(body, "Propósito X") {
		t.Errorf("proposal.md del repositorio de specs sin el propósito:\n%s", body)
	}
	if got := runGit(t, ws.Specs, "branch", "--show-current"); got != "main" {
		t.Errorf("el repositorio de specs debía seguir en su rama de integración, está en %q", got)
	}
	if branchExists(t, ws.Specs, "feat/x") {
		t.Error("el repositorio de specs no tiene rama de cambio")
	}
}

// REQ-25.19 escenario 5: el escritor guardado invocado sin `ensure` en `main` se niega con
// `wrong_branch` y la continuación de `ensure`, y no crea la carpeta; en la rama del cambio escribe.
func TestWriteIncrementWithoutEnsureOnMainIsWrongBranch(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)

	_, err := svc.writeIncrement(ws.Home, "x", CreateIncrementRequest{Name: "x"})

	refusal := wantRefusal(t, err, branch.CodeWrongBranch)
	if !strings.Contains(refusal.Continuation, "axiom branch ensure") || !strings.Contains(refusal.Continuation, "--change x") {
		t.Errorf("la continuación debe ser la orden de ensure del cambio x: %q", refusal.Continuation)
	}
	wantAbsent(t, filepath.Join(ws.Home, "changes", "x"))

	runGit(t, ws.Master, "switch", "-q", "-c", "feat/x")
	res, err := svc.writeIncrement(ws.Home, "x", CreateIncrementRequest{Name: "x"})
	if err != nil {
		t.Fatalf("en feat/x el escritor debía escribir: %v", err)
	}
	wantFile(t, res.Path)
}

const cumulativeTasks = "# Tareas\n- [x] 1.1 hecha\n- [ ] 1.2 pendiente\n"

// La migración al acumulativo guarda el cambio de origen (`wrong_branch` en `main`, sin tocar su
// `tasks.md` ni crear el acumulativo); el destino `cumulative-<rol>` está exento por nombre.
func TestMigrateIncompleteTasksToCumulativeGuardsTheOriginAndExemptsTheSink(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "orig", "tasks.md"), cumulativeTasks)
	gitProject(t, ws.Master, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)
	origin := filepath.Join(ws.Home, "changes", "orig", "tasks.md")
	sink := filepath.Join(ws.Home, "changes", "cumulative-backend")

	moved, err := svc.MigrateIncompleteTasksToCumulative("orig", "backend")

	wantRefusal(t, err, branch.CodeWrongBranch)
	if moved != 0 {
		t.Errorf("tareas migradas = %d, se esperaba 0 con el rechazo", moved)
	}
	if got := wantFile(t, origin); got != cumulativeTasks {
		t.Errorf("el rechazo no debía modificar tasks.md del origen:\n%s", got)
	}
	wantAbsent(t, sink)

	// El destino es un sumidero exento por nombre: la guarda no lo bloquea desde `main`.
	if err := cli.GuardChangeWriter(sink, "cumulative-backend", "escribir el acumulativo"); err != nil {
		t.Errorf("cumulative-backend debía estar exento: %v", err)
	}

	// En la rama del origen la migración procede y escribe el destino aunque esa rama no sea la suya.
	runGit(t, ws.Master, "switch", "-q", "-c", "feat/orig")
	moved, err = svc.MigrateIncompleteTasksToCumulative("orig", "backend")
	if err != nil || moved != 1 {
		t.Fatalf("en feat/orig la migración debía mover 1 tarea: moved=%d err=%v", moved, err)
	}
	if got := wantFile(t, filepath.Join(sink, "tasks.md")); !strings.Contains(got, "1.2 pendiente") {
		t.Errorf("el acumulativo no recoge la tarea:\n%s", got)
	}
}

// El endpoint de migración devuelve el rechazo de la guarda sin escribir nada.
func TestMigrateCumulativeEndpointRefusesOutsideTheChangeBranch(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "orig", "tasks.md"), cumulativeTasks)
	gitProject(t, ws.Master, hub.WorktreeModeWorktree)
	payload, _ := json.Marshal(MigrateCumulativeRequest{Change: "orig", Role: "backend"})

	rr := httptest.NewRecorder()
	NewServer(NewService(ws.Master)).Router().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/increments/migrate-cumulative", bytes.NewReader(payload)))

	if rr.Code != http.StatusInternalServerError || !strings.Contains(rr.Body.String(), "wrong_branch") {
		t.Errorf("POST migrate-cumulative = %d, se esperaba el rechazo wrong_branch: %s", rr.Code, rr.Body.String())
	}
	wantAbsent(t, filepath.Join(ws.Home, "changes", "cumulative-backend"))
}

// `CreateHandoff` consulta la guarda A antes de escribir `handoff.md`: `wrong_branch` en `main` y,
// en la rama del cambio, escribe.
func TestCreateHandoffIsGuardedByTheChangeBranch(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "x", "proposal.md"), "# x\n")
	gitProject(t, ws.Master, hub.WorktreeModeWorktree)
	svc := NewService(ws.Master)
	request := CreateHandoffRequest{Change: "x", FromPhase: "design", ToPhase: "tasks", ExecutiveSummary: "Resumen de prueba"}
	handoffPath := filepath.Join(ws.Home, "changes", "x", "handoff.md")

	_, err := svc.CreateHandoff(request)

	refusal := wantRefusal(t, err, branch.CodeWrongBranch)
	if !strings.Contains(refusal.Continuation, "--change x") {
		t.Errorf("la continuación debe nombrar el cambio x: %q", refusal.Continuation)
	}
	wantAbsent(t, handoffPath)

	runGit(t, ws.Master, "switch", "-q", "-c", "feat/x")
	res, err := svc.CreateHandoff(request)
	if err != nil {
		t.Fatalf("en feat/x el handoff debía escribirse: %v", err)
	}
	wantSameDirectory(t, "carpeta del handoff", filepath.Dir(res.FilePath), filepath.Dir(handoffPath))
	wantFile(t, handoffPath)
}
