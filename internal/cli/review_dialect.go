package cli

import (
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// reviewAxiomCommandTool is the published tool name of the axiom dialect. The
// default tool stays reviewTransitionCommandTool until the producers learn to
// echo the negotiated dialect.
const reviewAxiomCommandTool = "axiom"

// isReviewContractV2 reports whether contract negotiates the review
// integration v2 lifecycle in either dialect: the canonical
// axiom.review-integration/v2 or its legacy gentle-ai.review-integration/v2
// alias. Every other contract (v1, empty, unknown, or another contract family)
// is not the v2 lifecycle. reviewtransaction.ResolveReviewContract owns the
// alias table, so a future alias is accepted here without touching a caller.
func isReviewContractV2(contract string) bool {
	canonical, _, err := reviewtransaction.ResolveReviewContract(contract)
	return err == nil && canonical == reviewtransaction.AxiomReviewIntegrationV2Contract
}

// reviewCommandCanonicalTool rewrites a leading axiom tool to the default
// published tool so a validator can compare a command line in either dialect
// against the one renderer that owns the bytes. A command that does not start
// with the axiom tool is returned unchanged.
func reviewCommandCanonicalTool(command string) string {
	if rest, found := strings.CutPrefix(command, reviewAxiomCommandTool+" "); found {
		return reviewTransitionCommandTool + " " + rest
	}
	return command
}
