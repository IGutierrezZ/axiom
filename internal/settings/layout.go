package settings

import (
	"bytes"
	"regexp"
	"strings"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// commentGap casa los espacios que preceden a un comentario de línea: yaml.v3 los
// normaliza a uno solo al serializar.
var commentGap = regexp.MustCompile(`(\S)\s+#`)

// splitLines divide el texto en líneas sin finales de línea (`\n` o `\r\n`) y
// devuelve si terminaba en salto de línea.
func splitLines(text string) (lines []string, finalNewline bool) {
	if text == "" {
		return nil, false
	}
	finalNewline = strings.HasSuffix(text, "\n")
	text = strings.TrimSuffix(text, "\n")
	lines = strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines, finalNewline
}

func isBlank(line string) bool { return strings.TrimSpace(line) == "" }

// sameLine compara dos líneas sin espacios finales y sin la separación previa a un
// comentario de línea, que el codificador normaliza.
func sameLine(a, b string) bool {
	norm := func(s string) string { return commentGap.ReplaceAllString(strings.TrimRight(s, " \t"), "$1 #") }
	return norm(a) == norm(b)
}

// layoutOp es un paso del alineamiento entre el original y el codificado.
type layoutOp struct {
	orig int // índice en el original; -1 si la línea es solo del codificado
	enc  int // índice en el codificado; -1 si la línea es solo del original
}

func (o layoutOp) matched() bool { return o.orig >= 0 && o.enc >= 0 }

// alignLines alinea por la subsecuencia común más larga las líneas del original y
// del codificado.
func alignLines(orig, enc []string) []layoutOp {
	n, m := len(orig), len(enc)
	table := make([][]int32, n+1)
	for i := range table {
		table[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case sameLine(orig[i], enc[j]):
				table[i][j] = table[i+1][j+1] + 1
			case table[i+1][j] >= table[i][j+1]:
				table[i][j] = table[i+1][j]
			default:
				table[i][j] = table[i][j+1]
			}
		}
	}
	ops := make([]layoutOp, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case sameLine(orig[i], enc[j]):
			ops = append(ops, layoutOp{orig: i, enc: j})
			i++
			j++
		case table[i][j+1] >= table[i+1][j]:
			ops = append(ops, layoutOp{orig: -1, enc: j})
			j++
		default:
			ops = append(ops, layoutOp{orig: i, enc: -1})
			i++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, layoutOp{orig: i, enc: -1})
	}
	for ; j < m; j++ {
		ops = append(ops, layoutOp{orig: -1, enc: j})
	}
	return ops
}

// restoreLayout devuelve encoded con el formato de original que yaml.v3 pierde: las
// líneas en blanco, los finales de línea CRLF y el BOM UTF-8. Alinea por LCS las
// líneas de ambos textos y recorre los tramos entre líneas comunes:
//   - las líneas comunes salen tal como estaban en el original (con sus espacios
//     finales y la separación de sus comentarios), de modo que un parche sin efecto
//     reproduce el original byte a byte;
//   - de un tramo salen primero las líneas nuevas del codificado y después las blancas
//     del original (la blanca que separaba dos bloques queda tras lo añadido al
//     primero); si el tramo también borra contenido, las blancas van delante y, sin
//     líneas nuevas, se conserva a lo sumo una;
//   - una clave de primer nivel nueva queda separada por una línea en blanco de lo que
//     la rodea;
//   - las blancas que añade el codificador (p. ej. tras el comentario de documento)
//     se descartan: solo valen las del original.
//
// Un error de alineamiento como mucho mueve una línea en blanco: nunca cambia datos,
// y las líneas en blanco de un escalar literal `|` o plegado `>` están en los dos
// textos, de modo que se alinean entre sí y no se duplican.
func restoreLayout(original, encoded []byte) []byte {
	bom := bytes.HasPrefix(original, utf8BOM)
	original = bytes.TrimPrefix(original, utf8BOM)
	crlf := bytes.Contains(original, []byte("\r\n"))

	origLines, origFinal := splitLines(string(original))
	encLines, _ := splitLines(string(encoded))
	ops := alignLines(origLines, encLines)

	out := make([]string, 0, len(ops))
	lastNonBlank := func() bool { return len(out) > 0 && !isBlank(out[len(out)-1]) }
	separateNext := false // la última clave de primer nivel añadida pide una blanca tras ella
	endsWithDrop := false
	for i := 0; i < len(ops); {
		if op := ops[i]; op.matched() {
			line := origLines[op.orig]
			if separateNext && lastNonBlank() && startsTopLevelItem(line) {
				out = append(out, "")
			}
			separateNext = false
			out = append(out, line)
			i++
			continue
		}
		end := i
		for end < len(ops) && !ops[end].matched() {
			end++
		}
		var adds, lead, blanks []string // blanks: las posteriores al contenido borrado, o todas si no se borra nada
		dropped := false
		for _, op := range ops[i:end] {
			switch {
			case op.enc >= 0:
				if line := encLines[op.enc]; !isBlank(line) {
					adds = append(adds, line)
				}
			case isBlank(origLines[op.orig]) && !dropped:
				lead = append(lead, origLines[op.orig])
			case isBlank(origLines[op.orig]):
				blanks = append(blanks, origLines[op.orig])
			default:
				dropped = true
			}
		}
		if !dropped {
			blanks, lead = lead, nil // sin contenido borrado, todas van tras lo añadido
		}
		if dropped && len(adds) == 0 {
			// Borrado puro: a lo sumo una blanca separa lo anterior de lo siguiente.
			if all := append(lead, blanks...); len(all) > 0 {
				lead, blanks = all[:1], nil
			}
		}
		out = append(out, lead...)
		for _, line := range adds {
			if opensTopLevelKey(line) && !dropped {
				if lastNonBlank() {
					out = append(out, "")
				}
				separateNext = true
			}
			out = append(out, line)
		}
		if len(blanks) > 0 {
			out = append(out, blanks...)
			separateNext = false
		}
		endsWithDrop = end == len(ops) && dropped && len(adds) == 0
		i = end
	}
	if endsWithDrop {
		// Se borró contenido al final del fichero: las blancas que lo separaban de lo
		// anterior sobran.
		for len(out) > 0 && isBlank(out[len(out)-1]) {
			out = out[:len(out)-1]
		}
	}

	eol := "\n"
	if crlf {
		eol = "\r\n"
	}
	text := strings.Join(out, eol)
	if len(out) > 0 && (origFinal || len(origLines) == 0) {
		text += eol
	}
	if bom {
		return append(append([]byte{}, utf8BOM...), text...)
	}
	return []byte(text)
}

// startsTopLevelItem informa de si la línea empieza en la columna 0 y no es un
// elemento de lista: una clave o un comentario de primer nivel.
func startsTopLevelItem(line string) bool {
	if line == "" {
		return false
	}
	switch line[0] {
	case ' ', '\t', '-':
		return false
	}
	return true
}

// opensTopLevelKey informa de si la línea abre una clave de primer nivel: empieza
// en la columna 0 y no es un comentario, un elemento de lista ni un separador.
func opensTopLevelKey(line string) bool {
	return startsTopLevelItem(line) && line[0] != '#' && strings.Contains(line, ":")
}
