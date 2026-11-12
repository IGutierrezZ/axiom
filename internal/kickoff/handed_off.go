package kickoff

import (
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
)

// Approval returns the FIRST approved record of gate, the one that closes it
// (resolveGateStatus applies the same rule): a later record never replaces it,
// so a HandedOff mark written after it would be invisible. The boolean reports
// whether the gate has been approved at all.
func (l GateLedger) Approval(gate GateKey) (GateRecord, bool) {
	for _, r := range l.Records {
		if r.Gate == gate && r.Decision == DecisionApproved {
			return r, true
		}
	}
	return GateRecord{}, false
}

// HandedOffTo names the phase, or role phase, that exit 2 of a stop hands to
// somebody else once gate is approved (REQ-26.7):
//
//   - a fixed gate hands over what it blocks: design, tasks or apply;
//   - a role-apply:<role> gate hands over verify when it closes the last blocking role
//     of the roster, and otherwise "apply:<r1>,<r2>" with the roles that still
//     have no approved role-apply gate, in alphabetical order.
//
// gate is treated as approved even when states still shows it pending: the
// function answers "what does approving THIS gate hand over", so the recorder
// can ask before the record exists and `status` can ask afterwards, with the
// same result.
func HandedOffTo(gate GateState, roster []multirole.RoleAssignment, states []GateState) string {
	if !strings.HasPrefix(string(gate.Key), roleApplyGatePrefix) {
		if gate.Blocks != "" {
			return gate.Blocks
		}
		// A caller that only knows the gate key (the recorder, before the gate has a state)
		// gets the same wording EvaluateGates would put in Blocks.
		for _, fixed := range fixedGateOrder {
			if fixed.Key == gate.Key {
				return fixed.Blocks
			}
		}
		return ""
	}

	withGate := make([]GateState, 0, len(states)+1)
	found := false
	for _, s := range states {
		if s.Key == gate.Key {
			s.Status = "approved"
			found = true
		}
		withGate = append(withGate, s)
	}
	if !found {
		approved := gate
		approved.Status = "approved"
		withGate = append(withGate, approved)
	}

	if LastBlockingRoleClosed(roster, withGate) {
		return "verify"
	}

	approvedKeys := make(map[GateKey]bool, len(withGate))
	for _, s := range withGate {
		if s.Status == "approved" {
			approvedKeys[s.Key] = true
		}
	}
	var pending []string
	for _, role := range roster {
		if !approvedKeys[RoleApplyGate(role.Role)] {
			pending = append(pending, role.Role)
		}
	}
	if len(pending) == 0 {
		// Roster vacio: ningun rol cierra, pero tampoco hay un rol que nombrar.
		return "apply"
	}
	sort.Strings(pending)
	return "apply:" + strings.Join(pending, ",")
}

// IntermediateMerge reports whether a change whose planning files are already
// in the integration branch is NOT finished (REQ-26.11, DD-09): some approval
// was recorded with the hand-off mark, so a merge of that pull request only
// publishes an intermediate stop, and not every role of the roster has its
// role-apply gate approved yet.
//
// It only reads what the hand-off itself leaves integrated (gates.yaml and
// kickoff.yaml), and only the roles with policy blocking count (REQ-21.15). A roster
// with no blocking role satisfies the second condition, in the
// same spirit as LastBlockingRoleClosed, which never closes over an empty set.
func IntermediateMerge(k Kickoff, l GateLedger) bool {
	handedOff := false
	for _, r := range l.Records {
		if r.Decision == DecisionApproved && r.HandedOff {
			handedOff = true
			break
		}
	}
	if !handedOff {
		return false
	}
	// Solo los roles blocking deciden (REQ-21.15, H4): un rol deferred u optional
	// que nunca se aprueba no puede hacer que el archivado se rechace como una
	// fusión intermedia.
	blocking := 0
	for _, role := range k.Config.Roles {
		if !IsBlockingRole(multirole.RoleAssignment{Role: role.Role, GatePolicy: role.GatePolicy}) {
			continue
		}
		blocking++
		if _, ok := l.Approval(RoleApplyGate(role.Role)); !ok {
			return true
		}
	}
	return blocking == 0
}
