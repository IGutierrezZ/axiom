package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// wantWorkflow comprueba el carril resuelto de una configuración cargada.
func wantWorkflow(t *testing.T, cfg *WorkspaceConfig, lane WorkflowLane) {
	t.Helper()
	if cfg == nil || cfg.Workflow == nil {
		t.Fatalf("la configuración debe llevar el carril resuelto: %+v", cfg)
	}
	if cfg.Workflow.Lane != lane {
		t.Errorf("workflow.lane = %q, se esperaba %q", cfg.Workflow.Lane, lane)
	}
}

// REQ-27.1: odd, sdd y ask son los únicos valores válidos y se proyectan tal cual.
func TestWorkflowValidLanesAreDeclared(t *testing.T) {
	for _, lane := range []WorkflowLane{WorkflowLaneODD, WorkflowLaneSDD, WorkflowLaneAsk} {
		t.Run(string(lane), func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, "workflow:\n  lane: "+string(lane)+"\n")
			if baseErr != nil || declarations.WorkflowErr != nil {
				t.Fatalf("base=%v workflow=%v", baseErr, declarations.WorkflowErr)
			}
			if declarations.Workflow == nil || declarations.Workflow.Lane != lane {
				t.Fatalf("lo declarado se conserva: %+v", declarations.Workflow)
			}
			wantWorkflow(t, cfg, lane)
		})
	}
}

// REQ-27.1: sin bloque, o con el bloque sin `lane`, el carril efectivo es sdd.
func TestWorkflowAbsentBlockOrLaneResolvesToSDD(t *testing.T) {
	if got := resolveWorkflow(nil); got == nil || got.Lane != WorkflowLaneSDD {
		t.Fatalf("resolveWorkflow(nil) = %+v, se esperaba sdd", got)
	}
	if got := resolveWorkflow(&WorkflowConfig{}); got.Lane != WorkflowLaneSDD {
		t.Fatalf("resolveWorkflow(sin lane) = %+v, se esperaba sdd", got)
	}
	declared := &WorkflowConfig{Lane: WorkflowLaneODD}
	if got := resolveWorkflow(declared); got.Lane != WorkflowLaneODD || got == declared {
		t.Fatalf("resolveWorkflow no modifica ni reutiliza lo declarado: %+v", got)
	}

	cfg, baseErr, declarations := parseLayered([]byte(declarationsBase))
	if baseErr != nil || declarations.Workflow != nil || declarations.WorkflowErr != nil {
		t.Fatalf("sin bloque no hay declaración ni error: base=%v %+v", baseErr, declarations)
	}
	wantWorkflow(t, cfg, WorkflowLaneSDD)

	viaParse, err := ParseConfig([]byte(declarationsBase))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	wantWorkflow(t, viaParse, WorkflowLaneSDD)
}

// Un bloque `workflow: {}` es un mapeo sin `lane`: es válido y resuelve a sdd.
func TestWorkflowBlockWithoutLaneIsValid(t *testing.T) {
	cfg, baseErr, declarations := parseDeclarations(t, "workflow: {}\n")
	if baseErr != nil || declarations.WorkflowErr != nil {
		t.Fatalf("base=%v workflow=%v", baseErr, declarations.WorkflowErr)
	}
	if declarations.Workflow == nil || declarations.Workflow.Lane != "" {
		t.Fatalf("el bloque sin lane conserva Lane vacío: %+v", declarations.Workflow)
	}
	wantWorkflow(t, cfg, WorkflowLaneSDD)
}

// REQ-27.1: valores inválidos (mayúsculas, sinónimos, nulo, lista, mapeo) se
// rechazan con un error que nombra `workflow.lane`.
func TestWorkflowInvalidValuesAreLoadErrorsNamingTheKey(t *testing.T) {
	tests := []struct {
		name  string
		block string
		wants []string
	}{
		{"mayúsculas", "workflow:\n  lane: SDD\n", []string{"workflow.lane", "'SDD'", "odd, sdd o ask"}},
		{"sinónimo", "workflow:\n  lane: agil\n", []string{"workflow.lane", "'agil'", "odd, sdd o ask"}},
		{"vacío", "workflow:\n  lane: \"\"\n", []string{"workflow.lane", "odd, sdd o ask"}},
		{"nulo", "workflow:\n  lane:\n", []string{"workflow.lane", "nulo"}},
		{"lista", "workflow:\n  lane: [odd]\n", []string{"workflow.lane", "lista"}},
		{"mapeo", "workflow:\n  lane: {a: b}\n", []string{"workflow.lane", "mapeo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, tt.block)
			if baseErr != nil {
				t.Fatalf("la base es válida: %v", baseErr)
			}
			wantErrorContains(t, declarations.WorkflowErr, tt.wants...)
			if declarations.Workflow != nil || cfg.Workflow != nil {
				t.Errorf("un bloque inválido no se proyecta: %+v / %+v", declarations.Workflow, cfg.Workflow)
			}

			_, err := ParseConfig([]byte(declarationsBase + tt.block))
			wantErrorContains(t, err, tt.wants...)
		})
	}
}

func TestWorkflowUnknownAndDuplicatedKeysAreRejected(t *testing.T) {
	_, _, declarations := parseDeclarations(t, "workflow:\n  lane: odd\n  mode: fast\n")
	wantErrorContains(t, declarations.WorkflowErr, "mode", "workflow", "lane")

	_, _, declarations = parseDeclarations(t, "workflow:\n  lane: odd\n  lane: sdd\n")
	wantErrorContains(t, declarations.WorkflowErr, "lane", "duplicada")
}

func TestWorkflowBlockMustBeAMapping(t *testing.T) {
	for name, block := range map[string]string{
		"escalar": "workflow: sdd\n",
		"nulo":    "workflow:\n",
		"lista":   "workflow: [sdd]\n",
	} {
		t.Run(name, func(t *testing.T) {
			cfg, baseErr, declarations := parseDeclarations(t, block)
			if baseErr != nil {
				t.Fatalf("la base es válida: %v", baseErr)
			}
			wantErrorContains(t, declarations.WorkflowErr, "'workflow'", "mapeo", "lane")
			if declarations.Workflow != nil || cfg.Workflow != nil {
				t.Errorf("un bloque que no es un mapeo no se proyecta: %+v / %+v", declarations.Workflow, cfg.Workflow)
			}
		})
	}
}

// Un `workflow:` inválido no oculta los errores de otros bloques ni de la base.
func TestWorkflowErrorsShareTheLoadWithTheOtherBlocks(t *testing.T) {
	_, err := ParseConfig([]byte(declarationsBase + "workflow:\n  lane: nope\ngit:\n  branch_prefix: \"inc//\"\n"))
	wantErrorContains(t, err, "workflow.lane", "git.branch_prefix")

	cfg, baseErr, declarations := parseDeclarations(t, "workflow:\n  lane: nope\ngit:\n  branch_prefix: inc\n")
	if baseErr != nil || declarations.GitErr != nil || declarations.WorkflowErr == nil {
		t.Fatalf("base=%v git=%v workflow=%v", baseErr, declarations.GitErr, declarations.WorkflowErr)
	}
	wantGit(t, cfg, "main", "inc")
}

// REQ-27.1 y REQ-24.17: un binario anterior que no conoce `workflow:` carga el
// mismo axiom.yaml ignorando la clave, aunque el bloque sea inválido.
func TestWorkflowBlockIsToleratedByThePreviousBinary(t *testing.T) {
	for name, block := range map[string]string{
		"válido":             "workflow:\n  lane: ask\n",
		"con valor inválido": "workflow:\n  lane: SDD\n",
		"nulo":               "workflow:\n",
	} {
		t.Run(name, func(t *testing.T) {
			wantPreviousBinaryToleratesTheBlock(t, block)
		})
	}

	var current WorkspaceConfig
	if err := yaml.Unmarshal([]byte(declarationsBase+"workflow:\n  lane: odd\n"), &current); err != nil {
		t.Fatalf("decodificación base: %v", err)
	}
	if current.Workflow != nil {
		t.Errorf("la decodificación base no debe rellenar Workflow: %+v", current.Workflow)
	}
}

// S8 y REQ-27.1: el axiom.yaml de este repositorio declara el carril `ask` y la
// plantilla carga sin error con su bloque comentado.
func TestRepositoryAxiomYAMLDeclaresTheAskLane(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "axiom.yaml"))
	if err != nil {
		t.Fatalf("no se pudo leer axiom.yaml: %v", err)
	}
	cfg, baseErr, declarations := parseLayered(data)
	if baseErr != nil || declarations.WorkflowErr != nil {
		t.Fatalf("axiom.yaml debe cargar sin errores: base=%v workflow=%v", baseErr, declarations.WorkflowErr)
	}
	if declarations.Workflow == nil || declarations.Workflow.Lane != WorkflowLaneAsk {
		t.Errorf("axiom.yaml declara workflow.lane: ask: %+v", declarations.Workflow)
	}
	wantWorkflow(t, cfg, WorkflowLaneAsk)

	example, err := os.ReadFile(filepath.Join("..", "..", "axiom.example.yaml"))
	if err != nil {
		t.Fatalf("no se pudo leer axiom.example.yaml: %v", err)
	}
	exCfg, exErr, exDecl := parseLayered(example)
	if exErr != nil || exDecl.WorkflowErr != nil {
		t.Fatalf("la plantilla debe cargar sin errores: base=%v workflow=%v", exErr, exDecl.WorkflowErr)
	}
	wantWorkflow(t, exCfg, WorkflowLaneSDD)
}

// REQ-27.1: `Locate` nunca se rechaza por un `workflow:` inválido; el motivo queda
// en Declarations.WorkflowErr y Config no lo proyecta.
func TestLocateIsNeverRejectedByAnInvalidWorkflowBlock(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), locateEmbeddedConfig+"workflow:\n  lane: nope\n")
	loc := mustLocate(t, ws.Master)
	if loc.Config == nil || loc.ConfigErr != nil {
		t.Fatalf("la base válida carga y el bloque inválido queda en su canal: %+v", loc)
	}
	if loc.Declarations.WorkflowErr == nil || loc.Declarations.Workflow != nil || loc.Config.Workflow != nil {
		t.Fatalf("el error va a Declarations.WorkflowErr y no se proyecta: %+v", loc.Declarations)
	}
}
