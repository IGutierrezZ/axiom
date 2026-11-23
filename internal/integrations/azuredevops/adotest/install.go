package adotest

import (
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

// Install instala en las costuras de prueba un registro cuyo plugin azure-devops apunta al servidor
// local (URL, cliente HTTP solo local y reloj manual) y un almacén de secretos en memoria (cadena
// entorno -> Memory). Deja el catálogo de manifiestos por defecto, que ya contiene azure-devops. Todo
// se restaura con Cleanup. Devuelve el almacén en memoria para que la prueba guarde el PAT.
//
// Estado de paquete: las pruebas que llaman a Install NO deben llamar a t.Parallel().
func Install(t interface {
	Helper()
	Cleanup(func())
}, srv *Server) *secrets.Memory {
	t.Helper()
	integrations.SetDefaultRegistryForTest(t, azuredevops.New(srv.Options()))

	memory := secrets.NewMemory()
	secrets.SetDefaultForTest(t, secrets.NewChain(secrets.Env{}, memory))
	return memory
}
