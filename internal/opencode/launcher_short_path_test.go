package opencode

import (
	"errors"
	"testing"
)

func TestWindowsLauncherTargetUsesASCIIShortPathForNonASCIITargets(t *testing.T) {
	const long = `C:\Users\IñigoGutiérrez\AppData\Roaming\npm\opencode.cmd`
	const short = `C:\Users\IIGOGU~1\AppData\Roaming\npm\opencode.cmd`
	tests := []struct {
		name   string
		goos   string
		target string
		short  string
		err    error
		want   string
		probed bool
	}{
		{name: "non-ASCII target takes the 8.3 form", goos: "windows", target: long, short: short, want: short, probed: true},
		{name: "ASCII target is never probed", goos: "windows", target: `C:\Program Files\OpenCode\opencode.exe`, want: `C:\Program Files\OpenCode\opencode.exe`},
		{name: "volume without short names keeps the long path", goos: "windows", target: long, err: errors.New("not supported"), want: long, probed: true},
		{name: "short form that is still non-ASCII keeps the long path", goos: "windows", target: long, short: `C:\Users\Iñigo~1\opencode.cmd`, want: long, probed: true},
		{name: "empty short form keeps the long path", goos: "windows", target: long, want: long, probed: true},
		{name: "POSIX launcher keeps its target", goos: "linux", target: `/home/zoë/opencode`, want: `/home/zoë/opencode`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previous := shortPathName
			t.Cleanup(func() { shortPathName = previous })
			probed := false
			shortPathName = func(path string) (string, error) {
				probed = true
				if path != tt.target {
					t.Fatalf("shortPathName(%q), want %q", path, tt.target)
				}
				return tt.short, tt.err
			}
			if got := windowsLauncherTarget(tt.goos, tt.target); got != tt.want {
				t.Fatalf("windowsLauncherTarget(%q, %q) = %q, want %q", tt.goos, tt.target, got, tt.want)
			}
			if probed != tt.probed {
				t.Fatalf("shortPathName probed = %t, want %t", probed, tt.probed)
			}
		})
	}
}
