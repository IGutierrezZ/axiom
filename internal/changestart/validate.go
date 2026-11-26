package changestart

import (
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// Profile es el perfil de partida del cambio (D-07).
type Profile string

const (
	ProfileIncrement Profile = "incremento"
	ProfileBug       Profile = "bug"
)

// Lane es el carril del cambio.
type Lane string

const (
	LaneSDD Lane = "sdd"
	LaneODD Lane = "odd"
)

// mapType busca el tipo del work item en item_keys sin distinguir mayúsculas ni espacios extremos y
// devuelve la entrada con la grafía configurada, que es el dato de confianza (D-04).
func mapType(keys []manifest.TypePrefix, itemType string) (manifest.TypePrefix, bool) {
	want := strings.TrimSpace(itemType)
	for _, k := range keys {
		if strings.EqualFold(strings.TrimSpace(k.Type), want) {
			return k, true
		}
	}
	return manifest.TypePrefix{}, false
}

// profileFor es bug solo si el tipo es literalmente Bug (sin distinguir mayúsculas ni espacios
// extremos) e incremento en cualquier otro caso (D-07). Un tipo de defecto con otro nombre, como
// Defect, produce incremento.
func profileFor(itemType string) Profile {
	if strings.EqualFold(strings.TrimSpace(itemType), "Bug") {
		return ProfileBug
	}
	return ProfileIncrement
}

// closedCategory dice si la categoría del estado cierra el work item. Se decide por categoría, nunca
// por el nombre del estado.
func closedCategory(category string) bool {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "completed", "removed":
		return true
	}
	return false
}
