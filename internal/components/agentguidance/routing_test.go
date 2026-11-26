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
const supportedAgentCount = 11

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

			// Security invariant: no fetch without the user's authorization.
			if !strings.Contains(section, "Never pass `--fetch` without the explicit authorization of the user.") {
				t.Fatalf("RenderRouting(%q) subsection lost the --fetch authorization rule:\n%s", agent.ID, section)
			}

			// The runtime-specific worktree primitives are named once, only to ban them.
			for _, name := range []string{"EnterWorktree", "ExitWorktree"} {
				if got := strings.Count(rendered, name); got != 1 {
					t.Fatalf("RenderRouting(%q) names %s %d times, want exactly once (in the ban line)", agent.ID, name, got)
				}
			}
		})
	}
}

// TestRenderRoutingCarriesTheLivingSpecTokens pins, once and identically for
// every adapter, the literals that `axiom archive integrate` and the living-spec
// parser read from the ODD document.
func TestRenderRoutingCarriesTheLivingSpecTokens(t *testing.T) {
	t.Parallel()

	for _, agent := range catalog.AllAgents() {
		t.Run(string(agent.ID), func(t *testing.T) {
			t.Parallel()

			rendered, err := RenderRouting(agent.ID)
			if err != nil {
				t.Fatalf("RenderRouting(%q) error = %v", agent.ID, err)
			}
			for _, want := range []string{
				"`## Spec viva`",
				"`### Capacidad: <domain>`",
				"`Tipo: bug`",
				"`## Impacto en la spec`",
				"`Sin impacto en la spec`",
				"`Motivo: <one line>`",
				"`delivery.split_hint_lines`",
			} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("RenderRouting(%q) is missing the token %q", agent.ID, want)
				}
			}
			if got := strings.Count(rendered, "axiom archive integrate --odd <feature>"); got != 1 {
				t.Fatalf("RenderRouting(%q) names the integration command %d times, want exactly once", agent.ID, got)
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
