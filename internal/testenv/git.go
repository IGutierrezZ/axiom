package testenv

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// GitConfig es el gitconfig global de las pruebas: sin gc ni mantenimiento
// automático en segundo plano (que compiten con la limpieza del directorio
// temporal) y sin fsmonitor, caché de ficheros sin seguimiento ni fsync, que solo
// encarecen cada proceso git en Windows.
const GitConfig = `[gc]
	auto = 0
	autoDetach = false
[maintenance]
	auto = false
[core]
	fsmonitor = false
	untrackedCache = false
	fsync = none
	autocrlf = false
[init]
	defaultBranch = main
`

// Identidad fija de los commits de las pruebas.
const (
	GitAuthorName  = "Autor de Prueba"
	GitAuthorEmail = "autor@ejemplo.invalid"
)

// scratchRoot es la carpeta temporal del binario de pruebas que fijó IsolateGit.
// Vacía si IsolateGit no se llamó.
var scratchRoot string

// IsolateGit fija, para todo el binario de pruebas, un entorno de git ajeno al de
// la máquina: HOME y USERPROFILE vacíos, el gitconfig global GitConfig, sin
// configuración del sistema, una plantilla de `git init` vacía (sin hooks de
// ejemplo), autor y committer fijos, sin terminal y sin GIT_DIR, GIT_WORK_TREE ni
// GIT_INDEX_FILE heredados. Es para TestMain: al fijarse una sola vez, las pruebas
// no necesitan t.Setenv y pueden marcarse con t.Parallel().
//
// prefix nombra la carpeta temporal. Devuelve esa carpeta y una función que la
// borra; TestMain la llama después de m.Run.
func IsolateGit(prefix string) (root string, cleanup func(), err error) {
	root, err = os.MkdirTemp("", prefix+"-pruebas-*")
	if err != nil {
		return "", nil, fmt.Errorf("crear la carpeta temporal de las pruebas: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(root) }
	fail := func(err error) (string, func(), error) {
		cleanup()
		return "", nil, err
	}
	home := filepath.Join(root, "home")
	template := filepath.Join(root, "git-template")
	for _, dir := range []string{home, template} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fail(fmt.Errorf("crear %s: %w", dir, err))
		}
	}
	gitConfig := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(gitConfig, []byte(GitConfig), 0o644); err != nil {
		return fail(fmt.Errorf("escribir el gitconfig aislado: %w", err))
	}
	for key, value := range map[string]string{
		"HOME":                home,
		"USERPROFILE":         home,
		"XDG_CONFIG_HOME":     home,
		"GIT_CONFIG_GLOBAL":   gitConfig,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_TEMPLATE_DIR":    template,
		"GIT_AUTHOR_NAME":     GitAuthorName,
		"GIT_AUTHOR_EMAIL":    GitAuthorEmail,
		"GIT_COMMITTER_NAME":  GitAuthorName,
		"GIT_COMMITTER_EMAIL": GitAuthorEmail,
		"GIT_TERMINAL_PROMPT": "0",
	} {
		if err := os.Setenv(key, value); err != nil {
			return fail(fmt.Errorf("fijar %s: %w", key, err))
		}
	}
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_CONFIG_COUNT"} {
		if err := os.Unsetenv(key); err != nil {
			return fail(fmt.Errorf("vaciar %s: %w", key, err))
		}
	}
	scratchRoot = root
	return root, cleanup, nil
}

// RequireGit omite la prueba con -short o cuando git no está en el PATH.
func RequireGit(t testing.TB) {
	t.Helper()
	if testing.Short() {
		t.Skip("la prueba ejecuta git: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está disponible: %v", err)
	}
}

// TryGit ejecuta git en dir y devuelve su salida combinada sin el salto final.
func TryGit(dir string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	return strings.TrimRight(string(output), "\r\n"), err
}

// Git ejecuta git en dir y devuelve su salida sin el salto final; falla la prueba
// si git falla. Ejecuta git directamente, sin los adaptadores del producto, para
// que el oráculo de una prueba no dependa de lo que prueba.
func Git(t testing.TB, dir string, args ...string) string {
	t.Helper()
	output, err := TryGit(dir, args...)
	if err != nil {
		t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
	return output
}

// RepoTemplate es un árbol de repositorios git que se construye una sola vez por
// binario de pruebas y del que cada prueba recibe una copia. Copiar ficheros es
// mucho más barato que repetir `git init`, `add`, `commit` y `push`, sobre todo en
// Windows, donde lo caro es arrancar procesos.
//
// La construcción recibe una carpeta vacía y puede crear dentro varios
// repositorios (por ejemplo, un checkout y su remoto desnudo). Las rutas absolutas
// de esa carpeta que git guarda en los ficheros `config`, `gitdir` y `.git` se
// reescriben en cada copia, de modo que un `origin` de la plantilla apunta al
// remoto de la copia y nunca al de la plantilla.
type RepoTemplate struct {
	name  string
	build func(t testing.TB, dir string)

	once  sync.Once
	dir   string
	built bool
}

// NewRepoTemplate declara una plantilla. build se ejecuta en la primera llamada a
// Copy, con la prueba que la hace; si falla, esa prueba y las siguientes que pidan
// la plantilla fallan.
func NewRepoTemplate(name string, build func(t testing.TB, dir string)) *RepoTemplate {
	return &RepoTemplate{name: name, build: build}
}

// Copy devuelve una copia nueva de la plantilla en una carpeta temporal de la
// prueba. Requiere que TestMain haya llamado a IsolateGit.
func (r *RepoTemplate) Copy(t testing.TB) string {
	t.Helper()
	RequireGit(t)
	r.once.Do(func() {
		if scratchRoot == "" {
			t.Fatalf("plantilla %s: TestMain no llamó a testenv.IsolateGit", r.name)
		}
		dir, err := os.MkdirTemp(scratchRoot, "plantilla-"+r.name+"-*")
		if err != nil {
			t.Fatalf("plantilla %s: %v", r.name, err)
		}
		r.dir = dir
		r.build(t, dir)
		r.built = true
	})
	if !r.built {
		t.Fatalf("plantilla %s: la construcción falló en una prueba anterior", r.name)
	}
	target := t.TempDir()
	if err := copyTemplate(r.dir, target); err != nil {
		t.Fatalf("copiar la plantilla %s: %v", r.name, err)
	}
	return target
}

// copyTemplate copia el árbol src en dst, que ya existe y está vacío, y reescribe
// las rutas de src en los ficheros donde git guarda rutas absolutas.
func copyTemplate(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			if rel == "." {
				return nil
			}
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("%s: la plantilla solo admite ficheros y carpetas", path)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if holdsGitPaths(entry.Name()) {
			return rewriteFile(path, target, info.Mode().Perm(), src, dst)
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

// holdsGitPaths informa de si git puede guardar rutas absolutas en un fichero con
// ese nombre: la configuración (remotos, `core.worktree`), el `gitdir` de un
// worktree enlazado y el fichero `.git` que apunta a su carpeta de administración.
func holdsGitPaths(name string) bool {
	switch name {
	case "config", "gitdir", ".git":
		return true
	}
	return false
}

func rewriteFile(src, dst string, perm fs.FileMode, from, to string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	text := string(data)
	for _, pair := range [][2]string{
		{filepath.ToSlash(from), filepath.ToSlash(to)},
		{from, to},
		{strings.ReplaceAll(from, `\`, `\\`), strings.ReplaceAll(to, `\`, `\\`)},
	} {
		text = strings.ReplaceAll(text, pair[0], pair[1])
	}
	return os.WriteFile(dst, []byte(text), perm)
}

func copyFile(src, dst string, perm fs.FileMode) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, out.Close()) }()
	_, err = io.Copy(out, in)
	return err
}
