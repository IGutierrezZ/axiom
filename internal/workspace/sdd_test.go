package workspace

import (
	"errors"
	"testing"

	"gopkg.in/yaml.v3"
)

func sddNode(t *testing.T, source string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(source), &doc); err != nil {
		t.Fatalf("YAML de prueba inválido: %v", err)
	}
	node, ok := mappingValue(doc.Content[0], "sdd")
	if !ok {
		t.Fatalf("el YAML de prueba no tiene bloque sdd")
	}
	return node
}

// REQ-29.1: el booleano es estricto y la lista de claves, cerrada.
func TestDecodeSDD(t *testing.T) {
	valid := []struct {
		name   string
		source string
		want   bool
	}{
		{"iso true", "sdd:\n  iso: true\n", true},
		{"iso false", "sdd:\n  iso: false\n", false},
		{"mapeo vacío", "sdd: {}\n", false},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeSDD(sddNode(t, tc.source))
			if err != nil || got == nil || got.ISO != tc.want {
				t.Fatalf("decodeSDD = %+v, %v; se esperaba iso=%v", got, err, tc.want)
			}
		})
	}

	invalid := []struct {
		name   string
		source string
		wants  []string
	}{
		{"valor no booleano", "sdd:\n  iso: quizás\n", []string{"sdd.iso", "quizás", "true o false"}},
		{"booleano entre comillas", "sdd:\n  iso: \"true\"\n", []string{"sdd.iso", "sin comillas"}},
		{"nulo", "sdd:\n  iso:\n", []string{"sdd.iso", "nulo"}},
		{"lista", "sdd:\n  iso: [true]\n", []string{"sdd.iso", "lista"}},
		{"mapeo como valor", "sdd:\n  iso: {a: b}\n", []string{"sdd.iso", "mapeo"}},
		{"clave desconocida archive", "sdd:\n  archive: true\n", []string{"archive", "claves admitidas: iso"}},
		{"clave desconocida design_triggers", "sdd:\n  design_triggers: true\n", []string{"design_triggers"}},
		{"clave duplicada", "sdd:\n  iso: true\n  iso: false\n", []string{"clave duplicada 'iso'"}},
		{"bloque nulo", "sdd:\n", []string{"bloque 'sdd'", "mapeo"}},
		{"bloque lista", "sdd:\n  - iso\n", []string{"bloque 'sdd'", "mapeo"}},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeSDD(sddNode(t, tc.source))
			if got != nil {
				t.Errorf("un bloque inválido no devuelve configuración: %+v", got)
			}
			wantErrorContains(t, err, tc.wants...)
		})
	}
}

// REQ-29.1: solo iso true con lane odd es contradictorio.
func TestCheckISOWorkflow(t *testing.T) {
	lanes := []*WorkflowConfig{nil, {}, {Lane: WorkflowLaneODD}, {Lane: WorkflowLaneSDD}, {Lane: WorkflowLaneAsk}}
	isos := []*SDDConfig{nil, {ISO: false}, {ISO: true}}
	for _, sdd := range isos {
		for _, workflow := range lanes {
			err := checkISOWorkflow(sdd, workflow)
			contradiction := sdd != nil && sdd.ISO && workflow != nil && workflow.Lane == WorkflowLaneODD
			if contradiction != (err != nil) {
				t.Errorf("sdd=%+v workflow=%+v: error=%v, contradicción esperada=%v", sdd, workflow, err, contradiction)
				continue
			}
			if err == nil {
				continue
			}
			if !errors.Is(err, ErrISOWorkflowContradiction) {
				t.Errorf("el error debe envolver ErrISOWorkflowContradiction: %v", err)
			}
			wantErrorContains(t, err, "sdd.iso", "workflow.lane", "retira workflow.lane: odd", "retira sdd.iso")
		}
	}
}
