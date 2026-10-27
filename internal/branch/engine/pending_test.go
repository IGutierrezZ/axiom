package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de los cambios pendientes de archivar (REQ-25.20, design D-17) y de `status`
// sin `--change` (REQ-25.15, escenario «`status` sin `--change` lista registrados y
// pendientes»), con git real y sin red.

// addChanges confirma en la rama actual de repo las carpetas de cambio dadas bajo
// `<base>/openspec/changes/` (base vacía = la raíz del repositorio): es como si el PR del
// cambio ya se hubiera fusionado y el cambio siguiera sin archivar.
func addChanges(t *testing.T, repo, base string, names ...string) {
	t.Helper()
	for _, name := range names {
		writeFile(t, filepath.Join(repo, base, "openspec", "changes", name, "proposal.md"), "propuesta\n")
	}
	runGit(t, repo, "add", "-A")
	commit(t, repo, "cambios "+strings.Join(names, ", "))
}

func embeddedProject(t *testing.T) string {
	t.Helper()
	return newProject(t, configWithRoles(map[string][]string{"core": {"."}}, ""))
}

// pendingList devuelve la lista de pendientes de un resultado de `status` sin `--change`
// y exige que se haya evaluado (lista, no booleano).
func pendingList(t *testing.T, result branch.Result) []branch.PendingChange {
	t.Helper()
	if result.PendingArchive == nil || result.PendingArchive.Requested != nil || result.PendingArchive.Changes == nil {
		t.Fatalf("pendingArchive = %+v, se esperaba una lista evaluada (aunque vacía)", result.PendingArchive)
	}
	return result.PendingArchive.Changes
}

func pendingNames(t *testing.T, result branch.Result) []string {
	t.Helper()
	names := []string{}
	for _, pending := range pendingList(t, result) {
		names = append(names, pending.Change)
	}
	return names
}

// wantPendingNames comprueba los nombres pendientes y que cada uno nombra la
// continuación para archivarlo (REQ-25.20).
func wantPendingNames(t *testing.T, result branch.Result, want ...string) {
	t.Helper()
	if want == nil {
		want = []string{}
	}
	if got := pendingNames(t, result); !reflect.DeepEqual(got, want) {
		t.Fatalf("pendientes de archivar = %q, se esperaba %q", got, want)
	}
	for _, pending := range pendingList(t, result) {
		if wantContinuation := "axiom branch ensure --change " + pathquote.ShellWord(pending.Change) + " --archive"; pending.Continuation != wantContinuation {
			t.Errorf("continuación de %s = %q, se esperaba %q", pending.Change, pending.Continuation, wantContinuation)
		}
	}
}

func requested(t *testing.T, result branch.Result) bool {
	t.Helper()
	if result.PendingArchive == nil || result.PendingArchive.Requested == nil || result.PendingArchive.Changes != nil {
		t.Fatalf("pendingArchive = %+v, se esperaba un booleano", result.PendingArchive)
	}
	return *result.PendingArchive.Requested
}

// REQ-25.20, escenario «Un cambio fusionado y sin archivar figura como pendiente», y
// «Los pendientes no producen código de fallo»: con dos pendientes, la salida es 0.
func TestStatusWithoutChangeListsMergedAndUnarchivedChangesAsPending(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)
	addChanges(t, repo, "", "inc-99-ejemplo", "inc-98-otro")

	result := status(t, repo, "", branch.LaneSDD)

	wantReady(t, result)
	if result.Operation != "status" || result.Change != "" || result.Lane != branch.LaneSDD || result.Topology != workspace.TopologyMonorepoEmbedded {
		t.Errorf("operación %q, cambio %q, carril %q, topología %q", result.Operation, result.Change, result.Lane, result.Topology)
	}
	wantSameDirectory(t, "workspaceRoot", result.WorkspaceRoot, repo)
	wantSameDirectory(t, "planningHome", result.PlanningHome, filepath.Join(repo, "openspec"))
	wantPendingNames(t, result, "inc-98-otro", "inc-99-ejemplo")
	if result.Code != "" || result.Continuation != "" || len(result.Warnings) != 0 {
		t.Errorf("los pendientes no son una negativa ni un aviso: código %q, continuación %q, avisos %v", result.Code, result.Continuation, result.Warnings)
	}
	if len(result.Repositories) != 0 || result.Registry != nil {
		t.Errorf("sin --change no hay plan que describir: %+v", result)
	}
}

// REQ-25.20, escenarios «Solo `archive/` no genera pendientes» y «Un fichero suelto no
// cuenta como cambio»: la lista está vacía, pero evaluada (se imprime `[]`).
func TestPendingArchiveIgnoresTheArchiveFolderAndLooseFiles(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)
	writeFile(t, filepath.Join(repo, "openspec", "changes", "archive", "2026-10-01-viejo", "proposal.md"), "archivado\n")
	writeFile(t, filepath.Join(repo, "openspec", "changes", ".gitkeep"), "")
	writeFile(t, filepath.Join(repo, "openspec", "changes", "LEEME.md"), "suelto\n")
	runGit(t, repo, "add", "-A")
	commit(t, repo, "archive y ficheros sueltos")

	result := status(t, repo, "", branch.LaneSDD)

	wantReady(t, result)
	wantPendingNames(t, result)
	if len(result.Warnings) != 0 {
		t.Errorf("avisos = %v, se esperaba ninguno", result.Warnings)
	}
}

// REQ-25.20, escenario «Los sumideros exentos no son pendientes»: `e2e-cumulative` y
// `cumulative-<rol>` se excluyen por nombre; lo que solo se les parece, no.
func TestPendingArchiveExemptsTheCumulativeSinksByName(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)
	addChanges(t, repo, "", "archive", "e2e-cumulative", "cumulative-core", "cumulative-qa.v2_b")

	t.Run("solo los sumideros y archive", func(t *testing.T) {
		wantPendingNames(t, status(t, repo, "", branch.LaneSDD))
	})
	t.Run("lo que se les parece es un cambio", func(t *testing.T) {
		addChanges(t, repo, "", "cumulative", "cumulative-", "e2e-cumulative-2", "inc-1")
		result := status(t, repo, "", branch.LaneSDD)

		wantReady(t, result)
		wantPendingNames(t, result, "cumulative", "cumulative-", "e2e-cumulative-2", "inc-1")
	})
}

func TestIsPendingChange(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]bool{
		"inc-99-ejemplo":         true,
		"archive":                false,
		"Archive":                true,
		"e2e-cumulative":         false,
		"e2e-cumulative-2":       true,
		"cumulative-core":        false,
		"cumulative-a.b_c-d":     false,
		"cumulative-":            true,
		"cumulative":             true,
		"cumulative-ñ":           true,
		"cumulative-con espacio": true,
		"x-cumulative-core":      true,
	} {
		if got := isPendingChange(name); got != want {
			t.Errorf("isPendingChange(%q) = %v, se esperaba %v", name, got, want)
		}
	}
}

// El nombre de una carpeta con espacios sale citado en la continuación, de modo que la
// orden se puede copiar tal cual.
func TestPendingArchiveQuotesUnusualNamesInTheContinuation(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)
	addChanges(t, repo, "", "mi cambio")

	result := status(t, repo, "", branch.LaneSDD)

	wantReady(t, result)
	wantPendingNames(t, result, "mi cambio")
	if got := pendingList(t, result)[0].Continuation; got != "axiom branch ensure --change 'mi cambio' --archive" {
		t.Errorf("continuación = %q", got)
	}
}

// REQ-25.20 y design D-06: la evaluación usa la referencia de integración local de
// REQ-25.10: `origin/<integración>` si existe en local y, si no, la rama local.
func TestPendingArchiveEvaluatesTheLocalIntegrationReference(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)
	remote := filepath.Join(t.TempDir(), "remoto.git")
	runGit(t, repo, "init", "-q", "--bare", "-b", "main", remote)
	runGit(t, repo, "remote", "add", "origin", remote)
	addChanges(t, repo, "", "en-origin")
	runGit(t, repo, "push", "-q", "origin", "main")
	addChanges(t, repo, "", "solo-local")

	t.Run("con origin/main en local, manda origin/main", func(t *testing.T) {
		wantPendingNames(t, status(t, repo, "", branch.LaneSDD), "en-origin")
	})
	t.Run("sin referencia remota local, la rama local", func(t *testing.T) {
		runGit(t, repo, "update-ref", "-d", "refs/remotes/origin/main")
		wantPendingNames(t, status(t, repo, "", branch.LaneSDD), "en-origin", "solo-local")
	})
}

// REQ-25.20, escenario «La evaluación no usa la red» y «sin cambiar de rama»: con un
// remoto inalcanzable configurado y otra rama desplegada, ninguna invocación de git es
// remota, solo se lanza la lista cerrada de lecturas, y ni la rama actual ni el árbol de
// trabajo cambian.
func TestPendingArchiveUsesNoNetworkAndNeverChangesBranch(t *testing.T) {
	repo := embeddedProject(t)
	runGit(t, repo, "remote", "add", "origin", "https://ejemplo.invalid/repo.git")
	addChanges(t, repo, "", "inc-99-ejemplo")
	runGit(t, repo, "switch", "-q", "-c", "feat/otro")
	head := runGit(t, repo, "rev-parse", "HEAD")
	seen := observe(t)

	for _, change := range []string{"", "inc-99-ejemplo"} {
		wantReached(t, status(t, repo, change, branch.LaneSDD))
	}

	if len(*seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación: la prueba sería vacua")
	}
	sawTree := false
	for _, invocation := range *seen {
		if remoteOperation(invocation[1]) {
			t.Errorf("la evaluación lanzó la operación remota git %s: %v", invocation[1], invocation)
		}
		switch invocation[1] {
		case "rev-parse", "worktree", "status", "rev-list":
		case "ls-tree":
			sawTree = true
		default:
			t.Errorf("la evaluación lanzó %q, fuera de la lista cerrada de lecturas: %v", invocation[1], invocation)
		}
	}
	if !sawTree {
		t.Error("la lista de pendientes se lee con `git ls-tree`, y el observador no la vio")
	}
	if current := runGit(t, repo, "branch", "--show-current"); current != "feat/otro" {
		t.Errorf("la rama actual cambió a %q", current)
	}
	if after := runGit(t, repo, "rev-parse", "HEAD"); after != head {
		t.Errorf("HEAD cambió de %s a %s", head, after)
	}
	if dirty := runGit(t, repo, "status", "--porcelain"); dirty != "" {
		t.Errorf("el árbol de trabajo quedó modificado: %s", dirty)
	}
}

// REQ-25.20, escenario «Sin referencia de integración la lista se omite con aviso»: sin
// la rama de integración ni local ni remota local, la salida sigue siendo 0, el campo no
// se imprime y un aviso lo explica.
func TestPendingArchiveWithoutTheIntegrationReferenceIsOmittedWithAWarning(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, "git:\n  integration_branch: develop\n"))
	addChanges(t, repo, "", "inc-99-ejemplo")

	for _, change := range []string{"", "inc-99-ejemplo"} {
		result := status(t, repo, change, branch.LaneSDD)

		wantReached(t, result)
		if result.PendingArchive != nil {
			t.Errorf("--change %q: pendingArchive = %+v, se esperaba omitido", change, result.PendingArchive)
		}
		var about []string
		for _, warning := range result.Warnings {
			if strings.Contains(warning, "pendientes de archivar") {
				about = append(about, warning)
			}
		}
		if len(about) != 1 || !strings.Contains(about[0], "«develop»") {
			t.Errorf("--change %q: avisos %v; se esperaba uno sobre los pendientes que nombre la rama de integración develop", change, result.Warnings)
		}
	}
}

// REQ-25.20, escenario «El repositorio de specs de un multirepo no tiene pendientes»: los
// cambios activos viven allí en la rama de integración de forma normal, así que la lista
// está vacía (evaluada) y sin avisos, y `--change` indica que no está pendiente.
func TestPendingArchiveOfAMultirepoSpecsRepositoryIsEmpty(t *testing.T) {
	t.Parallel()
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}}, "repo-backend")
	addChanges(t, specs, "", "inc-99-activo")
	sealChangeAt(t, specs, "x", "backend")

	t.Run("sin --change", func(t *testing.T) {
		result := status(t, master, "", branch.LaneSDD)

		wantReady(t, result)
		wantPendingNames(t, result)
		if len(result.Warnings) != 0 || result.Topology != workspace.TopologyMultirepo {
			t.Errorf("topología %q, avisos %v; se esperaba multirepo sin avisos", result.Topology, result.Warnings)
		}
	})
	t.Run("con --change", func(t *testing.T) {
		for _, change := range []string{"x", "inc-99-activo"} {
			result := status(t, master, change, branch.LaneSDD)

			wantReached(t, result)
			if requested(t, result) || result.PendingArchiveContinuation != "" {
				t.Errorf("%s: en el repositorio de specs de un multirepo nada está pendiente de archivar: %+v / %q", change, result.PendingArchive, result.PendingArchiveContinuation)
			}
		}
	})
}

// El planning home puede estar en una subcarpeta del repositorio: la lista se lee de
// `<subcarpeta>/openspec/changes`, no de la raíz del repositorio.
func TestPendingArchiveReadsThePlanningHomeOfASubfolderProject(t *testing.T) {
	t.Parallel()
	repo := newRepository(t)
	app := filepath.Join(repo, "app")
	writeFile(t, filepath.Join(app, "axiom.yaml"), configWithRoles(map[string][]string{"core": {"."}}, ""))
	runGit(t, repo, "add", "-A")
	commit(t, repo, "proyecto en app")
	addChanges(t, repo, "app", "inc-1-en-app")
	addChanges(t, repo, "", "inc-2-en-la-raiz")

	result := status(t, app, "", branch.LaneSDD)

	wantReady(t, result)
	wantPendingNames(t, result, "inc-1-en-app")
	wantSameDirectory(t, "planningHome", result.PlanningHome, filepath.Join(app, "openspec"))
}

// Un repositorio sin la carpeta `openspec/changes` en la integración no tiene pendientes,
// y no es un error ni un aviso.
func TestPendingArchiveOfAProjectWithoutAChangesFolderIsEmpty(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)

	result := status(t, repo, "", branch.LaneSDD)

	wantReady(t, result)
	wantPendingNames(t, result)
	if len(result.Warnings) != 0 {
		t.Errorf("avisos = %v, se esperaba ninguno", result.Warnings)
	}
}

// REQ-25.15 y REQ-25.20, escenario «`status --change` indica que el cambio solicitado
// está pendiente»: verdadero con la continuación para archivarlo, o falso; un cambio
// fusionado que nunca tuvo ramas sigue siendo `not_started`.
func TestStatusWithChangeInformsWhetherItIsPendingToArchive(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)
	addChanges(t, repo, "", "inc-99-ejemplo")

	t.Run("pendiente", func(t *testing.T) {
		result := status(t, repo, "inc-99-ejemplo", branch.LaneSDD)

		wantReached(t, result)
		if !requested(t, result) || result.PendingArchiveContinuation != "axiom branch ensure --change inc-99-ejemplo --archive" {
			t.Errorf("pendingArchive = %+v, continuación %q; se esperaba verdadero con la orden para archivarlo", result.PendingArchive, result.PendingArchiveContinuation)
		}
		if result.Status != branch.StatusNotStarted || result.ExitCode() != branch.ExitReady {
			t.Errorf("status %q, salida %d; un cambio sin ramas es not_started con salida 0", result.Status, result.ExitCode())
		}
		if len(result.Registered) != 0 {
			t.Errorf("con --change no se listan los registrados: %+v", result.Registered)
		}
	})
	t.Run("no pendiente", func(t *testing.T) {
		result := status(t, repo, "x", branch.LaneSDD)

		wantReached(t, result)
		if requested(t, result) || result.PendingArchiveContinuation != "" {
			t.Errorf("pendingArchive = %+v, continuación %q; se esperaba falso sin continuación", result.PendingArchive, result.PendingArchiveContinuation)
		}
	})
	t.Run("el carril odd no archiva", func(t *testing.T) {
		result := status(t, repo, "inc-99-ejemplo", branch.LaneODD)

		wantReached(t, result)
		if result.PendingArchive != nil || result.PendingArchiveContinuation != "" {
			t.Errorf("pendingArchive = %+v; el carril ODD no tiene archivado, se esperaba omitido", result.PendingArchive)
		}
	})
	t.Run("un cambio con rama en un worktree sigue indicándolo", func(t *testing.T) {
		worktree := plannedWorktree(t, repo, "inc-99-ejemplo", filepath.Base(repo))
		runGit(t, repo, "worktree", "add", "-q", "-b", "feat/inc-99-ejemplo", worktree, "main")

		result := status(t, repo, "inc-99-ejemplo", branch.LaneSDD)

		wantReady(t, result)
		if !requested(t, result) {
			t.Errorf("pendingArchive = %+v, se esperaba verdadero", result.PendingArchive)
		}
	})
}

// REQ-25.15, escenario «`status` sin `--change` lista registrados y pendientes»: `x`
// registrado y `y` fusionado y pendiente; los registrados salen ordenados, con su modo y
// sus ramas con rutas absolutas, y solo los del carril pedido.
func TestStatusWithoutChangeListsRegisteredAndPending(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)
	addChanges(t, repo, "", "y")
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	registered := func(change string, lane branch.Lane, mode branch.Mode) {
		record := registeredRecord(repo, branch.RecordBranch{Name: "feat/" + change, Kind: branch.KindChange, Worktree: worktree})
		record.Change, record.Lane, record.Mode = change, lane, mode
		writeRegistry(t, commonDir(repo), record)
	}
	registered("x", branch.LaneSDD, branch.ModeWorktree)
	registered("a", branch.LaneSDD, branch.ModeInPlace)
	registered("solo-odd", branch.LaneODD, branch.ModeWorktree)

	result := status(t, repo, "", branch.LaneSDD)

	wantReady(t, result)
	var changes []string
	for _, entry := range result.Registered {
		changes = append(changes, entry.Change)
	}
	if want := []string{"a", "x"}; !reflect.DeepEqual(changes, want) {
		t.Fatalf("registrados = %v, se esperaba %v (ordenados, solo el carril sdd)", changes, want)
	}
	first, second := result.Registered[0], result.Registered[1]
	if first.Lane != branch.LaneSDD || first.Mode != branch.ModeInPlace || first.ModeLabel != "en la carpeta del proyecto" || first.ModeSource != branch.ModeSourceHub || first.ModeSourceLabel != "preferencias personales del hub" {
		t.Errorf("registrado a = %+v", first)
	}
	if len(second.Branches) != 1 || second.Branches[0].Name != "feat/x" || second.Branches[0].Repository != filepath.Base(repo) || second.Branches[0].KindLabel != "rama del cambio" || second.Branches[0].Worktree != worktree {
		t.Errorf("ramas registradas de x = %+v", second.Branches)
	}
	wantPendingNames(t, result, "y")

	t.Run("el carril odd lista los suyos, y los pendientes siguen siendo del proyecto", func(t *testing.T) {
		odd := status(t, repo, "", branch.LaneODD)

		wantReady(t, odd)
		if len(odd.Registered) != 1 || odd.Registered[0].Change != "solo-odd" || odd.Registered[0].Lane != branch.LaneODD {
			t.Errorf("registrados odd = %+v, se esperaba solo-odd", odd.Registered)
		}
		wantPendingNames(t, odd, "y")
	})
}

// Sin nada registrado ni pendiente, `status` sin `--change` no falla: ambas listas
// vacías pero evaluadas.
func TestStatusWithoutChangeOfAnEmptyProject(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)

	result := status(t, repo, "", branch.LaneSDD)

	wantReady(t, result)
	if result.Registered == nil || len(result.Registered) != 0 {
		t.Errorf("registered = %+v, se esperaba una lista vacía pero evaluada", result.Registered)
	}
	wantPendingNames(t, result)
	if _, err := os.Stat(filepath.Join(commonDir(repo), "axiom")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("`status` es de solo lectura y no debe crear el registro (%v)", err)
	}
}

// El registro es estado derivado: un fichero corrupto es `config_invalid` con el fichero
// y la orden para borrarlo, igual que con `--change`.
func TestStatusWithoutChangeRefusesACorruptRegistryNamingTheFile(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)
	path := recordFile(commonDir(repo), "sdd", "roto")
	writeFile(t, path, "esto no es JSON {")

	result := status(t, repo, "", branch.LaneSDD)

	wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
	wantSameFile(t, "details.path", result.Details["path"], path)
	for _, want := range []string{"borra", pathquote.Quote(result.Details["path"]), "axiom branch status --cwd", pathquote.Quote(repo)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
	if result.PendingArchive != nil || result.Registered != nil {
		t.Errorf("un rechazo no lleva datos: %+v", result)
	}
}

// Un carril que no puede nombrar un registro se rechaza sin leer nada, y nombra cómo
// pedir uno válido.
func TestStatusWithoutChangeRefusesALaneThatCannotNameARegistry(t *testing.T) {
	t.Parallel()
	repo := embeddedProject(t)

	result := status(t, repo, "", branch.Lane("../v1"))

	wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
	if !strings.Contains(result.Message, "carril") || len(result.Details) != 0 {
		t.Errorf("mensaje %q, details %v; se esperaba nombrar el carril y ningún fichero", result.Message, result.Details)
	}
	for _, want := range []string{"axiom branch status --lane sdd --cwd", pathquote.Quote(repo)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
}

// Los rechazos de la resolución del proyecto, los mismos que con `--change`: fuera de un
// repositorio, `not_a_git_repository` con la orden para indicar uno.
func TestStatusWithoutChangeRefusesOutsideAGitRepository(t *testing.T) {
	t.Parallel()
	requireGit(t)
	outside := t.TempDir()
	for current := outside; ; {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			t.Skipf("el directorio temporal %s está dentro de un repositorio git", outside)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	result := status(t, outside, "", branch.LaneSDD)

	wantRefusal(t, result, branch.CodeNotAGitRepository, branch.StatusRefused)
	for _, want := range []string{"axiom branch status --cwd", pathquote.Quote(outside)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
}

// fakeGit es un doble de las lecturas de git para forzar los fallos que un repositorio
// real no produce: la lista de pendientes nunca falla ni bloquea, avisa.
type fakeGit struct {
	topLevel    func(dir string) (string, error)
	resolve     func(ref string) (string, bool, error)
	treeDirs    func(rev, relPath string) ([]string, bool, error)
	treeDirArgs [][2]string
}

func (f *fakeGit) TopLevel(_ context.Context, dir string) (string, error) { return f.topLevel(dir) }
func (f *fakeGit) CommonDir(context.Context, string) (string, error) {
	return "", errors.New("no se usa")
}
func (f *fakeGit) ResolveCommit(_ context.Context, _, ref string) (string, bool, error) {
	return f.resolve(ref)
}
func (f *fakeGit) BranchExists(context.Context, string, string) (bool, error) {
	return false, errors.New("no se usa")
}
func (f *fakeGit) Worktrees(context.Context, string) ([]branch.WorktreeEntry, error) {
	return nil, errors.New("no se usa")
}
func (f *fakeGit) DirtyPaths(context.Context, string) ([]string, error) {
	return nil, errors.New("no se usa")
}
func (f *fakeGit) AheadBehind(context.Context, string, string, string) (int, int, error) {
	return 0, 0, errors.New("no se usa")
}
func (f *fakeGit) TreeDirs(_ context.Context, _, rev, relPath string) ([]string, bool, error) {
	f.treeDirArgs = append(f.treeDirArgs, [2]string{rev, relPath})
	return f.treeDirs(rev, relPath)
}
func (f *fakeGit) IsAncestor(context.Context, string, string, string) (bool, error) {
	return false, errors.New("no se usa")
}

// design D-17: «sin referencia, la lista se omite con un aviso y sin error»; lo mismo
// ante cualquier fallo de git al evaluar: nunca falla, nunca bloquea, y la lista no se
// inventa (nil, no vacía).
func TestPendingArchiveNeverFailsWhenGitDoes(t *testing.T) {
	t.Parallel()
	master := t.TempDir()
	location := workspace.Location{Master: master, WorkspaceRoot: master, PlanningHome: filepath.Join(master, "openspec")}
	boom := errors.New("fallo simulado de git")
	ok := func(rev, relPath string) ([]string, bool, error) { return []string{"inc-1"}, true, nil }

	tests := []struct {
		name string
		git  *fakeGit
		want string
	}{
		{"no se puede localizar la raíz del repositorio", &fakeGit{
			topLevel: func(string) (string, error) { return "", boom },
			resolve:  func(string) (string, bool, error) { return "abc", true, nil },
			treeDirs: ok,
		}, "fallo simulado"},
		{"no se puede resolver la referencia de integración", &fakeGit{
			topLevel: func(string) (string, error) { return master, nil },
			resolve:  func(string) (string, bool, error) { return "", false, boom },
			treeDirs: ok,
		}, "fallo simulado"},
		{"no se puede leer el árbol", &fakeGit{
			topLevel: func(string) (string, error) { return master, nil },
			resolve:  func(string) (string, bool, error) { return "abc", true, nil },
			treeDirs: func(string, string) ([]string, bool, error) { return nil, false, boom },
		}, "fallo simulado"},
		{"el planning home no está dentro del repositorio", &fakeGit{
			topLevel: func(string) (string, error) { return t.TempDir(), nil },
			resolve:  func(string) (string, bool, error) { return "abc", true, nil },
			treeDirs: ok,
		}, "fuera del repositorio"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := &Engine{git: tc.git, registry: registry.New(), goos: "linux"}

			pending, warnings := engine.PendingArchive(context.Background(), location)

			if pending != nil {
				t.Errorf("pendientes = %+v, se esperaba nil (omitida)", pending)
			}
			if len(warnings) != 1 || !strings.Contains(warnings[0], "pendientes de archivar") || !strings.Contains(warnings[0], tc.want) {
				t.Errorf("avisos = %v; se esperaba uno sobre los pendientes que contenga %q", warnings, tc.want)
			}
		})
	}

	t.Run("lee el árbol de la referencia resuelta, no de la rama de HEAD", func(t *testing.T) {
		git := &fakeGit{
			topLevel: func(string) (string, error) { return master, nil },
			resolve: func(ref string) (string, bool, error) {
				return "sha-de-" + filepath.Base(ref), ref == "refs/heads/main", nil
			},
			treeDirs: ok,
		}
		engine := &Engine{git: git, registry: registry.New(), goos: "linux"}

		pending, warnings := engine.PendingArchive(context.Background(), location)

		if len(warnings) != 0 || len(pending) != 1 || pending[0].Change != "inc-1" {
			t.Fatalf("pendientes %+v, avisos %v", pending, warnings)
		}
		if want := ([2]string{"sha-de-main", "openspec/changes"}); len(git.treeDirArgs) != 1 || git.treeDirArgs[0] != want {
			t.Errorf("TreeDirs recibió %v, se esperaba una sola lectura con %v", git.treeDirArgs, want)
		}
	})
}
