package sddstatus

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// D-05 narrowed: a continuous seal projects its role roster but never a
// gate, and no precondition that only exists because gates do (the "integration"
// gate before archive, handoff.md before verify) may apply to it. The tests
// below pin both halves: the roster is visible, and every gate consumer stays
// inert.

func resolveContinuousRoles(t *testing.T, name string, tasksByRole map[string]string, roles ...string) (Status, string, string) {
	t.Helper()
	root := t.TempDir()
	changeRoot := seedMultiRoleChange(t, root, name, kickoff.ExecutionContinuous, tasksByRole, roles...)
	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: name})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	return status, changeRoot, root
}

func TestContinuousMultiRoleProjectsRosterWithoutGates(t *testing.T) {
	status, _, _ := resolveContinuousRoles(t, "roster", map[string]string{
		"backend": "- [x] 1.1 Model\n- [ ] 1.2 Endpoint\n", "frontend": "- [ ] 1.1 Screen\n",
	}, "backend", "frontend")

	if status.Governance == nil {
		t.Fatal("Governance = nil, want the roster of a sealed continuous change")
	}
	if want := (GovernanceRoster{Source: "kickoff", Roles: []string{"backend", "frontend"}}); !reflect.DeepEqual(status.Governance.Roster, want) {
		t.Fatalf("Roster = %+v, want %+v", status.Governance.Roster, want)
	}
	if len(status.Governance.Gates) != 0 {
		t.Fatalf("Gates = %+v, want none in continuous mode (REQ-21.7)", status.Governance.Gates)
	}
	if status.GateQuestion != nil {
		t.Fatalf("GateQuestion = %+v, want nil: continuous never presents a gate", status.GateQuestion)
	}
	if status.NextRecommended != "apply" || status.NextRecommended == "await-gate" {
		t.Fatalf("NextRecommended = %q, want apply", status.NextRecommended)
	}
	if len(status.BlockedReasons) != 0 {
		t.Fatalf("BlockedReasons = %v, want none (no gate may block a continuous change)", status.BlockedReasons)
	}

	encoded, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), `"gateQuestion"`) {
		t.Fatalf("serialized status names gateQuestion in continuous mode: %s", encoded)
	}
	projection, err := ProjectStatusV2(status)
	if err != nil {
		t.Fatalf("ProjectStatusV2() error = %v", err)
	}
	if projection.Governance == nil || !reflect.DeepEqual(projection.Governance.Roster.Roles, []string{"backend", "frontend"}) || len(projection.Governance.Gates) != 0 {
		t.Fatalf("v2 governance = %+v, want roster [backend frontend] and no gates", projection.Governance)
	}
}

// With every role done, archive is ready and recommended: the "integration"
// gate does not exist in continuous mode, so it cannot be missing (REQ-21.16
// constrains gated changes only).
func TestContinuousMultiRoleArchiveNotBlockedByIntegrationGate(t *testing.T) {
	status, changeRoot, root := resolveContinuousRoles(t, "finished", map[string]string{
		"backend": "- [x] 1.1 Endpoint\n", "frontend": "- [x] 1.1 Screen\n",
	}, "backend", "frontend")
	write(t, filepath.Join(changeRoot, "verify-report.md"), "# Verify Report\n\nVerdict: pass\n")

	status2, err := Resolve(ResolveOptions{CWD: root, ChangeName: "finished"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	for name, got := range map[string]Status{"without verify report": status, "with verify report": status2} {
		if got.ApplyState != ApplyAllDone || got.Dependencies.Archive != DependencyReady || got.NextRecommended != "archive" {
			t.Fatalf("%s: applyState = %q archive = %q next = %q, want all_done/ready/archive", name, got.ApplyState, got.Dependencies.Archive, got.NextRecommended)
		}
		if joined := strings.Join(got.BlockedReasons, "\n"); strings.Contains(joined, "integration") || len(got.BlockedReasons) != 0 {
			t.Fatalf("%s: BlockedReasons = %v, want none", name, got.BlockedReasons)
		}
	}
}

// Multi-role (and per_checkpoint) gated changes require handoff.md before
// verify (REQ-21.15). handoff.md is only written when the last role-apply
// gate is approved, which a continuous change never has.
func TestContinuousMultiRoleVerifyNotBlockedByMissingHandoff(t *testing.T) {
	status, changeRoot, _ := resolveContinuousRoles(t, "no-handoff", map[string]string{
		"backend": "- [x] 1.1 Endpoint\n", "frontend": "- [x] 1.1 Screen\n",
	}, "backend", "frontend")
	if status.Dependencies.Verify == DependencyBlocked {
		t.Fatalf("Dependencies.Verify = %q, want not blocked by a missing handoff.md", status.Dependencies.Verify)
	}
	if reasons := strings.Join(status.BlockedReasons, "\n"); strings.Contains(reasons, "handoff.md") {
		t.Fatalf("BlockedReasons = %v, want no handoff.md reason in continuous mode", status.BlockedReasons)
	}
	if got, err := loadGovernance(changeRoot); err != nil || got == nil || len(got.Gates) != 0 {
		t.Fatalf("loadGovernance() = (%+v, %v), want a gate-free roster", got, err)
	}
}

// A single fullstack role keeps projecting a one-role roster: a uniform
// shape for every sealed change, with no gate side effect.
func TestContinuousFullstackProjectsSingleRoleRosterWithoutGateEffects(t *testing.T) {
	root := t.TempDir()
	changeRoot := seedReadyChange(t, root, "solo", "- [x] 1.1 Wire routes\n")
	sealPerCheckpointFullstackKickoff(t, changeRoot, "solo", kickoff.ExecutionContinuous)

	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: "solo"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if status.Governance == nil || !reflect.DeepEqual(status.Governance.Roster.Roles, []string{"fullstack"}) || len(status.Governance.Gates) != 0 {
		t.Fatalf("Governance = %+v, want a one-role roster without gates", status.Governance)
	}
	if status.NextRecommended != "archive" || status.Dependencies.Archive != DependencyReady || status.Dependencies.Verify == DependencyBlocked || len(status.BlockedReasons) != 0 {
		t.Fatalf("next = %q archive = %q verify = %q blockedReasons = %v, want archive ready with nothing blocked even with handoff_policy per_checkpoint", status.NextRecommended, status.Dependencies.Archive, status.Dependencies.Verify, status.BlockedReasons)
	}
}

// The consumers must not rely on Gates being empty: even a continuous
// governance that (hypothetically) carried open, approved or handoff-worthy
// state is inert, because they check the execution style explicitly.
func TestGateConsumersAreInertForContinuousGovernance(t *testing.T) {
	continuous := &Governance{
		Kickoff: kickoff.Kickoff{Config: kickoff.FlowConfig{
			ExecutionStyle: kickoff.ExecutionContinuous, HandoffPolicy: kickoff.HandoffPerCheckpoint,
		}},
		Gates: []kickoff.GateState{
			{Key: kickoff.RoleApplyGate("backend"), Status: "approved"},
			{Key: kickoff.RoleApplyGate("frontend"), Status: "approved"},
			{Key: kickoff.GateIntegration, Status: "pending"},
		},
		Roster: GovernanceRoster{Source: "kickoff", Roles: []string{"backend", "frontend"}},
	}

	if governanceGated(continuous) {
		t.Fatal("governanceGated(continuous) = true, want false")
	}
	if gate, ok := firstOpenGate(continuous); ok {
		t.Fatalf("firstOpenGate() = %+v, want no gate in continuous mode", gate)
	}
	if next, ok := governanceNextRecommended(continuous); ok {
		t.Fatalf("governanceNextRecommended() = %q, want no override", next)
	}
	if got := resolveNextRecommended(Dependencies{Apply: DependencyReady}, ApplyReady, continuous); got != "apply" {
		t.Fatalf("resolveNextRecommended() = %q, want apply", got)
	}
	if state, reason := archiveDependencyFromGovernance(continuous, "c", DependencyReady); state != DependencyReady || reason != "" {
		t.Fatalf("archiveDependencyFromGovernance() = (%q, %q), want ready without a reason", state, reason)
	}
	if state, reason := verifyDependencyFromHandoff(continuous, t.TempDir(), DependencyReady); state != DependencyReady || reason != "" {
		t.Fatalf("verifyDependencyFromHandoff() = (%q, %q), want the baseline without a reason", state, reason)
	}
	reasons := artifactBlockedReasons(map[string]ArtifactState{}, TaskProgress{}, "c", continuous)
	baseline := artifactBlockedReasons(map[string]ArtifactState{}, TaskProgress{}, "c", nil)
	if !reflect.DeepEqual(reasons, baseline) {
		t.Fatalf("artifactBlockedReasons(continuous) = %+v, want identical to the ungoverned %+v", reasons, baseline)
	}
}
