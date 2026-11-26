package sdd

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

const testSDDSessionPreflightInitAnchor = "### SDD Init Guard (MANDATORY)"
const testSDDSessionPreflightEntryAnchor = "### SDD Entry Routing (MANDATORY)"

func assertFallbackSessionPreflight(t *testing.T, got string) {
	t.Helper()
	open, close := "<!-- axiom:sdd-session-preflight -->", "<!-- /axiom:sdd-session-preflight -->"
	if strings.Count(got, open) != 1 || strings.Count(got, close) != 1 {
		t.Fatal("installed prompt requires exactly one bounded preflight")
	}
	start, end := strings.Index(got, open), strings.Index(got, close)
	if end <= start || end >= strings.Index(got, "### SDD Init Guard (MANDATORY)") {
		t.Fatal("complete preflight must precede init")
	}
}

func TestSessionPreflightActualProjectionInventory(t *testing.T) {
	paths, err := fs.Glob(assets.FS, "*/sdd-orchestrator.md")
	if err != nil {
		t.Fatal(err)
	}
	covered := map[string]bool{}
	for _, agent := range []model.AgentID{model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI, model.AgentCursor, model.AgentGeminiCLI, model.AgentAntigravity, model.AgentQwenCode, model.AgentCodex, model.AgentOpenCode, model.AgentClaudeCode} {
		t.Run(string(agent), func(t *testing.T) {
			// OpenCode share a template. Claude's full projection is lazy;
			// its always-on bootstrap is a consumer.
			covered[sddOrchestratorAsset(agent)] = true
			content, err := composeOpenCodeOrchestratorPrompt(agent)
			if agent == model.AgentClaudeCode {
				content, err = renderClaudeSessionPreflight()
			}
			if err != nil {
				t.Fatal(err)
			}
			open, end, err := sddSessionPreflightMarkerRange(content)
			if err != nil || open < 0 {
				t.Fatalf("missing projection: %v", err)
			}
			block := content[open:end]
			for name, invalid := range map[string]string{
				"missing":         strings.Replace(content, block, "", 1),
				"duplicate":       block + "\n" + content,
				"orphan":          strings.Replace(content, sddSessionPreflightEnd, "", 1),
				"reversed":        strings.Replace(content, block, sddSessionPreflightEnd+"\n"+sddSessionPreflightMarker, 1),
				"misplaced":       strings.Replace(content, block, "", 1) + "\n" + block,
				"stale":           strings.Replace(content, "Interactive -> `interactive`", "Interactive -> `auto`", 1),
				"wrong transport": strings.Replace(content, block, sddSessionPreflightBlockWithTool("wrong-tool"), 1),
				"mixed endings":   content + "\r\n",
				"bare CR":         content + "\r",
			} {
				if err := validateRenderedSessionPreflight(invalid, agent); err == nil {
					t.Errorf("accepted %s", name)
				}
			}
			for _, valid := range []string{content, strings.ReplaceAll(content, "\n", "\r\n")} {
				if err := validateRenderedSessionPreflight(valid, agent); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	if len(paths) != len(covered) {
		t.Fatalf("actual templates %v; covered %v", paths, covered)
	}
	for _, path := range paths {
		if !covered[path] {
			t.Errorf("uncovered template %s", path)
		}
	}
}

func TestFallbackSessionPreflightCohort(t *testing.T) {
	for _, agent := range []model.AgentID{model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI, model.AgentCursor, model.AgentGeminiCLI, model.AgentAntigravity, model.AgentQwenCode, model.AgentCodex} {
		t.Run(string(agent), func(t *testing.T) {
			got := composeOrchestratorPrompt(agent)
			if agent == model.AgentCodex && !strings.Contains(got, "{{CODEX_PHASE_EFFORTS}}") {
				t.Error("Codex projection lost its phase-efforts placeholder")
			}
			for _, marker := range []string{sddSessionPreflightMarker, sddSessionPreflightEnd} {
				if strings.Count(got, marker) != 1 {
					t.Fatalf("expected exactly one %s", marker)
				}
			}
			end := strings.Index(got, sddSessionPreflightEnd)
			if end > strings.Index(got, testSDDSessionPreflightInitAnchor) {
				t.Fatal("preflight follows init")
			}
			block := got[strings.Index(got, sddSessionPreflightMarker):end]
			for _, want := range []string{"Interactive -> `interactive`", "Automatic -> `auto`"} {
				if !strings.Contains(block, want) {
					t.Errorf("preflight missing mapping %q", want)
				}
			}
		})
	}
}

func TestFallbackSessionPreflightProjectionBoundaries(t *testing.T) {
	anchor := testSDDSessionPreflightInitAnchor
	template := "prefix\n" + anchor + "\nsuffix"
	got, err := projectSDDSessionPreflightWithTool(template, anchor, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, content string
		valid         bool
	}{
		{"canonical", got, true},
		{"CRLF", strings.ReplaceAll(got, "\n", "\r\n"), true},
		{"missing", template, false},
		{"duplicate", sddSessionPreflightBlockWithTool("") + "\n" + got, false},
		{"after init", template + "\n" + sddSessionPreflightBlockWithTool(""), false},
		{"native transport", strings.Replace(got, sddSessionPreflightBlockWithTool(""), sddSessionPreflightBlock(), 1), false},
		{"wrong mapping", strings.ReplaceAll(got, "Interactive -> `interactive`", "Interactive -> `auto`"), false},
		{"legacy mapping", strings.Replace(got, "- Automatic -> `auto`", "- Automatic -> `auto`\n- Both -> `both`", 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateSDDSessionPreflightProjection(tc.content, anchor, ""); (err == nil) != tc.valid {
				t.Fatalf("validation = %v; valid = %v", err, tc.valid)
			}
		})
	}
	if again, err := projectSDDSessionPreflightWithTool(got, anchor, ""); err != nil || again != got {
		t.Fatalf("projection not idempotent: %v", err)
	}
}

func TestClaudeSessionPreflightProjectionStructure(t *testing.T) {
	const tool = "AskUserQuestion"
	anchor := testSDDSessionPreflightEntryAnchor
	template := "prefix\n" + anchor + "\n" + sddSessionPreflightInit + "\nsuffix"
	got, err := projectSDDSessionPreflightWithTool(template, anchor, tool)
	if err != nil {
		t.Fatal(err)
	}
	block := strings.ReplaceAll(sddSessionPreflightBlock(), "`question`", "`AskUserQuestion`")
	for _, tc := range []struct {
		name, content string
		valid         bool
	}{
		{"canonical", got, true},
		{"CRLF", strings.ReplaceAll(got, "\n", "\r\n"), true},
		{"missing", template, false},
		{"duplicate", block + "\n" + got, false},
		{"after init", template + "\n" + block, false},
		{"after routing", strings.Replace(template, anchor, anchor+"\n"+block, 1), false},
		{"missing close", strings.Replace(got, sddSessionPreflightEnd, "", 1), false},
		{"wrong tool", strings.ReplaceAll(got, "`AskUserQuestion`", "`question`"), false},
		{"legacy mapping", strings.Replace(got, "- Automatic -> `auto`", "- Automatic -> `auto`\n- Both -> `both`", 1), false},
		{"duplicate init", got + "\n" + sddSessionPreflightInit, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSDDSessionPreflightProjection(tc.content, anchor, tool)
			if (err == nil) != tc.valid {
				t.Fatalf("validation = %v; valid = %v", err, tc.valid)
			}
		})
	}
	if again, err := projectSDDSessionPreflightWithTool(got, anchor, tool); err != nil || again != got {
		t.Fatalf("projection not idempotent: %v", err)
	}
}

func TestOpenCodeSessionPreflightCompositionIsRuntimeScoped(t *testing.T) {
	for _, agent := range []model.AgentID{model.AgentOpenCode} {
		content, err := composeOpenCodeOrchestratorPrompt(agent)
		if err != nil {
			t.Fatalf("composeOpenCodeOrchestratorPrompt(%s) error = %v", agent, err)
		}
		if !strings.Contains(content, sddSessionPreflightMarker) {
			t.Fatalf("composeOpenCodeOrchestratorPrompt(%s) omitted canonical session preflight", agent)
		}
	}
	claude, err := composeOpenCodeOrchestratorPrompt(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("composeOpenCodeOrchestratorPrompt(claude) error = %v", err)
	}
	if strings.Contains(claude, sddSessionPreflightMarker) {
		t.Fatal("Claude composition unexpectedly received the OpenCode session preflight projection")
	}
}
func TestMigratePreservedSDDSessionPreflightReplacesOnlyOwnedBytes(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		for _, testCase := range []struct {
			name, prompt, want string
		}{
			{name: "unmarked", prompt: "PREFIX" + newline + "SUFFIX", want: "PREFIX" + newline + "SUFFIX" + newline + newline},
			{name: "legacy", prompt: "PREFIX" + newline + legacySDDSessionPreflightMarker + newline + "Both -> `both`" + newline + legacySDDSessionPreflightEnd + newline + "SUFFIX", want: "PREFIX" + newline},
			{name: "canonical", prompt: "PREFIX" + newline + sddSessionPreflightMarker + newline + "Both -> `both`" + newline + sddSessionPreflightEnd + newline + "SUFFIX", want: "PREFIX" + newline},
		} {
			t.Run(testCase.name+"/"+strings.ReplaceAll(newline, "\r", "cr"), func(t *testing.T) {
				got, err := migratePreservedSDDSessionPreflight(testCase.prompt)
				if err != nil {
					t.Fatal(err)
				}
				block := strings.ReplaceAll(sddSessionPreflightBlock(), "\n", newline)
				want := testCase.want + block
				if testCase.name != "unmarked" {
					want += newline + "SUFFIX"
				}
				if got != want || strings.Contains(got, "Both -> `both`") {
					t.Fatalf("migration = %q, want %q", got, want)
				}
			})
		}
	}
}

func TestSDDSessionPreflightProjectionCanonicalAndBounded(t *testing.T) {
	block := sddSessionPreflightBlock()
	for _, newline := range []string{"\n", "\r\n"} {
		rendered := strings.Join([]string{"before", testSDDSessionPreflightEntryAnchor, testSDDSessionPreflightInitAnchor, "after"}, newline)
		got, err := projectSDDSessionPreflight(rendered, testSDDSessionPreflightEntryAnchor)
		if err != nil {
			t.Fatalf("projectSDDSessionPreflight() error = %v", err)
		}
		want := strings.Join([]string{"before", strings.ReplaceAll(block, "\n", newline), testSDDSessionPreflightEntryAnchor, testSDDSessionPreflightInitAnchor, "after"}, newline)
		if got != want {
			t.Fatalf("projected preflight = %q, want %q", got, want)
		}
	}
	for _, rendered := range []string{"prefix " + testSDDSessionPreflightEntryAnchor + "\n" + testSDDSessionPreflightInitAnchor, testSDDSessionPreflightEntryAnchor + " suffix\n" + testSDDSessionPreflightInitAnchor} {
		if _, err := projectSDDSessionPreflight(rendered, testSDDSessionPreflightEntryAnchor); err == nil {
			t.Fatal("projectSDDSessionPreflight() accepted an inline anchor")
		}
	}
}

func TestMigratePreservedSDDSessionPreflightReplacesTheBlockOfThePreviousName(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		prompt := "PREFIX" + newline + legacyNamedSDDSessionPreflightMarker + newline + "old body" + newline + legacyNamedSDDSessionPreflightEnd + newline + "SUFFIX"
		got, err := migratePreservedSDDSessionPreflight(prompt)
		if err != nil {
			t.Fatal(err)
		}
		block := strings.ReplaceAll(sddSessionPreflightBlock(), "\n", newline)
		if want := "PREFIX" + newline + block + newline + "SUFFIX"; got != want {
			t.Fatalf("migration = %q, want %q", got, want)
		}
		if strings.Contains(got, "gentle-ai:") {
			t.Fatalf("legacy marker survived: %q", got)
		}
		if again, err := migratePreservedSDDSessionPreflight(got); err != nil || again != got {
			t.Fatalf("migration is not idempotent: %q, %v", again, err)
		}
		mixed := prompt + newline + sddSessionPreflightBlock()
		if err := validatePreservedSDDSessionPreflight(mixed); err == nil {
			t.Fatal("canonical and legacy-named pairs together must be rejected")
		}
	}
}
