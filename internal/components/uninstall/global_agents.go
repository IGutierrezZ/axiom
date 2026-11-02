package uninstall

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/sdd"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// Restos de una instalación global que quedaban en la carpeta de usuario tras
// `uninstall --scope global --all`:
//
//   - Kilo: ~/.config/kilo/opencode.json (agentes con la marca de Axiom, servidores
//     MCP, share) y ~/.config/kilo/plugins/*.ts. El adaptador nunca pasó por el
//     código de OpenCode en el ámbito global (heredado de la versión anterior).
//   - Kiro: ~/.kiro/steering/axiom.md, que lleva la persona con
//     `inclusion: always`. Sus secciones con marcadores sí se retiraban, pero la
//     persona entera, sin marcadores, no (también heredado).
//   - Codex: la tabla [mcp_servers.context7] y las claves de delegación
//     (features.multi_agent y agents.max_threads/max_depth) de ~/.codex/config.toml.
//     Solo se retiraba la tabla de Engram y las claves de sus ficheros de instrucciones.
//
// Las funciones de este fichero aplican a esos restos el criterio de la
// desinstalación de workspace: solo se retira lo que tiene la forma de Axiom y lo que
// el usuario haya editado se conserva con aviso. El diagnóstico (GlobalAgentLeftovers)
// usa los mismos limpiadores que la desinstalación, de modo que lo que se informa es
// exactamente lo que `axiom uninstall --scope global --all` retira.

// usesManagedMCPShape dice si la desinstalación global del agente retira sus servidores
// MCP comprobando la forma de Axiom en lugar de borrar la clave por nombre. Lo hace
// con Kilo, que comparte el formato `mcp` de OpenCode pero nunca tuvo código propio
// de retirada; el resto de agentes conserva su comportamiento de siempre.
func usesManagedMCPShape(adapter agents.Adapter) bool {
	return adapter.Agent() == model.AgentKilocode
}

// rewriteMCPServerEntry retira la entrada parent.name (engram o context7) de un
// fichero de configuración de MCP global. La clave se retira por nombre, como
// siempre, pero el fichero que queda vacío solo se borra si la entrada retirada tiene
// la forma de Axiom (mcpEntryIsManaged).
//
// Criterio de borrado: la desinstalación no puede saber si Axiom creó el fichero o ya
// existía antes (el estado no lo registra). Lo más conservador que puede comprobarse
// es que el fichero no contuviera nada más que lo que Axiom escribió, en su forma
// original: un documento que se vacía al quitar un servidor con la forma de Axiom. Si la
// entrada tenía otra forma (la definió el usuario), el fichero era suyo y se deja
// como documento vacío ({}). Un fichero del usuario que estuviera vacío antes de
// instalar es indistinguible del que crea Axiom y se sigue borrando.
func rewriteMCPServerEntry(path string, parent jsonPath, name string) operation {
	entry := append(slices.Clone(parent), name)
	return rewriteJSONGuarded(path, func(raw []byte) ([]byte, bool, error) {
		return removeJSONPaths(raw, entry)
	}, func(raw []byte) bool {
		return mcpEntryIsManaged(raw, parent, name)
	})
}

// mcpEntryIsManaged dice si parent.name del documento raw existe y tiene la forma de
// la entrada que escribe Axiom.
func mcpEntryIsManaged(raw []byte, parent jsonPath, name string) bool {
	root, err := unmarshalJSONObject(raw)
	if err != nil {
		return false
	}
	var container any = root
	for _, key := range parent {
		object, ok := container.(map[string]any)
		if !ok {
			return false
		}
		container = object[key]
	}
	servers, ok := container.(map[string]any)
	if !ok {
		return false
	}
	entry, declared := servers[name]
	return declared && isManagedMCPServer(name, entry)
}

// globalSteeringFiles son los ficheros de steering que Axiom escribe enteros en la
// carpeta de usuario de un agente (Kiro): llevan la cabecera `inclusion: always` y
// la persona sin marcadores.
func (s *Service) globalSteeringFiles(adapter agents.Adapter) []string {
	if s.workspace || !adapter.SupportsSystemPrompt() || adapter.SystemPromptStrategy() != model.StrategySteeringFile {
		return nil
	}
	return systemPromptFiles(adapter, s.homeDir)
}

// isManagedSteeringContent dice si un fichero de steering conserva solo lo que Axiom
// escribió: sus secciones con marcadores, la cabecera o la persona incrustada. Es el
// mismo criterio con el que sweepInstructionsFile decide borrarlo.
func isManagedSteeringContent(content string) bool {
	if strings.Contains(content, "<!-- axiom:") || strings.Contains(content, "<!-- gentle-ai:") {
		return true
	}
	body := strings.TrimSpace(stripManagedFrontmatter(content))
	return body == "" || slices.Contains(embeddedPersonas(), body)
}

// GlobalLeftover es un resto de una instalación global de un agente en la carpeta de
// usuario.
type GlobalLeftover struct {
	Agent model.AgentID
	// Label dice qué es, para el diagnóstico (`axiom doctor`).
	Label string
	Paths []string
}

// GlobalAgentLeftovers lista lo que Axiom dejó en HOME para Kilo, Kiro y Codex y que
// `axiom uninstall --scope global --all` retira. Solo lee. Tras desinstalar no
// devuelve nada: usa los limpiadores de la propia desinstalación y comprueba la forma
// de Axiom, de modo que un fichero que el usuario editó, y que la desinstalación
// conserva, no se informa como resto.
func GlobalAgentLeftovers(homeDir string) []GlobalLeftover {
	var found []GlobalLeftover

	if adapter, err := agents.NewAdapter(model.AgentKilocode); err == nil {
		pluginsDir := filepath.Join(adapter.GlobalConfigDir(homeDir), "plugins")
		var plugins []string
		for _, name := range kiloPluginNames() {
			path := filepath.Join(pluginsDir, name)
			if info, err := os.Lstat(path); err == nil && !info.IsDir() {
				plugins = append(plugins, path)
			}
		}
		if len(plugins) > 0 {
			found = append(found, GlobalLeftover{Agent: model.AgentKilocode, Label: "plugins de Kilo en HOME", Paths: plugins})
		}
		if path := adapter.SettingsPath(homeDir); path != "" && kiloConfigHasAxiomEntries(path) {
			found = append(found, GlobalLeftover{Agent: model.AgentKilocode, Label: "configuración de Kilo en HOME", Paths: []string{path}})
		}
	}

	if adapter, err := agents.NewAdapter(model.AgentKiroIDE); err == nil && adapter.SupportsSystemPrompt() {
		for _, path := range systemPromptFiles(adapter, homeDir) {
			content, err := os.ReadFile(path)
			if err == nil && isManagedSteeringContent(string(content)) {
				found = append(found, GlobalLeftover{Agent: model.AgentKiroIDE, Label: "steering de Kiro en HOME (persona con inclusion: always)", Paths: []string{path}})
			}
		}
	}

	if adapter, err := agents.NewAdapter(model.AgentCodex); err == nil {
		if path := adapter.MCPConfigPath(homeDir, "engram"); path != "" {
			if content, err := os.ReadFile(path); err == nil {
				normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
				if removeCodexMultiAgentKeys(removeCodexContext7Table(normalized)) != normalized {
					found = append(found, GlobalLeftover{Agent: model.AgentCodex, Label: "config.toml de Codex en HOME (context7, features.multi_agent, agents.max_*)", Paths: []string{path}})
				}
			}
		}
	}
	return found
}

// kiloPluginNames son los plugins que la instalación escribe en la carpeta de
// plugins de Kilo, los mismos que retira la desinstalación.
func kiloPluginNames() []string {
	return append([]string{"background-agents.ts"}, sdd.OpenCodePluginLifecycleNames(model.AgentKilocode)...)
}

// kiloConfigHasAxiomEntries dice si el opencode.json de Kilo conserva agentes con la
// marca de Axiom o servidores MCP con la forma de Axiom.
func kiloConfigHasAxiomEntries(path string) bool {
	raw, err := readManagedFile(path)
	if err != nil {
		return false
	}
	if _, changed, err := removeManagedAgentEntries(raw); err == nil && changed {
		return true
	}
	for _, name := range []string{"engram", "context7"} {
		if mcpEntryIsManaged(raw, jsonPath{"mcp"}, name) {
			return true
		}
	}
	return false
}
