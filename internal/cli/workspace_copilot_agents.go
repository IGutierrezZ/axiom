package cli

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/sdd"
	"github.com/IGutierrezZ/axiom/v3/internal/copilotconfig"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// Los agentes por fase de Copilot (.agent.md, con el modelo y el esfuerzo elegidos)
// viven solo en el proyecto, en .github/agents: los lee VS Code y los lee Copilot CLI.
// No se escriben en ~/.copilot/agents porque Copilot CLI da prioridad al agente
// personal frente al del proyecto cuando coinciden los identificadores, y el modelo por
// fase se guarda en el estado del proyecto: un agente global sin modelo taparía la
// elección del proyecto.
//
// Los escribe un único responsable. Con VS Code Copilot instalado en el proyecto es su
// adaptador, ligado al workspace (sdd.Inject). Cuando el proyecto solo tiene Copilot CLI,
// que no tiene modo workspace propio, es copilotProjectAgentsFromCLI. Nunca los dos: el
// resultado sería el mismo fichero, pero así se escribe una sola vez.

// copilotProjectAgentsFromCLI indica si el sync o la instalación de un proyecto debe
// escribir .github/agents por Copilot CLI: ámbito workspace, sin VS Code Copilot entre
// los adaptadores del proyecto (que ya los escribe), y con Copilot CLI presente, sea
// porque consta en el estado del proyecto o porque el usuario lo tiene configurado.
func copilotProjectAgentsFromCLI(homeDir, workspaceDir string, scope InstallScope, adapters []agents.Adapter) bool {
	if scope != ScopeWorkspace || workspaceDir == "" {
		return false
	}
	for _, adapter := range adapters {
		if adapter.Agent() == model.AgentVSCodeCopilot {
			return false
		}
	}
	if current, err := StateStoreFor(scope, homeDir, workspaceDir).Read(); err == nil {
		for _, id := range current.InstalledAgents {
			if model.AgentID(id) == model.AgentGitHubCopilotCLI {
				return true
			}
		}
	}
	return copilotconfig.HasCLIState(copilotconfig.Root(homeDir))
}

// injectCopilotProjectAgents escribe .github/agents con las asignaciones de la selección.
func injectCopilotProjectAgents(homeDir, workspaceDir string, selection model.Selection) (sdd.InjectionResult, error) {
	adapter, err := newScopedAdapter(ScopeWorkspace, workspaceDir, model.AgentVSCodeCopilot)
	if err != nil {
		return sdd.InjectionResult{}, err
	}
	return sdd.InjectCopilotProjectAgents(workspaceDir, adapter, sdd.InjectOptions{
		CopilotPhaseAssignments:   selection.CopilotPhaseAssignments,
		CodeGraphGuidanceMarkdown: codeGraphGuidanceMarkdownForSDD(homeDir, selection.CommunityTools),
	})
}

// syncCopilotProjectAgentsOnly completa un sync de proyecto sin adaptadores que
// escribir: si el proyecto solo tiene Copilot CLI (que no entra en el ámbito workspace),
// regenera igualmente .github/agents. Sin SDD en la selección o sin Copilot CLI presente
// devuelve el resultado tal cual.
func syncCopilotProjectAgentsOnly(homeDir, workspaceDir string, scope InstallScope, selection model.Selection, result SyncResult) (SyncResult, error) {
	if !hasComponent(selection.Components, model.ComponentSDD) || !copilotProjectAgentsFromCLI(homeDir, workspaceDir, scope, nil) {
		return result, nil
	}
	res, err := injectCopilotProjectAgents(homeDir, workspaceDir, selection)
	if err != nil {
		return result, fmt.Errorf("sync copilot project agents: %w", err)
	}
	result.ChangedFiles = dedupPaths(res.Files)
	result.FilesChanged = len(result.ChangedFiles)
	result.NoOp = !res.Changed
	return result, nil
}
