package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/update"
)

// Windows does not let a running executable be overwritten or deleted, but it
// does let it be renamed. `go install` relies on that by moving the old binary
// to the fixed name `<name>~` and deleting it afterwards. When a previous
// process is still running from `<name>~` (for example a dashboard that kept
// serving while an external `go install` replaced its binary), that fixed slot
// is taken and the next install fails with a sharing violation.
//
// The self-tool install on Windows therefore never lets `go install` touch the
// destination: it builds into a staging directory next to the destination
// (same volume, so the final move is a rename), moves the current binary aside
// to a unique `.old` name and moves the new one into place (RF-UPD-11).

// selfInstallStagingPrefix names the staging directories created inside the
// destination directory. They are removed after every attempt.
const selfInstallStagingPrefix = ".axiom-install-"

// selfInstallMkdirTemp creates the staging directory. Package-level so tests can
// make it fail.
var selfInstallMkdirTemp = os.MkdirTemp

// selfInstallNow stamps the aside name. Package-level so tests are deterministic.
var selfInstallNow = time.Now

// goInstallSelfWindows runs `go install target` with GOBIN pointing at a fresh
// staging directory inside destDir and then swaps the staged binary into
// destDir. env is the environment the command must run with (GOBIN is added).
func goInstallSelfWindows(tool update.ToolInfo, target string, env []string, destDir string) error {
	stageDir, err := selfInstallMkdirTemp(destDir, selfInstallStagingPrefix)
	if err != nil {
		return fmt.Errorf("go install %s: create staging directory in %s: %w", target, destDir, err)
	}
	defer os.RemoveAll(stageDir)

	cmd := execCommand("go", "install", target)
	cmd.Stdin = nil
	if env == nil {
		env = os.Environ()
	}
	cmd.Env = setEnvValue(append([]string(nil), env...), "GOBIN", stageDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go install %s: %w (output: %s)", target, err, string(out))
	}

	name := goInstallBinaryName(tool.Name, "windows")
	staged := filepath.Join(stageDir, name)
	if _, err := os.Stat(staged); err != nil {
		return fmt.Errorf("go install %s: the built binary was not found at %s: %w", target, staged, err)
	}
	return swapSelfBinary(staged, filepath.Join(destDir, name))
}

// swapSelfBinary replaces dst with staged. The current dst, if any, is moved
// aside to a unique `.old` name first (a rename works even while that binary
// is running); if the new binary cannot be put in place, the old one is moved
// back. Leftover aside copies from earlier upgrades are removed when free.
func swapSelfBinary(staged, dst string) error {
	aside := ""
	if _, err := os.Lstat(dst); err == nil {
		aside = uniqueAsideName(dst)
		if err := renameFn(dst, aside); err != nil {
			return fmt.Errorf("move %s aside to %s: %w", dst, aside, err)
		}
	}

	if err := atomicReplace(staged, dst); err != nil {
		if aside == "" {
			return err
		}
		if restoreErr := renameFn(aside, dst); restoreErr != nil {
			return fmt.Errorf("%w; restoring the previous binary from %s also failed: %v", err, aside, restoreErr)
		}
		return fmt.Errorf("%w; the previous binary was restored at %s", err, dst)
	}

	if aside != "" {
		// Fails while the old binary is still running; the next upgrade retries.
		_ = os.Remove(aside)
	}
	removeStaleAsideCopies(dst)
	return nil
}

// uniqueAsideName returns a free `<dst>.<stamp>.old` name. The `.old` suffix is
// what the dashboard relaunch already skips when it looks for its executable.
func uniqueAsideName(dst string) string {
	base := dst + "." + selfInstallNow().UTC().Format("20060102T150405.000000000")
	candidate := base + ".old"
	for i := 1; ; i++ {
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return candidate
		}
		candidate = base + "-" + strconv.Itoa(i) + ".old"
	}
}

// removeStaleAsideCopies deletes the `.old` copies left by earlier swaps and the
// `~` copy `go install` leaves behind. Files still in use stay; nothing fails.
func removeStaleAsideCopies(dst string) {
	matches, _ := filepath.Glob(dst + ".*.old")
	for _, path := range append(matches, dst+"~") {
		_ = os.Remove(path)
	}
}
