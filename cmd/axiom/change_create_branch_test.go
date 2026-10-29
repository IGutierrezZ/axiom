package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// REQ-25.32 y decisión T4 (INC-25 S8-c4): `axiom change create` comparte escritor con el dashboard
// (`Service.CreateIncrement`): ejecuta `ensure` y escribe `proposal.md` en el planning home que
// devuelve. Se ejecuta en proceso (`runChange`, sin compilar ningún ejecutable propio) sobre un
// monorepo embebido con git real, entorno aislado, preferencias de worktree en el hub del HOME de
// pruebas y la raíz de worktrees temporal. No usa el subproceso de runAxiom: su TestMain relanzado
// crea otro HOME y no vería las preferencias del hub.

// changeCreateProject convierte root en un repositorio git en `main` con un commit base y le da las
// preferencias de worktree de mode, con la raíz de worktrees en un directorio temporal.
func changeCreateProject(t *testing.T, root, mode string) {
	t.Helper()
	requireRealGit(t)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Autor de Prueba")
	t.Setenv("GIT_AUTHOR_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Autor de Prueba")
	t.Setenv("GIT_COMMITTER_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	handoffGuardGit(t, root, "init", "-q")
	handoffGuardGit(t, root, "symbolic-ref", "HEAD", "refs/heads/main")
	handoffGuardGit(t, root, "config", "core.autocrlf", "false")
	handoffGuardGit(t, root, "add", "-A")
	handoffGuardGit(t, root, "commit", "-q", "--allow-empty", "-m", "base")

	manager, err := hub.NewManager("")
	if err != nil {
		t.Fatalf("hub: %v", err)
	}
	if _, err := manager.Register(root, "", ""); err != nil {
		t.Fatalf("registrar %s en el hub: %v", root, err)
	}
	prefs := hub.WorktreePrefs{Mode: mode, Root: filepath.Join(t.TempDir(), "worktrees")}
	if _, err := manager.SetWorktreePrefs(root, prefs); err != nil {
		t.Fatalf("preferencias de worktree de %s: %v", root, err)
	}
}

// changeCreate ejecuta `axiom change create` en proceso y devuelve su salida combinada y su cÃ³digo.
func changeCreate(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runChange(append([]string{"create"}, args...), &stdout, &stderr)
	return stdout.String() + stderr.String(), code
}

func changeCreateGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
	return strings.TrimRight(string(output), "\r\n")
}

// changeCreateWorktreeOf devuelve el worktree donde está desplegada la rama, o falla.
func changeCreateWorktreeOf(t *testing.T, repo, name string) string {
	t.Helper()
	var current string
	for _, line := range strings.Split(changeCreateGit(t, repo, "worktree", "list", "--porcelain"), "\n") {
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

// REQ-25.32 escenario 1, vía CLI: desde `main`, `axiom change create x` crea la rama `feat/x` y su
// worktree y escribe `proposal.md` en el `openspec/changes/x/` del worktree; el checkout principal
// no recibe la carpeta del cambio.
func TestChangeCreateRunsEnsureAndWritesInTheWorktree(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	changeCreateProject(t, ws.Master, hub.WorktreeModeWorktree)

	output, code := changeCreate(t, "x", "--intent", "Propósito X", "--cwd", ws.Master)

	if code != 0 {
		t.Fatalf("change create salió con %d:\n%s", code, output)
	}
	t.Logf("salida de change create (código %d):\n%s", code, output)
	if changeCreateGit(t, ws.Master, "branch", "--list", "feat/x") == "" {
		t.Fatal("ensure debía crear la rama feat/x")
	}
	worktree := changeCreateWorktreeOf(t, ws.Master, "feat/x")
	if pathidentity.SameDirectory(worktree, ws.Master) {
		t.Fatalf("feat/x no debía desplegarse en el checkout principal %s", ws.Master)
	}
	proposal := filepath.Join(worktree, "openspec", "changes", "x", "proposal.md")
	data, err := os.ReadFile(proposal)
	if err != nil {
		t.Fatalf("proposal.md debía escribirse en el worktree: %v", err)
	}
	if !strings.Contains(string(data), "Propósito X") {
		t.Errorf("proposal.md sin el propósito:\n%s", data)
	}
	// La salida cita la ruta de proposal.md en el worktree (bajo la raíz de worktrees del hub); se
	// compara por un tramo de la ruta porque la salida puede llevar el nombre corto 8.3 de Windows.
	if !strings.Contains(output, "creado correctamente") || !strings.Contains(output, filepath.Join("worktrees", "x")) {
		t.Errorf("la salida debe citar la ruta del worktree:\n%s", output)
	}

	wantFileAbsent(t, filepath.Join(ws.Home, "changes", "x"))
	if got := changeCreateGit(t, ws.Master, "branch", "--show-current"); got != "main" {
		t.Errorf("el checkout principal debía seguir en main, está en %q", got)
	}
	if status := changeCreateGit(t, ws.Master, "status", "--porcelain"); status != "" {
		t.Errorf("el checkout principal debía quedar limpio:\n%s", status)
	}
}

// REQ-25.32 escenario 4, vía CLI: un rechazo de `ensure` (árbol sucio en `in_place`) sale con 1, nombra
// la continuación y no deja carpeta ni rama.
func TestChangeCreateRefusedEnsureLeavesNoFolder(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	changeCreateProject(t, ws.Master, hub.WorktreeModeInPlace)
	if err := os.WriteFile(filepath.Join(ws.Master, "stray.txt"), []byte("sin confirmar\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, code := changeCreate(t, "x", "--cwd", ws.Master)

	if code != 1 {
		t.Fatalf("change create con el árbol sucio salió con %d, se esperaba 1:\n%s", code, output)
	}
	t.Logf("salida de change create con rechazo (código %d):\n%s", code, output)
	if !strings.Contains(output, "axiom branch") {
		t.Errorf("la salida debe llevar la continuación de ensure:\n%s", output)
	}
	wantFileAbsent(t, filepath.Join(ws.Home, "changes", "x"))
	if changeCreateGit(t, ws.Master, "branch", "--list", "feat/x") != "" {
		t.Error("un rechazo de ensure no debía dejar la rama feat/x")
	}
}
