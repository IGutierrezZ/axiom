package codex

import (
	"path/filepath"
)

// Distribución de proyecto de Codex (documentación oficial,
// https://learn.chatgpt.com/docs/config-file/config-basic, config-advanced,
// config-reference, hooks, agent-configuration/subagents y build-skills):
//
//   - <proyecto>/.codex/config.toml: configuración de proyecto, solo en proyectos de
//     confianza. Prevalece sobre ~/.codex/config.toml clave a clave y admite
//     mcp_servers, features, agents y developer_instructions. Codex ignora en ella
//     (y avisa al arrancar) openai_base_url, chatgpt_base_url, apps_mcp_product_sku,
//     model_provider, model_providers, notify, profile, profiles,
//     experimental_realtime_ws_base_url y otel: Axiom nunca las escribe aquí.
//   - developer_instructions (cadena): «instrucciones de desarrollador adicionales
//     inyectadas en la sesión», que se añaden a las instrucciones base. Es donde van
//     las instrucciones de Axiom (orquestador, persona, protocolo de memoria).
//     model_instructions_file, en cambio, las sustituye, y AGENTS.md es del equipo.
//   - <proyecto>/.codex/hooks.json: hooks de proyecto, que además se aprueban uno a
//     uno por hash.
//   - <proyecto>/.agents/skills: skills de proyecto. Codex no documenta un
//     .codex/skills de proyecto; los lee .agents/skills desde la carpeta de trabajo
//     hasta la raíz del repositorio.
//   - AGENTS.md, de la raíz del proyecto hacia el directorio actual, no
//     .codex/AGENTS.md: donde Axiom escribía las instrucciones con --scope workspace
//     y Codex no leía.
//
// Todo esto exige que el proyecto sea de confianza.
const (
	// projectInstructionsFileName es el fichero de Axiom que reúne las instrucciones
	// (persona, orquestador, protocolo de memoria, guía de enrutado). Codex no lo lee:
	// es la fuente de la que Axiom copia el bloque gestionado a developer_instructions.
	projectInstructionsFileName = "axiom-instructions.md"
	// developerInstructionsKey es la clave de config.toml que recibe el texto.
	developerInstructionsKey = "developer_instructions"
)

// WorkspaceAdapter es el adaptador de Codex ligado a la raíz de un proyecto. Para
// esa raíz sitúa las instrucciones en un fichero propio de .codex (fuente de
// developer_instructions) y las skills en .agents/skills; para cualquier otra
// carpeta (la de usuario) da las rutas globales de siempre. Es un tipo aparte del
// adaptador global para que las capacidades que solo tienen sentido en el proyecto
// (registro en developer_instructions y retirada del .codex/AGENTS.md antiguo) no
// existan en la instalación global.
type WorkspaceAdapter struct {
	*Adapter
	root string
}

// WithWorkspace devuelve el adaptador ligado a la raíz de un proyecto.
func (a *Adapter) WithWorkspace(root string) *WorkspaceAdapter {
	return &WorkspaceAdapter{Adapter: a, root: root}
}

// inWorkspace indica si dir es la raíz del proyecto a la que está ligado el
// adaptador.
func (a *WorkspaceAdapter) inWorkspace(dir string) bool {
	return a.root != "" && dir != "" && filepath.Clean(dir) == filepath.Clean(a.root)
}

// SystemPromptFile es ~/.codex/AGENTS.md en la carpeta de usuario. En el proyecto
// es el fichero propio de Axiom (.codex/axiom-instructions.md), cuyo contenido se
// registra en developer_instructions: ni el AGENTS.md de la raíz, que es del equipo,
// ni .codex/AGENTS.md, que Codex no lee, sirven.
func (a *WorkspaceAdapter) SystemPromptFile(dir string) string {
	if a.inWorkspace(dir) {
		return filepath.Join(dir, ".codex", projectInstructionsFileName)
	}
	return a.Adapter.SystemPromptFile(dir)
}

// SkillsDir es ~/.codex/skills en la carpeta de usuario. En el proyecto es
// .agents/skills, la única carpeta de skills de proyecto que documenta Codex.
func (a *WorkspaceAdapter) SkillsDir(dir string) string {
	if a.inWorkspace(dir) {
		return filepath.Join(dir, ".agents", "skills")
	}
	return a.Adapter.SkillsDir(dir)
}

// LegacySystemPromptFiles lista el .codex/AGENTS.md que escribían las instalaciones
// workspace anteriores. Solo se retira si se prueba que es de Axiom.
func (a *WorkspaceAdapter) LegacySystemPromptFiles(dir string) []string {
	if !a.inWorkspace(dir) {
		return nil
	}
	return []string{filepath.Join(dir, ".codex", "AGENTS.md")}
}

// WorkspaceInlineInstructions indica dónde se registra el texto de Axiom: el
// fichero de configuración de proyecto (.codex/config.toml) y la clave
// (developer_instructions). ok es false fuera del proyecto.
func (a *WorkspaceAdapter) WorkspaceInlineInstructions(dir string) (configPath, key string, ok bool) {
	if !a.inWorkspace(dir) {
		return "", "", false
	}
	return a.Adapter.MCPConfigPath(dir, ""), developerInstructionsKey, true
}
