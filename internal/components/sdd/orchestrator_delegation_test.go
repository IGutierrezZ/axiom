package sdd

import (
	"path"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/components/agentguidance"
)

// TestEveryOrchestratorRendersTheSingleSourceDelegationDoctrine pins decision P2
// of guia-orquestador-upstream: the delegation rules and triggers every
// runtime receives are the bytes agentguidance renders, exactly once, with no
// marker left.
func TestEveryOrchestratorRendersTheSingleSourceDelegationDoctrine(t *testing.T) {
	rules := agentguidance.DelegationRules()
	triggers := agentguidance.DelegationTriggers()

	for _, templatePath := range orchestratorTemplatePaths(t) {
		t.Run(path.Dir(templatePath), func(t *testing.T) {
			rendered := resolvedOrchestratorSource(templatePath)

			for _, marker := range []string{delegationRulesMarker, delegationTriggersMarker} {
				if strings.Contains(rendered, marker) {
					t.Fatalf("%s kept the raw %s marker", templatePath, marker)
				}
			}
			if got := strings.Count(rendered, rules); got != 1 {
				t.Errorf("%s renders the delegation rules %d times, want 1", templatePath, got)
			}
			if got := strings.Count(rendered, triggers); got != 1 {
				t.Errorf("%s renders the delegation triggers %d times, want 1", templatePath, got)
			}
		})
	}
}
