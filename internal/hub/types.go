package hub

import (
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
)

// WorkspaceRecord modela un proyecto registrado en el Hub global de Axiom.
type WorkspaceRecord struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Topology     string    `json:"topology"`
	RegisteredAt time.Time `json:"registered_at"`
	LastAccessed time.Time `json:"last_accessed"`
	IsConfigured bool      `json:"is_configured"`
	Tech         []string  `json:"tech,omitempty"`
	// WorktreeMode y WorktreeRoot son las preferencias personales de worktree del proyecto
	// (REQ-25.29, prefs.go): opcionales, no cambian el esquema `"1.0"` y un binario anterior
	// los ignora.
	WorktreeMode string `json:"worktree_mode,omitempty"`
	WorktreeRoot string `json:"worktree_root,omitempty"`
	// RDDMode es la preferencia personal de revisión (RDD) del proyecto: "on", "off" o vacío
	// para heredar el interruptor global (rddpref.go). Es del usuario y nunca se versiona.
	RDDMode string `json:"rdd_mode,omitempty"`
}

// HubConfig representa el archivo de configuración central ~/.axiom/workspaces.json.
type HubConfig struct {
	Version         string            `json:"version"`
	ActiveWorkspace string            `json:"active_workspace"` // ID o ruta
	Workspaces      []WorkspaceRecord `json:"workspaces"`
}

// RoleInput define un rol provisto desde la UI o CLI para la configuración del proyecto.
type RoleInput struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	Repositories []string `json:"repositories"`
	NonBlocking  bool     `json:"non_blocking"`
	Tech         []string `json:"tech,omitempty"`
}

// DetectedProject representa un subproyecto o componente encontrado en la estructura existente.
type DetectedProject struct {
	Path        string `json:"path"`
	Stack       string `json:"stack"`
	TestCommand string `json:"test_command,omitempty"`
	Role        string `json:"role,omitempty"`
}

// AdoptedSkillInfo describe una skill o agente preexistente descubierto en el proyecto adoptado.
type AdoptedSkillInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Role        string `json:"role,omitempty"`
	Category    string `json:"category"` // "process", "tech", "rule"
	Description string `json:"description"`
}

// TechDetection encapsula los hallazgos del detector de tecnologías sobre un repositorio.
type TechDetection struct {
	PrimaryLanguage  string              `json:"primary_language"`
	Frameworks       []string            `json:"frameworks"`
	HasTests         bool                `json:"has_tests"`
	RecommendedRoles map[string][]string `json:"recommended_roles"`
	DetectedFiles    []string            `json:"detected_files"`
	HasExistingSDD   bool                `json:"has_existing_sdd"`
	SpecsRepository  string              `json:"specs_repository,omitempty"`
	DomainContext    string              `json:"domain_context,omitempty"`
	Projects         []DetectedProject   `json:"projects,omitempty"`
	ConfiguredRoles  []RoleInput         `json:"configured_roles,omitempty"`
	AdoptedSkills    []AdoptedSkillInfo  `json:"adopted_skills,omitempty"`
	// Adoption es el informe de orígenes de migración y candidatos a technical_context del
	// directorio (REQ-30.26): el inventario de `axiom archive adopt scan`, sin escribir nada.
	Adoption *adoption.Summary `json:"adoption,omitempty"`
}

// InitOptions define las opciones de configuración para inicializar un proyecto.
type InitOptions struct {
	Path     string
	Name     string
	Topology string // monorepo-embedded, multirepo
	Force    bool
	Roles    []RoleInput
	Profile  string // "knowledge", "spec-only", "full"
	// SpecsRepository es, solo en multirepo, la ruta relativa a la carpeta maestra (Path)
	// del repositorio de especificaciones propio. El alta lo crea si no existe, escribe en él
	// el axiom.yaml y deja en la carpeta maestra el puntero .axiom-workspace. Vacío, el
	// alta es la de siempre: axiom.yaml en la carpeta maestra y specs en openspec/.
	SpecsRepository string
	// WorktreePrefs son las preferencias personales de worktree que se guardan en la entrada
	// del hub al registrar el proyecto (REQ-25.30); vacías, no se guarda ninguna.
	WorktreePrefs WorktreePrefs
}

// InitResult detalla los resultados de la operación de inicialización.
type InitResult struct {
	ConfigPath         string
	Record             WorkspaceRecord
	CreatedFiles       []string
	AlreadyExisted     bool
	AdoptedSkillsCount int
	// Notes reúne los avisos de la inicialización: la omisión de openspec/specs/
	// e INDEX.md con un axiom.yaml ilegible, la conservación de un axiom.yaml
	// con --force y las violaciones de la validación de las declaraciones.
	Notes []string
	// SpecsSkipped indica que no se crearon openspec/specs/ ni openspec/INDEX.md
	// porque el axiom.yaml existente no se pudo leer.
	SpecsSkipped bool
	// Adoption es el informe de orígenes de migración y candidatos a technical_context del
	// repositorio de especificaciones recién dado de alta (REQ-30.27); nil si no hay ninguno.
	Adoption *adoption.Summary `json:"adoption,omitempty"`
	// AdoptionInstruction es la orden y la frase para lanzar la adopción desde el agente; vacía
	// si el informe no lista ningún origen de requisitos. El alta no ejecuta nada.
	AdoptionInstruction string `json:"adoption_instruction,omitempty"`
}
