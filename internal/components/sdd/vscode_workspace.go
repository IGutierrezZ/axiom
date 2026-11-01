package sdd

import (
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// vscodeWorkspaceConventionsClause es la frase del orquestador genérico que sitúa los
// ficheros de convención (engram-convention.md, persistence-contract.md...) en la
// carpeta de skills del agente, y para el proyecto da una ruta (.agent/skills) que
// VS Code Copilot no usa. Con la instalación global el texto no se toca.
const vscodeWorkspaceConventionsClause = "Convention files under the agent's global skills directory (global) or `.agent/skills/_shared/` (workspace):"

// adaptVSCodeOrchestratorForWorkspace reescribe la ruta de los ficheros de convención
// del orquestador para una instalación de proyecto de VS Code Copilot: están en la
// carpeta de skills del proyecto (.github/skills/_shared), que es donde los escribe el
// componente de skills. Con otro agente, o con la instalación global, no cambia nada.
func adaptVSCodeOrchestratorForWorkspace(content string, adapter agents.Adapter, workspaceRoot string) string {
	if adapter.Agent() != model.AgentVSCodeCopilot || strings.TrimSpace(workspaceRoot) == "" {
		return content
	}
	rel, err := filepath.Rel(workspaceRoot, adapter.SkillsDir(workspaceRoot))
	if err != nil || rel == "" {
		return content
	}
	return strings.ReplaceAll(content, vscodeWorkspaceConventionsClause,
		"Convention files under the project skills directory `"+filepath.ToSlash(rel)+"/_shared/`:")
}
