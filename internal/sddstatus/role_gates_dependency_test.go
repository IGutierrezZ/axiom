package sddstatus

import (
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/handoff"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// REQ-21.15 and DD-10: the global verify phase opens when every role of a
// multi-role roster has its role-apply gate approved. No handoff.md takes part
// in that decision, and handoff_policy has no effect on it (REQ-26.9).

// sealPerCheckpointFullstackKickoff seals a single-role "fullstack" kickoff
// with handoff_policy per_checkpoint, the value REQ-26.9 turns into a no-op.
func sealPerCheckpointFullstackKickoff(t *testing.T, changeRoot, changeName string, style kickoff.ExecutionStyle) {
	t.Helper()
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: changeName, SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: style, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffPerCheckpoint,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	})
	if err != nil || !ok {
		t.Fatalf("sealPerCheckpointFullstackKickoff: kickoff.Seal() = (ok=%v, err=%v), want a successful preparation seal", ok, err)
	}
}

func writeLegacyIntegrationHandoff(t *testing.T, changeRoot, change string, status handoff.Status) {
	t.Helper()
	h := &handoff.Handoff{
		Metadata: handoff.Metadata{
			Change: change, FromPhase: handoff.PhaseApply, ToPhase: handoff.PhaseVerify,
			FromRole: "core", ToRole: "web", Timestamp: time.Now().UTC(), Status: status,
		},
		Sections: handoff.Sections{
			ExecutiveSummary: "resumen", Artifacts: "artefactos", Decisions: "decisiones",
			RisksAndBlockers: "riesgos", DirectInstructions: "instrucciones",
		},
	}
	if err := handoff.WriteFile(changeRoot+"/handoff.md", h); err != nil {
		t.Fatalf("handoff.WriteFile() error = %v", err)
	}
}

func approveGates(t *testing.T, changeRoot string, gates ...kickoff.GateKey) {
	t.Helper()
	for _, gate := range gates {
		if err := kickoff.AppendGate(changeRoot, kickoff.GateRecord{
			Gate: gate, Decision: kickoff.DecisionApproved, Reason: "conforme", Actor: "reviewer",
		}); err != nil {
			t.Fatalf("kickoff.AppendGate(%s) error = %v", gate, err)
		}
	}
}

func rejectGate(t *testing.T, changeRoot string, gate kickoff.GateKey) {
	t.Helper()
	if err := kickoff.AppendGate(changeRoot, kickoff.GateRecord{
		Gate: gate, Decision: kickoff.DecisionRejected, Reason: "falta cobertura", Actor: "reviewer",
	}); err != nil {
		t.Fatalf("kickoff.AppendGate(%s) error = %v", gate, err)
	}
}

// resolveTwoRoleChange builds a checkpointed change with roles core and web,
// every task done, and the planning gates approved, then resolves it.
func resolveTwoRoleChange(t *testing.T, name string, mutate func(changeRoot string)) Status {
	t.Helper()
	root := t.TempDir()
	changeRoot := seedMultiRoleChange(t, root, name, kickoff.ExecutionCheckpointed, map[string]string{
		"core": "- [x] 1.1 Model\n", "web": "- [x] 1.1 Screen\n",
	}, "core", "web")
	approveGates(t, changeRoot, kickoff.GateSpec, kickoff.GateDesign, kickoff.GateTasks)
	if mutate != nil {
		mutate(changeRoot)
	}
	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: name})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	return status
}

func TestVerifyBlockedWhileARoleHasNoApprovedRoleApplyGate(t *testing.T) {
	status := resolveTwoRoleChange(t, "two-roles-open", func(changeRoot string) {
		approveGates(t, changeRoot, kickoff.RoleApplyGate("core"))
	})
	if status.Dependencies.Verify != DependencyBlocked {
		t.Fatalf("Dependencies.Verify = %q, want blocked while web has no approved role-apply gate", status.Dependencies.Verify)
	}
	reasons := strings.Join(status.BlockedReasons, "\n")
	if !strings.Contains(reasons, "web") || strings.Contains(reasons, "core") {
		t.Fatalf("BlockedReasons = %v, want a reason naming web and not core", status.BlockedReasons)
	}
}

func TestVerifyReadyOnceEveryRoleApprovedWithoutAHandoffFile(t *testing.T) {
	status := resolveTwoRoleChange(t, "two-roles-closed", func(changeRoot string) {
		approveGates(t, changeRoot, kickoff.RoleApplyGate("core"), kickoff.RoleApplyGate("web"))
	})
	if status.Dependencies.Verify != DependencyReady {
		t.Fatalf("Dependencies.Verify = %q, want ready with both roles approved and no handoff.md", status.Dependencies.Verify)
	}
	// The route is not asserted: with every task done it also depends on the
	// integration gate, which this change does not touch.
	if reasons := strings.Join(status.BlockedReasons, "\n"); strings.Contains(reasons, "handoff.md") || strings.Contains(reasons, "verify") {
		t.Fatalf("BlockedReasons = %v, want no mention of handoff.md", status.BlockedReasons)
	}
}

// A handoff.md that older versions wrote is ignored in either status.
func TestVerifyIgnoresALegacyHandoffFile(t *testing.T) {
	for _, status := range []handoff.Status{handoff.StatusBlocked, handoff.StatusNeedsClarification} {
		resolved := resolveTwoRoleChange(t, "two-roles-legacy", func(changeRoot string) {
			approveGates(t, changeRoot, kickoff.RoleApplyGate("core"), kickoff.RoleApplyGate("web"))
			writeLegacyIntegrationHandoff(t, changeRoot, "two-roles-legacy", status)
		})
		if resolved.Dependencies.Verify != DependencyReady {
			t.Fatalf("handoff.md %q: Dependencies.Verify = %q, want ready", status, resolved.Dependencies.Verify)
		}
		if reasons := strings.Join(resolved.BlockedReasons, "\n"); strings.Contains(reasons, "relevo") {
			t.Fatalf("handoff.md %q: BlockedReasons = %v, want no relay reason", status, resolved.BlockedReasons)
		}
	}
	open := resolveTwoRoleChange(t, "two-roles-legacy-open", func(changeRoot string) {
		approveGates(t, changeRoot, kickoff.RoleApplyGate("core"))
		writeLegacyIntegrationHandoff(t, changeRoot, "two-roles-legacy-open", handoff.StatusReady)
	})
	if open.Dependencies.Verify != DependencyBlocked {
		t.Fatalf("a ready handoff.md must not unblock verify while web is open: Dependencies.Verify = %q", open.Dependencies.Verify)
	}
}

func TestVerifyStaysBlockedWhenARoleApplyGateIsRejected(t *testing.T) {
	status := resolveTwoRoleChange(t, "two-roles-rejected", func(changeRoot string) {
		approveGates(t, changeRoot, kickoff.RoleApplyGate("core"))
		rejectGate(t, changeRoot, kickoff.RoleApplyGate("web"))
	})
	if status.Dependencies.Verify != DependencyBlocked {
		t.Fatalf("Dependencies.Verify = %q, want blocked with web rejected", status.Dependencies.Verify)
	}
}

// A single role or a continuous seal keeps the baseline untouched, whatever
// handoff_policy says.
func TestVerifyKeepsBaselineForSingleRoleAndContinuous(t *testing.T) {
	root := t.TempDir()
	changeRoot := seedReadyChange(t, root, "solo-per-checkpoint", "- [x] 1.1 Wire routes\n")
	sealPerCheckpointFullstackKickoff(t, changeRoot, "solo-per-checkpoint", kickoff.ExecutionCheckpointed)
	status, err := Resolve(ResolveOptions{CWD: root, ChangeName: "solo-per-checkpoint"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if status.Dependencies.Verify != DependencyReady {
		t.Fatalf("single role: Dependencies.Verify = %q, want ready", status.Dependencies.Verify)
	}

	continuousRoot := t.TempDir()
	seedMultiRoleChange(t, continuousRoot, "continuous", kickoff.ExecutionContinuous, map[string]string{
		"core": "- [x] 1.1 Model\n", "web": "- [x] 1.1 Screen\n",
	}, "core", "web")
	continuous, err := Resolve(ResolveOptions{CWD: continuousRoot, ChangeName: "continuous"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if continuous.Dependencies.Verify == DependencyBlocked {
		t.Fatalf("continuous: Dependencies.Verify = %q, want not blocked", continuous.Dependencies.Verify)
	}
}

// REQ-26.9: per_checkpoint and none produce the same result.
func TestVerifyDependencyIgnoresHandoffPolicy(t *testing.T) {
	build := func(policy kickoff.HandoffPolicy, webApproved bool) *Governance {
		webStatus := "pending"
		if webApproved {
			webStatus = "approved"
		}
		return &Governance{
			Kickoff: kickoff.Kickoff{Config: kickoff.FlowConfig{ExecutionStyle: kickoff.ExecutionCheckpointed, HandoffPolicy: policy}},
			Gates: []kickoff.GateState{
				{Key: kickoff.RoleApplyGate("core"), Status: "approved", Blocks: "core"},
				{Key: kickoff.RoleApplyGate("web"), Status: webStatus, Blocks: "web"},
			},
			Roster: GovernanceRoster{Source: governanceRosterSourceKickoff, Roles: []string{"core", "web"}},
		}
	}
	for _, webApproved := range []bool{false, true} {
		noneState, noneReason := verifyDependencyFromRoleGates(build(kickoff.HandoffNone, webApproved), DependencyReady)
		perState, perReason := verifyDependencyFromRoleGates(build(kickoff.HandoffPerCheckpoint, webApproved), DependencyReady)
		emptyState, emptyReason := verifyDependencyFromRoleGates(build("", webApproved), DependencyReady)
		if noneState != perState || noneReason != perReason || noneState != emptyState || noneReason != emptyReason {
			t.Fatalf("webApproved=%v: none=(%s,%q) per_checkpoint=(%s,%q) empty=(%s,%q), want identical", webApproved, noneState, noneReason, perState, perReason, emptyState, emptyReason)
		}
	}
	// A baseline that is already blocked or not ready is never rewritten.
	if state, reason := verifyDependencyFromRoleGates(build(kickoff.HandoffNone, false), DependencyBlocked); state != DependencyBlocked || reason != "" {
		t.Fatalf("blocked baseline: got (%s, %q), want it untouched", state, reason)
	}
	if state, reason := verifyDependencyFromRoleGates(nil, DependencyReady); state != DependencyReady || reason != "" {
		t.Fatalf("nil governance: got (%s, %q), want it untouched", state, reason)
	}
}
