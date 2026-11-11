package agents

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/antigravity"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/codex"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/copilotcli"
	cursoradapter "github.com/IGutierrezZ/axiom/v3/internal/agents/cursor"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/gemini"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/hermes"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/kilocode"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/kimi"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/kiro"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/openclaw"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/opencode"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/pi"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/qwen"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/trae"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/vscode"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/windsurf"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

var defaultAgentIDs = []model.AgentID{
	model.AgentClaudeCode,
	model.AgentOpenCode,
	model.AgentKilocode,
	model.AgentGeminiCLI,
	model.AgentCursor,
	model.AgentVSCodeCopilot,
	model.AgentCodex,
	model.AgentAntigravity,
	model.AgentWindsurf,
	model.AgentKimi,
	model.AgentQwenCode,
	model.AgentKiroIDE,
	model.AgentOpenClaw,
	model.AgentPi,
	model.AgentTrae,
	model.AgentHermes,
	model.AgentGitHubCopilotCLI,
}

func NewAdapter(agent model.AgentID) (Adapter, error) {
	switch agent {
	case model.AgentClaudeCode:
		return claude.NewAdapter(), nil
	case model.AgentOpenCode:
		return opencode.NewAdapter(), nil
	case model.AgentKilocode:
		return kilocode.NewAdapter(), nil
	case model.AgentGeminiCLI:
		return gemini.NewAdapter(), nil
	case model.AgentCursor:
		return cursoradapter.NewAdapter(), nil
	case model.AgentVSCodeCopilot:
		return vscode.NewAdapter(), nil
	case model.AgentCodex:
		return codex.NewAdapter(), nil
	case model.AgentAntigravity:
		return antigravity.NewAdapter(), nil
	case model.AgentWindsurf:
		return windsurf.NewAdapter(), nil
	case model.AgentKimi:
		return kimi.NewAdapter(), nil
	case model.AgentQwenCode:
		return qwen.NewAdapter(), nil
	case model.AgentKiroIDE:
		return kiro.NewAdapter(), nil
	case model.AgentOpenClaw:
		return openclaw.NewAdapter(), nil
	case model.AgentPi:
		return pi.NewAdapter(), nil
	case model.AgentTrae:
		return trae.NewAdapter(), nil
	case model.AgentHermes:
		return hermes.NewAdapter(), nil
	case model.AgentGitHubCopilotCLI:
		return copilotcli.NewAdapter(), nil
	default:
		return nil, AgentNotSupportedError{Agent: agent}
	}
}

func NewDefaultRegistry() (*Registry, error) {
	adapters := make([]Adapter, 0, len(defaultAgentIDs))

	for _, agent := range defaultAgentIDs {
		adapter, err := NewAdapter(agent)
		if err != nil {
			return nil, fmt.Errorf("create %s adapter: %w", agent, err)
		}
		adapters = append(adapters, adapter)
	}

	registry, err := NewRegistry(adapters...)
	if err != nil {
		return nil, fmt.Errorf("create registry: %w", err)
	}

	return registry, nil
}

// NewMVPRegistry creates a registry with only the MVP agents (Claude Code, OpenCode).
// Kept for backward compatibility.
func NewMVPRegistry() (*Registry, error) {
	claudeAdapter, err := NewAdapter(model.AgentClaudeCode)
	if err != nil {
		return nil, fmt.Errorf("create claude adapter: %w", err)
	}

	opencodeAdapter, err := NewAdapter(model.AgentOpenCode)
	if err != nil {
		return nil, fmt.Errorf("create opencode adapter: %w", err)
	}

	registry, err := NewRegistry(claudeAdapter, opencodeAdapter)
	if err != nil {
		return nil, fmt.Errorf("create registry: %w", err)
	}

	return registry, nil
}

// BindWorkspace devuelve el adaptador ligado a la raíz de un proyecto. Los
// adaptadores cuya distribución de proyecto difiere de la global (OpenCode y
// Kilo: opencode.json y .opencode, .kilo) resuelven sus rutas con esa raíz en la
// distribución de proyecto; las rutas que se piden con otra carpeta (la de
// usuario) siguen siendo las globales. Gemini CLI se liga para que sus
// instrucciones vayan a un fichero propio, registrado en context.fileName, y no
// a un .gemini/GEMINI.md que no lee. Codex, para que sus instrucciones vayan a
// developer_instructions de .codex/config.toml. VS Code Copilot, para que
// instrucciones y MCP vayan a .github/ y .vscode/. Además, en el proyecto las
// skills de todos los que leen .agents/skills (Codex, VS Code, OpenCode, Kilo, Gemini
// y Cursor) van a esa carpeta compartida (.agents/skills) y no a una
// propia por agente. El resto de adaptadores (Claude Code, Kiro...) se devuelven sin
// cambios: su carpeta de configuración ya es una carpeta de punto en la raíz.
func BindWorkspace(adapter Adapter, root string) Adapter {
	if root == "" {
		return adapter
	}
	switch concrete := adapter.(type) {
	case *opencode.Adapter:
		return concrete.WithWorkspace(root)
	case *kilocode.Adapter:
		return concrete.WithWorkspace(root)
	case *gemini.Adapter:
		return concrete.WithWorkspace(root)
	case *codex.Adapter:
		return concrete.WithWorkspace(root)
	case *vscode.Adapter:
		return concrete.WithWorkspace(root)
	case *cursoradapter.Adapter:
		return concrete.WithWorkspace(root)
	default:
		return adapter
	}
}
