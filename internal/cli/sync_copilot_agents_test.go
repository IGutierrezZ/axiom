package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// axiom sync regenera los agentes `.agent.md` de Copilot con las asignaciones del
// estado, y los dos agentes que comparten ~/.copilot/agents dejan un único fichero
// por fase.
func TestRunSync_WritesCopilotAgentsFromPersistedAssignments(t *testing.T) {
	home := t.TempDir()
	if err := state.Write(home, state.InstallState{
		InstalledAgents: []string{"vscode-copilot", "github-copilot-cli"},
		CopilotPhaseAssignments: map[string]state.CopilotPhaseAssignmentState{
			"sdd-apply": {Model: "Claude Sonnet 4.6 (copilot)", Effort: "high"},
		},
	}); err != nil {
		t.Fatalf("state.Write() error = %v", err)
	}
	setSyncTestHome(t, home)

	if _, err := RunSync([]string{"--agents", "vscode-copilot,github-copilot-cli"}); err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	agentsDir := filepath.Join(home, ".copilot", "agents")
	apply, err := os.ReadFile(filepath.Join(agentsDir, "sdd-apply.agent.md"))
	if err != nil {
		t.Fatalf("sync no escribió sdd-apply.agent.md: %v", err)
	}
	for _, line := range []string{"model: Claude Sonnet 4.6 (copilot)\n", "reasoning-effort: high\n"} {
		if !strings.Contains(string(apply), line) {
			t.Errorf("sdd-apply.agent.md no contiene %q:\n%s", line, apply)
		}
	}
	design, err := os.ReadFile(filepath.Join(agentsDir, "sdd-design.agent.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(design), "model:") || strings.Contains(string(design), "reasoning-effort:") {
		t.Errorf("sdd-design no tiene asignación y no debe llevar modelo:\n%s", design)
	}
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".agent.md") {
			t.Errorf("%s no es un .agent.md", entry.Name())
		}
	}

	// Una segunda sincronización deja los mismos bytes.
	if _, err := RunSync([]string{"--agents", "vscode-copilot,github-copilot-cli"}); err != nil {
		t.Fatalf("segunda RunSync() error = %v", err)
	}
	again, err := os.ReadFile(filepath.Join(agentsDir, "sdd-apply.agent.md"))
	if err != nil || string(again) != string(apply) {
		t.Errorf("la segunda sincronización cambió sdd-apply.agent.md (%v)", err)
	}
}
