package agentguidance

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/capabilitymanifest"
)

// This file is the single source of the delegation doctrine (decision P2 of
// guia-orquestador-upstream). RenderRouting renders it into the always-installed
// routing block, and the SDD orchestrator sections resolve their delegation
// markers with the very same functions, so the rule can never drift between the
// routing block, the twelve orchestrators and the migrated OpenCode prompt.
//
// The doctrine delegates by what the evidence costs, never by how many files it
// touches. Every number comes from capabilitymanifest.CanonicalImplementationRouting,
// whose digests pin it.

// DelegationRules returns the delegation rules of the SDD orchestrators: the
// topology preamble, the evidence budget and the inline-versus-worker table.
// The result carries no heading, so each consumer decides where it sits.
func DelegationRules() string {
	return renderDelegationRules(capabilitymanifest.CanonicalImplementationRouting())
}

// DelegationTriggers returns the conditional delegation triggers shared by the
// routing block, the SDD orchestrators and the migrated OpenCode prompt. The
// result carries no heading.
func DelegationTriggers() string {
	return renderDelegationTriggers(capabilitymanifest.CanonicalImplementationRouting())
}

func renderDelegationRules(routing capabilitymanifest.ImplementationRoutingFacts) string {
	var output strings.Builder
	output.WriteString("These rules select execution topology, not the implementation method. Crossing the evidence budget selects **delegated direct** work; it never selects SDD, creates SDD state, or invokes an `sdd-*` phase. Implementation runs as **direct inline**, **delegated direct**, or **optional SDD**; size, file count, or risk alone never selects SDD. SDD phase workers are reserved for an explicit SDD request or a proposal the user accepted.\n\n")
	output.WriteString("Core principle: **does this inflate the parent context without need?** If yes, use one bounded worker. If no, do it inline.\n\n")
	output.WriteString(evidenceBudgetSentence(routing))
	output.WriteString("\n\n")
	output.WriteString("| Action | Direct inline | Delegated direct worker |\n")
	output.WriteString("|--------|---------------|-------------------------|\n")
	output.WriteString("| Read to decide/verify (within the evidence budget) | ✅ | — |\n")
	output.WriteString("| Read to explore/understand (beyond the evidence budget) | — | ✅ one read-only explorer with a short handoff |\n")
	output.WriteString("| Read as preparation for writing | ✅ when the write is inline | ✅ together with the write, only when a writer is delegated |\n")
	_, _ = fmt.Fprintf(&output, "| Write a unit of about %d changed lines or less | ✅ always | — |\n", routing.DirectInline.MaxInlineUnitLines)
	output.WriteString("| Write a larger unit | ✅ by default | ✅ one bounded writer, only for a named reason: parallelism or protecting the parent context |\n")
	output.WriteString("| Bash for state (`git`, `gh`) | ✅ | — |\n")
	output.WriteString("| Tests, builds, installs, or native review actions | allowed as a bounded action | ✅ fresh per-action worker without changing route |")
	return output.String()
}

func renderDelegationTriggers(routing capabilitymanifest.ImplementationRoutingFacts) string {
	var output strings.Builder
	output.WriteString("Each trigger holds only when its condition truly holds, and only then is it mandatory, not advisory; no trigger holds by default, and when in doubt work inline. When one fires, stop and delegate through the runtime's subagent mechanism before continuing; executing past a fired trigger inline is a routing defect even if the work succeeds. Delegation keeps the parent context thin enough to orchestrate; it does not slow the work down. These are parent-orchestrator routing boundaries: do not pass them to child agents as permission to orchestrate.\n\n")
	_, _ = fmt.Fprintf(&output, "- **Mapping trigger:** when understanding exceeds the inline evidence budget (one parallel batch, at most %d calls, approximately %dk tokens), delegate one read-only explorer before deciding or writing anything. It returns a handoff of at most approximately %dk tokens with `path:line` evidence; the parent makes %d spot check at most and does not reread the whole mapped evidence. The number of files never fires this trigger, and never explore to read files you will read anyway before writing inline.\n",
		routing.DirectInline.MaxEvidenceCalls,
		routing.DirectInline.ApproxEvidenceTokens/1000,
		routing.DelegatedDirect.ApproxHandoffTokens/1000,
		routing.DelegatedDirect.MaxParentSpotChecks,
	)
	_, _ = fmt.Fprintf(&output, "- **Writer trigger:** a writer is delegated only for a named reason, one bounded writer per unit: (a) parallelism: two or more independent units of behavior with disjoint edit surfaces, each needing several minutes of work, launched together; (b) context: protecting the parent context under the long-session backstop below. A unit is one independent piece of behavior with its code, tests, and docs together: never split one behavior into test, docs, and code writers, and a delivery that depends on another is not independent. Units of about %d changed lines or less always run inline: writing them costs less than starting workers. Never delegate a writer for size, file count, or because the task is large; without a named reason the parent works inline.\n",
		routing.DirectInline.MaxInlineUnitLines,
	)
	output.WriteString("- **Preparation trigger:** when the Writer trigger delegates a write, the reading that prepares it, and broad research or context compression, delegate together with or ahead of the write instead of filling the parent context. Inline writes read inline.\n")
	output.WriteString("- **Per-action trigger:** tests, builds, installs, and native review actors may use fresh workers without changing the implementation route or creating SDD state.\n")
	output.WriteString("- **Optional SDD trigger:** propose SDD only when durable proposal/spec/design/tasks materially reduce substantial ambiguity. Select SDD only after an explicit request, the project lane `workflow.lane: sdd` for substantial work, an ISO project (`iso.project: iso`) for all work, or an accepted proposal; risk alone never forces SDD.\n")
	output.WriteString("- **Proportionality:** validate consequential premises against available evidence before acting on them; reuse relevant sibling investigation instead of repeating it; run focused checks while iterating and all applicable full checks when a task closes. Keep spend proportional without a hard spend or line gate.\n")
	_, _ = fmt.Fprintf(&output, "- **Long-session backstop:** at approximately %dk parent-context tokens, consider delegating the next bounded unit to protect the parent context. This is advisory guidance, not a mechanically observed or enforced limit: do not claim runtime telemetry, and it never fires on a count of calls, reads, edits, or files.\n",
		routing.DelegatedDirect.ApproxParentContextTokens/1000,
	)
	output.WriteString("- **Route declaration:** for substantial work, record the chosen route per task (inline or delegated) and the trigger evidence in the feature document, so skipped delegation is observable instead of silent.\n")
	output.WriteString("- These triggers only choose between direct inline and delegated direct; they never select SDD and never create SDD artifacts.")
	return output.String()
}

// evidenceBudgetSentence states the inline evidence budget once, for every
// consumer that needs the full sentence.
func evidenceBudgetSentence(routing capabilitymanifest.ImplementationRoutingFacts) string {
	return fmt.Sprintf(
		"Evidence budget: inline evidence is one parallel batch of at most %d calls and approximately %dk tokens, using bounded searches or line ranges rather than whole large files. When it does not fit, one read-only explorer returns a handoff of at most approximately %dk tokens with `path:line` evidence, and the parent makes %d spot check at most. The number of files never decides: file count never triggers delegation.",
		routing.DirectInline.MaxEvidenceCalls,
		routing.DirectInline.ApproxEvidenceTokens/1000,
		routing.DelegatedDirect.ApproxHandoffTokens/1000,
		routing.DelegatedDirect.MaxParentSpotChecks,
	)
}
