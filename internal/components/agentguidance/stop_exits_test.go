package agentguidance

import (
	"strings"
	"testing"
)

// TestStopExitsKeepsTheRemoteOperationsWithTheHuman pins the security invariant
// of the shared stops text: pushing code and fetching stay with the human unless
// the user authorizes them.
func TestStopExitsKeepsTheRemoteOperationsWithTheHuman(t *testing.T) {
	t.Parallel()

	text := StopExits()
	for _, want := range []string{
		"(or `axiom branch ensure --fetch` with the user's authorization)",
		"Pushing code, `fetch`, opening a pull request, and merging still belong to the human: present the exact commands and data, and never run them yourself.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("StopExits() is missing %q:\n%s", want, text)
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
