package agentguidance

import (
	"fmt"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/capabilitymanifest"
	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
)

// TestDelegationDoctrineIsRenderedFromTheManifest pins that every number of the
// doctrine comes from the canonical routing facts.
func TestDelegationDoctrineIsRenderedFromTheManifest(t *testing.T) {
	t.Parallel()

	routing := capabilitymanifest.CanonicalImplementationRouting()
	rules := DelegationRules()
	triggers := DelegationTriggers()

	budget := fmt.Sprintf("one parallel batch of at most %d calls and approximately %dk tokens",
		routing.DirectInline.MaxEvidenceCalls, routing.DirectInline.ApproxEvidenceTokens/1000)
	handoff := fmt.Sprintf("a handoff of at most approximately %dk tokens with `path:line` evidence",
		routing.DelegatedDirect.ApproxHandoffTokens/1000)
	unit := fmt.Sprintf("about %d changed lines or less", routing.DirectInline.MaxInlineUnitLines)

	for name, section := range map[string]string{"rules": rules, "triggers": triggers} {
		if strings.Contains(section, "{{") {
			t.Errorf("%s keep an unresolved marker:\n%s", name, section)
		}
	}
	for _, want := range []string{budget, handoff, unit} {
		if !strings.Contains(rules, want) {
			t.Errorf("DelegationRules() is missing %q:\n%s", want, rules)
		}
	}
	for _, want := range []string{
		fmt.Sprintf("one parallel batch, at most %d calls, approximately %dk tokens", routing.DirectInline.MaxEvidenceCalls, routing.DirectInline.ApproxEvidenceTokens/1000),
		handoff,
		unit,
	} {
		if !strings.Contains(triggers, want) {
			t.Errorf("DelegationTriggers() is missing %q:\n%s", want, triggers)
		}
	}
}

// TestRoutingBlockEmbedsTheSharedDelegationTriggers is the single-source pin
// (decision P2): the routing block carries the very bytes the orchestrators
// resolve, for every supported agent.
func TestRoutingBlockEmbedsTheSharedDelegationTriggers(t *testing.T) {
	t.Parallel()

	triggers := DelegationTriggers()
	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			if !strings.Contains(rendered, triggers) {
				t.Fatalf("RenderRouting(%q) does not embed DelegationTriggers() verbatim", agent.ID)
			}
			if got := strings.Count(rendered, "### Mandatory Delegation Triggers"); got != 1 {
				t.Fatalf("RenderRouting(%q) renders the delegation triggers heading %d times, want 1", agent.ID, got)
			}
		})
	}
}
