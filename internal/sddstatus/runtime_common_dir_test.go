package sddstatus

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// commonDirOracle resuelve el git common dir por la vía independiente que fija D3:
// EvalSymlinks sobre la salida absoluta de git.
func commonDirOracle(t *testing.T, dir string) string {
	t.Helper()
	command := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git rev-parse --git-common-dir en %s: %v", dir, err)
	}
	resolved, err := filepath.EvalSymlinks(filepath.FromSlash(strings.TrimSpace(string(output))))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(resolved)
}

func assertRuntimeCommonDir(t *testing.T, open, oracleDir string) {
	t.Helper()
	want := commonDirOracle(t, oracleDir)
	store, err := OpenRuntimeStore(context.Background(), open, "common-dir-change")
	if err != nil {
		t.Fatal(err)
	}
	if store.commonDir != want {
		t.Fatalf("commonDir = %q, want %q", store.commonDir, want)
	}
	if !strings.HasPrefix(store.Dir, filepath.Join(want, "gentle-ai", "sdd-runtime")+string(filepath.Separator)) {
		t.Fatalf("Dir = %q, want under %q", store.Dir, filepath.Join(want, "gentle-ai", "sdd-runtime"))
	}
	got, err := runtimeGitCommonDir(context.Background(), open)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("runtimeGitCommonDir = %q, want %q", got, want)
	}
}

func TestOpenRuntimeStoreCommonDirMainCheckout(t *testing.T) {
	repo := initRuntimeLedgerRepo(t)
	assertRuntimeCommonDir(t, repo, repo)
}

func TestOpenRuntimeStoreCommonDirLinkedWorktree(t *testing.T) {
	repo := initRuntimeLedgerRepo(t)
	linked := filepath.Join(t.TempDir(), "linked")
	runRuntimeLedgerGit(t, repo, "worktree", "add", "-q", "-b", "linked-branch", linked)
	assertRuntimeCommonDir(t, linked, repo)
	main, err := OpenRuntimeStore(context.Background(), repo, "common-dir-change")
	if err != nil {
		t.Fatal(err)
	}
	other, err := OpenRuntimeStore(context.Background(), linked, "common-dir-change")
	if err != nil {
		t.Fatal(err)
	}
	if main.commonDir != other.commonDir || main.Dir != other.Dir {
		t.Fatalf("linked worktree resolved %q/%q, main checkout %q/%q", other.commonDir, other.Dir, main.commonDir, main.Dir)
	}
}

func TestOpenRuntimeStoreCommonDirSymlinkedPath(t *testing.T) {
	repo := initRuntimeLedgerRepo(t)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(repo, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	assertRuntimeCommonDir(t, link, repo)
}

func TestOpenRuntimeStoreCommonDirShortTempName(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("8.3 short names exist only on Windows")
	}
	repo := initRuntimeLedgerRepo(t)
	short := windowsShortPath(t, repo)
	assertRuntimeCommonDir(t, short, repo)
}

func TestOpenRuntimeStoreCommonDirFromSubdirectory(t *testing.T) {
	repo := initRuntimeLedgerRepo(t)
	sub := filepath.Join(repo, "nested", "deeper")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	assertRuntimeCommonDir(t, sub, repo)
}

func TestOpenRuntimeStoreOutsideRepositoryKeepsRequiredError(t *testing.T) {
	_, err := OpenRuntimeStore(context.Background(), t.TempDir(), "common-dir-change")
	if err == nil {
		t.Fatal("expected an error outside a repository")
	}
	if _, ok := err.(*RuntimeRepositoryRequiredError); !ok {
		t.Fatalf("error = %T %v, want *RuntimeRepositoryRequiredError", err, err)
	}
}
