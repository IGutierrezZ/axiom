package agentguidance

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
)

// TestRenderRoutingCarriesInformedApprovalOnceForEveryAgent pins the single
// source of the informed-approval and continuity doctrine: every supported
// agent receives it exactly once, with its limits, and no SDD or ODD wording
// around it restates it.
func TestRenderRoutingCarriesInformedApprovalOnceForEveryAgent(t *testing.T) {
	t.Parallel()

	doctrine := InformedApprovalAndContinuity()
	for _, want := range []string{
		"immediately before it, and approves exactly that",
		"never widens the scope",
		"does not choose: ask again",
		"gentle-ai.review-integration.consent/v3",
		"axiom.sdd-integration.consent/v1",
		"exact allowed answers of a Lossless Blocking Prompt",
		"publishing, remote, and destructive actions",
		"does not cancel or replace the objective already authorized",
	} {
		if !strings.Contains(doctrine, want) {
			t.Errorf("InformedApprovalAndContinuity() is missing %q:\n%s", want, doctrine)
		}
	}

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			if got := strings.Count(rendered, doctrine); got != 1 {
				t.Fatalf("RenderRouting(%q) renders the doctrine %d times, want exactly once", agent.ID, got)
			}
			if got := strings.Count(rendered, "### Informed approval and continuity\n"); got != 1 {
				t.Fatalf("RenderRouting(%q) renders the heading %d times, want exactly once", agent.ID, got)
			}
		})
	}
}
