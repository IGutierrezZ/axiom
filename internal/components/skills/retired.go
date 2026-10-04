package skills

import "github.com/IGutierrezZ/axiom/v3/internal/model"

// retiredSkills are skills that used to ship embedded and no longer do (#70).
// A persisted or explicit selection can still name them, and they have no
// embedded asset to inject, so selections must drop them before verification
// demands a SKILL.md that is never written. Copies already installed on disk
// are handled elsewhere; this list only keeps them out of the selection.
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
