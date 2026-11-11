//go:build windows

package reviewtransaction

import "testing"

// unvalidatableGitCommonDir makes the Git common directory fail RAR ownership
// validation the way an unsupported volume does: the directory exists, Git
// resolves it, and validateRARRepositoryParent refuses it. It is the Windows
// stand-in for the #4670 SMB/UNC common directory, whose literal UNC refusal
// is covered in the path safety tests.
func unvalidatableGitCommonDir(t *testing.T) {
	t.Helper()
	previous := rarWindowsAuthorityOwnerUnsafe
	rarWindowsAuthorityOwnerUnsafe = ownerStubAlwaysReject
	t.Cleanup(func() { rarWindowsAuthorityOwnerUnsafe = previous })
}
