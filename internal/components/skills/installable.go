package skills

import (
	"io/fs"

	"github.com/IGutierrezZ/axiom/v3/internal/assets"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// Installable returns ids without the skills that cannot be installed: the
// retired ones and any id with no embedded `skills/<id>` directory (for example
// a skill a previous release shipped and this one does not). Injection already
// skips such ids, but post-sync verification demands a SKILL.md for every
// selected id, so a persisted or explicit selection must drop them before both.
// It preserves order and never aliases or mutates the input slice.
func Installable(ids []model.SkillID) []model.SkillID {
	kept := make([]model.SkillID, 0, len(ids))
	for _, id := range ids {
		if IsRetired(id) || !embedded(id) {
			continue
		}
		kept = append(kept, id)
	}

	return kept
}

// embedded reports whether assets.FS carries a directory for the skill.
func embedded(id model.SkillID) bool {
	entries, err := fs.ReadDir(assets.FS, "skills/"+string(id))
	return err == nil && len(entries) > 0
}
