package reqid

import "strings"

// Match es el resultado de resolver el nombre de un requisito de una delta contra los
// nombres de la canónica.
type Match int

const (
	// MatchNone: ningún requisito de la canónica corresponde al nombre.
	MatchNone Match = iota
	// MatchFound: exactamente un requisito corresponde; Resolve devuelve su índice.
	MatchFound
	// MatchAmbiguous: varios requisitos llevan el identificador y ninguno tiene el
	// título exacto.
	MatchAmbiguous
)

// SupersededSuffix es la marca que el compositor añade al nombre de un requisito
// superado o deprecado.
const SupersededSuffix = " [SUPERSEDED / DEPRECADO]"

// IsSuperseded dice si el nombre de un requisito lleva la marca de superado.
func IsSuperseded(name string) bool { return strings.HasSuffix(name, SupersededSuffix) }

// Resolve devuelve el índice, en names (los requisitos de la canónica, en orden), del
// requisito al que se refiere el nombre de una delta (RF-ARC-11):
//
//   - un nombre sin identificador válido se busca por título exacto (la primera
//     coincidencia);
//   - un nombre con identificador se busca entre los requisitos no superados que
//     llevan ese mismo identificador: exactamente uno es el buscado; ninguno es
//     MatchNone; varios se desempatan por el título exacto y, si ninguno lo tiene,
//     el resultado es MatchAmbiguous.
//
// Los requisitos superados quedan fuera de la resolución por identificador: su
// identificador está retirado y un MODIFIED no debe reabrirlos.
func (p Pattern) Resolve(names []string, name string) (int, Match) {
	id := p.InTitle(name)
	if id == "" {
		for i, n := range names {
			if n == name {
				return i, MatchFound
			}
		}
		return -1, MatchNone
	}
	var carriers []int
	for i, n := range names {
		if !IsSuperseded(n) && p.InTitle(n) == id {
			carriers = append(carriers, i)
		}
	}
	switch len(carriers) {
	case 0:
		// Ningún no superado lleva el identificador: solo vale el título exacto, que
		// conserva la composición de hoy cuando la delta cita un superado tal cual.
		for i, n := range names {
			if n == name {
				return i, MatchFound
			}
		}
		return -1, MatchNone
	case 1:
		return carriers[0], MatchFound
	}
	for _, i := range carriers {
		if names[i] == name {
			return i, MatchFound
		}
	}
	return -1, MatchAmbiguous
}
