package cli

import "testing"

func TestParseCommitReviewMode(t *testing.T) {
	for _, valid := range []string{"notify", "stop", "stop-before-commit"} {
		if mode, err := parseCommitReviewMode(valid); err != nil || string(mode) != valid {
			t.Fatalf("parseCommitReviewMode(%q) = %q, %v", valid, mode, err)
		}
	}
	for _, invalid := range []string{"", "Notify", "otro"} {
		if _, err := parseCommitReviewMode(invalid); err == nil {
			t.Fatalf("parseCommitReviewMode(%q) sin error", invalid)
		}
	}
}
