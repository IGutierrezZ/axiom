package engine

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/gitcli"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Pruebas del registro local con escritura de `ensure` (REQ-25.14, design D-05 fases 5 y 6 y
// D-07): `ensure` guarda el plan materializado en el git common dir solo si cambian sus bytes,
// lo reconstruye desde git cuando falta, y un bloqueo por cambio impide dos ejecuciones a la
// vez. Con git real, entorno aislado y sin red.

func loadRecord(t *testing.T, repo, change string) *branch.Record {
	t.Helper()
	record, err := registry.New().Load(commonDir(repo), branch.LaneSDD, change)
	if err != nil || record == nil {
		t.Fatalf("el registro de %s debía existir: %+v, %v", change, record, err)
	}
	return record
}

func wantNoRegistryFile(t *testing.T, repo, change string) {
	t.Helper()
	if _, err := os.Stat(recordFile(commonDir(repo), "sdd", change)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("no debía escribirse el registro de %s (%v)", change, err)
	}
}

func setOldModTime(t *testing.T, path string) time.Time {
	t.Helper()
	old := time.Date(2001, time.February, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	return old
}

// REQ-25.14 y D-05 (fase 5): `ensure` guarda el plan en
// `<common dir>/axiom/branch/v1/sdd/x.json` con rutas absolutas; una segunda ejecución con el
// mismo estado no reescribe el fichero, y las dos describen el mismo estado final (también
// `registry.present`). El registro no toca `gentle-ai/`.
func TestEnsureWritesTheRegistryAndRepeatingDoesNotRewriteIt(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	lineage := filepath.Join(commonDir(repo), "gentle-ai", "review-transactions", "linaje-1", "estado.json")
	writeFile(t, lineage, `{"linaje":"abierto"}`)
	oldLineage := setOldModTime(t, lineage)

	first := ensureOf(t, repo, "x")

	wantReady(t, first)
	if first.Registry == nil || !first.Registry.Present {
		t.Errorf("registry = %+v; tras escribirlo el registro está presente", first.Registry)
	}
	record := loadRecord(t, repo, "x")
	if record.Schema != branch.RegistrySchema || record.Lane != branch.LaneSDD || record.Change != "x" {
		t.Errorf("esquema %q, carril %q, cambio %q", record.Schema, record.Lane, record.Change)
	}
	if record.Mode != branch.ModeWorktree || record.ModeSource != branch.ModeSourceDefault {
		t.Errorf("modo %q con origen %q; se esperaba worktree por defecto", record.Mode, record.ModeSource)
	}
	wantSameDirectory(t, "planningRepo", record.PlanningRepo, repo)
	if len(record.Repositories) != 1 {
		t.Fatalf("repositorios = %+v, se esperaba uno", record.Repositories)
	}
	repository := record.Repositories[0]
	wantSameDirectory(t, "gitRoot", repository.GitRoot, repo)
	wantSameDirectory(t, "commonDir", repository.CommonDir, commonDir(repo))
	if repository.Name != filepath.Base(repo) || repository.Role != branch.RepoChangeBranch || len(repository.Branches) != 1 {
		t.Fatalf("repositorio registrado = %+v", repository)
	}
	registered := repository.Branches[0]
	if registered.Name != "feat/x" || registered.Kind != branch.KindChange || registered.RoleID != "" {
		t.Errorf("rama registrada = %+v", registered)
	}
	wantSameDirectory(t, "worktree registrado", registered.Worktree, plannedWorktree(t, repo, "x", filepath.Base(repo)))

	path := recordFile(commonDir(repo), "sdd", "x")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(before), "\r") || !strings.HasSuffix(string(before), "}\n") {
		t.Errorf("el registro se escribe con `\\n` y un salto final: %q", before)
	}
	old := setOldModTime(t, path)
	firstJSON := resultJSON(t, first)

	second := ensureOf(t, repo, "x")

	wantReady(t, second)
	if got := resultJSON(t, second); got != firstJSON {
		t.Errorf("la segunda ejecución cambió el JSON:\n%s\n---\n%s", firstJSON, got)
	}
	info, err := os.Stat(path)
	if err != nil || !info.ModTime().Equal(old) {
		t.Errorf("sin cambios en los bytes el registro no se reescribe (%v, %v)", info, err)
	}
	if after, err := os.ReadFile(path); err != nil || string(after) != string(before) {
		t.Errorf("el registro cambió entre ejecuciones: %q, %v", after, err)
	}
	if data, err := os.ReadFile(lineage); err != nil || string(data) != `{"linaje":"abierto"}` {
		t.Errorf("el linaje de RDD cambió: %q, %v", data, err)
	}
	if info, err := os.Stat(lineage); err != nil || !info.ModTime().Equal(oldLineage) {
		t.Errorf("`ensure` no debe tocar gentle-ai/ (%v, %v)", info, err)
	}
}

// REQ-25.14, escenario «Borrar el registro no daña el repositorio»: con el registro borrado,
// `ensure` reutiliza las ramas y los worktrees sin tocarlos (ninguna escritura de git), el
// resultado es el mismo y el registro se reconstruye con los mismos bytes y rutas.
func TestEnsureRebuildsADeletedRegistryWithoutTouchingTheRepository(t *testing.T) {
	repo := embeddedCoreProject(t)
	firstJSON := resultJSON(t, ensureOf(t, repo, "x"))
	path := recordFile(commonDir(repo), "sdd", "x")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	state := repositoryState(t, repo)
	seen := observe(t)

	second := ensureOf(t, repo, "x")

	wantReady(t, second)
	if got := resultJSON(t, second); got != firstJSON {
		t.Errorf("el resultado tras borrar el registro cambió:\n%s\n---\n%s", firstJSON, got)
	}
	if got := repositoryState(t, repo); got != state {
		t.Errorf("borrar el registro no debe dañar el repositorio:\n%s\n---\n%s", state, got)
	}
	if len(*seen) == 0 {
		t.Fatal("el observador no vio ninguna invocación: la prueba sería vacua")
	}
	for _, invocation := range *seen {
		if isWriteInvocation(invocation) {
			t.Errorf("las ramas y los worktrees existentes se reutilizan y se lanzó %v", invocation)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Errorf("el registro reconstruido debe tener los mismos bytes (%v):\n%s\n---\n%s", err, before, after)
	}
}

// REQ-25.14 y D-07 («Reconstrucción»): sin fichero, `ensure` lo reconstruye desde las ramas
// planificadas que existen y los worktrees que hay en las rutas planificadas, aunque los
// crease otra herramienta, y sin lanzar ninguna escritura de git.
func TestEnsureReconstructsTheRegistryFromBranchesAndWorktreesCreatedByOtherTools(t *testing.T) {
	repo := embeddedCoreProject(t)
	planned := plannedWorktree(t, repo, "x", filepath.Base(repo))
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", planned, "main")
	wantNoRegistryFile(t, repo, "x")
	seen := observe(t)

	result := ensureOf(t, repo, "x")

	wantReady(t, result)
	for _, invocation := range *seen {
		if isWriteInvocation(invocation) {
			t.Errorf("no había nada que crear y se lanzó %v", invocation)
		}
	}
	record := loadRecord(t, repo, "x")
	if len(record.Repositories) != 1 || len(record.Repositories[0].Branches) != 1 {
		t.Fatalf("registro reconstruido = %+v", record)
	}
	registered := record.Repositories[0].Branches[0]
	if registered.Name != "feat/x" || registered.Kind != branch.KindChange {
		t.Errorf("rama reconstruida = %+v", registered)
	}
	wantSameDirectory(t, "worktree reconstruido", registered.Worktree, planned)
}

// D-05 (fase 5) y REQ-25.7: el registro refleja el plan completo. Antes del sellado guarda
// la rama del cambio; con dos roles sellados, la siguiente ejecución lo reescribe con las
// tres ramas, cada una con su tipo, su rol y su worktree.
func TestEnsureUpdatesTheRegistryWhenSealingCompletesTheRoleBranches(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))
	name := filepath.Base(repo)
	wantReady(t, ensureOf(t, repo, "x"))
	if got := loadRecord(t, repo, "x").Repositories[0].Branches; len(got) != 1 || got[0].Name != "feat/x" {
		t.Fatalf("antes del sellado el registro tiene solo feat/x: %+v", got)
	}
	sealChange(t, repo, "x", "core", "qa")

	wantReady(t, ensureOf(t, repo, "x"))

	got := loadRecord(t, repo, "x").Repositories[0].Branches
	if len(got) != 3 {
		t.Fatalf("ramas registradas = %+v, se esperaban feat/x, feat/x-core y feat/x-qa", got)
	}
	want := []struct {
		name string
		kind branch.BranchKind
		role string
		dir  string
	}{
		{"feat/x", branch.KindChange, "", "x"},
		{"feat/x-core", branch.KindRole, "core", "x-core"},
		{"feat/x-qa", branch.KindRole, "qa", "x-qa"},
	}
	for i, expected := range want {
		if got[i].Name != expected.name || got[i].Kind != expected.kind || got[i].RoleID != expected.role {
			t.Errorf("rama %d = %+v, se esperaba %+v", i, got[i], expected)
		}
		wantSameDirectory(t, "worktree de "+expected.name, got[i].Worktree, plannedWorktree(t, repo, expected.dir, name))
	}
}

// D-08 y REQ-25.14: un registro que ya existe fija el modo y su origen (prioridad: registro
// > `--mode` > hub > defecto); `ensure` completa sus ramas y conserva el origen.
func TestEnsureKeepsTheModeAndSourceOfAnExistingRegistry(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	writeRegistry(t, commonDir(repo), registeredRecord(repo))

	result := ensureOf(t, repo, "x")

	wantReady(t, result)
	if result.Mode != branch.ModeWorktree || result.ModeSource != branch.ModeSourceHub {
		t.Errorf("modo %q con origen %q; el registro existente manda", result.Mode, result.ModeSource)
	}
	record := loadRecord(t, repo, "x")
	if record.Mode != branch.ModeWorktree || record.ModeSource != branch.ModeSourceHub {
		t.Errorf("el registro conserva su modo y su origen: %q, %q", record.Mode, record.ModeSource)
	}
	if got := record.Repositories[0].Branches; len(got) != 1 || got[0].Name != "feat/x" {
		t.Errorf("ramas registradas = %+v, se esperaba feat/x", got)
	}
}

// D-05 y REQ-25.14: un bloqueo exclusivo por cambio impide dos `ensure` a la vez. Con el
// bloqueo ocupado, `plan_conflict` con el mensaje fijado por el diseño, sin crear nada ni
// escribir el registro; otro cambio no se bloquea; al liberarlo, `ensure` procede.
func TestEnsureRefusesWhileAnotherRunHoldsTheLockOfTheChange(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	unlock, err := registry.New().Lock(context.Background(), commonDir(repo), branch.LaneSDD, "x")
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	released := false
	t.Cleanup(func() {
		if !released {
			unlock()
		}
	})
	state := repositoryState(t, repo)

	result := ensureOf(t, repo, "x")

	wantRefusal(t, result, branch.CodePlanConflict, branch.StatusRefused)
	if !strings.Contains(result.Message, "otra ejecución de `axiom branch` está modificando el cambio") {
		t.Errorf("el mensaje %q debe ser el del diseño (D-05)", result.Message)
	}
	for _, want := range []string{"axiom branch ensure --change x --cwd", pathquote.Quote(repo)} {
		if !strings.Contains(result.Continuation, want) {
			t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
		}
	}
	wantNothingCreated(t, repo, state, "x")
	wantNoRegistryFile(t, repo, "x")

	t.Run("otro cambio no se bloquea", func(t *testing.T) {
		wantReady(t, ensureOf(t, repo, "y"))
	})

	unlock()
	released = true
	wantReady(t, ensureOf(t, repo, "x"))
}

// El bloqueo se libera también cuando `ensure` rechaza: tras un `base_not_found`, el cambio
// puede volver a bloquearse.
func TestEnsureReleasesTheLockWhenItRefuses(t *testing.T) {
	t.Parallel()
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}}, "git:\n  integration_branch: develop\n"))

	wantRefusal(t, ensureOf(t, repo, "x"), branch.CodeBaseNotFound, branch.StatusRefused)

	unlock, err := registry.New().Lock(context.Background(), commonDir(repo), branch.LaneSDD, "x")
	if err != nil {
		t.Fatalf("el rechazo debía liberar el bloqueo: %v", err)
	}
	unlock()
	wantNoRegistryFile(t, repo, "x")
}

// REQ-25.14 y D-07 («Fichero corrupto»): `config_invalid` nombrando el fichero y la
// continuación de borrarlo, sin crear nada; borrado, `ensure` lo reconstruye.
func TestEnsureRefusesACorruptRegistryAndRebuildsItOnceDeleted(t *testing.T) {
	t.Parallel()
	repo := embeddedCoreProject(t)
	path := recordFile(commonDir(repo), "sdd", "x")
	writeFile(t, path, `{"schema": `)
	state := repositoryState(t, repo)

	result := ensureOf(t, repo, "x")

	wantRefusal(t, result, branch.CodeConfigInvalid, branch.StatusRefused)
	wantSameFile(t, "details.path", result.Details["path"], path)
	if !strings.Contains(result.Continuation, pathquote.Quote(result.Details["path"])) || !strings.Contains(result.Continuation, "axiom branch ensure --change x --cwd") {
		t.Errorf("el rechazo debe citar el fichero %s y la orden que lo reconstruye: %q", result.Details["path"], result.Continuation)
	}
	wantNothingCreated(t, repo, state, "x")

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	wantReady(t, ensureOf(t, repo, "x"))
	loadRecord(t, repo, "x")
}

// D-05: el registro se escribe en la fase 5, tras aplicar. Un rechazo de la fase 3 o un
// fallo de git a medias no lo escriben; repetir `ensure` completa lo pendiente y lo guarda.
func TestEnsureWritesTheRegistryOnlyAfterApplyingThePlan(t *testing.T) {
	t.Parallel()
	t.Run("un rechazo de la comprobación", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		runGit(t, repo, "switch", "-q", "-c", "feat/x")

		wantRefusal(t, ensureOf(t, repo, "x"), branch.CodeBranchCheckedOutElsewhere, branch.StatusRefused)

		wantNoRegistryFile(t, repo, "x")
	})
	t.Run("un fallo de git a medias", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		flaky := newEngine(runtime.GOOS)
		flaky.writer = failingWriter{real: gitcli.New(), failWorktree: true}

		result := flaky.Ensure(context.Background(), EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD})

		wantRefusal(t, result, branch.CodeGitFailed, branch.StatusError)
		wantNoRegistryFile(t, repo, "x")

		wantReady(t, ensureOf(t, repo, "x"))
		loadRecord(t, repo, "x")
	})
}

// REQ-25.14, escenario «El registro se resuelve con el git common dir»: con un directorio git
// que no se llama `.git` (`--separate-git-dir`), el registro se crea bajo el directorio que
// devuelve `git rev-parse --git-common-dir` y nunca bajo un `.git` supuesto.
func TestEnsureResolvesTheRegistryWithTheGitCommonDirWhenItIsNotDotGit(t *testing.T) {
	t.Parallel()
	requireGit(t)
	repo := t.TempDir()
	separate := filepath.Join(t.TempDir(), "datos-de-git")
	runGit(t, repo, "init", "-q", "-b", "main", "--separate-git-dir", separate)
	commit(t, repo, "inicial")
	writeFile(t, filepath.Join(repo, "axiom.yaml"), configWithRoles(map[string][]string{"core": {"."}}, ""))
	runGit(t, repo, "add", "axiom.yaml")
	commit(t, repo, "configuración")

	wantReady(t, ensureOf(t, repo, "x"))

	common := filepath.Clean(filepath.FromSlash(runGit(t, repo, "rev-parse", "--path-format=absolute", "--git-common-dir")))
	if !pathidentity.SameDirectory(common, separate) {
		t.Fatalf("la preparación debía dejar el common dir en %s, git devolvió %s", separate, common)
	}
	if info, err := os.Stat(recordFile(common, "sdd", "x")); err != nil || info.IsDir() {
		t.Errorf("el registro debe estar bajo el common dir %s: %v", common, err)
	}
	if info, err := os.Stat(filepath.Join(repo, ".git")); err != nil || info.IsDir() {
		t.Errorf("en este repositorio `.git` es un fichero que apunta al directorio de datos (%v)", err)
	}
}
