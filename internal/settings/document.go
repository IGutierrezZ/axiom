package settings

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Snapshot es la lectura de un axiom.yaml: sus valores tal como están escritos, su
// revisión y el catálogo expandido para ese documento.
type Snapshot struct {
	ConfigPath         string
	Revision           string
	Topology           workspace.TopologyType
	ISO                bool
	Writable           bool
	WriteBlockedReason string
	DocumentErrors     []string
	Entries            []Entry
}

// revisionOf es la revisión de un contenido: el SHA-256 hexadecimal de sus bytes.
func revisionOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// document es un axiom.yaml decodificado a yaml.Node.
type document struct {
	data     []byte
	revision string
	root     *yaml.Node // mapeo de primer nivel; nil si el documento no es un mapeo
	blocked  string     // motivo por el que no se puede reescribir; "" si se puede
}

// parseDocument decodifica data y detecta las construcciones que impiden
// reescribirlo sin riesgo: más de un documento (reescribirlo borraría los
// siguientes), anclas, alias, claves de mezcla `<<` y claves repetidas.
func parseDocument(data []byte) *document {
	doc := &document{data: data, revision: revisionOf(data)}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var first yaml.Node
	if err := decoder.Decode(&first); err != nil {
		if errors.Is(err, io.EOF) {
			doc.blocked = "el documento está vacío"
		} else {
			doc.blocked = "el documento no es YAML válido: " + firstLine(err.Error())
		}
		return doc
	}
	var second yaml.Node
	switch err := decoder.Decode(&second); {
	case err == nil:
		line := second.Line
		if len(second.Content) > 0 {
			line = second.Content[0].Line
		}
		doc.blocked = fmt.Sprintf("contiene más de un documento YAML (el segundo empieza en la línea %d): reescribirlo borraría los siguientes", line)
		return doc
	case !errors.Is(err, io.EOF):
		doc.blocked = "el documento no es YAML válido: " + firstLine(err.Error())
		return doc
	}
	if len(first.Content) == 0 || first.Content[0].Kind != yaml.MappingNode {
		doc.blocked = "el documento no es un mapeo de claves"
		return doc
	}
	doc.root = first.Content[0]
	doc.blocked = unsupportedConstruct(doc.root)
	return doc
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

// unsupportedConstruct devuelve el motivo de rechazo de la primera construcción no
// soportada del árbol, con su línea, o "" si no hay ninguna. El bloque
// `integrations` no se recorre: nunca se lee ni se edita desde aquí.
func unsupportedConstruct(root *yaml.Node) string {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if reason := keyConstruct(root, root.Content[i]); reason != "" {
			return reason
		}
		if root.Content[i].Value == "integrations" {
			continue
		}
		if reason := walkConstructs(root.Content[i+1]); reason != "" {
			return reason
		}
	}
	return ""
}

// keyConstruct comprueba la clave key del mapeo m: `<<` y claves repetidas.
func keyConstruct(m, key *yaml.Node) string {
	if key.Tag == "!!merge" || key.Value == "<<" {
		return fmt.Sprintf("usa una clave de mezcla `<<` (línea %d)", key.Line)
	}
	if key.Anchor != "" {
		return fmt.Sprintf("usa el ancla &%s (línea %d)", key.Anchor, key.Line)
	}
	count := 0
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key.Value {
			count++
			if count == 2 && m.Content[i] == key {
				return fmt.Sprintf("repite la clave %q (línea %d)", key.Value, key.Line)
			}
		}
	}
	return ""
}

func walkConstructs(n *yaml.Node) string {
	if n == nil {
		return ""
	}
	switch {
	case n.Kind == yaml.AliasNode:
		return fmt.Sprintf("usa el alias *%s (línea %d)", n.Value, n.Line)
	case n.Anchor != "":
		return fmt.Sprintf("usa el ancla &%s (línea %d)", n.Anchor, n.Line)
	}
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			if reason := keyConstruct(n, n.Content[i]); reason != "" {
				return reason
			}
		}
	}
	for _, child := range n.Content {
		if reason := walkConstructs(child); reason != "" {
			return reason
		}
	}
	return ""
}

// mapValue devuelve el valor y el nodo clave de key en el mapeo m, o nil si falta.
func mapValue(m *yaml.Node, key string) (value, keyNode *yaml.Node) {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1], m.Content[i]
		}
	}
	return nil, nil
}

// pathValue sigue una ruta de claves `a.b` desde el mapeo m.
func pathValue(m *yaml.Node, dotted string) *yaml.Node {
	node := m
	for _, part := range strings.Split(dotted, ".") {
		node, _ = mapValue(node, part)
		if node == nil {
			return nil
		}
	}
	return node
}

func isNullNode(n *yaml.Node) bool {
	return n == nil || (n.Kind == yaml.ScalarNode && n.Tag == "!!null")
}

// topologyOf lee la topología como `Location.GoverningTopology`: solo `multirepo`
// es multirepo; cualquier otro valor, o su ausencia, se trata como embebida.
func (d *document) topologyOf() workspace.TopologyType {
	if n := pathValue(d.root, "workspace.topology"); n != nil && n.Kind == yaml.ScalarNode && n.Value == string(workspace.TopologyMultirepo) {
		return workspace.TopologyMultirepo
	}
	return workspace.TopologyMonorepoEmbedded
}

// isoOf informa de si `sdd.iso` es el booleano `true`.
func (d *document) isoOf() bool {
	n := pathValue(d.root, "sdd.iso")
	return n != nil && n.Kind == yaml.ScalarNode && n.Tag == "!!bool" && n.Value == "true"
}

// serialize escribe el subárbol n como YAML con sangría de dos espacios.
func serialize(n *yaml.Node) string {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(n); err != nil {
		return ""
	}
	_ = enc.Close()
	return strings.TrimRight(buf.String(), "\r\n")
}

// valueOf devuelve el valor de n tal como está escrito, según el tipo del catálogo.
// Un valor de tipo distinto del esperado se devuelve en crudo, para que la pantalla
// muestre lo que hay y la persona lo corrija.
func valueOf(n *yaml.Node, t ValueType) any {
	if isNullNode(n) {
		return nil
	}
	if t == TypeYAML || n.Kind == yaml.MappingNode {
		return serialize(n)
	}
	if n.Kind == yaml.SequenceNode {
		if t != TypeStringList {
			return serialize(n)
		}
		items := make([]string, 0, len(n.Content))
		for _, item := range n.Content {
			if item.Kind == yaml.ScalarNode {
				items = append(items, item.Value)
			} else {
				items = append(items, serialize(item))
			}
		}
		return items
	}
	switch {
	case t == TypeBool && n.Tag == "!!bool":
		return n.Value == "true"
	case t == TypeInt && n.Tag == "!!int":
		if v, err := strconv.Atoi(n.Value); err == nil {
			return v
		}
	}
	return n.Value
}

// Read lee configPath y devuelve el catálogo expandido para ese documento, con el
// valor de cada clave tal como está escrito, aunque el documento sea inválido.
func Read(configPath string) (*Snapshot, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, newError(CodeConfigMissing, "el proyecto no tiene un axiom.yaml en "+configPath)
		}
		return nil, fmt.Errorf("leer %s: %w", configPath, err)
	}
	doc := parseDocument(data)
	snap := &Snapshot{
		ConfigPath:         configPath,
		Revision:           doc.revision,
		Topology:           workspace.TopologyMonorepoEmbedded,
		Writable:           doc.blocked == "",
		WriteBlockedReason: doc.blocked,
		DocumentErrors:     documentErrors(data),
	}
	if doc.root == nil {
		return snap, nil
	}
	snap.Topology = doc.topologyOf()
	snap.ISO = doc.isoOf()
	snap.Entries = doc.entries(snap.Topology, snap.ISO)
	return snap, nil
}

// documentErrors devuelve las violaciones de ParseConfig, una por línea.
func documentErrors(data []byte) []string {
	_, err := workspace.ParseConfig(data)
	if err == nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(err.Error(), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// repositoryCandidates son las rutas de repositorio normalizadas que admiten
// ajustes: las entradas de `repositories:` y las de los repositorios de los roles.
func (d *document) repositoryCandidates() []string {
	seen := map[string]bool{}
	if repos, _ := mapValue(d.root, "repositories"); repos != nil && repos.Kind == yaml.SequenceNode {
		for _, item := range repos.Content {
			if p, _ := mapValue(item, "path"); p != nil && p.Kind == yaml.ScalarNode && p.Value != "" {
				seen[normalizeRepoPath(p.Value)] = true
			}
		}
	}
	if roles, _ := mapValue(d.root, "roles"); roles != nil && roles.Kind == yaml.MappingNode {
		for i := 1; i < len(roles.Content); i += 2 {
			list, _ := mapValue(roles.Content[i], "repositories")
			if list == nil || list.Kind != yaml.SequenceNode {
				continue
			}
			for _, item := range list.Content {
				if p, _ := mapValue(item, "path"); p != nil && p.Kind == yaml.ScalarNode && p.Value != "" {
					seen[normalizeRepoPath(p.Value)] = true
				}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// roleIDs devuelve los ids de los roles declarados, ordenados.
func (d *document) roleIDs() []string {
	roles, _ := mapValue(d.root, "roles")
	if roles == nil || roles.Kind != yaml.MappingNode {
		return nil
	}
	var ids []string
	for i := 0; i+1 < len(roles.Content); i += 2 {
		ids = append(ids, roles.Content[i].Value)
	}
	sort.Strings(ids)
	return ids
}

// repositoryEntry devuelve el nodo de la entrada de `repositories:` cuya ruta
// normalizada es p, o nil.
func (d *document) repositoryEntry(p string) *yaml.Node {
	repos, _ := mapValue(d.root, "repositories")
	if repos == nil || repos.Kind != yaml.SequenceNode {
		return nil
	}
	for _, item := range repos.Content {
		if v, _ := mapValue(item, "path"); v != nil && v.Kind == yaml.ScalarNode && normalizeRepoPath(v.Value) == p {
			return item
		}
	}
	return nil
}

// entries expande el catálogo para este documento.
func (d *document) entries(topology workspace.TopologyType, iso bool) []Entry {
	multi := topology == workspace.TopologyMultirepo
	var out []Entry
	build := func(spec keySpec, key string, node *yaml.Node, editable bool) Entry {
		entry := Entry{
			Key:         key,
			Section:     spec.section,
			Label:       spec.label,
			Description: spec.description,
			Effect:      spec.effect,
			Type:        spec.typ,
			Editable:    editable,
			Required:    spec.required,
			Declared:    node != nil,
			Value:       valueOf(node, spec.typ),
		}
		if spec.def != nil {
			entry.Default = spec.def()
		}
		if vocab := spec.vocabulary(); vocab != nil {
			entry.Allowed = slices.DeleteFunc(slices.Clone(vocab), func(v string) bool {
				return iso && slices.Contains(spec.isoDrop, v)
			})
		}
		if spec.multirepoOnly {
			entry.Topologies = []string{topologyMulti}
		} else {
			entry.Topologies = []string{topologyMonorepo, topologyMulti}
		}
		if !editable {
			entry.ReadOnlyReason = readOnlyReason
		}
		return entry
	}

	// Claves de solo lectura de workspace, antes de las editables.
	for _, spec := range readOnlySpecs[:4] {
		out = append(out, build(spec, spec.key, pathValue(d.root, spec.key), false))
	}
	for _, spec := range editableSpecs {
		if spec.multirepoOnly && !multi {
			continue
		}
		switch spec.scope {
		case scopeBlock:
			out = append(out, build(spec, spec.key, pathValue(d.root, spec.key), true))
		case scopeRepo:
			field := strings.TrimPrefix(spec.key, "repositories[].")
			for _, p := range d.repositoryCandidates() {
				var node *yaml.Node
				if entry := d.repositoryEntry(p); entry != nil {
					node, _ = mapValue(entry, field)
				}
				out = append(out, build(spec, "repositories["+p+"]."+field, node, true))
			}
		case scopeRole:
			field := strings.TrimPrefix(spec.key, "roles[].")
			roles, _ := mapValue(d.root, "roles")
			for _, id := range d.roleIDs() {
				role, _ := mapValue(roles, id)
				node, _ := mapValue(role, field)
				out = append(out, build(spec, "roles["+id+"]."+field, node, true))
			}
		}
	}
	tc, _ := mapValue(d.root, "technical_context")
	out = append(out, build(readOnlySpecs[4], readOnlySpecs[4].key, tc, false))
	if gov, _ := mapValue(d.root, "governance"); gov != nil && gov.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(gov.Content); i += 2 {
			spec := governanceSpec(gov.Content[i].Value)
			if v := gov.Content[i+1]; v.Kind != yaml.ScalarNode {
				spec.typ = TypeYAML
			}
			out = append(out, build(spec, spec.key, gov.Content[i+1], false))
		}
	}
	return out
}
