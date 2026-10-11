package specbaseline

import (
	"bytes"
	"strings"
)

// utf8BOM es la marca de orden de bytes de UTF-8.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// defaultEOL es el final de línea que se usa cuando un documento no tiene
// ninguna línea con final propio.
const defaultEOL = "\n"

// Line es una línea del documento sin su final de línea, con el final que tenía
// en el fichero: "\n", "\r\n", o vacío en la última línea sin salto (D-12). Un
// "\r" suelto no es un final de línea y forma parte del texto.
type Line struct {
	Text string
	EOL  string
}

// Document es un documento Markdown como serie de líneas. Conserva el BOM y el
// final de línea de cada línea, de modo que volver a serializar un documento sin
// editarlo reproduce sus bytes exactamente (REQ-24.45).
type Document struct {
	BOM   bool
	Lines []Line
}

// ParseDocument divide data en líneas. Detecta y retira el BOM UTF-8; Bytes lo
// vuelve a anteponer. Nunca se añade un BOM a un documento que no lo tenía.
func ParseDocument(data []byte) Document {
	var doc Document
	if bytes.HasPrefix(data, utf8BOM) {
		doc.BOM = true
		data = data[len(utf8BOM):]
	}
	for len(data) > 0 {
		newline := bytes.IndexByte(data, '\n')
		if newline < 0 {
			doc.Lines = append(doc.Lines, Line{Text: string(data)})
			break
		}
		text, eol := data[:newline], "\n"
		if len(text) > 0 && text[len(text)-1] == '\r' {
			text, eol = text[:len(text)-1], "\r\n"
		}
		doc.Lines = append(doc.Lines, Line{Text: string(text), EOL: eol})
		data = data[newline+1:]
	}
	return doc
}

// Bytes serializa el documento: BOM, si lo tenía, y cada línea con su final.
func (d Document) Bytes() []byte {
	size := 0
	for _, line := range d.Lines {
		size += len(line.Text) + len(line.EOL)
	}
	out := make([]byte, 0, size+len(utf8BOM))
	if d.BOM {
		out = append(out, utf8BOM...)
	}
	for _, line := range d.Lines {
		out = append(out, line.Text...)
		out = append(out, line.EOL...)
	}
	return out
}

// EOL devuelve el final de línea del documento: el de su primera línea con final
// propio, o "\n" si ninguna lo tiene. Se usa solo como respaldo cuando la línea
// de referencia no tiene final (última línea sin salto).
func (d Document) EOL() string {
	for _, line := range d.Lines {
		if line.EOL != "" {
			return line.EOL
		}
	}
	return defaultEOL
}

// Heading es un encabezado ATX de un documento.
type Heading struct {
	Index int    // índice de la línea, desde 0
	Level int    // de 1 a 6
	Title string // sin los `#` de apertura ni los de cierre opcionales
}

// Headings devuelve los encabezados ATX del documento fuera de bloques de
// código con vallas, en orden. Los encabezados setext no se reconocen (R33).
func (d Document) Headings() []Heading {
	return scanHeadings(d.Lines)
}

func scanHeadings(lines []Line) []Heading {
	fenced := fencedLines(lines)
	var headings []Heading
	for i, line := range lines {
		if fenced[i] {
			continue
		}
		if level, title, ok := parseATX(line.Text); ok {
			headings = append(headings, Heading{Index: i, Level: level, Title: title})
		}
	}
	return headings
}

// parseATX reconoce un encabezado ATX: entre 0 y 3 espacios iniciales, de 1 a 6
// `#` y después un espacio o el fin de línea (D-09). El título se devuelve sin
// los `#` de cierre opcionales.
func parseATX(text string) (level int, title string, ok bool) {
	rest := text
	indent := 0
	for indent < len(rest) && indent < 4 && rest[indent] == ' ' {
		indent++
	}
	if indent > 3 {
		return 0, "", false
	}
	rest = rest[indent:]
	for level < len(rest) && rest[level] == '#' {
		level++
	}
	if level < 1 || level > 6 {
		return 0, "", false
	}
	rest = rest[level:]
	if rest != "" && rest[0] != ' ' {
		return 0, "", false
	}
	return level, trimClosingSequence(strings.Trim(rest, " \t")), true
}

// trimClosingSequence retira la secuencia de cierre opcional de un encabezado
// ATX: una serie de `#` al final, separada del texto por un espacio (o que es
// todo el texto).
func trimClosingSequence(title string) string {
	trimmed := strings.TrimRight(title, "#")
	if trimmed == title {
		return title
	}
	if trimmed == "" {
		return ""
	}
	if last := trimmed[len(trimmed)-1]; last == ' ' || last == '\t' {
		return strings.TrimRight(trimmed, " \t")
	}
	return title
}

// fenceMarker dice si line abre o cierra una valla ``` o ~~~: el carácter de la
// valla, la longitud de su serie y si le sigue texto (una cadena de información,
// que solo puede llevar una apertura válida). Es la misma regla que aplica el
// motor nativo de composición (sddstatus.fenceMarker), que no se reutiliza para
// no tocar código compartido con upstream (D-08).
func fenceMarker(line string) (ch byte, length int, hasInfo, ok bool) {
	if len(line) < 3 || (line[0] != '`' && line[0] != '~') {
		return 0, 0, false, false
	}
	ch = line[0]
	for length < len(line) && line[length] == ch {
		length++
	}
	if length < 3 {
		return 0, 0, false, false
	}
	return ch, length, strings.TrimSpace(line[length:]) != "", true
}

// fencedLines marca las líneas que pertenecen a un bloque de código con vallas,
// incluidas la de apertura y la de cierre. El cierre exige el mismo carácter, una
// serie al menos igual de larga y ningún texto detrás; una valla sin cerrar se
// extiende hasta el final del documento.
func fencedLines(lines []Line) []bool {
	fenced := make([]bool, len(lines))
	inFence, fenceChar, fenceLen := false, byte(0), 0
	for i, line := range lines {
		if !inFence {
			if ch, n, _, ok := fenceMarker(line.Text); ok {
				inFence, fenceChar, fenceLen = true, ch, n
				fenced[i] = true
			}
			continue
		}
		fenced[i] = true
		if ch, n, hasInfo, ok := fenceMarker(line.Text); ok && ch == fenceChar && n >= fenceLen && !hasInfo {
			inFence = false
		}
	}
	return fenced
}

// Requirement es un requisito reconocido por el perfil (REQ-24.39).
type Requirement struct {
	Start int    // índice de la línea de encabezado, desde 0
	End   int    // índice de la primera línea posterior al bloque
	ID    string // grupo `id` del patrón
	Title string // grupo `title` del patrón
	Key   string // identificador según la identidad: el id o el título
	Line  string // texto del encabezado, sin final de línea
	Tail  []TailSection
}

// LineNumber devuelve el número de línea del encabezado, desde 1.
func (r Requirement) LineNumber() int { return r.Start + 1 }

// TailSection es una sección no requisito de la cola contigua de un requisito
// (D-09): un encabezado del nivel de los requisitos que no cumple `heading` y
// que sigue directamente al bloque.
type TailSection struct {
	Index int    // índice de la línea de encabezado, desde 0
	Line  string // texto del encabezado, sin final de línea
}

// LineNumber devuelve el número de línea del encabezado, desde 1.
func (s TailSection) LineNumber() int { return s.Index + 1 }

// Requirements devuelve los requisitos del documento según el perfil, en orden.
// Un requisito es un encabezado ATX de nivel `level` fuera de vallas cuya línea
// completa cumple `heading`. Su bloque va desde el encabezado hasta el siguiente
// encabezado ATX de nivel menor o igual que `level` fuera de vallas, de modo que
// las subsecciones de nivel mayor pertenecen al bloque (REQ-24.41). Los
// identificadores duplicados no se rechazan aquí: la composición los
// rechaza al operar sobre ellos (REQ-24.40).
func (d Document) Requirements(p Profile) []Requirement {
	headings := d.Headings()
	var requirements []Requirement
	for i, heading := range headings {
		if heading.Level != p.Level {
			continue
		}
		text := d.Lines[heading.Index].Text
		id, title, ok := p.Match(text)
		if !ok {
			continue
		}
		requirement := Requirement{
			Start: heading.Index,
			End:   blockEnd(headings, i, p.Level, len(d.Lines)),
			ID:    id,
			Title: title,
			Key:   p.Key(id, title),
			Line:  text,
		}
		requirement.Tail = d.tail(headings, i, p, requirement.End)
		requirements = append(requirements, requirement)
	}
	return requirements
}

// blockEnd devuelve el índice de la primera línea tras el bloque que abre el
// encabezado headings[at]: el siguiente encabezado de nivel menor o igual que
// level, o el final del documento.
func blockEnd(headings []Heading, at, level, total int) int {
	for _, next := range headings[at+1:] {
		if next.Level <= level {
			return next.Index
		}
	}
	return total
}

// tail recorre las secciones no requisito que siguen directamente al bloque: se
// detiene en el siguiente requisito, en un encabezado de nivel menor que `level`
// o al final del documento.
func (d Document) tail(headings []Heading, at int, p Profile, blockEndIndex int) []TailSection {
	var tail []TailSection
	position := blockEndIndex
	for position < len(d.Lines) {
		next := -1
		for j := at + 1; j < len(headings); j++ {
			if headings[j].Index == position {
				next = j
				break
			}
		}
		if next < 0 {
			return tail
		}
		heading := headings[next]
		if heading.Level != p.Level {
			return tail
		}
		text := d.Lines[heading.Index].Text
		if _, _, isRequirement := p.Match(text); isRequirement {
			return tail
		}
		tail = append(tail, TailSection{Index: heading.Index, Line: text})
		position = blockEnd(headings, next, p.Level, len(d.Lines))
	}
	return tail
}
