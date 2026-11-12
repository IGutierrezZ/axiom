package isoevidence

import (
	"reflect"
	"strings"
	"testing"
)

func TestRequirementIDs(t *testing.T) {
	cases := []struct {
		name          string
		in            string
		wantValid     []string
		wantMalformed []string
	}{
		{"sin identificador", "Cobro con IVA", nil, nil},
		{"válido entre paréntesis", "Cobro con IVA (REQ-30.1)", []string{"REQ-30.1"}, nil},
		{"válido con coma detrás", "Cobro REQ-30.1, con IVA", []string{"REQ-30.1"}, nil},
		{"válido con punto final", "Cobro con IVA REQ-30.1.", []string{"REQ-30.1"}, nil},
		{"sin punto decimal", "Cobro (REQ-30)", nil, []string{"REQ-30"}},
		{"letra en lugar de número", "Cobro (REQ-x.1)", nil, []string{"REQ-x.1"}},
		{"tres componentes", "Cobro (REQ-30.1.2)", nil, []string{"REQ-30.1.2"}},
		{"dos distintos", "Cobro y devolución (REQ-30.1, REQ-30.2)", []string{"REQ-30.1", "REQ-30.2"}, nil},
		{"el mismo repetido cuenta como uno", "REQ-30.1 Cobro (REQ-30.1)", []string{"REQ-30.1"}, nil},
		{"válido y mal formado", "Cobro (REQ-30.1) y (REQ-9)", []string{"REQ-30.1"}, []string{"REQ-9"}},
		{"no es límite de palabra", "PREREQ-30.1", nil, nil},
		{"prefijo en minúsculas", "req-30.1", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			valid, malformed := RequirementIDs(tc.in)
			if !reflect.DeepEqual(valid, tc.wantValid) {
				t.Errorf("válidos = %v, quería %v", valid, tc.wantValid)
			}
			if !reflect.DeepEqual(malformed, tc.wantMalformed) {
				t.Errorf("mal formados = %v, quería %v", malformed, tc.wantMalformed)
			}
		})
	}
}

// reqBlock construye el bloque de un requisito como lo devuelve el lector nativo.
func reqBlock(name string) string {
	return "### Requirement: " + name + "\n\nTexto.\n\n#### Scenario: Uno\n\n- **DADO** algo\n"
}

func delta(section Section, domain, name string) Requirement {
	return Requirement{Domain: domain, Section: section, Name: name, Text: reqBlock(name)}
}

func base(domain, name string) Requirement {
	return Requirement{Domain: domain, Name: name, Text: reqBlock(name)}
}

func TestCheckRequirementIDs(t *testing.T) {
	cases := []struct {
		name     string
		delta    []Requirement
		baseline []Requirement
		// wantContains tiene un elemento por hallazgo esperado: fragmentos que
		// debe contener su motivo. Vacío significa «ningún hallazgo».
		wantContains [][]string
	}{
		{
			name:         "ADDED con identificador válido",
			delta:        []Requirement{delta(SectionAdded, "pagos", "Cobro con IVA (REQ-30.1)")},
			wantContains: nil,
		},
		{
			name:  "ADDED sin identificador",
			delta: []Requirement{delta(SectionAdded, "pagos", "Cobro con IVA")},
			wantContains: [][]string{{
				"blocked(iso_requirement_id)", `"pagos"`, "ADDED", `"Cobro con IVA"`, "REQ-<n>.<m>",
			}},
		},
		{
			name:  "identificador mal formado no cuenta",
			delta: []Requirement{delta(SectionAdded, "pagos", "Cobro con IVA (REQ-30)")},
			wantContains: [][]string{{
				"blocked(iso_requirement_id)", "forma no válida: REQ-30",
			}},
		},
		{
			name:  "identificador con letra no cuenta",
			delta: []Requirement{delta(SectionAdded, "pagos", "Cobro con IVA (REQ-x.1)")},
			wantContains: [][]string{{
				"blocked(iso_requirement_id)", "REQ-x.1",
			}},
		},
		{
			name:  "nombre con dos identificadores distintos es ambiguo",
			delta: []Requirement{delta(SectionAdded, "pagos", "Cobro y devolución (REQ-30.1, REQ-30.2)")},
			wantContains: [][]string{{
				"ambiguo", "REQ-30.1", "REQ-30.2",
			}},
		},
		{
			name:         "el mismo identificador repetido en un nombre cuenta como uno",
			delta:        []Requirement{delta(SectionAdded, "pagos", "REQ-30.1 Cobro (REQ-30.1)")},
			wantContains: nil,
		},
		{
			name:     "ADDED con un identificador que ya existe en la baseline",
			delta:    []Requirement{delta(SectionAdded, "pagos", "Otro requisito (REQ-13.1)")},
			baseline: []Requirement{base("sdd", "Subcomando axiom sdd status (REQ-13.1)")},
			wantContains: [][]string{{
				"REQ-13.1", "ya existe en la baseline", `"sdd"`,
			}},
		},
		{
			name: "dos ADDED con el mismo identificador en el mismo dominio",
			delta: []Requirement{
				delta(SectionAdded, "pagos", "Cobro (REQ-30.1)"),
				delta(SectionAdded, "pagos", "Cobro con IVA (REQ-30.1)"),
			},
			wantContains: [][]string{{
				"REQ-30.1", "se repite", `"Cobro (REQ-30.1)"`, `"Cobro con IVA (REQ-30.1)"`,
			}},
		},
		{
			name: "dos ADDED con el mismo identificador en dominios distintos",
			delta: []Requirement{
				delta(SectionAdded, "pagos", "Cobro (REQ-30.1)"),
				delta(SectionAdded, "facturas", "Factura (REQ-30.1)"),
			},
			wantContains: [][]string{{
				"REQ-30.1", "se repite", `"pagos"`, `"facturas"`,
			}},
		},
		{
			name:     "MODIFIED con identificador que existe con ese nombre",
			delta:    []Requirement{delta(SectionModified, "sdd", "Subcomando axiom sdd status (REQ-13.1)")},
			baseline: []Requirement{base("sdd", "Subcomando axiom sdd status (REQ-13.1)")},
		},
		{
			name:  "MODIFIED sin identificador remite al RENAMED",
			delta: []Requirement{delta(SectionModified, "sdd", "Subcomando axiom sdd status")},
			wantContains: [][]string{{
				"MODIFIED", "RENAMED",
			}},
		},
		{
			name:  "REMOVED sin identificador remite al RENAMED",
			delta: []Requirement{delta(SectionRemoved, "sdd", "Subcomando antiguo")},
			wantContains: [][]string{{
				"REMOVED", "RENAMED",
			}},
		},
		{
			name: "RENAMED que conserva el identificador",
			delta: []Requirement{{
				Domain: "pagos", Section: SectionRenamed,
				OldName: "Cobro (REQ-30.1)", Name: "Cobro con IVA (REQ-30.1)",
			}},
			baseline: []Requirement{base("pagos", "Cobro (REQ-30.1)")},
		},
		{
			name: "RENAMED que cambia el identificador",
			delta: []Requirement{{
				Domain: "pagos", Section: SectionRenamed,
				OldName: "Cobro (REQ-30.1)", Name: "Cobro con IVA (REQ-30.2)",
			}},
			baseline: []Requirement{base("pagos", "Cobro (REQ-30.1)")},
			wantContains: [][]string{{
				"RENAMED", "REQ-30.1", "REQ-30.2",
			}},
		},
		{
			name: "RENAMED que añade el identificador a un requisito sin él",
			delta: []Requirement{{
				Domain: "pagos", Section: SectionRenamed,
				OldName: "Cobro", Name: "Cobro (REQ-30.3)",
			}},
			baseline: []Requirement{{Domain: "pagos", Name: "Cobro", Text: reqBlock("Cobro")}},
		},
		{
			name: "RENAMED que añade un identificador que ya existe",
			delta: []Requirement{{
				Domain: "pagos", Section: SectionRenamed,
				OldName: "Cobro", Name: "Cobro (REQ-13.1)",
			}},
			baseline: []Requirement{base("sdd", "Subcomando axiom sdd status (REQ-13.1)")},
			wantContains: [][]string{{
				"REQ-13.1", "ya existe en la baseline",
			}},
		},
		{
			name: "RENAMED sin identificador en el nombre nuevo",
			delta: []Requirement{{
				Domain: "pagos", Section: SectionRenamed,
				OldName: "Cobro (REQ-30.1)", Name: "Cobro con IVA",
			}},
			wantContains: [][]string{{
				"RENAMED", "no contiene un identificador",
			}},
		},
		{
			name:     "los requisitos que la delta no toca no se comprueban",
			delta:    []Requirement{delta(SectionAdded, "pagos", "Cobro (REQ-30.1)")},
			baseline: []Requirement{{Domain: "pagos", Name: "Sin identificador", Text: reqBlock("Sin identificador")}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckRequirementIDs(tc.delta, tc.baseline)
			assertFindings(t, got, CodeRequirementID, StageSpecs, tc.wantContains)
		})
	}
}

// assertFindings comprueba número, token, etapa y fragmentos de cada motivo.
func assertFindings(t *testing.T, got []Finding, code Code, stage Stage, want [][]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("hallazgos = %d, quería %d: %v", len(got), len(want), reasons(got))
	}
	for i, f := range got {
		if f.Code != code || f.Stage != stage {
			t.Errorf("hallazgo %d: token/etapa = %s/%s, quería %s/%s", i, f.Code, f.Stage, code, stage)
		}
		reason := f.Reason()
		for _, fragment := range want[i] {
			if !strings.Contains(reason, fragment) {
				t.Errorf("hallazgo %d: el motivo %q no contiene %q", i, reason, fragment)
			}
		}
	}
}

func reasons(findings []Finding) []string {
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, f.Reason())
	}
	return out
}

// TestCheckRequirementIDsInterruptedArchive cubre la tolerancia D17: el ADDED
// ya compuesto por un archivado interrumpido no bloquea, y cualquier diferencia
// sigue bloqueando.
func TestCheckRequirementIDsInterruptedArchive(t *testing.T) {
	const name = "Cobro con IVA (REQ-30.1)"
	added := Requirement{Domain: "pagos", Section: SectionAdded, Name: name, Text: reqBlock(name)}
	composed := Requirement{Domain: "pagos", Name: name, Text: reqBlock(name)}

	cases := []struct {
		name     string
		baseline Requirement
		wantFind bool
	}{
		{"mismo dominio, nombre y bloque", composed, false},
		{"un byte distinto en el bloque", Requirement{Domain: "pagos", Name: name, Text: strings.Replace(reqBlock(name), "Texto.", "Texto,", 1)}, true},
		{"un espacio distinto", Requirement{Domain: "pagos", Name: name, Text: reqBlock(name) + " "}, true},
		{"un escenario más", Requirement{Domain: "pagos", Name: name, Text: reqBlock(name) + "\n#### Scenario: Dos\n"}, true},
		{"otro nombre con el mismo identificador", Requirement{Domain: "pagos", Name: "Cobro general (REQ-30.1)", Text: reqBlock("Cobro general (REQ-30.1)")}, true},
		{"otro dominio", Requirement{Domain: "facturas", Name: name, Text: reqBlock(name)}, true},
		{"final de línea CRLF en lugar de LF", Requirement{Domain: "pagos", Name: name, Text: strings.ReplaceAll(reqBlock(name), "\n", "\r\n")}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckRequirementIDs([]Requirement{added}, []Requirement{tc.baseline})
			if tc.wantFind != (len(got) > 0) {
				t.Fatalf("bloqueo = %v, quería %v: %v", len(got) > 0, tc.wantFind, reasons(got))
			}
			if tc.wantFind && !strings.Contains(got[0].Reason(), "REQ-30.1") {
				t.Errorf("el motivo no nombra REQ-30.1: %q", got[0].Reason())
			}
		})
	}

	t.Run("el compositor solo añade el salto de línea final", func(t *testing.T) {
		noNewline := Requirement{Domain: "pagos", Section: SectionAdded, Name: name, Text: strings.TrimSuffix(reqBlock(name), "\n")}
		if got := CheckRequirementIDs([]Requirement{noNewline}, []Requirement{composed}); len(got) != 0 {
			t.Fatalf("un ADDED sin salto final debía tolerarse tras ensureTrailingNewline: %v", reasons(got))
		}
	})
	t.Run("la baseline sin salto final no coincide con el bloque de la delta", func(t *testing.T) {
		noNewlineBaseline := Requirement{Domain: "pagos", Name: name, Text: strings.TrimSuffix(reqBlock(name), "\n")}
		if got := CheckRequirementIDs([]Requirement{added}, []Requirement{noNewlineBaseline}); len(got) == 0 {
			t.Fatal("la diferencia solo en el salto de línea final de la baseline debía seguir bloqueando")
		}
	})
}
