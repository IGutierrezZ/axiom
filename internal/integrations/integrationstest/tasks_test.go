package integrationstest

import (
	"context"
	"errors"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

var taskConfig = workspace.IntegrationPluginConfig{ID: FakeID, Enabled: true, Values: map[string]any{"project": "p"}}

func TestFakeDoesNotOfferTheTasksPortButTaskFakeDoes(t *testing.T) {
	var plain integrations.Plugin = NewFake()
	if _, ok := plain.(integrations.TaskTracker); ok {
		t.Error("Fake no debe ofrecer el puerto de Tasks")
	}
	var tasks integrations.Plugin = NewTaskFake()
	if _, ok := tasks.(integrations.TaskTracker); !ok {
		t.Error("TaskFake debe ofrecer el puerto de Tasks")
	}

	_, err := integrations.WithTaskTracker(plain, taskConfig, secrets.NewMemory(), func(tt integrations.TaskTracker, req integrations.PortRequest) (int, error) {
		t.Fatal("no debe invocarse sin puerto")
		return 0, nil
	})
	if !errors.Is(err, integrations.ErrPortUnsupported) {
		t.Errorf("WithTaskTracker sin puerto = %v", err)
	}
}

func TestTaskFakeEnsureChildTaskReusesByMarkAndDetectsAmbiguity(t *testing.T) {
	f := NewTaskFake()
	ctx := context.Background()
	req := integrations.CreateTaskRequest{ParentID: "100", Title: "c / r", Mark: "axiom-task:c:r", EstimateHours: 6}

	created, err := f.EnsureChildTask(ctx, req)
	if err != nil || created.Outcome != integrations.TaskCreated || created.Task.Rev != 1 || created.Task.Hours.Original != 6 || created.Task.ParentID != "100" {
		t.Fatalf("creación = %+v, %v", created, err)
	}
	req.Mark = "AXIOM-TASK:C:R"
	again, err := f.EnsureChildTask(ctx, req)
	if err != nil || again.Outcome != integrations.TaskFoundByMark || again.Task.ID != created.Task.ID {
		t.Fatalf("reutilización = %+v, %v", again, err)
	}
	children, _ := f.ChildTasks(ctx, integrations.ChildTasksRequest{ParentID: "100"})
	if len(children) != 1 {
		t.Errorf("hijas = %d, quiere 1", len(children))
	}

	f.SeedTask(integrations.TaskItem{ID: "9", ParentID: "100", Tags: []string{"axiom-task:c:r"}})
	if _, err := f.EnsureChildTask(ctx, req); !errors.Is(err, integrations.ErrTaskAmbiguous) {
		t.Errorf("dos marcas = %v, quiere task_ambiguous", err)
	}
}

func TestTaskFakeValidateReadAndErrors(t *testing.T) {
	f := NewTaskFake().
		SeedTask(integrations.TaskItem{ID: "1", ParentID: "100"}).
		SeedTask(integrations.TaskItem{ID: "2", ParentID: "100", Type: "Bug"}).
		SeedTask(integrations.TaskItem{ID: "3", ParentID: "200"})
	ctx := context.Background()

	tests := []struct {
		id   string
		want error
	}{{"1", nil}, {"2", integrations.ErrNotATask}, {"3", integrations.ErrNotChildOfWorkItem}, {"4", integrations.ErrTaskNotFound}}
	for _, tc := range tests {
		_, err := f.ValidateTask(ctx, integrations.TaskRequest{TaskID: tc.id, ParentID: "100"})
		if !errors.Is(err, tc.want) && !(tc.want == nil && err == nil) {
			t.Errorf("ValidateTask(%s) = %v, quiere %v", tc.id, err, tc.want)
		}
	}
	if _, err := f.ReadTask(ctx, integrations.TaskRequest{TaskID: "9"}); !errors.Is(err, integrations.ErrTaskNotFound) {
		t.Errorf("ReadTask inexistente = %v", err)
	}
	boom := errors.New("boom")
	f.FailMethod("ReadTask", boom)
	if _, err := f.ReadTask(ctx, integrations.TaskRequest{TaskID: "1"}); !errors.Is(err, boom) {
		t.Errorf("fallo programado = %v", err)
	}
}

func TestTaskFakeHoursWithRevisionControl(t *testing.T) {
	f := NewTaskFake().SeedTask(integrations.TaskItem{ID: "1", ParentID: "100", Hours: integrations.TaskHours{Original: 8, Remaining: 8, HasOriginal: true}})
	ctx := context.Background()
	base, _ := f.ReadTask(ctx, integrations.TaskRequest{TaskID: "1"})

	res, err := f.RecordHours(ctx, integrations.HoursRequest{Base: base, Hours: 3})
	if err != nil || res.Task.Hours.Completed != 3 || res.Task.Hours.Remaining != 5 || res.Task.Rev != base.Rev+1 {
		t.Fatalf("RecordHours = %+v, %v", res, err)
	}

	// Revisión obsoleta: conflicto con los valores releídos y sin reaplicar.
	_, err = f.RecordHours(ctx, integrations.HoursRequest{Base: base, Hours: 1})
	var conflict *integrations.RevConflictError
	if !errors.Is(err, integrations.ErrRevConflict) || !errors.As(err, &conflict) || conflict.Hours.Completed != 3 {
		t.Fatalf("revisión obsoleta = %v", err)
	}
	if got, _ := f.Task("1"); got.Hours.Completed != 3 {
		t.Errorf("un conflicto no debe aplicar horas: %+v", got.Hours)
	}

	// Piso en cero y conflicto programado.
	fresh, _ := f.ReadTask(ctx, integrations.TaskRequest{TaskID: "1"})
	res, err = f.RecordHours(ctx, integrations.HoursRequest{Base: fresh, Hours: 20})
	if err != nil || res.Task.Hours.Remaining != 0 {
		t.Errorf("piso en cero = %+v, %v", res.Task.Hours, err)
	}
	f.ConflictNext(1)
	fresh, _ = f.ReadTask(ctx, integrations.TaskRequest{TaskID: "1"})
	if _, err := f.RecordHours(ctx, integrations.HoursRequest{Base: fresh, Hours: 1}); !errors.Is(err, integrations.ErrRevConflict) {
		t.Errorf("conflicto programado = %v", err)
	}
}

func TestTaskFakeEstimateTransitionAndSummary(t *testing.T) {
	f := NewTaskFake().SeedTask(integrations.TaskItem{ID: "1", ParentID: "100", State: "To Do"})
	ctx := context.Background()

	res, err := f.SetEstimate(ctx, integrations.EstimateRequest{TaskID: "1", Hours: 5, Mode: integrations.EstimateIfMissing})
	if err != nil || !res.Written || res.Task.Hours.Original != 5 {
		t.Fatalf("if_missing sin estimación = %+v, %v", res, err)
	}
	res, err = f.SetEstimate(ctx, integrations.EstimateRequest{TaskID: "1", Hours: 9, Mode: integrations.EstimateIfMissing})
	if err != nil || res.Written || res.Task.Hours.Original != 5 {
		t.Errorf("if_missing con estimación no debe escribir: %+v, %v", res, err)
	}

	id := integrations.TrackerIdentity{Account: "org", ID: "u-1"}
	tr, err := f.TransitionTask(ctx, integrations.TransitionRequest{TaskID: "1", State: "Active", Rule: integrations.AdvanceFromProposed, Assignee: &id})
	if err != nil || !tr.StateChanged || !tr.Assigned || tr.Task.AssignedToID != "u-1" {
		t.Fatalf("transición = %+v, %v", tr, err)
	}
	tr, _ = f.TransitionTask(ctx, integrations.TransitionRequest{TaskID: "1", State: "Active", Rule: integrations.AdvanceFromProposed})
	if tr.StateChanged || tr.Assigned {
		t.Errorf("transición repetida no debe escribir: %+v", tr)
	}

	sum, err := f.UpdateSummary(ctx, integrations.SummaryRequest{ItemID: "100", HTML: "<p>x</p>", Tags: []string{"req:prov:A:1"}})
	if err != nil || !sum.Wrote || len(f.Summaries()) != 1 {
		t.Errorf("UpdateSummary = %+v, %v", sum, err)
	}
	if calls := f.TaskCalls(); len(calls) != 5 || calls[0] != "SetEstimate" || calls[4] != "UpdateSummary" {
		t.Errorf("llamadas = %v", calls)
	}
}
