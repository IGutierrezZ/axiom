package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// unvalidatableReviewModeSwitchRoot leaves an override directory where this
// clone's switch reads it, reached through a symlinked gentle-ai ancestor that
// RAR path validation refuses. It is the portable stand-in for a #4670 common
// directory the authority checks cannot accept.
func unvalidatableReviewModeSwitchRoot(t *testing.T, repo string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs a privilege Windows runners do not grant; the UNC refusal is covered in reviewtransaction")
	}
	target := filepath.Join(t.TempDir(), "gentle-ai")
	if err := os.MkdirAll(filepath.Join(target, "review-mode", "rar-authority", "v1", "rdd-mode"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(repo, ".git", "gentle-ai")); err != nil {
		t.Fatal(err)
	}
}

// #4670: while the mode is off without the clone override (here the default,
// since nobody opted in) nothing has to be read from the clone, so an
// unvalidatable clone-local location must not make status fail.
func TestReviewModeStatusOffSucceedsWhenTheCloneOverrideCannotBeValidated(t *testing.T) {
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	if err := RunReviewMode([]string{"disable", "--cwd", repo, "--scope", "clone"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	unvalidatableReviewModeSwitchRoot(t, repo)

	var output bytes.Buffer
	if err := RunReviewMode([]string{"status", "--cwd", repo}, &output); err != nil {
		t.Fatalf("status with the mode off failed: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "receipt-driven development: off (decided by default)") {
		t.Fatalf("status output = %q, want off decided by default", output.String())
	}
}

// #4670: a resolution failure is not a clone-local decision. With the global
// mode on, the override can still decide, so status fails closed -- but it says
// the mode is unavailable instead of printing an "off (decided by clone_local)"
// that no stored record made.
func TestReviewModeStatusReportsAFailedResolutionAsUnavailable(t *testing.T) {
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)
	if err := RunReviewMode([]string{"enable", "--cwd", repo, "--scope", "global"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	unvalidatableReviewModeSwitchRoot(t, repo)

	var output bytes.Buffer
	if err := RunReviewMode([]string{"status", "--cwd", repo}, &output); err == nil {
		t.Fatalf("status resolved an unvalidatable override without an error:\n%s", output.String())
	}
	if strings.Contains(output.String(), "decided by") {
		t.Fatalf("failed resolution reported a deciding source: %q", output.String())
	}
	if !strings.Contains(output.String(), "receipt-driven development: unavailable") {
		t.Fatalf("status output = %q, want the mode marked unavailable", output.String())
	}
}

// #4670: an authority filesystem that cannot keep review state private is a
// typed refusal whose continuation the operator can actually run.
func TestReviewModeUnsupportedAuthorityFilesystemNamesItsContinuation(t *testing.T) {
	path := `\\server\share\repo\.git`
	cause := fmt.Errorf("resolve review mode: %w",
		&reviewtransaction.UnsupportedAuthorityFilesystemError{Path: path, Remote: true})

	err := reviewModeUnreadable(context.Background(), t.TempDir(), reviewtransaction.RDDGlobalMode{}, cause)

	var unsupported *reviewtransaction.UnsupportedAuthorityFilesystemError
	if !errors.As(err, &unsupported) {
		t.Fatalf("refusal lost its typed cause: %#v", err)
	}
	for _, want := range []string{path, "`axiom review mode disable`", "--scope global", "local NTFS or ReFS volume"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q does not contain %q", err.Error(), want)
		}
	}
	if strings.Contains(err.Error(), "check the path and rerun") {
		t.Fatalf("refusal %q still names an unrunnable repair", err.Error())
	}
}

// The stop hook has nothing to check while the mode is off, so a clone-local
// switch location that fails validation must not turn every turn into an error.
func TestReviewStopHookSilentWhenRDDOffAndTheCloneOverrideCannotBeValidated(t *testing.T) {
	home := reviewModeHome(t)
	repo := initReviewStopHookRepo(t)
	stageReviewStopHookCandidate(t, repo)
	if err := RunReviewMode([]string{"disable", "--cwd", repo, "--scope", "clone"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	unvalidatableReviewModeSwitchRoot(t, repo)

	stdin := strings.NewReader(reviewStopHookTestPayload(t, "sess-off-unvalidatable", repo, false, nil))
	var stdout, stderr bytes.Buffer
	if err := runReviewStopHook([]string{"--agent", "claude-code"}, stdin, &stdout, &stderr); err != nil {
		t.Fatalf("runReviewStopHook: %v\nstderr: %s", err, stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("expected silence, got stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if _, err := os.Stat(reviewStopHookStateFile(home, "sess-off-unvalidatable")); !os.IsNotExist(err) {
		t.Fatalf("expected no reminder state file while RDD is off, stat err=%v", err)
	}
}

// A review authority that cannot live on this filesystem means no review can
// start here, so the reminder has nothing to point at. The hook stays quiet for
// that typed refusal and still fails on anything else.
func TestReviewStopHookQuietOnlyForAnUnsupportedAuthorityFilesystem(t *testing.T) {
	unsupported := fmt.Errorf("resolve review mode: %w",
		&reviewtransaction.UnsupportedAuthorityFilesystemError{Path: `\\server\share\repo\.git`, Remote: true})
	if !reviewStopHookAuthorityUnsupported(unsupported) {
		t.Fatalf("typed unsupported-filesystem refusal was not recognised: %v", unsupported)
	}
	if reviewStopHookAuthorityUnsupported(errors.New("unsafe RAR authority path")) {
		t.Fatal("an ordinary resolution failure was silenced")
	}
}
