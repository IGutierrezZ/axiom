package uninstall

import "path/filepath"

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
