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
	rr := postIncrement(t, svc, CreateIncrementRequest{Name: "x"})
	if rr.Code != http.StatusBadRequest {
		t.Errorf("POST /api/increments = %d, se esperaba el rechazo (el 409 tipado llega en S10-c1): %s", rr.Code, rr.Body.String())
	}

	wantAbsent(t, filepath.Join(ws.Home, "changes", "x"))
	if branchExists(t, ws.Master, "feat/x") {
		t.Error("un rechazo de ensure no debía dejar la rama feat/x")
	}
	if got := runGit(t, ws.Master, "branch", "--show-current"); got != "main" {
		t.Errorf("el checkout debía seguir en main, está en %q", got)
	}
}

// Modo `ask` sin respuesta: el rechazo es `decision_required` con su decisión y no se crea nada
// (la respuesta HTTP 409 y el campo `mode` llegan en S10-c1).
func TestCreateIncrementInAskModeReturnsTheDecisionWithoutWriting(t *testing.T) {
	ws := embeddedProject(t, hub.WorktreeModeAsk)
	svc := NewService(ws.Master)

	_, err := svc.CreateIncrement(CreateIncrementRequest{Name: "x"})
	refusal := wantRefusal(t, err, branch.CodeDecisionRequired)
	if refusal.Decision == nil || len(refusal.Decision.Choices) != 2 {
		t.Errorf("decision_required debía llevar la decisión con dos respuestas: %+v", refusal.Decision)
	}
	wantAbsent(t, filepath.Join(ws.Home, "changes", "x"))
	if branchExists(t, ws.Master, "feat/x") {
		t.Error("decision_required no debía crear la rama")
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
