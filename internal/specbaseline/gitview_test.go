package specbaseline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// isolateGit omite la prueba cuando no puede ejecutar git. El aislamiento del
// entorno de git (sin configuración del usuario, con un autor fijo y sin red) lo
// fija TestMain una sola vez para todo el binario (D-15, design §8).
func isolateGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("la prueba ejecuta git: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está disponible: %v", err)
	}
}

// runGit ejecuta git en dir y falla la prueba si termina con error.
func runGit(t testing.TB, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
	return strings.TrimRight(string(output), "\r\n")
}

// emptyRepoTemplate es el repositorio recién creado que newRepo copia en cada
// prueba.
var emptyRepoTemplate = &treeTemplate{name: "repositorio-vacio"}

// newRepo crea un repositorio git en dir, con la rama main y sin conversión de
// finales de línea. El repositorio es la copia de uno que se crea una sola vez con
// `git init`: cada prueba recibe el suyo, independiente.
func newRepo(t testing.TB, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	emptyRepoTemplate.materialize(t, dir, func(t testing.TB, fresh string) {
		runGit(t, fresh, "init", "-q")
		runGit(t, fresh, "symbolic-ref", "HEAD", "refs/heads/main")
		runGit(t, fresh, "config", "core.autocrlf", "false")
	})
	return dir
}

func writeRepoFile(t testing.TB, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commitAll(t testing.TB, dir, message string) string {
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

var siblingsTemplate = &treeTemplate{name: "hermanos"}

// siblings monta la carpeta maestra de la matriz de amenazas: <maestra>/specs y
// <maestra>/repo-backend son repositorios hermanos. Se construye una sola vez y se
// copia a la carpeta de cada prueba.
func siblings(t *testing.T) (master, specs, backend string) {
	t.Helper()
	isolateGit(t)
	master = t.TempDir()
	specs = filepath.Join(master, "specs")
	backend = filepath.Join(master, "repo-backend")
	siblingsTemplate.materialize(t, master, func(t testing.TB, dir string) {
		specs := newRepo(t, filepath.Join(dir, "specs"))
		backend := newRepo(t, filepath.Join(dir, "repo-backend"))
		writeRepoFile(t, specs, "01_Requisitos.md", "# Requisitos\n")
		writeRepoFile(t, backend, "docs/API.md", "# API\n")
		commitAll(t, specs, "specs")
		commitAll(t, backend, "backend")
	})
	return master, specs, backend
}

// Selección de repositorio (design §8): el repositorio es el de la ruta, nunca el
// del proceso.
func TestGitOpenRepositorySelectsTheRepositoryOfThePath(t *testing.T) {
	// Sin t.Parallel: cambia el directorio del proceso y las variables de entorno de git (t.Chdir, t.Setenv).
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// gitIndexPath es la ruta del índice que git usa en ese árbol de trabajo, según
// `git rev-parse --git-path index`: en un worktree enlazado o con GIT_DIR no es
// `<raíz>/.git/index`.
func gitIndexPath(t testing.TB, dir string) string {
	t.Helper()
	path := filepath.FromSlash(runGit(t, dir, "rev-parse", "--git-path", "index"))
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	return path
}

// gitIndexState es la huella del directorio donde vive el índice de ese árbol de
// trabajo: los nombres de todo lo que hay en él (un `index.lock` o un
// `sharedindex.*` nuevos cuentan) y los bytes del índice y de los índices
// compartidos. Si no hay índice, la huella lo refleja.
func gitIndexState(t testing.TB, dir string) string {
	t.Helper()
	gitDir := filepath.Dir(gitIndexPath(t, dir))
	entries, err := os.ReadDir(gitDir)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	for _, entry := range entries {
		fmt.Fprintf(hash, "%s\n", entry.Name())
		if entry.Name() != "index" && !strings.HasPrefix(entry.Name(), "sharedindex.") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(gitDir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		hash.Write(data)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// makeIndexStale desfasa las marcas de tiempo de todos los ficheros seguidos sin
// cambiar su contenido (lo que hace, por ejemplo, un `touch` tras el commit): el
// índice guarda otras marcas y cualquier `git diff` las refresca y reescribe.
func makeIndexStale(t testing.TB, dir string) {
	t.Helper()
	old := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	for _, rel := range strings.Split(runGit(t, dir, "ls-files", "-z"), "\x00") {
		if rel == "" {
			continue
		}
		if err := os.Chtimes(filepath.Join(dir, filepath.FromSlash(rel)), old, old); err != nil {
			t.Fatal(err)
		}
	}
}

// committedRepo es un repositorio con tres ficheros confirmados, uno de ellos con
// nombre no ASCII.
func committedRepo(t *testing.T) string {
	t.Helper()
	isolateGit(t)
	dir := newRepo(t, filepath.Join(t.TempDir(), "repo"))
	writeRepoFile(t, dir, "a.md", "# A\n")
	writeRepoFile(t, dir, "docs/b.md", "# B\n")
	writeRepoFile(t, dir, "notas ñ.md", "ñ\n")
	commitAll(t, dir, "uno")
	return dir
}

// staleModifiedRepo es committedRepo con el índice desfasado y a.md con otro
// contenido: lo único que `git diff` contra HEAD debe devolver es `M a.md`.
func staleModifiedRepo(t *testing.T) (dir string, repo gitRepository, head string) {
	t.Helper()
	ctx := context.Background()
	dir = committedRepo(t)
	repo, err := openRepository(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if head, err = repo.head(ctx); err != nil || head == "" {
		t.Fatalf("head = %q, %v", head, err)
	}
	makeIndexStale(t, dir)
	writeRepoFile(t, dir, "a.md", "# A modificado\n")
	return dir, repo, head
}

var wantOnlyAModified = []nameStatus{{Status: "M", Path: "a.md"}}

// Con las marcas de tiempo del índice desfasadas, `git diff` reescribe `.git/index`
// aunque se invoque con GIT_OPTIONAL_LOCKS=0 (git 2.41): Verify promete solo leer,
// de modo que ninguna invocación de la vista de git puede dejar el índice real
// distinto. La tabla recorre todas las invocaciones de git del paquete.
func TestGitInvocationsNeverRewriteAStaleIndex(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name string
		run  func(t *testing.T, dir string, repo gitRepository, head string)
	}{
		{name: "rev-parse --show-toplevel", run: func(t *testing.T, dir string, _ gitRepository, _ string) {
			if _, err := openRepository(ctx, filepath.Join(dir, "a.md")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "rev-parse HEAD", run: func(t *testing.T, _ string, repo gitRepository, head string) {
			if got, err := repo.head(ctx); err != nil || got != head {
				t.Fatalf("head = %q, %v; quería %s", got, err, head)
			}
		}},
		{name: "status", run: func(t *testing.T, _ string, repo gitRepository, _ string) {
			entries, err := repo.status(ctx)
			if err != nil || len(entries) != 1 || entries[0] != (statusEntry{Path: "a.md", Code: " M"}) {
				t.Fatalf("status = %v, %v", entries, err)
			}
		}},
		{name: "upstream", run: func(t *testing.T, _ string, repo gitRepository, _ string) {
			if state := repo.upstream(ctx); state.Behind != 0 || len(state.Notes) != 1 {
				t.Fatalf("upstream = %+v", state)
			}
		}},
		{name: "cat-file -e", run: func(t *testing.T, _ string, repo gitRepository, head string) {
			if exists, err := repo.commitExists(ctx, head); err != nil || !exists {
				t.Fatalf("commitExists = %v, %v", exists, err)
			}
		}},
		{name: "diff --name-status", run: func(t *testing.T, _ string, repo gitRepository, head string) {
			if got, err := repo.diffNames(ctx, head); err != nil || !reflect.DeepEqual(got, wantOnlyAModified) {
				t.Fatalf("diffNames = %v, %v; quería %v", got, err, wantOnlyAModified)
			}
		}},
		{name: "ls-files --cached --others", run: func(t *testing.T, _ string, repo gitRepository, _ string) {
			got, err := repo.listFiles(ctx, "--cached", "--others", "--exclude-standard")
			if want := []string{"a.md", "docs/b.md", "notas ñ.md"}; err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("listFiles = %q, %v; quería %q", got, err, want)
			}
		}},
		{name: "ls-files --others", run: func(t *testing.T, _ string, repo gitRepository, _ string) {
			if got, err := repo.listFiles(ctx, "--others", "--exclude-standard"); err != nil || len(got) != 0 {
				t.Fatalf("listFiles = %q, %v", got, err)
			}
		}},
		{name: "ls-tree y cat-file blob", run: func(t *testing.T, _ string, repo gitRepository, head string) {
			if got, existed, err := repo.committedContent(ctx, head, "docs/b.md"); err != nil || !existed || string(got) != "# B\n" {
				t.Fatalf("committedContent = %q, %v, %v", got, existed, err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, repo, head := staleModifiedRepo(t)
			before := gitIndexState(t, dir)
			tt.run(t, dir, repo, head)
			if gitIndexState(t, dir) != before {
				t.Errorf("%s reescribió el índice de git con las marcas de tiempo desfasadas", tt.name)
			}
		})
	}
}

// El diff se ejecuta sobre una copia temporal del índice que está fuera del
// repositorio y se borra al terminar, también si git falla: ni queda nada en la
// carpeta temporal ni se crea nada junto al índice real.
func TestGitDiffNamesRemovesItsTemporaryIndexAlsoOnError(t *testing.T) {
	// Sin t.Parallel: fija la carpeta temporal del proceso (t.Setenv).
	ctx := context.Background()
	dir, repo, head := staleModifiedRepo(t)
	before := gitIndexState(t, dir)
	scratch := t.TempDir()
	for _, name := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(name, scratch)
	}
	leftovers := func() []string {
		entries, err := os.ReadDir(scratch)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		return names
	}

	t.Run("con éxito", func(t *testing.T) {
		if got, err := repo.diffNames(ctx, head); err != nil || !reflect.DeepEqual(got, wantOnlyAModified) {
			t.Fatalf("diffNames = %v, %v; quería %v", got, err, wantOnlyAModified)
		}
		if names := leftovers(); len(names) != 0 {
			t.Errorf("quedó en la carpeta temporal: %q", names)
		}
	})
	t.Run("con un commit que no existe", func(t *testing.T) {
		if _, err := repo.diffNames(ctx, strings.Repeat("0", 40)); err == nil {
			t.Fatal("diffNames contra un commit inexistente debía fallar")
		}
		if names := leftovers(); len(names) != 0 {
			t.Errorf("quedó en la carpeta temporal: %q", names)
		}
	})
	if gitIndexState(t, dir) != before {
		t.Error("el índice real o su directorio cambiaron")
	}
}

// El índice es el que git indica con `--git-path`, nunca el GIT_INDEX_FILE del
// proceso: una variable heredada que apunte a otro sitio no cambia ni el resultado
// ni crea ese fichero.
func TestGitDiffNamesIgnoresAnIndexFileFromTheProcessEnvironment(t *testing.T) {
	// Sin t.Parallel: fija una variable de entorno de git (t.Setenv).
	ctx := context.Background()
	dir, repo, head := staleModifiedRepo(t)
	before := gitIndexState(t, dir)
	foreign := filepath.Join(t.TempDir(), "indice-ajeno")
	t.Setenv("GIT_INDEX_FILE", foreign)
	if got, err := repo.diffNames(ctx, head); err != nil || !reflect.DeepEqual(got, wantOnlyAModified) {
		t.Fatalf("diffNames = %v, %v; quería %v", got, err, wantOnlyAModified)
	}
	if _, err := os.Stat(foreign); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("el GIT_INDEX_FILE del proceso se usó: %v", err)
	}
	// La huella la calcula git de la prueba, que sí obedecería la variable.
	if err := os.Unsetenv("GIT_INDEX_FILE"); err != nil {
		t.Fatal(err)
	}
	if gitIndexState(t, dir) != before {
		t.Error("el índice real cambió")
	}
}

// Un repositorio sin índice (se borró o nunca se creó) se compara como lo haría git:
// con el índice vacío. El diff no crea el índice ni nada junto a él.
func TestGitDiffNamesWithoutAnIndexDoesNotCreateOne(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := committedRepo(t)
	repo, err := openRepository(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	head, err := repo.head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	index := gitIndexPath(t, dir)
	if err := os.Remove(index); err != nil {
		t.Fatal(err)
	}
	before := gitIndexState(t, dir)
	got, err := repo.diffNames(ctx, head)
	if err != nil {
		t.Fatalf("diffNames sin índice: %v", err)
	}
	status := map[string]string{}
	for _, entry := range got {
		status[entry.Path] = entry.Status
	}
	if want := map[string]string{"a.md": "D", "docs/b.md": "D", "notas ñ.md": "D"}; !reflect.DeepEqual(status, want) {
		t.Errorf("diffNames sin índice = %v, quería %v (el índice vacío no sigue ningún fichero)", status, want)
	}
	if _, err := os.Stat(index); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("el diff creó el índice: %v", err)
	}
	if gitIndexState(t, dir) != before {
		t.Error("el diff creó algo junto al índice")
	}
}

// Con `core.splitIndex` el índice enlaza con `sharedindex.*` en el directorio git:
// la copia temporal sigue funcionando (git los resuelve desde el directorio git, no
// desde la ruta del índice) y ni el índice ni los índices compartidos cambian.
func TestGitDiffNamesWithASplitIndexDoesNotTouchTheGitDirectory(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	isolateGit(t)
	dir := newRepo(t, filepath.Join(t.TempDir(), "repo"))
	for i := 0; i < 40; i++ {
		writeRepoFile(t, dir, fmt.Sprintf("f%02d.md", i), fmt.Sprintf("fichero %d\n", i))
	}
	commitAll(t, dir, "uno")
	runGit(t, dir, "config", "core.splitIndex", "true")
	runGit(t, dir, "update-index", "--split-index")
	writeRepoFile(t, dir, "f00.md", "otro contenido\n")
	commitAll(t, dir, "dos") // el índice partido sigue enlazado al compartido, con entradas propias
	entries, err := os.ReadDir(filepath.Dir(gitIndexPath(t, dir)))
	if err != nil {
		t.Fatal(err)
	}
	shared := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "sharedindex.") {
			shared++
		}
	}
	if shared == 0 {
		t.Skip("este git no creó un índice compartido: no hay índice partido que comprobar")
	}

	repo, err := openRepository(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	head, err := repo.head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	makeIndexStale(t, dir)
	writeRepoFile(t, dir, "f01.md", "modificado\n")
	before := gitIndexState(t, dir)
	got, err := repo.diffNames(ctx, head)
	if want := []nameStatus{{Status: "M", Path: "f01.md"}}; err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("diffNames con índice partido = %v, %v; quería %v", got, err, want)
	}
	if gitIndexState(t, dir) != before {
		t.Error("el diff cambió el índice o los índices compartidos del directorio git")
	}
}

// En un worktree enlazado `.git` es un fichero y el índice vive en
// `<repositorio>/.git/worktrees/<nombre>/index`: se localiza con `--git-path`, no con
// una ruta fija. Ni el índice del worktree ni el del repositorio principal cambian.
func TestGitDiffNamesResolvesTheIndexOfALinkedWorktree(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	principal := committedRepo(t)
	linked := filepath.Join(t.TempDir(), "arbol")
	runGit(t, principal, "worktree", "add", "-q", "-b", "rama-enlazada", linked)
	repo, err := openRepository(ctx, linked)
	if err != nil {
		t.Fatal(err)
	}
	head, err := repo.head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if index := gitIndexPath(t, linked); index == filepath.Join(linked, ".git", "index") {
		t.Fatalf("el índice del worktree enlazado no es <raíz>/.git/index: %s", index)
	}
	makeIndexStale(t, linked)
	writeRepoFile(t, linked, "a.md", "# A modificado\n")
	before, principalBefore := gitIndexState(t, linked), gitIndexState(t, principal)
	if got, err := repo.diffNames(ctx, head); err != nil || !reflect.DeepEqual(got, wantOnlyAModified) {
		t.Fatalf("diffNames en un worktree enlazado = %v, %v; quería %v", got, err, wantOnlyAModified)
	}
	if gitIndexState(t, linked) != before || gitIndexState(t, principal) != principalBefore {
		t.Error("el diff cambió el índice del worktree o el del repositorio principal")
	}
}
