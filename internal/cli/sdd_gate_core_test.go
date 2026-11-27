package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

var recordedAtLine = regexp.MustCompile(`(?m)^\s*recorded_at:.*$`)

func readLedgerWithoutTimestamps(t *testing.T, changeRoot string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(changeRoot, "gates.yaml"))
	if err != nil {
		t.Fatalf("gates.yaml: %v", err)
	}
	return recordedAtLine.ReplaceAllString(string(data), "recorded_at: X")
}

// prov:GAT:1: la CLI y el núcleo escriben el mismo registro sobre dos copias del mismo cambio.
func TestRecordGateMatchesCLI(t *testing.T) {
	cliRoot, cliChange := newGovernanceWorkspace(t, "inc-99-example")
	sealForGateTest(t, cliRoot, "inc-99-example")
	coreRoot, coreChange := newGovernanceWorkspace(t, "inc-99-example")
	sealForGateTest(t, coreRoot, "inc-99-example")

	var out bytes.Buffer
	if err := RunSDDGate([]string{"record", "--cwd", cliRoot, "--change", "inc-99-example", "--gate", "spec", "--decision", "approved", "--reason", "ok", "--actor", "maintainer"}, &out); err != nil {
		t.Fatalf("RunSDDGate: %v", err)
	}
	outcome, err := RecordGate(context.Background(), GateRecordInput{
		CWD: coreRoot, Change: "inc-99-example", Gate: kickoff.GateSpec, Decision: kickoff.DecisionApproved, Reason: "ok", Actor: "maintainer",
	})
	if err != nil {
		t.Fatalf("RecordGate: %v", err)
	}

	if got, want := readLedgerWithoutTimestamps(t, coreChange), readLedgerWithoutTimestamps(t, cliChange); got != want {
		t.Fatalf("gates.yaml difiere:\nnúcleo:\n%s\nCLI:\n%s", got, want)
	}
	if outcome.Record.Gate != kickoff.GateSpec || outcome.Record.Actor != "maintainer" || outcome.Change != "inc-99-example" {
		t.Fatalf("resultado = %+v", outcome)
	}
}

// D5: aprobar spec despacha exactamente un evento, desde RecordGate.
func TestRecordGateEmitsIntegrationEventOnce(t *testing.T) {
	root, _, fake := integrationsWorkspace(t, fakeBlock)
	captureIntegrationWarnings(t)

	if _, err := RecordGate(context.Background(), GateRecordInput{CWD: root, Change: "x", Gate: kickoff.GateSpec, Decision: kickoff.DecisionApproved, Actor: "maintainer"}); err != nil {
		t.Fatalf("RecordGate: %v", err)
	}
	if keys := fake.DeliveredKeys(); len(keys) != 1 || !strings.HasPrefix(keys[0], "gate.spec.approved/x/") {
		t.Fatalf("claves entregadas = %v, se esperaba exactamente un gate.spec.approved", keys)
	}

	// Rechazar no emite.
	if _, err := RecordGate(context.Background(), GateRecordInput{CWD: root, Change: "x", Gate: kickoff.GateDesign, Decision: kickoff.DecisionRejected, Reason: "falta", Actor: "maintainer"}); err != nil {
		t.Fatalf("RecordGate rechazo: %v", err)
	}
	if fake.Calls() != 1 {
		t.Fatalf("un rechazo no despacha: llamadas = %d", fake.Calls())
	}
}

// RF-BRA-20 y G-10: el núcleo devuelve el mismo *branch.RefusalError que la CLI, sin escribir.
func TestRecordGateWrongBranchRefusal(t *testing.T) {
	ws := guardProject(t)
	before := consentCLISnapshot(t, guardChangeRoot(ws))

	_, err := RecordGate(context.Background(), GateRecordInput{CWD: ws.Master, Change: sealChange, Gate: kickoff.GateSpec, Decision: kickoff.DecisionApproved, Reason: "ok"})
	wantWrongBranch(t, err, "main", "feat/x", ws.Master)
	_, cliErr := gateRecord(t, ws, "--gate", "spec", "--decision", "approved", "--reason", "ok")
	if cliErr == nil || err.Error() != cliErr.Error() {
		t.Fatalf("el núcleo y la CLI deben rechazar igual:\nnúcleo: %v\nCLI: %v", err, cliErr)
	}
	wantNoWrite(t, ws, before)

	integrateGit(t, ws.Master, "switch", "-q", "feat/x")
	_, err = RecordGate(context.Background(), GateRecordInput{
		CWD: ws.Master, Change: sealChange, Gate: kickoff.GateIntegration, Decision: kickoff.DecisionApproved,
		EvidenceKind: kickoff.EvidenceAttestation, Evidence: "validado a mano",
	})
	wantWrongBranch(t, err, "feat/x", guardArchiveBranch, ws.Master)
	wantContinuation(t, err, " --archive")
}

// D2: los avisos viajan como datos, en orden y también con error.
func TestRecordGateCollectsNoticesInOrder(t *testing.T) {
	root, changeRoot, _ := integrationsWorkspace(t, fakeBlock)
	captureIntegrationWarnings(t)
	original := resolveActorIdentity
	resolveActorIdentity = func(context.Context, string) gitidentity.Identity { return gitidentity.New("", "") }
	t.Cleanup(func() { resolveActorIdentity = original })
	guardOriginal := integrationWriterGuard
	integrationWriterGuard = func(_, _, _ string, _ branch.Context) error { return errors.New("rama equivocada") }
	t.Cleanup(func() { integrationWriterGuard = guardOriginal })

	outcome, err := RecordGate(context.Background(), GateRecordInput{CWD: root, Change: "x", Gate: kickoff.GateSpec, Decision: kickoff.DecisionApproved})
	if err != nil {
		t.Fatalf("RecordGate: %v", err)
	}
	var sources []string
	for _, n := range outcome.Notices {
		sources = append(sources, n.Source)
	}
	if len(sources) != 3 || sources[0] != gateNoticeBranchGuard || sources[1] != gateNoticeActorIdentity || sources[2] != gateNoticeIntegrationDispatch {
		t.Fatalf("avisos = %+v, se esperaba branch_guard, actor_identity e integration_dispatch en ese orden", outcome.Notices)
	}
	if !strings.Contains(outcome.Notices[2].Text, "rama equivocada") || strings.HasPrefix(outcome.Notices[2].Text, "Aviso:") {
		t.Fatalf("el aviso del despacho va sin prefijo y con su causa: %q", outcome.Notices[2].Text)
	}
	if _, statErr := os.Stat(filepath.Join(changeRoot, "gates.yaml")); statErr != nil {
		t.Fatalf("el registro se escribe pese al aviso: %v", statErr)
	}

	// Con error: un rol sin kickoff sellado rechaza tras la guarda y devuelve el resultado igualmente.
	failed, err := RecordGate(context.Background(), GateRecordInput{CWD: root, Change: "x", Gate: kickoff.RoleApplyGate("core"), Decision: kickoff.DecisionApproved})
	if err == nil {
		t.Fatal("se esperaba el rechazo por falta de kickoff sellado")
	}
	if failed.Change != "x" || failed.WorkspaceRoot == "" {
		t.Fatalf("el resultado con error conserva el contexto: %+v", failed)
	}
}
