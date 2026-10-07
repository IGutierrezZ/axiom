package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/kiro"
	"github.com/IGutierrezZ/axiom/v3/internal/backup"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/components/persona"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/pipeline"
	"github.com/IGutierrezZ/axiom/v3/internal/planner"
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
	managed := filemerge.InjectMarkdownSection(string(body), filemerge.SDDOrchestratorSectionID, "managed\n")
	if err := os.WriteFile(current, []byte(managed), 0o644); err != nil {
		t.Fatal(err)
	}
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

// TestLegacyPromptFilesAreSnapshottedBeforeInstallAndSync proves the deletion is
// inside the pre-apply backup of both flows, and that restoring that snapshot,
// which is what a rollback does, brings the retired file back.
func TestLegacyPromptFilesAreSnapshottedBeforeInstallAndSync(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	adapter := kiro.NewAdapter()
	adapters := []agents.Adapter{adapter}
	if _, err := persona.Inject(home, adapter, model.PersonaGentleman); err != nil {
		t.Fatal(err)
	}
	current := adapter.SystemPromptFile(home)
	body, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	managed := filemerge.InjectMarkdownSection(string(body), filemerge.SDDOrchestratorSectionID, "managed\n")
	if err := os.WriteFile(current, []byte(managed), 0o644); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(filepath.Dir(current), "gentle-ai.md")
	if err := os.WriteFile(legacy, []byte(managed), 0o644); err != nil {
		t.Fatal(err)
	}

	selection := model.Selection{Agents: []model.AgentID{model.AgentKiroIDE}, Components: []model.ComponentID{model.ComponentPersona}}
	syncTargets, err := syncBackupTargetsScoped(home, workspace, ScopeGlobal, selection, adapters)
	if err != nil {
		t.Fatal(err)
	}
	installTargets, err := backupTargets(home, workspace, ScopeGlobal, selection, planner.ResolvedPlan{Agents: selection.Agents, OrderedComponents: selection.Components})
	if err != nil {
		t.Fatal(err)
	}
	for name, targets := range map[string][]string{"sync": syncTargets, "install": installTargets} {
		if !slices.Contains(targets, legacy) {
			t.Fatalf("%s backup targets lack the legacy prompt %s: %v", name, legacy, targets)
		}
	}

	manifest, err := backup.NewSnapshotter().Create(filepath.Join(t.TempDir(), "snapshot"), syncTargets)
	if err != nil {
		t.Fatal(err)
	}
	step := legacyPromptRetirementStep{
		state:     &runtimeState{},
		adapters:  adapters,
		targetDir: func(agents.Adapter) string { return home },
	}
	if err := step.Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("retirement did not remove the managed legacy prompt: %v", err)
	}

	if err := (backup.RestoreService{Roots: []string{home}}).Restore(manifest); err != nil {
		t.Fatalf("rollback restore: %v", err)
	}
	if got, err := os.ReadFile(legacy); err != nil || string(got) != managed {
		t.Fatalf("rollback did not restore the legacy prompt: %v / %q", err, got)
	}
}
