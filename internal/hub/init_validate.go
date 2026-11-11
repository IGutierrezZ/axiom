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
	DomainContext string
	Roles         []RoleInput
	AdoptedSkills int
}

// render genera el contenido del axiom.yaml del plan.
func (p configPlan) render() string {
	return buildAxiomYamlWithRoles(p.Name, p.Topology, p.SpecsRepo, p.DomainContext, p.Roles)
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
	markers, err := inspectOwnMarkers(absPath)
	if err != nil {
		return err
	}
	if opts.Name == "" {
		opts.Name = filepath.Base(absPath)
	}
	if opts.Topology == "" {
		opts.Topology = "monorepo-embedded"
	}
	if !willWriteConfig(markers, opts) {
		return nil
	}
	_, err = ini.checkPlan(absPath, opts)
	return err
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
	return configPlan{
		Name:          opts.Name,
		Topology:      opts.Topology,
		SpecsRepo:     specsRepo,
		DomainContext: tech.DomainContext,
		Roles:         effectiveRoles,
		AdoptedSkills: len(tech.AdoptedSkills),
	}
}

// checkPlan planifica el axiom.yaml de opts y lo valida antes de escribir nada.
// Devuelve el plan, o un *InitValidationError con todas las violaciones.
func (ini *Initializer) checkPlan(absPath string, opts InitOptions) (configPlan, error) {
	plan := ini.planConfig(absPath, opts)
	if violations := validatePlan(absPath, plan); len(violations) > 0 {
		return plan, &InitValidationError{Violations: violations}
	}
	return plan, nil
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
	fsys := newPlannedFS(absPath)
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

func newPlannedFS(absPath string) plannedFS {
	return plannedFS{
		base:    workspace.DefaultFS(),
		planned: []string{filepath.Clean(absPath), filepath.Join(absPath, "openspec")},
	}
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
