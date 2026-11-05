package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/components/gga"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func TestComponentSyncStepGGAPreservesConfiguredProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))

	custom := "PROVIDER=\"ollama:llama3\"\nTIMEOUT=\"60\"\n"
	path := gga.ConfigPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	step := componentSyncStep{
		id:        "sync:gga",
		component: model.ComponentGGA,
		homeDir:   home,
		agents:    []model.AgentID{model.AgentClaudeCode},
		selection: model.Selection{},
	}
	if err := step.Run(); err != nil {
		t.Fatalf("componentSyncStep.Run() GGA error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != custom {
		t.Fatalf("sync pisó el config de GGA:\n%s", got)
	}
}
