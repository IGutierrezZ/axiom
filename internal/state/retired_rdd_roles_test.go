package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// TestReadToleratesRetiredRDDRolesAndSkill pins that a state written by a
// release that still shipped RDD loads without error: the model assignments
// of the retired `rdd-*`/`review-*` roles (Codex, Claude, OpenCode) and the
// retired skill stay inert data, and the rest of the state keeps its values.
func TestReadToleratesRetiredRDDRolesAndSkill(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, stateDir), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	legacy := `{
  "installed_agents": ["claude-code", "codex", "opencode"],
  "skills": ["go-testing", "rdd-defect-workflow"],
  "claude_phase_assignments": {
    "sdd-apply": {"model": "sonnet"},
    "risk": {"model": "haiku"},
    "refuter": {"model": "haiku"}
  },
  "codexModelAssignments": {"sdd-apply": "medium", "rdd-risk": "high"},
  "codexPhaseModelAssignments": {"sdd-apply": "gpt-6-luna", "rdd-validator": "gpt-6-astra"},
  "model_assignments": {
    "sdd-apply": {"provider_id": "openai", "model_id": "gpt-5"},
    "review-risk": {"provider_id": "openai", "model_id": "gpt-5"},
    "review-refuter": {"provider_id": "openai", "model_id": "gpt-5"}
  },
  "pending_sync": true
}
`
	if err := os.WriteFile(Path(home), []byte(legacy), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := Read(home)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !got.PendingSync || len(got.InstalledAgents) != 3 {
		t.Fatalf("the rest of the state changed: %+v", got)
	}
	if len(got.Skills) != 2 || got.Skills[1] != model.SkillID("rdd-defect-workflow") {
		t.Fatalf("the persisted skill selection must load untouched, got %v", got.Skills)
	}
	if _, ok := got.ClaudePhaseAssignments["sdd-apply"]; !ok {
		t.Fatalf("the live Claude assignment was lost: %+v", got.ClaudePhaseAssignments)
	}
	if got.CodexPhaseModelAssignments["sdd-apply"] != "gpt-6-luna" || got.ModelAssignments["sdd-apply"].ModelID != "gpt-5" {
		t.Fatalf("the live Codex or OpenCode assignment was lost: %+v / %+v", got.CodexPhaseModelAssignments, got.ModelAssignments)
	}
}
