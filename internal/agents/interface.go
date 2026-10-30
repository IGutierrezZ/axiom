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
