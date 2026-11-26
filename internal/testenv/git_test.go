package testenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	_, cleanup, err := IsolateGit("testenv")
	if err != nil {
		fmt.Fprintf(os.Stderr, "testenv: %v\n", err)
		return 1
	}
	defer cleanup()
	return m.Run()
}

func TestIsolateGitFixesTheEnvironment(t *testing.T) {
	t.Parallel()
	RequireGit(t)
	home := os.Getenv("HOME")
	if home == "" || !strings.HasPrefix(home, scratchRoot) || os.Getenv("USERPROFILE") != home {
		t.Fatalf("HOME=%q USERPROFILE=%q, se esperaba un HOME bajo %s", home, os.Getenv("USERPROFILE"), scratchRoot)
	}
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE"} {
		if value, set := os.LookupEnv(key); set {
			t.Errorf("%s=%q sigue heredada", key, value)
		}
	}
	dir := t.TempDir()
	Git(t, dir, "init", "-q")
	if got := Git(t, dir, "config", "--get", "gc.auto"); got != "0" {
		t.Errorf("gc.auto = %q, se esperaba 0", got)
	}
	if got := Git(t, dir, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Errorf("rama inicial = %q, se esperaba main", got)
	}
	if entries, err := os.ReadDir(filepath.Join(dir, ".git", "hooks")); err == nil && len(entries) > 0 {
		t.Errorf("git init copió %d hooks de ejemplo; la plantilla debe estar vacía", len(entries))
	}
	Git(t, dir, "commit", "-q", "--allow-empty", "-m", "inicial")
	if got := Git(t, dir, "log", "-1", "--format=%an <%ae>"); got != GitAuthorName+" <"+GitAuthorEmail+">" {
		t.Errorf("autor = %q", got)
	}
}

func TestRepoTemplateBuildsOnceAndCopiesAreIndependent(t *testing.T) {
	t.Parallel()
	builds := 0
	template := NewRepoTemplate("independiente", func(t testing.TB, dir string) {
		builds++
		repo, bare := filepath.Join(dir, "repo"), filepath.Join(dir, "bare")
		for _, d := range []string{repo, bare} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		Git(t, bare, "init", "-q", "--bare")
		Git(t, repo, "init", "-q")
		Git(t, repo, "commit", "-q", "--allow-empty", "-m", "inicial")
		Git(t, repo, "remote", "add", "origin", filepath.ToSlash(bare))
		Git(t, repo, "push", "-q", "origin", "main")
	})

	first, second := template.Copy(t), template.Copy(t)
	if builds != 1 {
		t.Fatalf("la plantilla se construyó %d veces, se esperaba una", builds)
	}
	url := Git(t, filepath.Join(first, "repo"), "remote", "get-url", "origin")
	if filepath.Clean(filepath.FromSlash(url)) != filepath.Join(first, "bare") {
		t.Fatalf("origin de la copia = %q, se esperaba %s", url, filepath.Join(first, "bare"))
	}

	Git(t, filepath.Join(first, "repo"), "commit", "-q", "--allow-empty", "-m", "solo en la primera")
	Git(t, filepath.Join(first, "repo"), "push", "-q", "origin", "main")

	head := Git(t, filepath.Join(first, "repo"), "rev-parse", "HEAD")
	if got := Git(t, filepath.Join(first, "bare"), "rev-parse", "main"); got != head {
		t.Errorf("el remoto de la primera copia tiene %s, se esperaba %s", got, head)
	}
	for _, bare := range []string{filepath.Join(second, "bare"), filepath.Join(template.dir, "bare")} {
		if got := Git(t, bare, "rev-parse", "main"); got == head {
			t.Errorf("el push de la primera copia llegó a %s", bare)
		}
	}
	if got := Git(t, filepath.Join(second, "repo"), "status", "--porcelain"); got != "" {
		t.Errorf("la segunda copia no está limpia:\n%s", got)
	}
}

func TestCopyTemplateRewritesOnlyGitPathFiles(t *testing.T) {
	t.Parallel()
	src, dst := t.TempDir(), t.TempDir()
	slash := filepath.ToSlash(src)
	files := map[string]string{
		filepath.Join("repo", ".git", "config"):                   "url = " + slash + "/bare\n",
		filepath.Join("repo", ".git", "worktrees", "w", "gitdir"): src + string(filepath.Separator) + "w\n",
		filepath.Join("w", ".git"):                                "gitdir: " + slash + "/repo/.git/worktrees/w\n",
		filepath.Join("repo", "notas.txt"):                        slash + "\n",
	}
	for name, content := range files {
		path := filepath.Join(src, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := copyTemplate(src, dst); err != nil {
		t.Fatal(err)
	}

	for name, content := range files {
		data, err := os.ReadFile(filepath.Join(dst, name))
		if err != nil {
			t.Fatal(err)
		}
		want := content
		if holdsGitPaths(filepath.Base(name)) {
			want = strings.ReplaceAll(strings.ReplaceAll(content, slash, filepath.ToSlash(dst)), src, dst)
		}
		if string(data) != want {
			t.Errorf("%s = %q, se esperaba %q", name, data, want)
		}
	}
}
