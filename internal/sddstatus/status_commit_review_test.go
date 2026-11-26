package sddstatus

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// Proyección de `commitReview` en la puerta `tasks` (RF-GAT-22, RF-GAT-23, RF-GAT-24).

func approvedGate(gate kickoff.GateKey, mutate ...func(*kickoff.GateRecord)) kickoff.GateRecord {
	record := kickoff.GateRecord{Gate: gate, Decision: kickoff.DecisionApproved, Reason: "conforme", Actor: "reviewer"}
	for _, apply := range mutate {
		apply(&record)
	}
	return record
}

func withCommitReview(mode kickoff.CommitReviewMode) func(*kickoff.GateRecord) {
	return func(r *kickoff.GateRecord) { r.CommitReview = mode }
}

func withHandOff(r *kickoff.GateRecord) {
	r.HandedOff = true
	r.HandedOffTo = "apply"
}

func commitReviewFixture(t *testing.T, style kickoff.ExecutionStyle, records ...kickoff.GateRecord) (Status, StatusV2Projection) {
	t.Helper()
	root := t.TempDir()
	changeName := "governed-tasks-mode"
	changeRoot := seedReadyChange(t, root, changeName, "- [ ] 1.1 Wire routes\n")
	sealFullstackKickoff(t, changeRoot, changeName, style)
	for _, record := range records {
		if err := kickoff.AppendGate(changeRoot, record); err != nil {
			t.Fatalf("kickoff.AppendGate(%s) error = %v", record.Gate, err)
		}
	}
	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: changeName})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	projection, err := ProjectStatusV2(status)
	if err != nil {
		t.Fatalf("ProjectStatusV2() error = %v", err)
	}
	return status, projection
}

func gatesJSON(t *testing.T, projection StatusV2Projection) string {
	t.Helper()
	if projection.Governance == nil {
		t.Fatal("projection.Governance = nil")
	}
	encoded, err := json.Marshal(projection.Governance.Gates)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func planningApproved(tasks ...func(*kickoff.GateRecord)) []kickoff.GateRecord {
	return []kickoff.GateRecord{
		approvedGate(kickoff.GateSpec),
		approvedGate(kickoff.GateDesign),
		approvedGate(kickoff.GateTasks, tasks...),
	}
}

func TestStatusProjectsTheRecordedCommitReviewMode(t *testing.T) {
	t.Parallel()
	for _, mode := range kickoff.CommitReviewModes() {
		t.Run(string(mode), func(t *testing.T) {
			_, projection := commitReviewFixture(t, kickoff.ExecutionCheckpointed, planningApproved(withCommitReview(mode))...)
			tasks := gateV2ByKey(t, projection, kickoff.GateTasks)
			if tasks.Status != "approved" || tasks.CommitReview != string(mode) {
				t.Fatalf("tasks gate = %+v, want approved with commitReview %q", tasks, mode)
			}
			if !strings.Contains(gatesJSON(t, projection), `"commitReview":"`+string(mode)+`"`) {
				t.Fatalf("gates JSON does not carry commitReview %q: %s", mode, gatesJSON(t, projection))
			}
		})
	}
}

func TestStatusProjectsNotifyWhenTheApprovalCarriesNoMode(t *testing.T) {
	t.Parallel()
	_, projection := commitReviewFixture(t, kickoff.ExecutionCheckpointed, planningApproved()...)
	tasks := gateV2ByKey(t, projection, kickoff.GateTasks)
	if tasks.Status != "approved" || tasks.CommitReview != "notify" {
		t.Fatalf("tasks gate = %+v, want approved with the effective notify", tasks)
	}
}

func TestStatusCommitReviewSurvivesTheHandOff(t *testing.T) {
	t.Parallel()
	_, projection := commitReviewFixture(t, kickoff.ExecutionCheckpointed,
		planningApproved(withHandOff, withCommitReview(kickoff.CommitReviewStopBeforeCommit))...)
	tasks := gateV2ByKey(t, projection, kickoff.GateTasks)
	if !tasks.HandedOff || tasks.HandedOffTo != "apply" || tasks.CommitReview != "stop-before-commit" {
		t.Fatalf("tasks gate = %+v, want hand-off and commitReview together", tasks)
	}
}

// RF-GAT-22: ni una puerta tasks pendiente, ni otra puerta aprobada, ni un cambio continuo lo proyectan.
func TestStatusOmitsCommitReviewWhereItDoesNotApply(t *testing.T) {
	t.Parallel()
	t.Run("tasks pendiente", func(t *testing.T) {
		_, projection := commitReviewFixture(t, kickoff.ExecutionCheckpointed,
			approvedGate(kickoff.GateSpec), approvedGate(kickoff.GateDesign))
		if tasks := gateV2ByKey(t, projection, kickoff.GateTasks); tasks.Status != "pending" || tasks.CommitReview != "" {
			t.Fatalf("tasks gate = %+v, want pending without commitReview", tasks)
		}
		if strings.Contains(gatesJSON(t, projection), "commitReview") {
			t.Fatalf("a pending tasks gate serializes commitReview: %s", gatesJSON(t, projection))
		}
	})
	t.Run("design aprobada", func(t *testing.T) {
		_, projection := commitReviewFixture(t, kickoff.ExecutionCheckpointed, planningApproved(withCommitReview(kickoff.CommitReviewStop))...)
		for _, key := range []kickoff.GateKey{kickoff.GateSpec, kickoff.GateDesign} {
			if gate := gateV2ByKey(t, projection, key); gate.Status != "approved" || gate.CommitReview != "" {
				t.Fatalf("%s gate = %+v, want approved without commitReview", key, gate)
			}
		}
	})
	t.Run("continuous", func(t *testing.T) {
		_, projection := commitReviewFixture(t, kickoff.ExecutionContinuous)
		if strings.Contains(gatesJSON(t, projection), "commitReview") {
			t.Fatalf("a continuous change serializes commitReview: %s", gatesJSON(t, projection))
		}
	})
	t.Run("tasks rechazada", func(t *testing.T) {
		_, projection := commitReviewFixture(t, kickoff.ExecutionCheckpointed,
			approvedGate(kickoff.GateSpec), approvedGate(kickoff.GateDesign),
			kickoff.GateRecord{Gate: kickoff.GateTasks, Decision: kickoff.DecisionRejected, Reason: "falta", Actor: "reviewer"})
		if strings.Contains(gatesJSON(t, projection), "commitReview") {
			t.Fatalf("a rejected tasks gate serializes commitReview: %s", gatesJSON(t, projection))
		}
	})
}

// RF-GAT-24: un rechazo posterior a la aprobación no cambia el modo proyectado.
func TestStatusCommitReviewIsNotChangedByALaterRejection(t *testing.T) {
	t.Parallel()
	records := append(planningApproved(withCommitReview(kickoff.CommitReviewStop)),
		kickoff.GateRecord{Gate: kickoff.GateTasks, Decision: kickoff.DecisionRejected, Reason: "tarde", Actor: "reviewer"})
	_, projection := commitReviewFixture(t, kickoff.ExecutionCheckpointed, records...)
	if tasks := gateV2ByKey(t, projection, kickoff.GateTasks); tasks.Status != "approved" || tasks.CommitReview != "stop" {
		t.Fatalf("tasks gate = %+v, want approved with commitReview stop", tasks)
	}
}

// RF-GAT-22: la proyección es informativa; la ruta, las dependencias y los avisos no cambian.
func TestStatusCommitReviewNeverChangesTheRoute(t *testing.T) {
	t.Parallel()
	plain, _ := commitReviewFixture(t, kickoff.ExecutionCheckpointed, planningApproved()...)
	for _, mode := range kickoff.CommitReviewModes() {
		t.Run(string(mode), func(t *testing.T) {
			status, _ := commitReviewFixture(t, kickoff.ExecutionCheckpointed, planningApproved(withCommitReview(mode))...)
			if status.NextRecommended != plain.NextRecommended {
				t.Errorf("NextRecommended = %q con modo, %q sin modo", status.NextRecommended, plain.NextRecommended)
			}
			if status.Dependencies != plain.Dependencies {
				t.Errorf("Dependencies = %+v con modo, %+v sin modo", status.Dependencies, plain.Dependencies)
			}
			if strings.Join(status.BlockedReasons, "\n") != strings.Join(plain.BlockedReasons, "\n") {
				t.Errorf("BlockedReasons = %v con modo, %v sin modo", status.BlockedReasons, plain.BlockedReasons)
			}
		})
	}
}

// RF-GAT-23: la pregunta tipada de la puerta `tasks` conserva sus tres opciones y no menciona el modo.
func TestTasksGateQuestionKeepsThreeOptionsAndNeverMentionsTheMode(t *testing.T) {
	t.Parallel()
	status, _ := commitReviewFixture(t, kickoff.ExecutionCheckpointed,
		approvedGate(kickoff.GateSpec), approvedGate(kickoff.GateDesign))
	if status.GateQuestion == nil {
		t.Fatal("GateQuestion = nil, want the pending tasks gate question")
	}
	if status.GateQuestion.Gate != string(kickoff.GateTasks) {
		t.Fatalf("GateQuestion.Gate = %q, want tasks", status.GateQuestion.Gate)
	}
	if got := strings.Join(gateAnswers(*status.GateQuestion), ","); got != "rejected,hand_off,approved" {
		t.Fatalf("answers = %s, want rejected,hand_off,approved", got)
	}
	encoded, err := json.Marshal(status.GateQuestion)
	if err != nil {
		t.Fatal(err)
	}
	lowered := strings.ToLower(string(encoded))
	for _, forbidden := range []string{"commit-review", "commit_review", "commitreview", "commit review", "notify", "stop-before-commit"} {
		if strings.Contains(lowered, forbidden) {
			t.Errorf("the typed tasks gate question mentions %q: %s", forbidden, encoded)
		}
	}
}
