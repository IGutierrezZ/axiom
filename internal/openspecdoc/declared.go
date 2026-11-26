package openspecdoc

import "github.com/IGutierrezZ/axiom/v3/internal/reqid"

// ParseChangeSpec lee el `specs/<capacidad>/spec.md` de un cambio. Una delta con secciones
// ADDED, MODIFIED, REMOVED o RENAMED se lee como tal; una capacidad nueva escrita como
// especificación completa, sin secciones de delta, cuenta como ADDED de todos sus
// requisitos y fullSpec es verdadero.
func ParseChangeSpec(text string) (d Delta, fullSpec bool) {
	d = ParseDelta(text)
	if !d.Empty() {
		return d, false
	}
	if reqs := ParseCanonical(text); len(reqs) > 0 {
		d.Added = reqs
		return d, true
	}
	return d, false
}

// DeclaredRequirement es un requisito que la delta de un cambio crea o toca, tal como lo
// proyecta `axiom sdd status --json` (RF-CLI-14): `Requirement` es el nombre exacto de la
// delta (el antiguo, en un RENAMED); `ID` solo si ese nombre lleva un identificador válido
// o un provisional.
type DeclaredRequirement struct {
	Capability  string `json:"capability"`
	ID          string `json:"id,omitempty"`
	Provisional bool   `json:"provisional,omitempty"`
	Operation   string `json:"operation"`
	Requirement string `json:"requirement"`
}

// Declaration son los requisitos que un cambio toca (MODIFIED, REMOVED y RENAMED) y los
// que crea (ADDED).
type Declaration struct {
	Modifies []DeclaredRequirement `json:"modifies"`
	Creates  []DeclaredRequirement `json:"creates"`
}

// Declared devuelve los requisitos que declara la delta de una capacidad, en el orden del
// texto dentro de cada operación: `Modifies` con los MODIFIED, REMOVED y RENAMED (por ese
// orden) y `Creates` con los ADDED. No lee la canónica: un nombre sin identificador se
// resuelve por título y no lleva `id`.
func Declared(capability string, d Delta, p reqid.Pattern) Declaration {
	declare := func(operation, name string) DeclaredRequirement {
		item := DeclaredRequirement{Capability: capability, Operation: operation, Requirement: name}
		item.ID, item.Provisional = keyOfName(name, p)
		return item
	}
	var out Declaration
	for _, r := range d.Modified {
		out.Modifies = append(out.Modifies, declare("MODIFIED", r.Name))
	}
	for _, r := range d.Removed {
		out.Modifies = append(out.Modifies, declare("REMOVED", r.Name))
	}
	for _, r := range d.Renamed {
		name := r.Name
		if oldName, _, ok := SplitRename(r.Name); ok {
			name = oldName
		}
		out.Modifies = append(out.Modifies, declare("RENAMED", name))
	}
	for _, r := range d.Added {
		out.Creates = append(out.Creates, declare("ADDED", r.Name))
	}
	return out
}

// keyOfName devuelve el identificador del nombre de un requisito: el provisional que el
// patrón admite, o el último identificador válido; vacío si no lleva ninguno.
func keyOfName(name string, p reqid.Pattern) (id string, provisional bool) {
	if token, ok := reqid.ProvisionalInTitle(name); ok {
		if _, err := p.ParseProvisional(token); err == nil {
			return token, true
		}
		return "", false
	}
	return p.InTitle(name), false
}
