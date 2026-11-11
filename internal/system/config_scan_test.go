package system

import (
	"os"
	"path/filepath"
	"testing"
)

// TestScanConfigs_ReturnsAllKnownAgentsWithExistsFlag verifies the canonical
// ScanConfigs contract: ALL known registry agents are returned, with Exists=true
// for those whose config dir is present on disk and Exists=false for those absent.
//
// This is the TUI contract: the detection screen shows "present"/"missing" for
// every known agent. The shim must enumerate all adapters from the default
// registry, not just the ones that are installed.
func TestScanConfigs_ReturnsAllKnownAgentsWithExistsFlag(t *testing.T) {
	home := t.TempDir()

	// Create only claude-code config dir — others intentionally absent.
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	configs := ScanConfigs(home)

	// Must return at least as many entries as the registry has adapters with
	// a non-empty GlobalConfigDir. Currently 16 agents are supported.
	if len(configs) < 16 {
		t.Fatalf("ScanConfigs() returned %d entries, want >= 16; got %v", len(configs), configs)
	}

	// Find claude — must be Exists=true.
	var claudeState *ConfigState
	for i := range configs {
		if configs[i].Agent == "claude-code" {
			claudeState = &configs[i]
			break
		}
	}
	if claudeState == nil {
		t.Fatalf("ScanConfigs() missing claude-code entry; got %v", configs)
	}
	if !claudeState.Exists {
		t.Errorf("ScanConfigs() claude-code Exists = false, want true (dir was created)")
	}
	if !claudeState.IsDirectory {
		t.Errorf("ScanConfigs() claude-code IsDirectory = false, want true")
	}

	// All other agents must appear with Exists=false (their dirs weren't created).
	existsTrueCount := 0
	for _, c := range configs {
		if c.Exists {
			existsTrueCount++
		}
	}
	if existsTrueCount != 1 {
		t.Errorf("ScanConfigs() Exists=true count = %d, want 1 (only claude-code created); states: %v", existsTrueCount, configs)
	}
}

// TestScanConfigs_AgentFieldMatchesModelAgentID verifies that the Agent field
// in each ConfigState matches the canonical model.AgentID string values used
// by the TUI and validate.go switch statements.
func TestScanConfigs_AgentFieldMatchesModelAgentID(t *testing.T) {
	home := t.TempDir()
	configs := ScanConfigs(home)

	// These are the AgentID string values the TUI switch statements check.
	knownAgents := map[string]bool{
		"claude-code":        false,
		"opencode":           false,
		"kilocode":           false,
		"gemini-cli":         false,
		"cursor":             false,
		"vscode-copilot":     false,
		"codex":              false,
		"antigravity":        false,
		"windsurf":           false,
		"kimi":               false,
		"qwen-code":          false,
		"kiro-ide":           false,
		"openclaw":           false,
		"pi":                 false,
		"trae-ide":           false,
		"hermes":             false,
		"github-copilot-cli": false,
	}

	for _, c := range configs {
		if _, known := knownAgents[c.Agent]; known {
			knownAgents[c.Agent] = true
		}
	}

	// All known agents must appear.
	for agent, seen := range knownAgents {
		if !seen {
			t.Errorf("ScanConfigs() missing agent %q — TUI switch statements require it; got agents: %v", agent, agentNames(configs))
		}
	}
}

// TestScanConfigs_PathFieldIsNonEmpty verifies that every ConfigState has a
// non-empty Path — the TUI and validate.go use the path for display purposes.
func TestScanConfigs_PathFieldIsNonEmpty(t *testing.T) {
	home := t.TempDir()
	configs := ScanConfigs(home)

	for _, c := range configs {
		if c.Path == "" {
			t.Errorf("ScanConfigs() agent %q has empty Path — all entries must have non-empty paths", c.Agent)
		}
	}
}

// TestScanConfigs_ExistsFalseWhenDirAbsent verifies that agents whose
// GlobalConfigDir does not exist on disk have Exists=false.
func TestScanConfigs_ExistsFalseWhenDirAbsent(t *testing.T) {
	home := t.TempDir()
	// No dirs created — all agents should have Exists=false.

	configs := ScanConfigs(home)

	for _, c := range configs {
		if c.Exists {
			t.Errorf("ScanConfigs() agent %q Exists = true, want false (no dirs created)", c.Agent)
		}
	}
}

// TestScanConfigs_IsDirectorySetForExistingDirs verifies that IsDirectory is
// set correctly for existing directories.
func TestScanConfigs_IsDirectorySetForExistingDirs(t *testing.T) {
	home := t.TempDir()

	// Create two agent dirs.
	for _, rel := range []string{".claude", ".config/opencode"} {
		if err := os.MkdirAll(filepath.Join(home, rel), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", rel, err)
		}
	}

	configs := ScanConfigs(home)

	claudeFound, opencodeFound := false, false
	for _, c := range configs {
		switch c.Agent {
		case "claude-code":
			claudeFound = true
			if !c.Exists || !c.IsDirectory {
				t.Errorf("claude-code: Exists=%v IsDirectory=%v, want both true", c.Exists, c.IsDirectory)
			}
		case "opencode":
			opencodeFound = true
			if !c.Exists || !c.IsDirectory {
				t.Errorf("opencode: Exists=%v IsDirectory=%v, want both true", c.Exists, c.IsDirectory)
			}
		}
	}

	if !claudeFound {
		t.Error("ScanConfigs() missing claude-code entry")
	}
	if !opencodeFound {
		t.Error("ScanConfigs() missing opencode entry")
	}
}

// agentNames extracts agent name strings for error messages.
func agentNames(configs []ConfigState) []string {
	names := make([]string, len(configs))
	for i, c := range configs {
		names[i] = c.Agent
	}
	return names
}

// useRealHome hace que os.UserHomeDir devuelva home: solo entonces se respeta
// COPILOT_HOME.
func useRealHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func scanStateFor(t *testing.T, configs []ConfigState, agent string) ConfigState {
	t.Helper()
	for _, c := range configs {
		if c.Agent == agent {
			return c
		}
	}
	t.Fatalf("ScanConfigs() missing %s entry", agent)
	return ConfigState{}
}

// Copilot CLI y VS Code Copilot comparten ~/.copilot, pero ninguno se da por
// presente solo porque la carpeta exista.
func TestScanConfigs_CopilotCLIAndVSCodeDoNotCollide(t *testing.T) {
	home := t.TempDir()
	useRealHome(t, home)
	t.Setenv("COPILOT_HOME", "")
	root := filepath.Join(home, ".copilot")

	// Solo las skills compartidas: ni uno ni otro.
	if err := os.MkdirAll(filepath.Join(root, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	configs := ScanConfigs(home)
	if cli := scanStateFor(t, configs, "github-copilot-cli"); cli.Exists || cli.Path != root {
		t.Fatalf("github-copilot-cli = %+v, want ausente en %s", cli, root)
	}
	if vs := scanStateFor(t, configs, "vscode-copilot"); vs.Exists {
		t.Fatalf("vscode-copilot = %+v, want ausente sin extensión", vs)
	}

	// Estado propio del CLI: solo el CLI.
	if err := os.WriteFile(filepath.Join(root, "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	configs = ScanConfigs(home)
	if !scanStateFor(t, configs, "github-copilot-cli").Exists {
		t.Fatal("github-copilot-cli debe detectarse con settings.json")
	}
	if scanStateFor(t, configs, "vscode-copilot").Exists {
		t.Fatal("vscode-copilot no debe detectarse por la carpeta del CLI")
	}

	// La extensión de VS Code activa VS Code Copilot.
	for _, ext := range []string{"github.copilot-chat-0.12.0", "github.copilot-1.2.3"} {
		if err := os.MkdirAll(filepath.Join(home, ".vscode", "extensions", ext), 0o755); err != nil {
			t.Fatal(err)
		}
		if !scanStateFor(t, ScanConfigs(home), "vscode-copilot").Exists {
			t.Errorf("vscode-copilot Exists = false, want true con %s", ext)
		}
	}
}

func TestScanConfigs_UsesCopilotHome(t *testing.T) {
	home := t.TempDir()
	useRealHome(t, home)
	root := filepath.Join(t.TempDir(), "custom-copilot")
	t.Setenv("COPILOT_HOME", root)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "mcp-config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := scanStateFor(t, ScanConfigs(home), "github-copilot-cli")
	if c.Path != root || !c.Exists || !c.IsDirectory {
		t.Fatalf("github-copilot-cli = %+v, want directorio %q", c, root)
	}
}

func TestIsCopilotExtensionDir(t *testing.T) {
	for name, want := range map[string]bool{
		"github.copilot": true, "github.copilot-1.2.3": true, "github.copilot-chat-0.12.0": true,
		"github.copilot-nightly": true, "github.copilot_other": false,
		"github.other": false, "copilot": false, "": false,
	} {
		if got := IsCopilotExtensionDir(name); got != want {
			t.Errorf("IsCopilotExtensionDir(%q) = %v, want %v", name, got, want)
		}
	}
}
