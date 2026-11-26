package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/changestart"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

// Métodos de work item de la pasarela (D-04, D-08): las fases 1 y 2 y el descarte de
// `axiom change start`, expuestos al dashboard sin duplicar su lógica. Los rechazos salen como
// *WorkItemRefusal con el mismo texto y continuación que la CLI (changeStartRefusalText); nunca llevan
// texto del work item (dato de terceros), solo los campos de confianza de la lista blanca.

// workItemsNotApplicableCode es el código del rechazo que solo existe en la pasarela: la aplicabilidad
// se perdió entre la comprobación del dashboard y la llamada (el plugin o el PAT desaparecieron).
const workItemsNotApplicableCode = "work_items_not_applicable"

// WorkItemStartOutcome es el resultado de la fase 1.
type WorkItemStartOutcome struct {
	Change, ExternalRef string
	Status              string // staged | already_staged | already_adopted
	Warnings            []string
}

// WorkItemStagedOutcome es el staging pendiente de un cambio (reintento sin repetir la fase 1).
type WorkItemStagedOutcome struct {
	Change, ExternalRef string
}

// WorkItemAdoptOutcome es el resultado de la fase 2.
type WorkItemAdoptOutcome struct {
	Change, Status string // adopted | already_adopted
	Warnings       []string
}

// WorkItemDiscardOutcome es el resultado del descarte del staging.
type WorkItemDiscardOutcome struct {
	Change, Status string // discarded | nothing_to_discard
	Warnings       []string
}

// WorkItemRefusal es un rechazo tipado de las fases del alta desde un work item. Fields lleva solo
// datos de confianza (motivo, tipo, etiqueta, cambio y carril).
type WorkItemRefusal struct {
	Code, Message, Continuation string
	Fields                      map[string]string
}

func (r *WorkItemRefusal) Error() string { return r.Message }

// workItemFieldAllowlist son los campos de un rechazo que salen de la pasarela; el resto (p. ej. el
// detalle de una lectura fallida) se queda en el mensaje ya compuesto.
var workItemFieldAllowlist = []string{"reason", "type", "tag", "change", "lane", "record_lane"}

// workItemContext resuelve el plugin aplicable y el entorno de las operaciones desde cwd.
func workItemContext(cwd string) (changestart.Env, *workItemTracker, *WorkItemRefusal) {
	loc, app := resolveWorkItemTracker(cwd)
	if app.tracker == nil {
		return changestart.Env{}, nil, &WorkItemRefusal{
			Code:         workItemsNotApplicableCode,
			Message:      "El alta desde un work item no aplica: " + workItemReason(app),
			Continuation: "Activa un plugin de work items en la pantalla de integraciones y vuelve a intentarlo",
		}
	}
	gitRoot, common, err := changestart.ResolveRepository(loc.WorkspaceRoot)
	if err != nil {
		return changestart.Env{}, app.tracker, workItemRefusal("git_repository_required", nil, app.tracker.PluginID())
	}
	return changestart.Env{WorkspaceRoot: loc.WorkspaceRoot, PlanningHome: loc.PlanningHome, GitRoot: gitRoot, CommonDir: common}, app.tracker, nil
}

// workItemRefusal compone el rechazo con el texto de la CLI y la lista blanca de campos.
func workItemRefusal(code string, fields map[string]string, plugin string) *WorkItemRefusal {
	usage := ""
	if code == changestart.CodeUsageInvalid {
		usage = "el nombre del cambio no es válido"
	}
	message, continuation := changeStartRefusalText(code, fields, plugin, usage)
	out := &WorkItemRefusal{Code: code, Message: message, Continuation: continuation, Fields: map[string]string{}}
	for _, key := range workItemFieldAllowlist {
		if v := fields[key]; v != "" {
			out.Fields[key] = v
		}
	}
	return out
}

// workItemError traduce el error de una operación de changestart: un rechazo tipado, el puerto no
// admitido o, en cualquier otro caso, un fallo de escritura. Un error de contexto se devuelve tal cual
// para que el dashboard lo distinga como plazo agotado.
func workItemError(err error, change, plugin string) error {
	var refusal *changestart.Refusal
	switch {
	case errors.As(err, &refusal):
		fields := map[string]string{}
		for k, v := range refusal.Fields {
			fields[k] = v
		}
		if fields["change"] == "" && change != "" {
			fields["change"] = change
		}
		return workItemRefusal(refusal.Code, fields, plugin)
	case errors.Is(err, integrations.ErrPortUnsupported):
		return workItemRefusal("port_unsupported", nil, plugin)
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	}
	return workItemRefusal(changestart.CodeWriteFailed, map[string]string{"change": change}, plugin)
}

// checkChangeName rechaza un nombre de cambio que no sea un componente de rama válido.
func checkChangeName(change, plugin string) *WorkItemRefusal {
	if err := changestart.ValidChangeName(change); err != nil {
		return workItemRefusal(changestart.CodeUsageInvalid, nil, plugin)
	}
	return nil
}

// StartWorkItem ejecuta la fase 1: lee el work item y deja el staging. Es idempotente.
func (*IntegrationsGateway) StartWorkItem(ctx context.Context, cwd, ref string) (WorkItemStartOutcome, error) {
	env, tracker, refusal := workItemContext(cwd)
	if refusal != nil {
		return WorkItemStartOutcome{}, refusal
	}
	res, err := changestart.Start(ctx, env, tracker, ref)
	if err != nil {
		return WorkItemStartOutcome{}, workItemError(err, "", tracker.PluginID())
	}
	return WorkItemStartOutcome{Change: res.Change, ExternalRef: res.ExternalRef, Status: res.Status, Warnings: res.Warnings}, nil
}

// WorkItemStaged dice si el cambio tiene un staging de la fase 1 pendiente; sin él, responde
// staging_not_found sin crear nada.
func (*IntegrationsGateway) WorkItemStaged(cwd, change string) (WorkItemStagedOutcome, error) {
	env, tracker, refusal := workItemContext(cwd)
	if refusal != nil {
		return WorkItemStagedOutcome{}, refusal
	}
	if bad := checkChangeName(change, tracker.PluginID()); bad != nil {
		return WorkItemStagedOutcome{}, bad
	}
	origin, ok := changestart.WorkItemFacts(env.CommonDir, change, "")
	if !ok {
		return WorkItemStagedOutcome{}, workItemRefusal(changestart.CodeStagingNotFound, map[string]string{"change": change}, tracker.PluginID())
	}
	return WorkItemStagedOutcome{Change: change, ExternalRef: tracker.RefAlias() + ":" + origin.ID}, nil
}

// AdoptWorkItem ejecuta la fase 2 en el carril SDD desde el checkout de cwd (en modo worktree, el
// del cambio). Es reintentable e idempotente.
func (*IntegrationsGateway) AdoptWorkItem(ctx context.Context, cwd, change string) (WorkItemAdoptOutcome, error) {
	env, tracker, refusal := workItemContext(cwd)
	if refusal != nil {
		return WorkItemAdoptOutcome{}, refusal
	}
	if bad := checkChangeName(change, tracker.PluginID()); bad != nil {
		return WorkItemAdoptOutcome{}, bad
	}
	res, err := changestart.Adopt(ctx, env, tracker, change, changestart.LaneSDD)
	if err != nil {
		return WorkItemAdoptOutcome{}, workItemError(err, change, tracker.PluginID())
	}
	return WorkItemAdoptOutcome{Change: res.Change, Status: res.Status, Warnings: res.Warnings}, nil
}

// DiscardWorkItem retira solo el staging del cambio, como `axiom change start --discard`. Sin red.
func (*IntegrationsGateway) DiscardWorkItem(cwd, change string) (WorkItemDiscardOutcome, error) {
	env, tracker, refusal := workItemContext(cwd)
	if refusal != nil {
		return WorkItemDiscardOutcome{}, refusal
	}
	if bad := checkChangeName(change, tracker.PluginID()); bad != nil {
		return WorkItemDiscardOutcome{}, bad
	}
	res, err := changestart.Discard(env, change)
	if err != nil {
		return WorkItemDiscardOutcome{}, workItemError(err, change, tracker.PluginID())
	}
	return WorkItemDiscardOutcome{Change: res.Change, Status: res.Status, Warnings: res.Warnings}, nil
}

// workItemReason explica en castellano por qué no hay plugin de work items aplicable.
func workItemReason(app workItemApplicability) string {
	switch {
	case len(app.warnings) > 0:
		return strings.Join(app.warnings, "; ")
	case app.portUnsupported:
		return "el plugin activo no admite work items: elige un plugin que los admita en la pantalla de integraciones"
	}
	return "no hay ningún plugin de integración activo: configúralo en la pantalla de integraciones"
}
