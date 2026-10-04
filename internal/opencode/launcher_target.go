package opencode

import (
	"io"
	"os"
	"strings"
)

// launcherProbeBytes bounds how much of a candidate file ManagedLauncherTarget
// reads. A generated launcher is a handful of short lines, so anything that does
// not fit is not one of ours and is never read in full.
const launcherProbeBytes = 4096

// ManagedLauncherTarget reports the OpenCode executable wrapped by the
// Axiom-managed launcher stored at path. ok is false when the file is not a
// launcher written by this package, in which case target is empty.
//
// It reads at most launcherProbeBytes from a regular file and never executes
// it. Recognition is stricter than the ownership check used when activating or
// removing launchers (which accepts the marker anywhere in the file): the
// marker must be the exact header comment of one of the generated formats (POSIX
// sh, Windows cmd, Windows PowerShell) and the invocation line must round-trip
// through the same quoting the generators apply. A file that merely mentions
// the marker, or carries another launcher version, is not reported.
func ManagedLauncherTarget(path string) (target string, ok bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, launcherProbeBytes))
	if err != nil {
		return "", false
	}
	return parseManagedLauncher(string(data))
}

// parseManagedLauncher extracts the wrapped target from generated launcher
// content. The header marker must sit on one of the first two lines (after the
// shebang or "@echo off" line, or first for PowerShell) and the invocation must
// follow it.
func parseManagedLauncher(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	header := -1
	for i := 0; i < len(lines) && i < 2; i++ {
		if lines[i] == "# "+OwnershipMarker || lines[i] == "rem "+OwnershipMarker {
			header = i
			break
		}
	}
	if header < 0 {
		return "", false
	}
	for _, line := range lines[header+1:] {
		if target, ok := parseLauncherInvocation(line); ok {
			return target, true
		}
	}
	return "", false
}

// parseLauncherInvocation recognizes the invocation line of each generated
// launcher and recovers the target it quotes.
func parseLauncherInvocation(line string) (string, bool) {
	const (
		cmdSuffix = `" %*`
		cmdPSHead = "powershell -NoProfile -ExecutionPolicy Bypass -File \""
	)
	if inner, ok := between(line, "exec ", ` "$@"`); ok {
		return unquoteSingle(inner, `'\''`, shellQuote)
	}
	if inner, ok := between(line, `"`, cmdSuffix); ok {
		return unquoteCMD(inner)
	}
	if inner, ok := between(line, cmdPSHead, cmdSuffix); ok {
		return unquoteCMD(inner)
	}
	if inner, ok := between(line, "& ", " @args"); ok {
		return unquoteSingle(inner, `''`, powershellQuote)
	}
	return "", false
}

// between returns what lies between a prefix and a suffix that must not
// overlap, so a line like `exec "$@"` cannot be sliced out of bounds.
func between(line, prefix, suffix string) (string, bool) {
	if len(line) < len(prefix)+len(suffix) || !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
		return "", false
	}
	return line[len(prefix) : len(line)-len(suffix)], true
}

// unquoteSingle reverses a single-quoted token and accepts the result only when
// the generator's own quoting function reproduces the input exactly.
func unquoteSingle(quoted, escapedQuote string, requote func(string) string) (string, bool) {
	if len(quoted) < 2 || quoted[0] != '\'' || quoted[len(quoted)-1] != '\'' {
		return "", false
	}
	target := strings.ReplaceAll(quoted[1:len(quoted)-1], escapedQuote, `'`)
	if target == "" || requote(target) != quoted {
		return "", false
	}
	return target, true
}

// unquoteCMD reverses the double-quote escaping of windowsCMDLauncher and
// accepts the result only when re-escaping it reproduces the input.
func unquoteCMD(escaped string) (string, bool) {
	target := strings.ReplaceAll(escaped, `""`, `"`)
	if target == "" || strings.ReplaceAll(target, `"`, `""`) != escaped {
		return "", false
	}
	return target, true
}
