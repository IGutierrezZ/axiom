package kickoff

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Pruebas del modo de revisión por commit (REQ-33.1, REQ-33.3, REQ-33.4, REQ-33.6).

func TestCommitReviewVocabularyIsClosedAndOrdered(t *testing.T) {
	want := []CommitReviewMode{"notify", "stop", "stop-before-commit"}
	if got := CommitReviewModes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("CommitReviewModes() = %v, se esperaba %v", got, want)
	}
	for mode, effective := range map[CommitReviewMode]CommitReviewMode{
		"":                   CommitReviewNotify,
		"notify":             CommitReviewNotify,
		"stop":               CommitReviewStop,
		"stop-before-commit": CommitReviewStopBeforeCommit,
	} {
		if got := EffectiveCommitReview(mode); got != effective {
			t.Errorf("EffectiveCommitReview(%q) = %q, se esperaba %q", mode, got, effective)
		}
	}
}

func TestGateRecordValidateCommitReview(t *testing.T) {
	tasksApproved := func(mode CommitReviewMode) GateRecord {
		return GateRecord{Gate: GateTasks, Decision: DecisionApproved, CommitReview: mode}
	}
	cases := []struct {
		name    string
		record  GateRecord
		wantErr string
	}{
		{"sin campo", tasksApproved(""), ""},
		{"notify", tasksApproved(CommitReviewNotify), ""},
		{"stop", tasksApproved(CommitReviewStop), ""},
		{"stop-before-commit", tasksApproved(CommitReviewStopBeforeCommit), ""},
		{"Stop con mayúscula", tasksApproved("Stop"), "commit_review desconocido"},
		{"pause", tasksApproved("pause"), "commit_review desconocido"},
		{"always", tasksApproved("always"), "commit_review desconocido"},
		{"otra puerta", GateRecord{Gate: GateDesign, Decision: DecisionApproved, CommitReview: CommitReviewStop}, "solo se admite en la aprobacion de la compuerta tasks"},
		{"role-apply", GateRecord{Gate: RoleApplyGate("core"), Decision: DecisionApproved, CommitReview: CommitReviewStop}, "solo se admite"},
		{"rechazo de tasks", GateRecord{Gate: GateTasks, Decision: DecisionRejected, CommitReview: CommitReviewStop}, "solo se admite"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.record.Validate()
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("Validate() error = %v, se esperaba nil", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Fatalf("Validate() error = %v, se esperaba que contuviera %q", err, tc.wantErr)
			}
		})
	}
}

func commitReviewLedgerYAML(mode string) string {
	return "schema: axiom.sdd-gate-ledger/v1\nchange: x\nrecords:\n" +
		"  - gate: tasks\n    decision: approved\n    reason: \"\"\n" + mode +
		"    actor: maintainer\n    recorded_at: 2026-01-01T00:00:00Z\n"
}

func TestParseGateLedgerCommitReview(t *testing.T) {
	t.Run("un valor inválido persistido se rechaza al leer", func(t *testing.T) {
		_, err := ParseGateLedger([]byte(commitReviewLedgerYAML("    commit_review: always\n")))
		if err == nil || !strings.Contains(err.Error(), "commit_review desconocido") {
			t.Fatalf("ParseGateLedger() error = %v", err)
		}
	})
	t.Run("un ledger antiguo sin campo se lee como notify", func(t *testing.T) {
		ledger, err := ParseGateLedger([]byte(commitReviewLedgerYAML("")))
		if err != nil {
			t.Fatalf("ParseGateLedger() error = %v", err)
		}
		if ledger.Records[0].CommitReview != "" || EffectiveCommitReview(ledger.Records[0].CommitReview) != CommitReviewNotify {
			t.Fatalf("registro = %+v", ledger.Records[0])
		}
	})
	t.Run("el campo se serializa solo cuando existe", func(t *testing.T) {
		with, err := yaml.Marshal(GateRecord{Gate: GateTasks, Decision: DecisionApproved, CommitReview: CommitReviewStop})
		if err != nil || !strings.Contains(string(with), "commit_review: stop") {
			t.Fatalf("serialización = %q (%v)", with, err)
		}
		without, err := yaml.Marshal(GateRecord{Gate: GateTasks, Decision: DecisionApproved})
		if err != nil || strings.Contains(string(without), "commit_review") {
			t.Fatalf("sin modo no debe haber clave: %q (%v)", without, err)
		}
	})
	t.Run("un decodificador anterior no estricto ignora el campo", func(t *testing.T) {
		// previousRecord es el registro previo a este cambio: no conoce commit_review.
		type previousRecord struct {
			Gate     GateKey      `yaml:"gate"`
			Decision GateDecision `yaml:"decision"`
			Reason   string       `yaml:"reason"`
		}
		var ledger struct {
			Records []previousRecord `yaml:"records"`
		}
		if err := yaml.Unmarshal([]byte(commitReviewLedgerYAML("    commit_review: stop\n")), &ledger); err != nil {
			t.Fatalf("decodificación anterior: %v", err)
		}
		if len(ledger.Records) != 1 || ledger.Records[0].Decision != DecisionApproved {
			t.Fatalf("registros = %+v, se esperaba approved", ledger.Records)
		}
	})
}

// REQ-33.4 y REQ-33.6: el modo es el del primer approved de la puerta; un rechazo posterior no lo
// cambia y un ledger anterior da modo vacío (efectivo notify).
func TestEvaluateGatesCommitReviewComesFromTheClosingApproval(t *testing.T) {
	cases := []struct {
		name    string
		records []GateRecord
		want    CommitReviewMode
	}{
		{
			name:    "approved con stop",
			records: []GateRecord{{Gate: GateTasks, Decision: DecisionApproved, ArtifactDigest: "d1", CommitReview: CommitReviewStop}},
			want:    CommitReviewStop,
		},
		{
			name:    "approved sin campo da modo vacío",
			records: []GateRecord{{Gate: GateTasks, Decision: DecisionApproved, ArtifactDigest: "d1"}},
			want:    "",
		},
		{
			name: "un rejected posterior no cambia el modo",
			records: []GateRecord{
				{Gate: GateTasks, Decision: DecisionApproved, ArtifactDigest: "d1", CommitReview: CommitReviewStopBeforeCommit},
				{Gate: GateTasks, Decision: DecisionRejected, Reason: "x", ArtifactDigest: "d1"},
			},
			want: CommitReviewStopBeforeCommit,
		},
		{
			name: "un rejected previo no lleva modo y el approved lo aporta",
			records: []GateRecord{
				{Gate: GateTasks, Decision: DecisionRejected, Reason: "x", ArtifactDigest: "d0"},
				{Gate: GateTasks, Decision: DecisionApproved, ArtifactDigest: "d1", CommitReview: CommitReviewStop},
			},
			want: CommitReviewStop,
		},
		{
			name:    "puerta pendiente sin registros",
			records: nil,
			want:    "",
		},
		{
			name:    "puerta rechazada",
			records: []GateRecord{{Gate: GateTasks, Decision: DecisionRejected, Reason: "x", ArtifactDigest: "d1"}},
			want:    "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			states, err := EvaluateGates(Inputs{
				Execution: ExecutionCheckpointed,
				Artifacts: map[string]string{"tasks": "d1"},
				Ledger:    GateLedger{Schema: GateLedgerSchemaV1, Records: tc.records},
			})
			if err != nil {
				t.Fatalf("EvaluateGates() error = %v", err)
			}
			state := findGateState(t, states, GateTasks)
			if state.CommitReview != tc.want {
				t.Errorf("CommitReview = %q, se esperaba %q (estado %+v)", state.CommitReview, tc.want, state)
			}
		})
	}
}
