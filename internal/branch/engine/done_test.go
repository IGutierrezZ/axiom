package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/gitcli"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// Pruebas de `done` con git real, entorno aislado y sin red (REQ-25.16, REQ-25.17, design D-09,
// matriz de amenazas «Borrado de ramas y worktrees»): la fusión se confirma por ascendencia, el
// desmontaje retira el worktree y después borra la rama local desde el checkout principal, cada
// rama (de rol, del cambio y de archivado) se procesa por separado y nunca se toca una rama remota.
// Cada `git worktree add` cuesta segundos en Windows: las pruebas comparten worktrees y usan ramas
// sueltas o `in_place` cuando el worktree no es lo que se comprueba.

func doneRequest(cwd string) DoneRequest {
	return DoneRequest{CWD: cwd, Change: "x", Lane: branch.LaneSDD}
}

func doneOf(t *testing.T, request DoneRequest) branch.Result {
	t.Helper()
	return Default().Done(context.Background(), request)
}

// mergeInto fusiona name en la rama actual de repo con un merge commit.
func mergeInto(t *testing.T, repo, name string) {
	t.Helper()
	runGit(t, repo, "merge", "-q", "--no-ff", "-m", "fusiona "+name, name)
}

func branchExists(t *testing.T, repo, name string) bool {
	t.Helper()
	return runGit(t, repo, "branch", "--list", name) != ""
}

// isDoneWrite dice si una invocación de git es de las que `done` puede lanzar para cambiar algo:
// retirar un worktree, borrar una rama local o cambiar de rama (`in_place`).
func isDoneWrite(invocation []string) bool {
	return invocation[1] == "branch" || invocation[1] == "switch" || (invocation[1] == "worktree" && len(invocation) > 2 && invocation[2] == "remove")
}

func doneWrites(seen [][]string) []string {
	var writes []string
	for _, invocation := range seen {
		if isDoneWrite(invocation) {
			writes = append(writes, strings.Join(invocation[1:], " "))
		}
	}
	return writes
}

// wantNoRemote comprueba que ninguna invocación observada es remota (REQ-25.1): ni fetch, ni pull,
// ni push, ni ls-remote.
func wantNoRemote(t *testing.T, seen [][]string) {
	t.Helper()
	for _, invocation := range seen {
		if remoteOperation(invocation[1]) {
			t.Errorf("`done` sin --fetch no hace ninguna operación remota, pero lanzó %v", invocation)
		}
	}
}

// wantDoneStatus comprueba el estado tipado de `done` (`refused`, `partial`...), su código, su
// salida 1 y que lleva mensaje y continuación. A diferencia de un rechazo de `ensure`, un `done`
// que ya reunió las ramas las describe también cuando no alcanza el estado pedido.
func wantDoneStatus(t *testing.T, result branch.Result, status branch.Status, code branch.Code) {
	t.Helper()
	if result.Operation != "done" || result.Status != status || result.Code != code || result.ExitCode() != branch.ExitRefused {
		t.Fatalf("operación %q, status %q, código %q (%s), salida %d; se esperaba done, %s con %s y salida 1", result.Operation, result.Status, result.Code, result.Message, result.ExitCode(), status, code)
	}
	if result.Message == "" || result.Continuation == "" || result.StatusLabel == "" {
		t.Errorf("todo rechazo lleva mensaje, continuación y la etiqueta del estado: %+v", result)
	}
}

func wantOutcome(t *testing.T, described branch.BranchResult, outcome branch.DoneOutcome, layer branch.MergeLayer, code branch.Code) {
	t.Helper()
	if described.Outcome != outcome || described.ConfirmedBy != layer || described.Code != code {
		t.Fatalf("rama %s: resultado %q confirmado por %q con código %q; se esperaba %q, %q y %q (%s)", described.Name, described.Outcome, described.ConfirmedBy, described.Code, outcome, layer, code, described.Message)
	}
	if described.OutcomeLabel == "" || (layer != "" && described.ConfirmedByLabel == "") || (code != "" && (described.Message == "" || described.Continuation == "")) {
		t.Errorf("rama %s: cada token lleva su etiqueta y todo rechazo, mensaje y continuación: %+v", described.Name, described)
	}
}

func branchNames(repository branch.RepositoryResult) []string {
	var names []string
	for _, described := range repository.Branches {
		names = append(names, described.Name)
	}
	return names
}

// REQ-25.16 (escenario «Fusión real detectada por ascendencia») y REQ-25.17 (escenarios
// «Eliminación tras una fusión confirmada» y «La rama de archivado se elimina como las demás»): con
// el cambio y su rama de archivado fusionados con un merge commit, `done` retira cada worktree y
// después borra la rama local desde el checkout principal, la confirma con el token `ancestry` y
// su etiqueta, borra el registro, no hace ninguna operación remota y deja intacta la rama remota.
// Repetirlo no cambia nada y sale con 0.
func TestDoneRemovesAMergedWorktreeAndItsLocalBranchFromTheMainCheckout(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"}))
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	writeFile(t, filepath.Join(worktree, "cambio.txt"), "trabajo del cambio\n")
	runGit(t, worktree, "add", "-A")
	commit(t, worktree, "trabajo del cambio")
	runGit(t, repo, "branch", "feat/x-archive", "main")
	archive := archiveWorktree(t, repo)
	runGit(t, repo, "worktree", "add", "-q", archive, "feat/x-archive")
	runGit(t, repo, "push", "-q", "origin", "feat/x")
	remoteTip := runGit(t, repo, "rev-parse", "refs/remotes/origin/feat/x")
	mergeInto(t, repo, "feat/x")
	seen := observe(t)

	result := doneOf(t, doneRequest(repo))

	wantReady(t, result)
	if result.Operation != "done" || result.Change != "x" || result.Lane != branch.LaneSDD || result.Code != "" {
		t.Errorf("operación %q, cambio %q, carril %q, código %q", result.Operation, result.Change, result.Lane, result.Code)
	}
	repository := onlyRepository(t, result)
	if want := []string{"feat/x", "feat/x-archive"}; !slices.Equal(branchNames(repository), want) {
		t.Fatalf("ramas = %v, se esperaban las del cambio y de archivado por separado: %v", branchNames(repository), want)
	}
	change, archived := branchByName(t, repository, "feat/x"), branchByName(t, repository, "feat/x-archive")
	wantOutcome(t, change, branch.OutcomeRemoved, branch.LayerAncestry, "")
	wantOutcome(t, archived, branch.OutcomeRemoved, branch.LayerAncestry, "")
	if change.Kind != branch.KindChange || archived.Kind != branch.KindArchive || change.Exists || archived.Exists {
		t.Errorf("rama del cambio %+v, rama de archivado %+v", change, archived)
	}
	wantSamePlace(t, "worktree retirado de feat/x", change.Worktree, worktree)
	wantSamePlace(t, "worktree retirado de feat/x-archive", archived.Worktree, archive)
	for _, path := range []string{worktree, archive} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("el worktree %s debía desaparecer del disco: %v", path, err)
		}
	}
	for _, name := range []string{"feat/x", "feat/x-archive"} {
		if branchExists(t, repo, name) {
			t.Errorf("la rama local %s debía eliminarse", name)
		}
	}
	if listed := runGit(t, repo, "worktree", "list", "--porcelain"); strings.Count(listed, "worktree ") != 1 {
		t.Errorf("solo debe quedar el checkout principal:\n%s", listed)
	}
	if result.Registry == nil || result.Registry.Present {
		t.Errorf("registry = %+v, sin ramas registradas el registro se borra", result.Registry)
	}
	wantNoRegistryFile(t, repo, "x")
	if got := runGit(t, repo, "rev-parse", "refs/remotes/origin/feat/x"); got != remoteTip {
		t.Errorf("la referencia remota local cambió de %s a %s", remoteTip, got)
	}
	if runGit(t, remote, "branch", "--list", "feat/x") == "" {
		t.Error("la rama remota feat/x debía quedar intacta")
	}
	for _, want := range []string{`"outcome": "removed"`, `"outcomeLabel": "eliminada"`, `"confirmedBy": "ancestry"`, `"confirmedByLabel": "ascendencia"`} {
		if json := resultJSON(t, result); !strings.Contains(json, want) {
			t.Errorf("el JSON debe contener %s:\n%s", want, json)
		}
	}

	wantNoRemote(t, *seen)
	wantWrites := []string{"worktree remove " + change.Worktree, "branch -D feat/x", "worktree remove " + archived.Worktree, "branch -D feat/x-archive"}
	if writes := doneWrites(*seen); !slices.Equal(writes, wantWrites) {
		t.Errorf("escrituras = %v, se esperaban %v: cada rama se procesa por separado y su worktree se retira antes que la rama", writes, wantWrites)
	}
	for _, invocation := range *seen {
		if isDoneWrite(invocation) && !pathidentity.SameDirectory(invocation[0], repo) {
			t.Errorf("toda escritura se lanza desde el checkout principal %s, pero %v se lanzó desde %s", repo, invocation[1:], invocation[0])
		}
	}

	t.Run("repetirlo no cambia nada y sale con 0", func(t *testing.T) {
		state := repositoryState(t, repo)
		again := observe(t)

		second := doneOf(t, doneRequest(repo))

		wantReady(t, second)
		wantOutcome(t, branchByName(t, onlyRepository(t, second), "feat/x"), branch.OutcomeAbsent, "", "")
		if writes := doneWrites(*again); len(writes) != 0 {
			t.Errorf("la repetición no escribe, pero lanzó %v", writes)
		}
		if got := repositoryState(t, repo); got != state {
			t.Errorf("la repetición cambió el repositorio:\n%s\n---\n%s", state, got)
		}
	})
}

// REQ-25.17 (escenarios «Árbol sucio impide la eliminación», «Rama sin fusionar impide la
// eliminación» y «`--force` elimina con fusión no confirmada y avisa de lo descartado») y matriz de
// amenazas «Borrado de ramas y worktrees»: un worktree con cambios sin confirmar da `dirty_worktree`
// aunque la rama esté fusionada; limpio pero sin fusionar, `merge_not_confirmed`; en ambos casos se
// conserva todo y la continuación ofrece `--force`. Con `--force` se eliminan, el JSON lista lo
// descartado, la punta (SHA) de la rama eliminada y la capa `forced`, y no hay operaciones remotas.
func TestDoneRefusesADirtyWorktreeOrAnUnmergedBranchUnlessForced(t *testing.T) {
	repo := embeddedCoreProject(t)
	wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"}))
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	writeFile(t, filepath.Join(worktree, "trabajo.txt"), "trabajo\n")
	runGit(t, worktree, "add", "-A")
	commit(t, worktree, "primer commit")
	mergeInto(t, repo, "feat/x")
	writeFile(t, filepath.Join(worktree, "sucio.txt"), "sin añadir\n")
	seen := observe(t)

	t.Run("fusionada pero con cambios sin confirmar: dirty_worktree", func(t *testing.T) {
		state := repositoryState(t, repo)

		result := doneOf(t, doneRequest(repo))

		wantDoneStatus(t, result, branch.StatusRefused, branch.CodeDirtyWorktree)
		described := branchByName(t, onlyRepository(t, result), "feat/x")
		wantOutcome(t, described, branch.OutcomeKept, "", branch.CodeDirtyWorktree)
		if !strings.Contains(result.Message, "sucio.txt") || !strings.Contains(result.Continuation, "--force") || !strings.Contains(result.Continuation, "stash") {
			t.Errorf("el mensaje %q debe nombrar el fichero y la continuación %q ofrecer guardarlo o --force", result.Message, result.Continuation)
		}
		if !strings.Contains(result.Details["paths"], "sucio.txt") || len(described.Discarded) != 0 {
			t.Errorf("details = %v, descartados = %v: un rechazo no descarta nada", result.Details, described.Discarded)
		}
		if got := repositoryState(t, repo); got != state || !branchExists(t, repo, "feat/x") {
			t.Errorf("el rechazo no debe tocar worktrees ni ramas:\n%s\n---\n%s", state, got)
		}
		if _, err := os.Stat(filepath.Join(worktree, "sucio.txt")); err != nil {
			t.Errorf("el rechazo no debe perder el fichero sin confirmar: %v", err)
		}
		if result.Registry == nil || !result.Registry.Present {
			t.Errorf("registry = %+v, la rama se conserva y el registro también", result.Registry)
		}
	})

	t.Run("sin fusionar y con el árbol limpio: merge_not_confirmed", func(t *testing.T) {
		if err := os.Remove(filepath.Join(worktree, "sucio.txt")); err != nil {
			t.Fatal(err)
		}
		commit(t, worktree, "segundo commit, ya sin fusionar")
		state := repositoryState(t, repo)

		result := doneOf(t, doneRequest(repo))

		wantDoneStatus(t, result, branch.StatusRefused, branch.CodeMergeNotConfirmed)
		wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeKept, "", branch.CodeMergeNotConfirmed)
		if want := "axiom branch done --change x --cwd"; !strings.Contains(result.Continuation, want) || !strings.Contains(result.Continuation, "--force") {
			t.Errorf("la continuación %q debe repetir `%s` y ofrecer --force", result.Continuation, want)
		}
		if got := repositoryState(t, repo); got != state || !branchExists(t, repo, "feat/x") {
			t.Errorf("el rechazo no debe tocar worktrees ni ramas:\n%s\n---\n%s", state, got)
		}
	})

	t.Run("--force elimina, lista lo descartado y deja la punta para recuperar la rama", func(t *testing.T) {
		writeFile(t, filepath.Join(worktree, "sucio.txt"), "sin añadir\n")
		tip := runGit(t, repo, "rev-parse", "refs/heads/feat/x")
		request := doneRequest(repo)
		request.Force = true

		result := doneOf(t, request)

		wantReady(t, result)
		described := branchByName(t, onlyRepository(t, result), "feat/x")
		wantOutcome(t, described, branch.OutcomeRemoved, branch.LayerForced, "")
		if described.ConfirmedByLabel != "forzado" || described.TipSha != tip || !slices.Equal(described.Discarded, []string{"sucio.txt"}) {
			t.Errorf("rama = %+v, se esperaba la capa forzado, la punta %s y descartado sucio.txt", described, tip)
		}
		if _, err := os.Stat(worktree); !os.IsNotExist(err) || branchExists(t, repo, "feat/x") {
			t.Errorf("con --force se eliminan el worktree (%v) y la rama local (%v)", err, branchExists(t, repo, "feat/x"))
		}
		if got := runGit(t, repo, "rev-parse", tip+"^{commit}"); got != tip {
			t.Errorf("la punta %s debe seguir en el repositorio para recuperar la rama", tip)
		}
		for _, want := range []string{`"discarded"`, `"tipSha": "` + tip + `"`, `"confirmedBy": "forced"`, `"confirmedByLabel": "forzado"`} {
			if json := resultJSON(t, result); !strings.Contains(json, want) {
				t.Errorf("el JSON debe contener %s:\n%s", want, json)
			}
		}
	})
	wantNoRemote(t, *seen)
}

// REQ-25.17 (escenario «Ramas de rol y de seguimiento se procesan por separado»): con `feat/x-core`
// fusionada en `feat/x` y `feat/x` aún sin fusionar en `main`, se elimina `feat/x-core` y se
// conservan `feat/x` y la rama de `qa` (sin fusionar) con `merge_not_confirmed`; el resultado es
// `partial`, el código de salida es 1 y el registro conserva solo lo que queda. Las ramas de rol
// van antes que la del cambio. Una vez fusionado todo, el repositorio `in_place` vuelve a la rama de
// integración, se elimina el resto y desaparece el registro.
func TestDoneProcessesTheRoleAndChangeBranchesSeparately(t *testing.T) {
	repo := newProject(t, configWithRoles(map[string][]string{"core": {"."}, "qa": {"."}}, ""))
	sealChange(t, repo, "x", "core", "qa")
	runGit(t, repo, "add", "-A")
	commit(t, repo, "kickoff")
	wantReady(t, ensureInPlace(t, repo, "x"))
	runGit(t, repo, "switch", "-q", "feat/x-core")
	commit(t, repo, "trabajo de core")
	runGit(t, repo, "switch", "-q", "feat/x")
	mergeInto(t, repo, "feat/x-core")
	runGit(t, repo, "switch", "-q", "feat/x-qa")
	commit(t, repo, "trabajo de qa, sin fusionar")
	runGit(t, repo, "switch", "-q", "feat/x")
	seen := observe(t)

	result := doneOf(t, doneRequest(repo))

	wantDoneStatus(t, result, branch.StatusPartial, branch.CodeMergeNotConfirmed)
	if result.StatusLabel != "parcial" {
		t.Errorf("etiqueta %q, se esperaba «parcial»", result.StatusLabel)
	}
	repository := onlyRepository(t, result)
	if want := []string{"feat/x-core", "feat/x-qa", "feat/x"}; !slices.Equal(branchNames(repository), want) {
		t.Fatalf("ramas = %v, se esperaban las de rol antes que la del cambio: %v", branchNames(repository), want)
	}
	core := branchByName(t, repository, "feat/x-core")
	wantOutcome(t, core, branch.OutcomeRemoved, branch.LayerAncestry, "")
	if core.Kind != branch.KindRole || core.Role != "core" || core.PRTarget != "feat/x" {
		t.Errorf("rama de rol = %+v", core)
	}
	wantOutcome(t, branchByName(t, repository, "feat/x-qa"), branch.OutcomeKept, "", branch.CodeMergeNotConfirmed)
	wantOutcome(t, branchByName(t, repository, "feat/x"), branch.OutcomeKept, "", branch.CodeMergeNotConfirmed)
	if branchExists(t, repo, "feat/x-core") || !branchExists(t, repo, "feat/x-qa") || !branchExists(t, repo, "feat/x") {
		t.Error("solo la rama de core debía eliminarse")
	}
	if got := currentBranch(t, repo); got != "feat/x" {
		t.Errorf("el repositorio está en %s: sin fusión confirmada no cambia de rama", got)
	}
	if writes := doneWrites(*seen); !slices.Equal(writes, []string{"branch -D feat/x-core"}) {
		t.Errorf("escrituras = %v, solo se borra la rama de core", writes)
	}
	record := loadRecord(t, repo, "x")
	var recorded []string
	for _, registered := range record.Repositories[0].Branches {
		recorded = append(recorded, registered.Name)
	}
	if !slices.Equal(recorded, []string{"feat/x", "feat/x-qa"}) && !slices.Equal(recorded, []string{"feat/x-qa", "feat/x"}) {
		t.Errorf("el registro guarda %v, se esperaban solo las ramas conservadas", recorded)
	}
	if result.Registry == nil || !result.Registry.Present {
		t.Errorf("registry = %+v, el registro conserva las ramas que quedan", result.Registry)
	}
	for _, text := range []string{"feat/x-core (rama de rol): eliminada", "feat/x (rama del cambio): conservada"} {
		var out, errOut strings.Builder
		result.WriteText(&out, &errOut)
		if !strings.Contains(out.String(), text) || !strings.Contains(errOut.String(), "Error [merge_not_confirmed]") {
			t.Errorf("el texto debe describir cada rama en stdout (%q) y el error en stderr:\n%s\n---\n%s", text, out.String(), errOut.String())
		}
	}

	t.Run("fusionado todo, in_place vuelve a la integración y elimina el resto", func(t *testing.T) {
		mergeInto(t, repo, "feat/x-qa")
		runGit(t, repo, "branch", "-f", "main", "feat/x")

		result := doneOf(t, doneRequest(repo))

		wantReady(t, result)
		repository := onlyRepository(t, result)
		wantOutcome(t, branchByName(t, repository, "feat/x-core"), branch.OutcomeAbsent, "", "")
		wantOutcome(t, branchByName(t, repository, "feat/x-qa"), branch.OutcomeRemoved, branch.LayerAncestry, "")
		wantOutcome(t, branchByName(t, repository, "feat/x"), branch.OutcomeRemoved, branch.LayerAncestry, "")
		if repository.CurrentBranch != "main" || currentBranch(t, repo) != "main" {
			t.Errorf("el repositorio debe quedar en main: resultado %q, git %q", repository.CurrentBranch, currentBranch(t, repo))
		}
		if branchExists(t, repo, "feat/x") || branchExists(t, repo, "feat/x-qa") {
			t.Error("las ramas fusionadas debían eliminarse")
		}
		wantNoRegistryFile(t, repo, "x")
	})
}

// REQ-25.17 (escenarios de `in_place`): con el árbol sucio, `done` y `done --force` se rechazan con
// `dirty_worktree` sin cambiar de rama ni borrar nada (`--force` no salta la limpieza y la
// continuación no lo ofrece); limpio y sin fusionar, `merge_not_confirmed` sin cambiar de rama; con
// `--force` y el árbol limpio, vuelve a la integración, elimina la rama y deja la punta en el
// informe.
func TestDoneInPlaceNeverSkipsACleanTreeEvenWhenForced(t *testing.T) {
	repo := embeddedCoreProject(t)
	wantReady(t, ensureInPlace(t, repo, "x"))
	commit(t, repo, "trabajo sin fusionar")
	writeFile(t, filepath.Join(repo, "sucio.txt"), "sin añadir\n")
	state := inPlaceState(t, repo)
	seen := observe(t)

	for _, force := range []bool{false, true} {
		request := doneRequest(repo)
		request.Force = force

		result := doneOf(t, request)

		wantDoneStatus(t, result, branch.StatusRefused, branch.CodeDirtyWorktree)
		wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeKept, "", branch.CodeDirtyWorktree)
		if offered := branch.DoneInvocation("x", repo, branch.LaneSDD, "--force"); strings.Contains(result.Continuation, offered) {
			t.Errorf("force=%v: en in_place la continuación no ofrece `%s`: %s", force, offered, result.Continuation)
		}
		if got := inPlaceState(t, repo); got != state {
			t.Errorf("force=%v: el rechazo no debe tocar el repositorio:\n%s\n---\n%s", force, state, got)
		}
	}
	if writes := doneWrites(*seen); len(writes) != 0 {
		t.Errorf("el rechazo no escribe nada, pero lanzó %v", writes)
	}
	wantNoRemote(t, *seen)

	t.Run("limpio y sin fusionar: merge_not_confirmed sin cambiar de rama", func(t *testing.T) {
		resetTree(t, repo)
		seen := observe(t)

		result := doneOf(t, doneRequest(repo))

		wantDoneStatus(t, result, branch.StatusRefused, branch.CodeMergeNotConfirmed)
		if got := currentBranch(t, repo); got != "feat/x" || !branchExists(t, repo, "feat/x") {
			t.Errorf("el repositorio está en %s: sin fusión confirmada no cambia de rama ni borra nada", got)
		}
		if writes := doneWrites(*seen); len(writes) != 0 {
			t.Errorf("el rechazo no escribe nada, pero lanzó %v", writes)
		}
	})

	t.Run("--force con el árbol limpio vuelve a la integración y elimina la rama", func(t *testing.T) {
		tip := runGit(t, repo, "rev-parse", "refs/heads/feat/x")
		request := doneRequest(repo)
		request.Force = true

		result := doneOf(t, request)

		wantReady(t, result)
		described := branchByName(t, onlyRepository(t, result), "feat/x")
		wantOutcome(t, described, branch.OutcomeRemoved, branch.LayerForced, "")
		if described.TipSha != tip || len(described.Discarded) != 0 || described.Worktree != "" {
			t.Errorf("rama = %+v, se esperaba la punta %s, nada descartado y ningún worktree", described, tip)
		}
		if got := currentBranch(t, repo); got != "main" || branchExists(t, repo, "feat/x") {
			t.Errorf("el repositorio está en %s, se esperaba main sin feat/x", got)
		}
		wantNoRegistryFile(t, repo, "x")
	})
}

// REQ-25.16 y REQ-25.17 (rama de archivado): la rama de archivado se procesa aparte de la del cambio
// y se confirma con la integración; una de archivado sin fusionar se conserva con
// `merge_not_confirmed` aunque la del cambio ya se haya eliminado, y la siguiente ejecución, con
// ambas fusionadas, sale con 0. (Que `pr_merged` no confirma el archivado lo fija
// TestConfirmMergeStopsAtTheFirstLayerThatConfirms.)
func TestDoneProcessesTheArchiveBranchSeparatelyFromTheChangeBranch(t *testing.T) {
	repo := embeddedCoreProject(t)
	runGit(t, repo, "branch", "feat/x", "main")
	runGit(t, repo, "switch", "-q", "-c", "feat/x-archive")
	commit(t, repo, "documentación de archivado")
	runGit(t, repo, "switch", "-q", "main")

	result := doneOf(t, doneRequest(repo))

	wantDoneStatus(t, result, branch.StatusPartial, branch.CodeMergeNotConfirmed)
	repository := onlyRepository(t, result)
	wantOutcome(t, branchByName(t, repository, "feat/x"), branch.OutcomeRemoved, branch.LayerAncestry, "")
	archived := branchByName(t, repository, "feat/x-archive")
	wantOutcome(t, archived, branch.OutcomeKept, "", branch.CodeMergeNotConfirmed)
	if archived.Kind != branch.KindArchive || archived.PRTarget != "main" {
		t.Errorf("rama de archivado = %+v, su PR va a la integración", archived)
	}
	if branchExists(t, repo, "feat/x") || !branchExists(t, repo, "feat/x-archive") {
		t.Error("solo la rama del cambio debía eliminarse")
	}

	mergeInto(t, repo, "feat/x-archive")
	second := doneOf(t, doneRequest(repo))

	wantReady(t, second)
	wantOutcome(t, branchByName(t, onlyRepository(t, second), "feat/x-archive"), branch.OutcomeRemoved, branch.LayerAncestry, "")
	if branchExists(t, repo, "feat/x-archive") {
		t.Error("la rama de archivado fusionada debía eliminarse")
	}
}

// design D-09: el registro es el plan materializado, de modo que `done` procesa también las ramas que
// guarda aunque el plan ya no las conozca (el kickoff de un cambio archivado deja de estar en
// `changes/<cambio>/` y con él los roles sellados). Una rama de rol registrada se confirma contra la
// del cambio y, con todo eliminado, se borra el registro.
func TestDoneProcessesTheRoleBranchesTheRegistryKnowsWhenTheKickoffIsGone(t *testing.T) {
	repo := embeddedCoreProject(t)
	runGit(t, repo, "branch", "feat/x", "main")
	runGit(t, repo, "branch", "feat/x-core", "feat/x")
	record := registeredRecord(repo,
		branch.RecordBranch{Name: "feat/x", Kind: branch.KindChange},
		branch.RecordBranch{Name: "feat/x-core", Kind: branch.KindRole, RoleID: "core"},
	)
	if err := registry.New().Save(commonDir(repo), record); err != nil {
		t.Fatal(err)
	}

	result := doneOf(t, doneRequest(repo))

	wantReady(t, result)
	repository := onlyRepository(t, result)
	if want := []string{"feat/x-core", "feat/x"}; !slices.Equal(branchNames(repository), want) {
		t.Fatalf("ramas = %v, se esperaban las del registro, las de rol antes: %v", branchNames(repository), want)
	}
	core := branchByName(t, repository, "feat/x-core")
	wantOutcome(t, core, branch.OutcomeRemoved, branch.LayerAncestry, "")
	if core.Kind != branch.KindRole || core.Role != "core" || core.PRTarget != "feat/x" {
		t.Errorf("rama de rol registrada = %+v", core)
	}
	if branchExists(t, repo, "feat/x") || branchExists(t, repo, "feat/x-core") {
		t.Error("ambas ramas debían eliminarse")
	}
	wantNoRegistryFile(t, repo, "x")
}

// REQ-25.10 y REQ-25.16 («solo con `--fetch` PUEDE actualizarse el remoto»): con `--fetch`, `done`
// ejecuta un único `git fetch origin` antes de evaluar y de escribir; si el fetch falla rechaza con
// `fetch_failed` sin eliminar nada y sin escribir.
func TestDoneFetchRunsOneFetchBeforeEvaluatingAndAFailedFetchRemovesNothing(t *testing.T) {
	repo := embeddedCoreProject(t)
	remote := remoteWithMain(t, repo)
	runGit(t, repo, "branch", "feat/x", "main")
	request := doneRequest(repo)
	request.Fetch = true

	t.Run("un remoto inalcanzable da fetch_failed sin eliminar nada", func(t *testing.T) {
		runGit(t, repo, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "no-existe"))
		seen := observe(t)

		result := doneOf(t, request)

		wantDoneStatus(t, result, branch.StatusRefused, branch.CodeFetchFailed)
		if !strings.Contains(result.Continuation, "--fetch") || !strings.Contains(result.Message, "no se ha eliminado") {
			t.Errorf("mensaje %q y continuación %q deben decir que no se eliminó nada y nombrar la orden", result.Message, result.Continuation)
		}
		if !branchExists(t, repo, "feat/x") || len(doneWrites(*seen)) != 0 {
			t.Errorf("un fetch fallido no elimina ni escribe nada: %v", doneWrites(*seen))
		}
	})

	t.Run("con el remoto alcanzable se trae una vez antes de escribir", func(t *testing.T) {
		runGit(t, repo, "remote", "set-url", "origin", remote)
		seen := observe(t)

		result := doneOf(t, request)

		wantReady(t, result)
		var fetches [][]string
		for _, invocation := range *seen {
			if invocation[1] == "fetch" {
				fetches = append(fetches, invocation[1:])
			}
			if invocation[1] == "push" || invocation[1] == "pull" || invocation[1] == "ls-remote" {
				t.Errorf("ninguna operación remota distinta de `fetch origin`: %v", invocation)
			}
		}
		if len(fetches) != 1 || !slices.Equal(fetches[0], []string{"fetch", "origin"}) {
			t.Errorf("se esperaba un único `git fetch origin`: %v", *seen)
		}
		fetchAt := slices.IndexFunc(*seen, func(invocation []string) bool { return invocation[1] == "fetch" })
		writeAt := slices.IndexFunc(*seen, isDoneWrite)
		if fetchAt < 0 || writeAt < 0 || fetchAt > writeAt {
			t.Errorf("el fetch (%d) va antes de toda escritura (%d)", fetchAt, writeAt)
		}
		if branchExists(t, repo, "feat/x") {
			t.Error("la rama fusionada debía eliminarse")
		}
	})
}

// Rechazos previos a cualquier lectura del cambio: un nombre inválido y un directorio de partida
// que no sirve no llevan datos del cambio ni tocan nada.
func TestDoneRefusesAnInvalidChangeNameAndAnUnusableDirectory(t *testing.T) {
	repo := embeddedCoreProject(t)
	runGit(t, repo, "branch", "feat/x", "main")

	invalid := doneOf(t, DoneRequest{CWD: repo, Change: "../x", Lane: branch.LaneSDD})

	wantRefusal(t, invalid, branch.CodeInvalidName, branch.StatusRefused)
	missing := doneOf(t, DoneRequest{CWD: filepath.Join(repo, "no-existe"), Change: "x", Lane: branch.LaneSDD})
	wantRefusal(t, missing, branch.CodeNotAGitRepository, branch.StatusRefused)
	if !branchExists(t, repo, "feat/x") {
		t.Error("los rechazos no eliminan nada")
	}
}

// design D-09 (Windows): un worktree que es el directorio actual de algún proceso no se puede borrar.
// La rama se conserva, el resultado es `git_failed` sin pérdida de lo demás y la continuación pide
// repetir con `--cwd` sobre el checkout principal desde otro directorio.
func TestDoneKeepsTheBranchWhenAProcessHasTheWorktreeAsItsCurrentDirectory(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("solo Windows impide borrar el directorio actual de un proceso")
	}
	repo := embeddedCoreProject(t)
	wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"}))
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	holder := exec.Command("ping", "-n", "120", "127.0.0.1")
	holder.Dir = worktree
	if err := holder.Start(); err != nil {
		t.Fatalf("lanzar el proceso que retiene el worktree: %v", err)
	}
	t.Cleanup(func() {
		_ = holder.Process.Kill()
		_ = holder.Wait()
	})

	result := doneOf(t, doneRequest(repo))

	wantDoneStatus(t, result, branch.StatusError, branch.CodeGitFailed)
	wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeKept, branch.LayerAncestry, branch.CodeGitFailed)
	if !branchExists(t, repo, "feat/x") {
		t.Error("sin el worktree borrado, la rama se conserva")
	}
	if !strings.Contains(result.Continuation, "--cwd") || !strings.Contains(result.Continuation, repo) {
		t.Errorf("la continuación %q debe pedir --cwd sobre el checkout principal %s", result.Continuation, repo)
	}
	if loadRecord(t, repo, "x").Repositories[0].Branches[0].Name != "feat/x" {
		t.Error("la rama conservada sigue en el registro")
	}

	// Con el proceso terminado, repetir la orden completa el desmontaje.
	_ = holder.Process.Kill()
	_ = holder.Wait()
	wantReady(t, doneOf(t, doneRequest(repo)))
	if branchExists(t, repo, "feat/x") {
		t.Error("liberado el worktree, la rama debía eliminarse")
	}
	if listed := runGit(t, repo, "worktree", "list", "--porcelain"); strings.Contains(listed, "feat/x") {
		t.Errorf("liberado el worktree, git no debe seguir conociéndolo:\n%s", listed)
	}
}

// wantSamePlace compara dos rutas que pueden no existir ya (un worktree retirado): por identidad si
// existen y, si no, como lo hace el motor.
func wantSamePlace(t *testing.T, what, got, want string) {
	t.Helper()
	if !filepath.IsAbs(got) || !Default().samePlace(got, want) {
		t.Errorf("%s = %q, se esperaba la ruta absoluta %q", what, got, want)
	}
}

// REQ-25.17 («Si la eliminación del worktree falla, NO DEBE eliminarse la rama») y REQ-25.2: un fallo
// de git al retirar el worktree conserva el worktree y la rama; uno al borrar la rama, ya sin
// worktree, conserva la rama. Ambos son `git_failed`, nombran la operación y resumen el error, y
// repetir `done` completa el desmontaje.
func TestDoneKeepsTheBranchWhenGitFailsToRemoveTheWorktreeOrToDeleteTheBranch(t *testing.T) {
	repo := embeddedCoreProject(t)
	wantReady(t, ensureWith(t, hubWorktree, EnsureRequest{CWD: repo, Change: "x"}))
	worktree := plannedWorktree(t, repo, "x", filepath.Base(repo))
	mergeInto(t, repo, "feat/x")
	flaky := func(writer failingWriter) branch.Result {
		engine := Default()
		writer.real = gitcli.New()
		engine.writer = writer
		return engine.Done(context.Background(), doneRequest(repo))
	}

	t.Run("fallo al retirar el worktree: se conservan el worktree y la rama", func(t *testing.T) {
		state := repositoryState(t, repo)

		result := flaky(failingWriter{failRemove: true})

		wantDoneStatus(t, result, branch.StatusError, branch.CodeGitFailed)
		wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeKept, branch.LayerAncestry, branch.CodeGitFailed)
		for _, want := range []string{"retirar el worktree", "fallo simulado de git al retirar el worktree"} {
			if !strings.Contains(result.Message, want) {
				t.Errorf("el mensaje %q debe contener %q", result.Message, want)
			}
		}
		if got := repositoryState(t, repo); got != state {
			t.Errorf("el fallo no debe tocar worktrees ni ramas:\n%s\n---\n%s", state, got)
		}
		if result.Registry == nil || !result.Registry.Present {
			t.Errorf("registry = %+v, la rama se conserva y el registro también", result.Registry)
		}
	})

	t.Run("fallo al borrar la rama: el worktree ya no está y la rama se conserva", func(t *testing.T) {
		result := flaky(failingWriter{failDelete: true})

		wantDoneStatus(t, result, branch.StatusError, branch.CodeGitFailed)
		wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeKept, branch.LayerAncestry, branch.CodeGitFailed)
		if !strings.Contains(result.Message, "eliminar la rama feat/x") {
			t.Errorf("el mensaje %q debe nombrar la operación", result.Message)
		}
		if !branchExists(t, repo, "feat/x") || strings.Contains(runGit(t, repo, "worktree", "list", "--porcelain"), "feat/x") {
			t.Error("el worktree ya se había retirado y la rama debía conservarse")
		}
		if _, err := os.Stat(worktree); !os.IsNotExist(err) {
			t.Errorf("el worktree debía haberse retirado: %v", err)
		}
	})

	t.Run("repetir la orden completa el desmontaje", func(t *testing.T) {
		wantReady(t, doneOf(t, doneRequest(repo)))
		if branchExists(t, repo, "feat/x") {
			t.Error("la rama debía eliminarse")
		}
		wantNoRegistryFile(t, repo, "x")
	})
}
