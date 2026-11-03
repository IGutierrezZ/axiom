package sddstatus

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/consentenvelope"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

func gateAnswers(result SDDGovernanceGateResult) []string {
	answers := make([]string, 0, len(result.Choices))
	for _, choice := range result.Choices {
		answers = append(answers, choice.Answer)
	}
	return answers
}

// REQ-26.1 / REQ-26.2: a planning gate offers iterate, approve-push-and-stop
// and approve-and-continue, in that order, as one verb with shared selectors.
func TestGateQuestionOffersThreeExitsInOrder(t *testing.T) {
	for _, store := range []ArtifactStore{ArtifactStoreOpenSpec, ArtifactStoreHybrid} {
		for _, gate := range []kickoff.GateState{
			{Key: kickoff.GateSpec},
			{Key: kickoff.GateDesign},
			{Key: kickoff.GateTasks},
			{Key: kickoff.RoleApplyGate("core"), Blocks: "core"},
		} {
			result, err := newGovernanceGateQuestion("inc-1", "/repo", gate, ArtifactPaths{}, store)
			if err != nil {
				t.Fatalf("store %s gate %s: %v", store, gate.Key, err)
			}
			if got := strings.Join(gateAnswers(result), ","); got != "rejected,hand_off,approved" {
				t.Fatalf("store %s gate %s: answers = %s, want rejected,hand_off,approved", store, gate.Key, got)
			}
			if result.HandOffUnavailable != "" {
				t.Fatalf("store %s gate %s: handOffUnavailable = %q, want empty", store, gate.Key, result.HandOffUnavailable)
			}
			shared := "--cwd " + pathquote.Quote("/repo") + " --change inc-1 --gate " + string(gate.Key)
			wantTail := []string{
				" --decision rejected --reason <your-reason>",
				" --decision approved --hand-off --reason <your-reason>",
				" --decision approved --reason <your-reason>",
			}
			for i, choice := range result.Choices {
				want := gateRecordInvocationPrefix + shared + wantTail[i]
				if choice.Invocation != want {
					t.Fatalf("choice %s invocation = %q, want %q", choice.Answer, choice.Invocation, want)
				}
			}
		}
	}
}

// REQ-26.6: a store without files keeps iterate and continue and says why the
// third exit is missing.
func TestGateQuestionWithoutFilesDeclaresHandOffUnavailable(t *testing.T) {
	for _, store := range []ArtifactStore{ArtifactStoreNone, ArtifactStoreEngram} {
		result, err := newGovernanceGateQuestion("inc-1", "/repo", kickoff.GateState{Key: kickoff.GateSpec}, ArtifactPaths{}, store)
		if err != nil {
			t.Fatalf("store %s: %v", store, err)
		}
		if got := strings.Join(gateAnswers(result), ","); got != "rejected,approved" {
			t.Fatalf("store %s: answers = %s, want rejected,approved", store, got)
		}
		if result.HandOffUnavailable != string(store) {
			t.Fatalf("store %s: handOffUnavailable = %q", store, result.HandOffUnavailable)
		}
		if !strings.Contains(result.Value, "not available") || !strings.Contains(result.Value, string(store)) {
			t.Fatalf("store %s: value does not declare the hand-off exit unavailable: %q", store, result.Value)
		}
		for _, choice := range result.Choices {
			if strings.Contains(choice.Invocation, "--hand-off") {
				t.Fatalf("store %s: choice %s carries --hand-off", store, choice.Answer)
			}
		}
	}
}

// REQ-26.2: integration keeps its two-way choice whatever the store.
func TestGateQuestionIntegrationKeepsItsTwoChoices(t *testing.T) {
	for _, store := range []ArtifactStore{ArtifactStoreOpenSpec, ArtifactStoreHybrid, ArtifactStoreNone} {
		result, err := newGovernanceGateQuestion("inc-1", "/repo", kickoff.GateState{Key: kickoff.GateIntegration}, ArtifactPaths{}, store)
		if err != nil {
			t.Fatalf("store %s: %v", store, err)
		}
		if got := strings.Join(gateAnswers(result), ","); got != "approved,rejected" {
			t.Fatalf("store %s: answers = %s, want approved,rejected", store, got)
		}
		if result.HandOffUnavailable != "" {
			t.Fatalf("store %s: integration carries handOffUnavailable %q", store, result.HandOffUnavailable)
		}
	}
}

func TestGateQuestionValidateRejectsMalformedThreeExitShapes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*SDDGovernanceGateResult)
	}{
		{name: "two options without handOffUnavailable", mutate: func(r *SDDGovernanceGateResult) {
			r.Choices = []consentenvelope.Choice{r.Choices[0], r.Choices[2]}
		}},
		{name: "two options with an unknown handOffUnavailable", mutate: func(r *SDDGovernanceGateResult) {
			r.Choices = []consentenvelope.Choice{r.Choices[0], r.Choices[2]}
			r.HandOffUnavailable = "openspec"
		}},
		{name: "three options with handOffUnavailable", mutate: func(r *SDDGovernanceGateResult) { r.HandOffUnavailable = "none" }},
		{name: "integration with handOffUnavailable", mutate: func(r *SDDGovernanceGateResult) {
			*r = validIntegrationGateResult()
			r.HandOffUnavailable = "none"
		}},
		{name: "integration with three options", mutate: func(r *SDDGovernanceGateResult) {
			original := validGovernanceGateResult()
			r.Gate = string(kickoff.GateIntegration)
			r.Criteria = gateCriteria(kickoff.GateState{Key: kickoff.GateIntegration})
			r.Choices = original.Choices
		}},
		{name: "duplicate token", mutate: func(r *SDDGovernanceGateResult) { r.Choices[1].Answer = gateAnswerRejected }},
		{name: "duplicate invocation", mutate: func(r *SDDGovernanceGateResult) { r.Choices[2].Invocation = r.Choices[0].Invocation }},
		{name: "other verb on hand_off", mutate: func(r *SDDGovernanceGateResult) {
			r.Choices[1].Invocation = "axiom sdd-attempt grant --cwd . --change inc --hand-off"
		}},
		{name: "hand_off without --hand-off", mutate: func(r *SDDGovernanceGateResult) {
			r.Choices[1].Invocation = "axiom sdd gate record --cwd . --change inc --gate spec --decision approved --reason other"
		}},
		{name: "approved carrying --hand-off", mutate: func(r *SDDGovernanceGateResult) {
			r.Choices[2].Invocation = "axiom sdd gate record --cwd . --change inc --gate spec --decision approved --hand-off --reason other"
		}},
		{name: "rejected carrying --hand-off", mutate: func(r *SDDGovernanceGateResult) {
			r.Choices[0].Invocation = "axiom sdd gate record --cwd . --change inc --gate spec --decision rejected --hand-off"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validGovernanceGateResult()
			tt.mutate(&result)
			if err := result.Validate(); err == nil {
				t.Fatal("Validate() accepted a malformed gate question")
			}
		})
	}
}

func TestGateQuestionValidateAcceptsTwoExitsWithHandOffUnavailable(t *testing.T) {
	result := validGovernanceGateResult()
	result.Choices = []consentenvelope.Choice{result.Choices[0], result.Choices[2]}
	result.HandOffUnavailable = "none"
	if err := result.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want acceptance of the two-exit shape that explains itself", err)
	}
}

// REQ-26.2 (second scenario) and design R-4: widening the shared validator
// must never let the edit-authority envelope grow a third choice.
func TestEditAuthorityConsentStillRejectsThreeChoices(t *testing.T) {
	consent := newEditAuthorityConsent("inc-1", "/workspace", []string{"/workspace/service-a"}, "sdd-b2f4c0a1d8e64953a7c15b9d3e0f2a68", "")
	if err := consent.Validate(); err != nil {
		t.Fatalf("fixture precondition: %v", err)
	}
	consent.Choices = append(consent.Choices, consentenvelope.Choice{
		Answer: gateAnswerHandOff, Label: "Hand off", Effect: "Hands off.",
		Invocation: "axiom sdd gate record --cwd /workspace --change inc-1 --gate spec --decision approved --hand-off",
	})
	if err := consent.Validate(); err == nil {
		t.Fatal("the edit-authority envelope validated with three choices")
	}
}

func TestGateQuestionStorePrefersTheDeclaredStore(t *testing.T) {
	root := t.TempDir()
	if got := gateQuestionStore(root, ArtifactStoreOpenSpec); got != ArtifactStoreOpenSpec {
		t.Fatalf("no declaration: store = %q, want the resolved openspec", got)
	}
	writeFile(t, filepath.Join(root, "openspec", "config.yaml"), "artifact_store: none\n")
	if got := gateQuestionStore(root, ArtifactStoreOpenSpec); got != ArtifactStoreNone {
		t.Fatalf("declared none: store = %q, want none", got)
	}
}

// End to end through Resolve: the pending gate of a governed change carries
// three exits, or two plus handOffUnavailable under a declared `none`.
func TestResolveGateQuestionCarriesTheThreeExits(t *testing.T) {
	root := t.TempDir()
	changeName := "governed-three-exits"
	changeRoot := seedReadyChange(t, root, changeName, "- [ ] 1.1 Wire routes\n")
	sealFullstackKickoff(t, changeRoot, changeName, kickoff.ExecutionCheckpointed)

	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: changeName})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if status.GateQuestion == nil {
		t.Fatal("GateQuestion = nil, want the pending gate question")
	}
	if got := strings.Join(gateAnswers(*status.GateQuestion), ","); got != "rejected,hand_off,approved" {
		t.Fatalf("answers = %s, want rejected,hand_off,approved", got)
	}

	writeFile(t, filepath.Join(root, "openspec", "config.yaml"), "artifact_store: none\n")
	status, err = Resolve(ResolveOptions{CWD: root, ChangeName: changeName})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if status.GateQuestion == nil {
		t.Fatal("GateQuestion = nil under a declared none store")
	}
	if got := strings.Join(gateAnswers(*status.GateQuestion), ","); got != "rejected,approved" || status.GateQuestion.HandOffUnavailable != "none" {
		t.Fatalf("answers = %s handOffUnavailable = %q, want rejected,approved / none", got, status.GateQuestion.HandOffUnavailable)
	}
}
