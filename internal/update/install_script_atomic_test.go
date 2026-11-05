package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readInstallSh(t *testing.T) string {
	t.Helper()
	return readRepositoryFile(t, "scripts", "install.sh")
}

// extractBashFunction returns the text of a top-level `name() {` function up to
// its closing `}` at column zero.
func extractBashFunction(t *testing.T, script, name string) string {
	t.Helper()
	start := strings.Index(script, "\n"+name+"() {")
	if start == -1 {
		t.Fatalf("scripts/install.sh is missing the %s function", name)
	}
	start++
	end := strings.Index(script[start:], "\n}\n")
	if end == -1 {
		t.Fatalf("could not locate the end of %s", name)
	}
	return script[start : start+end+3]
}

func installBinaryBody(t *testing.T) string {
	t.Helper()
	return extractBashFunction(t, readInstallSh(t), "install_binary")
}

// The binary is staged next to the destination and renamed in place, so a
// failed or interrupted copy never leaves a truncated axiom.
func TestInstallScriptReplacesBinaryAtomically(t *testing.T) {
	body := installBinaryBody(t)

	if regexp.MustCompile(`\bcp\s`).MatchString(body) {
		t.Error("install_binary must not use cp to write the destination; stage and rename")
	}
	if !strings.Contains(body, `"${install_dir}/${BINARY_NAME}.staging.$$"`) {
		t.Error("install_binary must stage in the destination directory (same filesystem) so the rename is atomic")
	}
	if !strings.Contains(body, `mv -f -- "$staging" "$final"`) {
		t.Error("install_binary must publish with mv -f -- \"$staging\" \"$final\"")
	}
	if strings.Contains(body, "chmod +x") {
		t.Error("install_binary must set the mode with install -m 0755, not chmod +x (umask dependent)")
	}
	if !strings.Contains(body, "install -m 0755 --") {
		t.Error("install_binary must use install -m 0755")
	}
}

// Under sudo the paths travel as arguments; no shell string interpolates them.
func TestInstallScriptSudoDoesNotInterpolatePaths(t *testing.T) {
	body := installBinaryBody(t)

	if regexp.MustCompile(`sudo\s+(--\s+)?(sh|bash)\s+-c`).MatchString(body) {
		t.Error("install_binary must not run sudo sh/bash -c with an interpolated string")
	}
	for _, want := range []string{
		`sudo -- install -m 0755 -- "${tmpdir}/${BINARY_NAME}" "$staging"`,
		`sudo -- mv -f -- "$staging" "$final"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("install_binary is missing the sudo argument form %q", want)
		}
	}
}

// The staging file and the temp directory are removed on any exit, including
// TERM and INT, and the two cleanups share one EXIT trap.
func TestInstallScriptCleansStagingOnAnyExit(t *testing.T) {
	script := readInstallSh(t)
	body := extractBashFunction(t, script, "install_binary")

	for _, want := range []string{"trap cleanup_install EXIT", "trap 'exit 143' TERM", "trap 'exit 130' INT"} {
		if !strings.Contains(body, want) {
			t.Errorf("install_binary is missing %q", want)
		}
	}
	if len(regexp.MustCompile(`trap\s+\S+\s+EXIT`).FindAllString(body, -1)) != 1 {
		t.Error("install_binary must register a single EXIT trap so the staging cleanup does not replace the temp dir cleanup")
	}

	bash, err := workingBash()
	if err != nil {
		t.Skipf("working bash shell required: %v", err)
	}
	dir := t.TempDir()
	staging := filepath.Join(dir, "axiom.staging.1")
	tmp := filepath.Join(dir, "tmp")
	if err := os.WriteFile(staging, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(tmp, 0o700); err != nil {
		t.Fatal(err)
	}
	fn := extractBashFunction(t, script, "cleanup_install")
	cmd := exec.Command(bash, "-c", fn+"\nset -u\nINSTALL_STAGING=\"$1\"\nINSTALL_TMPDIR=\"$2\"\ncleanup_install\n", "_", filepath.ToSlash(staging), filepath.ToSlash(tmp))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cleanup_install: %v\n%s", err, out)
	}
	for _, p := range []string{staging, tmp} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("cleanup_install left %s behind", p)
		}
	}
}
