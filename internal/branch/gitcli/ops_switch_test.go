package gitcli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Pruebas de Switch (INC-25, S4.7): el cambio de rama de `ensure` en modo `in_place`
// (REQ-25.8, REQ-25.9). Se lanza con cmd.Dir en el propio repositorio y solo cambia de rama:
// nunca crea la rama, ni la adivina desde una remota, ni toca nada si git se niega.

// Switch despliega en el repositorio una rama local existente y no lanza ninguna operación
// remota: el único subcomando de git es `switch`.
func TestSwitchDeploysAnExistingLocalBranchLaunchingOnlySwitch(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	runGit(t, repo, "branch", "feat/x")
	seen := observe(t)

	if err := New().Switch(ctx, repo, "feat/x"); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	if got := runGit(t, repo, "symbolic-ref", "--short", "HEAD"); got != "feat/x" {
		t.Errorf("el repositorio está en %s, se esperaba feat/x", got)
	}
	if len(*seen) != 1 || (*seen)[0][1] != "switch" || remoteOperationFor((*seen)[0][1:]) {
		t.Errorf("la escritura debe ser un único `git switch`, sin operaciones remotas: %v", *seen)
	}
}

// Switch nunca crea la rama: si no existe, git falla y el repositorio queda donde estaba.
// Tampoco la adivina desde una referencia remota local del mismo nombre (`--no-guess`): eso
// crearía una rama que sigue a un remoto, y `axiom branch` nunca sigue ni publica.
func TestSwitchNeverCreatesOrGuessesTheBranch(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	remote := t.TempDir()
	runGit(t, remote, "init", "-q", "--bare", "-b", "main")
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-q", "origin", "main:feat/remota")
	runGit(t, repo, "fetch", "-q", "origin")
	git := New()

	for name, branch := range map[string]string{"que no existe": "feat/no-existe", "que solo existe en la remota": "feat/remota"} {
		t.Run(name, func(t *testing.T) {
			err := git.Switch(ctx, repo, branch)

			if err == nil {
				t.Fatalf("cambiar a la rama %s, que no existe en local, debía fallar", branch)
			}
			if !strings.Contains(err.Error(), "switch") {
				t.Errorf("el error %q debe nombrar el comando de git", err)
			}
			if got := runGit(t, repo, "symbolic-ref", "--short", "HEAD"); got != "main" {
				t.Errorf("el repositorio está en %s, se esperaba main", got)
			}
			if runGit(t, repo, "branch", "--list", branch) != "" {
				t.Errorf("Switch no debe crear la rama %s", branch)
			}
		})
	}
}

// Switch no sobrescribe cambios sin confirmar: con un conflicto, git se niega y el fichero
// conserva su contenido (la comprobación de suciedad de `ensure` es previa, esta es la red de
// seguridad).
func TestSwitchRefusesToOverwriteLocalChanges(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	writeFile(t, filepath.Join(repo, "datos.txt"), "base\n")
	runGit(t, repo, "add", "datos.txt")
	commit(t, repo, "datos")
	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	writeFile(t, filepath.Join(repo, "datos.txt"), "en la rama\n")
	runGit(t, repo, "commit", "-q", "-am", "cambia datos")
	runGit(t, repo, "switch", "-q", "main")
	writeFile(t, filepath.Join(repo, "datos.txt"), "sin confirmar\n")

	err := New().Switch(ctx, repo, "feat/x")

	if err == nil {
		t.Fatal("cambiar de rama con un cambio que se sobrescribiría debía fallar")
	}
	if got := runGit(t, repo, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Errorf("el repositorio está en %s, se esperaba main", got)
	}
	if content, readErr := os.ReadFile(filepath.Join(repo, "datos.txt")); readErr != nil || string(content) != "sin confirmar\n" {
		t.Errorf("el fichero perdió su contenido (%v): %q", readErr, content)
	}
}

// Un nombre que git leería como opción, o vacío, se rechaza antes de lanzar nada.
func TestSwitchRejectsABranchThatLooksLikeAnOptionBeforeLaunchingGit(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	seen := observe(t)

	for name, branch := range map[string]string{"con guion": "-c", "vacío": ""} {
		if err := New().Switch(ctx, repo, branch); err == nil {
			t.Errorf("%s: debía rechazarse", name)
		}
	}
	if len(*seen) != 0 {
		t.Errorf("el rechazo es previo a lanzar git, pero el observador vio %v", *seen)
	}
}
