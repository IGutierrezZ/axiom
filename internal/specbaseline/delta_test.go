package specbaseline

import (
	"reflect"
	"strings"
	"testing"
)

// join une líneas con LF y añade el salto final, para escribir textos de prueba.
func join(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func bodyText(block deltaBlock) []string {
	var out []string
	for _, line := range block.Body {
		out = append(out, line.Text)
	}
	return out
}

func TestDeltaParseSplitsSectionsAndRequirements(t *testing.T) {
	text := join(
		"# Delta de prueba",
		"",
		"Texto previo que se ignora.",
		"",
		"## ADDED Requirements",
		"",
		"### Requirement: RF-9001: Reintento de pago",
		"",
		"El sistema DEBE reintentar el pago.",
		"",
		"## MODIFIED Requirements",
		"",
		"### Requirement: RF-100: Alta de cliente",
		"",
		"Cuerpo nuevo.",
		"",
		"### Requirement: RF-101: Baja de cliente",
		"",
		"Otro cuerpo.",
		"",
		"## REMOVED Requirements",
		"",
		"### Requirement: RF-102: Consulta",
		"",
		"(Reason: sustituido por el listado)",
		"",
		"## RENAMED Requirements",
		"",
		"### Requirement: Alta → Alta de cliente",
		"",
		"(Reason: nombre más claro)",
	)
	forEachVariant(t, func(t *testing.T, v byteVariant) {
		parsed := parseDelta(v.apply(text))
		names := func(blocks []deltaBlock) []string {
			var out []string
			for _, block := range blocks {
				out = append(out, block.Name)
			}
			return out
		}
		if got, want := names(parsed.Added), []string{"RF-9001: Reintento de pago"}; !reflect.DeepEqual(got, want) {
			t.Errorf("ADDED = %q, se esperaba %q", got, want)
		}
		if got, want := names(parsed.Modified), []string{"RF-100: Alta de cliente", "RF-101: Baja de cliente"}; !reflect.DeepEqual(got, want) {
			t.Errorf("MODIFIED = %q, se esperaba %q", got, want)
		}
		if got, want := names(parsed.Removed), []string{"RF-102: Consulta"}; !reflect.DeepEqual(got, want) {
			t.Errorf("REMOVED = %q, se esperaba %q", got, want)
		}
		if got, want := names(parsed.Renamed), []string{"Alta → Alta de cliente"}; !reflect.DeepEqual(got, want) {
			t.Errorf("RENAMED = %q, se esperaba %q", got, want)
		}
		if parsed.empty() {
			t.Error("el delta no debe estar vacío")
		}
		if got := parsed.Modified[0].Line; got != 13 {
			t.Errorf("línea del primer MODIFIED = %d, se esperaba 13", got)
		}
	})
}

func TestDeltaParseBodyDropsNotesAndTrailingBlankLines(t *testing.T) {
	text := join(
		"## MODIFIED Requirements",
		"",
		"### Requirement: RF-100: Alta",
		"(Placement: after RF-001)",
		"",
		"Primer párrafo.",
		"",
		"(Previously: texto anterior)",
		"(Migration: sin migración)",
		"Segundo párrafo.",
		"",
		"",
	)
	parsed := parseDelta([]byte(text))
	if len(parsed.Modified) != 1 {
		t.Fatalf("MODIFIED = %d, se esperaba 1", len(parsed.Modified))
	}
	want := []string{"", "Primer párrafo.", "", "Segundo párrafo."}
	if got := bodyText(parsed.Modified[0]); !reflect.DeepEqual(got, want) {
		t.Errorf("cuerpo = %q, se esperaba %q", got, want)
	}
	if got := parsed.Modified[0].Body[1].Number; got != 6 {
		t.Errorf("número de línea de «Primer párrafo.» = %d, se esperaba 6", got)
	}
}

func TestDeltaParseReadsTheReasonNote(t *testing.T) {
	text := join(
		"## REMOVED Requirements",
		"",
		"### Requirement: RF-100: Alta",
		"",
		"(Reason:  sustituido por el nuevo flujo )",
		"",
		"### Requirement: RF-101: Baja",
		"",
		"Sin nota de motivo.",
	)
	parsed := parseDelta([]byte(text))
	if len(parsed.Removed) != 2 {
		t.Fatalf("REMOVED = %d, se esperaban 2", len(parsed.Removed))
	}
	if first := parsed.Removed[0]; !first.HasReason || first.Reason != "sustituido por el nuevo flujo" {
		t.Errorf("primera nota = (%v, %q)", first.HasReason, first.Reason)
	}
	if second := parsed.Removed[1]; second.HasReason {
		t.Errorf("el segundo bloque no tiene nota de motivo: %+v", second)
	}
}

func TestDeltaParseIgnoresFencedHeadingsAndNotes(t *testing.T) {
	text := join(
		"## MODIFIED Requirements",
		"",
		"### Requirement: RF-100: Alta",
		"",
		"```md",
		"## ADDED Requirements",
		"### Requirement: RF-777: Falso",
		"(Reason: no es una nota)",
		"```",
		"",
		"Después de la valla.",
	)
	parsed := parseDelta([]byte(text))
	if len(parsed.Added) != 0 || len(parsed.Modified) != 1 {
		t.Fatalf("ADDED = %d y MODIFIED = %d, se esperaban 0 y 1", len(parsed.Added), len(parsed.Modified))
	}
	block := parsed.Modified[0]
	if block.HasReason {
		t.Error("una nota dentro de una valla no es una nota")
	}
	want := []string{"", "```md", "## ADDED Requirements", "### Requirement: RF-777: Falso", "(Reason: no es una nota)", "```", "", "Después de la valla."}
	if got := bodyText(block); !reflect.DeepEqual(got, want) {
		t.Errorf("cuerpo = %q, se esperaba %q", got, want)
	}
}

func TestDeltaParseWithoutSectionsIsEmpty(t *testing.T) {
	for name, text := range map[string]string{
		"vacío":                      "",
		"solo texto":                 "Un párrafo.\n",
		"requisito fuera de sección": "### Requirement: RF-1: A\n\nCuerpo.\n",
		"sección sin requisitos":     "## MODIFIED Requirements\n\nTexto suelto.\n",
		"sección con otro nombre":    "## OTHER Requirements\n\n### Requirement: RF-1: A\n",
	} {
		t.Run(name, func(t *testing.T) {
			if parsed := parseDelta([]byte(text)); !parsed.empty() {
				t.Errorf("el delta debería estar vacío: %+v", parsed)
			}
		})
	}
}

func TestDeltaSplitRenamePair(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantOld string
		wantNew string
		wantOK  bool
	}{
		{name: "flecha unicode", in: "Alta → Alta de cliente", wantOld: "Alta", wantNew: "Alta de cliente", wantOK: true},
		{name: "flecha ASCII", in: "Alta -> Alta de cliente", wantOld: "Alta", wantNew: "Alta de cliente", wantOK: true},
		{name: "sin flecha", in: "Alta"},
		{name: "sin nombre nuevo", in: "Alta →"},
		{name: "sin nombre antiguo", in: "→ Alta"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldName, newName, ok := splitRenamePair(tt.in)
			if ok != tt.wantOK {
				t.Fatalf("splitRenamePair(%q) ok = %v, se esperaba %v", tt.in, ok, tt.wantOK)
			}
			if ok && (oldName != tt.wantOld || newName != tt.wantNew) {
				t.Errorf("splitRenamePair(%q) = (%q, %q)", tt.in, oldName, newName)
			}
		})
	}
}

func TestDeltaBodyViolationsFollowTheProfileLevel(t *testing.T) {
	body := func(texts ...string) []deltaLine {
		lines := make([]deltaLine, len(texts))
		for i, text := range texts {
			lines[i] = deltaLine{Text: text, Number: 10 + i}
		}
		return lines
	}
	tests := []struct {
		name  string
		level int
		body  []deltaLine
		want  []headingViolation
	}{
		{name: "sin encabezados", level: 3, body: body("texto", "más texto")},
		{name: "los escenarios de nivel 4 no son infracción con nivel 3", level: 3, body: body("", "#### Escenario: A", "", "- **CUANDO** x")},
		{name: "mismo nivel", level: 3, body: body("texto", "### Notas de implantación"), want: []headingViolation{{Text: "### Notas de implantación", Level: 3, Number: 11}}},
		{name: "nivel menor", level: 3, body: body("## Ejemplo"), want: []headingViolation{{Text: "## Ejemplo", Level: 2, Number: 10}}},
		{name: "un escenario de nivel 4 sí es infracción con nivel 4", level: 4, body: body("#### Escenario: A"), want: []headingViolation{{Text: "#### Escenario: A", Level: 4, Number: 10}}},
		{name: "dentro de una valla no cuenta", level: 3, body: body("```", "## Ejemplo", "```", "~~~", "### Otro", "~~~")},
		{name: "valla sin cerrar", level: 3, body: body("```", "## Ejemplo")},
		{name: "varias infracciones", level: 2, body: body("# Uno", "texto", "## Dos"), want: []headingViolation{{Text: "# Uno", Level: 1, Number: 10}, {Text: "## Dos", Level: 2, Number: 12}}},
		{name: "indentación de cuatro espacios no es encabezado", level: 3, body: body("    ### código")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bodyViolations(tt.body, tt.level)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("infracciones = %+v, se esperaba %+v", got, tt.want)
			}
		})
	}
}
