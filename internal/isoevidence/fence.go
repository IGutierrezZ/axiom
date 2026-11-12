package isoevidence

import "strings"

// textLine es una línea de un texto Markdown, con su número (base 1) y si queda
// dentro de una valla de código. Las propias líneas de apertura y cierre de la
// valla cuentan como vallas.
type textLine struct {
	Num    int
	Text   string // sin el final de línea
	Fenced bool
}

// splitLines divide text en líneas (acepta `\n` y `\r\n`) y marca las que están
// dentro de una valla ``` o ~~~. Una valla sin cerrar abarca hasta el final.
func splitLines(text string) []textLine {
	if text == "" {
		return nil
	}
	raw := strings.Split(text, "\n")
	if raw[len(raw)-1] == "" {
		raw = raw[:len(raw)-1]
	}
	lines := make([]textLine, 0, len(raw))
	inFence, fenceChar, fenceLen := false, byte(0), 0
	for i, l := range raw {
		l = strings.TrimSuffix(l, "\r")
		line := textLine{Num: i + 1, Text: l}
		ch, n, hasInfo, ok := fenceMarker(l)
		switch {
		case !inFence && ok:
			inFence, fenceChar, fenceLen = true, ch, n
			line.Fenced = true
		case inFence:
			line.Fenced = true
			if ok && ch == fenceChar && n >= fenceLen && !hasInfo {
				inFence = false
			}
		}
		lines = append(lines, line)
	}
	return lines
}

// fenceMarker dice si la línea abre o cierra una valla: el carácter, la
// longitud de la tirada y si lleva texto detrás (una apertura puede llevar
// información; un cierre no).
func fenceMarker(line string) (ch byte, length int, hasInfo, ok bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || trimmed == "" {
		return 0, 0, false, false
	}
	ch = trimmed[0]
	if ch != '`' && ch != '~' {
		return 0, 0, false, false
	}
	for length < len(trimmed) && trimmed[length] == ch {
		length++
	}
	if length < 3 {
		return 0, 0, false, false
	}
	rest := strings.TrimSpace(trimmed[length:])
	if ch == '`' && strings.Contains(rest, "`") {
		return 0, 0, false, false
	}
	return ch, length, rest != "", true
}
