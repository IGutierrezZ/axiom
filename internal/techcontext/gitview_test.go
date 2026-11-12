package techcontext

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

// runGit ejecuta git en dir con gitcommon.Run, el único punto de ejecución de git
// del producto, y falla la prueba si termina con error.
func runGit(t testing.TB, dir string, args ...string) string {
	t.Helper()
	result, err := gitcommon.Run(context.Background(), dir, args...)
	if err != nil {
		t.Fatalf("git %v en %s: %v", args, dir, err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("git %v en %s terminó con %d: %s", args, dir, result.ExitCode, result.Stderr)
	}
	return string(result.Stdout)
}

func writeTestFile(t testing.TB, file, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newRepo crea un repositorio git temporal con la rama main y un primer commit
// que contiene a.txt y docs/viejo.md. Devuelve su raíz.
func newRepo(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	writeTestFile(t, filepath.Join(dir, "a.txt"), "a\n")
	writeTestFile(t, filepath.Join(dir, "docs", "viejo.md"), "# viejo\n\nTexto largo para que git detecte el renombrado sin dudas.\nOtra línea.\nY otra más.\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", "inicial")
	return dir
}

// relative devuelve las rutas de ChangedSince relativas a la raíz del repositorio,
// con `/` y ordenadas, para compararlas con independencia del sistema.
func relative(t testing.TB, repo string, paths []string) []string {
	t.Helper()
	top, err := gitcommon.TopLevel(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		rel, err := filepath.Rel(top, p)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	sort.Strings(out)
	return out
}

func changedRelative(t testing.TB, repo, base string) []string {
	t.Helper()
	paths, err := ChangedSince(context.Background(), repo, base)
	if err != nil {
		t.Fatalf("ChangedSince: %v", err)
	}
	return relative(t, repo, paths)
}

// Amenaza «estado de commit» (REQ-30.6, D7): ChangedSince une los cuatro estados
// en los que una edición puede estar, y una prueba fija cada uno.
func TestGitViewReportsEachCommitState(t *testing.T) {
	t.Run("confirmado en la rama de la feature", func(t *testing.T) {
		repo := newRepo(t)
		base := runGit(t, repo, "rev-parse", "HEAD")[:40]
		runGit(t, repo, "checkout", "-q", "-b", "feat")
		writeTestFile(t, filepath.Join(repo, "docs", "nuevo.md"), "# nuevo\n")
		runGit(t, repo, "add", "-A")
		runGit(t, repo, "commit", "-q", "-m", "feature")
		if got, want := changedRelative(t, repo, base), []string{"docs/nuevo.md"}; !reflect.DeepEqual(got, want) {
			t.Errorf("cambiados = %q, quería %q", got, want)
		}
	})
	t.Run("en el índice", func(t *testing.T) {
		repo := newRepo(t)
		base := runGit(t, repo, "rev-parse", "HEAD")[:40]
		writeTestFile(t, filepath.Join(repo, "a.txt"), "cambiado\n")
		runGit(t, repo, "add", "a.txt")
		if got, want := changedRelative(t, repo, base), []string{"a.txt"}; !reflect.DeepEqual(got, want) {
			t.Errorf("cambiados = %q, quería %q", got, want)
		}
	})
	t.Run("sin confirmar en el árbol de trabajo", func(t *testing.T) {
		repo := newRepo(t)
		base := runGit(t, repo, "rev-parse", "HEAD")[:40]
		writeTestFile(t, filepath.Join(repo, "docs", "viejo.md"), "# viejo editado\n")
		if got, want := changedRelative(t, repo, base), []string{"docs/viejo.md"}; !reflect.DeepEqual(got, want) {
			t.Errorf("cambiados = %q, quería %q", got, want)
		}
	})
	t.Run("no seguido", func(t *testing.T) {
		repo := newRepo(t)
		base := runGit(t, repo, "rev-parse", "HEAD")[:40]
		writeTestFile(t, filepath.Join(repo, "docs", "suelto.md"), "# suelto\n")
		if got, want := changedRelative(t, repo, base), []string{"docs/suelto.md"}; !reflect.DeepEqual(got, want) {
			t.Errorf("cambiados = %q, quería %q", got, want)
		}
	})
	t.Run("un fichero ignorado no cuenta", func(t *testing.T) {
		repo := newRepo(t)
		base := runGit(t, repo, "rev-parse", "HEAD")[:40]
		writeTestFile(t, filepath.Join(repo, ".gitignore"), "*.log\n")
		writeTestFile(t, filepath.Join(repo, "ruido.log"), "x\n")
		if got, want := changedRelative(t, repo, base), []string{".gitignore"}; !reflect.DeepEqual(got, want) {
			t.Errorf("cambiados = %q, quería %q", got, want)
		}
	})
	t.Run("renombrado: cuentan ambas rutas", func(t *testing.T) {
		repo := newRepo(t)
		base := runGit(t, repo, "rev-parse", "HEAD")[:40]
		runGit(t, repo, "mv", filepath.Join("docs", "viejo.md"), filepath.Join("docs", "renombrado.md"))
		if got, want := changedRelative(t, repo, base), []string{"docs/renombrado.md", "docs/viejo.md"}; !reflect.DeepEqual(got, want) {
			t.Errorf("cambiados = %q, quería ambas rutas %q", got, want)
		}
	})
	t.Run("renombrado confirmado: cuentan ambas rutas", func(t *testing.T) {
		repo := newRepo(t)
		base := runGit(t, repo, "rev-parse", "HEAD")[:40]
		runGit(t, repo, "checkout", "-q", "-b", "feat")
		runGit(t, repo, "mv", filepath.Join("docs", "viejo.md"), filepath.Join("docs", "renombrado.md"))
		runGit(t, repo, "commit", "-q", "-m", "renombrar")
		if got, want := changedRelative(t, repo, base), []string{"docs/renombrado.md", "docs/viejo.md"}; !reflect.DeepEqual(got, want) {
			t.Errorf("cambiados = %q, quería ambas rutas %q", got, want)
		}
	})
	t.Run("sin cambios", func(t *testing.T) {
		repo := newRepo(t)
		base := runGit(t, repo, "rev-parse", "HEAD")[:40]
		if got := changedRelative(t, repo, base); len(got) != 0 {
			t.Errorf("cambiados = %q, quería ninguno", got)
		}
	})
}

func TestGitViewWithoutBaseComparesAgainstTheEmptyTree(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	writeTestFile(t, filepath.Join(dir, "uno.md"), "# uno\n")
	runGit(t, dir, "add", "uno.md")
	writeTestFile(t, filepath.Join(dir, "dos.md"), "# dos\n")
	if got, want := changedRelative(t, dir, ""), []string{"dos.md", "uno.md"}; !reflect.DeepEqual(got, want) {
		t.Errorf("cambiados = %q, quería %q", got, want)
	}
}

func TestGitViewIsReadOnly(t *testing.T) {
	repo := newRepo(t)
	base := runGit(t, repo, "rev-parse", "HEAD")[:40]
	writeTestFile(t, filepath.Join(repo, "a.txt"), "cambiado\n")
	writeTestFile(t, filepath.Join(repo, "suelto.md"), "x\n")
	index := filepath.Join(repo, ".git", "index")
	before, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ChangedSince(context.Background(), repo, base); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("ChangedSince reescribió el índice de git: la consulta debe ser de solo lectura")
	}
}

func TestGitViewRejectsABaseThatDoesNotExist(t *testing.T) {
	repo := newRepo(t)
	if _, err := ChangedSince(context.Background(), repo, "no-existe-esta-rama"); err == nil {
		t.Fatal("una base irresoluble debe fallar cerrada, no devolver un conjunto vacío")
	}
}

// Amenaza «selección de repositorio git»: el repositorio es el de la ruta que se
// pregunta, no el del directorio del proceso.
func TestGitViewUsesTheRepositoryOfThePathNotTheProcessDirectory(t *testing.T) {
	repo := newRepo(t)
	other := newRepo(t)
	base := runGit(t, repo, "rev-parse", "HEAD")[:40]
	writeTestFile(t, filepath.Join(repo, "docs", "suelto.md"), "x\n")
	t.Chdir(other)
	if got, want := changedRelative(t, filepath.Join(repo, "docs"), base), []string{"docs/suelto.md"}; !reflect.DeepEqual(got, want) {
		t.Errorf("cambiados = %q, quería %q", got, want)
	}
}

func TestGitViewOutsideARepositoryFails(t *testing.T) {
	if _, err := ChangedSince(context.Background(), t.TempDir(), ""); err == nil {
		t.Fatal("fuera de un repositorio git debe fallar")
	}
}
