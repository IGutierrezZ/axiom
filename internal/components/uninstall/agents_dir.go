package uninstall

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// subAgentsDirStaysInUse dice si la carpeta de agentes sigue siendo de otro agente
// que permanece instalado en el mismo ámbito y que escribe ahí los mismos ficheros.
// VS Code Copilot y Copilot CLI comparten un único `.agent.md` por fase en
// ~/.copilot/agents; si se retira solo uno de los dos, los agentes se quedan para
// el otro. Es la misma idea que skillsDirStaysInUse.
func (s *Service) subAgentsDirStaysInUse(agentsDir string) bool {
	if agentsDir == "" {
		return false
	}
	root := s.homeDir
	if s.workspace {
		root = s.workspaceDir
	}
	for _, other := range s.otherInstalledAgentsInScope() {
		adapter, ok := s.adapter(other)
		if !ok || !adapter.SupportsSubAgents() {
			continue
		}
		if filepath.Clean(adapter.SubAgentsDir(root)) == filepath.Clean(agentsDir) {
			return true
		}
	}
	return false
}

// copilotProjectAgentsStayInUse dice si, en el ámbito de proyecto, los agentes por fase de
// Copilot (.github/agents) siguen siendo de la otra mitad de la pareja: VS Code Copilot y
// Copilot CLI los comparten, y el CLI los lee aunque no tenga modo workspace propio. Si se
// retira solo uno de los dos y el otro sigue en el estado del proyecto, los agentes se quedan.
func (s *Service) copilotProjectAgentsStayInUse(removing model.AgentID) bool {
	counterpart := model.AgentGitHubCopilotCLI
	if removing == model.AgentGitHubCopilotCLI {
		counterpart = model.AgentVSCodeCopilot
	}
	return slices.Contains(s.otherInstalledAgentsInScope(), counterpart)
}

// copilotProjectAgentOperations retira de .github/agents los agentes por fase de Axiom (por
// el nombre de las plantillas) y la carpeta si queda vacía. Los agentes del equipo se quedan.
func (s *Service) copilotProjectAgentOperations() (ops []operation, targets []string, err error) {
	entries, err := fs.ReadDir(assets.FS, "copilot/agents")
	if err != nil {
		return nil, nil, fmt.Errorf("read embedded sub-agents: %w", err)
	}
	agentsDir := filepath.Join(s.workspaceDir, ".github", "agents")
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(agentsDir, entry.Name())
		targets = append(targets, path)
		ops = append(ops, removeFile(path))
	}
	return append(ops, removeDirIfEmpty(agentsDir)), targets, nil
}
