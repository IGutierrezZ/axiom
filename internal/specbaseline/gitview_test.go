package specbaseline

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// isolateGit deja a los procesos git de la prueba sin acceso a la configuración
// del usuario, con un autor fijo y sin red: ninguna prueba de este fichero la
// necesita (D-15, design §8).
func isolateGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("la prueba ejecuta git: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está disponible: %v", err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Autor de Prueba")
	t.Setenv("GIT_AUTHOR_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Autor de Prueba")
	t.Setenv("GIT_COMMITTER_EMAIL", "autor@ejemplo.invalid")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
}

// runGit ejecuta git en dir y falla la prueba si termina con error.
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

// newRepo crea un repositorio git en dir, con la rama main y sin conversión de
// finales de línea.
func newRepo(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "symbolic-ref", "HEAD", "refs/heads/main")
	runGit(t, dir, "config", "core.autocrlf", "false")
	return dir
}

func writeRepoFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commitAll(t *testing.T, dir, message string) string {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", message)
	return runGit(t, dir, "rev-parse", "HEAD")
}

func sameDir(t *testing.T, got, want string) {
	t.Helper()
	if !pathidentity.SameDirectory(got, want) {
		t.Errorf("repositorio = %s, quería %s", got, want)
	}
}

// siblings monta la carpeta maestra de la matriz de amenazas: <maestra>/specs y
// <maestra>/repo-backend son repositorios hermanos.
func siblings(t *testing.T) (master, specs, backend string) {
	t.Helper()
	isolateGit(t)
	master = t.TempDir()
	specs = newRepo(t, filepath.Join(master, "specs"))
	backend = newRepo(t, filepath.Join(master, "repo-backend"))
	writeRepoFile(t, specs, "01_Requisitos.md", "# Requisitos\n")
	writeRepoFile(t, backend, "docs/API.md", "# API\n")
	commitAll(t, specs, "specs")
	commitAll(t, backend, "backend")
	return master, specs, backend
}

// Selección de repositorio (design §8): el repositorio es el de la ruta, nunca el
// del proceso.
func TestGitOpenRepositorySelectsTheRepositoryOfThePath(t *testing.T) {
	ctx := context.Background()
	master, specs, backend := siblings(t)
	nested := newRepo(t, filepath.Join(specs, "vendor", "libreria"))
	writeRepoFile(t, nested, "LEEME.md", "# Librería\n")

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "ruta absoluta de un documento", path: filepath.Join(specs, "01_Requisitos.md"), want: specs},
		{name: "ruta mantenida con `..` al repositorio hermano", path: filepath.Join(specs, "..", "repo-backend", "docs", "API.md"), want: backend},
		{name: "documento por crear en un directorio que existe", path: filepath.Join(backend, "docs", "NUEVO.md"), want: backend},
		{name: "documento por crear en un directorio que tampoco existe", path: filepath.Join(backend, "docs", "nuevo", "dir", "NUEVO.md"), want: backend},
		{name: "el propio directorio del repositorio", path: backend, want: backend},
		{name: "repositorio anidado tratado como propio", path: filepath.Join(nested, "LEEME.md"), want: nested},
		{name: "un fichero del repositorio contenedor de un anidado", path: filepath.Join(specs, "01_Requisitos.md"), want: specs},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := openRepository(ctx, filepath.Clean(tt.path))
			if err != nil {
				t.Fatalf("openRepository: %v", err)
			}
			sameDir(t, repo.root, tt.want)
		})
	}

	t.Run("ejecutar desde un cwd ajeno no cambia el repositorio elegido", func(t *testing.T) {
		t.Chdir(backend)
		repo, err := openRepository(ctx, filepath.Join(specs, "01_Requisitos.md"))
		if err != nil {
			t.Fatal(err)
		}
		sameDir(t, repo.root, specs)
		t.Chdir(master) // la carpeta maestra no es un repositorio
		repo, err = openRepository(ctx, filepath.Join(backend, "docs", "API.md"))
		if err != nil {
			t.Fatal(err)
		}
		sameDir(t, repo.root, backend)
	})

	t.Run("las variables que redirigen git no cambian el repositorio elegido", func(t *testing.T) {
		t.Setenv("GIT_DIR", filepath.Join(backend, ".git"))
		t.Setenv("GIT_WORK_TREE", backend)
		t.Setenv("GIT_INDEX_FILE", filepath.Join(backend, ".git", "index"))
		repo, err := openRepository(ctx, filepath.Join(specs, "01_Requisitos.md"))
		if err != nil {
			t.Fatal(err)
		}
		sameDir(t, repo.root, specs)
		status, err := repo.status(ctx)
		if err != nil || len(status) != 1 || status[0].Path != "vendor/libreria/" || status[0].Code != "??" {
			t.Fatalf("el estado es el de specs (solo el repositorio anidado, sin seguimiento): %v %v", status, err)
		}
	})
}

func TestGitOpenRepositoryRejectsWhatItCannotTrust(t *testing.T) {
	ctx := context.Background()
	isolateGit(t)
	base := t.TempDir()
	plain := filepath.Join(base, "sin-git")
	if err := os.MkdirAll(filepath.Join(plain, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := openRepository(ctx, filepath.Join(plain, "docs", "A.md"))
	var notGit *NotGitRepositoryError
	if !errors.As(err, &notGit) {
		t.Fatalf("un directorio sin .git debe producir *NotGitRepositoryError, no %v", err)
	}
	if !strings.Contains(err.Error(), "repositorio git") || !strings.Contains(err.Error(), "git init") {
		t.Errorf("el rechazo debe explicar que la verificación por diff exige git y cómo versionar: %v", err)
	}

	if _, err := openRepository(ctx, filepath.Join("docs", "A.md")); err == nil || !strings.Contains(err.Error(), "ruta absoluta") {
		t.Errorf("nunca se resuelve una ruta relativa contra el cwd del proceso: %v", err)
	}
}

func TestGitStatusParsesPorcelainWithoutQuoting(t *testing.T) {
	ctx := context.Background()
	_, specs, _ := siblings(t)
	writeRepoFile(t, specs, "notas ñ.md", "uno\n")
	writeRepoFile(t, specs, "carpeta/dos.md", "dos\n")
	commitAll(t, specs, "más ficheros")

	writeRepoFile(t, specs, "01_Requisitos.md", "# Requisitos\n\nsin stage\n") // ` M`
	writeRepoFile(t, specs, "carpeta/dos.md", "dos con stage\n")
	runGit(t, specs, "add", "carpeta/dos.md")                            // `M `
	writeRepoFile(t, specs, "sin seguimiento/árbol/nuevo.md", "nuevo\n") // `??`, cada fichero por separado
	runGit(t, specs, "mv", "notas ñ.md", "notas renombradas ñ.md")       // `R ` con dos rutas

	repo, err := openRepository(ctx, specs)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := repo.status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	got := map[string]string{}
	for _, entry := range entries {
		got[entry.Path] = entry.Code
	}
	want := map[string]string{
		"01_Requisitos.md":               " M",
		"carpeta/dos.md":                 "M ",
		"sin seguimiento/árbol/nuevo.md": "??",
		"notas renombradas ñ.md":         "R ",
	}
	if len(got) != len(want) {
		t.Fatalf("entradas = %v, quería %v (el origen de un renombrado no es una entrada)", got, want)
	}
	for path, code := range want {
		if got[path] != code {
			t.Errorf("%q = %q, quería %q", path, got[path], code)
		}
	}
}

func TestGitStatusOnAnEmptyRepositoryAndHead(t *testing.T) {
	ctx := context.Background()
	isolateGit(t)
	dir := newRepo(t, filepath.Join(t.TempDir(), "vacio"))
	repo, err := openRepository(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	head, err := repo.head(ctx)
	if err != nil || head != "" {
		t.Fatalf("head de un repositorio sin commits = %q, %v; quería vacío sin error", head, err)
	}
	writeRepoFile(t, dir, "a.md", "a\n")
	entries, err := repo.status(ctx)
	if err != nil || len(entries) != 1 || entries[0].Code != "??" {
		t.Fatalf("status sin commits = %v, %v", entries, err)
	}
	commit := commitAll(t, dir, "primero")
	if head, err = repo.head(ctx); err != nil || head != commit {
		t.Fatalf("head = %q, %v; quería %s", head, err, commit)
	}
}

// Estado del push (design §8): solo referencias locales, nunca la red.
func TestGitUpstreamReportsOnlyWhatLocalReferencesSay(t *testing.T) {
	ctx := context.Background()
	isolateGit(t)
	base := t.TempDir()
	origin := newRepo(t, filepath.Join(base, "origen"))
	writeRepoFile(t, origin, "a.md", "a\n")
	commitAll(t, origin, "uno")
	clone := filepath.Join(base, "clon")
	runGit(t, base, "clone", "-q", origin, clone)
	runGit(t, clone, "config", "core.autocrlf", "false")
	repo, err := openRepository(ctx, clone)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("al día", func(t *testing.T) {
		state := repo.upstream(ctx)
		if state.Behind != 0 || len(state.Notes) != 0 {
			t.Fatalf("estado = %+v", state)
		}
	})

	writeRepoFile(t, origin, "b.md", "b\n")
	commitAll(t, origin, "dos")
	writeRepoFile(t, origin, "c.md", "c\n")
	commitAll(t, origin, "tres")

	t.Run("sin fetch no hay noticia del remoto", func(t *testing.T) {
		if state := repo.upstream(ctx); state.Behind != 0 {
			t.Fatalf("el dato procede de referencias locales y Axiom nunca hace fetch: %+v", state)
		}
	})

	runGit(t, clone, "fetch", "-q", "origin") // lo hace la prueba, con un remoto local; el código nunca
	t.Run("rama por detrás tras actualizar las referencias locales", func(t *testing.T) {
		state := repo.upstream(ctx)
		if state.Behind != 2 {
			t.Fatalf("Behind = %d, quería 2", state.Behind)
		}
	})

	t.Run("sin upstream", func(t *testing.T) {
		runGit(t, clone, "checkout", "-q", "-b", "local")
		state := repo.upstream(ctx)
		if state.Behind != 0 || len(state.Notes) != 1 || !strings.Contains(state.Notes[0], "upstream") {
			t.Fatalf("estado = %+v, quería una nota sobre el upstream", state)
		}
	})

	t.Run("HEAD desacoplado", func(t *testing.T) {
		runGit(t, clone, "checkout", "-q", "--detach")
		state := repo.upstream(ctx)
		if state.Behind != 0 || len(state.Notes) != 1 || !strings.Contains(state.Notes[0], "desacoplado") {
			t.Fatalf("estado = %+v, quería una nota sobre el HEAD desacoplado", state)
		}
	})

	t.Run("repositorio sin commits", func(t *testing.T) {
		empty := newRepo(t, filepath.Join(base, "vacio"))
		vacio, err := openRepository(ctx, empty)
		if err != nil {
			t.Fatal(err)
		}
		if state := vacio.upstream(ctx); state.Behind != 0 || len(state.Notes) != 1 {
			t.Fatalf("estado = %+v", state)
		}
	})
}

// El código de producción solo lee: ni fetch, ni pull, ni push, ni add, ni commit,
// ni `-C` con rutas relativas (D-15, design §8).
func TestGitViewNeverRunsAMutatingCommand(t *testing.T) {
	source, err := os.ReadFile("gitview.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, forbidden := range []string{`"fetch"`, `"pull"`, `"push"`, `"add"`, `"commit"`, `"-C"`, `"checkout"`, `"reset"`, `"stash"`, `"clean"`} {
		if strings.Contains(text, forbidden) {
			t.Errorf("gitview.go no puede usar %s", forbidden)
		}
	}
	for _, required := range []string{"GIT_OPTIONAL_LOCKS=0", "--show-toplevel", "--porcelain=v1", "--untracked-files=all", "cmd.Dir"} {
		if !strings.Contains(text, required) {
			t.Errorf("gitview.go debe usar %q", required)
		}
	}
}
