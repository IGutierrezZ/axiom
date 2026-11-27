package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

const gatesGatewayChange = "inc-99-example"

// writeSpecArtifacts deja un cambio sellado con proposal y spec reales, de modo que `spec` es la
// primera puerta abierta.
func writeSpecArtifacts(t *testing.T, changeRoot string) {
	t.Helper()
	for name, body := range map[string]string{"proposal.md": "# Propuesta\n", "spec.md": "# Spec\n\nContenido real.\n"} {
		if err := os.WriteFile(filepath.Join(changeRoot, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// prov:WEB:1: la parada es la que proyecta el status, con las opciones en el orden de la
// GateQuestion y el digest del artefacto que se decide.
func TestGatesGatewayStop(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, gatesGatewayChange)
	sealForGateTest(t, root, gatesGatewayChange)
	writeSpecArtifacts(t, changeRoot)

	stop, err := NewGatesGateway().Stop(context.Background(), root, gatesGatewayChange)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if stop.NextRecommended != "await-gate" || stop.Question == nil || stop.Question.Gate != "spec" {
		t.Fatalf("parada = %+v", stop)
	}
	if stop.ExecutionStyle != "checkpointed" || stop.Change != gatesGatewayChange {
		t.Fatalf("estilo/cambio = %q/%q", stop.ExecutionStyle, stop.Change)
	}
	if len(stop.Question.Choices) != 3 {
		t.Fatalf("opciones = %d, se esperaban 3", len(stop.Question.Choices))
	}
	if !sameRoot(stop.RecordRoot, root) || !sameRoot(stop.WorkspaceRoot, root) {
		t.Fatalf("raíces = %q / %q, se esperaba %q", stop.RecordRoot, stop.WorkspaceRoot, root)
	}
	sealed, err := kickoff.Load(changeRoot)
	if err != nil {
		t.Fatal(err)
	}
	want, err := gateArtifactDigest(changeRoot, kickoff.GateSpec, sealed)
	if err != nil || want == "" || stop.Digest != want {
		t.Fatalf("digest = %q, se esperaba %q (%v)", stop.Digest, want, err)
	}
	if stop.LastDecision != nil {
		t.Fatalf("sin registros no hay última decisión: %+v", stop.LastDecision)
	}
}

// D9: tras aprobar, la parada desaparece pero el último registro y el estilo siguen devolviéndose
// para reconstruir «aprobada sin publicar».
func TestGatesGatewayStopAfterApprovalKeepsLastDecision(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, gatesGatewayChange)
	sealForGateTest(t, root, gatesGatewayChange)
	writeSpecArtifacts(t, changeRoot)
	gw := NewGatesGateway()

	before, err := gw.Stop(context.Background(), root, gatesGatewayChange)
	if err != nil || before.Question == nil {
		t.Fatalf("Stop previo: %+v, %v", before, err)
	}
	if _, err := gw.Record(context.Background(), GateRecordInput{
		CWD: root, Change: gatesGatewayChange, Gate: kickoff.GateSpec, Decision: kickoff.DecisionApproved,
		Reason: "ok", Actor: "ignorado", ExpectedDigest: before.Digest,
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	after, err := gw.Stop(context.Background(), root, gatesGatewayChange)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if after.Question != nil || after.NextRecommended == "await-gate" {
		t.Fatalf("tras aprobar no hay parada de spec: %+v", after)
	}
	if after.LastDecision == nil || after.LastDecision.Gate != kickoff.GateSpec || after.LastDecision.Decision != kickoff.DecisionApproved {
		t.Fatalf("lastDecision = %+v", after.LastDecision)
	}
	if after.LastDecision.Actor == "ignorado" {
		t.Fatal("el actor nunca viaja: lo resuelve el núcleo")
	}
	if after.ExecutionStyle != "checkpointed" {
		t.Fatalf("executionStyle = %q", after.ExecutionStyle)
	}
}

// continuous no proyecta gobernanza de puertas: sin parada.
func TestGatesGatewayStopContinuousHasNoStop(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, gatesGatewayChange)
	if err := RunSDDKickoff([]string{"seal", "--cwd", root, "--change", gatesGatewayChange, "--execution-style", "continuous"}, &discardWriter{}); err != nil {
		t.Fatalf("seal continuous: %v", err)
	}
	writeSpecArtifacts(t, changeRoot)

	stop, err := NewGatesGateway().Stop(context.Background(), root, gatesGatewayChange)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if stop.Question != nil || stop.NextRecommended == "await-gate" || stop.ExecutionStyle != "continuous" {
		t.Fatalf("continuous no tiene parada: %+v", stop)
	}
}

// Sin worktree de archivado registrado, `integration` se registra en la raíz del status (la guarda A
// del núcleo niega entonces con la continuación de `axiom branch ensure --archive`).
func TestGatesGatewayArchiveRecordRootFallsBackToStatusRoot(t *testing.T) {
	root, _ := newGovernanceWorkspace(t, gatesGatewayChange)
	if got := archiveRecordRoot(context.Background(), root, gatesGatewayChange); !sameRoot(got, root) {
		t.Fatalf("archiveRecordRoot = %q, se esperaba la raíz del status %q", got, root)
	}
}
