package skillregistry

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// requireGit skips the test when git is not on PATH and isolates git from the
// developer's own configuration: a global excludes file that ignores `.claude`
// would otherwise change what these fixtures mean.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	emptyConfig := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(emptyConfig, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", emptyConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// newGitRepo returns a fresh git repository (no commits are needed: the
// classification only reads ignore rules and the index).
func newGitRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q")
	return dir
}

// stubGitCheckIgnore replaces the git seam for one test.
func stubGitCheckIgnore(t *testing.T, fn func(cwd string, stdin []byte) ([]byte, int, error)) {
	t.Helper()
	previous := runGitCheckIgnore
	runGitCheckIgnore = fn
	t.Cleanup(func() { runGitCheckIgnore = previous })
}

func describedSkill(name, description string) string {
	return "---\nname: " + name + "\ndescription: " + description + "\n---\n\nBody.\n"
}

// hasRow reports whether a rendered table has a row for the skill name.
func hasRow(content, name string) bool {
	return strings.Contains(content, "| `"+name+"` |")
}

// captureMirror returns a MirrorFunc that records the last request it saw.
func captureMirror() (MirrorFunc, *MirrorRequest) {
	var last MirrorRequest
	return func(req MirrorRequest) error {
		last = req
		return nil
	}, &last
}

// TestRegenerateVersionedViewDropsIgnoredCopies covers the core contract: a
// gitignored machine-local copy never reaches AGENTS.md or the Engram mirror,
// the versioned copy wins the name, and .atl/skill-registry.md keeps the full
// view.
func TestRegenerateVersionedViewDropsIgnoredCopies(t *testing.T) {
	cwd := newGitRepo(t)
	home := t.TempDir()
	writeSkill(t, filepath.Join(cwd, ".gitignore"), ".claude/\n")
	writeSkill(t, filepath.Join(cwd, "skills", "x", "SKILL.md"), describedSkill("x", "canonical x"))
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "x", "SKILL.md"), describedSkill("x", "local copy of x"))
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "y", "SKILL.md"), minimalSkill("y"))
	// Non-ASCII and spaces travel through the NUL-separated protocol unquoted.
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "ñandú z", "SKILL.md"), describedSkill("nandu", "local only"))
	writeSkill(t, filepath.Join(cwd, AgentsRelPath), agentsFixture)

	mirror, mirrored := captureMirror()
	result, err := Regenerate(cwd, home, RegenerateOptions{Mirror: mirror})
	if err != nil {
		t.Fatalf("Regenerate() error = %v", err)
	}
	if result.SkillCount != 3 {
		t.Fatalf("SkillCount = %d, want 3 (it counts the full view that .atl lists)", result.SkillCount)
	}

	agents := readFile(t, filepath.Join(cwd, AgentsRelPath))
	registry := readFile(t, filepath.Join(cwd, RegistryRelPath))
	for name, content := range map[string]string{"AGENTS.md": agents, "mirror": mirrored.Content} {
		if !hasRow(content, "x") || !strings.Contains(content, "`skills/x/SKILL.md`") {
			t.Fatalf("%s must list x at its versioned relative path:\n%s", name, content)
		}
		if !strings.Contains(content, "canonical x") || strings.Contains(content, "local copy of x") {
			t.Fatalf("%s must carry the versioned copy of x:\n%s", name, content)
		}
		for _, local := range []string{"y", "nandu"} {
			if hasRow(content, local) {
				t.Fatalf("%s must not list the gitignored skill %q:\n%s", name, local, content)
			}
		}
		if strings.Contains(content, ".claude") {
			t.Fatalf("%s must not mention ignored paths:\n%s", name, content)
		}
	}
	for _, name := range []string{"x", "y", "nandu"} {
		if !hasRow(registry, name) {
			t.Fatalf(".atl/skill-registry.md must keep the full view, missing %q:\n%s", name, registry)
		}
	}
}

// TestRegenerateVersionedViewDedupesAfterFiltering proves the filter runs BEFORE
// the name dedupe: the ignored copy scans first (`.claude` precedes `.github`),
// wins the full view, and must not hide the versioned copy in the shared view.
func TestRegenerateVersionedViewDedupesAfterFiltering(t *testing.T) {
	cwd := newGitRepo(t)
	home := t.TempDir()
	writeSkill(t, filepath.Join(cwd, ".gitignore"), ".claude/\n")
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "dup", "SKILL.md"), describedSkill("dup", "ignored copy"))
	writeSkill(t, filepath.Join(cwd, ".github", "skills", "dup", "SKILL.md"), describedSkill("dup", "versioned copy"))
	writeSkill(t, filepath.Join(cwd, AgentsRelPath), agentsFixture)

	if _, err := Regenerate(cwd, home, RegenerateOptions{}); err != nil {
		t.Fatalf("Regenerate() error = %v", err)
	}

	agents := readFile(t, filepath.Join(cwd, AgentsRelPath))
	if !strings.Contains(agents, "`.github/skills/dup/SKILL.md`") || strings.Contains(agents, ".claude") {
		t.Fatalf("AGENTS.md must list the versioned copy of dup:\n%s", agents)
	}
	registry := readFile(t, filepath.Join(cwd, RegistryRelPath))
	if !strings.Contains(registry, filepath.Join(cwd, ".claude", "skills", "dup", "SKILL.md")) {
		t.Fatalf(".atl keeps today's full-view dedupe (first scanned copy wins):\n%s", registry)
	}
}

// TestRegenerateVersionedViewKeepsVersionedAgentDirs covers a project that does
// version `.claude/skills`: nothing ignores it, so its rows stay.
func TestRegenerateVersionedViewKeepsVersionedAgentDirs(t *testing.T) {
	cwd := newGitRepo(t)
	home := t.TempDir()
	writeSkill(t, filepath.Join(cwd, ".gitignore"), ".atl/\n")
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "z", "SKILL.md"), minimalSkill("z"))
	writeSkill(t, filepath.Join(cwd, AgentsRelPath), agentsFixture)

	if _, err := Regenerate(cwd, home, RegenerateOptions{}); err != nil {
		t.Fatalf("Regenerate() error = %v", err)
	}

	agents := readFile(t, filepath.Join(cwd, AgentsRelPath))
	if !hasRow(agents, "z") || !strings.Contains(agents, "`.claude/skills/z/SKILL.md`") {
		t.Fatalf("a versioned .claude/skills entry must keep its relative row:\n%s", agents)
	}
}

// TestRegenerateVersionedViewCountsTrackedFilesUnderIgnoredDirs proves the check
// runs without --no-index: a file git tracks is versioned even when an ignore
// rule would otherwise cover it, and its untracked siblings are not.
func TestRegenerateVersionedViewCountsTrackedFilesUnderIgnoredDirs(t *testing.T) {
	cwd := newGitRepo(t)
	home := t.TempDir()
	writeSkill(t, filepath.Join(cwd, ".gitignore"), ".claude/\n")
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "tracked", "SKILL.md"), minimalSkill("tracked"))
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "loose", "SKILL.md"), minimalSkill("loose"))
	gitRun(t, cwd, "add", "-f", ".claude/skills/tracked/SKILL.md")
	writeSkill(t, filepath.Join(cwd, AgentsRelPath), agentsFixture)

	if _, err := Regenerate(cwd, home, RegenerateOptions{}); err != nil {
		t.Fatalf("Regenerate() error = %v", err)
	}

	agents := readFile(t, filepath.Join(cwd, AgentsRelPath))
	if !hasRow(agents, "tracked") {
		t.Fatalf("a tracked file under an ignored dir counts as versioned:\n%s", agents)
	}
	if hasRow(agents, "loose") {
		t.Fatalf("an untracked file under an ignored dir stays out:\n%s", agents)
	}
}

// TestRegenerateVersionedViewExcludesUserScope: skills outside cwd live only in
// the full view, regardless of git.
func TestRegenerateVersionedViewExcludesUserScope(t *testing.T) {
	cwd := newGitRepo(t)
	home := t.TempDir()
	writeSkill(t, filepath.Join(cwd, "skills", "x", "SKILL.md"), minimalSkill("x"))
	writeSkill(t, filepath.Join(home, ".claude", "skills", "global-only", "SKILL.md"), minimalSkill("global-only"))
	writeSkill(t, filepath.Join(cwd, AgentsRelPath), agentsFixture)

	mirror, mirrored := captureMirror()
	if _, err := Regenerate(cwd, home, RegenerateOptions{Mirror: mirror}); err != nil {
		t.Fatalf("Regenerate() error = %v", err)
	}

	registry := readFile(t, filepath.Join(cwd, RegistryRelPath))
	if !hasRow(registry, "global-only") {
		t.Fatalf(".atl must list user-scope skills:\n%s", registry)
	}
	agents := readFile(t, filepath.Join(cwd, AgentsRelPath))
	for name, content := range map[string]string{"AGENTS.md": agents, "mirror": mirrored.Content} {
		if hasRow(content, "global-only") || strings.Contains(content, home) {
			t.Fatalf("%s must not list user-scope skills or absolute paths:\n%s", name, content)
		}
		if !hasRow(content, "x") {
			t.Fatalf("%s lost the project skill:\n%s", name, content)
		}
	}
}

// TestRegenerateWithoutGitFallsBackToProjectEntries covers the fallback: with no
// git answer every entry inside cwd counts as versionable (the behavior before
// the versioned view), while $HOME entries still never reach the shared files.
func TestRegenerateWithoutGitFallsBackToProjectEntries(t *testing.T) {
	fixture := func(t *testing.T) (cwd, home string) {
		t.Helper()
		cwd = t.TempDir()
		home = t.TempDir()
		writeSkill(t, filepath.Join(cwd, "skills", "x", "SKILL.md"), minimalSkill("x"))
		writeSkill(t, filepath.Join(cwd, ".claude", "skills", "c", "SKILL.md"), minimalSkill("c"))
		writeSkill(t, filepath.Join(home, ".claude", "skills", "global-only", "SKILL.md"), minimalSkill("global-only"))
		writeSkill(t, filepath.Join(cwd, AgentsRelPath), agentsFixture)
		return cwd, home
	}
	assertFallback := func(t *testing.T, cwd, home string) {
		t.Helper()
		mirror, mirrored := captureMirror()
		result, err := Regenerate(cwd, home, RegenerateOptions{Mirror: mirror})
		if err != nil {
			t.Fatalf("git failure must never be fatal: %v", err)
		}
		if !result.Regenerated || result.Mirror.Status != MirrorOK {
			t.Fatalf("Result = %+v, want a normal regeneration", result)
		}
		agents := readFile(t, filepath.Join(cwd, AgentsRelPath))
		for name, content := range map[string]string{"AGENTS.md": agents, "mirror": mirrored.Content} {
			if !hasRow(content, "x") || !hasRow(content, "c") || hasRow(content, "global-only") {
				t.Fatalf("%s fallback must keep every project entry and no user entry:\n%s", name, content)
			}
		}
	}

	t.Run("directory that is not a git repository", func(t *testing.T) {
		cwd, home := fixture(t)
		assertFallback(t, cwd, home)
	})

	failures := map[string]func(string, []byte) ([]byte, int, error){
		"git cannot run":        func(string, []byte) ([]byte, int, error) { return nil, -1, errors.New("git not found") },
		"exit 128":              func(string, []byte) ([]byte, int, error) { return nil, 128, nil },
		"unexpected exit code":  func(string, []byte) ([]byte, int, error) { return []byte("junk"), 2, nil },
		"killed by the timeout": func(string, []byte) ([]byte, int, error) { return nil, 1, errors.New("context deadline exceeded") },
	}
	for name, fail := range failures {
		t.Run(name, func(t *testing.T) {
			cwd, home := fixture(t)
			stubGitCheckIgnore(t, fail)
			assertFallback(t, cwd, home)
		})
	}
}

// TestRegenerateAsksGitOnceWithRelativeSlashPaths pins the git protocol: one
// invocation per regeneration, NUL-separated cwd-relative slash paths, and only
// entries inside cwd (never $HOME paths).
func TestRegenerateAsksGitOnceWithRelativeSlashPaths(t *testing.T) {
	cwd := t.TempDir()
	home := t.TempDir()
	writeSkill(t, filepath.Join(cwd, "skills", "x", "SKILL.md"), minimalSkill("x"))
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "y", "SKILL.md"), minimalSkill("y"))
	writeSkill(t, filepath.Join(home, ".claude", "skills", "g", "SKILL.md"), minimalSkill("g"))

	var calls int
	var gotCwd string
	var gotStdin []byte
	stubGitCheckIgnore(t, func(dir string, stdin []byte) ([]byte, int, error) {
		calls++
		gotCwd, gotStdin = dir, stdin
		return nil, 1, nil
	})
	if _, err := Regenerate(cwd, home, RegenerateOptions{}); err != nil {
		t.Fatalf("Regenerate() error = %v", err)
	}

	if calls != 1 {
		t.Fatalf("git was invoked %d times, want exactly 1 per regeneration", calls)
	}
	if gotCwd != cwd {
		t.Fatalf("git ran in %q, want the workspace %q", gotCwd, cwd)
	}
	if !strings.HasSuffix(string(gotStdin), "\x00") {
		t.Fatalf("stdin must be NUL-terminated: %q", gotStdin)
	}
	paths := strings.Split(strings.TrimSuffix(string(gotStdin), "\x00"), "\x00")
	sort.Strings(paths)
	want := []string{".claude/skills/y/SKILL.md", "skills/x/SKILL.md"}
	if strings.Join(paths, "|") != strings.Join(want, "|") {
		t.Fatalf("stdin paths = %q, want %q", paths, want)
	}
}

// TestRegenerateVersionedViewFollowsSymlinkedSkillDirs: git refuses to check a
// path below a symlink and would fail the whole invocation, silently turning
// the filter off. A symlinked skill dir must be classified by the link itself.
func TestRegenerateVersionedViewFollowsSymlinkedSkillDirs(t *testing.T) {
	cwd := newGitRepo(t)
	home := t.TempDir()
	writeSkill(t, filepath.Join(cwd, ".gitignore"), ".claude/\n")
	writeSkill(t, filepath.Join(cwd, "shared", "linked-skill", "SKILL.md"), minimalSkill("linked-skill"))
	if err := os.MkdirAll(filepath.Join(cwd, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(cwd, "shared", "linked-skill"), filepath.Join(cwd, "skills", "linked-skill")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "local", "SKILL.md"), minimalSkill("local"))
	writeSkill(t, filepath.Join(cwd, AgentsRelPath), agentsFixture)

	if _, err := Regenerate(cwd, home, RegenerateOptions{}); err != nil {
		t.Fatalf("Regenerate() error = %v", err)
	}

	agents := readFile(t, filepath.Join(cwd, AgentsRelPath))
	if !hasRow(agents, "linked-skill") {
		t.Fatalf("a symlinked skill dir that git does not ignore stays listed:\n%s", agents)
	}
	if hasRow(agents, "local") {
		t.Fatalf("the symlink must not disable git filtering for the other entries:\n%s", agents)
	}
}

// TestRegenerateGitignoreEditInvalidatesCache: editing .gitignore changes what
// is versionable without touching any SKILL.md, so the cache must miss and the
// row must go; a further run with nothing changed is a byte-identical cache-hit.
func TestRegenerateGitignoreEditInvalidatesCache(t *testing.T) {
	cwd := newGitRepo(t)
	home := t.TempDir()
	writeSkill(t, filepath.Join(cwd, ".gitignore"), ".atl/\n")
	writeSkill(t, filepath.Join(cwd, "skills", "x", "SKILL.md"), minimalSkill("x"))
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "z", "SKILL.md"), minimalSkill("z"))
	writeSkill(t, filepath.Join(cwd, AgentsRelPath), agentsFixture)
	agentsPath := filepath.Join(cwd, AgentsRelPath)

	first, err := Regenerate(cwd, home, RegenerateOptions{})
	if err != nil {
		t.Fatalf("first Regenerate() error = %v", err)
	}
	if !first.Regenerated || !hasRow(readFile(t, agentsPath), "z") {
		t.Fatalf("first run must list the versioned z: %+v\n%s", first, readFile(t, agentsPath))
	}

	writeSkill(t, filepath.Join(cwd, ".gitignore"), ".atl/\n.claude/\n")
	second, err := Regenerate(cwd, home, RegenerateOptions{})
	if err != nil {
		t.Fatalf("second Regenerate() error = %v", err)
	}
	if !second.Regenerated || second.Reason != "fingerprint-changed" {
		t.Fatalf("a .gitignore edit must miss the cache, got %+v", second)
	}
	if agents := readFile(t, agentsPath); hasRow(agents, "z") || !hasRow(agents, "x") {
		t.Fatalf("z is ignored now and must leave AGENTS.md while x stays:\n%s", agents)
	}
	if !hasRow(readFile(t, filepath.Join(cwd, RegistryRelPath)), "z") {
		t.Fatal(".atl keeps the full view after the .gitignore edit")
	}

	beforeAgents := readFile(t, agentsPath)
	beforeRegistry := readFile(t, filepath.Join(cwd, RegistryRelPath))
	third, err := Regenerate(cwd, home, RegenerateOptions{})
	if err != nil {
		t.Fatalf("third Regenerate() error = %v", err)
	}
	if third.Regenerated || third.Reason != "cache-hit" {
		t.Fatalf("nothing changed, want cache-hit, got %+v", third)
	}
	if readFile(t, agentsPath) != beforeAgents || readFile(t, filepath.Join(cwd, RegistryRelPath)) != beforeRegistry {
		t.Fatal("cache-hit must leave AGENTS.md and the registry byte-identical")
	}
}

// TestRegenerateVersionedViewIsIdempotent covers REQ-22.12 for the new view: a
// forced regeneration over unchanged inputs rewrites AGENTS.md byte for byte.
func TestRegenerateVersionedViewIsIdempotent(t *testing.T) {
	cwd := newGitRepo(t)
	home := t.TempDir()
	writeSkill(t, filepath.Join(cwd, ".gitignore"), ".claude/\n")
	writeSkill(t, filepath.Join(cwd, "skills", "x", "SKILL.md"), minimalSkill("x"))
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "y", "SKILL.md"), minimalSkill("y"))
	writeSkill(t, filepath.Join(cwd, AgentsRelPath), agentsFixture)
	agentsPath := filepath.Join(cwd, AgentsRelPath)

	if _, err := Regenerate(cwd, home, RegenerateOptions{}); err != nil {
		t.Fatalf("first Regenerate() error = %v", err)
	}
	first := readFile(t, agentsPath)

	cached, err := Regenerate(cwd, home, RegenerateOptions{})
	if err != nil {
		t.Fatalf("second Regenerate() error = %v", err)
	}
	if cached.Regenerated || cached.Reason != "cache-hit" || readFile(t, agentsPath) != first {
		t.Fatalf("second run must be a byte-identical cache-hit, got %+v", cached)
	}

	forced, err := Regenerate(cwd, home, RegenerateOptions{Force: true})
	if err != nil {
		t.Fatalf("forced Regenerate() error = %v", err)
	}
	if !forced.Regenerated || readFile(t, agentsPath) != first {
		t.Fatalf("forced run must rewrite AGENTS.md byte-identically, got %+v", forced)
	}
}

func TestVersionedFingerprintDependsOnTheVersionableSet(t *testing.T) {
	scan := Fingerprint(nil)
	both := map[string]bool{"/ws/a/SKILL.md": true, "/ws/b/SKILL.md": true}
	reordered := map[string]bool{"/ws/b/SKILL.md": true, "/ws/a/SKILL.md": true}
	onlyA := map[string]bool{"/ws/a/SKILL.md": true}

	if versionedFingerprint(scan, both) != versionedFingerprint(scan, reordered) {
		t.Fatal("the fingerprint must not depend on map iteration order")
	}
	if versionedFingerprint(scan, both) == versionedFingerprint(scan, onlyA) {
		t.Fatal("the fingerprint must change when the versionable set changes")
	}
	if versionedFingerprint(scan, nil) == scan {
		t.Fatal("the persisted fingerprint must differ from the bare scan fingerprint")
	}
}
