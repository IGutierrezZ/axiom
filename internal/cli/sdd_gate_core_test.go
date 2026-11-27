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
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
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

// ledgerRecords devuelve los registros de gates.yaml de un cambio (vacío si no existe).
func ledgerRecords(t *testing.T, changeRoot string) []kickoff.GateRecord {
	t.Helper()
	ledger, err := kickoff.LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("gates.yaml: %v", err)
	}
	return ledger.Records
}

func wantGateCode(t *testing.T, err error, code string) *GateRecordError {
	t.Helper()
	var coded *GateRecordError
	if !errors.As(err, &coded) || coded.Code != code {
		t.Fatalf("error = %v, se esperaba el código %q", err, code)
	}
	if coded.Error() != coded.Unwrap().Error() {
		t.Fatalf("Error() debe ser idéntico al del error envuelto: %q frente a %q", coded.Error(), coded.Unwrap())
	}
	return coded
}

// prov:WEB:2 y D4: un digest visto distinto se rechaza sin escribir; una aprobación previa se niega
// bajo el cerrojo; tras un rechazo del mismo digest se puede volver a decidir (nota 4).
func TestRecordGateExpectedDigest(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
	sealForGateTest(t, root, "inc-99-example")
	if err := os.WriteFile(filepath.Join(changeRoot, "spec.md"), []byte("# spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sealed, err := kickoff.Load(changeRoot)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := gateArtifactDigest(changeRoot, kickoff.GateSpec, sealed)
	if err != nil || digest == "" {
		t.Fatalf("digest = %q, %v", digest, err)
	}
	input := func(decision kickoff.GateDecision, expected string) GateRecordInput {
		return GateRecordInput{CWD: root, Change: "inc-99-example", Gate: kickoff.GateSpec, Decision: decision, Reason: "motivo", Actor: "maintainer", ExpectedDigest: expected}
	}

	_, err = RecordGate(context.Background(), input(kickoff.DecisionApproved, "sha256:otro"))
	coded := wantGateCode(t, err, GateCodeDigestChanged)
	if !strings.Contains(coded.Error(), "axiom sdd status") {
		t.Fatalf("el rechazo nombra su resolución: %q", coded.Error())
	}
	if n := len(ledgerRecords(t, changeRoot)); n != 0 {
		t.Fatalf("un digest distinto no escribe: registros = %d", n)
	}

	if _, err = RecordGate(context.Background(), input(kickoff.DecisionRejected, digest)); err != nil {
		t.Fatalf("rechazo con el digest visto: %v", err)
	}
	if _, err = RecordGate(context.Background(), input(kickoff.DecisionApproved, digest)); err != nil {
		t.Fatalf("tras un rechazo del mismo digest se puede volver a decidir: %v", err)
	}
	_, err = RecordGate(context.Background(), input(kickoff.DecisionApproved, digest))
	wantGateCode(t, err, GateCodeAlreadyResolved)
	if n := len(ledgerRecords(t, changeRoot)); n != 2 {
		t.Fatalf("la aprobación repetida no escribe: registros = %d, se esperaban 2", n)
	}
}

type gateRefusalCase struct {
	cwd, change string
	in          GateRecordInput
	cli         []string
}

// D3 y nota 1: cada negativa conserva el código y el mensaje exacto de la CLI.
func TestRecordGateTypedRefusalsMatchCLI(t *testing.T) {
	spec := func(extra GateRecordInput) GateRecordInput {
		extra.Gate, extra.Decision = kickoff.GateSpec, kickoff.DecisionApproved
		return extra
	}
	cases := []struct {
		name  string
		code  string
		setup func(t *testing.T) gateRefusalCase
	}{
		{"argumentos inválidos", GateCodeArgsInvalid, func(t *testing.T) gateRefusalCase {
			ws := handOffProject(t, guardSealArgs)
			return gateRefusalCase{ws.Master, "x", GateRecordInput{Gate: kickoff.GateSpec, Decision: kickoff.DecisionRejected},
				[]string{"--gate", "spec", "--decision", "rejected"}}
		}},
		{"integration pr_merged sin commit", GateCodeArgsInvalid, func(t *testing.T) gateRefusalCase {
			ws := handOffProject(t, guardSealArgs)
			return gateRefusalCase{ws.Master, "x", GateRecordInput{Gate: kickoff.GateIntegration, Decision: kickoff.DecisionApproved, EvidenceKind: kickoff.EvidencePRMerged},
				[]string{"--gate", "integration", "--decision", "approved", "--evidence-kind", "pr_merged"}}
		}},
		{"cambio inexistente", GateCodeChangeNotFound, func(t *testing.T) gateRefusalCase {
			ws := handOffProject(t, guardSealArgs)
			return gateRefusalCase{ws.Master, "no-existe", spec(GateRecordInput{}), []string{"--gate", "spec", "--decision", "approved"}}
		}},
		{"cambio archivado", GateCodeChangeArchived, func(t *testing.T) gateRefusalCase {
			root, _ := newGovernanceWorkspace(t, "inc-99-example")
			if err := os.MkdirAll(filepath.Join(root, "openspec", "changes", "archive", "2026-01-01-viejo"), 0o755); err != nil {
				t.Fatal(err)
			}
			return gateRefusalCase{root, "viejo", spec(GateRecordInput{}), []string{"--gate", "spec", "--decision", "approved"}}
		}},
		{"cesión en un cambio continuo", GateCodeNoStops, func(t *testing.T) gateRefusalCase {
			ws := handOffProject(t, continuousArgs)
			return gateRefusalCase{ws.Master, "x", spec(GateRecordInput{HandOff: true}), []string{"--gate", "spec", "--decision", "approved", "--hand-off"}}
		}},
		{"modo de revisión en un cambio continuo", GateCodeNoStops, func(t *testing.T) gateRefusalCase {
			ws := handOffProject(t, continuousArgs)
			return gateRefusalCase{ws.Master, "x", GateRecordInput{Gate: kickoff.GateTasks, Decision: kickoff.DecisionApproved, CommitReview: kickoff.CommitReviewNotify},
				[]string{"--gate", "tasks", "--decision", "approved", "--commit-review", "notify"}}
		}},
		{"rol sin kickoff sellado", GateCodeKickoffMissing, func(t *testing.T) gateRefusalCase {
			root, _ := newGovernanceWorkspace(t, "inc-99-example")
			return gateRefusalCase{root, "inc-99-example", GateRecordInput{Gate: kickoff.RoleApplyGate("core"), Decision: kickoff.DecisionApproved},
				[]string{"--gate", "role-apply:core", "--decision", "approved"}}
		}},
		{"rol fuera del roster", GateCodeRoleNotInRoster, func(t *testing.T) gateRefusalCase {
			ws := handOffProject(t, guardSealArgs)
			return gateRefusalCase{ws.Master, "x", GateRecordInput{Gate: kickoff.RoleApplyGate("zzz"), Decision: kickoff.DecisionApproved},
				[]string{"--gate", "role-apply:zzz", "--decision", "approved"}}
		}},
		{"almacén sin ficheros", GateCodeHandOffNA, func(t *testing.T) gateRefusalCase {
			ws := handOffProject(t, guardSealArgs)
			workspacetest.WriteFile(t, filepath.Join(ws.Home, "config.yaml"), "artifact_store: engram\n")
			integrateCommit(t, ws.Master, "almacén")
			return gateRefusalCase{ws.Master, "x", spec(GateRecordInput{HandOff: true}), []string{"--gate", "spec", "--decision", "approved", "--hand-off"}}
		}},
		{"artefacto ausente", GateCodeArtifactMissing, func(t *testing.T) gateRefusalCase {
			ws := handOffProject(t, guardSealArgs)
			if err := os.Remove(filepath.Join(guardChangeRoot(ws), "design.md")); err != nil {
				t.Fatal(err)
			}
			integrateCommit(t, ws.Master, "sin diseño")
			return gateRefusalCase{ws.Master, "x", GateRecordInput{Gate: kickoff.GateDesign, Decision: kickoff.DecisionApproved, HandOff: true},
				[]string{"--gate", "design", "--decision", "approved", "--hand-off"}}
		}},
		{"puerta ya resuelta", GateCodeAlreadyResolved, func(t *testing.T) gateRefusalCase {
			ws := handOffProject(t, guardSealArgs)
			if _, err := gateRecord(t, ws, "--gate", "spec", "--decision", "approved"); err != nil {
				t.Fatal(err)
			}
			return gateRefusalCase{ws.Master, "x", spec(GateRecordInput{HandOff: true}), []string{"--gate", "spec", "--decision", "approved", "--hand-off"}}
		}},
		{"evidencia rechazada", GateCodeEvidenceRefused, func(t *testing.T) gateRefusalCase {
			root, _ := newGovernanceGitWorkspace(t, "inc-99-example")
			runGateTestGit(t, root, "checkout", "-q", "-b", "feature-x")
			writeCLIFixture(t, filepath.Join(root, "feature.txt"), "feature\n")
			runGateTestGit(t, root, "add", "--", "feature.txt")
			runGateTestGit(t, root, "commit", "-q", "-m", "feature")
			divergent := strings.TrimSpace(runGateTestGit(t, root, "rev-parse", "HEAD"))
			runGateTestGit(t, root, "checkout", "-q", "main")
			return gateRefusalCase{root, "inc-99-example",
				GateRecordInput{Gate: kickoff.GateIntegration, Decision: kickoff.DecisionApproved, Reason: "PR fusionado", EvidenceKind: kickoff.EvidencePRMerged, Commit: divergent},
				[]string{"--gate", "integration", "--decision", "approved", "--reason", "PR fusionado", "--evidence-kind", "pr_merged", "--commit", divergent}}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rc := c.setup(t)
			rc.in.CWD, rc.in.Change = rc.cwd, rc.change
			cliErr := RunSDDGate(append([]string{"record", "--cwd", rc.cwd, "--change", rc.change}, rc.cli...), &bytes.Buffer{})
			_, err := RecordGate(context.Background(), rc.in)
			if cliErr == nil {
				t.Fatal("la CLI debía rechazar")
			}
			wantGateCode(t, err, c.code)
			if err.Error() != cliErr.Error() {
				t.Fatalf("mensaje distinto de la CLI:\nnúcleo: %q\nCLI:    %q", err, cliErr)
			}
		})
	}
}
