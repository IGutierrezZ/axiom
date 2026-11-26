package htmlmd

import (
	"html"
	"strings"
)

type tokenKind int

const (
	tokText tokenKind = iota
	tokStart
	tokEnd
)

// tagAttrs son los únicos atributos que se leen de una etiqueta. Span indica que colspan o rowspan
// están presentes con un valor distinto de 1.
type tagAttrs struct {
	Href, Src, Alt string
	Span           bool
}

type token struct {
	kind        tokenKind
	name        string // en minúsculas; vacío en un token de texto
	text        string // texto ya desescapado (solo tokText)
	attrs       tagAttrs
	selfClosing bool
}

// discarded son los elementos cuyo contenido completo se descarta. Si no tienen cierre, se descarta
// hasta el final de la entrada.
var discarded = map[string]bool{
	"script": true, "style": true, "noscript": true, "template": true, "iframe": true, "object": true,
	"svg": true, "math": true, "textarea": true, "title": true, "xmp": true, "noembed": true,
	"noframes": true,
}

// tokenizer es un tokenizador conservador: ante cualquier construcción dudosa descarta o deja el
// texto, nunca emite una etiqueta. Una etiqueta sin terminador (fin de entrada dentro de ella) hace
// que se descarte el resto, como un navegador; así el recorrido es lineal.
type tokenizer struct {
	src string
	low string // src con ASCII en minúsculas, de la misma longitud
	pos int
}

func newTokenizer(src string) *tokenizer {
	return &tokenizer{src: src, low: asciiLower(src)}
}

func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

// markupAt indica si en i empieza una construcción de marcado: `<` seguido de letra, `/`, `!` o `?`.
func (t *tokenizer) markupAt(i int) bool {
	if i+1 >= len(t.src) || t.src[i] != '<' {
		return false
	}
	c := t.src[i+1]
	return isLetter(c) || c == '/' || c == '!' || c == '?'
}

func (t *tokenizer) nextMarkup(from int) int {
	for i := from; i < len(t.src); i++ {
		if t.markupAt(i) {
			return i
		}
	}
	return len(t.src)
}

// next devuelve el siguiente token útil, o false al terminar.
func (t *tokenizer) next() (token, bool) {
	for t.pos < len(t.src) {
		if !t.markupAt(t.pos) {
			end := t.nextMarkup(t.pos + 1)
			text := t.src[t.pos:end]
			t.pos = end
			return token{kind: tokText, text: html.UnescapeString(text)}, true
		}
		switch {
		case strings.HasPrefix(t.src[t.pos:], "<!--"):
			t.pos = t.skipPast("-->", t.pos+4)
		case t.src[t.pos+1] == '!' || t.src[t.pos+1] == '?':
			t.pos = t.skipPast(">", t.pos+2)
		case t.src[t.pos+1] == '/':
			if t.pos+2 < len(t.src) && isLetter(t.src[t.pos+2]) {
				name, _, _, end, ok := t.parseTag(t.pos + 2)
				if !ok {
					t.pos = len(t.src)
					continue
				}
				t.pos = end
				return token{kind: tokEnd, name: name}, true
			}
			t.pos = t.skipPast(">", t.pos+2) // cierre falso: comentario "bogus"
		default:
			name, attrs, self, end, ok := t.parseTag(t.pos + 1)
			if !ok {
				t.pos = len(t.src)
				continue
			}
			t.pos = end
			if discarded[name] {
				t.skipRaw(name)
				continue
			}
			return token{kind: tokStart, name: name, attrs: attrs, selfClosing: self}, true
		}
	}
	return token{}, false
}

// skipPast devuelve la posición posterior a la primera aparición de marker desde from, o el fin.
func (t *tokenizer) skipPast(marker string, from int) int {
	if from > len(t.src) {
		return len(t.src)
	}
	i := strings.Index(t.src[from:], marker)
	if i < 0 {
		return len(t.src)
	}
	return from + i + len(marker)
}

// skipRaw descarta el contenido de un elemento descartado hasta su cierre inclusive.
func (t *tokenizer) skipRaw(name string) {
	i := strings.Index(t.low[t.pos:], "</"+name)
	if i < 0 {
		t.pos = len(t.src)
		return
	}
	t.pos = t.skipPast(">", t.pos+i)
}

// parseTag lee nombre y atributos de la etiqueta cuyo nombre empieza en i. Devuelve la posición
// posterior al `>`. ok es false si la etiqueta no termina antes del fin de la entrada.
func (t *tokenizer) parseTag(i int) (name string, attrs tagAttrs, selfClosing bool, end int, ok bool) {
	s := t.src
	start := i
	for i < len(s) && !isSpaceByte(s[i]) && s[i] != '/' && s[i] != '>' {
		i++
	}
	name = t.low[start:i]
	for i < len(s) {
		c := s[i]
		switch {
		case isSpaceByte(c):
			i++
		case c == '>':
			return name, attrs, selfClosing, i + 1, true
		case c == '/':
			if i+1 < len(s) && s[i+1] == '>' {
				selfClosing = true
			}
			i++
		default:
			ns := i
			for i < len(s) && !isSpaceByte(s[i]) && s[i] != '=' && s[i] != '>' && s[i] != '/' {
				i++
			}
			if i == ns { // carácter suelto (comilla o `=` sin nombre): se salta
				i++
				continue
			}
			attr := t.low[ns:i]
			for i < len(s) && isSpaceByte(s[i]) {
				i++
			}
			value := ""
			if i < len(s) && s[i] == '=' {
				i++
				for i < len(s) && isSpaceByte(s[i]) {
					i++
				}
				if i < len(s) && (s[i] == '"' || s[i] == '\'') {
					q := s[i]
					j := strings.IndexByte(s[i+1:], q)
					if j < 0 {
						return "", tagAttrs{}, false, 0, false
					}
					value = s[i+1 : i+1+j]
					i += j + 2
				} else {
					vs := i
					for i < len(s) && !isSpaceByte(s[i]) && s[i] != '>' {
						i++
					}
					value = s[vs:i]
				}
			}
			attrs.set(attr, value)
		}
	}
	return "", tagAttrs{}, false, 0, false
}

func (a *tagAttrs) set(name, value string) {
	switch name {
	case "href":
		a.Href = html.UnescapeString(value)
	case "src":
		a.Src = html.UnescapeString(value)
	case "alt":
		a.Alt = html.UnescapeString(value)
	case "colspan", "rowspan":
		if v := strings.TrimSpace(value); v != "" && v != "1" {
			a.Span = true
		}
	}
}
