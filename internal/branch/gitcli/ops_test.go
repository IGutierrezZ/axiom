package gitcli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// TestMain deja a los procesos git de las pruebas sin acceso a la configuración del
// usuario, con un autor fijo y sin red (tasks.md, regla 9), con el mismo patrón que
// internal/specbaseline, y fija el techo de descubrimiento de proyectos de las
// pruebas (workspace.DiscoveryCeilingOverride), como el resto de paquetes que usan
// repositorios temporales.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	root, err := os.MkdirTemp("", "gitcli-pruebas-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gitcli: crear la carpeta temporal de las pruebas: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(root) }()

	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "gitcli: crear el HOME aislado: %v\n", err)
		return 1
	}
	for key, value := range map[string]string{
		"HOME":                home,
		"USERPROFILE":         home,
		"GIT_CONFIG_GLOBAL":   filepath.Join(home, "gitconfig"),
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME":     "Autor de Prueba",
		"GIT_AUTHOR_EMAIL":    "autor@ejemplo.invalid",
		"GIT_COMMITTER_NAME":  "Autor de Prueba",
		"GIT_COMMITTER_EMAIL": "autor@ejemplo.invalid",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		if err := os.Setenv(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "gitcli: fijar %s: %v\n", key, err)
			return 1
		}
	}
	workspace.DiscoveryCeilingOverride = []string{os.TempDir()}
	return m.Run()
}

func requireGit(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("prueba con git real: se omite con -short")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git no está instalado: %v", err)
	}
}

// runGit ejecuta git directamente, sin pasar por gitcommon.Run, para que el oráculo de
// una prueba de las lecturas no dependa de ellas. Devuelve la salida sin el salto final.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v en %s: %v\n%s", args, dir, err, output)
	}
	return strings.TrimRight(string(output), "\r\n")
}

// newRepository crea un repositorio real en la rama main y con un commit.
func newRepository(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	commit(t, dir, "inicial")
	return dir
}

func commit(t *testing.T, dir, message string) {
	t.Helper()
	runGit(t, dir, "commit", "-q", "--allow-empty", "-m", message)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func wantSameDirectory(t *testing.T, what, got, want string) {
	t.Helper()
	if !filepath.IsAbs(got) || !pathidentity.SameDirectory(got, want) {
		t.Errorf("%s = %q, se esperaba el directorio absoluto %q", what, got, want)
	}
}

// TopLevel y CommonDir delegan en gitcommon: la raíz del checkout y su git common
// dir, que en un worktree enlazado es el del checkout principal.
func TestTopLevelAndCommonDirDelegateToGitcommon(t *testing.T) {
	ctx := context.Background()
	main := newRepository(t)
	linked := filepath.Join(t.TempDir(), "enlazado")
	runGit(t, main, "worktree", "add", "-q", "-b", "feat/x", linked)
	git := New()

	top, err := git.TopLevel(ctx, linked)
	if err != nil {
		t.Fatalf("TopLevel: %v", err)
	}
	wantSameDirectory(t, "TopLevel del worktree enlazado", top, linked)
	common, err := git.CommonDir(ctx, linked)
	if err != nil {
		t.Fatalf("CommonDir: %v", err)
	}
	wantSameDirectory(t, "CommonDir del worktree enlazado", common, filepath.Join(main, ".git"))

	// Fuera de un repositorio, el error conserva el centinela que el motor traduce a
	// `not_a_git_repository`.
	outside := insideNoRepository(t)
	if _, err := git.TopLevel(ctx, outside); !errors.Is(err, gitcommon.ErrNotGitRepository) {
		t.Errorf("TopLevel fuera de un repositorio = %v, se esperaba gitcommon.ErrNotGitRepository", err)
	}
	if _, err := git.CommonDir(ctx, outside); !errors.Is(err, gitcommon.ErrNotGitRepository) {
		t.Errorf("CommonDir fuera de un repositorio = %v, se esperaba gitcommon.ErrNotGitRepository", err)
	}
}

func TestResolveCommit(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	runGit(t, repo, "branch", "feat/x")
	runGit(t, repo, "tag", "-a", "-m", "anotada", "v1")
	head := runGit(t, repo, "rev-parse", "HEAD")
	git := New()

	for _, ref := range []string{"refs/heads/main", "refs/heads/feat/x", "refs/tags/v1", "HEAD", head} {
		t.Run("resuelve "+ref, func(t *testing.T) {
			sha, ok, err := git.ResolveCommit(ctx, repo, ref)
			if err != nil || !ok || sha != head {
				t.Errorf("ResolveCommit(%q) = %q, %v, %v; se esperaba %q, true, nil (también pela una etiqueta anotada)", ref, sha, ok, err, head)
			}
		})
	}
	t.Run("una referencia que no existe no es un error", func(t *testing.T) {
		sha, ok, err := git.ResolveCommit(ctx, repo, "refs/remotes/origin/main")
		if err != nil || ok || sha != "" {
			t.Errorf("ResolveCommit = %q, %v, %v; se esperaba una respuesta vacía, false y sin error", sha, ok, err)
		}
	})
	t.Run("un repositorio sin commits no tiene ninguna rama que resolver", func(t *testing.T) {
		empty := t.TempDir()
		runGit(t, empty, "init", "-q", "-b", "main")
		if sha, ok, err := git.ResolveCommit(ctx, empty, "refs/heads/main"); err != nil || ok {
			t.Errorf("ResolveCommit = %q, %v, %v; se esperaba false y sin error", sha, ok, err)
		}
	})
	t.Run("fuera de un repositorio es un error", func(t *testing.T) {
		if _, _, err := git.ResolveCommit(ctx, insideNoRepository(t), "refs/heads/main"); err == nil {
			t.Error("fuera de un repositorio git debía fallar")
		}
	})
}

// Una referencia que empieza por «-» se confundiría con una opción de git: las lecturas
// que reciben nombres la rechazan antes de lanzar nada.
func TestReadsRejectArgumentsThatLookLikeOptions(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	seen := observe(t)
	git := New()

	if _, _, err := git.ResolveCommit(ctx, repo, "--output=ajeno"); err == nil {
		t.Error("ResolveCommit debía rechazar una referencia que parece una opción")
	}
	if _, err := git.BranchExists(ctx, repo, "-D"); err == nil {
		t.Error("BranchExists debía rechazar un nombre que parece una opción")
	}
	if _, _, err := git.AheadBehind(ctx, repo, "--all", "refs/heads/main"); err == nil {
		t.Error("AheadBehind debía rechazar una base que parece una opción")
	}
	if _, _, err := git.AheadBehind(ctx, repo, "refs/heads/main", ""); err == nil {
		t.Error("AheadBehind debía rechazar una rama vacía")
	}
	if len(*seen) != 0 {
		t.Errorf("el rechazo es previo a lanzar git, pero el observador vio %v", *seen)
	}
}

func TestBranchExistsOnlyCountsBranches(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	runGit(t, repo, "branch", "feat/x")
	runGit(t, repo, "tag", "solo-etiqueta")
	git := New()

	for name, want := range map[string]bool{
		"main":            true,
		"feat/x":          true,
		"feat/y":          false,
		"solo-etiqueta":   false,
		"feat":            false,
		"origin/main":     false,
		"refs/heads/main": false,
	} {
		got, err := git.BranchExists(ctx, repo, name)
		if err != nil || got != want {
			t.Errorf("BranchExists(%q) = %v, %v; se esperaba %v", name, got, err, want)
		}
	}
	if _, err := git.BranchExists(ctx, insideNoRepository(t), "main"); err == nil {
		t.Error("fuera de un repositorio git debía fallar")
	}
}

// Worktrees lista el checkout principal y los enlazados, con la ruta absoluta (con
// espacios incluidos), el commit y la rama o el HEAD desacoplado.
func TestWorktreesListsTheMainAndLinkedWorktrees(t *testing.T) {
	ctx := context.Background()
	main := newRepository(t)
	parent := t.TempDir()
	onBranch := filepath.Join(parent, "con espacio", "rama")
	detached := filepath.Join(parent, "desacoplado")
	runGit(t, main, "worktree", "add", "-q", "-b", "feat/x", onBranch)
	runGit(t, main, "worktree", "add", "-q", "--detach", detached)
	head := runGit(t, main, "rev-parse", "HEAD")

	// Desde un worktree enlazado se ve la misma lista que desde el principal.
	for name, from := range map[string]string{"principal": main, "enlazado": onBranch} {
		t.Run("desde el "+name, func(t *testing.T) {
			entries, err := New().Worktrees(ctx, from)
			if err != nil {
				t.Fatalf("Worktrees: %v", err)
			}
			if len(entries) != 3 {
				t.Fatalf("entradas = %+v, se esperaban tres", entries)
			}
			wantSameDirectory(t, "checkout principal", entries[0].Path, main)
			if entries[0].Branch != "main" || entries[0].Detached || entries[0].Head != head {
				t.Errorf("checkout principal = %+v, se esperaba la rama main en %s", entries[0], head)
			}
			byPath := map[string]int{}
			for i, entry := range entries[1:] {
				if pathidentity.SameDirectory(entry.Path, onBranch) {
					byPath["rama"] = i + 1
				}
				if pathidentity.SameDirectory(entry.Path, detached) {
					byPath["desacoplado"] = i + 1
				}
			}
			if i, found := byPath["rama"]; !found || entries[i].Branch != "feat/x" || entries[i].Detached || !filepath.IsAbs(entries[i].Path) {
				t.Errorf("el worktree con espacios en la ruta = %+v (%v), se esperaba feat/x", entries, byPath)
			}
			if i, found := byPath["desacoplado"]; !found || entries[i].Branch != "" || !entries[i].Detached || entries[i].Head != head {
				t.Errorf("el worktree desacoplado = %+v (%v), se esperaba sin rama y desacoplado en %s", entries, byPath, head)
			}
		})
	}
	if _, err := New().Worktrees(ctx, insideNoRepository(t)); err == nil {
		t.Error("fuera de un repositorio git debía fallar")
	}
}

// parseWorktrees tolera CRLF, los atributos que no usa (`locked`, `prunable`, `bare`) y
// la ausencia de la línea en blanco final.
func TestParseWorktreesToleratesExtraAttributesAndLineEndings(t *testing.T) {
	output := "worktree C:/repos/a\r\nHEAD 1111111111111111111111111111111111111111\r\nbranch refs/heads/main\r\n\r\n" +
		"worktree C:/repos/b\r\nHEAD 2222222222222222222222222222222222222222\r\ndetached\r\nlocked razón\r\n\r\n" +
		"worktree C:/repos/c\nbare\n\n" +
		"worktree C:/repos/d\nHEAD 3333333333333333333333333333333333333333\nbranch refs/heads/feat/x\nprunable gitdir file points to non-existent location"
	entries, err := parseWorktrees(output)
	if err != nil {
		t.Fatalf("parseWorktrees: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("entradas = %+v, se esperaban cuatro", entries)
	}
	if entries[0].Branch != "main" || entries[0].Path != filepath.Clean(filepath.FromSlash("C:/repos/a")) {
		t.Errorf("primera entrada = %+v", entries[0])
	}
	if !entries[1].Detached || entries[1].Branch != "" || entries[1].Head != strings.Repeat("2", 40) {
		t.Errorf("segunda entrada = %+v", entries[1])
	}
	if entries[2].Branch != "" || entries[2].Detached {
		t.Errorf("un worktree desnudo no tiene rama ni HEAD desacoplado: %+v", entries[2])
	}
	if entries[3].Branch != "feat/x" {
		t.Errorf("cuarta entrada = %+v", entries[3])
	}
	if _, err := parseWorktrees("HEAD 1111\nbranch refs/heads/x\n"); err == nil {
		t.Error("una entrada que no empieza por `worktree` es un error de formato")
	}
}

func TestDirtyPaths(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	writeFile(t, filepath.Join(repo, "rastreado.txt"), "uno\n")
	writeFile(t, filepath.Join(repo, "origen.txt"), "dos\n")
	runGit(t, repo, "add", ".")
	commit(t, repo, "ficheros")
	git := New()

	t.Run("un árbol limpio no tiene rutas", func(t *testing.T) {
		paths, err := git.DirtyPaths(ctx, repo)
		if err != nil || len(paths) != 0 {
			t.Errorf("DirtyPaths = %v, %v; se esperaba vacío", paths, err)
		}
	})
	t.Run("un fichero sin seguimiento cuenta, también con espacios y acentos", func(t *testing.T) {
		writeFile(t, filepath.Join(repo, "ñandú con espacios.txt"), "x\n")
		paths, err := git.DirtyPaths(ctx, repo)
		if err != nil || len(paths) != 1 || paths[0] != "ñandú con espacios.txt" {
			t.Errorf("DirtyPaths = %q, %v; se esperaba la ruta tal cual, sin comillas", paths, err)
		}
		if err := os.Remove(filepath.Join(repo, "ñandú con espacios.txt")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("una carpeta sin seguimiento cuenta una vez, con su barra final", func(t *testing.T) {
		writeFile(t, filepath.Join(repo, "carpeta nueva", "a.txt"), "x\n")
		writeFile(t, filepath.Join(repo, "carpeta nueva", "b.txt"), "x\n")
		paths, err := git.DirtyPaths(ctx, repo)
		if err != nil || len(paths) != 1 || paths[0] != "carpeta nueva/" {
			t.Errorf("DirtyPaths = %q, %v; se esperaba la carpeta una sola vez", paths, err)
		}
		if err := os.RemoveAll(filepath.Join(repo, "carpeta nueva")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("un cambio sin confirmar, uno preparado y un renombrado, sin repetir el origen", func(t *testing.T) {
		writeFile(t, filepath.Join(repo, "rastreado.txt"), "cambiado\n")
		writeFile(t, filepath.Join(repo, "nuevo.txt"), "n\n")
		runGit(t, repo, "add", "nuevo.txt")
		runGit(t, repo, "mv", "origen.txt", "destino.txt")
		paths, err := git.DirtyPaths(ctx, repo)
		if err != nil {
			t.Fatalf("DirtyPaths: %v", err)
		}
		want := map[string]bool{"rastreado.txt": true, "nuevo.txt": true, "destino.txt": true}
		if len(paths) != len(want) {
			t.Errorf("DirtyPaths = %q, se esperaban %d rutas (el origen del renombrado no cuenta)", paths, len(want))
		}
		for _, path := range paths {
			if !want[path] {
				t.Errorf("ruta inesperada %q en %q", path, paths)
			}
		}
	})
	t.Run("fuera de un repositorio es un error", func(t *testing.T) {
		if _, err := git.DirtyPaths(ctx, insideNoRepository(t)); err == nil {
			t.Error("fuera de un repositorio git debía fallar")
		}
	})
}

// AheadBehind cuenta, con referencias locales, los commits de la rama que no tiene la
// base (adelanto) y los de la base que no tiene la rama (atraso).
func TestAheadBehindCountsCommitsInBothDirections(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	runGit(t, repo, "branch", "feat/x")
	runGit(t, repo, "branch", "feat/igual")
	for i := 0; i < 3; i++ {
		commit(t, repo, fmt.Sprintf("en main %d", i))
	}
	runGit(t, repo, "switch", "-q", "feat/x")
	for i := 0; i < 2; i++ {
		commit(t, repo, fmt.Sprintf("en feat/x %d", i))
	}
	git := New()

	for name, tc := range map[string]struct {
		base, branch          string
		wantAhead, wantBehind int
	}{
		"adelanto y atraso":   {"refs/heads/main", "refs/heads/feat/x", 2, 3},
		"sin diferencias":     {"refs/heads/main", "refs/heads/main", 0, 0},
		"solo atraso":         {"refs/heads/main", "refs/heads/feat/igual", 0, 3},
		"solo adelanto":       {"refs/heads/feat/igual", "refs/heads/feat/x", 2, 0},
		"con identificadores": {runGit(t, repo, "rev-parse", "main"), runGit(t, repo, "rev-parse", "feat/x"), 2, 3},
	} {
		t.Run(name, func(t *testing.T) {
			ahead, behind, err := git.AheadBehind(ctx, repo, tc.base, tc.branch)
			if err != nil || ahead != tc.wantAhead || behind != tc.wantBehind {
				t.Errorf("AheadBehind = %d, %d, %v; se esperaba adelanto %d y atraso %d", ahead, behind, err, tc.wantAhead, tc.wantBehind)
			}
		})
	}
	if _, _, err := git.AheadBehind(ctx, repo, "refs/heads/main", "refs/heads/no-existe"); err == nil {
		t.Error("una rama que no existe debía fallar")
	}
}

// REQ-25.1 y G-7: las lecturas lanzan solo la lista cerrada de subcomandos de lectura,
// con un remoto configurado, y ninguna es una operación remota.
func TestReadsLaunchOnlyTheClosedListOfReadSubcommands(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	runGit(t, repo, "remote", "add", "origin", "https://ejemplo.invalid/repo.git")
	runGit(t, repo, "branch", "feat/x")
	seen := observe(t)
	git := New()

	_, _ = git.TopLevel(ctx, repo)
	_, _ = git.CommonDir(ctx, repo)
	_, _, _ = git.ResolveCommit(ctx, repo, "refs/remotes/origin/main")
	_, _ = git.BranchExists(ctx, repo, "feat/x")
	_, _ = git.Worktrees(ctx, repo)
	_, _ = git.DirtyPaths(ctx, repo)
	_, _, _ = git.AheadBehind(ctx, repo, "refs/heads/main", "refs/heads/feat/x")
	_, _, _ = git.TreeDirs(ctx, repo, "refs/heads/main", "openspec/changes")

	allowed := map[string]bool{"rev-parse": true, "worktree": true, "status": true, "rev-list": true, "ls-tree": true}
	if len(*seen) != 8 {
		t.Fatalf("el observador vio %d invocaciones, se esperaban 8: %v", len(*seen), *seen)
	}
	for _, invocation := range *seen {
		if !allowed[invocation[1]] {
			t.Errorf("una lectura lanzó el subcomando %q, fuera de la lista cerrada: %v", invocation[1], invocation)
		}
		if invocation[1] == "worktree" && invocation[2] != "list" {
			t.Errorf("la lectura de worktrees solo usa `worktree list`: %v", invocation)
		}
	}
}

// observe instala un observador que anota cada invocación de git y lo retira al
// terminar la prueba.
func observe(t *testing.T) *[][]string {
	t.Helper()
	var seen [][]string
	gitcommon.Observe = func(dir string, args []string) {
		seen = append(seen, append([]string{dir}, args...))
	}
	t.Cleanup(func() { gitcommon.Observe = nil })
	return &seen
}

// insideNoRepository devuelve un directorio fuera de cualquier repositorio git, u omite
// la prueba si el temporal de la máquina está dentro de uno.
func insideNoRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for current := dir; ; {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			t.Skipf("el directorio temporal %s está dentro de un repositorio git", dir)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return dir
		}
		current = parent
	}
}

// changesTree crea un repositorio con `openspec/changes/` como lo deja el archivado:
// carpetas de cambios (una con espacio y otra con una letra no ASCII), `archive/`, un
// fichero suelto y un submódulo, todo confirmado en main.
func changesTree(t *testing.T) string {
	t.Helper()
	repo := newRepository(t)
	for _, name := range []string{"archive", "x", "mi cambio", "ñandú"} {
		writeFile(t, filepath.Join(repo, "openspec", "changes", name, "proposal.md"), "contenido\n")
	}
	writeFile(t, filepath.Join(repo, "openspec", "changes", ".gitkeep"), "")
	writeFile(t, filepath.Join(repo, "openspec", "solo-ficheros", "a.md"), "a\n")
	runGit(t, repo, "add", "-A")
	// Una entrada de tipo commit (un submódulo) no es un directorio del árbol.
	runGit(t, repo, "update-index", "--add", "--cacheinfo", "160000,"+runGit(t, repo, "rev-parse", "HEAD")+",openspec/changes/submodulo")
	commit(t, repo, "cambios")
	return repo
}

// design D-17: `TreeDirs` lista los subdirectorios inmediatos de una carpeta en el
// árbol de una referencia (`git ls-tree -z`): solo entradas de tipo `tree`, en el orden
// de git, con nombres con espacios y no ASCII íntegros, y dice si la carpeta existe.
func TestTreeDirsListsTheImmediateDirectoriesOfAFolder(t *testing.T) {
	ctx := context.Background()
	repo := changesTree(t)
	head := runGit(t, repo, "rev-parse", "HEAD")
	git := New()

	t.Run("solo los directorios, sin ficheros ni submódulos", func(t *testing.T) {
		dirs, exists, err := git.TreeDirs(ctx, repo, "refs/heads/main", "openspec/changes")
		if err != nil || !exists {
			t.Fatalf("TreeDirs = %v, %v, %v; se esperaba que la carpeta exista", dirs, exists, err)
		}
		if want := []string{"archive", "mi cambio", "x", "ñandú"}; !reflect.DeepEqual(dirs, want) {
			t.Errorf("directorios = %q, se esperaba %q", dirs, want)
		}
	})
	t.Run("la referencia puede ser un commit", func(t *testing.T) {
		dirs, exists, err := git.TreeDirs(ctx, repo, head, "openspec/changes")
		if err != nil || !exists || len(dirs) != 4 {
			t.Fatalf("TreeDirs(%s) = %q, %v, %v", head, dirs, exists, err)
		}
	})
	t.Run("no es recursivo", func(t *testing.T) {
		dirs, _, err := git.TreeDirs(ctx, repo, head, "openspec")
		if err != nil || !reflect.DeepEqual(dirs, []string{"changes", "solo-ficheros"}) {
			t.Errorf("TreeDirs(openspec) = %q, %v; se esperaban solo sus hijos directos", dirs, err)
		}
	})
	t.Run("desde una subcarpeta la ruta sigue siendo la de la raíz", func(t *testing.T) {
		dirs, exists, err := git.TreeDirs(ctx, filepath.Join(repo, "openspec"), head, "openspec/changes")
		if err != nil || !exists || len(dirs) != 4 {
			t.Errorf("TreeDirs desde openspec/ = %q, %v, %v", dirs, exists, err)
		}
	})
	t.Run("una carpeta con solo ficheros existe y no tiene directorios", func(t *testing.T) {
		dirs, exists, err := git.TreeDirs(ctx, repo, head, "openspec/solo-ficheros")
		if err != nil || !exists || len(dirs) != 0 {
			t.Errorf("TreeDirs = %q, %v, %v; se esperaba existente y sin directorios", dirs, exists, err)
		}
	})
	t.Run("una ruta que no existe no es un error", func(t *testing.T) {
		for _, missing := range []string{"nope/changes", "openspec/changes/no-existe", "openspec/changes/.gitkeep"} {
			dirs, exists, err := git.TreeDirs(ctx, repo, head, missing)
			if err != nil || exists || len(dirs) != 0 {
				t.Errorf("TreeDirs(%q) = %q, %v, %v; se esperaba inexistente sin error (un fichero no es una carpeta)", missing, dirs, exists, err)
			}
		}
	})
	t.Run("no lee el árbol de trabajo", func(t *testing.T) {
		writeFile(t, filepath.Join(repo, "openspec", "changes", "sin-confirmar", "a.md"), "a\n")
		dirs, _, err := git.TreeDirs(ctx, repo, head, "openspec/changes")
		if err != nil || len(dirs) != 4 {
			t.Errorf("TreeDirs = %q, %v; una carpeta sin confirmar no está en el árbol de la referencia", dirs, err)
		}
	})
}

// Un nombre sin sentido para git nunca llega a lanzarlo, y una referencia que no
// existe es un error que nombra el comando y el directorio.
func TestTreeDirsRejectsArgumentsThatGitWouldReadAsOptionsOrMagic(t *testing.T) {
	ctx := context.Background()
	repo := changesTree(t)
	seen := observe(t)
	git := New()

	for _, relPath := range []string{"", ".", "/openspec", "../fuera", "openspec/../x", "-openspec", ":(top)openspec", "openspec\x00x"} {
		if _, _, err := git.TreeDirs(ctx, repo, "refs/heads/main", relPath); err == nil {
			t.Errorf("TreeDirs con la ruta %q debía fallar", relPath)
		}
	}
	for _, rev := range []string{"", "-r"} {
		if _, _, err := git.TreeDirs(ctx, repo, rev, "openspec/changes"); err == nil {
			t.Errorf("TreeDirs con la referencia %q debía fallar", rev)
		}
	}
	if len(*seen) != 0 {
		t.Errorf("un argumento inválido se rechaza antes de lanzar git, pero el observador vio %v", *seen)
	}

	_, _, err := git.TreeDirs(ctx, repo, "refs/heads/no-existe", "openspec/changes")
	if err == nil || !strings.Contains(err.Error(), "ls-tree") || !strings.Contains(err.Error(), repo) {
		t.Errorf("una referencia inexistente da un error que nombra el comando y %s: %v", repo, err)
	}
}

// Los nombres con caracteres que git trataría de forma especial en una salida
// entrecomillada o en un patrón salen íntegros: la salida es `-z` y la ruta es un prefijo
// literal, no un patrón (git ls-tree no expande comodines).
func TestTreeDirsKeepsUnusualNamesIntact(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	names := []string{"con'apóstrofe", "a#b", "100%", "[corchetes]", "a{b}", "a;b", "a&b", "a$b", "a  b"}
	for _, name := range names {
		writeFile(t, filepath.Join(repo, "openspec", "changes", name, "f"), "f\n")
	}
	writeFile(t, filepath.Join(repo, "raro[dir]", "changes", "uno", "f"), "f\n")
	// `rarod` coincidiría con `raro[dir]` si la ruta se leyese como un patrón.
	writeFile(t, filepath.Join(repo, "rarod", "changes", "falso", "f"), "f\n")
	runGit(t, repo, "add", "-A")
	commit(t, repo, "nombres raros")
	git := New()

	t.Run("los nombres de las carpetas", func(t *testing.T) {
		dirs, exists, err := git.TreeDirs(ctx, repo, "refs/heads/main", "openspec/changes")
		if err != nil || !exists {
			t.Fatalf("TreeDirs = %q, %v, %v", dirs, exists, err)
		}
		for _, name := range names {
			if !slices.Contains(dirs, name) {
				t.Errorf("falta %q entre %q", name, dirs)
			}
		}
	})
	t.Run("la ruta se toma literalmente, sin comodines", func(t *testing.T) {
		dirs, exists, err := git.TreeDirs(ctx, repo, "refs/heads/main", "raro[dir]/changes")
		if err != nil || !exists || !reflect.DeepEqual(dirs, []string{"uno"}) {
			t.Errorf("TreeDirs(raro[dir]/changes) = %q, %v, %v; se esperaba solo [uno]", dirs, exists, err)
		}
	})
}
