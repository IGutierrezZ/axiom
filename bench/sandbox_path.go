package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// The sandbox PATH is a closed allow-list, never the user's PATH.
//
// It used to inherit the whole parent PATH. Every journey that needs an agent
// ships a POSIX `#!/bin/sh` shim for it, and a host that cannot execute that
// shim (Windows resolves only PATHEXT extensions) quietly fell through to the
// REAL agent installed in a user directory: a real `claude` answered "Not logged
// in" inside a journey. Closing the PATH removes the fall-through.

// posixShimSystemDirs hold what the POSIX shims need and nothing else: /bin/sh
// is reached through the shebang, and the shims call cat, sed and head by name.
// These are the only system directories the sandbox PATH carries.
var posixShimSystemDirs = []string{"/usr/bin", "/bin"}

// closedPathEntries lists the directories, besides the sandbox's own, that the
// sandbox PATH may carry.
//
//   - The directory `git` resolves in: the product shells out to git. The whole
//     directory is added, so whatever else lives there is reachable too. On
//     Windows it is Git\cmd, whose git.exe locates mingw64 relative to itself, so
//     it can be neither copied nor linked.
//   - POSIX only: /usr/bin and /bin, for the shims' cat/sed/head. Windows cannot
//     execute the POSIX shims, so it needs no system directory.
//
// `go`, `node` and the like are deliberately absent: nothing a journey runs
// under this PATH resolves them (node, for the SDD task-result axis, is looked
// up by the harness on the parent's PATH and executed by absolute path).
func closedPathEntries(goos, gitDir string) []string {
	var entries []string
	add := func(dir string) {
		if dir == "" {
			return
		}
		for _, existing := range entries {
			if existing == dir {
				return
			}
		}
		entries = append(entries, dir)
	}
	add(gitDir)
	if goos != "windows" {
		for _, dir := range posixShimSystemDirs {
			add(dir)
		}
	}
	return entries
}

// sandboxGitDir is the directory the parent resolves `git` in, or "" when git is
// not installed. It is not cached: tests change PATH between calls.
func sandboxGitDir() string {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return ""
	}
	if abs, err := filepath.Abs(gitPath); err == nil {
		gitPath = abs
	}
	return filepath.Dir(gitPath)
}

// path is the PATH every process of this sandbox receives: the journey's own
// override first, then the policy runtime fixture, then the closed allow-list.
func (s *Sandbox) path() string {
	var entries []string
	if s.PathOverride != "" {
		entries = append(entries, s.PathOverride)
	}
	if s.policyRuntimeBin != "" {
		entries = append(entries, s.policyRuntimeBin)
	}
	entries = append(entries, closedPathEntries(runtime.GOOS, sandboxGitDir())...)
	return strings.Join(entries, string(os.PathListSeparator))
}
