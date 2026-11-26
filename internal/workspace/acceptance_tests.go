package workspace

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// AcceptanceTestsConfig es el bloque `acceptance_tests:` de axiom.yaml (RF-TOP-19).
// Enabled activa la batería de pruebas de aceptación (`pruebas.md`) y
// Environments son los entornos declarados contra los que se prueba. El valor
// cero equivale a «función no activa».
type AcceptanceTestsConfig struct {
	Enabled      bool
	Environments []string
}

// acceptanceTestsKeys es la lista cerrada de claves del bloque `acceptance_tests:`.
var acceptanceTestsKeys = []string{"enabled", "environments"}

// decodeAcceptanceTests decodifica y valida el bloque `acceptance_tests:`.
// Rechaza un nodo nulo o que no es un mapeo (error tipado, sin asumir valores),
// las claves fuera de acceptanceTestsKeys o repetidas, un `enabled` que no sea el
// escalar booleano estricto y un `environments` que no sea una lista de cadenas
// no vacías y sin `;`. Con `enabled: true` la lista debe existir, no estar vacía
// y no repetir entornos; con `enabled: false` no se exige. Acumula todos los
// errores, con su línea.
func decodeAcceptanceTests(node *yaml.Node) (*AcceptanceTestsConfig, error) {
	node = resolveAlias(node)
	if isNull(node) {
		return nil, errors.New("el bloque 'acceptance_tests' no puede ser nulo: declara 'enabled' (true o false) y 'environments'; para no usar la función, elimina el bloque")
	}
	if node.Kind != yaml.MappingNode {
		return nil, errors.New("el bloque 'acceptance_tests' debe ser un mapeo con las claves 'enabled' (booleano) y 'environments' (lista de cadenas)")
	}

	errs := unknownKeys(node, "acceptance_tests", acceptanceTestsKeys)
	config := &AcceptanceTestsConfig{}
	seen := map[string]bool{}
	var (
		enabledSeen bool
		envsNode    *yaml.Node
		envsLine    int
		envs        []string
		envsValid   = true
	)
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if !containsString(acceptanceTestsKeys, key.Value) {
			continue // ya reportada por unknownKeys
		}
		if seen[key.Value] {
			errs = append(errs, fmt.Errorf("clave duplicada '%s' en 'acceptance_tests' (línea %d)", key.Value, key.Line))
			continue
		}
		seen[key.Value] = true

		value := node.Content[i+1]
		switch key.Value {
		case "enabled":
			enabled, err := acceptanceTestsBool(value)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			config.Enabled = enabled
			enabledSeen = true
		case "environments":
			envsNode, envsLine = value, key.Line
			var envErrs []error
			envs, envErrs = acceptanceTestsEnvironments(value)
			if len(envErrs) > 0 {
				envsValid = false
				errs = append(errs, envErrs...)
			}
		}
	}

	if !seen["enabled"] {
		errs = append(errs, errors.New("el bloque 'acceptance_tests' requiere la clave 'enabled' (true o false): no se asume ningún valor"))
	}
	if enabledSeen && config.Enabled && envsValid {
		switch {
		case envsNode == nil:
			errs = append(errs, errors.New("'acceptance_tests.environments' es obligatorio con 'enabled: true': declara al menos un entorno"))
		case len(envs) == 0:
			errs = append(errs, fmt.Errorf("'acceptance_tests.environments' (línea %d) debe ser no vacío con 'enabled: true': declara al menos un entorno", envsLine))
		default:
			seenEnv := map[string]bool{}
			for _, env := range envs {
				if seenEnv[env] {
					errs = append(errs, fmt.Errorf("el entorno '%s' está duplicado en 'acceptance_tests.environments' (línea %d)", env, envsLine))
				}
				seenEnv[env] = true
			}
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	config.Environments = envs
	return config, nil
}

// acceptanceTestsBool lee un booleano estricto: solo el escalar con etiqueta
// !!bool, de modo que `"sí"`, `"true"` entre comillas, el nulo, las listas y los
// mapeos se rechazan.
func acceptanceTestsBool(node *yaml.Node) (bool, error) {
	node = resolveAlias(node)
	const label = "acceptance_tests.enabled"
	switch {
	case isNull(node):
		return false, fmt.Errorf("%s no puede ser un valor nulo: debe ser true o false", label)
	case node.Kind == yaml.SequenceNode:
		return false, fmt.Errorf("%s debe ser true o false, no una lista (línea %d)", label, node.Line)
	case node.Kind != yaml.ScalarNode:
		return false, fmt.Errorf("%s debe ser true o false, no un mapeo (línea %d)", label, node.Line)
	case node.Tag != "!!bool":
		return false, fmt.Errorf("%s: el valor '%s' no es válido (línea %d): usa true o false, sin comillas", label, node.Value, node.Line)
	}
	var value bool
	if err := node.Decode(&value); err != nil {
		return false, fmt.Errorf("%s: el valor '%s' no es válido (línea %d): usa true o false", label, strings.TrimSpace(node.Value), node.Line)
	}
	return value, nil
}

// acceptanceTestsEnvironments lee la lista de entornos: cadenas no vacías y sin
// `;`, que separa los entornos de cada caso en `pruebas.md`. No comprueba los
// duplicados ni la obligatoriedad: dependen de `enabled`.
func acceptanceTestsEnvironments(node *yaml.Node) ([]string, []error) {
	node = resolveAlias(node)
	const label = "acceptance_tests.environments"
	switch {
	case isNull(node):
		return nil, []error{fmt.Errorf("%s no puede ser un valor nulo: debe ser una lista de cadenas", label)}
	case node.Kind != yaml.SequenceNode:
		return nil, []error{fmt.Errorf("%s debe ser una lista de cadenas (línea %d)", label, node.Line)}
	}
	var (
		errs []error
		envs []string
	)
	for _, item := range node.Content {
		item = resolveAlias(item)
		switch {
		case item.Kind != yaml.ScalarNode || item.Tag != "!!str":
			errs = append(errs, fmt.Errorf("%s: cada entorno debe ser una cadena no vacía (línea %d)", label, item.Line))
		case strings.TrimSpace(item.Value) == "":
			errs = append(errs, fmt.Errorf("%s: un entorno no puede estar vacío (línea %d)", label, item.Line))
		case strings.Contains(item.Value, ";"):
			errs = append(errs, fmt.Errorf("%s: el entorno '%s' no puede contener ';', que separa los entornos de cada caso en pruebas.md (línea %d)", label, item.Value, item.Line))
		default:
			envs = append(envs, item.Value)
		}
	}
	return envs, errs
}

// acceptanceTestsTopologyError aplica la regla de topología de RF-TOP-19: la
// batería de pruebas solo se activa en multirepo. Con la función no activa (o sin
// bloque) no hay nada que rechazar. La usan workspace.Validate y el accesor
// EffectiveAcceptanceTests, para que una declaración que Validate rechaza nunca
// active la doctrina.
func acceptanceTestsTopologyError(cfg *AcceptanceTestsConfig, topology TopologyType) error {
	if cfg == nil || !cfg.Enabled || topology == TopologyMultirepo {
		return nil
	}
	return fmt.Errorf("'acceptance_tests.enabled: true' solo se admite en la topología 'multirepo', pero axiom.yaml declara '%s'. Para migrar, pasa a 'workspace.topology: multirepo' con un 'specs_repository' propio o declara 'acceptance_tests: { enabled: false }'", topology)
}

// EffectiveAcceptanceTests devuelve la configuración efectiva de
// `acceptance_tests` para la topología dada: el valor cero (función no activa)
// sin bloque. Con un `axiom.yaml` ilegible, un bloque inválido o la regla de
// topología violada devuelve un error que envuelve la causa (%w) y nunca asume un
// valor (RF-TOP-19).
func (d Declarations) EffectiveAcceptanceTests(topology TopologyType) (AcceptanceTestsConfig, error) {
	if d.ParseErr != nil {
		return AcceptanceTestsConfig{}, fmt.Errorf("no se puede leer axiom.yaml para conocer acceptance_tests: %w", d.ParseErr)
	}
	if d.AcceptanceTestsErr != nil {
		return AcceptanceTestsConfig{}, fmt.Errorf("el bloque acceptance_tests de axiom.yaml no es válido y no se puede conocer: %w", d.AcceptanceTestsErr)
	}
	if d.AcceptanceTests == nil {
		return AcceptanceTestsConfig{}, nil
	}
	if err := acceptanceTestsTopologyError(d.AcceptanceTests, topology); err != nil {
		return AcceptanceTestsConfig{}, fmt.Errorf("la declaración acceptance_tests de axiom.yaml no es válida para esta topología: %w", err)
	}
	cfg := *d.AcceptanceTests
	cfg.Environments = append([]string(nil), cfg.Environments...)
	return cfg, nil
}
