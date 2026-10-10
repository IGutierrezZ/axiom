package specbaseline

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// seedContinuation es la continuación que acompaña a una familia de IDs sin
// ningún ID en el documento destino (REQ-24.59 b, D-11). Es literal: responde a
// la nota (b) de la compuerta `spec`.
const seedContinuation = "siembra la numeración añadiendo al documento, en un commit propio del repositorio de especificaciones, el primer requisito de la familia con su ID inicial (por ejemplo `NFR-001`, con el relleno de ceros que heredará la familia), o cambia el ADDED a una familia existente; después ejecuta `axiom sdd integrate check --cwd \"<workspaceRoot>\" --change <cambio>`"

// renameProvisionalContinuation es la continuación común de las detenciones por
// un ID provisional no resoluble o ambiguo.
const renameProvisionalContinuation = "renombra el ID provisional en las delta specs del cambio o normaliza antes el documento del proyecto"

// IDAssignment es la correspondencia de un ID provisional, el del encabezado de
// un requisito ADDED del delta, con el ID definitivo que archive le asigna
// (REQ-24.43, REQ-24.59).
type IDAssignment struct {
	Provisional string `yaml:"provisional" json:"provisional"`
	Definitive  string `yaml:"definitive" json:"definitive"`
	Domain      string `yaml:"domain" json:"domain"`
}

// IDMap es la correspondencia de IDs provisionales a definitivos de un cambio
// entero. Se calcula una sola vez con PlanIDs y conserva un orden estable: el de
// los documentos con perfil y, dentro de cada uno, el de los ADDED del delta.
type IDMap struct {
	assignments   []IDAssignment
	byProvisional map[string]int
	// byLength son los índices de assignments ordenados del ID provisional más
	// largo al más corto, para que Substitute elija siempre la aparición más larga.
	byLength []int
}

func newIDMap(assignments []IDAssignment) IDMap {
	m := IDMap{assignments: assignments, byProvisional: make(map[string]int, len(assignments))}
	for i, a := range assignments {
		m.byProvisional[a.Provisional] = i
		m.byLength = append(m.byLength, i)
	}
	sort.SliceStable(m.byLength, func(a, b int) bool {
		return len(assignments[m.byLength[a]].Provisional) > len(assignments[m.byLength[b]].Provisional)
	})
	return m
}

// Assignments devuelve las correspondencias en su orden estable.
func (m IDMap) Assignments() []IDAssignment {
	return append([]IDAssignment(nil), m.assignments...)
}

// Lookup devuelve la correspondencia de un ID provisional.
func (m IDMap) Lookup(provisional string) (IDAssignment, bool) {
	at, ok := m.byProvisional[provisional]
	if !ok {
		return IDAssignment{}, false
	}
	return m.assignments[at], true
}

// tokenSpan es una aparición de un ID provisional como token completo: el
// intervalo [start, end) del texto y el índice de su asignación.
type tokenSpan struct {
	start, end int
	assignment int
}

// tokenSpans localiza, en orden de aparición, los tokens completos de todos los
// IDs provisionales del texto. Un token es una aparición exacta, sensible a
// mayúsculas, que no va precedida ni seguida por una letra Unicode, un dígito, `_`
// o `-`. Con provisionales que se solapan gana el más largo, y un intervalo ya
// reservado no se vuelve a reservar. Es la única regla de token: la comparten
// Substitute y CountProvisional, de modo que la verificación (e) cuenta
// exactamente lo que la sustitución habría reemplazado.
func (m IDMap) tokenSpans(text string) []tokenSpan {
	if len(m.assignments) == 0 || text == "" {
		return nil
	}
	var spans []tokenSpan
	overlaps := func(start, end int) bool {
		for _, s := range spans {
			if start < s.end && s.start < end {
				return true
			}
		}
		return false
	}
	for _, at := range m.byLength {
		provisional := m.assignments[at].Provisional
		for from := 0; from < len(text); {
			found := strings.Index(text[from:], provisional)
			if found < 0 {
				break
			}
			start := from + found
			end := start + len(provisional)
			from = start + 1
			if !isTokenAt(text, start, end) || overlaps(start, end) {
				continue
			}
			spans = append(spans, tokenSpan{start: start, end: end, assignment: at})
		}
	}
	sort.Slice(spans, func(a, b int) bool { return spans[a].start < spans[b].start })
	return spans
}

// Substitute reemplaza cada ID provisional del cambio por su ID definitivo. Solo
// sustituye tokens completos: una aparición exacta, sensible a mayúsculas, que no
// va precedida ni seguida por una letra Unicode, un dígito, `_` o `-`
// (REQ-24.59, D-11). La sustitución es de una sola pasada: el texto ya
// sustituido no se vuelve a examinar. Quien llama la aplica solo al texto que
// integra, nunca al texto preexistente de los documentos.
func (m IDMap) Substitute(text string) string {
	spans := m.tokenSpans(text)
	if len(spans) == 0 {
		return text
	}
	var out strings.Builder
	cursor := 0
	for _, s := range spans {
		out.WriteString(text[cursor:s.start])
		out.WriteString(m.assignments[s.assignment].Definitive)
		cursor = s.end
	}
	out.WriteString(text[cursor:])
	return out.String()
}

// CountProvisional cuenta, por ID provisional, las apariciones como token
// completo en el texto, con la misma regla de token que Substitute. Solo cuenta
// los provisionales cuyo ID definitivo es distinto: un provisional igual a su
// propio definitivo se admite (D-11) y no es una sustitución pendiente. La
// verificación nativa (e) compara este recuento del contenido actual de un
// documento con el del commit registrado por `prepare`, una medida que tolera el
// reajuste del texto preexistente y no depende del final de línea (D-12, D-15).
func (m IDMap) CountProvisional(text string) map[string]int {
	counts := map[string]int{}
	for _, s := range m.tokenSpans(text) {
		if a := m.assignments[s.assignment]; a.Provisional != a.Definitive {
			counts[a.Provisional]++
		}
	}
	return counts
}

// isTokenAt dice si text[start:end] es un token completo: ni precedido ni
// seguido por un carácter de identificador.
func isTokenAt(text string, start, end int) bool {
	if start > 0 {
		if r, _ := utf8.DecodeLastRuneInString(text[:start]); isIDRune(r) {
			return false
		}
	}
	if end < len(text) {
		if r, _ := utf8.DecodeRuneInString(text[end:]); isIDRune(r) {
			return false
		}
	}
	return true
}

func isIDRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

// splitID separa un ID en su familia, el prefijo previo a la última serie de
// dígitos (`RF-` en `RF-123`), y su sufijo numérico. Solo reconoce dígitos ASCII.
func splitID(id string) (family, digits string) {
	family = strings.TrimRight(id, "0123456789")
	return family, id[len(family):]
}

// familyStats resume una familia de IDs en el documento original: el mayor
// número, contando los duplicados, y el ancho del ID de mayor número, que es el
// relleno de ceros de los definitivos (D-11). Con números iguales, el ancho mayor.
type familyStats struct {
	max   *big.Int
	width int
	next  *big.Int // último número asignado en este cambio
}

// docIDs es lo que PlanIDs necesita saber de un documento con perfil.
type docIDs struct {
	existing map[string]int // ID -> línea del primer encabezado
	families map[string]*familyStats
}

func scanDocIDs(doc PlannedDocument) docIDs {
	ids := docIDs{existing: map[string]int{}, families: map[string]*familyStats{}}
	for _, requirement := range ParseDocument(doc.Original).Requirements(doc.Profile) {
		if requirement.ID == "" {
			continue
		}
		if _, seen := ids.existing[requirement.ID]; !seen {
			ids.existing[requirement.ID] = requirement.LineNumber()
		}
		family, digits := splitID(requirement.ID)
		if digits == "" {
			continue
		}
		number, _ := new(big.Int).SetString(digits, 10)
		stats := ids.families[family]
		switch {
		case stats == nil:
			ids.families[family] = &familyStats{max: number, width: len(digits)}
		case number.Cmp(stats.max) > 0:
			stats.max, stats.width = number, len(digits)
		case number.Cmp(stats.max) == 0 && len(digits) > stats.width:
			stats.width = len(digits)
		}
	}
	return ids
}

// PlanIDs calcula, una sola vez para todo el cambio, la correspondencia de los
// IDs provisionales de sus requisitos ADDED a los IDs definitivos (REQ-24.43,
// REQ-24.59, D-11). Recorre los documentos en el orden recibido, que es el de
// `baseline.documents`, y los ADDED en el orden de cada delta. Solo los
// documentos con identidad `id` tienen IDs provisionales.
//
// El definitivo es el siguiente número libre posterior al mayor de la familia del
// ID en el documento original, antes de aplicar ninguna operación y contando los
// duplicados, con el relleno de ceros del ID de mayor número de la familia. Forma
// parte de la validación previa a cualquier escritura: devuelve un *ComposeError
// sin correspondencia parcial cuando
//
//   - un ID provisional no termina en un sufijo numérico (a);
//   - su familia no tiene ningún ID en el documento destino (b);
//   - coincide con un ID existente del documento;
//   - lo declaran dos ADDED del cambio (c); o
//   - coincide con el ID definitivo de otro requisito del cambio (d).
//
// Que un ID provisional coincida con su propio definitivo se admite.
func PlanIDs(docs []PlannedDocument) (IDMap, error) {
	type declaration struct {
		domain string
		name   string
	}
	declared := map[string]declaration{}
	var assignments []IDAssignment
	var names []declaration // nombre del requisito de cada asignación

	for _, doc := range docs {
		if doc.Profile.Heading == nil {
			return IDMap{}, missingProfileError(doc.Domain)
		}
		if doc.Profile.Identity != workspace.IdentityID {
			continue
		}
		added := parseDelta(doc.Delta).Added
		if len(added) == 0 {
			continue
		}
		scanned := scanDocIDs(doc)
		for _, block := range added {
			provisional, _, _, err := parseRequirementName(doc.Domain, doc.Profile, sectionAdded, block.Name)
			if err != nil {
				return IDMap{}, err
			}
			fail := func(reason, continuation string) error {
				return &ComposeError{Domain: doc.Domain, Section: sectionAdded, Requirement: block.Name, Reason: reason, Continuation: continuation}
			}
			family, digits := splitID(provisional)
			if digits == "" {
				return IDMap{}, fail(fmt.Sprintf("el ID provisional %q no termina en un sufijo numérico, de modo que no se puede calcular su ID definitivo", provisional), renameProvisionalContinuation)
			}
			if first, ok := declared[provisional]; ok {
				return IDMap{}, fail(fmt.Sprintf("el ID provisional %q lo declaran dos requisitos ADDED del cambio: %q (dominio %q) y %q (dominio %q)",
					provisional, first.name, first.domain, block.Name, doc.Domain), renameProvisionalContinuation)
			}
			if line, ok := scanned.existing[provisional]; ok {
				return IDMap{}, fail(fmt.Sprintf("el ID provisional %q coincide con un ID existente del documento (línea %d de %s): un ID provisional no debe coincidir con un ID existente", provisional, line, documentLabel(doc)), renameProvisionalContinuation)
			}
			stats := scanned.families[family]
			if stats == nil {
				return IDMap{}, fail(fmt.Sprintf("la familia de IDs %q del ID provisional %q no tiene ningún ID existente en el documento %s", family, provisional, documentLabel(doc)), seedContinuation)
			}
			if stats.next == nil {
				stats.next = new(big.Int).Set(stats.max)
			}
			stats.next.Add(stats.next, big.NewInt(1))
			number := stats.next.String()
			if pad := stats.width - len(number); pad > 0 {
				number = strings.Repeat("0", pad) + number
			}
			declared[provisional] = declaration{domain: doc.Domain, name: block.Name}
			assignments = append(assignments, IDAssignment{Provisional: provisional, Definitive: family + number, Domain: doc.Domain})
			names = append(names, declaration{domain: doc.Domain, name: block.Name})
		}
	}

	// (d): un provisional no puede coincidir con el definitivo de OTRO requisito.
	for i, a := range assignments {
		for j, b := range assignments {
			if i == j || a.Provisional != b.Definitive {
				continue
			}
			return IDMap{}, &ComposeError{
				Domain: a.Domain, Section: sectionAdded, Requirement: names[i].name,
				Reason: fmt.Sprintf("la sustitución sería ambigua: el ID provisional %q coincide con el ID definitivo %q que recibe el requisito ADDED %q (ID provisional %q)",
					a.Provisional, b.Definitive, names[j].name, b.Provisional),
				Continuation: renameProvisionalContinuation,
			}
		}
	}
	return newIDMap(assignments), nil
}

func documentLabel(doc PlannedDocument) string {
	if doc.RelPath != "" {
		return doc.RelPath
	}
	return fmt.Sprintf("<dominio %s>", doc.Domain)
}
