package cli

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// quotedCwdArgument matches the double-quoted value of a printed `--cwd`.
var quotedCwdArgument = regexp.MustCompile(`--cwd "([^"]*)"`)

// TestEmittedContinuationsQuoteSpacedRepositoryPaths proves that every
// continuation below, which names the repository the caller must run it in,
// prints a path with spaces as one quoted word that is byte-identical to the
// real path. Copying the command into a shell must not split the path at its
// space, and a Windows path must not have its separators doubled the way a
// Go-quoted %q renders them.
func TestEmittedContinuationsQuoteSpacedRepositoryPaths(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "mi carpeta", "repo con espacios")

	tests := []struct {
		name string
		text string
	}{
		{name: "abandon inputs refusal", text: reviewAbandonInputsRefusal(repo).Error()},
		{
			name: "correction context budget abandon command",
			text: reviewCorrectionContextBudgetAction(
				reviewtransaction.CompactAbandonEligibility{Eligible: true, Revision: "rev-1", SnapshotIdentity: "sha256:frozen"}, repo, "review-0001"),
		},
		{name: "stop hook status command", text: reviewStopHookReasonText("sha256:target", repo, "claude-code", "axiom review start --consent=relay")},
		{name: "recover command", text: reviewRecoverCommand(repo, "predecessor", "rev-1", "successor", "scope_changed")},
		{name: "unreadable clone mode commands", text: strings.Join(ReviewModeUnreadableScope{Scope: reviewModeScopeClone, Repo: repo}.commands(), "\n")},
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
