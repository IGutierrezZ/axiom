package sddstatus

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Pruebas de S9-c4 (REQ-25.44, design D-16): `status` y `continue` resuelven los cambios que viven en
// los worktrees que `axiom branch ensure` registró. El checkout principal y los worktrees se montan solo
// con ficheros (un `.git` por repositorio y un `commondir` por worktree enlazado) y los registros se
// escriben con registry.Store, de modo que ninguna prueba necesita git.

// mainCheckout monta el checkout principal de un monorepo embebido: sin ningún cambio activo, con su
// `.git` y en la rama de integración.
func mainCheckout(t *testing.T) string {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	switchTo(t, ws.Master, "main")
	return ws.Master
}

// linkedWorktree monta un worktree enlazado del checkout principal main con el mismo axiom.yaml y, si
// tasks no es vacío, el cambio dado con ese plan de tareas.
func linkedWorktree(t *testing.T, main, change, tasks string) string {
	t.Helper()
	dir := realPath(t, t.TempDir())
	gitDir := filepath.Join(dir, ".gitdir")
	write(t, filepath.Join(dir, ".git"), "gitdir: "+gitDir+"\n")
	write(t, filepath.Join(gitDir, "commondir"), filepath.Join(main, ".git")+"\n")
	write(t, filepath.Join(gitDir, "HEAD"), "ref: refs/heads/feat/"+change+"\n")
	write(t, filepath.Join(dir, "axiom.yaml"), readFile(t, filepath.Join(main, "axiom.yaml")))
	if tasks != "" {
		seedReadyChange(t, dir, change, tasks)
	}
	return dir
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// saveChangeRegistry escribe el registro local del cambio dado en el git common dir del planning home.
func saveChangeRegistry(t *testing.T, planningRoot, change string, repos ...branch.RecordRepo) {
	t.Helper()
	common, err := gitcommon.CommonDirFromFiles(planningRoot)
	if err != nil {
		t.Fatal(err)
	}
	record := branch.Record{
		Schema: branch.RegistrySchema, Lane: branch.LaneSDD, Change: change,
		Mode: branch.ModeWorktree, ModeSource: branch.ModeSourceDefault, PlanningRepo: planningRoot, Repositories: repos,
	}
	if err := registry.New().Save(common, record); err != nil {
		t.Fatalf("guardar el registro de %s: %v", change, err)
	}
}

func kindBranch(name string, kind branch.BranchKind, worktree string) branch.RecordBranch {
	return branch.RecordBranch{Name: name, Kind: kind, Worktree: worktree}
}

// registerChange escribe el registro del cambio dado en el common dir del checkout principal.
func registerChange(t *testing.T, main, change string, branches ...branch.RecordBranch) {
	t.Helper()
	common, err := gitcommon.CommonDirFromFiles(main)
	if err != nil {
		t.Fatal(err)
	}
	repo := branch.RecordRepo{Name: filepath.Base(main), GitRoot: main, CommonDir: common, Role: branch.RepoChangeBranch, Branches: branches}
	saveChangeRegistry(t, main, change, repo)
}

func wantResolvedIn(t *testing.T, status Status, root, change string) {
	t.Helper()
	if status.ChangeRoot == nil || *status.ChangeRoot != filepath.Join(root, "openspec", "changes", change) {
		got := "<nil>"
		if status.ChangeRoot != nil {
			got = *status.ChangeRoot
		}
		t.Errorf("changeRoot = %s, se esperaba %s", got, filepath.Join(root, "openspec", "changes", change))
	}
	if status.ActionContext.WorkspaceRoot != root {
		t.Errorf("workspaceRoot = %s, se esperaba %s", status.ActionContext.WorkspaceRoot, root)
	}
	if want := filepath.Join(root, "openspec"); status.PlanningHome.Path != want {
		t.Errorf("planningHome = %s, se esperaba %s", status.PlanningHome.Path, want)
	}
	for _, path := range status.ArtifactPaths.Tasks {
		if !strings.HasPrefix(path, root) {
			t.Errorf("artifactPaths.tasks = %v debe colgar de %s", status.ArtifactPaths.Tasks, root)
		}
	}
}

func notesAbout(status Status, fragment string) []string {
	var matching []string
	for _, note := range status.Notes {
		if strings.Contains(note, fragment) {
			matching = append(matching, note)
		}
	}
	return matching
}

// REQ-25.44, escenario «Status desde el checkout principal resuelve el worktree»: con nombre, el
// cambio que solo vive en su worktree registrado se resuelve en él.
func TestResolveFromTheMainCheckoutResolvesTheRegisteredWorktree(t *testing.T) {
	main := mainCheckout(t)
	worktree := linkedWorktree(t, main, "x", tasksPending)
	registerChange(t, main, "x", kindBranch("feat/x", branch.KindChange, worktree))

	status := resolveFrom(t, main, "x")

	wantResolvedIn(t, status, worktree, "x")
	if status.ApplyState != ApplyReady || status.NextRecommended != "apply" {
		t.Errorf("applyState %q y nextRecommended %q, se esperaba ready y apply", status.ApplyState, status.NextRecommended)
	}
	if want := []string{worktree}; !reflect.DeepEqual(allowedRoots(status), want) {
		t.Errorf("allowedEditRoots = %v, se esperaba %v", allowedRoots(status), want)
	}
	if notes := notesAbout(status, "se resuelve en el worktree"); len(notes) != 1 || !strings.Contains(notes[0], worktree) || !strings.Contains(notes[0], "feat/x") {
		t.Errorf("la nota debe explicar la reubicación y nombrar el worktree y la rama: %v", status.Notes)
	}
}

// REQ-25.44, escenario «Sin nombre de cambio se consideran los worktrees registrados».
func TestResolveWithoutANameConsidersTheRegisteredWorktrees(t *testing.T) {
	main := mainCheckout(t)
	worktree := linkedWorktree(t, main, "x", tasksPending)
	registerChange(t, main, "x", kindBranch("feat/x", branch.KindChange, worktree))

	status := resolveFrom(t, main, "")

	if status.ChangeName == nil || *status.ChangeName != "x" {
		t.Fatalf("el cambio activo resuelto es %v, se esperaba x", status.ChangeName)
	}
	wantResolvedIn(t, status, worktree, "x")
}

// Sin nombre y con varios candidatos, el `select-change` de hoy se hace sobre la unión del checkout
// principal y los worktrees, y cada continuación se resuelve por nombre.
func TestResolveWithoutANameSelectsOverTheUnionOfCandidates(t *testing.T) {
	main := mainCheckout(t)
	seedReadyChange(t, main, "y", tasksPending)
	worktree := linkedWorktree(t, main, "x", tasksPending)
	registerChange(t, main, "x", kindBranch("feat/x", branch.KindChange, worktree))

	status := resolveFrom(t, main, "")

	if status.NextRecommended != "select-change" || status.ChangeName != nil {
		t.Fatalf("nextRecommended %q y changeName %v, se esperaba select-change sin cambio", status.NextRecommended, status.ChangeName)
	}
	reasons := strings.Join(status.BlockedReasons, "\n")
	for _, want := range []string{"x, y", "axiom sdd status x --cwd", "axiom sdd status y --cwd"} {
		if !strings.Contains(reasons, want) {
			t.Errorf("blockedReasons debe contener %q: %v", want, status.BlockedReasons)
		}
	}
}

// Dos cambios que solo viven en worktrees registrados también son ambiguos sin nombre.
func TestResolveWithoutANameIsAmbiguousBetweenTwoRegisteredWorktrees(t *testing.T) {
	main := mainCheckout(t)
	one := linkedWorktree(t, main, "x", tasksPending)
	two := linkedWorktree(t, main, "z", tasksPending)
	registerChange(t, main, "x", kindBranch("feat/x", branch.KindChange, one))
	registerChange(t, main, "z", kindBranch("feat/z", branch.KindChange, two))

	status := resolveFrom(t, main, "")

	if status.NextRecommended != "select-change" || !containsReason(status.BlockedReasons, "x, z") {
		t.Errorf("nextRecommended %q, motivos %v; se esperaba select-change entre x y z", status.NextRecommended, status.BlockedReasons)
	}
}

// REQ-25.44, escenario «El worktree prevalece sobre el checkout principal».
func TestResolveTheWorktreePrevailsOverTheMainCheckout(t *testing.T) {
	main := mainCheckout(t)
	seedReadyChange(t, main, "x", tasksPending)
	worktree := linkedWorktree(t, main, "x", tasksDone)
	registerChange(t, main, "x", kindBranch("feat/x", branch.KindChange, worktree))

	status := resolveFrom(t, main, "x")

	wantResolvedIn(t, status, worktree, "x")
	if !status.TaskProgress.AllComplete {
		t.Errorf("el status debe ser el del worktree (tareas cerradas): %+v", status.TaskProgress)
	}
	// Sin nombre, la unión sigue siendo un único cambio y el worktree también prevalece.
	wantResolvedIn(t, resolveFrom(t, main, ""), worktree, "x")
}

// REQ-25.44, escenario «El worktree de la rama de archivado prevalece sobre el de la rama del cambio».
func TestResolveTheArchiveWorktreePrevailsOverTheChangeWorktree(t *testing.T) {
	main := mainCheckout(t)
	changeWorktree := linkedWorktree(t, main, "x", tasksPending)
	archiveWorktree := linkedWorktree(t, main, "x", tasksDone)
	registerChange(t, main, "x",
		kindBranch("feat/x", branch.KindChange, changeWorktree),
		kindBranch("feat/x-archive", branch.KindArchive, archiveWorktree),
	)

	wantResolvedIn(t, resolveFrom(t, main, "x"), archiveWorktree, "x")
}

// Si el worktree de archivado aún no tiene la carpeta del cambio, prevalece el del cambio; y sin la
// carpeta en ningún worktree el status se queda en el checkout principal.
func TestResolveSkipsWorktreesWithoutTheChangeFolder(t *testing.T) {
	main := mainCheckout(t)
	seedReadyChange(t, main, "x", tasksPending)
	changeWorktree := linkedWorktree(t, main, "x", tasksDone)
	archiveWorktree := linkedWorktree(t, main, "x", "")
	registerChange(t, main, "x",
		kindBranch("feat/x", branch.KindChange, changeWorktree),
		kindBranch("feat/x-archive", branch.KindArchive, archiveWorktree),
	)
	wantResolvedIn(t, resolveFrom(t, main, "x"), changeWorktree, "x")

	registerChange(t, main, "x", kindBranch("feat/x", branch.KindChange, linkedWorktree(t, main, "x", "")))
	wantResolvedIn(t, resolveFrom(t, main, "x"), main, "x")
}

// REQ-25.44, escenario «Un worktree desaparecido se omite con aviso»: el status se devuelve sin ese
// cambio y con un aviso que nombra la ruta ausente y la continuación.
func TestResolveOmitsAMissingWorktreeWithANote(t *testing.T) {
	main := mainCheckout(t)
	seedReadyChange(t, main, "y", tasksPending)
	missing := filepath.Join(realPath(t, t.TempDir()), "ya-no-existe")
	registerChange(t, main, "x", kindBranch("feat/x", branch.KindChange, missing))

	status := resolveFrom(t, main, "")

	wantResolvedIn(t, status, main, "y")
	notes := notesAbout(status, missing)
	if len(notes) != 1 || !strings.HasPrefix(notes[0], "note(registered_worktrees): ") || !strings.Contains(notes[0], "axiom branch ensure --change x") {
		t.Errorf("notes = %q, se esperaba un aviso `note(registered_worktrees)` que nombre %s y la continuación", status.Notes, missing)
	}
	if status.ApplyState == ApplyBlocked && len(status.BlockedReasons) != 0 && containsReason(status.BlockedReasons, missing) {
		t.Errorf("el aviso no bloquea: %v", status.BlockedReasons)
	}
}

// REQ-25.44: con `--cwd` dentro de un worktree, el cambio se resuelve en su propio árbol, aunque otro
// worktree registrado (el de archivado) tuviera precedencia desde el checkout principal.
func TestResolveInsideAWorktreeKeepsItsOwnTree(t *testing.T) {
	main := mainCheckout(t)
	changeWorktree := linkedWorktree(t, main, "x", tasksPending)
	archiveWorktree := linkedWorktree(t, main, "x", tasksDone)
	registerChange(t, main, "x",
		kindBranch("feat/x", branch.KindChange, changeWorktree),
		kindBranch("feat/x-archive", branch.KindArchive, archiveWorktree),
	)

	status := resolveFrom(t, changeWorktree, "x")

	wantResolvedIn(t, status, changeWorktree, "x")
	if notes := notesAbout(status, "se resuelve en el worktree"); len(notes) != 0 {
		t.Errorf("dentro de un worktree no hay reubicación: %v", notes)
	}
}

// Una raíz de workspace explícita (las llamadas internas que ya conocen su raíz) nunca se reenraíza.
func TestResolveWithAnExplicitWorkspaceRootIsNotRerooted(t *testing.T) {
	main := mainCheckout(t)
	seedReadyChange(t, main, "x", tasksPending)
	worktree := linkedWorktree(t, main, "x", tasksDone)
	registerChange(t, main, "x", kindBranch("feat/x", branch.KindChange, worktree))

	status, err := Resolve(ResolveOptions{WorkspaceRoot: main, ChangeName: "x"})
	if err != nil {
		t.Fatal(err)
	}

	wantResolvedIn(t, status, main, "x")
}

// Sin registro, o sin ramas con worktree, el status es el de antes del incremento: sin avisos nuevos.
func TestResolveWithoutRegisteredWorktreesIsUnchanged(t *testing.T) {
	main := mainCheckout(t)
	seedReadyChange(t, main, "x", tasksPending)

	before := resolveFrom(t, main, "")
	registerChange(t, main, "x", branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange, InPlace: true})
	after := resolveFrom(t, main, "")

	wantResolvedIn(t, before, main, "x")
	if !reflect.DeepEqual(before, after) {
		t.Errorf("un registro sin worktrees no cambia el status:\nantes:   %+v\ndespués: %+v", before, after)
	}
	if len(before.Notes) != 0 {
		t.Errorf("sin registro no hay avisos: %v", before.Notes)
	}
}
