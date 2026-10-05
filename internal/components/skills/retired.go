package skills

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// retiredSkillFile is the only file the retired skills ever shipped.
const retiredSkillFile = "SKILL.md"

// maxRetiredSkillBytes bounds what is read to prove ownership. Every shipped
// copy is far smaller, so a bigger file cannot be a pristine one.
const maxRetiredSkillBytes = 64 * 1024

// retiredSkills are skills that used to ship embedded and no longer do (#70).
// A persisted or explicit selection can still name them, and they have no
// embedded asset to inject, so selections must drop them before verification
// demands a SKILL.md that is never written. Copies already installed on disk
// are removed by RetireInstalled; this list also keeps them out of the selection.
var retiredSkills = []model.SkillID{
	"branch-pr",
	"gentle-ai-bench",
}

// IsRetired reports whether id names a skill that is no longer embedded.
func IsRetired(id model.SkillID) bool {
	for _, retired := range retiredSkills {
		if id == retired {
			return true
		}
	}

	return false
}

// WithoutRetired returns ids without the retired skills, preserving order.
// It never aliases or mutates the input slice.
func WithoutRetired(ids []model.SkillID) []model.SkillID {
	kept := make([]model.SkillID, 0, len(ids))
	for _, id := range ids {
		if !IsRetired(id) {
			kept = append(kept, id)
		}
	}

	return kept
}

// retiredFingerprints lists, per retired skill, the SHA-256 of every SKILL.md
// release that ever shipped, computed over the content with CRLF folded to LF
// (a Windows build could have installed CRLF copies). A copy is Axiom's own
// only when it matches one of them; anything else is the user's and stays.
var retiredFingerprints = map[model.SkillID][]string{
	"branch-pr": {
		"e6c67d0617d23e97fa7d27f1e3e92c7ac6d9ad182260a2d5c7efe6b969e31bd2",
		"90ab6ec413dc2ffd5fedb03330ad7d9def0971c5008129b2dbb36eaa659b8283",
		"8553fdde3397c7dc1c2fda2b4ca848baf5d9185d0db65635b151755c4e627dc8",
		"6192409014cafe16867ad46a7b66e95d0866d24907ffd8c4d1a92962c7c5ab54",
	},
	"gentle-ai-bench": {
		"a7c9576cde7bfadf30156e4c69a467b80b72de5ab0fc2a51168839dcbd109eab",
		"49ac672887b4afe88b107de17edd8c4dff34bc4ada54903bcd45a68532dcc91e",
	},
}

// RetiredCopies reports the retired skills found under one skills root.
type RetiredCopies struct {
	// Owned holds the SKILL.md paths proven to be unmodified Axiom copies.
	// After RetireInstalled it holds the ones actually removed.
	Owned []string
	// Kept holds the skill directories left in place: modified copies, links,
	// unreadable files and removals that failed. They are never touched.
	Kept []string
}

// InspectRetired is the read-only view of RetireInstalled: it writes nothing.
func InspectRetired(skillsDir string) RetiredCopies {
	return InspectRetiredAgainst(skillsDir, retiredFingerprints)
}

// RetireInstalled removes the unmodified installed copies of retired skills
// under skillsDir. Ownership is proven by content fingerprint, never by name;
// a skill directory that is a link, or whose SKILL.md differs, is kept. Only
// SKILL.md is deleted, and the directory only if that leaves it empty, so
// files the user added are never lost. A failed removal is reported as kept
// rather than aborting the caller: this is cleanup of a stale duplicate.
func RetireInstalled(skillsDir string) RetiredCopies {
	return RetireInstalledAgainst(skillsDir, retiredFingerprints)
}

// InspectRetiredAgainst is InspectRetired with an explicit fingerprint set, so
// callers can prove ownership against synthetic content in tests.
func InspectRetiredAgainst(skillsDir string, fingerprints map[model.SkillID][]string) RetiredCopies {
	var found RetiredCopies
	for _, id := range retiredSkills {
		dir := filepath.Join(skillsDir, string(id))
		switch inspectRetiredCopy(dir, fingerprints[id]) {
		case retiredOwned:
			found.Owned = append(found.Owned, filepath.Join(dir, retiredSkillFile))
		case retiredKept:
			found.Kept = append(found.Kept, dir)
		}
	}
	return found
}

// RetireInstalledAgainst is RetireInstalled with an explicit fingerprint set.
func RetireInstalledAgainst(skillsDir string, fingerprints map[model.SkillID][]string) RetiredCopies {
	var result RetiredCopies
	for _, id := range retiredSkills {
		dir := filepath.Join(skillsDir, string(id))
		path := filepath.Join(dir, retiredSkillFile)
		// Ownership is re-proven right before the delete, not trusted from an
		// earlier inspection.
		switch inspectRetiredCopy(dir, fingerprints[id]) {
		case retiredOwned:
			if err := os.Remove(path); err != nil {
				result.Kept = append(result.Kept, dir)
				continue
			}
			result.Owned = append(result.Owned, path)
			// Plain Remove: it fails on a non-empty directory, which keeps
			// whatever else the user put there.
			_ = os.Remove(dir)
		case retiredKept:
			result.Kept = append(result.Kept, dir)
		}
	}
	return result
}

type retiredState int

const (
	retiredAbsent retiredState = iota
	retiredOwned
	retiredKept
)

// inspectRetiredCopy classifies <dir>/SKILL.md. Anything that cannot be proven
// an unmodified regular file inside a real directory is kept.
func inspectRetiredCopy(dir string, fingerprints []string) retiredState {
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return retiredAbsent
	}
	if err != nil || !info.IsDir() || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return retiredKept
	}
	path := filepath.Join(dir, retiredSkillFile)
	info, err = os.Lstat(path)
	if os.IsNotExist(err) {
		return retiredAbsent
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxRetiredSkillBytes {
		return retiredKept
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return retiredKept
	}
	sum := sha256.Sum256(bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n")))
	digest := hex.EncodeToString(sum[:])
	for _, want := range fingerprints {
		if digest == want {
			return retiredOwned
		}
	}
	return retiredKept
}

// ErrRetiredSkillKept explains why a retired skill is still installed: sync
// only removes copies it can prove unmodified.
var ErrRetiredSkillKept = errors.New("retired skill still installed with local edits, so sync kept it; delete its directory if you no longer need it")

// StillNamedRetired returns the skill directories under skillsDir whose
// SKILL.md frontmatter name is still the retired ID. Only those compete with
// the Axiom replacement: a user who rewrote the skill and renamed it (for
// example to axiom-branch-pr) is not reported. It is read-only and needs no
// fingerprints, so it also covers copies kept because they were edited.
func StillNamedRetired(skillsDir string) []string {
	var dirs []string
	for _, id := range retiredSkills {
		dir := filepath.Join(skillsDir, string(id))
		path := filepath.Join(dir, retiredSkillFile)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxRetiredSkillBytes {
			continue
		}
		content, err := os.ReadFile(path)
		if err == nil && frontmatterName(string(content)) == string(id) {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// frontmatterName reads the top-level name key of a leading YAML block.
func frontmatterName(content string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		if value, ok := strings.CutPrefix(line, "name:"); ok {
			return strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return ""
}
