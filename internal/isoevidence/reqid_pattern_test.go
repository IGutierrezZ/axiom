package isoevidence

import (
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

func rfPattern(t testing.TB) reqid.Pattern {
	t.Helper()
	p, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// REQ-29.3 y REQ-44.5: con RF-<DOM>-<NN> el análisis del nombre lo hace el patrón del proyecto.
func TestRequirementIDsWithDeclaredPattern(t *testing.T) {
	cases := []struct {
		name          string
		in            string
		wantValid     []string
		wantMalformed []string
	}{
		{"válido", "Alta (RF-USU-01)", []string{"RF-USU-01"}, nil},
		{"sin relleno es mal formado", "Alta (RF-USU-1)", nil, []string{"RF-USU-1"}},
		{"sufijo extra es mal formado", "Alta (RF-USU-01-bis)", nil, []string{"RF-USU-01-bis"}},
		{"minúsculas no son un identificador", "Alta (rf-usu-01)", nil, nil},
		{"límite de palabra", "Alta (XRF-USU-01)", nil, nil},
		{"el formato por defecto no es un identificador en este proyecto", "Alta (REQ-30.1)", nil, nil},
		{"repetido en un título cuenta una vez", "RF-USU-01 Alta (RF-USU-01)", []string{"RF-USU-01"}, nil},
		{"dos distintos", "Alta (RF-FAC-01, RF-FAC-02)", []string{"RF-FAC-01", "RF-FAC-02"}, nil},
	}
	p := rfPattern(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			valid, _, malformed := RequirementKeys(tc.in, p)
			if !reflect.DeepEqual(valid, tc.wantValid) || !reflect.DeepEqual(malformed, tc.wantMalformed) {
				t.Errorf("RequirementKeys(%q) = %v, %v; quería %v, %v", tc.in, valid, malformed, tc.wantValid, tc.wantMalformed)
			}
		})
	}
}

// REQ-29.3: los motivos nombran el patrón efectivo; con el defecto son idénticos a los de antes.
func TestCheckRequirementIDsWithDeclaredPattern(t *testing.T) {
	p := rfPattern(t)
	cases := []struct {
		name     string
		delta    []Requirement
		baseline []Requirement
		want     [][]string
	}{
		{"nombre válido no bloquea", []Requirement{delta(SectionAdded, "usuarios", "Alta (RF-USU-01)")}, nil, nil},
		{
			"ADDED sin identificador nombra el patrón efectivo",
			[]Requirement{delta(SectionAdded, "usuarios", "Alta")}, nil,
			[][]string{{"blocked(iso_requirement_id)", "RF-<DOM>-<NN>", "ADDED"}},
		},
		{
			"el formato por defecto bloquea nombrando el patrón",
			[]Requirement{delta(SectionAdded, "usuarios", "Alta (REQ-30.1)")}, nil,
			[][]string{{"RF-<DOM>-<NN>"}},
		},
		{
			"mal formado",
			[]Requirement{delta(SectionAdded, "usuarios", "Alta (RF-USU-1)")}, nil,
			[][]string{{"forma no válida: RF-USU-1"}},
		},
		{
			"dos identificadores distintos",
			[]Requirement{delta(SectionAdded, "usuarios", "Alta (RF-FAC-01, RF-FAC-02)")}, nil,
			[][]string{{"ambiguo", "RF-FAC-01", "RF-FAC-02"}},
		},
		{
			"ADDED con un identificador existente en la baseline",
			[]Requirement{delta(SectionAdded, "usuarios", "Otra alta (RF-USU-01)")},
			[]Requirement{base("usuarios", "Alta (RF-USU-01)")},
			[][]string{{"RF-USU-01", "ya existe en la baseline", `"usuarios"`}},
		},
		{
			"MODIFIED sin identificador remite al RENAMED",
			[]Requirement{delta(SectionModified, "usuarios", "Alta")}, nil,
			[][]string{{"MODIFIED", "RENAMED", "RF-<DOM>-<NN>"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertFindings(t, CheckRequirementIDs(tc.delta, tc.baseline, p), CodeRequirementID, StageSpecs, tc.want)
		})
	}
}

// REQ-29.3 y REQ-30.22: un MODIFIED o REMOVED con el título cambiado y el mismo
// identificador pasa igual que con el título exacto, en CheckRequirementIDs, en el índice
// de la baseline y en la tolerancia D17; ADDED sigue por título.
func TestRequirementIDsByIDWithChangedTitle(t *testing.T) {
	for _, tc := range []struct {
		name     string
		template string
		oldName  string
		newName  string
		id       string
	}{
		{"REQ-<n>.<m>", "REQ-<n>.<m>", "Cobro (REQ-30.1)", "Cobro con IVA (REQ-30.1)", "REQ-30.1"},
		{"RF-<DOM>-<NN>", "RF-<DOM>-<NN>", "Alta (RF-USU-01)", "Alta masiva (RF-USU-01)", "RF-USU-01"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := reqid.Parse(tc.template)
			if err != nil {
				t.Fatal(err)
			}
			baseline := []Requirement{base("pagos", tc.oldName)}
			for _, section := range []Section{SectionModified, SectionRemoved} {
				same := CheckRequirementIDs([]Requirement{delta(section, "pagos", tc.oldName)}, baseline, p)
				changed := CheckRequirementIDs([]Requirement{delta(section, "pagos", tc.newName)}, baseline, p)
				if len(same) != 0 || len(changed) != 0 {
					t.Errorf("%s: el título cambiado con el mismo ID debe pasar igual: %v / %v", section, reasons(same), reasons(changed))
				}
			}

			// El índice de la baseline sigue indexando por identificador, no por título.
			index := NewBaselineIndex(baseline, p)
			if len(index[tc.id]) != 1 || index[tc.id][0].Name != tc.oldName {
				t.Errorf("el índice debe resolver %s a la baseline: %v", tc.id, index)
			}

			// ADDED sigue por título: el mismo ID bajo otro título bloquea; el mismo título y
			// bloque (archivado interrumpido, D17) se tolera.
			if got := CheckRequirementIDs([]Requirement{delta(SectionAdded, "pagos", tc.newName)}, baseline, p); len(got) != 1 {
				t.Errorf("un ADDED con un ID existente bajo otro título debe bloquear: %v", reasons(got))
			}
			if got := CheckRequirementIDs([]Requirement{delta(SectionAdded, "pagos", tc.oldName)}, baseline, p); len(got) != 0 {
				t.Errorf("un ADDED ya compuesto (mismo nombre y bloque) se tolera: %v", reasons(got))
			}
		})
	}
}

// REQ-29.14: con un ID repetido en la baseline el índice conserva a todos y la consulta del
// bug lo trata como ambiguo; el título cambiado de un MODIFIED no lo altera.
func TestBaselineRepeatedIDStaysAmbiguous(t *testing.T) {
	p := rfPattern(t)
	baseline := []Requirement{base("usuarios", "Alta (RF-USU-01)"), base("usuarios", "Alta vieja (RF-USU-01)")}
	index := NewBaselineIndex(baseline, p)
	if len(index["RF-USU-01"]) != 2 {
		t.Fatalf("el índice conserva todos los requisitos del ID: %v", index)
	}
	if got := CheckRequirementIDs([]Requirement{delta(SectionModified, "usuarios", "Alta nueva (RF-USU-01)")}, baseline, p); len(got) != 0 {
		t.Errorf("el MODIFIED por ID no bloquea en la evidencia ISO: %v", reasons(got))
	}
	cubre := []CubreLine{{File: "tasks.md", Line: 3, IDs: []string{"RF-USU-01"}}}
	got := CheckExistingRequirementBug(index, cubre, p)
	assertFindings(t, got, CodeBugRequirement, StageTasks, [][]string{{"RF-USU-01", "ambiguo"}})
}

// REQ-29.4 y REQ-29.14: cobertura y consulta del bug con RF-<DOM>-<NN>.
func TestCoverageAndBugQueryWithDeclaredPattern(t *testing.T) {
	p := rfPattern(t)
	deltas := []Requirement{
		delta(SectionAdded, "usuarios", "Alta (RF-USU-02)"),
		delta(SectionAdded, "usuarios", "Baja (RF-USU-03)"),
	}
	index := NewBaselineIndex([]Requirement{req("", "usuarios", "Alta masiva (RF-USU-01)", "Escenario A")}, p)

	t.Run("Cubre con varios identificadores sin motivo", func(t *testing.T) {
		cubre := []CubreLine{{File: "tasks.md", Line: 4, IDs: []string{"RF-USU-02", "RF-USU-03"}}}
		if got := CheckCoverage(deltas, index, cubre, p); len(got) != 0 {
			t.Fatalf("la cobertura completa no bloquea: %v", reasons(got))
		}
	})
	t.Run("un requisito sin cubrir nombra su identificador y la continuación", func(t *testing.T) {
		cubre := []CubreLine{{File: "tasks.md", Line: 4, IDs: []string{"RF-USU-02"}}}
		got := CheckCoverage(deltas, index, cubre, p)
		assertFindings(t, got, CodeCoverage, StageTasks, [][]string{{"RF-USU-03", "Cubre: RF-USU-03"}})
	})
	t.Run("un identificador del formato por defecto es desconocido", func(t *testing.T) {
		cubre := []CubreLine{{File: "tasks.md", Line: 4, IDs: []string{"RF-USU-02", "RF-USU-03", "REQ-30.1"}}}
		got := CheckCoverage(deltas, index, cubre, p)
		assertFindings(t, got, CodeCubreUnknown, StageTasks, [][]string{{"REQ-30.1"}})
	})
	t.Run("consulta de bug con un requisito existente de la baseline", func(t *testing.T) {
		cubre := []CubreLine{{File: "tasks.md", Line: 4, IDs: []string{"RF-USU-01"}}}
		if got := CheckExistingRequirementBug(index, cubre, p); len(got) != 0 {
			t.Fatalf("el requisito existe y tiene escenarios: %v", reasons(got))
		}
		if got := AffectedFromBaseline(index, []string{"RF-USU-01"}); !reflect.DeepEqual(got, []ScenarioKey{{"RF-USU-01", "Escenario A"}}) {
			t.Errorf("AffectedFromBaseline = %v", got)
		}
	})
	t.Run("sin Cubre la continuación del bug nombra la plantilla", func(t *testing.T) {
		got := CheckExistingRequirementBug(index, nil, p)
		if len(got) != 1 || !strings.Contains(got[0].Reason(), "Cubre: RF-<DOM>-<NN>") {
			t.Fatalf("la continuación debe citar el patrón efectivo: %v", reasons(got))
		}
	})
	t.Run("escenarios afectados por identificador del patrón", func(t *testing.T) {
		d := []Requirement{req(SectionAdded, "usuarios", "Alta (RF-USU-02)", "A", "B"), req(SectionAdded, "usuarios", "Otra (REQ-30.1)", "C")}
		want := []ScenarioKey{{"RF-USU-02", "A"}, {"RF-USU-02", "B"}}
		if got := AffectedFromDelta(d, p); !reflect.DeepEqual(got, want) {
			t.Errorf("AffectedFromDelta = %v, quería %v", got, want)
		}
	})
}

// REQ-29.3 y REQ-29.4: con el defecto los textos de continuación no cambian ni un byte.
func TestDefaultPatternKeepsContinuationTexts(t *testing.T) {
	p := reqid.Default()
	got := CheckRequirementIDs([]Requirement{delta(SectionAdded, "pagos", "Cobro")}, nil, p)
	if len(got) != 1 || got[0].Continuation != "añade o corrige en el nombre un identificador REQ-<n>.<m>" {
		t.Fatalf("continuación = %v", reasons(got))
	}
	if !strings.Contains(got[0].Detail, "el nombre no contiene un identificador REQ-<n>.<m>") {
		t.Errorf("detalle = %q", got[0].Detail)
	}
	bug := CheckExistingRequirementBug(NewBaselineIndex(nil, p), nil, p)
	if len(bug) != 1 || bug[0].Continuation != "cita el requisito que el bug restaura con `Cubre: REQ-<n>.<m>` en alguna tarea" {
		t.Fatalf("continuación del bug = %v", reasons(bug))
	}
}
