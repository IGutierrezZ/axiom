package agents

import "github.com/IGutierrezZ/axiom/v3/internal/agents/sharedskills"

// IsSharedWorkspaceSkillsDir indica si skillsDir es la carpeta de skills compartida
// (.agents/skills) de la raíz del proyecto root: la que comparten los agentes que la
// leen (Codex, VS Code Copilot, OpenCode, Kilo, Gemini CLI y Cursor) en lugar de tener
// una copia cada uno. Claude Code no la lee y conserva .claude/skills.
func IsSharedWorkspaceSkillsDir(root, skillsDir string) bool {
	return sharedskills.Is(root, skillsDir)
}
