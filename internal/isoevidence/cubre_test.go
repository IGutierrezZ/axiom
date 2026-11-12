package isoevidence

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseCubre(t *testing.T) {
	text := strings.Join([]string{
		"- [ ] 1.1 Tarea uno",
		"  Cubre: REQ-29.1",
		"- [ ] 1.2 Tarea dos",
		"  Cubre: REQ-29.1, REQ-29.2 ,REQ-9",
		"- Cubre: REQ-30.1 REQ-30.2",
		"Cubre:",
		"```",
		"Cubre: REQ-99.9",
		"```",
		"Cubre REQ-1.1 (sin dos puntos)",
		"Texto con Cubre: REQ-1.2 en medio",
		"cubre: REQ-1.3",
	}, "\n")
	got := ParseCubre("tasks.md", text)
	want := []CubreLine{
		{File: "tasks.md", Line: 2, IDs: []string{"REQ-29.1"}},
		{File: "tasks.md", Line: 4, IDs: []string{"REQ-29.1", "REQ-29.2", "REQ-9"}},
		{File: "tasks.md", Line: 5, IDs: []string{"REQ-30.1", "REQ-30.2"}},
		{File: "tasks.md", Line: 6, IDs: []string{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseCubre = %#v, quería %#v", got, want)
	}
}

func req(section Section, domain, name string, scenarios ...string) Requirement {
	text := "### Requirement: " + name + "\n\nTexto.\n"
	for _, s := range scenarios {
		text += "\n#### Scenario: " + s + "\n\n- **DADO** algo\n"
	}
	return Requirement{Domain: domain, Section: section, Name: name, Text: text}
}

func cubreOf(file string, line int, ids ...string) CubreLine {
	return CubreLine{File: file, Line: line, IDs: ids}
}

func TestCheckCoverage(t *testing.T) {
	d := []Requirement{
		req(SectionAdded, "iso", "Declaración ISO (REQ-29.1)", "A"),
		req(SectionAdded, "iso", "Proyección (REQ-29.2)", "B"),
	}
	index := NewBaselineIndex([]Requirement{req("", "sdd", "Subcomando axiom sdd status (REQ-13.1)", "S1", "S2")})

	cases := []struct {
		name  string
		delta []Requirement
		cubre []CubreLine
		want  [][]string
		codes []Code
	}{
		{
			name:  "un requisito sin tarea que lo cubra",
			delta: d,
			cubre: []CubreLine{cubreOf("tasks.md", 3, "REQ-29.1")},
			want:  [][]string{{"REQ-29.2", "no está cubierto"}},
			codes: []Code{CodeCoverage},
		},
		{
			name:  "la unión de varios ficheros cubre la delta",
			delta: d,
			cubre: []CubreLine{cubreOf("tasks.core.md", 3, "REQ-29.1"), cubreOf("tasks.web.md", 8, "REQ-29.2")},
		},
		{
			name:  "una tarea puede citar varios requisitos",
			delta: d,
			cubre: []CubreLine{cubreOf("tasks.md", 3, "REQ-29.1", "REQ-29.2")},
		},
		{
			name:  "un identificador inexistente bloquea",
			delta: d,
			cubre: []CubreLine{cubreOf("tasks.md", 3, "REQ-29.1", "REQ-29.2", "REQ-999.1")},
			want:  [][]string{{"tasks.md:3", "REQ-999.1", "ni en la baseline"}},
			codes: []Code{CodeCubreUnknown},
		},
		{
			name:  "un identificador mal formado es desconocido, una vez",
			delta: d,
			cubre: []CubreLine{cubreOf("tasks.md", 3, "REQ-29.1", "REQ-29.2", "REQ-9"), cubreOf("tasks.md", 9, "REQ-9")},
			want:  [][]string{{"tasks.md:3", "REQ-9"}},
			codes: []Code{CodeCubreUnknown},
		},
		{
			name:  "un requisito de la baseline se admite y no cubre la delta",
			delta: d,
			cubre: []CubreLine{cubreOf("tasks.md", 3, "REQ-29.1", "REQ-29.2"), cubreOf("tasks.md", 5, "REQ-13.1")},
		},
		{
			name:  "un requisito de la baseline citado no suple la cobertura de la delta",
			delta: d[:1],
			cubre: []CubreLine{cubreOf("tasks.md", 5, "REQ-13.1")},
			want:  [][]string{{"REQ-29.1", "no está cubierto"}},
			codes: []Code{CodeCoverage},
		},
		{
			name:  "baseline sin delta: solo se comprueban los desconocidos",
			delta: nil,
			cubre: []CubreLine{cubreOf("tasks.md", 2, "REQ-13.1", "REQ-13.99")},
			want:  [][]string{{"REQ-13.99"}},
			codes: []Code{CodeCubreUnknown},
		},
		{
			name:  "sin ningún Cubre: se reporta cada requisito de la delta",
			delta: d,
			want:  [][]string{{"REQ-29.1"}, {"REQ-29.2"}},
			codes: []Code{CodeCoverage, CodeCoverage},
		},
		{
			name:  "un RENAMED se cubre con el identificador de su nombre nuevo",
			delta: []Requirement{{Domain: "pagos", Section: SectionRenamed, OldName: "Cobro (REQ-30.1)", Name: "Cobro con IVA (REQ-30.1)"}},
			cubre: []CubreLine{cubreOf("tasks.md", 2, "REQ-30.1")},
		},
		{
			name:  "un motivo por hallazgo, cobertura antes que desconocidos",
			delta: d,
			cubre: []CubreLine{cubreOf("tasks.md", 2, "REQ-1.1")},
			want:  [][]string{{"REQ-29.1"}, {"REQ-29.2"}, {"REQ-1.1"}},
			codes: []Code{CodeCoverage, CodeCoverage, CodeCubreUnknown},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckCoverage(tc.delta, index, tc.cubre)
			if len(got) != len(tc.want) {
				t.Fatalf("hallazgos = %v, quería %d", reasons(got), len(tc.want))
			}
			for i, f := range got {
				if f.Code != tc.codes[i] || f.Stage != StageTasks {
					t.Errorf("hallazgo %d: %s/%s, quería %s/%s", i, f.Code, f.Stage, tc.codes[i], StageTasks)
				}
				for _, fragment := range tc.want[i] {
					if !strings.Contains(f.Reason(), fragment) {
						t.Errorf("hallazgo %d: %q no contiene %q", i, f.Reason(), fragment)
					}
				}
			}
		})
	}
}

func TestCheckExistingRequirementBug(t *testing.T) {
	index := NewBaselineIndex([]Requirement{
		req("", "sdd", "Subcomando axiom sdd status (REQ-13.1)", "S1", "S2"),
		req("", "sdd", "Requisito sin escenarios (REQ-13.2)"),
		req("", "sdd", "Requisito sin identificador"),
		req("", "a", "Duplicado uno (REQ-14.1)", "X"),
		req("", "b", "Duplicado dos (REQ-14.1)", "Y"),
	})
	cases := []struct {
		name  string
		cubre []CubreLine
		want  [][]string
	}{
		{
			name:  "cita un requisito existente con escenarios",
			cubre: []CubreLine{cubreOf("tasks.md", 2, "REQ-13.1")},
		},
		{
			name: "ningún requisito citado",
			want: [][]string{{"blocked(iso_bug_requirement)", "no cita ningún requisito"}},
		},
		{
			name:  "Cubre: sin identificadores cuenta como ninguno citado",
			cubre: []CubreLine{cubreOf("tasks.md", 2)},
			want:  [][]string{{"no cita ningún requisito"}},
		},
		{
			name:  "un identificador inexistente lo reporta CheckCoverage, no esta regla",
			cubre: []CubreLine{cubreOf("tasks.md", 2, "REQ-13.99")},
		},
		{
			name:  "el requisito citado sin escenarios remite al cambio normal",
			cubre: []CubreLine{cubreOf("tasks.md", 2, "REQ-13.2")},
			want:  [][]string{{"REQ-13.2", "ningún escenario", "cambio normal con delta"}},
		},
		{
			name:  "ambiguo en la baseline",
			cubre: []CubreLine{cubreOf("tasks.md", 2, "REQ-14.1")},
			want:  [][]string{{"REQ-14.1", "ambiguo en la baseline"}},
		},
		{
			name:  "un requisito sin identificador no puede citarse",
			cubre: []CubreLine{cubreOf("tasks.md", 2, "Requisito sin identificador")},
		},
		{
			name:  "un requisito sin escenarios citado dos veces se reporta una vez",
			cubre: []CubreLine{cubreOf("tasks.md", 2, "REQ-13.2"), cubreOf("tasks.md", 7, "REQ-13.2")},
			want:  [][]string{{"REQ-13.2"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertFindings(t, CheckExistingRequirementBug(index, tc.cubre), CodeBugRequirement, StageTasks, tc.want)
		})
	}
}

func TestCheckNoImpactBug(t *testing.T) {
	cases := []struct {
		name  string
		cubre []CubreLine
		want  [][]string
	}{
		{name: "sin Cubre: no hay contradicción"},
		{
			name:  "un Cubre: nombra el fichero y el requisito",
			cubre: []CubreLine{cubreOf("tasks.md", 4, "REQ-13.1")},
			want:  [][]string{{"tasks.md:4", "REQ-13.1", "Sin impacto en la spec", "retira el `Cubre:`", "bug con requisito existente"}},
		},
		{
			name:  "un hallazgo por identificador y fichero",
			cubre: []CubreLine{cubreOf("tasks.core.md", 4, "REQ-1.1", "REQ-1.2"), cubreOf("tasks.web.md", 2, "REQ-1.1")},
			want:  [][]string{{"tasks.core.md:4", "REQ-1.1"}, {"tasks.core.md:4", "REQ-1.2"}, {"tasks.web.md:2", "REQ-1.1"}},
		},
		{
			name:  "un Cubre: vacío también contradice la marca",
			cubre: []CubreLine{cubreOf("tasks.md", 9)},
			want:  [][]string{{"tasks.md:9"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertFindings(t, CheckNoImpactBug(tc.cubre), CodeBugContradiction, StageTasks, tc.want)
		})
	}
}

func TestAffectedScenarios(t *testing.T) {
	delta := []Requirement{
		req(SectionAdded, "iso", "Uno (REQ-1.1)", "A", "B", "A"),
		req(SectionModified, "iso", "Dos (REQ-1.2)", "C"),
		req(SectionRemoved, "iso", "Tres (REQ-1.3)", "D"),
		{Domain: "iso", Section: SectionRenamed, OldName: "x", Name: "Cuatro (REQ-1.4)"},
		req(SectionAdded, "iso", "Sin identificador", "E"),
	}
	gotDelta := AffectedFromDelta(delta)
	wantDelta := []ScenarioKey{{"REQ-1.1", "A"}, {"REQ-1.1", "B"}, {"REQ-1.2", "C"}}
	if !reflect.DeepEqual(gotDelta, wantDelta) {
		t.Errorf("AffectedFromDelta = %v, quería %v", gotDelta, wantDelta)
	}

	index := NewBaselineIndex([]Requirement{
		req("", "sdd", "Status (REQ-13.1)", "S1", "S2"),
		req("", "sdd", "Vacío (REQ-13.2)"),
	})
	gotBase := AffectedFromBaseline(index, []string{"REQ-13.1", "REQ-13.2", "REQ-13.99", "REQ-13.1"})
	wantBase := []ScenarioKey{{"REQ-13.1", "S1"}, {"REQ-13.1", "S2"}}
	if !reflect.DeepEqual(gotBase, wantBase) {
		t.Errorf("AffectedFromBaseline = %v, quería %v", gotBase, wantBase)
	}
	if got := AffectedFromBaseline(index, nil); len(got) != 0 {
		t.Errorf("sin identificadores no hay escenarios afectados, y hay %v", got)
	}
}

func TestNewBaselineIndex(t *testing.T) {
	index := NewBaselineIndex([]Requirement{
		req("", "sdd", "Con identificador (REQ-13.1)", "A"),
		req("", "sdd", "Sin identificador", "B"),
		req("", "sdd", "Mal formado (REQ-13)", "C"),
	})
	if len(index) != 1 || len(index["REQ-13.1"]) != 1 {
		t.Fatalf("el índice solo debía contener REQ-13.1: %v", index)
	}
}
