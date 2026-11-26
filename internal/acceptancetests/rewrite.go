package acceptancetests

import (
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// RewriteCubre sustituye, en la columna `Cubre` de la tabla de `pruebas.md`, los
// identificadores del mapa por su sustituto, solo en el tramo de cada entrada anterior al
// primer `#`. No toca ningún otro byte: ni el nombre del escenario, ni las demás columnas, ni
// el texto fuera de la tabla, ni los finales de línea (RF-KOF-23). Usa la misma selección de
// tabla que Parse; sin mapa o sin tabla legible devuelve el texto igual. Es idempotente.
func RewriteCubre(text string, m reqid.IDMap) string {
	if m.Count() == 0 {
		return text
	}
	lines := isoevidence.SplitLines(text)
	t, ok, _ := readTable(lines)
	if !ok || len(t.rows) == 0 {
		return text
	}
	isRow := map[int]bool{}
	for _, i := range t.rows {
		isRow[i] = true
	}
	raw := strings.SplitAfter(text, "\n")
	var b strings.Builder
	for i, l := range raw {
		if !isRow[i] {
			b.WriteString(l)
			continue
		}
		content := strings.TrimRight(l, "\r\n")
		if len(isoevidence.SplitCells(content)) != t.width {
			b.WriteString(l)
			continue
		}
		start, end, found := isoevidence.RawCellSpan(content, t.cols["Cubre"])
		if !found {
			b.WriteString(l)
			continue
		}
		b.WriteString(content[:start])
		b.WriteString(rewriteCell(content[start:end], m))
		b.WriteString(content[end:])
		b.WriteString(l[len(content):])
	}
	return b.String()
}

// rewriteCell aplica el mapa solo al tramo de cada entrada anterior al primer `#`.
func rewriteCell(cell string, m reqid.IDMap) string {
	parts := strings.Split(cell, ";")
	for i, p := range parts {
		if before, after, hasHash := strings.Cut(p, "#"); hasHash {
			parts[i] = m.Substitute(before) + "#" + after
		} else {
			parts[i] = m.Substitute(p)
		}
	}
	return strings.Join(parts, ";")
}
