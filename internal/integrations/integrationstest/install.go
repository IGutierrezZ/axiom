package integrationstest

import (
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

// Install instala en las costuras de prueba el catálogo de manifiestos de plugins, un registro con
// ellos y un almacén de secretos en memoria (cadena entorno -> Memory). Todo se restaura con
// Cleanup. Devuelve el almacén en memoria para que la prueba guarde secretos.
//
// Estado de paquete: las pruebas que llaman a Install NO deben llamar a t.Parallel().
func Install(t interface {
	Helper()
	Cleanup(func())
}, plugins ...integrations.Plugin) *secrets.Memory {
	t.Helper()
	manifests := make([]manifest.Manifest, len(plugins))
	for i, p := range plugins {
		manifests[i] = p.Manifest()
	}
	manifest.SetCatalogForTest(t, manifests...)
	integrations.SetDefaultRegistryForTest(t, plugins...)

	memory := secrets.NewMemory()
	secrets.SetDefaultForTest(t, secrets.NewChain(secrets.Env{}, memory))
	return memory
}
