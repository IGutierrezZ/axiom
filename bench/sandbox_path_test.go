package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFakeAgent drops a stand-in for a real agent CLI into dir, named the way
// the host resolves it: `name` on POSIX (executable bit set), `name.exe` on
// Windows (the extension is what makes it resolvable there).
func writeFakeAgent(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	file := name
	if runtime.GOOS == "windows" {
		file += ".exe"
	}
	path := filepath.Join(dir, file)
	if err := os.WriteFile(path, []byte("not a real agent"), 0o755); err != nil {
		t.Fatalf("write fake agent: %v", err)
	}
	return path
}

func envPath(t *testing.T, env []string) string {
	t.Helper()
	for _, entry := range env {
		if value, ok := strings.CutPrefix(entry, "PATH="); ok {
			return value
		}
	}
	t.Fatal("sandbox env carries no PATH")
	return ""
}

// The defect this file exists for: the sandbox used to inherit the user's whole
// PATH, so a shim that the host could not execute (every POSIX shim on Windows)
// silently fell through to the REAL agent installed in a user directory — a real
// `claude` answered "Not logged in" inside a journey. The sandbox PATH must not
// carry any directory the user put on their own PATH.
func TestSandboxEnvPathDoesNotInheritUserDirectories(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is required by the product and by the sandbox PATH")
	}
	userDir := filepath.Join(t.TempDir(), "user-bin")
	writeFakeAgent(t, userDir, "claude")
	t.Setenv("PATH", userDir+string(os.PathListSeparator)+filepath.Dir(gitPath))

	// A literal, not newSandbox: newSandbox hardlinks the running test binary
	// into the root, which Windows refuses to delete when the test ends.
	root := t.TempDir()
	sandbox := &Sandbox{Root: root, Home: filepath.Join(root, "home"), Binary: "gentle-ai"}
	for _, entry := range filepath.SplitList(envPath(t, sandbox.env())) {
		if entry == userDir {
			t.Fatalf("sandbox PATH inherited the user directory %q, so the real agent in it is reachable", userDir)
		}
	}
}

func TestClosedPathEntries(t *testing.T) {
	gitDir := filepath.Join("opt", "git", "cmd")
	for _, tt := range []struct {
		name   string
		goos   string
		gitDir string
		want   []string
	}{
		{"windows carries only git", "windows", gitDir, []string{gitDir}},
		{"posix adds the shim system directories", "linux", gitDir, []string{gitDir, "/usr/bin", "/bin"}},
		{"darwin matches linux", "darwin", gitDir, []string{gitDir, "/usr/bin", "/bin"}},
		{"git in a system directory is not repeated", "linux", "/usr/bin", []string{"/usr/bin", "/bin"}},
		{"unresolved git adds no empty entry", "linux", "", []string{"/usr/bin", "/bin"}},
		{"unresolved git on windows is empty", "windows", "", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := closedPathEntries(tt.goos, tt.gitDir)
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Fatalf("closedPathEntries(%q, %q) = %v, want %v", tt.goos, tt.gitDir, got, tt.want)
			}
		})
	}
}
