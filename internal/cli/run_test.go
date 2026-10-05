package cli

import (
	"runtime"
	"testing"
)

// setOpenCodeTestHome keeps XDG resolution bound to the test home on Windows,
// where os.UserHomeDir reads USERPROFILE rather than HOME.
func setOpenCodeTestHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
}
