package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// projectStateTUIFixture deja como directorio actual un proyecto con `.git` y
// devuelve su raíz y un HOME aislado. Sin AXIOM_INSTALL_SCOPE el ámbito de la TUI
// es workspace.
func projectStateTUIFixture(t *testing.T) (root, home string) {
	t.Helper()
	t.Setenv(cli.ScopeAxiomEnvVar, "")
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

func TestInstallStateStoreFollowsTheInstallScope(t *testing.T) {
	root, home := projectStateTUIFixture(t)

	if store := installStateStore(home); !store.IsProject() || store.Path() != state.ProjectPath(root) {
		t.Fatalf("installStateStore() = %q, want el estado del proyecto %q", store.Path(), state.ProjectPath(root))
	}
	// La UI no recibe --scope: AXIOM_INSTALL_SCOPE=global ya no la lleva al estado
	// de usuario y sigue trabajando con el del proyecto.
	t.Setenv(cli.ScopeAxiomEnvVar, string(cli.ScopeGlobal))
	if store := installStateStore(home); !store.IsProject() || store.Path() != state.ProjectPath(root) {
		t.Errorf("con AXIOM_INSTALL_SCOPE=global el almacén debe seguir siendo el del proyecto %q: %q", state.ProjectPath(root), store.Path())
	}
}
