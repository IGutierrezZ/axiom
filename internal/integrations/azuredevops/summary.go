package azuredevops

import (
	"context"
	"errors"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

// errPendingSummary marca el resumen que implementa la unidad siguiente del cambio.
var errPendingSummary = errors.New("el resumen de work items todavía no está disponible en este adaptador")

// UpdateSummary implementa integrations.TaskTracker. La escritura de la descripción y las etiquetas
// llega en la unidad siguiente del cambio; hasta entonces el puerto rechaza la llamada sin red.
func (p *Plugin) UpdateSummary(context.Context, integrations.SummaryRequest) (integrations.SummaryResult, error) {
	return integrations.SummaryResult{}, errPendingSummary
}
