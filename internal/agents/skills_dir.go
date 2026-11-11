package agents

import (
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/sharedskills"
)

// IsSharedWorkspaceSkillsDir indica si skillsDir es la carpeta de skills compartida
// (.agents/skills) de la raíz del proyecto root: la que comparten los agentes que la
// leen (Codex, VS Code Copilot, OpenCode, Kilo, Gemini CLI y Cursor) en lugar de tener
// una copia cada uno. Claude Code no la lee y conserva .claude/skills.
func IsSharedWorkspaceSkillsDir(root, skillsDir string) bool {
	return sharedskills.Is(root, skillsDir)
}

// SharesSkillsDirWithAnotherAdapter indica si algún otro adaptador registrado
// escribe sus skills en la misma carpeta que adapter para homeDir.
//
// Hoy ocurre con VS Code Copilot y Copilot CLI, que comparten ~/.copilot/skills. Es
// una propiedad estática del registro y no de qué agentes estén seleccionados: el
// contenido de la carpeta no puede depender de quién la escriba ni de cuántos haya,
// o dos agentes se pisarían la copia en cada sync.
func SharesSkillsDirWithAnotherAdapter(homeDir string, adapter Adapter) bool {
	if !adapter.SupportsSkills() {
		return false
	}
	dir := filepath.Clean(adapter.SkillsDir(homeDir))
	if dir == "." {
		return false
	}
	for _, id := range defaultAgentIDs {
		if id == adapter.Agent() {
			continue
		}
		other, err := NewAdapter(id)
		if err != nil || !other.SupportsSkills() {
			continue
		}
		if filepath.Clean(other.SkillsDir(homeDir)) == dir {
			return true
		}
	}
	return false
}
