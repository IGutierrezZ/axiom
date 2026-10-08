package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// reviewDialect names which of the two published spellings of the review
// integration protocol a caller negotiated: the legacy gentle-ai one or the
// axiom one. It is derived from the --contract string on every invocation and
// never persisted, so a lineage carries no dialect: each caller is answered in
// the dialect it asked for, and the zero value is the legacy one (v1, and any
// caller that negotiated nothing).
//
// Only the command tool and the v2 contract identifier are echoed. The schema
// identifiers, hash domain separators, and persisted ids keep their published
// gentle-ai spelling on purpose: they are wire contract, not dialect.
type reviewDialect struct{ axiom bool }

// reviewAxiomCommandTool is the published tool name of the axiom dialect.
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

// reviewDialectForContract resolves the dialect a --contract string negotiates.
// Only the canonical axiom v2 identifier selects axiom; the legacy alias, v1,
// an empty value, and anything unresolvable stay on the legacy dialect so their
// bytes do not change.
func reviewDialectForContract(contract string) reviewDialect {
	canonical, legacy, err := reviewtransaction.ResolveReviewContract(contract)
	return reviewDialect{axiom: err == nil && !legacy && canonical == reviewtransaction.AxiomReviewIntegrationV2Contract}
}

// captureContractArguments is the one extra argument the provider-issued
// tokens of the capture-* and acknowledge-approved commands carry for a caller
// that negotiated the axiom dialect: those commands read their dialect from
// --contract, so STATUS hands it back to them. The legacy dialect adds nothing,
// so its tokens (and the continuations the commands then emit) keep the exact
// bytes they always had. The argument is always the first one of the token list.
func (dialect reviewDialect) captureContractArguments() []ReviewTransitionArgument {
	if !dialect.axiom {
		return nil
	}
	return []ReviewTransitionArgument{{Name: "contract", Value: dialect.ContractV2()}}
}

// bindingContract is the contract an opaque provider Task binding carries for
// this dialect: the axiom identifier, or the empty string for the legacy dialect
// so its Task bytes never change. It is how a host relay that holds no --contract
// hands the dialect back to the transport that closes the review.
func (dialect reviewDialect) bindingContract() string {
	if !dialect.axiom {
		return ""
	}
	return dialect.ContractV2()
}

// reviewDialectForBindingContract resolves the dialect an opaque Task binding
// names. Only the empty string (legacy) and the axiom v2 identifier are
// admissible: anything else was not issued by this product.
func reviewDialectForBindingContract(contract string) (reviewDialect, bool) {
	switch contract {
	case "":
		return reviewDialect{}, true
	case (reviewDialect{axiom: true}).ContractV2():
		return reviewDialect{axiom: true}, true
	}
	return reviewDialect{}, false
}

// withCaptureContract returns arguments led by the capture contract argument of
// this dialect (none for the legacy one), without touching the input slice.
func (dialect reviewDialect) withCaptureContract(arguments []ReviewTransitionArgument) []ReviewTransitionArgument {
	return append(dialect.captureContractArguments(), arguments...)
}

// reviewCaptureCommandDialect resolves the dialect of a capture-* or
// acknowledge-approved invocation from its optional --contract. Without the flag
// the command answers exactly as it always did, in the legacy dialect. With it,
// only the v2 lifecycle in either spelling is accepted: those commands exist only
// there, so v1 and unknown values are refused before anything is read or written.
func reviewCaptureCommandDialect(flags *flag.FlagSet, command, contract string) (reviewDialect, error) {
	if !reviewFlagWasProvided(flags, "contract") {
		return reviewDialect{}, nil
	}
	if !isReviewContractV2(contract) {
		return reviewDialect{}, reviewPreflightError(fmt.Errorf("review %s accepts only --contract %s or --contract %s, not %q", command, ReviewIntegrationContractV2, AxiomReviewIntegrationContractV2, contract)) // refusal:by-design operator-knowledge: the capture and acknowledgement commands exist only in the v2 lifecycle; run the exact tokens STATUS issued
	}
	return reviewDialectForContract(contract), nil
}

// Tool returns the command tool this dialect publishes.
func (dialect reviewDialect) Tool() string {
	if dialect.axiom {
		return reviewAxiomCommandTool
	}
	return reviewTransitionCommandTool
}

// ContractV2 returns the v2 contract identifier this dialect publishes.
func (dialect reviewDialect) ContractV2() string {
	return reviewtransaction.MatchReviewDialect(ReviewIntegrationContractV2, !dialect.axiom)
}

// refreshCommand is the exact bootstrap STATUS command of this dialect.
func (dialect reviewDialect) refreshCommand() string {
	return dialect.Tool() + " review status --cwd <repo> --contract " + dialect.ContractV2() + " --next-transition"
}

// commandPrefix is the leading words every command line of this dialect shares
// for one review verb, e.g. "axiom review start ".
func (dialect reviewDialect) commandPrefix(verb string) string {
	return dialect.Tool() + " review " + verb + " "
}

// ownsCommand reports whether command is a review command line of this
// dialect's tool, so a validator can refuse a command that echoes the wrong one.
func (dialect reviewDialect) ownsCommand(command string) bool {
	return strings.HasPrefix(command, dialect.Tool()+" review ")
}

// commandLines lists every runnable command line a transition publishes: the
// execute command and each unachievable-slot withdraw command.
func (transition ReviewNextTransition) commandLines() []string {
	var commands []string
	if transition.Execute != nil {
		commands = append(commands, transition.Execute.Command)
	}
	if transition.UnachievableLensSlots != nil {
		for _, slot := range *transition.UnachievableLensSlots {
			commands = append(commands, slot.Withdraw.Command)
		}
	}
	return commands
}

// validateReviewDialectCommands refuses a result that names the tool of the
// other dialect: every command line must use the tool of the dialect the
// result's own contract declares, so a gentle-ai result never publishes an
// axiom command and an axiom result never publishes a gentle-ai one.
func validateReviewDialectCommands(contract string, commands ...string) error {
	dialect := reviewDialectForContract(contract)
	for _, command := range commands {
		if command != "" && !dialect.ownsCommand(command) {
			return fmt.Errorf("review command %q does not use the %s tool of contract %q", command, dialect.Tool(), contract) // refusal:by-design world-action: a producer must echo the dialect its contract declares; the exit is a code fix, not a command
		}
	}
	return nil
}

// validateReviewDialectCaptureTokens refuses a result whose capture tokens do
// not follow the dialect of the result's own contract: the argument list and the
// submission descriptor of every native capture input, every unachievable-slot
// withdraw command, and every acknowledgement lead with --contract exactly when
// the contract is axiom v2, so a gentle-ai result never hands a caller a token
// it must not replay and an axiom result never drops the one that keeps its
// dialect through the command it runs.
func validateReviewDialectCaptureTokens(contract string, transition *ReviewNextTransition, acknowledgements ...*ReviewTransitionExecution) error {
	dialect := reviewDialectForContract(contract)
	arguments := func(list []ReviewTransitionArgument) error {
		count, err := reviewCaptureContractArgumentCount(list)
		if err != nil {
			return err
		}
		if dialect.axiom != (count == 1) || count == 1 && list[0].Value != dialect.ContractV2() {
			return fmt.Errorf("capture token list does not follow the %s dialect of contract %q", dialect.Tool(), contract) // refusal:by-design world-action: a producer must echo the dialect its contract declares; the exit is a code fix, not a command
		}
		return nil
	}
	if transition != nil {
		if transition.Collect != nil {
			for _, input := range transition.Collect.Inputs {
				if _, native := reviewNativeCaptureVerb(input.CaptureOperation); !native {
					continue
				}
				if err := arguments(input.Arguments); err != nil {
					return err
				}
				if input.Submission == nil {
					continue
				}
				count, err := reviewCaptureContractTokenCount(input.Submission.ArgumentTokens)
				if err != nil {
					return err
				}
				if dialect.axiom != (count == 1) {
					return fmt.Errorf("capture submission descriptor does not follow the %s dialect of contract %q", dialect.Tool(), contract) // refusal:by-design world-action: a producer must echo the dialect its contract declares; the exit is a code fix, not a command
				}
			}
		}
		if transition.UnachievableLensSlots != nil {
			for _, slot := range *transition.UnachievableLensSlots {
				if err := arguments(slot.Withdraw.Arguments); err != nil {
					return err
				}
			}
		}
		acknowledgements = append(acknowledgements, transition.Execute)
	}
	for _, execution := range acknowledgements {
		if execution == nil || execution.Operation != "review.acknowledge-approved" {
			continue
		}
		if err := arguments(execution.Arguments); err != nil {
			return err
		}
	}
	return nil
}
