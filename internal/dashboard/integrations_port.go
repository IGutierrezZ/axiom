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
