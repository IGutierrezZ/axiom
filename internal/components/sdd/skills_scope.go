package sdd

import (
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
)

// Marcadores de ruta de skills de los assets de SDD (subagentes, comandos y
// prompts). Los assets no codifican `~/.claude/skills/...` ni equivalentes:
// escriben el marcador y la inyección lo resuelve según el ámbito de la
// instalación, de modo que un subagente nunca busque en el home del usuario
// unas skills que se instalaron solo en el workspace.
//
//   - skillsDirPlaceholder: directorio de skills con separador `/`.
//     Global: `~/<dir de skills del adaptador>`. Workspace: la misma ruta
//     relativa a la raíz del proyecto.
//   - skillsDirWindowsPlaceholder: la misma ruta con la grafía de Windows que
//     usa Kiro en sus agentes (`%USERPROFILE%\\.kiro\\skills`, con barra
//     invertida doble literal por estar en Markdown).
const (
	skillsDirPlaceholder        = "{{AXIOM_SKILLS_DIR}}"
	skillsDirWindowsPlaceholder = "{{AXIOM_SKILLS_DIR_WIN}}"
)

// installsInWorkspace indica si la inyección de rootDir cae en el workspace.
// Un adaptador sin modo workspace (Codex, VS Code, Trae, Windsurf,
// Antigravity) recibe el home aunque la instalación pida workspace, y
// entonces sus assets deben apuntar al home.
func installsInWorkspace(rootDir string, opts InjectOptions) bool {
	if strings.TrimSpace(opts.WorkspaceDir) == "" || strings.TrimSpace(rootDir) == "" {
		return false
	}
	return filepath.Clean(opts.WorkspaceDir) == filepath.Clean(rootDir)
}

// resolveSkillsDirPlaceholders sustituye los marcadores de ruta de skills por
// la ruta que corresponde al ámbito. La parte relativa sale de
// adapter.SkillsDir, la misma función que decide dónde se escriben las skills,
// así que la ruta que leen los subagentes no puede divergir de la instalada.
func resolveSkillsDirPlaceholders(content string, adapter agents.Adapter, workspace bool) string {
	if !strings.Contains(content, "{{AXIOM_SKILLS_DIR") {
		return content
	}
	rel := filepath.ToSlash(adapter.SkillsDir(""))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return content
	}
	posix := "~/" + rel
	windows := `%USERPROFILE%\\` + strings.ReplaceAll(rel, "/", `\\`)
	if workspace {
		posix = rel
		windows = strings.ReplaceAll(rel, "/", `\\`)
	}
	content = strings.ReplaceAll(content, skillsDirWindowsPlaceholder, windows)
	return strings.ReplaceAll(content, skillsDirPlaceholder, posix)
}
