package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// workspaceAgentsNoticeWriter recibe el aviso de que un agente se omite en el
// ámbito workspace. Es una variable para que las pruebas lo capturen.
var workspaceAgentsNoticeWriter io.Writer = os.Stderr

// agentSupportsWorkspace indica si el agente puede alojar su configuración en la
// raíz del proyecto. Los demás solo tienen configuración global: con
// --scope workspace se omiten enteros en lugar de escribir en la carpeta de usuario.
func agentSupportsWorkspace(id model.AgentID) bool {
	switch id {
	case model.AgentTrae, model.AgentWindsurf, model.AgentAntigravity:
		return false
	default:
		return true
	}
}

// splitAgentsForScope separa los agentes que se pueden instalar en el ámbito de
// los que se omiten. Es el único punto donde install, sync y la TUI aplican la
// regla: con --scope global no filtra nada y devuelve la misma lista.
//
// Los agentes omitidos nunca llegan al plan, así que ninguno de los componentes
// que resuelven rutas por agente puede escribir por ellos en la carpeta de usuario.
func splitAgentsForScope(scope InstallScope, agentIDs []model.AgentID) (kept, omitted []model.AgentID) {
	if scope != ScopeWorkspace {
		return agentIDs, nil
	}
	for _, id := range agentIDs {
		if !agentSupportsWorkspace(id) {
			omitted = append(omitted, id)
		}
	}
	if len(omitted) == 0 {
		return agentIDs, nil
	}
	kept = make([]model.AgentID, 0, len(agentIDs)-len(omitted))
	for _, id := range agentIDs {
		if agentSupportsWorkspace(id) {
			kept = append(kept, id)
		}
	}
	return kept, omitted
}

// AgentsForScope devuelve los agentes que se instalan en el ámbito dado: con
// --scope workspace, sin los que no admiten ese ámbito. Lo usa la TUI para que el
// estado persistido coincida con lo que realmente se instaló.
func AgentsForScope(scope InstallScope, agentIDs []model.AgentID) []model.AgentID {
	kept, _ := splitAgentsForScope(scope, agentIDs)
	return kept
}

// omittedWorkspaceAgentNotice es el aviso de un agente omitido en ámbito workspace.
func omittedWorkspaceAgentNotice(id model.AgentID) string {
	return fmt.Sprintf("%s omitido: no admite el ámbito workspace; Axiom no escribe en su configuración global si no se pide con --scope global.", id)
}

// omittedWorkspaceAgentNotices devuelve un aviso por agente omitido (no uno por
// componente), en el orden en que se seleccionaron.
func omittedWorkspaceAgentNotices(omitted []model.AgentID) []string {
	if len(omitted) == 0 {
		return nil
	}
	notices := make([]string, 0, len(omitted))
	for _, id := range omitted {
		notices = append(notices, omittedWorkspaceAgentNotice(id))
	}
	return notices
}

// noteOmittedWorkspaceAgents imprime por stderr un aviso por cada agente omitido.
func noteOmittedWorkspaceAgents(omitted []model.AgentID) {
	for _, notice := range omittedWorkspaceAgentNotices(omitted) {
		fmt.Fprintln(workspaceAgentsNoticeWriter, notice)
	}
}

// errNoWorkspaceAgents es el error de una instalación en la que ningún agente
// seleccionado admite el ámbito workspace: sin agentes solo quedarían los
// componentes de máquina, que no es lo que el usuario pidió instalar.
func errNoWorkspaceAgents(omitted []model.AgentID) error {
	names := make([]string, 0, len(omitted))
	for _, id := range omitted {
		names = append(names, string(id))
	}
	return fmt.Errorf("ningún agente seleccionado admite el ámbito workspace (%s); vuelve a ejecutar con --scope global para instalarlos en su configuración global", strings.Join(names, ", "))
}

// scopedAdapter liga el adaptador a la raíz del proyecto cuando el ámbito es
// workspace. OpenCode, Kilo, Gemini, Codex y VS Code Copilot tienen en el proyecto otra
// distribución que en la carpeta de usuario (opencode.json y .opencode, .kilo,
// .gemini/AXIOM.md, .codex/config.toml y .agents/skills, .github/ y .vscode/): el adaptador ligado la devuelve para la raíz
// del proyecto y mantiene la global para cualquier otra carpeta. Con ámbito global,
// o para el resto de agentes, no cambia nada.
func scopedAdapter(scope InstallScope, workspaceDir string, adapter agents.Adapter) agents.Adapter {
	if scope != ScopeWorkspace || adapter == nil {
		return adapter
	}
	return agents.BindWorkspace(adapter, workspaceDir)
}

// newScopedAdapter crea el adaptador de un agente ya ligado al ámbito.
func newScopedAdapter(scope InstallScope, workspaceDir string, id model.AgentID) (agents.Adapter, error) {
	adapter, err := agents.NewAdapter(id)
	if err != nil {
		return nil, err
	}
	return scopedAdapter(scope, workspaceDir, adapter), nil
}

// resolveAdaptersScoped es resolveAdapters con los adaptadores ligados al ámbito.
func resolveAdaptersScoped(scope InstallScope, workspaceDir string, agentIDs []model.AgentID) []agents.Adapter {
	adapters := resolveAdapters(agentIDs)
	for i, adapter := range adapters {
		adapters[i] = scopedAdapter(scope, workspaceDir, adapter)
	}
	return adapters
}
