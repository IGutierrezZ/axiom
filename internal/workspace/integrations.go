package workspace

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// IntegrationsConfig es el bloque `integrations:` de axiom.yaml: un mapeo de plugins del catálogo
// cerrado de manifiestos (internal/integrations/manifest) a sus valores de configuración.
type IntegrationsConfig struct {
	Plugins []IntegrationPluginConfig // en orden de declaración
}

// IntegrationPluginConfig es la configuración declarada de un plugin.
type IntegrationPluginConfig struct {
	ID      string
	Enabled bool
	// Values guarda cada clave declarada por el manifiesto con su tipo (string, bool, int o
	// []string) y los Default aplicados a las claves ausentes. Nunca contiene `enabled`.
	Values map[string]any
	Line   int
}

// Active devuelve los plugins con enabled distinto de false, en orden de declaración.
func (c *IntegrationsConfig) Active() []IntegrationPluginConfig {
	if c == nil {
		return nil
	}
	var active []IntegrationPluginConfig
	for _, p := range c.Plugins {
		if p.Enabled {
			active = append(active, p)
		}
	}
	return active
}

// integrationsEnabledKey es la clave reservada que activa o desactiva un plugin.
const integrationsEnabledKey = manifest.ReservedConfigKey

// decodeIntegrations decodifica y valida el bloque `integrations:` contra el catálogo cerrado de
// manifiestos. Rechaza un bloque nulo o que no es un mapeo, los plugins fuera del catálogo, los
// valores de plugin que no son un mapeo, las claves fuera del manifiesto (los secretos no son
// configuración: una clave de secreto es desconocida y su valor nunca se imprime), las repetidas,
// los tipos incorrectos, los valores fuera del enumerado y las claves obligatorias ausentes (salvo
// con `enabled: false`). Acumula todas las violaciones con su línea y las alternativas.
func decodeIntegrations(node *yaml.Node) (*IntegrationsConfig, error) {
	node = resolveAlias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, errors.New("el bloque 'integrations' debe ser un mapeo de plugins (`integrations: {}` no configura ninguno); para no declararlo, elimina el bloque")
	}

	var errs []error
	config := &IntegrationsConfig{}
	seen := map[string]bool{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		m, known := manifest.Lookup(key.Value)
		if !known {
			errs = append(errs, fmt.Errorf("plugin desconocido '%s' en 'integrations' (línea %d); plugins admitidos: %s", key.Value, key.Line, supportedPlugins()))
			continue
		}
		if seen[key.Value] {
			errs = append(errs, fmt.Errorf("plugin duplicado '%s' en 'integrations' (línea %d)", key.Value, key.Line))
			continue
		}
		seen[key.Value] = true

		plugin, pluginErrs := decodeIntegrationPlugin(m, key, node.Content[i+1])
		errs = append(errs, pluginErrs...)
		if len(pluginErrs) == 0 {
			config.Plugins = append(config.Plugins, plugin)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return config, nil
}

// supportedPlugins lista los plugins del catálogo, o dice que no hay ninguno en esta versión.
func supportedPlugins() string {
	catalog := manifest.Catalog()
	if len(catalog) == 0 {
		return "ninguno en esta versión"
	}
	ids := make([]string, len(catalog))
	for i, m := range catalog {
		ids[i] = m.ID
	}
	return strings.Join(ids, ", ")
}

func decodeIntegrationPlugin(m manifest.Manifest, key, value *yaml.Node) (IntegrationPluginConfig, []error) {
	label := "integrations." + m.ID
	value = resolveAlias(value)
	if isNull(value) || value.Kind != yaml.MappingNode {
		return IntegrationPluginConfig{}, []error{fmt.Errorf("%s debe ser un mapeo con 'enabled' y las claves del plugin (línea %d); para no configurarlo, elimínalo o usa `%s: {}`", label, key.Line, m.ID)}
	}

	allowed := []string{integrationsEnabledKey}
	fields := map[string]manifest.ConfigField{}
	for _, f := range m.Config {
		allowed = append(allowed, f.Key)
		fields[f.Key] = f
	}

	errs := unknownKeys(value, label, allowed)
	plugin := IntegrationPluginConfig{ID: m.ID, Enabled: true, Values: map[string]any{}, Line: key.Line}
	declared := map[string]bool{}
	for i := 0; i+1 < len(value.Content); i += 2 {
		k := value.Content[i]
		if !containsString(allowed, k.Value) {
			continue // ya reportada por unknownKeys
		}
		if declared[k.Value] {
			errs = append(errs, fmt.Errorf("clave duplicada '%s' en '%s' (línea %d)", k.Value, label, k.Line))
			continue
		}
		declared[k.Value] = true

		if k.Value == integrationsEnabledKey {
			enabled, err := sddBool(label+"."+k.Value, value.Content[i+1])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			plugin.Enabled = enabled
			continue
		}
		v, err := decodeIntegrationValue(label+"."+k.Value, fields[k.Value], value.Content[i+1])
		if err != nil {
			errs = append(errs, err)
			continue
		}
		plugin.Values[k.Value] = v
	}

	for _, f := range m.Config {
		if declared[f.Key] {
			continue
		}
		if f.Default != nil {
			plugin.Values[f.Key] = normalizeDefault(f)
			continue
		}
		if f.Required && plugin.Enabled {
			errs = append(errs, fmt.Errorf("%s.%s es obligatoria y falta (plugin declarado en la línea %d de 'integrations'); añádela o desactiva el plugin con `enabled: false`", label, f.Key, key.Line))
		}
	}
	return plugin, errs
}

// decodeIntegrationValue lee un valor con el tipo estricto que declara el manifiesto.
func decodeIntegrationValue(label string, field manifest.ConfigField, node *yaml.Node) (any, error) {
	node = resolveAlias(node)
	if isNull(node) {
		return nil, fmt.Errorf("%s no puede ser un valor nulo: %s", label, expectedType(field.Type))
	}
	switch field.Type {
	case manifest.FieldString:
		s, err := scalarString(label, node)
		if err != nil {
			return nil, err
		}
		if err := checkEnum(label, s, field.Enum); err != nil {
			return s, err
		}
		return s, checkStringFormat(label, field, s)
	case manifest.FieldBool:
		return sddBool(label, node)
	case manifest.FieldInt:
		n, err := scalarInt(label, node)
		if err != nil {
			return n, err
		}
		return n, checkIntRange(label, field, n)
	case manifest.FieldStringList:
		return stringList(label, field, node)
	}
	return nil, fmt.Errorf("%s: tipo %q no admitido por el manifiesto", label, field.Type)
}

func expectedType(t manifest.FieldType) string {
	switch t {
	case manifest.FieldString:
		return "debe ser una cadena"
	case manifest.FieldBool:
		return "debe ser true o false"
	case manifest.FieldInt:
		return "debe ser un entero"
	case manifest.FieldStringList:
		return "debe ser una lista de cadenas"
	}
	return "valor no admitido"
}

func scalarString(label string, node *yaml.Node) (string, error) {
	switch {
	case node.Kind == yaml.SequenceNode:
		return "", fmt.Errorf("%s debe ser una cadena, no una lista", label)
	case node.Kind != yaml.ScalarNode:
		return "", fmt.Errorf("%s debe ser una cadena, no un mapeo", label)
	case node.Tag != "!!str":
		return "", fmt.Errorf("%s debe ser una cadena: el valor '%s' no lo es (entrecomíllalo si es texto)", label, node.Value)
	}
	return node.Value, nil
}

func scalarInt(label string, node *yaml.Node) (int, error) {
	switch {
	case node.Kind == yaml.SequenceNode:
		return 0, fmt.Errorf("%s debe ser un entero, no una lista", label)
	case node.Kind != yaml.ScalarNode:
		return 0, fmt.Errorf("%s debe ser un entero, no un mapeo", label)
	case node.Tag != "!!int":
		return 0, fmt.Errorf("%s debe ser un entero: el valor '%s' no lo es", label, node.Value)
	}
	var n int
	if err := node.Decode(&n); err != nil {
		return 0, fmt.Errorf("%s debe ser un entero: el valor '%s' no es válido", label, node.Value)
	}
	return n, nil
}

func stringList(label string, field manifest.ConfigField, node *yaml.Node) ([]string, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("%s debe ser una lista de cadenas, no un valor suelto", label)
	}
	var errs []error
	list := make([]string, 0, len(node.Content))
	lines := make([]int, 0, len(node.Content))
	for _, item := range node.Content {
		item = resolveAlias(item)
		if isNull(item) || item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
			errs = append(errs, fmt.Errorf("%s debe ser una lista de cadenas: el elemento de la línea %d no lo es", label, item.Line))
			continue
		}
		if err := checkEnum(label, item.Value, field.Enum); err != nil {
			errs = append(errs, err)
			continue
		}
		list = append(list, item.Value)
		lines = append(lines, item.Line)
	}
	if len(errs) == 0 && field.Format == manifest.FormatTypePrefixPairs {
		_, problems := manifest.ParseTypePrefixes(list)
		for _, p := range problems {
			errs = append(errs, fmt.Errorf("%s: el elemento '%s' (línea %d) no es válido: %s", label, p.Element, lines[p.Index], p.Reason))
		}
	}
	if len(errs) == 0 && field.Format == manifest.FormatWeekdaySet {
		for _, p := range manifest.ParseWeekdaySet(list) {
			if p.Index < 0 {
				errs = append(errs, fmt.Errorf("%s: %s", label, p.Reason))
				continue
			}
			errs = append(errs, fmt.Errorf("%s: el elemento '%s' (línea %d) no es válido: %s", label, p.Element, lines[p.Index], p.Reason))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return list, nil
}

// checkStringFormat aplica los formatos non_empty y tag_prefix a un valor string.
func checkStringFormat(label string, field manifest.ConfigField, value string) error {
	var reason string
	switch field.Format {
	case manifest.FormatNonEmpty:
		reason = manifest.CheckNonEmpty(value)
	case manifest.FormatTagPrefix:
		reason = manifest.CheckTagPrefix(value)
	}
	if reason != "" {
		return fmt.Errorf("%s: el valor '%s' no es válido: %s", label, value, reason)
	}
	return nil
}

// checkIntRange aplica Min y Max (inclusivos) de un campo int.
func checkIntRange(label string, field manifest.ConfigField, n int) error {
	if field.Min != nil && n < *field.Min || field.Max != nil && n > *field.Max {
		return fmt.Errorf("%s: el valor %d está fuera del rango admitido (%s)", label, n, intRangeText(field))
	}
	return nil
}

func intRangeText(f manifest.ConfigField) string {
	switch {
	case f.Min != nil && f.Max != nil:
		return fmt.Sprintf("de %d a %d", *f.Min, *f.Max)
	case f.Min != nil:
		return fmt.Sprintf("mínimo %d", *f.Min)
	}
	return fmt.Sprintf("máximo %d", *f.Max)
}

func checkEnum(label, value string, enum []string) error {
	if len(enum) == 0 || containsString(enum, value) {
		return nil
	}
	return fmt.Errorf("%s: el valor '%s' no es válido; valores admitidos: %s", label, value, strings.Join(enum, ", "))
}

// normalizeDefault convierte el Default de un manifiesto (que tras un JSON puede ser float64 o
// []any) al tipo con el que se proyectan los valores declarados.
func normalizeDefault(f manifest.ConfigField) any {
	switch f.Type {
	case manifest.FieldInt:
		switch n := f.Default.(type) {
		case float64:
			return int(n)
		case int64:
			return int(n)
		}
	case manifest.FieldStringList:
		if l, ok := f.Default.([]any); ok {
			out := make([]string, 0, len(l))
			for _, e := range l {
				if s, ok := e.(string); ok {
					out = append(out, s)
				}
			}
			return out
		}
		if l, ok := f.Default.([]string); ok {
			return append([]string(nil), l...)
		}
	}
	return f.Default
}
