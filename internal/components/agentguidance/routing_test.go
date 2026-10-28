package agentguidance

import (
	"errors"
	"fmt"
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
const supportedAgentCount = 16

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
				fmt.Sprintf("%d–%d files", routing.DirectInline.MinUnderstandingFiles, routing.DirectInline.MaxUnderstandingFiles),
				fmt.Sprintf("%d+ files", routing.DelegatedDirect.MappingMinUnderstandingFiles),
				fmt.Sprintf("%d+ non-trivial files", routing.DelegatedDirect.WriterMinNonTrivialFiles),
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
				"SDD is selected only by an explicit request or an accepted proposal.",
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
				"SDD is a branch inside ODD, entered only by an explicit request or an accepted proposal.",
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
// organic-agent-trigger-rules REQ-24.58 (inc-24, design D-19): the ODD close
// step carries one deterministic English sentence about the maintained
// technical context. It must sit inside step 7, render once, be identical in
// two renders and for every adapter, and leave every other step untouched.
func TestRenderRoutingClosesODDWithTheMaintainedTechnicalContext(t *testing.T) {
	t.Parallel()

	const sentence = "When the project declares a maintained technical context (`technical_context.maintained` in `axiom.yaml`), a work unit that changes something that context documents updates the affected documentation in that same work-unit commit; documentation declared as reference (`technical_context.reference`) is never modified, and without a maintained context this adds no requirement."

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
				"native review runs only under the user-owned receipt-driven development switch.",
			} {
				if !strings.Contains(closeStep, kept) {
					t.Fatalf("RenderRouting(%q) step 7 lost its existing text %q:\n%s", agent.ID, kept, closeStep)
				}
			}

			// No other ODD step changes, and the SDD lane gains nothing.
			for _, step := range []string{"0. **Evaluate scope and lane.**", "1. **Authorize.**", "2. **Explore.**", "3. **Resolve uncertainty.**", "4. **Classify.**", "5. **Track before the first write.**", "6. **Implement task by task.**"} {
				offset := strings.Index(first, step)
				if offset < 0 {
					t.Fatalf("RenderRouting(%q) lost step %q", agent.ID, step)
				}
				end := strings.Index(first[offset:], "\n")
				if strings.Contains(first[offset:offset+end], "technical context") {
					t.Fatalf("RenderRouting(%q) step %q must not mention the technical context", agent.ID, step)
				}
			}
			if strings.Count(first, "technical_context") != 2 {
				t.Fatalf("RenderRouting(%q) must mention technical_context only inside the new sentence", agent.ID)
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
				"Route each task through the smallest useful topology below, honoring its mandatory delegation triggers, with the configured TDD mode and applicable checks.",
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

			// It sits after the ODD protocol and its reference detail, ahead of the
			// receipt-driven switch, so no adapter reads it as an SDD-only rule.
			detail := strings.Index(rendered, "### Organic Driven Development\n")
			branches := strings.Index(rendered, "### Branches and worktrees\n")
			review := strings.Index(rendered, "### Receipt-driven development is user-owned\n")
			if detail < 0 || branches < detail || review < branches {
				t.Fatalf("RenderRouting(%q) subsection order is wrong (detail=%d branches=%d review=%d)", agent.ID, detail, branches, review)
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

	routing := capabilitymanifest.CanonicalImplementationRouting()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			for _, want := range []string{
				"### Mandatory Delegation Triggers",
				"These triggers are mandatory, not advisory.",
				fmt.Sprintf("%d or more files", routing.DelegatedDirect.MappingMinUnderstandingFiles),
				fmt.Sprintf("%d or more non-trivial files", routing.DelegatedDirect.WriterMinNonTrivialFiles),
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

// TestRenderRoutingMakesTheReviewKillSwitchDiscoverable guards the product
// promise that configuring an agent tells it what it may do. The kill switch is
// only real for the user if every configured agent can name it, so the exact
// command surface must be projected into the unconditional routing block.
func TestRenderRoutingMakesTheReviewKillSwitchDiscoverable(t *testing.T) {
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

			for _, want := range []string{
				"axiom review mode enable|disable|status",
				"`status` is read-only",
				"deciding source and the effective mode",
			} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("RenderRouting(%q) hides the kill switch, missing %q:\n%s", agent.ID, want, rendered)
				}
			}
		})
	}
}

// TestRenderRoutingObeysTheUserOnReviewMode asserts the specific instructional
// phrases, not merely the topic, so that prose drift which softens "run disable"
// into a negotiation, or which lets an agent switch review back on unbidden,
// fails here instead of shipping to every configured agent.
func TestRenderRoutingObeysTheUserOnReviewMode(t *testing.T) {
	t.Parallel()

	rendered, err := RenderRouting(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("RenderRouting error = %v", err)
	}

	for _, want := range []string{
		"When the user asks to stop using receipt-driven development, run `disable`.",
		"Do not argue, do not work around it, and do not propose alternatives first.",
		"Never enable receipt-driven development on the user's behalf unless the user explicitly asks for it.",
		"do not start reviews, do not retry, do not reactivate it, and do not fall back to any retired path",
		"reports `disabled/unmanaged`",
		"never a fabricated approval",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered routing is missing the exact instruction %q:\n%s", want, rendered)
		}
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
				if strings.Contains(strings.ToLower(rendered), needle) {
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
