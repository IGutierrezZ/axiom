//go:build unix

package reviewtransaction

import (
	"os"
	"testing"
)

// unvalidatableGitCommonDir makes the Git common directory fail RAR ownership
// validation the same way a share owned by another principal does: the
// directory exists, Git resolves it, and validateRARRepositoryParent refuses
// it. It is the POSIX stand-in for the #4670 SMB/UNC common directory.
func unvalidatableGitCommonDir(t *testing.T) {
	t.Helper()
	previous := rarCurrentEUID
	rarCurrentEUID = func() int { return os.Geteuid() + 1 }
	t.Cleanup(func() { rarCurrentEUID = previous })
}
