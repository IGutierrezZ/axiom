package sdd

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/capabilitymanifest"
	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/components/agentguidance"
)

// retiredFileCountDelegationRule matches every delegation rule keyed to a number
// of files, in any wording the twelve orchestrators ever used: "4+ files",
// "1–3 files", "1-3 files", "2+ non-trivial files", "4-file rule", the old rule
// names and the old table rows. Delegation decides by evidence cost instead.
var retiredFileCountDelegationRule = regexp.MustCompile(`(?i)\d\s*\+\s*(non-trivial\s+)?files|\d\s*[–-]\s*\d\s+files|up to \d+ files|\d+\s+or\s+more\s+(non-trivial\s+)?files|\d+-file rule|Bounded read rule|\*\*Write rule\*\*|\*\*Context rule\*\*|Multiple files, new logic`)

// TestEveryOrchestratorRendersTheSingleSourceDelegationDoctrine pins decision P2
// of guia-orquestador-upstream: the delegation rules and triggers every
// runtime receives are the bytes agentguidance renders, exactly once, with no
// marker left and no file-count rule anywhere.
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
			for _, want := range []string{"**direct inline**", "**delegated direct**", "**optional SDD**", "never selects SDD", "creates SDD state", "`sdd-*`"} {
				if !strings.Contains(rendered, want) {
					t.Errorf("%s is missing route-neutral clause %q", templatePath, want)
				}
			}
			if match := retiredFileCountDelegationRule.FindString(rendered); match != "" {
				t.Errorf("%s keeps the retired file-count delegation rule %q", templatePath, match)
			}
		})
	}
}

// TestSmallModelDelegationRulesMatchTheManifest keeps the numbers that the
// generic small-model section repeats in lockstep with the canonical routing.
func TestSmallModelDelegationRulesMatchTheManifest(t *testing.T) {
	routing := capabilitymanifest.CanonicalImplementationRouting()
	content := assets.MustRead("generic/sdd-orchestrator.md")
	start := strings.Index(content, "<!-- section:model-small -->")
	end := strings.Index(content, "<!-- /section:model-small -->")
	if start < 0 || end <= start {
		t.Fatal("generic orchestrator has no model-small section")
	}
	small := content[start:end]

	for _, want := range []string{
		fmt.Sprintf("at most %d calls and about %dk tokens", routing.DirectInline.MaxEvidenceCalls, routing.DirectInline.ApproxEvidenceTokens/1000),
		fmt.Sprintf("at most about %dk tokens", routing.DelegatedDirect.ApproxHandoffTokens/1000),
		fmt.Sprintf("about %d changed lines or less", routing.DirectInline.MaxInlineUnitLines),
		"named reason",
	} {
		if !strings.Contains(small, want) {
			t.Errorf("model-small delegation rules are missing %q", want)
		}
	}
	if match := retiredFileCountDelegationRule.FindString(small); match != "" {
		t.Errorf("model-small section keeps the retired file-count delegation rule %q", match)
	}
}
