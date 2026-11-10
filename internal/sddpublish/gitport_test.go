package sddpublish

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// Pruebas del adaptador del puerto Git (DD-08, REQ-27.6, REQ-27.9, REQ-27.11) con git real,
// repositorios y remotos desnudos temporales y sin red.

func TestGitPortRemoteAndResolveCommit(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()
	repo, _ := newRepoWithRemote(t)
	port := NewGit()

	if ok, err := port.RemoteConfigured(ctx, repo, "origin"); err != nil || !ok {
		t.Errorf("RemoteConfigured(origin) = %v, %v; se esperaba true", ok, err)
	}
	if ok, err := port.RemoteConfigured(ctx, repo, "otro"); err != nil || ok {
		t.Errorf("RemoteConfigured(otro) = %v, %v; se esperaba false", ok, err)
	}
	head := runGit(t, repo, "rev-parse", "HEAD")
	if sha, ok, err := port.ResolveCommit(ctx, repo, "HEAD"); err != nil || !ok || sha != head {
		t.Errorf("ResolveCommit(HEAD) = %q, %v, %v; se esperaba %s", sha, ok, err, head)
	}
	if sha, ok, err := port.ResolveCommit(ctx, repo, "refs/remotes/origin/main"); err != nil || !ok || sha != head {
		t.Errorf("ResolveCommit(origin/main) = %q, %v, %v", sha, ok, err)
	}
	if _, ok, err := port.ResolveCommit(ctx, repo, "refs/remotes/origin/inexistente"); err != nil || ok {
		t.Errorf("ResolveCommit de una referencia inexistente = %v, %v; se esperaba false sin error", ok, err)
	}
}

// Ningún valor externo llega a git como opción: una referencia, un remoto, una rama o una ruta
// que git leería como opción se rechazan antes de ejecutar nada.
func TestGitPortAdapterRejectsArgumentsGitCouldReadAsOptions(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()
	repo, _ := newRepoWithRemote(t)
	port := NewGit()
	seen := observe(t)

	checks := map[string]func() error{
		"remoto":          func() error { _, err := port.RemoteConfigured(ctx, repo, "--upload-pack=x"); return err },
		"referencia":      func() error { _, _, err := port.ResolveCommit(ctx, repo, "--help"); return err },
		"exclusión":       func() error { _, err := port.UnpublishedCommits(ctx, repo, "HEAD", []string{"-x"}, 5); return err },
		"cabeza":          func() error { _, err := port.UnpublishedCommits(ctx, repo, "--all", nil, 5); return err },
		"rama del push":   func() error { _, err := port.Push(ctx, repo, "origin", "--delete", false); return err },
		"remoto del push": func() error { _, err := port.Push(ctx, repo, "--mirror", "main", false); return err },
		"ruta absoluta":   func() error { _, err := port.DirtyPaths(ctx, repo, []string{"/etc"}); return err },
		"ruta con ..":     func() error { _, err := port.DirtyPaths(ctx, repo, []string{"a/../b"}); return err },
		"ruta vacía":      func() error { _, err := port.Commit(ctx, repo, "m", []string{""}, nil); return err },
		"SHA no hex":      func() error { _, err := port.CommitPaths(ctx, repo, "--root"); return err },
		"hook inválido":   func() error { _, err := port.HookPresent(ctx, repo, "../x"); return err },
	}
	for name, call := range checks {
		if call() == nil {
			t.Errorf("%s: se esperaba un rechazo", name)
		}
	}
	if len(*seen) != 0 {
		t.Errorf("los argumentos inválidos no deben llegar a git, se ejecutó %v", *seen)
	}
}

// Matriz «Estado del commit»: DirtyPaths incluye modificados, borrados, preparados y sin
// seguimiento fichero a fichero, no sigue los renombrados y cita las rutas como literales.
func TestGitPortDirtyPathsIncludesDeletionsAndDoesNotFollowRenames(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()
	repo := newRepo(t)
	writeFile(t, filepath.Join(repo, "docs", "a.md"), "a\n")
	writeFile(t, filepath.Join(repo, "docs", "b.md"), "b\n")
	writeFile(t, filepath.Join(repo, "docs", "viejo.md"), "contenido largo para detectar el renombrado\n")
	writeFile(t, filepath.Join(repo, "docs", "[x]", "c.md"), "c\n")
	writeFile(t, filepath.Join(repo, "otro", "z.md"), "z\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "docs")
	port := NewGit()

	writeFile(t, filepath.Join(repo, "docs", "a.md"), "editado\n")         // modificado
	if err := os.Remove(filepath.Join(repo, "docs", "b.md")); err != nil { // borrado
		t.Fatal(err)
	}
	runGit(t, repo, "mv", "docs/viejo.md", "docs/nuevo.md")               // renombrado y preparado
	writeFile(t, filepath.Join(repo, "docs", "nueva", "d.md"), "d\n")     // sin seguimiento, dentro de una carpeta nueva
	writeFile(t, filepath.Join(repo, "docs", "[x]", "c.md"), "editado\n") // ruta con corchetes
	writeFile(t, filepath.Join(repo, "otro", "z.md"), "editado\n")        // fuera de las raíces

	got, err := port.DirtyPaths(ctx, repo, []string{"docs"})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	want := []string{"docs/[x]/c.md", "docs/a.md", "docs/b.md", "docs/nueva/d.md", "docs/nuevo.md", "docs/viejo.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DirtyPaths:\n got %q\nwant %q", got, want)
	}

	// Un patrón no se interpreta: la raíz `docs/[x]` es la carpeta literal de ese nombre.
	literal, err := port.DirtyPaths(ctx, repo, []string{"docs/[x]"})
	if err != nil || !reflect.DeepEqual(literal, []string{"docs/[x]/c.md"}) {
		t.Errorf("DirtyPaths(docs/[x]) = %q, %v", literal, err)
	}
	if none, err := port.DirtyPaths(ctx, repo, []string{"docs/n*"}); err != nil || len(none) != 0 {
		t.Errorf("DirtyPaths(docs/n*) = %q, %v; el asterisco es literal y no coincide con nada", none, err)
	}
}

func TestGitPortUnpublishedCommitsHonoursLimitAndExclusions(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()
	repo, _ := newRepoWithRemote(t)
	port := NewGit()
	var shas []string
	for _, name := range []string{"uno", "dos", "tres"} {
		writeFile(t, filepath.Join(repo, name+".txt"), name+"\n")
		runGit(t, repo, "add", "-A")
		runGit(t, repo, "commit", "-q", "-m", name)
		shas = append(shas, runGit(t, repo, "rev-parse", "HEAD"))
	}

	got, err := port.UnpublishedCommits(ctx, repo, "HEAD", []string{"refs/remotes/origin/main"}, 10)
	if err != nil || !reflect.DeepEqual(got, shas) {
		t.Errorf("UnpublishedCommits = %q, %v; se esperaba %q (del más antiguo al más reciente)", got, err, shas)
	}
	if got, _ := port.UnpublishedCommits(ctx, repo, "HEAD", []string{"refs/remotes/origin/main"}, 2); len(got) != 3 {
		t.Errorf("con límite 2 se devuelven hasta 3 SHA (límite + 1), hay %d: %q", len(got), got)
	}
	// Con dos referencias a excluir, se restan ambas.
	runGit(t, repo, "branch", "integracion", shas[1])
	got, err = port.UnpublishedCommits(ctx, repo, "HEAD", []string{"refs/remotes/origin/main", "refs/heads/integracion"}, 10)
	if err != nil || !reflect.DeepEqual(got, shas[2:]) {
		t.Errorf("con dos exclusiones = %q, %v; se esperaba %q", got, err, shas[2:])
	}
	if _, err := port.UnpublishedCommits(ctx, repo, "HEAD", nil, 0); err == nil {
		t.Error("un límite no positivo debe rechazarse")
	}
}

// El diff combinado (`--cc`) deja de una fusión solo lo que la fusión resolvió por su cuenta.
func TestGitPortCommitPathsOfAMergeUsesTheCombinedDiff(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()
	repo := newRepo(t)
	port := NewGit()

	runGit(t, repo, "switch", "-q", "-c", "lado")
	writeFile(t, filepath.Join(repo, "lado.txt"), "lado\n")
	writeFile(t, filepath.Join(repo, "choque.txt"), "versión del lado\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "lado")
	runGit(t, repo, "switch", "-q", "main")
	writeFile(t, filepath.Join(repo, "principal.txt"), "principal\n")
	writeFile(t, filepath.Join(repo, "choque.txt"), "versión de main\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "principal")
	if out, err := tryGit(repo, "merge", "--no-ff", "-q", "lado", "-m", "fusión"); err == nil {
		t.Fatalf("la fusión debía conflictuar: %s", out)
	}
	writeFile(t, filepath.Join(repo, "choque.txt"), "resuelto a mano\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "--no-edit")
	merge := runGit(t, repo, "rev-parse", "HEAD")

	got, err := port.CommitPaths(ctx, repo, merge)
	if err != nil || !reflect.DeepEqual(got, []string{"choque.txt"}) {
		t.Errorf("CommitPaths de la fusión = %q, %v; se esperaba solo choque.txt", got, err)
	}
	// Un commit ordinario y el commit raíz listan sus rutas.
	writeFile(t, filepath.Join(repo, "x", "y.txt"), "y\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "ordinario")
	if got, err := port.CommitPaths(ctx, repo, runGit(t, repo, "rev-parse", "HEAD")); err != nil || !reflect.DeepEqual(got, []string{"x/y.txt"}) {
		t.Errorf("CommitPaths ordinario = %q, %v", got, err)
	}
	root := runGit(t, repo, "rev-list", "--max-parents=0", "HEAD")
	if got, err := port.CommitPaths(ctx, repo, root); err != nil || !reflect.DeepEqual(got, []string{"README.md"}) {
		t.Errorf("CommitPaths del commit raíz = %q, %v", got, err)
	}
}

// Matriz «Estado del commit»: el código preparado sigue preparado, el no preparado sigue intacto, los
// borrados se confirman y la exclusión deja fuera el estado local.
func TestGitPortCommitConfirmsOnlyTheGivenPaths(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()
	repo := newRepo(t)
	writeFile(t, filepath.Join(repo, "docs", "viejo.md"), "viejo\n")
	writeFile(t, filepath.Join(repo, "src", "a.go"), "package a\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "base")
	port := NewGit()

	writeFile(t, filepath.Join(repo, "docs", "nuevo.md"), "nuevo\n")
	writeFile(t, filepath.Join(repo, "docs", ".gates.lock"), "")
	if err := os.Remove(filepath.Join(repo, "docs", "viejo.md")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "src", "preparado.go"), "package a\n")
	runGit(t, repo, "add", "src/preparado.go")
	writeFile(t, filepath.Join(repo, "src", "a.go"), "package b\n")

	sha, err := port.Commit(ctx, repo, "docs(sdd): publicar los documentos de x", []string{"docs"}, []string{"docs/.*"})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if sha != runGit(t, repo, "rev-parse", "HEAD") {
		t.Errorf("Commit devolvió %s, que no es HEAD", sha)
	}
	touched := strings.Fields(runGit(t, repo, "diff-tree", "-r", "--no-commit-id", "--name-only", "HEAD"))
	slices.Sort(touched)
	if want := []string{"docs/nuevo.md", "docs/viejo.md"}; !reflect.DeepEqual(touched, want) {
		t.Errorf("el commit tocó %q, se esperaba %q (con el borrado y sin el estado local)", touched, want)
	}
	if got := runGit(t, repo, "diff", "--cached", "--name-only"); got != "src/preparado.go" {
		t.Errorf("el código preparado debe seguir preparado, el índice tiene %q", got)
	}
	if got := runGit(t, repo, "diff", "--name-only"); got != "src/a.go" {
		t.Errorf("el código sin preparar debe seguir sin preparar, hay %q", got)
	}
	if got := runGit(t, repo, "status", "--porcelain", "--untracked-files=all", "--", "docs"); got != "?? docs/.gates.lock" {
		t.Errorf("el estado local queda sin seguimiento, hay %q", got)
	}
}

// Un fallo del commit es un *CommandError con el paso y el `stderr`.
func TestGitPortCommitReportsTheFailingStep(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()
	repo := newRepo(t)
	writeFile(t, filepath.Join(repo, "docs", "a.md"), "a\n")
	hook := filepath.Join(repo, ".git", "hooks", "pre-commit")
	writeFile(t, hook, "#!/bin/sh\necho 'rechazado por el hook' >&2\nexit 1\n")
	if err := os.Chmod(hook, 0o755); err != nil {
		t.Fatal(err)
	}
	port := NewGit()

	_, err := port.Commit(ctx, repo, "m", []string{"docs"}, nil)

	var failure *CommandError
	if !errors.As(err, &failure) || failure.Step != "commit" || failure.ExitCode == 0 {
		t.Fatalf("error = %v, se esperaba un *CommandError del paso commit", err)
	}
	if runtime.GOOS != "windows" && !strings.Contains(failure.Stderr, "rechazado por el hook") {
		t.Errorf("stderr = %q, debe conservar lo que dijo el hook", failure.Stderr)
	}
	// Las rutas quedan preparadas: es el estado que deja git.
	if got := runGit(t, repo, "diff", "--cached", "--name-only"); got != "docs/a.md" {
		t.Errorf("tras el fallo del commit las rutas quedan preparadas, el índice tiene %q", got)
	}
}

// Supuesto abierto de design §9: `git rev-parse --git-path hooks/<hook>` respeta `core.hooksPath`.
// Esta prueba lo comprueba con la versión de git instalada: un hook del directorio de hooks por
// defecto deja de contar cuando `core.hooksPath` lo sustituye, y uno del directorio configurado
// cuenta.
func TestGitPortHookPresentRespectsCoreHooksPath(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()
	repo := newRepo(t)
	port := NewGit()
	executable := func(path string) {
		writeFile(t, path, "#!/bin/sh\nexit 1\n")
		if err := os.Chmod(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if ok, err := port.HookPresent(ctx, repo, "pre-commit"); err != nil || ok {
		t.Fatalf("sin hooks = %v, %v", ok, err)
	}
	executable(filepath.Join(repo, ".git", "hooks", "pre-commit"))
	if ok, err := port.HookPresent(ctx, repo, "pre-commit"); err != nil || !ok {
		t.Errorf("hook por defecto = %v, %v; se esperaba true", ok, err)
	}
	if ok, _ := port.HookPresent(ctx, repo, "pre-push"); ok {
		t.Error("un hook inexistente no está presente")
	}

	custom := filepath.Join(t.TempDir(), "hooks")
	runGit(t, repo, "config", "core.hooksPath", custom)
	if ok, err := port.HookPresent(ctx, repo, "pre-commit"); err != nil || ok {
		t.Errorf("con core.hooksPath el hook por defecto no cuenta, = %v, %v", ok, err)
	}
	executable(filepath.Join(custom, "pre-push"))
	if ok, err := port.HookPresent(ctx, repo, "pre-push"); err != nil || !ok {
		t.Errorf("hook de core.hooksPath absoluto = %v, %v; se esperaba true", ok, err)
	}
	// Una ruta relativa se resuelve desde la raíz del repositorio, que es el directorio de git.
	runGit(t, repo, "config", "core.hooksPath", ".githooks")
	executable(filepath.Join(repo, ".githooks", "commit-msg"))
	if ok, err := port.HookPresent(ctx, repo, "commit-msg"); err != nil || !ok {
		t.Errorf("hook de core.hooksPath relativo = %v, %v; se esperaba true", ok, err)
	}
	// Un worktree enlazado comparte los hooks del repositorio común (el directorio por defecto y
	// el de una `core.hooksPath` absoluta).
	linked := filepath.Join(t.TempDir(), "enlazado")
	runGit(t, repo, "worktree", "add", "-q", "-b", "otra", linked)
	runGit(t, repo, "config", "--unset", "core.hooksPath")
	if ok, err := port.HookPresent(ctx, linked, "pre-commit"); err != nil || !ok {
		t.Errorf("hook por defecto visto desde un worktree enlazado = %v, %v; se esperaba true", ok, err)
	}
	runGit(t, repo, "config", "core.hooksPath", custom)
	if ok, err := port.HookPresent(ctx, linked, "pre-push"); err != nil || !ok {
		t.Errorf("hook de core.hooksPath absoluto visto desde un worktree enlazado = %v, %v; se esperaba true", ok, err)
	}
}

func TestGitPortIdentityConfigured(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()
	repo := newRepo(t)
	port := NewGit()

	if ok, err := port.IdentityConfigured(ctx, repo); err != nil || !ok {
		t.Fatalf("con identidad = %v, %v", ok, err)
	}
	for _, name := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}
	runGit(t, repo, "config", "user.useConfigOnly", "true")
	if ok, err := port.IdentityConfigured(ctx, repo); err != nil || ok {
		t.Errorf("sin identidad = %v, %v; se esperaba false", ok, err)
	}
	runGit(t, repo, "config", "user.name", "Persona")
	runGit(t, repo, "config", "user.email", "persona@example.invalid")
	if ok, err := port.IdentityConfigured(ctx, repo); err != nil || !ok {
		t.Errorf("identidad de la configuración = %v, %v; se esperaba true", ok, err)
	}
}

func TestGitPortPushOutcomes(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()
	repo, bare := newRepoWithRemote(t)
	port := NewGit()
	runGit(t, repo, "switch", "-q", "-c", "inc/x")
	writeFile(t, filepath.Join(repo, "a.md"), "a\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "a")

	t.Run("primer push con --set-upstream", func(t *testing.T) {
		outcome, err := port.Push(ctx, repo, "origin", "inc/x", true)
		if err != nil || outcome.ExitCode != 0 || outcome.TimedOut {
			t.Fatalf("Push = %+v, %v", outcome, err)
		}
		if !strings.Contains(outcome.Stdout, "refs/heads/inc/x:refs/heads/inc/x") || !strings.Contains(outcome.Stdout, "[new branch]") {
			t.Errorf("la salida --porcelain debe describir la rama nueva:\n%s", outcome.Stdout)
		}
		if got := runGit(t, bare, "rev-parse", "refs/heads/inc/x"); got != runGit(t, repo, "rev-parse", "HEAD") {
			t.Errorf("el remoto tiene %s, se esperaba HEAD", got)
		}
		if got := runGit(t, repo, "config", "branch.inc/x.remote"); got != "origin" {
			t.Errorf("--set-upstream debía configurar el seguimiento, remote = %q", got)
		}
	})
	t.Run("avance rápido sin --set-upstream", func(t *testing.T) {
		writeFile(t, filepath.Join(repo, "b.md"), "b\n")
		runGit(t, repo, "add", "-A")
		runGit(t, repo, "commit", "-q", "-m", "b")
		outcome, err := port.Push(ctx, repo, "origin", "inc/x", false)
		if err != nil || outcome.ExitCode != 0 {
			t.Fatalf("Push = %+v, %v", outcome, err)
		}
	})
	t.Run("sin nada que subir", func(t *testing.T) {
		outcome, err := port.Push(ctx, repo, "origin", "inc/x", false)
		if err != nil || outcome.ExitCode != 0 || !strings.Contains(outcome.Stdout, "[up to date]") {
			t.Fatalf("Push = %+v, %v; se esperaba «[up to date]»", outcome, err)
		}
	})
	t.Run("non-fast-forward", func(t *testing.T) {
		other := t.TempDir()
		runGit(t, other, "clone", "-q", bare, "clon")
		clone := filepath.Join(other, "clon")
		runGit(t, clone, "switch", "-q", "inc/x")
		writeFile(t, filepath.Join(clone, "ajeno.md"), "ajeno\n")
		runGit(t, clone, "add", "-A")
		runGit(t, clone, "commit", "-q", "-m", "ajeno")
		runGit(t, clone, "push", "-q", "origin", "inc/x")
		writeFile(t, filepath.Join(repo, "c.md"), "c\n")
		runGit(t, repo, "add", "-A")
		runGit(t, repo, "commit", "-q", "-m", "c")

		outcome, err := port.Push(ctx, repo, "origin", "inc/x", false)

		if err != nil || outcome.ExitCode == 0 {
			t.Fatalf("Push = %+v, %v; se esperaba un rechazo", outcome, err)
		}
		if !strings.Contains(outcome.Stdout, "[rejected]") || !(strings.Contains(outcome.Stdout, "non-fast-forward") || strings.Contains(outcome.Stdout, "fetch first")) {
			t.Errorf("la salida --porcelain debe describir el rechazo:\n%s", outcome.Stdout)
		}
	})
	t.Run("tiempo máximo", func(t *testing.T) {
		previous := pushTimeout
		pushTimeout = time.Nanosecond
		t.Cleanup(func() { pushTimeout = previous })
		outcome, err := port.Push(ctx, repo, "origin", "inc/x", false)
		if err != nil || !outcome.TimedOut {
			t.Fatalf("Push = %+v, %v; se esperaba TimedOut", outcome, err)
		}
	})
}

// Matriz «Selección del repositorio git»: el adaptador actúa sobre el directorio que recibe aunque el
// proceso tenga `GIT_DIR` (u otras variables que redirigen el repositorio), que gitcommon retira.
func TestPublishIgnoresGitDirEnvironment(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()
	repo := newRepo(t)
	decoy := newRepo(t)
	t.Setenv("GIT_DIR", filepath.Join(decoy, ".git"))
	t.Setenv("GIT_WORK_TREE", decoy)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(decoy, ".git", "index"))
	writeFile(t, filepath.Join(repo, "docs", "a.md"), "a\n")
	writeFile(t, filepath.Join(decoy, "docs", "senuelo.md"), "señuelo\n")
	port := NewGit()

	dirty, err := port.DirtyPaths(ctx, repo, []string{"docs"})
	if err != nil || !reflect.DeepEqual(dirty, []string{"docs/a.md"}) {
		t.Fatalf("DirtyPaths = %q, %v; se esperaba solo docs/a.md del repositorio pedido", dirty, err)
	}
	sha, err := port.Commit(ctx, repo, "docs(sdd): publicar los documentos de x", []string{"docs"}, nil)
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("GIT_INDEX_FILE")
	if got := runGit(t, repo, "rev-parse", "HEAD"); got != sha {
		t.Errorf("el commit no se hizo en el repositorio pedido: HEAD = %s, commit = %s", got, sha)
	}
	if got := runGit(t, decoy, "log", "--oneline"); strings.Contains(got, "publicar") {
		t.Errorf("el repositorio señuelo no debe recibir el commit:\n%s", got)
	}
}
