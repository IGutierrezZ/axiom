package vscode

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/capabilitymanifest"
	"github.com/IGutierrezZ/axiom/v3/internal/copilotconfig"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

type Adapter struct {
	lookPath func(string) (string, error)
}

func NewAdapter() *Adapter {
	return &Adapter{
		lookPath: exec.LookPath,
	}
}

// --- Identity ---

func (a *Adapter) Agent() model.AgentID {
	return model.AgentVSCodeCopilot
}

func (a *Adapter) Tier() model.SupportTier {
	return model.TierFull
}

// --- Detection ---

func (a *Adapter) Detect(_ context.Context, homeDir string) (bool, string, string, bool, error) {
	binaryPath, err := a.lookPath("code")
	if err != nil {
		if !errors.Is(err, exec.ErrNotFound) {
			return false, "", "", false, err
		}
		return false, "", "", false, nil
	}
	extensionsDir := filepath.Join(homeDir, ".vscode", "extensions")
	entries, err := os.ReadDir(extensionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, binaryPath, extensionsDir, false, nil
		}
		return false, "", extensionsDir, false, err
	}
	for _, entry := range entries {
		if entry.IsDir() && system.IsCopilotExtensionDir(entry.Name()) {
			return true, binaryPath, extensionsDir, true, nil
		}
	}
	return false, binaryPath, extensionsDir, false, nil
}

// --- Installation ---

func (a *Adapter) CapabilityManifest() capabilitymanifest.AgentCapabilityManifest {
	return capabilitymanifest.MustForAgent(model.AgentVSCodeCopilot)
}

func (a *Adapter) InstallCommand(_ system.PlatformProfile) ([][]string, error) {
	return nil, AgentNotInstallableError{Agent: model.AgentVSCodeCopilot}
}

// --- Config paths ---
// VS Code Copilot reads .instructions.md files from the VS Code User prompts folder.
// Skills are loaded from ~/.copilot/skills/ (global), .github/skills/ (workspace),
// ~/.claude/skills/, and .claude/skills/. We target ~/.copilot/skills/ for global reach.
// MCP servers go to the Copilot harness user file, <COPILOT_HOME|~/.copilot>/mcp-config.json
// (see MCPConfigPath), not to the VS Code User profile mcp.json of the Local harness.

// GlobalConfigDir es ~/.copilot aunque COPILOT_HOME esté definida; véase SkillsDir.
func (a *Adapter) GlobalConfigDir(homeDir string) string {
	return filepath.Join(homeDir, ".copilot")
}

func (a *Adapter) SystemPromptDir(homeDir string) string {
	return filepath.Join(a.vscodeUserDir(homeDir), "prompts")
}

func (a *Adapter) SystemPromptFile(homeDir string) string {
	return filepath.Join(a.SystemPromptDir(homeDir), "axiom.instructions.md")
}

// LegacySystemPromptFiles lists the earlier prompt file names Axiom no longer
// writes. A managed copy is retired on sync so the prompt is not duplicated.
func (a *Adapter) LegacySystemPromptFiles(homeDir string) []string {
	return []string{filepath.Join(a.SystemPromptDir(homeDir), "gentle-ai.instructions.md")}
}

// SkillsDir es ~/.copilot/skills, la carpeta personal de skills del arnés de Copilot
// en VS Code. No usa copilotconfig.Root, a diferencia de MCPConfigPath y de Copilot
// CLI: VS Code documenta COPILOT_HOME solo para mcp-config.json
// (code.visualstudio.com/docs/agents/reference/mcp-configuration). Para las skills
// documenta ~/.copilot/skills
// (code.visualstudio.com/docs/copilot/customization/agent-skills), y para los agentes
// de usuario ~/.copilot/agents
// (code.visualstudio.com/docs/agents/reference/custom-agent-file-reference), sin
// condicionarlos a COPILOT_HOME. Sin esa variable, la carpeta es la misma que la de
// Copilot CLI. Con ella, Copilot CLI usa $COPILOT_HOME/skills y cada agente tiene la
// suya.
func (a *Adapter) SkillsDir(homeDir string) string {
	return filepath.Join(homeDir, ".copilot", "skills")
}

func (a *Adapter) SettingsPath(homeDir string) string {
	return filepath.Join(a.vscodeUserDir(homeDir), "settings.json")
}

// --- Config strategies ---

func (a *Adapter) SystemPromptStrategy() model.SystemPromptStrategy {
	return model.StrategyInstructionsFile
}

func (a *Adapter) MCPStrategy() model.MCPStrategy {
	return model.StrategyMCPConfigFile
}

// --- MCP ---

// MCPConfigPath es <COPILOT_HOME|~/.copilot>/mcp-config.json, con la clave
// `mcpServers`: el destino de usuario del arnés de Copilot, que lo lee directamente
// (code.visualstudio.com/docs/agents/reference/mcp-configuration). Es el mismo fichero
// que escribe el adaptador de Copilot CLI, con la misma entrada para cada servidor, así
// que con los dos instalados `sync` sigue siendo idempotente.
func (a *Adapter) MCPConfigPath(homeDir string, _ string) string {
	return filepath.Join(copilotconfig.Root(homeDir), "mcp-config.json")
}

// LegacyMCPConfigPath es el mcp.json del perfil de usuario de VS Code, donde las
// instalaciones anteriores declaraban los servidores MCP con la clave `servers`. Ese
// formato es del arnés Local, deprecado y que VS Code retira en febrero de 2027: el
// arnés de Copilot no lo lee, solo lo reenvía el editor. `sync` y la desinstalación
// retiran de él las entradas de Axiom.
func (a *Adapter) LegacyMCPConfigPath(homeDir string) string {
	return filepath.Join(a.vscodeUserDir(homeDir), "mcp.json")
}

// vscodeUserDir returns the OS-specific VS Code User config directory.
//
// Environment overrides (XDG_CONFIG_HOME, APPDATA) are honored only when
// homeDir is the real user home: a caller that passes a custom installation
// root (sandboxed installs, tests) must stay contained inside that root, so
// ambient environment can never redirect a write outside it.
func (a *Adapter) vscodeUserDir(homeDir string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Code", "User")
	case "windows":
		if appData := strings.TrimSpace(os.Getenv("APPDATA")); filepath.IsAbs(appData) && isRealUserHome(homeDir) {
			return filepath.Join(appData, "Code", "User")
		}
		return filepath.Join(homeDir, "AppData", "Roaming", "Code", "User")
	default:
		if xdgConfigHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); filepath.IsAbs(xdgConfigHome) && isRealUserHome(homeDir) {
			return filepath.Join(xdgConfigHome, "Code", "User")
		}
		return filepath.Join(homeDir, ".config", "Code", "User")
	}
}

// isRealUserHome reports whether homeDir is the current user's actual home
// directory — the only case where process-wide environment overrides may
// legitimately steer config resolution away from homeDir.
func isRealUserHome(homeDir string) bool {
	userHome, err := os.UserHomeDir()
	return err == nil && filepath.Clean(homeDir) == filepath.Clean(userHome)
}

// --- Optional capabilities ---

func (a *Adapter) SupportsOutputStyles() bool {
	return a.CapabilityManifest().Features.OutputStyles
}

func (a *Adapter) OutputStyleDir(_ string) string {
	return ""
}

func (a *Adapter) SupportsSlashCommands() bool {
	return a.CapabilityManifest().Features.SlashCommands
}

func (a *Adapter) CommandsDir(_ string) string {
	return ""
}

func (a *Adapter) SupportsSubAgents() bool {
	return a.CapabilityManifest().Features.FileSubAgents
}

// SubAgentsDir es vacío en la carpeta de usuario: los agentes por fase de Copilot
// viven solo en el proyecto (.github/agents, véase WorkspaceAdapter.SubAgentsDir).
// Copilot CLI da prioridad al agente personal de ~/.copilot/agents frente al del
// proyecto cuando coinciden los identificadores, así que un agente global sin el
// modelo elegido en el proyecto lo taparía.
func (a *Adapter) SubAgentsDir(_ string) string {
	return ""
}

// EmbeddedSubAgentsDir son las plantillas `.agent.md` de Copilot, compartidas con
// el adaptador de Copilot CLI.
func (a *Adapter) EmbeddedSubAgentsDir() string {
	return "copilot/agents"
}

func (a *Adapter) SupportsSkills() bool {
	return a.CapabilityManifest().Features.Skills
}

func (a *Adapter) SupportsSystemPrompt() bool {
	return a.CapabilityManifest().Features.SystemPrompt
}

func (a *Adapter) SupportsMCP() bool {
	return a.CapabilityManifest().Features.MCP
}

// AgentNotInstallableError is returned when InstallCommand is called on a desktop-only agent.
type AgentNotInstallableError struct {
	Agent model.AgentID
}

func (e AgentNotInstallableError) Error() string {
	return "agent " + string(e.Agent) + " is a desktop app and cannot be installed via CLI"
}
