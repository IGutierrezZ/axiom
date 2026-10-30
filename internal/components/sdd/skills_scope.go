package sdd

import (
	"encoding/json"
	"fmt"
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

// workspaceRootFor devuelve la raíz del proyecto cuando la inyección de rootDir
// cae en el workspace, y "" cuando es global.
func workspaceRootFor(rootDir string, opts InjectOptions) string {
	if installsInWorkspace(rootDir, opts) {
		return rootDir
	}
	return ""
}

// resolveSkillsDirPlaceholders sustituye los marcadores de ruta de skills por
// la ruta que corresponde al ámbito. La parte relativa sale de
// adapter.SkillsDir, la misma función que decide dónde se escriben las skills,
// así que la ruta que leen los subagentes no puede divergir de la instalada.
//
// workspaceRoot vacío es el ámbito global: la ruta relativa a la carpeta de
// usuario con prefijo `~/`. Con una raíz de proyecto es la ruta relativa a esa
// raíz, tal como la resuelve el adaptador para el proyecto (OpenCode y Kilo la
// tienen en .opencode/skills y .kilo/skills, no bajo .config).
func resolveSkillsDirPlaceholders(content string, adapter agents.Adapter, workspaceRoot string) string {
	if !strings.Contains(content, "{{AXIOM_SKILLS_DIR") {
		return content
	}
	workspace := strings.TrimSpace(workspaceRoot) != ""
	var rel string
	if workspace {
		if r, err := filepath.Rel(workspaceRoot, adapter.SkillsDir(workspaceRoot)); err == nil {
			rel = filepath.ToSlash(r)
		}
	}
	if rel == "" {
		rel = strings.TrimPrefix(filepath.ToSlash(adapter.SkillsDir("")), "/")
	}
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

// resolveOverlaySkillsDir resuelve el marcador de ruta de skills en un overlay
// JSON de OpenCode o Kilo, donde el marcador vive dentro de cadenas JSON. Reutiliza
// resolveSkillsDirPlaceholders y comprueba después que el documento sigue siendo
// JSON válido, de modo que una ruta con caracteres que exigieran escape nunca deje
// un opencode.json corrupto: es preferible fallar la inyección.
func resolveOverlaySkillsDir(content string, adapter agents.Adapter, workspaceRoot string) (string, error) {
	resolved := resolveSkillsDirPlaceholders(content, adapter, workspaceRoot)
	if resolved == content {
		return content, nil
	}
	if !json.Valid([]byte(resolved)) {
		return "", fmt.Errorf("resolver la ruta de skills del overlay de %s: el resultado no es JSON válido", adapter.Agent())
	}
	return resolved, nil
}
