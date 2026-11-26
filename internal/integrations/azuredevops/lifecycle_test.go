package azuredevops_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

const taskID = 77

// seedLifecycleTask siembra la Task 77, hija del work item 42, en el estado dado.
func (e *taskEnv) seedLifecycleTask(state string, extra map[string]any) {
	e.seedTask(taskID, 1, taskMark, state, taskParent, extra)
}

func (e *taskEnv) transition(t *testing.T, state string, rule integrations.TransitionRule, assignee *integrations.TrackerIdentity) (integrations.TransitionResult, error) {
	t.Helper()
	return e.p.TransitionTask(context.Background(), integrations.TransitionRequest{
		PortRequest: e.port, TaskID: itoa(taskID), State: state, Rule: rule, Assignee: assignee,
	})
}

func warningCodes(ws []integrations.Warning) []string {
	codes := make([]string, 0, len(ws))
	for _, w := range ws {
		codes = append(codes, w.Code)
	}
	return codes
}

func hasCode(ws []integrations.Warning, code string) bool {
	for _, w := range ws {
		if w.Code == code {
			return true
		}
	}
	return false
}

func TestTransitionTaskMovesAndRepeatedCallDoesNotWrite(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("To Do", nil)

	res, err := e.transition(t, "Active", integrations.AdvanceFromProposed, nil)
	if err != nil || !res.StateChanged || res.Task.State != "Active" {
		t.Fatalf("role.start = %+v, %v", res, err)
	}
	if e.srv.AppliedPatches(taskID) != 1 {
		t.Fatalf("PATCH aplicados = %d, esperaba 1", e.srv.AppliedPatches(taskID))
	}

	res, err = e.transition(t, "Active", integrations.AdvanceFromProposed, nil)
	if err != nil || res.StateChanged || len(res.Warnings) != 0 {
		t.Fatalf("repetida = %+v, %v", res, err)
	}
	if e.srv.AppliedPatches(taskID) != 1 {
		t.Fatalf("una transición repetida no escribe: PATCH = %d", e.srv.AppliedPatches(taskID))
	}
}

func TestTransitionTaskApplyApprovedMovesToTechReviewUnlessClosed(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("Active", nil)
	res, err := e.transition(t, "Tech Review", integrations.AdvanceUnlessClosed, nil)
	if err != nil || !res.StateChanged || res.Task.State != "Tech Review" {
		t.Fatalf("apply aprobado = %+v, %v", res, err)
	}

	e.seedLifecycleTask("Done", nil)
	res, err = e.transition(t, "Tech Review", integrations.AdvanceUnlessClosed, nil)
	if err != nil || res.StateChanged || !hasCode(res.Warnings, "task_state_kept") {
		t.Fatalf("una Task cerrada no se mueve y avisa: %+v, %v", res, err)
	}
	if got := e.read(t, taskID).State; got != "Done" {
		t.Fatalf("la Task cerrada sigue en Done, está en %q", got)
	}
}

func TestTransitionTaskReentryAfterTechReviewKeepsStateAndWarns(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("Tech Review", nil)

	res, err := e.transition(t, "Active", integrations.AdvanceFromProposed, nil)

	if err != nil || res.StateChanged || !hasCode(res.Warnings, "task_state_kept") {
		t.Fatalf("reentrada = %+v, %v", res, err)
	}
	if e.srv.AppliedPatches(taskID) != 0 {
		t.Fatal("la reentrada no escribe")
	}
}

func TestTransitionTaskUnknownStateListsValidOnesWithoutWriting(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("To Do", nil)

	_, err := e.transition(t, "Inexistente", integrations.AdvanceFromProposed, nil)

	var unknown *integrations.StateUnknownError
	if !errors.As(err, &unknown) || !errors.Is(err, integrations.ErrStateUnknown) {
		t.Fatalf("error = %v", err)
	}
	if strings.Join(unknown.Valid, ",") != "To Do,Active,Tech Review,Done,Removed" {
		t.Fatalf("válidos = %v", unknown.Valid)
	}
	if e.srv.AppliedPatches(taskID) != 0 {
		t.Fatal("un estado inexistente no escribe")
	}
}

func TestTransitionTaskProcessRulesRejectionIsTrackerRejected(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("To Do", nil)
	e.srv.RejectStateChange("TF401320: la regla impide pasar de To Do a Active")

	_, err := e.transition(t, "Active", integrations.AdvanceFromProposed, nil)

	if !errors.Is(err, integrations.ErrTrackerRejected) || !strings.Contains(err.Error(), "TF401320") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), testPAT) {
		t.Fatal("el error no filtra el PAT")
	}
}

func TestTransitionTaskMissingTaskIsNotFound(t *testing.T) {
	e := newTaskEnv(t)

	_, err := e.transition(t, "Active", integrations.AdvanceFromProposed, nil)

	if !errors.Is(err, integrations.ErrTaskNotFound) {
		t.Fatalf("error = %v", err)
	}
}

func ana() *integrations.TrackerIdentity {
	return &integrations.TrackerIdentity{Account: "contoso", ID: "a1", DisplayName: "Ana Gil", Source: "connection_data"}
}

func seedMembers(e *taskEnv, members ...adotest.Member) { e.srv.SeedMembers(members...) }

func TestTransitionTaskAssignsOnceWithTheStateAndNeverReadsTheUniqueName(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("To Do", nil)
	seedMembers(e,
		adotest.Member{ID: "a1", DisplayName: "Ana Gil", UniqueName: "ana.gil.privado"},
		adotest.Member{ID: "b2", DisplayName: "Berto Ruiz", UniqueName: "berto.privado"})

	res, err := e.transition(t, "Active", integrations.AdvanceFromProposed, ana())

	if err != nil || !res.StateChanged || !res.Assigned || res.Task.AssignedToID != "a1" || len(res.Warnings) != 0 {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	if e.srv.AppliedPatches(taskID) != 1 {
		t.Fatalf("estado y asignación van en un solo PATCH, hubo %d", e.srv.AppliedPatches(taskID))
	}
	encoded, _ := json.Marshal(res)
	if strings.Contains(string(encoded), "privado") {
		t.Fatalf("el resultado no contiene el uniqueName: %s", encoded)
	}
}

func TestTransitionTaskNilAssigneeDoesNotAssign(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("To Do", nil)
	seedMembers(e, adotest.Member{ID: "a1", DisplayName: "Ana Gil"})

	res, err := e.transition(t, "Active", integrations.AdvanceFromProposed, nil)

	if err != nil || res.Assigned || res.Task.AssignedToID != "" {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	if e.srv.Count("GET", "/contoso/_apis/projects/Fabrikam/teams/Fabrikam Team/members") != 0 {
		t.Fatal("sin asignado no se leen los miembros")
	}
}

func TestTransitionTaskHomonymsAreNotAssignedButTheStateIsApplied(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("To Do", nil)
	seedMembers(e,
		adotest.Member{ID: "a1", DisplayName: "Ana Gil"},
		adotest.Member{ID: "a9", DisplayName: "ana gil"})

	res, err := e.transition(t, "Active", integrations.AdvanceFromProposed, ana())

	if err != nil || !res.StateChanged || res.Assigned || !hasCode(res.Warnings, "assignment_ambiguous") {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	if got := e.read(t, taskID); got.State != "Active" || got.AssignedToID != "" {
		t.Fatalf("Task = %+v", got)
	}
}

func TestTransitionTaskAlreadyAssignedToAnotherIsNotReassigned(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("To Do", map[string]any{"System.AssignedTo": map[string]any{"id": "b2", "displayName": "Berto Ruiz"}})
	seedMembers(e, adotest.Member{ID: "a1", DisplayName: "Ana Gil"})

	res, err := e.transition(t, "Active", integrations.AdvanceFromProposed, ana())

	if err != nil || !res.StateChanged || res.Assigned || !hasCode(res.Warnings, "task_assigned_to_other") {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	if got := e.read(t, taskID).AssignedToID; got != "b2" {
		t.Fatalf("asignada a %q, esperaba b2", got)
	}
}

func TestTransitionTaskReturnedIDDifferentFromStoredWarnsMismatch(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("To Do", nil)
	// El equipo resuelve «Ana Gil» a otro id que el guardado.
	seedMembers(e, adotest.Member{ID: "otro-id", DisplayName: "Ana Gil"})

	res, err := e.transition(t, "Active", integrations.AdvanceFromProposed, ana())

	if err != nil || res.Assigned || !hasCode(res.Warnings, "assignment_mismatch") {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
}

func TestTransitionTaskAlreadyActiveButUnassignedReceivesOnlyTheAssignment(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("Active", nil)
	seedMembers(e, adotest.Member{ID: "a1", DisplayName: "Ana Gil"})

	res, err := e.transition(t, "Active", integrations.AdvanceFromProposed, ana())

	if err != nil || res.StateChanged || !res.Assigned {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	var patch adotest.Request
	for _, r := range e.srv.Requests() {
		if r.Method == "PATCH" {
			patch = r
		}
	}
	ops := patchOps(t, patch.Body)
	if opValue(ops, "/fields/System.AssignedTo") == nil || opValue(ops, "/fields/System.State") != nil {
		t.Fatalf("el PATCH solo asigna: %v", ops)
	}
}

func TestTransitionTaskKeptByRuleStillReceivesTheAssignment(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("Tech Review", nil)
	seedMembers(e, adotest.Member{ID: "a1", DisplayName: "Ana Gil"})

	res, err := e.transition(t, "Active", integrations.AdvanceFromProposed, ana())

	if err != nil || res.StateChanged || !res.Assigned || !hasCode(res.Warnings, "task_state_kept") {
		t.Fatalf("resultado = %+v, %v (avisos %v)", res, err, warningCodes(res.Warnings))
	}
}

func TestTransitionTaskAlreadyAssignedToTheSamePersonAndInTargetDoesNotWrite(t *testing.T) {
	e := newTaskEnv(t)
	e.seedLifecycleTask("Active", map[string]any{"System.AssignedTo": map[string]any{"id": "a1", "displayName": "Ana Gil"}})

	res, err := e.transition(t, "Active", integrations.AdvanceFromProposed, ana())

	if err != nil || res.StateChanged || res.Assigned || e.srv.AppliedPatches(taskID) != 0 {
		t.Fatalf("resultado = %+v, %v, PATCH = %d", res, err, e.srv.AppliedPatches(taskID))
	}
}
