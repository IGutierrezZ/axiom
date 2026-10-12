package testenv

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func linkErr(errno syscall.Errno) error {
	return &os.LinkError{Op: "symlink", Old: "a", New: "b", Err: errno}
}

func TestDecideSymlink(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		goos    string
		want    outcome
		message string
	}{
		{"nil error runs the test", nil, "windows", outcomeRun, ""},
		{"nil error runs the test on linux", nil, "linux", outcomeRun, ""},
		{"1314 on windows skips", linkErr(1314), "windows", outcomeSkip, symlinkSkipMessage},
		{"1314 on linux is fatal", linkErr(1314), "linux", outcomeFail, "unexpectedly"},
		{"another errno on windows is fatal", linkErr(syscall.EACCES), "windows", outcomeFail, "unexpectedly"},
		{"non-link error on windows is fatal", errors.New("disk full"), "windows", outcomeFail, "disk full"},
		{"permission error on linux is fatal", linkErr(syscall.EPERM), "linux", outcomeFail, "unexpectedly"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, msg := decideSymlink(tt.err, tt.goos)
			if got != tt.want {
				t.Fatalf("decideSymlink() outcome = %v, want %v (msg %q)", got, tt.want, msg)
			}
			if !strings.Contains(msg, tt.message) {
				t.Fatalf("decideSymlink() message = %q, want it to contain %q", msg, tt.message)
			}
		})
	}
}

func TestSymlinkSkipMessageNamesTheCapabilityAndHowToEnableIt(t *testing.T) {
	for _, want := range []string{"symlink", "1314", "Developer Mode", "administrator"} {
		if !strings.Contains(symlinkSkipMessage, want) {
			t.Errorf("skip message %q does not mention %q", symlinkSkipMessage, want)
		}
	}
}

func TestIsSymlinkPrivilegeErrorOnlyMatchesWrappedLinkErrors(t *testing.T) {
	if !isSymlinkPrivilegeError(linkErr(1314)) {
		t.Error("a LinkError wrapping 1314 must match")
	}
	if isSymlinkPrivilegeError(errors.New("1314")) {
		t.Error("a plain error must not match")
	}
	if isSymlinkPrivilegeError(&os.LinkError{Op: "symlink", Err: errors.New("x")}) {
		t.Error("a LinkError without an errno must not match")
	}
}

func TestProbeSymlinkDecisionIsConsistentWithTheProbe(t *testing.T) {
	err := probeSymlink()
	got, msg := decideSymlink(err, runtime.GOOS)
	if err == nil && got != outcomeRun {
		t.Fatalf("probe succeeded but the decision is %v", got)
	}
	if err != nil && got == outcomeRun {
		t.Fatalf("probe failed with %v but the decision is to run", err)
	}
	t.Logf("probe err=%v decision=%v msg=%q", err, got, msg)
}

func TestRequireSymlinkRunsWhereTheCapabilityExists(t *testing.T) {
	// Where symlinks work, RequireSymlink must not skip. Where Windows lacks
	// the privilege, it skips and the body below is not reached.
	RequireSymlink(t)
	if err := probeSymlink(); err != nil {
		t.Fatalf("RequireSymlink returned but the probe fails: %v", err)
	}
}
