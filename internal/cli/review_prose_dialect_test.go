package cli

import (
	"strings"
	"testing"
)

// TestReviewProseNamesTheAxiomContract pins the refusal wording that tells a
// caller which contract to negotiate. Prose is read by a person or an agent who
// then runs what it names, so it must name the axiom v2 identifier: naming the
// legacy gentle-ai alias steered callers into the dialect whose later
// transitions publish `gentle-ai review ...` commands. The dialect itself, and
// the legacy inputs it still accepts, are untouched.
func TestReviewProseNamesTheAxiomContract(t *testing.T) {
	unsupportedContractErr := validateReviewIntegrationContract("no-such-contract")
	if unsupportedContractErr == nil {
		t.Fatal("an unknown review contract was accepted")
	}

	tests := []struct {
		name  string
		prose string
		// keepsV1 marks wording that also names the v1 contract, which has no
		// axiom spelling and so legitimately keeps its gentle-ai identifier.
		keepsV1 bool
	}{
		{name: "action eligibility needs a contract", prose: reviewContractRequiredForActionEligibilityReason, keepsV1: true},
		{name: "target selectors need a contract", prose: reviewStatusTargetSelectorsRequireContractReason, keepsV1: true},
		{name: "start target needs a contract", prose: reviewStartTargetRequiresContractReason},
		{name: "start consent needs the negotiated form", prose: reviewStartConsentRequiresContractReason},
		{name: "unsupported contract retry", prose: unsupportedContractErr.Error(), keepsV1: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.prose, AxiomReviewIntegrationContractV2) {
				t.Fatalf("prose %q does not name the axiom contract %q", tt.prose, AxiomReviewIntegrationContractV2)
			}
			if strings.Contains(tt.prose, ReviewIntegrationContractV2) {
				t.Fatalf("prose %q names the legacy v2 contract %q", tt.prose, ReviewIntegrationContractV2)
			}
			remaining := tt.prose
			if tt.keepsV1 {
				remaining = strings.ReplaceAll(remaining, ReviewIntegrationContractV1, "")
			}
			if strings.Contains(remaining, "gentle-ai") {
				t.Fatalf("prose %q still names the gentle-ai tool or contract", tt.prose)
			}
		})
	}
}

// TestReviewProseKeepsAcceptingLegacyContracts proves the wording fix is not a
// compatibility break: every spelling a caller could already negotiate is still
// accepted, even though prose no longer advertises the legacy alias.
func TestReviewProseKeepsAcceptingLegacyContracts(t *testing.T) {
	for _, contract := range []string{ReviewIntegrationContractV1, ReviewIntegrationContractV2, AxiomReviewIntegrationContractV2} {
		t.Run(contract, func(t *testing.T) {
			if err := validateReviewIntegrationContract(contract); err != nil {
				t.Fatalf("contract %q was rejected: %v", contract, err)
			}
		})
	}
}
