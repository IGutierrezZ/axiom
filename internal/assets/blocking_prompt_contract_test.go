package assets

import (
	"fmt"
	"strings"
	"testing"
)

type blockingPromptRoute struct {
	nativeTool string
}

// generic is the provider-neutral source that every agent-specific block must project.
const canonicalOrchestratorPath = "generic/sdd-orchestrator.md"

var blockingPromptRoutes = map[string]blockingPromptRoute{
	"antigravity/sdd-orchestrator.md": {},
	"claude/sdd-orchestrator.md":      {nativeTool: "`AskUserQuestion`"},
	"codex/sdd-orchestrator.md":       {},
	"cursor/sdd-orchestrator.md":      {},
	"gemini/sdd-orchestrator.md":      {},
	"generic/sdd-orchestrator.md":     {},
	"kiro/sdd-orchestrator.md":        {},
	"opencode/sdd-orchestrator.md":    {nativeTool: "`question`"},
	"qwen/sdd-orchestrator.md":        {},
	"windsurf/sdd-orchestrator.md":    {},
}

func TestCoordinatorOrchestratorsCarryLosslessBlockingPromptRule(t *testing.T) {
	allPaths := allSDDOrchestratorAssetPaths(t)
	if len(allPaths) != len(blockingPromptRoutes) {
		t.Fatalf("discovered %d orchestrator variants, but %d have an explicit blocking-prompt route; classify every variant",
			len(allPaths), len(blockingPromptRoutes))
	}

	for _, path := range allPaths {
		route, classified := blockingPromptRoutes[path]
		if !classified {
			t.Fatalf("unclassified orchestrator variant %q; native/fallback routing must fail closed", path)
		}

		t.Run(path, func(t *testing.T) {
			contract := blockingPromptContractSection(t, path)
			for _, required := range []string{
				"complete user-facing choice envelope",
				"why input is required",
				"every group and question in original order",
				"including every group header",
				"every option label and description",
				"exact allowed-answer domain",
				"Never summarize, abbreviate, reorder, relabel, merge, or omit choices",
				"Never silently split an atomic business choice",
				"COMPLETE choice envelope as a plain chat or terminal response",
				"unavailable, denied, the runtime is noninteractive",
				"question-count, option-count, or text-length limits",
				"Then STOP",
				"Do not choose, default, infer, launch dependent work, or continue",
				"Accept an answer only when each response belongs to the exact allowed-answer domain",
				"free text or multi-select only when the original prompt allowed it",
				"request for information, not a candidate answer",
				"answer it directly from the envelope already held",
				"re-present the complete choice envelope and keep waiting",
				"invalid or ambiguous",
				"same blocked actor exactly once",
			} {
				if !strings.Contains(contract, required) {
					t.Errorf("lossless blocking-prompt contract missing %q", required)
				}
			}

			if route.nativeTool == "" {
				const fallbackOnly = "This variant has no classified native question UI for this contract; always use the plain chat or terminal fallback below."
				if !strings.Contains(contract, fallbackOnly) {
					t.Errorf("fallback-only variant missing explicit route %q", fallbackOnly)
				}
				return
			}

			for _, required := range []string{
				fmt.Sprintf("The classified native question UI is %s.", route.nativeTool),
				"Use it only when it is available in the current interactive runtime",
				"exactly representable in one grouped interaction",
			} {
				if !strings.Contains(contract, required) {
					t.Errorf("native route missing %q", required)
				}
			}
		})
	}
}

// TestCoordinatorOrchestratorsCarryClosedSingleSelectDomainContract guards
// issue #3070: the closed-domain rule lives inside the existing Native route
// and Answer validation clauses. The ordinal alias domain must be EXPLICIT
// (not "for example") per Matere413's review on 2026-08-14. The issue's
// reproduction (`la 1`) must be present.
func TestCoordinatorOrchestratorsCarryClosedSingleSelectDomainContract(t *testing.T) {
	for _, path := range allSDDOrchestratorAssetPaths(t) {
		t.Run(path, func(t *testing.T) {
			contract := blockingPromptContractSection(t, path)
			for _, required := range []string{
				"closed domain of a single-select envelope",
				"EXACTLY ONE presented option",
				"reject zero matches and reject multiple matches",
				"canonical internal token once",
				"Accepted ordinal aliases, for each presented option index N",
				"the bare numeral `N`",
				"`la N`",
				"`opción N`",
				"`first` is additionally accepted for index 1",
			} {
				if !strings.Contains(contract, required) {
					t.Errorf("%s missing closed-domain amendment %q", path, required)
				}
			}
			// Matere413: ordinal alias domain must be explicit, not "for example".
			if strings.Contains(contract, "Ordinal aliases") && strings.Contains(contract, "for example") {
				idx := strings.Index(contract, "Ordinal aliases")
				end := strings.Index(contract[idx:], "\n")
				if end < 0 {
					end = len(contract)
				} else {
					end += idx
				}
				if strings.Contains(contract[idx:end], "for example") {
					t.Errorf("%s uses 'for example' for ordinal aliases (Matere413: must be explicit list)", path)
				}
			}
		})
	}
}

func TestBlockingPromptFallbackCoversWindsurfToolResults(t *testing.T) {
	content := mustReadResolved("windsurf/sdd-orchestrator.md")
	if !strings.Contains(content, "There are no sub-agents") {
		t.Fatal("Windsurf must still identify its solo-agent execution model")
	}
	contract := blockingPromptContractSection(t, "windsurf/sdd-orchestrator.md")
	for _, required := range []string{
		"sub-agent or tool",
		"always use the plain chat or terminal fallback",
		"Then STOP",
	} {
		if !strings.Contains(contract, required) {
			t.Errorf("Windsurf tool-result fallback missing %q", required)
		}
	}
}

func TestNativeBlockingPromptRulesRetainInteractiveUIWithFailClosedFallback(t *testing.T) {
	tests := []struct {
		name string
		path string
		tool string
	}{
		{name: "Claude", path: "claude/sdd-orchestrator.md", tool: "`AskUserQuestion`"},
		{name: "OpenCode", path: "opencode/sdd-orchestrator.md", tool: "`question`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contract := blockingPromptContractSection(t, tt.path)
			if !strings.Contains(contract, "The classified native question UI is "+tt.tool+".") {
				t.Fatalf("%s does not retain its native interactive question UI", tt.path)
			}
			for _, scenario := range []string{
				"unavailable",
				"denied",
				"the runtime is noninteractive",
				"question-count, option-count, or text-length limits",
			} {
				if !strings.Contains(contract, scenario) {
					t.Errorf("%s has no complete fallback for %s", tt.path, scenario)
				}
			}
		})
	}
}

// TestCoordinatorOrchestratorsCarryNoProviderDefectHandoff pins the RDD suspension
// (C-01-c): the provider-defect handoff only served the RDD consent envelope, so
// no orchestrator carries its section, its header, or its answer tokens.
func TestCoordinatorOrchestratorsCarryNoProviderDefectHandoff(t *testing.T) {
	for _, path := range allSDDOrchestratorAssetPaths(t) {
		t.Run(path, func(t *testing.T) {
			content := mustReadResolved(path)
			for _, forbidden := range []string{
				"Provider Defect Handoff",
				"report_and_continue",
				"continue_without_reporting",
				"review-integration.consent",
				"declined_this_candidate",
			} {
				if strings.Contains(content, forbidden) {
					t.Errorf("%s still carries the provider-defect handoff: %q", path, forbidden)
				}
			}
		})
	}
}

// TestCoordinatorOrchestratorsCarrySDDEditAuthorityConsentRelay is #2570's
// (S6 of #2540) guard: every orchestrator variant teaches the lossless relay
// of the typed SDD edit-authority consent envelope that native status emits
// on blocked(edit_authority_missing) (#2563), byte-identical across variants.
func TestCoordinatorOrchestratorsCarrySDDEditAuthorityConsentRelay(t *testing.T) {
	requirements := []string{
		"When native SDD status reports `blocked(edit_authority_missing)`",
		"typed `axiom.sdd-integration.consent/v1` envelope",
		"optional `consent` block",
		"Treat that envelope as a Lossless Blocking Prompt under this contract",
		"with the same discipline.",
		"Present the complete envelope once in the active conversation language",
		"faithfully translate the headline, reason, `value`, the missing-root evidence, choice labels, every choice `effect`, and the off-path note",
		"preserving the original choices, order, selection mode, exact allowed-answer domain, and answer tokens",
		"Never translate or alter the machine answer tokens (`granted`, `declined`), commands, paths, or invocations",
		"Never summarize, reshape, reorder, merge, or omit any part",
		"never answer on the human's behalf and never run the grant unprompted",
		"Only after the human's explicit `granted` answer",
		"execute the envelope's exact grant invocation verbatim, exactly once",
		"then re-enter through native status",
		"granted roots project into `allowedEditRoots`",
		"per-change, audited, and dies with archive",
		"run the envelope's decline invocation",
		"nothing is persisted",
		"names both exits",
		"edit tasks.md so every work unit stays inside the authorized edit roots, or grant this change edit authority",
		"A blocked status without a `consent` block names the same two exits; relay them and stop.",
	}

	for _, path := range allSDDOrchestratorAssetPaths(t) {
		t.Run(path, func(t *testing.T) {
			contract := sddConsentRelaySection(t, path)
			if canonical := sddConsentRelaySection(t, canonicalOrchestratorPath); contract != canonical {
				t.Error("SDD edit-authority consent relay differs from the canonical cross-variant block")
			}
			for _, required := range requirements {
				if !strings.Contains(contract, required) {
					t.Errorf("SDD edit-authority consent relay missing %q", required)
				}
			}
		})
	}
}

func sddConsentRelaySection(t *testing.T, path string) string {
	t.Helper()
	const heading = "#### SDD Edit-Authority Consent Relay (MANDATORY)"
	content := mustReadResolved(path)
	start := strings.Index(content, heading)
	if start == -1 {
		t.Fatalf("%s missing %q", path, heading)
	}
	contract := content[start:]
	const endMarker = "A blocked status without a `consent` block names the same two exits; relay them and stop."
	end := strings.Index(contract, endMarker)
	if end == -1 {
		t.Fatalf("%s SDD edit-authority consent relay missing terminal boundary", path)
	}
	return strings.TrimSpace(contract[:end+len(endMarker)])
}

func blockingPromptContractSection(t *testing.T, path string) string {
	t.Helper()
	const heading = "### Lossless Blocking Prompts (MANDATORY)"
	content := mustReadResolved(path)
	start := strings.Index(content, heading)
	if start == -1 {
		t.Fatalf("%s missing %q", path, heading)
	}
	contract := content[start:]
	if end := strings.Index(contract[len(heading):], "\n##"); end >= 0 {
		contract = contract[:len(heading)+end]
	}
	return contract
}
