//go:build windows

package skillregistry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// extendedPrefix is the Win32 extended-length namespace prefix.
const extendedPrefix = `\\?\`

// requireDriveLetterPath skips the test when path is not spelled with a drive
// letter (a UNC temp directory has no drive-letter or `\\?\` form to exercise).
func requireDriveLetterPath(t *testing.T, path string) {
	t.Helper()
	if volume := filepath.VolumeName(path); len(volume) != 2 || volume[1] != ':' {
		t.Skipf("%q has no drive letter", path)
	}
}

// TestHomeDirectoryIsRefusedByIdentityWindowsSpellings is the incident itself:
// %USERPROFILE% and a hook's --cwd name one directory with different strings. A
// lexical comparison missed all of these, hasProjectMarker(home) was true
// because ~/.claude/skills exists, and Regenerate then created ~/.atl.
// %TEMP% is typically `C:\Users\IGUTIE~1\...`, so the short form is the one a
// real session produces.
func TestHomeDirectoryIsRefusedByIdentityWindowsSpellings(t *testing.T) {
	home := newFakeHome(t)
	requireDriveLetterPath(t, home)
	short := shortPathOf(t, home)

	cases := []struct {
		name    string
		cwd     string
		homeArg string
	}{
		{"cwd in 8.3 short form", short, home},
		{"home in 8.3 short form", home, short},
		{"cwd with a lower-case drive letter", flipDriveCase(home), home},
		{"home with a lower-case drive letter", home, flipDriveCase(home)},
		{"cwd in upper case", strings.ToUpper(home), home},
		{"cwd with the extended-length prefix", extendedPrefix + home, home},
		{"home with the extended-length prefix", home, extendedPrefix + home},
		{"cwd with the extended-length prefix and 8.3 short form", extendedPrefix + short, home},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Without the identity check each of these is a real lexical miss.
			if filepath.Clean(tc.cwd) == filepath.Clean(tc.homeArg) {
				t.Skipf("%q and %q already compare equal; nothing to prove", tc.cwd, tc.homeArg)
			}
			assertHomeRefused(t, tc.cwd, tc.homeArg, home)
		})
	}
}

// TestFilesystemRootWindowsSpellings pins that every spelling of a volume root
// is refused. The lexical test already covers them (a root's parent is itself,
// whatever the case or the `\\?\` prefix), so this guards the identity rewrite
// against regressing that.
func TestFilesystemRootWindowsSpellings(t *testing.T) {
	drive := filepath.VolumeName(os.Getenv("SystemDrive"))
	if len(drive) != 2 || drive[1] != ':' {
		t.Skipf("SystemDrive %q has no drive letter", drive)
	}
	home := t.TempDir()
	for name, root := range map[string]string{
		"drive root":                     drive + `\`,
		"lower-case drive letter":        strings.ToLower(drive) + `\`,
		"forward slash":                  drive + `/`,
		"extended-length prefix":         extendedPrefix + drive + `\`,
		"extended-length lower-case":     extendedPrefix + strings.ToLower(drive) + `\`,
		"parent of a first-level folder": drive + `\Windows\..`,
		"current directory of the drive": drive + `\.`,
		"extended-length trailing dot":   extendedPrefix + drive + `\.`,
		"device namespace prefix":        `\\.\` + drive + `\`,
	} {
		t.Run(name, func(t *testing.T) {
			if got := RefreshSkip(root, home); got != SkipFilesystemRoot {
				t.Fatalf("RefreshSkip(%q) = %q, want %q", root, got, SkipFilesystemRoot)
			}
		})
	}
}
