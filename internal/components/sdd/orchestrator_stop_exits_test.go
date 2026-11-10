package sdd

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/catalog"
	"github.com/IGutierrezZ/axiom/v3/internal/components/agentguidance"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// renderedOrchestratorForStops renders what an agent receives as its SDD
// orchestrator. Claude Code keeps the section in its lazy workflow file, so that
// file is part of what it renders.
func renderedOrchestratorForStops(t *testing.T, agent model.AgentID) string {
	t.Helper()
	content := renderSDDOrchestratorAsset(agent)
	if agent == model.AgentClaudeCode {
		lazy, err := renderClaudeSessionPreflight()
		if err != nil {
			t.Fatalf("renderClaudeSessionPreflight() error = %v", err)
		}
		content += lazy
	}
	return content
}

// TestEveryOrchestratorRendersTheSingleStopExitsText guards REQ-26.8: the twelve
// composed orchestrator templates (every runtime that shares them, Pi through the
// generic one included) carry the very text agentguidance.StopExits returns, once,
// with the marker resolved.
func TestEveryOrchestratorRendersTheSingleStopExitsText(t *testing.T) {
	want := agentguidance.StopExits()
	if want == "" {
		t.Fatal("agentguidance.StopExits() is empty")
	}

	for _, entry := range catalog.AllAgents() {
		t.Run(string(entry.ID), func(t *testing.T) {
			rendered := renderedOrchestratorForStops(t, entry.ID)

			if got := strings.Count(rendered, want); got != 1 {
				t.Fatalf("%s renders the stop-exits text %d times, want exactly once", entry.ID, got)
			}
			if strings.Contains(rendered, stopExitsMarker) {
				t.Fatalf("%s kept the raw %s marker", entry.ID, stopExitsMarker)
			}
			if strings.Contains(rendered, sharedOrchestratorSectionOpen) {
				t.Fatalf("%s kept a raw shared-section placeholder", entry.ID)
			}
		})
	}
}

// TestEveryOrchestratorRecordsTheThreeExitsWithGateRecordOnly pins the SDD half of
// REQ-26.8: `axiom sdd gate record` is the only way to record the decision, the
// three exits map to its three invocations, and no orchestrator names the
// deprecated --handoff-policy flag.
func TestEveryOrchestratorRecordsTheThreeExitsWithGateRecordOnly(t *testing.T) {
	for _, entry := range catalog.AllAgents() {
		t.Run(string(entry.ID), func(t *testing.T) {
			rendered := renderedOrchestratorForStops(t, entry.ID)

			for _, want := range []string{
				"`axiom sdd gate record --cwd <repo> --change <name> --gate spec|design|tasks|role-apply:<role>` is the only way to record the decision",
				"Iterate is `--decision rejected --reason \"<text>\"`",
				"Approve, publish and stop is `--decision approved --hand-off`",
				"Approve, publish and continue is `--decision approved`.",
				"re-present the SAME gate afterward, never skip ahead to the next one",
				"The `integration` gate keeps its two exits",
			} {
				if !strings.Contains(rendered, want) {
					t.Fatalf("%s is missing %q", entry.ID, want)
				}
			}
			if strings.Contains(rendered, "--handoff-policy") {
				t.Fatalf("%s names --handoff-policy, which is deprecated and never required", entry.ID)
			}
		})
	}
}

// TestStopExitsMarkerIsResolvedInASecondExplicitStep proves the marker is a token
// distinct from the section placeholder: a section body carrying it is resolved by
// the second step, and content without the marker is left untouched.
func TestStopExitsMarkerIsResolvedInASecondExplicitStep(t *testing.T) {
	want := agentguidance.StopExits()

	got := substituteSharedOrchestratorSections("before\n\n" + stopExitsMarker + "\n\nafter")
	if got != "before\n\n"+want+"\n\nafter" {
		t.Fatalf("substituteSharedOrchestratorSections() did not resolve the marker with StopExits():\n%s", got)
	}

	const plain = "no marker here, and no placeholder either"
	if got := substituteSharedOrchestratorSections(plain); got != plain {
		t.Fatalf("substituteSharedOrchestratorSections() changed content without a marker: %q", got)
	}
}
