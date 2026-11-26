package adoption

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// Códigos estables de los conflictos de la propuesta de patrón (RF-ADP-01).
const (
	// Conflictos de esquema: impiden proponer plantilla.
	ConflictMixedPrefixes   = "mixed_prefixes"
	ConflictMixedSeparators = "mixed_separators"
	ConflictMixedDomain     = "mixed_domain"
	ConflictTemplateInvalid = "template_invalid"
	// Conflictos de unidad: no impiden la plantilla.
	ConflictAmbiguousUnitID    = "ambiguous_unit_id"
	ConflictDuplicateForeignID = "duplicate_foreign_id"
)

// Dimensiones de la evidencia, en el orden fijo en que se listan.
const (
	dimPrefix    = "prefix"
	dimSeparator = "separator"
	dimDomain    = "domain"
	dimWidth     = "width"
)

// PatternProposal es el patrón de identificadores (`sdd.id_pattern`) que el escaneo
// propone a partir de los identificadores propios de las unidades. Nunca se escribe en
// axiom.yaml: es una propuesta para que la persona la declare.
type PatternProposal struct {
	Template  string            `yaml:"template,omitempty" json:"template,omitempty"`
	IDs       int               `yaml:"ids" json:"ids"`
	Evidence  []PatternEvidence `yaml:"evidence" json:"evidence"`
	Conflicts []PatternConflict `yaml:"conflicts,omitempty" json:"conflicts,omitempty"`
}

// PatternEvidence cuenta cuántos identificadores propios tienen un valor de una dimensión.
type PatternEvidence struct {
	Dimension string `yaml:"dimension" json:"dimension"` // prefix|separator|domain|width
	Value     string `yaml:"value" json:"value"`
	Count     int    `yaml:"count" json:"count"`
}

// PatternConflict es una incoherencia de los identificadores propios.
type PatternConflict struct {
	Code   string   `yaml:"code" json:"code"`
	Detail string   `yaml:"detail" json:"detail"`
	Units  []string `yaml:"units,omitempty" json:"units,omitempty"` // <ruta>#<ancla>
	IDs    []string `yaml:"ids,omitempty" json:"ids,omitempty"`
}

// SchemeConflicts devuelve los conflictos de esquema (impiden la plantilla).
func (p *PatternProposal) SchemeConflicts() []PatternConflict {
	return p.conflictsOfScheme(true)
}

// UnitConflicts devuelve los conflictos de unidad (no impiden la plantilla).
func (p *PatternProposal) UnitConflicts() []PatternConflict {
	return p.conflictsOfScheme(false)
}

func (p *PatternProposal) conflictsOfScheme(scheme bool) []PatternConflict {
	if p == nil {
		return nil
	}
	var out []PatternConflict
	for _, c := range p.Conflicts {
		isScheme := c.Code != ConflictAmbiguousUnitID && c.Code != ConflictDuplicateForeignID
		if isScheme == scheme {
			out = append(out, c)
		}
	}
	return out
}

// idShape descompone un identificador ajeno: prefijo, separador, dominio opcional con su
// separador y número. Cubre todo lo que produce foreignIDPattern.
var idShape = regexp.MustCompile(`^([A-Z]+)([-_])(?:([A-Z]{2,5})([-_]))?([0-9]{1,5})$`)

type idParts struct {
	id     string
	prefix string
	sep1   string
	domain string
	sep2   string
	number string
}

func splitID(id string) (idParts, bool) {
	m := idShape.FindStringSubmatch(id)
	if m == nil {
		return idParts{}, false
	}
	return idParts{id: id, prefix: m[1], sep1: m[2], domain: m[3], sep2: m[4], number: m[5]}, true
}

// IdentityIDs devuelve los identificadores propios de una unidad: los del título y, si
// el título no tiene ninguno, los de su sección. Las referencias cruzadas del cuerpo no
// cuentan cuando el título ya identifica la unidad.
func IdentityIDs(u Unit) []string {
	if ids := foreignIDs(u.Title); len(ids) > 0 {
		return ids
	}
	return u.ForeignIDs
}

type ownedID struct {
	unit  string
	parts idParts
}

// ProposePattern infiere el patrón de identificadores a partir de los identificadores
// propios de las unidades. Es determinista y devuelve nil si no hay ninguno.
func ProposePattern(sources []Source) *PatternProposal {
	var owned []ownedID
	var unitKeys []string
	unitIDs := map[string][]string{}
	for _, s := range sources {
		for _, u := range s.Units {
			key := s.Path
			if u.Anchor != "" {
				key += "#" + u.Anchor
			}
			ids := IdentityIDs(u)
			if len(ids) == 0 {
				continue
			}
			unitKeys = append(unitKeys, key)
			unitIDs[key] = append(unitIDs[key], ids...)
			for _, id := range ids {
				if parts, ok := splitID(id); ok {
					owned = append(owned, ownedID{unit: key, parts: parts})
				}
			}
		}
	}
	if len(owned) == 0 {
		return nil
	}

	p := &PatternProposal{IDs: len(owned), Evidence: evidenceOf(owned)}
	scheme := schemeConflicts(owned)
	p.Conflicts = append(p.Conflicts, scheme...)
	if len(scheme) == 0 {
		template := buildTemplate(owned)
		if bad := validateTemplate(template, owned); bad != nil {
			p.Conflicts = append(p.Conflicts, *bad)
		} else {
			p.Template = template
		}
	}
	p.Conflicts = append(p.Conflicts, unitConflicts(p.Template, unitKeys, unitIDs)...)
	return p
}

// evidenceOf cuenta los identificadores por valor de cada dimensión, en orden fijo.
func evidenceOf(owned []ownedID) []PatternEvidence {
	counts := map[string]map[string]int{dimPrefix: {}, dimSeparator: {}, dimDomain: {}, dimWidth: {}}
	for _, o := range owned {
		sep := o.parts.sep1
		if o.parts.domain != "" {
			sep = o.parts.sep1 + "<DOM>" + o.parts.sep2
		}
		counts[dimPrefix][o.parts.prefix]++
		counts[dimSeparator][sep]++
		counts[dimDomain][o.parts.domain]++
		counts[dimWidth][strconv.Itoa(len(o.parts.number))]++
	}
	var out []PatternEvidence
	for _, dim := range []string{dimPrefix, dimSeparator, dimDomain, dimWidth} {
		values := make([]string, 0, len(counts[dim]))
		for v := range counts[dim] {
			values = append(values, v)
		}
		if dim == dimWidth {
			sort.Slice(values, func(i, j int) bool {
				a, _ := strconv.Atoi(values[i])
				b, _ := strconv.Atoi(values[j])
				return a < b
			})
		} else {
			sort.Strings(values)
		}
		for _, v := range values {
			out = append(out, PatternEvidence{Dimension: dim, Value: v, Count: counts[dim][v]})
		}
	}
	return out
}

// schemeConflicts detecta las mezclas de prefijo, de separador y de presencia de dominio.
func schemeConflicts(owned []ownedID) []PatternConflict {
	prefixes := map[string]string{}
	sep1 := map[string]string{}
	sep2 := map[string]string{}
	withDomain, withoutDomain := map[string]string{}, map[string]string{}
	for _, o := range owned {
		prefixes[o.parts.prefix] = firstOf(prefixes[o.parts.prefix], o.parts.id)
		sep1[o.parts.sep1] = firstOf(sep1[o.parts.sep1], o.parts.id)
		if o.parts.domain != "" {
			sep2[o.parts.sep2] = firstOf(sep2[o.parts.sep2], o.parts.id)
			withDomain[o.parts.id] = o.parts.id
		} else {
			withoutDomain[o.parts.id] = o.parts.id
		}
	}
	var out []PatternConflict
	if len(prefixes) > 1 {
		out = append(out, PatternConflict{Code: ConflictMixedPrefixes,
			Detail: fmt.Sprintf("los identificadores mezclan los prefijos %s", quoteKeys(prefixes)), IDs: examples(prefixes)})
	}
	if len(sep1) > 1 || len(sep2) > 1 {
		ex := map[string]string{}
		for k, v := range sep1 {
			ex["«"+k+"»"] = v
		}
		for k, v := range sep2 {
			ex["«"+k+"» tras el dominio"] = v
		}
		out = append(out, PatternConflict{Code: ConflictMixedSeparators,
			Detail: fmt.Sprintf("los identificadores mezclan los separadores %s", strings.Join(sortedKeys(ex), ", ")), IDs: examples(ex)})
	}
	if len(withDomain) > 0 && len(withoutDomain) > 0 {
		out = append(out, PatternConflict{Code: ConflictMixedDomain,
			Detail: fmt.Sprintf("hay identificadores con dominio (%d) y sin dominio (%d)", len(withDomain), len(withoutDomain)),
			IDs:    examples(map[string]string{"con dominio": minKey(withDomain), "sin dominio": minKey(withoutDomain)})})
	}
	return out
}

func firstOf(cur, id string) string {
	if cur == "" || id < cur {
		return id
	}
	return cur
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func minKey(m map[string]string) string {
	keys := sortedKeys(m)
	return keys[0]
}

func quoteKeys(m map[string]string) string {
	keys := sortedKeys(m)
	for i, k := range keys {
		keys[i] = "`" + k + "`"
	}
	return strings.Join(keys, ", ")
}

// examples devuelve un identificador de ejemplo por valor, en el orden de los valores.
func examples(m map[string]string) []string {
	var ids []string
	for _, k := range sortedKeys(m) {
		ids = append(ids, m[k])
	}
	return ids
}

// buildTemplate traduce el esquema coherente de los identificadores a la gramática de
// reqid: literal inicial, <DOM> si todos llevan dominio y marcador de número.
func buildTemplate(owned []ownedID) string {
	first := owned[0].parts
	var b strings.Builder
	b.WriteString(first.prefix + first.sep1)
	if first.domain != "" {
		b.WriteString("<DOM>" + first.sep2)
	}
	b.WriteString(numberMarker(owned))
	return b.String()
}

// numberMarker elige <N×k> si hay prueba de relleno con ceros (algún número del ancho
// mínimo k empieza por 0) y los demás tienen ancho k o más, sin cero inicial; en otro
// caso <m>. No existe marcador de ancho 1 ni de ancho mayor que 6.
func numberMarker(owned []ownedID) string {
	k := len(owned[0].parts.number)
	for _, o := range owned {
		if n := len(o.parts.number); n < k {
			k = n
		}
	}
	if k < 2 || k > 6 {
		return "<m>"
	}
	padded := false
	for _, o := range owned {
		n := o.parts.number
		switch {
		case len(n) == k:
			if n[0] == '0' {
				padded = true
			}
		case n[0] == '0':
			return "<m>"
		}
	}
	if !padded {
		return "<m>"
	}
	return "<" + strings.Repeat("N", k) + ">"
}

// validateTemplate comprueba la plantilla con reqid: debe parsear y aceptar todos los
// identificadores (defensa; con un esquema coherente no debería fallar).
func validateTemplate(template string, owned []ownedID) *PatternConflict {
	pattern, err := reqid.Parse(template)
	if err != nil {
		return &PatternConflict{Code: ConflictTemplateInvalid, Detail: fmt.Sprintf("la plantilla %q no es válida: %v", template, err)}
	}
	var bad []string
	for _, o := range owned {
		if !pattern.Valid(o.parts.id) {
			bad = append(bad, o.parts.id)
		}
	}
	if len(bad) > 0 {
		return &PatternConflict{Code: ConflictTemplateInvalid,
			Detail: fmt.Sprintf("la plantilla %q no reconoce %d identificador(es)", template, len(bad)), IDs: firstN(bad, 3)}
	}
	return nil
}

func firstN(ids []string, n int) []string {
	if len(ids) > n {
		return ids[:n]
	}
	return ids
}

// unitConflicts detecta las unidades con varios identificadores propios y los
// identificadores repetidos en varias unidades u orígenes. Con plantilla se compara por
// (serie, número) y sin ella, por cadena.
func unitConflicts(template string, unitKeys []string, unitIDs map[string][]string) []PatternConflict {
	var out []PatternConflict
	seen := map[string]bool{}
	var orderedUnits []string
	for _, key := range unitKeys {
		if seen[key] {
			continue
		}
		seen[key] = true
		orderedUnits = append(orderedUnits, key)
	}
	for _, key := range orderedUnits {
		if ids := unitIDs[key]; len(ids) > 1 {
			out = append(out, PatternConflict{Code: ConflictAmbiguousUnitID,
				Detail: fmt.Sprintf("la unidad %s trae varios identificadores propios (%s)", key, strings.Join(ids, ", ")),
				Units:  []string{key}, IDs: ids})
		}
	}

	var pattern reqid.Pattern
	havePattern := false
	if template != "" {
		if pt, err := reqid.Parse(template); err == nil {
			pattern, havePattern = pt, true
		}
	}
	compare := func(id string) string {
		if havePattern {
			if s, ok := pattern.Series(id); ok {
				if n, ok := pattern.Number(id); ok {
					return fmt.Sprintf("%s#%d", s.Key(), n)
				}
			}
		}
		return id
	}
	type group struct {
		units []string
		ids   []string
	}
	groups := map[string]*group{}
	var order []string
	for _, key := range orderedUnits {
		for _, id := range unitIDs[key] {
			ck := compare(id)
			g := groups[ck]
			if g == nil {
				g = &group{}
				groups[ck] = g
				order = append(order, ck)
			}
			if !contains(g.units, key) {
				g.units = append(g.units, key)
			}
			if !contains(g.ids, id) {
				g.ids = append(g.ids, id)
			}
		}
	}
	for _, ck := range order {
		g := groups[ck]
		if len(g.units) > 1 {
			sort.Strings(g.ids)
			out = append(out, PatternConflict{Code: ConflictDuplicateForeignID,
				Detail: fmt.Sprintf("el identificador %s aparece en %d unidades (%s)", strings.Join(g.ids, ", "), len(g.units), strings.Join(g.units, ", ")),
				Units:  g.units, IDs: g.ids})
		}
	}
	return out
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
