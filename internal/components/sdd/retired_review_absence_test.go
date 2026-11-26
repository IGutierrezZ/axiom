package sdd

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func retiredReviewInstallEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())
}

// walkInstalledFiles returns the path and content of every file under root.
func walkInstalledFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(file)
		if readErr != nil {
			return readErr
		}
		files[file] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s) error = %v", root, err)
	}
	return files
}

// TestInjectLeavesResidualRDDFilesUntouched pins RF-ORQ-26: what an earlier
// release installed (review agents, the stop hook, the OpenCode review agents
// and the legacy review plugin) is neither modified nor removed, and the
// injection ends without error while keeping the user's own hooks.
func TestInjectLeavesResidualRDDFilesUntouched(t *testing.T) {
	t.Run("claude-code", func(t *testing.T) {
		retiredReviewInstallEnv(t)
		root := t.TempDir()
		adapter, err := agents.NewAdapter(model.AgentClaudeCode)
		if err != nil {
			t.Fatal(err)
		}
		bound := agents.BindWorkspace(adapter, root)
		dir := bound.SubAgentsDir(root)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		residual := map[string]string{}
		for _, name := range []string{"review-risk.md", "review-refuter.md", "review-mine.md"} {
			residual[filepath.Join(dir, name)] = "residual " + name + "\n"
		}
		for file, content := range residual {
			if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		settingsPath := bound.SettingsPath(root)
		if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
			t.Fatal(err)
		}
		const seeded = `{
  "hooks": {
    "Stop": [
      {"matcher": "", "hooks": [
        {"type": "command", "command": "axiom review stop-hook --agent claude-code", "timeout": 60},
        {"type": "command", "command": "echo mi-hook-de-usuario"}
      ]}
    ],
    "SessionStart": [
      {"matcher": "startup|resume|clear|compact", "hooks": [
        {"type": "command", "command": "gentle-ai review stop-hook --agent claude-code", "timeout": 30}
      ]}
    ]
  }
}`
		if err := os.WriteFile(settingsPath, []byte(seeded), 0o644); err != nil {
			t.Fatal(err)
		}

		if _, err := Inject(root, bound, model.SDDModeSingle, InjectOptions{WorkspaceDir: root}); err != nil {
			t.Fatalf("Inject() error = %v", err)
		}
		for file, want := range residual {
			got, err := os.ReadFile(file)
			if err != nil || string(got) != want {
				t.Errorf("%s was removed or modified (err = %v, content %q)", file, err, got)
			}
		}
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			t.Fatal(err)
		}
		var settings struct {
			Hooks map[string][]struct {
				Hooks []struct {
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"hooks"`
		}
		if err := json.Unmarshal(data, &settings); err != nil {
			t.Fatalf("settings.json is not valid JSON: %v", err)
		}
		var stop []string
		for _, entry := range settings.Hooks["Stop"] {
			for _, hook := range entry.Hooks {
				stop = append(stop, hook.Command)
			}
		}
		sort.Strings(stop)
		if len(stop) != 2 || stop[0] != "axiom review stop-hook --agent claude-code" || stop[1] != "echo mi-hook-de-usuario" {
			t.Errorf("the Stop hooks changed: %v", stop)
		}
		if !strings.Contains(string(data), "gentle-ai review stop-hook --agent claude-code") {
			t.Errorf("the residual SessionStart hook was removed:\n%s", data)
		}
	})

	t.Run("opencode", func(t *testing.T) {
		retiredReviewInstallEnv(t)
		root := t.TempDir()
		adapter, err := agents.NewAdapter(model.AgentOpenCode)
		if err != nil {
			t.Fatal(err)
		}
		bound := agents.BindWorkspace(adapter, root)
		settingsPath := bound.SettingsPath(root)
		if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
			t.Fatal(err)
		}
		seed := map[string]any{"agent": map[string]any{
			"review-risk": map[string]any{
				"mode": "subagent", "prompt": "You are R1 Risk.", "permission": map[string]any{"*": "deny"},
			},
			"axiom-orchestrator": map[string]any{
				"mode": "primary", "prompt": "orq",
				"permission": map[string]any{"task": map[string]any{"review-risk": "allow", "sdd-apply": "allow"}},
			},
		}}
		raw, err := json.Marshal(seed)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(settingsPath, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		pluginsDir := filepath.Join(filepath.Dir(settingsPath), "plugins")
		if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
			t.Fatal(err)
		}
		legacyPlugin := filepath.Join(pluginsDir, "review-result-artifacts.ts")
		if err := os.WriteFile(legacyPlugin, []byte("legacy review plugin\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		if _, err := Inject(root, bound, model.SDDModeMulti, InjectOptions{WorkspaceDir: root}); err != nil {
			t.Fatalf("Inject() error = %v", err)
		}
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			t.Fatal(err)
		}
		var settings map[string]any
		if err := json.Unmarshal(data, &settings); err != nil {
			t.Fatalf("opencode.json is not valid JSON: %v", err)
		}
		agentsMap, _ := settings["agent"].(map[string]any)
		if _, has := agentsMap["review-risk"]; !has {
			t.Error("the residual review-risk agent was removed")
		}
		if _, has := agentsMap["sdd-apply"]; !has {
			t.Error("the SDD agents were not written")
		}
		for _, name := range []string{"review-readability", "review-reliability", "review-resilience", "review-refuter", "review-validator"} {
			if _, has := agentsMap[name]; has {
				t.Errorf("Inject wrote the retired review agent %s", name)
			}
		}
		if kept, err := os.ReadFile(legacyPlugin); err != nil || string(kept) != "legacy review plugin\n" {
			t.Errorf("the legacy review plugin was removed or modified (err = %v, content %q)", err, kept)
		}
		if _, err := os.Stat(filepath.Join(pluginsDir, "opencode-review-transport.ts")); !os.IsNotExist(err) {
			t.Errorf("the retired review transport plugin was written (err = %v)", err)
		}
	})
}

// TestOpenCodeOverlaysRenderReadOnlyJudgmentDayRoles keeps the OpenCode judges
// on the native role-only judgment contract with read-only tools.
func TestOpenCodeOverlaysRenderReadOnlyJudgmentDayRoles(t *testing.T) {
	for _, file := range []string{"opencode/sdd-overlay-single.json", "opencode/sdd-overlay-multi.json"} {
		t.Run(file, func(t *testing.T) {
			var root map[string]any
			if err := json.Unmarshal([]byte(assets.MustRead(file)), &root); err != nil {
				t.Fatal(err)
			}
			agentsMap := root["agent"].(map[string]any)
			expandOpenCodeJudgmentDayAgents(agentsMap)
			for _, name := range []string{"jd-judge-a", "jd-judge-b"} {
				agent := agentsMap[name].(map[string]any)
				prompt := agent["prompt"].(string)
				if prompt != judgmentDayReviewerContract() {
					t.Errorf("%s %s does not use the native role-only judgment contract", file, name)
				}
				tools := agent["tools"].(map[string]any)
				want := map[string]bool{"*": false, "read": true, "write": false, "edit": false, "bash": false, "task": false}
				for tool, expected := range want {
					if got, ok := tools[tool].(bool); !ok || got != expected {
						t.Errorf("%s %s tool %s = %v, want %v", file, name, tool, tools[tool], expected)
					}
				}
			}
		})
	}
}

// TestInjectIgnoresPersistedAssignmentsOfRetiredReviewRoles pins that a Claude
// selection still carrying the retired lens roles injects without error and
// never recreates a review agent.
func TestInjectIgnoresPersistedAssignmentsOfRetiredReviewRoles(t *testing.T) {
	retiredReviewInstallEnv(t)
	root := t.TempDir()
	adapter, err := agents.NewAdapter(model.AgentClaudeCode)
	if err != nil {
		t.Fatal(err)
	}
	bound := agents.BindWorkspace(adapter, root)
	options := InjectOptions{
		WorkspaceDir: root,
		ClaudeModelAssignments: map[string]model.ClaudeModelAlias{
			"sdd-apply": model.ClaudeModelSonnet,
			"risk":      model.ClaudeModelHaiku,
			"refuter":   model.ClaudeModelHaiku,
		},
	}
	if _, err := Inject(root, bound, model.SDDModeSingle, options); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	for file := range walkInstalledFiles(t, root) {
		if strings.HasPrefix(filepath.Base(file), "review-") {
			t.Errorf("the retired review agent %s was written", file)
		}
	}
}
