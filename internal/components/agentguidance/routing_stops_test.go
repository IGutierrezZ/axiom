package agentguidance

import (
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

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

// isoRoutingSegment returns the ISO condition at the start of step 0, between
// the first status read and the `Otherwise` that hands over to the project lane.
func isoRoutingSegment(t *testing.T, agent model.AgentID) string {
	t.Helper()

	rendered, err := RenderRouting(agent)
	if err != nil {
		t.Fatalf("RenderRouting(%q) error = %v", agent, err)
	}
	step0 := routingLineStartingWith(t, rendered, "0. **Evaluate scope and lane.**")
	start := strings.Index(step0, "When its `iso` field projects `project: iso`")
	end := strings.Index(step0, "Otherwise, read the project lane")
	if start < 0 || end < 0 || start > end {
		t.Fatalf("RenderRouting(%q) step 0 lost the ISO condition ahead of the lane read (start=%d end=%d):\n%s", agent, start, end, step0)
	}
	return step0[start:end]
}

// TestRoutingISOSendsAllWorkToSDD pins REQ-29.18: the ISO condition comes before
// the lane question, covers small work and the `ask` lane, and creates no ODD
// feature document.
func TestRoutingISOSendsAllWorkToSDD(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			segment := isoRoutingSegment(t, agent.ID)
			for _, want := range []string{
				"the project is ISO and only uses SDD",
				"all work, including small, understood work, goes through the SDD lane without asking the lane question",
				"also when `workflow.lane` is `ask`",
				"no `odd/tasks/<feature-name>.md` is created",
			} {
				if !strings.Contains(segment, want) {
					t.Fatalf("RenderRouting(%q) ISO condition is missing %q:\n%s", agent.ID, want, segment)
				}
			}

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			step0 := routingLineStartingWith(t, rendered, "0. **Evaluate scope and lane.**")
			if strings.Index(step0, "`project: iso`") > strings.Index(step0, "ask the blocking lane-selection question") {
				t.Fatalf("RenderRouting(%q) must state the ISO condition before the lane question:\n%s", agent.ID, step0)
			}
		})
	}
}

// TestRoutingISOBugRoutes pins the three bug routes and their literals.
func TestRoutingISOBugRoutes(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			segment := isoRoutingSegment(t, agent.ID)
			for _, want := range []string{
				"A bug in an ISO project always goes through SDD with the bug profile, along one of three routes",
				"when it restores a requirement that already exists, no delta and a `Cubre:` line citing that requirement",
				"when it touches no requirement, no delta and a `## Impacto en la spec` section holding the line `Sin impacto en la spec` and a `Motivo:` line",
				"when the requirement was wrongly specified or missing, a delta, as a normal change",
				"`## Informe de bug` section with `Observado:` and `Esperado:` lines in `proposal.md`",
			} {
				if !strings.Contains(segment, want) {
					t.Fatalf("RenderRouting(%q) ISO bug routes are missing %q:\n%s", agent.ID, want, segment)
				}
			}
		})
	}
}

// TestRoutingISOStaysNeutral pins that the ISO condition names no precedence
// between the declaration and the ODD lane, no integration rule for ODD and no
// stop: the contradiction is rejected when axiom.yaml loads, and the live-spec
// integration of ODD is not defined here.
func TestRoutingISOStaysNeutral(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			segment := isoRoutingSegment(t, agent.ID)
			for _, forbidden := range []string{
				"sdd.iso",
				"lane: odd",
				"precedence",
				"prevails",
				"integrat",
				"Plan stop",
				"three exits",
				"STOP",
			} {
				if strings.Contains(segment, forbidden) {
					t.Fatalf("RenderRouting(%q) ISO condition carries %q:\n%s", agent.ID, forbidden, segment)
				}
			}
		})
	}
}

// TestRoutingISONamesTheRouteInEveryEntryPoint pins that the three places which
// enumerate how SDD is entered name the ISO project, with the policy literal
// unchanged.
func TestRoutingISONamesTheRouteInEveryEntryPoint(t *testing.T) {
	t.Parallel()

	if err := checkRoutingPolicy("explicit_request_project_lane_or_accepted_proposal"); err != nil {
		t.Fatalf("checkRoutingPolicy(current) = %v, want nil", err)
	}

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			for _, want := range []string{
				"by the project lane declared in `axiom.yaml` (including an ISO project, `iso.project: iso`), or by an accepted proposal.",
				"- **Optional SDD:** propose SDD only when durable proposal, spec, design, and tasks would materially reduce substantial ambiguity. SDD is selected only by an explicit request, by the project lane `workflow.lane: sdd` for substantial work, by an ISO project (`iso.project: iso`) for all work, or by an accepted proposal.",
				"- **Optional SDD trigger:** propose SDD only when durable proposal/spec/design/tasks materially reduce substantial ambiguity. Select SDD only after an explicit request, the project lane `workflow.lane: sdd` for substantial work, an ISO project (`iso.project: iso`) for all work, or an accepted proposal; risk alone never forces SDD.",
			} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("RenderRouting(%q) is missing %q", agent.ID, want)
				}
			}
		})
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
