package agentguidance

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// planStopLinePrefix opens the ODD detail bullet that defines the plan stop.
const planStopLinePrefix = "- Plan stop: "

// TestRenderRoutingProjectsStopsAndExitsFromTheSingleSource guards REQ-26.8 and
// REQ-26.10: every adapter renders the "Stops and exits" section exactly once,
// after "Branches and worktrees" and before the receipt-driven switch, and its
// body is the very text StopExits returns.
func TestRenderRoutingProjectsStopsAndExitsFromTheSingleSource(t *testing.T) {
	t.Parallel()

	want := StopExits()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			if got := strings.Count(rendered, "### Stops and exits\n"); got != 1 {
				t.Fatalf("RenderRouting(%q) renders the section %d times, want exactly once", agent.ID, got)
			}
			if got := strings.TrimSpace(routingSection(t, rendered, "Stops and exits")); got != want {
				t.Fatalf("RenderRouting(%q) section differs from StopExits():\n got: %q\nwant: %q", agent.ID, got, want)
			}
			if !strings.Contains(rendered, want) {
				t.Fatalf("RenderRouting(%q) does not contain StopExits() verbatim", agent.ID)
			}

			branches := strings.Index(rendered, "### Branches and worktrees\n")
			stops := strings.Index(rendered, "### Stops and exits\n")
			review := strings.Index(rendered, "### Receipt-driven development is user-owned\n")
			if branches < 0 || stops < branches || review < stops {
				t.Fatalf("RenderRouting(%q) section order is wrong (branches=%d stops=%d review=%d)", agent.ID, branches, stops, review)
			}
		})
	}
}

// TestRenderRoutingCitesThePlanStopAsAnException guards the step 0 and step 5
// scenarios of REQ-26.10: step 0 keeps "do not ask any further governance
// question" and says the single plan stop is not one; step 5 presents the plan
// stop without naming axiom branch, which belongs to step 6 alone.
func TestRenderRoutingCitesThePlanStopAsAnException(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}

			step0 := routingLineStartingWith(t, rendered, "0. **Evaluate scope and lane.**")
			for _, want := range []string{
				"do not ask any further governance question",
				"The single plan stop after step 5 (see *Stops and exits*) is not a governance question.",
			} {
				if !strings.Contains(step0, want) {
					t.Fatalf("RenderRouting(%q) step 0 is missing %q:\n%s", agent.ID, want, step0)
				}
			}

			step5 := routingLineStartingWith(t, rendered, "5. **Track before the first write.**")
			for _, want := range []string{
				"Then present the plan stop: the single ODD stop, after the feature document exists and before the first code write, with the three exits of *Stops and exits*.",
				"Tell the user in one line which feature document was created and how many tasks it holds.",
			} {
				if !strings.Contains(step5, want) {
					t.Fatalf("RenderRouting(%q) step 5 is missing %q:\n%s", agent.ID, want, step5)
				}
			}
			if strings.Contains(step5, "axiom branch") {
				t.Fatalf("RenderRouting(%q) step 5 names axiom branch, which only step 6 may do:\n%s", agent.ID, step5)
			}
		})
	}
}

// TestRenderRoutingDefinesThePlanStopInTheOrganicDetail pins the plan stop
// bullet: one stop with a feature document only, no SDD artifact, the
// `Plan stop:` line, the commit of the feature document alone, and the
// multirepo case.
func TestRenderRoutingDefinesThePlanStopInTheOrganicDetail(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			if got := strings.Count(rendered, planStopLinePrefix); got != 1 {
				t.Fatalf("RenderRouting(%q) renders the plan stop bullet %d times, want exactly once", agent.ID, got)
			}
			bullet := routingLineStartingWith(t, rendered, planStopLinePrefix)

			detail := strings.Index(rendered, "### Organic Driven Development\n")
			branches := strings.Index(rendered, "### Branches and worktrees\n")
			at := strings.Index(rendered, bullet)
			if detail < 0 || at < detail || branches < at {
				t.Fatalf("RenderRouting(%q) plan stop bullet sits outside the organic detail (detail=%d bullet=%d branches=%d)", agent.ID, detail, at, branches)
			}

			for _, want := range []string{
				"small, understood work has no stop",
				"It is not a governance question, creates no SDD artifact, and never creates `handoff.md`.",
				"`Plan stop: approved <YYYY-MM-DD>; implementation continues`",
				"`Plan stop: approved <YYYY-MM-DD>; implementation handed off`",
				"never present the stop again when the document already records it, for example on resume",
				"Iterate adjusts the file and the Engram mirror and presents the stop again.",
				"commits only `odd/tasks/<feature-name>.md`",
				"`git -C <path> add -- odd/tasks/<feature-name>.md`",
				"`git -C <path> commit -m \"<message>\" -- odd/tasks/<feature-name>.md`",
				"`axiom branch status --lane odd --change <feature-name> --json`",
				"presents the branch, the `prTarget`, the exact push command, and the pull request data (`head` and `base`) for the human to run, and stops without writing code",
				"In a `multirepo` the feature document lives in the specs repository on its integration branch: commit there and present the push for the human, with no pull request.",
			} {
				if !strings.Contains(bullet, want) {
					t.Fatalf("RenderRouting(%q) plan stop bullet is missing %q:\n%s", agent.ID, want, bullet)
				}
			}
		})
	}
}

// TestRenderRoutingResumesWithoutTheEngramMirror pins the pick-up scenario of
// REQ-26.10: the resume line starts from the task file, tolerates a missing
// mirror, never reviews an approved plan, and continues with the first
// unfinished task.
func TestRenderRoutingResumesWithoutTheEngramMirror(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			resume := routingLineStartingWith(t, rendered, "Resume an interrupted feature with")
			for _, want := range []string{
				"then the task file itself; reconcile before continuing the next unfinished task.",
				"When the Engram mirror does not exist (for example, the feature was handed off to you), start from the task file alone, never fail for the missing mirror, never review an approved plan again, and continue with the first unfinished task.",
			} {
				if !strings.Contains(resume, want) {
					t.Fatalf("RenderRouting(%q) resume line is missing %q:\n%s", agent.ID, want, resume)
				}
			}
		})
	}
}

// TestRenderRoutingNeverOrdersRemoteOperationsAtTheStop guards the last
// scenario of REQ-26.10: neither the stops section nor the plan stop bullet
// orders push, fetch, opening the pull request, or merging; they present them
// as the human's.
func TestRenderRoutingNeverOrdersRemoteOperationsAtTheStop(t *testing.T) {
	t.Parallel()

	rendered, err := RenderRouting(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("RenderRouting(%q) error = %v", model.AgentClaudeCode, err)
	}
	stop := routingSection(t, rendered, "Stops and exits") + "\n" + routingLineStartingWith(t, rendered, planStopLinePrefix)

	for _, forbidden := range []string{"git push", "git fetch", "git pull", "git merge", "gh pr", "glab ", "&&"} {
		if strings.Contains(stop, forbidden) {
			t.Fatalf("the stop text carries the executable remote command %q:\n%s", forbidden, stop)
		}
	}
	for _, want := range []string{
		"`push`, `fetch`, opening the pull request, and merging belong to the human",
		"never run them yourself",
		"for the human to run",
	} {
		if !strings.Contains(stop, want) {
			t.Fatalf("the stop text does not present the remote operations as the human's, missing %q:\n%s", want, stop)
		}
	}
}

// TestRenderRoutingStopsTextIsIdenticalAcrossAgentsAndRenders guards the
// determinism of the new text: the section and the plan stop bullet are the same
// for every adapter and on a second render.
func TestRenderRoutingStopsTextIsIdenticalAcrossAgentsAndRenders(t *testing.T) {
	t.Parallel()

	reference, err := RenderRouting(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("RenderRouting(%q) error = %v", model.AgentClaudeCode, err)
	}
	referenceSection := routingSection(t, reference, "Stops and exits")
	referenceBullet := routingLineStartingWith(t, reference, planStopLinePrefix)

	for _, agent := range catalog.AllAgents() {
		for render := 0; render < 2; render++ {
			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			if routingSection(t, rendered, "Stops and exits") != referenceSection {
				t.Fatalf("RenderRouting(%q) render %d differs in the stops section", agent.ID, render)
			}
			if routingLineStartingWith(t, rendered, planStopLinePrefix) != referenceBullet {
				t.Fatalf("RenderRouting(%q) render %d differs in the plan stop bullet", agent.ID, render)
			}
		}
	}
}
