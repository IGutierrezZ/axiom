package workspace

import (
	"errors"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
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

// REQ-29.1 y REQ-29.2: carga por capas (parseLayered, ParseConfig, ReadDeclarations
// y Locate) de un bloque `sdd:` válido, inválido y contradictorio.
func TestSDDLoad(t *testing.T) {
	t.Run("sin bloque nada cambia", func(t *testing.T) {
		cfg, baseErr, declarations := parseLayered([]byte(declarationsBase))
		if baseErr != nil || declarations.SDD != nil || declarations.SDDErr != nil {
			t.Fatalf("sin bloque no hay declaración ni error: base=%v %+v", baseErr, declarations)
		}
		wantWorkflow(t, cfg, WorkflowLaneSDD)
		if _, err := ParseConfig([]byte(declarationsBase)); err != nil {
			t.Fatalf("ParseConfig sin bloque: %v", err)
		}
	})

	for name, blocks := range map[string]string{
		"iso true solo":            "sdd:\n  iso: true\n",
		"iso true con lane sdd":    "sdd:\n  iso: true\nworkflow:\n  lane: sdd\n",
		"iso true con lane ask":    "sdd:\n  iso: true\nworkflow:\n  lane: ask\n",
		"iso false con lane odd":   "sdd:\n  iso: false\nworkflow:\n  lane: odd\n",
		"iso true con workflow {}": "sdd:\n  iso: true\nworkflow: {}\n",
	} {
		t.Run("valido/"+name, func(t *testing.T) {
			_, baseErr, declarations := parseDeclarations(t, blocks)
			if baseErr != nil || declarations.SDDErr != nil || declarations.WorkflowErr != nil || declarations.SDD == nil {
				t.Fatalf("base=%v sdd=%v workflow=%v declarado=%+v", baseErr, declarations.SDDErr, declarations.WorkflowErr, declarations.SDD)
			}
			if _, err := ParseConfig([]byte(declarationsBase + blocks)); err != nil {
				t.Fatalf("ParseConfig debe aceptarlo: %v", err)
			}
		})
	}

	t.Run("contradiccion", func(t *testing.T) {
		blocks := "sdd:\n  iso: true\nworkflow:\n  lane: odd\n"
		cfg, baseErr, declarations := parseDeclarations(t, blocks)
		if baseErr != nil {
			t.Fatalf("la base es válida: %v", baseErr)
		}
		if declarations.SDD != nil || !errors.Is(declarations.SDDErr, ErrISOWorkflowContradiction) {
			t.Fatalf("la contradicción va a SDDErr y deja SDD en nil: %+v", declarations)
		}
		if declarations.WorkflowErr != nil || declarations.Workflow == nil || declarations.Workflow.Lane != WorkflowLaneODD {
			t.Errorf("workflow se decodifica igual que siempre: %+v err=%v", declarations.Workflow, declarations.WorkflowErr)
		}
		wantWorkflow(t, cfg, WorkflowLaneODD)
		_, err := ParseConfig([]byte(declarationsBase + blocks))
		wantErrorContains(t, err, "sdd.iso", "workflow.lane")
		if !errors.Is(err, ErrISOWorkflowContradiction) {
			t.Errorf("ParseConfig debe conservar el centinela: %v", err)
		}
	})

	t.Run("sdd invalido no cambia workflow", func(t *testing.T) {
		blocks := "sdd:\n  iso: quizás\nworkflow:\n  lane: odd\n"
		cfg, baseErr, declarations := parseDeclarations(t, blocks)
		if baseErr != nil || declarations.SDD != nil || declarations.SDDErr == nil || errors.Is(declarations.SDDErr, ErrISOWorkflowContradiction) {
			t.Fatalf("base=%v %+v", baseErr, declarations)
		}
		wantWorkflow(t, cfg, WorkflowLaneODD)
		_, err := ParseConfig([]byte(declarationsBase + blocks))
		wantErrorContains(t, err, "sdd.iso", "quizás")
	})

	t.Run("errores de sdd y de otro bloque se acumulan", func(t *testing.T) {
		_, err := ParseConfig([]byte(declarationsBase + "sdd:\n  iso: nope\nworkflow:\n  lane: nope\n"))
		wantErrorContains(t, err, "sdd.iso", "lane")
	})

	t.Run("ReadDeclarations y Locate", func(t *testing.T) {
		ws := workspacetest.Embedded(t, ".")
		configPath := filepath.Join(ws.Master, "axiom.yaml")
		workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+"sdd:\n  iso: true\nworkflow:\n  lane: odd\n")
		got := ReadDeclarations(configPath)
		if got.SDD != nil || !errors.Is(got.SDDErr, ErrISOWorkflowContradiction) {
			t.Fatalf("ReadDeclarations: %+v", got)
		}
		loc := mustLocate(t, ws.Master)
		if loc.Config == nil || loc.ConfigErr != nil || !errors.Is(loc.Declarations.SDDErr, ErrISOWorkflowContradiction) {
			t.Fatalf("Locate no se rechaza por la contradicción; queda en SDDErr: %+v", loc)
		}
	})
}

// REQ-29.1: workspace validate carga con LoadConfig y falla (salida 1) con un
// `sdd:` inválido o contradictorio; con un bloque válido carga sin error.
func TestWorkspaceValidateSDD(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	configPath := filepath.Join(ws.Master, "axiom.yaml")

	workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+"sdd:\n  iso: true\n")
	if _, err := LoadConfig(configPath); err != nil {
		t.Fatalf("un sdd válido debe cargar: %v", err)
	}
	for name, blocks := range map[string]string{
		"contradictorio": "sdd:\n  iso: true\nworkflow:\n  lane: odd\n",
		"inválido":       "sdd:\n  archive: true\n",
	} {
		workspacetest.WriteFile(t, configPath, locateEmbeddedConfig+blocks)
		if _, err := LoadConfig(configPath); err == nil {
			t.Errorf("%s: LoadConfig debe fallar", name)
		}
	}
}
