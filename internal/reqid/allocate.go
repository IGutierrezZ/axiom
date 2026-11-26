package reqid

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// AllocationError describe una asignación que no puede formar un identificador válido:
// un número que desborda o un identificador ocupado que no es válido para el patrón.
type AllocationError struct {
	ID     string
	Reason string
}

func (e *AllocationError) Error() string {
	if e.ID == "" {
		return "no se puede asignar un identificador: " + e.Reason
	}
	return fmt.Sprintf("no se puede asignar un identificador a partir de %q: %s", e.ID, e.Reason)
}

// buildSeries compone la serie a partir de los valores de los marcadores `<DOM>` y `<n>`
// en orden de plantilla, igual que Series con un identificador de esa serie.
func (p Pattern) buildSeries(vals []string) Series {
	var canon strings.Builder
	var key []string
	numAt, k := 0, 0
	for _, s := range p.get().segs {
		switch s.kind {
		case segLiteral:
			canon.WriteString(s.text)
		case segNumber:
			numAt = canon.Len()
		case segDomain:
			key = append(key, vals[k])
			canon.WriteString(vals[k])
			k++
		case segSeries:
			v := vals[k]
			if n, err := strconv.Atoi(v); err == nil {
				v = strconv.Itoa(n)
			}
			k++
			key = append(key, v)
			canon.WriteString(v)
		}
	}
	text := canon.String()
	return Series{prefix: text[:numAt], suffix: text[numAt:], key: strings.Join(key, "-")}
}

// SeriesOf reconstruye una serie a partir de su clave (`Series.Key()`: `30`, `USU`,
// `USU-3`, o vacía en una serie única). Falso si la clave no casa con los marcadores del
// patrón.
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

// Allocator asigna el siguiente número libre de una serie y la siguiente serie libre de
// una etiqueta de serie nueva. Solo cuenta lo que se le dice que está ocupado y lo que él
// mismo asigna; no lee ficheros.
type Allocator struct {
	pattern Pattern
	max     map[Series]int // mayor número ocupado; una serie presente con 0 está tomada sin números
}

// NewAllocator crea un asignador vacío para el patrón.
func NewAllocator(p Pattern) *Allocator {
	return &Allocator{pattern: p, max: map[Series]int{}}
}

// Clone devuelve una copia independiente del asignador.
func (a *Allocator) Clone() *Allocator {
	c := &Allocator{pattern: a.pattern, max: make(map[Series]int, len(a.max))}
	for s, n := range a.max {
		c.max[s] = n
	}
	return c
}

// Occupy marca un identificador definitivo como ocupado. Ocuparlo varias veces no altera
// el máximo de su serie. Un identificador no válido para el patrón da *AllocationError.
func (a *Allocator) Occupy(id string) error {
	series, ok := a.pattern.Series(id)
	number, okNumber := a.pattern.Number(id)
	if !ok || !okNumber {
		return &AllocationError{ID: id, Reason: "no es un identificador válido para el patrón " + a.pattern.Template()}
	}
	if current, seen := a.max[series]; !seen || number > current {
		a.max[series] = number
	}
	return nil
}

func (a *Allocator) taken(s Series) bool {
	_, ok := a.max[s]
	return ok
}

// Next devuelve el siguiente identificador libre de la serie (el máximo ocupado más uno,
// con el relleno de la plantilla; en una serie vacía, el `1`) y lo da por ocupado.
func (a *Allocator) Next(s Series) (string, error) {
	current := a.max[s]
	if current == math.MaxInt {
		return "", &AllocationError{Reason: fmt.Sprintf("el número de la serie %s desborda", s)}
	}
	id := a.pattern.Format(s, current+1)
	if !a.pattern.Valid(id) {
		return "", &AllocationError{ID: id, Reason: "el identificador resultante no es válido para el patrón " + a.pattern.Template()}
	}
	a.max[s] = current + 1
	return id, nil
}

// NewSeries devuelve la serie de una etiqueta de serie nueva: el siguiente `<n>` libre
// entre las series que comparten los demás componentes (el máximo más uno) y la da por
// tomada, de modo que otra etiqueta no recae en ella. Las series que fija el propio
// cambio deben haberse asignado antes con Next.
func (a *Allocator) NewSeries(pv Provisional) (Series, error) {
	if pv.labelAt < 0 || pv.seriesAt < 0 {
		return Series{}, &AllocationError{ID: pv.token, Reason: "el provisional no lleva una etiqueta de serie nueva"}
	}
	top := 0
	for s := range a.max {
		vals := strings.Split(s.key, "-")
		if len(vals) != len(pv.vals) {
			continue
		}
		same := true
		for i, v := range vals {
			if i != pv.seriesAt && v != pv.vals[i] {
				same = false
			}
		}
		if !same {
			continue
		}
		if n, err := strconv.Atoi(vals[pv.seriesAt]); err == nil && n > top {
			top = n
		}
	}
	if top == math.MaxInt {
		return Series{}, &AllocationError{ID: pv.token, Reason: "el número de serie desborda"}
	}
	vals := append([]string(nil), pv.vals...)
	vals[pv.seriesAt] = strconv.Itoa(top + 1)
	s := a.pattern.buildSeries(vals)
	if !a.taken(s) {
		a.max[s] = 0
	}
	return s, nil
}
