// Package techcontext integra el contexto técnico mantenido de un cambio: el
// delta `context-impact.md`, sus anclas, el alcance de las rutas declaradas en
// `technical_context`, la traza `integration.yaml` y la verificación por diff de
// git. Nació como la extracción dirigida de la rama de contexto de la integración
// de la baseline ajena (retirada), sin baseline, perfiles ni identificadores
// provisionales (D2 de spec-viva-openspec).
package techcontext

import "strings"

// utf8BOM es la marca de orden de bytes de UTF-8.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Line es una línea del documento sin su final de línea, con el final que tenía
// en el fichero: "\n", "\r\n", o vacío en la última línea sin salto. Un "\r"
// suelto no es un final de línea y forma parte del texto.
type Line struct {
	Text string
	EOL  string
}

// Document es un documento Markdown como serie de líneas. Solo se lee: el
// contexto técnico nunca se reescribe desde aquí.
type Document struct {
	BOM   bool
	Lines []Line
}

// ParseDocument divide data en líneas y detecta el BOM UTF-8, que retira.
func ParseDocument(data []byte) Document {
	var doc Document
	if len(data) >= len(utf8BOM) && string(data[:len(utf8BOM)]) == string(utf8BOM) {
		doc.BOM = true
		data = data[len(utf8BOM):]
	}
	for len(data) > 0 {
		newline := -1
		for i, b := range data {
			if b == '\n' {
				newline = i
				break
			}
		}
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
