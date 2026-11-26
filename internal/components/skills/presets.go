package skills

import "github.com/IGutierrezZ/axiom/v3/internal/model"

// sddSkills are the SDD orchestrator skills — always included.
var sddSkills = []model.SkillID{
	model.SkillSDDInit,
	model.SkillSDDExplore,
	model.SkillSDDResearch,
	model.SkillSDDPropose,
	model.SkillSDDSpec,
	model.SkillSDDDesign,
	model.SkillSDDTasks,
	model.SkillSDDApply,
	model.SkillSDDVerify,
	model.SkillSDDArchive,
	model.SkillSDDOnboard,
	model.SkillJudgmentDay,
}

// contributorSkills are this repository's own workflow skills. They stay
// selectable through explicit `--skills` resolution, but the default skill set
// does not install them.
var contributorSkills = []model.SkillID{
	model.SkillIssueCreation,
	model.SkillCommentWriter,
	model.SkillSystemicIssueTriage,
}

// selectableFoundationSkills is the canonical display order of every non-SDD
// skill. Contributor skills keep their historical positions between the
// product skills; only their membership in the default set changes.
var selectableFoundationSkills = []model.SkillID{
	model.SkillGoTesting,
	model.SkillCreator,
	model.SkillImprover,
	model.SkillIssueCreation,
	model.SkillSkillRegistry,
	model.SkillChainedPR,
	model.SkillCognitiveDoc,
	model.SkillCommentWriter,
	model.SkillWorkUnitCommits,
	model.SkillSystemicIssueTriage,
}

// foundationSkills are the non-SDD product skills of the default set: the
// selectable inventory minus the contributor skills.
var foundationSkills = excludeSkills(selectableFoundationSkills, contributorSkills)

func excludeSkills(src, exclude []model.SkillID) []model.SkillID {
	excluded := make(map[model.SkillID]struct{}, len(exclude))
	for _, id := range exclude {
		excluded[id] = struct{}{}
	}
	out := make([]model.SkillID, 0, len(src))
	for _, id := range src {
		if _, ok := excluded[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}

// DefaultSkills returns the skills installed when none are chosen explicitly:
// the SDD suite plus the foundation skills. Contributor skills are excluded.
func DefaultSkills() []model.SkillID {
	all := make([]model.SkillID, 0, len(sddSkills)+len(foundationSkills))
	all = append(all, sddSkills...)
	all = append(all, foundationSkills...)
	return all
}

// AllSkillIDs returns every selectable skill ID in canonical display order:
// the SDD suite first, then the non-SDD skills. It is the full selectable
// inventory and includes the contributor skills that the default set omits.
func AllSkillIDs() []model.SkillID {
	all := make([]model.SkillID, 0, len(sddSkills)+len(selectableFoundationSkills))
	all = append(all, sddSkills...)
	all = append(all, selectableFoundationSkills...)
	return all
}
