package opencode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// windowsCMDLauncherFixture is the launcher written by existing installs. It is
// literal on purpose: recognition is a contract with files already on disk, so
// it must not follow the generator if the generator ever changes.
const windowsCMDLauncherFixture = "@echo off\r\n" +
	"rem gentle-ai:managed-opencode-launcher/v1\r\n" +
	"setlocal\r\n" +
	"if not defined OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS set \"OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS=true\"\r\n" +
	"\"C:\\Users\\dev\\AppData\\Roaming\\npm\\opencode.cmd\" %*\r\n" +
	"exit /b %ERRORLEVEL%\r\n"

func TestParseManagedLauncherRecognizesInstalledFixtures(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "windows cmd from an existing install", content: windowsCMDLauncherFixture, want: `C:\Users\dev\AppData\Roaming\npm\opencode.cmd`},
		{
			name: "posix sh",
			content: "#!/bin/sh\n# gentle-ai:managed-opencode-launcher/v1\nset -eu\n" +
				"if [ -z \"${OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS+x}\" ]; then\n  export OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS=true\nfi\n" +
				"exec '/home/dev/.npm-global/bin/opencode' \"$@\"\n",
			want: "/home/dev/.npm-global/bin/opencode",
		},
		{
			name: "windows powershell",
			content: "# gentle-ai:managed-opencode-launcher/v1\r\n$ErrorActionPreference = 'Stop'\r\n" +
				"if (-not (Test-Path Env:OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS)) { $env:OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS = 'true' }\r\n" +
				"& 'C:\\Users\\dev\\AppData\\Roaming\\npm\\opencode.ps1' @args\r\nexit $LASTEXITCODE\r\n",
			want: `C:\Users\dev\AppData\Roaming\npm\opencode.ps1`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseManagedLauncher(tt.content)
			if !ok || got != tt.want {
				t.Fatalf("parseManagedLauncher() = %q, %t, want %q, true", got, ok, tt.want)
			}
		})
	}
}

// TestParseManagedLauncherRoundTripsGeneratedLaunchers keeps the parser aligned
// with the generators, including targets that need quoting.
func TestParseManagedLauncherRoundTripsGeneratedLaunchers(t *testing.T) {
	targets := []string{
		`/usr/local/bin/opencode`,
		`/opt/Open Code/it's/opencode`,
		`C:\Program Files\OpenCode\opencode.exe`,
		`C:\Users\dev\AppData\Roaming\npm\opencode.cmd`,
		`C:\Users\dev\tools\open"code\opencode.cmd`,
		`C:\Users\dev\it's\opencode.ps1`,
		// Characters cmd and PowerShell treat specially outside quotes, a
		// non-ASCII profile path, and an upper-case .PS1 extension, which the
		// cmd generator still matches case-insensitively.
		`C:\Users\dev\100%\opencode.cmd`,
		`C:\Users\dev\R&D\opencode.cmd`,
		`C:\Users\dev\It's 100% R&D\opencode.exe`,
		`C:\Users\Iñaki\café\opencode.cmd`,
		`/home/zoë/日本語/opencode`,
		`C:\Users\dev\tools\OPENCODE.PS1`,
		`C:\Users\dev\R&D\it's\OPENCODE.PS1`,
	}
	for _, target := range targets {
		for _, goos := range []string{"linux", "windows"} {
			for name, content := range launcherContent(goos, target) {
				got, ok := parseManagedLauncher(content)
				if !ok || got != target {
					t.Errorf("%s launcher for %q parsed as %q, %t", name, target, got, ok)
				}
			}
		}
	}
}

func TestParseManagedLauncherRejectsWhatIsNotAGeneratedLauncher(t *testing.T) {
	const invoke = "\"C:\\npm\\opencode.cmd\" %*\r\n"
	tests := []struct {
		name    string
		content string
	}{
		{name: "empty", content: ""},
		{name: "marker only", content: "rem " + OwnershipMarker + "\r\n"},
		{name: "marker outside the header", content: "@echo off\r\nsetlocal\r\nrem " + OwnershipMarker + "\r\n" + invoke},
		{name: "marker quoted in prose", content: "@echo off\r\necho see " + OwnershipMarker + "\r\n" + invoke},
		{name: "marker with a trailing suffix", content: "@echo off\r\nrem " + OwnershipMarker + "0\r\n" + invoke},
		{name: "newer launcher version", content: "@echo off\r\nrem gentle-ai:managed-opencode-launcher/v2\r\n" + invoke},
		{name: "invocation before the marker", content: invoke + "rem " + OwnershipMarker + "\r\n"},
		{name: "unrelated user script", content: "@echo off\r\n" + invoke},
		{name: "posix invocation without a target", content: "#!/bin/sh\n# " + OwnershipMarker + "\nexec \"$@\"\n"},
		{name: "posix empty target", content: "#!/bin/sh\n# " + OwnershipMarker + "\nexec '' \"$@\"\n"},
		{name: "posix target quoted wrongly", content: "#!/bin/sh\n# " + OwnershipMarker + "\nexec '/a/b'c' \"$@\"\n"},
		{name: "posix unquoted target", content: "#!/bin/sh\n# " + OwnershipMarker + "\nexec /usr/bin/opencode \"$@\"\n"},
		{name: "cmd lone suffix", content: "@echo off\r\nrem " + OwnershipMarker + "\r\n\" %*\r\n"},
		{name: "cmd unbalanced quote", content: "@echo off\r\nrem " + OwnershipMarker + "\r\n\"C:\\a\"b\" %*\r\n"},
		{name: "powershell lone arguments", content: "# " + OwnershipMarker + "\r\n& @args\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := parseManagedLauncher(tt.content); ok || got != "" {
				t.Fatalf("parseManagedLauncher() = %q, %t, want not recognized", got, ok)
			}
		})
	}
}

func TestManagedLauncherTargetReadsFilesWithoutExecutingThem(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
		return path
	}

	if got, ok := ManagedLauncherTarget(write("opencode.cmd", windowsCMDLauncherFixture)); !ok || got != `C:\Users\dev\AppData\Roaming\npm\opencode.cmd` {
		t.Fatalf("ManagedLauncherTarget(launcher) = %q, %t", got, ok)
	}
	if got, ok := ManagedLauncherTarget(write("plain", "just a binary\n")); ok || got != "" {
		t.Fatalf("ManagedLauncherTarget(plain file) = %q, %t, want not recognized", got, ok)
	}
	if got, ok := ManagedLauncherTarget(filepath.Join(dir, "missing")); ok || got != "" {
		t.Fatalf("ManagedLauncherTarget(missing) = %q, %t, want not recognized", got, ok)
	}
	if got, ok := ManagedLauncherTarget(dir); ok || got != "" {
		t.Fatalf("ManagedLauncherTarget(directory) = %q, %t, want not recognized", got, ok)
	}
}

// TestManagedLauncherTargetIgnoresTheTailBeyondTheProbe proves the read is
// bounded: an invocation that only exists past the probe is never seen.
func TestManagedLauncherTargetIgnoresTheTailBeyondTheProbe(t *testing.T) {
	filler := strings.Repeat("rem padding\r\n", launcherProbeBytes/len("rem padding\r\n")+1)
	content := "@echo off\r\nrem " + OwnershipMarker + "\r\n" + filler + "\"C:\\npm\\opencode.cmd\" %*\r\n"
	if len(content) <= launcherProbeBytes {
		t.Fatalf("fixture is %d bytes, want it to exceed the %d byte probe", len(content), launcherProbeBytes)
	}
	path := filepath.Join(t.TempDir(), "opencode.cmd")
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, ok := ManagedLauncherTarget(path); ok || got != "" {
		t.Fatalf("ManagedLauncherTarget(oversized) = %q, %t, want not recognized", got, ok)
	}
}
