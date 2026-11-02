package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/pipeline"
)

// byteOrderMark es la marca de orden de bytes de UTF-8 que algunos editores añaden
// al principio de un fichero.
const byteOrderMark = "\xef\xbb\xbf"

// workspaceInstructionsNoticeWriter recibe el aviso de que no se pudo registrar
// el fichero de instrucciones. Es una variable para que las pruebas lo capturen.
var workspaceInstructionsNoticeWriter io.Writer = os.Stderr

// registerWorkspaceInstructions declara el fichero de instrucciones propio de
// Axiom en la lista instructions de la configuración del proyecto.
//
// En workspace, OpenCode y Kilo no cargan solos un fichero suelto de su carpeta de
// recursos: leen AGENTS.md (que es del equipo y Axiom no toca) y los ficheros que
// enumera instructions. Esta función añade la entrada de Axiom a esa lista sin
// duplicarla y conservando las del usuario, y deja el resto del documento como
// estaba.
//
// Solo registra el fichero si existe: sin él (ni persona ni memoria instaladas)
// no hay nada que cargar, y un puntero a un fichero que falta es ruido. Un
// documento que no es JSON, o cuyo instructions no es una lista, se deja intacto
// y se avisa: es un fichero del usuario, normalmente versionado.
func registerWorkspaceInstructions(adapter agents.Adapter, root string) (changed bool, files []string, err error) {
	if _, byName := adapter.(agents.WorkspaceContextFileProvider); byName {
		return registerWorkspaceContextFile(adapter, root)
	}
	if _, inline := adapter.(agents.WorkspaceInlineInstructionsProvider); inline {
		return registerWorkspaceInlineInstructions(adapter, root)
	}
	provider, ok := adapter.(agents.WorkspaceInstructionsProvider)
	if !ok {
		return false, nil, nil
	}
	settingsPath, entry, ok := provider.WorkspaceInstructions(root)
	if !ok || settingsPath == "" {
		return false, nil, nil
	}
	if info, statErr := os.Stat(adapter.SystemPromptFile(root)); statErr != nil || !info.Mode().IsRegular() {
		return false, nil, nil
	}

	current, err := os.ReadFile(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return false, nil, fmt.Errorf("leer %s: %w", settingsPath, err)
	}
	if len(current) > 0 {
		parsed, parseErr := filemerge.UnmarshalJSONObject(current)
		if parseErr != nil {
			return false, nil, fmt.Errorf("no se registran las instrucciones de Axiom en %s: no es JSON válido: %w", settingsPath, parseErr)
		}
		if existing, declared := parsed["instructions"]; declared {
			if _, isList := existing.([]any); !isList {
				fmt.Fprintf(workspaceInstructionsNoticeWriter, "Aviso: %s declara instructions y no es una lista; Axiom no lo modifica y no registra %s.\n", settingsPath, entry)
				return false, nil, nil
			}
		}
	}

	overlay := []byte(fmt.Sprintf(`{"instructions":[%q]}`, entry))
	overlay, err = filemerge.UnionJSONArrays(current, overlay, []string{"instructions"})
	if err != nil {
		return false, nil, err
	}
	merged, err := filemerge.MergeJSONObjectsForPath(settingsPath, current, overlay)
	if err != nil {
		return false, nil, fmt.Errorf("registrar %s en %s: %w", entry, settingsPath, err)
	}
	write, err := filemerge.WriteFileAtomic(settingsPath, merged, 0o644)
	if err != nil {
		return false, nil, err
	}
	return write.Changed, []string{settingsPath}, nil
}

// registerWorkspaceContextFile declara el fichero de instrucciones propio de Axiom
// en context.fileName de la configuración de proyecto de Gemini CLI.
//
// Gemini no carga solo un fichero suelto de .gemini: lee los nombres que enumera
// context.fileName (por defecto GEMINI.md) en cada carpeta de la raíz del proyecto
// hacia arriba, uniéndolos con path.join. Un nombre con separador, como
// .gemini/AXIOM.md, carga el fichero propio de Axiom sin tocar el GEMINI.md de la
// raíz, que es del equipo.
//
// El context.fileName del proyecto sustituye al del usuario, no se fusiona. Por eso
// el valor escrito conserva los nombres que el proyecto ya declaraba (o GEMINI.md si
// no declaraba ninguno): sin él, el GEMINI.md de la raíz y el global dejarían de
// cargarse. Una cadena existente se convierte en lista conservándola. Si context o
// context.fileName tienen un tipo inesperado, el documento se deja intacto y se avisa.
//
// Solo registra el fichero si existe, por la misma razón que registerWorkspaceInstructions.
func registerWorkspaceContextFile(adapter agents.Adapter, root string) (changed bool, files []string, err error) {
	provider, ok := adapter.(agents.WorkspaceContextFileProvider)
	if !ok {
		return false, nil, nil
	}
	settingsPath, entry, defaults, ok := provider.WorkspaceContextFile(root)
	if !ok || settingsPath == "" {
		return false, nil, nil
	}
	if info, statErr := os.Stat(adapter.SystemPromptFile(root)); statErr != nil || !info.Mode().IsRegular() {
		return false, nil, nil
	}

	current, err := os.ReadFile(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return false, nil, fmt.Errorf("leer %s: %w", settingsPath, err)
	}
	names := append([]string(nil), defaults...)
	if len(bytes.TrimSpace(current)) > 0 {
		parsed, parseErr := filemerge.UnmarshalJSONObject(current)
		if parseErr != nil {
			return false, nil, fmt.Errorf("no se registran las instrucciones de Axiom en %s: no es JSON válido: %w", settingsPath, parseErr)
		}
		declared, usable := declaredContextFileNames(parsed)
		if !usable {
			fmt.Fprintf(workspaceInstructionsNoticeWriter, "Aviso: %s declara context o context.fileName con un tipo que Axiom no reconoce (se espera un objeto y una cadena o una lista de cadenas); Axiom no lo modifica y no registra %s.\n", settingsPath, entry)
			return false, nil, nil
		}
		if len(declared) > 0 {
			names = declared
		}
	}
	if !slices.Contains(names, entry) {
		names = append(names, entry)
	}

	encoded, err := json.Marshal(map[string]any{"context": map[string]any{"fileName": names}})
	if err != nil {
		return false, nil, fmt.Errorf("registrar %s en %s: %w", entry, settingsPath, err)
	}
	merged, err := filemerge.MergeJSONObjectsForPath(settingsPath, current, encoded)
	if err != nil {
		return false, nil, fmt.Errorf("registrar %s en %s: %w", entry, settingsPath, err)
	}
	write, err := filemerge.WriteFileAtomic(settingsPath, merged, 0o644)
	if err != nil {
		return false, nil, err
	}
	return write.Changed, []string{settingsPath}, nil
}

// registerWorkspaceInlineInstructions copia el fichero de instrucciones de Axiom al
// valor de developer_instructions de la configuración de proyecto de Codex.
//
// Codex no carga ningún fichero suelto de .codex, y las instrucciones no pueden ir
// al AGENTS.md de la raíz (es del equipo y está versionado) ni a model_instructions_file
// (sustituye las instrucciones base en lugar de añadirse a ellas). developer_instructions
// sí se añade a las base, así que el contenido de .codex/axiom-instructions.md, que es
// donde los componentes escriben, se copia a un bloque delimitado por marcadores de
// Axiom dentro de ese valor.
//
// El texto que el usuario ya tuviera en developer_instructions queda fuera del bloque
// y no se toca; repetir la operación no cambia ni un byte. Si la clave existe con un
// tipo que no es cadena, o el documento no es TOML válido, se deja intacto y se avisa
// (o se falla, en el segundo caso, con el mismo criterio que la comprobación previa).
// Solo copia si el fichero fuente existe: sin él no hay nada que cargar.
func registerWorkspaceInlineInstructions(adapter agents.Adapter, root string) (changed bool, files []string, err error) {
	provider, ok := adapter.(agents.WorkspaceInlineInstructionsProvider)
	if !ok {
		return false, nil, nil
	}
	configPath, key, ok := provider.WorkspaceInlineInstructions(root)
	if !ok || configPath == "" {
		return false, nil, nil
	}
	source := adapter.SystemPromptFile(root)
	if info, statErr := os.Stat(source); statErr != nil || !info.Mode().IsRegular() {
		return false, nil, nil
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		return false, nil, fmt.Errorf("leer %s: %w", source, err)
	}
	instructions := strings.TrimPrefix(strings.ReplaceAll(string(raw), "\r\n", "\n"), byteOrderMark)

	current, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return false, nil, fmt.Errorf("leer %s: %w", configPath, err)
	}
	existing, declared, isString, err := filemerge.TopLevelTOMLString(string(current), key)
	if err != nil {
		return false, nil, fmt.Errorf("no se registran las instrucciones de Axiom en %s: no es TOML válido: %w", configPath, err)
	}
	if declared && !isString {
		fmt.Fprintf(workspaceInstructionsNoticeWriter, "Aviso: %s declara %s y no es una cadena; Axiom no lo modifica y no registra sus instrucciones.\n", configPath, key)
		return false, nil, nil
	}

	updated := filemerge.InjectMarkdownSection(existing, filemerge.WorkspaceInstructionsSectionID, instructions)
	if (declared && updated == existing) || (!declared && updated == "") {
		return false, []string{configPath}, nil
	}
	rendered := filemerge.UpsertTopLevelTOMLValue(string(current), key, filemerge.FormatTOMLString(updated))
	write, err := filemerge.WriteFileAtomic(configPath, []byte(rendered), 0o644)
	if err != nil {
		return false, nil, err
	}
	return write.Changed, []string{configPath}, nil
}

// declaredContextFileNames devuelve los nombres que el documento declara en
// context.fileName, en su orden y sin repetidos. usable es false cuando context o
// context.fileName tienen un tipo que no se puede conservar con seguridad. Una
// cadena o una lista vacías cuentan como no declarado.
func declaredContextFileNames(settings map[string]any) (names []string, usable bool) {
	rawContext, declared := settings["context"]
	if !declared {
		return nil, true
	}
	contextObject, isObject := rawContext.(map[string]any)
	if !isObject {
		return nil, false
	}
	rawName, declared := contextObject["fileName"]
	if !declared {
		return nil, true
	}
	add := func(name string) {
		if name != "" && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	switch value := rawName.(type) {
	case string:
		add(value)
	case []any:
		for _, item := range value {
			name, isString := item.(string)
			if !isString {
				return nil, false
			}
			add(name)
		}
	default:
		return nil, false
	}
	return names, true
}

// workspaceInstructionsSettingsFile devuelve el fichero de configuración de
// proyecto donde un adaptador registra el fichero de instrucciones de Axiom, o ""
// si no lo registra.
func workspaceInstructionsSettingsFile(adapter agents.Adapter, workspaceDir string) string {
	if provider, ok := adapter.(agents.WorkspaceInstructionsProvider); ok {
		if settingsPath, _, ok := provider.WorkspaceInstructions(workspaceDir); ok {
			return settingsPath
		}
	}
	if provider, ok := adapter.(agents.WorkspaceContextFileProvider); ok {
		if settingsPath, _, _, ok := provider.WorkspaceContextFile(workspaceDir); ok {
			return settingsPath
		}
	}
	if provider, ok := adapter.(agents.WorkspaceInlineInstructionsProvider); ok {
		if configPath, _, ok := provider.WorkspaceInlineInstructions(workspaceDir); ok {
			return configPath
		}
	}
	return ""
}

// registersWorkspaceInstructions indica si alguno de los agentes seleccionados
// necesita que el fichero de instrucciones de Axiom se registre en su
// configuración de proyecto: OpenCode y Kilo en instructions, Gemini en
// context.fileName y Codex en developer_instructions.
func registersWorkspaceInstructions(agentIDs []model.AgentID) bool {
	return containsAgent(agentIDs, model.AgentOpenCode) ||
		containsAgent(agentIDs, model.AgentKilocode) ||
		containsAgent(agentIDs, model.AgentGeminiCLI) ||
		containsAgent(agentIDs, model.AgentCodex)
}

// workspaceInstructionsSettingsPaths devuelve los ficheros de configuración que
// registerWorkspaceInstructions puede reescribir: son objetivo de copia de
// seguridad aunque ningún componente seleccionado los toque.
func workspaceInstructionsSettingsPaths(workspaceDir string, scope InstallScope, adapters []agents.Adapter) []string {
	if scope != ScopeWorkspace {
		return nil
	}
	var paths []string
	for _, adapter := range adapters {
		if settingsPath := workspaceInstructionsSettingsFile(adapter, workspaceDir); settingsPath != "" {
			paths = append(paths, settingsPath)
		}
	}
	return paths
}

// workspaceTrustNotes devuelve el aviso informativo de la confianza del proyecto.
//
// Codex solo carga la configuración de proyecto (.codex/config.toml, con las
// instrucciones, el MCP y las claves de multiagente, y .codex/hooks.json) cuando el
// proyecto es de confianza, y cada hook se aprueba además por hash. Axiom no escribe
// esa confianza en ~/.codex/config.toml (sería una escritura en la carpeta de usuario
// que el ámbito workspace no hace): se lo pide Codex al usuario la primera vez.
//
// VS Code Copilot hereda Workspace Trust: en un espacio de trabajo restringido no
// arranca los servidores MCP de .vscode/mcp.json (ni cargaría agentes o hooks de
// .github/, que Axiom no instala). La confianza la concede el usuario en VS Code.
func workspaceTrustNotes(scope InstallScope, agentIDs []model.AgentID) []string {
	if scope != ScopeWorkspace {
		return nil
	}
	var notes []string
	if containsAgent(agentIDs, model.AgentCodex) {
		notes = append(notes, "Codex solo carga la configuración de .codex/ (instrucciones, MCP y hooks) si el proyecto es de confianza: te lo pedirá Codex la primera vez que lo abras, y Axiom no escribe esa confianza en ~/.codex/config.toml. Además, Codex te pide aprobar cada hook de .codex/hooks.json por su hash.")
	}
	if containsAgent(agentIDs, model.AgentVSCodeCopilot) {
		notes = append(notes, "VS Code solo arranca los servidores MCP de .vscode/mcp.json (los que Axiom declara, como Context7 y Engram) si el espacio de trabajo es de confianza (Workspace Trust): en modo restringido no se cargan. Axiom no concede esa confianza; la concedes tú en VS Code al abrir la carpeta.")
	}
	return notes
}

// workspaceInstructionsStep registra el fichero de instrucciones de Axiom de
// cada agente del proyecto que lo necesita (OpenCode, Kilo, Gemini y Codex). Va después de los componentes
// y de la guía de enrutado, que son quienes escriben ese fichero.
type workspaceInstructionsStep struct {
	id           string
	workspaceDir string
	agents       []model.AgentID

	// changedFiles es el acumulador de sync. Install lo deja en nil.
	changedFiles *[]string
}

func (s workspaceInstructionsStep) ID() string { return s.id }

func (s workspaceInstructionsStep) Run() error {
	for _, adapter := range resolveAdaptersScoped(ScopeWorkspace, s.workspaceDir, s.agents) {
		changed, files, err := registerWorkspaceInstructions(adapter, s.workspaceDir)
		if err != nil {
			return fmt.Errorf("registrar las instrucciones de %s: %w", adapter.Agent(), err)
		}
		if changed && s.changedFiles != nil {
			*s.changedFiles = append(*s.changedFiles, files...)
		}
	}
	return nil
}

// workspaceSettingsPreflightStep rechaza, antes de escribir nada, una
// configuración de proyecto de OpenCode, Kilo, Gemini, VS Code Copilot (JSON) o
// Codex (TOML) que no se puede leer.
//
// El resto de fusiones sustituyen una base ilegible por una vacía: en la
// configuración global es aceptable (la copia de seguridad previa la conserva),
// pero opencode.json (o .gemini/settings.json) de un proyecto es un fichero del usuario, normalmente
// versionado, y reescribirlo desde cero perdería su contenido sin vuelta atrás.
type workspaceSettingsPreflightStep struct {
	id           string
	workspaceDir string
	agents       []model.AgentID
}

func (s workspaceSettingsPreflightStep) ID() string { return s.id }

func (s workspaceSettingsPreflightStep) Run() error {
	noteStaleWorkspaceConfig(s.workspaceDir, s.agents)
	for _, adapter := range resolveAdaptersScoped(ScopeWorkspace, s.workspaceDir, s.agents) {
		var files []string
		if settingsPath := workspaceInstructionsSettingsFile(adapter, s.workspaceDir); settingsPath != "" {
			files = append(files, settingsPath)
		}
		files = append(files, workspaceUserConfigFiles(adapter, s.workspaceDir)...)
		for _, settingsPath := range files {
			if err := checkWorkspaceConfigReadable(settingsPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkWorkspaceConfigReadable rechaza un fichero de configuración de proyecto que
// existe, tiene contenido y no es JSON (o TOML, según la extensión) válido. Uno
// ausente o vacío se acepta: Axiom lo creará.
func checkWorkspaceConfigReadable(settingsPath string) error {
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("leer %s: %w", settingsPath, err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	format, parse := "JSON", func() error { _, err := filemerge.UnmarshalJSONObject(raw); return err }
	if strings.EqualFold(filepath.Ext(settingsPath), ".toml") {
		format, parse = "TOML", func() error { return filemerge.ValidateTOML(string(raw)) }
	}
	if err := parse(); err != nil {
		return fmt.Errorf("%s no es %s válido y Axiom no lo sobrescribe en un proyecto: corrígelo o muévelo y vuelve a ejecutar: %w", settingsPath, format, err)
	}
	return nil
}

// workspaceUserConfigFiles devuelve los ficheros de configuración del usuario, en la
// raíz del proyecto, que un adaptador fusiona sin registrarlos como instrucciones
// (.vscode/mcp.json de VS Code Copilot).
func workspaceUserConfigFiles(adapter agents.Adapter, workspaceDir string) []string {
	provider, ok := adapter.(agents.WorkspaceUserConfigFilesProvider)
	if !ok {
		return nil
	}
	return provider.WorkspaceUserConfigFiles(workspaceDir)
}

// needsWorkspaceSettingsPreflight indica si alguno de los agentes seleccionados tiene
// en el proyecto una configuración del usuario que no se puede sobrescribir si es
// ilegible: la de los que registran instrucciones y .vscode/mcp.json de VS Code Copilot.
func needsWorkspaceSettingsPreflight(agentIDs []model.AgentID) bool {
	return registersWorkspaceInstructions(agentIDs) || containsAgent(agentIDs, model.AgentVSCodeCopilot)
}

// withWorkspaceSettingsPreflight antepone la comprobación de la configuración de
// proyecto a los pasos de preparación cuando hay OpenCode, Kilo, Gemini, Codex o VS
// Code Copilot en workspace.
func withWorkspaceSettingsPreflight(prepare []pipeline.Step, scope InstallScope, workspaceDir string, agentIDs []model.AgentID) []pipeline.Step {
	if scope != ScopeWorkspace || !needsWorkspaceSettingsPreflight(agentIDs) {
		return prepare
	}
	step := workspaceSettingsPreflightStep{id: "prepare:workspace-settings", workspaceDir: workspaceDir, agents: agentIDs}
	return append([]pipeline.Step{step}, prepare...)
}

// staleWorkspaceConfigSignals son ficheros que solo escribe Axiom. Que alguno
// exista bajo <proyecto>/.config/<agente> delata una instalación workspace
// anterior, que escribía ahí porque trataba la raíz del proyecto como si fuera la
// carpeta de usuario.
var staleWorkspaceConfigSignals = []string{
	filepath.Join("skills", "sdd-init", "SKILL.md"),
	filepath.Join("commands", "sdd-init.md"),
	filepath.Join("plugins", "skill-registry.ts"),
	filepath.Join("plugins", "sdd-task-result-artifacts.ts"),
}

// staleWorkspaceConfigDirs devuelve las carpetas <proyecto>/.config/opencode y
// <proyecto>/.config/kilo que contienen restos de una instalación anterior.
//
// No se borra nada: ninguno de los dos agentes lee esas carpetas, pero los
// ficheros que contienen (opencode.json, AGENTS.md, skills y comandos) pueden
// mezclar contenido del usuario con el de Axiom, y no hay forma de probar cuál es
// cuál. A diferencia de .claude/mcp/engram.json, que tiene una forma exacta y
// única, aquí lo seguro es avisar y dejar la decisión al usuario.
func staleWorkspaceConfigDirs(workspaceDir string, agentIDs []model.AgentID) []string {
	candidates := map[model.AgentID]string{
		model.AgentOpenCode: filepath.Join(workspaceDir, ".config", "opencode"),
		model.AgentKilocode: filepath.Join(workspaceDir, ".config", "kilo"),
	}
	var stale []string
	for _, id := range []model.AgentID{model.AgentOpenCode, model.AgentKilocode} {
		dir, selected := candidates[id], containsAgent(agentIDs, id)
		if !selected {
			continue
		}
		for _, signal := range staleWorkspaceConfigSignals {
			if info, err := os.Stat(filepath.Join(dir, signal)); err == nil && info.Mode().IsRegular() {
				stale = append(stale, dir)
				break
			}
		}
	}
	return stale
}

// noteStaleWorkspaceConfig avisa de los restos de instalaciones anteriores.
func noteStaleWorkspaceConfig(workspaceDir string, agentIDs []model.AgentID) {
	for _, dir := range staleWorkspaceConfigDirs(workspaceDir, agentIDs) {
		fmt.Fprintf(workspaceInstructionsNoticeWriter, "Aviso: %s contiene restos de una instalación anterior de Axiom que ningún agente lee; ahora se instala en la raíz del proyecto y en su carpeta de recursos. Axiom no la borra porque puede mezclar ficheros tuyos: revísala y bórrala si ya no la necesitas.\n", dir)
	}
}
