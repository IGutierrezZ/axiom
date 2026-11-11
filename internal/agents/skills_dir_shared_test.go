package agents

import (
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func TestSharesSkillsDirWithAnotherAdapter(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("COPILOT_HOME", "")

	shares := map[model.AgentID]bool{}
	for _, id := range defaultAgentIDs {
		adapter, err := NewAdapter(id)
		if err != nil {
			t.Fatal(err)
		}
		shares[id] = SharesSkillsDirWithAnotherAdapter(home, adapter)
	}
	for id, got := range shares {
		want := id == model.AgentVSCodeCopilot || id == model.AgentGitHubCopilotCLI
		if got != want {
			t.Errorf("SharesSkillsDirWithAnotherAdapter(%s) = %v, want %v", id, got, want)
		}
	}
}

// Con COPILOT_HOME, Copilot CLI deja de compartir carpeta con VS Code Copilot.
func TestSharesSkillsDirEndsWithCopilotHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("COPILOT_HOME", filepath.Join(t.TempDir(), "custom-copilot"))

	for _, id := range []model.AgentID{model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI} {
		adapter, err := NewAdapter(id)
		if err != nil {
			t.Fatal(err)
		}
		if SharesSkillsDirWithAnotherAdapter(home, adapter) {
			t.Errorf("%s no debería compartir carpeta con COPILOT_HOME definido", id)
		}
	}
}
