package azuredevops

import (
	"context"
	"errors"

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

// errPendingTransition marca la transición de estados que implementa la unidad siguiente del cambio.
var errPendingTransition = errors.New("la transición de Tasks todavía no está disponible en este adaptador")

// TransitionTask implementa integrations.TaskTracker. La transición de estados y la asignación llegan
// en la unidad siguiente del cambio; hasta entonces el puerto rechaza la llamada sin red.
func (p *Plugin) TransitionTask(context.Context, integrations.TransitionRequest) (integrations.TransitionResult, error) {
	return integrations.TransitionResult{}, errPendingTransition
}
