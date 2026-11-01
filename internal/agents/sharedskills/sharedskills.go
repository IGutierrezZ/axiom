// Package sharedskills fija la carpeta de skills de proyecto que comparten los
// agentes que leen .agents/skills. Vive en un paquete aparte para que los
// adaptadores (que no pueden importar el paquete agents) y los componentes den la
// misma ruta.
package sharedskills

import "path/filepath"

// Dir es la carpeta de skills de proyecto de la raíz root: .agents/skills. La leen
// Codex, VS Code Copilot, OpenCode, Kilo, Gemini CLI y Cursor, así que una sola copia
// sirve a todos; con una carpeta propia por agente, cada uno leería además las de los
// demás y vería las skills duplicadas. Claude Code no la lee y conserva .claude/skills.
func Dir(root string) string {
	return filepath.Join(root, ".agents", "skills")
}

// Is indica si skillsDir es la carpeta compartida de la raíz root.
func Is(root, skillsDir string) bool {
	if root == "" || skillsDir == "" {
		return false
	}
	return filepath.Clean(skillsDir) == filepath.Clean(Dir(root))
}
