package acceptancetests

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
)

var caseIDPattern = regexp.MustCompile(`^P-[0-9]{2,}$`)

// table es la tabla localizada: la posición de cada columna y las líneas de sus filas.
type table struct {
	cols  map[string]int
	width int
	rows  []int // índices en el slice de líneas
}

func tableFinding(line int, subject, msg string) Finding {
	return Finding{Code: CodeInvalidTable, File: FileName, Line: line, Subject: subject, Message: msg}
}

// readTable localiza la tabla única (D3): la primera línea de tabla fuera de vallas. Una
// segunda tabla fuera de vallas es un hallazgo. ok es falso cuando la forma impide leer filas.
func readTable(lines []isoevidence.TextLine) (t table, ok bool, issues []Finding) {
	h := -1
	for i, l := range lines {
		if !l.Fenced && isoevidence.IsTableLine(l.Text) {
			h = i
			break
		}
	}
	if h < 0 {
		return t, false, []Finding{tableFinding(0, "", "no hay ninguna tabla de casos en "+FileName)}
	}
	known := map[string]bool{}
	for _, c := range columns {
		known[c] = true
	}
	idx := map[string]int{}
	var problems []Finding
	header := isoevidence.SplitCells(lines[h].Text)
	for i, name := range header {
		_, dup := idx[name]
		switch {
		case !known[name]:
			problems = append(problems, tableFinding(lines[h].Num, name,
				fmt.Sprintf("columna no admitida %q (los nombres son exactos y sensibles a mayúsculas)", name)))
		case dup:
			problems = append(problems, tableFinding(lines[h].Num, name, fmt.Sprintf("columna duplicada %q", name)))
		default:
			idx[name] = i
		}
	}
	for _, c := range columns {
		if _, found := idx[c]; !found {
			problems = append(problems, tableFinding(lines[h].Num, c, fmt.Sprintf("falta la columna %q", c)))
		}
	}
	if len(problems) > 0 {
		return t, false, problems
	}
	if h+1 >= len(lines) || lines[h+1].Fenced || !isoevidence.IsSeparatorLine(lines[h+1].Text) {
		return t, false, []Finding{tableFinding(lines[h].Num, "", "falta la línea separadora tras la cabecera")}
	}
	t = table{cols: idx, width: len(header)}
	j := h + 2
	for ; j < len(lines) && !lines[j].Fenced && isoevidence.IsTableLine(lines[j].Text); j++ {
		t.rows = append(t.rows, j)
	}
	prev := false
	for ; j < len(lines); j++ {
		switch {
		case lines[j].Fenced:
			prev = false
		case isoevidence.IsTableLine(lines[j].Text):
			if !prev {
				issues = append(issues, tableFinding(lines[j].Num, "", "hay una segunda tabla: "+FileName+" solo admite una"))
			}
			prev = true
		default:
			prev = false
		}
	}
	return t, true, issues
}

// Parse lee la tabla de `pruebas.md` (RF-KOF-18 y RF-KOF-19). No valida dominios: eso es de
// Check. Las filas con un número de celdas distinto del de la cabecera dan `tabla_invalida`
// con su línea y se descartan; el resto se sigue leyendo.
func Parse(text string) Battery {
	lines := isoevidence.SplitLines(text)
	t, ok, issues := readTable(lines)
	b := Battery{Issues: issues}
	if !ok {
		return b
	}
	for _, i := range t.rows {
		cells := isoevidence.SplitCells(lines[i].Text)
		if len(cells) != t.width {
			b.Issues = append(b.Issues, tableFinding(lines[i].Num, "",
				fmt.Sprintf("la fila tiene %d celdas y la cabecera %d", len(cells), t.width)))
			continue
		}
		cell := func(name string) string { return cells[t.cols[name]] }
		raw := cell("Cubre")
		b.Cases = append(b.Cases, Case{
			Line:          lines[i].Num,
			ID:            cell("ID"),
			Role:          cell("Rol"),
			Type:          cell("Tipo"),
			Priority:      cell("Prioridad"),
			Preconditions: cell("Precondiciones"),
			Steps:         cell("Pasos"),
			Expected:      cell("Resultado esperado"),
			CubreRaw:      raw,
			Cubre:         parseCubre(raw),
			Environments:  splitList(cell("Entornos")),
		})
	}
	return b
}

// splitList separa por `;` y recorta cada elemento. Una celda vacía es una lista vacía.
func splitList(cell string) []string {
	if strings.TrimSpace(cell) == "" {
		return nil
	}
	parts := strings.Split(cell, ";")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// parseCubre separa las entradas por `;` y cada una por el primer `#`. `ID#` sin nombre y
// una entrada vacía son entradas vacías (ID vacío).
func parseCubre(raw string) []CubreEntry {
	parts := splitList(raw)
	if parts == nil {
		return nil
	}
	out := make([]CubreEntry, 0, len(parts))
	for _, p := range parts {
		id, scenario, hasHash := strings.Cut(p, "#")
		id, scenario = strings.TrimSpace(id), strings.TrimSpace(scenario)
		if hasHash && scenario == "" {
			out = append(out, CubreEntry{})
			continue
		}
		out = append(out, CubreEntry{ID: id, Scenario: scenario})
	}
	return out
}
