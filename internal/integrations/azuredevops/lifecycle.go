package azuredevops

import (
	"context"
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

// Valores por defecto de las claves del ciclo de vida: los mismos que declara el manifiesto.
const (
	defaultTaskStateStart         = "Active"
	defaultTaskStateApplyApproved = "Tech Review"
	defaultReqTagPrefix           = "req:"
	defaultReqTagMax              = 50
	defaultWorkdayHours           = 8
)

// defaultWorkdays son los días laborables por defecto.
var defaultWorkdays = []string{"mon", "tue", "wed", "thu", "fri"}

// lifecycleConfig son las claves del ciclo de vida de integrations.azure-devops con tipo y valor por
// defecto aplicados. TaskStateClose es el efectivo: sin valor propio, el de TaskStateApplyApproved.
// WorkItemStateArchive vacío significa que el cierre no transiciona el work item.
type lifecycleConfig struct {
	TaskStateStart         string
	TaskStateApplyApproved string
	TaskStateClose         string
	WorkItemStateArchive   string
	ReqTagPrefix           string
	ReqTagMax              int
	WorkdayHours           int
	Workdays               []string
}

// lifecycleConfigFrom lee las claves del ciclo de vida de la configuración del plugin. La carga de
// axiom.yaml ya validó rangos y formatos y aplicó los valores por defecto del manifiesto; esta lectura
// repite los valores por defecto para quien construye la configuración a mano.
func lifecycleConfigFrom(cfg map[string]any) lifecycleConfig {
	out := lifecycleConfig{
		TaskStateStart:         stringValue(cfg, "task_state_start", defaultTaskStateStart),
		TaskStateApplyApproved: stringValue(cfg, "task_state_apply_approved", defaultTaskStateApplyApproved),
		WorkItemStateArchive:   stringValue(cfg, "work_item_state_archive", ""),
		ReqTagPrefix:           stringValue(cfg, "req_tag_prefix", defaultReqTagPrefix),
		ReqTagMax:              intValue(cfg, "req_tag_max", defaultReqTagMax),
		WorkdayHours:           intValue(cfg, "workday_hours", defaultWorkdayHours),
		Workdays:               listValue(cfg, "workdays", defaultWorkdays),
	}
	out.TaskStateClose = stringValue(cfg, "task_state_close", out.TaskStateApplyApproved)
	return out
}

func stringValue(cfg map[string]any, key, fallback string) string {
	if s, ok := cfg[key].(string); ok && s != "" {
		return s
	}
	return fallback
}

func intValue(cfg map[string]any, key string, fallback int) int {
	switch n := cfg[key].(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return fallback
}

func listValue(cfg map[string]any, key string, fallback []string) []string {
	switch l := cfg[key].(type) {
	case []string:
		if len(l) > 0 {
			return append([]string(nil), l...)
		}
	case []any:
		out := make([]string, 0, len(l))
		for _, v := range l {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return append([]string(nil), fallback...)
}

// Categorías de estado de Azure DevOps que intervienen en las reglas de avance.
const (
	categoryProposed  = "Proposed"
	categoryCompleted = "Completed"
)

// TransitionTask implementa integrations.TaskTracker (D13): mueve la Task al estado pedido según la
// regla de avance, sin retrocesos, y asigna la Task a req.Assignee cuando procede. La asignación es
// independiente de la regla de estado: una Task ya en el destino o retenida por task_state_kept, sin
// asignar, recibe un PATCH solo con System.AssignedTo. Nunca lee ni emite el uniqueName.
func (p *Plugin) TransitionTask(ctx context.Context, req integrations.TransitionRequest) (integrations.TransitionResult, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.TransitionResult{}, err
	}
	id, err := parseTaskID(req.TaskID)
	if err != nil {
		return integrations.TransitionResult{}, err
	}
	wi, err := client.GetWorkItem(ctx, id)
	if err != nil {
		return integrations.TransitionResult{}, mapReadError(err)
	}
	states, err := client.TypeStates(ctx, itemTypeTask)
	if err != nil {
		return integrations.TransitionResult{}, err
	}
	target, ok := findState(states, req.State)
	if !ok {
		return integrations.TransitionResult{}, &integrations.StateUnknownError{State: req.State, Valid: stateNames(states)}
	}

	var result integrations.TransitionResult
	var ops []PatchOp
	current := fieldString(wi.Fields, fieldState)
	switch move, warn := transitionDecision(req.Rule, current, stateCategory(states, current), target); {
	case warn != nil:
		result.Warnings = append(result.Warnings, *warn)
	case move:
		ops = append(ops, PatchOp{Op: "add", Path: "/fields/" + fieldState, Value: target.Name})
		result.StateChanged = true
	}

	assignOp, warns := client.assignmentOp(ctx, wi, req.Assignee)
	result.Warnings = append(result.Warnings, warns...)
	if assignOp != nil {
		ops = append(ops, *assignOp)
	}

	if len(ops) == 0 {
		result.Task = client.taskItem(wi, categoriesOf(states))
		return result, nil
	}
	item, _, err := client.patchHours(ctx, id, wi.Rev, ops, func(got WorkItem) bool {
		if result.StateChanged && !strings.EqualFold(fieldString(got.Fields, fieldState), target.Name) {
			return false
		}
		return assignOp == nil || assignedID(got.Fields) == req.Assignee.ID
	})
	if err != nil {
		return integrations.TransitionResult{}, err
	}
	result.Task = client.taskItem(item, categoriesOf(states))
	if result.Task.ParentID == "" {
		result.Task.ParentID = client.parentOf(wi)
	}
	if assignOp != nil {
		if result.Task.AssignedToID == req.Assignee.ID {
			result.Assigned = true
		} else {
			result.Warnings = append(result.Warnings, integrations.Warning{
				Code:    "assignment_mismatch",
				Message: "el tracker asignó la Task a una identidad distinta de la guardada; revisa la asignación en Azure DevOps",
			})
		}
	}
	return result, nil
}

func findState(states []TypeState, name string) (TypeState, bool) {
	for _, s := range states {
		if strings.EqualFold(s.Name, strings.TrimSpace(name)) {
			return s, true
		}
	}
	return TypeState{}, false
}

func stateNames(states []TypeState) []string {
	names := make([]string, 0, len(states))
	for _, s := range states {
		names = append(names, s.Name)
	}
	return names
}

func stateCategory(states []TypeState, name string) string {
	if s, ok := findState(states, name); ok {
		return s.Category
	}
	return ""
}

func categoriesOf(states []TypeState) map[string]string {
	cats := make(map[string]string, len(states))
	for _, s := range states {
		cats[strings.ToLower(s.Name)] = s.Category
	}
	return cats
}

// transitionDecision aplica la regla de avance (D13). move indica si hay que escribir el estado; un
// aviso task_state_kept indica que la Task se conserva en su estado porque no se retrocede ni se
// reabre. En el estado destino no se escribe y no se avisa.
func transitionDecision(rule integrations.TransitionRule, current, category string, target TypeState) (bool, *integrations.Warning) {
	if strings.EqualFold(current, target.Name) {
		return false, nil
	}
	kept := func(why string) (bool, *integrations.Warning) {
		return false, &integrations.Warning{
			Code:    "task_state_kept",
			Message: fmt.Sprintf("la Task sigue en %q (%s) y no pasa a %q", current, why, target.Name),
		}
	}
	switch rule {
	case integrations.AdvanceFromProposed:
		if category != categoryProposed {
			return kept("ya no está en la categoría Proposed")
		}
	case integrations.AdvanceUnlessClosed:
		if category == categoryCompleted || category == categoryRemoved {
			return kept("está cerrada")
		}
	}
	return true, nil
}

// assignmentOp decide la asignación de la Task: devuelve la operación de System.AssignedTo cuando
// procede y los avisos de lo que se omite. assignee nil significa no asignar. Solo se asigna a una
// Task sin asignar y si el equipo no tiene dos miembros con el nombre visible guardado; una Task ya
// asignada a la misma persona no se toca y una asignada a otra no se reasigna.
func (c *Client) assignmentOp(ctx context.Context, wi WorkItem, assignee *integrations.TrackerIdentity) (*PatchOp, []integrations.Warning) {
	if assignee == nil {
		return nil, nil
	}
	switch owner := assignedID(wi.Fields); {
	case owner != "" && strings.EqualFold(owner, assignee.ID):
		return nil, nil
	case owner != "":
		return nil, []integrations.Warning{{
			Code:    "task_assigned_to_other",
			Message: "la Task ya está asignada a otra persona y no se reasigna",
		}}
	}
	members, err := c.listMembers(ctx)
	if err != nil {
		return nil, []integrations.Warning{{
			Code:    "assignment_skipped",
			Message: "no se pudieron leer los miembros del equipo para comprobar homónimos; la Task no se asigna",
		}}
	}
	same := 0
	for _, m := range members {
		if strings.EqualFold(m.DisplayName, assignee.DisplayName) {
			same++
		}
	}
	if same > 1 {
		return nil, []integrations.Warning{{
			Code:    "assignment_ambiguous",
			Message: fmt.Sprintf("hay %d miembros del equipo con el nombre visible %q; la Task no se asigna", same, assignee.DisplayName),
		}}
	}
	return &PatchOp{Op: "add", Path: "/fields/" + fieldAssignedTo, Value: assignee.DisplayName}, nil
}
