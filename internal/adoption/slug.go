package adoption

import (
	"strconv"
	"strings"
	"unicode"
)

// Slug devuelve el ancla de un encabezado al estilo GitHub: minúsculas, letras Unicode,
// dígitos, `-` y `_` se conservan, cada espacio pasa a `-` y el resto de la puntuación
// se descarta. No resuelve duplicados: para eso está el slugger de cada fichero.
func Slug(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '-', r == '_':
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteByte('-')
		}
	}
	return b.String()
}

// slugger asigna las anclas de los encabezados de un fichero: el primero conserva su
// slug y cada repetido recibe `-1`, `-2`… (sin chocar con un slug ya asignado).
type slugger struct {
	used   map[string]bool
	counts map[string]int
}

func newSlugger() *slugger {
	return &slugger{used: map[string]bool{}, counts: map[string]int{}}
}

func (s *slugger) next(title string) string {
	base := Slug(title)
	anchor := base
	for s.used[anchor] {
		s.counts[base]++
		anchor = base + "-" + strconv.Itoa(s.counts[base])
	}
	s.used[anchor] = true
	return anchor
}
