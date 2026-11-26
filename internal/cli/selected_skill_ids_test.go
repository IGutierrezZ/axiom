package cli

import (
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/components/skills"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func TestSelectedSkillIDsDropsRetired(t *testing.T) {
	t.Run("mixed explicit list keeps only live skills in order", func(t *testing.T) {
		got := selectedSkillIDs(model.Selection{Skills: []model.SkillID{
			model.SkillGoTesting, "branch-pr", model.SkillChainedPR, "gentle-ai-bench",
		}})
		want := []model.SkillID{model.SkillGoTesting, model.SkillChainedPR}
		if !slices.Equal(got, want) {
			t.Fatalf("selectedSkillIDs() = %v, want %v", got, want)
		}
	})

	t.Run("explicit list of only retired skills stays empty without default fallback", func(t *testing.T) {
		got := selectedSkillIDs(model.Selection{
			Skills: []model.SkillID{"branch-pr", "gentle-ai-bench"},
		})
		if len(got) != 0 {
			t.Fatalf("selectedSkillIDs() = %v, want empty (no default fallback)", got)
		}
	})

	t.Run("a skill with no embedded asset is dropped from an explicit list", func(t *testing.T) {
		got := selectedSkillIDs(model.Selection{Skills: []model.SkillID{
			model.SkillGoTesting, "rdd-defect-workflow", model.SkillChainedPR,
		}})
		want := []model.SkillID{model.SkillGoTesting, model.SkillChainedPR}
		if !slices.Equal(got, want) {
			t.Fatalf("selectedSkillIDs() = %v, want %v", got, want)
		}
	})

	t.Run("no explicit skills falls back to the default set", func(t *testing.T) {
		got := selectedSkillIDs(model.Selection{})
		if want := skills.DefaultSkills(); !slices.Equal(got, want) {
			t.Fatalf("selectedSkillIDs() = %v, want default skills %v", got, want)
		}
	})
}
