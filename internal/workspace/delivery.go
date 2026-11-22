package workspace

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// DeliveryConfig es el bloque `delivery:` de axiom.yaml.
// SplitHintLines es el umbral del aviso no bloqueante de cambio grande;
// 0 significa que la clave no se declaró y no hay aviso.
type DeliveryConfig struct {
	SplitHintLines int
}

// deliveryKeys es la lista cerrada de claves del bloque `delivery:`.
var deliveryKeys = []string{"split_hint_lines"}

// decodeDelivery decodifica y valida el bloque `delivery:`. Rechaza un nodo que
// no es un mapeo (incluido el nulo), las claves fuera de deliveryKeys o
// repetidas y un `split_hint_lines` que no sea un entero sin comillas mayor que
// cero. Acumula todos los errores.
func decodeDelivery(node *yaml.Node) (*DeliveryConfig, error) {
	node = resolveAlias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, errors.New("el bloque 'delivery' debe ser un mapeo con la clave opcional 'split_hint_lines' (entero mayor que cero); para no recibir el aviso, elimina el bloque")
	}

	errs := unknownKeys(node, "delivery", deliveryKeys)
	config := &DeliveryConfig{}
	seen := map[string]bool{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if !containsString(deliveryKeys, key.Value) {
			continue // ya reportada por unknownKeys
		}
		if seen[key.Value] {
			errs = append(errs, fmt.Errorf("clave duplicada '%s' en 'delivery' (línea %d)", key.Value, key.Line))
			continue
		}
		seen[key.Value] = true

		value, err := deliveryPositiveInt("delivery."+key.Value, node.Content[i+1])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		config.SplitHintLines = value
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return config, nil
}

// deliveryPositiveInt lee un entero estricto mayor que cero: solo el escalar con
// etiqueta !!int, de modo que `"600"` entre comillas, `6e2`, `1.5`, `true`, el
// nulo, las listas y los mapeos se rechazan, igual que un desbordamiento de int.
func deliveryPositiveInt(label string, node *yaml.Node) (int, error) {
	node = resolveAlias(node)
	switch {
	case isNull(node):
		return 0, fmt.Errorf("%s no puede ser un valor nulo: debe ser un entero mayor que cero", label)
	case node.Kind == yaml.SequenceNode:
		return 0, fmt.Errorf("%s debe ser un entero mayor que cero, no una lista", label)
	case node.Kind != yaml.ScalarNode:
		return 0, fmt.Errorf("%s debe ser un entero mayor que cero, no un mapeo", label)
	case node.Tag != "!!int":
		return 0, fmt.Errorf("%s: el valor '%s' no es válido: usa un entero mayor que cero, sin comillas", label, node.Value)
	}
	var value int
	if err := node.Decode(&value); err != nil {
		return 0, fmt.Errorf("%s: el valor '%s' no es válido: usa un entero mayor que cero", label, strings.TrimSpace(node.Value))
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s: el valor '%d' no es válido: debe ser mayor que cero", label, value)
	}
	return value, nil
}
