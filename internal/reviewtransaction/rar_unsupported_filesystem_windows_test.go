//go:build windows

package reviewtransaction

import "testing"

// #5533: every path in the Win32 `\\` family is refused as an authority
// parent, but only a server share is remote. The local long-path and device
// forms of a drive letter or volume are on this machine, so a refusal for one
// must not describe an SMB/UNC share. The rule is pure so it runs everywhere.
func TestWindowsNetworkPathOnlyClassifiesServerSharesAsRemote(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{`\\server\share\repo\.git`, true},
		{`\\?\UNC\server\share\repo\.git`, true},
		{`\\?\unc\server\share\repo\.git`, true},
		{`\\.\UNC\server\share\repo\.git`, true},
		{`\\?\C:\repo\.git`, false},
		{`\\?\c:\repo\.git`, false},
		{`\\.\C:\repo\.git`, false},
		{`\\?\Volume{0b1d6a7e-0000-0000-0000-100000000000}\repo\.git`, false},
		{`C:\repo\.git`, false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := windowsNetworkPath(tt.path); got != tt.want {
				t.Fatalf("windowsNetworkPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
