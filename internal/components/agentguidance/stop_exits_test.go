package agentguidance

import (
	"strings"
	"testing"
)

// TestStopExitsStatesTheThreeExitsInOrder pins REQ-26.1 and REQ-26.8: the
// canonical text names the three exits in their fixed order and says iterating
// has no limit.
func TestStopExitsStatesTheThreeExitsInOrder(t *testing.T) {
	t.Parallel()

	text := StopExits()
	if !strings.HasPrefix(text, "**Stops and exits.**") {
		t.Fatalf("StopExits() must open with the bold lead paragraph, got:\n%s", text)
	}

	previous := -1
	for _, exit := range []string{
		"1. **Iterate:**",
		"2. **Approve, push and stop:**",
		"3. **Approve and continue:**",
	} {
		offset := strings.Index(text, exit)
		if offset < 0 {
			t.Fatalf("StopExits() is missing the exit %q:\n%s", exit, text)
		}
		if offset < previous {
			t.Fatalf("StopExits() lists the exit %q out of order:\n%s", exit, text)
		}
		previous = offset
	}
	if !strings.Contains(text, "There is no limit to the iterations.") {
		t.Fatalf("StopExits() must say that iterating has no limit:\n%s", text)
	}
}

// TestStopExitsDefinesWhatPushMeansAndWhoRunsIt pins REQ-26.8's last scenario:
// exit 2 needs a merged pull request, the multirepo specs repository commits on
// the integration branch with a human push, the remote operations are the
// human's, and a file-less artifact store cannot offer exit 2.
func TestStopExitsDefinesWhatPushMeansAndWhoRunsIt(t *testing.T) {
	t.Parallel()

	text := StopExits()
	for _, want := range []string{
		"a pull request merged into its target (`prTarget`) before anyone starts the next phase",
		"`axiom branch ensure` on another machine creates the branch from the merged base and does not recover work that is not merged",
		"In the specs repository of a `multirepo` there is no pull request: exit 2 is a commit on the integration branch and a push done by the human.",
		"`push`, `fetch`, opening the pull request, and merging belong to the human: present the exact commands and data, and never run them yourself.",
		"With the `engram` or `none` artifact store the artifacts are not files, so exit 2 is not available: offer only exits 1 and 3.",
		"The person who picks up a handed-off phase starts the next phase without reviewing the previous one again.",
		"A `continuous` change has no stops.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("StopExits() is missing %q:\n%s", want, text)
		}
	}
}

// TestStopExitsNamesNoGateInternals keeps the shared text SDD-only (REQ-27.17):
// ODD has no stops, so the text names neither ODD, its plan stop nor its feature
// document, and the typed-question field names stay out of it (the SDD gates
// section forbids them too).
func TestStopExitsNamesNoGateInternals(t *testing.T) {
	t.Parallel()

	text := StopExits()
	for _, forbidden := range []string{"ODD", "plan stop", "odd/tasks", "gateQuestion", "gateCriteria", "gateEvidence"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("StopExits() names %q, which does not belong to the SDD-only stops text", forbidden)
		}
	}
}

// TestStopExitsIsDeterministicAndTrimmed guards the determinism REQ-26.10
// demands of the projection: the same bytes on every call, without the
// whitespace of the asset file around them.
func TestStopExitsIsDeterministicAndTrimmed(t *testing.T) {
	t.Parallel()

	first := StopExits()
	second := StopExits()
	if first != second {
		t.Fatal("StopExits() is not deterministic")
	}
	if first != strings.TrimSpace(first) {
		t.Fatalf("StopExits() must not carry leading or trailing whitespace: %q", first)
	}
	if strings.Contains(first, "\n#") {
		t.Fatalf("StopExits() must not carry a heading of its own; each consumer places it:\n%s", first)
	}
}
