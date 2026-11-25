package sdd

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// retiredReviewProse lists the phrases that identify the retired review by
// receipts in a rendered orchestrator (REQ-46.3).
var retiredReviewProse = []string{
	"receipt-driven",
	"Review Execution Contract",
	"Provider Defect Handoff",
	"Native Checking Contract",
	"review-integration.consent",
	"declined_this_candidate",
	"axiom review",
	"RDD",
	"native review",
	"review-refuter",
	"rdd-defect-workflow",
}

// TestRenderedOrchestratorsCarryNoRDDProse pins REQ-46.3: every orchestrator
// renders free of review-by-receipt prose for every runtime, and OpenCode keeps
// the generic native question route.
func TestRenderedOrchestratorsCarryNoRDDProse(t *testing.T) {
	for _, entry := range catalog.AllAgents() {
		agent := entry.ID
		t.Run(string(agent), func(t *testing.T) {
			rendered := renderSDDOrchestratorAsset(agent)
			if agent == model.AgentOpenCode {
				var err error
				if rendered, err = composeOpenCodeOrchestratorPrompt(agent); err != nil {
					t.Fatalf("composeOpenCodeOrchestratorPrompt(%s) error = %v", agent, err)
				}
			}
			for _, forbidden := range retiredReviewProse {
				if strings.Contains(rendered, forbidden) {
					t.Errorf("%s orchestrator still renders retired review prose %q", agent, forbidden)
				}
			}
			if strings.Contains(rendered, "{{GENTLE_AI_") {
				t.Errorf("%s orchestrator kept a raw placeholder", agent)
			}
			if agent == model.AgentOpenCode && !strings.Contains(rendered, "- Native route: The classified native question UI is `question`.") {
				t.Errorf("%s lost the generic native question route", agent)
			}
		})
	}
}

// TestApplyCommandsCarryNoReviewLifecycle pins that the apply commands carry
// neither a raw placeholder nor the acknowledgement paragraph, and keep the
// Judgment Day sentence.
func TestApplyCommandsCarryNoReviewLifecycle(t *testing.T) {
	for file, agent := range map[string]model.AgentID{
		"claude/commands/sdd-apply.md":   model.AgentClaudeCode,
		"opencode/commands/sdd-apply.md": model.AgentOpenCode,
	} {
		rendered := renderManagedAsset(agent, file)
		for _, forbidden := range []string{"{{GENTLE_AI_", "Authority-First Terminal Procedure", "acknowledgement continuation", "burns authority", "review lineage"} {
			if strings.Contains(rendered, forbidden) {
				t.Errorf("%s still renders review lifecycle prose %q", file, forbidden)
			}
		}
	}
	if rendered := renderManagedAsset(model.AgentClaudeCode, "claude/commands/sdd-apply.md"); !strings.Contains(rendered, "Never auto-launch Judgment Day.") {
		t.Error("claude apply command lost the Judgment Day sentence")
	}
}

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

var retiredInstallProse = regexp.MustCompile(`(?i)receipt-driven|Review Execution Contract|Provider Defect Handoff|review stop-hook`)

// TestInjectWorkspaceInstallsNoRDD pins REQ-46.4: installing SDD in a clean
// project writes no `review-*` agent, registers no review stop hook and leaves
// no review-by-receipt prose in any file, for every runtime.
func TestInjectWorkspaceInstallsNoRDD(t *testing.T) {
	for _, entry := range catalog.AllAgents() {
		agent := entry.ID
		modes := []model.SDDModeID{model.SDDModeSingle}
		if agent == model.AgentOpenCode {
			modes = append(modes, model.SDDModeMulti)
		}
		for _, mode := range modes {
			t.Run(string(agent)+"/"+string(mode), func(t *testing.T) {
				retiredReviewInstallEnv(t)
				root := t.TempDir()
				adapter, err := agents.NewAdapter(agent)
				if err != nil {
					t.Fatal(err)
				}
				bound := agents.BindWorkspace(adapter, root)
				if _, err := Inject(root, bound, mode, InjectOptions{WorkspaceDir: root}); err != nil {
					t.Fatalf("Inject(%s, %s) error = %v", agent, mode, err)
				}

				written := walkInstalledFiles(t, root)
				if len(written) == 0 {
					t.Skipf("%s writes nothing in the project", agent)
				}
				for file, content := range written {
					if strings.HasPrefix(filepath.Base(file), "review-") {
						t.Errorf("%s: the review file %s was written", agent, file)
					}
					if strings.Contains(content, "opencode-review-transport") {
						t.Errorf("%s: %s names the retired review transport plugin", agent, file)
					}
					if match := retiredInstallProse.FindString(content); match != "" {
						t.Errorf("%s: %s carries retired review prose %q", agent, file, match)
					}
				}
			})
		}
	}
}

// TestInjectLeavesResidualRDDFilesUntouched pins REQ-46.4: what an earlier
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

// TestJudgmentDayAgentsRenderReadOnlyRoleContracts keeps the Judgment Day
// judges, in every runtime that installs them, as read-only reviewers that
// state their candidate-causal contract and never claim a lens envelope.
func TestJudgmentDayAgentsRenderReadOnlyRoleContracts(t *testing.T) {
	var judges []string
	err := fs.WalkDir(assets.FS, ".", func(file string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if strings.HasPrefix(path.Base(file), "jd-judge-") && strings.Contains(file, "/agents/") {
			judges = append(judges, file)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded assets: %v", err)
	}
	if len(judges) == 0 {
		t.Fatal("no judgment-day judge assets found; the guard would pass vacuously")
	}
	sort.Strings(judges)
	for _, judge := range judges {
		t.Run(judge, func(t *testing.T) {
			content := renderManagedAsset(agentForAssetPath(t, judge), judge)
			assertTextContainsClauses(t, judge, content, []string{"candidate", "BLOCKER", "CRITICAL", "causal", "proof"})
			if !strings.Contains(content, "read-only") && !strings.Contains(content, "Never edit") {
				t.Errorf("%s does not state its non-mutating role", judge)
			}
			if strings.Contains(content, `"subject_hash"`) || strings.Contains(content, `"inspection"`) {
				t.Errorf("%s claims a lens envelope; judgment-day judge results carry neither field", judge)
			}
			for _, forbidden := range []string{"axiom review", "capture-result"} {
				if strings.Contains(content, forbidden) {
					t.Errorf("%s names the retired review command %q", judge, forbidden)
				}
			}
		})
	}
	for _, kiro := range []string{"kiro/agents/jd-judge-a.md", "kiro/agents/jd-judge-b.md"} {
		if frontmatter := markdownFrontmatter(t, kiro); !strings.Contains(frontmatter, `tools: ["read"]`) {
			t.Errorf("%s is not read-only:\n%s", kiro, frontmatter)
		}
	}
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
