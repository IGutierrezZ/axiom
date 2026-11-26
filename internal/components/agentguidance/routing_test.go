package agentguidance

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/capabilitymanifest"
	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// supportedAgentCount guards the catalog itself: routing is unconditional for
// every supported adapter, so a silently shrinking catalog must fail here
// instead of quietly reducing coverage of the table-driven tests below.
const supportedAgentCount = 11

// retiredRemoteControlPlaneVocabulary is the wire and ceremony vocabulary the
// organic routing projection must never carry. Rendering any of these would
// reintroduce the removed remote control plane into always-on adapter guidance.
var retiredRemoteControlPlaneVocabulary = []string{
	"work-capabilities",
	"work-start",
	"work-advance",
	"work-route",
	"work-status",
	"work-transition",
	"work-reconcile",
	"work-verification-decide",
	"workrun",
	"connector",
	"GENTLE_AI_PRODUCTIVE_RUNTIME",
	"dormant",
	"advertised",
	"exposure",
	"authorizedTransition",
	"expected-revision",
	"--contract",
	"http://",
	"https://",
	"token",
	"certificate authority",
	"daemon",
}

func TestRenderRoutingSucceedsForEverySupportedAgent(t *testing.T) {
	t.Parallel()

	all := catalog.AllAgents()
	if len(all) != supportedAgentCount {
		t.Fatalf("catalog.AllAgents() returned %d agents, want %d", len(all), supportedAgentCount)
	}

	routing := capabilitymanifest.CanonicalImplementationRouting()

	for _, agent := range all {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			if strings.TrimSpace(rendered) == "" {
				t.Fatalf("RenderRouting(%q) returned blank guidance", agent.ID)
			}

			for _, want := range []string{
				"Direct inline",
				"Delegated direct",
				"Optional SDD",
			} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("RenderRouting(%q) is missing route %q:\n%s", agent.ID, want, rendered)
				}
			}

			// The rendered thresholds must come from the canonical manifest, not
			// from prose invented by the renderer.
			for _, want := range []string{
				fmt.Sprintf("at most %d calls and approximately %dk tokens", routing.DirectInline.MaxEvidenceCalls, routing.DirectInline.ApproxEvidenceTokens/1000),
				fmt.Sprintf("about %d changed lines or less", routing.DirectInline.MaxInlineUnitLines),
				fmt.Sprintf("at most approximately %dk tokens", routing.DelegatedDirect.ApproxHandoffTokens/1000),
			} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("RenderRouting(%q) is missing canonical threshold %q:\n%s", agent.ID, want, rendered)
				}
			}
		})
	}
}

func TestRenderRoutingKeepsSDDSelectionExplicit(t *testing.T) {
	t.Parallel()

	rendered, err := RenderRouting(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("RenderRouting error = %v", err)
	}

	lowered := strings.ToLower(rendered)
	for _, want := range []string{
		"explicit request",
		"accepted proposal",
		"project lane",
		"workflow",
	} {
		if !strings.Contains(lowered, want) {
			t.Fatalf("rendered routing does not require %q:\n%s", want, rendered)
		}
	}
	if !strings.Contains(lowered, "never select") {
		t.Fatalf("rendered routing does not state that size or risk alone never selects SDD:\n%s", rendered)
	}
}

func TestRenderRoutingAuthorizesOutcomesBeforeSelectingTopology(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			guard := "First establish whether the requested outcome explicitly authorizes a change."
			guardOffset := strings.Index(rendered, guard)
			topologyOffset := strings.Index(rendered, "**Direct inline:**")
			if guardOffset < 0 || topologyOffset < 0 || guardOffset > topologyOffset {
				t.Fatalf("RenderRouting(%q) must place outcome authorization before topology:\n%s", agent.ID, rendered)
			}

			for _, want := range []string{
				"Investigation, explanation, review, audit, comparison, and solution-proposal or planning-only requests are read-only",
				"must not write or edit files, delegate a writer, invoke apply, or create implementation artifacts",
				"If change intent is ambiguous or conditional, ask one clarification and remain read-only until answered.",
				"After explicit change intent is established",
				"SDD is selected only by an explicit request, by the project lane `workflow.lane: sdd` for substantial work, by an ISO project (`iso.project: iso`) for all work, or by an accepted proposal.",
				"Automatic SDD pace is not mutation authorization",
			} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("RenderRouting(%q) is missing outcome-authorization clause %q:\n%s", agent.ID, want, rendered)
				}
			}
		})
	}
}

// TestRenderRoutingProjectsODDProtocolBeforeTopology guards INC-20 F6.1: the
// Organic Driven Development protocol absorbed from upstream (1b202d77,
// 70c774f8, cfc415ce, dcd2fa07) must be projected as the orchestrator's
// predefined workflow, rendered before the direct/delegated/SDD topology so
// no adapter reads the topology as the entry point into implementation.
func TestRenderRoutingProjectsODDProtocolBeforeTopology(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			oddOffset := strings.Index(rendered, "### ODD protocol")
			topologyOffset := strings.Index(rendered, "**Direct inline:**")
			if oddOffset < 0 || topologyOffset < 0 || oddOffset > topologyOffset {
				t.Fatalf("RenderRouting(%q) must project the ODD protocol before implementation topology:\n%s", agent.ID, rendered)
			}

			for _, want := range []string{
				"Organic Driven Development (ODD) is the predefined workflow of this orchestrator.",
				"SDD is a branch inside ODD, entered only by an explicit request, by the project lane declared in `axiom.yaml` (including an ISO project, `iso.project: iso`), or by an accepted proposal.",
				"1. **Authorize.**",
				"2. **Explore.**",
				"3. **Resolve uncertainty.**",
				"4. **Classify.**",
				"5. **Track before the first write.**",
				"6. **Implement task by task.**",
				"7. **Close.**",
				"`odd/tasks/<feature-name>.md`",
				"`odd/<feature-name>/tasks`",
				"work-unit commit",
				"Conventional Commit",
			} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("RenderRouting(%q) is missing ODD protocol fact %q:\n%s", agent.ID, want, rendered)
				}
			}
		})
	}
}

// TestRenderRoutingClosesODDWithTheMaintainedTechnicalContext guards
// organic-agent-trigger-rules REQ-24.58, as revised by C-04-b (decision P6): ODD
// reads the maintained technical context while exploring (step 2) and updates it
// when it closes (step 7). Each step carries one deterministic English sentence,
// rendered once, inside its own step, identical in two renders and for every
// adapter. The locators come from `axiom workspace context --cwd <repo> --json`,
// never from `axiom sdd status --json`. The inc-24 rule that forbade mentioning
// the technical context outside step 7 is deliberately revised: steps 0, 1, 3, 4,
// 5 and 6 must still not mention it.
func TestRenderRoutingClosesODDWithTheMaintainedTechnicalContext(t *testing.T) {
	t.Parallel()

	const exploreSentence = "When the project declares a maintained technical context, read its entry before proposing or writing: get the locators from `axiom workspace context --cwd <repo> --json`, open `technicalContext.maintained.entry` first and consult the remaining `technicalContext.maintained.paths` only on demand; without a declared maintained context this step reads nothing more."
	const sentence = "When the project declares a maintained technical context, a work unit that changes something that context documents updates the affected documentation in that same work-unit commit; documentation declared as reference (`technical_context.reference`) is never modified, and without a maintained context this adds no requirement. The locators of that maintained context are the ones read in step 2, from the output of `axiom workspace context --cwd <repo> --json`."

	reference, err := RenderRouting(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("RenderRouting(%q) error = %v", model.AgentClaudeCode, err)
	}

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			first, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			second, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("second RenderRouting(%q) error = %v", agent.ID, err)
			}
			if first != second {
				t.Fatalf("RenderRouting(%q) is not identical across two renders", agent.ID)
			}
			if first != reference {
				t.Fatalf("RenderRouting(%q) differs from the reference adapter: the sentence must not depend on the agent", agent.ID)
			}

			if got := strings.Count(first, sentence); got != 1 {
				t.Fatalf("RenderRouting(%q) renders the maintained-context sentence %d times, want exactly once:\n%s", agent.ID, got, first)
			}

			// The sentence belongs to step 7 ("Close."), after the existing text of
			// that step and before the paragraph that follows the protocol.
			closeOffset := strings.Index(first, "7. **Close.**")
			sentenceOffset := strings.Index(first, sentence)
			resumeOffset := strings.Index(first, "Resume an interrupted feature with")
			if closeOffset < 0 || sentenceOffset < closeOffset || resumeOffset < sentenceOffset {
				t.Fatalf("RenderRouting(%q) must place the sentence inside step 7 (close=%d sentence=%d resume=%d):\n%s", agent.ID, closeOffset, sentenceOffset, resumeOffset, first)
			}
			closeStep := first[closeOffset:resumeOffset]
			if strings.Count(closeStep, "\n") > 2 {
				t.Fatalf("RenderRouting(%q) step 7 must stay a single paragraph, got:\n%q", agent.ID, closeStep)
			}
			for _, kept := range []string{
				"Report the verified outcome, every failed, skipped, or pending check, and the next step.",
			} {
				if !strings.Contains(closeStep, kept) {
					t.Fatalf("RenderRouting(%q) step 7 lost its existing text %q:\n%s", agent.ID, kept, closeStep)
				}
			}

			// Step 2 carries the read sentence once, inside step 2 only.
			if got := strings.Count(first, exploreSentence); got != 1 {
				t.Fatalf("RenderRouting(%q) renders the step-2 sentence %d times, want exactly once", agent.ID, got)
			}
			exploreOffset := strings.Index(first, "2. **Explore.**")
			exploreEnd := exploreOffset + strings.Index(first[exploreOffset:], "\n")
			if at := strings.Index(first, exploreSentence); exploreOffset < 0 || at < exploreOffset || at+len(exploreSentence) > exploreEnd {
				t.Fatalf("RenderRouting(%q) must place the read sentence inside step 2", agent.ID)
			}
			if !strings.Contains(first[exploreOffset:exploreEnd], "Explore the existing code and requirements first, proportionately to the request, before proposing or writing anything.") {
				t.Fatalf("RenderRouting(%q) step 2 lost its existing sentence", agent.ID)
			}
			// The technical-context locator no longer comes from the SDD status.
			if strings.Contains(first, "technicalContext.maintained.paths` in the output of `axiom sdd status") {
				t.Fatalf("RenderRouting(%q) must not name `axiom sdd status` as the technical-context locator", agent.ID)
			}
			if strings.Contains(closeStep, "axiom sdd status") || strings.Contains(closeStep, "in `axiom.yaml`") {
				t.Fatalf("RenderRouting(%q) step 7 must point to the workspace command only:\n%s", agent.ID, closeStep)
			}

			// The other ODD steps stay untouched, and the SDD lane gains nothing.
			for _, step := range []string{"0. **Evaluate scope and lane.**", "1. **Authorize.**", "3. **Resolve uncertainty.**", "4. **Classify.**", "5. **Track before the first write.**", "6. **Implement task by task.**"} {
				offset := strings.Index(first, step)
				if offset < 0 {
					t.Fatalf("RenderRouting(%q) lost step %q", agent.ID, step)
				}
				end := strings.Index(first[offset:], "\n")
				if strings.Contains(first[offset:offset+end], "technical context") {
					t.Fatalf("RenderRouting(%q) step %q must not mention the technical context", agent.ID, step)
				}
			}
			if strings.Count(first, "technical_context") != 1 {
				t.Fatalf("RenderRouting(%q) must mention technical_context only inside the step-7 sentence", agent.ID)
			}
		})
	}
}

// worktreePrimitiveBanLine is the single line of the routing doctrine that names
// the two runtime-specific worktree primitives (REQ-25.41). It is the only place
// in the product that may contain those names.
const worktreePrimitiveBanLine = "Never use `EnterWorktree` or `ExitWorktree`; work in another worktree only through the absolute paths that `axiom branch ensure` returns, with `--cwd`, `git -C`, and absolute paths."

// routingLineStartingWith returns the full rendered line that starts with prefix.
func routingLineStartingWith(t *testing.T, rendered, prefix string) string {
	t.Helper()

	for _, line := range strings.Split(rendered, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	t.Fatalf("rendered routing has no line starting with %q:\n%s", prefix, rendered)
	return ""
}

// routingSection returns the body of the "### <heading>" subsection, up to the
// next heading of the same or a higher level.
func routingSection(t *testing.T, rendered, heading string) string {
	t.Helper()

	marker := "### " + heading + "\n"
	start := strings.Index(rendered, marker)
	if start < 0 {
		t.Fatalf("rendered routing has no %q subsection:\n%s", "### "+heading, rendered)
	}
	body := rendered[start+len(marker):]
	if end := strings.Index(body, "\n### "); end >= 0 {
		body = body[:end]
	}
	return body
}

// TestRenderRoutingPreparesTheBranchWithAxiomBranchEnsure guards
// organic-agent-trigger-rules REQ-25.40 (inc-25, design D-19): step 6 only
// swaps its branch clause for the one that delegates to `axiom branch ensure`,
// every other step keeps its text, and the clause is the same for every adapter.
func TestRenderRoutingPreparesTheBranchWithAxiomBranchEnsure(t *testing.T) {
	t.Parallel()

	const (
		newClause    = "on the branch that `axiom branch ensure --lane odd --change <feature-name>` prepared before the first write of `odd/tasks/<feature-name>.md` (see *Branches and worktrees*)"
		retiredText  = "branch first when on the default branch"
		retiredShort = "on the feature branch"
	)

	reference, err := RenderRouting(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("RenderRouting(%q) error = %v", model.AgentClaudeCode, err)
	}
	referenceStep := routingLineStartingWith(t, reference, "6. **Implement task by task.**")

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			if strings.Contains(rendered, retiredText) {
				t.Fatalf("RenderRouting(%q) still carries the retired clause %q", agent.ID, retiredText)
			}

			step := routingLineStartingWith(t, rendered, "6. **Implement task by task.**")
			if step != referenceStep {
				t.Fatalf("RenderRouting(%q) step 6 differs from the reference adapter:\n got: %q\nwant: %q", agent.ID, step, referenceStep)
			}
			if strings.Count(step, newClause) != 1 {
				t.Fatalf("RenderRouting(%q) step 6 must carry the ensure clause exactly once:\n%s", agent.ID, step)
			}
			if strings.Contains(step, "on the feature branch, ") {
				t.Fatalf("RenderRouting(%q) step 6 keeps the retired %q clause:\n%s", agent.ID, retiredShort, step)
			}
			// The rest of the step keeps its existing text around the clause.
			for _, kept := range []string{
				"Route each task through the smallest useful topology below, honoring its delegation triggers when their conditions truly hold, with the configured TDD mode and applicable checks.",
				"Every task closes with at least one work-unit commit " + newClause + ", with tests and docs alongside the behavior, using a Conventional Commit message;",
				"push, pull request creation, and merge remain the user's decisions under ordinary repository policy.",
			} {
				if !strings.Contains(step, kept) {
					t.Fatalf("RenderRouting(%q) step 6 lost its existing text %q:\n%s", agent.ID, kept, step)
				}
			}

			// The branch rule belongs to step 6 only; the other steps never name it.
			for _, other := range []string{"0. **Evaluate scope and lane.**", "1. **Authorize.**", "2. **Explore.**", "3. **Resolve uncertainty.**", "4. **Classify.**", "5. **Track before the first write.**", "7. **Close.**"} {
				if strings.Contains(routingLineStartingWith(t, rendered, other), "axiom branch") {
					t.Fatalf("RenderRouting(%q) step %q must not mention axiom branch", agent.ID, other)
				}
			}
		})
	}
}

// TestRenderRoutingProjectsTheBranchesAndWorktreesSubsection guards REQ-25.40
// and REQ-25.41: the subsection is identical for every adapter, names the
// mechanism (JSON paths with --cwd and git -C), keeps decision_required as the
// only interaction, never authorizes --fetch, and carries the one line that
// bans the runtime-specific worktree primitives.
func TestRenderRoutingProjectsTheBranchesAndWorktreesSubsection(t *testing.T) {
	t.Parallel()

	reference, err := RenderRouting(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("RenderRouting(%q) error = %v", model.AgentClaudeCode, err)
	}
	referenceSection := routingSection(t, reference, "Branches and worktrees")

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			if got := strings.Count(rendered, "### Branches and worktrees\n"); got != 1 {
				t.Fatalf("RenderRouting(%q) renders the subsection %d times, want exactly once", agent.ID, got)
			}
			section := routingSection(t, rendered, "Branches and worktrees")
			if section != referenceSection {
				t.Fatalf("RenderRouting(%q) subsection differs from the reference adapter:\n got: %q\nwant: %q", agent.ID, section, referenceSection)
			}

			// It sits after the ODD protocol and its reference detail, so no adapter
			// reads it as an SDD-only rule.
			detail := strings.Index(rendered, "### Organic Driven Development\n")
			branches := strings.Index(rendered, "### Branches and worktrees\n")
			if detail < 0 || branches < detail {
				t.Fatalf("RenderRouting(%q) subsection order is wrong (detail=%d branches=%d)", agent.ID, detail, branches)
			}

			for _, want := range []string{
				"`axiom branch ensure --lane odd --change <feature-name> --json`",
				"Before the first write of `odd/tasks/<feature-name>.md` or of any code, run `axiom branch ensure",
				"absolute paths",
				"`--cwd`",
				"`git -C`",
				"`decision_required`",
				"the only interaction",
				"Never pass `--fetch` without the explicit authorization of the user.",
			} {
				if !strings.Contains(section, want) {
					t.Fatalf("RenderRouting(%q) subsection is missing %q:\n%s", agent.ID, want, section)
				}
			}

			// The ask mode adds no governance question of its own.
			if !strings.Contains(section, "do not add a governance question") {
				t.Fatalf("RenderRouting(%q) subsection must forbid an extra governance question:\n%s", agent.ID, section)
			}

			// The ban is one line, present once, and the only place the two
			// primitive names appear in the whole rendered doctrine.
			if got := strings.Count(rendered, worktreePrimitiveBanLine+"\n"); got != 1 {
				t.Fatalf("RenderRouting(%q) renders the ban line %d times, want exactly once:\n%s", agent.ID, got, rendered)
			}
			for _, name := range []string{"EnterWorktree", "ExitWorktree"} {
				if got := strings.Count(rendered, name); got != 1 {
					t.Fatalf("RenderRouting(%q) names %s %d times, want exactly once (in the ban line)", agent.ID, name, got)
				}
			}
		})
	}
}

// TestRenderRoutingBranchGuidanceIsDeterministic guards the REQ-25.40 scenario
// "El renderizado sigue siendo determinista" for the new text.
func TestRenderRoutingBranchGuidanceIsDeterministic(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		first, err := RenderRouting(agent.ID)
		if err != nil {
			t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
		}
		second, err := RenderRouting(agent.ID)
		if err != nil {
			t.Fatalf("RenderRouting(%q) second call error = %v", agent.ID, err)
		}
		if routingSection(t, first, "Branches and worktrees") != routingSection(t, second, "Branches and worktrees") {
			t.Fatalf("RenderRouting(%q) renders a different branch subsection on a second call", agent.ID)
		}
	}
}

// TestRenderRoutingAsksLaneSelectionBeforeAuthorize guards INC-21's Step 0
// (REQ-21.1-21.3, design.md S4.7/H-2): the ODD/SDD lane-selection blocking
// question, and ODD's own "no additional questions" instruction (REQ-21.2),
// must both render BEFORE "1. **Authorize.**" so no adapter reads the
// existing seven-step protocol as the entry point into a request whose lane
// has not been decided yet.
func TestRenderRoutingAsksLaneSelectionBeforeAuthorize(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			authorizeOffset := strings.Index(rendered, "1. **Authorize.**")
			if authorizeOffset < 0 {
				t.Fatalf("RenderRouting(%q) is missing the existing \"1. **Authorize.**\" step:\n%s", agent.ID, rendered)
			}

			for _, want := range []string{
				"0. **Evaluate scope and lane.**",
				"agile ODD lane or the formal SDD lane",
				"STOP",
				"unambiguously architectural",
				"enter the SDD pre-flight questionnaire directly",
				"do not ask any further governance question",
			} {
				offset := strings.Index(rendered, want)
				if offset < 0 {
					t.Fatalf("RenderRouting(%q) is missing Step 0 clause %q:\n%s", agent.ID, want, rendered)
				}
				if offset > authorizeOffset {
					t.Fatalf("RenderRouting(%q) renders Step 0 clause %q AFTER \"1. **Authorize.**\", want before:\n%s", agent.ID, want, rendered)
				}
			}
		})
	}
}

// TestRenderRoutingRendersMandatoryDelegationTriggers guards the upstream
// dcd2fa07 absorption: delegation triggers must be behavioral (rendered as
// mandatory stop-and-delegate rules keyed to the canonical manifest
// thresholds), not left as the advisory "smallest useful topology" framing
// alone.
func TestRenderRoutingRendersMandatoryDelegationTriggers(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			for _, want := range []string{
				"### Mandatory Delegation Triggers",
				"holds only when its condition truly holds, and only then is it mandatory, not advisory",
				"when in doubt work inline",
				"**Mapping trigger:**",
				"**Writer trigger:**",
				"only for a named reason",
				"Never delegate a writer for size, file count, or because the task is large",
				"**Proportionality:**",
				"**Long-session backstop:**",
				"advisory guidance, not a mechanically observed or enforced limit",
				"Long-session backstop",
				"Route declaration",
			} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("RenderRouting(%q) is missing delegation trigger %q:\n%s", agent.ID, want, rendered)
				}
			}
		})
	}
}

// TestRenderRoutingOmitsRDDGuidance guards the suspension of RDD (C-01-c): no
// configured agent is told about receipt-driven development, the review switch,
// or the native review lifecycle.
func TestRenderRoutingOmitsRDDGuidance(t *testing.T) {
	t.Parallel()

	all := catalog.AllAgents()
	if len(all) != supportedAgentCount {
		t.Fatalf("catalog.AllAgents() returned %d agents, want %d", len(all), supportedAgentCount)
	}

	for _, agent := range all {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			for _, forbidden := range []string{
				"receipt-driven",
				"RDD",
				"axiom review",
				"native review",
				"review actor",
				"review-integration",
				"user-owned",
				"disabled/unmanaged",
			} {
				if strings.Contains(rendered, forbidden) {
					t.Fatalf("RenderRouting(%q) still projects RDD guidance %q", agent.ID, forbidden)
				}
			}
			if !strings.Contains(rendered, "Preserve applicable functional verification.") {
				t.Fatalf("RenderRouting(%q) lost the functional verification bullet", agent.ID)
			}
		})
	}
}

func TestRenderRoutingOmitsRetiredRemoteControlPlaneVocabulary(t *testing.T) {
	t.Parallel()

	for _, forbidden := range retiredRemoteControlPlaneVocabulary {
		t.Run(forbidden, func(t *testing.T) {
			t.Parallel()

			needle := strings.ToLower(forbidden)
			for _, agent := range catalog.AllAgents() {
				rendered, err := RenderRouting(agent.ID)
				if err != nil {
					t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
				}
				if strings.Contains(strings.ToLower(routingVocabularyForGuard(rendered, forbidden)), needle) {
					t.Fatalf("RenderRouting(%q) leaks retired vocabulary %q:\n%s", agent.ID, forbidden, rendered)
				}
			}
		})
	}
}

func TestRenderRoutingIsSemanticallyEqualAcrossAgents(t *testing.T) {
	t.Parallel()

	var (
		referenceAgent model.AgentID
		reference      []string
	)

	for _, agent := range catalog.AllAgents() {
		rendered, err := RenderRouting(agent.ID)
		if err != nil {
			t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
		}

		semantics := routingSemantics(rendered)
		if len(semantics) == 0 {
			t.Fatalf("RenderRouting(%q) carries no routing semantics", agent.ID)
		}

		if reference == nil {
			referenceAgent = agent.ID
			reference = semantics
			continue
		}
		if len(semantics) != len(reference) {
			t.Fatalf("agent %q renders %d routing facts, agent %q renders %d",
				agent.ID, len(semantics), referenceAgent, len(reference))
		}
		for i := range semantics {
			if semantics[i] != reference[i] {
				t.Fatalf("agent %q drifted from %q:\n got: %q\nwant: %q",
					agent.ID, referenceAgent, semantics[i], reference[i])
			}
		}
	}
}

func TestRenderRoutingIsDeterministic(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		first, err := RenderRouting(agent.ID)
		if err != nil {
			t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
		}
		second, err := RenderRouting(agent.ID)
		if err != nil {
			t.Fatalf("RenderRouting(%q) second call error = %v", agent.ID, err)
		}
		if first != second {
			t.Fatalf("RenderRouting(%q) is not deterministic", agent.ID)
		}
	}
}

func TestRenderRoutingRejectsUnregisteredAgent(t *testing.T) {
	t.Parallel()

	rendered, err := RenderRouting(model.AgentID("totally-unregistered-agent"))
	if err == nil {
		t.Fatalf("RenderRouting accepted an unregistered agent and rendered:\n%s", rendered)
	}
	if !errors.Is(err, capabilitymanifest.ErrUnsupportedAgent) {
		t.Fatalf("RenderRouting error = %v, want it to wrap ErrUnsupportedAgent", err)
	}
	if rendered != "" {
		t.Fatalf("RenderRouting invented guidance for an unregistered agent:\n%s", rendered)
	}
}

// routingSemantics strips markdown formatting so two adapters may present the
// same routing facts differently without the parity check reading formatting
// as a semantic difference.
func routingSemantics(rendered string) []string {
	var facts []string
	for _, line := range strings.Split(rendered, "\n") {
		stripped := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, line)
		stripped = strings.Join(strings.Fields(stripped), " ")
		if stripped != "" {
			facts = append(facts, stripped)
		}
	}
	return facts
}

// evidenceSize matches the token sizes of the evidence budget ("10k tokens").
// They are sizes, not the authorization tokens the retired-vocabulary guard
// exists to keep out of the routing block.
var evidenceSize = regexp.MustCompile(`\d+k (?:parent-context )?tokens`)

// routingBlockWithoutEvidenceSizes drops the evidence-budget sizes so the
// guard for the word "token" keeps meaning authorization tokens only.
func routingBlockWithoutEvidenceSizes(block string) string {
	return evidenceSize.ReplaceAllString(block, "")
}

// routingVocabularyForGuard returns the text a retired-vocabulary needle is
// matched against.
func routingVocabularyForGuard(block, forbidden string) string {
	if forbidden == "token" {
		return routingBlockWithoutEvidenceSizes(block)
	}
	return block
}

// TestRenderRoutingClosesODDWithTheLivingSpecIntegration guards
// organic-agent-trigger-rules REQ-24.58 (part 2) and REQ-30.20
// (spec-viva-openspec): the ODD close step tells the agent where the OpenSpec
// delta goes (`## Spec viva`), what a bug declares when it has no delta (the
// exact `Sin impacto en la spec` line plus `Motivo:`, the same contract as the
// SDD lane, with no `## Informe de bug`), and to run
// `axiom archive integrate --odd <feature>` in the work-unit commit that closes
// the feature, with no stop. It stays one paragraph inside step 7, renders once,
// is identical for every adapter and names no lane value or ISO condition.
func TestRenderRoutingClosesODDWithTheLivingSpecIntegration(t *testing.T) {
	t.Parallel()

	reference, err := RenderRouting(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("RenderRouting(%q) error = %v", model.AgentClaudeCode, err)
	}

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			if rendered != reference {
				t.Fatalf("RenderRouting(%q) differs from the reference adapter: the living-spec rule must not depend on the agent", agent.ID)
			}

			closeOffset := strings.Index(rendered, "7. **Close.**")
			resumeOffset := strings.Index(rendered, "Resume an interrupted feature with")
			if closeOffset < 0 || resumeOffset < closeOffset {
				t.Fatalf("RenderRouting(%q) lost step 7 (close=%d resume=%d)", agent.ID, closeOffset, resumeOffset)
			}
			closeStep := rendered[closeOffset:resumeOffset]
			if strings.Count(closeStep, "\n") > 2 {
				t.Fatalf("RenderRouting(%q) step 7 must stay a single paragraph, got:\n%q", agent.ID, closeStep)
			}

			for _, want := range []string{
				"`## Spec viva`",
				"`odd/tasks/<feature>.md`",
				"`### Capacidad: <domain>`",
				"`openspec`",
				"`Tipo: bug`",
				"`## Impacto en la spec`",
				"the exact line `Sin impacto en la spec`",
				"`Motivo: <one line>`",
				"ignored",
				"`## Informe de bug`",
				"every new requirement in an ADDED section carries a provisional identifier instead of a number picked by hand",
				"`axiom archive integrate --odd <feature>`",
				"it assigns the number of each provisional identifier and rewrites it in the ODD document",
				"work-unit commit that closes the feature",
				"`openspec/specs/`",
			} {
				if !strings.Contains(closeStep, want) {
					t.Fatalf("RenderRouting(%q) step 7 must contain %q:\n%s", agent.ID, want, closeStep)
				}
			}
			if got := strings.Count(rendered, "axiom archive integrate --odd <feature>"); got != 1 {
				t.Fatalf("RenderRouting(%q) names the integration command %d times, want exactly once", agent.ID, got)
			}

			// ODD has no stop (REQ-27.17), and the rule carries no lane value or ISO condition.
			for _, forbidden := range []string{"three exits", "Plan stop", "plan stop"} {
				if strings.Contains(closeStep, forbidden) {
					t.Fatalf("RenderRouting(%q) step 7 must not carry %q", agent.ID, forbidden)
				}
			}
			for _, forbidden := range []string{"ISO", "iso ", "kickoff"} {
				if strings.Contains(closeStep, forbidden) {
					t.Fatalf("RenderRouting(%q) step 7 must not carry the lane value or ISO condition (%q):\n%s", agent.ID, forbidden, closeStep)
				}
			}

			// The first sentence keeps its exact text (pinned by the maintained-context test).
			if !strings.Contains(closeStep, "without a maintained context this adds no requirement.") {
				t.Fatalf("RenderRouting(%q) step 7 lost the maintained-context sentence", agent.ID)
			}
		})
	}
}

// TestRenderRoutingClosesODDWithTheMaintainedContextLocator guards
// organic-agent-trigger-rules REQ-24.58 (part 1, spec-viva-openspec): the ODD
// explore step (2) tells the agent where the maintained technical context is
// located (`technicalContext.maintained.entry` and `.paths` of `axiom workspace
// context --cwd <repo> --json`, no longer `axiom sdd status --json`, C-04-b) and
// to open `entry` first; the close step (7) points back to that same command
// after the maintained-context sentence, inside its single paragraph. Both render
// once, are identical for every adapter and neither authorizes touching reference
// documentation nor requires anything when no maintained context is declared.
func TestRenderRoutingClosesODDWithTheMaintainedContextLocator(t *testing.T) {
	t.Parallel()

	const maintained = "without a maintained context this adds no requirement."

	reference, err := RenderRouting(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("RenderRouting(%q) error = %v", model.AgentClaudeCode, err)
	}

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			if rendered != reference {
				t.Fatalf("RenderRouting(%q) differs from the reference adapter: the locator rule must not depend on the agent", agent.ID)
			}

			closeOffset := strings.Index(rendered, "7. **Close.**")
			resumeOffset := strings.Index(rendered, "Resume an interrupted feature with")
			if closeOffset < 0 || resumeOffset < closeOffset {
				t.Fatalf("RenderRouting(%q) lost step 7 (close=%d resume=%d)", agent.ID, closeOffset, resumeOffset)
			}
			closeStep := rendered[closeOffset:resumeOffset]
			if strings.Count(closeStep, "\n") > 2 {
				t.Fatalf("RenderRouting(%q) step 7 must stay a single paragraph, got:\n%q", agent.ID, closeStep)
			}

			exploreOffset := strings.Index(rendered, "2. **Explore.**")
			exploreStep := rendered[exploreOffset : exploreOffset+strings.Index(rendered[exploreOffset:], "\n")]
			for _, want := range []string{
				"`technicalContext.maintained.entry`",
				"`technicalContext.maintained.paths`",
				"`axiom workspace context --cwd <repo> --json`",
				"open `technicalContext.maintained.entry` first",
			} {
				if got := strings.Count(exploreStep, want); got != 1 {
					t.Fatalf("RenderRouting(%q) step 2 must contain %q exactly once, got %d:\n%s", agent.ID, want, got, exploreStep)
				}
			}
			if got := strings.Count(rendered, "`technicalContext.maintained.entry`"); got != 1 {
				t.Fatalf("RenderRouting(%q) renders the locator %d times, want exactly once", agent.ID, got)
			}
			if got := strings.Count(closeStep, "`axiom workspace context --cwd <repo> --json`"); got != 1 {
				t.Fatalf("RenderRouting(%q) step 7 must name the workspace command exactly once, got %d:\n%s", agent.ID, got, closeStep)
			}

			maintainedOffset := strings.Index(closeStep, maintained)
			locatorOffset := strings.Index(closeStep, "`axiom workspace context --cwd <repo> --json`")
			specOffset := strings.Index(closeStep, "`## Spec viva`")
			if maintainedOffset < 0 || locatorOffset < maintainedOffset || specOffset < locatorOffset {
				t.Fatalf("RenderRouting(%q) must place the locator after the maintained-context sentence and before the living-spec rule (maintained=%d locator=%d spec=%d)", agent.ID, maintainedOffset, locatorOffset, specOffset)
			}

			// The locator is read only when a maintained context exists and never
			// widens what may be edited.
			locatorSentence := exploreStep
			if strings.Contains(locatorSentence, "reference") {
				t.Fatalf("RenderRouting(%q) the locator sentence must not mention reference documentation: %q", agent.ID, locatorSentence)
			}
		})
	}
}

// TestRenderRoutingNotifiesPerCommitAndSharesTheCommitRules guards
// organic-agent-trigger-rules REQ-33.12 (C-02-d): ODD always runs in notify mode
// (one line per work-unit commit, no wait), carries the common commit and pull
// request rules in its own bullet, and projects none of the SDD-only commit review
// vocabulary nor the SDD gate vocabulary guarded by REQ-27.17.
func TestRenderRoutingNotifiesPerCommitAndSharesTheCommitRules(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			for _, want := range []string{
				"After each work-unit commit, tell the person in one line the short sha, the task ID and `git show <sha>`, then continue with the next task without waiting.",
				"- Commits and the pull request. Asking for the pull request validates every commit of the feature: there is no commit-by-commit approval and nothing blocks it.",
				"Record in a `## Commits con cambios pedidos` section of `odd/tasks/<feature>.md` only the commits the person asked to change, one line each, never a checkbox.",
				"park it with `git stash push --include-untracked` and restore it afterwards",
				"`git branch -r --contains <sha>`, never fetch to decide",
				"`git commit --fixup=<sha>`",
				"`git -c sequence.editor=true rebase -i --autosquash --autostash <base>` only when the pull request is requested",
				"add a new corrective commit and never rewrite published history",
				"Recommend merging with a merge commit, without squash.",
			} {
				if !strings.Contains(rendered, want) {
					t.Errorf("RenderRouting(%q) is missing the commit doctrine %q", agent.ID, want)
				}
			}

			// The notice closes step 6 and the bullet follows the delivery bullet.
			step6 := strings.Index(rendered, "6. **Implement task by task.**")
			notice := strings.Index(rendered, "After each work-unit commit, tell the person")
			step7 := strings.Index(rendered, "7. **Close.**")
			if step6 < 0 || notice < step6 || (step7 >= 0 && notice > step7) {
				t.Errorf("RenderRouting(%q) must place the commit notice inside step 6", agent.ID)
			}
			delivery := strings.Index(rendered, "- Delivery follows work units.")
			commits := strings.Index(rendered, "- Commits and the pull request.")
			if delivery < 0 || commits < delivery {
				t.Errorf("RenderRouting(%q) must place the commits bullet after the delivery bullet", agent.ID)
			}

			for _, forbidden := range []string{
				"`stop`",
				"stop-before-commit",
				"--commit-review",
				"commit review",
				"Commit Review",
				"three exits",
				"Plan stop",
				"plan stop",
			} {
				if strings.Contains(rendered, forbidden) {
					t.Errorf("RenderRouting(%q) leaks SDD-only vocabulary %q", agent.ID, forbidden)
				}
			}
		})
	}
}

// TestRenderRoutingDeliversInASinglePRByDefault guards organic-agent-trigger-rules
// REQ-32.4 (C-02-c): ODD ships a feature as `single-pr` unless the person asks to
// chain it, warns once and never blocks when the forecast exceeds
// `delivery.split_hint_lines`, names no line-count threshold, and no longer offers
// the retired delivery vocabulary or the size gates it relied on. The delivery
// bullet is self-contained so that a later change can extend its last sentence.
func TestRenderRoutingDeliversInASinglePRByDefault(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			for _, want := range []string{
				"Size each ODD task as the smallest coherent behavior with its tests and docs; there is no line-count target, cap, or forced split.",
				"Never delete spaces, blank lines, or comments for cosmetic line savings; never omit tests, minify, add gratuitous abstractions, or split artificially.",
				"Forward this same instruction when delegating tasks to subagents.",
				"Delivery follows work units. By default a feature ships as `single-pr`: one pull request holding one work-unit commit per task.",
				"At feature-document creation, forecast the approximate authored changed lines (additions plus deletions, generated files excluded) from the task list.",
				"When the project's `axiom.yaml` declares `delivery.split_hint_lines` and the forecast exceeds it, tell the person once for this feature that the change is large and that they can ask for chained pull requests, then continue without waiting; without that key, give no size warning and cite no line number.",
				"Use `chained` only when the person asks to chain, stack, or slice the pull requests",
				"ask once whether `stacked-to-main` or `feature-branch-chain`",
				"record slice boundaries, which commits each pull request holds, in the feature document.",
				"Resolve the `work-unit-commits` skill by registry name for every feature and the `chained-pr` skill only when chaining, never hardcoding their paths.",
			} {
				if !strings.Contains(rendered, want) {
					t.Errorf("RenderRouting(%q) is missing the delivery doctrine %q", agent.ID, want)
				}
			}

			for _, retired := range []string{
				"400",
				"ask-on-risk",
				"auto-chain",
				"exception-ok",
				"Existing PR size gates",
				"delivery budget",
				"Choose one delivery strategy per feature",
			} {
				if strings.Contains(rendered, retired) {
					t.Errorf("RenderRouting(%q) still carries the retired delivery text %q", agent.ID, retired)
				}
			}

			// The delivery bullet ends in a sentence a later change can extend.
			start := strings.Index(rendered, "- Delivery follows work units.")
			if start < 0 {
				t.Fatalf("RenderRouting(%q) lost the delivery bullet", agent.ID)
			}
			bullet := rendered[start:]
			if end := strings.Index(bullet, "\n"); end >= 0 {
				bullet = bullet[:end]
			}
			if !strings.HasSuffix(bullet, "never hardcoding their paths.") {
				t.Errorf("RenderRouting(%q) delivery bullet must end in a sentence that can be extended: %q", agent.ID, bullet)
			}
		})
	}
}
