package integrations

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// PortTasks es el puerto opcional que crea, lee y actualiza las Tasks hijas del work item de origen
// de un cambio (C-07-e). Como el resto de puertos opcionales, se detecta por aserción de tipo: un
// plugin sin él sigue funcionando y WithTaskTracker devuelve *PortUnsupportedError sin red.
const PortTasks = "tasks"

// TaskHours son las horas de una Task; un campo ausente en el elemento se lee como 0.
type TaskHours struct {
	Original, Remaining, Completed float64
	// HasOriginal indica que la Task trae una estimación original (distinta de «ausente»).
	HasOriginal bool
}

// TaskItem es la lectura neutral de una Task. AssignedToID es solo el id del tracker: nunca el
// correo ni el nombre de usuario.
type TaskItem struct {
	ID, URL, Type, State, StateCategory, ParentID string
	AssignedToID                                  string
	Rev                                           int
	Tags                                          []string
	Hours                                         TaskHours
}

// TaskOutcome es cómo se obtuvo la Task de un rol.
type TaskOutcome string

// Resultados de EnsureChildTask.
const (
	TaskCreated     TaskOutcome = "created"
	TaskFoundByMark TaskOutcome = "found_by_mark"
	TaskRepaired    TaskOutcome = "repaired"
)

// Warning es un aviso tipado de un puerto: no detiene la operación.
type Warning struct{ Code, Message, Continuation string }

// ChildTasksRequest pide las Tasks hijas de ParentID.
type ChildTasksRequest struct {
	PortRequest
	ParentID string
}

// CreateTaskRequest pide encontrar o crear la Task identificada por Mark bajo ParentID.
type CreateTaskRequest struct {
	PortRequest
	ParentID, Title, Mark string
	EstimateHours         float64 // 0: sin estimación
}

// CreateTaskResult es el resultado de EnsureChildTask.
type CreateTaskResult struct {
	Task     TaskItem
	Outcome  TaskOutcome
	Warnings []Warning
}

// TaskRequest pide leer o validar una Task. ParentID solo lo usa ValidateTask.
type TaskRequest struct {
	PortRequest
	TaskID, ParentID string
}

// EstimateMode es la regla con la que SetEstimate escribe la estimación.
type EstimateMode string

// Reglas de estimación.
const (
	EstimateIfMissing EstimateMode = "if_missing" // bind: solo si la Task no tiene estimación original
	EstimateReplace   EstimateMode = "replace"    // create con una estimación distinta
)

// EstimateRequest pide fijar la estimación de una Task.
type EstimateRequest struct {
	PortRequest
	TaskID string
	Hours  float64
	Mode   EstimateMode
	Base   TaskItem // lectura previa (Rev y Completed); obligatoria con EstimateReplace
}

// EstimateResult es el resultado de SetEstimate.
type EstimateResult struct {
	Task      TaskItem
	Written   bool // false en if_missing si la Task ya tenía estimación original
	Recovered bool // respuesta perdida verificada
}

// HoursRequest pide imputar Hours a una Task a partir de la lectura previa Base.
type HoursRequest struct {
	PortRequest
	Base  TaskItem // lectura previa: Rev y Hours
	Hours float64
}

// HoursResult es el resultado de RecordHours.
type HoursResult struct {
	Task      TaskItem
	Recovered bool // respuesta perdida verificada
}

// TransitionRule es la regla de avance de estado de una Task.
type TransitionRule string

// Reglas de transición.
const (
	AdvanceFromProposed TransitionRule = "advance_from_proposed" // role.start
	AdvanceUnlessClosed TransitionRule = "advance_unless_closed" // apply aprobado y cierre
)

// TransitionRequest pide llevar la Task TaskID al estado State bajo Rule.
type TransitionRequest struct {
	PortRequest
	TaskID, State string
	Rule          TransitionRule
	// Assignee es la identidad a la que asignar; nil = no asignar. Deliver de role.start la pasa solo
	// si el actor del evento es quien entrega y hay identidad guardada.
	Assignee *TrackerIdentity
}

// TransitionResult es el resultado de TransitionTask.
type TransitionResult struct {
	Task         TaskItem
	StateChanged bool
	Assigned     bool
	// Warnings: task_state_kept, assignment_ambiguous, assignment_mismatch, task_assigned_to_other.
	Warnings []Warning
}

// SummaryRequest pide actualizar la descripción y las etiquetas de un elemento.
type SummaryRequest struct {
	PortRequest
	ItemID, HTML, TagPrefix string
	Tags                    []string // IDs ya con prefijo, en orden estable
	TagMax                  int
}

// SummaryResult es el resultado de UpdateSummary.
type SummaryResult struct {
	Wrote, DescriptionChanged, TagsSent bool
	Added, Removed                      []string
	Warnings                            []Warning
}

// TaskTracker es el puerto opcional de Tasks.
type TaskTracker interface {
	ChildTasks(ctx context.Context, req ChildTasksRequest) ([]TaskItem, error)
	EnsureChildTask(ctx context.Context, req CreateTaskRequest) (CreateTaskResult, error)
	ReadTask(ctx context.Context, req TaskRequest) (TaskItem, error)
	ValidateTask(ctx context.Context, req TaskRequest) (TaskItem, error)
	SetEstimate(ctx context.Context, req EstimateRequest) (EstimateResult, error)
	RecordHours(ctx context.Context, req HoursRequest) (HoursResult, error)
	TransitionTask(ctx context.Context, req TransitionRequest) (TransitionResult, error)
	UpdateSummary(ctx context.Context, req SummaryRequest) (SummaryResult, error)
}

// WithTaskTracker detecta el puerto de p por aserción de tipo antes de resolver ningún secreto: sin
// él devuelve *PortUnsupportedError{Port: PortTasks} sin red. Con él compone la PortRequest con
// secretos acotados y rastreados, y ejecuta call bajo invokePort (recupera un pánico y sanea el texto
// del error; la causa se conserva en Unwrap).
func WithTaskTracker[T any](p Plugin, cfg workspace.IntegrationPluginConfig, store secrets.Store,
	call func(t TaskTracker, req PortRequest) (T, error)) (T, error) {
	m := p.Manifest()
	t, ok := p.(TaskTracker)
	if !ok {
		var zero T
		return zero, &PortUnsupportedError{Plugin: m.ID, Port: PortTasks}
	}
	tracker := newSecretTracker()
	req := portRequest(p, cfg, store, tracker)
	return invokePort(tracker, m.ID, PortTasks, func() (T, error) {
		return call(t, req)
	})
}

// Errores neutrales del puerto de Tasks. Las cadenas son códigos y no frases: el texto que lee la
// persona lo compone la CLI con su continuación, de modo que el núcleo no declara rechazos.
var (
	ErrTaskNotFound         = errors.New("task_not_found")
	ErrNotATask             = errors.New("not_a_task")
	ErrNotChildOfWorkItem   = errors.New("not_child_of_work_item")
	ErrTaskAmbiguous        = errors.New("task_ambiguous")
	ErrTaskParentLinkFailed = errors.New("task_parent_link_failed")
	ErrFieldMissing         = errors.New("field_missing")
	ErrStateUnknown         = errors.New("state_unknown")
	ErrRevConflict          = errors.New("rev_conflict")
	ErrTrackerRejected      = errors.New("tracker_rejected")
	ErrTaskNotBound         = errors.New("task_not_bound")
)

// TaskAmbiguousError indica que más de una Task lleva la marca Mark. errors.Is(err, ErrTaskAmbiguous).
type TaskAmbiguousError struct {
	Mark string
	IDs  []string
}

func (e *TaskAmbiguousError) Error() string {
	return fmt.Sprintf("%s: la marca %s la llevan varias Tasks (%s)", ErrTaskAmbiguous.Error(), e.Mark, strings.Join(e.IDs, ", "))
}

func (e *TaskAmbiguousError) Is(target error) bool { return target == ErrTaskAmbiguous }

// FieldMissingError indica que el proceso del tracker no define el campo Field.
// errors.Is(err, ErrFieldMissing).
type FieldMissingError struct{ Field string }

func (e *FieldMissingError) Error() string {
	return fmt.Sprintf("%s: el proceso no define el campo %s", ErrFieldMissing.Error(), e.Field)
}

func (e *FieldMissingError) Is(target error) bool { return target == ErrFieldMissing }

// StateUnknownError indica que State no existe en el tipo y lista los estados válidos.
// errors.Is(err, ErrStateUnknown).
type StateUnknownError struct {
	State string
	Valid []string
}

func (e *StateUnknownError) Error() string {
	return fmt.Sprintf("%s: el estado %q no existe; válidos: %s", ErrStateUnknown.Error(), e.State, strings.Join(e.Valid, ", "))
}

func (e *StateUnknownError) Is(target error) bool { return target == ErrStateUnknown }

// RevConflictError indica que la Task cambió desde la lectura previa; Rev y Hours son los valores
// releídos. AfterRetry es true cuando la petición se reenvió tras un fallo de red y la relectura no
// verificó que se hubiera aplicado: el resultado es incierto, no un conflicto firme del primer intento.
// errors.Is(err, ErrRevConflict).
type RevConflictError struct {
	Rev        int
	Hours      TaskHours
	AfterRetry bool
}

func (e *RevConflictError) Error() string {
	return fmt.Sprintf("%s: la Task está en la revisión %d", ErrRevConflict.Error(), e.Rev)
}

func (e *RevConflictError) Is(target error) bool { return target == ErrRevConflict }
