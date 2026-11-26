package integrations

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// taskPlugin ofrece el puerto de Tasks con un solo método programable; el resto devuelve vacío.
type taskPlugin struct {
	*testPlugin
	read func(ctx context.Context, req TaskRequest) (TaskItem, error)
}

func (p *taskPlugin) ChildTasks(context.Context, ChildTasksRequest) ([]TaskItem, error) {
	return nil, nil
}
func (p *taskPlugin) EnsureChildTask(context.Context, CreateTaskRequest) (CreateTaskResult, error) {
	return CreateTaskResult{}, nil
}
func (p *taskPlugin) ReadTask(ctx context.Context, req TaskRequest) (TaskItem, error) {
	return p.read(ctx, req)
}
func (p *taskPlugin) ValidateTask(context.Context, TaskRequest) (TaskItem, error) {
	return TaskItem{}, nil
}
func (p *taskPlugin) SetEstimate(context.Context, EstimateRequest) (EstimateResult, error) {
	return EstimateResult{}, nil
}
func (p *taskPlugin) RecordHours(context.Context, HoursRequest) (HoursResult, error) {
	return HoursResult{}, nil
}
func (p *taskPlugin) TransitionTask(context.Context, TransitionRequest) (TransitionResult, error) {
	return TransitionResult{}, nil
}
func (p *taskPlugin) UpdateSummary(context.Context, SummaryRequest) (SummaryResult, error) {
	return SummaryResult{}, nil
}

func readTask(p Plugin, store *countingStore) (TaskItem, error) {
	return WithTaskTracker(p, testPortConfig, store, func(t TaskTracker, req PortRequest) (TaskItem, error) {
		return t.ReadTask(context.Background(), TaskRequest{PortRequest: req, TaskID: "7"})
	})
}

func TestWithTaskTrackerWithoutPortMakesNoRequestsAndResolvesNoSecrets(t *testing.T) {
	store := newCountingStore(t, "valor-de-prueba")
	_, err := readTask(newTestPlugin(), store)

	var unsupported *PortUnsupportedError
	if !errors.Is(err, ErrPortUnsupported) || !errors.As(err, &unsupported) || unsupported.Port != PortTasks || unsupported.Plugin != "fake" {
		t.Fatalf("sin puerto = %v", err)
	}
	if !strings.Contains(err.Error(), "Tasks") {
		t.Errorf("el error debe nombrar el puerto: %q", err)
	}
	if store.calls.Load() != 0 {
		t.Errorf("sin puerto no se resuelve ningún secreto (consultas: %d)", store.calls.Load())
	}
}

func TestWithTaskTrackerRecoversPanicAndSanitizesSecret(t *testing.T) {
	const secret = "valor-secreto-0123"
	store := newCountingStore(t, secret)
	cause := errors.New("el tracker rechazó la credencial")

	leaking := &taskPlugin{testPlugin: newTestPlugin()}
	leaking.read = func(ctx context.Context, req TaskRequest) (TaskItem, error) {
		got, err := req.Secrets.Secret(ctx, "token")
		if err != nil {
			return TaskItem{}, err
		}
		return TaskItem{}, errors.Join(cause, errors.New("token "+got.Reveal()))
	}
	_, err := readTask(leaking, store)
	assertSanitizedPortError(t, err, cause, secret, PortTasks)

	panicking := &taskPlugin{testPlugin: newTestPlugin()}
	panicking.read = func(ctx context.Context, req TaskRequest) (TaskItem, error) {
		got, _ := req.Secrets.Secret(ctx, "token")
		panic("fallo con " + got.Reveal())
	}
	_, err = readTask(panicking, store)
	var portErr *PortError
	if !errors.As(err, &portErr) || portErr.Port != PortTasks {
		t.Fatalf("un pánico debe volver como *PortError del puerto: %v", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("el pánico filtra el secreto: %q", err)
	}
}

func TestTaskTrackerNeutralErrorsAreRecoverable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		sentinel error
		check    func(t *testing.T, err error)
	}{
		{"no encontrada", ErrTaskNotFound, ErrTaskNotFound, nil},
		{"no es Task", ErrNotATask, ErrNotATask, nil},
		{"otro padre", ErrNotChildOfWorkItem, ErrNotChildOfWorkItem, nil},
		{"enlace con el padre", ErrTaskParentLinkFailed, ErrTaskParentLinkFailed, nil},
		{"rechazo del tracker", ErrTrackerRejected, ErrTrackerRejected, nil},
		{"sin vínculo", ErrTaskNotBound, ErrTaskNotBound, nil},
		{"ambigua", &TaskAmbiguousError{Mark: "axiom-task:c:r", IDs: []string{"1", "2"}}, ErrTaskAmbiguous, func(t *testing.T, err error) {
			var e *TaskAmbiguousError
			if !errors.As(err, &e) || len(e.IDs) != 2 || e.Mark != "axiom-task:c:r" {
				t.Errorf("errors.As = %v", e)
			}
		}},
		{"campo ausente", &FieldMissingError{Field: "Microsoft.VSTS.Scheduling.CompletedWork"}, ErrFieldMissing, func(t *testing.T, err error) {
			var e *FieldMissingError
			if !errors.As(err, &e) || e.Field == "" {
				t.Errorf("errors.As = %v", e)
			}
		}},
		{"estado desconocido", &StateUnknownError{State: "Nope", Valid: []string{"To Do", "Done"}}, ErrStateUnknown, func(t *testing.T, err error) {
			var e *StateUnknownError
			if !errors.As(err, &e) || len(e.Valid) != 2 || !strings.Contains(err.Error(), "Done") {
				t.Errorf("errors.As = %v (%q)", e, err)
			}
		}},
		{"conflicto de revisión", &RevConflictError{Rev: 4, Hours: TaskHours{Completed: 2}}, ErrRevConflict, func(t *testing.T, err error) {
			var e *RevConflictError
			if !errors.As(err, &e) || e.Rev != 4 || e.Hours.Completed != 2 {
				t.Errorf("errors.As = %v", e)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plugin := &taskPlugin{testPlugin: newTestPlugin()}
			plugin.read = func(context.Context, TaskRequest) (TaskItem, error) { return TaskItem{}, tc.err }
			_, err := readTask(plugin, newCountingStore(t, "valor-de-prueba"))
			if !errors.Is(err, tc.sentinel) {
				t.Fatalf("errors.Is(%v, %v) = false", err, tc.sentinel)
			}
			if tc.check != nil {
				tc.check(t, err)
			}
		})
	}
}

func TestWithTaskTrackerReturnsTheResultAndScopesTheRequest(t *testing.T) {
	plugin := &taskPlugin{testPlugin: newTestPlugin()}
	plugin.read = func(_ context.Context, req TaskRequest) (TaskItem, error) {
		if req.Config["project"] != "p" || req.Secrets == nil {
			t.Errorf("la petición debe traer la configuración y los secretos: %+v", req.PortRequest)
		}
		return TaskItem{ID: req.TaskID, Rev: 3}, nil
	}
	got, err := readTask(plugin, newCountingStore(t, "valor-de-prueba"))
	if err != nil || got.ID != "7" || got.Rev != 3 {
		t.Fatalf("ReadTask = %+v, %v", got, err)
	}
}
