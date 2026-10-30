package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/pipeline"
	"github.com/IGutierrezZ/axiom/v3/internal/planner"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// captureTUIInstallRuntime sustituye el plan de la TUI por uno vacío y devuelve
// el ámbito y la raíz de workspace con los que se construyó el runtime.
func captureTUIInstallRuntime(t *testing.T, home string) (InstallScope, string) {
	t.Helper()
	var gotScope InstallScope
	var gotWorkspace string
	previous := tuiInstallStagePlan
	tuiInstallStagePlan = func(runtime *installRuntime) pipeline.StagePlan {
		gotScope = runtime.scope
		gotWorkspace = runtime.workspaceDir
		return pipeline.StagePlan{}
	}
	t.Cleanup(func() { tuiInstallStagePlan = previous })

	result, _ := ExecuteTUIInstallWithBackgroundAndOrchestrator(home, model.Selection{}, planner.ResolvedPlan{}, system.PlatformProfile{}, model.OpenCodeBackgroundOff, model.PiBackgroundIntent(""), nil)
	if result.Err != nil {
		t.Fatalf("ExecuteTUIInstallWithBackgroundAndOrchestrator() error = %v", result.Err)
	}
	return gotScope, gotWorkspace
}

func TestTUIInstallDefaultsToWorkspaceScope(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv(ScopeAxiomEnvVar, "")
	t.Setenv(ScopeGentleAIEnvVar, "")
	t.Chdir(project)

	scope, workspaceDir := captureTUIInstallRuntime(t, home)
	if scope != ScopeWorkspace {
		t.Fatalf("scope = %q, want %q", scope, ScopeWorkspace)
	}
	if want, _ := workspace.ResolveRoot(project, home); workspaceDir != want {
		t.Fatalf("workspaceDir = %q, want %q", workspaceDir, want)
	}
}

func TestTUIInstallHonoursExplicitGlobalScope(t *testing.T) {
	home := t.TempDir()
	t.Setenv(ScopeAxiomEnvVar, string(ScopeGlobal))
	t.Chdir(t.TempDir())

	if scope, _ := captureTUIInstallRuntime(t, home); scope != ScopeGlobal {
		t.Fatalf("scope = %q, want %q", scope, ScopeGlobal)
	}
}

func TestTUIInstallRejectsInvalidScope(t *testing.T) {
	t.Setenv(ScopeAxiomEnvVar, "galaxia")
	t.Chdir(t.TempDir())

	result, _ := ExecuteTUIInstallWithBackgroundAndOrchestrator(t.TempDir(), model.Selection{}, planner.ResolvedPlan{}, system.PlatformProfile{}, model.OpenCodeBackgroundOff, model.PiBackgroundIntent(""), nil)
	if result.Err == nil {
		t.Fatal("ExecuteTUIInstallWithBackgroundAndOrchestrator() error = nil, want unsupported scope")
	}
}

// La TUI aplica el mismo filtro que la CLI: con el ámbito workspace el plan no
// contiene agentes sin modo workspace y el aviso viaja en las acciones manuales,
// sin escribir por stderr.
func TestTUIInstallWorkspaceOmitsAgentsWithoutWorkspaceMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv(ScopeAxiomEnvVar, string(ScopeWorkspace))
	t.Chdir(t.TempDir())
	notices := captureWorkspaceAgentsNotice(t)

	var planned []model.AgentID
	var resolvedAgents []model.AgentID
	previous := tuiInstallStagePlan
	tuiInstallStagePlan = func(runtime *installRuntime) pipeline.StagePlan {
		planned = runtime.selection.Agents
		resolvedAgents = runtime.resolved.Agents
		return pipeline.StagePlan{}
	}
	t.Cleanup(func() { tuiInstallStagePlan = previous })

	agentsSelected := []model.AgentID{model.AgentClaudeCode, model.AgentCodex, model.AgentWindsurf}
	result, _ := ExecuteTUIInstallWithBackgroundAndOrchestrator(home, model.Selection{Agents: agentsSelected}, planner.ResolvedPlan{Agents: agentsSelected}, system.PlatformProfile{}, model.OpenCodeBackgroundOff, model.PiBackgroundIntent(""), nil)
	if result.Err != nil {
		t.Fatalf("ExecuteTUIInstallWithBackgroundAndOrchestrator() error = %v", result.Err)
	}

	want := []model.AgentID{model.AgentClaudeCode}
	if !slices.Equal(planned, want) || !slices.Equal(resolvedAgents, want) {
		t.Errorf("agentes del plan = %v / %v, want %v", planned, resolvedAgents, want)
	}
	if notices.Len() != 0 {
		t.Errorf("la TUI no debe escribir por stderr:\n%s", notices)
	}
	joined := strings.Join(result.ManualActions, "\n")
	for _, id := range []model.AgentID{model.AgentCodex, model.AgentWindsurf} {
		if !strings.Contains(joined, string(id)+" omitido") {
			t.Errorf("falta el aviso de %s entre las acciones manuales:\n%s", id, joined)
		}
	}
}

// Sin ningún agente con modo workspace la TUI no instala nada y lo explica.
func TestTUIInstallWorkspaceWithOnlyAgentsWithoutWorkspaceModeFails(t *testing.T) {
	t.Setenv(ScopeAxiomEnvVar, string(ScopeWorkspace))
	t.Chdir(t.TempDir())
	previous := tuiInstallStagePlan
	tuiInstallStagePlan = func(*installRuntime) pipeline.StagePlan {
		t.Error("no debe construirse ningún plan sin agentes con modo workspace")
		return pipeline.StagePlan{}
	}
	t.Cleanup(func() { tuiInstallStagePlan = previous })

	only := []model.AgentID{model.AgentCodex}
	result, _ := ExecuteTUIInstallWithBackgroundAndOrchestrator(t.TempDir(), model.Selection{Agents: only}, planner.ResolvedPlan{Agents: only}, system.PlatformProfile{}, model.OpenCodeBackgroundOff, model.PiBackgroundIntent(""), nil)
	if result.Err == nil || !strings.Contains(result.Err.Error(), "ningún agente") {
		t.Fatalf("error = %v, want el de ningún agente con modo workspace", result.Err)
	}
}

// Con el ámbito global la TUI conserva todos los agentes.
func TestTUIInstallGlobalKeepsAgentsWithoutWorkspaceMode(t *testing.T) {
	t.Setenv(ScopeAxiomEnvVar, string(ScopeGlobal))
	t.Chdir(t.TempDir())
	var planned []model.AgentID
	previous := tuiInstallStagePlan
	tuiInstallStagePlan = func(runtime *installRuntime) pipeline.StagePlan {
		planned = runtime.selection.Agents
		return pipeline.StagePlan{}
	}
	t.Cleanup(func() { tuiInstallStagePlan = previous })

	selected := []model.AgentID{model.AgentClaudeCode, model.AgentCodex}
	result, _ := ExecuteTUIInstallWithBackgroundAndOrchestrator(t.TempDir(), model.Selection{Agents: selected}, planner.ResolvedPlan{Agents: selected}, system.PlatformProfile{}, model.OpenCodeBackgroundOff, model.PiBackgroundIntent(""), nil)
	if result.Err != nil {
		t.Fatalf("error = %v", result.Err)
	}
	if !slices.Equal(planned, selected) {
		t.Errorf("agentes del plan = %v, want %v", planned, selected)
	}
}
