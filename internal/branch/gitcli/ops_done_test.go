package gitcli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Pruebas de las operaciones de `done` (INC-25, S6.4, REQ-25.16, REQ-25.17): IsAncestor, la capa 1
// de la detección de fusión, y las dos escrituras del desmontaje, RemoveWorktree y DeleteBranch.
// Ninguna es remota y las escrituras se lanzan siempre con cmd.Dir en el checkout principal.

// IsAncestor responde por las dos direcciones, cuenta el propio commit como ancestro y no
// confunde una rama divergente con una fusionada.
func TestIsAncestorAnswersTheAncestryOfTwoBranches(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	runGit(t, repo, "branch", "feat/base")
	commit(t, repo, "avance de main")
	runGit(t, repo, "switch", "-q", "-c", "feat/otra", "feat/base")
	commit(t, repo, "commit propio")
	runGit(t, repo, "switch", "-q", "main")
	git := New()

	for name, tc := range map[string]struct {
		ancestor, descendant string
		want                 bool
	}{
		"un ancestro de main":                   {"refs/heads/feat/base", "refs/heads/main", true},
		"el propio commit":                      {"refs/heads/main", "refs/heads/main", true},
		"main no es ancestro de su ancestro":    {"refs/heads/main", "refs/heads/feat/base", false},
		"una rama divergente no está fusionada": {"refs/heads/feat/otra", "refs/heads/main", false},
		"main no es ancestro de la divergente":  {"refs/heads/main", "refs/heads/feat/otra", false},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := git.IsAncestor(ctx, repo, tc.ancestor, tc.descendant)

			if err != nil || got != tc.want {
				t.Errorf("IsAncestor(%s, %s) = (%v, %v), se esperaba (%v, nil)", tc.ancestor, tc.descendant, got, err, tc.want)
			}
		})
	}
}

// Una referencia que git no resuelve es un error y no un «no es ancestro»: darla por no fusionada
// escondería un fallo detrás de un rechazo de `done`. Una referencia con forma de opción se rechaza
// antes de lanzar nada.
func TestIsAncestorReportsAnUnresolvableReferenceAsAnError(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	seen := observe(t)

	if _, err := New().IsAncestor(ctx, repo, "refs/heads/no-existe", "refs/heads/main"); err == nil || !strings.Contains(err.Error(), "merge-base") {
		t.Errorf("una referencia inexistente debe ser un error que nombre el comando: %v", err)
	}
	launched := len(*seen)
	for _, option := range []string{"--all", "-x", ""} {
		if _, err := New().IsAncestor(ctx, repo, option, "refs/heads/main"); err == nil {
			t.Errorf("%q no es una referencia válida y debía rechazarse", option)
		}
		if _, err := New().IsAncestor(ctx, repo, "refs/heads/main", option); err == nil {
			t.Errorf("%q no es una referencia válida y debía rechazarse", option)
		}
	}
	if len(*seen) != launched {
		t.Errorf("las referencias inválidas no lanzan git: %v", (*seen)[launched:])
	}
}

// RemoveWorktree retira un worktree limpio (el directorio y su registro en git) desde el checkout
// principal, sin tocar la rama ni lanzar ninguna operación remota: el único subcomando es
// `worktree`.
func TestRemoveWorktreeRemovesACleanWorktreeButKeepsTheBranch(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	worktree := filepath.Join(t.TempDir(), "x", "repo")
	runGit(t, repo, "branch", "feat/x")
	if err := New().AddWorktree(ctx, repo, worktree, "feat/x"); err != nil {
		t.Fatal(err)
	}
	seen := observe(t)

	if err := New().RemoveWorktree(ctx, repo, worktree, false); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}

	if _, err := os.Stat(worktree); !os.IsNotExist(err) {
		t.Errorf("el directorio del worktree debía desaparecer: %v", err)
	}
	if listed := runGit(t, repo, "worktree", "list", "--porcelain"); strings.Contains(listed, "feat/x") {
		t.Errorf("git no debe seguir conociendo el worktree:\n%s", listed)
	}
	if runGit(t, repo, "branch", "--list", "feat/x") == "" {
		t.Error("retirar el worktree no borra la rama")
	}
	if len(*seen) != 1 || (*seen)[0][0] != repo || (*seen)[0][1] != "worktree" || (*seen)[0][2] != "remove" || remoteOperationFor((*seen)[0][1:]) {
		t.Errorf("la escritura debe ser un único `git worktree remove` desde el checkout principal: %v", *seen)
	}
}

// Sin force, un worktree con cambios sin confirmar no se retira y no se pierde nada; con force
// se descartan. Un worktree no se retira con una ruta relativa.
func TestRemoveWorktreeRefusesUncommittedChangesUnlessForced(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	worktree := filepath.Join(t.TempDir(), "x", "repo")
	runGit(t, repo, "branch", "feat/x")
	if err := New().AddWorktree(ctx, repo, worktree, "feat/x"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(worktree, "suelto.txt"), "sin añadir\n")

	err := New().RemoveWorktree(ctx, repo, worktree, false)

	if err == nil || !strings.Contains(err.Error(), "worktree remove") {
		t.Fatalf("un worktree sucio no se retira sin force y el error nombra el comando: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(worktree, "suelto.txt")); statErr != nil {
		t.Errorf("el rechazo no debe perder el fichero sin confirmar: %v", statErr)
	}
	if err := New().RemoveWorktree(ctx, repo, "relativa/x", false); err == nil || !strings.Contains(err.Error(), "absoluta") {
		t.Errorf("una ruta relativa se rechaza antes de lanzar git: %v", err)
	}

	if err := New().RemoveWorktree(ctx, repo, worktree, true); err != nil {
		t.Fatalf("con force se retira aunque esté sucio: %v", err)
	}
	if _, err := os.Stat(worktree); !os.IsNotExist(err) {
		t.Errorf("el directorio del worktree debía desaparecer: %v", err)
	}
}

// DeleteBranch elimina la rama local aunque git no la considere fusionada (tras un squash), solo
// toca `refs/heads` y no lanza ninguna operación remota: la referencia remota local y el remoto
// quedan intactos.
func TestDeleteBranchDeletesAnUnmergedLocalBranchAndNeverARemoteOne(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	remote := t.TempDir()
	runGit(t, remote, "init", "-q", "--bare", "-b", "main")
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	commit(t, repo, "commit propio sin fusionar")
	runGit(t, repo, "push", "-q", "origin", "feat/x")
	runGit(t, repo, "switch", "-q", "main")
	tracking := runGit(t, repo, "rev-parse", "refs/remotes/origin/feat/x")
	seen := observe(t)

	if err := New().DeleteBranch(ctx, repo, "feat/x"); err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}

	if runGit(t, repo, "branch", "--list", "feat/x") != "" {
		t.Error("la rama local debía eliminarse aunque no esté fusionada")
	}
	if got := runGit(t, repo, "rev-parse", "refs/remotes/origin/feat/x"); got != tracking {
		t.Errorf("la referencia remota local cambió de %s a %s", tracking, got)
	}
	if listed := runGit(t, remote, "branch", "--list", "feat/x"); listed == "" {
		t.Error("la rama del remoto debía quedar intacta")
	}
	if len(*seen) != 1 || (*seen)[0][1] != "branch" || (*seen)[0][2] != "-D" || remoteOperationFor((*seen)[0][1:]) {
		t.Errorf("la escritura debe ser un único `git branch -D`, sin operaciones remotas: %v", *seen)
	}
}

// DeleteBranch no puede eliminar una rama desplegada en un worktree: git se niega y la rama queda
// donde estaba, de modo que `done` retira siempre el worktree antes. Un nombre con forma de opción
// o vacío se rechaza antes de lanzar nada.
func TestDeleteBranchRefusesABranchDeployedInAWorktreeAndOptionLikeNames(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	worktree := filepath.Join(t.TempDir(), "x", "repo")
	runGit(t, repo, "branch", "feat/x")
	if err := New().AddWorktree(ctx, repo, worktree, "feat/x"); err != nil {
		t.Fatal(err)
	}

	if err := New().DeleteBranch(ctx, repo, "feat/x"); err == nil || !strings.Contains(err.Error(), "branch -D") {
		t.Errorf("una rama desplegada no se elimina y el error nombra el comando: %v", err)
	}
	if runGit(t, repo, "branch", "--list", "feat/x") == "" {
		t.Error("el rechazo no debe eliminar la rama")
	}
	seen := observe(t)
	for _, name := range []string{"", "-D", "--all"} {
		if err := New().DeleteBranch(ctx, repo, name); err == nil {
			t.Errorf("%q no es un nombre de rama válido y debía rechazarse", name)
		}
	}
	if len(*seen) != 0 {
		t.Errorf("los nombres inválidos no lanzan git: %v", *seen)
	}
}
