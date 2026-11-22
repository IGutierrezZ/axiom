// Package azuredevops es el adaptador de Azure DevOps del marco de integraciones: el plugin
// azure-devops, su cliente REST 7.1 y las operaciones sobre work items y repositorios.
//
// Solo el ensamblado de plugins (internal/integrations/builtin) lo importa; el núcleo
// internal/integrations no lo conoce.
package azuredevops

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// Options configura el plugin. El valor cero es el de producción.
type Options struct {
	// Endpoint es la URL base del servicio. Solo para pruebas: por defecto https://dev.azure.com y
	// axiom.yaml no puede cambiarla.
	Endpoint string
	// HTTPClient es el cliente HTTP. Por defecto uno nuevo con el transporte por defecto.
	HTTPClient *http.Client
}

// Plugin es el adaptador de Azure DevOps. Implementa integrations.Plugin.
type Plugin struct {
	opts Options

	mu       sync.Mutex
	sessions map[string]*orgState // clave: organización en minúsculas
}

var _ integrations.Plugin = (*Plugin)(nil)

// New construye el plugin.
func New(opts Options) *Plugin {
	return &Plugin{opts: opts, sessions: map[string]*orgState{}}
}

// Manifest devuelve el manifiesto del catálogo cerrado.
func (p *Plugin) Manifest() manifest.Manifest { return manifest.AzureDevOps() }

// Deliver no entrega nada: devuelve un error permanente sin tocar la red ni el disco. El manifiesto
// solo escucha change.start, que ningún verbo emite todavía; si algo lo encolara, Permanent lo deja
// abandonado en el primer intento, sin bucle de reintentos.
func (p *Plugin) Deliver(context.Context, integrations.Delivery) (integrations.Outcome, error) {
	return integrations.Outcome{}, integrations.Permanent(ErrLifecycleUnsupported)
}

// orgState es el estado compartido por organización durante la vida del proceso.
type orgState struct {
	mu sync.Mutex
}

// session devuelve el estado de la organización, creándolo la primera vez. Las organizaciones de
// Azure DevOps no distinguen mayúsculas: la clave va en minúsculas.
func (p *Plugin) session(organization string) *orgState {
	key := strings.ToLower(organization)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.sessions == nil {
		p.sessions = map[string]*orgState{}
	}
	s, ok := p.sessions[key]
	if !ok {
		s = &orgState{}
		p.sessions[key] = s
	}
	return s
}

// Target es el destino de una llamada: organización, proyecto y equipo de integrations.azure-devops.
type Target struct{ Organization, Project, Team string }

// TargetFromConfig lee organization, project y team de la configuración del plugin. Cada clave debe
// ser una cadena no vacía: si no, devuelve ErrConfig nombrando la clave.
func TargetFromConfig(cfg map[string]any) (Target, error) {
	var t Target
	for _, field := range []struct {
		key string
		dst *string
	}{{"organization", &t.Organization}, {"project", &t.Project}, {"team", &t.Team}} {
		raw, ok := cfg[field.key]
		if !ok {
			return Target{}, fmt.Errorf("%w: falta la clave %q", ErrConfig, field.key)
		}
		s, ok := raw.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return Target{}, fmt.Errorf("%w: la clave %q debe ser una cadena no vacía", ErrConfig, field.key)
		}
		*field.dst = s
	}
	return t, nil
}
