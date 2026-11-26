// Package reqid es el puerto único de identificadores de requisito: el único lugar
// donde existe la forma de un identificador. Compila la plantilla de segmentos
// declarada en `sdd.id_pattern` de `axiom.yaml` y ofrece validación, extracción del
// título, detección laxa de candidatos, serie, número, formato del siguiente
// identificador, notaciones para mensajes y el identificador sintético de livingdoc.
//
// Es un paquete hoja: solo usa la biblioteca estándar (REQ-44.4).
package reqid

import (
	"fmt"
	"strconv"
	"strings"
)

// Pattern es una plantilla de identificador compilada e inmutable. Es comparable y su
// valor cero se comporta exactamente como Default(): `REQ-<n>.<m>`.
type Pattern struct{ c *compiled }

// defaultTemplate es la plantilla que rige cuando el proyecto no declara ninguna.
const defaultTemplate = "REQ-<n>.<m>"

var defaultCompiled = mustCompile(defaultTemplate)

func mustCompile(template string) *compiled {
	c, err := compileTemplate(template)
	if err != nil {
		panic(err)
	}
	return c
}

// Default devuelve la plantilla por defecto, `REQ-<n>.<m>` (el valor cero de Pattern).
func Default() Pattern { return Pattern{} }

// Parse compila una plantilla. Devuelve *TemplateError si no cumple la gramática.
func Parse(template string) (Pattern, error) {
	if template == defaultTemplate {
		return Pattern{}, nil
	}
	c, err := parseCached(template)
	if err != nil {
		return Pattern{}, err
	}
	return Pattern{c: c}, nil
}

// TemplateError describe una plantilla que no cumple la gramática.
type TemplateError struct {
	Template string
	Reason   string
}

func (e *TemplateError) Error() string {
	return fmt.Sprintf("la plantilla %q no es válida: %s", e.Template, e.Reason)
}

func (p Pattern) get() *compiled {
	if p.c == nil {
		return defaultCompiled
	}
	return p.c
}

// Template devuelve la plantilla tal cual: `REQ-<n>.<m>`, `RF-<DOM>-<NN>`.
func (p Pattern) Template() string { return p.get().template }

// Compact devuelve la plantilla sin ángulos: `REQ-n.m`, `RF-DOM-NN`.
func (p Pattern) Compact() string { return p.get().compact }

// SeriesCompact devuelve la notación de serie: `REQ-n.*`, `RF-DOM-*`.
func (p Pattern) SeriesCompact() string { return p.get().seriesCompact }

// Valid dice si id encaja entero con la plantilla y su número cabe en int.
func (p Pattern) Valid(id string) bool {
	_, ok := p.match(id)
	return ok
}

// match devuelve los límites de submatches de un identificador válido.
func (p Pattern) match(id string) ([]int, bool) {
	c := p.get()
	loc := c.valid.FindStringSubmatchIndex(id)
	if loc == nil {
		return nil, false
	}
	g := c.groupOf[c.numSeg]
	if _, err := strconv.Atoi(id[loc[2*g]:loc[2*g+1]]); err != nil {
		return nil, false
	}
	return loc, true
}

// InTitle devuelve el último `(ID)` válido del título, o "".
func (p Pattern) InTitle(title string) string {
	id := ""
	for _, m := range p.get().inTitle.FindAllStringSubmatch(title, -1) {
		if p.Valid(m[1]) {
			id = m[1]
		}
	}
	return id
}

// Scan devuelve los candidatos del texto, válidos y mal formados, sin repetidos y en
// orden de aparición. Un candidato empieza, en límite de palabra, por el literal
// inicial de la plantilla y no incluye la puntuación final.
func (p Pattern) Scan(text string) (valid, malformed []string) {
	seenValid := map[string]bool{}
	seenMalformed := map[string]bool{}
	// Los tokens provisionales son otra clase de token: se enmascaran para que ni su clave
	// (`prov:RF-3:1` con `RF-<DOM>-<n>.<m>`) ni nada de su forma sea un candidato (REQ-47.1).
	for _, token := range p.get().candidate.FindAllString(maskProvisionals(text), -1) {
		if p.Valid(token) {
			if !seenValid[token] {
				seenValid[token] = true
				valid = append(valid, token)
			}
			continue
		}
		if !seenMalformed[token] {
			seenMalformed[token] = true
			malformed = append(malformed, token)
		}
	}
	return valid, malformed
}

// TrailingCandidate es la lectura laxa de livingdoc: separa del título un
// `(<literal inicial>…)` final sin validarlo. Si no lo hay devuelve el título sin
// espacios en los extremos y ok = false.
func (p Pattern) TrailingCandidate(title string) (rest, id string, ok bool) {
	if m := p.get().trailing.FindStringSubmatch(title); m != nil {
		return m[1], m[2], true
	}
	return strings.TrimSpace(title), "", false
}

// Series devuelve la serie del identificador (el ID sin su número). Cada `<n>` se
// identifica por su valor entero, como hacía `strconv.Atoi` con el patrón por defecto:
// `REQ-09.1` y `REQ-9.2` son la serie `REQ-9.*`, y un `<n>` que desborda int no forma
// serie. `<DOM>` se conserva tal cual.
func (p Pattern) Series(id string) (Series, bool) {
	loc, ok := p.match(id)
	if !ok {
		return Series{}, false
	}
	c := p.get()
	var canon strings.Builder
	var key []string
	last, numStart, numEnd := 0, 0, 0
	for i, s := range c.segs {
		if i != c.numSeg && s.kind != segDomain && s.kind != segSeries {
			continue
		}
		gi := c.groupOf[i]
		start, end := loc[2*gi], loc[2*gi+1]
		canon.WriteString(id[last:start])
		last = end
		value := id[start:end]
		switch {
		case i == c.numSeg:
			numStart = canon.Len()
			canon.WriteString(value)
			numEnd = canon.Len()
			continue
		case s.kind == segSeries:
			n, err := strconv.Atoi(value)
			if err != nil {
				return Series{}, false
			}
			value = strconv.Itoa(n)
		}
		key = append(key, value)
		canon.WriteString(value)
	}
	canon.WriteString(id[last:])
	text := canon.String()
	return Series{prefix: text[:numStart], suffix: text[numEnd:], key: strings.Join(key, "-")}, true
}

// Number devuelve el valor entero del segmento de número del identificador.
func (p Pattern) Number(id string) (int, bool) {
	loc, ok := p.match(id)
	if !ok {
		return 0, false
	}
	g := p.get().groupOf[p.get().numSeg]
	n, err := strconv.Atoi(id[loc[2*g]:loc[2*g+1]])
	return n, err == nil
}

// Format compone el identificador de número n (n >= 0) en una serie obtenida de este
// patrón, con el relleno que declara la plantilla y sin truncar.
func (p Pattern) Format(s Series, n int) string {
	number := strconv.Itoa(n)
	if w := p.get().numWidth; w > 0 {
		number = fmt.Sprintf("%0*d", w, n)
	}
	return s.prefix + number + s.suffix
}
