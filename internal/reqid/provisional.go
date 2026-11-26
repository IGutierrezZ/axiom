package reqid

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Códigos estables de ProvisionalError.Code.
const (
	// CodeProvisionalMalformed: el token no cumple la gramática o sus componentes no casan
	// con el patrón.
	CodeProvisionalMalformed = "provisional_malformed"
	// CodeProvisionalNewSeriesUnsupported: etiqueta de serie nueva con un patrón sin `<n>`.
	CodeProvisionalNewSeriesUnsupported = "provisional_new_series_unsupported"
)

// provisionalPrefix es el literal que abre un provisional. Es la única aparición de la
// forma en código de producción (arch_test.go).
const provisionalPrefix = "prov:"

// ProvisionalText escribe un provisional con la clave y el ordinal dados, tal como se
// teclean. Los mensajes, la ayuda y las continuaciones de los demás paquetes lo usan para
// citar la forma (con marcadores como `<serie>`) sin repetir el literal fuera de este
// paquete; no valida nada.
func ProvisionalText(key, ordinal string) string {
	return provisionalPrefix + key + ":" + ordinal
}

var (
	// provisionalLax es la forma laxa de un token provisional.
	provisionalLax = regexp.MustCompile(`prov:[A-Za-z0-9-]*:[0-9]+`)
	// provisionalTitle extrae `(prov:…)` de un título.
	provisionalTitle = regexp.MustCompile(`\((prov:[A-Za-z0-9-]*:[0-9]+)\)`)
	// provisionalForm es la forma laxa anclada, con la clave y el ordinal en grupos.
	provisionalForm = regexp.MustCompile(`^prov:([A-Za-z0-9-]*):([0-9]+)$`)

	componentDomain = regexp.MustCompile(`^[A-Z][A-Z0-9]*$`)
	componentSeries = regexp.MustCompile(`^[0-9]+$`)
	componentLabel  = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	ordinalForm     = regexp.MustCompile(`^[1-9][0-9]*$`)
)

// ProvisionalError describe un token con la forma de un provisional que no cumple la
// gramática o cuyos componentes no casan con el patrón.
type ProvisionalError struct {
	Token  string
	Code   string
	Reason string
}

func (e *ProvisionalError) Error() string {
	return fmt.Sprintf("el identificador provisional %q no es válido: %s", e.Token, e.Reason)
}

// Provisional es un identificador provisional ya validado contra un patrón: lleva la
// serie fijada (dominio y número de serie) o, en la posición de `<n>`, una etiqueta de
// serie nueva.
type Provisional struct {
	pattern  Pattern
	token    string
	vals     []string // valores de los marcadores `<DOM>` y `<n>`, en orden; la etiqueta, si la hay, en labelAt
	labelAt  int      // índice en vals de la etiqueta de serie nueva; -1 si la serie está fijada
	seriesAt int      // índice en vals del número de serie; -1 si el patrón no tiene `<n>`
}

// String devuelve el token tal cual se escribió.
func (p Provisional) String() string { return p.token }

// Label devuelve la etiqueta de serie nueva, si el provisional la lleva.
func (p Provisional) Label() (string, bool) {
	if p.labelAt < 0 {
		return "", false
	}
	return p.vals[p.labelAt], true
}

// Series devuelve la serie fijada del provisional; falso si lleva una etiqueta de serie
// nueva (la serie aún no existe).
func (p Provisional) Series() (Series, bool) {
	if p.labelAt >= 0 {
		return Series{}, false
	}
	return p.pattern.buildSeries(p.vals), true
}

// ProvisionalInTitle devuelve el último `(prov:…)` del título, con la forma laxa; la
// validación contra el patrón la hace Pattern.ParseProvisional.
func ProvisionalInTitle(title string) (token string, ok bool) {
	for _, m := range provisionalTitle.FindAllStringSubmatch(title, -1) {
		token, ok = m[1], true
	}
	return token, ok
}

// Span es una aparición de un token provisional en un texto: [Start, End) en bytes. Lax
// dice que el token tiene la forma laxa pero no cumple la gramática general.
type Span struct {
	Start, End int
	Token      string
	Lax        bool
}

// ScanProvisional devuelve los tokens provisionales de un texto en orden de aparición.
// Un token es completo cuando no lo precede ni lo sigue una letra, una cifra, `_` o `-`.
func ScanProvisional(text string) []Span {
	var spans []Span
	for _, loc := range provisionalLax.FindAllStringIndex(text, -1) {
		if !tokenBounded(text, loc[0], loc[1]) {
			continue
		}
		token := text[loc[0]:loc[1]]
		spans = append(spans, Span{Start: loc[0], End: loc[1], Token: token, Lax: !generalGrammar(token)})
	}
	return spans
}

// ScanProvisionalRefs devuelve, como ScanProvisional, los tokens provisionales de un texto
// Markdown, salvo los que están dentro de código: un bloque vallado (``` o ~~~) o un tramo
// de código en línea. Un token dentro de código es un ejemplo de la forma, no una
// referencia a un requisito, y no se valida.
func ScanProvisionalRefs(text string) []Span {
	spans := ScanProvisional(text)
	if len(spans) == 0 {
		return nil
	}
	code := codeRanges(text)
	var out []Span
	for _, s := range spans {
		if !insideRanges(code, s.Start) {
			out = append(out, s)
		}
	}
	return out
}

// codeRanges devuelve los tramos [inicio, fin) en bytes de un texto Markdown que son
// código: las líneas de un bloque vallado, incluidas su apertura y su cierre, y los tramos
// de código en línea de las demás líneas. Una valla se abre con al menos tres comillas invertidas o `~`
// (con hasta tres espacios delante) y se cierra con el mismo carácter, una tirada al menos
// igual de larga y sin texto detrás; sin cierre abarca hasta el final. Un tramo de código en
// línea se abre con una tirada de comillas invertidas y se cierra con otra de la misma
// longitud en la misma línea; una tirada sin cierre es literal.
func codeRanges(text string) [][2]int {
	var ranges [][2]int
	inFence, fenceChar, fenceLen := false, byte(0), 0
	for pos := 0; pos < len(text); {
		end := strings.IndexByte(text[pos:], '\n')
		next := len(text)
		if end < 0 {
			end = len(text)
		} else {
			end += pos
			next = end + 1
		}
		line := strings.TrimSuffix(text[pos:end], "\r")
		ch, n, hasInfo, ok := fenceMarker(line)
		switch {
		case inFence:
			ranges = append(ranges, [2]int{pos, next})
			if ok && ch == fenceChar && n >= fenceLen && !hasInfo {
				inFence = false
			}
		case ok:
			inFence, fenceChar, fenceLen = true, ch, n
			ranges = append(ranges, [2]int{pos, next})
		default:
			ranges = append(ranges, inlineCode(line, pos)...)
		}
		pos = next
	}
	return ranges
}

// fenceMarker dice si la línea abre o cierra una valla: el carácter, la longitud de la
// tirada y si lleva texto detrás (una apertura puede llevar información; un cierre no).
func fenceMarker(line string) (ch byte, length int, hasInfo, ok bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || trimmed == "" || (trimmed[0] != '`' && trimmed[0] != '~') {
		return 0, 0, false, false
	}
	ch = trimmed[0]
	for length < len(trimmed) && trimmed[length] == ch {
		length++
	}
	if length < 3 {
		return 0, 0, false, false
	}
	return ch, length, strings.TrimSpace(trimmed[length:]) != "", true
}

// inlineCode devuelve los tramos de código en línea de una línea, desplazados por offset.
func inlineCode(line string, offset int) [][2]int {
	var ranges [][2]int
	runAt := func(i int) int {
		n := 0
		for i+n < len(line) && line[i+n] == '`' {
			n++
		}
		return n
	}
	for i := 0; i < len(line); {
		if line[i] != '`' {
			i++
			continue
		}
		open := runAt(i)
		closed := false
		for j := i + open; j < len(line); {
			if line[j] != '`' {
				j++
				continue
			}
			n := runAt(j)
			if n == open {
				ranges = append(ranges, [2]int{offset + i, offset + j + n})
				i, closed = j+n, true
				break
			}
			j += n
		}
		if !closed {
			i += open
		}
	}
	return ranges
}

// insideRanges dice si pos cae en alguno de los tramos [inicio, fin).
func insideRanges(ranges [][2]int, pos int) bool {
	for _, r := range ranges {
		if pos >= r[0] && pos < r[1] {
			return true
		}
	}
	return false
}

// generalGrammar comprueba la gramática de D1 sin patrón.
func generalGrammar(token string) bool {
	m := provisionalForm.FindStringSubmatch(token)
	if m == nil || !ordinalForm.MatchString(m[2]) {
		return false
	}
	if m[1] == "" {
		return true
	}
	for _, c := range strings.Split(m[1], "-") {
		if !componentDomain.MatchString(c) && !componentSeries.MatchString(c) && !componentLabel.MatchString(c) {
			return false
		}
	}
	return true
}

// maskProvisionals sustituye por espacios los tokens provisionales del texto, de modo que
// la búsqueda de candidatos de un identificador no los vea.
func maskProvisionals(text string) string {
	spans := ScanProvisional(text)
	if len(spans) == 0 {
		return text
	}
	b := []byte(text)
	for _, s := range spans {
		for i := s.Start; i < s.End; i++ {
			b[i] = ' '
		}
	}
	return string(b)
}

// ParseProvisional valida un token provisional contra el patrón: los componentes de la
// clave deben coincidir en número, orden y clase con los marcadores `<DOM>` y `<n>`.
// Devuelve *ProvisionalError si no.
func (p Pattern) ParseProvisional(token string) (Provisional, error) {
	fail := func(code, format string, args ...any) (Provisional, error) {
		return Provisional{}, &ProvisionalError{Token: token, Code: code, Reason: fmt.Sprintf(format, args...)}
	}
	m := provisionalForm.FindStringSubmatch(token)
	if m == nil {
		return fail(CodeProvisionalMalformed, "no tiene la forma `%s<clave>:<ordinal>`", provisionalPrefix)
	}
	if !ordinalForm.MatchString(m[2]) {
		return fail(CodeProvisionalMalformed, "el ordinal %q debe ser un entero positivo sin ceros iniciales", m[2])
	}
	if _, err := strconv.Atoi(m[2]); err != nil {
		return fail(CodeProvisionalMalformed, "el ordinal %q no cabe en un entero", m[2])
	}
	var comps []string
	if m[1] != "" {
		comps = strings.Split(m[1], "-")
	}
	var kinds []segKind
	for _, s := range p.get().segs {
		if s.kind == segDomain || s.kind == segSeries {
			kinds = append(kinds, s.kind)
		}
	}
	hasSeries := false
	for _, k := range kinds {
		hasSeries = hasSeries || k == segSeries
	}
	out := Provisional{pattern: p, token: token, vals: comps, labelAt: -1, seriesAt: -1}
	for i, k := range kinds {
		if k == segSeries {
			out.seriesAt = i
		}
	}
	for _, c := range comps {
		if !componentDomain.MatchString(c) && !componentSeries.MatchString(c) && !componentLabel.MatchString(c) {
			return fail(CodeProvisionalMalformed, "el componente %q de la clave no es un dominio, un número de serie ni una etiqueta", c)
		}
	}
	if len(comps) != len(kinds) {
		for _, c := range comps {
			if componentLabel.MatchString(c) && !hasSeries {
				return fail(CodeProvisionalNewSeriesUnsupported, "el patrón %s no tiene `<n>`: una etiqueta de serie nueva no es posible; fija el dominio en el provisional", p.Template())
			}
		}
		return fail(CodeProvisionalMalformed, "la clave tiene %d componentes y el patrón %s espera %d", len(comps), p.Template(), len(kinds))
	}
	for i, k := range kinds {
		c := comps[i]
		switch {
		case k == segDomain && componentDomain.MatchString(c):
		case k == segSeries && componentSeries.MatchString(c):
			if _, err := strconv.Atoi(c); err != nil {
				return fail(CodeProvisionalMalformed, "la serie %q no cabe en un entero", c)
			}
		case k == segSeries && componentLabel.MatchString(c):
			out.labelAt = i
		case componentLabel.MatchString(c) && !hasSeries:
			return fail(CodeProvisionalNewSeriesUnsupported, "el patrón %s no tiene `<n>`: una etiqueta de serie nueva no es posible; fija el dominio en el provisional", p.Template())
		default:
			return fail(CodeProvisionalMalformed, "el componente %q no es del tipo que el patrón %s espera en esa posición", c, p.Template())
		}
	}
	return out, nil
}
