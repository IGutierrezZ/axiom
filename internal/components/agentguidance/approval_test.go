package agentguidance

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
)

// TestRenderRoutingCarriesInformedApprovalOnceForEveryAgent pins the single
// source of the informed-approval and continuity doctrine: every supported
// agent receives it exactly once, and it keeps the consent envelope token.
func TestRenderRoutingCarriesInformedApprovalOnceForEveryAgent(t *testing.T) {
	t.Parallel()

	doctrine := InformedApprovalAndContinuity()
	if !strings.Contains(doctrine, "axiom.sdd-integration.consent/v1") {
		t.Errorf("InformedApprovalAndContinuity() is missing the consent envelope token:\n%s", doctrine)
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
