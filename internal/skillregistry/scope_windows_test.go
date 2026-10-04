//go:build windows

package skillregistry

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// shortPathOf returns the 8.3 form of an existing path (every component
// shortened), or skips the test when the volume has 8.3 name generation off and
// Windows hands the long name back.
func shortPathOf(t *testing.T, path string) string {
	t.Helper()
	from, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	size, err := windows.GetShortPathName(from, nil, 0)
	if err != nil || size == 0 {
		t.Skipf("GetShortPathName(%q) is unavailable: %v", path, err)
	}
	buf := make([]uint16, size)
	n, err := windows.GetShortPathName(from, &buf[0], size)
	if err != nil || n == 0 || n >= size {
		t.Skipf("GetShortPathName(%q) failed: n=%d err=%v", path, n, err)
	}
	short := windows.UTF16ToString(buf[:n])
	if strings.EqualFold(short, path) {
		t.Skipf("8.3 short names are disabled on this volume (%q has no short form)", path)
	}
	return short
}

// init makes the shared link helper create junctions: unlike os.Symlink they
// need no privilege, so the link tests run on a default Windows account too.
func init() {
	makeDirLink = func(target, link string) error {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
			return fmt.Errorf("mklink /J: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
}

// newSpellingProject lays out a project whose directory name is long enough to
// have an 8.3 alias and returns it with its skill file.
func newSpellingProject(t *testing.T) (project, file string) {
	t.Helper()
	project = filepath.Join(t.TempDir(), "AProjectWithALongName")
	file = filepath.Join(project, "skills", "demo", "SKILL.md")
	writeSkill(t, file, minimalSkill("demo"))
	return project, file
}

// TestScopeForPathWindowsShortNames is the case behind the change: the same
// directory spelled with 8.3 short names on one side and long names on the
// other. %TEMP% is typically `C:\Users\IGUTIE~1\...`, so this happens whenever
// a hook passes --cwd in the short form while axiom.yaml spells a path long.
func TestScopeForPathWindowsShortNames(t *testing.T) {
	project, file := newSpellingProject(t)
	shortProject := shortPathOf(t, project)
	shortFile := shortPathOf(t, file)
	relFile := filepath.Join("skills", "demo", "SKILL.md")

	// Guard: without short-name handling this fixture is a real miss. The rule
	// every other path relies on would otherwise yield a `..`-prefixed path.
	if naive, err := filepath.Rel(shortProject, file); err != nil || !strings.HasPrefix(naive, "..") {
		t.Fatalf("fixture does not reproduce the short-name mismatch: filepath.Rel = %q, %v", naive, err)
	}

	assertScopeCases(t, []scopeCase{
		{name: "short cwd, long path", cwd: shortProject, path: file, wantRel: relFile},
		{name: "long cwd, short path", cwd: project, path: shortFile, wantRel: relFile},
		{name: "short cwd, short path", cwd: shortProject, path: shortFile, wantRel: relFile},
		{name: "short cwd, long path of a file not created yet", cwd: shortProject, path: filepath.Join(project, "new", "SKILL.md"), wantRel: filepath.Join("new", "SKILL.md")},
		{name: "short cwd and lower-case drive letter", cwd: flipDriveCase(shortProject), path: file, wantRel: relFile},
		{name: "short cwd, sibling of the project", cwd: shortProject, path: filepath.Join(filepath.Dir(project), "AProjectWithALongName2", "SKILL.md")},
		{name: "short cwd, the project itself", cwd: shortProject, path: project},
	})
}

// TestScopeForPathWindowsJunctions covers the reason identity is stat'ed rather
// than resolved: a junction inside the project must neither pull a skill out of
// it nor let the physically-outside target in.
func TestScopeForPathWindowsJunctions(t *testing.T) {
	project, _ := newSpellingProject(t)
	shortProject := shortPathOf(t, project)
	shared := filepath.Join(t.TempDir(), "shared-skill")
	writeSkill(t, filepath.Join(shared, "SKILL.md"), minimalSkill("linked"))
	linkOrSkip(t, shared, filepath.Join(project, "skills", "linked"))
	relLinked := filepath.Join("skills", "linked", "SKILL.md")

	assertScopeCases(t, []scopeCase{
		// The skill dir is a junction to a directory outside the project; by path
		// it is inside, whatever spelling cwd uses.
		{name: "junctioned skill dir, same spelling", cwd: project, path: filepath.Join(project, relLinked), wantRel: relLinked},
		{name: "junctioned skill dir, other case", cwd: strings.ToUpper(project), path: filepath.Join(project, relLinked), wantRel: relLinked},
		{name: "junctioned skill dir, short cwd", cwd: shortProject, path: filepath.Join(project, relLinked), wantRel: relLinked},
		// The shared target is outside; the junction pointing at it changes nothing.
		{name: "physical target of the junction", cwd: project, path: filepath.Join(shared, "SKILL.md")},
		{name: "physical target of the junction, short cwd", cwd: shortProject, path: filepath.Join(shared, "SKILL.md")},
	})
}

// writeRoleRepoYAML declares one role whose repository is an ABSOLUTE path.
// That is the one way a skill is discovered under a spelling of the project
// other than the cwd the caller passed: every default skill root is built from
// cwd itself, so it always shares cwd's spelling.
func writeRoleRepoYAML(t *testing.T, cwd, repoPath string) {
	t.Helper()
	cfg := workspace.WorkspaceConfig{
		Workspace: workspace.WorkspaceSection{
			Name:            "Spellings",
			Topology:        workspace.TopologyMonorepoEmbedded,
			SpecsRepository: ".",
		},
		Roles: map[string]workspace.RoleConfig{
			"backend": {Name: "Backend", Repositories: []workspace.RepositoryEntry{{Path: repoPath}}},
		},
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	writeSkill(t, filepath.Join(cwd, "axiom.yaml"), string(data))
}

// TestRegenerateWindowsSpellingOfCwd is the end-to-end form of the bug: cwd is
// passed in another case, or in 8.3 short form, while a role repository is
// declared as an absolute path in the project's original spelling. Before the
// fix the declared skill was classified "user" and silently left AGENTS.md and
// the Engram mirror; it must be listed with its repo-relative path.
func TestRegenerateWindowsSpellingOfCwd(t *testing.T) {
	spellings := map[string]func(t *testing.T, project string) string{
		"upper case": func(_ *testing.T, project string) string { return strings.ToUpper(project) },
		"drive letter case": func(_ *testing.T, project string) string {
			return flipDriveCase(project)
		},
		"8.3 short names": func(t *testing.T, project string) string { return shortPathOf(t, project) },
	}
	for name, respell := range spellings {
		t.Run(name, func(t *testing.T) {
			project := newGitRepo(t)
			home := t.TempDir()
			writeSkill(t, filepath.Join(project, "skills", "local", "SKILL.md"), describedSkill("local", "under cwd"))
			writeSkill(t, filepath.Join(project, "backend", "skills", "declared", "SKILL.md"), describedSkill("declared", "under an absolute role path"))
			writeRoleRepoYAML(t, project, filepath.Join(project, "backend"))
			writeSkill(t, filepath.Join(project, AgentsRelPath), agentsFixture)
			cwd := respell(t, project)

			mirror, mirrored := captureMirror()
			if _, err := Regenerate(cwd, home, RegenerateOptions{Mirror: mirror}); err != nil {
				t.Fatalf("Regenerate(%q) error = %v", cwd, err)
			}

			agents := readFile(t, filepath.Join(project, AgentsRelPath))
			for label, content := range map[string]string{"AGENTS.md": agents, "mirror": mirrored.Content} {
				for skill, wantPath := range map[string]string{
					"local":    "`skills/local/SKILL.md`",
					"declared": "`backend/skills/declared/SKILL.md`",
				} {
					if !hasRow(content, skill) || !strings.Contains(content, wantPath) {
						t.Fatalf("%s must list %q at %s with cwd spelled %q:\n%s", label, skill, wantPath, cwd, content)
					}
				}
				if strings.Contains(content, "..") {
					t.Fatalf("%s must not carry a `..` relative path:\n%s", label, content)
				}
			}
		})
	}
}
