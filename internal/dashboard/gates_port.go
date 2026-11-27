package dashboard

import (
	"context"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
)

// GatesGateway es la frontera con el núcleo de puertas (C-06-f, D7): el dashboard no registra
// decisiones ni ejecuta git; consulta la parada que proyecta `axiom sdd status`, registra con el
// mismo núcleo que `axiom sdd gate record` y publica con el mismo que `axiom sdd publish`
// (cli.GatesGateway en producción).
type GatesGateway interface {
	// Stop devuelve la parada del cambio (o su ausencia) con la publicación y el último registro.
	Stop(ctx context.Context, cwd, change string) (cli.GateStop, error)
	// Record registra la decisión de una puerta. El actor nunca viaja: lo resuelve el núcleo.
	Record(ctx context.Context, in cli.GateRecordInput) (cli.GateRecordOutcome, error)
	// Publish publica solo los documentos del cambio en su rama atada.
	Publish(ctx context.Context, req cli.ChangePublishRequest) (cli.ChangePublishResult, error)
}

// SetGatesGateway inyecta la pasarela de puertas. Sin ella, las rutas de puertas responden 503.
func (s *Service) SetGatesGateway(g GatesGateway) {
	s.gatesMu.Lock()
	defer s.gatesMu.Unlock()
	s.gatesGW = g
}

func (s *Service) gatesGateway() GatesGateway {
	s.gatesMu.Lock()
	defer s.gatesMu.Unlock()
	return s.gatesGW
}
