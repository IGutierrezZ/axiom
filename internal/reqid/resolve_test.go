package reqid_test

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// REQ-30.22: el núcleo de la resolución de un nombre de delta contra los nombres
// de la canónica (D-05).
func TestResolve(t *testing.T) {
	sup := reqid.SupersededSuffix
	cases := []struct {
		name     string
		template string
		names    []string
		query    string
		wantIdx  int
		want     reqid.Match
	}{
		{"nombre sin ID, título exacto", "REQ-<n>.<m>", []string{"Alta", "Baja", "Baja"}, "Baja", 1, reqid.MatchFound},
		{"nombre sin ID, sin título", "REQ-<n>.<m>", []string{"Alta"}, "Otra", -1, reqid.MatchNone},
		{"ID único, título cambiado", "REQ-<n>.<m>", []string{"Alta (REQ-1.1)", "Cobro viejo (REQ-1.2)"}, "Cobro nuevo (REQ-1.2)", 1, reqid.MatchFound},
		{"ID único, mismo título", "REQ-<n>.<m>", []string{"Alta (REQ-1.1)", "Cobro (REQ-1.2)"}, "Cobro (REQ-1.2)", 1, reqid.MatchFound},
		{"ningún no superado con el ID", "REQ-<n>.<m>", []string{"Alta (REQ-1.1)"}, "Cobro (REQ-1.2)", -1, reqid.MatchNone},
		{"varios con el ID y uno con título exacto", "REQ-<n>.<m>", []string{"A (REQ-1.1)", "B (REQ-1.1)"}, "B (REQ-1.1)", 1, reqid.MatchFound},
		{"varios con el ID y ninguno exacto", "REQ-<n>.<m>", []string{"A (REQ-1.1)", "B (REQ-1.1)"}, "C (REQ-1.1)", -1, reqid.MatchAmbiguous},
		{"un superado no cuenta para la resolución por ID", "REQ-<n>.<m>", []string{"A (REQ-1.1)" + sup, "B (REQ-1.2)"}, "A nuevo (REQ-1.1)", -1, reqid.MatchNone},
		{"la delta que cita un superado tal cual sigue por título exacto", "REQ-<n>.<m>", []string{"A (REQ-1.1)" + sup, "B (REQ-1.2)"}, "A (REQ-1.1)" + sup, 0, reqid.MatchFound},
		{"el superado no hace ambiguo al vivo", "REQ-<n>.<m>", []string{"A (REQ-1.1)" + sup, "B (REQ-1.1)"}, "C (REQ-1.1)", 1, reqid.MatchFound},
		{"ID mal formado resuelve por título", "REQ-<n>.<m>", []string{"A (REQ-1)", "B"}, "A (REQ-1)", 0, reqid.MatchFound},
		{"RF: ID único, título cambiado", "RF-<DOM>-<NN>", []string{"Alta (RF-USU-01)", "Baja (RF-USU-02)"}, "Baja masiva (RF-USU-02)", 1, reqid.MatchFound},
		{"RF: otro patrón no es un ID", "RF-<DOM>-<NN>", []string{"Alta (REQ-30.1)"}, "Alta nueva (REQ-30.1)", -1, reqid.MatchNone},
		{"RF: otro patrón resuelve por título", "RF-<DOM>-<NN>", []string{"Alta (REQ-30.1)"}, "Alta (REQ-30.1)", 0, reqid.MatchFound},
		{"RF: repetido y exacto", "RF-<DOM>-<NN>", []string{"A (RF-USU-01)", "B (RF-USU-01)"}, "A (RF-USU-01)", 0, reqid.MatchFound},
		{"RF: repetido y ambiguo", "RF-<DOM>-<NN>", []string{"A (RF-USU-01)", "B (RF-USU-01)"}, "C (RF-USU-01)", -1, reqid.MatchAmbiguous},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := mustParse(t, tc.template)
			idx, got := p.Resolve(tc.names, tc.query)
			if got != tc.want || (got == reqid.MatchFound && idx != tc.wantIdx) {
				t.Fatalf("Resolve(%q) = %d, %v; quería %d, %v", tc.query, idx, got, tc.wantIdx, tc.want)
			}
		})
	}
}

func TestIsSuperseded(t *testing.T) {
	if !reqid.IsSuperseded("Alta (REQ-1.1)" + reqid.SupersededSuffix) {
		t.Error("un nombre con el sufijo es superado")
	}
	if reqid.IsSuperseded("Alta (REQ-1.1)") {
		t.Error("un nombre sin el sufijo no es superado")
	}
	if reqid.SupersededSuffix != " [SUPERSEDED / DEPRECADO]" {
		t.Errorf("sufijo = %q", reqid.SupersededSuffix)
	}
}
