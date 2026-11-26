package upgrade

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/update"
)

func writeFileContent(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

// writeStagedSelfBinary is for tests that stub `go install` for the Windows
// self-install: the real command builds into a staging directory inside the
// destination, so the stub leaves the binary there, as `go install` would.
func writeStagedSelfBinary(t *testing.T, destDir string) {
	t.Helper()
	stages, _ := filepath.Glob(filepath.Join(destDir, selfInstallStagingPrefix+"*"))
	for _, stage := range stages {
		writeFakeBinary(t, stage, "axiom.exe")
	}
}

func readFileContent(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// stubSelfGoInstall emulates a `go install` that succeeds and, when build is
// true, leaves a binary with the given content in the staging directory. The
// binary is written when the staging directory is created, which happens before
// the command runs. It returns the staging directory it created.
func stubSelfGoInstall(t *testing.T, content string, build bool) *string {
	t.Helper()
	original := execCommand
	t.Cleanup(func() { execCommand = original })
	execCommand = func(string, ...string) *exec.Cmd { return mockCmd("true") }
	var gobin string
	originalMkdir := selfInstallMkdirTemp
	t.Cleanup(func() { selfInstallMkdirTemp = originalMkdir })
	selfInstallMkdirTemp = func(dir, pattern string) (string, error) {
		stage, err := os.MkdirTemp(dir, pattern)
		if err == nil {
			gobin = stage
			if build {
				writeFileContent(t, filepath.Join(stage, "axiom.exe"), content)
			}
		}
		return stage, err
	}
	return &gobin
}

func TestGoInstallSelfWindowsStagesTheBuildAndSwapsItIn(t *testing.T) {
	destDir := t.TempDir()
	dst := filepath.Join(destDir, "axiom.exe")
	writeFileContent(t, dst, "old")
	// A `~` copy left by an earlier `go install`; on Windows this test keeps it
	// open, which is what used to make the next install fail.
	tilde := dst + "~"
	writeFileContent(t, tilde, "older")
	held, err := os.Open(tilde)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()

	gobin := stubSelfGoInstall(t, "new", true)
	if err := goInstallSelfWindows(update.ToolInfo{Name: "axiom"}, "example.com/axiom/cmd/axiom@main", nil, destDir); err != nil {
		t.Fatalf("goInstallSelfWindows: %v", err)
	}

	if got := readFileContent(t, dst); got != "new" {
		t.Fatalf("%s = %q, want the new binary", dst, got)
	}
	if *gobin == "" || filepath.Dir(*gobin) != destDir || !strings.HasPrefix(filepath.Base(*gobin), selfInstallStagingPrefix) {
		t.Fatalf("staging directory = %q, want a %s* directory inside %s", *gobin, selfInstallStagingPrefix, destDir)
	}
	if _, err := os.Stat(*gobin); !os.IsNotExist(err) {
		t.Fatalf("staging directory %s must be removed after the install, stat err = %v", *gobin, err)
	}
	leftovers, _ := filepath.Glob(dst + ".*.old")
	if len(leftovers) != 0 {
		t.Fatalf("the previous binary is not in use and must be removed, found %v", leftovers)
	}
}

func TestGoInstallSelfWindowsPassesGOBINAndTheCallerEnvironment(t *testing.T) {
	destDir := t.TempDir()
	writeFileContent(t, filepath.Join(destDir, "axiom.exe"), "old")
	gobin := stubSelfGoInstall(t, "new", true)

	var gotCmd *exec.Cmd
	execCommand = func(name string, args ...string) *exec.Cmd {
		gotCmd = mockCmd("true")
		if name != "go" || len(args) != 2 || args[0] != "install" || args[1] != "example.com/axiom/cmd/axiom@main" {
			t.Fatalf("command = %s %v, want go install example.com/axiom/cmd/axiom@main", name, args)
		}
		return gotCmd
	}

	env := []string{"GONOSUMDB=example.com/axiom"}
	if err := goInstallSelfWindows(update.ToolInfo{Name: "axiom"}, "example.com/axiom/cmd/axiom@main", env, destDir); err != nil {
		t.Fatalf("goInstallSelfWindows: %v", err)
	}
	if gotCmd == nil {
		t.Fatal("go install was not run")
	}
	if !envContains(gotCmd.Env, "GOBIN="+*gobin) {
		t.Fatalf("go install env must set GOBIN to the staging directory %s, got %v", *gobin, gotCmd.Env)
	}
	if !envContains(gotCmd.Env, "GONOSUMDB=example.com/axiom") {
		t.Fatalf("go install env must keep the caller environment, got %v", gotCmd.Env)
	}
	if len(env) != 1 {
		t.Fatalf("the caller environment slice must not be modified, got %v", env)
	}
}

func TestGoInstallSelfWindowsFailsWithoutTouchingTheBinaryWhenNothingWasBuilt(t *testing.T) {
	destDir := t.TempDir()
	dst := filepath.Join(destDir, "axiom.exe")
	writeFileContent(t, dst, "old")
	stubSelfGoInstall(t, "", false)

	err := goInstallSelfWindows(update.ToolInfo{Name: "axiom"}, "example.com/axiom/cmd/axiom@v1.2.3", nil, destDir)
	if err == nil || !strings.Contains(err.Error(), "the built binary was not found") {
		t.Fatalf("err = %v, want the missing build to be reported", err)
	}
	if got := readFileContent(t, dst); got != "old" {
		t.Fatalf("%s = %q, the previous binary must stay untouched", dst, got)
	}
}

func TestGoInstallSelfWindowsReportsAFailedBuild(t *testing.T) {
	destDir := t.TempDir()
	dst := filepath.Join(destDir, "axiom.exe")
	writeFileContent(t, dst, "old")
	stubSelfGoInstall(t, "", false)
	execCommand = func(name string, args ...string) *exec.Cmd { return mockCmd("false") }

	err := goInstallSelfWindows(update.ToolInfo{Name: "axiom"}, "example.com/axiom/cmd/axiom@v1.2.3", nil, destDir)
	if err == nil || !strings.Contains(err.Error(), "go install example.com/axiom/cmd/axiom@v1.2.3") {
		t.Fatalf("err = %v, want the go install failure", err)
	}
	if got := readFileContent(t, dst); got != "old" {
		t.Fatalf("%s = %q, the previous binary must stay untouched", dst, got)
	}
}

func TestSwapSelfBinaryRestoresThePreviousBinaryWhenTheMoveFails(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "axiom.exe")
	staged := filepath.Join(dir, "stage", "axiom.exe")
	writeFileContent(t, dst, "old")
	writeFileContent(t, staged, "new")

	original := renameFn
	t.Cleanup(func() { renameFn = original })
	renameFn = func(from, to string) error {
		if from == staged {
			return errors.New("sharing violation")
		}
		return os.Rename(from, to)
	}

	err := swapSelfBinary(staged, dst)
	if err == nil || !strings.Contains(err.Error(), "the previous binary was restored") {
		t.Fatalf("err = %v, want the failure with the previous binary restored", err)
	}
	if got := readFileContent(t, dst); got != "old" {
		t.Fatalf("%s = %q, want the previous binary back in place", dst, got)
	}
}

func TestSwapSelfBinaryInstallsWhenThereIsNoPreviousBinary(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "axiom.exe")
	staged := filepath.Join(dir, "stage", "axiom.exe")
	writeFileContent(t, staged, "new")

	if err := swapSelfBinary(staged, dst); err != nil {
		t.Fatalf("swapSelfBinary: %v", err)
	}
	if got := readFileContent(t, dst); got != "new" {
		t.Fatalf("%s = %q, want the new binary", dst, got)
	}
}

func TestUniqueAsideNameSkipsNamesInUse(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "axiom.exe")
	original := selfInstallNow
	t.Cleanup(func() { selfInstallNow = original })
	selfInstallNow = func() time.Time { return time.Date(2026, 11, 26, 16, 15, 0, 0, time.UTC) }

	first := uniqueAsideName(dst)
	if !strings.HasSuffix(first, ".old") || !strings.HasPrefix(first, dst+".") {
		t.Fatalf("aside name = %q, want %s.<stamp>.old", first, dst)
	}
	writeFileContent(t, first, "taken")
	second := uniqueAsideName(dst)
	if second == first || !strings.HasSuffix(second, ".old") {
		t.Fatalf("aside name = %q, want a free name different from %q ending in .old", second, first)
	}
}
