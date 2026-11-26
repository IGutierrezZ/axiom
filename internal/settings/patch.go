package settings

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Patch es un parche de claves del catálogo a valores; un valor nil retira la clave.
// Tipos dinámicos admitidos: string, bool, int, float64 entero y []any o []string de
// cadenas.
type Patch map[string]any

// canonicalBlocks es el orden canónico de los bloques de primer nivel que el escritor
// crea. Coincide con el axiom.yaml del repositorio y deja los bloques nuevos antes
// de `roles` y `governance` en los ficheros que genera el alta.
var canonicalBlocks = []string{"workspace", "git", "repositories", "workflow", "sdd", "delivery", "acceptance_tests", "integrations"}

// operation es un cambio ya resuelto contra el catálogo y el documento.
type operation struct {
	key   string
	spec  keySpec
	parse parsedKey
	value any // nil = retirar
}

// isReadOnlyKey informa de si la clave es una de solo lectura del catálogo.
func isReadOnlyKey(key string) bool {
	if key == "technical_context" || strings.HasPrefix(key, "technical_context.") || strings.HasPrefix(key, "governance.") {
		return true
	}
	return slices.ContainsFunc(readOnlySpecs, func(s keySpec) bool { return s.key == key })
}

// applyPatch aplica el parche entero sobre el árbol del documento. Es un lote: si
// una clave falla devuelve su error y el llamante descarta el árbol, así que no se
// escribe ninguna.
func (d *document) applyPatch(patch Patch) *Error {
	keys := make([]string, 0, len(patch))
	for key := range patch {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	topology := d.topologyOf()
	ops := make([]operation, 0, len(keys))
	for _, key := range keys {
		op, err := d.resolve(key, patch[key], topology)
		if err != nil {
			return err
		}
		ops = append(ops, op)
	}
	for _, op := range ops {
		if err := d.apply(op); err != nil {
			return err
		}
	}
	return nil
}

// resolve comprueba una clave del parche contra el catálogo y el documento y
// convierte su valor al tipo de la clave.
func (d *document) resolve(key string, raw any, topology workspace.TopologyType) (operation, *Error) {
	if isReadOnlyKey(key) {
		return operation{}, newError(CodeKeyNotEditable, fmt.Sprintf("la clave %q es de solo lectura: edítala a mano en axiom.yaml", key))
	}
	parsed := parseKey(key)
	var spec keySpec
	var ok bool
	if parsed.scope == scopePlugin {
		spec, ok = findPluginSpec(parsed.selector, parsed.field)
	} else {
		spec, ok = findSpec(parsed.template)
	}
	if !ok {
		return operation{}, newError(CodeUnknownKey, fmt.Sprintf("la clave %q no figura en el catálogo de ajustes del proyecto", key))
	}
	if spec.multirepoOnly && topology != workspace.TopologyMultirepo {
		return operation{}, newError(CodeKeyNotApplicable, fmt.Sprintf("la clave %q solo aplica a proyectos multirepo", key))
	}
	switch spec.scope {
	case scopePlugin:
		// el plugin y el campo ya se comprobaron contra el manifiesto
	case scopeRepo:
		selector := normalizeRepoPath(parsed.selector)
		if !slices.Contains(d.repositoryCandidates(), selector) {
			return operation{}, newError(CodeKeyNotApplicable, fmt.Sprintf("el repositorio %q no figura en axiom.yaml: %q no aplica", parsed.selector, key))
		}
		parsed.selector = selector
	case scopeRole:
		if !slices.Contains(d.roleIDs(), parsed.selector) {
			return operation{}, newError(CodeKeyNotApplicable, fmt.Sprintf("el rol %q no existe en axiom.yaml: %q no aplica", parsed.selector, key))
		}
	}
	value, err := convertValue(key, spec, raw)
	if err != nil {
		return operation{}, err
	}
	return operation{key: key, spec: spec, parse: parsed, value: value}, nil
}

// convertValue valida el tipo de raw y lo normaliza: string, bool, int o []string.
// El dominio de los valores (rangos, formato de ramas, vocabularios que decodifica el
// motor) lo decide la validación del documento resultante.
func convertValue(key string, spec keySpec, raw any) (any, *Error) {
	if raw == nil {
		if spec.required {
			return nil, newError(CodeInvalidValue, fmt.Sprintf("la clave %q es obligatoria y no se puede retirar", key))
		}
		return nil, nil
	}
	wrong := func(want string) *Error {
		return newError(CodeInvalidValue, fmt.Sprintf("la clave %q debe ser %s", key, want))
	}
	switch spec.typ {
	case TypeString, TypeEnum:
		text, ok := raw.(string)
		if !ok {
			return nil, wrong("un texto")
		}
		if spec.typ == TypeEnum && spec.scope == scopeRole {
			text = strings.ToLower(strings.TrimSpace(text))
		}
		if !spec.engineChecked && spec.allowed != nil && !slices.Contains(spec.vocabulary(), text) {
			return nil, newError(CodeInvalidValue, fmt.Sprintf("el valor %q de %q no es válido: usa %s", text, key, strings.Join(spec.vocabulary(), ", ")))
		}
		return text, nil
	case TypeBool:
		flag, ok := raw.(bool)
		if !ok {
			return nil, wrong("true o false")
		}
		return flag, nil
	case TypeInt:
		n, ok := intFromAny(raw)
		if !ok {
			return nil, wrong("un número entero")
		}
		return n, nil
	case TypeStringList:
		switch list := raw.(type) {
		case []string:
			return slices.Clone(list), nil
		case []any:
			out := make([]string, 0, len(list))
			for _, item := range list {
				text, ok := item.(string)
				if !ok {
					return nil, wrong("una lista de textos")
				}
				out = append(out, text)
			}
			return out, nil
		}
		return nil, wrong("una lista de textos")
	}
	return nil, newError(CodeKeyNotEditable, fmt.Sprintf("la clave %q no se puede editar", key))
}

func intFromAny(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		if n == float64(int(n)) {
			return int(n), true
		}
	}
	return 0, false
}

// apply ejecuta una operación sobre el árbol.
func (d *document) apply(op operation) *Error {
	switch op.spec.scope {
	case scopeRepo:
		return d.applyRepository(op)
	case scopePlugin:
		return d.applyPlugin(op)
	case scopeRole:
		roles, _ := mapValue(d.root, "roles")
		role, _ := mapValue(roles, op.parse.selector)
		if op.value == nil {
			removeKey(role, op.parse.field) // el mapeo del rol no se borra nunca
			return nil
		}
		setValue(role, op.parse.field, op.spec.typ, op.value)
		return nil
	}
	block, field, _ := strings.Cut(op.spec.key, ".")
	if op.value == nil {
		node, _ := mapValue(d.root, block)
		if node == nil || node.Kind != yaml.MappingNode {
			return nil
		}
		removeKey(node, field)
		if len(node.Content) == 0 {
			removeKey(d.root, block)
		}
		return nil
	}
	setValue(d.ensureBlock(block, yaml.MappingNode), field, op.spec.typ, op.value)
	return nil
}

// applyRepository aplica un campo de `repositories[<ruta>]`: crea la entrada y el
// bloque si faltan y, al retirar, borra la entrada y el bloque que queden vacíos.
func (d *document) applyRepository(op operation) *Error {
	entry := d.repositoryEntry(op.parse.selector)
	if op.value == nil {
		if entry == nil {
			return nil
		}
		removeKey(entry, op.parse.field)
		if onlyPath(entry) {
			seq, _ := mapValue(d.root, "repositories")
			seq.Content = slices.DeleteFunc(seq.Content, func(n *yaml.Node) bool { return n == entry })
			if len(seq.Content) == 0 {
				removeKey(d.root, "repositories")
			}
		}
		return nil
	}
	if entry == nil {
		seq := d.ensureBlock("repositories", yaml.SequenceNode)
		entry = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setValue(entry, "path", TypeString, op.parse.selector)
		seq.Content = append(seq.Content, entry)
	}
	setValue(entry, op.parse.field, op.spec.typ, op.value)
	return nil
}

// applyPlugin aplica un campo de `integrations[<plugin>]`: crea el bloque y la entrada del
// plugin si faltan (un mapeo en estilo de flujo, como `{}`, pasa a estilo de bloque) y, al
// retirar, borra la entrada del plugin y el bloque que queden vacíos.
func (d *document) applyPlugin(op operation) *Error {
	if op.value == nil {
		block, _ := mapValue(d.root, "integrations")
		plugin, _ := mapValue(block, op.parse.selector)
		if plugin == nil || plugin.Kind != yaml.MappingNode {
			return nil
		}
		removeKey(plugin, op.parse.field)
		if len(plugin.Content) == 0 {
			removeKey(block, op.parse.selector)
		}
		if len(block.Content) == 0 {
			removeKey(d.root, "integrations")
		}
		return nil
	}
	block := d.ensureBlock("integrations", yaml.MappingNode)
	block.Style &^= yaml.FlowStyle
	plugin, _ := mapValue(block, op.parse.selector)
	if plugin == nil || plugin.Kind != yaml.MappingNode {
		plugin = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		replaced := false
		for i := 0; i+1 < len(block.Content); i += 2 {
			if block.Content[i].Value == op.parse.selector {
				block.Content[i+1] = plugin
				replaced = true
			}
		}
		if !replaced {
			block.Content = append(block.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: op.parse.selector}, plugin)
		}
	}
	plugin.Style &^= yaml.FlowStyle
	setValue(plugin, op.parse.field, op.spec.typ, op.value)
	return nil
}

// onlyPath informa de si el mapeo de una entrada de repositories ya no declara más
// que su `path`.
func onlyPath(entry *yaml.Node) bool {
	return len(entry.Content) == 2 && entry.Content[0].Value == "path"
}

// ensureBlock devuelve el bloque de primer nivel de ese tipo de nodo, creándolo en
// el orden canónico si falta o si es un escalar (p. ej. un `git:` nulo).
func (d *document) ensureBlock(name string, kind yaml.Kind) *yaml.Node {
	if node, _ := mapValue(d.root, name); node != nil && node.Kind == kind {
		return node
	}
	tag := "!!map"
	if kind == yaml.SequenceNode {
		tag = "!!seq"
	}
	node := &yaml.Node{Kind: kind, Tag: tag}
	if _, keyNode := mapValue(d.root, name); keyNode != nil {
		for i := 0; i+1 < len(d.root.Content); i += 2 {
			if d.root.Content[i] == keyNode {
				d.root.Content[i+1] = node
				return node
			}
		}
	}
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name}
	at := d.insertionIndex(name)
	d.root.Content = slices.Insert(d.root.Content, at, keyNode, node)
	return node
}

// insertionIndex es la posición en Content donde va el bloque nuevo: tras la última
// clave de primer nivel que lo precede en el orden canónico; si no hay ninguna, tras
// `workspace` y, si tampoco existe, al final.
func (d *document) insertionIndex(name string) int {
	rank := slices.Index(canonicalBlocks, name)
	last := -1
	for i := 0; i+1 < len(d.root.Content); i += 2 {
		if r := slices.Index(canonicalBlocks, d.root.Content[i].Value); r >= 0 && r < rank {
			last = i
		}
	}
	if last < 0 {
		for i := 0; i+1 < len(d.root.Content); i += 2 {
			if d.root.Content[i].Value == "workspace" {
				last = i
			}
		}
	}
	if last < 0 {
		return len(d.root.Content)
	}
	return last + 2
}

// removeKey borra el par clave-valor key del mapeo m, si existe.
func removeKey(m *yaml.Node, key string) {
	if m == nil || m.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = slices.Delete(m.Content, i, i+2)
			return
		}
	}
}

// setValue fija key en el mapeo m. En un escalar existente sustituye Value y Tag y
// conserva Style; si la clave falta, la añade al final del mapeo con la cadena en
// comillas dobles, y el booleano y el entero sin comillas.
func setValue(m *yaml.Node, key string, typ ValueType, value any) {
	current, _ := mapValue(m, key)
	fresh := buildValue(typ, value, current)
	if current == nil {
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, fresh)
		return
	}
	if current.Kind == yaml.ScalarNode && fresh.Kind == yaml.ScalarNode {
		current.Value, current.Tag = fresh.Value, fresh.Tag
		current.Style = fresh.Style
		return
	}
	if current.Kind == yaml.SequenceNode && fresh.Kind == yaml.SequenceNode {
		current.Content = fresh.Content
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			fresh.HeadComment, fresh.LineComment, fresh.FootComment = current.HeadComment, current.LineComment, current.FootComment
			m.Content[i+1] = fresh
			return
		}
	}
}

// buildValue construye el nodo de un valor. Si ya hay un nodo, conserva su estilo de
// comillas para las cadenas (`lane: "ask"` sigue entre comillas).
func buildValue(typ ValueType, value any, existing *yaml.Node) *yaml.Node {
	switch typ {
	case TypeBool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: fmt.Sprintf("%t", value.(bool))}
	case TypeInt:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%d", value.(int))}
	case TypeStringList:
		items := value.([]string)
		seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		itemStyle := yaml.DoubleQuotedStyle
		if existing != nil && existing.Kind == yaml.SequenceNode {
			seq.Style = existing.Style
			if len(existing.Content) > 0 {
				itemStyle = existing.Content[0].Style
			}
		}
		for _, item := range items {
			seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: item, Style: itemStyle})
		}
		return seq
	}
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value.(string), Style: yaml.DoubleQuotedStyle}
	if existing != nil && existing.Kind == yaml.ScalarNode && existing.Tag == "!!str" {
		node.Style = existing.Style
		if node.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 && !strings.Contains(node.Value, "\n") {
			node.Style = yaml.DoubleQuotedStyle
		}
	}
	return node
}
