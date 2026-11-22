package builtin

import (
	"reflect"
	"sort"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// TestDefaultRegistryMatchesCatalog sustituye a la antigua prueba del registro vacío: los
// identificadores del registro de producción coinciden exactamente con los del catálogo cerrado.
func TestDefaultRegistryMatchesCatalog(t *testing.T) {
	reg, err := DefaultRegistry()
	if err != nil {
		t.Fatalf("DefaultRegistry: %v", err)
	}
	var catalogIDs []string
	for _, m := range manifest.Catalog() {
		catalogIDs = append(catalogIDs, m.ID)
		if _, ok := reg.Get(m.ID); !ok {
			t.Errorf("el plugin %q del catálogo no está en el registro", m.ID)
		}
	}
	var pluginIDs []string
	for _, p := range Plugins() {
		pluginIDs = append(pluginIDs, p.Manifest().ID)
	}
	sort.Strings(catalogIDs)
	sort.Strings(pluginIDs)
	if !reflect.DeepEqual(catalogIDs, pluginIDs) {
		t.Fatalf("ids de Plugins() %v distintos de los de Catalog() %v", pluginIDs, catalogIDs)
	}
	if !reflect.DeepEqual(pluginIDs, []string{"azure-devops"}) {
		t.Fatalf("la lista cerrada de esta versión es exactamente azure-devops, obtuvo %v", pluginIDs)
	}
}
