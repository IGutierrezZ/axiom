package reviewtransaction

import (
	"path/filepath"
	"regexp"
	"testing"
)

// quotedCwdArgument matches the double-quoted value of a printed `--cwd`.
var quotedCwdArgument = regexp.MustCompile(`--cwd "([^"]*)"`)

// TestContinuationsQuoteSpacedRepositoryPaths proves the continuations this
// package renders print a repository path with spaces as one quoted word that
// is byte-identical to the real path, so the command can be copied and run as
// printed. fmt's %q would instead double the separators of a Windows path.
func TestContinuationsQuoteSpacedRepositoryPaths(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "mi carpeta", "repo con espacios")

	tests := []struct {
		name string
		text string
	}{
		{
			name: "abandon rerun",
			text: compactAbandonV2Rerun(repo, "review-0001", "rev-1", "sha256:frozen", "maintainer", CompactAbandonReasonOperatorDisposition, CompactDiscardedWorkSummary{}),
		},
		{
			name: "store reset refused while reviews are in flight",
			text: (&StoreResetInFlightError{
				Repository: repo,
				Lineages:   []StoreResetLineage{{LineageID: "review-0001", State: StateReviewing}},
			}).Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := quotedCwdArgument.FindStringSubmatch(tt.text)
			if match == nil {
				t.Fatalf("no double-quoted --cwd in %q", tt.text)
			}
			if match[1] != repo {
				t.Fatalf("quoted --cwd = %q, want the exact path %q", match[1], repo)
			}
		})
	}
}
