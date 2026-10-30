package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/pipeline"
)

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

// workspaceInstructionsSettingsPaths devuelve los ficheros de configuración que
// registerWorkspaceInstructions puede reescribir: son objetivo de copia de
// seguridad aunque ningún componente seleccionado los toque.
func workspaceInstructionsSettingsPaths(workspaceDir string, scope InstallScope, adapters []agents.Adapter) []string {
	if scope != ScopeWorkspace {
		return nil
	}
	var paths []string
	for _, adapter := range adapters {
		if provider, ok := adapter.(agents.WorkspaceInstructionsProvider); ok {
			if settingsPath, _, ok := provider.WorkspaceInstructions(workspaceDir); ok && settingsPath != "" {
				paths = append(paths, settingsPath)
			}
		}
	}
	return paths
}

// workspaceInstructionsStep registra el fichero de instrucciones de Axiom de
// cada agente de la familia OpenCode del proyecto. Va después de los componentes
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
// configuración de proyecto de OpenCode o Kilo que no se puede leer como JSON.
//
// El resto de fusiones sustituyen una base ilegible por una vacía: en la
// configuración global es aceptable (la copia de seguridad previa la conserva),
// pero opencode.json de un proyecto es un fichero del usuario, normalmente
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
		provider, ok := adapter.(agents.WorkspaceInstructionsProvider)
		if !ok {
			continue
		}
		settingsPath, _, ok := provider.WorkspaceInstructions(s.workspaceDir)
		if !ok || settingsPath == "" {
			continue
		}
		raw, err := os.ReadFile(settingsPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("leer %s: %w", settingsPath, err)
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		if _, err := filemerge.UnmarshalJSONObject(raw); err != nil {
			return fmt.Errorf("%s no es JSON válido y Axiom no lo sobrescribe en un proyecto: corrígelo o muévelo y vuelve a ejecutar: %w", settingsPath, err)
		}
	}
	return nil
}

// withWorkspaceSettingsPreflight antepone la comprobación de la configuración de
// proyecto a los pasos de preparación cuando hay OpenCode o Kilo en workspace.
func withWorkspaceSettingsPreflight(prepare []pipeline.Step, scope InstallScope, workspaceDir string, agentIDs []model.AgentID) []pipeline.Step {
	if scope != ScopeWorkspace || (!containsAgent(agentIDs, model.AgentOpenCode) && !containsAgent(agentIDs, model.AgentKilocode)) {
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
