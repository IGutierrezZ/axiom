package sddstatus

import (
	"encoding/json"
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

// --- REQ-26.7: the hand-off is visible in `status` and never changes the route.

func approveSpecGate(t *testing.T, changeRoot string, handedOff ...bool) {
	t.Helper()
	for _, mark := range handedOff {
		if err := kickoff.AppendGate(changeRoot, kickoff.GateRecord{
			Gate: kickoff.GateSpec, Decision: kickoff.DecisionApproved, Reason: "cubre la intencion",
			HandedOff: mark, Actor: "reviewer",
		}); err != nil {
			t.Fatalf("kickoff.AppendGate() error = %v", err)
		}
	}
}

func handedOffFixture(t *testing.T, marks ...bool) (Status, StatusV2Projection) {
	t.Helper()
	root := t.TempDir()
	changeName := "governed-handed-off"
	changeRoot := seedReadyChange(t, root, changeName, "- [ ] 1.1 Wire routes\n")
	sealFullstackKickoff(t, changeRoot, changeName, kickoff.ExecutionCheckpointed)
	approveSpecGate(t, changeRoot, marks...)
	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: changeName})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	projection, err := ProjectStatusV2(status)
	if err != nil {
		t.Fatalf("ProjectStatusV2() error = %v", err)
	}
	return status, projection
}

func gateV2ByKey(t *testing.T, projection StatusV2Projection, key kickoff.GateKey) gateV2 {
	t.Helper()
	if projection.Governance == nil {
		t.Fatal("projection.Governance = nil")
	}
	for _, gate := range projection.Governance.Gates {
		if gate.Key == string(key) {
			return gate
		}
	}
	t.Fatalf("gate %s not projected", key)
	return gateV2{}
}

func TestStatusShowsTheHandOffOfSpecWithoutChangingTheRoute(t *testing.T) {
	normal, _ := handedOffFixture(t, false)
	status, projection := handedOffFixture(t, true)

	spec := gateV2ByKey(t, projection, kickoff.GateSpec)
	if spec.Status != "approved" || !spec.HandedOff || spec.HandedOffTo != "design" {
		t.Fatalf("spec gate = %+v, want approved, handedOff, handedOffTo design", spec)
	}
	if status.NextRecommended != normal.NextRecommended {
		t.Fatalf("NextRecommended = %q after a hand-off, %q after a normal approval; the route must not change", status.NextRecommended, normal.NextRecommended)
	}
	joined := strings.Join(status.Notes, "\n")
	if !strings.Contains(joined, `La compuerta "spec" se aprobó cediendo `+"`design`"+` a otra persona`) {
		t.Fatalf("Notes = %v, want the hand-off note for spec", status.Notes)
	}
	if strings.Join(normal.Notes, "") != "" {
		t.Fatalf("a normal approval carries notes: %v", normal.Notes)
	}
	// The hand-off never blocks anything.
	if strings.Join(status.BlockedReasons, "\n") != strings.Join(normal.BlockedReasons, "\n") {
		t.Fatalf("BlockedReasons = %v after a hand-off, %v after a normal approval", status.BlockedReasons, normal.BlockedReasons)
	}
	if status.Dependencies != normal.Dependencies {
		t.Fatalf("Dependencies = %+v after a hand-off, %+v after a normal approval", status.Dependencies, normal.Dependencies)
	}
}

func TestStatusProjectsNoHandOffForExits1And3(t *testing.T) {
	status, projection := handedOffFixture(t, false)
	spec := gateV2ByKey(t, projection, kickoff.GateSpec)
	if spec.HandedOff || spec.HandedOffTo != "" {
		t.Fatalf("spec gate = %+v, want no hand-off", spec)
	}
	if len(status.Notes) != 0 {
		t.Fatalf("Notes = %v, want none", status.Notes)
	}
	encoded, err := json.Marshal(projection.Governance.Gates)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "handedOff") {
		t.Fatalf("a change without the mark serializes a hand-off key: %s", encoded)
	}
}

// REQ-26.7 (last scenario): the mark is the one of the FIRST approved record.
func TestStatusHandOffMarkIsTheOneOfTheRecordThatClosedTheGate(t *testing.T) {
	_, withMarkFirst := handedOffFixture(t, true, false)
	if gate := gateV2ByKey(t, withMarkFirst, kickoff.GateSpec); !gate.HandedOff || gate.HandedOffTo != "design" {
		t.Fatalf("mark then plain approval: gate = %+v, want the hand-off kept", gate)
	}
	_, plainFirst := handedOffFixture(t, false, true)
	if gate := gateV2ByKey(t, plainFirst, kickoff.GateSpec); gate.HandedOff || gate.HandedOffTo != "" {
		t.Fatalf("plain approval then mark: gate = %+v, want no hand-off", gate)
	}
}

func twoRoleGovernance(coreApproved, webApproved, coreHandedOff, webHandedOff bool) *Governance {
	state := func(role string, approved, handedOff bool) kickoff.GateState {
		status := "pending"
		if approved {
			status = "approved"
		}
		return kickoff.GateState{Key: kickoff.RoleApplyGate(role), Status: status, Blocks: role, HandedOff: handedOff}
	}
	return &Governance{
		Kickoff: kickoff.Kickoff{Config: kickoff.FlowConfig{ExecutionStyle: kickoff.ExecutionCheckpointed}},
		Gates: []kickoff.GateState{
			state("core", coreApproved, coreHandedOff),
			state("web", webApproved, webHandedOff),
		},
		Roster: GovernanceRoster{Source: governanceRosterSourceKickoff, Roles: []string{"core", "web"}},
	}
}

func TestHandedOffToNamesTheNextRoleOrVerify(t *testing.T) {
	pendingWeb := twoRoleGovernance(true, false, true, false)
	if got := governanceHandedOffTo(pendingWeb, pendingWeb.Gates[0]); got != "apply:web" {
		t.Fatalf("core handed off with web pending: handedOffTo = %q, want apply:web", got)
	}
	closed := twoRoleGovernance(true, true, false, true)
	if got := governanceHandedOffTo(closed, closed.Gates[1]); got != "verify" {
		t.Fatalf("last role handed off: handedOffTo = %q, want verify", got)
	}
	if got := governanceHandedOffTo(closed, closed.Gates[0]); got != "" {
		t.Fatalf("a role without the mark reports handedOffTo = %q", got)
	}
	notes := handedOffNotes(closed)
	if len(notes) != 1 || !strings.Contains(notes[0], `"role-apply:web"`) || !strings.Contains(notes[0], "`verify`") {
		t.Fatalf("notes = %v, want one note naming role-apply:web and verify", notes)
	}
	if got := handedOffNotes(pendingWeb); len(got) != 1 || !strings.Contains(got[0], "`apply:web`") {
		t.Fatalf("notes = %v, want one note naming apply:web", got)
	}
}

// W-01: the note reflects the phase handed over when the record was written. core
// handed off apply:web while web was pending; web was approved afterwards without
// a mark, and the roster is now closed, but core still says apply:web.
func TestHandedOffToKeepsThePhaseHandedOverAtRecordingTime(t *testing.T) {
	closed := twoRoleGovernance(true, true, true, false)
	closed.Gates[0].HandedOffTo = "apply:web"
	if got := governanceHandedOffTo(closed, closed.Gates[0]); got != "apply:web" {
		t.Fatalf("core handed off apply:web before web closed: handedOffTo = %q, want apply:web", got)
	}
	notes := handedOffNotes(closed)
	if len(notes) != 1 || !strings.Contains(notes[0], `"role-apply:core"`) || !strings.Contains(notes[0], "`apply:web`") || strings.Contains(notes[0], "verify") {
		t.Fatalf("notes = %v, want one note naming role-apply:core and apply:web", notes)
	}
}

// A record written before handed_off_to existed has no persisted phase: it is
// still resolved by calculation against the current roster.
func TestHandedOffToFallsBackToTheCalculationWithoutAPersistedPhase(t *testing.T) {
	closed := twoRoleGovernance(true, true, true, false)
	if closed.Gates[0].HandedOffTo != "" {
		t.Fatalf("fixture carries a persisted phase: %+v", closed.Gates[0])
	}
	if got := governanceHandedOffTo(closed, closed.Gates[0]); got != "verify" {
		t.Fatalf("record without handed_off_to: handedOffTo = %q, want the calculated verify", got)
	}
}

// End to end through the ledger: the persisted phase survives a later approval
// of the other role.
func TestStatusProjectsThePersistedPhaseAfterTheRosterCloses(t *testing.T) {
	root := t.TempDir()
	changeName := "governed-persisted-handoff"
	changeRoot := seedReadyChange(t, root, changeName, "- [x] 1.1 Wire routes\n")
	sealFullstackKickoff(t, changeRoot, changeName, kickoff.ExecutionCheckpointed)
	if err := kickoff.AppendGate(changeRoot, kickoff.GateRecord{
		Gate: kickoff.GateSpec, Decision: kickoff.DecisionApproved, Reason: "cubre la intencion",
		HandedOff: true, HandedOffTo: "tasks", Actor: "reviewer",
	}); err != nil {
		t.Fatalf("kickoff.AppendGate() error = %v", err)
	}
	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: changeName})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	projection, err := ProjectStatusV2(status)
	if err != nil {
		t.Fatalf("ProjectStatusV2() error = %v", err)
	}
	if gate := gateV2ByKey(t, projection, kickoff.GateSpec); !gate.HandedOff || gate.HandedOffTo != "tasks" {
		t.Fatalf("spec gate = %+v, want the persisted phase tasks", gate)
	}
}

func TestHandedOffNotesAreEmptyWithoutGatesOrMarks(t *testing.T) {
	if got := handedOffNotes(nil); got != nil {
		t.Fatalf("nil governance: notes = %v", got)
	}
	continuous := twoRoleGovernance(true, true, true, true)
	continuous.Kickoff.Config.ExecutionStyle = kickoff.ExecutionContinuous
	if got := handedOffNotes(continuous); got != nil {
		t.Fatalf("continuous governance: notes = %v", got)
	}
	if got := handedOffNotes(twoRoleGovernance(true, true, false, false)); got != nil {
		t.Fatalf("no marks: notes = %v", got)
	}
}
