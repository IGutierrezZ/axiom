package agentguidance

import "strings"

// informedApprovalAndContinuity is the doctrine for short affirmative replies
// and for the continuity of an authorized objective. It is absorbed from
// upstream #5850 (informed approval) and #5486 (authorized work continuity).
//
// It lives beside the routing doctrine, not in the shared SDD sections,
// because it governs every request that enters the orchestrator, not only SDD
// work. RenderRouting is its single source: the 12 orchestrator templates
// receive it through the routing marker and none restates it.
const informedApprovalAndContinuity = `### Informed approval and continuity

- A short affirmative reply ("yes", "go ahead", "that one", "ok", or the equivalent in the user's language) is read against the proposal or question immediately before it, and approves exactly that, within its stated scope. It never widens the scope.
- If that proposal was ambiguous, or offered several options, a short reply does not choose: ask again.
- Contextual approval never replaces provider-owned consent (` + "`gentle-ai.review-integration.consent/v3`, `axiom.sdd-integration.consent/v1`" + `), the exact allowed answers of a Lossless Blocking Prompt, or the separate explicit permission that publishing, remote, and destructive actions require.
- Resolving a subdecision or an intermediate question does not cancel or replace the objective already authorized: after the answer, continue with the authorized work that remains.
`

// InformedApprovalAndContinuity returns the informed-approval and continuity
// doctrine rendered into every routing block.
func InformedApprovalAndContinuity() string {
	return strings.TrimSpace(informedApprovalAndContinuity)
}
