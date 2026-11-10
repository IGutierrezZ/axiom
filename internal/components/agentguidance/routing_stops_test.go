package agentguidance

import (
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// TestRenderRoutingHasNoOddStops guards REQ-27.17: ODD has no stop of its own.
// No adapter renders the plan stop, the three-exit section, or the hand-off of a
// feature document; the stops and exits belong to the SDD gates only.
func TestRenderRoutingHasNoOddStops(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			for _, forbidden := range []string{
				"Plan stop",
				"plan stop",
				"### Stops and exits",
				"three exits",
				"handed off",
				"approved plan",
			} {
				if strings.Contains(rendered, forbidden) {
					t.Fatalf("RenderRouting(%q) still carries %q; ODD has no stops (REQ-27.17)", agent.ID, forbidden)
				}
			}
			if strings.Contains(rendered, StopExits()) {
				t.Fatalf("RenderRouting(%q) projects StopExits(), which belongs to the SDD orchestrators only", agent.ID)
			}
		})
	}
}

// TestRenderRoutingStepFiveCreatesTheFeatureDocumentWithoutStopping pins step 5:
// the feature document and its mirror are created, the user is told in one line,
// and the next step is implementation, not a stop.
func TestRenderRoutingStepFiveCreatesTheFeatureDocumentWithoutStopping(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			step5 := routingLineStartingWith(t, rendered, "5. **Track before the first write.**")
			if !strings.Contains(step5, "Tell the user in one line which feature document was created and how many tasks it holds.") {
				t.Fatalf("RenderRouting(%q) step 5 lost the one-line notice:\n%s", agent.ID, step5)
			}
			if !strings.HasSuffix(strings.TrimSpace(step5), "how many tasks it holds.") {
				t.Fatalf("RenderRouting(%q) step 5 must end after the notice, with no stop:\n%s", agent.ID, step5)
			}
			if strings.Contains(step5, "axiom branch") {
				t.Fatalf("RenderRouting(%q) step 5 names axiom branch, which only step 6 may do:\n%s", agent.ID, step5)
			}
		})
	}
}

// TestRenderRoutingResumesWithoutTheEngramMirror pins the resume line: it starts
// from the task file, tolerates a missing mirror and continues with the first
// unfinished task, without any reference to a handed-off plan.
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
				"When the Engram mirror does not exist, start from the task file alone, never fail for the missing mirror, and continue with the first unfinished task.",
			} {
				if !strings.Contains(resume, want) {
					t.Fatalf("RenderRouting(%q) resume line is missing %q:\n%s", agent.ID, want, resume)
				}
			}
		})
	}
}

// TestRenderRoutingStepZeroReadsTheProjectLane pins REQ-27.16: step 0 reads the
// lane from the `workflow` field of the SDD status, covers the three lanes and
// the fail-closed sources, and keeps the blocking lane question for `ask`.
func TestRenderRoutingStepZeroReadsTheProjectLane(t *testing.T) {
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
				"axiom sdd status --cwd <repo> --json",
				"`workflow` field",
				"`lane` (`odd`, `sdd` or `ask`) and `source`",
				"with or without a selected change",
				"Small, understood work stays small in every lane and creates no work artifact.",
				"with `lane: sdd`, enter the SDD pre-flight questionnaire directly, without asking the lane question",
				"with `lane: odd`, work through ODD without asking, even for an architectural scope, unless the user explicitly asks for SDD in this request",
				"with `lane: ask`, when `source` is `no_workspace` or `invalid`, or when the status cannot be read, ask the blocking lane-selection question",
				"do you want to approach this through the agile ODD lane or the formal SDD lane?",
				"creating no work artifact while it is pending",
				"under `ask`, a scope that is unambiguously architectural or large skips that binary question",
				"Whenever the work goes through ODD, do not ask any further governance question",
			} {
				if !strings.Contains(step0, want) {
					t.Fatalf("RenderRouting(%q) step 0 is missing %q:\n%s", agent.ID, want, step0)
				}
			}
			if strings.Contains(step0, "plan stop") {
				t.Fatalf("RenderRouting(%q) step 0 still mentions the plan stop:\n%s", agent.ID, step0)
			}
		})
	}
}

// TestRenderRoutingLaneTextIsIdenticalAcrossAgentsAndRenders guards determinism:
// step 0 is the same for every adapter and on a second render.
func TestRenderRoutingLaneTextIsIdenticalAcrossAgentsAndRenders(t *testing.T) {
	t.Parallel()

	reference, err := RenderRouting(model.AgentClaudeCode)
	if err != nil {
		t.Fatalf("RenderRouting(%q) error = %v", model.AgentClaudeCode, err)
	}
	referenceStep := routingLineStartingWith(t, reference, "0. **Evaluate scope and lane.**")

	for _, agent := range catalog.AllAgents() {
		for render := 0; render < 2; render++ {
			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			if routingLineStartingWith(t, rendered, "0. **Evaluate scope and lane.**") != referenceStep {
				t.Fatalf("RenderRouting(%q) render %d differs in step 0", agent.ID, render)
			}
		}
	}
}

// TestUnknownRoutingPolicyFailsClosed guards DD-03: the renderer compares the
// manifest policy against the current literal and rejects any other, including the
// retired one, instead of rendering stale prose over a changed policy.
func TestUnknownRoutingPolicyFailsClosed(t *testing.T) {
	t.Parallel()

	for _, policy := range []string{"explicit_request_or_accepted_proposal", "automatic", ""} {
		err := checkRoutingPolicy(policy)
		if !errors.Is(err, ErrUnknownRoutingPolicy) {
			t.Fatalf("checkRoutingPolicy(%q) = %v, want ErrUnknownRoutingPolicy", policy, err)
		}
	}
	if err := checkRoutingPolicy("explicit_request_project_lane_or_accepted_proposal"); err != nil {
		t.Fatalf("checkRoutingPolicy(current) = %v, want nil", err)
	}
}
