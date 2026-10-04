package skillregistry

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// scopeCase is one containment question. wantRel is the cwd-relative path when
// path is inside cwd; empty means the answer is "user".
type scopeCase struct {
	name    string
	cwd     string
	path    string
	wantRel string
}

// assertScopeCases checks the three consumers of the containment rule against
// the same table, because they must never disagree: ScopeForPath (the scope
// column), projectRelativePath (what versionableFiles hands to git) and
// renderSkillPath (the Path cell of AGENTS.md).
func assertScopeCases(t *testing.T, cases []scopeCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantScope := "user"
			if tc.wantRel != "" {
				wantScope = "project"
			}
			if got := ScopeForPath(tc.cwd, tc.path); got != wantScope {
				t.Errorf("ScopeForPath(%q, %q) = %q, want %q", tc.cwd, tc.path, got, wantScope)
			}
			rel, inside := projectRelativePath(tc.cwd, tc.path)
			if inside != (tc.wantRel != "") || rel != tc.wantRel {
				t.Errorf("projectRelativePath(%q, %q) = (%q, %v), want (%q, %v)",
					tc.cwd, tc.path, rel, inside, tc.wantRel, tc.wantRel != "")
			}
			wantCell := tc.path
			if tc.wantRel != "" {
				wantCell = filepath.ToSlash(tc.wantRel)
			}
			if got := renderSkillPath(tc.cwd, tc.path, ScopeForPath(tc.cwd, tc.path), PathRepoRelative); got != wantCell {
				t.Errorf("renderSkillPath(%q, %q) = %q, want %q", tc.cwd, tc.path, got, wantCell)
			}
		})
	}
}

// flipDriveCase swaps the case of a Windows drive letter and leaves any other
// path untouched.
func flipDriveCase(path string) string {
	volume := filepath.VolumeName(path)
	if len(volume) != 2 || volume[1] != ':' {
		return path
	}
	letter := volume[:1]
	if flipped := strings.ToLower(letter); flipped != letter {
		return flipped + path[1:]
	}
	return strings.ToUpper(letter) + path[1:]
}

// TestScopeForPath pins the rule every OS shares: strictly inside cwd, judged on
// path components rather than on a raw string prefix.
func TestScopeForPath(t *testing.T) {
	sep := string(os.PathSeparator)
	root := filepath.Join(t.TempDir(), "repo")
	file := filepath.Join(root, "skills", "x", "SKILL.md")
	sibling := filepath.Join(root+"2", "skills", "x", "SKILL.md")
	other := filepath.Join(t.TempDir(), "other", "SKILL.md")
	// The directories exist so that on Windows the negative cases also reach the
	// filesystem identity check instead of failing early on a missing cwd.
	for _, path := range []string{file, sibling, other} {
		writeSkill(t, path, minimalSkill("x"))
	}
	relFile := filepath.Join("skills", "x", "SKILL.md")

	assertScopeCases(t, []scopeCase{
		{name: "identical paths", cwd: root, path: root},
		{name: "child path", cwd: root, path: file, wantRel: relFile},
		{name: "direct child", cwd: root, path: filepath.Join(root, "SKILL.md"), wantRel: "SKILL.md"},
		{name: "sibling sharing the prefix", cwd: root, path: sibling},
		{name: "parent directory", cwd: root, path: filepath.Dir(root)},
		{name: "unrelated tree", cwd: root, path: other},
		{name: "cwd with a trailing separator", cwd: root + sep, path: file, wantRel: relFile},
		{
			name:    "unclean path that resolves inside",
			cwd:     root,
			path:    root + sep + "skills" + sep + ".." + sep + "skills" + sep + "x" + sep + "SKILL.md",
			wantRel: relFile,
		},
		{
			name: "unclean path that escapes into the sibling",
			cwd:  root,
			path: root + sep + ".." + sep + "repo2" + sep + "skills" + sep + "x" + sep + "SKILL.md",
		},
	})
}

// TestScopeForPathCaseSensitivity is the OS contract of the change: only on
// Windows are two spellings of one directory the same directory. Elsewhere the
// exact, case-sensitive lexical check stays in force.
func TestScopeForPathCaseSensitivity(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Repo")
	file := filepath.Join(root, "skills", "x", "SKILL.md")
	writeSkill(t, file, minimalSkill("x"))
	otherCaseCwd := filepath.Join(filepath.Dir(root), "REPO")

	want := "user"
	if runtime.GOOS == "windows" {
		want = "project"
	}
	if got := ScopeForPath(otherCaseCwd, file); got != want {
		t.Fatalf("ScopeForPath(%q, %q) = %q, want %q on %s", otherCaseCwd, file, got, want, runtime.GOOS)
	}
}

// TestScopeForPathWindowsCaseSpellings covers the case-only differences that
// filepath.Rel already treats as equal on Windows: the drive letter and any
// directory. Expected relative paths keep the spelling of the discovered path.
func TestScopeForPathWindowsCaseSpellings(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("case-insensitive paths are a Windows behaviour")
	}
	root := filepath.Join(t.TempDir(), "Project Dir")
	file := filepath.Join(root, "skills", "demo", "SKILL.md")
	sibling := filepath.Join(root+"2", "skills", "demo", "SKILL.md")
	writeSkill(t, file, minimalSkill("demo"))
	writeSkill(t, sibling, minimalSkill("demo"))

	parent := filepath.Dir(root)
	relFile := filepath.Join("skills", "demo", "SKILL.md")
	upperDirCwd := filepath.Join(parent, "PROJECT DIR")
	lowerDirFile := filepath.Join(parent, "project dir", "SKILLS", "demo", "skill.md")
	upperSibling := filepath.Join(parent, "PROJECT DIR2", "skills", "demo", "SKILL.md")

	assertScopeCases(t, []scopeCase{
		{name: "same spelling", cwd: root, path: file, wantRel: relFile},
		{name: "drive letter case differs in cwd", cwd: flipDriveCase(root), path: file, wantRel: relFile},
		{name: "drive letter case differs in path", cwd: root, path: flipDriveCase(file), wantRel: relFile},
		{name: "directory case differs in cwd", cwd: upperDirCwd, path: file, wantRel: relFile},
		{
			name:    "directory case differs in path",
			cwd:     root,
			path:    lowerDirFile,
			wantRel: filepath.Join("SKILLS", "demo", "skill.md"),
		},
		{name: "sibling sharing the prefix in another case", cwd: root, path: upperSibling},
		{name: "case-only difference on the cwd itself", cwd: upperDirCwd, path: root},
	})
}

// TestIsDescendantRel pins what counts as "below the base" in a filepath.Rel
// result, including the directory literally named `..x`.
func TestIsDescendantRel(t *testing.T) {
	tests := []struct {
		rel  string
		want bool
	}{
		{".", false},
		{"..", false},
		{filepath.Join("..", "x"), false},
		{filepath.Join("..", "..", "x", "y"), false},
		{"a", true},
		{filepath.Join("a", "b"), true},
		{"..x", true},
		{filepath.Join("..x", "y"), true},
	}
	for _, tc := range tests {
		if got := isDescendantRel(tc.rel); got != tc.want {
			t.Errorf("isDescendantRel(%q) = %v, want %v", tc.rel, got, tc.want)
		}
	}
}

// makeDirLink creates a directory link at link pointing to target. It is a
// symlink everywhere; scope_windows_test.go swaps in a junction, which needs no
// privilege.
var makeDirLink = os.Symlink

// linkOrSkip creates a directory link or skips the test where the platform or
// the account cannot make one.
func linkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := makeDirLink(target, link); err != nil {
		t.Skipf("directory links unavailable: %v", err)
	}
}

// TestRelativeByIdentity exercises the filesystem identity check directly. It
// is OS-independent, which is what lets Linux and macOS run it even though only
// Windows ever calls it from projectRelativePath. The link is a sibling of the
// target, so both spell the directory with the same number of components, which
// is the only kind of alias the check recognises.
func TestRelativeByIdentity(t *testing.T) {
	target := t.TempDir()
	file := filepath.Join(target, "skills", "x", "SKILL.md")
	writeSkill(t, file, minimalSkill("x"))
	link := filepath.Join(filepath.Dir(target), "link")
	linkOrSkip(t, target, link)
	deeperLink := filepath.Join(t.TempDir(), "nested", "link")
	if err := os.MkdirAll(filepath.Dir(deeperLink), 0o755); err != nil {
		t.Fatal(err)
	}
	linkOrSkip(t, target, deeperLink)
	relFile := filepath.Join("skills", "x", "SKILL.md")

	tests := []struct {
		name    string
		cwd     string
		path    string
		wantRel string
	}{
		{name: "cwd is an alias of the directory", cwd: link, path: file, wantRel: relFile},
		{name: "path reached through an alias", cwd: target, path: filepath.Join(link, "skills", "x", "SKILL.md"), wantRel: relFile},
		{name: "path that does not exist yet", cwd: link, path: filepath.Join(target, "new", "SKILL.md"), wantRel: filepath.Join("new", "SKILL.md")},
		{name: "the directory itself is not inside it", cwd: link, path: target},
		{name: "a shallower path is not inside it", cwd: link, path: filepath.Dir(target)},
		{name: "unrelated directory at the same depth", cwd: link, path: filepath.Join(t.TempDir(), "SKILL.md")},
		{name: "alias of a different depth is out of scope", cwd: deeperLink, path: file},
		{name: "missing cwd", cwd: filepath.Join(target, "missing"), path: file},
		{name: "cwd that is a file", cwd: file, path: filepath.Join(file, "child")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rel, inside := relativeByIdentity(tc.cwd, tc.path)
			if inside != (tc.wantRel != "") || rel != tc.wantRel {
				t.Fatalf("relativeByIdentity(%q, %q) = (%q, %v), want (%q, %v)",
					tc.cwd, tc.path, rel, inside, tc.wantRel, tc.wantRel != "")
			}
		})
	}
}

// TestRelativeByIdentityKeepsLinkedSkillDirsInside is the reason identity is
// stat'ed and not resolved with EvalSymlinks: a skill directory that links OUT
// of the project is still inside it by path, and a path physically outside
// never becomes inside because a project directory happens to link to it.
func TestRelativeByIdentityKeepsLinkedSkillDirsInside(t *testing.T) {
	project := t.TempDir()
	shared := t.TempDir()
	writeSkill(t, filepath.Join(shared, "linked", "SKILL.md"), minimalSkill("linked"))
	if err := os.MkdirAll(filepath.Join(project, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	linkOrSkip(t, filepath.Join(shared, "linked"), filepath.Join(project, "skills", "linked"))

	rel, inside := relativeByIdentity(project, filepath.Join(project, "skills", "linked", "SKILL.md"))
	if want := filepath.Join("skills", "linked", "SKILL.md"); !inside || rel != want {
		t.Fatalf("a linked skill dir must stay inside its project: got (%q, %v), want (%q, true)", rel, inside, want)
	}
	if rel, inside := relativeByIdentity(project, filepath.Join(shared, "linked", "SKILL.md")); inside {
		t.Fatalf("a path physically outside must not become inside through a link, got %q", rel)
	}
}

// TestScopeForPathThroughASymlinkedCwd pins that other OSes did not change: a
// cwd that is a symlink to the project is a different spelling, so only Windows
// (which resolves same-depth spellings by identity) calls the target path
// inside it.
func TestScopeForPathThroughASymlinkedCwd(t *testing.T) {
	target := t.TempDir()
	file := filepath.Join(target, "skills", "x", "SKILL.md")
	writeSkill(t, file, minimalSkill("x"))
	link := filepath.Join(filepath.Dir(target), "link")
	linkOrSkip(t, target, link)

	want := "user"
	if runtime.GOOS == "windows" {
		want = "project"
	}
	if got := ScopeForPath(link, file); got != want {
		t.Fatalf("ScopeForPath(%q, %q) = %q, want %q on %s", link, file, got, want, runtime.GOOS)
	}
}
