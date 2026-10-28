package engine

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de RegisteredChanges (REQ-25.44, design D-16): el listado de los cambios que viven en
// worktrees registrados se lee solo de ficheros (el registro local del git common dir del planning
// home), de modo que estas pruebas montan los repositorios con una carpeta `.git` y no necesitan git.

// listingProject monta un repositorio con carpeta `.git` y devuelve su raíz y la ubicación de su
// planning home.
func listingProject(t *testing.T) (string, workspace.Location) {
	t.Helper()
	root := canonicalDirectory(t, t.TempDir())
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, workspace.Location{Master: root, WorkspaceRoot: root, PlanningHome: filepath.Join(root, "openspec")}
}

// listingWorktree crea la carpeta de un worktree y, si withChange, la del cambio dentro de su planning home.
func listingWorktree(t *testing.T, change string, withChange bool) string {
	t.Helper()
	dir := canonicalDirectory(t, t.TempDir())
	if withChange {
		if err := os.MkdirAll(filepath.Join(dir, "openspec", "changes", change), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// saveListing escribe en el common dir de root el registro del cambio con las ramas dadas del repositorio
// del planning home.
func saveListing(t *testing.T, root, change string, mode branch.Mode, branches ...branch.RecordBranch) {
	t.Helper()
	saveListingRepos(t, root, change, mode, branch.RecordRepo{
		Name: filepath.Base(root), GitRoot: root, CommonDir: commonDir(root), Role: branch.RepoChangeBranch, Branches: branches,
	})
}

func saveListingRepos(t *testing.T, root, change string, mode branch.Mode, repos ...branch.RecordRepo) {
	t.Helper()
	record := branch.Record{
		Schema: branch.RegistrySchema, Lane: branch.LaneSDD, Change: change, Mode: mode,
		ModeSource: branch.ModeSourceDefault, PlanningRepo: root, Repositories: repos,
	}
	if err := registry.New().Save(commonDir(root), record); err != nil {
		t.Fatalf("guardar el registro de %s: %v", change, err)
	}
}

func listingBranch(name string, kind branch.BranchKind, worktree string) branch.RecordBranch {
	return branch.RecordBranch{Name: name, Kind: kind, Worktree: worktree}
}

func listingChanges(t *testing.T, loc workspace.Location) ([]RegisteredChange, []string) {
	t.Helper()
	var processes [][]string
	gitcommon.Observe = func(dir string, args []string) { processes = append(processes, append([]string{dir}, args...)) }
	t.Cleanup(func() { gitcommon.Observe = nil })
	changes, warnings := Default().RegisteredChanges(context.Background(), loc)
	if len(processes) != 0 {
		t.Errorf("RegisteredChanges solo lee ficheros y no lanza procesos de git: %v", processes)
	}
	return changes, warnings
}

// REQ-25.44, D-16: por entrada devuelve nombre, rama, tipo, raíz del worktree, planning home y
// `changeRoot` si existe; las ramas de cambio y de archivado cuentan, la de rol no.
func TestRegisteredChangesListsTheChangeAndArchiveWorktrees(t *testing.T) {
	root, loc := listingProject(t)
	changeWorktree := listingWorktree(t, "x", true)
	archiveWorktree := listingWorktree(t, "x", true)
	roleWorktree := listingWorktree(t, "x", true)
	saveListing(t, root, "x", branch.ModeWorktree,
		listingBranch("feat/x", branch.KindChange, changeWorktree),
		listingBranch("feat/x-core", branch.KindRole, roleWorktree),
		listingBranch("feat/x-archive", branch.KindArchive, archiveWorktree),
	)

	got, warnings := listingChanges(t, loc)

	if len(warnings) != 0 {
		t.Errorf("no se esperaban avisos: %v", warnings)
	}
	want := []RegisteredChange{
		{Name: "x", Branch: "feat/x-archive", Kind: branch.KindArchive, WorktreeRoot: archiveWorktree,
			PlanningHome: filepath.Join(archiveWorktree, "openspec"), ChangeRoot: filepath.Join(archiveWorktree, "openspec", "changes", "x")},
		{Name: "x", Branch: "feat/x", Kind: branch.KindChange, WorktreeRoot: changeWorktree,
			PlanningHome: filepath.Join(changeWorktree, "openspec"), ChangeRoot: filepath.Join(changeWorktree, "openspec", "changes", "x")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RegisteredChanges() = %+v, se esperaba %+v", got, want)
	}
}

// El planning home de un worktree es el mismo camino relativo que el del checkout desde el que se
// consulta, y `changeRoot` solo está si la carpeta del cambio existe en él.
func TestRegisteredChangesKeepsThePlanningHomeRelativePathAndOmitsAMissingChangeRoot(t *testing.T) {
	root, _ := listingProject(t)
	loc := workspace.Location{Master: root, WorkspaceRoot: filepath.Join(root, "docs"), PlanningHome: filepath.Join(root, "docs", "openspec")}
	worktree := listingWorktree(t, "x", false)
	saveListing(t, root, "x", branch.ModeWorktree, listingBranch("feat/x", branch.KindChange, worktree))

	got, warnings := listingChanges(t, loc)

	if len(warnings) != 0 || len(got) != 1 {
		t.Fatalf("RegisteredChanges() = %+v, %v; se esperaba una entrada sin avisos", got, warnings)
	}
	if want := filepath.Join(worktree, "docs", "openspec"); got[0].PlanningHome != want {
		t.Errorf("PlanningHome = %s, se esperaba %s", got[0].PlanningHome, want)
	}
	if got[0].ChangeRoot != "" {
		t.Errorf("ChangeRoot = %s, se esperaba vacío: la carpeta del cambio no existe en el worktree", got[0].ChangeRoot)
	}
}

// REQ-25.44, escenario «Un worktree desaparecido se omite con aviso»: la entrada se omite y el aviso
// nombra la ruta ausente y la continuación.
func TestRegisteredChangesOmitsAMissingWorktreeWithAWarning(t *testing.T) {
	root, loc := listingProject(t)
	present := listingWorktree(t, "a", true)
	missing := filepath.Join(canonicalDirectory(t, t.TempDir()), "ya-no-existe")
	saveListing(t, root, "a", branch.ModeWorktree, listingBranch("feat/a", branch.KindChange, present))
	saveListing(t, root, "b", branch.ModeWorktree, listingBranch("feat/b", branch.KindChange, missing))

	got, warnings := listingChanges(t, loc)

	if len(got) != 1 || got[0].Name != "a" {
		t.Errorf("RegisteredChanges() = %+v, se esperaba solo el cambio a", got)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], missing) || !strings.Contains(warnings[0], "axiom branch ensure --change b") {
		t.Errorf("avisos = %q, se esperaba uno que nombre %s y la continuación `axiom branch ensure --change b`", warnings, missing)
	}
}

func TestRegisteredChangesIgnoresWhatHasNoWorktreeOfTheChange(t *testing.T) {
	root, loc := listingProject(t)
	other := canonicalDirectory(t, t.TempDir())
	if err := os.MkdirAll(filepath.Join(other, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	codeWorktree := listingWorktree(t, "x", true)
	saveListingRepos(t, root, "x", branch.ModeInPlace,
		branch.RecordRepo{Name: "planning", GitRoot: root, CommonDir: commonDir(root), Role: branch.RepoChangeBranch,
			Branches: []branch.RecordBranch{{Name: "feat/x", Kind: branch.KindChange, InPlace: true}, {Name: "feat/x-archive", Kind: branch.KindArchive}}},
		branch.RecordRepo{Name: "code", GitRoot: other, CommonDir: commonDir(other), Role: branch.RepoCode,
			Branches: []branch.RecordBranch{listingBranch("feat/x", branch.KindChange, codeWorktree)}},
	)

	got, warnings := listingChanges(t, loc)

	if len(got) != 0 || len(warnings) != 0 {
		t.Errorf("RegisteredChanges() = %+v, %v; en `in_place`, sin worktree y en otro repositorio no hay cambios en worktrees del planning home", got, warnings)
	}
}

// REQ-25.34, D-16: la agregación por nombre que comparten el dashboard y la TUI deja un solo cambio por
// nombre, con el worktree de archivado por delante del del cambio, y no cuenta lo que no tiene la carpeta
// del cambio en su worktree.
func TestRegisteredChangesByNameKeepsTheArchiveWorktreeAndSkipsMissingChangeRoots(t *testing.T) {
	root, loc := listingProject(t)
	changeWorktree := listingWorktree(t, "x", true)
	archiveWorktree := listingWorktree(t, "x", true)
	withoutChange := listingWorktree(t, "z", false)
	saveListing(t, root, "x", branch.ModeWorktree,
		listingBranch("feat/x", branch.KindChange, changeWorktree),
		listingBranch("feat/x-archive", branch.KindArchive, archiveWorktree),
	)
	saveListing(t, root, "z", branch.ModeWorktree, listingBranch("feat/z", branch.KindChange, withoutChange))

	got, warnings := Default().RegisteredChangesByName(context.Background(), loc)

	if len(warnings) != 0 {
		t.Errorf("no se esperaban avisos: %v", warnings)
	}
	if len(got) != 1 {
		t.Fatalf("RegisteredChangesByName() = %+v, se esperaba solo x", got)
	}
	if x := got["x"]; x.Branch != "feat/x-archive" || x.WorktreeRoot != archiveWorktree {
		t.Errorf("x = %+v, debía prevalecer el worktree de archivado %s", x, archiveWorktree)
	}
}

// Sin repositorio git, sin registro o con un registro ilegible no se lista nada y nunca falla; solo el
// ilegible deja un aviso con la continuación de borrarlo.
func TestRegisteredChangesWithoutARegistryReturnsNothing(t *testing.T) {
	plain := canonicalDirectory(t, t.TempDir())
	if got, warnings := listingChanges(t, workspace.Location{WorkspaceRoot: plain, PlanningHome: filepath.Join(plain, "openspec")}); got != nil || warnings != nil {
		t.Errorf("sin git: RegisteredChanges() = %+v, %v; se esperaba nil, nil", got, warnings)
	}
	_, loc := listingProject(t)
	if got, warnings := listingChanges(t, loc); len(got) != 0 || len(warnings) != 0 {
		t.Errorf("sin registro: RegisteredChanges() = %+v, %v; se esperaba nada", got, warnings)
	}
}

func TestRegisteredChangesWarnsAboutAnUnreadableRegistryFile(t *testing.T) {
	root, loc := listingProject(t)
	file := recordFile(commonDir(root), "sdd", "x")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("{no es json"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, warnings := listingChanges(t, loc)

	if len(got) != 0 {
		t.Errorf("RegisteredChanges() = %+v, no se esperaban cambios", got)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], file) || !strings.Contains(warnings[0], "axiom branch ensure --change") {
		t.Errorf("avisos = %q, se esperaba uno que nombre %s y la continuación de reconstruirlo", warnings, file)
	}
}
