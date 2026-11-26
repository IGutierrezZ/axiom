package agentguidance

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
)

// TestODDRoutingNeverInvokesTrackerVerbs pins prov:ORQ:2 for every runtime: the ODD doctrine names
// the three tracker verb families only to forbid them, and says nothing that runs them.
func TestODDRoutingNeverInvokesTrackerVerbs(t *testing.T) {
	t.Parallel()

	const ban = "ODD never invokes `axiom integrations tasks`, `axiom integrations summary` or `axiom integrations time`"
	for _, agent := range catalog.AllAgents() {
		agent := agent
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()
			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			if !strings.Contains(workItemRoutingSection(t, string(agent.ID)), ban) {
				t.Errorf("%s ODD work-item section lacks the tracker-verb ban", agent.ID)
			}
			for _, verb := range []string{"axiom integrations tasks", "axiom integrations summary", "axiom integrations time"} {
				if got := strings.Count(rendered, verb); got != 1 {
					t.Errorf("%s ODD routing names %q %d times, want 1 (the ban)", agent.ID, verb, got)
				}
			}
		})
	}
}
