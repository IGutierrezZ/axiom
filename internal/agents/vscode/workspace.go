package vscode

import (
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/sharedskills"
)

// Distribución de proyecto de VS Code Copilot (documentación oficial de VS Code,
// code.visualstudio.com/docs/copilot/customization):
//
//   - .github/instructions/**/*.instructions.md: instrucciones de proyecto. Con
//     `applyTo: "**"` en la cabecera se aplican a todas las peticiones; sin ella (ni
//     description) solo se adjuntan a mano. Se suman a las de usuario.
//   - .github/skills/<nombre>/SKILL.md: skills de proyecto. Las sesiones locales (la
//     vista de chat) leen además .claude/skills y .agents/skills; Axiom escribe la
//     copia compartida en .agents/skills y no en .github/skills.
//   - .mcp.json en la raíz: servidores MCP del arnés de Copilot, con la clave raíz
//     `mcpServers` (code.visualstudio.com/docs/agents/reference/mcp-configuration). Es
//     el mismo fichero que lee Claude Code. .vscode/mcp.json (`servers`) es el formato
//     del arnés Local, deprecado y que se retira en febrero de 2027: el arnés de Copilot
//     no lo lee, y Axiom solo retira de él las entradas que escribió. Los dos heredan
//     la confianza del espacio de trabajo (Workspace Trust).
//   - .github/agents/*.agent.md: agentes personalizados. Axiom escribe uno por fase
//     SDD, con el modelo y el esfuerzo elegidos, y es el mismo fichero que lee
//     Copilot CLI desde ~/.copilot/agents. VS Code lee además .claude/agents, así que
//     en un proyecto con Claude Code también ve los agentes de Claude.
//   - .github/hooks/*.json: hooks del arnés de Copilot (en preview; exigen carpeta de
//     confianza y `chat.useHooks` los apaga). Axiom instala uno propio,
//     .github/hooks/axiom-branch-guard.json, con la guarda de rama (`axiom branch
//     guard --agent vscode-copilot`) en `preToolUse` y `postToolUse`. El arnés lee
//     además los hooks de .claude/settings.json del repositorio.
//   - .github/prompts/*.prompt.md: prompts. Axiom no instala ninguno, así que no hay
//     carpeta de proyecto que mapear para ellos.
//
// No se usan .github/copilot-instructions.md ni el AGENTS.md de la raíz: son
// ficheros del equipo, normalmente versionados y compartidos con otras
// herramientas. El perfil de usuario de VS Code (%APPDATA%\Code\User, ~/.config/Code/User,
// ~/Library/Application Support/Code/User) tampoco se toca en un proyecto.
const (
	// instructionsDirName es la carpeta de instrucciones de proyecto, relativa a .github.
	instructionsDirName = "instructions"
	// skillsDirName es la carpeta de skills de proyecto, relativa a .github.
	skillsDirName = "skills"
	// agentsDirName es la carpeta de agentes personalizados de proyecto, relativa a .github.
	agentsDirName = "agents"
	// vscodeDirName es la carpeta de configuración de proyecto de VS Code.
	vscodeDirName = ".vscode"
	// githubDirName es la carpeta donde VS Code busca instrucciones, skills y agentes.
	githubDirName = ".github"
)

// WorkspaceAdapter es el adaptador de VS Code Copilot ligado a la raíz de un
// proyecto. Para esa raíz sitúa instrucciones y MCP en la distribución de proyecto
// (.github/ y .mcp.json) y las skills en .agents/skills; para cualquier otra carpeta (la de usuario) da las
// rutas globales de siempre. Es un tipo aparte del adaptador global para que las
// capacidades que solo tienen sentido en el proyecto no existan en la instalación
// global.
type WorkspaceAdapter struct {
	*Adapter
	root string
}

// WithWorkspace devuelve el adaptador ligado a la raíz de un proyecto.
func (a *Adapter) WithWorkspace(root string) *WorkspaceAdapter {
	return &WorkspaceAdapter{Adapter: a, root: root}
}

// InWorkspace indica si dir es la raíz del proyecto a la que está ligado el
// adaptador.
func (a *WorkspaceAdapter) InWorkspace(dir string) bool {
	return a.root != "" && dir != "" && filepath.Clean(dir) == filepath.Clean(a.root)
}

// GlobalConfigDir es ~/.copilot en la carpeta de usuario y .github en el proyecto.
func (a *WorkspaceAdapter) GlobalConfigDir(dir string) string {
	if a.InWorkspace(dir) {
		return filepath.Join(dir, githubDirName)
	}
	return a.Adapter.GlobalConfigDir(dir)
}

// SystemPromptDir es la carpeta prompts del perfil de usuario de VS Code en la
// carpeta de usuario y .github/instructions en el proyecto.
func (a *WorkspaceAdapter) SystemPromptDir(dir string) string {
	if a.InWorkspace(dir) {
		return filepath.Join(dir, githubDirName, instructionsDirName)
	}
	return a.Adapter.SystemPromptDir(dir)
}

// SystemPromptFile es el fichero propio de Axiom con las instrucciones. En el
// proyecto es .github/instructions/axiom.instructions.md: su cabecera lleva
// `applyTo: "**"`, de modo que VS Code lo aplica siempre y convive con las
// instrucciones de usuario. .github/copilot-instructions.md es del equipo y no se toca.
func (a *WorkspaceAdapter) SystemPromptFile(dir string) string {
	if a.InWorkspace(dir) {
		return filepath.Join(a.SystemPromptDir(dir), "axiom.instructions.md")
	}
	return a.Adapter.SystemPromptFile(dir)
}

// LegacySystemPromptFiles lista los nombres anteriores del fichero de
// instrucciones. Solo existieron en la carpeta de usuario: Axiom nunca escribió un
// gentle-ai.instructions.md en un proyecto, así que allí no hay nada que retirar.
func (a *WorkspaceAdapter) LegacySystemPromptFiles(dir string) []string {
	if a.InWorkspace(dir) {
		return nil
	}
	return a.Adapter.LegacySystemPromptFiles(dir)
}

// SkillsDir es ~/.copilot/skills en la carpeta de usuario y, en el proyecto, la
// compartida .agents/skills: las sesiones locales de VS Code (la vista de chat) la
// leen junto a .github/skills y .claude/skills, y una sola copia evita las skills
// duplicadas. Las sesiones del arnés de Copilot (Copilot CLI, el agente en la nube)
// documentan .github/skills como su carpeta de proyecto; GitHub lista además
// .agents/skills entre las ubicaciones de proyecto de Copilot.
func (a *WorkspaceAdapter) SkillsDir(dir string) string {
	if a.InWorkspace(dir) {
		return sharedskills.Dir(dir)
	}
	return a.Adapter.SkillsDir(dir)
}

// SubAgentsDir es ~/.copilot/agents en la carpeta de usuario y .github/agents en
// el proyecto: las dos carpetas de agentes personalizados que lee VS Code.
func (a *WorkspaceAdapter) SubAgentsDir(dir string) string {
	if a.InWorkspace(dir) {
		return filepath.Join(dir, githubDirName, agentsDirName)
	}
	return a.Adapter.SubAgentsDir(dir)
}

// LegacyWorkspaceSkillsDirs lista la carpeta de skills que usaban las instalaciones
// de proyecto anteriores (.github/skills), ahora sustituida por .agents/skills. Solo se
// retira lo que se prueba que es de Axiom, y la carpeta solo si queda vacía: .github
// es del equipo. Fuera del proyecto no devuelve nada.
func (a *WorkspaceAdapter) LegacyWorkspaceSkillsDirs(dir string) []string {
	if !a.InWorkspace(dir) {
		return nil
	}
	return []string{filepath.Join(dir, githubDirName, skillsDirName)}
}

// SettingsPath es el settings.json del perfil de usuario en la carpeta de usuario y
// .vscode/settings.json en el proyecto. Axiom no escribe en ninguno de los dos:
// no hay permisos de VS Code que instalar.
func (a *WorkspaceAdapter) SettingsPath(dir string) string {
	if a.InWorkspace(dir) {
		return filepath.Join(dir, vscodeDirName, "settings.json")
	}
	return a.Adapter.SettingsPath(dir)
}

// MCPConfigPath es el mcp.json del perfil de usuario en la carpeta de usuario y
// <proyecto>/.mcp.json (clave `mcpServers`) en el proyecto.
func (a *WorkspaceAdapter) MCPConfigPath(dir string, serverName string) string {
	if a.InWorkspace(dir) {
		return filepath.Join(dir, ".mcp.json")
	}
	return a.Adapter.MCPConfigPath(dir, serverName)
}

// BranchGuardHooksFile es .github/hooks/axiom-branch-guard.json en el proyecto: el
// fichero propio de Axiom con el hook de la guarda de rama. Fuera del proyecto no
// devuelve nada: la instalación global no instala hooks en VS Code.
func (a *WorkspaceAdapter) BranchGuardHooksFile(dir string) string {
	if !a.InWorkspace(dir) {
		return ""
	}
	return filepath.Join(dir, githubDirName, "hooks", "axiom-branch-guard.json")
}

// LegacyWorkspaceMCPConfigPath es .vscode/mcp.json en el proyecto, donde las
// instalaciones anteriores declaraban los servidores MCP con la clave `servers`.
// Fuera del proyecto no devuelve nada: el perfil de usuario no cambia.
func (a *WorkspaceAdapter) LegacyWorkspaceMCPConfigPath(dir string) string {
	if !a.InWorkspace(dir) {
		return ""
	}
	return filepath.Join(dir, vscodeDirName, "mcp.json")
}

// WorkspaceUserConfigFiles lista los ficheros de configuración de proyecto del usuario
// que Axiom fusiona: .mcp.json, versionado y compartido con Claude Code. Si no se
// puede leer, se rechaza la operación antes de escribir en lugar de sustituirlo por
// un documento vacío. Fuera del proyecto no devuelve nada.
func (a *WorkspaceAdapter) WorkspaceUserConfigFiles(dir string) []string {
	if !a.InWorkspace(dir) {
		return nil
	}
	return []string{a.MCPConfigPath(dir, "")}
}
