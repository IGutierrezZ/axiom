package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/kiro"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/components/persona"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/pipeline"
)

func TestLegacyPromptAdaptersSelectsOnlyRenamedPromptAgents(t *testing.T) {
	selected := legacyPromptAdapters([]agents.Adapter{claude.NewAdapter(), kiro.NewAdapter()})
	if len(selected) != 1 || selected[0].Agent() != model.AgentKiroIDE {
		t.Fatalf("legacyPromptAdapters() = %v, want only kiro-ide", selected)
	}
}

// TestLegacyPromptRetirementStepReportsRetiredAndKeptFiles runs the step as the
// sync pipeline does and checks both outcomes reach the places sync reports
// from: the removed file as a changed file, the kept one as a warning.
func TestLegacyPromptRetirementStepReportsRetiredAndKeptFiles(t *testing.T) {
	home := t.TempDir()
	adapter := kiro.NewAdapter()
	if _, err := persona.Inject(home, adapter, model.PersonaGentleman); err != nil {
		t.Fatal(err)
	}
	current := adapter.SystemPromptFile(home)
	body, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	managed := filemerge.InjectMarkdownSection(string(body), "sdd-orchestrator", "managed\n")
	legacy := filepath.Join(filepath.Dir(current), "gentle-ai.md")

	step := legacyPromptRetirementStep{
		id:        "sync:retire-legacy-prompt-files",
		state:     &runtimeState{},
		adapters:  []agents.Adapter{adapter},
		targetDir: func(agents.Adapter) string { return home },
	}

	if err := os.WriteFile(legacy, []byte(managed), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := step.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(step.state.retiredFiles) != 1 || step.state.retiredFiles[0] != legacy || len(step.state.retirementNotes) != 0 {
		t.Fatalf("after a managed legacy file: retired=%q notes=%q", step.state.retiredFiles, step.state.retirementNotes)
	}

	step.state = &runtimeState{}
	if err := os.WriteFile(legacy, []byte(managed+"\nMy own rule.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := step.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(step.state.retiredFiles) != 0 || len(step.state.retirementNotes) != 1 {
		t.Fatalf("after an edited legacy file: retired=%q notes=%q", step.state.retiredFiles, step.state.retirementNotes)
	}

	// The kept-file warning is what the sync report and the TUI print.
	result := SyncResult{Execution: pipeline.ExecutionResult{ManualActions: step.state.retirementNotes}}
	if warnings := result.Warnings(); len(warnings) != 1 || !strings.Contains(warnings[0], legacy) {
		t.Fatalf("Warnings() = %q, want one warning naming %s", warnings, legacy)
	}
}
