// Package builtin ensambla la lista cerrada de plugins de integración compilados en esta versión.
// Es el único punto que conoce los adaptadores concretos: internal/integrations no los importa.
package builtin

import (
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
)

// Plugins devuelve la lista cerrada de plugins de esta versión, en el orden del catálogo.
func Plugins() []integrations.Plugin {
	return []integrations.Plugin{azuredevops.New(azuredevops.Options{})}
}

// DefaultRegistry es el registro de producción. Respeta la costura
// integrations.SetDefaultRegistryForTest.
func DefaultRegistry() (*integrations.Registry, error) {
	return integrations.ResolveRegistry(Plugins())
}
