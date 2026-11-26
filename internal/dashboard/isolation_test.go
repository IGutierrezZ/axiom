package dashboard

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/app"
)

// Helpers that let a test run a side-effecting dashboard flow (sync, upgrade,
// archive sync, git status) against a throwaway project instead of the
// repository checkout. See TestMain for the package-wide guarantees.

// writeFixtureFile writes content to rel (slash-separated) under root, creating
// the parent directories.
func writeFixtureFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("crear directorio de %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escribir %s: %v", rel, err)
	}
}

// initGitFixture turns root into a git work tree with one empty commit on
// branch main and no remote, so nothing in it can reach the network. The test is
// skipped when git is not installed.
func initGitFixture(t *testing.T, root string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no está disponible en el PATH")
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.name=axiom-test", "-c", "user.email=axiom-test@example.invalid",
			"-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s en %s: %v\n%s", strings.Join(args, " "), root, err, out)
		}
	}
}

// syncRecorder captures what the dashboard asked `axiom` to run, from which
// working directory, and which release channels the upgrade step was given.
type syncRecorder struct {
	args     [][]string
	dirs     []string
	channels []string
}

// stubAppRun replaces the in-process `axiom` runner (the sync step) and the
// binary upgrade with no-ops that succeed, and records each invocation. It leaves
// the upgrade->sync chain itself untouched, so its sync phase is recorded too.
// Without it, the ecosystem endpoints run a real `axiom sync` and a real binary
// upgrade (network access and replacement of the installed tools).
func stubAppRun(t *testing.T) *syncRecorder {
	t.Helper()
	rec := &syncRecorder{}
	origRun := runAppArgsFn
	origReport := upgradeSequenceReportFn
	t.Cleanup(func() {
		runAppArgsFn = origRun
		upgradeSequenceReportFn = origReport
	})

	runAppArgsFn = func(args []string, _ io.Writer) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		rec.args = append(rec.args, append([]string(nil), args...))
		rec.dirs = append(rec.dirs, wd)
		return nil
	}
	upgradeSequenceReportFn = func(_ context.Context, _ io.Writer, channel ...string) (app.UpgradeRunReport, error) {
		rec.channels = append(rec.channels, strings.Join(channel, ","))
		return app.UpgradeRunReport{Status: app.UpgradeStatusSucceeded, SelfToolName: "axiom"}, nil
	}
	return rec
}

// sameDirectory reports whether a and b are the same directory on disk, which
// is robust to case and 8.3 short-name differences in Windows temp paths.
func sameDirectory(t *testing.T, a, b string) bool {
	t.Helper()
	infoA, err := os.Stat(a)
	if err != nil {
		t.Fatalf("Stat(%s): %v", a, err)
	}
	infoB, err := os.Stat(b)
	if err != nil {
		t.Fatalf("Stat(%s): %v", b, err)
	}
	return os.SameFile(infoA, infoB)
}
