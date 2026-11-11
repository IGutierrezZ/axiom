// Package copilotcli es el adaptador de GitHub Copilot CLI (binario `copilot`).
//
// Es deliberadamente distinto del adaptador de VS Code Copilot: solo comparten la
// carpeta de skills (<raíz>/skills). El resto difiere. El prompt global va en
// <raíz>/copilot-instructions.md, el MCP en <raíz>/mcp-config.json con raíz
// `mcpServers`, y los ajustes en <raíz>/settings.json. La raíz es COPILOT_HOME o
// ~/.copilot (véase copilotconfig.Root).
package copilotcli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/capabilitymanifest"
	"github.com/IGutierrezZ/axiom/v3/internal/copilotconfig"
	"github.com/IGutierrezZ/axiom/v3/internal/installcmd"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// LookPathOverride permite a los tests sustituir la búsqueda del binario.
var LookPathOverride = exec.LookPath

// Binary es el ejecutable que instala `npm install -g @github/copilot`.
const Binary = "copilot"

type statResult struct {
	isDir bool
	err   error
}

// Adapter implementa agents.Adapter para GitHub Copilot CLI.
type Adapter struct {
	lookPath func(string) (string, error)
	statPath func(string) statResult
	hasState func(string) bool
	resolver installcmd.Resolver
}

func NewAdapter() *Adapter {
	return &Adapter{lookPath: LookPathOverride, statPath: defaultStat, hasState: copilotconfig.HasCLIState}
}

// --- Identity ---

func (a *Adapter) Agent() model.AgentID    { return model.AgentGitHubCopilotCLI }
func (a *Adapter) Tier() model.SupportTier { return model.TierFull }

// --- Detection ---

func (a *Adapter) Detect(_ context.Context, homeDir string) (bool, string, string, bool, error) {
	configPath := copilotconfig.Root(homeDir)

	binaryPath, err := a.lookPath(Binary)
	installed := err == nil

	stat := a.statPath(configPath)
	if stat.err != nil {
		if os.IsNotExist(stat.err) {
			return installed, binaryPath, configPath, false, nil
		}
		return false, "", "", false, stat.err
	}
	// La carpeta existe, pero con solo las skills compartidas es de VS Code
	// Copilot: no se da por configuración de Copilot CLI.
	return installed, binaryPath, configPath, stat.isDir && a.hasState(configPath), nil
}

// --- Installation ---

func (a *Adapter) CapabilityManifest() capabilitymanifest.AgentCapabilityManifest {
	return capabilitymanifest.MustForAgent(model.AgentGitHubCopilotCLI)
}

func (a *Adapter) InstallCommand(profile system.PlatformProfile) ([][]string, error) {
	resolver := a.resolver
	if resolver == nil {
		resolver = installcmd.NewResolver()
	}
	return resolver.ResolveAgentInstall(profile, a.Agent())
}

// --- Config paths ---

func (a *Adapter) GlobalConfigDir(homeDir string) string { return copilotconfig.Root(homeDir) }
func (a *Adapter) SystemPromptDir(homeDir string) string { return copilotconfig.Root(homeDir) }

func (a *Adapter) SystemPromptFile(homeDir string) string {
	return filepath.Join(copilotconfig.Root(homeDir), "copilot-instructions.md")
}

// SkillsDir es la misma carpeta que usa el adaptador de VS Code Copilot cuando no
// hay COPILOT_HOME. Que dos agentes escriban en ella es intencionado: las skills
// son idénticas, y el uninstall de uno no debe retirar las del otro.
func (a *Adapter) SkillsDir(homeDir string) string {
	return filepath.Join(copilotconfig.Root(homeDir), "skills")
}

func (a *Adapter) SettingsPath(homeDir string) string {
	return filepath.Join(copilotconfig.Root(homeDir), "settings.json")
}

// --- Config strategies ---

// SystemPromptStrategy: copilot-instructions.md es un fichero que Axiom gestiona
// por completo, igual que el resto de agentes con un único fichero de instrucciones.
func (a *Adapter) SystemPromptStrategy() model.SystemPromptStrategy {
	return model.StrategyFileReplace
}

func (a *Adapter) MCPStrategy() model.MCPStrategy { return model.StrategyMCPConfigFile }

// --- MCP ---

// MCPConfigPath es <raíz>/mcp-config.json, con raíz `mcpServers` (no `servers`, que
// es la de VS Code).
func (a *Adapter) MCPConfigPath(homeDir string, _ string) string {
	return filepath.Join(copilotconfig.Root(homeDir), "mcp-config.json")
}

// --- Optional capabilities ---

func (a *Adapter) SupportsOutputStyles() bool     { return a.CapabilityManifest().Features.OutputStyles }
func (a *Adapter) OutputStyleDir(_ string) string { return "" }
func (a *Adapter) SupportsSlashCommands() bool    { return a.CapabilityManifest().Features.SlashCommands }
func (a *Adapter) CommandsDir(_ string) string    { return "" }
func (a *Adapter) SupportsSubAgents() bool        { return a.CapabilityManifest().Features.FileSubAgents }
func (a *Adapter) SubAgentsDir(_ string) string   { return "" }
func (a *Adapter) EmbeddedSubAgentsDir() string   { return "" }
func (a *Adapter) SupportsSkills() bool           { return a.CapabilityManifest().Features.Skills }
func (a *Adapter) SupportsSystemPrompt() bool     { return a.CapabilityManifest().Features.SystemPrompt }
func (a *Adapter) SupportsMCP() bool              { return a.CapabilityManifest().Features.MCP }

func defaultStat(path string) statResult {
	info, err := os.Stat(path)
	if err != nil {
		return statResult{err: err}
	}
	return statResult{isDir: info.IsDir()}
}
