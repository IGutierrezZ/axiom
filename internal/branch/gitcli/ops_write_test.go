package gitcli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// Pruebas de las escrituras de `ensure` (INC-25, S3.4): CreateBranch y AddWorktree. Son
// las dos únicas escrituras de esta rebanada y se lanzan siempre con cmd.Dir en el
// checkout principal que recibe quien llama (REQ-25.8, matriz de amenazas «Selección de
// repositorio git»).

// remoteOperationFor dice si una invocación de git es de las remotas (REQ-25.1).
func remoteOperationFor(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "fetch", "pull", "push", "ls-remote":
			return true
		}
	}
	return false
}

// CreateBranch crea la rama desde el punto de partida sin configurar ningún seguimiento:
// con la referencia remota local como punto de partida, git configuraría un upstream y
// `axiom branch` nunca publica ni sigue una rama remota.
func TestCreateBranchCreatesTheBranchAtTheStartPointWithoutTracking(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	first := runGit(t, repo, "rev-parse", "HEAD")
	commit(t, repo, "segundo")
	remote := t.TempDir()
	runGit(t, remote, "init", "-q", "--bare", "-b", "main")
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-q", "origin", "main")
	git := New()

	for name, tc := range map[string]struct{ branch, start, want string }{
		"desde un commit":                  {"feat/a", first, first},
		"desde una rama local completa":    {"feat/b", "refs/heads/main", runGit(t, repo, "rev-parse", "main")},
		"desde la referencia remota local": {"feat/c", "refs/remotes/origin/main", runGit(t, repo, "rev-parse", "origin/main")},
	} {
		t.Run(name, func(t *testing.T) {
			if err := git.CreateBranch(ctx, repo, tc.branch, tc.start); err != nil {
				t.Fatalf("CreateBranch: %v", err)
			}
			if got := runGit(t, repo, "rev-parse", "refs/heads/"+tc.branch); got != tc.want {
				t.Errorf("la rama %s apunta a %s, se esperaba %s", tc.branch, got, tc.want)
			}
			if upstream := runGit(t, repo, "for-each-ref", "--format=%(upstream)", "refs/heads/"+tc.branch); upstream != "" {
				t.Errorf("la rama %s no debe configurar seguimiento: %s", tc.branch, upstream)
			}
		})
	}
}

// Una rama que ya existe no se mueve ni se reinicia: CreateBranch falla y la deja donde
// estaba (REQ-25.7).
func TestCreateBranchNeverMovesAnExistingBranch(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	runGit(t, repo, "branch", "feat/x")
	before := runGit(t, repo, "rev-parse", "refs/heads/feat/x")
	commit(t, repo, "segundo")

	err := New().CreateBranch(ctx, repo, "feat/x", runGit(t, repo, "rev-parse", "HEAD"))

	if err == nil {
		t.Fatal("crear una rama que existe debía fallar")
	}
	if !strings.Contains(err.Error(), "feat/x") {
		t.Errorf("el error %q debe nombrar la rama", err)
	}
	if after := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); after != before {
		t.Errorf("la rama se movió de %s a %s", before, after)
	}
}

// AddWorktree despliega una rama que existe en una ruta nueva, con cmd.Dir en el checkout
// principal: no cambia el HEAD ni la rama del principal, no mueve la rama y admite rutas
// con espacios.
func TestAddWorktreeChecksOutAnExistingBranchAtThePath(t *testing.T) {
	ctx := context.Background()
	main := newRepository(t)
	runGit(t, main, "branch", "feat/x")
	branchSHA := runGit(t, main, "rev-parse", "refs/heads/feat/x")
	path := filepath.Join(t.TempDir(), "con espacio", "x", "repo")
	seen := observe(t)

	if err := New().AddWorktree(ctx, main, path, "feat/x"); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	launched := append([][]string(nil), *seen...)

	entries, err := New().Worktrees(ctx, main)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[1].Branch != "feat/x" || entries[1].Detached {
		t.Fatalf("worktrees = %+v, se esperaba el principal y uno enlazado en feat/x", entries)
	}
	wantSameDirectory(t, "worktree creado", entries[1].Path, path)
	if got := runGit(t, main, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Errorf("el checkout principal pasó a %s", got)
	}
	if got := runGit(t, main, "rev-parse", "refs/heads/feat/x"); got != branchSHA {
		t.Errorf("la rama se movió de %s a %s", branchSHA, got)
	}
	if len(launched) != 1 || launched[0][0] != main {
		t.Errorf("la escritura debía lanzarse una vez desde el checkout principal %s: %v", main, launched)
	}
}

// AddWorktree nunca crea una rama: con una que no existe falla y no deja nada.
func TestAddWorktreeNeverCreatesTheBranch(t *testing.T) {
	ctx := context.Background()
	main := newRepository(t)
	path := filepath.Join(t.TempDir(), "x")

	if err := New().AddWorktree(ctx, main, path, "feat/no-existe"); err == nil {
		t.Fatal("desplegar una rama que no existe debía fallar")
	}

	if branches := runGit(t, main, "branch", "--list", "feat/no-existe"); branches != "" {
		t.Errorf("AddWorktree creó la rama: %q", branches)
	}
	if entries, _ := New().Worktrees(ctx, main); len(entries) != 1 {
		t.Errorf("worktrees = %+v, se esperaba solo el principal", entries)
	}
}

// Las escrituras rechazan, antes de lanzar nada, los argumentos que git leería como una
// opción y las rutas que no son absolutas.
func TestWritesRejectArgumentsThatLookLikeOptions(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	runGit(t, repo, "branch", "feat/x")
	seen := observe(t)
	git := New()
	absolute := filepath.Join(t.TempDir(), "x")

	for name, err := range map[string]error{
		"nombre de rama con guion":            git.CreateBranch(ctx, repo, "-D", "refs/heads/main"),
		"nombre de rama vacío":                git.CreateBranch(ctx, repo, "", "refs/heads/main"),
		"punto de partida con guion":          git.CreateBranch(ctx, repo, "feat/y", "--all"),
		"punto de partida vacío":              git.CreateBranch(ctx, repo, "feat/y", ""),
		"worktree con ruta relativa":          git.AddWorktree(ctx, repo, "relativa", "feat/x"),
		"worktree con ruta vacía":             git.AddWorktree(ctx, repo, "", "feat/x"),
		"worktree con rama que parece opción": git.AddWorktree(ctx, repo, absolute, "-b"),
		"worktree con rama vacía":             git.AddWorktree(ctx, repo, absolute, ""),
	} {
		if err == nil {
			t.Errorf("%s: debía rechazarse", name)
		}
	}
	if len(*seen) != 0 {
		t.Errorf("el rechazo es previo a lanzar git, pero el observador vio %v", *seen)
	}
}

// REQ-25.1 y G-7: las escrituras lanzan solo `git branch` y `git worktree add`, y
// ninguna es una operación remota ni publica nada.
func TestWritesLaunchOnlyBranchAndWorktreeAdd(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	runGit(t, repo, "remote", "add", "origin", "https://ejemplo.invalid/repo.git")
	seen := observe(t)
	git := New()

	if err := git.CreateBranch(ctx, repo, "feat/x", "refs/heads/main"); err != nil {
		t.Fatal(err)
	}
	if err := git.AddWorktree(ctx, repo, filepath.Join(t.TempDir(), "x"), "feat/x"); err != nil {
		t.Fatal(err)
	}

	if len(*seen) != 2 {
		t.Fatalf("el observador vio %d invocaciones, se esperaban 2: %v", len(*seen), *seen)
	}
	if (*seen)[0][1] != "branch" || (*seen)[1][1] != "worktree" || (*seen)[1][2] != "add" {
		t.Errorf("las escrituras deben ser `git branch` y `git worktree add`: %v", *seen)
	}
	for _, invocation := range *seen {
		if remoteOperationFor(invocation[1:]) {
			t.Errorf("una escritura lanzó una operación remota: %v", invocation)
		}
	}
}

// Fetch (INC-25, S3.11) es la única operación remota del puerto: `git fetch origin` desde el
// directorio que recibe, sin `--prune`, y solo trae referencias: ni mueve la rama actual ni
// toca el árbol de trabajo.
func TestFetchBringsTheRemoteCommitsLaunchingOnlyFetchOrigin(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	remote := t.TempDir()
	runGit(t, remote, "init", "-q", "--bare", "-b", "main")
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-q", "origin", "main")
	scratch := t.TempDir()
	clone := filepath.Join(scratch, "clon")
	runGit(t, scratch, "clone", "-q", remote, clone)
	commit(t, clone, "avance del remoto")
	runGit(t, clone, "push", "-q", "origin", "main")
	advanced := runGit(t, clone, "rev-parse", "HEAD")
	local := runGit(t, repo, "rev-parse", "main")
	seen := observe(t)

	if err := New().Fetch(ctx, repo, "origin"); err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if got := runGit(t, repo, "rev-parse", "refs/remotes/origin/main"); got != advanced {
		t.Errorf("origin/main apunta a %s, se esperaba el commit del remoto %s", got, advanced)
	}
	if got := runGit(t, repo, "rev-parse", "main"); got != local {
		t.Errorf("fetch no mueve la rama local: main pasó de %s a %s", local, got)
	}
	if len(*seen) != 1 || (*seen)[0][0] != repo || strings.Join((*seen)[0][1:], " ") != "fetch origin" {
		t.Errorf("se esperaba una única invocación `git fetch origin` en %s, el observador vio %v", repo, *seen)
	}
}

// Un fetch que falla devuelve un error en castellano que nombra el comando, el directorio y
// el resumen de git: un remoto que no existe y un repositorio sin `origin`.
func TestFetchFailureNamesTheCommandAndTheDirectory(t *testing.T) {
	ctx := context.Background()
	t.Run("remoto que no existe", func(t *testing.T) {
		repo := newRepository(t)
		runGit(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "no-existe.git"))

		err := New().Fetch(ctx, repo, "origin")

		if err == nil || !strings.Contains(err.Error(), "git fetch origin") || !strings.Contains(err.Error(), repo) {
			t.Errorf("error = %v; debía nombrar `git fetch origin` y %s", err, repo)
		}
	})
	t.Run("sin remoto origin", func(t *testing.T) {
		repo := newRepository(t)

		err := New().Fetch(ctx, repo, "origin")

		if err == nil || !strings.Contains(err.Error(), "git fetch origin") {
			t.Errorf("error = %v; debía nombrar `git fetch origin`", err)
		}
	})
}

// Fetch rechaza, antes de lanzar nada, un remoto que git leería como una opción o vacío.
func TestFetchRejectsARemoteThatLooksLikeAnOption(t *testing.T) {
	repo := newRepository(t)
	seen := observe(t)

	for name, remote := range map[string]string{"opción": "--all", "guion": "-v", "vacío": ""} {
		if err := New().Fetch(context.Background(), repo, remote); err == nil {
			t.Errorf("%s: debía rechazarse", name)
		}
	}
	if len(*seen) != 0 {
		t.Errorf("el rechazo es previo a lanzar git, pero el observador vio %v", *seen)
	}
}
