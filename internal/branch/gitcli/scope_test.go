package gitcli

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// commitFile escribe un fichero, lo añade y lo confirma, y devuelve el SHA del commit.
func commitFile(t *testing.T, repo, rel, content, message string) string {
	t.Helper()
	writeFile(t, filepath.Join(repo, filepath.FromSlash(rel)), content)
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", message)
	return runGit(t, repo, "rev-parse", "HEAD")
}

// REQ-25.51: RevList lee la lista de commits con sus padres, con y sin `--first-parent`, y
// admite revisiones excluidas con «^».
func TestRevListReadsCommitsWithTheirParents(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	base := runGit(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "switch", "-q", "-c", "lateral")
	side := commitFile(t, repo, "lateral.txt", "l\n", "lateral")
	runGit(t, repo, "switch", "-q", "main")
	first := commitFile(t, repo, "a.txt", "a\n", "primero")
	runGit(t, repo, "merge", "-q", "--no-ff", "-m", "fusion", "lateral")
	merge := runGit(t, repo, "rev-parse", "HEAD")
	git := New()

	t.Run("con --first-parent no entra la rama fusionada", func(t *testing.T) {
		entries, err := git.RevList(ctx, repo, true, "main", "^"+base)
		if err != nil {
			t.Fatalf("RevList: %v", err)
		}
		want := []branch.RevEntry{{SHA: merge, Parents: []string{first, side}}, {SHA: first, Parents: []string{base}}}
		if !reflect.DeepEqual(entries, want) {
			t.Errorf("RevList --first-parent = %+v, se esperaba %+v", entries, want)
		}
	})
	t.Run("sin --first-parent entran todos", func(t *testing.T) {
		entries, err := git.RevList(ctx, repo, false, "main", "^"+base)
		if err != nil {
			t.Fatalf("RevList: %v", err)
		}
		if len(entries) != 3 {
			t.Errorf("RevList = %+v, se esperaban 3 commits", entries)
		}
	})
	t.Run("el commit raíz no tiene padres", func(t *testing.T) {
		entries, err := git.RevList(ctx, repo, true, base)
		if err != nil || len(entries) != 1 || entries[0].SHA != base || len(entries[0].Parents) != 0 {
			t.Errorf("RevList = %+v, %v; se esperaba el commit raíz sin padres", entries, err)
		}
	})
	t.Run("una revisión que parece una opción se rechaza", func(t *testing.T) {
		if _, err := git.RevList(ctx, repo, true, "--all"); err == nil {
			t.Error("una revisión que empieza por «-» debía rechazarse")
		}
		if _, err := git.RevList(ctx, repo, true); err == nil {
			t.Error("sin revisiones debía fallar")
		}
	})
}

// REQ-25.51: CommitFiles devuelve los commits con sus ficheros en una sola lectura, íntegros
// con espacios y caracteres no ASCII, con las dos rutas de un renombrado y con `merge: true`
// en un commit de fusión (que git no lista con `--name-only`).
func TestCommitFilesListsFilesPerCommit(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	base := runGit(t, repo, "rev-parse", "HEAD")
	one := commitFile(t, repo, "mi carpeta/ñandú.go", "x\n", "uno")
	runGit(t, repo, "mv", "mi carpeta/ñandú.go", "mi carpeta/renombrado.go")
	runGit(t, repo, "commit", "-q", "-m", "renombrar")
	renamed := runGit(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "commit", "-q", "--allow-empty", "-m", "vacío")
	empty := runGit(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "switch", "-q", "-c", "lateral", base)
	commitFile(t, repo, "lateral.txt", "l\n", "lateral")
	runGit(t, repo, "switch", "-q", "main")
	runGit(t, repo, "merge", "-q", "--no-ff", "-m", "fusion", "lateral")
	merge := runGit(t, repo, "rev-parse", "HEAD")
	git := New()

	commits, err := git.CommitFiles(ctx, repo, "main", "^"+base)
	if err != nil {
		t.Fatalf("CommitFiles: %v", err)
	}
	byCommit := map[string]branch.CommitPaths{}
	for _, c := range commits {
		byCommit[c.SHA] = c
	}
	if len(commits) != 5 {
		t.Fatalf("CommitFiles devolvió %d commits, se esperaban 5: %+v", len(commits), commits)
	}
	if got := byCommit[one].Paths; !reflect.DeepEqual(got, []string{"mi carpeta/ñandú.go"}) {
		t.Errorf("rutas del primer commit = %q, se esperaba la ruta con espacio y no ASCII íntegra", got)
	}
	if got := byCommit[renamed].Paths; len(got) != 2 || !strings.Contains(strings.Join(got, "|"), "ñandú.go") || !strings.Contains(strings.Join(got, "|"), "renombrado.go") {
		t.Errorf("rutas del renombrado = %q, se esperaban las dos rutas", got)
	}
	if got := byCommit[empty]; len(got.Paths) != 0 || len(got.Parents) != 1 {
		t.Errorf("el commit vacío = %+v, se esperaba sin rutas y con un padre", got)
	}
	if got := byCommit[merge]; len(got.Parents) != 2 || len(got.Paths) != 0 {
		t.Errorf("el commit de fusión = %+v, se esperaban dos padres y ninguna ruta (merge: true lo deduce el motor)", got)
	}
	if _, err := git.CommitFiles(ctx, repo, "--all"); err == nil {
		t.Error("una revisión que empieza por «-» debía rechazarse")
	}
}

// REQ-25.53: BranchCreation lee la entrada más antigua del reflog de la rama local: válida si
// la rama se creó con `git branch`, no válida si se adoptó de `origin/<rama>` ni si no hay entrada.
func TestBranchCreationReadsTheOldestReflogEntry(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	start := runGit(t, repo, "rev-parse", "HEAD")
	runGit(t, repo, "branch", "--no-track", "feat/creada", "refs/heads/main")
	commitFile(t, repo, "a.txt", "a\n", "avanza")
	runGit(t, repo, "remote", "add", "origin", "https://ejemplo.invalid/repo.git")
	runGit(t, repo, "update-ref", "refs/remotes/origin/feat/adoptada", "HEAD")
	runGit(t, repo, "branch", "--track", "feat/adoptada", "refs/remotes/origin/feat/adoptada")
	git := New()

	t.Run("rama creada con branch", func(t *testing.T) {
		// Una entrada posterior no cambia la más antigua.
		runGit(t, repo, "update-ref", "-m", "avance", "refs/heads/feat/creada", "HEAD")
		sha, from, ok, err := git.BranchCreation(ctx, repo, "feat/creada")
		if err != nil || !ok || sha != start || from != "refs/heads/main" {
			t.Errorf("BranchCreation = %q, %q, %v, %v; se esperaba %q, %q, true", sha, from, ok, err, start, "refs/heads/main")
		}
	})
	t.Run("rama adoptada de origin", func(t *testing.T) {
		if _, _, ok, err := git.BranchCreation(ctx, repo, "feat/adoptada"); err != nil || ok {
			t.Errorf("BranchCreation de una rama adoptada = ok %v, %v; se esperaba no válida y sin error", ok, err)
		}
	})
	t.Run("rama sin reflog", func(t *testing.T) {
		if _, _, ok, err := git.BranchCreation(ctx, repo, "feat/no-existe"); err != nil || ok {
			t.Errorf("BranchCreation de una rama que no existe = ok %v, %v; se esperaba no válida y sin error", ok, err)
		}
	})
	t.Run("un nombre que parece una opción se rechaza", func(t *testing.T) {
		if _, _, _, err := git.BranchCreation(ctx, repo, "--all"); err == nil {
			t.Error("un nombre que empieza por «-» debía rechazarse")
		}
	})
}
