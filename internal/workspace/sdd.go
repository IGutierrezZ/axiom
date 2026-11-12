package workspace

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SDDConfig es el bloque `sdd:` de axiom.yaml (REQ-29.1).
type SDDConfig struct {
	ISO bool
}

// sddKeys es la lista cerrada de claves del bloque `sdd:`.
var sddKeys = []string{"iso"}

// ErrISOWorkflowContradiction identifica la declaración contradictoria de
// REQ-29.1: `sdd.iso: true` junto a `workflow.lane: odd`. errors.Is lo reconoce
// dentro de Declarations.SDDErr.
var ErrISOWorkflowContradiction = errors.New("sdd.iso: true es incompatible con workflow.lane: odd")

// decodeSDD decodifica y valida el bloque `sdd:`. Rechaza un nodo que no es un
// mapeo (incluido el nulo), las claves fuera de sddKeys o repetidas y un `iso`
// que no sea un escalar booleano (`true` o `false` sin comillas).
func decodeSDD(node *yaml.Node) (*SDDConfig, error) {
	node = resolveAlias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, errors.New("el bloque 'sdd' debe ser un mapeo con la clave opcional 'iso' (true o false); para no declararlo, elimina el bloque")
	}

	errs := unknownKeys(node, "sdd", sddKeys)
	config := &SDDConfig{}
	seen := map[string]bool{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if !containsString(sddKeys, key.Value) {
			continue // ya reportada por unknownKeys
		}
		if seen[key.Value] {
			errs = append(errs, fmt.Errorf("clave duplicada '%s' en 'sdd' (línea %d)", key.Value, key.Line))
			continue
		}
		seen[key.Value] = true

		value, err := sddBool("sdd."+key.Value, node.Content[i+1])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		config.ISO = value
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return config, nil
}

// sddBool lee un booleano estricto: solo el escalar con etiqueta !!bool, de modo
// que `"true"` entre comillas, el nulo, las listas y los mapeos se rechazan.
func sddBool(label string, node *yaml.Node) (bool, error) {
	node = resolveAlias(node)
	switch {
	case isNull(node):
		return false, fmt.Errorf("%s no puede ser un valor nulo: debe ser true o false", label)
	case node.Kind == yaml.SequenceNode:
		return false, fmt.Errorf("%s debe ser true o false, no una lista", label)
	case node.Kind != yaml.ScalarNode:
		return false, fmt.Errorf("%s debe ser true o false, no un mapeo", label)
	case node.Tag != "!!bool":
		return false, fmt.Errorf("%s: el valor '%s' no es válido: usa true o false, sin comillas", label, node.Value)
	}
	var value bool
	if err := node.Decode(&value); err != nil {
		return false, fmt.Errorf("%s: el valor '%s' no es válido: usa true o false", label, strings.TrimSpace(node.Value))
	}
	return value, nil
}

// checkISOWorkflow devuelve un error que envuelve ErrISOWorkflowContradiction
// solo con `sdd.iso` true y `workflow.lane` odd; con cualquier otra combinación,
// también sin alguno de los dos bloques, devuelve nil.
func checkISOWorkflow(sdd *SDDConfig, workflow *WorkflowConfig) error {
	if sdd == nil || !sdd.ISO || workflow == nil || workflow.Lane != WorkflowLaneODD {
		return nil
	}
	return fmt.Errorf("declaración contradictoria: sdd.iso es true y workflow.lane es odd, pero un proyecto ISO tramita todo el trabajo por el carril SDD; retira workflow.lane: odd (o fíjalo en sdd), o retira sdd.iso (o ponlo en false): %w", ErrISOWorkflowContradiction)
}
