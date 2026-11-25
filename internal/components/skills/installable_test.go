package skills

import (
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func TestInstallable(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []model.SkillID
		want []model.SkillID
	}{
		{"nil", nil, []model.SkillID{}},
		{"empty", []model.SkillID{}, []model.SkillID{}},
		{
			"every id has an embedded asset and is untouched",
			[]model.SkillID{model.SkillGoTesting, model.SkillChainedPR},
			[]model.SkillID{model.SkillGoTesting, model.SkillChainedPR},
		},
		{
			"mixed keeps the order of the live skills",
			[]model.SkillID{"branch-pr", model.SkillGoTesting, "gentle-ai-bench", model.SkillChainedPR},
			[]model.SkillID{model.SkillGoTesting, model.SkillChainedPR},
		},
		{"only retired", []model.SkillID{"branch-pr", "gentle-ai-bench"}, []model.SkillID{}},
		{
			"an id without an embedded asset is dropped, retired or not",
			[]model.SkillID{"rdd-defect-workflow", model.SkillGoTesting, "id-inventado"},
			[]model.SkillID{model.SkillGoTesting},
		},
		{"only ids without an embedded asset", []model.SkillID{"rdd-defect-workflow", "id-inventado"}, []model.SkillID{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := slices.Clone(tc.in)
			got := Installable(tc.in)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("Installable(%v) = %v, want %v", tc.in, got, tc.want)
			}
			if !slices.Equal(tc.in, in) {
				t.Fatalf("Installable mutated its input: %v, was %v", tc.in, in)
			}
		})
	}
}

func TestInstallableDoesNotAliasItsInput(t *testing.T) {
	in := []model.SkillID{model.SkillGoTesting, model.SkillChainedPR}
	got := Installable(in)
	got[0] = "otra"
	if in[0] != model.SkillGoTesting {
		t.Fatalf("Installable aliased its input: %v", in)
	}
}
