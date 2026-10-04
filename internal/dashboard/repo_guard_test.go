package dashboard

import (
	"context"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// repoGuard records `git status --porcelain --ignored` of the checkout the
// tests run in, so the run can prove it left the checkout as it found it.
//
// Known limitations:
//   - It can false-positive when another process writes to the checkout during
//     the run (an editor, another agent session in the same checkout, or another
//     package's tests under `go test ./...`). The report lists the exact entries,
//     so such a case can be told apart from a real regression.
//   - It compares status entries, so it sees added, removed or re-classified
//     paths (including ignored ones) but not a rewrite of a file that was already
//     dirty, and an untracked or ignored directory that already exists is one
//     collapsed entry.
//
// That is enough to catch the regression this guards (a test running sync,
// install or an index regeneration against the repository root) on a clean
// checkout such as CI or a fresh worktree. It does nothing when git is missing
// or the tests do not run inside a work tree.
type repoGuard struct {
	before  []string
	enabled bool
}

func newRepoGuard() *repoGuard {
	inside, ok := runGitStatusCommand("rev-parse", "--is-inside-work-tree")
	if !ok || strings.TrimSpace(inside) != "true" {
		return &repoGuard{}
	}
	status, ok := runGitStatusCommand("status", "--porcelain", "--ignored")
	if !ok {
		return &repoGuard{}
	}
	return &repoGuard{before: statusLines(status), enabled: true}
}

// changes returns a human-readable report of the status entries that differ
// from the snapshot taken before the tests, or "" when nothing changed (or the
// guard is disabled, or the second snapshot could not be taken).
func (g *repoGuard) changes() string {
	if !g.enabled {
		return ""
	}
	status, ok := runGitStatusCommand("status", "--porcelain", "--ignored")
	if !ok {
		return ""
	}
	after := statusLines(status)

	var report []string
	for _, line := range subtractLines(after, g.before) {
		report = append(report, "  + "+line)
	}
	for _, line := range subtractLines(g.before, after) {
		report = append(report, "  - "+line)
	}
	if len(report) == 0 {
		return ""
	}
	return "dashboard tests modified the checkout they ran in (git status --porcelain --ignored):\n" +
		strings.Join(report, "\n") +
		"\nUsually a test is running a real sync/install/index flow against the repository: give it a t.TempDir() project." +
		"\nIf no test does, something else wrote to the checkout while the tests ran."
}

// runGitStatusCommand runs a read-only git command in the package directory.
// GIT_OPTIONAL_LOCKS=0 stops `git status` from refreshing the index, so the
// guard itself never writes to the checkout or contends with other git users.
func runGitStatusCommand(args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

func statusLines(status string) []string {
	var lines []string
	for _, line := range strings.Split(status, "\n") {
		if line = strings.TrimRight(line, "\r"); line != "" {
			lines = append(lines, line)
		}
	}
	sort.Strings(lines)
	return lines
}

// subtractLines returns the entries of a that are not in b.
func subtractLines(a, b []string) []string {
	inB := make(map[string]struct{}, len(b))
	for _, line := range b {
		inB[line] = struct{}{}
	}
	var diff []string
	for _, line := range a {
		if _, ok := inB[line]; !ok {
			diff = append(diff, line)
		}
	}
	return diff
}
