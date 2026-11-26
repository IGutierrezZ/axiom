package reqid

import (
	"strconv"
	"strings"
)

// SeriesOf reconstruye una serie a partir de su clave (`Series.Key()`: `30`, `USU`,
// `USU-3`, o vacía en una serie única). Falso si la clave no casa con los marcadores del
// patrón. Es un auxiliar de las pruebas: ningún código de producción construye una serie a
// partir de su clave, así que no forma parte de la API (evita código muerto exportado).
func (p Pattern) SeriesOf(key string) (Series, bool) {
	var kinds []segKind
	for _, s := range p.get().segs {
		if s.kind == segDomain || s.kind == segSeries {
			kinds = append(kinds, s.kind)
		}
	}
	var vals []string
	if key != "" {
		vals = strings.Split(key, "-")
	}
	if len(vals) != len(kinds) {
		return Series{}, false
	}
	for i, k := range kinds {
		switch k {
		case segDomain:
			if !componentDomain.MatchString(vals[i]) {
				return Series{}, false
			}
		default:
			if !componentSeries.MatchString(vals[i]) {
				return Series{}, false
			}
			if _, err := strconv.Atoi(vals[i]); err != nil {
				return Series{}, false
			}
		}
	}
	return p.buildSeries(vals), true
}
