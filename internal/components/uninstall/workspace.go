package uninstall

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/components/agentguidance"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	opencodeactivation "github.com/IGutierrezZ/axiom/v3/internal/opencode"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

const (
	// skillsIndexSectionID es la sección que el registro de skills mantiene en el
	// AGENTS.md del proyecto.
	skillsIndexSectionID = "skills-index"
	// context7RemoteURL es la URL del servidor remoto de Context7 que Axiom declara.
	context7RemoteURL = "https://mcp.context7.com/mcp"
	// codexCompactPromptValue es el valor que Axiom escribe en
	// experimental_compact_prompt_file de la configuración de proyecto de Codex.
	codexCompactPromptValue = "engram-compact-prompt.md"
)

// sddMarkdownSectionIDs son las secciones del prompt que retira el componente SDD:
// el orquestador, el modo TDD estricto y las guías que Axiom entrega junto a él
// (enrutado y autorización remota).
var sddMarkdownSectionIDs = []string{
	"sdd-orchestrator",
	"strict-tdd-mode",
	agentguidance.RoutingSectionID,
	filemerge.RemoteAuthorizationSectionID,
	filemerge.CodeGraphGuidanceSectionID,
}

// allAxiomSectionIDs son todas las secciones de prompt que Axiom escribe.
func allAxiomSectionIDs() []string {
	ids := []string{
		filemerge.PersonaSectionID,
		filemerge.EngramProtocolSectionID,
		filemerge.EngramPassiveCaptureSectionID,
	}
	return append(ids, sddMarkdownSectionIDs...)
}

// skillRegistryGeneratedFiles son los ficheros que el registro de skills genera en
// .atl del proyecto.
var skillRegistryGeneratedFiles = []string{"skill-registry.md", ".skill-registry.cache.json"}

// managedInstructionFileNames son los ficheros de instrucciones que Axiom escribe
// enteros en el proyecto (no comparte con el usuario): si el usuario los editó, se
// conservan con aviso.
var managedInstructionFileNames = []string{
	"axiom.md",
	"AXIOM.md",
	"axiom-instructions.md",
	"axiom.instructions.md",
}

// settingsTargets son los ficheros de configuración del agente que la desinstalación
// reescribe. En el ámbito global sigue la resolución de siempre (incluida la
// configuración heredada de OpenCode); en el proyecto es solo el fichero del adaptador
// ligado a la raíz.
func (s *Service) settingsTargets(adapter agents.Adapter) []string {
	if !s.workspace {
		return settingsTargets(s.homeDir, adapter)
	}
	if path := adapter.SettingsPath(s.workspaceDir); path != "" {
		return []string{path}
	}
	return nil
}

// otherAgentsInProject devuelve los agentes que el estado de proyecto da por
// instalados y que esta desinstalación no retira.
func (s *Service) otherAgentsInProject() []model.AgentID {
	if !s.workspace {
		return nil
	}
	return s.otherInstalledAgentsInScope()
}

// otherInstalledAgentsInScope devuelve los agentes que el estado del ámbito (global o
// proyecto) da por instalados y que esta desinstalación no retira. Sin estado legible
// devuelve nil: no hay nadie que Axiom sepa que depende de lo que se va a quitar.
func (s *Service) otherInstalledAgentsInScope() []model.AgentID {
	current, err := s.readInstallState()
	if err != nil {
		return nil
	}
	var others []model.AgentID
	for _, installed := range current.InstalledAgents {
		if !slices.Contains(s.removing, model.AgentID(installed)) {
			others = append(others, model.AgentID(installed))
		}
	}
	return others
}

// skillsDirStaysInUse dice si la carpeta de skills sigue siendo de otro agente que
// permanece instalado en el mismo ámbito. En el proyecto, varios agentes comparten
// .agents/skills; en el global, VS Code Copilot y Copilot CLI comparten
// ~/.copilot/skills. Si se retira solo uno, las skills se quedan para los demás.
func (s *Service) skillsDirStaysInUse(skillDir string) bool {
	if skillDir == "" {
		return false
	}
	root := s.homeDir
	if s.workspace {
		root = s.workspaceDir
	}
	for _, other := range s.otherInstalledAgentsInScope() {
		adapter, ok := s.adapter(other)
		if !ok || !adapter.SupportsSkills() {
			continue
		}
		if filepath.Clean(adapter.SkillsDir(root)) == filepath.Clean(skillDir) {
			return true
		}
	}
	return false
}

// isManagedMCPServer dice si la entrada de un servidor MCP tiene la forma que
// Axiom escribe para ese servidor. Una entrada del usuario con el mismo nombre pero
// otra definición no se retira.
func isManagedMCPServer(name string, value any) bool {
	entry, ok := value.(map[string]any)
	if !ok {
		return false
	}
	switch name {
	case "engram":
		if command, ok := entry["command"].(string); ok {
			args, _ := entry["args"].([]any)
			return isEngramBinary(command) && len(args) > 0 && args[0] == "mcp"
		}
		// OpenCode y Kilo: command es la lista completa.
		if command, ok := entry["command"].([]any); ok && len(command) >= 2 {
			first, _ := command[0].(string)
			return isEngramBinary(first) && command[1] == "mcp"
		}
	case "context7":
		for _, key := range []string{"url", "serverUrl"} {
			if url, ok := entry[key].(string); ok && strings.TrimRight(url, "/") == context7RemoteURL {
				return true
			}
		}
		if entry["command"] == "npx" {
			args, _ := entry["args"].([]any)
			hasPackage, hasBinary := false, false
			for _, arg := range args {
				text, _ := arg.(string)
				hasPackage = hasPackage || strings.HasPrefix(text, "--package=@upstash/context7-mcp@")
				hasBinary = hasBinary || text == "context7-mcp"
			}
			return hasPackage && hasBinary
		}
	}
	return false
}

func isEngramBinary(command string) bool {
	base := strings.ToLower(filepath.Base(strings.ReplaceAll(command, `\`, "/")))
	return base == "engram" || base == "engram.exe"
}

// rewriteManagedMCPServer retira de un fichero JSON la entrada parent.name solo si
// es la que escribe Axiom. Si hay una entrada con ese nombre pero distinta, la
// conserva y avisa; un contenedor que se queda vacío se quita con ella.
func (s *Service) rewriteManagedMCPServer(path string, parent jsonPath, name string) operation {
	return rewriteJSONWith(path, func(raw []byte) ([]byte, bool, error) {
		root, err := unmarshalJSONObject(raw)
		if err != nil {
			return nil, false, err
		}
		var container any = root
		for _, key := range parent {
			object, ok := container.(map[string]any)
			if !ok {
				return raw, false, nil
			}
			container = object[key]
		}
		servers, ok := container.(map[string]any)
		if !ok {
			return raw, false, nil
		}
		entry, declared := servers[name]
		if !declared {
			return raw, false, nil
		}
		if !isManagedMCPServer(name, entry) {
			s.notices = append(s.notices, fmt.Sprintf("Conservado el servidor MCP %q de %s: no coincide con el que declara Axiom (lo definiste tú).", name, path))
			return raw, false, nil
		}
		return removeJSONPaths(raw, append(slices.Clone(parent), name))
	})
}

// managedMCPOperations devuelve las operaciones que retiran el servidor MCP name
// (context7 o engram) de la configuración de un agente, sobre la raíz del ámbito
// (el proyecto en workspace; la carpeta de usuario en global, donde solo la usa Kilo).
// Solo se retiran las entradas con la forma de Axiom; el resto del fichero es del
// usuario o del equipo.
func (s *Service) managedMCPOperations(adapter agents.Adapter, name string) (targets []string, ops []operation) {
	root := s.root()
	switch adapter.MCPStrategy() {
	case model.StrategySeparateMCPFiles:
		if adapter.Agent() != model.AgentClaudeCode {
			path := adapter.MCPConfigPath(root, name)
			return []string{path}, []operation{removeFile(path), quietDirIfEmpty(filepath.Dir(path))}
		}
		// Claude Code lee los servidores de proyecto de <raíz>/.mcp.json. El bloque
		// de .claude/settings.json y el fichero independiente de .claude/mcp son
		// restos de instalaciones anteriores.
		mcpJSON := filepath.Join(root, ".mcp.json")
		settings := adapter.SettingsPath(root)
		legacy := adapter.MCPConfigPath(root, name)
		targets = []string{mcpJSON, settings, legacy}
		ops = []operation{
			s.rewriteManagedMCPServer(mcpJSON, jsonPath{"mcpServers"}, name),
			s.rewriteManagedMCPServer(settings, jsonPath{"mcpServers"}, name),
		}
		if name == "context7" {
			ops = append(ops, removeManagedContext7File(legacy))
		} else {
			ops = append(ops, removeManagedEngramFile(legacy))
		}
		ops = append(ops, quietDirIfEmpty(filepath.Dir(legacy)))
		return targets, ops
	case model.StrategyMergeIntoSettings:
		parent := jsonPath{"mcpServers"}
		switch adapter.Agent() {
		case model.AgentOpenCode, model.AgentKilocode:
			parent = jsonPath{"mcp"}
		case model.AgentOpenClaw:
			parent = jsonPath{"mcp", "servers"}
		}
		for _, path := range s.settingsTargets(adapter) {
			targets = append(targets, path)
			ops = append(ops, s.rewriteManagedMCPServer(path, parent, name))
		}
		return targets, ops
	case model.StrategyMCPConfigFile:
		path := adapter.MCPConfigPath(root, name)
		if path == "" {
			return nil, nil
		}
		parent := jsonPath{"mcpServers"}
		if adapter.Agent() == model.AgentVSCodeCopilot {
			parent = jsonPath{"servers"}
		}
		return []string{path}, []operation{s.rewriteManagedMCPServer(path, parent, name)}
	case model.StrategyTOMLFile:
		configPath := adapter.MCPConfigPath(root, name)
		if configPath == "" {
			return nil, nil
		}
		targets = []string{configPath}
		ops = []operation{rewriteTOMLFile(configPath, func(content string) (string, bool) {
			return cleanCodexWorkspaceTOML(content, name)
		})}
		if name == "engram" {
			compact := filepath.Join(filepath.Dir(configPath), codexCompactPromptValue)
			targets = append(targets, compact)
			ops = append(ops, removeFile(compact), quietDirIfEmpty(filepath.Dir(compact)))
		}
		return targets, ops
	}
	return nil, nil
}

// removeManagedOutputStyle borra un estilo de salida solo si su contenido es el que
// Axiom incrusta; si el usuario lo editó, lo conserva y avisa.
func (s *Service) removeManagedOutputStyle(path, assetPath string) operation {
	return operation{
		typeID: opRemoveFile,
		path:   path,
		apply: func(path string) (bool, bool, error) {
			content, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return false, false, nil
				}
				return false, false, err
			}
			embedded, embeddedErr := assets.Read(assetPath)
			if embeddedErr != nil || strings.ReplaceAll(string(content), "\r\n", "\n") != strings.ReplaceAll(embedded, "\r\n", "\n") {
				s.notices = append(s.notices, fmt.Sprintf("Conservado %s: su contenido no es el que instala Axiom (lo editaste tú). Bórralo a mano si ya no lo necesitas.", path))
				return false, false, nil
			}
			if err := removeFileIfExists(path); err != nil {
				return false, false, err
			}
			return true, true, nil
		},
	}
}

// quietDirIfEmpty es removeDirIfEmpty sin aviso cuando el directorio conserva
// contenido: en el proyecto casi siempre es contenido del equipo.
func quietDirIfEmpty(path string) operation {
	op := removeDirIfEmpty(path)
	op.quiet = true
	return op
}

// tomlTableBody devuelve el cuerpo de la tabla [name] de un TOML y si existe.
func tomlTableBody(content, name string) (string, bool) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	header := "[" + name + "]"
	var body []string
	inside, found := false, false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == header {
			inside, found = true, true
			continue
		}
		if inside && strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			break
		}
		if inside {
			body = append(body, line)
		}
	}
	return strings.Join(body, "\n"), found
}

// tomlTableKeyValue devuelve el valor (sin espacios) de una clave simple de una
// tabla TOML y si está declarada.
func tomlTableKeyValue(content, table, key string) (string, bool) {
	body, found := tomlTableBody(content, table)
	if !found {
		return "", false
	}
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		name, value, ok := strings.Cut(trimmed, "=")
		if ok && strings.TrimSpace(name) == key {
			return strings.TrimSpace(value), true
		}
	}
	return "", false
}

// removeEmptyTOMLTable quita la cabecera [name] si su cuerpo no tiene nada más que
// líneas en blanco.
func removeEmptyTOMLTable(content, name string) string {
	body, found := tomlTableBody(content, name)
	if !found || strings.TrimSpace(body) != "" {
		return content
	}
	return filemerge.RemoveTOMLTable(content, name)
}

// cleanCodexWorkspaceTOML retira del config.toml de proyecto de Codex lo que escribe
// el componente name. Solo se quita lo que tiene la forma de Axiom:
//   - engram: la tabla [mcp_servers.engram], experimental_compact_prompt_file si apunta
//     al fichero de Axiom, y features.multi_agent, agents.max_threads y
//     agents.max_depth si valen lo que Axiom escribe (true, 4 y 2). Las tablas que se
//     quedan vacías se quitan; un valor distinto es del usuario y se conserva.
//   - context7: la tabla [mcp_servers.context7] si apunta al servidor remoto de Axiom.
func cleanCodexWorkspaceTOML(content, name string) (string, bool) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	updated := normalized

	switch name {
	case "context7":
		updated = removeCodexContext7Table(updated)
	case "engram":
		if body, found := tomlTableBody(updated, "mcp_servers.engram"); found && strings.Contains(body, "--tools=agent") {
			updated = filemerge.RemoveTOMLTable(updated, "mcp_servers.engram")
		}
		if value, declared, isString, err := filemerge.TopLevelTOMLString(updated, "experimental_compact_prompt_file"); err == nil && declared && isString && value == codexCompactPromptValue {
			updated = filemerge.RemoveTopLevelTOMLKey(updated, "experimental_compact_prompt_file")
		}
		updated = removeCodexMultiAgentKeys(updated)
	}

	updated = strings.TrimSpace(updated)
	if updated != "" {
		updated += "\n"
	}
	return updated, updated != normalized
}

// codexMultiAgentKeys son las claves que Axiom escribe en config.toml de Codex para
// habilitar la delegación de SDD, con los valores exactos del instalador.
var codexMultiAgentKeys = []struct{ table, key, value string }{
	{"features", "multi_agent", "true"},
	{"agents", "max_threads", "4"},
	{"agents", "max_depth", "2"},
}

// removeCodexMultiAgentKeys retira de un config.toml de Codex las claves de
// codexMultiAgentKeys que valen lo que escribe Axiom y las tablas que quedan vacías.
// Un valor distinto es del usuario y se conserva.
func removeCodexMultiAgentKeys(content string) string {
	updated := content
	for _, entry := range codexMultiAgentKeys {
		if value, declared := tomlTableKeyValue(updated, entry.table, entry.key); declared && value == entry.value {
			updated = filemerge.RemoveTOMLTableKeys(updated, entry.table, []string{entry.key})
		}
	}
	return removeEmptyTOMLTable(removeEmptyTOMLTable(updated, "features"), "agents")
}

// removeCodexContext7Table retira la tabla [mcp_servers.context7] de un config.toml de
// Codex solo si apunta al servidor remoto que declara Axiom.
func removeCodexContext7Table(content string) string {
	if body, found := tomlTableBody(content, "mcp_servers.context7"); found && strings.Contains(body, context7RemoteURL) {
		return filemerge.RemoveTOMLTable(content, "mcp_servers.context7")
	}
	return content
}

// removeCodexDeveloperInstructionsBlock quita de developer_instructions el bloque
// delimitado por los marcadores de Axiom. El texto del usuario fuera del bloque se
// conserva; si la cadena queda vacía, la clave desaparece.
func removeCodexDeveloperInstructionsBlock(content, key string) (string, bool) {
	existing, declared, isString, err := filemerge.TopLevelTOMLString(content, key)
	if err != nil || !declared || !isString {
		return content, false
	}
	cleaned := filemerge.InjectMarkdownSection(existing, filemerge.WorkspaceInstructionsSectionID, "")
	if cleaned == existing {
		return content, false
	}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	if strings.TrimSpace(cleaned) == "" {
		return filemerge.RemoveTopLevelTOMLKey(normalized, key), true
	}
	return filemerge.UpsertTopLevelTOMLValue(normalized, key, filemerge.FormatTOMLString(cleaned)), true
}

// instructionsEntryRemoval retira de la lista instructions del fichero de
// configuración de OpenCode o Kilo la entrada que registra el fichero de Axiom.
func instructionsEntryRemoval(entry string) func(raw []byte) ([]byte, bool, error) {
	return func(raw []byte) ([]byte, bool, error) {
		root, err := unmarshalJSONObject(raw)
		if err != nil {
			return nil, false, err
		}
		list, ok := root["instructions"].([]any)
		if !ok {
			return raw, false, nil
		}
		kept := make([]any, 0, len(list))
		for _, item := range list {
			if text, isText := item.(string); isText && text == entry {
				continue
			}
			kept = append(kept, item)
		}
		if len(kept) == len(list) {
			return raw, false, nil
		}
		if len(kept) == 0 {
			delete(root, "instructions")
		} else {
			root["instructions"] = kept
		}
		return marshalLike(raw, root)
	}
}

// contextFileNameRemoval retira de context.fileName de Gemini el fichero de Axiom.
// Axiom escribió la lista con los nombres previos del proyecto más el suyo: si al
// quitarlo solo quedan los nombres por defecto, la clave entera es de Axiom y se
// retira; si quedan otros, el usuario conserva su lista.
func contextFileNameRemoval(entry string, defaults []string) func(raw []byte) ([]byte, bool, error) {
	return func(raw []byte) ([]byte, bool, error) {
		root, err := unmarshalJSONObject(raw)
		if err != nil {
			return nil, false, err
		}
		contextObject, ok := root["context"].(map[string]any)
		if !ok {
			return raw, false, nil
		}
		list, ok := contextObject["fileName"].([]any)
		if !ok {
			return raw, false, nil
		}
		kept := make([]string, 0, len(list))
		found := false
		for _, item := range list {
			text, isText := item.(string)
			if !isText {
				return raw, false, nil
			}
			if text == entry {
				found = true
				continue
			}
			kept = append(kept, text)
		}
		if !found {
			return raw, false, nil
		}
		if len(kept) == 0 || slices.Equal(kept, defaults) {
			delete(contextObject, "fileName")
			if len(contextObject) == 0 {
				delete(root, "context")
			}
		} else {
			contextObject["fileName"] = kept
		}
		return marshalLike(raw, root)
	}
}

// removeManagedAgentEntries retira de agent.* las definiciones que llevan la marca
// de Axiom y deja las del usuario. El overlay de OpenCode y Kilo ya no escribe el
// marcador __managed_by, así que se identifican por nombre y forma (orquestador,
// fases de SDD, jueces, revisores y perfiles), y el marcador heredado de las
// instalaciones previas sigue valiendo.
func removeManagedAgentEntries(raw []byte) ([]byte, bool, error) {
	root, err := unmarshalJSONObject(raw)
	if err != nil {
		return nil, false, err
	}
	agentsObject, ok := root["agent"].(map[string]any)
	if !ok {
		return raw, false, nil
	}
	changed := false
	for name, value := range agentsObject {
		definition, isObject := value.(map[string]any)
		if !isObject {
			continue
		}
		if opencodeactivation.IsManagedAgent(name, definition) {
			delete(agentsObject, name)
			changed = true
		}
	}
	if !changed {
		return raw, false, nil
	}
	if len(agentsObject) == 0 {
		delete(root, "agent")
	}
	return marshalLike(raw, root)
}

// removeDefaultOpenCodeShare retira share = "disabled", el valor por defecto que
// la instalación de SDD añade si el usuario no eligió modo de compartición. Un
// valor distinto (manual, auto) es del usuario y se conserva. Un "disabled" que ya
// existía no se distingue del que añadió Axiom, pero sin la clave OpenCode usa su
// modo manual, que tampoco comparte nada por sí solo.
func removeDefaultOpenCodeShare(raw []byte) ([]byte, bool, error) {
	root, err := unmarshalJSONObject(raw)
	if err != nil {
		return nil, false, err
	}
	if root["share"] != "disabled" {
		return raw, false, nil
	}
	delete(root, "share")
	return marshalLike(raw, root)
}

// marshalLike serializa root con la sangría y el fin de línea del documento raw.
func marshalLike(raw []byte, root map[string]any) ([]byte, bool, error) {
	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("marshal json after cleanup: %w", err)
	}
	eol := detectEOL(string(raw))
	return []byte(strings.ReplaceAll(string(encoded), "\n", eol) + eol), true, nil
}

// stripManagedFrontmatter quita la cabecera que Axiom escribe al crear un fichero
// de instrucciones con cabecera propia (VS Code y Kiro).
func stripManagedFrontmatter(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	for _, header := range []string{
		filemerge.SDDInstructionsFrontmatter,
		filemerge.LegacySDDInstructionsFrontmatter,
		filemerge.SDDSteeringFrontmatter,
	} {
		if strings.HasPrefix(normalized, header) {
			return normalized[len(header):]
		}
	}
	return normalized
}

// sweepInstructionsFile borra un fichero de instrucciones que, tras retirar las
// secciones de Axiom, no tiene más que la cabecera que Axiom escribió. Si queda
// contenido del usuario, el fichero se conserva y, en una desinstalación completa,
// se avisa cuando es uno de los que Axiom escribe enteros.
func (s *Service) sweepInstructionsFile(path string, full bool) operation {
	return operation{
		typeID: opRewriteFile,
		path:   path,
		apply: func(path string) (bool, bool, error) {
			if _, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					return false, false, nil
				}
				return false, false, err
			}
			content, err := readFileOrEmpty(path)
			if err != nil {
				return false, false, err
			}
			// Sin secciones, lo único que puede quedar de Axiom es la cabecera o la
			// persona que los agentes con sustitución de fichero reciben sin marcadores.
			body := strings.TrimSpace(stripManagedFrontmatter(content))
			if body == "" || slices.Contains(embeddedPersonas(), body) {
				if err := removeFileIfExists(path); err != nil {
					return false, false, err
				}
				return true, true, nil
			}
			if full && slices.Contains(managedInstructionFileNames, filepath.Base(path)) {
				s.notices = append(s.notices, fmt.Sprintf("Conservado %s: tiene contenido que Axiom no gestiona (lo editaste tú). Revísalo y bórralo a mano si ya no lo necesitas.", path))
			}
			return false, false, nil
		},
	}
}

// workspaceAgentRegistrationOperations son las operaciones del agente que no
// dependen de un componente concreto: los ficheros de instrucciones y su
// registro en la configuración del proyecto. Se ejecutan con la retirada completa
// del agente, porque el fichero de instrucciones reúne las secciones de varios
// componentes y su registro solo sobra cuando ya no queda ninguna.
func (s *Service) workspaceAgentRegistrationOperations(adapter agents.Adapter, full bool) (targets []string, ops []operation) {
	root := s.workspaceDir
	if adapter.SupportsSystemPrompt() {
		for _, path := range systemPromptFiles(adapter, root) {
			targets = append(targets, path)
			ops = append(ops, s.sweepInstructionsFile(path, full))
		}
	}
	// La guía de enrutado de VS Code Copilot se entrega con el adaptador global sobre
	// la raíz del proyecto, así que deja un axiom.instructions.md con la distribución
	// de usuario (AppData/Roaming/Code/User/prompts, ~/.config/Code/User/prompts)
	// dentro del proyecto. Se limpia con el mismo criterio que el resto.
	if _, bound := adapter.(agents.WorkspaceScopedProvider); bound && adapter.SupportsSystemPrompt() {
		if global, ok := s.registry.Get(adapter.Agent()); ok {
			stray := global.SystemPromptFile(root)
			if stray != "" && !slices.Contains(systemPromptFiles(adapter, root), stray) {
				targets = append(targets, stray)
				ops = append(ops,
					rewriteMarkdownFile(stray, func(content string) (string, bool) {
						return removeMarkdownSections(content, allAxiomSectionIDs()...)
					}),
					s.sweepInstructionsFile(stray, full),
				)
			}
		}
	}
	if !full {
		return targets, ops
	}

	if provider, ok := adapter.(agents.WorkspaceInstructionsProvider); ok {
		if settingsPath, entry, ok := provider.WorkspaceInstructions(root); ok && settingsPath != "" {
			targets = append(targets, settingsPath)
			ops = append(ops, rewriteJSONWith(settingsPath, instructionsEntryRemoval(entry)))
		}
	}
	if provider, ok := adapter.(agents.WorkspaceContextFileProvider); ok {
		if settingsPath, entry, defaults, ok := provider.WorkspaceContextFile(root); ok && settingsPath != "" {
			targets = append(targets, settingsPath)
			ops = append(ops, rewriteJSONWith(settingsPath, contextFileNameRemoval(entry, defaults)))
		}
	}
	if provider, ok := adapter.(agents.WorkspaceInlineInstructionsProvider); ok {
		if configPath, key, ok := provider.WorkspaceInlineInstructions(root); ok && configPath != "" {
			targets = append(targets, configPath)
			ops = append(ops, rewriteTOMLFile(configPath, func(content string) (string, bool) {
				return removeCodexDeveloperInstructionsBlock(content, key)
			}))
		}
	}
	return targets, ops
}

// workspaceEmptyDirOperations intenta vaciar las carpetas de configuración que
// Axiom creó en el proyecto (.claude, .opencode, .codex...) y sus padres hasta la
// raíz, sin llegar a ella. Solo se quitan si están vacías: lo que el equipo tenga
// dentro nunca se toca.
func (s *Service) workspaceEmptyDirOperations(adapter agents.Adapter) []operation {
	root := s.workspaceDir
	candidates := []string{
		adapter.GlobalConfigDir(root),
		filepath.Dir(adapter.SettingsPath(root)),
		filepath.Dir(adapter.MCPConfigPath(root, "engram")),
	}
	if adapter.SupportsSystemPrompt() {
		candidates = append(candidates, filepath.Dir(adapter.SystemPromptFile(root)))
		if _, bound := adapter.(agents.WorkspaceScopedProvider); bound {
			if global, ok := s.registry.Get(adapter.Agent()); ok {
				candidates = append(candidates, filepath.Dir(global.SystemPromptFile(root)))
			}
		}
	}
	if adapter.SupportsSkills() {
		if skillDir := adapter.SkillsDir(root); skillDir != "" && !s.skillsDirStaysInUse(skillDir) {
			candidates = append(candidates, skillDir)
		}
	}
	if adapter.SupportsSlashCommands() {
		candidates = append(candidates, adapter.CommandsDir(root))
	}
	if adapter.SupportsSubAgents() && adapter.SubAgentsDir(root) != "" {
		candidates = append(candidates, adapter.SubAgentsDir(root))
	}
	if adapter.SupportsOutputStyles() {
		candidates = append(candidates, adapter.OutputStyleDir(root))
	}

	var dirs []string
	for _, candidate := range candidates {
		for dir := filepath.Clean(candidate); isStrictlyInside(root, dir); dir = filepath.Dir(dir) {
			if !slices.Contains(dirs, dir) {
				dirs = append(dirs, dir)
			}
		}
	}
	ops := make([]operation, 0, len(dirs))
	for _, dir := range dirs {
		ops = append(ops, quietDirIfEmpty(dir))
	}
	return ops
}

// isStrictlyInside indica si path está dentro de root sin ser root.
func isStrictlyInside(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil || rel == "." || rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// addWorkspaceOperations completa el plan de una desinstalación de workspace con lo
// que no pertenece a un componente: los ficheros de instrucciones y su registro, las
// carpetas que quedan vacías, el índice de skills del AGENTS.md del proyecto y el
// estado de proyecto como objetivo de copia de seguridad.
func (s *Service) addWorkspaceOperations(agentIDs []model.AgentID, componentIDs []model.ComponentID, backupTargets map[string]struct{}, operationsByKey map[string]operation) error {
	full := removesAllAgentComponents(componentIDs)
	for _, agentID := range agentIDs {
		adapter, ok := s.adapter(agentID)
		if !ok {
			continue
		}
		targets, ops := s.workspaceAgentRegistrationOperations(adapter, full)
		for _, target := range targets {
			files, err := expandBackupTarget(target)
			if err != nil {
				return fmt.Errorf("expand backup target %q: %w", target, err)
			}
			for _, file := range files {
				backupTargets[file] = struct{}{}
			}
		}
		for _, op := range ops {
			addPlanOperation(operationsByKey, agentID, op)
		}
		for _, op := range s.workspaceEmptyDirOperations(adapter) {
			addPlanOperation(operationsByKey, agentID, op)
		}
	}

	backupTargets[state.ProjectPath(s.workspaceDir)] = struct{}{}

	// El índice de skills de AGENTS.md es del proyecto, no de un agente: solo se
	// retira cuando no queda ninguno instalado.
	if full && len(s.otherAgentsInProject()) == 0 {
		agentsMD := filepath.Join(s.workspaceDir, "AGENTS.md")
		backupTargets[agentsMD] = struct{}{}
		op := rewriteMarkdownFile(agentsMD, func(content string) (string, bool) {
			return removeMarkdownSections(content, skillsIndexSectionID)
		})
		if len(agentIDs) > 0 {
			addPlanOperation(operationsByKey, agentIDs[0], op)
		}

		// El registro de skills que genera `skill-registry refresh` (en sync y en los
		// hooks) vive en .atl; sus skills propias (.atl/skills) son del usuario.
		if len(agentIDs) > 0 {
			atlDir := filepath.Join(s.workspaceDir, ".atl")
			for _, name := range skillRegistryGeneratedFiles {
				path := filepath.Join(atlDir, name)
				backupTargets[path] = struct{}{}
				addPlanOperation(operationsByKey, agentIDs[0], removeFile(path))
			}
			addPlanOperation(operationsByKey, agentIDs[0], quietDirIfEmpty(atlDir))
		}
	}
	return nil
}
