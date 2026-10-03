package skillregistry

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// writeAxiomYAML writes an axiom.yaml for a single-repo workspace that declares
// the given skill roots verbatim (valid or not). The single role points at "."
// and specs_repository is "." so the config derives no extra skill directory:
// only skill_roots can change what ProjectSkillDirs returns.
func writeAxiomYAML(t *testing.T, cwd string, skillRoots ...string) {
	t.Helper()
	cfg := workspace.WorkspaceConfig{
		Workspace: workspace.WorkspaceSection{
			Name:            "DeclaredRoots",
			Topology:        workspace.TopologyMonorepoEmbedded,
			SpecsRepository: ".",
			SkillRoots:      skillRoots,
		},
		Roles: map[string]workspace.RoleConfig{
			"main": {Name: "Main", Repositories: []workspace.RepositoryEntry{{Path: "."}}},
		},
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	writeSkill(t, filepath.Join(cwd, "axiom.yaml"), string(data))
}

// TestProjectSkillDirsDeclaredRootsFollowSkills pins the insertion point:
// declared roots sit between `skills/` and the agent-native directories, in
// declaration order, and the rest of the list is untouched.
func TestProjectSkillDirsDeclaredRootsFollowSkills(t *testing.T) {
	cwd := t.TempDir()
	writeAxiomYAML(t, cwd)
	base := ProjectSkillDirs(cwd)
	if len(base) < 2 || base[0] != filepath.Join(cwd, "skills") {
		t.Fatalf("unexpected baseline directories: %v", base)
	}

	writeAxiomYAML(t, cwd, "internal/assets/skills", "docs/skills")
	got := ProjectSkillDirs(cwd)

	want := []string{
		filepath.Join(cwd, "skills"),
		filepath.Join(cwd, "internal", "assets", "skills"),
		filepath.Join(cwd, "docs", "skills"),
	}
	want = append(want, base[1:]...)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ProjectSkillDirs() =\n%v\nwant\n%v", got, want)
	}
}

// TestProjectSkillDirsDropsInvalidDeclaredRoots proves invalid entries are
// ignored without failing the scan or disturbing the valid ones.
func TestProjectSkillDirsDropsInvalidDeclaredRoots(t *testing.T) {
	cwd := t.TempDir()
	writeAxiomYAML(t, cwd)
	base := ProjectSkillDirs(cwd)

	absolute := t.TempDir()
	writeAxiomYAML(t, cwd, absolute, "../outside", ".", "", "  ", "docs/skills")
	got := ProjectSkillDirs(cwd)

	want := append([]string{filepath.Join(cwd, "skills"), filepath.Join(cwd, "docs", "skills")}, base[1:]...)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ProjectSkillDirs() =\n%v\nwant only the valid root inserted:\n%v", got, want)
	}
}

// TestProjectSkillDirsEmptyCwdUnchanged pins the static order that
// internal/assets/skill_registry_plugin_guard_test.go compares against the
// OpenCode plugin's PROJECT_MARKERS: without a config, declared roots must add
// nothing, and the list stays byte-identical.
func TestProjectSkillDirsEmptyCwdUnchanged(t *testing.T) {
	// ProjectSkillDirs("") resolves axiom.yaml from the process directory; run it
	// somewhere without one so the result is the static list.
	t.Chdir(t.TempDir())

	want := []string{
		"skills",
		filepath.Join(".opencode", "skills"),
		filepath.Join(".claude", "skills"),
		filepath.Join(".gemini", "skills"),
		filepath.Join(".cursor", "skills"),
		filepath.Join(".github", "skills"),
		filepath.Join(".codex", "skills"),
		filepath.Join(".qwen", "skills"),
		filepath.Join(".kiro", "skills"),
		filepath.Join(".openclaw", "skills"),
		filepath.Join(".pi", "skills"),
		filepath.Join(".agent", "skills"),
		filepath.Join(".agents", "skills"),
		filepath.Join(".atl", "skills"),
		filepath.Join(".hermes", "skills"),
	}
	if got := ProjectSkillDirs(""); !reflect.DeepEqual(got, want) {
		t.Fatalf("ProjectSkillDirs(\"\") =\n%v\nwant\n%v", got, want)
	}
}

// TestRegenerateDeclaredRootListsCanonicalSkill is the end-to-end case that
// motivates workspace.skill_roots: a skill that lives only in a directory that
// is not a default scan root (the canonical copy) reaches AGENTS.md at its
// versioned path, while the gitignored agent copy and any untracked agent copy
// scanned later never take its place.
func TestRegenerateDeclaredRootListsCanonicalSkill(t *testing.T) {
	cwd := newGitRepo(t)
	home := t.TempDir()
	writeSkill(t, filepath.Join(cwd, ".gitignore"), ".claude/\n")
	writeSkill(t, filepath.Join(cwd, "internal", "assets", "skills", "a", "SKILL.md"), describedSkill("a", "canonical a"))
	writeSkill(t, filepath.Join(cwd, ".claude", "skills", "a", "SKILL.md"), describedSkill("a", "ignored local copy of a"))
	writeSkill(t, filepath.Join(cwd, "skills", "b", "SKILL.md"), describedSkill("b", "plain skills b"))
	writeSkill(t, filepath.Join(cwd, AgentsRelPath), agentsFixture)
	agentsPath := filepath.Join(cwd, AgentsRelPath)

	// Without the declaration the canonical directory is not scanned and the only
	// copy of a is the ignored one, so AGENTS.md has no row for it.
	if _, err := Regenerate(cwd, home, RegenerateOptions{}); err != nil {
		t.Fatalf("Regenerate() without declaration error = %v", err)
	}
	agents := readFile(t, agentsPath)
	if hasRow(agents, "a") {
		t.Fatalf("a must be missing from AGENTS.md before the root is declared:\n%s", agents)
	}
	if !hasRow(agents, "b") {
		t.Fatalf("AGENTS.md must list b:\n%s", agents)
	}

	writeAxiomYAML(t, cwd, "internal/assets/skills")
	mirror, mirrored := captureMirror()
	if _, err := Regenerate(cwd, home, RegenerateOptions{Mirror: mirror}); err != nil {
		t.Fatalf("Regenerate() with declaration error = %v", err)
	}
	assertCanonicalA := func(stage string) {
		t.Helper()
		agents := readFile(t, agentsPath)
		for name, content := range map[string]string{"AGENTS.md": agents, "mirror": mirrored.Content} {
			if !hasRow(content, "a") || !strings.Contains(content, "`internal/assets/skills/a/SKILL.md`") {
				t.Fatalf("%s (%s) must list a at its declared root:\n%s", name, stage, content)
			}
			if !strings.Contains(content, "canonical a") {
				t.Fatalf("%s (%s) must carry the canonical description of a:\n%s", name, stage, content)
			}
			if strings.Contains(content, ".claude") || strings.Contains(content, ".gemini") {
				t.Fatalf("%s (%s) must not mention agent directories:\n%s", name, stage, content)
			}
			if !hasRow(content, "b") || !strings.Contains(content, "`skills/b/SKILL.md`") {
				t.Fatalf("%s (%s) must still list b at skills/:\n%s", name, stage, content)
			}
		}
	}
	assertCanonicalA("declared root")

	// An untracked, NOT ignored copy in .gemini is versionable, so it survives the
	// git filter; the declared root still wins because it is scanned first.
	writeSkill(t, filepath.Join(cwd, ".gemini", "skills", "a", "SKILL.md"), describedSkill("a", "untracked copy of a"))
	if _, err := Regenerate(cwd, home, RegenerateOptions{Force: true, Mirror: mirror}); err != nil {
		t.Fatalf("Regenerate() with .gemini copy error = %v", err)
	}
	assertCanonicalA("with untracked .gemini copy")
}
