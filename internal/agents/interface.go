package agents

import (
	"context"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// Capability tags for adapter feature checks.
type Capability string

// Adapter is the core abstraction for AI agent integration. Components use
// adapter methods instead of switch statements on AgentID, making it trivial
// to add new agents without modifying component code.
type Adapter interface {
	// Identity
	Agent() model.AgentID
	Tier() model.SupportTier

	// Detection
	Detect(ctx context.Context, homeDir string) (installed bool, binaryPath string, configPath string, configFound bool, err error)

	// Installation
	InstallCommand(profile system.PlatformProfile) ([][]string, error)

	// Config paths — components use these instead of hardcoding paths per agent.
	GlobalConfigDir(homeDir string) string
	SystemPromptDir(homeDir string) string
	SystemPromptFile(homeDir string) string
	SkillsDir(homeDir string) string
	SettingsPath(homeDir string) string

	// Config strategies — HOW to inject content, not WHERE (that's paths above).
	SystemPromptStrategy() model.SystemPromptStrategy
	MCPStrategy() model.MCPStrategy

	// MCP path resolution
	MCPConfigPath(homeDir string, serverName string) string

	// Optional capabilities — compatibility projections of the adapter's
	// canonical AgentCapabilityManifest.
	SupportsOutputStyles() bool
	OutputStyleDir(homeDir string) string

	SupportsSlashCommands() bool
	CommandsDir(homeDir string) string

	SupportsSubAgents() bool
	SubAgentsDir(homeDir string) string
	EmbeddedSubAgentsDir() string

	SupportsSkills() bool
	SupportsSystemPrompt() bool
	SupportsMCP() bool
}

// LegacyPromptFileProvider is an optional adapter capability for agents whose
// system prompt file was renamed. LegacySystemPromptFiles lists the earlier file
// names Axiom no longer writes; sync retires a copy only when it proves Axiom
// wrote it, so the prompt is not delivered twice.
type LegacyPromptFileProvider interface {
	LegacySystemPromptFiles(homeDir string) []string
}

// EffectiveCodeGraphWiringDetector is an optional adapter capability for agents
// whose configuration format requires semantic validation beyond marker checks.
type EffectiveCodeGraphWiringDetector interface {
	EffectiveCodeGraphWiring(homeDir string) (path string, configured bool)
}

// WorkspacePromptsDirProvider es una capacidad opcional de los adaptadores cuya
// distribución de proyecto no coincide con la global: indica dónde van los
// prompts compartidos de SDD cuando dir es la raíz del proyecto. ok es false
// fuera de él, y el llamante usa entonces la ruta global.
type WorkspacePromptsDirProvider interface {
	WorkspacePromptsDir(dir string) (path string, ok bool)
}

// WorkspaceInstructionsProvider es una capacidad opcional de los adaptadores que
// no cargan solos el fichero propio de instrucciones de Axiom en el proyecto: hay
// que registrarlo en la lista instructions de su fichero de configuración.
// settingsPath es ese fichero y entry la entrada, relativa a la raíz del
// proyecto. ok es false fuera del proyecto.
type WorkspaceInstructionsProvider interface {
	WorkspaceInstructions(dir string) (settingsPath, entry string, ok bool)
}

// WorkspaceInlineInstructionsProvider es una capacidad opcional de los adaptadores
// cuyo agente no carga ningún fichero de instrucciones propio del proyecto, pero sí
// un texto que se declara como valor de una clave de su configuración (Codex,
// developer_instructions en config.toml, que se añade a las instrucciones base).
// configPath es ese fichero de configuración y key la clave. El texto sale del
// fichero que da SystemPromptFile, que hace de fuente. ok es false fuera del
// proyecto.
type WorkspaceInlineInstructionsProvider interface {
	WorkspaceInlineInstructions(dir string) (configPath, key string, ok bool)
}

// WorkspaceContextFileProvider es una capacidad opcional de los adaptadores cuyo
// agente carga los ficheros de contexto por nombre, declarado en su configuración
// (Gemini CLI, context.fileName), y no por una lista de rutas. settingsPath es el
// fichero de configuración, entry el nombre que se añade (relativo a la carpeta de
// búsqueda) y defaults los nombres que carga el agente cuando el proyecto no
// declara ninguno: la lista del proyecto sustituye a la del usuario, así que hay que
// conservarlos. ok es false fuera del proyecto.
type WorkspaceContextFileProvider interface {
	WorkspaceContextFile(dir string) (settingsPath, entry string, defaults []string, ok bool)
}

// WorkspaceUserConfigFilesProvider es una capacidad opcional de los adaptadores cuyo
// agente lee en el proyecto ficheros de configuración del usuario que Axiom fusiona
// (VS Code Copilot, .vscode/mcp.json): son del equipo, normalmente versionados, y si
// no se pueden leer no se sustituyen por un documento vacío. Devuelve los de la raíz
// del proyecto dir, o nada fuera de ella.
type WorkspaceUserConfigFilesProvider interface {
	WorkspaceUserConfigFiles(dir string) []string
}

// WorkspaceScopedProvider es una capacidad opcional de los adaptadores ligados a la
// raíz de un proyecto: indica si dir es esa raíz. Lo usan los componentes que, con
// una instalación de proyecto, no deben tocar rutas que solo tienen sentido en la
// carpeta de usuario.
type WorkspaceScopedProvider interface {
	InWorkspace(dir string) bool
}
