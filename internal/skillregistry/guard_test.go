package skillregistry

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// homeAgentsContent is the AGENTS.md a fake home carries, so a test can prove a
// refused Regenerate left it byte for byte as it found it.
const homeAgentsContent = "# Notes that belong to the user, not to a project\n"

// newFakeHome lays out a stand-in for the user's home directory that, like the
// real one, holds a project skills directory (`.claude/skills`, which is what
// makes hasProjectMarker true there) and an AGENTS.md. The directory name is
// long enough to have an 8.3 alias. It is always a temp directory: no test here
// may ever point at the real home.
func newFakeHome(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "AHomeWithALongName")
	writeSkill(t, filepath.Join(home, ".claude", "skills", "demo", "SKILL.md"), minimalSkill("demo"))
	writeSkill(t, filepath.Join(home, AgentsRelPath), homeAgentsContent)
	return home
}

// assertHomeRefused is the contract of the home guard seen from both of its
// layers: RefreshSkip skips the directory as the home directory, and Regenerate
// itself refuses to write there even when a caller never consulted RefreshSkip.
// cwd and homeArg are two spellings of the fake home; the checks fail loudly if
// the fixture does not hold (the home marker exists, so without the guard the
// registry WOULD be written).
func assertHomeRefused(t *testing.T, cwd, homeArg, home string) {
	t.Helper()
	if got := RefreshSkip(cwd, homeArg); got != SkipHomeDirectory {
		t.Errorf("RefreshSkip(%q, %q) = %q, want %q", cwd, homeArg, got, SkipHomeDirectory)
	}

	mirror, mirrored := captureMirror()
	_, err := Regenerate(cwd, homeArg, RegenerateOptions{Force: true, Mirror: mirror})
	if err == nil {
		t.Errorf("Regenerate(%q, %q) must refuse the home directory", cwd, homeArg)
	} else if !strings.Contains(err.Error(), string(SkipHomeDirectory)) {
		t.Errorf("Regenerate(%q, %q) error = %q, want it to name %q", cwd, homeArg, err, SkipHomeDirectory)
	}
	if mirrored.TopicKey != "" {
		t.Errorf("a refused Regenerate must not reach the Engram mirror, got %+v", *mirrored)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".atl")); !os.IsNotExist(statErr) {
		t.Errorf("a refused Regenerate must not create .atl in the home directory: stat err = %v", statErr)
	}
	if got := readFile(t, filepath.Join(home, AgentsRelPath)); got != homeAgentsContent {
		t.Errorf("a refused Regenerate must not touch the home AGENTS.md, got %q", got)
	}
}

// otherCaseSpelling returns dir in upper case when the volume resolves that to
// the same directory (Windows and default macOS volumes), and skips the test on a
// case-sensitive one, where no such alias exists.
func otherCaseSpelling(t *testing.T, dir string) string {
	t.Helper()
	spelled := strings.ToUpper(dir)
	if spelled == dir || !pathidentity.SameDirectory(spelled, dir) {
		t.Skipf("the volume is case-sensitive: %q is not an alias of %q", spelled, dir)
	}
	return spelled
}

// aliasThroughLink returns a path that reaches dir through a directory link.
func aliasThroughLink(t *testing.T, dir string) string {
	t.Helper()
	link := filepath.Join(t.TempDir(), "alias")
	linkOrSkip(t, dir, link)
	return link
}

func TestRefreshSkipFilesystemRoot(t *testing.T) {
	root := "/"
	if runtime.GOOS == "windows" {
		root = filepath.VolumeName(os.Getenv("SystemDrive")) + `\`
	}
	home := t.TempDir()
	if got := RefreshSkip(root, home); got != SkipFilesystemRoot {
		t.Fatalf("RefreshSkip(%q) = %q, want %q", root, got, SkipFilesystemRoot)
	}
}

func TestRefreshSkipHomeDirectoryEvenWithMarkers(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".atl"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := RefreshSkip(home, home); got != SkipHomeDirectory {
		t.Fatalf("RefreshSkip(home, home) = %q, want %q", got, SkipHomeDirectory)
	}
}

func TestRefreshSkipNoProjectMarker(t *testing.T) {
	dir := t.TempDir()
	if got := RefreshSkip(dir, t.TempDir()); got != SkipNoProjectMarker {
		t.Fatalf("RefreshSkip(markerless dir) = %q, want %q", got, SkipNoProjectMarker)
	}
}

func TestRefreshSkipAcceptsProjectMarkers(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, dir string)
	}{
		{"git directory", func(t *testing.T, dir string) {
			if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"git worktree file", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /elsewhere\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"existing atl registry", func(t *testing.T, dir string) {
			if err := os.MkdirAll(filepath.Join(dir, ".atl"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"project skills workspace", func(t *testing.T, dir string) {
			if err := os.MkdirAll(filepath.Join(dir, ".opencode", "skills"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"axiom.yaml file", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "axiom.yaml"), []byte("workspace:\n  name: Test\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{".axiom-workspace pointer file", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, ".axiom-workspace"), []byte("config: repo-specs/axiom.yaml\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)
			if got := RefreshSkip(dir, t.TempDir()); got != SkipNone {
				t.Fatalf("RefreshSkip(%s) = %q, want SkipNone", tc.name, got)
			}
		})
	}
}

// TestHomeDirectoryIsRefusedByIdentity covers the defect behind the guard: a
// home that a hook spells differently from %USERPROFILE% (another case, a link)
// used to compare unequal, so the directory was initialized as a project and
// ~/.atl appeared. The Windows-only spellings live in guard_windows_test.go.
func TestHomeDirectoryIsRefusedByIdentity(t *testing.T) {
	cases := []struct {
		name string
		// spell returns the cwd and the home argument for a fake home.
		spell func(t *testing.T, home string) (cwd, homeArg string)
		// alias is true when the two spellings must differ as strings, which is
		// what makes the case a regression test and not a restatement of equality.
		alias bool
	}{
		{
			name:  "same spelling",
			spell: func(_ *testing.T, home string) (string, string) { return home, home },
		},
		{
			name: "cwd with dot segments",
			spell: func(_ *testing.T, home string) (string, string) {
				sep := string(os.PathSeparator)
				return home + sep + "x" + sep + ".." + sep, home
			},
		},
		{
			name:  "cwd in another case",
			alias: true,
			spell: func(t *testing.T, home string) (string, string) { return otherCaseSpelling(t, home), home },
		},
		{
			name:  "home in another case",
			alias: true,
			spell: func(t *testing.T, home string) (string, string) { return home, otherCaseSpelling(t, home) },
		},
		{
			name:  "cwd through a link",
			alias: true,
			spell: func(t *testing.T, home string) (string, string) { return aliasThroughLink(t, home), home },
		},
		{
			name:  "home through a link",
			alias: true,
			spell: func(t *testing.T, home string) (string, string) { return home, aliasThroughLink(t, home) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := newFakeHome(t)
			if err := os.MkdirAll(filepath.Join(home, "x"), 0o755); err != nil {
				t.Fatal(err)
			}
			cwd, homeArg := tc.spell(t, home)
			if tc.alias && filepath.Clean(cwd) == filepath.Clean(homeArg) {
				t.Fatalf("fixture does not reproduce the mismatch: both spellings are %q", cwd)
			}
			assertHomeRefused(t, cwd, homeArg, home)
		})
	}
}

// TestDirectoriesNearHomeAreStillProjects keeps the identity check from
// over-blocking: only the home directory itself is refused, never a project
// below it, a sibling that merely shares its name as a prefix, or any directory
// when no home is known.
func TestDirectoriesNearHomeAreStillProjects(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, home string) (cwd, homeArg string)
	}{
		{
			name: "project below home",
			setup: func(_ *testing.T, home string) (string, string) {
				return filepath.Join(home, "project"), home
			},
		},
		{
			name: "sibling sharing the name of home as a prefix",
			setup: func(_ *testing.T, home string) (string, string) {
				return home + "2", home
			},
		},
		{
			name: "empty home",
			setup: func(_ *testing.T, home string) (string, string) {
				return filepath.Join(home, "project"), ""
			},
		},
		{
			name: "relative home never aliases the process directory",
			setup: func(t *testing.T, home string) (string, string) {
				project := filepath.Join(home, "project")
				if err := os.MkdirAll(project, 0o755); err != nil {
					t.Fatal(err)
				}
				t.Chdir(project)
				return project, "."
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := newFakeHome(t)
			cwd, homeArg := tc.setup(t, home)
			if err := os.MkdirAll(filepath.Join(cwd, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			if got := RefreshSkip(cwd, homeArg); got != SkipNone {
				t.Fatalf("RefreshSkip(%q, %q) = %q, want SkipNone", cwd, homeArg, got)
			}
			mirror, _ := captureMirror()
			if _, err := Regenerate(cwd, homeArg, RegenerateOptions{Mirror: mirror}); err != nil {
				t.Fatalf("Regenerate(%q, %q) must proceed in a project: %v", cwd, homeArg, err)
			}
			if _, err := os.Stat(filepath.Join(cwd, RegistryRelPath)); err != nil {
				t.Fatalf("the project registry must be written: %v", err)
			}
		})
	}
}

// TestFilesystemRootThroughALinkIsRefused covers a directory that merely
// resolves to the filesystem root. The lexical test only recognises a root
// spelled as one; identity against the root of the volume catches the link.
// Regenerate is deliberately not called here: if the guard failed it would
// write at the real root.
func TestFilesystemRootThroughALinkIsRefused(t *testing.T) {
	root := "/"
	if runtime.GOOS == "windows" {
		root = filepath.VolumeName(os.Getenv("SystemDrive")) + `\`
	}
	link := filepath.Join(t.TempDir(), "rootlink")
	linkOrSkip(t, root, link)
	// Remove the link itself before the temp directory is swept, so no cleanup
	// can ever walk into the root it points at (os.Remove unlinks a symlink or
	// junction without following it).
	t.Cleanup(func() { _ = os.Remove(link) })

	if got := RefreshSkip(link, t.TempDir()); got != SkipFilesystemRoot {
		t.Fatalf("RefreshSkip(%q) = %q, want %q", link, got, SkipFilesystemRoot)
	}
}
