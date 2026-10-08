package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The closed PATH of a sandbox must pass the very guard that protects it.
func TestClosedSandboxPathPassesTheIsolationGuard(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is required by the product and by the sandbox PATH")
	}
	userDir := filepath.Join(t.TempDir(), "user-bin")
	writeFakeAgent(t, userDir, "claude")
	t.Setenv("PATH", userDir+string(os.PathListSeparator)+filepath.Dir(gitPath))
	root := t.TempDir()
	sandbox := &Sandbox{Root: root, Home: filepath.Join(root, "home"), Binary: "gentle-ai"}
	if err := sandbox.checkIsolation(); err != nil {
		t.Fatalf("closed sandbox PATH still fails the isolation guard: %v", err)
	}
}

// The same defect, stated against the guard: the PATH the old env() produced
// (user directories first or last, sandbox directories prepended) must be
// rejected naming the agent and where it resolved.
func TestCheckAgentIsolationDetectsAnAgentOutsideTheSandboxRoot(t *testing.T) {
	root := t.TempDir()
	userDir := filepath.Join(t.TempDir(), "user-bin")
	leaked := writeFakeAgent(t, userDir, "claude")
	path := filepath.Join(root, "policy-runtime-bin") + string(os.PathListSeparator) + userDir

	err := checkAgentIsolation(path, root, runtime.GOOS, agentBinaryNames)
	if err == nil {
		t.Fatal("an inherited user PATH holding a claude outside the sandbox root passed the guard")
	}
	for _, want := range []string{"claude", leaked, root} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("guard error %q does not name %q", err, want)
		}
	}
}

func TestCheckAgentIsolationAcceptsAgentsInsideTheSandboxRoot(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "provider-bin")
	writeFakeAgent(t, bin, "claude")
	if err := checkAgentIsolation(bin, root, runtime.GOOS, agentBinaryNames); err != nil {
		t.Fatalf("a shim inside the sandbox root is the sandbox's own: %v", err)
	}
}

// The root may be spelled in its 8.3 short form (Windows temp dirs under a long
// user name are) while the PATH entry uses the long one, or the other way
// round. Both spellings name the same directory and must not trip the guard.
func TestCheckAgentIsolationAcceptsShortAndLongSpellingsOfTheRoot(t *testing.T) {
	root := t.TempDir()
	long, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct{ name, root, bin string }{
		{"root as given, PATH resolved", root, filepath.Join(long, "bin")},
		{"root resolved, PATH as given", long, filepath.Join(root, "bin")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			writeFakeAgent(t, filepath.Join(root, "bin"), "claude")
			if err := checkAgentIsolation(tt.bin, tt.root, runtime.GOOS, agentBinaryNames); err != nil {
				t.Fatalf("two spellings of the same root tripped the guard: %v", err)
			}
		})
	}
}

// Resolution is first-match-wins, exactly like the host: an outside agent that
// a sandbox shim shadows is unreachable, one that shadows the shim is not.
func TestCheckAgentIsolationFollowsResolutionOrder(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "bin")
	outside := filepath.Join(t.TempDir(), "user-bin")
	writeFakeAgent(t, inside, "codex")
	writeFakeAgent(t, outside, "codex")
	sep := string(os.PathListSeparator)

	if err := checkAgentIsolation(inside+sep+outside, root, runtime.GOOS, agentBinaryNames); err != nil {
		t.Fatalf("an outside agent shadowed by a sandbox shim is unreachable: %v", err)
	}
	if err := checkAgentIsolation(outside+sep+inside, root, runtime.GOOS, agentBinaryNames); err == nil {
		t.Fatal("an outside agent that shadows the sandbox shim passed the guard")
	}
}

func TestCheckAgentIsolationCoversEveryKnownAgentName(t *testing.T) {
	if len(agentBinaryNames) == 0 {
		t.Fatal("no agent names configured")
	}
	for _, name := range agentBinaryNames {
		t.Run(name, func(t *testing.T) {
			outside := filepath.Join(t.TempDir(), "user-bin")
			writeFakeAgent(t, outside, name)
			err := checkAgentIsolation(outside, t.TempDir(), runtime.GOOS, agentBinaryNames)
			if err == nil || !strings.Contains(err.Error(), `"`+name+`"`) {
				t.Fatalf("guard error = %v, want it to name %q", err, name)
			}
		})
	}
}

// A link inside the root that leads outside must not launder an outside agent
// into the sandbox. On Windows a junction makes EvalSymlinks fail for every file
// beneath it, so the guard has to treat "cannot resolve" as outside instead of
// trusting the raw path; on POSIX the symlink resolves and lands outside.
func TestCheckAgentIsolationRejectsALinkInsideTheRootThatLeadsOutside(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "user-bin")
	writeFakeAgent(t, outside, "claude")
	link := filepath.Join(root, "link")
	if runtime.GOOS == "windows" {
		if output, err := exec.Command("cmd", "/c", "mklink", "/J", link, outside).CombinedOutput(); err != nil {
			t.Skipf("cannot create a junction here: %v: %s", err, output)
		}
	} else if err := os.Symlink(outside, link); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	if err := checkAgentIsolation(link, root, runtime.GOOS, agentBinaryNames); err == nil {
		t.Fatal("an agent reached through a link that leaves the sandbox root passed the guard")
	}
}

// The fail-closed branch on its own: a file that cannot be resolved is outside,
// whatever its spelling says.
func TestPathWithinTreatsAnUnresolvablePathAsOutside(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing", "claude")
	if pathWithin(root, missing, runtime.GOOS) {
		t.Fatal("an unresolvable file under the root counted as inside it")
	}
	if pathWithin(filepath.Join(root, "missing"), filepath.Join(root, "x"), runtime.GOOS) {
		t.Fatal("an unresolvable root contained something")
	}
	present := writeFakeAgent(t, filepath.Join(root, "bin"), "claude")
	if !pathWithin(root, present, runtime.GOOS) {
		t.Fatal("a resolvable file under the root did not count as inside it")
	}
}

type irregularInfo struct{ mode fs.FileMode }

func (irregularInfo) Name() string        { return "claude.exe" }
func (irregularInfo) Size() int64         { return 0 }
func (i irregularInfo) Mode() fs.FileMode { return i.mode }
func (irregularInfo) ModTime() time.Time  { return time.Time{} }
func (i irregularInfo) IsDir() bool       { return i.mode.IsDir() }
func (irregularInfo) Sys() any            { return nil }

// Windows resolves anything that is not a directory (os/exec's chkStat), so a
// reparse point that is not a regular file still counts; POSIX demands a regular
// file with an execute bit.
func TestExecutableCandidateFollowsHostRules(t *testing.T) {
	for _, tt := range []struct {
		name string
		goos string
		mode fs.FileMode
		want bool
	}{
		{"windows regular file", "windows", 0o644, true},
		{"windows irregular file", "windows", fs.ModeIrregular | 0o644, true},
		{"windows symlink entry", "windows", fs.ModeSymlink | 0o644, true},
		{"windows directory", "windows", fs.ModeDir | 0o755, false},
		{"linux executable file", "linux", 0o755, true},
		{"linux file without execute bit", "linux", 0o644, false},
		{"linux irregular executable", "linux", fs.ModeIrregular | 0o755, false},
		{"linux directory", "linux", fs.ModeDir | 0o755, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := executableCandidate(irregularInfo{mode: tt.mode}, tt.goos); got != tt.want {
				t.Fatalf("executableCandidate(%s, %v) = %v, want %v", tt.goos, tt.mode, got, tt.want)
			}
		})
	}
}

// Host resolution rules, exercised for the Windows spelling on any OS: a bare
// extensionless file is not executable there (which is exactly why a POSIX
// shim never ran), while an .exe/.cmd is.
func TestLookPathInFollowsHostExecutableRules(t *testing.T) {
	dir := t.TempDir()
	bare := filepath.Join(dir, "claude")
	if err := os.WriteFile(bare, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := lookPathIn(dir, "claude", "windows"); got != "" {
		t.Fatalf("windows resolved an extensionless file: %q", got)
	}
	if runtime.GOOS != "windows" {
		if got := lookPathIn(dir, "claude", "linux"); got != bare {
			t.Fatalf("linux did not resolve an executable file: %q", got)
		}
		if err := os.Chmod(bare, 0o644); err != nil {
			t.Fatal(err)
		}
		if got := lookPathIn(dir, "claude", "linux"); got != "" {
			t.Fatalf("linux resolved a file without execute bits: %q", got)
		}
	}
	cmd := filepath.Join(dir, "gemini.cmd")
	if err := os.WriteFile(cmd, []byte("@echo off\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := lookPathIn(dir, "gemini", "windows"); got != cmd {
		t.Fatalf("windows did not resolve gemini.cmd: %q", got)
	}
	if err := os.Mkdir(filepath.Join(dir, "pi.exe"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := lookPathIn(dir, "pi", "windows"); got != "" {
		t.Fatalf("windows resolved a directory named pi.exe: %q", got)
	}
	if got := lookPathIn(filepath.Join(dir, "missing"), "gemini", "windows"); got != "" {
		t.Fatalf("resolved from a directory that does not exist: %q", got)
	}
}

// A PATH entry that is empty or relative resolves against whatever the working
// directory happens to be, so it can never be proven to sit inside the sandbox.
func TestCheckAgentIsolationTreatsRelativeEntriesAsOutside(t *testing.T) {
	root := t.TempDir()
	work := t.TempDir()
	writeFakeAgent(t, filepath.Join(work, "rel"), "pi")
	t.Chdir(work)
	if err := checkAgentIsolation("rel", root, runtime.GOOS, agentBinaryNames); err == nil {
		t.Fatal("an agent reachable through a relative PATH entry passed the guard")
	}
}

// The list lives in this module because bench cannot import the product. The
// product's own lookups are the source of truth, so a new adapter that resolves
// a new binary must fail here until the guard learns its name.
func TestAgentBinaryNamesCoverEveryBinaryTheProductResolves(t *testing.T) {
	lookup := regexp.MustCompile(`(?i)lookPath\(\s*"([^"]+)"\s*\)`)
	known := map[string]bool{}
	for _, name := range agentBinaryNames {
		known[name] = true
	}
	// `code` is the VS Code editor host the vscode adapter probes; it is never
	// executed as an agent and sits in /usr/bin on ordinary Debian installs.
	exempt := map[string]bool{"code": true}
	scanned := 0
	for _, dir := range []string{"agents", "reviewerprovider", "agentbuilder"} {
		root := filepath.Join("..", "internal", dir)
		if _, err := os.Stat(root); err != nil {
			t.Skipf("product sources are not next to the bench module: %v", err)
		}
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return walkErr
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, match := range lookup.FindAllStringSubmatch(string(content), -1) {
				scanned++
				if name := match[1]; !known[name] && !exempt[name] {
					t.Errorf("%s resolves %q on PATH but agentBinaryNames does not list it", path, name)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if scanned == 0 {
		t.Fatal("found no lookPath(\"...\") call in the product sources: the pattern drifted")
	}
}

func TestRunJourneyAbortsWhenAFixtureLeaksARealAgent(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "user-bin")
	writeFakeAgent(t, outside, "claude")
	ran := false
	journey := Journey{ID: "jLeak", Review: reviewUntouched, Steps: []Step{
		{Name: "fixture points the sandbox at a user directory", Fixture: func(sandbox *Sandbox) error {
			sandbox.PathOverride = outside
			return nil
		}},
		{Name: "must never run", Fixture: func(*Sandbox) error { ran = true; return nil }},
	}}
	result := runJourney("unused-binary", journey)
	if result.Status != StatusFailed {
		t.Fatalf("status = %s, want failed: a leaked agent must never degrade to pass or unsupported", result.Status)
	}
	if !strings.Contains(result.FailureReason, `"claude"`) || !strings.Contains(result.FailureReason, "not isolated") {
		t.Fatalf("failure reason = %q, want a clear isolation error naming the agent", result.FailureReason)
	}
	if ran {
		t.Fatal("a step ran after the isolation guard failed")
	}
}

func runWithBasePath(t *testing.T, path string) (int, Results, bool) {
	t.Helper()
	t.Setenv("PATH", path)
	ran := false
	journeys := func() []Journey {
		return []Journey{{ID: "known", Review: reviewUntouched, Steps: []Step{{
			Name: "must never run", Fixture: func(*Sandbox) error { ran = true; return nil },
		}}}}
	}
	out := filepath.Join(t.TempDir(), "results.json")
	exit := commandRunWith([]string{"--binary", "unused-binary", "--out", out}, func(string) bool { return true }, journeys)
	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("the evidence of the refusal must be written: %v", err)
	}
	var results Results
	if err := json.Unmarshal(content, &results); err != nil {
		t.Fatal(err)
	}
	return exit, results, ran
}

// Whole-run abort: when the directory git resolves in also holds an agent, no
// sandbox can be both useful and isolated, so nothing may run.
func TestCommandRunAbortsBeforeAnyJourneyWhenGitSharesADirectoryWithAnAgent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "mixed-bin")
	writeFakeAgent(t, dir, "git")
	writeFakeAgent(t, dir, "claude")
	exit, results, ran := runWithBasePath(t, dir)
	if exit != 1 || ran {
		t.Fatalf("exit = %d, journey ran = %v, want exit 1 and no journey", exit, ran)
	}
	if results.RunStatus != "failed" || !strings.HasPrefix(results.FailureReason, "sandbox_path_not_isolated: ") || !strings.Contains(results.FailureReason, `"claude"`) {
		t.Fatalf("run_status = %q, failure_reason = %q", results.RunStatus, results.FailureReason)
	}
}

// A missing git is an environment problem, not an isolation failure, and its
// label must say so.
func TestCommandRunReportsAMissingGitWithItsOwnReason(t *testing.T) {
	exit, results, ran := runWithBasePath(t, t.TempDir())
	if exit != 1 || ran {
		t.Fatalf("exit = %d, journey ran = %v, want exit 1 and no journey", exit, ran)
	}
	if results.RunStatus != "failed" || !strings.HasPrefix(results.FailureReason, "sandbox_git_unavailable: ") ||
		strings.Contains(results.FailureReason, "sandbox_path_not_isolated") {
		t.Fatalf("run_status = %q, failure_reason = %q", results.RunStatus, results.FailureReason)
	}
}
