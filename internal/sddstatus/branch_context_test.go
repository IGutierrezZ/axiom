package sddstatus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Pruebas de la guarda B de `axiom sdd status` (REQ-25.42, REQ-25.39, design D-13 y S-9): el status
// proyecta `blocked(wrong_branch)` cuando el veredicto de rama del cambio es desfavorable. Los
// repositorios de estas pruebas se montan solo con ficheros (`.git/HEAD`): la guarda lee ficheros y
// no lanza ningún proceso de git, y las pruebas lo comprueban con el observador de gitcommon.

const (
	branchChange     = "x"
	branchPlan       = "- [ ] 1.1 Trabajo de ejemplo\n"
	branchDonePlan   = "- [x] 1.1 Trabajo de ejemplo\n"
	branchVerifyDone = "# Verificación\n\nVeredicto favorable.\n"
)

// useBranchGuard reactiva la guarda B, que el TestMain de este paquete desactiva, y cuenta las
// consultas al motor. Cada prueba de la guarda la llama antes de resolver el status.
func useBranchGuard(t *testing.T) *int {
	t.Helper()
	queried := new(int)
	branchEngine = func() *engine.Engine { *queried++; return engine.Default() }
	t.Cleanup(func() { branchEngine = func() *engine.Engine { return nil } })
	return queried
}

// switchTo mueve el HEAD del repositorio cuya raíz es root a la rama dada, como lo haría un
// `git switch`, sin lanzar git.
func switchTo(t *testing.T, root, branchName string) {
	t.Helper()
	workspacetest.WriteFile(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/"+branchName+"\n")
}

// monorepoSnapshot es la instantánea de política git que `kickoff seal` escribe en un monorepo
// embebido: integración `main`, prefijo `feat`, la rama del cambio y la de archivado.
func monorepoSnapshot(repoName string) *kickoff.GitSnapshot {
	return &kickoff.GitSnapshot{
		IntegrationBranch: "main",
		BranchPrefix:      "feat",
		ChangeBranch:      "feat/x",
		Repositories: []kickoff.GitSnapshotRepo{{
			Path:              ".",
			Name:              repoName,
			Role:              "change_branch",
			IntegrationBranch: "main",
			Base:              "main",
			Branches: []kickoff.GitSnapshotBranch{
				{Name: "feat/x", Kind: "change"},
				{Name: "feat/x-archive", Kind: "archive"},
			},
		}},
	}
}

// sealWithSnapshot sella un kickoff continuo de un solo rol con la instantánea dada (nil: un cambio
// preexistente, retro-sellado sin instantánea, que queda exento).
func sealWithSnapshot(t *testing.T, changeRoot string, snapshot *kickoff.GitSnapshot) {
	t.Helper()
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: branchChange, SealedBy: "prueba",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		Git:       snapshot,
	})
	if err != nil || !ok {
		t.Fatalf("sellar %s: %v (publicado: %v)", branchChange, err, ok)
	}
}

// branchMonorepo monta un monorepo embebido sellado con instantánea, con el cambio x listo para
// aplicar (o terminado y verificado si archive) y el HEAD en la rama dada.
func branchMonorepo(t *testing.T, head string, archive bool) (root string) {
	t.Helper()
	useBranchGuard(t)
	ws := workspacetest.Embedded(t, ".")
	root = ws.Master
	switchTo(t, root, head)
	plan := branchPlan
	if archive {
		plan = branchDonePlan
	}
	changeRoot := seedReadyChange(t, root, branchChange, plan)
	if archive {
		write(t, filepath.Join(changeRoot, "verify-report.md"), branchVerifyDone)
	}
	sealWithSnapshot(t, changeRoot, monorepoSnapshot(filepath.Base(root)))
	return root
}

// wrongBranchReasons devuelve los motivos de bloqueo que empiezan por el token de la guarda.
func wrongBranchReasons(status Status) []string {
	var reasons []string
	for _, reason := range status.BlockedReasons {
		if strings.HasPrefix(reason, wrongBranchPrefix) {
			reasons = append(reasons, reason)
		}
	}
	return reasons
}

func wantNotWrongBranch(t *testing.T, status Status) {
	t.Helper()
	if reasons := wrongBranchReasons(status); len(reasons) != 0 || BlockedByWrongBranch(status) {
		t.Fatalf("no se esperaba blocked(wrong_branch), motivos: %v", reasons)
	}
}

// wantWrongBranchStatus comprueba el bloqueo tipado: primero en blockedReasons, con el repositorio,
// la rama actual, la esperada y la continuación exacta, y con `resolve-blockers` como ruta.
func wantWrongBranchStatus(t *testing.T, status Status, repo, current, expected, continuation string) {
	t.Helper()
	if len(status.BlockedReasons) == 0 || !strings.HasPrefix(status.BlockedReasons[0], wrongBranchPrefix) {
		t.Fatalf("blockedReasons = %v, se esperaba blocked(wrong_branch) en primera posición", status.BlockedReasons)
	}
	reason := status.BlockedReasons[0]
	for _, want := range []string{repo, "«" + current + "»", "«" + expected + "»", continuation} {
		if !strings.Contains(reason, want) {
			t.Errorf("el motivo %q debe contener %q", reason, want)
		}
	}
	if !BlockedByWrongBranch(status) {
		t.Error("BlockedByWrongBranch debe reconocer el bloqueo")
	}
	if status.ApplyState != ApplyBlocked || status.NextRecommended != "resolve-blockers" {
		t.Errorf("applyState %q y nextRecommended %q, se esperaba blocked y resolve-blockers", status.ApplyState, status.NextRecommended)
	}
	if status.Dependencies.Apply == DependencyReady || status.Dependencies.Archive == DependencyReady {
		t.Errorf("con la rama equivocada no se autoriza ninguna fase: %+v", status.Dependencies)
	}
	if status.Consent != nil || len(status.consentPreparationRoots) != 0 {
		t.Errorf("la continuación es una orden y no una concesión: sin sobre de consentimiento (%+v)", status.Consent)
	}
}

// REQ-25.42, escenarios «Status en la rama de integración de un monorepo» y «Status en la rama del
// cambio no bloquea»: la guarda consulta el motor por la variable de prueba y no lanza procesos.
func TestResolveBranchContextBlocksTheIntegrationBranchOfAMonorepo(t *testing.T) {
	root := branchMonorepo(t, "main", false)
	queried := useBranchGuard(t)
	var processes [][]string
	gitcommon.Observe = func(dir string, args []string) { processes = append(processes, append([]string{dir}, args...)) }
	t.Cleanup(func() { gitcommon.Observe = nil })

	status := resolveFrom(t, root, branchChange)

	if *queried == 0 {
		t.Error("la guarda debe consultar el motor por branchEngine")
	}
	if len(processes) != 0 {
		t.Errorf("la guarda es de solo lectura y no lanza procesos de git: %v", processes)
	}
	wantWrongBranchStatus(t, status, filepath.Base(root), "main", "feat/x", "axiom branch ensure --cwd ")
	if !strings.Contains(status.BlockedReasons[0], `" --change x`) {
		t.Errorf("la continuación debe nombrar el cambio con --change x: %q", status.BlockedReasons[0])
	}

	switchTo(t, root, "feat/x")
	wantNotWrongBranch(t, resolveFrom(t, root, branchChange))
	if got := resolveFrom(t, root, branchChange); got.ApplyState != ApplyReady || got.NextRecommended != "apply" {
		t.Errorf("en la rama del cambio applyState %q y nextRecommended %q, se esperaba ready y apply", got.ApplyState, got.NextRecommended)
	}
}

// REQ-25.42, escenario «Se evalúa en toda fase en un monorepo»: también con la planificación
// incompleta.
func TestResolveBranchContextIsEvaluatedInEveryPhaseOfAMonorepo(t *testing.T) {
	useBranchGuard(t)
	ws := workspacetest.Embedded(t, ".")
	switchTo(t, ws.Master, "main")
	changeRoot := filepath.Join(ws.Home, "changes", branchChange)
	write(t, filepath.Join(changeRoot, "proposal.md"), "# Propuesta\n")
	sealWithSnapshot(t, changeRoot, monorepoSnapshot(filepath.Base(ws.Master)))

	status := resolveFrom(t, ws.Master, branchChange)

	wantWrongBranchStatus(t, status, filepath.Base(ws.Master), "main", "feat/x", "axiom branch ensure --cwd ")
	// La ruta de planificación queda tapada solo mientras la rama es la equivocada.
	switchTo(t, ws.Master, "feat/x")
	if got := resolveFrom(t, ws.Master, branchChange); got.NextRecommended != "spec" {
		t.Errorf("en la rama del cambio nextRecommended = %q, se esperaba spec", got.NextRecommended)
	}
}

// REQ-25.42 y REQ-25.39, escenario «En la fase archive solo la rama de archivado es correcta»:
// feat/x, main y feat/x-core bloquean con la rama esperada feat/x-archive y la continuación
// `--archive`; en feat/x-archive no bloquea.
func TestResolveBranchContextOnlyTheArchiveBranchIsCorrectInTheArchivePhase(t *testing.T) {
	for _, current := range []string{"feat/x", "main", "feat/x-core"} {
		t.Run(current, func(t *testing.T) {
			root := branchMonorepo(t, current, true)

			status := resolveFrom(t, root, branchChange)

			wantWrongBranchStatus(t, status, filepath.Base(root), current, "feat/x-archive", "--change x --archive")
		})
	}

	root := branchMonorepo(t, "feat/x-archive", true)
	status := resolveFrom(t, root, branchChange)
	wantNotWrongBranch(t, status)
	if status.NextRecommended != "archive" {
		t.Errorf("en la rama de archivado nextRecommended = %q, se esperaba archive", status.NextRecommended)
	}
}

// REQ-25.42 y D-13: sin `verify-report` hecho no es la fase archive y rige la rama del cambio.
func TestResolveBranchContextIsNotTheArchivePhaseWithoutAVerifyReport(t *testing.T) {
	root := branchMonorepo(t, "feat/x", true)
	if err := os.Remove(filepath.Join(root, "openspec", "changes", branchChange, "verify-report.md")); err != nil {
		t.Fatal(err)
	}

	wantNotWrongBranch(t, resolveFrom(t, root, branchChange))

	switchTo(t, root, "feat/x-archive")
	status := resolveFrom(t, root, branchChange)
	wantWrongBranchStatus(t, status, filepath.Base(root), "feat/x-archive", "feat/x", "--change x")
	if strings.Contains(status.BlockedReasons[0], "--archive") {
		t.Errorf("fuera de la fase archive la continuación no lleva --archive: %q", status.BlockedReasons[0])
	}
}

// REQ-25.42, escenario «Los cambios activos en la rama de integración no bloquean»: otro cambio y
// presente en el planning home no altera el veredicto de x.
func TestResolveBranchContextIgnoresOtherActiveChanges(t *testing.T) {
	root := branchMonorepo(t, "feat/x", false)
	seedReadyChange(t, root, "y", branchDonePlan)

	wantNotWrongBranch(t, resolveFrom(t, root, branchChange))
}

// REQ-25.42, escenario «Un cambio exento no se bloquea»: un kickoff.yaml sin instantánea.
func TestResolveBranchContextDoesNotBlockAnExemptChange(t *testing.T) {
	useBranchGuard(t)
	ws := workspacetest.Embedded(t, ".")
	switchTo(t, ws.Master, "main")
	changeRoot := seedReadyChange(t, ws.Master, branchChange, branchPlan)
	sealWithSnapshot(t, changeRoot, nil)

	wantNotWrongBranch(t, resolveFrom(t, ws.Master, branchChange))
}

// REQ-25.42, escenario «Sin git no se bloquea»: ningún repositorio git; el status avisa en `notes`
// y no en `blockedReasons`.
func TestResolveBranchContextWithoutGitDoesNotBlockAndLeavesANote(t *testing.T) {
	useBranchGuard(t)
	ws := workspacetest.Embedded(t, ".")
	changeRoot := seedReadyChange(t, ws.Master, branchChange, branchPlan)
	sealWithSnapshot(t, changeRoot, monorepoSnapshot(filepath.Base(ws.Master)))

	status := resolveFrom(t, ws.Master, branchChange)

	wantNotWrongBranch(t, status)
	if status.ApplyState != ApplyReady {
		t.Errorf("applyState = %q, sin git se esperaba ready", status.ApplyState)
	}
	found := false
	for _, note := range status.Notes {
		found = found || (strings.HasPrefix(note, "note(branch_unchecked)") && strings.Contains(note, "git"))
	}
	if !found {
		t.Errorf("notes = %v, se esperaba el aviso note(branch_unchecked) que dice que no se pudo comprobar", status.Notes)
	}
	for _, reason := range status.BlockedReasons {
		if strings.Contains(reason, "branch_unchecked") {
			t.Errorf("el aviso no es un bloqueo: %q", reason)
		}
	}
}

// REQ-25.42, escenario «Un repositorio de código de un multirepo solo se evalúa en apply»: con
// `repo-backend` en main no bloquea en la fase spec; en apply con una unidad de trabajo que lo toca
// bloquea nombrando `repo-backend`.
func TestResolveBranchContextEvaluatesACodeRepositoryOnlyWhenApplying(t *testing.T) {
	useBranchGuard(t)
	ws := workspacetest.Canonical(t)
	switchTo(t, ws.Specs, "main")
	switchTo(t, ws.Backend, "main")
	workspacetest.WriteFile(t, filepath.Join(ws.Backend, "src", "a.go"), "package src\n")
	plan := "- [ ] 1.1 Editar `../repo-backend/src/a.go`\n"

	// Fase spec: solo hay propuesta y el plan; el repositorio de código no se evalúa.
	specsChange := filepath.Join(ws.Home, "changes", branchChange)
	write(t, filepath.Join(specsChange, "proposal.md"), "# Propuesta\n")
	write(t, filepath.Join(specsChange, "tasks.md"), plan)
	wantNotWrongBranch(t, resolveFrom(t, ws.Specs, branchChange))

	// Fase apply: la unidad de trabajo actual toca repo-backend, que sigue en main.
	seedReadyChange(t, ws.Specs, branchChange, plan)
	status := resolveFrom(t, ws.Specs, branchChange)
	wantWrongBranchStatus(t, status, "repo-backend", "main", "feat/x", "axiom branch ensure --cwd ")

	// S-9: con repo-backend en la rama del cambio desaparece wrong_branch y aparece
	// edit_authority_missing, que antes no se evaluaba.
	switchTo(t, ws.Backend, "feat/x")
	status = resolveFrom(t, ws.Specs, branchChange)
	wantNotWrongBranch(t, status)
	if status.ApplyState != ApplyBlocked || !containsReason(status.BlockedReasons, "blocked(edit_authority_missing)") {
		t.Fatalf("applyState %q y motivos %v, tras corregir la rama se esperaba blocked(edit_authority_missing)", status.ApplyState, status.BlockedReasons)
	}
}

// REQ-25.42 y REQ-25.39, escenario «En la fase archive del repositorio de specs de un multirepo se
// exige la integración»: feat/x-core bloquea con la esperada `main` y sin `--archive`.
func TestResolveBranchContextRequiresTheIntegrationBranchOfTheSpecsRepositoryInTheArchivePhase(t *testing.T) {
	useBranchGuard(t)
	ws := workspacetest.Canonical(t)
	switchTo(t, ws.Backend, "main")
	changeRoot := seedReadyChange(t, ws.Specs, branchChange, branchDonePlan)
	write(t, filepath.Join(changeRoot, "verify-report.md"), branchVerifyDone)

	switchTo(t, ws.Specs, "feat/x-core")
	status := resolveFrom(t, ws.Specs, branchChange)
	wantWrongBranchStatus(t, status, "repo-specs", "feat/x-core", "main", "axiom branch ensure --cwd ")
	if strings.Contains(status.BlockedReasons[0], "--archive") {
		t.Errorf("el repositorio de specs de un multirepo no tiene rama de archivado: %q", status.BlockedReasons[0])
	}

	switchTo(t, ws.Specs, "main")
	wantNotWrongBranch(t, resolveFrom(t, ws.Specs, branchChange))
}

// REQ-25.42, S-9: con la rama equivocada no se evalúa la autoridad de edición (ni hay sobre de
// consentimiento ni marcador pendiente); tras corregir la rama aparece `edit_authority_missing`.
func TestResolveBranchContextPrecedesTheEditAuthorityBlock(t *testing.T) {
	root := branchMonorepo(t, "main", false)
	outside := t.TempDir()
	switchTo(t, outside, "main")
	workspacetest.WriteFile(t, filepath.Join(outside, "a.go"), "package a\n")
	plan := "- [ ] 1.1 Editar `" + filepath.Join(outside, "a.go") + "`\n"
	write(t, filepath.Join(root, "openspec", "changes", branchChange, "tasks.md"), plan)

	status := resolveFrom(t, root, branchChange)

	wantWrongBranchStatus(t, status, filepath.Base(root), "main", "feat/x", "axiom branch ensure --cwd ")
	if containsReason(status.BlockedReasons, "blocked(edit_authority_missing)") {
		t.Errorf("con la rama equivocada no se evalúa la autoridad de edición: %v", status.BlockedReasons)
	}
	if ChangeInstanceMarkerPending(status) {
		t.Error("con la rama equivocada no hay marcador de instancia que preparar")
	}

	switchTo(t, root, "feat/x")
	status = resolveFrom(t, root, branchChange)
	wantNotWrongBranch(t, status)
	if !containsReason(status.BlockedReasons, "blocked(edit_authority_missing)") {
		t.Errorf("tras corregir la rama se esperaba blocked(edit_authority_missing): %v", status.BlockedReasons)
	}
}

// REQ-25.42 y D-13 (contrato v2): el bloqueo solo añade un texto en blockedReasons; las claves del
// status proyectado son las mismas con y sin bloqueo.
func TestResolveBranchContextOnlyGrowsTheStatusV2Contract(t *testing.T) {
	root := branchMonorepo(t, "feat/x", false)
	clean := statusJSONKeys(t, resolveFrom(t, root, branchChange))

	switchTo(t, root, "main")
	blocked := statusJSONKeys(t, resolveFrom(t, root, branchChange))

	if strings.Join(clean, ",") != strings.Join(blocked, ",") {
		t.Errorf("las claves del status cambian con el bloqueo:\nsin bloqueo: %v\ncon bloqueo: %v", clean, blocked)
	}
}

func containsReason(reasons []string, fragment string) bool {
	for _, reason := range reasons {
		if strings.Contains(reason, fragment) {
			return true
		}
	}
	return false
}
