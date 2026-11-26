package cli

import (
	"context"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/identitystore"
)

// IntegrationsGateway es la pasarela que el dashboard usa para probar la conexión, descubrir el
// proceso del tracker y saber si el alta desde un work item aplica (D-04). Envuelve las funciones
// de `axiom integrations connect|discover` y `axiom change start` sin moverlas ni duplicar su
// lógica; los errores del plugin ya llegan saneados (nunca llevan el PAT).
type IntegrationsGateway struct{}

// NewIntegrationsGateway devuelve la pasarela. No guarda estado.
func NewIntegrationsGateway() *IntegrationsGateway { return &IntegrationsGateway{} }

// IntegrationConnectOutcome es el resultado de una conexión correcta.
type IntegrationConnectOutcome struct {
	Identity   integrations.TrackerIdentity
	ResolvedAt string
	// Replaced es la identidad previa de la organización cuando la conexión la sustituye.
	Replaced *identitystore.Entry
}

// WorkItemApplicability dice si el alta desde un work item aplica y, si no, por qué.
type WorkItemApplicability struct {
	Applicable bool
	Plugin     string // id del plugin que aporta el puerto de work items
	RefAlias   string
	Reason     string // por qué no aplica, en castellano
}

// Connect resuelve la identidad por el puerto del plugin y la guarda. Un
// *integrations.IdentitySelectionError pide elegir usuario entre los miembros del equipo.
func (*IntegrationsGateway) Connect(ctx context.Context, cwd, pluginID, user string) (IntegrationConnectOutcome, error) {
	result, err := connectIntegration(ctx, cwd, pluginID, strings.TrimSpace(user))
	if err != nil {
		return IntegrationConnectOutcome{}, err
	}
	return IntegrationConnectOutcome{Identity: result.Identity, ResolvedAt: result.ResolvedAt, Replaced: result.Replaced}, nil
}

// Discover devuelve el proceso del tracker. No escribe ningún fichero.
func (*IntegrationsGateway) Discover(ctx context.Context, cwd, pluginID string) (integrations.Discovery, error) {
	return discoverIntegration(ctx, cwd, pluginID)
}

// WorkItemApplicability calcula la aplicabilidad con la misma regla que `axiom change start`.
func (*IntegrationsGateway) WorkItemApplicability(cwd string) WorkItemApplicability {
	_, app := resolveWorkItemTracker(cwd)
	if app.tracker != nil {
		return WorkItemApplicability{Applicable: true, Plugin: app.tracker.PluginID(), RefAlias: app.tracker.RefAlias()}
	}
	reason := "no hay ningún plugin de integración activo: configúralo en la pantalla de integraciones"
	switch {
	case len(app.warnings) > 0:
		reason = strings.Join(app.warnings, "; ")
	case app.portUnsupported:
		reason = "el plugin activo no admite work items: elige un plugin que los admita en la pantalla de integraciones"
	}
	return WorkItemApplicability{Reason: reason}
}
