package skills

import (
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
