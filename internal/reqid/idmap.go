package reqid

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Assignment es una sustitución de un identificador por otro: de un provisional a su
// definitivo o, en una reasignación, de un definitivo a otro.
type Assignment struct {
	Provisional string
	Definitive  string
}

// IDMap sustituye identificadores por token completo en un texto.
type IDMap struct {
	pairs []Assignment // más largo primero
}

// NewIDMap crea el mapa. Con una clave repetida vale la primera.
func NewIDMap(pairs []Assignment) IDMap {
	seen := map[string]bool{}
	var kept []Assignment
	for _, a := range pairs {
		if a.Provisional == "" || seen[a.Provisional] {
			continue
		}
		seen[a.Provisional] = true
		kept = append(kept, a)
	}
	sort.SliceStable(kept, func(i, j int) bool { return len(kept[i].Provisional) > len(kept[j].Provisional) })
	return IDMap{pairs: kept}
}

// Count devuelve el número de sustituciones del mapa; un mapa vacío no reescribe nada.
func (m IDMap) Count() int { return len(m.pairs) }

// Substitute sustituye, en una sola pasada, cada clave del mapa que aparezca como token
// completo (ni precedida ni seguida de una letra, una cifra, `_` o `-`). Entre dos claves
// que empiezan en el mismo punto gana la más larga, y el resultado de una sustitución no
// se vuelve a sustituir.
func (m IDMap) Substitute(text string) string {
	if len(m.pairs) == 0 {
		return text
	}
	var b strings.Builder
	last := 0
	for i := 0; i < len(text); {
		matched := false
		for _, a := range m.pairs {
			if !strings.HasPrefix(text[i:], a.Provisional) || !tokenBounded(text, i, i+len(a.Provisional)) {
				continue
			}
			b.WriteString(text[last:i])
			b.WriteString(a.Definitive)
			i += len(a.Provisional)
			last = i
			matched = true
			break
		}
		if !matched {
			i++
		}
	}
	b.WriteString(text[last:])
	return b.String()
}

// tokenBounded dice si text[start:end] queda delimitado: ni precedido ni seguido de una
// letra, una cifra, `_` o `-`.
func tokenBounded(text string, start, end int) bool {
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(text[:start])
		if isTokenRune(r) {
			return false
		}
	}
	if end < len(text) {
		r, _ := utf8.DecodeRuneInString(text[end:])
		if isTokenRune(r) {
			return false
		}
	}
	return true
}

func isTokenRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}
