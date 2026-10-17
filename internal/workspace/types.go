package workspace

// TopologyType define las topologías de espacio de trabajo admitidas en Axiom.
// La topología de repositorio de especificaciones separado que existió antes de
// INC-25 fue retirada (REQ-25.25) y ya no tiene constante: el cargador la
// rechaza con un mensaje que nombra `multirepo` (loader.go).
type TopologyType string

const (
	// TopologyMonorepoEmbedded: código y especificaciones conviven en el mismo repositorio.
	TopologyMonorepoEmbedded TopologyType = "monorepo-embedded"

	// TopologyMultirepo: múltiples repositorios de código por roles y repositorio canónico de especificaciones obligatorio.
	TopologyMultirepo TopologyType = "multirepo"
)

// WorkspaceSection encapsula la identidad, topología y rutas base del espacio de trabajo.
type WorkspaceSection struct {
	Name            string       `yaml:"name"`
	Topology        TopologyType `yaml:"topology"`
	SpecsRepository string       `yaml:"specs_repository"`
	Root            string       `yaml:"root,omitempty"`

	// SkillRoots declara directorios versionados con skills canónicas fuera de
	// "skills/" (p. ej. "internal/assets/skills"). Las rutas son relativas al
	// proyecto; CleanSkillRoot define cuáles son válidas.
	SkillRoots []string `yaml:"skill_roots,omitempty"`
}

// RepositoryEntry define una ruta a un repositorio o componente local.
type RepositoryEntry struct {
	Path string `yaml:"path"`
}

// RoleConfig define las responsabilidades, tecnologías y repositorios asignados a un rol.
type RoleConfig struct {
	Name         string            `yaml:"name"`
	Repositories []RepositoryEntry `yaml:"repositories"`
	Tech         []string          `yaml:"tech,omitempty"`
	GatePolicy   string            `yaml:"gate_policy,omitempty"` // blocking o advisory
}

// GovernanceConfig define configuraciones transversales del proyecto.
type GovernanceConfig struct {
	Language         string `yaml:"language,omitempty"`
	SharedMemory     string `yaml:"shared_memory,omitempty"`
	SemanticAnalysis string `yaml:"semantic_analysis,omitempty"`
	Context          string `yaml:"context,omitempty"`
}

// WorkspaceConfig es la raíz del archivo de configuración axiom.yaml.
//
// Baseline y TechnicalContext se decodifican aparte y de forma estricta (D-04),
// por eso llevan yaml:"-": la decodificación base ignora esas claves y
// parseLayered las rellena solo cuando el bloque es válido.
type WorkspaceConfig struct {
	Workspace        WorkspaceSection        `yaml:"workspace"`
	Roles            map[string]RoleConfig   `yaml:"roles"`
	Governance       GovernanceConfig        `yaml:"governance,omitempty"`
	Baseline         *BaselineConfig         `yaml:"-"`
	TechnicalContext *TechnicalContextConfig `yaml:"-"`
}

// ValidationReport contiene el resultado determinista de la validación del espacio de trabajo.
type ValidationReport struct {
	Valid         bool         `json:"valid"`
	Topology      TopologyType `json:"topology"`
	WorkspaceRoot string       `json:"workspace_root"`
	CheckedPaths  []string     `json:"checked_paths"`
	Errors        []string     `json:"errors"`
	Warnings      []string     `json:"warnings"`
}
