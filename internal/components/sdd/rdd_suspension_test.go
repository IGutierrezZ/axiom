package sdd

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/rddharness"
)

// rddProse lists the phrases that identify RDD prose in a rendered orchestrator.
var rddProse = []string{
	"receipt-driven",
	"Review Execution Contract",
	"Provider Defect Handoff",
	"Native Checking Contract",
	"review-integration.consent",
	"declined_this_candidate",
	"axiom review",
	"RDD",
	"native review",
}

// TestRenderedOrchestratorsCarryNoRDDProseWhileSuspended pins C-01-c: without
// the RDD injection point every orchestrator renders free of RDD prose, for
// every runtime, including the OpenCode and Pi consent routes.
func TestRenderedOrchestratorsCarryNoRDDProseWhileSuspended(t *testing.T) {
	t.Cleanup(rddharness.SetForTesting(false))

	for _, entry := range catalog.AllAgents() {
		agent := entry.ID
		t.Run(string(agent), func(t *testing.T) {
			rendered := renderSDDOrchestratorAsset(agent)
			if agent == model.AgentOpenCode || agent == model.AgentKilocode {
				var err error
				if rendered, err = composeOpenCodeOrchestratorPrompt(agent); err != nil {
					t.Fatalf("composeOpenCodeOrchestratorPrompt(%s) error = %v", agent, err)
				}
			}
			for _, forbidden := range rddProse {
				if strings.Contains(rendered, forbidden) {
					t.Errorf("%s orchestrator still renders RDD prose %q", agent, forbidden)
				}
			}
			if strings.Contains(rendered, "{{GENTLE_AI_") {
				t.Errorf("%s orchestrator kept a raw placeholder", agent)
			}
		})
	}
}

// TestSuspendedRenderingKeepsTheReviewContractExportable pins that the provider
// contract bundle can still obtain the dormant contract while RDD is suspended.
func TestSuspendedRenderingKeepsTheReviewContractExportable(t *testing.T) {
	t.Cleanup(rddharness.SetForTesting(false))

	if rendersReviewLifecycle(model.AgentPi) {
		t.Fatal("rendersReviewLifecycle(pi) = true with RDD suspended")
	}
	contract, err := ReviewExecutionContractFor(model.AgentPi)
	if err != nil {
		t.Fatalf("ReviewExecutionContractFor(pi) error = %v", err)
	}
	if !strings.Contains(contract, "## Entry rule") {
		t.Fatal("ReviewExecutionContractFor(pi) lost its dormant contract body")
	}
}

// TestInjectedOpenCodeConsentRouteIsRetiredWhileSuspended pins that a sync
// retires the consent route a previous release wrote into a preserved prompt.
func TestInjectedOpenCodeConsentRouteIsRetiredWhileSuspended(t *testing.T) {
	t.Cleanup(rddharness.SetForTesting(false))

	preserved := "user policy\n" + openCodeConsentV3QuestionRoute + "\n" + openCodeConsentV3FallbackClause
	migrated := renderPreservedOpenCodeOrchestratorPrompt(preserved, model.AgentOpenCode)
	if strings.Contains(migrated, "review-integration") {
		t.Fatalf("preserved OpenCode prompt kept the consent route:\n%s", migrated)
	}
	if !strings.Contains(migrated, openCodeNativeQuestionSourceRoute) || !strings.Contains(migrated, "user policy") {
		t.Fatalf("preserved OpenCode prompt lost the source route or the user policy:\n%s", migrated)
	}
}

// TestSuspendedApplyCommandsCarryNoReviewLifecycle pins that the apply commands
// neither keep the raw authority-first placeholder nor the acknowledgement
// paragraph while RDD is suspended.
func TestSuspendedApplyCommandsCarryNoReviewLifecycle(t *testing.T) {
	t.Cleanup(rddharness.SetForTesting(false))

	for path, agent := range map[string]model.AgentID{
		"claude/commands/sdd-apply.md":   model.AgentClaudeCode,
		"opencode/commands/sdd-apply.md": model.AgentOpenCode,
	} {
		rendered := renderBoundedReviewAsset(agent, path)
		for _, forbidden := range []string{"{{GENTLE_AI_", "Authority-First Terminal Procedure", "acknowledgement continuation", "burns authority"} {
			if strings.Contains(rendered, forbidden) {
				t.Errorf("%s still renders review lifecycle prose %q", path, forbidden)
			}
		}
	}
	if rendered := renderBoundedReviewAsset(model.AgentClaudeCode, "claude/commands/sdd-apply.md"); !strings.Contains(rendered, "Never auto-launch Judgment Day.") {
		t.Error("claude apply command lost the Judgment Day sentence")
	}
}
