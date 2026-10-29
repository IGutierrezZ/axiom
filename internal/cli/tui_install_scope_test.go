package cli

import (
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
