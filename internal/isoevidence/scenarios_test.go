package isoevidence

import (
	"reflect"
	"testing"
)

func TestScenarios(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"sin escenarios", "### Requirement: Uno (REQ-1.1)\n\nTexto.\n", nil},
		{"dos escenarios", "### Requirement: Uno (REQ-1.1)\n\n#### Scenario: A\n\n#### Scenario:   B  \n", []string{"A", "B"}},
		{"escenario dentro de una valla no cuenta", "### Requirement: Uno (REQ-1.1)\n\n```markdown\n#### Scenario: Falso\n```\n\n#### Scenario: Real\n", []string{"Real"}},
		{"valla de tildes", "~~~\n#### Scenario: Falso\n~~~\n", nil},
		{"valla sin cerrar cubre hasta el final", "```\n#### Scenario: Falso\n#### Scenario: Otro\n", nil},
		{"final de línea CRLF", "#### Scenario: A\r\n#### Scenario: B\r\n", []string{"A", "B"}},
		{"encabezado de otro nivel no cuenta", "### Scenario: A\n##### Scenario: B\n", nil},
		{"una valla de cierre más corta no cierra", "````\n```\n#### Scenario: Falso\n````\n#### Scenario: Real\n", []string{"Real"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Scenarios(tc.in); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Scenarios = %v, quería %v", got, tc.want)
			}
		})
	}
}

func TestCheckScenarios(t *testing.T) {
	withScenarios := "### Requirement: Cobro (REQ-30.1)\n\n#### Scenario: A\n\n#### Scenario: B\n"
	cases := []struct {
		name  string
		delta []Requirement
		want  [][]string
	}{
		{
			name:  "ADDED con escenarios",
			delta: []Requirement{{Domain: "pagos", Section: SectionAdded, Name: "Cobro (REQ-30.1)", Text: withScenarios}},
		},
		{
			name:  "ADDED sin escenarios",
			delta: []Requirement{{Domain: "pagos", Section: SectionAdded, Name: "Cobro (REQ-30.1)", Text: "### Requirement: Cobro (REQ-30.1)\n\nTexto.\n"}},
			want:  [][]string{{"blocked(iso_requirement_scenarios)", `"pagos"`, "ADDED", `"Cobro (REQ-30.1)"`, "#### Scenario:"}},
		},
		{
			name:  "MODIFIED sin escenarios",
			delta: []Requirement{{Domain: "pagos", Section: SectionModified, Name: "Cobro (REQ-30.1)", Text: "### Requirement: Cobro (REQ-30.1)\n"}},
			want:  [][]string{{"MODIFIED", "ningún escenario"}},
		},
		{
			name:  "escenario solo dentro de una valla cuenta como ausente",
			delta: []Requirement{{Domain: "pagos", Section: SectionAdded, Name: "Cobro (REQ-30.1)", Text: "### Requirement: Cobro (REQ-30.1)\n\n```\n#### Scenario: Falso\n```\n"}},
			want:  [][]string{{"ningún escenario"}},
		},
		{
			name: "REMOVED y RENAMED quedan exentos",
			delta: []Requirement{
				{Domain: "pagos", Section: SectionRemoved, Name: "Viejo (REQ-30.9)", Text: "### Requirement: Viejo (REQ-30.9)\n"},
				{Domain: "pagos", Section: SectionRenamed, OldName: "A", Name: "B (REQ-30.8)"},
			},
		},
		{
			name:  "nombre de escenario repetido dentro de un requisito",
			delta: []Requirement{{Domain: "pagos", Section: SectionAdded, Name: "Cobro (REQ-30.1)", Text: "### Requirement: Cobro (REQ-30.1)\n\n#### Scenario: A\n\n#### Scenario: A\n"}},
			want:  [][]string{{`"Cobro (REQ-30.1)"`, `"A"`, "se repite 2 veces"}},
		},
		{
			name: "el mismo nombre en requisitos distintos no es repetido",
			delta: []Requirement{
				{Domain: "pagos", Section: SectionAdded, Name: "Cobro (REQ-30.1)", Text: "### Requirement: Cobro (REQ-30.1)\n\n#### Scenario: A\n"},
				{Domain: "pagos", Section: SectionAdded, Name: "Pago (REQ-30.2)", Text: "### Requirement: Pago (REQ-30.2)\n\n#### Scenario: A\n"},
			},
		},
		{
			name: "un motivo por requisito afectado",
			delta: []Requirement{
				{Domain: "pagos", Section: SectionAdded, Name: "Cobro (REQ-30.1)", Text: "### Requirement: Cobro (REQ-30.1)\n"},
				{Domain: "facturas", Section: SectionModified, Name: "Factura (REQ-31.1)", Text: "### Requirement: Factura (REQ-31.1)\n"},
			},
			want: [][]string{{`"Cobro (REQ-30.1)"`}, {`"Factura (REQ-31.1)"`}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertFindings(t, CheckScenarios(tc.delta), CodeRequirementScenarios, StageSpecs, tc.want)
		})
	}
}
