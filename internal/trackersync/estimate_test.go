package trackersync

import (
	"errors"
	"strings"
	"testing"
)

func TestParseEstimates(t *testing.T) {
	roster := []string{"backend", "fullstack"}
	tests := []struct {
		name    string
		design  string
		want    map[string]float64
		wantErr string // fragmento de la línea culpable; vacío si no hay error
	}{
		{name: "entero", design: "# D\n\n## Estimación por rol\n\n- fullstack: 6 h\n", want: map[string]float64{"fullstack": 6}},
		{name: "decimal", design: "## Estimación por rol\n- fullstack: 2.5 h\n- backend: 0.25 h\n", want: map[string]float64{"fullstack": 2.5, "backend": 0.25}},
		{name: "sin sección no es error", design: "# D\n\n## Otra\n- fullstack: 6 h\n", want: map[string]float64{}},
		{name: "la justificación se ignora", design: "## Estimación por rol\n\nTexto libre.\n\n- fullstack: 6 h\n\nOtro párrafo.\n## Siguiente\n- backend: mucho\n", want: map[string]float64{"fullstack": 6}},
		{name: "en valla se ignora", design: "```\n## Estimación por rol\n- fullstack: 6 h\n```\n", want: map[string]float64{}},
		{name: "línea de datos en valla de la sección se ignora", design: "## Estimación por rol\n```\n- fullstack: mucho\n```\n- backend: 3 h\n", want: map[string]float64{"backend": 3}},
		{name: "malformada", design: "## Estimación por rol\n- fullstack: mucho\n", wantErr: "- fullstack: mucho"},
		{name: "tres decimales", design: "## Estimación por rol\n- fullstack: 1.255 h\n", wantErr: "1.255"},
		{name: "cero", design: "## Estimación por rol\n- fullstack: 0 h\n", wantErr: "- fullstack: 0 h"},
		{name: "por encima del tope", design: "## Estimación por rol\n- fullstack: 1000.5 h\n", wantErr: "1000.5"},
		{name: "en el tope", design: "## Estimación por rol\n- fullstack: 1000 h\n", want: map[string]float64{"fullstack": 1000}},
		{name: "rol repetido", design: "## Estimación por rol\n- fullstack: 1 h\n- fullstack: 2 h\n", wantErr: "- fullstack: 2 h"},
		{name: "rol fuera del roster", design: "## Estimación por rol\n- qa: 1 h\n", wantErr: "- qa: 1 h"},
		{name: "terminadores CRLF", design: "## Estimación por rol\r\n- fullstack: 4 h\r\n", want: map[string]float64{"fullstack": 4}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseEstimates(tc.design, roster)
			if tc.wantErr != "" {
				var est *EstimateError
				if !errors.As(err, &est) || !strings.Contains(est.Line, tc.wantErr) {
					t.Fatalf("ParseEstimates() error = %v, esperaba estimate_invalid nombrando %q", err, tc.wantErr)
				}
				if !strings.Contains(err.Error(), CodeEstimateInvalid) {
					t.Fatalf("el error debe llevar el código %s: %v", CodeEstimateInvalid, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseEstimates() error = %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("ParseEstimates() = %v, esperaba %v", got, tc.want)
			}
			for role, hours := range tc.want {
				if got[role] != hours {
					t.Fatalf("estimación de %s = %v, esperaba %v", role, got[role], hours)
				}
			}
		})
	}
}

func TestParseEstimateFlag(t *testing.T) {
	valid := map[string]struct {
		role  string
		hours float64
	}{
		"fullstack=6":     {"fullstack", 6},
		" FullStack=2.5 ": {"fullstack", 2.5},
		"backend=0.01":    {"backend", 0.01},
		"fullstack=1000":  {"fullstack", 1000},
	}
	for input, want := range valid {
		role, hours, err := ParseEstimateFlag(input)
		if err != nil || role != want.role || hours != want.hours {
			t.Errorf("ParseEstimateFlag(%q) = %q, %v, %v; esperaba %q, %v", input, role, hours, err, want.role, want.hours)
		}
	}
	for _, input := range []string{"", "fullstack", "fullstack=", "fullstack=0", "fullstack=-1", "fullstack=abc", "fullstack=1.255", "fullstack=1001", "fullstack=1,5", "=5", "fullstack=5 h"} {
		var est *EstimateError
		if _, _, err := ParseEstimateFlag(input); !errors.As(err, &est) {
			t.Errorf("ParseEstimateFlag(%q) debía dar *EstimateError, dio %v", input, err)
		}
	}
}
