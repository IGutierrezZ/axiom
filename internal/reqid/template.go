package reqid

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// maxTemplateLen es la longitud máxima de una plantilla (D-01, regla 1).
const maxTemplateLen = 64

// initialLiteral reconoce el literal inicial de una plantilla: una letra mayúscula,
// mayúsculas o cifras y un separador final. Es el ancla de la detección de candidatos.
var initialLiteral = regexp.MustCompile(`^[A-Z][A-Z0-9]*[-._]`)

type segKind int

const (
	segLiteral segKind = iota
	segDomain          // <DOM>
	segSeries          // <n>
	segNumber          // <m> o <NN>…<NNNNNN>
)

type segment struct {
	kind  segKind
	text  string // literal o marcador tal cual
	width int    // segNumber: 0 = sin relleno (<m>); k >= 2 = relleno a k cifras
}

// compiled es la plantilla compilada e inmutable que respalda a Pattern.
type compiled struct {
	template      string
	compact       string
	seriesCompact string
	segs          []segment
	numSeg        int // índice en segs del segmento de número
	numWidth      int
	valid         *regexp.Regexp // anclada, un grupo de captura por marcador
	inTitle       *regexp.Regexp // `(<id>)`, el ID completo en el grupo 1
	candidate     *regexp.Regexp
	trailing      *regexp.Regexp
	groupOf       map[int]int // índice de segmento -> índice de grupo de captura
}

var (
	cacheMu sync.Mutex
	cache   = map[string]*compiled{}
)

// compileTemplate valida la plantilla según la gramática de D-01 y la compila.
func compileTemplate(template string) (*compiled, error) {
	fail := func(format string, args ...any) (*compiled, error) {
		return nil, &TemplateError{Template: template, Reason: fmt.Sprintf(format, args...)}
	}
	if template == "" {
		return fail("está vacía")
	}
	if len(template) > maxTemplateLen {
		return fail("supera los %d caracteres", maxTemplateLen)
	}
	initial := initialLiteral.FindString(template)
	if initial == "" {
		return fail("debe empezar por un literal inicial: una letra mayúscula, mayúsculas o cifras y un separador (`-`, `.` o `_`), como `RF-`")
	}

	var segs []segment
	var lit strings.Builder
	flush := func() {
		if lit.Len() > 0 {
			segs = append(segs, segment{kind: segLiteral, text: lit.String()})
			lit.Reset()
		}
	}
	var domains, series, numbers int
	for i := 0; i < len(template); {
		c := template[i]
		switch {
		case c == '<':
			end := strings.IndexByte(template[i:], '>')
			next := strings.IndexByte(template[i+1:], '<')
			if end < 0 || (next >= 0 && next+1 < end) {
				return fail("el `<` de la posición %d queda sin cerrar con `>`", i+1)
			}
			marker := template[i : i+end+1]
			flush()
			switch name := marker[1 : len(marker)-1]; {
			case name == "DOM":
				domains++
				if domains > 1 {
					return fail("<DOM> solo puede aparecer una vez")
				}
				segs = append(segs, segment{kind: segDomain, text: marker})
			case name == "n":
				series++
				if series > 1 {
					return fail("<n> solo puede aparecer una vez")
				}
				segs = append(segs, segment{kind: segSeries, text: marker})
			case name == "m":
				numbers++
				if numbers > 1 {
					return fail("solo puede haber un único marcador de número")
				}
				segs = append(segs, segment{kind: segNumber, text: marker})
			case len(name) >= 2 && len(name) <= 6 && strings.Trim(name, "N") == "":
				numbers++
				if numbers > 1 {
					return fail("solo puede haber un único marcador de número")
				}
				segs = append(segs, segment{kind: segNumber, text: marker, width: len(name)})
			default:
				return fail("el marcador %s no existe; los admitidos son <DOM>, <n>, <m> y <NN> a <NNNNNN>", marker)
			}
			i += end + 1
		case c == '>':
			return fail("hay un `>` suelto en la posición %d", i+1)
		case (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-':
			lit.WriteByte(c)
			i++
		default:
			return fail("el literal contiene el carácter %q; solo admite mayúsculas, cifras, `.`, `_` y `-`", string(rune(c)))
		}
	}
	flush()
	if numbers == 0 {
		return fail("falta el marcador de número (`<m>` o `<NN>` a `<NNNNNN>`)")
	}

	for i := 1; i < len(segs); i++ {
		if segs[i].kind != segLiteral && segs[i-1].kind != segLiteral {
			return fail("los marcadores %s y %s no pueden ir seguidos; sepáralos con un literal", segs[i-1].text, segs[i].text)
		}
	}
	for i, s := range segs {
		if s.kind == segLiteral {
			continue
		}
		if i > 0 {
			last := segs[i-1].text[len(segs[i-1].text)-1]
			if inAlphabet(s.kind, last) {
				return fail("el literal %q pegado a %s termina con un carácter del alfabeto de ese marcador", segs[i-1].text, s.text)
			}
		}
		if i+1 < len(segs) {
			first := segs[i+1].text[0]
			if inAlphabet(s.kind, first) {
				return fail("el literal %q pegado a %s empieza con un carácter del alfabeto de ese marcador", segs[i+1].text, s.text)
			}
		}
	}

	c := &compiled{template: template, segs: segs}
	var compact, seriesCompact strings.Builder
	extras := strings.Builder{}
	seenExtra := map[byte]bool{}
	addExtras := func(text string) {
		for j := 0; j < len(text); j++ {
			b := text[j]
			if (b == '.' || b == '_' || b == '-') && !seenExtra[b] {
				seenExtra[b] = true
				// QuoteMeta no escapa '-': dentro de una clase formaría rangos.
				if b == '-' {
					extras.WriteString(`\-`)
				} else {
					extras.WriteString(regexp.QuoteMeta(string(b)))
				}
			}
		}
	}
	for i, s := range segs {
		switch s.kind {
		case segLiteral:
			compact.WriteString(s.text)
			seriesCompact.WriteString(s.text)
			if i == 0 {
				addExtras(s.text[len(initial):])
			} else {
				addExtras(s.text)
			}
		case segNumber:
			c.numSeg, c.numWidth = i, s.width
			compact.WriteString(strings.Trim(s.text, "<>"))
			seriesCompact.WriteString("*")
		default:
			compact.WriteString(strings.Trim(s.text, "<>"))
			seriesCompact.WriteString(strings.Trim(s.text, "<>"))
		}
	}
	c.compact = compact.String()
	c.seriesCompact = seriesCompact.String()

	body, groups := c.body(true)
	c.groupOf = groups
	plain, _ := c.body(false)
	c.valid = regexp.MustCompile(`^` + body + `$`)
	c.inTitle = regexp.MustCompile(`\((` + plain + `)\)`)
	q := regexp.QuoteMeta(initial)
	c.candidate = regexp.MustCompile(`\b` + q + `[0-9A-Za-z` + extras.String() + `]*[0-9A-Za-z]`)
	c.trailing = regexp.MustCompile(`^(.+?)\s*\((` + q + `[^)]+)\)\s*$`)
	return c, nil
}

// inAlphabet dice si un carácter de literal pertenece al alfabeto del marcador.
func inAlphabet(kind segKind, b byte) bool {
	digit := b >= '0' && b <= '9'
	if kind == segDomain {
		return digit || (b >= 'A' && b <= 'Z')
	}
	return digit
}

// body devuelve el cuerpo de la expresión de la plantilla; con capture, cada marcador
// va en un grupo de captura y se devuelve el índice de grupo de cada segmento.
func (c *compiled) body(capture bool) (string, map[int]int) {
	var b strings.Builder
	groups := map[int]int{}
	n := 0
	wrap := func(i int, expr string) {
		if capture {
			n++
			groups[i] = n
			b.WriteString("(" + expr + ")")
			return
		}
		b.WriteString("(?:" + expr + ")")
	}
	for i, s := range c.segs {
		switch s.kind {
		case segLiteral:
			b.WriteString(regexp.QuoteMeta(s.text))
		case segDomain:
			wrap(i, `[A-Z][A-Z0-9]*`)
		case segSeries:
			wrap(i, `[0-9]+`)
		case segNumber:
			if s.width == 0 {
				wrap(i, `[0-9]+`)
			} else {
				wrap(i, fmt.Sprintf(`[0-9]{%d}|[1-9][0-9]{%d,}`, s.width, s.width))
			}
		}
	}
	return b.String(), groups
}

// parseCached compila y memoriza por plantilla, de modo que dos Parse iguales dan
// valores Pattern iguales.
func parseCached(template string) (*compiled, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if c, ok := cache[template]; ok {
		return c, nil
	}
	c, err := compileTemplate(template)
	if err != nil {
		return nil, err
	}
	cache[template] = c
	return c, nil
}
