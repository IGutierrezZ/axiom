package engine

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/gitcli"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Modo `in_place` de `ensure` (REQ-25.8, REQ-25.9, design D-05 fases 3 y 4, matriz de
// amenazas «Estado del commit»): la comprobación de suciedad va en la fase 3, antes de
// cualquier escritura, y el cambio de rama es un `git switch` en el propio repositorio, sin
// worktree. Con git real, entorno aislado y sin red. `in_place` no crea worktrees, que son
// lo que más cuesta de estas pruebas en Windows: cada prueba reutiliza un único repositorio.

// ensureInPlace pide `--mode in_place` con un hub que no pregunta (origen `flag`).
func ensureInPlace(t *testing.T, cwd, change string) branch.Result {
	t.Helper()
	return ensureWith(t, hubWorktree, EnsureRequest{CWD: cwd, Change: change, Mode: branch.ModeInPlace})
}

// inPlaceState es todo lo que un rechazo de `ensure` no puede cambiar: el estado de las ramas
// y los worktrees, y HEAD, la rama actual, el índice y el estado del árbol de trabajo.
func inPlaceState(t *testing.T, repo string) string {
	t.Helper()
	return repositoryState(t, repo) + "\n==\n" + checkoutState(t, repo)
}

func currentBranch(t *testing.T, repo string) string {
	t.Helper()
	return runGit(t, repo, "symbolic-ref", "--short", "HEAD")
}

// wantNoWrites comprueba que las invocaciones observadas no incluyen ninguna escritura de
// ramas ni de worktrees: ni `branch`, ni `switch`, ni `worktree add`.
func wantNoWrites(t *testing.T, seen [][]string, why string) {
	t.Helper()
	for _, invocation := range seen {
		if isWriteInvocation(invocation) {
			t.Errorf("%s, pero se lanzó %v", why, invocation)
		}
	}
}

// resetTree devuelve el árbol de trabajo del repositorio al último commit, sin rastro.
func resetTree(t *testing.T, repo string) {
	t.Helper()
	runGit(t, repo, "reset", "-q", "--hard")
	runGit(t, repo, "clean", "-fdq")
}

// REQ-25.9 y matriz de amenazas «Estado del commit»: con cambios con stage, sin stage o sin
// seguimiento, `in_place` rechaza con `dirty_worktree` sin modificar el repositorio (HEAD,
// rama, índice y árbol idénticos antes y después), nombrando los ficheros y las salidas
// (confirmar o guardar, o el modo `worktree`), sin lanzar ninguna escritura y sin crear la
// rama ni el registro.
func TestEnsureInPlaceRefusesADirtyTreeWithoutModifyingTheRepository(t *testing.T) {
	repo := embeddedCoreProject(t)
	writeFile(t, filepath.Join(repo, "datos.txt"), "versión 1\n")
	runGit(t, repo, "add", "datos.txt")
	commit(t, repo, "fichero rastreado")

	dirty := map[string]struct {
		prepare func()
		file    string
	}{
		"modificado sin stage": {func() { writeFile(t, filepath.Join(repo, "datos.txt"), "versión 2\n") }, "datos.txt"},
		"modificado con stage": {func() {
			writeFile(t, filepath.Join(repo, "datos.txt"), "versión 3\n")
			runGit(t, repo, "add", "datos.txt")
		}, "datos.txt"},
		"fichero nuevo con stage": {func() {
			writeFile(t, filepath.Join(repo, "nuevo.txt"), "nuevo\n")
			runGit(t, repo, "add", "nuevo.txt")
		}, "nuevo.txt"},
		"sin seguimiento": {func() { writeFile(t, filepath.Join(repo, "suelto.txt"), "suelto\n") }, "suelto.txt"},
	}
	for name, tc := range dirty {
		t.Run(name, func(t *testing.T) {
			resetTree(t, repo)
			tc.prepare()
			before := inPlaceState(t, repo)
			content, err := os.ReadFile(filepath.Join(repo, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			seen := observe(t)

			result := ensureInPlace(t, repo, "x")

			wantRefusal(t, result, branch.CodeDirtyWorktree, branch.StatusRefused)
			if !strings.Contains(result.Message, tc.file) {
				t.Errorf("el mensaje %q debe nombrar el fichero %s", result.Message, tc.file)
			}
			for _, want := range []string{
				"axiom branch ensure --change x --mode worktree --cwd " + pathquote.Quote(repo),
				"axiom branch ensure --change x --mode in_place --cwd " + pathquote.Quote(repo),
			} {
				if !strings.Contains(result.Continuation, want) {
					t.Errorf("la continuación %q debe contener %q", result.Continuation, want)
				}
			}
			if after := inPlaceState(t, repo); after != before {
				t.Errorf("el rechazo cambió el repositorio:\nantes:\n%s\ndespués:\n%s", before, after)
			}
			if after, err := os.ReadFile(filepath.Join(repo, tc.file)); err != nil || string(after) != string(content) {
				t.Errorf("el fichero %s perdió su contenido (%v): %q", tc.file, err, after)
			}
			wantNoWrites(t, *seen, "dirty_worktree se decide en la fase 3, antes de cualquier escritura")
			if runGit(t, repo, "branch", "--list", "feat/x") != "" {
				t.Error("el rechazo no debe crear la rama del cambio")
			}
			wantNoRegistryFile(t, repo, "x")
			if details := result.Details; details["repository"] == "" || details["paths"] == "" {
				t.Errorf("details = %v, se esperaba el repositorio y los ficheros", details)
			}
		})
	}

	t.Run("nombra todos los ficheros afectados", func(t *testing.T) {
		resetTree(t, repo)
		writeFile(t, filepath.Join(repo, "datos.txt"), "versión 4\n")
		writeFile(t, filepath.Join(repo, "suelto.txt"), "suelto\n")

		result := ensureInPlace(t, repo, "x")

		wantRefusal(t, result, branch.CodeDirtyWorktree, branch.StatusRefused)
		for _, want := range []string{"datos.txt", "suelto.txt"} {
			if !strings.Contains(result.Message, want) {
				t.Errorf("el mensaje %q debe nombrar %s", result.Message, want)
			}
		}
	})
}

// REQ-25.9: un fichero ignorado no cuenta como cambio sin confirmar: `in_place` procede.
func TestEnsureInPlaceTreatsAnIgnoredFileAsClean(t *testing.T) {
	repo := embeddedCoreProject(t)
	writeFile(t, filepath.Join(repo, ".gitignore"), "*.log\n")
	runGit(t, repo, "add", ".gitignore")
	commit(t, repo, "ignorar los registros")
	writeFile(t, filepath.Join(repo, "salida.log"), "ruido\n")

	result := ensureInPlace(t, repo, "x")

	wantReady(t, result)
	if got := currentBranch(t, repo); got != "feat/x" {
		t.Errorf("el repositorio está en %s, se esperaba feat/x", got)
	}
	if _, err := os.Stat(filepath.Join(repo, "salida.log")); err != nil {
		t.Errorf("el fichero ignorado debe seguir en su sitio: %v", err)
	}
}

// REQ-25.8, escenario «Modo `in_place` hace checkout en el propio repositorio», y design D-05
// fase 4 y 5: con el árbol limpio, `ensure` crea la rama desde la base y la despliega con
// `git switch` en el propio repositorio, sin worktree ni raíz de worktrees; el registro guarda
// el modo con `inPlace` en lugar del worktree y repetir da el mismo JSON byte a byte, sin
// ninguna escritura y sin reescribir el registro.
func TestEnsureInPlaceSwitchesTheRepositoryToTheChangeBranchWithoutAWorktree(t *testing.T) {
	repo := embeddedCoreProject(t)
	head := runGit(t, repo, "rev-parse", "HEAD")

	result := ensureInPlace(t, repo, "x")

	wantReady(t, result)
	wantMode(t, result, branch.ModeInPlace, branch.ModeSourceFlag)
	if got := currentBranch(t, repo); got != "feat/x" {
		t.Fatalf("el repositorio está en %s, se esperaba feat/x", got)
	}
	if got := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); got != head {
		t.Errorf("feat/x apunta a %s, se esperaba la base %s", got, head)
	}
	if listed := runGit(t, repo, "worktree", "list", "--porcelain"); strings.Count(listed, "worktree ") != 1 {
		t.Errorf("in_place no crea ningún worktree:\n%s", listed)
	}
	if _, err := os.Stat(branch.DefaultWorktreeRoot(canonicalDirectory(t, repo))); err == nil {
		t.Error("in_place no crea la raíz de worktrees")
	}
	repository := onlyRepository(t, result)
	if repository.CurrentBranch != "feat/x" {
		t.Errorf("rama actual del resultado = %q, se esperaba feat/x", repository.CurrentBranch)
	}
	if feature := branchByName(t, repository, "feat/x"); !feature.Exists || feature.Dirty {
		t.Errorf("rama del cambio = %+v", feature)
	}
	// Sin worktree, el workspaceRoot y el planning home son los del propio checkout.
	wantSameDirectory(t, "workspaceRoot", result.WorkspaceRoot, repo)
	if _, inside := relativeWithin(repo, result.PlanningHome); !inside {
		t.Errorf("planningHome %q debe estar en el propio repositorio %s", result.PlanningHome, repo)
	}

	record := loadRecord(t, repo, "x")
	if record.Mode != branch.ModeInPlace || record.ModeSource != branch.ModeSourceFlag {
		t.Errorf("registro = %q con origen %q, se esperaba in_place con origen flag", record.Mode, record.ModeSource)
	}
	registered := record.Repositories[0].Branches
	if len(registered) != 1 || registered[0].Name != "feat/x" || !registered[0].InPlace || registered[0].Worktree != "" {
		t.Errorf("ramas registradas = %+v, se esperaba feat/x con inPlace y sin worktree", registered)
	}

	t.Run("repetir es idempotente byte a byte y no escribe", func(t *testing.T) {
		first := resultJSON(t, result)
		state := inPlaceState(t, repo)
		recordPath := recordFile(commonDir(repo), "sdd", "x")
		old := setOldModTime(t, recordPath)
		seen := observe(t)

		second := ensureInPlace(t, repo, "x")

		wantReady(t, second)
		if got := resultJSON(t, second); got != first {
			t.Errorf("repetir cambió el resultado:\n%s\n---\n%s", first, got)
		}
		wantNoWrites(t, *seen, "repetir no tiene nada que crear ni que cambiar de rama")
		if got := inPlaceState(t, repo); got != state {
			t.Errorf("repetir cambió el repositorio:\n%s\n---\n%s", state, got)
		}
		if info, err := os.Stat(recordPath); err != nil || !info.ModTime().Equal(old) {
			t.Errorf("repetir no debe reescribir el registro (%v)", err)
		}
	})

	t.Run("el estado del registro se reconstruye si se borra", func(t *testing.T) {
		if err := os.Remove(recordFile(commonDir(repo), "sdd", "x")); err != nil {
			t.Fatal(err)
		}
		// Sin registro, el hub (`worktree`) manda: `--mode in_place` fija el modo otra vez.
		again := ensureInPlace(t, repo, "x")
		wantReady(t, again)
		if rebuilt := loadRecord(t, repo, "x"); rebuilt.Mode != branch.ModeInPlace || !rebuilt.Repositories[0].Branches[0].InPlace {
			t.Errorf("el registro reconstruido = %+v", rebuilt)
		}
	})
}

// REQ-25.9, escenario «Ya en la rama esperada con el árbol sucio es un no-op»: sin cambiar de
// rama no hay nada que proteger; `ensure` termina con código 0 sin modificar nada (ni escribe
// el árbol, ni lanza una escritura de git), también si el registro aún no existía.
func TestEnsureInPlaceIsANoOpWhenAlreadyOnTheExpectedBranchWithADirtyTree(t *testing.T) {
	repo := embeddedCoreProject(t)
	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	writeFile(t, filepath.Join(repo, "axiom.yaml"), configWithRoles(map[string][]string{"core": {"."}}, "# cambio local\n"))
	writeFile(t, filepath.Join(repo, "suelto.txt"), "suelto\n")
	before := inPlaceState(t, repo)
	seen := observe(t)

	result := ensureInPlace(t, repo, "x")

	wantReady(t, result)
	if result.ExitCode() != branch.ExitReady {
		t.Errorf("salida %d, se esperaba 0", result.ExitCode())
	}
	wantNoWrites(t, *seen, "ya en la rama esperada no se cambia de rama ni se crea nada")
	if after := inPlaceState(t, repo); after != before {
		t.Errorf("el no-op cambió el repositorio:\n%s\n---\n%s", before, after)
	}
	if feature := branchByName(t, onlyRepository(t, result), "feat/x"); !feature.Dirty {
		t.Errorf("la rama del cambio debe describirse con cambios sin confirmar: %+v", feature)
	}
	if record := loadRecord(t, repo, "x"); record.Mode != branch.ModeInPlace || !record.Repositories[0].Branches[0].InPlace {
		t.Errorf("registro = %+v: el no-op también registra el modo y la rama desplegada", record)
	}

	first := resultJSON(t, result)
	if second := resultJSON(t, ensureInPlace(t, repo, "x")); second != first {
		t.Errorf("repetir con el árbol sucio cambió el resultado:\n%s\n---\n%s", first, second)
	}
}

// REQ-25.9 con un cambio ya materializado en `in_place`: si el repositorio salió de la rama
// del cambio y tiene el árbol sucio, `ensure` rechaza igual, pero no ofrece `--mode worktree`,
// que el registro rechazaría con `plan_conflict`: solo confirmar o guardar y repetir.
func TestEnsureInPlaceOffersNoWorktreeModeForAChangeAlreadyMaterializedInPlace(t *testing.T) {
	repo := embeddedCoreProject(t)
	wantReady(t, ensureInPlace(t, repo, "x"))
	runGit(t, repo, "switch", "-q", "main")
	writeFile(t, filepath.Join(repo, "suelto.txt"), "suelto\n")

	result := ensureInPlace(t, repo, "x")

	wantRefusal(t, result, branch.CodeDirtyWorktree, branch.StatusRefused)
	if strings.Contains(result.Continuation, "--mode worktree") || !strings.Contains(result.Continuation, "--mode in_place") {
		t.Errorf("la continuación %q debe repetir con in_place y no ofrecer worktree", result.Continuation)
	}
	if got := currentBranch(t, repo); got != "main" {
		t.Errorf("el repositorio está en %s, se esperaba main", got)
	}
}

// design D-05 fase 4: «nunca mueve ni reinicia una rama existente». Con la rama del cambio
// ya creada (en otro commit), `in_place` solo la despliega con `git switch`.
func TestEnsureInPlaceDeploysAnExistingBranchWithoutMovingIt(t *testing.T) {
	repo := embeddedCoreProject(t)
	runGit(t, repo, "branch", "feat/x")
	before := runGit(t, repo, "rev-parse", "refs/heads/feat/x")
	commit(t, repo, "avanza main")
	seen := observe(t)

	result := ensureInPlace(t, repo, "x")

	wantReady(t, result)
	if got := currentBranch(t, repo); got != "feat/x" {
		t.Errorf("el repositorio está en %s, se esperaba feat/x", got)
	}
	if after := runGit(t, repo, "rev-parse", "refs/heads/feat/x"); after != before {
		t.Errorf("feat/x se movió de %s a %s", before, after)
	}
	for _, invocation := range *seen {
		if invocation[1] == "branch" && len(invocation) > 2 && invocation[2] == "--no-track" {
			t.Errorf("la rama ya existía y no debía crearse: %v", invocation)
		}
	}
}

// design D-08 y REQ-25.8: el hub en `in_place` fija el modo cuando no hay registro ni
// `--mode` (origen `hub`), y un cambio materializado en `in_place` rechaza un `--mode worktree`
// con `plan_conflict` sin tocar nada.
func TestEnsureInPlaceFromTheHubAndAModeConflictWithTheRegistry(t *testing.T) {
	repo := embeddedCoreProject(t)

	result := ensureWith(t, hubInPlace, EnsureRequest{CWD: repo, Change: "x"})

	wantReady(t, result)
	wantMode(t, result, branch.ModeInPlace, branch.ModeSourceHub)
	if got := currentBranch(t, repo); got != "feat/x" {
		t.Fatalf("el repositorio está en %s, se esperaba feat/x", got)
	}

	state := inPlaceState(t, repo)
	conflict := ensureWith(t, hubInPlace, EnsureRequest{CWD: repo, Change: "x", Mode: branch.ModeWorktree})

	wantRefusal(t, conflict, branch.CodePlanConflict, branch.StatusRefused)
	if !strings.Contains(conflict.Continuation, "axiom branch ensure --change x --mode in_place --cwd") {
		t.Errorf("la continuación %q debe repetir con el modo materializado", conflict.Continuation)
	}
	if got := inPlaceState(t, repo); got != state {
		t.Errorf("el rechazo cambió el repositorio:\n%s\n---\n%s", state, got)
	}
}

// design D-05 fase 3 y R26: con ramas de rol, `in_place` crea todas las ramas del plan pero
// solo despliega la de seguimiento (la del cambio); las de rol quedan creadas desde ella.
func TestEnsureInPlaceCreatesTheRoleBranchesButOnlyDeploysTheTrackingBranch(t *testing.T) {
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))
	sealChange(t, repo, "x", "core", "qa")
	runGit(t, repo, "add", "-A")
	commit(t, repo, "kickoff sellado")
	head := runGit(t, repo, "rev-parse", "HEAD")

	result := ensureInPlace(t, repo, "x")

	wantReady(t, result)
	if got := currentBranch(t, repo); got != "feat/x" {
		t.Fatalf("el repositorio está en %s, se esperaba la rama de seguimiento feat/x", got)
	}
	for _, name := range []string{"feat/x", "feat/x-core", "feat/x-qa"} {
		if got := runGit(t, repo, "rev-parse", "refs/heads/"+name); got != head {
			t.Errorf("%s apunta a %s, se esperaba %s", name, got, head)
		}
	}
	if listed := runGit(t, repo, "worktree", "list", "--porcelain"); strings.Count(listed, "worktree ") != 1 {
		t.Errorf("in_place no crea ningún worktree, tampoco para las ramas de rol:\n%s", listed)
	}
	registered := loadRecord(t, repo, "x").Repositories[0].Branches
	if len(registered) != 3 {
		t.Fatalf("ramas registradas = %+v, se esperaban 3", registered)
	}
	for _, recorded := range registered {
		if want := recorded.Name == "feat/x"; recorded.InPlace != want || recorded.Worktree != "" {
			t.Errorf("rama registrada %+v: inPlace solo en la desplegada y ninguna con worktree", recorded)
		}
	}
}

// design D-05 y matriz de amenazas «Selección de repositorio git»: con la rama del cambio
// desplegada en un worktree que el plan no conoce, no se puede desplegar también en el
// repositorio: `branch_checked_out_elsewhere`, sin tocar nada (git no admite la misma rama
// en dos worktrees).
func TestEnsureInPlaceRefusesABranchCheckedOutInAnotherWorktree(t *testing.T) {
	repo := embeddedCoreProject(t)
	elsewhere := filepath.Join(t.TempDir(), "otro")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", elsewhere, "main")
	state := inPlaceState(t, repo)
	seen := observe(t)

	result := ensureInPlace(t, repo, "x")

	wantRefusal(t, result, branch.CodeBranchCheckedOutElsewhere, branch.StatusRefused)
	if !strings.Contains(result.Message, "feat/x") || !strings.Contains(result.Message, filepath.Base(elsewhere)) {
		t.Errorf("el mensaje %q debe nombrar la rama y su worktree", result.Message)
	}
	wantNoWrites(t, *seen, "el rechazo es de la fase 3")
	if got := inPlaceState(t, repo); got != state {
		t.Errorf("el rechazo cambió el repositorio:\n%s\n---\n%s", state, got)
	}
	wantNoRegistryFile(t, repo, "x")
}

// D-05, fase 3: la comprobación cubre TODOS los repositorios del plan antes de mutar ninguno.
// Con el segundo repositorio sucio, el primero (limpio) no cambia de rama ni crea la rama del
// cambio.
func TestEnsureInPlaceChecksEveryRepositoryForDirtinessBeforeSwitchingAny(t *testing.T) {
	master, specs := newMultirepo(t, map[string][]string{"backend": {"repo-backend"}, "frontend": {"repo-frontend"}}, "repo-backend", "repo-frontend")
	sealChangeAt(t, specs, "x", "backend", "frontend")
	runGit(t, specs, "add", "-A")
	commit(t, specs, "kickoff sellado")
	backend, frontend := filepath.Join(master, "repo-backend"), filepath.Join(master, "repo-frontend")
	writeFile(t, filepath.Join(frontend, "suelto.txt"), "sin añadir\n")
	backendState, frontendState := inPlaceState(t, backend), inPlaceState(t, frontend)

	result := ensureInPlace(t, master, "x")

	wantRefusal(t, result, branch.CodeDirtyWorktree, branch.StatusRefused)
	if !strings.Contains(result.Message, "repo-frontend") || !strings.Contains(result.Message, "suelto.txt") {
		t.Errorf("el mensaje %q debe nombrar el repositorio sucio y su fichero", result.Message)
	}
	if got := inPlaceState(t, backend); got != backendState {
		t.Errorf("repo-backend (limpio) no debía cambiar:\n%s\n---\n%s", backendState, got)
	}
	if got := inPlaceState(t, frontend); got != frontendState {
		t.Errorf("repo-frontend (sucio) no debía cambiar:\n%s\n---\n%s", frontendState, got)
	}

	t.Run("con los dos limpios cambian los dos de rama y el de specs no se toca", func(t *testing.T) {
		specsState := inPlaceState(t, specs)
		if err := os.Remove(filepath.Join(frontend, "suelto.txt")); err != nil {
			t.Fatal(err)
		}

		wantReady(t, ensureInPlace(t, master, "x"))

		for _, repo := range []string{backend, frontend} {
			if got := currentBranch(t, repo); got != "feat/x" {
				t.Errorf("%s está en %s, se esperaba feat/x", repo, got)
			}
		}
		if got := inPlaceState(t, specs); got != specsState {
			t.Errorf("el repositorio de specs no recibe ramas ni checkout:\n%s\n---\n%s", specsState, got)
		}
	})
}

// REQ-25.2 y D-05: un fallo de git en el `git switch` es `git_failed`, que nombra la operación;
// la rama ya creada se conserva, el repositorio no cambia de rama, no se escribe el registro y
// repetir `ensure` completa lo pendiente.
func TestEnsureInPlaceReportsGitFailedWhenSwitchFailsAndRepeatingCompletes(t *testing.T) {
	repo := embeddedCoreProject(t)
	engine := newEngine(runtime.GOOS)
	engine.writer = failingWriter{real: gitcli.New(), failSwitch: true}
	request := EnsureRequest{CWD: repo, Change: "x", Lane: branch.LaneSDD, Mode: branch.ModeInPlace}

	result := engine.Ensure(context.Background(), request)

	wantRefusal(t, result, branch.CodeGitFailed, branch.StatusError)
	if !strings.Contains(result.Details["operation"], "feat/x") || !strings.Contains(result.Message, "fallo simulado") {
		t.Errorf("la operación %q y el mensaje %q deben nombrar la rama y resumir el error", result.Details["operation"], result.Message)
	}
	if runGit(t, repo, "branch", "--list", "feat/x") == "" {
		t.Error("la rama ya creada se conserva: repetir completa el cambio de rama")
	}
	if got := currentBranch(t, repo); got != "main" {
		t.Errorf("el repositorio está en %s, se esperaba main", got)
	}
	wantNoRegistryFile(t, repo, "x")

	wantReady(t, ensureInPlace(t, repo, "x"))
	if got := currentBranch(t, repo); got != "feat/x" {
		t.Errorf("repetir ensure debía cambiar a feat/x, está en %s", got)
	}
}

// REQ-25.8 y REQ-25.9: la suciedad solo importa en `in_place`. En `worktree`, `ensure` no
// cambia el checkout principal y por tanto no la mira (escenario «El checkout principal no
// se toca en modo `worktree`»).
func TestEnsureWorktreeModeIgnoresADirtyTree(t *testing.T) {
	repo := embeddedCoreProject(t)
	writeFile(t, filepath.Join(repo, "suelto.txt"), "suelto\n")

	result := ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x", Mode: branch.ModeWorktree})

	wantReady(t, result)
	if got := currentBranch(t, repo); got != "main" {
		t.Errorf("el checkout principal está en %s, se esperaba main", got)
	}
}
