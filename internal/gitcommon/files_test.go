package gitcommon

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// requireGit omite la prueba con -short o sin git instalado: las pruebas con
// repositorios reales ejecutan procesos externos (skill go-testing, regla 9 de
// tasks.md).
func requireGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("prueba con git real: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está instalado: %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v en %s: %v\n%s", args, dir, err, output)
	}
}

// newRepository crea un repositorio real, con un commit, en un directorio
// temporal. El entorno aislado lo fija TestMain.
func newRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "commit", "-q", "--allow-empty", "-m", "inicial")
	return dir
}

func mkdirAll(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("crear %s: %v", dir, err)
	}
	return dir
}

func writeFile(t *testing.T, path, content string) string {
	t.Helper()
	mkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escribir %s: %v", path, err)
	}
	return path
}

// wantRoot comprueba que GitRootFromFiles(start) halla el directorio want. Se
// compara por identidad: los directorios temporales de Windows tienen nombres
// cortos (8.3) y la ruta que devuelve la función conserva la grafía de start.
func wantRoot(t *testing.T, start, want string) {
	t.Helper()
	root, ok := GitRootFromFiles(start)
	if !ok {
		t.Fatalf("GitRootFromFiles(%q) no halló una raíz git, se esperaba %q", start, want)
	}
	if !pathidentity.SameDirectory(root, want) {
		t.Errorf("GitRootFromFiles(%q) = %q, se esperaba el directorio %q", start, root, want)
	}
}

// insideGit dice si algún ancestro de dir contiene una entrada `.git`, de modo
// que una prueba «fuera de git» no es fiable en esa máquina.
func insideGit(dir string) bool {
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// workspace REQ-25.27 y D-03.4: la raíz git de una ruta sale de ascender hasta una
// entrada `.git`, desde cualquier punto de un repositorio real.
func TestGitRootFromFilesFindsTheRootOfARealRepository(t *testing.T) {
	requireGit(t)
	repo := newRepository(t)
	sub := mkdirAll(t, filepath.Join(repo, "app", "web"))
	file := writeFile(t, filepath.Join(sub, "index.ts"), "export {}\n")

	tests := []struct {
		name  string
		start string
	}{
		{"la propia raíz", repo},
		{"una subcarpeta", sub},
		{"un fichero", file},
		{"un descendiente que aún no existe", filepath.Join(repo, "app", "nuevo", "main.go")},
		{"una ruta con segmentos redundantes", filepath.Join(sub, "..", "web")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantRoot(t, tc.start, repo)
		})
	}
}

// design D-03.4: en un worktree enlazado `.git` es un fichero, y la raíz es la del
// worktree, no la del checkout principal.
func TestGitRootFromFilesStopsAtALinkedWorktree(t *testing.T) {
	requireGit(t)
	main := newRepository(t)
	linked := filepath.Join(t.TempDir(), "enlazado")
	runGit(t, main, "worktree", "add", "-q", "--detach", linked, "HEAD")

	info, err := os.Lstat(filepath.Join(linked, ".git"))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("en un worktree enlazado .git debe ser un fichero regular: %v (%v)", info, err)
	}

	sub := mkdirAll(t, filepath.Join(linked, "src", "interno"))
	t.Run("desde la raíz del worktree enlazado", func(t *testing.T) { wantRoot(t, linked, linked) })
	t.Run("desde una subcarpeta del worktree enlazado", func(t *testing.T) { wantRoot(t, sub, linked) })
	t.Run("el checkout principal sigue siendo el suyo", func(t *testing.T) {
		wantRoot(t, main, main)
		root, _ := GitRootFromFiles(main)
		if pathidentity.SameDirectory(root, linked) {
			t.Errorf("el checkout principal y el worktree enlazado son directorios distintos: %q", root)
		}
	})
}

// El repositorio más cercano gana: una carpeta con su propio repositorio dentro de
// otro (un clon vendorizado, un submódulo) tiene su propia raíz.
func TestGitRootFromFilesNearestRepositoryWins(t *testing.T) {
	requireGit(t)
	outer := newRepository(t)
	inner := mkdirAll(t, filepath.Join(outer, "vendor", "lib"))
	runGit(t, inner, "init", "-q", "-b", "main")

	wantRoot(t, filepath.Join(inner, "src"), inner)
	wantRoot(t, filepath.Join(outer, "otra"), outer)
}

// La función solo mira entradas `.git`, sin procesos: estas pruebas no necesitan
// git y siguen corriendo con -short.
func TestGitRootFromFilesRecognisesAnyGitEntry(t *testing.T) {
	t.Run("un directorio .git", func(t *testing.T) {
		repo := t.TempDir()
		mkdirAll(t, filepath.Join(repo, ".git"))
		sub := mkdirAll(t, filepath.Join(repo, "a", "b"))
		wantRoot(t, repo, repo)
		wantRoot(t, sub, repo)
	})

	t.Run("un fichero .git de worktree enlazado", func(t *testing.T) {
		repo := t.TempDir()
		writeFile(t, filepath.Join(repo, ".git"), "gitdir: ../principal/.git/worktrees/enlazado\n")
		wantRoot(t, filepath.Join(repo, "src"), repo)
	})

	t.Run("la entrada más cercana gana", func(t *testing.T) {
		outer := t.TempDir()
		mkdirAll(t, filepath.Join(outer, ".git"))
		inner := filepath.Join(outer, "vendor", "lib")
		writeFile(t, filepath.Join(inner, ".git"), "gitdir: ../../.git/modules/lib\n")
		wantRoot(t, filepath.Join(inner, "src", "x.go"), inner)
		wantRoot(t, filepath.Join(outer, "vendor", "otro"), outer)
	})

	t.Run("una ruta relativa se resuelve contra el directorio actual", func(t *testing.T) {
		repo := t.TempDir()
		mkdirAll(t, filepath.Join(repo, ".git"))
		sub := mkdirAll(t, filepath.Join(repo, "a"))
		t.Chdir(sub)
		wantRoot(t, ".", repo)
		wantRoot(t, filepath.Join("..", "a", "nuevo"), repo)
	})
}

// Una ruta que no pertenece a ningún repositorio git no tiene raíz: la función lo
// dice con ok == false y no inventa un directorio.
func TestGitRootFromFilesOutsideAnyRepository(t *testing.T) {
	outside := t.TempDir()
	if insideGit(outside) {
		t.Skipf("el directorio temporal %s está dentro de un repositorio git", outside)
	}
	sub := mkdirAll(t, filepath.Join(outside, "a", "b"))

	for name, start := range map[string]string{
		"un directorio existente":   sub,
		"un fichero":                writeFile(t, filepath.Join(outside, "nota.txt"), "x\n"),
		"una ruta que no existe":    filepath.Join(outside, "no", "existe"),
		"la ruta vacía":             "",
		"un nombre parecido a .git": writeFile(t, filepath.Join(outside, ".github", "x.yml"), "x\n"),
	} {
		t.Run(name, func(t *testing.T) {
			if root, ok := GitRootFromFiles(start); ok || root != "" {
				t.Errorf("GitRootFromFiles(%q) = (%q, %v), se esperaba (\"\", false)", start, root, ok)
			}
		})
	}
}

// design §3, REQ-25.1 y G-7: las lecturas de ficheros no lanzan procesos, así que
// files.go no puede importar os/exec.
func TestFilesGoLaunchesNoProcesses(t *testing.T) {
	parsed, err := parser.ParseFile(token.NewFileSet(), "files.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("no se pudo analizar files.go: %v", err)
	}
	for _, spec := range parsed.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("importación ilegible %s: %v", spec.Path.Value, err)
		}
		if path == "os/exec" || path == "net" || path == "net/http" {
			t.Errorf("files.go importa %q: las lecturas de ficheros no lanzan procesos ni usan la red", path)
		}
	}
}
