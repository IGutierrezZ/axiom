package changestart

import (
	"regexp"
	"strings"
	"unicode"
)

// neutralizedMarker sustituye a «AXIOM::» en el texto que procede del tracker.
const neutralizedMarker = "AXIOM(neutralizado):"

// inlineLimit es el tope de runas del texto del tracker que entra en un mensaje (D-28).
const inlineLimit = 64

// forgedDelimiter reconoce «AXIOM::» sin distinguir mayúsculas, aceptando los dos puntos de anchura
// completa y otros parecidos y cualquier espacio, saltos incluidos, entre las letras y los signos.
var forgedDelimiter = regexp.MustCompile(`(?i)axiom\s*[:：﹕꞉∶]\s*[:：﹕꞉∶]`)

// invisible informa de los caracteres de anchura cero y de control bidireccional que se eliminan:
// U+200B–U+200F, U+202A–U+202E, U+2060–U+2064, U+2066–U+2069, U+FEFF, y también U+00AD y U+061C,
// que permitirían escribir «AXIOM::» sin que lo parezca.
func invisible(r rune) bool {
	switch {
	case r >= 0x200B && r <= 0x200F, r >= 0x202A && r <= 0x202E, r >= 0x2060 && r <= 0x2064,
		r >= 0x2066 && r <= 0x2069, r == 0xFEFF, r == 0x00AD, r == 0x061C:
		return true
	}
	return false
}

// neutralize se aplica a todo texto del tracker antes de componer context/work-item.md: elimina los
// caracteres invisibles y sustituye cada «AXIOM::» (o su forma disfrazada) por AXIOM(neutralizado):.
// Así el contenido no puede cerrar el bloque de datos ni imitar la cabecera (D-10).
func neutralize(text string) string {
	text = strings.Map(func(r rune) rune {
		if invisible(r) {
			return -1
		}
		return r
	}, text)
	return forgedDelimiter.ReplaceAllString(text, neutralizedMarker)
}

// inlineNeutral deja el texto del tracker en una sola línea para un mensaje: neutraliza, cambia los
// caracteres de control y los saltos por un espacio, colapsa, recorta a 64 runas con «…» y lo
// envuelve en «» (D-28).
func inlineNeutral(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, neutralize(s))
	s = strings.Join(strings.Fields(s), " ")
	if runes := []rune(s); len(runes) > inlineLimit {
		s = string(runes[:inlineLimit]) + "…"
	}
	return "«" + s + "»"
}
