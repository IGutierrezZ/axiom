package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	opencodeactivation "github.com/IGutierrezZ/axiom/v3/internal/opencode"
)

// stubDoctorToolEnv points the doctor seams at goos and the given executable
// extensions, with lookPathFn resolving to resolved, until the test ends.
func stubDoctorToolEnv(t *testing.T, goos string, exts []string, resolved string) {
	t.Helper()
	origLook, origGOOS, origExts := lookPathFn, doctorGOOS, executableExtsFn
	t.Cleanup(func() {
		lookPathFn, doctorGOOS, executableExtsFn = origLook, origGOOS, origExts
	})
	doctorGOOS = goos
	executableExtsFn = func() []string { return exts }
	lookPathFn = func(string) (string, error) { return resolved, nil }
}

func writeDoctorFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// cmdManagedLauncher is the launcher Axiom writes on Windows, in the form
// existing installs carry it.
func cmdManagedLauncher(target string) string {
	return "@echo off\r\n" +
		"rem " + opencodeactivation.OwnershipMarker + "\r\n" +
		"setlocal\r\n" +
		"if not defined OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS set \"OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS=true\"\r\n" +
		"\"" + strings.ReplaceAll(target, `"`, `""`) + "\" %*\r\n" +
		"exit /b %ERRORLEVEL%\r\n"
}

// shManagedLauncher is the launcher Axiom writes on POSIX systems.
func shManagedLauncher(target string) string {
	return "#!/bin/sh\n# " + opencodeactivation.OwnershipMarker + "\nset -eu\n" +
		"if [ -z \"${OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS+x}\" ]; then\n  export OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS=true\nfi\n" +
		"exec '" + strings.ReplaceAll(target, "'", `'\''`) + "' \"$@\"\n"
}

func TestCheckOneTool_ManagedOpenCodeLauncherWrappingACopyIsNotADuplicate(t *testing.T) {
	axiomBin, npmBin := t.TempDir(), t.TempDir()
	npmCopy := writeDoctorFile(t, npmBin, "opencode.cmd", "@echo off\r\nrem real opencode\r\n")
	launcher := writeDoctorFile(t, axiomBin, "opencode.cmd", cmdManagedLauncher(npmCopy))
	stubDoctorToolEnv(t, "windows", []string{".cmd"}, launcher)

	got := checkOneTool("opencode", []string{axiomBin, npmBin})

	if got.Status != CheckStatusPass {
		t.Fatalf("expected pass for a managed launcher wrapping the other copy, got %s: %s", got.Status, got.Detail)
	}
	if got.Remedy != nil {
		t.Errorf("expected no remedy, got %+v", got.Remedy)
	}
	if strings.Contains(got.Detail, "copies found") {
		t.Errorf("did not expect a duplicate warning, got %s", got.Detail)
	}
	for _, want := range []string{"opencode found at " + launcher, "Axiom-managed launcher " + launcher + " wraps " + npmCopy} {
		if !strings.Contains(got.Detail, want) {
			t.Errorf("detail %q does not contain %q", got.Detail, want)
		}
	}
}

func TestCheckOneTool_ManagedOpenCodeLauncherLaterInPathIsNotADuplicate(t *testing.T) {
	npmBin, axiomBin := t.TempDir(), t.TempDir()
	npmCopy := writeDoctorFile(t, npmBin, "opencode.cmd", "@echo off\r\n")
	launcher := writeDoctorFile(t, axiomBin, "opencode.cmd", cmdManagedLauncher(npmCopy))
	stubDoctorToolEnv(t, "windows", []string{".cmd"}, npmCopy)

	got := checkOneTool("opencode", []string{npmBin, axiomBin})

	if got.Status != CheckStatusPass {
		t.Fatalf("expected pass regardless of PATH order, got %s: %s", got.Status, got.Detail)
	}
	if !strings.Contains(got.Detail, "opencode found at "+npmCopy) || !strings.Contains(got.Detail, launcher) {
		t.Errorf("unexpected detail: %s", got.Detail)
	}
}

func TestCheckOneTool_ManagedPOSIXOpenCodeLauncherWrappingACopyIsNotADuplicate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix execute-bit semantics do not apply on Windows")
	}
	axiomBin, npmBin := t.TempDir(), t.TempDir()
	npmCopy := writeDoctorFile(t, npmBin, "opencode", "#!/bin/sh\nexit 0\n")
	launcher := writeDoctorFile(t, axiomBin, "opencode", shManagedLauncher(npmCopy))
	stubDoctorToolEnv(t, "linux", []string{""}, launcher)

	got := checkOneTool("opencode", []string{axiomBin, npmBin})

	if got.Status != CheckStatusPass {
		t.Fatalf("expected pass for a managed POSIX launcher wrapping the other copy, got %s: %s", got.Status, got.Detail)
	}
	if !strings.Contains(got.Detail, "Axiom-managed launcher "+launcher+" wraps "+npmCopy) {
		t.Errorf("unexpected detail: %s", got.Detail)
	}
}

func TestCheckOneTool_ManagedOpenCodeLauncherAndTwoRealCopiesStillWarn(t *testing.T) {
	axiomBin, npmBin, otherBin := t.TempDir(), t.TempDir(), t.TempDir()
	npmCopy := writeDoctorFile(t, npmBin, "opencode.cmd", "@echo off\r\nrem npm\r\n")
	otherCopy := writeDoctorFile(t, otherBin, "opencode.cmd", "@echo off\r\nrem other\r\n")
	launcher := writeDoctorFile(t, axiomBin, "opencode.cmd", cmdManagedLauncher(npmCopy))
	stubDoctorToolEnv(t, "windows", []string{".cmd"}, launcher)

	got := checkOneTool("opencode", []string{axiomBin, npmBin, otherBin})

	if got.Status != CheckStatusWarn {
		t.Fatalf("expected warn for a real duplicate beyond the launcher pair, got %s: %s", got.Status, got.Detail)
	}
	if want := "2 copies found in PATH: " + npmCopy + ", " + otherCopy; !strings.Contains(got.Detail, want) {
		t.Errorf("detail %q does not contain %q", got.Detail, want)
	}
	if !strings.Contains(got.Detail, "Axiom-managed launcher "+launcher+" wraps "+npmCopy) {
		t.Errorf("expected the launcher to be named as not counted, got %s", got.Detail)
	}
	if got.Remedy == nil {
		t.Error("expected non-empty remedy")
	}
}

func TestCheckOneTool_ManagedOpenCodeLauncherWrappingAnAbsentPathStillWarns(t *testing.T) {
	axiomBin, npmBin, elsewhere := t.TempDir(), t.TempDir(), t.TempDir()
	npmCopy := writeDoctorFile(t, npmBin, "opencode.cmd", "@echo off\r\n")
	launcher := writeDoctorFile(t, axiomBin, "opencode.cmd", cmdManagedLauncher(filepath.Join(elsewhere, "opencode.cmd")))
	stubDoctorToolEnv(t, "windows", []string{".cmd"}, launcher)

	got := checkOneTool("opencode", []string{axiomBin, npmBin})

	if got.Status != CheckStatusWarn {
		t.Fatalf("expected warn when the launcher wraps a path that is not among the copies, got %s: %s", got.Status, got.Detail)
	}
	if want := "2 copies found in PATH: " + launcher + ", " + npmCopy; !strings.Contains(got.Detail, want) {
		t.Errorf("detail %q does not contain %q", got.Detail, want)
	}
}

func TestCheckOneTool_UnmanagedOpenCodeCopiesStillWarn(t *testing.T) {
	dir1, dir2 := t.TempDir(), t.TempDir()
	first := writeDoctorFile(t, dir1, "opencode.cmd", "@echo off\r\nrem one\r\n")
	writeDoctorFile(t, dir2, "opencode.cmd", "@echo off\r\nrem two\r\n")
	stubDoctorToolEnv(t, "windows", []string{".cmd"}, first)

	got := checkOneTool("opencode", []string{dir1, dir2})

	if got.Status != CheckStatusWarn || !strings.Contains(got.Detail, "2 copies found") {
		t.Fatalf("expected the duplicate warning for two unmanaged copies, got %s: %s", got.Status, got.Detail)
	}
	if strings.Contains(got.Detail, "Axiom-managed launcher") {
		t.Errorf("did not expect a launcher note, got %s", got.Detail)
	}
}

// TestCheckOneTool_OpenCodeFilesThatAreNotGeneratedLaunchersStillWarn pins the
// recognition rule: only the exact header marker of the current launcher format
// counts, so a file that mentions the marker elsewhere, or another version of
// it, does not hide a duplicate.
func TestCheckOneTool_OpenCodeFilesThatAreNotGeneratedLaunchersStillWarn(t *testing.T) {
	marker := opencodeactivation.OwnershipMarker
	tests := []struct {
		name    string
		content func(target string) string
	}{
		{name: "marker outside the header", content: func(target string) string {
			return "@echo off\r\nsetlocal\r\nrem " + marker + "\r\n\"" + target + "\" %*\r\n"
		}},
		{name: "marker mentioned in prose", content: func(target string) string {
			return "@echo off\r\necho managed by " + marker + "\r\n\"" + target + "\" %*\r\n"
		}},
		{name: "another launcher version", content: func(target string) string {
			return strings.Replace(cmdManagedLauncher(target), "/v1", "/v2", 1)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			axiomBin, npmBin := t.TempDir(), t.TempDir()
			npmCopy := writeDoctorFile(t, npmBin, "opencode.cmd", "@echo off\r\n")
			launcher := writeDoctorFile(t, axiomBin, "opencode.cmd", tt.content(npmCopy))
			stubDoctorToolEnv(t, "windows", []string{".cmd"}, launcher)

			got := checkOneTool("opencode", []string{axiomBin, npmBin})

			if got.Status != CheckStatusWarn || !strings.Contains(got.Detail, "2 copies found") {
				t.Fatalf("expected the duplicate warning, got %s: %s", got.Status, got.Detail)
			}
		})
	}
}

func TestCheckOneTool_ManagedOpenCodeLaunchersNeverCancelEachOtherOut(t *testing.T) {
	dir1, dir2 := t.TempDir(), t.TempDir()
	first := filepath.Join(dir1, "opencode.cmd")
	second := filepath.Join(dir2, "opencode.cmd")
	writeDoctorFile(t, dir1, "opencode.cmd", cmdManagedLauncher(second))
	writeDoctorFile(t, dir2, "opencode.cmd", cmdManagedLauncher(first))
	stubDoctorToolEnv(t, "windows", []string{".cmd"}, first)

	got := checkOneTool("opencode", []string{dir1, dir2})

	if got.Status != CheckStatusWarn || !strings.Contains(got.Detail, "2 copies found") {
		t.Fatalf("expected the warning when launchers only wrap each other, got %s: %s", got.Status, got.Detail)
	}
}

func TestCheckOneTool_TwoManagedOpenCodeLaunchersWrappingTheSameCopyAreNotDuplicates(t *testing.T) {
	axiomBin, legacyBin, npmBin := t.TempDir(), t.TempDir(), t.TempDir()
	npmCopy := writeDoctorFile(t, npmBin, "opencode.cmd", "@echo off\r\n")
	launcher := writeDoctorFile(t, axiomBin, "opencode.cmd", cmdManagedLauncher(npmCopy))
	writeDoctorFile(t, legacyBin, "opencode.cmd", cmdManagedLauncher(npmCopy))
	stubDoctorToolEnv(t, "windows", []string{".cmd"}, launcher)

	got := checkOneTool("opencode", []string{axiomBin, legacyBin, npmBin})

	if got.Status != CheckStatusPass {
		t.Fatalf("expected pass when every extra copy is a launcher wrapping the real one, got %s: %s", got.Status, got.Detail)
	}
}

// TestCheckOneTool_OtherToolsDoNotGetLauncherExemption keeps the exemption to
// the one tool Axiom wraps: a different tool carrying the same header is still
// a duplicate.
func TestCheckOneTool_OtherToolsDoNotGetLauncherExemption(t *testing.T) {
	dir1, dir2 := t.TempDir(), t.TempDir()
	realCopy := writeDoctorFile(t, dir2, "engram.cmd", "@echo off\r\n")
	launcher := writeDoctorFile(t, dir1, "engram.cmd", cmdManagedLauncher(realCopy))
	stubDoctorToolEnv(t, "windows", []string{".cmd"}, launcher)

	got := checkOneTool("engram", []string{dir1, dir2})

	if got.Status != CheckStatusWarn || !strings.Contains(got.Detail, "2 copies found") {
		t.Fatalf("expected the duplicate warning for a non-opencode tool, got %s: %s", got.Status, got.Detail)
	}
}
