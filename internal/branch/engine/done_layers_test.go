package engine

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/gitcli"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// Pruebas de las capas 2 y 3 de la detección de fusión de `done` y de la sonda de linajes RDD
// (REQ-25.16, REQ-25.17, design D-09, Q1 y A-4), con git real, entorno aislado y sin red.

// commitFile confirma en dir un fichero con contenido propio: un commit vacío tendría el mismo árbol
// que su base y la capa del árbol equivalente lo daría por fusionado.
func commitFile(t *testing.T, dir, name, message string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, name), message+"\n")
	runGit(t, dir, "add", name)
	commit(t, dir, message)
}

var gitVersionNumbers = regexp.MustCompile(`(\d+)\.(\d+)`)

// requireMergeTree omite la prueba con un git anterior a la 2.38, que no admite `git merge-tree
// --write-tree` (Q1, design D-09): la capa del árbol equivalente se omite y no se puede probar con
// git real.
func requireMergeTree(t *testing.T) {
	t.Helper()
	requireGit(t)
	output, err := exec.Command("git", "version").Output()
	if err != nil {
		t.Skipf("no se pudo leer la versión de git: %v", err)
	}
	numbers := gitVersionNumbers.FindStringSubmatch(string(output))
	if numbers == nil {
		t.Skipf("no se entiende la versión de git %q", output)
	}
	major, _ := strconv.Atoi(numbers[1])
	minor, _ := strconv.Atoi(numbers[2])
	if major < 2 || (major == 2 && minor < 38) {
		t.Skipf("la prueba real de la capa del árbol equivalente exige git 2.38 o superior, hay %s", strings.TrimSpace(string(output)))
	}
}

// mergeTreeSupport fija, durante la prueba, si el motor cree que git admite `merge-tree`
// (gitcli.MergeTreeSupportedOverride).
func mergeTreeSupport(t *testing.T, supported bool) {
	t.Helper()
	gitcli.MergeTreeSupportedOverride = &supported
	t.Cleanup(func() { gitcli.MergeTreeSupportedOverride = nil })
}

// squashedBranch crea `feat/x` con un cambio propio, lo integra en main con un squash (sin que la
// rama sea ancestro de main) y deja a main con un commit posterior, como tras el PR de un cambio.
func squashedBranch(t *testing.T, repo, file string) {
	t.Helper()
	runGit(t, repo, "switch", "-q", "-c", "feat/x")
	commitFile(t, repo, file, "cambio de x en "+file)
	runGit(t, repo, "switch", "-q", "main")
	runGit(t, repo, "merge", "--squash", "feat/x")
	commit(t, repo, "squash de "+file)
	commitFile(t, repo, "posterior-a-"+file, "main avanza tras "+file)
}

func invocations(seen [][]string, subcommand string) [][]string {
	var found [][]string
	for _, invocation := range seen {
		if invocation[1] == subcommand {
			found = append(found, invocation)
		}
	}
	return found
}

// REQ-25.16 (escenario «Merge por squash detectado por la capa `equivalent_tree`») y REQ-25.17
// (escenario «Una rama integrada con squash se elimina tras confirmarse por `equivalent_tree`»): con
// `feat/x` integrada en `main` mediante un squash y `main` con commits posteriores, la rama no es
// ancestro de `main` y la capa del árbol equivalente la confirma; se elimina aunque git no la
// considere fusionada, sin operaciones remotas.
func TestDoneConfirmsASquashMergeByEquivalentTreeEvenWhenMainAdvanced(t *testing.T) {
	requireMergeTree(t)
	repo := embeddedCoreProject(t)
	squashedBranch(t, repo, "cambio.txt")
	if exec.Command("git", "-C", repo, "merge-base", "--is-ancestor", "feat/x", "main").Run() == nil {
		t.Fatal("la preparación debía dejar a feat/x sin ser ancestro de main")
	}
	seen := observe(t)

	result := doneOf(t, doneRequest(repo))

	wantReady(t, result)
	described := branchByName(t, onlyRepository(t, result), "feat/x")
	wantOutcome(t, described, branch.OutcomeRemoved, branch.LayerEquivalentTree, "")
	if described.ConfirmedByLabel != "árbol equivalente" || described.TipSha != "" {
		t.Errorf("rama = %+v, se esperaba la etiqueta «árbol equivalente» y ninguna punta (la fusión está confirmada)", described)
	}
	if branchExists(t, repo, "feat/x") {
		t.Error("la rama integrada con squash debía eliminarse aunque git no la considere fusionada")
	}
	if len(result.Warnings) != 0 {
		t.Errorf("con git que admite merge-tree no hay avisos: %v", result.Warnings)
	}
	wantNoRemote(t, *seen)
	if mergeTrees := invocations(*seen, "merge-tree"); len(mergeTrees) == 0 || !slices.Equal(mergeTrees[0][2:], []string{"--write-tree", "--no-messages", "refs/heads/main", "refs/heads/feat/x"}) {
		t.Errorf("la capa 2 ejecuta `git merge-tree --write-tree --no-messages <destino> <rama>`: %v", mergeTrees)
	}
}

// REQ-25.16 (escenarios «Sin soporte de `merge-tree` la capa se omite» y «Nada confirma la fusión»)
// y Q1: sin soporte (la variable de prueba o un git anterior a la 2.38), la capa 2 se omite sin
// error y con un aviso, no se lanza `merge-tree` y, sin evidencia `pr_merged`, `done` exige
// `--force`, que da la rama por `forced` con su punta.
func TestDoneOmitsTheEquivalentTreeLayerWithAWarningWhenGitDoesNotSupportIt(t *testing.T) {
	repo := embeddedCoreProject(t)
	mergeTreeSupport(t, false)
	squashedBranch(t, repo, "cambio.txt")
	seen := observe(t)

	result := doneOf(t, doneRequest(repo))

	wantDoneStatus(t, result, branch.StatusRefused, branch.CodeMergeNotConfirmed)
	wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeKept, "", branch.CodeMergeNotConfirmed)
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "2.38") || !strings.Contains(result.Warnings[0], "merge-tree") {
		t.Errorf("avisos = %v, se esperaba uno que nombre merge-tree y git 2.38", result.Warnings)
	}
	if len(invocations(*seen, "merge-tree")) != 0 {
		t.Errorf("sin soporte no se lanza merge-tree: %v", invocations(*seen, "merge-tree"))
	}
	if !branchExists(t, repo, "feat/x") {
		t.Error("sin fusión confirmada la rama se conserva")
	}

	request := doneRequest(repo)
	request.Force = true
	forced := doneOf(t, request)

	wantReady(t, forced)
	wantOutcome(t, branchByName(t, onlyRepository(t, forced), "feat/x"), branch.OutcomeRemoved, branch.LayerForced, "")
	if branchExists(t, repo, "feat/x") {
		t.Error("con --force la rama se elimina")
	}
}

// recordIntegration anexa a `<changeRoot>/gates.yaml` una decisión de la compuerta `integration` con
// evidencia `pr_merged`.
func recordIntegration(t *testing.T, changeRoot string, decision kickoff.GateDecision, verified bool) {
	t.Helper()
	if err := os.MkdirAll(changeRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	err := kickoff.AppendGate(changeRoot, kickoff.GateRecord{
		Gate: kickoff.GateIntegration, Decision: decision, Reason: "PR fusionado",
		EvidenceKind: kickoff.EvidencePRMerged, EvidenceRef: "abc1234", EvidenceBaseRef: "main", Verified: verified,
		Actor: "prueba", RecordedAt: time.Date(2026, time.October, 26, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
}

// REQ-25.16 (escenario «Evidencia `pr_merged` confirma la fusión»): con una rama integrada con
// squash y sin soporte de `merge-tree`, el `gates.yaml` del cambio (activo o archivado, con
// kickoff.ArchivedChangeRoot) con la compuerta `integration` aprobada con evidencia `pr_merged` y
// `verified: true` confirma la rama por el token `pr_merged`. La evidencia no verificada o de una
// decisión rechazada no confirma.
func TestDoneConfirmsByPRMergedEvidenceWhenTheTreeLayerIsUnavailable(t *testing.T) {
	repo := embeddedCoreProject(t)
	mergeTreeSupport(t, false)
	squashedBranch(t, repo, "uno.txt")
	active := filepath.Join(repo, "openspec", "changes", "x")

	for name, tc := range map[string]struct {
		decision kickoff.GateDecision
		verified bool
	}{
		"evidencia sin verificar": {kickoff.DecisionApproved, false},
		"decisión rechazada":      {kickoff.DecisionRejected, true},
	} {
		t.Run(name+" no confirma", func(t *testing.T) {
			if err := os.RemoveAll(active); err != nil {
				t.Fatal(err)
			}
			recordIntegration(t, active, tc.decision, tc.verified)

			result := doneOf(t, doneRequest(repo))

			wantDoneStatus(t, result, branch.StatusRefused, branch.CodeMergeNotConfirmed)
			if !branchExists(t, repo, "feat/x") {
				t.Error("sin evidencia verificada la rama se conserva")
			}
		})
	}

	t.Run("gates.yaml activo con evidencia verificada confirma por pr_merged", func(t *testing.T) {
		if err := os.RemoveAll(active); err != nil {
			t.Fatal(err)
		}
		recordIntegration(t, active, kickoff.DecisionApproved, true)

		result := doneOf(t, doneRequest(repo))

		wantReady(t, result)
		described := branchByName(t, onlyRepository(t, result), "feat/x")
		wantOutcome(t, described, branch.OutcomeRemoved, branch.LayerPRMerged, "")
		if described.ConfirmedByLabel != "PR fusionado" || branchExists(t, repo, "feat/x") {
			t.Errorf("rama = %+v, se esperaba «PR fusionado» y la rama eliminada", described)
		}
	})

	t.Run("gates.yaml archivado con evidencia verificada confirma por pr_merged", func(t *testing.T) {
		if err := os.RemoveAll(active); err != nil {
			t.Fatal(err)
		}
		squashedBranch(t, repo, "dos.txt")
		recordIntegration(t, filepath.Join(repo, "openspec", "changes", "archive", "2026-10-26-x"), kickoff.DecisionApproved, true)

		result := doneOf(t, doneRequest(repo))

		wantReady(t, result)
		wantOutcome(t, branchByName(t, onlyRepository(t, result), "feat/x"), branch.OutcomeRemoved, branch.LayerPRMerged, "")
	})
}

// REQ-25.16 (escenarios «La evidencia `pr_merged` no confirma la rama de archivado» y «La rama de
// archivado se confirma por `equivalent_tree` tras un squash»): con evidencia `pr_merged` del PR del
// cambio, la rama de archivado sin fusionar se rechaza con `merge_not_confirmed`; integrada con un
// squash, la confirma la capa del árbol equivalente.
func TestDoneNeverConfirmsTheArchiveBranchByPRMergedButDoesByEquivalentTree(t *testing.T) {
	requireMergeTree(t)
	repo := embeddedCoreProject(t)
	recordIntegration(t, filepath.Join(repo, "openspec", "changes", "x"), kickoff.DecisionApproved, true)
	runGit(t, repo, "switch", "-q", "-c", "feat/x-archive")
	commitFile(t, repo, "archivado.md", "documentación de archivado")
	runGit(t, repo, "switch", "-q", "main")

	result := doneOf(t, doneRequest(repo))

	wantDoneStatus(t, result, branch.StatusRefused, branch.CodeMergeNotConfirmed)
	archived := branchByName(t, onlyRepository(t, result), "feat/x-archive")
	wantOutcome(t, archived, branch.OutcomeKept, "", branch.CodeMergeNotConfirmed)
	if archived.Kind != branch.KindArchive || !branchExists(t, repo, "feat/x-archive") {
		t.Errorf("rama = %+v: el PR del cambio no acredita el PR documental de archivado", archived)
	}

	runGit(t, repo, "merge", "--squash", "feat/x-archive")
	commit(t, repo, "squash del archivado")
	commitFile(t, repo, "posterior.txt", "main avanza")
	second := doneOf(t, doneRequest(repo))

	wantReady(t, second)
	wantOutcome(t, branchByName(t, onlyRepository(t, second), "feat/x-archive"), branch.OutcomeRemoved, branch.LayerEquivalentTree, "")
	if branchExists(t, repo, "feat/x-archive") {
		t.Error("la rama de archivado integrada con squash debía eliminarse")
	}
}

// fakeLineages es el doble de la sonda de linajes RDD: devuelve lo que se le da y anota los worktrees
// por los que se le preguntó.
type fakeLineages struct {
	lineages []string
	err      error
	asked    *[]string
}

func (f fakeLineages) OpenLineages(_ context.Context, worktreeRoot string) ([]string, error) {
	if f.asked != nil {
		*f.asked = append(*f.asked, worktreeRoot)
	}
	return f.lineages, f.err
}

// mergedInPlace deja un cambio `in_place` desplegado en `feat/x` y fusionado en `main`.
func mergedInPlace(t *testing.T) string {
	t.Helper()
	repo := embeddedCoreProject(t)
	wantReady(t, ensureInPlace(t, repo, "x"))
	runGit(t, repo, "branch", "-f", "main", "feat/x")
	return repo
}

// REQ-25.17 (escenario «Linaje RDD abierto genera un aviso», A-4): con un linaje RDD abierto en el
// worktree de una rama fusionada, `done` añade un aviso que nombra el linaje y `axiom review
// abandon` y elimina igualmente la rama; un fallo de la sonda tampoco impide la eliminación y se
// avisa; sin linajes no hay aviso; con la fusión sin confirmar no se consulta a la sonda.
func TestDoneWarnsOfAnOpenRDDLineageWithoutPreventingTheRemoval(t *testing.T) {
	withProbe := func(probe branch.LineageProbe, request DoneRequest) branch.Result {
		engine := Default()
		engine.lineages = probe
		return engine.Done(context.Background(), request)
	}

	t.Run("un linaje abierto añade el aviso y la eliminación se realiza", func(t *testing.T) {
		repo := mergedInPlace(t)
		var asked []string

		result := withProbe(fakeLineages{lineages: []string{"linaje-abierto"}, asked: &asked}, doneRequest(repo))

		wantReady(t, result)
		if len(result.Warnings) != 1 {
			t.Fatalf("avisos = %v, se esperaba uno", result.Warnings)
		}
		for _, want := range []string{"linaje-abierto", "axiom review abandon", "feat/x"} {
			if !strings.Contains(result.Warnings[0], want) {
				t.Errorf("el aviso %q debe contener %q", result.Warnings[0], want)
			}
		}
		if len(asked) != 1 || !pathidentity.SameDirectory(asked[0], repo) {
			t.Errorf("la sonda se consulta con el worktree de la rama (aquí, el repositorio in_place %s): %v", repo, asked)
		}
		if branchExists(t, repo, "feat/x") {
			t.Error("el aviso no impide eliminar la rama")
		}
	})

	t.Run("un fallo de la sonda se avisa y no impide eliminar", func(t *testing.T) {
		repo := mergedInPlace(t)

		result := withProbe(fakeLineages{err: errors.New("almacén ilegible")}, doneRequest(repo))

		wantReady(t, result)
		if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "almacén ilegible") || branchExists(t, repo, "feat/x") {
			t.Errorf("avisos = %v, rama existente = %v", result.Warnings, branchExists(t, repo, "feat/x"))
		}
	})

	t.Run("sin linajes abiertos no hay aviso", func(t *testing.T) {
		repo := mergedInPlace(t)

		result := withProbe(fakeLineages{}, doneRequest(repo))

		wantReady(t, result)
		if len(result.Warnings) != 0 {
			t.Errorf("avisos = %v", result.Warnings)
		}
	})

	t.Run("con la fusión sin confirmar no se consulta a la sonda", func(t *testing.T) {
		repo := embeddedCoreProject(t)
		wantReady(t, ensureInPlace(t, repo, "x"))
		commitFile(t, repo, "trabajo.txt", "trabajo sin fusionar")
		var asked []string

		result := withProbe(fakeLineages{lineages: []string{"linaje-abierto"}, asked: &asked}, doneRequest(repo))

		wantDoneStatus(t, result, branch.StatusRefused, branch.CodeMergeNotConfirmed)
		if len(asked) != 0 || len(result.Warnings) != 0 {
			t.Errorf("la sonda se consulta (%v) y avisa (%v) solo de lo que se va a eliminar", asked, result.Warnings)
		}
	})
}
