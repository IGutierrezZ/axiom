package gitcommon

import (
	"errors"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

// gitDirFixture crea un repositorio sintético, solo con ficheros: nada de lo que
// prueba GitDirFromFiles ni ReadHead necesita git.
func gitDirFixture(t *testing.T, head string) (root string) {
	t.Helper()
	root = t.TempDir()
	writeFile(t, filepath.Join(root, ".git", "HEAD"), head)
	return root
}

const (
	sampleSHA1   = "0123456789abcdef0123456789abcdef01234567"
	sampleSHA256 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

// REQ-25.21: el directorio git sale de leer la entrada `.git`, directorio o
// fichero `gitdir:`, sin lanzar procesos.
func TestGitDirFromFilesReadsADirectoryAndAGitdirFile(t *testing.T) {
	t.Run("un directorio .git", func(t *testing.T) {
		root := t.TempDir()
		mkdirAll(t, filepath.Join(root, ".git"))
		got, err := GitDirFromFiles(root)
		if err != nil {
			t.Fatalf("GitDirFromFiles(%q): %v", root, err)
		}
		wantSameDirectory(t, "GitDirFromFiles", got, filepath.Join(root, ".git"))
	})

	t.Run("un fichero gitdir con ruta absoluta", func(t *testing.T) {
		root := t.TempDir()
		target := mkdirAll(t, filepath.Join(t.TempDir(), "principal", ".git", "worktrees", "enlazado"))
		writeFile(t, filepath.Join(root, ".git"), "gitdir: "+filepath.ToSlash(target)+"\n")
		got, err := GitDirFromFiles(root)
		if err != nil {
			t.Fatalf("GitDirFromFiles(%q): %v", root, err)
		}
		wantSameDirectory(t, "GitDirFromFiles", got, target)
	})

	t.Run("un fichero gitdir con ruta relativa se resuelve contra la raíz", func(t *testing.T) {
		base := t.TempDir()
		root := mkdirAll(t, filepath.Join(base, "enlazado"))
		target := mkdirAll(t, filepath.Join(base, "principal", ".git", "worktrees", "enlazado"))
		writeFile(t, filepath.Join(root, ".git"), "gitdir: ../principal/.git/worktrees/enlazado\n")
		got, err := GitDirFromFiles(root)
		if err != nil {
			t.Fatalf("GitDirFromFiles(%q): %v", root, err)
		}
		wantSameDirectory(t, "GitDirFromFiles", got, target)
		if got != filepath.Clean(got) {
			t.Errorf("la ruta devuelta debe estar limpia, obtenida %q", got)
		}
	})

	t.Run("tolera CRLF, salto final y su ausencia", func(t *testing.T) {
		for name, content := range map[string]string{
			"CRLF":               "gitdir: ../datos\r\n",
			"salto de línea":     "gitdir: ../datos\n",
			"sin salto de línea": "gitdir: ../datos",
		} {
			base := t.TempDir()
			root := mkdirAll(t, filepath.Join(base, "enlazado"))
			target := mkdirAll(t, filepath.Join(base, "datos"))
			writeFile(t, filepath.Join(root, ".git"), content)
			got, err := GitDirFromFiles(root)
			if err != nil {
				t.Errorf("%s: GitDirFromFiles(%q): %v", name, root, err)
				continue
			}
			wantSameDirectory(t, name, got, target)
		}
	})

	t.Run("un worktree enlazado real", func(t *testing.T) {
		requireGit(t)
		main := newRepository(t)
		linked := filepath.Join(t.TempDir(), "enlazado")
		runGit(t, main, "worktree", "add", "-q", "--detach", linked, "HEAD")

		got, err := GitDirFromFiles(linked)
		if err != nil {
			t.Fatalf("GitDirFromFiles(%q): %v", linked, err)
		}
		want := filepath.FromSlash(gitOutput(t, linked, nil, "rev-parse", "--absolute-git-dir"))
		wantSameDirectory(t, "GitDirFromFiles en un worktree enlazado", got, want)
		if pathidentity.SameDirectory(got, filepath.Join(main, ".git")) {
			t.Errorf("el git dir de un worktree enlazado no es el del checkout principal: %q", got)
		}
	})
}

func TestGitDirFromFilesRefusesWhatIsNotAGitEntry(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) string
	}{
		{"sin entrada .git", func(t *testing.T) string { return t.TempDir() }},
		{"la ruta vacía", func(t *testing.T) string { return "" }},
		{"un fichero .git vacío", func(t *testing.T) string {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".git"), "")
			return root
		}},
		{"un fichero .git sin el prefijo gitdir", func(t *testing.T) string {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".git"), "otra cosa\n")
			return root
		}},
		{"un fichero .git con gitdir vacío", func(t *testing.T) string {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".git"), "gitdir: \n")
			return root
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := GitDirFromFiles(tc.setup(t)); err == nil {
				t.Errorf("GitDirFromFiles = %q, se esperaba un error", got)
			}
		})
	}
}

// design §4.6 y REQ-25.21: `ReadHead` distingue rama, HEAD desacoplado y forma
// desconocida a partir de `<gitdir>/HEAD`, tolerando CRLF y espacios finales.
func TestReadHeadClassifiesTheContentOfHEAD(t *testing.T) {
	tests := []struct {
		name         string
		head         string
		wantBranch   string
		wantDetached bool
		wantUnknown  bool
	}{
		{name: "una rama", head: "ref: refs/heads/main\n", wantBranch: "main"},
		{name: "una rama sin salto de línea", head: "ref: refs/heads/main", wantBranch: "main"},
		{name: "una rama con CRLF", head: "ref: refs/heads/main\r\n", wantBranch: "main"},
		{name: "una rama con espacios finales", head: "ref: refs/heads/main  \r\n", wantBranch: "main"},
		{name: "una rama con barras", head: "ref: refs/heads/inc/mi-cambio-core\n", wantBranch: "inc/mi-cambio-core"},
		{name: "HEAD desacoplado sha1", head: sampleSHA1 + "\n", wantDetached: true},
		{name: "HEAD desacoplado sha256", head: sampleSHA256 + "\n", wantDetached: true},
		{name: "HEAD desacoplado con CRLF", head: sampleSHA1 + "\r\n", wantDetached: true},
		{name: "backend reftable", head: "ref: refs/heads/.invalid\n", wantUnknown: true},
		{name: "una referencia que no es una rama", head: "ref: refs/remotes/origin/main\n", wantUnknown: true},
		{name: "una rama sin nombre", head: "ref: refs/heads/\n", wantUnknown: true},
		{name: "un fichero vacío", head: "", wantUnknown: true},
		{name: "contenido ajeno", head: "hola\n", wantUnknown: true},
		{name: "un hash demasiado corto", head: "abc123\n", wantUnknown: true},
		{name: "un hash con caracteres que no son hexadecimales", head: strings.Repeat("g", 40) + "\n", wantUnknown: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := gitDirFixture(t, tc.head)
			branch, detached, err := ReadHead(root)
			if tc.wantUnknown {
				if !errors.Is(err, ErrUnknownHead) {
					t.Fatalf("ReadHead = (%q, %v, %v), se esperaba ErrUnknownHead", branch, detached, err)
				}
				if branch != "" || detached {
					t.Errorf("una forma desconocida no devuelve rama ni HEAD desacoplado: (%q, %v)", branch, detached)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadHead: %v", err)
			}
			if branch != tc.wantBranch || detached != tc.wantDetached {
				t.Errorf("ReadHead = (%q, %v), se esperaba (%q, %v)", branch, detached, tc.wantBranch, tc.wantDetached)
			}
		})
	}
}

func TestReadHeadReportsAnUnreadableHEADAsUnknown(t *testing.T) {
	t.Run("sin fichero HEAD", func(t *testing.T) {
		root := t.TempDir()
		mkdirAll(t, filepath.Join(root, ".git"))
		if _, _, err := ReadHead(root); !errors.Is(err, ErrUnknownHead) {
			t.Errorf("ReadHead sin HEAD = %v, se esperaba ErrUnknownHead", err)
		}
	})
	t.Run("HEAD es un directorio", func(t *testing.T) {
		root := t.TempDir()
		mkdirAll(t, filepath.Join(root, ".git", "HEAD"))
		if _, _, err := ReadHead(root); !errors.Is(err, ErrUnknownHead) {
			t.Errorf("ReadHead con HEAD directorio = %v, se esperaba ErrUnknownHead", err)
		}
	})
	t.Run("sin repositorio", func(t *testing.T) {
		if _, _, err := ReadHead(t.TempDir()); !errors.Is(err, ErrUnknownHead) {
			t.Errorf("ReadHead sin .git = %v, se esperaba ErrUnknownHead", err)
		}
	})
}

// El HEAD se lee del git dir al que apunta el fichero `.git`, no de `.git/HEAD`.
func TestReadHeadFollowsAGitdirFile(t *testing.T) {
	base := t.TempDir()
	root := mkdirAll(t, filepath.Join(base, "enlazado"))
	writeFile(t, filepath.Join(base, "datos", "HEAD"), "ref: refs/heads/inc/x\r\n")
	writeFile(t, filepath.Join(root, ".git"), "gitdir: ../datos\r\n")

	branch, detached, err := ReadHead(root)
	if err != nil || branch != "inc/x" || detached {
		t.Errorf("ReadHead = (%q, %v, %v), se esperaba (\"inc/x\", false, nil)", branch, detached, err)
	}
}

// Con repositorios reales, la respuesta coincide con la de git, también en un
// worktree enlazado y antes del primer commit.
func TestReadHeadAgreesWithGitOnRealRepositories(t *testing.T) {
	requireGit(t)

	t.Run("rama del clon", func(t *testing.T) {
		repo := newRepository(t)
		branch, detached, err := ReadHead(repo)
		if err != nil || branch != "main" || detached {
			t.Errorf("ReadHead = (%q, %v, %v), se esperaba (\"main\", false, nil)", branch, detached, err)
		}
	})

	t.Run("HEAD desacoplado en el clon", func(t *testing.T) {
		repo := newRepository(t)
		runGit(t, repo, "checkout", "-q", "--detach")
		branch, detached, err := ReadHead(repo)
		if err != nil || branch != "" || !detached {
			t.Errorf("ReadHead = (%q, %v, %v), se esperaba (\"\", true, nil)", branch, detached, err)
		}
	})

	t.Run("rama con barras en un worktree enlazado", func(t *testing.T) {
		main := newRepository(t)
		linked := filepath.Join(t.TempDir(), "enlazado")
		runGit(t, main, "worktree", "add", "-q", "-b", "inc/mi-cambio", linked, "HEAD")

		branch, detached, err := ReadHead(linked)
		if err != nil || branch != "inc/mi-cambio" || detached {
			t.Errorf("ReadHead en el worktree enlazado = (%q, %v, %v), se esperaba (\"inc/mi-cambio\", false, nil)", branch, detached, err)
		}
		if want := gitOutput(t, linked, nil, "symbolic-ref", "--short", "HEAD"); branch != want {
			t.Errorf("ReadHead = %q, git dice %q", branch, want)
		}
		// El checkout principal conserva su propio HEAD.
		if mainBranch, _, err := ReadHead(main); err != nil || mainBranch != "main" {
			t.Errorf("ReadHead en el checkout principal = (%q, %v), se esperaba main", mainBranch, err)
		}
	})

	t.Run("worktree enlazado desacoplado", func(t *testing.T) {
		main := newRepository(t)
		linked := filepath.Join(t.TempDir(), "enlazado")
		runGit(t, main, "worktree", "add", "-q", "--detach", linked, "HEAD")
		branch, detached, err := ReadHead(linked)
		if err != nil || branch != "" || !detached {
			t.Errorf("ReadHead = (%q, %v, %v), se esperaba (\"\", true, nil)", branch, detached, err)
		}
	})

	t.Run("antes del primer commit", func(t *testing.T) {
		repo := t.TempDir()
		runGit(t, repo, "init", "-q", "-b", "principal")
		branch, detached, err := ReadHead(repo)
		if err != nil || branch != "principal" || detached {
			t.Errorf("ReadHead = (%q, %v, %v), se esperaba (\"principal\", false, nil)", branch, detached, err)
		}
	})
}
