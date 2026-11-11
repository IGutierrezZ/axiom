//go:build windows

package reviewtransaction

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// #4670: a Git common directory on an SMB share reaches the caller as the
// typed unsupported-filesystem refusal, still inside the unsafe-path family,
// and its message names a continuation instead of "check the path and rerun".
func TestRARWindowsUNCCommonDirIsATypedUnsupportedFilesystem(t *testing.T) {
	for _, path := range []string{`\\server\share\repo\.git`, `\\?\UNC\server\share\repo\.git`} {
		t.Run(path, func(t *testing.T) {
			err := validateRARRepositoryParent(path)

			var unsupported *UnsupportedAuthorityFilesystemError
			if !errors.As(err, &unsupported) {
				t.Fatalf("err = %v, want *UnsupportedAuthorityFilesystemError", err)
			}
			if unsupported.Path != path || !unsupported.Remote {
				t.Fatalf("typed refusal = %#v, want remote %q", unsupported, path)
			}
			if !errors.Is(err, errUnsafeRARAuthorityPath) || !errors.Is(err, errUnknownWindowsFilesystem) {
				t.Fatalf("err = %v, want the unsafe-path and unknown-filesystem sentinels", err)
			}
			if strings.Contains(err.Error(), "check the path and rerun") ||
				!strings.Contains(err.Error(), "axiom review mode disable") {
				t.Fatalf("message %q does not name the documented continuation", err.Error())
			}
		})
	}
}

// A remote drive letter is the same share behind a mapping, so it carries the
// same typed refusal; an exFAT volume is local but equally unable to keep
// authority private.
func TestRARWindowsUnsupportedVolumesAreTypedRefusals(t *testing.T) {
	origOwner := rarWindowsAuthorityOwnerUnsafe
	t.Cleanup(func() { rarWindowsAuthorityOwnerUnsafe = origOwner })
	rarWindowsAuthorityOwnerUnsafe = ownerStubAlwaysReject

	stub := &classifierStub{}
	origClassifier := rarWindowsAuthorityFilesystemClassifier
	t.Cleanup(func() { rarWindowsAuthorityFilesystemClassifier = origClassifier })
	rarWindowsAuthorityFilesystemClassifier = stub.classify

	tests := []struct {
		name       string
		fsType     string
		driveType  uint32
		wantRemote bool
	}{
		{"mapped network drive", "NTFS", windows.DRIVE_REMOTE, true},
		{"exFAT volume", "exFAT", windows.DRIVE_FIXED, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub.fstype, stub.driveType, stub.err = tc.fsType, tc.driveType, nil
			err := validateRARRepositoryParent(t.TempDir())

			var unsupported *UnsupportedAuthorityFilesystemError
			if !errors.As(err, &unsupported) || unsupported.Remote != tc.wantRemote {
				t.Fatalf("err = %v (typed %#v), want remote=%v", err, unsupported, tc.wantRemote)
			}
			if !errors.Is(err, errUnsafeRARAuthorityPath) {
				t.Fatalf("err = %v, want the unsafe-path sentinel", err)
			}
		})
	}
}
