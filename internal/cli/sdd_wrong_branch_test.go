package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
)

// Pruebas de `axiom sdd status` y `axiom sdd continue` con la guarda B (REQ-25.42, REQ-25.39, design
// D-11 y D-13). Entran por RunSDDStatus y RunSDDContinue, el punto de entrada de la CLI, con
// repositorios montados solo con ficheros (`.git/HEAD`): la guarda lee ficheros y no lanza procesos.

const wrongBranchChange = "x"

func wrongBranchSwitch(t *testing.T, root, branchName string) {
	t.Helper()
	writeSDDStatusFile(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/"+branchName+"\n")
}

// wrongBranchProject es un monorepo embebido con el cambio x sellado con instantánea de política git
// y el HEAD en la rama dada. Con done, el cambio está terminado y verificado (fase archive).
func wrongBranchProject(t *testing.T, head, tasks string, done bool) (root, changeRoot string) {
	t.Helper()
	root = t.TempDir()
	wrongBranchSwitch(t, root, head)
	return wrongBranchSealed(t, root, tasks, done)
}

// wrongBranchRealGitProject es wrongBranchProject con un repositorio git real (entorno aislado): lo
// necesita el flujo que prepara el marcador de instancia, cuyo libro de concesiones consulta git.
func wrongBranchRealGitProject(t *testing.T, head, tasks string) (root, changeRoot string) {
	t.Helper()
	isolateIntegrateGit(t)
	root = t.TempDir()
	integrateInitRepo(t, root)
	wrongBranchSwitch(t, root, head)
	return wrongBranchSealed(t, root, tasks, false)
}

func wrongBranchSealed(t *testing.T, root, tasks string, done bool) (string, string) {
	t.Helper()
	changeRoot := seedSDDStatusReadyChange(t, root, wrongBranchChange, tasks)
	if done {
		writeSDDStatusFile(t, filepath.Join(changeRoot, "verify-report.md"), "# Verificación\n\nVeredicto favorable.\n")
	}
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: wrongBranchChange, SealedBy: "prueba",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		Git: &kickoff.GitSnapshot{
			IntegrationBranch: "main", BranchPrefix: "feat", ChangeBranch: "feat/x",
			Repositories: []kickoff.GitSnapshotRepo{{
				Path: ".", Name: filepath.Base(root), Role: "change_branch", IntegrationBranch: "main", Base: "main",
				Branches: []kickoff.GitSnapshotBranch{{Name: "feat/x", Kind: "change"}, {Name: "feat/x-archive", Kind: "archive"}},
			}},
		},
	})
	if err != nil || !ok {
		t.Fatalf("sellar %s: %v (publicado: %v)", wrongBranchChange, err, ok)
	}
	return root, changeRoot
}

func wrongBranchStatus(t *testing.T, run func([]string, io.Writer) error, root string) sddstatus.Status {
	t.Helper()
	var stdout bytes.Buffer
	if err := run([]string{wrongBranchChange, "--cwd", root, "--json"}, &stdout); err != nil {
		t.Fatalf("la orden falló: %v", err)
	}
	var status sddstatus.Status
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatalf("decodificar el status: %v\n%s", err, stdout.String())
	}
	return status
}

func wrongBranchReason(status sddstatus.Status) string {
	for _, reason := range status.BlockedReasons {
		if strings.HasPrefix(reason, "blocked(wrong_branch)") {
			return reason
		}
	}
	return ""
}

// REQ-25.42, arnés de S9-c2: `axiom sdd status x --json` desde main bloquea, desde la rama del
// cambio no bloquea y, en la fase archive, solo la rama de archivado deja pasar.
func TestSDDWrongBranchStatusHarness(t *testing.T) {
	root, _ := wrongBranchProject(t, "main", "- [ ] 1.1 Trabajo de ejemplo\n", false)

	status := wrongBranchStatus(t, RunSDDStatus, root)
	reason := wrongBranchReason(status)
	if reason == "" || status.BlockedReasons[0] != reason {
		t.Fatalf("blockedReasons = %v, se esperaba blocked(wrong_branch) en primera posición", status.BlockedReasons)
	}
	for _, want := range []string{"«main»", "«feat/x»", "axiom branch ensure --cwd ", `--change x`} {
		if !strings.Contains(reason, want) {
			t.Errorf("el motivo %q debe contener %q", reason, want)
		}
	}
	if status.ApplyState != sddstatus.ApplyBlocked || status.NextRecommended != "resolve-blockers" {
		t.Errorf("applyState %q y nextRecommended %q, se esperaba blocked y resolve-blockers", status.ApplyState, status.NextRecommended)
	}
	t.Logf("sdd status x --json desde main: %s", reason)

	wrongBranchSwitch(t, root, "feat/x")
	if got := wrongBranchStatus(t, RunSDDStatus, root); wrongBranchReason(got) != "" || got.NextRecommended != "apply" {
		t.Errorf("en la rama del cambio: motivo %q y nextRecommended %q, se esperaba sin bloqueo y apply", wrongBranchReason(got), got.NextRecommended)
	}

	archiveRoot, _ := wrongBranchProject(t, "feat/x", "- [x] 1.1 Trabajo de ejemplo\n", true)
	archiveStatus := wrongBranchStatus(t, RunSDDStatus, archiveRoot)
	archiveReason := wrongBranchReason(archiveStatus)
	if archiveReason == "" || !strings.Contains(archiveReason, "«feat/x-archive»") || !strings.Contains(archiveReason, `--change x --archive`) {
		t.Fatalf("en la fase archive desde feat/x: %q, se esperaba blocked(wrong_branch) con feat/x-archive y --archive", archiveReason)
	}
	t.Logf("sdd status x --json en la fase archive desde feat/x: %s", archiveReason)
	wrongBranchSwitch(t, archiveRoot, "feat/x-archive")
	if got := wrongBranchStatus(t, RunSDDStatus, archiveRoot); wrongBranchReason(got) != "" || got.NextRecommended != "archive" {
		t.Errorf("desde feat/x-archive: motivo %q y nextRecommended %q, se esperaba sin bloqueo y archive", wrongBranchReason(got), got.NextRecommended)
	}
}

// REQ-25.42 y D-11, escenario «`axiom sdd continue` refleja el bloqueo»: con blocked(wrong_branch) no
// prepara el marcador de instancia ni emite sobre de consentimiento y no autoriza ninguna fase; en la
// rama del cambio el mismo plan sí prepara el marcador y pide la concesión (S-9).
func TestSDDWrongBranchContinueReflectsTheBlockWithoutPreparingTheMarker(t *testing.T) {
	root, changeRoot := wrongBranchRealGitProject(t, "main", "- [ ] 1.1 Update `../source/main.go`\n")
	writeSDDStatusFile(t, filepath.Join(root, "..", "source", "main.go"), "package source\n")
	marker := filepath.Join(changeRoot, ".axiom-instance")

	status := wrongBranchStatus(t, RunSDDContinue, root)

	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("con la rama equivocada continue no prepara el marcador: %v", err)
	}
	if wrongBranchReason(status) == "" || status.Consent != nil {
		t.Errorf("continue debe reflejar blocked(wrong_branch) sin sobre de consentimiento: %v, consent %#v", status.BlockedReasons, status.Consent)
	}
	if status.NextRecommended != "resolve-blockers" || status.Dependencies.Apply == sddstatus.DependencyReady || status.Dependencies.Archive == sddstatus.DependencyReady {
		t.Errorf("continue no autoriza ninguna fase: nextRecommended %q, dependencies %+v", status.NextRecommended, status.Dependencies)
	}

	wrongBranchSwitch(t, root, "feat/x")
	status = wrongBranchStatus(t, RunSDDContinue, root)

	if data, err := os.ReadFile(marker); err != nil || !strings.HasPrefix(strings.TrimSpace(string(data)), "sdd-") {
		t.Fatalf("en la rama del cambio continue prepara el marcador: %q, %v", data, err)
	}
	if wrongBranchReason(status) != "" || status.Consent == nil || status.ApplyState != sddstatus.ApplyBlocked {
		t.Errorf("tras corregir la rama se esperaba blocked(edit_authority_missing) con consentimiento: %v, consent %#v", status.BlockedReasons, status.Consent)
	}
}
