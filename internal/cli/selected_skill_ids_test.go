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

	t.Run("explicit list of only retired skills stays empty without preset fallback", func(t *testing.T) {
		got := selectedSkillIDs(model.Selection{
			Preset: model.PresetFullGentleman,
			Skills: []model.SkillID{"branch-pr", "gentle-ai-bench"},
		})
		if len(got) != 0 {
			t.Fatalf("selectedSkillIDs() = %v, want empty (no preset fallback)", got)
		}
	})

	t.Run("preset path is unaffected", func(t *testing.T) {
		preset := model.PresetFullGentleman
		got := selectedSkillIDs(model.Selection{Preset: preset})
		if want := skills.SkillsForPreset(preset); !slices.Equal(got, want) {
			t.Fatalf("selectedSkillIDs() = %v, want preset skills %v", got, want)
		}
	})
}
