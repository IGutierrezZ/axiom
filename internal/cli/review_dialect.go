package cli

import (
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

// reviewNoContractDialect is the dialect of a continuation emitted by a command
// that carries no --contract of its own (capture-* and acknowledge-approved).
// It is axiom by default and is the single exception to the echo rule: those
// commands have nothing to echo, and the transition schemas admit both tools.
var reviewNoContractDialect = reviewDialect{axiom: true}

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
