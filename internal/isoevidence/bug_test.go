package isoevidence

import (
	"strings"
	"testing"
)

const validReport = "## Informe de bug\n\nObservado: el total omite el IVA reducido.\nEsperado: el total aplica el IVA de cada línea.\n"

func TestParseBugDeclaration(t *testing.T) {
	cases := []struct {
		name        string
		proposal    string
		wantKind    BugKind
		wantValid   bool
		wantCodes   []Code
		wantMissing []string // fragmentos, uno por elemento de Missing
		wantMotivo  string
	}{
		{
			name:     "sin ninguna sección: estado idéntico al actual",
			proposal: "# Propuesta\n\n## Intención\n\nAlgo.\n",
			wantKind: BugKindNone,
		},
		{
			name:      "informe completo sin sección de impacto",
			proposal:  validReport,
			wantKind:  BugKindExistingRequirement,
			wantValid: true,
		},
		{
			name:      "la referencia es opcional y no cambia nada",
			proposal:  validReport + "Referencia: JIRA-1234\n",
			wantKind:  BugKindExistingRequirement,
			wantValid: true,
		},
		{
			name:        "sin Esperado:",
			proposal:    "## Informe de bug\n\nObservado: algo\n",
			wantKind:    BugKindExistingRequirement,
			wantCodes:   []Code{CodeBugReportIncomplete},
			wantMissing: []string{"`Esperado:`"},
		},
		{
			name:        "Esperado: vacío",
			proposal:    "## Informe de bug\n\nObservado: algo\nEsperado:\n",
			wantKind:    BugKindExistingRequirement,
			wantCodes:   []Code{CodeBugReportIncomplete},
			wantMissing: []string{"`Esperado:` con texto"},
		},
		{
			name:        "Observado: con solo espacios",
			proposal:    "## Informe de bug\n\nObservado:    \nEsperado: algo\n",
			wantKind:    BugKindExistingRequirement,
			wantCodes:   []Code{CodeBugReportIncomplete},
			wantMissing: []string{"`Observado:` con texto"},
		},
		{
			name:        "sin Observado: ni Esperado:",
			proposal:    "## Informe de bug\n\nTexto libre.\n",
			wantKind:    BugKindExistingRequirement,
			wantCodes:   []Code{CodeBugReportIncomplete, CodeBugReportIncomplete},
			wantMissing: []string{"`Observado:`", "`Esperado:`"},
		},
		{
			name:       "bug sin impacto completo",
			proposal:   validReport + "\n## Impacto en la spec\n\nSin impacto en la spec\nMotivo: el defecto estaba en el script de despliegue.\n",
			wantKind:   BugKindNoImpact,
			wantValid:  true,
			wantMotivo: "el defecto estaba en el script de despliegue.",
		},
		{
			name:        "la marca sin Motivo:",
			proposal:    validReport + "\n## Impacto en la spec\n\nSin impacto en la spec\n",
			wantKind:    BugKindNoImpact,
			wantCodes:   []Code{CodeBugNoImpactIncomplete},
			wantMissing: []string{"`Motivo:`"},
		},
		{
			name:        "la marca con Motivo: vacío",
			proposal:    validReport + "\n## Impacto en la spec\n\nSin impacto en la spec\nMotivo:\n",
			wantKind:    BugKindNoImpact,
			wantCodes:   []Code{CodeBugNoImpactIncomplete},
			wantMissing: []string{"`Motivo:` con texto"},
		},
		{
			name:        "la sección sin la línea literal",
			proposal:    validReport + "\n## Impacto en la spec\n\nImpacto: ninguno\nMotivo: ninguno\n",
			wantKind:    BugKindNoImpact,
			wantCodes:   []Code{CodeBugNoImpactIncomplete},
			wantMissing: []string{"`Sin impacto en la spec`"},
			wantMotivo:  "ninguno",
		},
		{
			name:        "la marca con punto final no se reconoce",
			proposal:    validReport + "\n## Impacto en la spec\n\nSin impacto en la spec.\nMotivo: x\n",
			wantKind:    BugKindNoImpact,
			wantCodes:   []Code{CodeBugNoImpactIncomplete},
			wantMissing: []string{"`Sin impacto en la spec`"},
			wantMotivo:  "x",
		},
		{
			name:        "la marca en minúsculas no se reconoce",
			proposal:    validReport + "\n## Impacto en la spec\n\nsin impacto en la spec\nMotivo: x\n",
			wantKind:    BugKindNoImpact,
			wantCodes:   []Code{CodeBugNoImpactIncomplete},
			wantMissing: []string{"`Sin impacto en la spec`"},
			wantMotivo:  "x",
		},
		{
			name:       "la marca tolera espacio inicial y final",
			proposal:   validReport + "\n## Impacto en la spec\n\n  Sin impacto en la spec  \nMotivo: x\n",
			wantKind:   BugKindNoImpact,
			wantValid:  true,
			wantMotivo: "x",
		},
		{
			name:        "impacto sin informe de bug",
			proposal:    "## Impacto en la spec\n\nSin impacto en la spec\nMotivo: x\n",
			wantKind:    BugKindNoImpact,
			wantCodes:   []Code{CodeBugReportIncomplete},
			wantMissing: []string{"`## Informe de bug`"},
			wantMotivo:  "x",
		},
		{
			name:        "faltan cosas de las dos secciones: salen los dos tokens",
			proposal:    "## Informe de bug\n\nObservado: algo\n\n## Impacto en la spec\n\nTexto.\n",
			wantKind:    BugKindNoImpact,
			wantCodes:   []Code{CodeBugReportIncomplete, CodeBugNoImpactIncomplete, CodeBugNoImpactIncomplete},
			wantMissing: []string{"`Esperado:`", "`Sin impacto en la spec`", "`Motivo:`"},
		},
		{
			name:     "secciones dentro de una valla no cuentan",
			proposal: "```markdown\n## Informe de bug\n\nObservado: a\nEsperado: b\n```\n",
			wantKind: BugKindNone,
		},
		{
			name:        "las líneas de una valla dentro de la sección no cuentan",
			proposal:    "## Informe de bug\n\nObservado: algo\n\n```\nEsperado: dentro de la valla\n```\n",
			wantKind:    BugKindExistingRequirement,
			wantCodes:   []Code{CodeBugReportIncomplete},
			wantMissing: []string{"`Esperado:`"},
		},
		{
			name:        "la sección termina en el siguiente encabezado de nivel 2",
			proposal:    "## Informe de bug\n\nObservado: algo\n\n## Otra\n\nEsperado: fuera de la sección\n",
			wantKind:    BugKindExistingRequirement,
			wantCodes:   []Code{CodeBugReportIncomplete},
			wantMissing: []string{"`Esperado:`"},
		},
		{
			name:      "un subencabezado de nivel 3 no cierra la sección",
			proposal:  "## Informe de bug\n\nObservado: a\n\n### Detalle\n\nEsperado: b\n",
			wantKind:  BugKindExistingRequirement,
			wantValid: true,
		},
		{
			name:     "el encabezado de nivel 3 no es la sección",
			proposal: "### Informe de bug\n\nObservado: a\nEsperado: b\n",
			wantKind: BugKindNone,
		},
		{
			name:      "final de línea CRLF",
			proposal:  strings.ReplaceAll(validReport, "\n", "\r\n"),
			wantKind:  BugKindExistingRequirement,
			wantValid: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := ParseBugDeclaration(tc.proposal)
			if d.Kind != tc.wantKind {
				t.Errorf("Kind = %q, quería %q", d.Kind, tc.wantKind)
			}
			if d.Valid != tc.wantValid {
				t.Errorf("Valid = %v, quería %v (falta: %v)", d.Valid, tc.wantValid, d.Missing)
			}
			if d.Motivo != tc.wantMotivo {
				t.Errorf("Motivo = %q, quería %q", d.Motivo, tc.wantMotivo)
			}
			if len(d.Missing) != len(tc.wantMissing) {
				t.Fatalf("Missing = %v, quería %d elementos", d.Missing, len(tc.wantMissing))
			}
			for i, fragment := range tc.wantMissing {
				if !strings.Contains(d.Missing[i], fragment) {
					t.Errorf("Missing[%d] = %q no contiene %q", i, d.Missing[i], fragment)
				}
			}
			findings := d.Findings()
			if len(findings) != len(tc.wantCodes) {
				t.Fatalf("hallazgos = %v, quería %d", reasons(findings), len(tc.wantCodes))
			}
			for i, f := range findings {
				if f.Code != tc.wantCodes[i] {
					t.Errorf("hallazgo %d: token %s, quería %s", i, f.Code, tc.wantCodes[i])
				}
				if f.Stage != StageSpecs {
					t.Errorf("hallazgo %d: etapa %s, quería %s", i, f.Stage, StageSpecs)
				}
				if !strings.Contains(f.Reason(), tc.wantMissing[i]) {
					t.Errorf("hallazgo %d: el motivo %q no nombra %q", i, f.Reason(), tc.wantMissing[i])
				}
			}
		})
	}
}
