package azuredevops

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

const (
	linkKindWorkItem = "work_item"
	// actorCLI es el actor de un evento sin identidad git: nunca se asigna una Task a «cli».
	actorCLI = "cli"
)

// Deliver entrega un evento del ciclo de vida (D13, D19). Solo entrega role.start (la Task del rol
// pasa a task_state_start y se asigna a quien arranca), role.apply.approved (la Task pasa a
// task_state_apply_approved) y archive (el work item pasa a work_item_state_archive, si lo hay). Los
// demás eventos son un error permanente sin red. Nunca devuelve vínculos ni imputaciones.
func (p *Plugin) Deliver(ctx context.Context, d integrations.Delivery) (integrations.Outcome, error) {
	cfg := lifecycleConfigFrom(d.Config)
	switch d.Event.Type {
	case manifest.EventRoleStart:
		return p.deliverTaskEvent(ctx, d, cfg.TaskStateStart, integrations.AdvanceFromProposed, true)
	case manifest.EventRoleApplyApproved:
		return p.deliverTaskEvent(ctx, d, cfg.TaskStateApplyApproved, integrations.AdvanceUnlessClosed, false)
	case manifest.EventArchive:
		return p.deliverArchive(ctx, d, cfg)
	}
	return integrations.Outcome{}, integrations.Permanent(fmt.Errorf("%w: %s", ErrLifecycleUnsupported, d.Event.Type))
}

// deliverTaskEvent mueve la Task del rol del evento al estado state. withAssignee decide si se intenta
// asignar la Task al actor del evento (solo en role.start).
func (p *Plugin) deliverTaskEvent(ctx context.Context, d integrations.Delivery, state string, rule integrations.TransitionRule, withAssignee bool) (integrations.Outcome, error) {
	task, ok := taskLinkOf(d)
	if !ok {
		return integrations.Outcome{}, integrations.Permanent(taskNotBound(d.Event))
	}
	var warnings []string
	var assignee *integrations.TrackerIdentity
	if withAssignee {
		var warn string
		assignee, warn = assigneeFor(d)
		if warn != "" {
			warnings = append(warnings, warn)
		}
	}
	result, err := p.TransitionTask(ctx, integrations.TransitionRequest{
		PortRequest: integrations.PortRequest{Config: d.Config, Secrets: d.Secrets},
		TaskID:      task.ID,
		State:       state,
		Rule:        rule,
		Assignee:    assignee,
	})
	if err != nil {
		return integrations.Outcome{}, deliveryError(err)
	}
	for _, w := range result.Warnings {
		warnings = append(warnings, warningText(w))
	}
	return integrations.Outcome{Warnings: warnings}, nil
}

// taskLinkOf devuelve el vínculo de la Task del rol del evento en la instantánea de la entrega.
func taskLinkOf(d integrations.Delivery) (integrations.Link, bool) {
	for _, l := range d.Links {
		if l.Plugin == manifest.AzureDevOpsID && l.Kind == integrations.TaskLinkKind && l.Role == d.Event.Role && l.ID != "" {
			return l, true
		}
	}
	return integrations.Link{}, false
}

// taskNotBound es el error permanente de un evento de rol sin Task vinculada: nombra las salidas
// reales (D13).
func taskNotBound(ev integrations.Event) error {
	return fmt.Errorf("%w: el rol %q del cambio %q no tiene Task vinculada; obtén el vínculo con `git pull` del integrations.yaml publicado o con `axiom integrations tasks create|bind`, y reactiva la entrega con `axiom integrations sync --retry-abandoned`",
		integrations.ErrTaskNotBound, ev.Role, ev.Change)
}

// assigneeFor decide si role.start asigna la Task: el actor del evento debe ser quien entrega y debe
// haber una identidad guardada en la organización. Si no, devuelve nil y el aviso assignment_skipped.
func assigneeFor(d integrations.Delivery) (*integrations.TrackerIdentity, string) {
	skipped := func(why string) (*integrations.TrackerIdentity, string) {
		return nil, warningText(integrations.Warning{
			Code:    "assignment_skipped",
			Message: why + "; la Task no se asigna (guarda tu identidad con `axiom integrations connect --plugin azure-devops`)",
		})
	}
	actor := strings.TrimSpace(d.Event.Actor)
	switch {
	case actor == "" || actor == actorCLI:
		return skipped("el evento no lleva una identidad git como actor")
	case actor != strings.TrimSpace(d.Deliverer):
		return skipped("el actor del evento no es quien entrega")
	case d.Identities == nil:
		return skipped("no hay identidad guardada")
	}
	target, err := TargetFromConfig(d.Config)
	if err != nil {
		return skipped("la configuración no identifica la organización")
	}
	identity, ok, err := d.Identities.Lookup(manifest.AzureDevOpsID, target.Organization)
	if err != nil || !ok || identity.ID == "" || identity.DisplayName == "" {
		return skipped("no hay identidad guardada para la organización " + target.Organization)
	}
	return &identity, ""
}

// warningText da a un aviso tipado la forma «<código>: <mensaje>» de Outcome.Warnings.
func warningText(w integrations.Warning) string { return w.Code + ": " + w.Message }

// deliverArchive transiciona el work item de origen al archivar (D19).
func (p *Plugin) deliverArchive(ctx context.Context, d integrations.Delivery, cfg lifecycleConfig) (integrations.Outcome, error) {
	if cfg.WorkItemStateArchive == "" {
		return integrations.Outcome{}, nil
	}
	link, ok := workItemLinkOf(d)
	if !ok {
		return integrations.Outcome{Warnings: []string{warningText(integrations.Warning{
			Code:    "work_item_not_linked",
			Message: "el cambio no tiene un work item vinculado; no hay nada que transicionar",
		})}}, nil
	}
	client, err := p.NewClient(d.Config, d.Secrets)
	if err != nil {
		return integrations.Outcome{}, err
	}
	id, err := strconv.Atoi(strings.TrimSpace(link.ID))
	if err != nil || id <= 0 {
		return integrations.Outcome{}, integrations.Permanent(invalidWorkItemID(id))
	}
	wi, err := client.GetWorkItem(ctx, id)
	if err != nil {
		return integrations.Outcome{}, deliveryError(err)
	}
	itemType := fieldString(wi.Fields, fieldType)
	states, err := client.TypeStates(ctx, itemType)
	if err != nil {
		return integrations.Outcome{}, err
	}
	target, ok := findState(states, cfg.WorkItemStateArchive)
	if !ok {
		return integrations.Outcome{}, integrations.Permanent(
			&integrations.StateUnknownError{State: cfg.WorkItemStateArchive, Valid: stateNames(states)})
	}
	current := fieldString(wi.Fields, fieldState)
	if strings.EqualFold(current, target.Name) {
		return integrations.Outcome{}, nil
	}
	if category := stateCategory(states, current); category == categoryCompleted || category == categoryRemoved {
		return integrations.Outcome{Warnings: []string{warningText(integrations.Warning{
			Code:    "work_item_state_kept",
			Message: fmt.Sprintf("el work item %d sigue en %q (cerrado) y no pasa a %q", id, current, target.Name),
		})}}, nil
	}
	_, _, err = client.patchHours(ctx, id, wi.Rev, []PatchOp{{Op: "add", Path: "/fields/" + fieldState, Value: target.Name}},
		func(got WorkItem) bool { return strings.EqualFold(fieldString(got.Fields, fieldState), target.Name) })
	if err != nil {
		return integrations.Outcome{}, deliveryError(err)
	}
	return integrations.Outcome{}, nil
}

func workItemLinkOf(d integrations.Delivery) (integrations.Link, bool) {
	for _, l := range d.Links {
		if l.Plugin == manifest.AzureDevOpsID && l.Kind == linkKindWorkItem && l.ID != "" {
			return l, true
		}
	}
	return integrations.Link{}, false
}

// deliveryError clasifica el fallo de una entrega: lo que reintentar no arregla (estado inexistente,
// elemento inexistente, rechazo de reglas del proceso) es permanente y queda abandonado en el primer
// intento; un conflicto de /rev, un fallo de red, un 5xx, el límite de tasa o la autenticación se
// devuelven tal cual y sync los reintenta.
func deliveryError(err error) error {
	switch {
	case errors.Is(err, integrations.ErrStateUnknown), errors.Is(err, integrations.ErrTaskNotFound),
		errors.Is(err, ErrNotFound):
		return integrations.Permanent(err)
	case errors.Is(err, integrations.ErrTrackerRejected) && !errors.Is(err, ErrAuth):
		return integrations.Permanent(err)
	}
	return err
}
