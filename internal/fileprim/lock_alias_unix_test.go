//go:build unix

package fileprim

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// TestSecureOpenLockParentRefusesASymlinkedAncestor reproduces, on any unix,
// the exact mechanism behind the 1773 boundary 3 report. On macOS the walk
// starts at / and the second component is var, which is a symlink to
// /private/var, so O_NOFOLLOW refuses it and every lock path under $TMPDIR
// fails. Here the symlinked ancestor is built explicitly instead of being
// inherited from the platform, and the refusal is identical.
//
// This is the no-follow protection doing its job, not a defect in it. The
// same target reached through its resolved spelling is accepted, which is
// what every production store path looks like -- see
// TestAuthoritativeStorePathsAreCanonicalUnderAnAliasedRepository.
func TestSecureOpenLockParentRefusesASymlinkedAncestor(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(base, "real", "store")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "real"), filepath.Join(base, "alias")); err != nil {
		t.Fatal(err)
	}
	aliased := filepath.Join(base, "alias", "store")

	if _, err := secureOpenLockParent(string(filepath.Separator), aliased); err == nil {
		t.Fatal("the root-anchored walk traversed a symlinked ancestor; O_NOFOLLOW protection is gone")
	}

	fd, err := secureOpenLockParent(string(filepath.Separator), real)
	if err != nil {
		t.Fatalf("the root-anchored walk refused a fully resolved path: %v", err)
	}
	defer unix.Close(fd)
	if !sameDirectory(t, fd, real) {
		t.Fatal("the root-anchored walk opened the wrong directory")
	}
}
