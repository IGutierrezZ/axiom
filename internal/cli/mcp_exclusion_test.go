package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/pipeline"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// mcpExclusionHome crea un home temporal con un proyecto registrado y las exclusiones
// guardadas en su hub, sin tocar el hub real.
func mcpExclusionHome(t *testing.T, exclusions map[string][]string) (home, project string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AXIOM_STATE_DIR", "")
	project = t.TempDir()
	manager, err := hub.NewManager(filepath.Join(system.AxiomDir(home), "workspaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Register(project, "p", ""); err != nil {
		t.Fatal(err)
	}
	if exclusions != nil {
		if _, _, err := manager.SetProjectMCPExclusions(project, exclusions); err != nil {
			t.Fatal(err)
		}
	}
	return home, project
}

func findStep(plan pipeline.StagePlan, id string) pipeline.Step {
	for _, step := range plan.Apply {
		if step.ID() == id {
			return step
		}
	}
	return nil
}

func TestSyncPlanAppliesMCPExclusionsOfTheSyncedWorkspace(t *testing.T) {
	home, project := mcpExclusionHome(t, map[string][]string{"claude-code": {"work-copilot"}})
	selection := model.Selection{}

	plan := (&syncRuntime{homeDir: home, workspaceDir: project, selection: selection, backupRoot: t.TempDir(), state: &runtimeState{}}).stagePlan()
	step := findStep(plan, "sync:mcp-exclusions")
	if step == nil {
		t.Fatal("sync debe planificar el paso de exclusiones MCP")
	}
	if err := step.Run(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(claude.LocalSettingsPath(project))
	if err != nil || !strings.Contains(string(raw), `"serverName": "work-copilot"`) {
		t.Fatalf("settings.local.json = %q (%v)", raw, err)
	}

	if without := (&syncRuntime{homeDir: home, selection: selection, backupRoot: t.TempDir(), state: &runtimeState{}}).stagePlan(); findStep(without, "sync:mcp-exclusions") != nil {
		t.Error("sin carpeta de workspace no debe haber paso de exclusiones")
	}
}

func TestMCPExclusionSyncStepSurfacesFailuresAsNotesWithoutError(t *testing.T) {
	home, project := mcpExclusionHome(t, map[string][]string{"claude-code": {"a"}})
	broken := claude.LocalSettingsPath(project)
	if err := os.MkdirAll(filepath.Dir(broken), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(broken, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	var notes []string

	err := mcpExclusionSyncStep{id: "x", homeDir: home, workspaceDir: project, notes: &notes}.Run()

	if err != nil {
		t.Fatalf("el paso no debe fallar: %v", err)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "exclusiones de servidores MCP") {
		t.Fatalf("notas = %v", notes)
	}
}

func TestEnsureAppliesMCPExclusionsToItsWorktrees(t *testing.T) {
	_, project := mcpExclusionHome(t, map[string][]string{"claude-code": {"work-copilot"}})
	worktree := t.TempDir()
	result := branch.Result{
		Status:       branch.StatusReady,
		Repositories: []branch.RepositoryResult{{Branches: []branch.BranchResult{{Name: "feat/x", Worktree: worktree}, {Name: "main"}}}},
	}

	applyMCPExclusionsAfterEnsure(&result, project)

	raw, err := os.ReadFile(claude.LocalSettingsPath(worktree))
	if err != nil || !strings.Contains(string(raw), `"serverName": "work-copilot"`) {
		t.Fatalf("el worktree no recibió la exclusión: %q (%v)", raw, err)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("avisos inesperados: %v", result.Warnings)
	}
}

func TestEnsureMCPExclusionsAreNoOpWithoutPreferenceOrWorktrees(t *testing.T) {
	_, project := mcpExclusionHome(t, nil)
	worktree := t.TempDir()
	result := branch.Result{
		Status:       branch.StatusReady,
		Repositories: []branch.RepositoryResult{{Branches: []branch.BranchResult{{Name: "feat/x", Worktree: worktree}}}},
	}

	applyMCPExclusionsAfterEnsure(&result, project)

	if _, err := os.Stat(claude.LocalSettingsPath(worktree)); !os.IsNotExist(err) {
		t.Fatalf("sin preferencia no debe crearse nada: %v", err)
	}
}
