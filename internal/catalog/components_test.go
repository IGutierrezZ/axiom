package catalog

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// El estado filtra los componentes retirados con model.IsKnownComponentID: todo
// componente del catálogo debe pasar ese filtro, o se perdería al leer el estado.
func TestModelKnownComponentsCoverCatalog(t *testing.T) {
	for _, component := range MVPComponents() {
		if !model.IsKnownComponentID(component.ID) {
			t.Errorf("model.IsKnownComponentID(%q) = false, pero el catálogo lo admite", component.ID)
		}
	}
	if model.IsKnownComponentID("gga") {
		t.Errorf("gga está retirado y no debe ser un componente conocido")
	}
}
