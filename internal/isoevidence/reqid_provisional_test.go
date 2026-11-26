package isoevidence

import (
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func TestRequirementKeys(t *testing.T) {
	rfDom, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name                         string
		pattern                      reqid.Pattern
		in                           string
		wantValid, wantProv, wantBad []string
	}{
		{"sin identificador", reqid.Default(), "Cobro con IVA", nil, nil, nil},
		{"válido", reqid.Default(), "Cobro (REQ-30.1)", []string{"REQ-30.1"}, nil, nil},
		{"provisional de serie fijada", reqid.Default(), "Cobro (prov:30:1)", nil, []string{"prov:30:1"}, nil},
		{"provisional de etiqueta nueva", reqid.Default(), "Cobro (prov:a:1)", nil, []string{"prov:a:1"}, nil},
		{"provisional y válido", reqid.Default(), "Cobro (prov:30:1, REQ-30.2)", []string{"REQ-30.2"}, []string{"prov:30:1"}, nil},
		{"dos provisionales distintos", reqid.Default(), "A prov:30:1 y prov:30:2", nil, []string{"prov:30:1", "prov:30:2"}, nil},
		{"el mismo provisional cuenta como uno", reqid.Default(), "prov:30:1 (prov:30:1)", nil, []string{"prov:30:1"}, nil},
		{"provisional que el patrón no admite es mal formado", reqid.Default(), "Cobro (prov:USU:1)", nil, nil, []string{"prov:USU:1"}},
		{"provisional laxo es mal formado", reqid.Default(), "Cobro (prov:a1B:1)", nil, nil, []string{"prov:a1B:1"}},
		{"dominio fijado con patrón DOM", rfDom, "Cobro (prov:USU:1)", nil, []string{"prov:USU:1"}, nil},
		{"etiqueta con patrón DOM es mal formada", rfDom, "Cobro (prov:a:1)", nil, nil, []string{"prov:a:1"}},
		{"provisional no es candidato mal formado del patrón", rfDom, "Cobro (prov:RF-3:1)", nil, nil, []string{"prov:RF-3:1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			valid, prov, bad := RequirementKeys(tc.in, tc.pattern)
			if !reflect.DeepEqual(valid, tc.wantValid) || !reflect.DeepEqual(prov, tc.wantProv) || !reflect.DeepEqual(bad, tc.wantBad) {
				t.Errorf("RequirementKeys = %v, %v, %v; quería %v, %v, %v", valid, prov, bad, tc.wantValid, tc.wantProv, tc.wantBad)
			}
		})
	}
}

// RF-ISO-03: un ADDED puede llevar un provisional; fuera de ADDED y combinado con otro
// identificador bloquea; el provisional es único en el cambio.
func TestCheckRequirementIDsProvisional(t *testing.T) {
	rfDom, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		pattern  reqid.Pattern
		delta    []Requirement
		baseline []Requirement
		want     [][]string
	}{
		{"un ADDED con un provisional no bloquea", reqid.Default(),
			[]Requirement{delta(SectionAdded, "pagos", "Cobro con IVA (prov:30:1)")}, nil, nil},
		{"un ADDED con etiqueta de serie nueva no bloquea", reqid.Default(),
			[]Requirement{delta(SectionAdded, "pagos", "Cobro con IVA (prov:a:1)")}, nil, nil},
		{"con patrón DOM, el dominio fijado no bloquea", rfDom,
			[]Requirement{delta(SectionAdded, "usuarios", "Alta (prov:USU:1)")}, nil, nil},
		{"con patrón DOM, la etiqueta bloquea como forma no válida", rfDom,
			[]Requirement{delta(SectionAdded, "usuarios", "Alta (prov:a:1)")}, nil,
			[][]string{{"iso_requirement_id", "usuarios", "ADDED", "RF-<DOM>-<NN>", "prov:a:1"}}},
		{"un provisional no choca con la baseline", reqid.Default(),
			[]Requirement{delta(SectionAdded, "pagos", "Cobro (prov:30:1)")},
			[]Requirement{base("pagos", "Cobro (REQ-30.1)")}, nil},
		{"dos ADDED con el mismo provisional bloquean, en otro dominio también", reqid.Default(),
			[]Requirement{delta(SectionAdded, "pagos", "Cobro (prov:30:1)"), delta(SectionAdded, "envios", "Envío (prov:30:1)")}, nil,
			[][]string{{"iso_requirement_id", "prov:30:1", "se repite en 2", "pagos", "envios"}}},
		{"un provisional en un MODIFIED bloquea", reqid.Default(),
			[]Requirement{delta(SectionModified, "pagos", "Cobro (prov:30:1)")}, nil,
			[][]string{{"iso_requirement_id", "MODIFIED", "prov:30:1", "solo es válido en un ADDED"}}},
		{"un provisional en un REMOVED bloquea", reqid.Default(),
			[]Requirement{delta(SectionRemoved, "pagos", "Cobro (prov:30:1)")}, nil,
			[][]string{{"iso_requirement_id", "REMOVED", "prov:30:1"}}},
		{"un provisional en el nombre nuevo de un RENAMED bloquea", reqid.Default(),
			[]Requirement{{Domain: "pagos", Section: SectionRenamed, Name: "Cobro (prov:30:1)", OldName: "Cobro (REQ-30.1)", Text: reqBlock("Cobro (prov:30:1)")}}, nil,
			[][]string{{"iso_requirement_id", "RENAMED", "prov:30:1"}}},
		{"un provisional en el nombre antiguo de un RENAMED bloquea", reqid.Default(),
			[]Requirement{{Domain: "pagos", Section: SectionRenamed, Name: "Cobro (REQ-30.1)", OldName: "Cobro (prov:30:1)", Text: reqBlock("Cobro (REQ-30.1)")}}, nil,
			[][]string{{"iso_requirement_id", "RENAMED", "nombre antiguo", "prov:30:1"}}},
		{"un nombre con un provisional y un identificador válido es ambiguo", reqid.Default(),
			[]Requirement{delta(SectionAdded, "pagos", "Cobro con IVA (prov:30:1, REQ-30.1)")}, nil,
			[][]string{{"iso_requirement_id", "ambiguo", "prov:30:1", "REQ-30.1"}}},
		{"un nombre con dos provisionales distintos es ambiguo", reqid.Default(),
			[]Requirement{delta(SectionAdded, "pagos", "Cobro (prov:30:1, prov:30:2)")}, nil,
			[][]string{{"iso_requirement_id", "ambiguo"}}},
		{"un provisional laxo bloquea nombrándolo", reqid.Default(),
			[]Requirement{delta(SectionAdded, "pagos", "Cobro (prov:a1B:1)")}, nil,
			[][]string{{"iso_requirement_id", "forma no válida", "prov:a1B:1"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertFindings(t, CheckRequirementIDs(tc.delta, tc.baseline, tc.pattern), CodeRequirementID, StageSpecs, tc.want)
		})
	}
}
