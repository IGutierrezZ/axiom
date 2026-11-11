package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// PhaseModelOverrides sustituye las asignaciones de modelo por fase de uno o más
// adaptadores durante una sincronización. Un mapa nil deja el adaptador como está
// (se restauran las asignaciones del estado); un mapa no nil, aunque esté vacío, es
// el resultado completo: vaciarlo quita todas las asignaciones del adaptador.
type PhaseModelOverrides struct {
	Claude   map[string]model.ClaudePhaseAssignment
	OpenCode map[string]model.ModelAssignment
	Copilot  map[string]model.CopilotPhaseAssignment
}

// apply escribe las sustituciones en la selección. Con un receptor nil no hace nada.
func (o *PhaseModelOverrides) apply(selection *model.Selection) {
	if o == nil {
		return
	}
	if o.Claude != nil {
		selection.ClaudePhaseAssignments = o.Claude
		// El mapa antiguo de solo modelo, si lo hubiera, ya no aplica: la elección del
		// usuario por fase lo sustituye entero.
		selection.ClaudeModelAssignments = nil
	}
	if o.OpenCode != nil {
		selection.ModelAssignments = o.OpenCode
	}
	if o.Copilot != nil {
		selection.CopilotPhaseAssignments = o.Copilot
	}
}

// SyncInWorkspace sincroniza los adaptadores agents en el proyecto de projectDir con
// el mismo núcleo que `axiom sync --scope workspace`, sin deducir la raíz de la
// carpeta actual del proceso ni cambiarla (sin os.Chdir). overrides sustituye las
// asignaciones de modelo por fase que nombra.
func SyncInWorkspace(projectDir string, agents []model.AgentID, overrides PhaseModelOverrides) (SyncResult, error) {
	root, err := resolveProjectRoot(projectDir)
	if err != nil {
		return SyncResult{}, err
	}
	ids := make([]string, 0, len(agents))
	for _, agent := range agents {
		ids = append(ids, string(agent))
	}
	flags, err := ParseSyncFlags([]string{"--scope", string(ScopeWorkspace), "--agent", strings.Join(ids, ",")})
	if err != nil {
		return SyncResult{}, err
	}
	homeDir, err := osUserHomeDir()
	if err != nil {
		return SyncResult{}, fmt.Errorf("resolve user home directory: %w", err)
	}
	return runSyncResolved(flags, ScopeWorkspace, homeDir, root, &overrides)
}

// resolveProjectRoot valida que projectDir sea una carpeta existente y la devuelve
// como ruta absoluta.
func resolveProjectRoot(projectDir string) (string, error) {
	root, err := filepath.Abs(strings.TrimSpace(projectDir))
	if err != nil || strings.TrimSpace(projectDir) == "" {
		// refusal:by-design operator-knowledge: solo quien hace la petición sabe en qué carpeta está el proyecto
		return "", fmt.Errorf("raíz del proyecto inválida %q", projectDir)
	}
	if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
		// refusal:by-design operator-knowledge: solo quien hace la petición sabe en qué carpeta está el proyecto
		return "", fmt.Errorf("la raíz del proyecto %q no existe o no es un directorio", root)
	}
	return filepath.Clean(root), nil
}

// ProjectPhaseModels son las asignaciones de modelo por fase guardadas en el estado
// del proyecto (.axiom/state.json), ya convertidas a los tipos del modelo.
type ProjectPhaseModels struct {
	Agents []string
	// SDDMode es el modo SDD de OpenCode guardado («single» o «multi»); vacío si no consta.
	SDDMode  string
	Claude   map[string]model.ClaudePhaseAssignment
	OpenCode map[string]model.ModelAssignment
	Copilot  map[string]model.CopilotPhaseAssignment
}

// ReadProjectPhaseModels lee el estado del proyecto de projectDir. Un proyecto sin
// estado devuelve valores vacíos y ningún error. Sin os.Chdir.
func ReadProjectPhaseModels(projectDir string) (ProjectPhaseModels, error) {
	store, err := projectStateStore(projectDir)
	if err != nil {
		return ProjectPhaseModels{}, err
	}
	current, err := store.Read()
	if errors.Is(err, os.ErrNotExist) {
		return ProjectPhaseModels{}, nil
	}
	if err != nil {
		return ProjectPhaseModels{}, fmt.Errorf("leer el estado del proyecto %s: %w", store.Path(), err)
	}
	return phaseModelsFromState(current), nil
}

func phaseModelsFromState(current state.InstallState) ProjectPhaseModels {
	out := ProjectPhaseModels{
		Agents:   append([]string(nil), current.InstalledAgents...),
		SDDMode:  string(current.SDDMode),
		Copilot:  current.CopilotAssignments(),
		OpenCode: map[string]model.ModelAssignment{},
	}
	if len(current.ClaudePhaseAssignments) > 0 {
		out.Claude = make(map[string]model.ClaudePhaseAssignment, len(current.ClaudePhaseAssignments))
		for phase, assignment := range current.ClaudePhaseAssignments {
			out.Claude[phase] = model.ClaudePhaseAssignment{Model: model.ClaudeModelAlias(assignment.Model), Effort: model.ClaudeEffort(assignment.Effort)}
		}
	} else if len(current.ClaudeModelAssignments) > 0 {
		legacy := make(map[string]model.ClaudeModelAlias, len(current.ClaudeModelAssignments))
		for phase, alias := range current.ClaudeModelAssignments {
			legacy[phase] = model.ClaudeModelAlias(alias)
		}
		out.Claude = model.ClaudePhaseAssignmentsFromLegacy(legacy)
	}
	for phase, assignment := range current.ModelAssignments {
		out.OpenCode[phase] = model.ModelAssignment{ProviderID: assignment.ProviderID, ModelID: assignment.ModelID, Effort: assignment.Effort}
	}
	return out
}

func projectStateStore(projectDir string) (state.Store, error) {
	root, err := resolveProjectRoot(projectDir)
	if err != nil {
		return state.Store{}, err
	}
	homeDir, err := osUserHomeDir()
	if err != nil {
		return state.Store{}, fmt.Errorf("resolve user home directory: %w", err)
	}
	return StateStoreFor(ScopeWorkspace, homeDir, root), nil
}

// SaveProjectPhaseModels guarda en el estado del proyecto las asignaciones de los
// adaptadores que nombra overrides (mapa completo, no un parche) y regenera en el
// proyecto los ficheros de agents. Si la regeneración falla, devuelve el estado al
// valor anterior y el error, de modo que el estado y los ficheros no se desvían.
// Es la escritura que hace el dashboard: no cambia el directorio actual del proceso.
func SaveProjectPhaseModels(projectDir string, agents []model.AgentID, overrides PhaseModelOverrides) (SyncResult, error) {
	store, err := projectStateStore(projectDir)
	if err != nil {
		return SyncResult{}, err
	}
	var previous state.InstallState
	if err := withStateStoreLock(store, func() error {
		current, readErr := store.Read()
		if errors.Is(readErr, os.ErrNotExist) {
			current = state.InstallState{}
		} else if readErr != nil {
			return fmt.Errorf("leer el estado del proyecto %s: %w; ejecuta `axiom install --scope workspace` para reescribirlo", store.Path(), readErr)
		}
		previous = current
		return store.WriteReconciled(applyOverridesToState(current, overrides))
	}); err != nil {
		return SyncResult{}, err
	}

	// Sin ningún adaptador que se pueda regenerar en el proyecto (Copilot CLI solo tiene
	// ámbito global) queda guardado el estado y no hay nada que escribir.
	if len(agents) == 0 {
		return SyncResult{NoOp: true, Scope: ScopeWorkspace}, nil
	}
	result, syncErr := SyncInWorkspace(projectDir, agents, overrides)
	if syncErr == nil {
		return result, nil
	}
	if restoreErr := withStateStoreLock(store, func() error { return store.WriteReconciled(previous) }); restoreErr != nil {
		syncErr = errors.Join(syncErr, fmt.Errorf("no se pudo devolver el estado a su valor anterior: %w", restoreErr))
	}
	return result, syncErr
}

// applyOverridesToState devuelve el estado con las asignaciones sustituidas. Un mapa
// vacío quita la clave del JSON (omitempty).
func applyOverridesToState(current state.InstallState, overrides PhaseModelOverrides) state.InstallState {
	if overrides.Claude != nil {
		current.ClaudePhaseAssignments = claudePhaseAssignmentsToState(overrides.Claude)
		current.ClaudeModelAssignments = nil
	}
	if overrides.OpenCode != nil {
		current.ModelAssignments = modelAssignmentsToState(overrides.OpenCode)
	}
	if overrides.Copilot != nil {
		current.CopilotPhaseAssignments = state.CopilotAssignmentsToState(overrides.Copilot)
	}
	return current
}
