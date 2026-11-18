package sdd

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// #3817: the SDD orchestrator contract is maintained as twelve hand-written
// near-duplicates. Measured across them, 19 of 21 shared subsections have
// drifted -- Delegation Rules alone has eleven variants across eleven runtimes
// -- so reconciling those is a decision per section, not a refactor.
//
// These five are the subsections that had NOT drifted semantically: three are
// byte-identical across every runtime that carries them, and the other two
// differ only cosmetically (a ```text fence, and "the user" for "user"). They
// move to one shared asset so the mechanism exists and so this set cannot drift
// again. Each runtime keeps its own heading line, which is why codex may hold a
// section at ## while the others hold it at ###.

// The fork renamed the binary and flattened the verb, so the shipped assets
// spell the status invocation `axiom sdd status`. Upstream still ships
// `gentle-ai sdd-status`, and absorbing it must not reopen this contract:
// either spelling satisfies it.
func containsStatusContinuationInvocation(text string) bool {
	return strings.Contains(text, "axiom sdd status [change] --cwd <repo> --json --instructions") ||
		strings.Contains(text, "gentle-ai sdd-status [change] --cwd <repo> --json --instructions")
}

// Text contracts prove shipped instructions, not execution by any agent host.
func assertStatusContinuationContract(t *testing.T, content string) {
	t.Helper()
	if !containsStatusContinuationInvocation(content) {
		t.Errorf("missing status/continuation contract invocation (neither `axiom sdd status` nor `gentle-ai sdd-status` spelling)")
	}
	for _, want := range []string{
		"every declared artifact store, including Engram", "native v2",
		"Inspection needs no execution preflight", "No recommendation is executed during inspection",
		"Only explicit authorized continuation", "current human scope covers the selected change-directory marker",
		"Read-only or excluded-marker scope forbids this mutating call",
		"Preparation grants no source roots or attempts", "actionContext",
		"nextRecommended", "blockedReasons", "non-authoritative",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("missing status/continuation contract %q", want)
		}
	}
	for _, forbidden := range []string{
		"sdd-continue [change] --cwd <repo>` or", "manual status schema",
		"do NOT invoke the native dispatcher", "resolve status entirely from Engram",
	} {
		if strings.Contains(content, forbidden) {
			t.Errorf("conflicting status/continuation instruction %q", forbidden)
		}
	}
}

func TestRegisteredAgentsRenderStatusContinuationContract(t *testing.T) {
	registry, err := agents.NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	ids := registry.SupportedAgents()
	if len(ids) != 17 {
		t.Fatalf("registered cohort changed: %v", ids)
	}
	templates := map[string]bool{}
	for _, agent := range ids {
		templates[sddOrchestratorAsset(agent)] = true
		t.Run(string(agent), func(t *testing.T) {
			content := renderSDDOrchestratorAsset(agent)
			if agent == model.AgentClaudeCode {
				if !strings.Contains(content, "~/.claude/skills/_shared/sdd-orchestrator-workflow.md") {
					t.Fatal("Claude bootstrap lost its lazy workflow reference")
				}
				lazy, err := renderClaudeSessionPreflight()
				if err != nil {
					t.Fatal(err)
				}
				content += lazy
			}
			assertStatusContinuationContract(t, content)
		})
	}
	if len(templates) != 12 {
		t.Fatalf("effective template coverage = %d, want 12", len(templates))
	}
	t.Run("claude-lazy-workflow", func(t *testing.T) {
		content, err := renderClaudeSessionPreflight()
		if err != nil {
			t.Fatal(err)
		}
		assertStatusContinuationContract(t, content)
	})
}

var sharedOrchestratorSectionNames = []string{
	"Native SDD Dispatcher Guard",
	"Language Domain Contract",
	"Dependency Graph",
	"Recovery Rule",
	// #4296: the RDD-aware, risk-gated delegated-verification rule. Every
	// runtime that names a concrete delegation mechanism in its own
	// Delegation Rules body (claude, codex, opencode, cursor, gemini,
	// antigravity, generic, hermes, kimi, kiro, qwen) carries the full form
	// under "Delegated Verification Gate (MANDATORY)". Windsurf ("Windsurf
	// has no subagents" in its own Delegation Rules body) is the only
	// runtime with no delegation mechanism, so it carries the reduced form
	// under its own distinct heading, "Delegated Verification Gate (Reduced
	// Form)".
	"Delegated Verification Gate (MANDATORY)",
	"Delegated Verification Gate (Reduced Form)",
	// INC-21 (design.md S4.7, H-2): the per-change kickoff questionnaire and
	// the four block-review gates, plus their normative boundary against
	// RDD (design.md S1.3). Registered here, in Phase 22, ahead of Phase
	// 23 inserting the {{GENTLE_AI_SDD_SECTION:...}} marker into the
	// twelve per-agent orchestrator assets: TestEveryRuntimeRendersTheSharedSections'
	// own "continue" guard keeps this registration a no-op for every
	// runtime until that marker exists.
	"SDD Change Kickoff and Block Gates",
	// sdd-revision-independiente (REQ-28.1 a REQ-28.3, REQ-28.10, REQ-28.14): the optional
	// independent review (per-role verify, judgment and coverage review) and its preference.
	"SDD Independent Review",
	// single-pr (C-02-c): the one canonical Delivery Strategy body (delivery is
	// never asked: `single-pr | chained`, aliases read only, one non-blocking size
	// warning). Every runtime references it by placeholder, so no inline copy of
	// the retired four-way strategy can come back.
	"Delivery Strategy",
}

// TestSDDChangeKickoffAndBlockGatesSectionIsDeterministic is task 22.3's own
// case: the new shared section renders the same content across repeated
// reads. sharedOrchestratorSection is a pure lookup over an embedded asset,
// so determinism holds by construction; this test makes that property
// observable instead of merely assumed.
func TestSDDChangeKickoffAndBlockGatesSectionIsDeterministic(t *testing.T) {
	const name = "SDD Change Kickoff and Block Gates"
	first := sharedOrchestratorSection(name)
	if strings.TrimSpace(first) == "" {
		t.Fatalf("shared asset carries no body for %q", name)
	}
	for i := 0; i < 5; i++ {
		if again := sharedOrchestratorSection(name); again != first {
			t.Fatalf("shared section %q is not deterministic across repeated reads (iteration %d):\nfirst:\n%s\nagain:\n%s", name, i, first, again)
		}
	}
}

// TestSharedOrchestratorSectionsHaveOneSource pins that each shared section
// body lives in the shared asset and NOT in the per-runtime orchestrators.
//
// It opens the runtime assets. An earlier version only checked the shared
// asset against a substring extracted from that same shared asset, which is a
// tautology: it could not fail, while its name promised that no duplicated
// body survives in the eleven per-runtime files.
func TestSharedOrchestratorSectionsHaveOneSource(t *testing.T) {
	runtimeAssets := map[string]string{}
	err := fs.WalkDir(assets.FS, ".", func(assetPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && path.Base(assetPath) == "sdd-orchestrator.md" {
			runtimeAssets[assetPath] = assets.MustRead(assetPath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk orchestrator assets: %v", err)
	}
	if len(runtimeAssets) == 0 {
		t.Fatal("no runtime orchestrator assets were found")
	}

	for _, name := range sharedOrchestratorSectionNames {
		body := sharedOrchestratorSection(name)
		if strings.TrimSpace(body) == "" {
			t.Fatalf("shared asset carries no body for %q", name)
		}
		for assetPath, content := range runtimeAssets {
			if strings.Contains(content, body) {
				t.Errorf("%s still carries the %q body inline; it must carry the placeholder instead", assetPath, name)
			}
		}
	}
}

// TestEveryRuntimeRendersTheSharedSections pins that the substitution actually
// reaches the rendered prompt: deleting duplication must not delete content.
func TestEveryRuntimeRendersTheSharedSections(t *testing.T) {
	for _, agent := range []model.AgentID{
		model.AgentOpenCode, model.AgentCursor, model.AgentGeminiCLI, model.AgentQwenCode,
		model.AgentHermes, model.AgentKimi, model.AgentWindsurf, model.AgentCodex,
		model.AgentClaudeCode, model.AgentKiroIDE, model.AgentAntigravity, model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI,
	} {
		rendered := renderSDDOrchestratorAsset(agent)
		for _, name := range sharedOrchestratorSectionNames {
			if !strings.Contains(rendered, name) {
				continue // a runtime that never carried this section keeps not carrying it
			}
			body := sharedOrchestratorSection(name)
			first := strings.SplitN(strings.TrimSpace(body), "\n", 2)[0]
			if !strings.Contains(rendered, first) {
				t.Errorf("%s rendered %q without its shared body", agent, name)
			}
		}
	}
}

// TestDelegatedVerificationGateCarriesNoRDDProse pins the RDD suspension
// (C-01-c): the canonical bodies (full and reduced forms) and every rendered
// orchestrator state the diff-inferred verification tier without mentioning RDD,
// the review switch, or the native risk assessment.
func TestDelegatedVerificationGateCarriesNoRDDProse(t *testing.T) {
	const sddBoundary = "For SDD work, run the applicable functional checks and configured TDD, report actual results, and follow the SDD phase instructions."
	forbidden := []string{"RDD", "receipt-driven", "axiom review", "review assess", "native review", "native risk", "consent"}

	for _, name := range []string{
		"Delegated Verification Gate (MANDATORY)",
		"Delegated Verification Gate (Reduced Form)",
	} {
		body := sharedOrchestratorSection(name)
		if !strings.Contains(body, sddBoundary) || !strings.Contains(body, "The parent spot check") {
			t.Errorf("shared section %q canonical body lacks the SDD boundary or the parent spot check", name)
		}
		for _, bad := range forbidden {
			if strings.Contains(body, bad) {
				t.Errorf("shared section %q still mentions %q", name, bad)
			}
		}
	}

	for _, entry := range catalog.AllAgents() {
		agent := entry.ID
		rendered := renderSDDOrchestratorAsset(agent)
		if !strings.Contains(rendered, "Delegated Verification Gate") {
			continue // a runtime that never carries either form keeps not carrying it
		}
		if !strings.Contains(rendered, sddBoundary) {
			t.Errorf("%s renders a Delegated Verification Gate section without the SDD exclusion", agent)
		}
	}
}

// TestNoRawSharedSectionPlaceholderSurvivesRendering pins that no placeholder
// reaches a rendered prompt.
func TestNoRawSharedSectionPlaceholderSurvivesRendering(t *testing.T) {
	for _, agent := range []model.AgentID{
		model.AgentOpenCode, model.AgentCursor, model.AgentGeminiCLI, model.AgentQwenCode,
		model.AgentHermes, model.AgentKimi, model.AgentWindsurf, model.AgentCodex,
		model.AgentKiroIDE, model.AgentAntigravity, model.AgentClaudeCode, model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI,
	} {
		if rendered := renderSDDOrchestratorAsset(agent); strings.Contains(rendered, "{{GENTLE_AI_SDD_SECTION:") {
			t.Errorf("%s kept a raw shared-section placeholder", agent)
		}
	}
}
