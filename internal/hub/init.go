package hub

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Initializer orquesta la creación del andamiaje de Axiom sobre repositorios nuevos o existentes.
type Initializer struct {
	manager  *Manager
	detector *Detector
}

// NewInitializer construye una instancia del inicializador.
func NewInitializer(mgr *Manager, det *Detector) *Initializer {
	if det == nil {
		det = NewDetector()
	}
	return &Initializer{
		manager:  mgr,
		detector: det,
	}
}

// ForeignPlanningHomeError es el rechazo de axiom init cuando el puntero propio
// del destino resuelve un planning home distinto de <destino>/openspec
// (multi-project-hub REQ-2.2): crear ese directorio produciría un segundo
// planning home y la ambigüedad de REQ-24.6 en todos los verbos.
type ForeignPlanningHomeError struct {
	PointerPath   string // .axiom-workspace del propio destino
	PlanningHome  string // planning home que resuelve el puntero
	WorkspaceRoot string // continuación: axiom init --path "<WorkspaceRoot>"
}

// Error nombra el puntero, el planning home resuelto y la continuación.
func (e *ForeignPlanningHomeError) Error() string {
	return fmt.Sprintf("el puntero %s resuelve el planning home %s y no el openspec de este directorio, así que inicializarlo crearía un segundo planning home. Inicializa el repositorio de especificaciones con `axiom init --path %s` o monta la topología a mano según la receta de docs/manual-de-inicio.md",
		pathquote.Quote(e.PointerPath), pathquote.Quote(e.PlanningHome), pathquote.Quote(e.WorkspaceRoot))
}

// ownMarkers es lo que Init encuentra en los marcadores del propio destino.
// Nunca incluye los de un ancestro: el marcador más cercano gana (REQ-24.2) y
// el destino pasa a ser el suyo.
type ownMarkers struct {
	configPath   string                 // axiom.yaml existente que Init considera suyo; "" si no hay
	pointer      bool                   // el destino tiene un .axiom-workspace propio
	declarations workspace.Declarations // declaraciones de ese axiom.yaml
	violations   []string               // violaciones de la validación de carga, una por línea
}

// declares informa de si el axiom.yaml existente declara `baseline:`,
// `technical_context:`, `git:`, `workflow:`, `sdd:` o `repositories:`, válidos o no, o no se puede
// leer: con --force no se sobrescribe un fichero cuyas declaraciones la
// regeneración perdería (m13). Un bloque inválido cuenta igual que uno válido,
// porque lo que se pierde es lo que el usuario escribió.
//
// `repositories:` con una lista vacía no deja entradas ni error y no cuenta:
// no hay nada que perder.
func (m ownMarkers) declares() bool {
	d := m.declarations
	return d.ParseErr != nil ||
		d.Baseline != nil || d.BaselineErr != nil ||
		d.TechnicalContext != nil || d.TechnicalContextErr != nil ||
		d.Git != nil || d.GitErr != nil ||
		d.Workflow != nil || d.WorkflowErr != nil ||
		d.SDD != nil || d.SDDErr != nil ||
		len(d.Repositories) > 0 || d.RepositoriesErr != nil
}

// inspectOwnMarkers evalúa solo los marcadores de absPath, antes de escribir ni
// registrar nada, y nunca llama a workspace.Locate (D-06).
//
//  1. Con un .axiom-workspace propio resuelve el puntero con
//     workspace.ResolveOwnPointer: rechaza con su mismo *LocateError si no
//     resuelve y con *ForeignPlanningHomeError si resuelve un planning home
//     distinto de <absPath>/openspec. La comparación es por ruta y no exige que
//     exista ninguno de los dos.
//  2. Con un axiom.yaml propio sin puntero lee sus declaraciones con
//     workspace.ReadDeclarations. Un rechazo ajeno, por ejemplo un repositorio de
//     especificaciones inexistente, no detiene a init.
//  3. Sin marcadores, init se comporta como antes.
//
// El axiom.yaml que encuentra se valida con el mismo validador de carga que
// LoadConfig, Locate y workspace validate (REQ-2.2, punto 4).
//
// expectedHome es el planning home que el alta va a usar: <absPath>/openspec, o el
// del repositorio de especificaciones propio en un alta multirepo. Un puntero que
// resuelve otro distinto se sigue rechazando.
func inspectOwnMarkers(absPath, expectedHome string) (ownMarkers, error) {
	var markers ownMarkers
	loc, found, err := workspace.ResolveOwnPointer(absPath)
	if err != nil {
		return ownMarkers{}, err
	}
	if found {
		if !workspace.SamePlanningHome(loc.PlanningHome, expectedHome) {
			return ownMarkers{}, &ForeignPlanningHomeError{
				PointerPath:   filepath.Join(absPath, workspace.WorkspacePointerFilename),
				PlanningHome:  loc.PlanningHome,
				WorkspaceRoot: loc.WorkspaceRoot,
			}
		}
		markers.configPath = loc.ConfigPath
		markers.declarations = loc.Declarations
		markers.pointer = true
	} else if configPath := filepath.Join(absPath, workspace.DefaultConfigFilename); fileExists(configPath) {
		markers.configPath = configPath
		markers.declarations = workspace.ReadDeclarations(configPath)
	}
	if markers.configPath != "" {
		_, loadErr := workspace.LoadConfig(markers.configPath)
		markers.violations = loadViolations(loadErr)
	}
	return markers, nil
}

// loadViolations separa el error acumulado de la carga en una violación por línea.
func loadViolations(err error) []string {
	if err == nil {
		return nil
	}
	var violations []string
	for _, line := range strings.Split(err.Error(), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			violations = append(violations, line)
		}
	}
	return violations
}

// Init inicializa un workspace generando axiom.yaml, carpetas canónicas y registrándolo en el Hub.
//
// Antes de escribir ni registrar nada, también antes de crear el directorio
// destino, evalúa los marcadores del propio destino (inspectOwnMarkers). Con una
// baseline del proyecto, o indeterminada, no crea openspec/specs/ ni
// openspec/INDEX.md, conserva el resto del andamiaje de su perfil e informa en
// InitResult.Notes (REQ-24.23). Las violaciones de las declaraciones se informan
// pero no cambian el resultado: REQ-24.23 sitúa a init entre las operaciones que
// completan su resultado.
func (ini *Initializer) Init(opts InitOptions) (*InitResult, error) {
	// Las preferencias de worktree se validan antes de escribir ni registrar nada.
	if err := ValidateWorktreePrefs(opts.WorktreePrefs); err != nil {
		return nil, err
	}
	if opts.Path == "" {
		opts.Path = "."
	}

	absPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return nil, fmt.Errorf("ruta de inicialización inválida: %w", err)
	}

	if opts.Name == "" {
		opts.Name = filepath.Base(absPath)
	}
	if opts.Topology == "" {
		opts.Topology = "monorepo-embedded"
	}
	layout := layoutFor(opts)

	markers, err := inspectOwnMarkers(absPath, layout.home(absPath))
	if err != nil {
		return nil, err
	}
	ownership, ownershipCause := markers.declarations.Ownership()

	// El axiom.yaml existente es el que indica el puntero propio, si lo hay, y no
	// <ruta>/axiom.yaml.
	configPath := filepath.Join(absPath, "axiom.yaml")
	alreadyExisted := markers.configPath != ""
	if alreadyExisted {
		configPath = markers.configPath
	}
	createdFiles := make([]string, 0)
	var adoptedSkillsCount int
	var notes []string
	if len(markers.violations) > 0 {
		for _, violation := range markers.violations {
			notes = append(notes, fmt.Sprintf("El axiom.yaml %s no es válido: %s", pathquote.Quote(configPath), violation))
		}
		notes = append(notes, fmt.Sprintf("Ejecuta `axiom workspace validate --path %s` para ver el informe completo, incluidas las reglas que dependen del sistema de ficheros.", pathquote.Quote(absPath)))
	}

	// --force no sobrescribe un axiom.yaml que declara baseline:, technical_context:,
	// git: o repositories: (o que no se puede leer): la regeneración perdería la
	// declaración (m13).
	preserveConfig := alreadyExisted && opts.Force && markers.declares()
	if preserveConfig {
		notes = append(notes, fmt.Sprintf("Se conserva %s sin sobrescribirlo aunque se pidió --force: declara `baseline:`, `technical_context:`, `git:`, `workflow:`, `sdd:` o `repositories:`, o no se puede leer, y regenerarlo perdería esa declaración. Edítalo a mano o retira esos bloques antes de volver a ejecutar `axiom init --force`.", pathquote.Quote(configPath)))
	}

	writeBaseline := ownership == workspace.BaselineDefault
	switch ownership {
	case workspace.BaselineProject:
		notes = append(notes, fmt.Sprintf("La baseline de especificaciones es del proyecto (%s declara los dominios %s): no se crean openspec/specs/ ni openspec/INDEX.md, para no producir una segunda baseline.",
			pathquote.Quote(configPath), strings.Join(markers.declarations.DeclaredDomains(), ", ")))
	case workspace.BaselineIndeterminate:
		notes = append(notes, fmt.Sprintf("No se puede determinar si la baseline de especificaciones es del proyecto (%v): no se crean openspec/specs/ ni openspec/INDEX.md. Ejecuta `axiom workspace validate --path %s` y corrige %s.",
			ownershipCause, pathquote.Quote(absPath), pathquote.Quote(configPath)))
	}

	// 1. Si no existe axiom.yaml o se fuerza la recreación. El axiom.yaml que se va
	// a escribir se planifica y se valida antes de crear ni escribir nada: una
	// configuración inválida se rechaza con todas sus violaciones y el disco queda
	// como estaba.
	writeConfig := willWriteConfig(markers, opts)
	plan, err := ini.validateAll(absPath, opts, markers)
	if err != nil {
		return nil, err
	}
	if writeConfig {
		opts.Topology = plan.Topology
	}

	// En un alta multirepo con repositorio de specs propio el axiom.yaml vive en
	// ese repositorio y la carpeta maestra solo lleva el puntero.
	home := layout.home(absPath)
	workspaceRoot := filepath.Dir(home)
	writePointer := layout.valid() && !markers.pointer
	if layout.valid() && !alreadyExisted {
		configPath = filepath.Join(workspaceRoot, workspace.DefaultConfigFilename)
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, fmt.Errorf("no se pudo asegurar la existencia del directorio destino: %w", err)
	}
	if layout.valid() {
		// El repositorio de specs se crea si no existe, sin git init.
		if err := os.MkdirAll(workspaceRoot, 0755); err != nil {
			return nil, fmt.Errorf("no se pudo crear el repositorio de especificaciones %s: %w", workspaceRoot, err)
		}
	}

	if writeConfig {
		if err := os.WriteFile(configPath, []byte(plan.render()), 0644); err != nil {
			return nil, fmt.Errorf("fallo al escribir %s: %w", configPath, err)
		}
		createdFiles = append(createdFiles, configPath)
		adoptedSkillsCount = plan.AdoptedSkills
	}

	if writePointer {
		pointerPath := filepath.Join(absPath, workspace.WorkspacePointerFilename)
		rel, relErr := filepath.Rel(absPath, configPath)
		if relErr != nil {
			return nil, fmt.Errorf("no se pudo calcular la ruta del axiom.yaml respecto a %s: %w", absPath, relErr)
		}
		if err := os.WriteFile(pointerPath, []byte("config: "+filepath.ToSlash(rel)+"\n"), 0644); err != nil {
			return nil, fmt.Errorf("fallo al escribir %s: %w", pointerPath, err)
		}
		createdFiles = append(createdFiles, pointerPath)
	}

	// 2. Crear carpetas canónicas del arnés SDD. Con una baseline que no es la de
	// OpenSpec por defecto se omite openspec/specs/ y se conserva el resto del
	// andamiaje del perfil.
	var standardDirs []string
	if opts.Profile != "knowledge" && opts.Profile != "spec-only" {
		standardDirs = append(standardDirs, filepath.Join(workspaceRoot, ".axiom", "inbox", "skills"))
	}
	if writeBaseline {
		standardDirs = append(standardDirs, filepath.Join(home, "specs"))
	}
	standardDirs = append(standardDirs, filepath.Join(home, "changes"))

	for _, d := range standardDirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("error creando directorio estándar %s: %w", d, err)
		}
	}

	// 2.1 En perfil knowledge, asegurar openspec/INDEX.md e inyectar .mcp.json
	if opts.Profile == "knowledge" || opts.Profile == "spec-only" {
		indexPath := filepath.Join(absPath, "openspec", "INDEX.md")
		if writeBaseline && !fileExists(indexPath) {
			initialIndex := fmt.Sprintf("# Catálogo Maestro de Especificaciones Vivas — %s\n\n> **Proyecto:** %s (Spec-Driven Knowledge Base)\n> **Perfil:** Knowledge\n\n---\n\n## Resumen de Especificaciones Vivas\n\n| Dominio | Título de la Especificación | Reqs | Escenarios | Enlace |\n| :--- | :--- | :---: | :---: | :--- |\n", opts.Name, opts.Name)
			if err := os.WriteFile(indexPath, []byte(initialIndex), 0644); err != nil {
				return nil, fmt.Errorf("fallo al escribir %s: %w", indexPath, err)
			}
			createdFiles = append(createdFiles, indexPath)
		}

		if err := InjectKnowledgeMCPServers(absPath); err != nil {
			return nil, fmt.Errorf("inyectar servidores MCP en .mcp.json: %w", err)
		}
		createdFiles = append(createdFiles, filepath.Join(absPath, ".mcp.json"))
	}

	// El alta multirepo se comprueba con el mismo localizador que usan los demás
	// verbos: un fallo aquí es un defecto del alta y se informa, no se ignora.
	if layout.valid() {
		loc, locErr := workspace.LocateAt(absPath)
		if locErr != nil {
			return nil, fmt.Errorf("el alta multirepo dejó un espacio que workspace.LocateAt no resuelve: %w", locErr)
		}
		if !workspace.SamePlanningHome(loc.PlanningHome, home) {
			return nil, fmt.Errorf("el alta multirepo esperaba el planning home %s y workspace.LocateAt resuelve %s", home, loc.PlanningHome)
		}
	}

	// 3. Registrar en el catálogo global si el gestor está provisto
	var record WorkspaceRecord
	if ini.manager != nil {
		rec, err := ini.manager.RegisterWithWorktreePrefs(absPath, opts.Name, opts.Topology, opts.WorktreePrefs)
		if err != nil {
			return nil, fmt.Errorf("proyecto configurado pero no pudo registrarse en el Hub: %w", err)
		}
		_, _ = ini.manager.SetActive(rec.ID)
		record = *rec
	} else {
		record = WorkspaceRecord{
			ID:           slugify(opts.Name),
			Name:         opts.Name,
			Path:         absPath,
			Topology:     opts.Topology,
			IsConfigured: true,
		}
	}

	// 4. Informe de adopción (REQ-30.27): el inventario del repositorio de especificaciones,
	// de solo lectura, y la instrucción para lanzar la adopción desde el agente. No ejecuta
	// nada ni crea el cambio de adopción. Un fallo del escaneo deja el informe ausente.
	var adoptionReport *adoption.Summary
	adoptionInstruction := ""
	if inv, scanErr := adoption.Scan(workspaceRoot); scanErr == nil {
		if summary := adoption.Summarize(inv); !summary.Empty() {
			adoptionReport = &summary
			if summary.SourceCount > 0 {
				adoptionInstruction = adoptionInstructionFor(absPath)
			}
		}
	}

	return &InitResult{
		ConfigPath:          configPath,
		Record:              record,
		CreatedFiles:        createdFiles,
		AlreadyExisted:      alreadyExisted && (!opts.Force || preserveConfig),
		AdoptedSkillsCount:  adoptedSkillsCount,
		Notes:               notes,
		BaselineOwnership:   ownership,
		Adoption:            adoptionReport,
		AdoptionInstruction: adoptionInstruction,
	}, nil
}

func buildAxiomYaml(name, topology string, tech *TechDetection) string {
	var roles []RoleInput
	if tech != nil && len(tech.ConfiguredRoles) > 0 {
		roles = tech.ConfiguredRoles
	} else {
		var techList []string
		if tech != nil {
			techList = tech.Frameworks
			if len(techList) == 0 && tech.PrimaryLanguage != "" {
				techList = []string{tech.PrimaryLanguage}
			}
		}
		roles = []RoleInput{
			{
				Key:          "fullstack",
				Name:         name + " Fullstack",
				Repositories: []string{"."},
				NonBlocking:  false,
				Tech:         techList,
			},
		}
	}

	specsRepo := "openspec"
	domainContext := ""
	if tech != nil {
		if tech.SpecsRepository != "" {
			specsRepo = tech.SpecsRepository
		}
		domainContext = tech.DomainContext
	}

	return buildAxiomYamlWithRoles(name, topology, specsRepo, ".", domainContext, roles)
}

func buildAxiomYamlWithRoles(name, topology, specsRepo, root, domainContext string, roles []RoleInput) string {
	var sb strings.Builder
	sb.WriteString("workspace:\n")
	sb.WriteString(fmt.Sprintf("  name: %q\n", name))
	sb.WriteString(fmt.Sprintf("  topology: %q\n", topology))
	if specsRepo == "" {
		specsRepo = "openspec"
	}
	sb.WriteString(fmt.Sprintf("  specs_repository: %q\n", specsRepo))
	// root solo se escribe cuando se indica: el alta multirepo con repositorio de
	// specs propio no lo declara, como el espacio canónico.
	if root != "" {
		sb.WriteString(fmt.Sprintf("  root: %q\n", root))
	}
	sb.WriteString("\n")

	sb.WriteString("roles:\n")
	for _, r := range roles {
		roleKey := strings.TrimSpace(r.Key)
		if roleKey == "" {
			roleKey = slugify(r.Name)
		}
		sb.WriteString(fmt.Sprintf("  %s:\n", yamlKey(roleKey)))
		roleName := strings.TrimSpace(r.Name)
		if roleName == "" {
			roleName = strings.Title(roleKey)
		}
		sb.WriteString(fmt.Sprintf("    name: %q\n", roleName))

		gatePolicy := "blocking"
		if r.NonBlocking {
			// El vocabulario de multirole es blocking, deferred y optional: un rol
			// no bloqueante se escribe como deferred (nunca como advisory).
			gatePolicy = "deferred"
		}
		sb.WriteString(fmt.Sprintf("    gate_policy: %q\n", gatePolicy))

		sb.WriteString("    repositories:\n")
		repos := r.Repositories
		if len(repos) == 0 {
			repos = []string{"."}
		}
		for _, repoPath := range repos {
			sb.WriteString(fmt.Sprintf("      - path: %q\n", filepath.ToSlash(strings.TrimSpace(repoPath))))
		}

		if len(r.Tech) > 0 {
			sb.WriteString("    tech:\n")
			for _, t := range r.Tech {
				sb.WriteString(fmt.Sprintf("      - %q\n", strings.TrimSpace(t)))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("governance:\n")
	sb.WriteString("  language: \"es\"\n")
	sb.WriteString("  shared_memory: \"engram\"\n")
	sb.WriteString("  semantic_analysis: \"auto\"\n")
	if strings.TrimSpace(domainContext) != "" {
		sb.WriteString("  context: |\n")
		for _, line := range strings.Split(domainContext, "\n") {
			sb.WriteString(fmt.Sprintf("    %s\n", line))
		}
	}

	return sb.String()
}

// AdoptionChange es el nombre del cambio SDD de adopción que propone la instrucción del alta.
const AdoptionChange = "adopcion-openspec"

// adoptionInstructionFor compone la orden y la frase para el agente (REQ-30.27). La ruta es la
// carpeta maestra, que `axiom archive adopt` resuelve al repositorio de especificaciones.
func adoptionInstructionFor(masterPath string) string {
	return fmt.Sprintf("Adopta los requisitos previos de este proyecto en OpenSpec: crea el cambio SDD `%[1]s`, ejecuta `axiom archive adopt scan --path %[2]s --change %[1]s` y redacta la delta siguiendo la referencia de adopción de la skill sdd-spec; comprueba el resultado con `axiom archive adopt check --path %[2]s --change %[1]s` y somete la compuerta `spec` a aprobación humana.",
		AdoptionChange, pathquote.Quote(masterPath))
}
