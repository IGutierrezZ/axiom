package adoptcheck

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// loadInventory lee el inventario guardado una sola vez y lo indexa por `<ruta>#<ancla>`.
// Si no se lee, c.inv queda nil y c.invErr dice por qué: las reglas que dependen de él se
// omiten y solo consta el fallo de `inventory_missing` de checkCoverage.
func (c *checker) loadInventory() {
	inv, err := adoption.ReadInventory(c.changeRoot)
	c.invErr = err
	if err != nil {
		return
	}
	c.inv = &inv
	c.invUnits = map[string]adoption.Unit{}
	for _, src := range inv.Sources {
		for _, u := range src.Units {
			c.invUnits[unitKey(src.Path, u.Anchor)] = u
		}
	}
}

// sameID compara dos identificadores por clave (serie y número del patrón efectivo) cuando
// los dos son válidos y por cadena exacta en otro caso.
func (c *checker) sameID(a, b string) bool {
	if a == b {
		return true
	}
	if !c.pattern.Valid(a) || !c.pattern.Valid(b) {
		return false
	}
	sa, okA := c.pattern.Series(a)
	sb, okB := c.pattern.Series(b)
	na, okNA := c.pattern.Number(a)
	nb, okNB := c.pattern.Number(b)
	return okA && okB && okNA && okNB && sa == sb && na == nb
}

func (c *checker) anySame(id string, list []string) bool {
	for _, other := range list {
		if c.sameID(id, other) {
			return true
		}
	}
	return false
}

// addDistinct añade los identificadores que aún no constan (por clave) en list.
func (c *checker) addDistinct(list []string, ids ...string) []string {
	for _, id := range ids {
		if !c.anySame(id, list) {
			list = append(list, id)
		}
	}
	return list
}

func quoteList(ids []string) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "`" + id + "`"
	}
	return strings.Join(quoted, ", ")
}

// checkOriginIDs aplica `origin_id` (D6): el título definitivo de cada requisito conserva
// el identificador del cliente, que sale del inventario (los IDs propios de las unidades
// citadas) y de los `<ID original>` de sus `Origen:`.
func (c *checker) checkOriginIDs() {
	for _, a := range c.added {
		origins := adoption.ParseOrigins(a.req.Body)
		if len(origins) == 0 {
			continue
		}
		var expected, fromUnits []string
		allCited := c.inv != nil
		for _, o := range origins {
			ref := unitKey(normalizeRel(o.Path), o.Anchor)
			var unitIDs []string
			if c.inv != nil {
				if u, ok := c.invUnits[ref]; ok {
					unitIDs = adoption.IdentityIDs(u)
				} else {
					allCited = false
				}
			}
			written := strings.TrimSpace(o.OriginalID)
			if written != "" {
				if ids := adoption.IdentityIDs(adoption.Unit{Title: written}); !(len(ids) == 1 && ids[0] == written) && !c.pattern.Valid(written) {
					c.fail(RuleOriginID, a.req.Name, "el `Origen:` %s del requisito (%s, línea %d) trae «%s» entre paréntesis; debe llevar un único identificador o ninguno",
						ref, a.file, a.req.Line, written)
					written = ""
				}
			}
			if written != "" && len(unitIDs) > 0 && !c.anySame(written, unitIDs) {
				c.fail(RuleOriginID, a.req.Name, "el `Origen:` %s del requisito (%s, línea %d) escribe `%s` pero su unidad trae %s; usa el identificador de la unidad",
					ref, a.file, a.req.Line, written, quoteList(unitIDs))
			}
			if written != "" {
				expected = c.addDistinct(expected, written)
			}
			expected = c.addDistinct(expected, unitIDs...)
			fromUnits = c.addDistinct(fromUnits, unitIDs...)
		}

		switch {
		case a.prov != "" && len(expected) > 0:
			c.fail(RuleOriginID, a.req.Name, "el requisito (%s, línea %d) lleva el provisional `(%s)` pero su origen trae identificador: conserva el identificador del cliente %s en el título",
				a.file, a.req.Line, a.prov, quoteList(expected))
		case a.id != "" && len(expected) == 0 && allCited:
			c.fail(RuleOriginID, a.req.Name, "el requisito (%s, línea %d) lleva el identificador `%s` pero la unidad de su `Origen:` no trae identificador; usa un provisional (%s) en lugar de inventarlo",
				a.file, a.req.Line, a.id, reqid.ProvisionalText("<clave>", "<ordinal>"))
		case a.id != "" && len(expected) > 0 && !c.anySame(a.id, expected):
			c.fail(RuleOriginID, a.req.Name, "el requisito (%s, línea %d) lleva el identificador `%s` pero su origen trae %s; conserva el identificador del cliente en el título",
				a.file, a.req.Line, a.id, quoteList(expected))
		case a.id != "":
			var lost []string
			for _, id := range fromUnits {
				if !c.sameID(a.id, id) {
					lost = append(lost, id)
				}
			}
			if len(lost) > 0 {
				c.fail(RuleOriginID, a.req.Name, "el requisito (%s, línea %d) cita varias unidades del cliente pero solo conserva `%s`; no se conservan %s: usa un requisito por unidad o un provisional para unirlas",
					a.file, a.req.Line, a.id, quoteList(lost))
			}
		}
	}
}

// checkProjectIDs aplica `pattern_undeclared` y `foreign_id_conflict` (D7) con los IDs
// propios de las unidades del inventario que no están en `## Descartado`.
func (c *checker) checkProjectIDs(discards []adoption.Discard) {
	if c.inv == nil {
		return
	}
	discarded := map[string]bool{}
	for _, d := range discards {
		discarded[unitKey(normalizeRel(d.Path), d.Anchor)] = true
	}
	var sources []adoption.Source
	var own []string
	for _, src := range c.inv.Sources {
		kept := adoption.Source{Path: src.Path, SHA256: src.SHA256, Kind: src.Kind}
		for _, u := range src.Units {
			if discarded[unitKey(src.Path, u.Anchor)] {
				continue
			}
			kept.Units = append(kept.Units, u)
			own = append(own, adoption.IdentityIDs(u)...)
		}
		if len(kept.Units) > 0 {
			sources = append(sources, kept)
		}
	}
	proposal := adoption.ProposePattern(sources)
	if proposal == nil {
		return
	}

	inConflict := map[string]bool{}
	for _, conflict := range proposal.UnitConflicts() {
		for _, id := range conflict.IDs {
			inConflict[id] = true
		}
	}
	var unknown, examples []string
	seen := map[string]bool{}
	for _, id := range own {
		if seen[id] || c.pattern.Valid(id) {
			continue
		}
		seen[id] = true
		unknown = append(unknown, id)
		if !inConflict[id] && len(examples) < 3 {
			examples = append(examples, id)
		}
	}
	if len(unknown) > 0 {
		c.failPatternUndeclared(proposal, len(unknown), examples)
	}

	for _, conflict := range proposal.UnitConflicts() {
		c.fail(RuleForeignIDConflict, strings.Join(conflict.Units, ", "),
			"%s [%s]; normaliza el origen o descarta la unidad con motivo en `## Descartado` de %s", conflict.Detail, conflict.Code, adoption.DiscardsFileName)
	}
}

func (c *checker) failPatternUndeclared(proposal *adoption.PatternProposal, count int, examples []string) {
	origin := "por defecto"
	if c.loc.Declarations.SDD != nil && c.loc.Declarations.SDD.IDPattern != nil {
		origin = "declarado en axiom.yaml"
	}
	head := fmt.Sprintf("el patrón efectivo `%s` (%s) no reconoce %d identificador(es) del origen", c.pattern.Template(), origin, count)
	if len(examples) > 0 {
		head += " (por ejemplo " + quoteList(examples) + ")"
	}
	scheme := proposal.SchemeConflicts()
	if len(scheme) == 0 && proposal.Template != "" {
		c.fail(RulePatternUndeclared, "sdd.id_pattern",
			"%s; el origen propone `%s`: copia en axiom.yaml las líneas `sdd:` e `  id_pattern: \"%s\"` (si ya existe el bloque `sdd:`, añade solo `id_pattern`) en la rama de trabajo de la adopción (la del cambio, o la de integración del repositorio de specs en multirepo) y vuelve a ejecutar `adopt check`",
			head, proposal.Template, proposal.Template)
		return
	}
	var parts []string
	for _, conflict := range scheme {
		parts = append(parts, fmt.Sprintf("[%s] %s", conflict.Code, conflict.Detail))
	}
	var counts []string
	for _, e := range proposal.Evidence {
		counts = append(counts, fmt.Sprintf("%s «%s» ×%d", e.Dimension, e.Value, e.Count))
	}
	c.fail(RulePatternUndeclared, "sdd.id_pattern",
		"%s y el origen mezcla esquemas, así que no hay plantilla que proponer: %d conflicto(s): %s (evidencia: %s); normaliza a mano los identificadores del origen (Axiom no los reescribe), vuelve a generar el inventario con `%s` y declara el patrón que proponga",
		head, len(scheme), strings.Join(parts, "; "), strings.Join(counts, " · "), c.scanCommand)
}
