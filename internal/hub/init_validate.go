package hub

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// InitValidationError es el rechazo de Init y de ValidateInit cuando la
// configuración que escribiría no es válida. Lleva todas las violaciones, no solo
// la primera, para que quien llama (la API del dashboard, axiom init) las muestre
// juntas. Cuando se devuelve, no se ha escrito nada en disco.
type InitValidationError struct {
	Violations []string
}

// Error lista las violaciones, una por línea.
func (e *InitValidationError) Error() string {
	var sb strings.Builder
	sb.WriteString("la configuración del proyecto no es válida:")
	for _, v := range e.Violations {
		sb.WriteString("\n- ")
		sb.WriteString(v)
	}
	return sb.String()
}

// configPlan es el axiom.yaml que Init va a escribir, antes de escribirlo.
type configPlan struct {
	Name          string
	Topology      string
	SpecsRepo     string
	Root          string
	Layout        specsLayout
	DomainContext string
	Roles         []RoleInput
	AdoptedSkills int
}

// render genera el contenido del axiom.yaml del plan.
func (p configPlan) render() string {
	return buildAxiomYamlWithRoles(p.Name, p.Topology, p.SpecsRepo, p.Root, p.DomainContext, p.Roles)
}

// ValidateInit comprueba, sin escribir nada, que la configuración que Init
// generaría para opts es válida: la pasa por workspace.ParseConfig y por
// workspace.Validate (con las carpetas que Init crea dadas por existentes) y
// aplica las reglas de roles y repositorios. Devuelve un *InitValidationError con
// todas las violaciones, o nil. Si Init conservaría el axiom.yaml existente, no hay
// configuración generada que validar y devuelve nil.
func ValidateInit(opts InitOptions) error {
	return NewInitializer(nil, nil).ValidateInit(opts)
}

// ValidateInit es ValidateInit con el detector del inicializador.
func (ini *Initializer) ValidateInit(opts InitOptions) error {
	if opts.Path == "" {
		opts.Path = "."
	}
	absPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return fmt.Errorf("ruta de inicialización inválida: %w", err)
	}
	if opts.Name == "" {
		opts.Name = filepath.Base(absPath)
	}
	if opts.Topology == "" {
		opts.Topology = "monorepo-embedded"
	}
	markers, err := inspectOwnMarkers(absPath, layoutFor(opts).home(absPath))
	if err != nil {
		return err
	}
	_, err = ini.validateAll(absPath, opts, markers)
	return err
}

// validateAll reúne las violaciones del alta: las de la disposición de las
// especificaciones (siempre) y, si Init va a generar un axiom.yaml, las de ese
// axiom.yaml. Devuelve el plan generado (vacío si no se genera ninguno).
func (ini *Initializer) validateAll(absPath string, opts InitOptions, markers ownMarkers) (configPlan, error) {
	violations := layoutViolations(opts, markers)
	var plan configPlan
	if willWriteConfig(markers, opts) {
		plan = ini.planConfig(absPath, opts)
		violations = append(violations, validatePlan(absPath, plan)...)
	}
	if len(violations) > 0 {
		return plan, &InitValidationError{Violations: violations}
	}
	return plan, nil
}

// specsLayout es la disposición de las especificaciones de un alta multirepo con
// repositorio de specs propio: el axiom.yaml en ese repositorio, el planning home
// en <repositorio>/openspec y el puntero en la carpeta maestra. El valor cero es
// la disposición de siempre (todo en la carpeta maestra, specs en openspec/).
type specsLayout struct {
	separate bool
	rel      string // specs_repository tal como se escribe, con "/"
}

// layoutFor decide la disposición de opts. Necesita la topología ya resuelta. Con
// los perfiles knowledge y spec-only, o con "openspec" (que ya es la disposición de
// siempre), no hay repositorio de specs propio.
func layoutFor(opts InitOptions) specsLayout {
	spec := strings.TrimSpace(opts.SpecsRepository)
	if spec == "" || opts.Topology != "multirepo" || opts.Profile == "knowledge" || opts.Profile == "spec-only" {
		return specsLayout{}
	}
	rel := filepath.ToSlash(filepath.Clean(spec))
	if rel == "openspec" {
		return specsLayout{}
	}
	return specsLayout{separate: true, rel: rel}
}

// valid dice si hay repositorio de specs propio y su ruta es admisible.
func (l specsLayout) valid() bool {
	return l.separate && l.rel != "." && repositoryPathProblem(l.rel) == ""
}

// home es el planning home del alta en absPath.
func (l specsLayout) home(absPath string) string {
	if !l.valid() {
		return filepath.Join(absPath, "openspec")
	}
	specs := filepath.Join(absPath, filepath.FromSlash(l.rel))
	if strings.EqualFold(filepath.Base(specs), "openspec") {
		return specs
	}
	return filepath.Join(specs, "openspec")
}

// layoutViolations son las reglas de SpecsRepository que no dependen del
// axiom.yaml que se genere: solo se admite en multirepo y no puede añadirse a un
// axiom.yaml directo que ya existe en la carpeta maestra (el puntero lo
// ocultaría).
func layoutViolations(opts InitOptions, markers ownMarkers) []string {
	spec := strings.TrimSpace(opts.SpecsRepository)
	if spec == "" {
		return nil
	}
	if opts.Topology != "multirepo" {
		return []string{"el repositorio de especificaciones propio solo se admite en la topología multirepo"}
	}
	if layoutFor(opts).separate && markers.configPath != "" && !markers.pointer {
		return []string{fmt.Sprintf("la carpeta ya tiene un axiom.yaml propio (%s): el repositorio de especificaciones solo se puede indicar en un alta nueva", filepath.Base(markers.configPath))}
	}
	return nil
}

// willWriteConfig dice si Init generará un axiom.yaml nuevo: no hay uno, o se
// fuerza y el existente no declara nada que la regeneración perdería.
func willWriteConfig(markers ownMarkers, opts InitOptions) bool {
	alreadyExisted := markers.configPath != ""
	preserveConfig := alreadyExisted && opts.Force && markers.declares()
	return !alreadyExisted || (opts.Force && !preserveConfig)
}

// planConfig decide los roles, la topología y el repositorio de specs del
// axiom.yaml que Init generaría para absPath. Solo lee del disco.
func (ini *Initializer) planConfig(absPath string, opts InitOptions) configPlan {
	tech, err := ini.detector.Detect(absPath)
	if err != nil {
		tech = &TechDetection{
			PrimaryLanguage:  "generic",
			RecommendedRoles: map[string][]string{"core": {"generic"}, "qa": {"verification"}},
		}
	}

	var effectiveRoles []RoleInput
	if opts.Profile == "knowledge" || opts.Profile == "spec-only" {
		opts.Topology = "multirepo"
		effectiveRoles = []RoleInput{
			{
				Key:          "knowledge",
				Name:         "Knowledge Explorer",
				Repositories: []string{"."},
				NonBlocking:  true,
				Tech:         []string{"markdown", "openspec"},
			},
		}
	} else if len(opts.Roles) > 0 {
		effectiveRoles = opts.Roles
	} else if len(tech.ConfiguredRoles) > 0 {
		effectiveRoles = tech.ConfiguredRoles
	} else {
		// Regla de negocio: si no se proveen ni detectan roles, rol único 'fullstack'.
		primaryTech := tech.Frameworks
		if len(primaryTech) == 0 && tech.PrimaryLanguage != "" {
			primaryTech = []string{tech.PrimaryLanguage}
		}
		effectiveRoles = []RoleInput{
			{
				Key:          "fullstack",
				Name:         opts.Name + " Fullstack",
				Repositories: []string{"."},
				NonBlocking:  false,
				Tech:         primaryTech,
			},
		}
	}

	specsRepo := "openspec"
	if tech.SpecsRepository != "" {
		specsRepo = tech.SpecsRepository
	}
	root := "."
	layout := layoutFor(opts)
	if layout.separate {
		specsRepo, root = layout.rel, ""
	}
	return configPlan{
		Name:          opts.Name,
		Topology:      opts.Topology,
		SpecsRepo:     specsRepo,
		Root:          root,
		Layout:        layout,
		DomainContext: tech.DomainContext,
		Roles:         effectiveRoles,
		AdoptedSkills: len(tech.AdoptedSkills),
	}
}

// validatePlan reúne las violaciones del plan. Primero las reglas propias de los
// roles; después, sobre una copia depurada de esos roles (sin claves repetidas ni
// repositorios inadmisibles, que ya se han informado), la validación estructural
// de workspace.ParseConfig y la de workspace.Validate sobre un sistema de
// ficheros que da por creado lo que Init crea.
func validatePlan(absPath string, plan configPlan) []string {
	cleanRoles, violations := sanitizeRoles(plan.Roles)

	cleaned := plan
	cleaned.Roles = cleanRoles
	if plan.Layout.separate {
		if problem := specsPathProblem(plan.Layout.rel); problem != "" {
			violations = append(violations, fmt.Sprintf("el repositorio de especificaciones %q %s", plan.Layout.rel, problem))
			// Se sigue validando con la disposición de siempre para no repetir el error.
			cleaned.SpecsRepo, cleaned.Root, cleaned.Layout = "openspec", ".", specsLayout{}
		}
	}
	cfg, err := workspace.ParseConfig([]byte(cleaned.render()))
	if err != nil {
		violations = append(violations, loadViolations(err)...)
		return violations
	}

	violations = append(violations, policyViolations(cfg)...)

	// Los repositorios de los roles se declaran, no se crean: el alta no exige que
	// existan todavía (un proyecto nuevo puede declarar roles sobre carpetas que
	// llegarán después, y la adopción de una configuración previa los trae tal
	// cual). Lo que sí se exige es que sean relativos y estén dentro del proyecto.
	fsys := newPlannedFS(absPath, cleaned.Layout)
	for _, role := range cfg.Roles {
		for _, repo := range role.Repositories {
			fsys.planned = append(fsys.planned, filepath.Join(absPath, filepath.FromSlash(strings.TrimSpace(repo.Path))))
		}
	}

	report, err := workspace.Validate(fsys, cfg, absPath)
	if err != nil {
		violations = append(violations, err.Error())
	} else if report != nil {
		violations = append(violations, report.Errors...)
	}
	return violations
}

// effectiveRoleKey es la clave con la que buildAxiomYamlWithRoles escribe el rol.
func effectiveRoleKey(r RoleInput) string {
	key := strings.TrimSpace(r.Key)
	if key == "" {
		key = slugify(r.Name)
	}
	return key
}

// sanitizeRoles aplica las reglas de roles y repositorios y devuelve los roles
// depurados para seguir validando, junto con las violaciones encontradas:
//   - clave vacía, o repetida (también salvo mayúsculas: multirole compara sin
//     distinguirlas);
//   - repositorios con ruta absoluta o que salen del proyecto.
//
// De cada clave repetida se conserva el primer rol; cada repositorio inadmisible
// pasa a "." en la copia depurada para no arrastrar un segundo error por lo mismo.
func sanitizeRoles(roles []RoleInput) ([]RoleInput, []string) {
	var violations []string
	seen := make(map[string]string)
	clean := make([]RoleInput, 0, len(roles))
	for _, role := range roles {
		key := effectiveRoleKey(role)
		if key == "" {
			violations = append(violations, "hay un rol sin clave ni nombre: indica al menos uno de los dos")
			continue
		}
		folded := strings.ToLower(key)
		if first, dup := seen[folded]; dup {
			if first == key {
				violations = append(violations, fmt.Sprintf("la clave de rol %q está repetida", key))
			} else {
				violations = append(violations, fmt.Sprintf("las claves de rol %q y %q solo se distinguen por las mayúsculas", first, key))
			}
			continue
		}
		seen[folded] = key

		role.Repositories = append([]string(nil), role.Repositories...)
		for i, repo := range role.Repositories {
			if reason := repositoryPathProblem(repo); reason != "" {
				violations = append(violations, fmt.Sprintf("el repositorio %q del rol %q %s", strings.TrimSpace(repo), key, reason))
				role.Repositories[i] = "."
			}
		}
		clean = append(clean, role)
	}
	return clean, violations
}

// repositoryPathProblem explica por qué la ruta de repositorio de un rol no
// es admisible, o devuelve "" si lo es. La ruta debe ser relativa y quedar dentro
// del proyecto. La ruta vacía la rechaza workspace.ParseConfig.
func repositoryPathProblem(repo string) string {
	trimmed := strings.TrimSpace(repo)
	if trimmed == "" {
		return ""
	}
	slashed := filepath.ToSlash(trimmed)
	if filepath.IsAbs(trimmed) || strings.HasPrefix(slashed, "/") || filepath.VolumeName(trimmed) != "" {
		return "es una ruta absoluta: debe ser relativa a la carpeta del proyecto"
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "sale de la carpeta del proyecto: debe quedar dentro de ella"
	}
	return ""
}

// policyViolations comprueba el gate_policy de cada rol de cfg contra el
// vocabulario de multirole (blocking, deferred y optional). Un rol sin política
// declarada es bloqueante.
func policyViolations(cfg *workspace.WorkspaceConfig) []string {
	var violations []string
	for key, role := range cfg.Roles {
		policy := multirole.GatePolicy(strings.ToLower(strings.TrimSpace(role.GatePolicy)))
		switch policy {
		case "", multirole.PolicyBlocking, multirole.PolicyDeferred, multirole.PolicyOptional:
		default:
			violations = append(violations, fmt.Sprintf("el gate_policy %q del rol %q no es válido (admitidos: blocking, deferred, optional)", role.GatePolicy, key))
		}
	}
	return violations
}

// plannedFS es el sistema de ficheros con el que se valida un alta: el real, más
// las carpetas que Init crea (el propio proyecto y openspec/) y las de los
// repositorios de los roles como si ya existieran.
type plannedFS struct {
	base    workspace.FS
	planned []string
}

func newPlannedFS(absPath string, layout specsLayout) plannedFS {
	planned := []string{filepath.Clean(absPath), filepath.Join(absPath, "openspec")}
	if layout.valid() {
		planned = append(planned, filepath.Join(absPath, filepath.FromSlash(layout.rel)), layout.home(absPath))
	}
	return plannedFS{base: workspace.DefaultFS(), planned: planned}
}

// specsPathProblem explica por qué la ruta del repositorio de especificaciones de
// un alta multirepo no es admisible, o devuelve "" si lo es: relativa a la carpeta
// maestra, dentro de ella y distinta de ella (workspace.Validate exige un
// repositorio aparte).
func specsPathProblem(rel string) string {
	if rel == "." {
		return "no puede ser la propia carpeta maestra: en multirepo debe ser un repositorio aparte dentro de ella"
	}
	return repositoryPathProblem(rel)
}

func (p plannedFS) isPlanned(name string) bool {
	clean := filepath.Clean(name)
	for _, dir := range p.planned {
		if clean == dir || (runtime.GOOS == "windows" && strings.EqualFold(clean, dir)) {
			return true
		}
	}
	return false
}

func (p plannedFS) Stat(name string) (os.FileInfo, error) {
	return p.base.Stat(name)
}

func (p plannedFS) IsDir(path string) (bool, error) {
	if p.isPlanned(path) {
		return true, nil
	}
	return p.base.IsDir(path)
}

// plainYAMLKey es lo que se puede escribir sin comillas como clave de rol.
var plainYAMLKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)

// yamlReservedKeys son las palabras que YAML 1.1 lee como booleano o nulo.
var yamlReservedKeys = map[string]bool{
	"true": true, "false": true, "yes": true, "no": true, "on": true, "off": true,
	"y": true, "n": true, "null": true, "~": true,
}

// yamlKey escribe una clave de rol de forma segura: tal cual si es un
// identificador simple que YAML no reinterpreta, y entre comillas en cualquier
// otro caso (con `:`, espacios, `#`, números o palabras reservadas).
func yamlKey(key string) string {
	if plainYAMLKey.MatchString(key) && !yamlReservedKeys[strings.ToLower(key)] {
		return key
	}
	return fmt.Sprintf("%q", key)
}
