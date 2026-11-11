package catalog

import "github.com/IGutierrezZ/axiom/v3/internal/model"

type Agent struct {
	ID         model.AgentID
	Name       string
	Tier       model.SupportTier
	ConfigPath string
}

var allAgents = []Agent{
	{ID: model.AgentClaudeCode, Name: "Claude Code", Tier: model.TierFull, ConfigPath: "~/.claude"},
	{ID: model.AgentOpenCode, Name: "OpenCode", Tier: model.TierFull, ConfigPath: "~/.config/opencode"},
	{ID: model.AgentKilocode, Name: "Kilo Code", Tier: model.TierFull, ConfigPath: "~/.config/kilo"},
	{ID: model.AgentGeminiCLI, Name: "Gemini CLI", Tier: model.TierFull, ConfigPath: "~/.gemini"},
	{ID: model.AgentCodex, Name: "Codex", Tier: model.TierFull, ConfigPath: "~/.codex"},
	{ID: model.AgentCursor, Name: "Cursor", Tier: model.TierFull, ConfigPath: "~/.cursor"},
	{ID: model.AgentVSCodeCopilot, Name: "VS Code Copilot", Tier: model.TierFull, ConfigPath: "~/.copilot"},
	{ID: model.AgentAntigravity, Name: "Google Antigravity", Tier: model.TierFull, ConfigPath: "~/.gemini/antigravity-cli"},
	{ID: model.AgentWindsurf, Name: "Windsurf", Tier: model.TierFull, ConfigPath: "~/.codeium/windsurf"},
	{ID: model.AgentKimi, Name: "Kimi Code", Tier: model.TierFull, ConfigPath: "~/.kimi"},
	{ID: model.AgentQwenCode, Name: "Qwen Code", Tier: model.TierFull, ConfigPath: "~/.qwen"},
	{ID: model.AgentKiroIDE, Name: "Kiro IDE", Tier: model.TierFull, ConfigPath: "~/.kiro"},
	{ID: model.AgentOpenClaw, Name: "OpenClaw", Tier: model.TierFull, ConfigPath: "~/.openclaw"},
	{ID: model.AgentPi, Name: "Pi", Tier: model.TierFull, ConfigPath: "~/.pi"},
	{ID: model.AgentTrae, Name: "Trae IDE", Tier: model.TierFull, ConfigPath: "~/.trae"},
	{ID: model.AgentHermes, Name: "Hermes", Tier: model.TierFull, ConfigPath: "~/.hermes"},
	{ID: model.AgentGitHubCopilotCLI, Name: "GitHub Copilot CLI", Tier: model.TierFull, ConfigPath: "~/.copilot"},
}

// mvpAgents are the original MVP agents (Claude Code, OpenCode).
var mvpAgents = []Agent{
	{ID: model.AgentClaudeCode, Name: "Claude Code", Tier: model.TierFull, ConfigPath: "~/.claude"},
	{ID: model.AgentOpenCode, Name: "OpenCode", Tier: model.TierFull, ConfigPath: "~/.config/opencode"},
}

func AllAgents() []Agent {
	agents := make([]Agent, len(allAgents))
	copy(agents, allAgents)
	return agents
}

func MVPAgents() []Agent {
	agents := make([]Agent, len(mvpAgents))
	copy(agents, mvpAgents)
	return agents
}

func IsMVPAgent(agent model.AgentID) bool {
	for _, current := range mvpAgents {
		if current.ID == agent {
			return true
		}
	}

	return false
}

// unselectableAgents son agentes cuyo adaptador sigue en el código pero que no se
// pueden elegir al instalar: no aparecen en la TUI, ni en las selecciones por
// defecto ni en el descubrimiento por carpetas, y `--agent` los rechaza.
//
// Motivo (decisión de producto, 2026-10-30): hoy no son utilizables con Axiom. En el
// ámbito workspace escriben en rutas que el agente no lee (Qwen Code solo lee
// QWEN.md en la raíz, Kimi no lee las skills de .config/agents/skills) y su MCP
// solo es global (Kimi y Hermes).
//
// Para reactivar uno basta con quitarlo de este mapa: el adaptador, sus
// componentes y sus pruebas siguen ahí. Una instalación global existente que ya
// los tiene en el estado se sigue sincronizando; uninstall, doctor y verify no
// cambian.
var unselectableAgents = map[model.AgentID]struct{}{
	model.AgentQwenCode: {},
	model.AgentHermes:   {},
	model.AgentKimi:     {},
}

// IsSelectableAgent indica si el agente se puede elegir al instalar. Es el único
// punto de verdad de lo seleccionable: un agente no seleccionable sigue siendo
// conocido (IsSupportedAgent, AllAgents).
func IsSelectableAgent(agent model.AgentID) bool {
	if !IsSupportedAgent(agent) {
		return false
	}
	_, hidden := unselectableAgents[agent]
	return !hidden
}

// SelectableAgents devuelve los agentes que se pueden elegir al instalar, en el
// orden del catálogo. AllAgents devuelve además los no seleccionables.
func SelectableAgents() []Agent {
	agents := make([]Agent, 0, len(allAgents))
	for _, agent := range allAgents {
		if IsSelectableAgent(agent.ID) {
			agents = append(agents, agent)
		}
	}
	return agents
}

func IsSupportedAgent(agent model.AgentID) bool {
	for _, current := range allAgents {
		if current.ID == agent {
			return true
		}
	}

	return false
}
