package sddstatus

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/consentenvelope"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
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

// publicationKeepsApprovalNote is the closing sentence every publishing effect and the value carry
// (REQ-27.12): a refused publication never undoes the recorded approval, and Axiom publishes
// documents only.
const publicationKeepsApprovalNote = "A refused publication keeps the approval. Code pushes, pull requests and merges remain yours."

// resolveGatePublication asks the branch engine, with reads only, where approving a planning gate
// publishes the change documents (REQ-27.7, REQ-27.12). It returns the publication when the target
// resolves, the refusal code when it does not (for example wrong_branch), and neither when no
// engine is wired (tests that disable the branch guard): the question then names no remote or
// branch and its choices stay the gate record ones.
func resolveGatePublication(change, workspaceRoot string) (*GatePublication, string) {
	eng := engine.Default()
	if branchEngine != nil {
		eng = branchEngine()
	}
	if eng == nil {
		return nil, ""
	}
	target, refusal := eng.PublicationTarget(context.Background(), branch.PublicationRequest{CWD: workspaceRoot, Change: change})
	if refusal != nil {
		return nil, string(refusal.Code)
	}
	return &GatePublication{
		Remote:     branch.PublishRemote,
		Branch:     target.Branch,
		PRTarget:   target.PRTarget,
		Invocation: branch.PublishInvocation(workspaceRoot, change, target.Branch),
	}, ""
}

// publicationBranchLabel names the branch that is pushed: the one the target resolved, or the change
// branch when it did not.
func publicationBranchLabel(publication *GatePublication) string {
	if publication == nil {
		return "the change branch"
	}
	return "`" + publication.Branch + "`"
}

// publicationDestination names where the documents go for the effects and the value: the remote
// branch when the target resolved, and the change branch on origin otherwise.
func publicationDestination(publication *GatePublication) string {
	if publication == nil {
		return "the change branch on `origin`"
	}
	return fmt.Sprintf("`%s/%s`", publication.Remote, publication.Branch)
}

// gateQuestionNarrative writes the headline, reason and value of the question.
// Integration keeps its two-way wording; every other gate describes the three
// exits, or the two that remain when the store rules the hand-off out.
func gateQuestionNarrative(change, gateKey, handOffUnavailable string, publication *GatePublication) (headline, reason, value string) {
	headline = fmt.Sprintf("Block review gate %q is awaiting your decision.", gateKey)
	reason = fmt.Sprintf("change %q has not approved or rejected this gate yet, and design.md's block-review policy blocks every later phase or role until it is (design.md S5.5).", change)
	if gateKey == string(kickoff.GateIntegration) {
		value = "Approving unblocks the next phase or role; rejecting keeps this gate open to remediation. Either decision, and its reason, is recorded in the change's gate ledger for later audit."
		return headline, reason, value
	}
	value = fmt.Sprintf("Iterate keeps this gate open to remediation as many times as needed; approve, publish and stop records the approval, marks the next phase as handed off and publishes the change documents to %s without forcing; approve, publish and continue records the approval, publishes the same documents and starts the next phase in this session. Choosing either publishing exit authorizes that push with the git credentials already configured, and nothing else. %s Every decision, and its reason, is recorded in the change's gate ledger for later audit.",
		publicationDestination(publication), publicationKeepsApprovalNote)
	if handOffUnavailable != "" {
		value = fmt.Sprintf("Iterate keeps this gate open to remediation as many times as needed; approve and continue records the approval and starts the next phase in this session. Approve, publish and stop is not available: the declared artifact store %q keeps no files to commit and push, and nothing is published. Every decision, and its reason, is recorded in the change's gate ledger for later audit.", handOffUnavailable)
	}
	return headline, reason, value
}

// gateQuestionChoices builds the choices in the order expectedGateAnswers
// declares. All of them share the gate record verb and differ only by
// --decision and --hand-off.
func gateQuestionChoices(cwd, change, gateKey string, answers []string, publication *GatePublication) []consentenvelope.Choice {
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
	destination := publicationDestination(publication)
	if gateKey != string(kickoff.GateIntegration) {
		approved.Label = "Approve and continue"
		approved.Effect = fmt.Sprintf("Records an approved decision for gate %q and unblocks the next phase or role in this session.", gateKey)
		if slices.Contains(answers, gateAnswerHandOff) {
			// A store that keeps files publishes on approval (REQ-27.12); one that keeps none publishes
			// nothing, so exit 3 keeps its plain wording and exit 2 is not offered (REQ-26.6).
			approved.Label = "Approve, publish and continue"
			approved.Effect = fmt.Sprintf("Records an approved decision for gate %q, publishes the change documents to %s with `axiom sdd publish`, and starts the next phase in this session. %s", gateKey, destination, publicationKeepsApprovalNote)
		}
		rejected.Label = "Iterate on this artifact"
		rejected.Effect = fmt.Sprintf("Keeps gate %q open to remediation; nothing later starts until it is re-approved, and the same gate is presented again afterwards.", gateKey)
	}
	byAnswer := map[string]consentenvelope.Choice{
		gateAnswerApproved: approved,
		gateAnswerRejected: rejected,
		gateAnswerHandOff: {
			Answer:     gateAnswerHandOff,
			Label:      "Approve, publish and stop",
			Effect:     fmt.Sprintf("Records an approved decision for gate %q marked as handed off, then publishes the change documents: `axiom sdd publish` commits only the documentation paths of the change and pushes %s to `origin` without forcing, using the git credentials already configured. Whoever continues runs `git fetch origin` and `axiom branch ensure`; no merged pull request is needed. %s", gateKey, publicationBranchLabel(publication), publicationKeepsApprovalNote),
			Invocation: base + " --decision approved" + gateHandOffFlag + " --reason <your-reason>",
		},
	}
	choices := make([]consentenvelope.Choice, 0, len(answers))
	for _, answer := range answers {
		choices = append(choices, byAnswer[answer])
	}
	return choices
}

// governanceRoleAssignments is the roster of the sealed kickoff in the shape
// kickoff.HandedOffTo and the blocking-role rules (REQ-21.15) read: the role
// names of the Governance roster, each with the gate policy the sealed kickoff
// gives it. A role the kickoff does not list keeps an empty policy, which is the
// default (blocking).
func governanceRoleAssignments(governance *Governance) []multirole.RoleAssignment {
	policies := make(map[string]multirole.GatePolicy, len(governance.Kickoff.Config.Roles))
	for _, role := range governance.Kickoff.Config.Roles {
		policies[role.Role] = role.GatePolicy
	}
	roster := make([]multirole.RoleAssignment, 0, len(governance.Roster.Roles))
	for _, role := range governance.Roster.Roles {
		roster = append(roster, multirole.RoleAssignment{Role: role, GatePolicy: policies[role]})
	}
	return roster
}

// governanceHandedOffTo names what an approved gate handed over (REQ-26.7). It
// answers "" for a gate that carries no hand-off mark, so a change that only
// ever used exits 1 and 3 projects exactly what it projected before. The phase
// the record persisted wins; only a record written before that field existed
// falls back to the calculation against the current roster.
func governanceHandedOffTo(governance *Governance, gate kickoff.GateState) string {
	if governance == nil || !gate.HandedOff {
		return ""
	}
	if gate.HandedOffTo != "" {
		return gate.HandedOffTo
	}
	return kickoff.HandedOffTo(gate, governanceRoleAssignments(governance), governance.Gates)
}

// handedOffNotes reports, as informational notes, every gate approved with exit
// 2 (REQ-26.7). The hand-off never blocks and never changes the route: a gate
// approved with the mark is approved like any other, so these notes are the
// only text that tells whoever picks the change up that the previous phase was
// not reviewed by them.
func handedOffNotes(governance *Governance) []string {
	if !governanceGated(governance) {
		return nil
	}
	var notes []string
	for _, gate := range governance.Gates {
		handedOffTo := governanceHandedOffTo(governance, gate)
		if gate.Status != "approved" || handedOffTo == "" {
			continue
		}
		notes = append(notes, fmt.Sprintf(
			"La compuerta %q se aprobó cediendo `%s` a otra persona: quien recoge el cambio empieza esa fase sin revisar la anterior.",
			string(gate.Key), handedOffTo))
	}
	return notes
}

// verifyDependencyFromRoleGates is DD-10's rule: REQ-21.15 forbids starting the
// global "verify" phase while a role of a multi-role roster with policy
// blocking has no approved role-apply gate. A role with policy deferred or
// optional never blocks it, whatever the state of its gate; nonBlockingRoleNotes
// reports those roles. It replaces the rule that consulted an integration handoff.md,
// which is no longer part of the SDD flow: the ledger already says which roles
// closed, so no second document has to agree with it.
//
// It only ever NARROWS baseline, from ready to blocked (the same shape as
// archiveDependencyFromGovernance), so it never adds a blocking reason to a
// change in spec, design or tasks, whose verify is already blocked for its own
// cause. A single-role roster, a continuous seal and an unsealed change keep
// baseline untouched, and handoff_policy has no effect on any of them
// (REQ-26.9).
//
// It returns the adjusted state and a non-empty genuine reason exactly when it
// blocks verify; an unchanged baseline always pairs with an empty reason.
func verifyDependencyFromRoleGates(governance *Governance, baseline DependencyState) (DependencyState, string) {
	if !governanceGated(governance) || len(governance.Roster.Roles) <= 1 || baseline != DependencyReady {
		return baseline, ""
	}
	if lastRoleClosedForGovernance(governance) {
		return baseline, ""
	}
	approved := make(map[kickoff.GateKey]bool, len(governance.Gates))
	for _, gate := range governance.Gates {
		if gate.Status == "approved" {
			approved[gate.Key] = true
		}
	}
	var open []string
	for _, role := range governanceRoleAssignments(governance) {
		if kickoff.IsBlockingRole(role) && !approved[kickoff.RoleApplyGate(role.Role)] {
			open = append(open, role.Role)
		}
	}
	if len(open) == 0 {
		// Ningún rol blocking: nada que esperar (un roster solo con roles
		// deferred u optional).
		return baseline, ""
	}
	return DependencyBlocked, fmt.Sprintf(
		"La fase verify global no puede iniciarse: faltan roles sin la compuerta role-apply aprobada (%s) (REQ-21.15).",
		strings.Join(open, ", "))
}

// nonBlockingRoleNotes reports, as informational notes, every role with policy
// deferred or optional whose role-apply gate is not approved: the role, its
// policy and the state of its gate. They never block verify (REQ-21.15); the
// notes only make visible that the global verification runs with that work
// still open. A change with only blocking roles projects no note.
func nonBlockingRoleNotes(governance *Governance) []string {
	if !governanceGated(governance) {
		return nil
	}
	var notes []string
	for _, pending := range kickoff.PendingNonBlockingRoles(governanceRoleAssignments(governance), governance.Gates) {
		state := "pendiente"
		if pending.Status == "rejected" {
			state = "rechazado"
		}
		notes = append(notes, fmt.Sprintf(
			"El rol %q (política %s) tiene su compuerta role-apply %s: no bloquea la fase verify global (REQ-21.15).",
			pending.Role, pending.Policy, state))
	}
	return notes
}
