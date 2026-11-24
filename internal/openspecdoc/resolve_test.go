package openspecdoc

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func mustPattern(t testing.TB, template string) reqid.Pattern {
	t.Helper()
	p, err := reqid.Parse(template)
	if err != nil {
		t.Fatalf("Parse(%q): %v", template, err)
	}
	return p
}

// REQ-30.22: Resolve empareja el nombre de una delta con la canónica por
// identificador, con vuelta al título exacto.
func TestResolve(t *testing.T) {
	sup := SupersededSuffix
	cases := []struct {
		name     string
		template string
		names    []string
		query    string
		wantIdx  int
		want     Match
	}{
		{"sin ID: título exacto, primera coincidencia", "REQ-<n>.<m>", []string{"Alta", "Baja", "Baja"}, "Baja", 1, MatchFound},
		{"sin ID: sin título", "REQ-<n>.<m>", []string{"Alta"}, "Otra", -1, MatchNone},
		{"ID único no superado", "REQ-<n>.<m>", []string{"Alta (REQ-1.1)", "Cobro viejo (REQ-1.2)"}, "Cobro nuevo (REQ-1.2)", 1, MatchFound},
		{"ningún no superado con el ID", "REQ-<n>.<m>", []string{"Alta (REQ-1.1)"}, "Cobro (REQ-1.2)", -1, MatchNone},
		{"varios con ID y uno exacto", "REQ-<n>.<m>", []string{"A (REQ-1.1)", "B (REQ-1.1)"}, "B (REQ-1.1)", 1, MatchFound},
		{"varios con ID y ninguno exacto", "REQ-<n>.<m>", []string{"A (REQ-1.1)", "B (REQ-1.1)"}, "C (REQ-1.1)", -1, MatchAmbiguous},
		{"los superados quedan fuera", "REQ-<n>.<m>", []string{"A (REQ-1.1)" + sup}, "A (REQ-1.1)", -1, MatchNone},
		{"RF: ID único con título cambiado", "RF-<DOM>-<NN>", []string{"Alta (RF-USU-01)", "Baja (RF-USU-02)"}, "Baja masiva (RF-USU-02)", 1, MatchFound},
		{"RF: ambiguo", "RF-<DOM>-<NN>", []string{"A (RF-USU-01)", "B (RF-USU-01)"}, "C (RF-USU-01)", -1, MatchAmbiguous},
		{"RF: exacto entre repetidos", "RF-<DOM>-<NN>", []string{"A (RF-USU-01)", "B (RF-USU-01)"}, "A (RF-USU-01)", 0, MatchFound},
		{"RF: ningún superado", "RF-<DOM>-<NN>", []string{"A (RF-USU-01)" + sup}, "A nuevo (RF-USU-01)", -1, MatchNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idx, got := Resolve(tc.names, tc.query, mustPattern(t, tc.template))
			if got != tc.want || (got == MatchFound && idx != tc.wantIdx) {
				t.Fatalf("Resolve(%q) = %d, %v; quería %d, %v", tc.query, idx, got, tc.wantIdx, tc.want)
			}
		})
	}
	// El valor cero del patrón es el defecto.
	if idx, got := Resolve([]string{"A (REQ-1.1)"}, "B (REQ-1.1)", reqid.Pattern{}); got != MatchFound || idx != 0 {
		t.Errorf("el patrón cero debe ser el defecto: %d, %v", idx, got)
	}
}
