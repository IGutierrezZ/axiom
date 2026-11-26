package cli

// Auxiliares de prueba. Antes eran puntos de entrada públicos que solo llamaba la
// TUI retirada; las pruebas de sincronización e instalación los siguen usando
// como atajo para ejercitar el mismo código que recorren `axiom sync` e `install`.

import (
	"fmt"
	"os"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/planner"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// RunSyncWithSelection is the programmatic entry point for sync.
// It skips flag parsing and agent discovery — the caller provides the homeDir
// and a fully-built Selection (agents + components + options).
// This is the function the TUI calls directly to avoid CLI flag parsing.
func RunSyncWithSelection(homeDir string, selection model.Selection) (SyncResult, error) {
	return RunSyncWithSelectionScoped(homeDir, "", ScopeGlobal, selection)
}

func RunSyncWithSelectionScoped(homeDir, workspaceDir string, scope InstallScope, selection model.Selection) (SyncResult, error) {
	if workspaceDir == "" {
		workspaceDir, _ = resolveWorkspaceDir(scope, homeDir)
	}
	persistedState, persistedStateErr := StateStoreFor(scope, homeDir, workspaceDir).Read()
	if persistedStateErr != nil && !os.IsNotExist(persistedStateErr) {
		return SyncResult{Agents: selection.Agents, Selection: selection}, fmt.Errorf("read persisted installation state: %w", persistedStateErr)
	}
	background, err := resolveOpenCodeBackgroundCLI(false, "", persistedState)
	if err != nil {
		return SyncResult{Agents: selection.Agents, Selection: selection}, err
	}
	background.activationPlan, err = prepareOpenCodeBackgroundActivation(homeDir, &background, containsAgent(selection.Agents, model.AgentOpenCode))
	if err != nil {
		return SyncResult{Agents: selection.Agents, Selection: selection, Background: background}, fmt.Errorf("prepare OpenCode background activation: %w", err)
	}
	piBackground, err := resolvePiBackgroundCLI(false, "", persistedState)
	if err != nil {
		return SyncResult{Agents: selection.Agents, Selection: selection, Background: background}, err
	}
	preparePiBackgroundProjection(homeDir, &piBackground, containsAgent(selection.Agents, model.AgentPi))
	return runSyncWithSelectionScoped(homeDir, workspaceDir, scope, selection, background, piBackground)
}

// SetCommandOutputStreaming toggles whether command stdout/stderr is streamed
// directly to the terminal. It returns a restore function.
func SetCommandOutputStreaming(enabled bool) func() {
	previous := streamCommandOutput
	streamCommandOutput = enabled
	return func() {
		streamCommandOutput = previous
	}
}

func newInstallRuntime(homeDir string, scope InstallScope, channel InstallChannel, selection model.Selection, resolved planner.ResolvedPlan, profile system.PlatformProfile) (*installRuntime, error) {
	// En ámbito workspace la raíz es la del proyecto, no la carpeta desde la que
	// se lanzó el comando. El aviso al usuario lo emite quien llama.
	workspaceDir, _ := resolveWorkspaceDir(scope, homeDir)
	return newInstallRuntimeAt(homeDir, workspaceDir, scope, channel, selection, resolved, profile)
}

// Warnings lists every warning a completed sync raised, in the order the CLI
// report prints them, without the "WARNING:" prefix. It is the single source of
// truth shared by RenderSyncReport and the TUI, which shows them on its sync
// result screen.
func (r SyncResult) Warnings() []string {
	var warnings []string
	if !r.SkillRegistryRefreshed && r.SkillRegistryError != "" {
		warnings = append(warnings, skillIndexRefreshWarning(r.SkillRegistryError))
	}
	return append(warnings, r.noteWarnings()...)
}
