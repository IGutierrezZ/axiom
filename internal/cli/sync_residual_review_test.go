package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/backup"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// TestRunSyncLeavesResidualReviewInstallUntouched pins REQ-46.4 end to end: an
// installation an earlier release left with review agents, a review stop hook
// next to the user's own hooks, an OpenCode config that still defines `review-*`
// agents and the legacy review plugin syncs without error, leaves every one of
// them as it was, and writes no review agent of its own.
func TestRunSyncLeavesResidualReviewInstallUntouched(t *testing.T) {
	workspace := t.TempDir()
	t.Chdir(workspace)
	home := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
	if err := state.Write(home, state.InstallState{
		InstalledAgents:     []string{"claude-code", "opencode"},
		SelectionConfigured: true,
		Components:          []model.ComponentID{model.ComponentSDD},
	}); err != nil {
		t.Fatal(err)
	}
	restoreHome := osUserHomeDir
	restoreBackupHome := backup.UserHomeDirFn
	restoreCommand := runCommand
	restoreLookPath := cmdLookPath
	osUserHomeDir = func() (string, error) { return home, nil }
	backup.UserHomeDirFn = func() (string, error) { return home, nil }
	runCommand = func(string, ...string) error { return nil }
	cmdLookPath = func(name string) (string, error) { return "/usr/local/bin/" + name, nil }
	t.Cleanup(func() {
		osUserHomeDir = restoreHome
		backup.UserHomeDirFn = restoreBackupHome
		runCommand = restoreCommand
		cmdLookPath = restoreLookPath
	})

	residualAgent := filepath.Join(home, ".claude", "agents", "review-risk.md")
	mustWriteFile(t, residualAgent, []byte("residual review agent\n"))
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	const claudeSettings = `{
  "hooks": {
    "Stop": [
      {"matcher": "", "hooks": [
        {"type": "command", "command": "axiom review stop-hook --agent claude-code", "timeout": 60},
        {"type": "command", "command": "echo mi-hook-de-usuario"}
      ]}
    ]
  }
}`
	mustWriteFile(t, settingsPath, []byte(claudeSettings))
	opencodePath := filepath.Join(home, ".config", "opencode", "opencode.json")
	mustWriteFile(t, opencodePath, []byte(`{"agent":{"review-risk":{"mode":"subagent","prompt":"R1","permission":{"*":"deny"}}}}`))
	legacyPlugin := filepath.Join(home, ".config", "opencode", "plugins", "review-result-artifacts.ts")
	mustWriteFile(t, legacyPlugin, []byte("legacy review plugin\n"))

	result, err := RunSync([]string{"--agents", "claude-code,opencode", "--sdd-mode", "multi"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	if !result.Verify.Ready {
		t.Fatalf("Verify.Ready = false, report = %#v", result.Verify)
	}

	if got, err := os.ReadFile(residualAgent); err != nil || string(got) != "residual review agent\n" {
		t.Errorf("the residual review agent was removed or modified (err = %v, content %q)", err, got)
	}
	if got, err := os.ReadFile(legacyPlugin); err != nil || string(got) != "legacy review plugin\n" {
		t.Errorf("the legacy review plugin was removed or modified (err = %v, content %q)", err, got)
	}
	settings, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, kept := range []string{"axiom review stop-hook --agent claude-code", "echo mi-hook-de-usuario"} {
		if !strings.Contains(string(settings), kept) {
			t.Errorf("settings.json lost the hook %q:\n%s", kept, settings)
		}
	}
	rawConfig, err := os.ReadFile(opencodePath)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Agent map[string]json.RawMessage `json:"agent"`
	}
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		t.Fatalf("opencode.json is not valid JSON: %v", err)
	}
	if _, ok := config.Agent["review-risk"]; !ok {
		t.Error("opencode.json lost the residual review-risk agent")
	}
	for _, name := range []string{"review-readability", "review-reliability", "review-resilience", "review-refuter", "review-validator"} {
		if _, ok := config.Agent[name]; ok {
			t.Errorf("sync wrote the retired review agent %q", name)
		}
	}
	if _, ok := config.Agent["sdd-apply"]; !ok {
		t.Error("sync did not write the SDD agents")
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "agents", "review-refuter.md")); !os.IsNotExist(err) {
		t.Errorf("sync wrote the retired review-refuter agent (err = %v)", err)
	}
}
