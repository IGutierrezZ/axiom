package permissions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/copilotcli"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// Axiom no gestiona ningún modelo de permisos de Copilot CLI: no hay overlay, no
// crea el settings.json y no toca el que ya existe.
func TestInjectCopilotCLIWritesNothingAndLeavesUserSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("COPILOT_HOME", "")
	adapter := copilotcli.NewAdapter()

	if Overlay(model.AgentGitHubCopilotCLI) != nil {
		t.Fatal("Copilot CLI no debe tener overlay de permisos")
	}

	settings := adapter.SettingsPath(home)
	result, err := Inject(home, adapter)
	if err != nil || result.Changed {
		t.Fatalf("Inject = %+v, %v; want sin efecto", result, err)
	}
	if _, err := os.Stat(settings); !os.IsNotExist(err) {
		t.Fatalf("Inject creó %q (err=%v)", settings, err)
	}

	const user = `{"trusted_folders":["/work"],"banner":"never"}`
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(user), 0o644); err != nil {
		t.Fatal(err)
	}
	if result, err := Inject(home, adapter); err != nil || result.Changed {
		t.Fatalf("Inject con settings del usuario = %+v, %v; want sin efecto", result, err)
	}
	if got, _ := os.ReadFile(settings); string(got) != user {
		t.Fatalf("settings.json cambió: %s", got)
	}
}
