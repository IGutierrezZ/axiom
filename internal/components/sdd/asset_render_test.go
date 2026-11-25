package sdd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func TestPiRenderedOrchestratorUsesOnlyTheClosedChoiceRoute(t *testing.T) {
	content := renderSDDOrchestratorAsset(model.AgentPi)
	for _, clause := range []string{
		"ask_user_choice",
		"2-4 ordered-option domain",
		"envelope-owned canonical option token as value",
		"returns exactly one value; map it to the exact envelope-owned choice once",
		"ask_user_question is the external open/free-text questionnaire and must not be used for a closed domain",
		"open/free-text questionnaires may use ask_user_question",
	} {
		if !strings.Contains(content, clause) {
			t.Errorf("Pi closed-choice route missing %q", clause)
		}
	}
	if strings.Contains(renderSDDOrchestratorAsset(model.AgentOpenCode), "ask_user_choice") {
		t.Fatal("generic fallback-only runtime received the Pi-only closed choice route")
	}
}

func TestPreservedSharedOrchestratorSubstitutesRuntimeAgentIdentity(t *testing.T) {
	t.Parallel()

	preserved := strings.Join([]string{
		"Bind this to the dedicated `sdd-orchestrator` agent only.",
		runtimeAgentIDPlaceholder,
	}, "\n")
	rendered := renderPreservedOpenCodeOrchestratorPrompt(
		preserved,
		model.AgentOpenCode,
	)
	if !strings.Contains(rendered, "Bind this to the dedicated `axiom-orchestrator` agent only.") && !strings.Contains(rendered, "Bind this to the dedicated `gentle-orchestrator` agent only.") {
		t.Fatalf("preserved prompt lost its migration:\n%s", rendered)
	}
	if strings.Contains(rendered, runtimeAgentIDPlaceholder) {
		t.Fatal("preserved prompt retained runtime agent placeholder")
	}
	if !strings.Contains(rendered, string(model.AgentOpenCode)) {
		t.Fatalf("preserved prompt missing runtime agent identity:\n%s", rendered)
	}
}

// The retired WorkRun commands no longer exist, so a preserved prompt that
// still mentions them must pass through migration untouched instead of being
// rewritten into a better-formed invocation of a deleted command.
func TestPreservedSharedOrchestratorLeavesRetiredWorkCommandsUntouched(t *testing.T) {
	t.Parallel()

	retired := []string{
		"gentle-ai work-capabilities --cwd <repo> --contract gentle-ai.work-capabilities/v2 --json",
		"gentle-ai work-start --cwd <repo> --contract gentle-ai.work-start/v1 --json",
	}
	rendered := renderPreservedOpenCodeOrchestratorPrompt(
		strings.Join(retired, "\n"),
		model.AgentOpenCode,
	)
	for _, command := range retired {
		if !strings.Contains(rendered, command) {
			t.Fatalf("preserved prompt rewrote retired command %q:\n%s", command, rendered)
		}
	}
	if strings.Contains(rendered, "--agent "+string(model.AgentOpenCode)) {
		t.Fatalf("preserved prompt injected an adapter identity into a retired command:\n%s", rendered)
	}
}

func TestJudgmentDayReviewersUseNativeResultSchema(t *testing.T) {
	for name, content := range map[string]string{
		"rendered contract": judgmentDayReviewerContract(),
		"skill reference":   assets.MustRead("skills/judgment-day/references/prompts-and-formats.md"),
	} {
		for _, want := range []string{nativeReviewerResultSchema, "Never emit", "skill_resolution", "unknown field", "orchestration metadata outside the native result JSON", `{"findings":[],"evidence":["what was inspected"]}`} {
			if !strings.Contains(content, want) {
				t.Errorf("%s missing %q", name, want)
			}
		}
	}
}

func TestOpenCodeAndClaudeArchiveInstructionsDoNotGateOnReviewAuthority(t *testing.T) {
	for _, path := range []string{"opencode/commands/sdd-archive.md", "claude/commands/sdd-archive.md"} {
		t.Run(path, func(t *testing.T) {
			content := assets.MustRead(path)
			for _, required := range []string{
				"an explicit archive request may close unfinished work without a verification certificate",
			} {
				if !strings.Contains(content, required) {
					t.Errorf("%s missing archive non-gate rule %q", path, required)
				}
			}
			for _, forbidden := range []string{
				"reviewGate.result: allow",
				"Only a PRESENT `reviewGate`",
				"native receipt and task completion gates",
				"exact transaction/ledger/receipt/gate-context references",
				"RDD",
			} {
				if strings.Contains(content, forbidden) {
					t.Errorf("%s retains obsolete archive review gate %q", path, forbidden)
				}
			}
		})
	}
}

func TestSDDPhaseCommandsNeverInviteReview(t *testing.T) {
	for _, path := range []string{
		"opencode/commands/sdd-apply.md", "opencode/commands/sdd-verify.md", "opencode/commands/sdd-archive.md",
		"claude/commands/sdd-apply.md", "claude/commands/sdd-verify.md", "claude/commands/sdd-archive.md",
	} {
		content := assets.MustRead(path)
		for _, forbidden := range []string{"reviewOffer", "gentle-ai review start", "may present and run"} {
			if strings.Contains(content, forbidden) {
				t.Errorf("%s retains review invitation %q", path, forbidden)
			}
		}
	}
}

// TestOpenCodeOverlaysDefineNoReviewAgents pins REQ-46.4: the OpenCode
// overlays declare no `review-*` agent and no delegation permission to one,
// while Judgment Day and the SDD agents stay.
func TestOpenCodeOverlaysDefineNoReviewAgents(t *testing.T) {
	for _, path := range []string{"opencode/sdd-overlay-single.json", "opencode/sdd-overlay-multi.json"} {
		t.Run(path, func(t *testing.T) {
			var root map[string]any
			if err := json.Unmarshal([]byte(assets.MustRead(path)), &root); err != nil {
				t.Fatalf("unmarshal %s: %v", path, err)
			}
			agentsMap := root["agent"].(map[string]any)
			for name, raw := range agentsMap {
				if strings.HasPrefix(name, "review-") {
					t.Errorf("%s defines the review agent %q", path, name)
				}
				definition, _ := raw.(map[string]any)
				permission, _ := definition["permission"].(map[string]any)
				task, _ := permission["task"].(map[string]any)
				if replacement, ok := task["__replace__"].(map[string]any); ok {
					task = replacement
				}
				for target := range task {
					if strings.HasPrefix(target, "review-") {
						t.Errorf("%s: %s may delegate to the review agent %q", path, name, target)
					}
				}
			}
			for _, required := range []string{"jd-judge-a", "jd-judge-b", "jd-fix-agent", "sdd-apply"} {
				if _, ok := agentsMap[required]; !ok {
					t.Errorf("%s lost the agent %q", path, required)
				}
			}
		})
	}
}
