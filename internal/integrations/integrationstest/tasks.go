package integrationstest

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

// TaskFake es el Fake con el puerto opcional de Tasks (integrations.TaskTracker): guarda las Tasks
// en memoria, con hijas, creación por marca, lectura, horas con control de revisión, transición y
// resumen, y deja programar fallos por método. Se separa de Fake para que este siga sin ofrecer el
// puerto y las pruebas del núcleo puedan comprobar el caso «sin puerto».
type TaskFake struct {
	*Fake

	tmu       sync.Mutex
	tasks     map[string]*integrations.TaskItem
	order     []string
	nextID    int
	methodErr map[string]error
	conflicts int
	calls     []string
	summaries []integrations.SummaryRequest
}

var _ integrations.TaskTracker = (*TaskFake)(nil)

// NewTaskFake crea el doble con el puerto de Tasks; los identificadores nuevos empiezan en 1000.
func NewTaskFake() *TaskFake {
	return &TaskFake{
		Fake:      NewFake(),
		tasks:     map[string]*integrations.TaskItem{},
		nextID:    1000,
		methodErr: map[string]error{},
	}
}

// SeedTask añade una Task (copia). Si no trae Type es una Task; si no trae Rev, 1.
func (f *TaskFake) SeedTask(t integrations.TaskItem) *TaskFake {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	if t.Type == "" {
		t.Type = "Task"
	}
	if t.Rev == 0 {
		t.Rev = 1
	}
	t.Tags = append([]string(nil), t.Tags...)
	f.tasks[t.ID] = &t
	f.order = append(f.order, t.ID)
	return f
}

// FailMethod hace que toda llamada al método (por su nombre, p. ej. "RecordHours") falle con err.
func (f *TaskFake) FailMethod(method string, err error) *TaskFake {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	f.methodErr[method] = err
	return f
}

// ConflictNext hace que los próximos n RecordHours o SetEstimate (con replace) fallen con un
// conflicto de revisión, sin aplicar nada, aunque la revisión base coincida.
func (f *TaskFake) ConflictNext(n int) *TaskFake {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	f.conflicts = n
	return f
}

// Task devuelve una copia de la Task id y si existe.
func (f *TaskFake) Task(id string) (integrations.TaskItem, bool) {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	t, ok := f.tasks[id]
	if !ok {
		return integrations.TaskItem{}, false
	}
	return cloneTask(*t), true
}

// TaskCalls devuelve los nombres de los métodos del puerto invocados, en orden.
func (f *TaskFake) TaskCalls() []string {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	return append([]string(nil), f.calls...)
}

// Summaries devuelve las peticiones de resumen recibidas, en orden.
func (f *TaskFake) Summaries() []integrations.SummaryRequest {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	return append([]integrations.SummaryRequest(nil), f.summaries...)
}

// enter registra la llamada y devuelve el fallo programado del método, si lo hay. Se llama con
// f.tmu tomado.
func (f *TaskFake) enter(ctx context.Context, method string) error {
	f.calls = append(f.calls, method)
	if err := ctx.Err(); err != nil {
		return err
	}
	return f.methodErr[method]
}

// ChildTasks implementa integrations.TaskTracker.
func (f *TaskFake) ChildTasks(ctx context.Context, req integrations.ChildTasksRequest) ([]integrations.TaskItem, error) {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	if err := f.enter(ctx, "ChildTasks"); err != nil {
		return nil, err
	}
	var out []integrations.TaskItem
	for _, id := range f.order {
		if t := f.tasks[id]; t.ParentID == req.ParentID {
			out = append(out, cloneTask(*t))
		}
	}
	return out, nil
}

// EnsureChildTask implementa integrations.TaskTracker: reutiliza la hija que lleva la marca, sin
// distinguir mayúsculas, o crea una nueva con la revisión 1.
func (f *TaskFake) EnsureChildTask(ctx context.Context, req integrations.CreateTaskRequest) (integrations.CreateTaskResult, error) {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	if err := f.enter(ctx, "EnsureChildTask"); err != nil {
		return integrations.CreateTaskResult{}, err
	}
	var found []string
	for _, id := range f.order {
		t := f.tasks[id]
		if t.ParentID != req.ParentID {
			continue
		}
		for _, tag := range t.Tags {
			if strings.EqualFold(tag, req.Mark) {
				found = append(found, id)
				break
			}
		}
	}
	switch len(found) {
	case 0:
	case 1:
		return integrations.CreateTaskResult{Task: cloneTask(*f.tasks[found[0]]), Outcome: integrations.TaskFoundByMark}, nil
	default:
		return integrations.CreateTaskResult{}, &integrations.TaskAmbiguousError{Mark: req.Mark, IDs: found}
	}
	f.nextID++
	id := strconv.Itoa(f.nextID)
	t := &integrations.TaskItem{
		ID: id, URL: "https://tracker.invalid/items/" + id, Type: "Task", State: "To Do", StateCategory: "Proposed",
		ParentID: req.ParentID, Rev: 1, Tags: []string{strings.ToLower(req.Mark)},
	}
	if req.EstimateHours > 0 {
		t.Hours = integrations.TaskHours{Original: req.EstimateHours, Remaining: req.EstimateHours, HasOriginal: true}
	}
	f.tasks[id] = t
	f.order = append(f.order, id)
	return integrations.CreateTaskResult{Task: cloneTask(*t), Outcome: integrations.TaskCreated}, nil
}

// ReadTask implementa integrations.TaskTracker.
func (f *TaskFake) ReadTask(ctx context.Context, req integrations.TaskRequest) (integrations.TaskItem, error) {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	if err := f.enter(ctx, "ReadTask"); err != nil {
		return integrations.TaskItem{}, err
	}
	t, ok := f.tasks[req.TaskID]
	if !ok {
		return integrations.TaskItem{}, integrations.ErrTaskNotFound
	}
	return cloneTask(*t), nil
}

// ValidateTask implementa integrations.TaskTracker: existe, es una Task y es hija de ParentID, en
// ese orden.
func (f *TaskFake) ValidateTask(ctx context.Context, req integrations.TaskRequest) (integrations.TaskItem, error) {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	if err := f.enter(ctx, "ValidateTask"); err != nil {
		return integrations.TaskItem{}, err
	}
	t, ok := f.tasks[req.TaskID]
	switch {
	case !ok:
		return integrations.TaskItem{}, integrations.ErrTaskNotFound
	case t.Type != "Task":
		return integrations.TaskItem{}, integrations.ErrNotATask
	case t.ParentID != req.ParentID:
		return integrations.TaskItem{}, integrations.ErrNotChildOfWorkItem
	}
	return cloneTask(*t), nil
}

// SetEstimate implementa integrations.TaskTracker.
func (f *TaskFake) SetEstimate(ctx context.Context, req integrations.EstimateRequest) (integrations.EstimateResult, error) {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	if err := f.enter(ctx, "SetEstimate"); err != nil {
		return integrations.EstimateResult{}, err
	}
	t, ok := f.tasks[req.TaskID]
	if !ok {
		return integrations.EstimateResult{}, integrations.ErrTaskNotFound
	}
	if req.Mode == integrations.EstimateIfMissing && t.Hours.HasOriginal {
		return integrations.EstimateResult{Task: cloneTask(*t)}, nil
	}
	if req.Mode == integrations.EstimateReplace {
		if f.conflicts > 0 {
			f.conflicts--
			return integrations.EstimateResult{}, &integrations.RevConflictError{Rev: t.Rev, Hours: t.Hours}
		}
		if t.Rev != req.Base.Rev {
			return integrations.EstimateResult{}, &integrations.RevConflictError{Rev: t.Rev, Hours: t.Hours}
		}
	}
	t.Hours.Original, t.Hours.HasOriginal = req.Hours, true
	t.Hours.Remaining = max(0, req.Hours-t.Hours.Completed)
	t.Rev++
	return integrations.EstimateResult{Task: cloneTask(*t), Written: true}, nil
}

// RecordHours implementa integrations.TaskTracker: suma las horas a CompletedWork y las resta de
// RemainingWork (con piso en cero) solo si la revisión coincide con la de la lectura previa.
func (f *TaskFake) RecordHours(ctx context.Context, req integrations.HoursRequest) (integrations.HoursResult, error) {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	if err := f.enter(ctx, "RecordHours"); err != nil {
		return integrations.HoursResult{}, err
	}
	t, ok := f.tasks[req.Base.ID]
	if !ok {
		return integrations.HoursResult{}, integrations.ErrTaskNotFound
	}
	if f.conflicts > 0 {
		f.conflicts--
		return integrations.HoursResult{}, &integrations.RevConflictError{Rev: t.Rev, Hours: t.Hours}
	}
	if t.Rev != req.Base.Rev {
		return integrations.HoursResult{}, &integrations.RevConflictError{Rev: t.Rev, Hours: t.Hours}
	}
	t.Hours.Completed += req.Hours
	t.Hours.Remaining = max(0, t.Hours.Remaining-req.Hours)
	t.Rev++
	return integrations.HoursResult{Task: cloneTask(*t)}, nil
}

// TransitionTask implementa integrations.TaskTracker: lleva la Task a State (sin escritura si ya
// está) y, con Assignee, la asigna.
func (f *TaskFake) TransitionTask(ctx context.Context, req integrations.TransitionRequest) (integrations.TransitionResult, error) {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	if err := f.enter(ctx, "TransitionTask"); err != nil {
		return integrations.TransitionResult{}, err
	}
	t, ok := f.tasks[req.TaskID]
	if !ok {
		return integrations.TransitionResult{}, integrations.ErrTaskNotFound
	}
	var res integrations.TransitionResult
	if t.State != req.State {
		t.State, res.StateChanged = req.State, true
	}
	if req.Assignee != nil && t.AssignedToID != req.Assignee.ID {
		t.AssignedToID, res.Assigned = req.Assignee.ID, true
	}
	if res.StateChanged || res.Assigned {
		t.Rev++
	}
	res.Task = cloneTask(*t)
	return res, nil
}

// UpdateSummary implementa integrations.TaskTracker: registra la petición y declara la escritura.
func (f *TaskFake) UpdateSummary(ctx context.Context, req integrations.SummaryRequest) (integrations.SummaryResult, error) {
	f.tmu.Lock()
	defer f.tmu.Unlock()
	if err := f.enter(ctx, "UpdateSummary"); err != nil {
		return integrations.SummaryResult{}, err
	}
	req.Tags = append([]string(nil), req.Tags...)
	f.summaries = append(f.summaries, req)
	return integrations.SummaryResult{Wrote: true, DescriptionChanged: true, TagsSent: len(req.Tags) > 0, Added: req.Tags}, nil
}

func cloneTask(t integrations.TaskItem) integrations.TaskItem {
	t.Tags = append([]string(nil), t.Tags...)
	return t
}
