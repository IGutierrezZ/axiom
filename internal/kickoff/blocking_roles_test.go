package kickoff

import (
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
)

// REQ-21.13 y REQ-21.15: solo los roles con política blocking abren la fase verify
// global y cierran el último rol; los deferred y optional se informan sin bloquear.

func roleGate(role, status string) GateState {
	return GateState{Key: RoleApplyGate(role), Blocks: role, Status: status}
}

func TestLastBlockingRoleClosed(t *testing.T) {
	blocking := multirole.PolicyBlocking
	deferred := multirole.PolicyDeferred
	optional := multirole.PolicyOptional

	tests := []struct {
		name   string
		roster []multirole.RoleAssignment
		gates  []GateState
		want   bool
	}{
		{
			name:   "todos blocking aprobados",
			roster: []multirole.RoleAssignment{{Role: "core", GatePolicy: blocking}, {Role: "web", GatePolicy: blocking}},
			gates:  []GateState{roleGate("core", "approved"), roleGate("web", "approved")},
			want:   true,
		},
		{
			name:   "todos blocking con uno pendiente",
			roster: []multirole.RoleAssignment{{Role: "core", GatePolicy: blocking}, {Role: "web", GatePolicy: blocking}},
			gates:  []GateState{roleGate("core", "approved"), roleGate("web", "pending")},
			want:   false,
		},
		{
			name:   "un blocking aprobado y un deferred pendiente",
			roster: []multirole.RoleAssignment{{Role: "core", GatePolicy: blocking}, {Role: "e2e", GatePolicy: deferred}},
			gates:  []GateState{roleGate("core", "approved"), roleGate("e2e", "pending")},
			want:   true,
		},
		{
			name:   "un blocking aprobado y un optional sin compuerta",
			roster: []multirole.RoleAssignment{{Role: "core", GatePolicy: blocking}, {Role: "docs", GatePolicy: optional}},
			gates:  []GateState{roleGate("core", "approved")},
			want:   true,
		},
		{
			name:   "un blocking aprobado y un deferred rechazado",
			roster: []multirole.RoleAssignment{{Role: "core", GatePolicy: blocking}, {Role: "e2e", GatePolicy: deferred}},
			gates:  []GateState{roleGate("core", "approved"), roleGate("e2e", "rejected")},
			want:   true,
		},
		{
			name:   "mezcla con un blocking rechazado",
			roster: []multirole.RoleAssignment{{Role: "core", GatePolicy: blocking}, {Role: "web", GatePolicy: blocking}, {Role: "e2e", GatePolicy: deferred}},
			gates:  []GateState{roleGate("core", "approved"), roleGate("web", "rejected"), roleGate("e2e", "approved")},
			want:   false,
		},
		{
			name:   "sin bloqueantes nunca cierra",
			roster: []multirole.RoleAssignment{{Role: "e2e", GatePolicy: deferred}, {Role: "docs", GatePolicy: optional}},
			gates:  []GateState{roleGate("e2e", "approved"), roleGate("docs", "approved")},
			want:   false,
		},
		{
			name:   "roster vacío nunca cierra",
			roster: nil,
			want:   false,
		},
		{
			name:   "sin política es blocking por defecto",
			roster: []multirole.RoleAssignment{{Role: "core"}, {Role: "web"}},
			gates:  []GateState{roleGate("core", "approved")},
			want:   false,
		},
		{
			name:   "rol único fullstack aprobado",
			roster: []multirole.RoleAssignment{{Role: "fullstack", GatePolicy: blocking}},
			gates:  []GateState{roleGate("fullstack", "approved")},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LastBlockingRoleClosed(tt.roster, tt.gates); got != tt.want {
				t.Errorf("LastBlockingRoleClosed() = %v, se esperaba %v", got, tt.want)
			}
		})
	}
}

func TestPendingNonBlockingRoles(t *testing.T) {
	roster := []multirole.RoleAssignment{
		{Role: "core", GatePolicy: multirole.PolicyBlocking},
		{Role: "e2e", GatePolicy: multirole.PolicyDeferred},
		{Role: "docs", GatePolicy: multirole.PolicyOptional},
		{Role: "perf", GatePolicy: multirole.PolicyDeferred},
	}
	gates := []GateState{
		roleGate("core", "pending"),
		roleGate("e2e", "rejected"),
		roleGate("perf", "approved"),
	}
	want := []PendingRole{
		{Role: "e2e", Policy: multirole.PolicyDeferred, Status: "rejected"},
		{Role: "docs", Policy: multirole.PolicyOptional, Status: "pending"},
	}
	if got := PendingNonBlockingRoles(roster, gates); !reflect.DeepEqual(got, want) {
		t.Errorf("PendingNonBlockingRoles() = %+v, se esperaba %+v", got, want)
	}

	allBlocking := []multirole.RoleAssignment{{Role: "core", GatePolicy: multirole.PolicyBlocking}, {Role: "web"}}
	if got := PendingNonBlockingRoles(allBlocking, nil); len(got) != 0 {
		t.Errorf("con todos los roles blocking no hay roles no bloqueantes: %+v", got)
	}
}

// REQ-21.15 y H4: la salida 2 cede verify al cerrar el último rol blocking, aunque
// un deferred u optional siga pendiente.
func TestHandedOffToIgnoresNonBlockingRoles(t *testing.T) {
	roster := []multirole.RoleAssignment{
		{Role: "core", GatePolicy: multirole.PolicyBlocking},
		{Role: "web", GatePolicy: multirole.PolicyBlocking},
		{Role: "e2e", GatePolicy: multirole.PolicyDeferred},
	}
	t.Run("el último blocking cede verify con un deferred pendiente", func(t *testing.T) {
		states := []GateState{roleGate("core", "approved"), roleGate("web", "pending"), roleGate("e2e", "pending")}
		if got := HandedOffTo(roleGate("web", "pending"), roster, states); got != "verify" {
			t.Errorf("HandedOffTo() = %q, se esperaba verify", got)
		}
	})
	t.Run("con un blocking pendiente cede apply", func(t *testing.T) {
		states := []GateState{roleGate("core", "pending"), roleGate("web", "pending"), roleGate("e2e", "pending")}
		if got := HandedOffTo(roleGate("core", "pending"), roster, states); got != "apply:e2e,web" {
			t.Errorf("HandedOffTo() = %q, se esperaba apply:e2e,web", got)
		}
	})
	t.Run("un deferred rechazado no impide ceder verify", func(t *testing.T) {
		states := []GateState{roleGate("core", "approved"), roleGate("web", "approved"), roleGate("e2e", "rejected")}
		if got := HandedOffTo(roleGate("core", "approved"), roster, states); got != "verify" {
			t.Errorf("HandedOffTo() = %q, se esperaba verify", got)
		}
	})
}

// H4: un rol deferred u optional que nunca se aprueba no hace que el archivado se
// rechace como una fusión intermedia.
func TestIntermediateMergeCountsOnlyBlockingRoles(t *testing.T) {
	kick := Kickoff{Config: FlowConfig{Roles: []KickoffRole{
		{Role: "core", GatePolicy: multirole.PolicyBlocking},
		{Role: "web", GatePolicy: multirole.PolicyBlocking},
		{Role: "e2e", GatePolicy: multirole.PolicyDeferred},
		{Role: "docs", GatePolicy: multirole.PolicyOptional},
	}}}
	approve := func(gate GateKey, handedOff bool) GateRecord {
		return GateRecord{Gate: gate, Decision: DecisionApproved, HandedOff: handedOff}
	}
	tests := []struct {
		name    string
		kickoff Kickoff
		records []GateRecord
		want    bool
	}{
		{
			"blocking aprobados con un deferred y un optional pendientes ya no es intermedia",
			kick,
			[]GateRecord{approve(RoleApplyGate("core"), false), approve(RoleApplyGate("web"), true)},
			false,
		},
		{
			"un blocking pendiente sigue siendo intermedia",
			kick,
			[]GateRecord{approve(RoleApplyGate("core"), true)},
			true,
		},
		{
			"sin bloqueantes y con marca es intermedia",
			Kickoff{Config: FlowConfig{Roles: []KickoffRole{{Role: "e2e", GatePolicy: multirole.PolicyDeferred}}}},
			[]GateRecord{approve(GateSpec, true)},
			true,
		},
		{
			"rol único fullstack: el resultado es el de siempre",
			Kickoff{Config: FlowConfig{Roles: []KickoffRole{{Role: "fullstack", GatePolicy: multirole.PolicyBlocking}}}},
			[]GateRecord{approve(RoleApplyGate("fullstack"), true)},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IntermediateMerge(tt.kickoff, GateLedger{Schema: GateLedgerSchemaV1, Records: tt.records}); got != tt.want {
				t.Errorf("IntermediateMerge() = %v, se esperaba %v", got, tt.want)
			}
		})
	}
}
