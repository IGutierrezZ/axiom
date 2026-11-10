package workspace

import (
	"errors"

	"gopkg.in/yaml.v3"
)

// WorkflowLane es el carril de trabajo que el proyecto declara en `workflow.lane`
// de axiom.yaml (REQ-27.1).
type WorkflowLane string

const (
	// WorkflowLaneODD fija el carril ágil: el trabajo sustancial entra por ODD sin
	// preguntar, salvo que la persona pida SDD de forma explícita.
	WorkflowLaneODD WorkflowLane = "odd"
	// WorkflowLaneSDD fija el carril formal: el trabajo sustancial entra por SDD
	// sin preguntar el carril.
	WorkflowLaneSDD WorkflowLane = "sdd"
	// WorkflowLaneAsk conserva la pregunta bloqueante de carril.
	WorkflowLaneAsk WorkflowLane = "ask"

	// DefaultWorkflowLane es el carril cuando axiom.yaml no declara `workflow.lane`
	// (REQ-27.1).
	DefaultWorkflowLane = WorkflowLaneSDD
)

// WorkflowConfig es el bloque `workflow:` de axiom.yaml (REQ-27.1).
//
// En Declarations.Workflow conserva lo escrito (Lane vacío si no se declaró la
// clave). En WorkspaceConfig.Workflow, Lane está siempre relleno.
type WorkflowConfig struct {
	Lane WorkflowLane `yaml:"lane,omitempty"`
}

// workflowKeys es la lista cerrada de claves del bloque `workflow:`.
var workflowKeys = []string{"lane"}

// workflowValidator devuelve el validador de la clave del bloque `workflow:`.
func workflowValidator(string) func(string) error {
	return func(raw string) error {
		switch WorkflowLane(raw) {
		case WorkflowLaneODD, WorkflowLaneSDD, WorkflowLaneAsk:
			return nil
		}
		return errors.New("usa odd, sdd o ask")
	}
}

// decodeWorkflow decodifica y valida el bloque `workflow:` con el mismo patrón
// que decodeGit: recorre el yaml.Node contra la lista cerrada de claves y acumula
// claves desconocidas o repetidas y valores que no son odd, sdd o ask (distingue
// mayúsculas). Un bloque que no es un mapeo, incluido el nulo (`workflow:` sin
// contenido), se rechaza: para usar el valor por defecto se elimina el bloque.
func decodeWorkflow(node *yaml.Node) (*WorkflowConfig, error) {
	node = resolveAlias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, errors.New("el bloque 'workflow' debe ser un mapeo con la clave opcional 'lane' (odd, sdd o ask); para usar el valor por defecto, elimina el bloque")
	}

	values, errs := decodeStringKeys(node, "workflow", workflowKeys, workflowValidator)
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &WorkflowConfig{Lane: WorkflowLane(values["lane"])}, nil
}

// resolveWorkflow devuelve el bloque efectivo a partir de lo declarado, que puede
// ser nil (sin bloque): sin `lane` toma DefaultWorkflowLane. Nunca devuelve nil ni
// modifica declared.
func resolveWorkflow(declared *WorkflowConfig) *WorkflowConfig {
	resolved := WorkflowConfig{Lane: DefaultWorkflowLane}
	if declared != nil && declared.Lane != "" {
		resolved.Lane = declared.Lane
	}
	return &resolved
}
