package sddstatus

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// G1 de ids-provisionales (REQ-47.8, REQ-21.17): el paso de archivado `axiom sdd
// archive-integrate` reescribe tasks.md, las deltas y verify-report.md; con las puertas ya
// aprobadas, esa reescritura no las devuelve a `pending` ni cambia el siguiente paso.

const (
	gateIntegrateTasksBefore  = "- [x] 1.1 Hacer\n  Cubre: prov:30:1\n"
	gateIntegrateTasksAfter   = "- [x] 1.1 Hacer\n  Cubre: REQ-30.3\n"
	gateIntegrateDeltaBefore  = "## ADDED Requirements\n\n### Requirement: Cobro (prov:30:1)\n\n#### Scenario: Cobro\n"
	gateIntegrateDeltaAfter   = "## ADDED Requirements\n\n### Requirement: Cobro (REQ-30.3)\n\n#### Scenario: Cobro\n"
	gateIntegrateVerifyBefore = "# V\n\n| Requisito | Escenario |\n|---|---|\n| prov:30:1 | Cobro |\n"
	gateIntegrateVerifyAfter  = "# V\n\n| Requisito | Escenario |\n|---|---|\n| REQ-30.3 | Cobro |\n"
)

func gateStatuses(t *testing.T, status Status) map[string]string {
	t.Helper()
	if status.Governance == nil {
		t.Fatal("el cambio checkpointed debe proyectar gobernanza")
	}
	got := map[string]string{}
	for _, gate := range status.Governance.Gates {
		got[string(gate.Key)] = gate.Status
	}
	return got
}

func TestGateApprovalsSurviveTheArchiveIntegrateRewrite(t *testing.T) {
	root := t.TempDir()
	changeName := "archive-integrate-gates"
	changeRoot := seedReadyChange(t, root, changeName, gateIntegrateTasksBefore)
	write(t, filepath.Join(changeRoot, "specs", "pagos", "spec.md"), gateIntegrateDeltaBefore)
	write(t, filepath.Join(changeRoot, "verify-report.md"), gateIntegrateVerifyBefore)
	sealFullstackKickoff(t, changeRoot, changeName, kickoff.ExecutionCheckpointed)
	for _, gate := range []kickoff.GateKey{kickoff.GateSpec, kickoff.GateDesign, kickoff.GateTasks, kickoff.RoleApplyGate("fullstack")} {
		if err := kickoff.AppendGate(changeRoot, kickoff.GateRecord{Gate: gate, Decision: kickoff.DecisionApproved, Reason: "cubre la intencion", Actor: "reviewer"}); err != nil {
			t.Fatalf("AppendGate(%s): %v", gate, err)
		}
	}

	before, err := Resolve(ResolveOptions{CWD: root, ChangeName: changeName})
	if err != nil {
		t.Fatalf("Resolve() antes de reescribir: %v", err)
	}
	approved := gateStatuses(t, before)
	for _, gate := range []kickoff.GateKey{kickoff.GateSpec, kickoff.GateDesign, kickoff.GateTasks, kickoff.RoleApplyGate("fullstack")} {
		if approved[string(gate)] != "approved" {
			t.Fatalf("la puerta %s debe estar aprobada antes de reescribir: %v", gate, approved)
		}
	}

	// Lo que reescribe el verbo.
	write(t, filepath.Join(changeRoot, "tasks.md"), gateIntegrateTasksAfter)
	write(t, filepath.Join(changeRoot, "specs", "pagos", "spec.md"), gateIntegrateDeltaAfter)
	write(t, filepath.Join(changeRoot, "verify-report.md"), gateIntegrateVerifyAfter)

	after, err := Resolve(ResolveOptions{CWD: root, ChangeName: changeName})
	if err != nil {
		t.Fatalf("Resolve() tras reescribir: %v", err)
	}
	if got := gateStatuses(t, after); !reflect.DeepEqual(got, approved) {
		t.Errorf("las puertas cambiaron tras la reescritura:\nantes %v\ndespués %v", approved, got)
	}
	if after.NextRecommended != before.NextRecommended {
		t.Errorf("nextRecommended cambió: %q → %q", before.NextRecommended, after.NextRecommended)
	}
	if !reflect.DeepEqual(after.Dependencies, before.Dependencies) {
		t.Errorf("las dependencias cambiaron:\nantes %+v\ndespués %+v", before.Dependencies, after.Dependencies)
	}
}

func TestGateContinuousChangeDoesNotEvaluateGatesAcrossTheRewrite(t *testing.T) {
	root := t.TempDir()
	changeName := "archive-integrate-continuous"
	changeRoot := seedReadyChange(t, root, changeName, gateIntegrateTasksBefore)
	write(t, filepath.Join(changeRoot, "verify-report.md"), gateIntegrateVerifyBefore)
	sealFullstackKickoff(t, changeRoot, changeName, kickoff.ExecutionContinuous)

	before, err := Resolve(ResolveOptions{CWD: root, ChangeName: changeName})
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(changeRoot, "tasks.md"), gateIntegrateTasksAfter)
	write(t, filepath.Join(changeRoot, "verify-report.md"), gateIntegrateVerifyAfter)
	after, err := Resolve(ResolveOptions{CWD: root, ChangeName: changeName})
	if err != nil {
		t.Fatal(err)
	}
	if after.NextRecommended != before.NextRecommended {
		t.Errorf("nextRecommended cambió en continuous: %q → %q", before.NextRecommended, after.NextRecommended)
	}
	if after.Governance != nil && len(after.Governance.Gates) != 0 {
		t.Errorf("continuous no evalúa puertas: %+v", after.Governance.Gates)
	}
}
