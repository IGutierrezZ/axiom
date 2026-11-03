package sddstatus

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/consentenvelope"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// gateHandOffFlag is the flag only the hand_off choice may carry (REQ-26.2):
// approving and continuing, approving and handing off, and iterating are the
// same `axiom sdd gate record` verb, told apart by --decision and --hand-off.
const gateHandOffFlag = " --hand-off"

// gateQuestionStore is the artifact store the gate question reasons about: the
// one the workspace declares and, only when it declares none, the one the
// status resolved. It is the same rule effectiveArtifactStore applies, kept
// here so status.go changes by a single argument.
func gateQuestionStore(workspaceRoot string, resolved ArtifactStore) ArtifactStore {
	if declared, ok := declaredArtifactStore(workspaceRoot); ok {
		return declared
	}
	return resolved
}

// handOffUnavailableFor names the store that rules the hand-off exit out
// (REQ-26.6): a store that keeps no files has nothing to commit, push or merge.
// The empty string means the exit is available.
func handOffUnavailableFor(store ArtifactStore) string {
	switch store {
	case ArtifactStoreEngram, ArtifactStoreNone:
		return string(store)
	}
	return ""
}

// expectedGateAnswers derives, from the gate key and from HandOffUnavailable,
// the exact answer list the question must carry (REQ-26.2, REQ-26.6):
//   - integration keeps its two-way [approved, rejected] choice;
//   - every other gate offers [rejected, hand_off, approved], or
//     [rejected, approved] when a file-less store marks the exit unavailable.
//
// A two-option planning gate without handOffUnavailable is not valid: the
// missing exit must always be explained.
func expectedGateAnswers(gate, handOffUnavailable string) ([]string, error) {
	if gate == string(kickoff.GateIntegration) {
		if handOffUnavailable != "" {
			return nil, fmt.Errorf("SDD governance gate %q has no hand-off exit to mark unavailable", gate) // refusal:by-design world-action: this envelope is built and validated by the same package; the exit is a code fix, not a command
		}
		return []string{gateAnswerApproved, gateAnswerRejected}, nil
	}
	switch handOffUnavailable {
	case "":
		return []string{gateAnswerRejected, gateAnswerHandOff, gateAnswerApproved}, nil
	case string(ArtifactStoreEngram), string(ArtifactStoreNone):
		return []string{gateAnswerRejected, gateAnswerApproved}, nil
	}
	return nil, fmt.Errorf("SDD governance gate handOffUnavailable %q is not a store without files", handOffUnavailable) // refusal:by-design world-action: this envelope is built and validated by the same package; the exit is a code fix, not a command
}

// validateGateChoiceInvocations enforces the two rules the three-way question
// adds on top of the shared prefix check: every invocation is different, and
// only the hand_off choice carries --hand-off. Without the first a relay could
// not tell two answers apart; without the second approving and continuing
// could silently hand the next phase to someone else.
func validateGateChoiceInvocations(choices []consentenvelope.Choice) error {
	seen := make(map[string]string, len(choices))
	for _, choice := range choices {
		if earlier, ok := seen[choice.Invocation]; ok {
			return fmt.Errorf("SDD governance gate choices %q and %q share the invocation %q", earlier, choice.Answer, choice.Invocation) // refusal:by-design world-action: this envelope is built and validated by the same package; the exit is a code fix, not a command
		}
		seen[choice.Invocation] = choice.Answer
		carriesHandOff := strings.Contains(choice.Invocation, gateHandOffFlag)
		if carriesHandOff != (choice.Answer == gateAnswerHandOff) {
			return fmt.Errorf("SDD governance gate choice %q must carry %q only on the hand_off answer: %q", choice.Answer, strings.TrimSpace(gateHandOffFlag), choice.Invocation) // refusal:by-design world-action: this envelope is built and validated by the same package; the exit is a code fix, not a command
		}
	}
	return nil
}

// gateQuestionNarrative writes the headline, reason and value of the question.
// Integration keeps its two-way wording; every other gate describes the three
// exits, or the two that remain when the store rules the hand-off out.
func gateQuestionNarrative(change, gateKey, handOffUnavailable string) (headline, reason, value string) {
	headline = fmt.Sprintf("Block review gate %q is awaiting your decision.", gateKey)
	reason = fmt.Sprintf("change %q has not approved or rejected this gate yet, and design.md's block-review policy blocks every later phase or role until it is (design.md S5.5).", change)
	if gateKey == string(kickoff.GateIntegration) {
		value = "Approving unblocks the next phase or role; rejecting keeps this gate open to remediation. Either decision, and its reason, is recorded in the change's gate ledger for later audit."
		return headline, reason, value
	}
	value = "Iterate keeps this gate open to remediation as many times as needed; approve, push and stop records the approval, marks the next phase as handed off and returns what to commit and push; approve and continue records the approval and starts the next phase in this session. Every decision, and its reason, is recorded in the change's gate ledger for later audit."
	if handOffUnavailable != "" {
		value = fmt.Sprintf("Iterate keeps this gate open to remediation as many times as needed; approve and continue records the approval and starts the next phase in this session. Approve, push and stop is not available: the declared artifact store %q keeps no files to commit and push. Every decision, and its reason, is recorded in the change's gate ledger for later audit.", handOffUnavailable)
	}
	return headline, reason, value
}

// gateQuestionChoices builds the choices in the order expectedGateAnswers
// declares. All of them share the gate record verb and differ only by
// --decision and --hand-off.
func gateQuestionChoices(cwd, change, gateKey string, answers []string) []consentenvelope.Choice {
	base := fmt.Sprintf("%s--cwd %s --change %s --gate %s", gateRecordInvocationPrefix, cwd, change, gateKey)
	approved := consentenvelope.Choice{
		Answer:     gateAnswerApproved,
		Label:      "Approve this gate",
		Effect:     fmt.Sprintf("Records an approved decision for gate %q and unblocks the next phase or role.", gateKey),
		Invocation: base + " --decision approved --reason <your-reason>",
	}
	rejected := consentenvelope.Choice{
		Answer:     gateAnswerRejected,
		Label:      "Reject this gate",
		Effect:     fmt.Sprintf("Keeps gate %q open to remediation; nothing later starts until it is re-approved.", gateKey),
		Invocation: base + " --decision rejected --reason <your-reason>",
	}
	if gateKey != string(kickoff.GateIntegration) {
		approved.Label = "Approve and continue"
		approved.Effect = fmt.Sprintf("Records an approved decision for gate %q and unblocks the next phase or role in this session.", gateKey)
		rejected.Label = "Iterate on this artifact"
		rejected.Effect = fmt.Sprintf("Keeps gate %q open to remediation; nothing later starts until it is re-approved, and the same gate is presented again afterwards.", gateKey)
	}
	byAnswer := map[string]consentenvelope.Choice{
		gateAnswerApproved: approved,
		gateAnswerRejected: rejected,
		gateAnswerHandOff: {
			Answer:     gateAnswerHandOff,
			Label:      "Approve, push and stop",
			Effect:     fmt.Sprintf("Records an approved decision for gate %q marked as handed off, refusing without writing anything while the artifacts are not committed, and returns the commit, push and pull request data to present before stopping; the next phase starts only after that pull request is merged.", gateKey),
			Invocation: base + " --decision approved" + gateHandOffFlag + " --reason <your-reason>",
		},
	}
	choices := make([]consentenvelope.Choice, 0, len(answers))
	for _, answer := range answers {
		choices = append(choices, byAnswer[answer])
	}
	return choices
}
