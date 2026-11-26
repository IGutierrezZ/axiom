package dashboard

import (
	"context"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

// IntegrationsGateway es la frontera del dashboard con las integraciones (D-04): probar la
// conexión, descubrir el proceso del tracker y saber si el alta desde un work item aplica. El
// dashboard no llama a ningún tracker por su cuenta; la implementa cli.IntegrationsGateway en
// producción y un doble en las pruebas.
type IntegrationsGateway interface {
	Connect(ctx context.Context, cwd, pluginID, user string) (cli.IntegrationConnectOutcome, error)
	Discover(ctx context.Context, cwd, pluginID string) (integrations.Discovery, error)
	WorkItemApplicability(cwd string) cli.WorkItemApplicability

	// Alta desde un work item (D-08): fase 1, staging pendiente, fase 2 y descarte. Los rechazos
	// tipados salen como *cli.WorkItemRefusal.
	StartWorkItem(ctx context.Context, cwd, ref string) (cli.WorkItemStartOutcome, error)
	WorkItemStaged(cwd, change string) (cli.WorkItemStagedOutcome, error)
	AdoptWorkItem(ctx context.Context, cwd, change string) (cli.WorkItemAdoptOutcome, error)
	DiscardWorkItem(cwd, change string) (cli.WorkItemDiscardOutcome, error)
}

// SetIntegrationsGateway inyecta la pasarela de integraciones. Sin ella, `connect` y `discover`
// responden 503 `integrations_gateway_missing` y `GET /api/integrations` se degrada (sin work items).
func (s *Service) SetIntegrationsGateway(g IntegrationsGateway) {
	s.integrationsMu.Lock()
	defer s.integrationsMu.Unlock()
	s.integrationsGW = g
}

func (s *Service) integrationsGateway() IntegrationsGateway {
	s.integrationsMu.Lock()
	defer s.integrationsMu.Unlock()
	return s.integrationsGW
}
