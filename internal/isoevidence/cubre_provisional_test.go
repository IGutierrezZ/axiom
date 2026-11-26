package isoevidence

import (
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// REQ-29.4: Cubre acepta el provisional de un ADDED y lo cuenta como cobertura.
func TestCheckCoverageProvisional(t *testing.T) {
	p := reqid.Default()
	added := req(SectionAdded, "pagos", "Cobro con IVA (prov:30:1)", "Con IVA")
	cases := []struct {
		name  string
		delta []Requirement
		cubre []CubreLine
		code  Code
		want  [][]string
	}{
		{"Cubre acepta el provisional de un ADDED", []Requirement{added}, []CubreLine{cubreOf("tasks.md", 3, "prov:30:1")}, CodeCoverage, nil},
		{"un ADDED provisional sin Cubre bloquea", []Requirement{added}, []CubreLine{cubreOf("tasks.md", 3)}, CodeCoverage,
			[][]string{{"iso_coverage", "prov:30:1", "no está cubierto", "Cubre: prov:30:1"}}},
		{"un provisional citado que ningún ADDED declara bloquea", []Requirement{added}, []CubreLine{cubreOf("tasks.md", 3, "prov:30:1", "prov:30:2")}, CodeCubreUnknown,
			[][]string{{"iso_cubre_unknown", "tasks.md:3", "provisional prov:30:2", "ningún requisito ADDED"}}},
		{"un provisional de otro cambio no se confunde con la baseline", nil, []CubreLine{cubreOf("tasks.md", 1, "prov:30:1")}, CodeCubreUnknown,
			[][]string{{"iso_cubre_unknown", "provisional prov:30:1"}}},
		{"un provisional laxo citado bloquea como desconocido", []Requirement{added}, []CubreLine{cubreOf("tasks.md", 3, "prov:30:1", "prov:a1B:1")}, CodeCubreUnknown,
			[][]string{{"iso_cubre_unknown", "prov:a1B:1"}}},
		{"un provisional en un MODIFIED no cuenta como requisito de la delta", []Requirement{req(SectionModified, "pagos", "Cobro (prov:30:1)", "S")}, nil, CodeCoverage, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckCoverage(tc.delta, BaselineIndex{}, tc.cubre, p)
			assertFindings(t, got, tc.code, StageTasks, tc.want)
		})
	}
}

func TestAffectedFromDeltaProvisional(t *testing.T) {
	p := reqid.Default()
	got := AffectedFromDelta([]Requirement{
		req(SectionAdded, "pagos", "Cobro con IVA (prov:30:1)", "Con IVA", "Sin IVA"),
		req(SectionAdded, "pagos", "Cobro (REQ-30.9)", "Efectivo"),
		req(SectionModified, "pagos", "Cobro (prov:30:2)", "No cuenta"),
	}, p)
	want := []ScenarioKey{
		{RequirementID: "prov:30:1", Scenario: "Con IVA"},
		{RequirementID: "prov:30:1", Scenario: "Sin IVA"},
		{RequirementID: "REQ-30.9", Scenario: "Efectivo"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AffectedFromDelta = %v, quería %v", got, want)
	}
}

func TestRewriteCubre(t *testing.T) {
	m := reqid.NewIDMap([]reqid.Assignment{
		{Provisional: "prov:30:1", Definitive: "REQ-31.2"},
		{Provisional: "prov:a:1", Definitive: "REQ-48.1"},
	})
	text := strings.Join([]string{
		"# Tareas",
		"- [ ] 1.1 Menciona prov:30:1 fuera de Cubre",
		"  Cubre: REQ-29.1, prov:30:1",
		"- Cubre:prov:a:1 ,REQ-29.2",
		"Cubre: prov:30:11",
		"```",
		"Cubre: prov:30:1",
		"```",
		"Texto con Cubre: prov:30:1 en medio",
		"",
	}, "\n")
	want := strings.Join([]string{
		"# Tareas",
		"- [ ] 1.1 Menciona prov:30:1 fuera de Cubre",
		"  Cubre: REQ-29.1, REQ-31.2",
		"- Cubre:REQ-48.1 ,REQ-29.2",
		"Cubre: prov:30:11",
		"```",
		"Cubre: prov:30:1",
		"```",
		"Texto con Cubre: prov:30:1 en medio",
		"",
	}, "\n")
	if got := RewriteCubre(text, m); got != want {
		t.Errorf("RewriteCubre =\n%s\nquería\n%s", got, want)
	}
	if got := RewriteCubre(text, reqid.NewIDMap(nil)); got != text {
		t.Error("un mapa vacío no reescribe nada")
	}
	if got := RewriteCubre(strings.ReplaceAll(text, "\n", "\r\n"), m); got != strings.ReplaceAll(want, "\n", "\r\n") {
		t.Error("los finales de línea CRLF se conservan")
	}
	if once := RewriteCubre(text, m); RewriteCubre(once, m) != once {
		t.Error("la reescritura es idempotente")
	}
}
