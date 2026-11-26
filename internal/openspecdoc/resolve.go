package openspecdoc

import "github.com/IGutierrezZ/axiom/v3/internal/reqid"

// Match es el resultado de Resolve: ningún requisito, uno o varios sin desempate.
type Match = reqid.Match

const (
	MatchNone      = reqid.MatchNone
	MatchFound     = reqid.MatchFound
	MatchAmbiguous = reqid.MatchAmbiguous
)

// SupersededSuffix es la marca que el compositor añade al nombre de un requisito
// superado o deprecado.
const SupersededSuffix = reqid.SupersededSuffix

// Resolve devuelve el índice, en names (la canónica en orden), del requisito al que se
// refiere el nombre de una delta (RF-ARC-11): por identificador cuando el nombre lleva
// uno válido para p, con vuelta al título exacto ante varios con el mismo identificador,
// y por título exacto en el resto. Lo usan el compositor, la clasificación idempotente
// de livespec y la comprobación previa al archivado, para que los tres coincidan. No
// añade lógica propia: delega en reqid.Pattern.Resolve.
func Resolve(names []string, name string, p reqid.Pattern) (int, Match) {
	return p.Resolve(names, name)
}
