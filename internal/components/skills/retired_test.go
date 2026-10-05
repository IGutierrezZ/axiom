package skills

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func TestIsRetired(t *testing.T) {
	for _, tc := range []struct {
		id   model.SkillID
		want bool
	}{
		{"branch-pr", true},
		{"gentle-ai-bench", true},
		{model.SkillGoTesting, false},
		{model.SkillChainedPR, false},
		{"", false},
		{"Branch-PR", false},
	} {
		if got := IsRetired(tc.id); got != tc.want {
			t.Errorf("IsRetired(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestWithoutRetired(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []model.SkillID
		want []model.SkillID
	}{
		{"nil", nil, []model.SkillID{}},
		{"empty", []model.SkillID{}, []model.SkillID{}},
		{
			"no retired untouched",
			[]model.SkillID{model.SkillGoTesting, model.SkillChainedPR},
			[]model.SkillID{model.SkillGoTesting, model.SkillChainedPR},
		},
		{
			"mixed keeps order",
			[]model.SkillID{"branch-pr", model.SkillGoTesting, "gentle-ai-bench", model.SkillChainedPR},
			[]model.SkillID{model.SkillGoTesting, model.SkillChainedPR},
		},
		{"only retired", []model.SkillID{"branch-pr", "gentle-ai-bench"}, []model.SkillID{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := slices.Clone(tc.in)
			got := WithoutRetired(tc.in)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("WithoutRetired(%v) = %v, want %v", tc.in, got, tc.want)
			}
			if !slices.Equal(tc.in, in) {
				t.Fatalf("WithoutRetired mutated its input: %v, was %v", tc.in, in)
			}
		})
	}
}

func TestRetiredFingerprintsCoverExactlyTheRetiredSkills(t *testing.T) {
	if len(retiredFingerprints) != len(retiredSkills) {
		t.Fatalf("fingerprint ids = %d, retired skills = %d", len(retiredFingerprints), len(retiredSkills))
	}
	for _, id := range retiredSkills {
		prints := retiredFingerprints[id]
		if len(prints) == 0 {
			t.Errorf("retired skill %q has no fingerprint", id)
		}
		for _, p := range prints {
			if len(p) != sha256.Size*2 || p != string(bytes.ToLower([]byte(p))) {
				t.Errorf("fingerprint %q of %q is not lowercase sha256 hex", p, id)
			}
		}
	}
}

const (
	pristineBranchPR = "pristine branch-pr\nsecond line\n"
	pristineBench    = "pristine bench\n"
)

func testFingerprint(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func testFingerprints() map[model.SkillID][]string {
	return map[model.SkillID][]string{
		"branch-pr":       {testFingerprint("an older release\n"), testFingerprint(pristineBranchPR)},
		"gentle-ai-bench": {testFingerprint(pristineBench)},
	}
}

func writeSkill(t *testing.T, root, id, content string) string {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func TestRetireInstalled(t *testing.T) {
	for _, tc := range []struct {
		name      string
		content   string
		extraFile bool
		wantGone  bool
		wantDir   bool
	}{
		{name: "unmodified copy is removed with its directory", content: pristineBranchPR, wantGone: true},
		{name: "CRLF copy of a known release is removed", content: "pristine branch-pr\r\nsecond line\r\n", wantGone: true},
		{name: "modified copy is kept", content: pristineBranchPR + "my note\n", wantDir: true},
		{name: "oversized copy is kept", content: pristineBranchPR + string(bytes.Repeat([]byte("x"), maxRetiredSkillBytes)), wantDir: true},
		{name: "user file keeps the directory", content: pristineBranchPR, extraFile: true, wantGone: true, wantDir: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := writeSkill(t, root, "branch-pr", tc.content)
			extra := filepath.Join(filepath.Dir(path), "notes.txt")
			if tc.extraFile {
				if err := os.WriteFile(extra, []byte("mine"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			got := RetireInstalledAgainst(root, testFingerprints())

			if gone := !exists(path); gone != tc.wantGone {
				t.Fatalf("SKILL.md removed = %v, want %v", gone, tc.wantGone)
			}
			if dir := exists(filepath.Dir(path)); dir != tc.wantDir {
				t.Fatalf("directory present = %v, want %v", dir, tc.wantDir)
			}
			if removed := len(got.Owned) == 1 && len(got.Kept) == 0; removed != tc.wantGone {
				t.Fatalf("result = %+v, want removed=%v", got, tc.wantGone)
			}
			if tc.extraFile && !exists(extra) {
				t.Fatal("user file was deleted")
			}

			if again := RetireInstalledAgainst(root, testFingerprints()); len(again.Owned) != 0 {
				t.Fatalf("second run removed %v, want a no-op", again.Owned)
			}
		})
	}
}

func TestRetireInstalledMissingRootAndDirsAreNoOps(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	if got := RetireInstalledAgainst(missing, testFingerprints()); len(got.Owned)+len(got.Kept) != 0 {
		t.Fatalf("missing root = %+v, want empty", got)
	}
	empty := t.TempDir()
	if err := os.Mkdir(filepath.Join(empty, "branch-pr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := RetireInstalledAgainst(empty, testFingerprints()); len(got.Owned)+len(got.Kept) != 0 {
		t.Fatalf("directory without SKILL.md = %+v, want empty", got)
	}
}

func TestRetireInstalledHandlesEachRetiredSkillIndependently(t *testing.T) {
	root := t.TempDir()
	branch := writeSkill(t, root, "branch-pr", pristineBranchPR)
	bench := writeSkill(t, root, "gentle-ai-bench", "edited by the user")
	other := writeSkill(t, root, "go-testing", pristineBranchPR)

	got := RetireInstalledAgainst(root, testFingerprints())

	if exists(branch) || !exists(bench) || !exists(other) {
		t.Fatalf("branch-pr gone=%v, bench kept=%v, unrelated skill kept=%v", !exists(branch), exists(bench), exists(other))
	}
	if !slices.Equal(got.Owned, []string{branch}) || !slices.Equal(got.Kept, []string{filepath.Dir(bench)}) {
		t.Fatalf("result = %+v", got)
	}
}

func TestInspectRetiredWritesNothing(t *testing.T) {
	root := t.TempDir()
	owned := writeSkill(t, root, "branch-pr", pristineBranchPR)
	kept := writeSkill(t, root, "gentle-ai-bench", "edited")

	got := InspectRetiredAgainst(root, testFingerprints())

	if !slices.Equal(got.Owned, []string{owned}) || !slices.Equal(got.Kept, []string{filepath.Dir(kept)}) {
		t.Fatalf("inspection = %+v", got)
	}
	if !exists(owned) || !exists(kept) {
		t.Fatal("inspection removed a file")
	}
}

func TestRetireInstalledKeepsLinkedSkillDirectory(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	linked := writeSkill(t, target, "real", pristineBranchPR)
	link := filepath.Join(root, "branch-pr")
	if err := os.Symlink(filepath.Dir(linked), link); err != nil {
		if runtime.GOOS != "windows" {
			t.Skipf("cannot create symlink: %v", err)
		}
		if out, jerr := exec.Command("cmd", "/c", "mklink", "/J", link, filepath.Dir(linked)).CombinedOutput(); jerr != nil {
			t.Skipf("cannot create symlink or junction: %v / %v: %s", err, jerr, out)
		}
	}

	got := RetireInstalledAgainst(root, testFingerprints())

	if !exists(linked) {
		t.Fatal("the link target was modified through the link")
	}
	if len(got.Owned) != 0 || !slices.Equal(got.Kept, []string{link}) {
		t.Fatalf("result = %+v, want the link kept", got)
	}
}
