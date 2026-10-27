package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// REQ-25.19 (guarda A): `axiom handoff create` escribe en la carpeta del cambio y por eso consulta
// el veredicto de rama antes de `handoff.WriteFile` (contexto `ordinary`, design §2.11). Se ejecuta
// como subproceso con el patrón runAxiom (el binario de `go test` relanzado; sin compilar ningún
// ejecutable propio) sobre un monorepo embebido con git real, entorno aislado y sin red.

const handoffGuardChange = "x"

func handoffGuardGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
}

// handoffGuardProject es un monorepo embebido con git en `main` y el cambio `x` sellado desde
// `feat/x` (con instantánea de política git); el checkout vuelve a `main`.
func handoffGuardProject(t *testing.T) workspacetest.Workspace {
	t.Helper()
	requireRealGit(t)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Autor de Prueba")
	t.Setenv("GIT_AUTHOR_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Autor de Prueba")
	t.Setenv("GIT_COMMITTER_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", handoffGuardChange, "proposal.md"), "# x\n")
	handoffGuardGit(t, ws.Master, "init", "-q")
	handoffGuardGit(t, ws.Master, "symbolic-ref", "HEAD", "refs/heads/main")
	handoffGuardGit(t, ws.Master, "config", "core.autocrlf", "false")
	handoffGuardGit(t, ws.Master, "add", "-A")
	handoffGuardGit(t, ws.Master, "commit", "-q", "-m", "base")

	handoffGuardGit(t, ws.Master, "switch", "-q", "-c", "feat/x")
	var sealed bytes.Buffer
	if err := cli.RunSDDKickoff([]string{"seal", "--cwd", ws.Master, "--change", handoffGuardChange, "--execution-style", "continuous", "--handoff-policy", "none"}, &sealed); err != nil {
		t.Fatalf("sellar en feat/x: %v", err)
	}
	handoffGuardGit(t, ws.Master, "switch", "-q", "main")
	return ws
}

func handoffCreate(t *testing.T, ws workspacetest.Workspace, change string) (string, int) {
	t.Helper()
	return runAxiom(t, "handoff", "create", "--change", change,
		"--from", "spec", "--to", "design", "--from-role", "backend", "--to-role", "frontend",
		"--path", ws.Master)
}

func wantFileAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s no debía existir (%v)", path, err)
	}
}

// Escenario «Cada escritor se niega fuera de rama»: desde `main`, `handoff create` sale con 1,
// nombra `wrong_branch` y la orden exacta de `ensure` con la carpeta maestra, y no escribe nada.
func TestHandoffCreateRefusesOutsideTheChangeBranch(t *testing.T) {
	ws := handoffGuardProject(t)

	output, code := handoffCreate(t, ws, handoffGuardChange)

	if code != 1 {
		t.Fatalf("handoff create desde main salió con %d, se esperaba 1:\n%s", code, output)
	}
	t.Logf("salida de handoff create desde main (código %d):\n%s", code, output)
	for _, want := range []string{"[ERROR]", "wrong_branch", "«main»", "«feat/x»", `axiom branch ensure --cwd "` + ws.Master + `" --change x`} {
		if !strings.Contains(output, want) {
			t.Errorf("la salida debe contener %q:\n%s", want, output)
		}
	}
	wantFileAbsent(t, filepath.Join(ws.Home, "changes", handoffGuardChange, "handoff.md"))
}

// Escenario «Cada escritor escribe en la rama correcta»: desde `feat/x` el relevo se escribe.
func TestHandoffCreateWritesOnTheChangeBranch(t *testing.T) {
	ws := handoffGuardProject(t)
	handoffGuardGit(t, ws.Master, "switch", "-q", "feat/x")

	output, code := handoffCreate(t, ws, handoffGuardChange)

	if code != 0 {
		t.Fatalf("handoff create desde feat/x salió con %d:\n%s", code, output)
	}
	if _, err := os.Stat(filepath.Join(ws.Home, "changes", handoffGuardChange, "handoff.md")); err != nil {
		t.Errorf("el relevo debe escribirse: %v", err)
	}
}

// Escenario «Un sumidero acumulativo está exento por nombre»: `cumulative-<rol>` escribe desde `main`.
func TestHandoffCreateOnACumulativeSinkIsExempt(t *testing.T) {
	ws := handoffGuardProject(t)

	output, code := handoffCreate(t, ws, "cumulative-core")

	if code != 0 {
		t.Fatalf("handoff create sobre cumulative-core desde main salió con %d:\n%s", code, output)
	}
	if _, err := os.Stat(filepath.Join(ws.Home, "changes", "cumulative-core", "handoff.md")); err != nil {
		t.Errorf("el relevo del sumidero debe escribirse: %v", err)
	}
}
