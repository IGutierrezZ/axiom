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
		"2. **Approve, publish and stop:**",
		"3. **Approve, publish and continue:**",
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

// TestStopExitsDefinesWhatPublishMeansAndWhoRunsIt pins REQ-26.8 and REQ-27.12: exits 2 and 3
// publish the change documents with `axiom sdd publish` (the approval is the explicit authorization of
// that push and of nothing else), a refused publication never undoes the approval, whoever continues
// adopts the published branch with fetch and ensure without a merged pull request, code pushes and
// pull requests stay the human's, and a file-less artifact store cannot offer exit 2.
func TestStopExitsDefinesWhatPublishMeansAndWhoRunsIt(t *testing.T) {
	t.Parallel()

	text := StopExits()
	for _, want := range []string{
		"approve the artifact with `axiom sdd gate record --hand-off`, publish the change documents, hand the next phase to another person, and stop. The session never starts the next phase. Code pushes, pull requests and merges remain yours.",
		"approve the artifact with `axiom sdd gate record`, publish the change documents, and continue yourself with the next phase. Code pushes, pull requests and merges remain yours.",
		"\"Publish\" in exits 2 and 3 is `axiom sdd publish`, run once right after the record with the invocation it returns",
		"commits only the documentation paths of the change and pushes only the change branch to `origin` (in the specs repository of a `multirepo`, its integration branch), never forced",
		"choosing exit 2 or 3 is the explicit authorization of that push with the git credentials already configured, and of nothing else",
		"A refused publication, such as `code_commits_unpublished` after a role-apply, never undoes the approval: present it as information",
		"Whoever continues runs `git fetch origin` (or `axiom branch ensure --fetch` with the user's authorization) and `axiom branch ensure --change <name>`, which adopts the published branch; no merged pull request is needed.",
		"Pushing code, `fetch`, opening a pull request, and merging still belong to the human: present the exact commands and data, and never run them yourself.",
		"With the `engram` or `none` artifact store the artifacts are not files: exit 2 is not available and exit 3 publishes nothing, so offer only exits 1 and 3.",
		"The person who picks up a handed-off phase starts the next phase without reviewing the previous one again.",
		"A `continuous` change has no stops.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("StopExits() is missing %q:\n%s", want, text)
		}
	}
	for _, retired := range []string{"push and stop", "merged into its target", "creates the branch from the merged base", "commit on the integration branch and a push done by the human"} {
		if strings.Contains(text, retired) {
			t.Fatalf("StopExits() keeps the retired wording %q:\n%s", retired, text)
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
