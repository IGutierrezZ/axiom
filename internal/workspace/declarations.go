package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// IntegrationMode es la forma en que archive integra un documento de la baseline.
type IntegrationMode string

// ScenariosMode indica si los escenarios de la delta spec pasan al documento.
type ScenariosMode string

// IdentityKind es la identidad de un requisito dentro de su documento.
type IdentityKind string

const (
	// IntegrationCompose es el modo derivado de un documento con perfil de
	// requisito: no se declara, se deduce de que existe `requirement`.
	IntegrationCompose IntegrationMode = "compose"
	// IntegrationDirected es el modo por defecto de un documento de prosa.
	IntegrationDirected IntegrationMode = "directed"
	// IntegrationRegister solo registra la integración pendiente.
	IntegrationRegister IntegrationMode = "register"

	// ScenariosInclude integra descripción y escenarios (valor por defecto).
	ScenariosInclude ScenariosMode = "include"
	// ScenariosOmit integra solo la descripción.
	ScenariosOmit ScenariosMode = "omit"

	// IdentityID identifica el requisito por el grupo `id` (valor por defecto).
	IdentityID IdentityKind = "id"
	// IdentityText lo identifica por el texto del título.
	IdentityText IdentityKind = "text"
)

// BaselineConfig es el bloque `baseline:` de axiom.yaml (REQ-24.13).
type BaselineConfig struct {
	Documents []BaselineDocument `yaml:"documents"`
}

// BaselineDocument declara un documento de la baseline del proyecto (REQ-24.14).
type BaselineDocument struct {
	Domain      string              `yaml:"domain"`
	Path        string              `yaml:"path"`
	Requirement *RequirementProfile `yaml:"requirement,omitempty"`
	Integration IntegrationMode     `yaml:"integration,omitempty"` // directed | register
	Scenarios   ScenariosMode       `yaml:"scenarios,omitempty"`   // include | omit
}

// RequirementProfile hace direccionable por requisito un documento (REQ-24.15).
type RequirementProfile struct {
	Heading  string       `yaml:"heading"`
	Level    int          `yaml:"level"`
	Identity IdentityKind `yaml:"identity,omitempty"` // id | text
	Render   string       `yaml:"render"`
}

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
	ParseErr            error // no nulo solo si existe axiom.yaml y no se puede leer como YAML
	Baseline            *BaselineConfig
	BaselineErr         error
	TechnicalContext    *TechnicalContextConfig
	TechnicalContextErr error
}

// EffectiveIntegration devuelve el modo de integración efectivo del documento:
// `compose` si tiene perfil de requisito, y `directed` o `register` si es de
// prosa (`directed` por defecto).
func (d BaselineDocument) EffectiveIntegration() IntegrationMode {
	if d.Requirement != nil {
		return IntegrationCompose
	}
	if d.Integration == "" {
		return IntegrationDirected
	}
	return d.Integration
}

// EffectiveScenarios devuelve el tratamiento efectivo de los escenarios (`include` por defecto).
func (d BaselineDocument) EffectiveScenarios() ScenariosMode {
	if d.Scenarios == "" {
		return ScenariosInclude
	}
	return d.Scenarios
}

// EffectiveIdentity devuelve la identidad efectiva del requisito (`id` por defecto).
func (p RequirementProfile) EffectiveIdentity() IdentityKind {
	if p.Identity == "" {
		return IdentityID
	}
	return p.Identity
}

// Compile compila el patrón `heading` (RE2) y comprueba que define los grupos
// con nombre `id` y `title`. El error no nombra el dominio: lo añade quien llama.
func (p RequirementProfile) Compile() (*regexp.Regexp, error) {
	re, err := regexp.Compile(p.Heading)
	if err != nil {
		return nil, fmt.Errorf("el patrón 'heading' no compila: %w", err)
	}
	var hasID, hasTitle bool
	for _, name := range re.SubexpNames() {
		switch name {
		case "id":
			hasID = true
		case "title":
			hasTitle = true
		}
	}
	if !hasID || !hasTitle {
		return nil, fmt.Errorf("el patrón 'heading' debe definir los grupos con nombre 'id' y 'title' (por ejemplo (?P<id>...) y (?P<title>...))")
	}
	return re, nil
}

// ReadDeclarations lee configPath y aplica la carga por capas; devuelve solo las
// declaraciones (las usa axiom init para el axiom.yaml propio del destino). Un
// fichero inexistente equivale a «sin declaraciones»; uno que existe y no se
// puede leer deja la causa en ParseErr.
func ReadDeclarations(configPath string) Declarations {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Declarations{}
		}
		return Declarations{ParseErr: fmt.Errorf("no se pudo leer el archivo '%s': %w", configPath, err)}
	}
	_, _, declarations := parseLayered(data)
	return declarations
}

// Claves admitidas por nodo (D-04). yaml.v3 no ofrece KnownFields en
// Node.Decode, así que cada mapeo de los bloques nuevos se recorre contra su
// lista cerrada; el resto de claves de axiom.yaml se sigue ignorando (REQ-24.17).
var (
	baselineKeys      = []string{"documents"}
	documentKeys      = []string{"domain", "path", "requirement", "integration", "scenarios"}
	requirementKeys   = []string{"heading", "level", "identity", "render"}
	technicalCtxKeys  = []string{"maintained", "reference"}
	contextModeKeys   = []string{"entry", "paths"}
	domainPattern     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	contextModeLabels = [...]string{"maintained", "reference"}
)

// maxScenarioLevel es el nivel de requisito más alto compatible con incluir los
// escenarios: los de las delta specs son `#### Scenario:` (nivel 4) y con un
// nivel de requisito igual o mayor cerrarían el bloque (REQ-24.60).
const maxScenarioLevel = 3

// readDeclarationBlocks localiza los bloques `baseline` y `technical_context` en
// el mapeo raíz del documento y los decodifica y valida de forma estricta.
// displacement es el desplazamiento léxico de la carpeta maestra a workspaceRoot.
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
	var baselinePaths []baselinePath
	if node, ok := mappingValue(root, "baseline"); ok {
		declarations.Baseline, baselinePaths, declarations.BaselineErr = decodeBaseline(node, displacement)
	}
	if node, ok := mappingValue(root, "technical_context"); ok {
		declarations.TechnicalContext, declarations.TechnicalContextErr = decodeTechnicalContext(node, displacement, baselinePaths)
	}
	return declarations
}

// baselinePath es la ruta normalizada de un documento de la baseline, para
// detectar solapes con el contexto técnico.
type baselinePath struct {
	domain string
	raw    string
	clean  string
}

func decodeBaseline(node *yaml.Node, displacement string) (*BaselineConfig, []baselinePath, error) {
	node = resolveAlias(node)
	if isNull(node) {
		return nil, nil, errors.New("la baseline declarada requiere al menos un documento en 'baseline.documents'")
	}
	if node.Kind != yaml.MappingNode {
		return nil, nil, errors.New("el bloque 'baseline' debe ser un mapeo con la lista 'documents'")
	}
	var errs []error
	errs = append(errs, unknownKeys(node, "baseline", baselineKeys)...)

	documentsNode, ok := mappingValue(node, "documents")
	documentsNode = resolveAlias(documentsNode)
	switch {
	case !ok || isNull(documentsNode) || (documentsNode.Kind == yaml.SequenceNode && len(documentsNode.Content) == 0):
		errs = append(errs, errors.New("la baseline declarada requiere al menos un documento en 'baseline.documents'"))
		return nil, nil, errors.Join(errs...)
	case documentsNode.Kind != yaml.SequenceNode:
		errs = append(errs, errors.New("'baseline.documents' debe ser una lista de documentos"))
		return nil, nil, errors.Join(errs...)
	}

	documents := make([]BaselineDocument, len(documentsNode.Content))
	decoded := make([]bool, len(documentsNode.Content))
	for i, item := range documentsNode.Content {
		item = resolveAlias(item)
		label := fmt.Sprintf("baseline.documents[%d]", i)
		if item.Kind != yaml.MappingNode {
			errs = append(errs, fmt.Errorf("%s debe ser un mapeo con 'domain' y 'path'", label))
			continue
		}
		errs = append(errs, unknownKeys(item, label, documentKeys)...)
		if requirementNode, present := mappingValue(item, "requirement"); present {
			requirementNode = resolveAlias(requirementNode)
			switch {
			case isNull(requirementNode):
				errs = append(errs, fmt.Errorf("%s: 'requirement' debe ser un mapeo con 'heading', 'level' y 'render'", label))
				continue
			case requirementNode.Kind == yaml.MappingNode:
				errs = append(errs, unknownKeys(requirementNode, label+".requirement", requirementKeys)...)
			}
		}
		if err := item.Decode(&documents[i]); err != nil {
			errs = append(errs, fmt.Errorf("%s no se pudo leer: %w", label, err))
			continue
		}
		decoded[i] = true
	}

	documentErrs, paths := validateDocuments(documents, decoded, displacement)
	errs = append(errs, documentErrs...)
	if err := errors.Join(errs...); err != nil {
		return nil, paths, err
	}
	return &BaselineConfig{Documents: documents}, paths, nil
}

// validateDocuments aplica las reglas sintácticas de REQ-24.14 a 24.16 y 24.60 a
// cada documento ya decodificado, sin acceder al disco.
func validateDocuments(documents []BaselineDocument, decoded []bool, displacement string) ([]error, []baselinePath) {
	var errs []error
	var paths []baselinePath
	domains := map[string]int{}
	pathDomains := map[string]string{}
	for i, doc := range documents {
		if !decoded[i] {
			continue
		}
		label := fmt.Sprintf("baseline.documents[%d]", i)
		name := label
		if doc.Domain != "" {
			name = fmt.Sprintf("el dominio '%s'", doc.Domain)
		}

		switch {
		case strings.TrimSpace(doc.Domain) == "":
			errs = append(errs, fmt.Errorf("%s no declara 'domain'", label))
		case !domainPattern.MatchString(doc.Domain):
			errs = append(errs, fmt.Errorf("el dominio '%s' (%s) no es válido: debe usar solo minúsculas, dígitos y guiones (formato %s)", doc.Domain, label, domainPattern.String()))
		default:
			if first, seen := domains[doc.Domain]; seen {
				errs = append(errs, fmt.Errorf("el dominio '%s' está duplicado en 'baseline.documents' (entradas %d y %d): cada dominio mapea a un único documento", doc.Domain, first, i))
			} else {
				domains[doc.Domain] = i
			}
		}

		if clean, err := checkDocumentPath(doc.Path); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		} else {
			key := strings.ToLower(clean)
			if other, seen := pathDomains[key]; seen {
				errs = append(errs, fmt.Errorf("la ruta '%s' se declara en los dominios '%s' y '%s': cada documento mapea a un único dominio", doc.Path, other, doc.Domain))
			} else {
				pathDomains[key] = doc.Domain
			}
			paths = append(paths, baselinePath{domain: doc.Domain, raw: doc.Path, clean: clean})
		}

		errs = append(errs, validateDocumentProfile(doc, name)...)
	}
	return errs, paths
}

// validateDocumentProfile valida `requirement`, `identity`, `integration` y
// `scenarios` de un documento (REQ-24.15, 24.16 y 24.60).
func validateDocumentProfile(doc BaselineDocument, name string) []error {
	var errs []error
	if doc.Requirement == nil {
		switch doc.Integration {
		case "", IntegrationDirected, IntegrationRegister:
		default:
			errs = append(errs, fmt.Errorf("%s: 'integration' debe ser 'directed' o 'register', no '%s'", name, doc.Integration))
		}
		if doc.Scenarios != "" {
			errs = append(errs, fmt.Errorf("%s: 'scenarios' solo se admite en documentos con perfil de requisito ('requirement')", name))
		}
		return errs
	}

	profile := doc.Requirement
	if doc.Integration != "" {
		errs = append(errs, fmt.Errorf("%s: 'integration' no se admite en un documento con perfil de requisito, cuya integración es la composición", name))
	}
	if strings.TrimSpace(profile.Heading) == "" {
		errs = append(errs, fmt.Errorf("%s: 'requirement.heading' es obligatorio", name))
	} else if _, err := profile.Compile(); err != nil {
		errs = append(errs, fmt.Errorf("%s: %w", name, err))
	}
	levelValid := profile.Level >= 1 && profile.Level <= 6
	if !levelValid {
		errs = append(errs, fmt.Errorf("%s: 'requirement.level' debe ser un entero entre 1 y 6, no %d", name, profile.Level))
	}
	switch profile.Identity {
	case "", IdentityID, IdentityText:
	default:
		errs = append(errs, fmt.Errorf("%s: 'requirement.identity' debe ser 'id' o 'text', no '%s'", name, profile.Identity))
	}
	errs = append(errs, validateRender(profile, levelValid, name)...)

	switch doc.Scenarios {
	case "", ScenariosInclude, ScenariosOmit:
	default:
		errs = append(errs, fmt.Errorf("%s: 'scenarios' debe ser 'include' u 'omit', no '%s'", name, doc.Scenarios))
	}
	if doc.EffectiveScenarios() == ScenariosInclude && levelValid && profile.Level > maxScenarioLevel {
		errs = append(errs, fmt.Errorf("%s: 'requirement.level' es %d y los escenarios se incluyen (valor por defecto o 'scenarios: include'): los encabezados '#### Scenario:' de las delta specs cerrarían el bloque del requisito de nivel %d. Declara 'scenarios: omit' o un 'level' menor o igual que %d", name, profile.Level, profile.Level, maxScenarioLevel))
	}
	return errs
}

func validateRender(profile *RequirementProfile, levelValid bool, name string) []error {
	var errs []error
	if strings.TrimSpace(profile.Render) == "" {
		return []error{fmt.Errorf("%s: 'requirement.render' es obligatorio", name)}
	}
	if levelValid && !strings.HasPrefix(profile.Render, strings.Repeat("#", profile.Level)+" ") {
		errs = append(errs, fmt.Errorf("%s: 'requirement.render' debe comenzar con exactamente %d signos '#' seguidos de un espacio (el 'level' declarado)", name, profile.Level))
	}
	if !strings.Contains(profile.Render, "{id}") || !strings.Contains(profile.Render, "{title}") {
		errs = append(errs, fmt.Errorf("%s: 'requirement.render' debe contener los marcadores {id} y {title}", name))
	}
	return errs
}

func decodeTechnicalContext(node *yaml.Node, displacement string, baseline []baselinePath) (*TechnicalContextConfig, error) {
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

	errs = append(errs, contextOverlaps(cleaned, baseline)...)
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

// contextOverlaps reporta los solapes entre los dos modos y con los documentos de
// la baseline. Dos rutas solapan si son iguales o una contiene a la otra; se
// comparan sin distinguir mayúsculas porque axiom.yaml se comparte entre Windows
// y Linux (la comprobación por identidad la hace workspace validate).
func contextOverlaps(modes [len(contextModeLabels)][]contextPath, baseline []baselinePath) []error {
	var errs []error
	for _, maintained := range modes[0] {
		for _, reference := range modes[1] {
			if lexicalOverlap(strings.ToLower(maintained.clean), strings.ToLower(reference.clean)) {
				errs = append(errs, fmt.Errorf("la ruta '%s' (%s) solapa con '%s' (%s): una ruta no puede pertenecer a los dos modos del contexto técnico, y dos rutas solapan si son iguales o una contiene a la otra", maintained.raw, maintained.mode, reference.raw, reference.mode))
			}
		}
	}
	for _, paths := range modes {
		for _, contextRoute := range paths {
			for _, document := range baseline {
				if lexicalOverlap(strings.ToLower(contextRoute.clean), strings.ToLower(document.clean)) {
					errs = append(errs, fmt.Errorf("la ruta '%s' del modo '%s' solapa con el documento '%s' del dominio '%s' de 'baseline.documents': una ruta no puede pertenecer a la baseline y al contexto técnico", contextRoute.raw, contextRoute.mode, document.raw, document.domain))
				}
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
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("la ruta está vacía")
	}
	slashed := strings.ReplaceAll(trimmed, `\`, "/")
	if strings.HasPrefix(slashed, "/") || hasDriveLetter(slashed) || filepath.IsAbs(trimmed) {
		return "", errors.New("debe ser relativa a workspaceRoot, sin unidad de disco ni barra inicial")
	}
	return path.Clean(slashed), nil
}

func escapes(clean string) bool {
	return clean == ".." || strings.HasPrefix(clean, "../")
}

// checkDocumentPath valida la ruta de un documento: relativa y contenida
// léxicamente en workspaceRoot. Que exista como fichero regular lo comprueba
// workspace validate (ValidateDeclarationPaths).
func checkDocumentPath(raw string) (string, error) {
	clean, err := checkRelative(raw)
	if err != nil {
		return "", fmt.Errorf("la ruta '%s' no es válida: %w", raw, err)
	}
	if escapes(clean) {
		return "", fmt.Errorf("la ruta '%s' sale de workspaceRoot: un documento de la baseline debe permanecer dentro del repositorio de especificaciones", raw)
	}
	if clean == "." {
		return "", fmt.Errorf("la ruta '%s' es el propio workspaceRoot: debe apuntar a un fichero", raw)
	}
	return clean, nil
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
