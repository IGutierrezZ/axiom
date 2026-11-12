package livespec

import (
	"errors"
	"strings"
	"testing"
)

// Las pruebas de la obligación de impacto de los bugs ODD (REQ-30.5). El contrato del
// marcador es el de REQ-29.15: la línea exacta `Sin impacto en la spec` más una línea
// `Motivo:` con texto. Todos los textos son inventados.

const (
	impactValid = "## Impacto en la spec\n\nSin impacto en la spec\nMotivo: el fallo era un error tipográfico en un mensaje\n"
)

func bugDelta() string {
	return "## Spec viva\n\n" + capability("facturacion", fence("## ADDED Requirements\n\n"+addedBlock("Alta", "REQ-31.1", "Cuerpo.")))
}

func TestCheckImpactBugContract(t *testing.T) {
	tests := []struct {
		name string
		text string
		// code es el código del problema esperado; vacío si se acepta.
		code string
	}{
		{"bug sin delta ni sección", featureDoc("Tipo: bug\n\n", ""), "impact_missing"},
		{"bug con sección vacía y sin marcador", featureDoc("Tipo: bug\n\n", "## Spec viva\n\nNada todavía.\n"), "impact_missing"},
		{"bug con marcador válido y sin delta", featureDoc("Tipo: bug\n\n", impactValid), ""},
		{"bug con marcador válido y espacios alrededor", featureDoc("Tipo: bug\n\n", "## Impacto en la spec\n\n   Sin impacto en la spec   \nMotivo: razón\n"), ""},
		{"Motivo vacío", featureDoc("Tipo: bug\n\n", "## Impacto en la spec\n\nSin impacto en la spec\nMotivo:\n"), "impact_reason_missing"},
		{"Motivo ausente", featureDoc("Tipo: bug\n\n", "## Impacto en la spec\n\nSin impacto en la spec\n"), "impact_reason_missing"},
		{"punto final", featureDoc("Tipo: bug\n\n", "## Impacto en la spec\n\nSin impacto en la spec.\nMotivo: razón\n"), "impact_marker_invalid"},
		{"otra capitalización", featureDoc("Tipo: bug\n\n", "## Impacto en la spec\n\nsin impacto en la spec\nMotivo: razón\n"), "impact_marker_invalid"},
		{"prefijo en la línea", featureDoc("Tipo: bug\n\n", "## Impacto en la spec\n\n- Sin impacto en la spec\nMotivo: razón\n"), "impact_marker_invalid"},
		{"forma antigua con dos puntos", featureDoc("Tipo: bug\n\n", "## Impacto en la spec\n\nSin impacto en la spec: razón\n"), "impact_marker_invalid"},
		{"bug con delta y sin marcador", featureDoc("Tipo: bug\n\n", bugDelta()), ""},
		{"marcador junto con delta no vacía se ignora", featureDoc("Tipo: bug\n\n", bugDelta()+impactValid), ""},
		{"marcador inválido junto con delta se ignora", featureDoc("Tipo: bug\n\n", bugDelta()+"## Impacto en la spec\n\nSin impacto en la spec.\n"), ""},
		{"bug sin informe de bug", featureDoc("Tipo: bug\n\n", impactValid), ""},
		{"incremento sin sección ni delta", featureDoc("Tipo: incremento\n\n", ""), ""},
		{"sin tipo es incremento", featureDoc("", ""), ""},
		{"incremento con marcador inválido no se rechaza", featureDoc("", "## Impacto en la spec\n\nSin impacto en la spec.\n"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckImpact(parseOK(t, tt.text))
			if tt.code == "" {
				if err != nil {
					t.Fatalf("CheckImpact = %v, se esperaba aceptar", err)
				}
				return
			}
			var docErr *DocError
			if !errors.As(err, &docErr) {
				t.Fatalf("CheckImpact = %v, se esperaba *DocError", err)
			}
			wantProblem(t, docErr.Problems, tt.code, "## Impacto en la spec")
		})
	}
}

func TestCheckImpactNamesTheSectionAndTheRule(t *testing.T) {
	err := CheckImpact(parseOK(t, featureDoc("Tipo: bug\n\n", "")))
	var docErr *DocError
	if !errors.As(err, &docErr) || len(docErr.Problems) != 1 {
		t.Fatalf("CheckImpact = %v", err)
	}
	for _, want := range []string{"## Impacto en la spec", "Sin impacto en la spec", "Motivo:", "## Spec viva"} {
		if !strings.Contains(docErr.Problems[0].Message, want) {
			t.Errorf("el mensaje debe nombrar %q: %s", want, docErr.Problems[0].Message)
		}
	}
	if docErr.Problems[0].Line <= 0 {
		t.Errorf("el problema debe llevar línea: %+v", docErr.Problems[0])
	}
}

func TestPlanIntegrationRejectsABugWithoutImpactBeforeReadingTheLivingSpec(t *testing.T) {
	loc, reg := planFixture(t, nil)
	_, err := PlanIntegration(loc, parseOK(t, featureDoc("Tipo: bug\n\n", "")), reg)
	var docErr *DocError
	if !errors.As(err, &docErr) {
		t.Fatalf("PlanIntegration = %v, se esperaba *DocError", err)
	}
	wantProblem(t, docErr.Problems, "impact_missing", "## Impacto en la spec")
}

func TestPlanIntegrationAcceptsABugWithAValidMarkerAndReportsNoImpact(t *testing.T) {
	loc, reg := planFixture(t, nil)
	plan, err := PlanIntegration(loc, parseOK(t, featureDoc("Tipo: bug\n\n", impactValid)), reg)
	if err != nil {
		t.Fatalf("PlanIntegration: %v", err)
	}
	if plan.Impact != ImpactNone || !plan.NothingToIntegrate() || plan.Reason == "" {
		t.Errorf("plan = %+v", plan)
	}
}
