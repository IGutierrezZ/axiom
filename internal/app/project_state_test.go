package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/planner"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// projectStateTUIFixture deja como directorio actual un proyecto con `.git` y
// devuelve su raíz y un HOME aislado. Sin AXIOM_INSTALL_SCOPE el ámbito de la TUI
// es workspace.
func projectStateTUIFixture(t *testing.T) (root, home string) {
	t.Helper()
	t.Setenv(cli.ScopeAxiomEnvVar, "")
	t.Setenv(cli.ScopeGentleAIEnvVar, "")
	t.Setenv("AXIOM_STATE_DIR", "")
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root, home
}

func claudeAssignmentOverrides(alias model.ClaudeModelAlias) *model.SyncOverrides {
	return &model.SyncOverrides{
		TargetAgents:           []model.AgentID{model.AgentClaudeCode},
		ClaudePhaseAssignments: model.ClaudePhaseAssignmentsFromLegacy(map[string]model.ClaudeModelAlias{"sdd-apply": alias}),
	}
}

// TestTuiSyncWorkspacePersistsAssignmentsInTheProject cubre el hueco que había: la
// sincronización de la TUI en ámbito workspace no persistía las asignaciones.
func TestTuiSyncWorkspacePersistsAssignmentsInTheProject(t *testing.T) {
	root, home := projectStateTUIFixture(t)

	if _, err := tuiSync(home)(claudeAssignmentOverrides(model.ClaudeModelSonnet)); err != nil {
		t.Fatalf("tuiSync() error = %v", err)
	}

	project, err := state.ProjectStore(home, root).Read()
	if err != nil {
		t.Fatalf("el estado del proyecto debe existir tras el sync: %v", err)
	}
	if got := project.ClaudePhaseAssignments["sdd-apply"]; got.Model != string(model.ClaudeModelSonnet) {
		t.Errorf("asignación persistida = %#v, want sonnet", got)
	}
	if _, err := os.Stat(state.ProjectPath(root)); err != nil {
		t.Errorf("falta %s: %v", state.ProjectPath(root), err)
	}
	if _, err := state.Read(home); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("la TUI en workspace no debe escribir el estado global; Read() error = %v", err)
	}
}

func TestTuiSyncRestoresTheProjectAssignmentsNotTheGlobalOnes(t *testing.T) {
	root, home := projectStateTUIFixture(t)
	if err := state.Write(home, state.InstallState{
		InstalledAgents:        []string{string(model.AgentClaudeCode)},
		ClaudeModelAssignments: map[string]string{"sdd-apply": "haiku"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := state.ProjectStore(home, root).Write(state.InstallState{
		InstalledAgents:        []string{string(model.AgentClaudeCode)},
		ClaudeModelAssignments: map[string]string{"sdd-apply": "opus"},
	}); err != nil {
		t.Fatal(err)
	}

	var selection model.Selection
	loadPersistedAssignments(cli.StateStoreFor(cli.ScopeWorkspace, home, root), &selection)

	if got := selection.ClaudeModelAssignments["sdd-apply"]; got != model.ClaudeModelOpus {
		t.Errorf("la asignación restaurada = %q, want la del proyecto (opus)", got)
	}
}

func TestTuiSyncOfTwoProjectsKeepsEachAssignment(t *testing.T) {
	rootA, home := projectStateTUIFixture(t)
	rootB, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(rootB, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := tuiSync(home)(claudeAssignmentOverrides(model.ClaudeModelSonnet)); err != nil {
		t.Fatalf("tuiSync(A) error = %v", err)
	}
	t.Chdir(rootB)
	if _, err := tuiSync(home)(claudeAssignmentOverrides(model.ClaudeModelOpus)); err != nil {
		t.Fatalf("tuiSync(B) error = %v", err)
	}

	a, errA := state.ProjectStore(home, rootA).Read()
	b, errB := state.ProjectStore(home, rootB).Read()
	if errA != nil || errB != nil {
		t.Fatalf("lectura de A/B = %v / %v", errA, errB)
	}
	if got := a.ClaudePhaseAssignments["sdd-apply"].Model; got != string(model.ClaudeModelSonnet) {
		t.Errorf("A = %q, want sonnet", got)
	}
	if got := b.ClaudePhaseAssignments["sdd-apply"].Model; got != string(model.ClaudeModelOpus) {
		t.Errorf("B = %q, want opus", got)
	}
}

func TestTuiExecuteWorkspacePersistsInstallStateInTheProject(t *testing.T) {
	root, home := projectStateTUIFixture(t)
	previous := appUserHomeDir
	appUserHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { appUserHomeDir = previous })
	selection := model.Selection{Preset: model.PresetCustom, Components: []model.ComponentID{}, Skills: []model.SkillID{}, SDDMode: model.SDDModeMulti, Persona: model.PersonaNeutral}

	result := tuiExecuteWithBackground(selection, planner.ResolvedPlan{}, system.DetectionResult{}, "", "", "", "", nil)

	if result.Err != nil {
		t.Fatalf("tuiExecuteWithBackground() error = %v", result.Err)
	}
	project, err := state.ProjectStore(home, root).Read()
	if err != nil || !project.SelectionConfigured || project.Persona != string(model.PersonaNeutral) {
		t.Fatalf("estado del proyecto = %#v, error = %v", project, err)
	}
	if _, err := os.Stat(state.ProjectPath(root)); err != nil {
		t.Errorf("falta %s: %v", state.ProjectPath(root), err)
	}
	if _, err := state.Read(home); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("la instalación de la TUI en workspace no debe escribir el estado global; Read() error = %v", err)
	}
}

func TestInstallStateStoreFollowsTheInstallScope(t *testing.T) {
	root, home := projectStateTUIFixture(t)

	if store := installStateStore(home); !store.IsProject() || store.Path() != state.ProjectPath(root) {
		t.Fatalf("installStateStore() = %q, want el estado del proyecto %q", store.Path(), state.ProjectPath(root))
	}
	t.Setenv(cli.ScopeAxiomEnvVar, string(cli.ScopeGlobal))
	if store := installStateStore(home); store.IsProject() {
		t.Errorf("con AXIOM_INSTALL_SCOPE=global el almacén debe ser el de usuario: %q", store.Path())
	}
}
