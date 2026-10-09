package knowledge

import "github.com/IGutierrezZ/axiom/v3/internal/workspace"

// defaultSpecsRoot resuelve la raíz de especificaciones por defecto de knowledge:
// el planning home que devuelve workspace.Locate desde la raíz de código que se
// analiza (REQ-24.10, D-07). La raíz de código y la de especificaciones son
// distintas: en el workspace canónico, analizar un repositorio de código lee las
// especificaciones del repositorio de especificaciones.
//
// Devuelve además los avisos no bloqueantes de la resolución (el de la ubicación
// antigua divergente, Q14), que los llamadores vuelcan en las notes de su
// resultado. Un fallo de Locate se propaga: nombra la causa y la continuación.
func defaultSpecsRoot(absWorkspace string) (specsRoot string, notes []string, err error) {
	loc, err := workspace.Locate(absWorkspace)
	if err != nil {
		return "", nil, err
	}
	if warning, ok := loc.LegacySpecsDivergence(); ok {
		notes = append(notes, warning)
	}
	return loc.PlanningHome, notes, nil
}
