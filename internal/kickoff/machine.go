package kickoff

import (
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
)

// Inputs is the complete input to the gate state machine. EvaluateGates is
// pure over Inputs: it never reads a file, never consults the clock, and
// never writes anything (D-08). Artifacts carries the digest already
// computed by the caller (ArtifactDigest) for each governed artifact,
// keyed "spec" | "design" | "tasks" | "tasks.<role>" (role lowercased); an
// absent or empty digest means that artifact has not reached "done" yet.
// RolePending carries each role's current pending task count from
// multirole.CountTasks; a role absent from this map is treated as unknown,
// never as "0 pending".
type Inputs struct {
	Execution   ExecutionStyle
	Roles       []multirole.RoleAssignment
	Artifacts   map[string]string
	RolePending map[string]int
	VerifyFound bool
	Ledger      GateLedger
}

// fixedGateOrder is the immutable evaluation order for the four
// artifact-driven gates that do not depend on the roster (D-08, D-09).
// EvaluateGates inserts one role-apply:<role> gate per roster role, sorted
// by role name, between the tasks and integration entries below — see
// evaluateRoleApplyGates. Extracted here so Phase 6 (reopening rules) and
// Phase 7 (LastBlockingRoleClosed) reuse the exact same order and Blocks wording
// instead of a second, driftable copy.
var fixedGateOrder = []Gate{
	{Key: GateSpec, Blocks: "design"},
	{Key: GateDesign, Blocks: "tasks"},
	{Key: GateTasks, Blocks: "apply"},
	{Key: GateIntegration, Blocks: "archive"},
}

// EvaluateGates projects the current state of every governed gate for one
// change, in the fixed order spec -> design -> tasks ->
// role-apply:<role...> -> integration, so a caller that reports only the
// first blocking entry always reports the same one regardless of map or
// slice iteration order.
//
// In continuous mode it returns (nil, nil) immediately: D-05 requires this
// to be a genuine zero-cost path, so it must not read Roles, Artifacts,
// RolePending, VerifyFound, or Ledger to reach that answer.
func EvaluateGates(in Inputs) ([]GateState, error) {
	if in.Execution == ExecutionContinuous {
		return nil, nil
	}

	recordsByGate := groupGateRecordsByKey(in.Ledger.Records)

	var states []GateState
	appendIfOpen := func(gate Gate, open bool, digest string) {
		if !open {
			return
		}
		status, reason, reopened, handedOff, handedOffTo := resolveGateStatus(digest, recordsByGate[gate.Key])
		states = append(states, GateState{
			Key:         gate.Key,
			Status:      status,
			Reason:      reason,
			Blocks:      gate.Blocks,
			Reopened:    reopened,
			HandedOff:   handedOff,
			HandedOffTo: handedOffTo,
		})
	}

	appendIfOpen(fixedGateOrder[0], in.Artifacts["spec"] != "", in.Artifacts["spec"])
	appendIfOpen(fixedGateOrder[1], in.Artifacts["design"] != "", in.Artifacts["design"])
	appendIfOpen(fixedGateOrder[2], tasksGateOpen(in), in.Artifacts["tasks"])

	for _, role := range sortedRoleNames(in.Roles) {
		gate := Gate{Key: RoleApplyGate(role), Blocks: role}
		appendIfOpen(gate, roleApplyGateOpen(role, in), roleApplyArtifactDigest(role, in))
	}

	appendIfOpen(fixedGateOrder[3], in.VerifyFound, "")

	return states, nil
}

// tasksGateOpen implements the "tasks" row of D-08's table: the shared
// tasks digest must be present, and, whenever the roster has more than one
// role, every role's own "tasks.<role>" digest must also be present. A
// single-role roster (including the fullstack default) never needs the
// per-role key: its one tasks file IS the shared "tasks" artifact.
func tasksGateOpen(in Inputs) bool {
	if in.Artifacts["tasks"] == "" {
		return false
	}
	if len(in.Roles) <= 1 {
		return true
	}
	for _, role := range in.Roles {
		if in.Artifacts[roleTasksArtifactKey(role.Role)] == "" {
			return false
		}
	}
	return true
}

// roleApplyGateOpen implements the "role-apply:<role>" row of D-08's
// table: the role must have an explicit, recorded pending-task count of
// exactly zero. A role missing from RolePending is treated as unknown, not
// as zero — CountTasks always returns a value for a role it actually
// inspected, so a missing entry means the caller never inspected this role
// at all.
func roleApplyGateOpen(role string, in Inputs) bool {
	pending, tracked := in.RolePending[role]
	return tracked && pending == 0
}

// roleApplyArtifactDigest resolves the digest role-apply:<role> is judged
// against: the role's own "tasks.<role>" artifact, mirroring the key
// convention documented on Inputs.Artifacts.
func roleApplyArtifactDigest(role string, in Inputs) string {
	return in.Artifacts[roleTasksArtifactKey(role)]
}

func roleTasksArtifactKey(role string) string {
	return "tasks." + strings.ToLower(role)
}

// sortedRoleNames returns the roster's role identifiers sorted
// alphabetically, so the role-apply gates EvaluateGates emits never depend
// on the order Roles happened to be declared in.
func sortedRoleNames(roles []multirole.RoleAssignment) []string {
	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, r.Role)
	}
	sort.Strings(names)
	return names
}

// groupGateRecordsByKey buckets an append-only ledger's records by gate
// key, preserving their original (chronological) order within each
// bucket.
func groupGateRecordsByKey(records []GateRecord) map[GateKey][]GateRecord {
	grouped := make(map[GateKey][]GateRecord, len(records))
	for _, r := range records {
		grouped[r.Gate] = append(grouped[r.Gate], r)
	}
	return grouped
}

// IsBlockingRole reports whether role holds up the global verify phase: its
// gate policy is "blocking". An empty policy is the default, blocking, exactly
// as multirole.NormalizeRoster resolves it, so a roster built without policies
// (the roster of a Governance, older callers) keeps its previous meaning.
func IsBlockingRole(role multirole.RoleAssignment) bool {
	policy := multirole.GatePolicy(strings.ToLower(strings.TrimSpace(string(role.GatePolicy))))
	return policy == "" || policy == multirole.PolicyBlocking
}

// RoleGateStatus is the status of one role's role-apply gate in gates:
// "approved", "rejected" or "pending". A role whose gate is absent from gates
// (its tasks are still open, or it was never evaluated) is "pending".
func RoleGateStatus(role multirole.RoleAssignment, gates []GateState) string {
	key := RoleApplyGate(role.Role)
	for _, g := range gates {
		if g.Key == key && (g.Status == "approved" || g.Status == "rejected") {
			return g.Status
		}
	}
	return "pending"
}

// LastBlockingRoleClosed reports whether every role of roster with policy
// "blocking" has its role-apply gate approved in gates (REQ-21.13, REQ-21.15):
// the condition that opens the global verify phase. Roles with policy deferred
// or optional never take part, whatever the status of their gates. With every
// role blocking, or with the single fullstack role, it requires every role-apply
// gate approved: a single pending or rejected gate keeps it false.
//
// It is never true over an empty set: a roster with no blocking role has no
// approval that could have produced this state, so a caller with a missing or
// not-yet-resolved roster can never fire the notice by accident.
func LastBlockingRoleClosed(roster []multirole.RoleAssignment, gates []GateState) bool {
	blocking := 0
	for _, role := range roster {
		if !IsBlockingRole(role) {
			continue
		}
		blocking++
		if RoleGateStatus(role, gates) != "approved" {
			return false
		}
	}
	return blocking > 0
}

// PendingRole is a non-blocking role whose role-apply gate is not approved yet:
// the role, its gate policy and the status of its gate ("pending" or
// "rejected").
type PendingRole struct {
	Role   string
	Policy multirole.GatePolicy
	Status string
}

// PendingNonBlockingRoles lists, in roster order, the roles with policy
// deferred or optional whose role-apply gate is not approved (REQ-21.13,
// REQ-21.15). They neither block verify nor repeat the last-role notice; they
// are only reported.
func PendingNonBlockingRoles(roster []multirole.RoleAssignment, gates []GateState) []PendingRole {
	var pending []PendingRole
	for _, role := range roster {
		if IsBlockingRole(role) {
			continue
		}
		if status := RoleGateStatus(role, gates); status != "approved" {
			pending = append(pending, PendingRole{Role: role.Role, Policy: multirole.GatePolicy(strings.ToLower(strings.TrimSpace(string(role.GatePolicy)))), Status: status})
		}
	}
	return pending
}

// resolveGateStatus derives one gate's current status from its recorded
// history, applying D-08's asymmetric reopening rule:
//
//   - An approval anywhere in the history is TERMINAL: it is returned
//     unconditionally, regardless of the current digest and regardless of
//     any record appended after it. A rejection describes one concrete
//     artifact and stops applying once that artifact changes; an approval
//     authorizes advancing, and advancing implies changing artifacts by
//     design (alternative 1 rejected in D-08 — invalidating approvals on
//     every digest change would deadlock sdd-apply against its own
//     tasks.md checkbox edits).
//   - Absent an approval, the latest rejection stands as long as digest
//     still matches the artifact it judged. Once the digest differs,
//     remediation happened: the gate returns to pending with
//     Reopened=true, and its prior reason is cleared (design.md S5.3:
//     Reason is empty in pending) — this is REQ-21.12 satisfied without a
//     manual "reopen" verb.
//   - No record at all yet: freshly pending.
//
// handedOff is the HandedOff mark of the FIRST approved record, the one that
// closes the gate (REQ-26.7): a later approved record neither adds nor removes
// it, and a later rejection leaves the gate approved with its mark intact.
// handedOffTo is the phase that same record persisted when it was written
// (empty in a ledger written before the field existed).
func resolveGateStatus(digest string, records []GateRecord) (status, reason string, reopened, handedOff bool, handedOffTo string) {
	for _, r := range records {
		if r.Decision == DecisionApproved {
			return "approved", r.Reason, false, r.HandedOff, r.HandedOffTo
		}
	}
	if len(records) == 0 {
		return "pending", "", false, false, ""
	}
	last := records[len(records)-1]
	if last.ArtifactDigest == digest {
		return "rejected", last.Reason, false, false, ""
	}
	return "pending", "", true, false, ""
}
