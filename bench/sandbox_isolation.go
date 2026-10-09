package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// The guard behind the closed sandbox PATH (see Sandbox.path): a PATH that can
// still resolve a real agent outside the sandbox must stop the run, never be
// measured and never degrade to a skip.

// agentBinaryNames are the executables an agent runtime is resolved by. They are
// the names the product itself looks up on PATH — the adapters in
// internal/agents (claude, codex, opencode, gemini, pi, kiro, kilo, kimi, qwen,
// hermes, openclaw), the reviewer providers (claude, codex) and the agent builder
// (claude, opencode, gemini, codex). bench cannot import the product, so
// TestAgentBinaryNamesCoverEveryBinaryTheProductResolves re-reads those sources
// and fails when a name is missing here.
//
// kiro-cli, cursor-agent and copilot are not resolved by the product today; they
// are listed because they are the other agent CLIs a user's PATH commonly holds,
// so a future adapter for them cannot slip past the guard. The VS Code host
// `code` is deliberately absent: it is probed for detection, never run as an
// agent, and it lives in /usr/bin on ordinary Debian installs, which would make
// every developer machine fail the guard.
var agentBinaryNames = []string{
	"claude", "codex", "opencode", "gemini", "pi",
	"kiro", "kiro-cli", "kilo", "kimi", "qwen", "hermes", "openclaw",
	"cursor-agent", "copilot",
}

// windowsDefaultPathExt is what a Go process resolves with when PATHEXT is
// unset, which is the sandbox children's case: env() does not carry PATHEXT.
var windowsDefaultPathExt = []string{".com", ".exe", ".bat", ".cmd"}

// errGitUnavailable marks a missing git, which is an environment problem and not
// an isolation failure.
var errGitUnavailable = errors.New("git is not on PATH: the product shells out to git, so no sandbox can be built")

// checkIsolation is the fail-closed guard: it errors when any known agent
// resolves, on this sandbox's PATH, to a file outside the sandbox root.
func (s *Sandbox) checkIsolation() error {
	return checkAgentIsolation(s.path(), s.Root, runtime.GOOS, agentBinaryNames)
}

// checkSandboxBasePath guards the part of the PATH every sandbox shares, before
// any journey exists, and requires the git the product shells out to. A missing
// git is errGitUnavailable; an agent next to it is an isolation failure.
func checkSandboxBasePath() error {
	gitDir := sandboxGitDir()
	if gitDir == "" {
		return errGitUnavailable
	}
	path := strings.Join(closedPathEntries(runtime.GOOS, gitDir), string(os.PathListSeparator))
	return checkAgentIsolation(path, "", runtime.GOOS, agentBinaryNames)
}

// checkAgentIsolation errors when any of names resolves on pathList to a file
// that is not provably under root. Resolution follows the host: first match
// wins, so an outside agent that a sandbox shim shadows is unreachable. An empty
// root puts every resolution outside it.
func checkAgentIsolation(pathList, root, goos string, names []string) error {
	var leaks []string
	for _, name := range names {
		found := lookPathIn(pathList, name, goos)
		if found == "" || (root != "" && pathWithin(root, found, goos)) {
			continue
		}
		leaks = append(leaks, fmt.Sprintf("agent %q resolves to %s", name, found))
	}
	if len(leaks) == 0 {
		return nil
	}
	where := "outside the sandbox root " + root
	if root == "" {
		where = "in the PATH shared by every sandbox, which may hold no agent"
	}
	return fmt.Errorf("sandbox PATH is not isolated from the host: %s, %s; refusing to run, because a journey step would drive a real agent",
		strings.Join(leaks, "; "), where)
}

// lookPathIn is exec.LookPath against an explicit PATH string. It returns the
// first executable file called name, or "". Empty and relative entries are kept
// as they are: they resolve against the working directory, so a hit through one
// is returned as a relative path and never counts as inside a root.
func lookPathIn(pathList, name, goos string) string {
	for _, dir := range filepath.SplitList(pathList) {
		if dir == "" {
			dir = "."
		}
		for _, candidate := range pathCandidates(name, goos) {
			file := filepath.Join(dir, candidate)
			if info, err := os.Stat(file); err == nil && executableCandidate(info, goos) {
				return file
			}
		}
	}
	return ""
}

// executableCandidate follows the host's own rule. Windows accepts anything that
// is not a directory, exactly like os/exec's chkStat in lp_windows.go: the
// extension is what makes a file executable there, and links or reparse points
// that are not regular files still run. POSIX needs a regular file with an
// execute bit.
func executableCandidate(info fs.FileInfo, goos string) bool {
	if goos == "windows" {
		return !info.IsDir()
	}
	return info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
}

// pathCandidates are the file names the host would try for name: on Windows
// only the PATHEXT spellings, never the bare name.
func pathCandidates(name, goos string) []string {
	if goos != "windows" {
		return []string{name}
	}
	candidates := make([]string, 0, len(windowsDefaultPathExt))
	for _, ext := range windowsDefaultPathExt {
		candidates = append(candidates, name+ext)
	}
	return candidates
}

// pathWithin reports whether file provably lies under root once both are
// resolved through symlinks (macOS temp roots sit behind one). It fails closed:
// a relative file, or any path that cannot be resolved, is not within. That
// matters on Windows, where a junction inside the root that points elsewhere
// makes EvalSymlinks fail for every file beneath it: falling back to the raw
// path would call a file outside the root inside it.
func pathWithin(root, file, goos string) bool {
	if !filepath.IsAbs(file) {
		return false
	}
	resolve := func(path string) (string, bool) {
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", false
		}
		resolved = filepath.Clean(resolved)
		if goos == "windows" {
			resolved = strings.ToLower(resolved)
		}
		return resolved, true
	}
	resolvedRoot, rootOK := resolve(root)
	resolvedFile, fileOK := resolve(file)
	if !rootOK || !fileOK {
		return false
	}
	rel, err := filepath.Rel(resolvedRoot, resolvedFile)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// sandboxBaseFailureReason is the run-level failure_reason for a base PATH that
// cannot be used: a missing git and a PATH holding an agent are different
// problems and must not share a label.
func sandboxBaseFailureReason(err error) string {
	if errors.Is(err, errGitUnavailable) {
		return "sandbox_git_unavailable"
	}
	return "sandbox_path_not_isolated"
}
