package gitcommon

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// gitOutput ejecuta git directamente, sin pasar por Run, para que el oráculo de
// una prueba de Run no dependa de Run. Devuelve la salida sin el salto final.
func gitOutput(t *testing.T, dir string, extraEnv []string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	command.Env = append(os.Environ(), extraEnv...)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git %v en %s: %v", args, dir, err)
	}
	return strings.TrimRight(string(output), "\r\n")
}

// observe instala un observador que anota cada invocación y lo retira al terminar
// la prueba: Observe es nula en producción.
func observe(t *testing.T) *[][]string {
	t.Helper()
	var seen [][]string
	Observe = func(dir string, args []string) {
		seen = append(seen, append([]string{dir}, args...))
	}
	t.Cleanup(func() { Observe = nil })
	return &seen
}

// wantSameDirectory compara dos directorios por identidad: los temporales de
// Windows tienen nombres cortos (8.3) y git devuelve rutas con barras.
func wantSameDirectory(t *testing.T, what, got, want string) {
	t.Helper()
	if got == "" || !filepath.IsAbs(got) {
		t.Errorf("%s = %q, se esperaba una ruta absoluta", what, got)
		return
	}
	if !pathidentity.SameDirectory(got, want) {
		t.Errorf("%s = %q, se esperaba el directorio %q", what, got, want)
	}
}

// design §8, «Selección de repositorio git»: las variables con las que un proceso
// padre fija el repositorio sobre el que actúa git no cambian la selección.
func TestRunRemovesRedirectingVariablesFromTheEnvironment(t *testing.T) {
	requireGit(t)
	ctx := context.Background()
	target := newRepository(t)
	other := newRepository(t)
	otherGitDir := filepath.Join(other, ".git")

	t.Run("GIT_DIR y GIT_WORK_TREE hacia otro repositorio", func(t *testing.T) {
		redirect := []string{"GIT_DIR=" + otherGitDir, "GIT_WORK_TREE=" + other}
		// Control: sin la retirada, git sí actúa sobre el otro repositorio.
		wantSameDirectory(t, "control sin retirada", filepath.FromSlash(gitOutput(t, target, redirect, "rev-parse", "--show-toplevel")), other)

		t.Setenv("GIT_DIR", otherGitDir)
		t.Setenv("GIT_WORK_TREE", other)
		top, err := TopLevel(ctx, target)
		if err != nil {
			t.Fatalf("TopLevel(%q): %v", target, err)
		}
		wantSameDirectory(t, "TopLevel con GIT_DIR hacia otro repositorio", top, target)
	})

	t.Run("GIT_COMMON_DIR hacia otro repositorio", func(t *testing.T) {
		redirect := []string{"GIT_COMMON_DIR=" + otherGitDir}
		wantSameDirectory(t, "control sin retirada", filepath.FromSlash(gitOutput(t, target, redirect, "rev-parse", "--path-format=absolute", "--git-common-dir")), otherGitDir)

		t.Setenv("GIT_COMMON_DIR", otherGitDir)
		common, err := CommonDir(ctx, target)
		if err != nil {
			t.Fatalf("CommonDir(%q): %v", target, err)
		}
		wantSameDirectory(t, "CommonDir con GIT_COMMON_DIR hacia otro repositorio", common, filepath.Join(target, ".git"))
	})

	t.Run("GIT_INDEX_FILE hacia otro índice", func(t *testing.T) {
		sentinel := filepath.Join(other, "indice-ajeno")
		redirect := []string{"GIT_INDEX_FILE=" + sentinel}
		if got := filepath.FromSlash(gitOutput(t, target, redirect, "rev-parse", "--git-path", "index")); !pathidentity.SameDirectory(filepath.Dir(got), other) {
			t.Fatalf("control sin retirada: git --git-path index = %q, se esperaba el índice ajeno %q", got, sentinel)
		}

		t.Setenv("GIT_INDEX_FILE", sentinel)
		result, err := Run(ctx, target, "rev-parse", "--git-path", "index")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		got := strings.TrimSpace(string(result.Stdout))
		if got != ".git/index" {
			t.Errorf("git --git-path index = %q, se esperaba el índice propio %q: GIT_INDEX_FILE no se retiró", got, ".git/index")
		}
	})
}

// design §3: el entorno de git es el del proceso sin las variables que redirigen
// el repositorio, con GIT_OPTIONAL_LOCKS=0 y sin preguntas interactivas.
func TestGitEnvironmentFixesLocksAndPromptsAndDropsRedirection(t *testing.T) {
	for _, name := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_PREFIX", "GIT_NAMESPACE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES"} {
		t.Setenv(name, "ajeno")
	}
	t.Setenv("GIT_OPTIONAL_LOCKS", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	t.Setenv("AXIOM_VARIABLE_NEUTRA", "se conserva")

	effective := map[string]string{}
	for _, entry := range gitEnvironment() {
		name, value, _ := strings.Cut(entry, "=")
		effective[strings.ToUpper(name)] = value // la última aparición gana, como en exec.Cmd
	}

	for _, name := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_PREFIX", "GIT_NAMESPACE", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES"} {
		if value, present := effective[name]; present {
			t.Errorf("%s = %q sigue en el entorno de git: una variable que redirige el repositorio debe retirarse", name, value)
		}
	}
	if got := effective["GIT_OPTIONAL_LOCKS"]; got != "0" {
		t.Errorf("GIT_OPTIONAL_LOCKS = %q, se esperaba \"0\" aunque el proceso la fije distinta", got)
	}
	if got := effective["GIT_TERMINAL_PROMPT"]; got != "0" {
		t.Errorf("GIT_TERMINAL_PROMPT = %q, se esperaba \"0\" aunque el proceso la fije distinta", got)
	}
	if got := effective["LC_ALL"]; got != "C" {
		t.Errorf("LC_ALL = %q, se esperaba \"C\": la clasificación de los fallos de git lee sus mensajes", got)
	}
	if got := effective["AXIOM_VARIABLE_NEUTRA"]; got != "se conserva" {
		t.Errorf("una variable ajena a git debe conservarse, obtenida %q", got)
	}
}

// Run devuelve la salida y el código de salida: un código distinto de cero no es
// un error de ejecución, lo decide quien llama.
func TestRunReturnsOutputAndExitCodeWithoutTreatingAFailureAsAnError(t *testing.T) {
	requireGit(t)
	ctx := context.Background()
	repo := newRepository(t)

	result, err := Run(ctx, repo, "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.ExitCode != 0 || strings.TrimSpace(string(result.Stdout)) == "" {
		t.Errorf("Run(rev-parse --show-toplevel) = %+v, se esperaba código 0 y una ruta en la salida", result)
	}

	result, err = Run(ctx, repo, "rev-parse", "--verify", "--quiet", "refs/heads/no-existe")
	if err != nil {
		t.Fatalf("un código de salida distinto de cero no es un error de ejecución: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, se esperaba 1", result.ExitCode)
	}

	result, err = Run(ctx, repo, "cat-file", "-t", "objeto-que-no-existe")
	if err != nil {
		t.Fatalf("un fallo de git no es un error de ejecución: %v", err)
	}
	if result.ExitCode == 0 || result.Stderr == "" {
		t.Errorf("Run con un objeto inexistente = %+v, se esperaba un código distinto de cero y un mensaje en Stderr", result)
	}
}

// El directorio de trabajo de git es siempre el que se pide, absoluto: nunca el
// del proceso (matriz de amenazas «Selección de repositorio git»).
func TestRunRefusesAnUnusableDirectoryAndEmptyArguments(t *testing.T) {
	requireGit(t)
	ctx := context.Background()
	repo := newRepository(t)

	tests := []struct {
		name string
		dir  string
		args []string
		want string
	}{
		{"directorio relativo", ".", []string{"status"}, "no es absoluta"},
		{"directorio vacío", "", []string{"status"}, "no es absoluta"},
		{"directorio que no existe", filepath.Join(repo, "no", "existe"), []string{"status"}, "no existe"},
		{"un fichero en lugar de un directorio", writeFile(t, filepath.Join(repo, "nota.txt"), "x\n"), []string{"status"}, "no es un directorio"},
		{"sin argumentos", repo, nil, "sin argumentos"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			seen := observe(t)
			_, err := Run(ctx, tc.dir, tc.args...)
			if err == nil {
				t.Fatalf("Run(%q, %v) no devolvió error", tc.dir, tc.args)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("el error %q no menciona %q", err, tc.want)
			}
			if len(*seen) != 0 {
				t.Errorf("no debía lanzarse git, el observador vio %v", *seen)
			}
		})
	}
}

// design §3: tiempo máximo y límite de salida. Los límites son variables de
// paquete para probarlos sin esperar 30 s ni producir 64 MiB.
func TestRunBoundsTheRunningTimeAndTheOutput(t *testing.T) {
	requireGit(t)
	ctx := context.Background()
	repo := newRepository(t)

	t.Run("tiempo máximo", func(t *testing.T) {
		previous := runTimeout
		runTimeout = time.Nanosecond
		t.Cleanup(func() { runTimeout = previous })

		_, err := Run(ctx, repo, "rev-parse", "--show-toplevel")
		if err == nil || !strings.Contains(err.Error(), "tiempo máximo") {
			t.Fatalf("Run con el tiempo agotado = %v, se esperaba un error que nombre el tiempo máximo", err)
		}
	})

	t.Run("límite de salida", func(t *testing.T) {
		previous := stdoutLimit
		stdoutLimit = 4
		t.Cleanup(func() { stdoutLimit = previous })

		result, err := Run(ctx, repo, "rev-parse", "--show-toplevel")
		if err == nil || !strings.Contains(err.Error(), "límite") {
			t.Fatalf("Run con la salida excedida = %v, se esperaba un error que nombre el límite", err)
		}
		if len(result.Stdout) > 4 {
			t.Errorf("la salida conservada (%d bytes) supera el límite de 4", len(result.Stdout))
		}
	})

	t.Run("un contexto cancelado por quien llama", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := Run(cancelled, repo, "rev-parse", "--show-toplevel"); err == nil {
			t.Fatal("Run con el contexto cancelado debía devolver un error")
		}
	})
}

// REQ-25.1, G-7: el observador recibe cada invocación, con el directorio y los
// argumentos, y es nulo en producción.
func TestObserveReceivesEveryInvocation(t *testing.T) {
	requireGit(t)
	if Observe != nil {
		t.Fatal("Observe debe ser nula en producción")
	}
	ctx := context.Background()
	repo := newRepository(t)
	seen := observe(t)

	if _, err := Run(ctx, repo, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := TopLevel(ctx, repo); err != nil {
		t.Fatalf("TopLevel: %v", err)
	}
	if _, err := CommonDir(ctx, repo); err != nil {
		t.Fatalf("CommonDir: %v", err)
	}

	want := [][]string{
		{repo, "rev-parse", "--verify", "--quiet", "HEAD"},
		{repo, "rev-parse", "--show-toplevel"},
		{repo, "rev-parse", "--path-format=absolute", "--git-common-dir"},
	}
	if !reflect.DeepEqual(*seen, want) {
		t.Errorf("invocaciones observadas:\n got %v\nwant %v", *seen, want)
	}
}

// El observador no puede alterar los argumentos que se ejecutan: recibe una copia.
func TestObserveCannotChangeTheArguments(t *testing.T) {
	requireGit(t)
	repo := newRepository(t)
	Observe = func(_ string, args []string) {
		for i := range args {
			args[i] = "--version"
		}
	}
	t.Cleanup(func() { Observe = nil })

	result, err := Run(context.Background(), repo, "rev-parse", "--show-toplevel")
	if err != nil || result.ExitCode != 0 || strings.Contains(string(result.Stdout), "git version") {
		t.Errorf("el observador alteró la invocación: %+v, %v", result, err)
	}
}

// REQ-25.6 y REQ-25.14: la raíz de trabajo y el git common dir salen de git, en un
// clon normal y en un worktree enlazado.
func TestTopLevelAndCommonDirInAnOrdinaryCloneAndALinkedWorktree(t *testing.T) {
	requireGit(t)
	ctx := context.Background()
	main := newRepository(t)
	linked := filepath.Join(t.TempDir(), "enlazado")
	runGit(t, main, "worktree", "add", "-q", "--detach", linked, "HEAD")
	subfolder := mkdirAll(t, filepath.Join(main, "app", "web"))
	linkedSubfolder := mkdirAll(t, filepath.Join(linked, "src"))

	tests := []struct {
		name       string
		dir        string
		wantTop    string
		wantCommon string
	}{
		{"raíz del clon", main, main, filepath.Join(main, ".git")},
		{"subcarpeta del clon", subfolder, main, filepath.Join(main, ".git")},
		{"raíz del worktree enlazado", linked, linked, filepath.Join(main, ".git")},
		{"subcarpeta del worktree enlazado", linkedSubfolder, linked, filepath.Join(main, ".git")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			top, err := TopLevel(ctx, tc.dir)
			if err != nil {
				t.Fatalf("TopLevel(%q): %v", tc.dir, err)
			}
			wantSameDirectory(t, "TopLevel", top, tc.wantTop)

			common, err := CommonDir(ctx, tc.dir)
			if err != nil {
				t.Fatalf("CommonDir(%q): %v", tc.dir, err)
			}
			wantSameDirectory(t, "CommonDir", common, tc.wantCommon)
		})
	}
}

// El git common dir no se supone `.git`: con un git dir enlazado (`.git` fichero
// hacia otro directorio) es el que git devuelve (REQ-25.14).
func TestCommonDirDoesNotAssumeDotGit(t *testing.T) {
	requireGit(t)
	ctx := context.Background()
	work := t.TempDir()
	separate := filepath.Join(t.TempDir(), "datos-de-git")
	runGit(t, work, "init", "-q", "-b", "main", "--separate-git-dir", separate)

	common, err := CommonDir(ctx, work)
	if err != nil {
		t.Fatalf("CommonDir(%q): %v", work, err)
	}
	wantSameDirectory(t, "CommonDir con git dir separado", common, separate)
}

// Fuera de un repositorio las dos lecturas devuelven ErrNotGitRepository, que es
// lo que `axiom branch` traduce a `not_a_git_repository`.
func TestTopLevelAndCommonDirOutsideARepository(t *testing.T) {
	requireGit(t)
	ctx := context.Background()
	outside := t.TempDir()
	if insideGit(outside) {
		t.Skipf("el directorio temporal %s está dentro de un repositorio git", outside)
	}

	if _, err := TopLevel(ctx, outside); !errors.Is(err, ErrNotGitRepository) {
		t.Errorf("TopLevel fuera de git = %v, se esperaba ErrNotGitRepository", err)
	}
	if _, err := CommonDir(ctx, outside); !errors.Is(err, ErrNotGitRepository) {
		t.Errorf("CommonDir fuera de git = %v, se esperaba ErrNotGitRepository", err)
	}
}

// Un fallo de git que no es «no es un repositorio» no se disfraza de ello: el
// error conserva el resumen de git.
func TestTopLevelReportsOtherFailuresWithGitsSummary(t *testing.T) {
	requireGit(t)
	ctx := context.Background()
	bare := t.TempDir()
	runGit(t, bare, "init", "-q", "--bare", "-b", "main")

	_, err := TopLevel(ctx, bare)
	if err == nil {
		t.Fatal("TopLevel en un repositorio desnudo debía devolver un error")
	}
	if errors.Is(err, ErrNotGitRepository) {
		t.Errorf("un repositorio desnudo no es «no es un repositorio git»: %v", err)
	}
	if !strings.Contains(err.Error(), "rev-parse --show-toplevel") {
		t.Errorf("el error %q debe nombrar la operación de git", err)
	}
}
