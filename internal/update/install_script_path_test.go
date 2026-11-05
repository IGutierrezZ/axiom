package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// pathFunctions returns the PATH helpers of install.sh ready to source in a
// scratch bash, with stubs for the logging helpers.
func pathFunctions(t *testing.T) string {
	t.Helper()
	script := readInstallSh(t)
	var b strings.Builder
	b.WriteString("PATH_MARKER=\"# axiom: PATH\"\nBINARY_NAME=axiom\nDIM=''\nNC=''\n")
	b.WriteString("warn() { echo \"WARN $*\"; }\nsuccess() { echo \"OK $*\"; }\n")
	for _, name := range []string{"path_rc_file", "path_rc_line", "add_dir_to_shell_rc", "ensure_dir_in_path"} {
		b.WriteString(extractBashFunction(t, script, name))
		b.WriteString("\n")
	}
	return b.String()
}

// runPathBash runs body with a throwaway HOME so the developer's real rc files
// are never read or written.
func runPathBash(t *testing.T, home, body string, extraEnv ...string) string {
	t.Helper()
	bash, err := workingBash()
	if err != nil {
		t.Skipf("working bash shell required: %v", err)
	}
	// The script goes through stdin: Windows argument quoting mangles the
	// backslashes and quotes the fixtures need.
	cmd := exec.Command(bash, "-s")
	cmd.Stdin = strings.NewReader(pathFunctions(t) + "\n" + body + "\n")
	cmd.Env = append(os.Environ(), append([]string{"HOME=" + filepath.ToSlash(home)}, extraEnv...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash fixture: %v\n%s", err, out)
	}
	return string(out)
}

func readRC(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestInstallScriptAddsDirToShellRCIdempotently(t *testing.T) {
	home := t.TempDir()
	runPathBash(t, home, `
add_dir_to_shell_rc /opt/axiom/bin bash Linux >/dev/null
add_dir_to_shell_rc /opt/axiom/bin bash Linux >/dev/null
add_dir_to_shell_rc /other/bin bash Linux >/dev/null
`)
	rc := readRC(t, filepath.Join(home, ".bashrc"))
	if got := strings.Count(rc, "# axiom: PATH"); got != 2 {
		t.Fatalf("expected one marked line per directory (2), got %d:\n%s", got, rc)
	}
	if got := strings.Count(rc, `export PATH="$PATH":/opt/axiom/bin # axiom: PATH`); got != 1 {
		t.Fatalf("/opt/axiom/bin must be added exactly once, got %d:\n%s", got, rc)
	}
}

func TestInstallScriptPicksRCFilePerShell(t *testing.T) {
	cases := []struct {
		shell, os, rel, want string
	}{
		{"bash", "Linux", ".bashrc", `export PATH="$PATH":/opt/axiom/bin # axiom: PATH`},
		{"bash", "Darwin", ".bash_profile", `export PATH="$PATH":/opt/axiom/bin # axiom: PATH`},
		{"zsh", "Darwin", ".zshrc", `export PATH="$PATH":/opt/axiom/bin # axiom: PATH`},
		{"fish", "Linux", filepath.Join(".config", "fish", "config.fish"), `fish_add_path '/opt/axiom/bin' # axiom: PATH`},
	}
	for _, tc := range cases {
		t.Run(tc.shell+"-"+tc.os, func(t *testing.T) {
			home := t.TempDir()
			out := runPathBash(t, home, "add_dir_to_shell_rc /opt/axiom/bin "+tc.shell+" "+tc.os)
			if !strings.Contains(out, filepath.ToSlash(tc.rel)) {
				t.Errorf("function should print the rc path containing %q, got %q", tc.rel, out)
			}
			if rc := readRC(t, filepath.Join(home, tc.rel)); !strings.Contains(rc, tc.want) {
				t.Errorf("rc is missing %q:\n%s", tc.want, rc)
			}
		})
	}
}

func TestInstallScriptLeavesUnknownShellAlone(t *testing.T) {
	home := t.TempDir()
	out := runPathBash(t, home, `
SHELL=/bin/tcsh
PATH=/usr/bin:/bin
ensure_dir_in_path /opt/axiom/bin
`)
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("an unrecognized shell must not create any file, found %v", entries)
	}
	if !strings.Contains(out, "/opt/axiom/bin is not in your PATH") {
		t.Errorf("expected the manual PATH warning, got:\n%s", out)
	}
}

func TestInstallScriptSkipsRCWhenDirAlreadyInPathOrOptedOut(t *testing.T) {
	home := t.TempDir()
	out := runPathBash(t, home, `
SHELL=/bin/bash
PATH="/usr/bin:/opt/axiom/bin:/bin"
ensure_dir_in_path /opt/axiom/bin
`)
	if strings.TrimSpace(out) != "" {
		t.Errorf("a directory already in PATH must produce no output, got %q", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".bashrc")); !os.IsNotExist(err) {
		t.Error("a directory already in PATH must not touch the rc file")
	}

	out = runPathBash(t, home, `
SHELL=/bin/bash
PATH=/usr/bin:/bin
ensure_dir_in_path /opt/axiom/bin
`, "AXIOM_NO_PERSISTENT_PATH=1")
	if !strings.Contains(out, "is not in your PATH") {
		t.Errorf("opt-out must fall back to the manual hint, got %q", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".bashrc")); !os.IsNotExist(err) {
		t.Error("AXIOM_NO_PERSISTENT_PATH=1 must not write the rc file")
	}
}

func TestInstallScriptFishLineEscapesQuotes(t *testing.T) {
	home := t.TempDir()
	out := runPathBash(t, home, `path_rc_line fish "/opt/it's\here"`)
	want := `fish_add_path '/opt/it\'s\\here' # axiom: PATH`
	if strings.TrimSpace(out) != want {
		t.Errorf("fish line = %q, want %q", strings.TrimSpace(out), want)
	}
}

func TestInstallScriptWritesRCLineWithoutShellInjection(t *testing.T) {
	home := t.TempDir()
	pwned := filepath.ToSlash(filepath.Join(home, "pwned"))
	dir := "/opt/ax iom/$(touch " + pwned + ")/`touch " + pwned + "`/it's"
	runPathBash(t, home, `add_dir_to_shell_rc "$AXIOM_TEST_DIR" bash Linux >/dev/null`, "AXIOM_TEST_DIR="+dir)

	// Sourcing the rc must append the literal directory and execute nothing.
	out := runPathBash(t, home, `PATH=/usr/bin:/bin; . "$HOME/.bashrc"; printf '%s' "$PATH"`)
	if !strings.HasSuffix(out, ":"+dir) {
		t.Errorf("sourcing the rc must append the literal directory, got %q", out)
	}
	if _, err := os.Stat(filepath.Join(home, "pwned")); !os.IsNotExist(err) {
		t.Error("the directory was executed as shell code when the rc was sourced")
	}
}
