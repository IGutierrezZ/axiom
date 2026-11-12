package workspace

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TechnicalContextConfig es el bloque `technical_context:` de axiom.yaml (REQ-24.26).
type TechnicalContextConfig struct {
	Maintained *ContextMode `yaml:"maintained,omitempty"`
	Reference  *ContextMode `yaml:"reference,omitempty"`
}

// ContextMode es uno de los dos modos del contexto técnico.
type ContextMode struct {
	Entry string   `yaml:"entry,omitempty"`
	Paths []string `yaml:"paths"`
}

// Declarations reúne las declaraciones estructurales de axiom.yaml y su estado
// de validación. El valor cero equivale a «sin declaraciones»: no hay
// axiom.yaml, o es legible y no declara los bloques nuevos. Cada bloque tiene su
// propio canal de error (D-03): una declaración inválida nunca impide leer la
// configuración base, y la configuración base inválida nunca oculta los errores
// de las declaraciones.
type Declarations struct {
	ParseErr            error                 // no nulo solo si existe axiom.yaml y no se puede leer como YAML
	BaselineRetiredErr  *BaselineRetiredError // no nulo si axiom.yaml aún declara `baseline:`, retirado (REQ-30.21)
	TechnicalContext    *TechnicalContextConfig
	TechnicalContextErr error
	Git                 *GitConfig // lo declarado en `git:`, sin defectos; nil sin bloque o con bloque inválido (REQ-25.26)
	GitErr              error
	Workflow            *WorkflowConfig // lo declarado en `workflow:`, sin defectos; nil sin bloque o con bloque inválido (REQ-27.1)
	WorkflowErr         error
	SDD                 *SDDConfig // lo declarado en `sdd:`, sin defectos; nil sin bloque, con bloque inválido o con la contradicción con `workflow.lane` (REQ-29.1)
	SDDErr              error
	Repositories        []RepositoryPolicy // lo declarado en `repositories:`; nil sin bloque, sin entradas o con bloque inválido (REQ-25.27)
	RepositoriesErr     error
}

// Claves admitidas por nodo (D-04). yaml.v3 no ofrece KnownFields en
// Node.Decode, así que cada mapeo de los bloques nuevos se recorre contra su
// lista cerrada; el resto de claves de axiom.yaml se sigue ignorando (REQ-24.17).
var (
	technicalCtxKeys  = []string{"maintained", "reference"}
	contextModeKeys   = []string{"entry", "paths"}
	contextModeLabels = [...]string{"maintained", "reference"}
)

// readDeclarationBlocks localiza los bloques `technical_context`,
// `git` y `repositories` en el mapeo raíz del documento y los decodifica y valida
// de forma estricta. displacement es el desplazamiento léxico de la carpeta
// maestra a workspaceRoot.
func readDeclarationBlocks(document *yaml.Node, displacement string) Declarations {
	root := document
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	root = resolveAlias(root)
	if root.Kind != yaml.MappingNode {
		return Declarations{}
	}

	var declarations Declarations
	if line, ok := mappingKeyLine(root, "baseline"); ok {
		// Un `baseline:` residual nunca se ignora ni se decodifica: es la causa
		// tipada de REQ-30.21, con la guía de migración.
		declarations.BaselineRetiredErr = &BaselineRetiredError{Line: line}
	}
	if node, ok := mappingValue(root, "technical_context"); ok {
		declarations.TechnicalContext, declarations.TechnicalContextErr = decodeTechnicalContext(node, displacement)
	}
	if node, ok := mappingValue(root, "git"); ok {
		declarations.Git, declarations.GitErr = decodeGit(node)
	}
	if node, ok := mappingValue(root, "workflow"); ok {
		declarations.Workflow, declarations.WorkflowErr = decodeWorkflow(node)
	}
	if node, ok := mappingValue(root, "sdd"); ok {
		declarations.SDD, declarations.SDDErr = decodeSDD(node)
		if declarations.SDDErr == nil && declarations.WorkflowErr == nil {
			if err := checkISOWorkflow(declarations.SDD, declarations.Workflow); err != nil {
				declarations.SDD, declarations.SDDErr = nil, err
			}
		}
	}
	if node, ok := mappingValue(root, "repositories"); ok {
		declarations.Repositories, declarations.RepositoriesErr = decodeRepositories(node)
	}
	return declarations
}

func decodeTechnicalContext(node *yaml.Node, displacement string) (*TechnicalContextConfig, error) {
	node = resolveAlias(node)
	if isNull(node) {
		return nil, errors.New("el bloque 'technical_context' requiere al menos el modo 'maintained' o 'reference'")
	}
	if node.Kind != yaml.MappingNode {
		return nil, errors.New("el bloque 'technical_context' debe ser un mapeo con los modos 'maintained' y 'reference'")
	}
	var errs []error
	errs = append(errs, unknownKeys(node, "technical_context", technicalCtxKeys)...)

	modes := [len(contextModeLabels)]*ContextMode{}
	cleaned := [len(contextModeLabels)][]contextPath{}
	for i, key := range contextModeLabels {
		modeNode, present := mappingValue(node, key)
		if !present {
			continue
		}
		label := "technical_context." + key
		modeNode = resolveAlias(modeNode)
		if isNull(modeNode) || modeNode.Kind != yaml.MappingNode {
			errs = append(errs, fmt.Errorf("el modo '%s' debe ser un mapeo con 'paths' y, opcionalmente, 'entry'", label))
			continue
		}
		errs = append(errs, unknownKeys(modeNode, label, contextModeKeys)...)
		var mode ContextMode
		if err := modeNode.Decode(&mode); err != nil {
			errs = append(errs, fmt.Errorf("el modo '%s' no se pudo leer: %w", label, err))
			continue
		}
		modes[i] = &mode
		var modeErrs []error
		cleaned[i], modeErrs = validateContextMode(label, mode, displacement)
		errs = append(errs, modeErrs...)
	}
	if modes[0] == nil && modes[1] == nil && len(errs) == 0 {
		errs = append(errs, errors.New("el bloque 'technical_context' requiere al menos el modo 'maintained' o 'reference'"))
	}

	errs = append(errs, contextOverlaps(cleaned)...)
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &TechnicalContextConfig{Maintained: modes[0], Reference: modes[1]}, nil
}

// contextPath es una ruta de contexto normalizada, con el modo al que pertenece.
type contextPath struct {
	mode  string
	raw   string
	clean string
}

// validateContextMode comprueba `paths` y `entry` de un modo (REQ-24.26 y 24.27).
func validateContextMode(label string, mode ContextMode, displacement string) ([]contextPath, []error) {
	var errs []error
	var paths []contextPath
	if len(mode.Paths) == 0 {
		errs = append(errs, fmt.Errorf("el modo '%s' requiere al menos una ruta en 'paths'", label))
	}
	for _, raw := range mode.Paths {
		clean, err := checkContextPath(raw, displacement)
		if err != nil {
			errs = append(errs, fmt.Errorf("la ruta '%s' de '%s.paths' no es válida: %w", raw, label, err))
			continue
		}
		paths = append(paths, contextPath{mode: label, raw: raw, clean: clean})
	}
	if strings.TrimSpace(mode.Entry) == "" {
		return paths, errs
	}
	entry, err := checkContextPath(mode.Entry, displacement)
	if err != nil {
		return paths, append(errs, fmt.Errorf("la ruta '%s' de '%s.entry' no es válida: %w", mode.Entry, label, err))
	}
	for _, p := range paths {
		if lexicalOverlapContains(strings.ToLower(p.clean), strings.ToLower(entry)) {
			return paths, errs
		}
	}
	return paths, append(errs, fmt.Errorf("'%s.entry' ('%s') debe estar contenida en alguna de las rutas de '%s.paths'", label, mode.Entry, label))
}

// contextOverlaps reporta los solapes entre los dos modos. Dos rutas solapan si son iguales o una contiene a la otra; se
// comparan sin distinguir mayúsculas porque axiom.yaml se comparte entre Windows
// y Linux (la comprobación por identidad la hace workspace validate).
func contextOverlaps(modes [len(contextModeLabels)][]contextPath) []error {
	var errs []error
	for _, maintained := range modes[0] {
		for _, reference := range modes[1] {
			if lexicalOverlap(strings.ToLower(maintained.clean), strings.ToLower(reference.clean)) {
				errs = append(errs, fmt.Errorf("la ruta '%s' (%s) solapa con '%s' (%s): una ruta no puede pertenecer a los dos modos del contexto técnico, y dos rutas solapan si son iguales o una contiene a la otra", maintained.raw, maintained.mode, reference.raw, reference.mode))
			}
		}
	}
	return errs
}

// checkRelative normaliza una ruta de axiom.yaml a la forma con «/» y rechaza la
// que no es relativa. El criterio es el de CleanSkillRoot e idéntico en todos los
// sistemas: axiom.yaml se versiona y lo comparten clones en Windows y Linux, así
// que «C:\x» o «/opt/x» se rechazan también donde no serían absolutas.
func checkRelative(raw string) (string, error) {
	return checkRelativeTo(raw, "workspaceRoot")
}

// checkRelativeTo es checkRelative para una ruta que se declara relativa a otro
// ancla, que el error nombra (por ejemplo, «la carpeta maestra»).
func checkRelativeTo(raw, anchor string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("la ruta está vacía")
	}
	slashed := strings.ReplaceAll(trimmed, `\`, "/")
	if strings.HasPrefix(slashed, "/") || hasDriveLetter(slashed) || filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("debe ser relativa a %s, sin unidad de disco ni barra inicial", anchor)
	}
	return path.Clean(slashed), nil
}

func escapes(clean string) bool {
	return clean == ".." || strings.HasPrefix(clean, "../")
}

// checkContextPath valida una ruta de contexto: relativa a workspaceRoot y
// contenida léxicamente en la carpeta maestra, a la que se llega desde
// workspaceRoot por displacement.
func checkContextPath(raw, displacement string) (string, error) {
	clean, err := checkRelative(raw)
	if err != nil {
		return "", err
	}
	if escapes(path.Join(displacement, clean)) {
		return "", errors.New("sale de la carpeta maestra: una ruta de contexto técnico debe permanecer dentro de ella")
	}
	return clean, nil
}

// lexicalOverlap dice si dos rutas limpias solapan: iguales o una contiene a la otra.
func lexicalOverlap(a, b string) bool {
	return lexicalOverlapContains(a, b) || lexicalOverlapContains(b, a)
}

// lexicalOverlapContains dice si outer es igual a inner o lo contiene.
func lexicalOverlapContains(outer, inner string) bool {
	return outer == inner || outer == "." || strings.HasPrefix(inner, outer+"/")
}

// lexicalDisplacement calcula, solo a partir de la configuración, el
// desplazamiento léxico de la carpeta maestra a workspaceRoot: vacío en
// embebido; specs_repository, o su directorio padre si termina en openspec, en
// las demás topologías. Si la topología no es legible, o specs_repository es
// absoluta y por tanto no tiene desplazamiento léxico, es vacío (la contención
// por identidad que hace workspace validate cubre ese caso).
func lexicalDisplacement(cfg *WorkspaceConfig) string {
	if !separatesSpecs(cfg) {
		return ""
	}
	raw := strings.TrimSpace(cfg.Workspace.SpecsRepository)
	if filepath.IsAbs(raw) || hasDriveLetter(strings.ReplaceAll(raw, `\`, "/")) {
		return ""
	}
	segments := specsSegments(raw)
	if len(segments) == 0 {
		return ""
	}
	repo := path.Join(segments...)
	if sameSegment(path.Base(repo), planningHomeDirName) {
		repo = path.Dir(repo)
	}
	if repo == "." {
		return ""
	}
	return repo
}

// unknownKeys reporta cada clave de un mapeo que no está en allowed, con su
// nombre, su línea y las claves admitidas.
func unknownKeys(node *yaml.Node, label string, allowed []string) []error {
	var errs []error
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if !containsString(allowed, key.Value) {
			errs = append(errs, fmt.Errorf("clave desconocida '%s' en '%s' (línea %d); claves admitidas: %s", key.Value, label, key.Line, strings.Join(allowed, ", ")))
		}
	}
	return errs
}

// mappingValue busca la clave de primer nivel de un mapeo y devuelve su valor.
func mappingValue(mapping *yaml.Node, key string) (*yaml.Node, bool) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1], true
		}
	}
	return nil, false
}

func resolveAlias(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}
	return node
}

func isNull(node *yaml.Node) bool {
	return node == nil || (node.Kind == yaml.ScalarNode && node.Tag == "!!null")
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// mappingKeyLine devuelve la línea de una clave de primer nivel de un mapeo.
func mappingKeyLine(mapping *yaml.Node, key string) (int, bool) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i].Line, true
		}
	}
	return 0, false
}
