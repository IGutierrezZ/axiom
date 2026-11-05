package update

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

// extractPowerShellBlock returns the script text from the line starting with
// startMarker up to (not including) the line starting with endMarker.
func extractPowerShellBlock(t *testing.T, script, startMarker, endMarker string) string {
	t.Helper()
	start := strings.Index(script, startMarker)
	if start == -1 {
		t.Fatalf("scripts/install.ps1 is missing %q", startMarker)
	}
	end := strings.Index(script[start:], endMarker)
	if end == -1 {
		t.Fatalf("scripts/install.ps1 is missing %q after %q", endMarker, startMarker)
	}
	return script[start : start+end]
}

func encodePowerShellCommand(script string) string {
	units := utf16.Encode([]rune(script))
	buf := make([]byte, 0, len(units)*2)
	for _, u := range units {
		buf = append(buf, byte(u), byte(u>>8))
	}
	return base64.StdEncoding.EncodeToString(buf)
}

// install.ps1 must follow the same user PATH criterion as internal/system/path.go:
// read the raw value without expanding %VARS%, keep the registry kind and never
// add the same entry twice. The test runs the real functions against a scratch
// key under HKCU\Software, never against HKCU\Environment.
func TestInstallPs1AddToUserPathPreservesRegistryKindAndIsIdempotent(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the user PATH registry is Windows only")
	}
	powershell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Skipf("powershell.exe required: %v", err)
	}
	script := readRepositoryFile(t, "scripts", "install.ps1")
	functions := extractPowerShellBlock(t, script, "function ConvertTo-PathKey {", "# Makes sure $Dir is reachable")

	subKey := fmt.Sprintf(`Software\AxiomInstallTest-%d`, os.Getpid())
	body := functions + `
$ErrorActionPreference = 'Stop'
$sub = '` + subKey + `'
function Seed($value, $kind) {
    $k = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey($sub)
    try {
        if ($null -eq $value) { if ($k.GetValueNames() -contains 'Path') { $k.DeleteValue('Path') } }
        else { $k.SetValue('Path', $value, $kind) }
    } finally { $k.Close() }
}
function Probe($name, $dir) {
    $result = Add-ToUserPath -Dir $dir -SubKey $sub
    $k = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey($sub)
    try {
        $raw = [string]$k.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        $kind = if ($k.GetValueNames() -contains 'Path') { [string]$k.GetValueKind('Path') } else { 'none' }
    } finally { $k.Close() }
    '{0}|{1}|{2}|{3}' -f $name, $result, $raw, $kind
}
try {
    Seed '%USERPROFILE%\bin;C:\Tools\' ([Microsoft.Win32.RegistryValueKind]::ExpandString)
    Probe 'expand-add' 'C:\axiom\bin'
    Probe 'expand-again' 'c:\AXIOM\bin\'
    Probe 'expand-trailing-case' 'c:\tools'
    Probe 'expand-variable-form' ([Environment]::ExpandEnvironmentVariables('%USERPROFILE%\bin'))
    Seed 'C:\Tools' ([Microsoft.Win32.RegistryValueKind]::String)
    Probe 'string-add' 'C:\axiom\bin'
    Seed $null $null
    Probe 'missing-add' 'C:\axiom\bin'
} finally {
    [Microsoft.Win32.Registry]::CurrentUser.DeleteSubKeyTree($sub, $false)
}
`
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, powershell, "-NoProfile", "-NonInteractive", "-EncodedCommand", encodePowerShellCommand(body))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run Add-ToUserPath fixture: %v\n%s", err, out)
	}

	got := map[string]string{}
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r", ""), "\n") {
		if name, rest, ok := strings.Cut(strings.TrimSpace(line), "|"); ok {
			got[name] = rest
		}
	}
	want := map[string]string{
		"expand-add":           `added|C:\axiom\bin;%USERPROFILE%\bin;C:\Tools\|ExpandString`,
		"expand-again":         `present|C:\axiom\bin;%USERPROFILE%\bin;C:\Tools\|ExpandString`,
		"expand-trailing-case": `present|C:\axiom\bin;%USERPROFILE%\bin;C:\Tools\|ExpandString`,
		"expand-variable-form": `present|C:\axiom\bin;%USERPROFILE%\bin;C:\Tools\|ExpandString`,
		"string-add":           `added|C:\axiom\bin;C:\Tools|String`,
		"missing-add":          `added|C:\axiom\bin|ExpandString`,
	}
	for name, w := range want {
		if got[name] != w {
			t.Errorf("%s = %q, want %q\nfull output:\n%s", name, got[name], w, out)
		}
	}
}

func TestInstallPs1HonorsNoPersistentPathAndWarnsOnFailure(t *testing.T) {
	script := readRepositoryFile(t, "scripts", "install.ps1")
	block := extractPowerShellBlock(t, script, "function Confirm-UserPath {", "function Add-GoEnvPattern")
	for _, want := range []string{`AXIOM_NO_PERSISTENT_PATH`, `$env:Path = `, `Open a new terminal`, `is not in your PATH`} {
		if !strings.Contains(block, want) {
			t.Errorf("Confirm-UserPath is missing %q", want)
		}
	}
	for _, forbidden := range []string{`SetEnvironmentVariable`, `GetEnvironmentVariable("Path", "User")`} {
		if strings.Contains(script, forbidden) {
			t.Errorf("install.ps1 must not use %s: it expands variables and rewrites the value as REG_SZ", forbidden)
		}
	}
}
