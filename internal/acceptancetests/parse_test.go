package acceptancetests

import (
	"strings"
	"testing"
)

func TestParseValidTable(t *testing.T) {
	text := "# Pruebas\n\nnota\n\n" + header + row() + row("ID", "P-02", "Entornos", "dev; pre")
	b := Parse(text)
	if len(b.Issues) != 0 || len(b.Cases) != 2 {
		t.Fatalf("issues=%v cases=%d", b.Issues, len(b.Cases))
	}
	if b.Cases[1].Line != 8 || len(b.Cases[1].Environments) != 2 || b.Cases[1].Environments[1] != "pre" {
		t.Fatalf("caso 2 = %+v", b.Cases[1])
	}
}

func TestParseColumnsInAnyOrder(t *testing.T) {
	text := "| Pasos | Resultado esperado | ID | Cubre | Rol | Tipo | Prioridad | Entornos | Precondiciones |\n" +
		"|---|---|---|---|---|---|---|---|---|\n" +
		"| p | r | P-07 | REQ-1.1 | fullstack | manual | alta | dev | |\n"
	b := Parse(text)
	if len(b.Issues) != 0 || len(b.Cases) != 1 {
		t.Fatalf("%+v", b)
	}
	c := b.Cases[0]
	if c.ID != "P-07" || c.Steps != "p" || c.Expected != "r" || c.CubreRaw != "REQ-1.1" || c.Preconditions != "" {
		t.Fatalf("caso = %+v", c)
	}
}

func TestParseInvalidHeaders(t *testing.T) {
	cases := map[string]string{
		"sin Cubre":      strings.Replace(header, "| Cubre ", "", 1),
		"columna extra":  strings.Replace(header, "Pasos |", "Pasos | Extra |", 1),
		"duplicada":      strings.Replace(header, "| Rol ", "| Tipo ", 1),
		"capitalización": strings.Replace(header, "Prioridad", "prioridad", 1),
	}
	for name, h := range cases {
		b := Parse(h + row())
		if len(b.Cases) != 0 || len(b.Issues) == 0 {
			t.Errorf("%s: cases=%d issues=%v", name, len(b.Cases), b.Issues)
			continue
		}
		for _, f := range b.Issues {
			if f.Code != CodeInvalidTable || f.Line != 1 || f.Subject == "" {
				t.Errorf("%s: hallazgo %+v", name, f)
			}
		}
	}
}

func TestParseNoTableAndMissingSeparator(t *testing.T) {
	if b := Parse("solo texto\n"); len(b.Issues) != 1 || b.Issues[0].Code != CodeInvalidTable {
		t.Fatalf("sin tabla: %+v", b)
	}
	noSep := strings.Split(header, "\n")[0] + "\n" + row()
	if b := Parse(noSep); len(b.Cases) != 0 || len(b.Issues) != 1 || b.Issues[0].Code != CodeInvalidTable {
		t.Fatalf("sin separadora: %+v", b)
	}
}

func TestParseFencedTableIsIgnored(t *testing.T) {
	text := "```\n" + header + row() + "```\n\n" + header + row("ID", "P-05")
	b := Parse(text)
	if len(b.Issues) != 0 || len(b.Cases) != 1 || b.Cases[0].ID != "P-05" {
		t.Fatalf("%+v", b)
	}
}

func TestParseSecondTableIsFinding(t *testing.T) {
	text := header + row() + "\ntexto\n\n" + header + row("ID", "P-02")
	b := Parse(text)
	if len(b.Cases) != 1 {
		t.Fatalf("casos = %d", len(b.Cases))
	}
	if len(b.Issues) != 1 || b.Issues[0].Code != CodeInvalidTable || b.Issues[0].Line != 7 {
		t.Fatalf("issues = %+v", b.Issues)
	}
}

func TestParseTableWithoutRows(t *testing.T) {
	b := Parse(header)
	if len(b.Cases) != 0 || len(b.Issues) != 0 {
		t.Fatalf("%+v", b)
	}
	r := Check(header, baseInput(req("d", "ADDED", "Uno REQ-1.1", "A", "B")))
	mustOnly(t, r, CodeUncoveredScenario, CodeUncoveredScenario, CodeUncoveredReq)
	if r.Cases != 0 || r.Scenarios != (Coverage{0, 2}) || r.Requirements != (Coverage{0, 1}) {
		t.Fatalf("informe = %+v", r)
	}
}

func TestParseRowWithWrongCellCountKeepsTheRest(t *testing.T) {
	text := header + "| P-01 | REQ-1.1 | solo |\n" + row("ID", "P-02")
	b := Parse(text)
	if len(b.Cases) != 1 || b.Cases[0].ID != "P-02" {
		t.Fatalf("casos = %+v", b.Cases)
	}
	if len(b.Issues) != 1 || b.Issues[0].Code != CodeInvalidTable || b.Issues[0].Line != 3 {
		t.Fatalf("issues = %+v", b.Issues)
	}
}

func TestParseEscapedPipeAndCommasInScenario(t *testing.T) {
	text := header + row("Pasos", `a \| b`, "Cubre", "REQ-1.1#nombre, con comas; REQ-1.2")
	b := Parse(text)
	if len(b.Cases) != 1 {
		t.Fatalf("%+v", b)
	}
	c := b.Cases[0]
	if c.Steps != "a | b" {
		t.Fatalf("Pasos = %q", c.Steps)
	}
	want := []CubreEntry{{"REQ-1.1", "nombre, con comas"}, {"REQ-1.2", ""}}
	if len(c.Cubre) != 2 || c.Cubre[0] != want[0] || c.Cubre[1] != want[1] {
		t.Fatalf("Cubre = %+v", c.Cubre)
	}
}

func TestParseCubreSplitsAtFirstHash(t *testing.T) {
	b := Parse(header + row("Cubre", "REQ-1.1#a#b"))
	if e := b.Cases[0].Cubre[0]; e.ID != "REQ-1.1" || e.Scenario != "a#b" {
		t.Fatalf("entrada = %+v", e)
	}
}

func TestParseEmptyCubreEntries(t *testing.T) {
	for _, raw := range []string{"REQ-1.1#", "REQ-1.1;;REQ-1.2", "REQ-1.1;"} {
		b := Parse(header + row("Cubre", raw))
		empty := false
		for _, e := range b.Cases[0].Cubre {
			if e.ID == "" {
				empty = true
			}
		}
		if !empty {
			t.Errorf("%q: no hay entrada vacía en %+v", raw, b.Cases[0].Cubre)
		}
	}
}
