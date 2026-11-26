package reqtrace

import "github.com/IGutierrezZ/axiom/v3/internal/reqid"

// resolveEquivalents es resolveEquivalentsLegacy sin mapa de IDs heredados. Solo lo usan las
// pruebas: la producción siempre pasa el resolutor (nil cuando no hay mapa).
func resolveEquivalents(q Query, p reqid.Pattern, traces []traceSet, aliases AliasResolver) (equivalence, []Warning) {
	return resolveEquivalentsLegacy(q, p, traces, aliases, nil)
}
