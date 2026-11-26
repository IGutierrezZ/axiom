package skills

import (
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

func TestDefaultSkillsIncludeSDDAndFoundationSkills(t *testing.T) {
	present := make(map[model.SkillID]bool)
	for _, skill := range DefaultSkills() {
		present[skill] = true
	}

	for _, want := range []model.SkillID{model.SkillSDDInit, model.SkillJudgmentDay, model.SkillGoTesting, model.SkillCreator} {
		if !present[want] {
			t.Fatalf("el conjunto por defecto debe incluir %q", want)
		}
	}
}

func TestDefaultSkillsCoverEverySelectableSkillExceptContributorOnes(t *testing.T) {
	defaults := DefaultSkills()
	all := AllSkillIDs()

	defaultSet := make(map[model.SkillID]bool, len(defaults))
	for _, id := range defaults {
		defaultSet[id] = true
	}

	var selectableOnly []model.SkillID
	for _, id := range all {
		if !defaultSet[id] {
			selectableOnly = append(selectableOnly, id)
		}
	}

	contributorOnly := map[model.SkillID]bool{
		model.SkillIssueCreation:       true,
		model.SkillCommentWriter:       true,
		model.SkillSystemicIssueTriage: true,
	}
	if len(selectableOnly) != len(contributorOnly) {
		t.Fatalf("el inventario seleccionable tiene %d skills fuera del conjunto por defecto (%v), want exactamente las %d de colaboradores", len(selectableOnly), selectableOnly, len(contributorOnly))
	}
	for _, id := range selectableOnly {
		if !contributorOnly[id] {
			t.Fatalf("el inventario seleccionable contiene la skill %q fuera del conjunto por defecto; solo las de colaboradores pueden quedar fuera", id)
		}
	}
}

func TestDefaultSkillsExcludeContributorSkills(t *testing.T) {
	excluded := []model.SkillID{
		model.SkillIssueCreation,
		model.SkillSystemicIssueTriage,
		model.SkillCommentWriter,
	}
	required := []model.SkillID{
		model.SkillChainedPR,
		model.SkillSkillRegistry,
		model.SkillWorkUnitCommits,
		model.SkillCognitiveDoc,
	}
	present := make(map[model.SkillID]bool)
	for _, skill := range DefaultSkills() {
		present[skill] = true
	}
	for _, skill := range excluded {
		if present[skill] {
			t.Errorf("DefaultSkills() incluye la skill de colaboradores %q", skill)
		}
	}
	for _, skill := range required {
		if !present[skill] {
			t.Errorf("DefaultSkills() no incluye la skill de producto %q", skill)
		}
	}
}

func TestDefaultSkillsReturnsAFreshSlice(t *testing.T) {
	first := DefaultSkills()
	first[0] = model.SkillCreator
	if got := DefaultSkills()[0]; got != model.SkillSDDInit {
		t.Fatalf("DefaultSkills()[0] = %q tras modificar un resultado anterior, want %q", got, model.SkillSDDInit)
	}
}

func TestAllSkillIDsIncludesEveryKnownSkill(t *testing.T) {
	all := AllSkillIDs()

	required := []model.SkillID{
		model.SkillSDDInit,
		model.SkillCreator,
		model.SkillSkillRegistry,
		model.SkillCognitiveDoc,
		model.SkillCommentWriter,
		model.SkillJudgmentDay,
		model.SkillImprover,
		model.SkillGoTesting,
		model.SkillSystemicIssueTriage,
	}

	skillSet := make(map[model.SkillID]struct{}, len(all))
	for _, skill := range all {
		skillSet[skill] = struct{}{}
	}

	for _, req := range required {
		if _, ok := skillSet[req]; !ok {
			t.Fatalf("AllSkillIDs() missing %q", req)
		}
	}
}

func TestRequestedBundledSkillsAreInDefaultSkillSet(t *testing.T) {
	required := []model.SkillID{
		model.SkillCreator,
		model.SkillSkillRegistry,
		model.SkillCognitiveDoc,
		model.SkillJudgmentDay,
		model.SkillSDDInit,
		model.SkillImprover,
	}

	skillSet := make(map[model.SkillID]struct{})
	for _, skill := range DefaultSkills() {
		skillSet[skill] = struct{}{}
	}

	for _, req := range required {
		if _, ok := skillSet[req]; !ok {
			t.Fatalf("DefaultSkills() missing requested bundled skill %q", req)
		}
	}
}
