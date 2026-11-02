package opencode

import (
	"path/filepath"
)

// Distribución de proyecto de OpenCode (documentación oficial, opencode.ai/docs):
//
//   - opencode.json (u opencode.jsonc) en la raíz del proyecto: claves mcp,
//     permission, agent, instructions...
//   - .opencode/{skills,commands,agents,plugins}: recursos del proyecto.
//   - instructions: lista de rutas de ficheros que se suman a AGENTS.md.
//
// OpenCode no lee <proyecto>/.config/opencode, que era donde escribía Axiom
// cuando trataba la raíz del proyecto como si fuera la carpeta de usuario.
const (
	// projectConfigDirName es la carpeta de recursos del proyecto.
	projectConfigDirName = ".opencode"
	// projectInstructionsFileName es el fichero propio de Axiom con las
	// instrucciones (persona, protocolo de memoria). AGENTS.md de la raíz es un
	// fichero compartido y versionado del equipo: Axiom no lo toca.
	projectInstructionsFileName = "axiom.md"
)

// WithWorkspace devuelve una copia del adaptador ligada a la raíz de un
// proyecto. Las rutas que se piden con esa raíz salen en la distribución de
// proyecto de OpenCode; las que se piden con cualquier otra carpeta (la de
// usuario, por ejemplo) siguen siendo las globales de siempre.
func (a *Adapter) WithWorkspace(root string) *Adapter {
	bound := *a
	bound.workspaceRoot = root
	return &bound
}

// inWorkspace indica si dir es la raíz del proyecto a la que está ligado el
// adaptador.
func (a *Adapter) inWorkspace(dir string) bool {
	return a.workspaceRoot != "" && dir != "" && filepath.Clean(dir) == filepath.Clean(a.workspaceRoot)
}

// configDir es la carpeta de recursos: .opencode en el proyecto, la carpeta de
// configuración global en cualquier otro caso.
func (a *Adapter) configDir(dir string) string {
	if a.inWorkspace(dir) {
		return filepath.Join(dir, projectConfigDirName)
	}
	return ConfigPath(dir)
}

// settingsDir es la carpeta donde vive el fichero de configuración: la raíz del
// proyecto, donde OpenCode busca opencode.json, o la carpeta global.
func (a *Adapter) settingsDir(dir string) string {
	if a.inWorkspace(dir) {
		return dir
	}
	return ConfigPath(dir)
}

// WorkspacePromptsDir devuelve la carpeta de los prompts compartidos de SDD
// cuando dir es la raíz del proyecto ligada. ok es false fuera del proyecto:
// el llamante usa entonces la ruta global.
func (a *Adapter) WorkspacePromptsDir(dir string) (string, bool) {
	if !a.inWorkspace(dir) {
		return "", false
	}
	return filepath.Join(dir, projectConfigDirName, "prompts", "sdd"), true
}

// WorkspaceInstructions indica dónde se registra el fichero de instrucciones de
// Axiom: el fichero de configuración que lo declara y la entrada de su lista
// instructions, relativa a la raíz del proyecto. ok es false fuera del proyecto.
func (a *Adapter) WorkspaceInstructions(dir string) (settingsPath, entry string, ok bool) {
	if !a.inWorkspace(dir) {
		return "", "", false
	}
	return a.SettingsPath(dir), filepath.ToSlash(filepath.Join(projectConfigDirName, projectInstructionsFileName)), true
}

// LegacyWorkspaceSkillsDirs lista la carpeta de skills que usaban las instalaciones
// de proyecto anteriores (.opencode/skills), ahora sustituida por .agents/skills.
// OpenCode las lee todas y vería cada skill dos veces. Solo se retira lo que se
// prueba que es de Axiom. Fuera del proyecto no devuelve nada.
func (a *Adapter) LegacyWorkspaceSkillsDirs(dir string) []string {
	if !a.inWorkspace(dir) {
		return nil
	}
	return []string{filepath.Join(dir, projectConfigDirName, "skills")}
}
