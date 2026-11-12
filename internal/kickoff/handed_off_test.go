package kickoff

import (
	"errors"
	"sync"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
)

func TestLedgerApprovalReturnsTheFirstApprovedRecord(t *testing.T) {
	ledger := GateLedger{Records: []GateRecord{
		{Gate: GateSpec, Decision: DecisionRejected, Reason: "falta"},
		{Gate: GateSpec, Decision: DecisionApproved, Reason: "primera", HandedOff: true},
		{Gate: GateSpec, Decision: DecisionApproved, Reason: "segunda"},
		{Gate: GateDesign, Decision: DecisionApproved, Reason: "otra puerta"},
	}}

	rec, ok := ledger.Approval(GateSpec)
	if !ok || rec.Reason != "primera" || !rec.HandedOff {
		t.Fatalf("Approval(spec) = %+v, %v; se esperaba el primer approved con su marca", rec, ok)
	}
	if _, ok := ledger.Approval(GateTasks); ok {
		t.Fatal("Approval(tasks) = true sin ningun registro")
	}
	if _, ok := (GateLedger{Records: []GateRecord{{Gate: GateSpec, Decision: DecisionRejected}}}).Approval(GateSpec); ok {
		t.Fatal("un rejected no es una aprobacion")
	}
}

func TestHandedOffToFixedGatesNamesWhatTheyBlock(t *testing.T) {
	for _, tc := range []struct {
		gate GateState
		want string
	}{
		{GateState{Key: GateSpec, Blocks: "design", Status: "approved"}, "design"},
		{GateState{Key: GateDesign, Blocks: "tasks", Status: "approved"}, "tasks"},
		{GateState{Key: GateTasks, Blocks: "apply", Status: "approved"}, "apply"},
	} {
		if got := HandedOffTo(tc.gate, nil, nil); got != tc.want {
			t.Errorf("HandedOffTo(%s) = %q, se esperaba %q", tc.gate.Key, got, tc.want)
		}
	}
}

func TestHandedOffToFixedGateWithoutStateResolvesTheBlocksWording(t *testing.T) {
	for key, want := range map[GateKey]string{GateSpec: "design", GateDesign: "tasks", GateTasks: "apply"} {
		if got := HandedOffTo(GateState{Key: key}, nil, nil); got != want {
			t.Errorf("HandedOffTo(%s) = %q, se esperaba %q", key, got, want)
		}
	}
}

func TestHandedOffToRoleApply(t *testing.T) {
	roster := []multirole.RoleAssignment{
		{Role: "web", GatePolicy: multirole.PolicyBlocking},
		{Role: "core", GatePolicy: multirole.PolicyBlocking},
		{Role: "qa", GatePolicy: multirole.PolicyDeferred},
	}
	gate := func(role, status string) GateState {
		return GateState{Key: RoleApplyGate(role), Blocks: role, Status: status}
	}

	t.Run("el ultimo rol cedido entrega verify", func(t *testing.T) {
		states := []GateState{gate("core", "approved"), gate("qa", "approved"), gate("web", "pending")}
		if got := HandedOffTo(gate("web", "pending"), roster, states); got != "verify" {
			t.Errorf("HandedOffTo() = %q, se esperaba verify", got)
		}
	})

	t.Run("con un rol pendiente entrega apply de ese rol", func(t *testing.T) {
		states := []GateState{gate("core", "approved"), gate("qa", "approved"), gate("web", "pending")}
		if got := HandedOffTo(gate("core", "approved"), roster, states); got != "apply:web" {
			t.Errorf("HandedOffTo() = %q, se esperaba apply:web", got)
		}
	})

	t.Run("con varios roles pendientes los ordena alfabeticamente", func(t *testing.T) {
		states := []GateState{gate("core", "pending"), gate("qa", "pending"), gate("web", "pending")}
		if got := HandedOffTo(gate("core", "pending"), roster, states); got != "apply:qa,web" {
			t.Errorf("HandedOffTo() = %q, se esperaba apply:qa,web", got)
		}
	})

	t.Run("un rol blocking rechazado cuenta como pendiente", func(t *testing.T) {
		// Con qa blocking, un rechazo lo deja pendiente. (REQ-21.15: con qa deferred la
		// verificación global ya está abierta, véase TestHandedOffToIgnoresNonBlockingRoles.)
		allBlocking := []multirole.RoleAssignment{
			{Role: "web", GatePolicy: multirole.PolicyBlocking},
			{Role: "core", GatePolicy: multirole.PolicyBlocking},
			{Role: "qa", GatePolicy: multirole.PolicyBlocking},
		}
		states := []GateState{gate("core", "approved"), gate("qa", "rejected"), gate("web", "approved")}
		if got := HandedOffTo(gate("core", "approved"), allBlocking, states); got != "apply:qa" {
			t.Errorf("HandedOffTo() = %q, se esperaba apply:qa", got)
		}
	})

	t.Run("un rol unico cierra el ultimo rol", func(t *testing.T) {
		single := []multirole.RoleAssignment{{Role: "fullstack", GatePolicy: multirole.PolicyBlocking}}
		if got := HandedOffTo(gate("fullstack", "pending"), single, nil); got != "verify" {
			t.Errorf("HandedOffTo() = %q, se esperaba verify", got)
		}
	})
}

func TestIntermediateMerge(t *testing.T) {
	kick := Kickoff{Config: FlowConfig{Roles: []KickoffRole{{Role: "core"}, {Role: "web"}}}}
	approve := func(gate GateKey, handedOff bool) GateRecord {
		return GateRecord{Gate: gate, Decision: DecisionApproved, HandedOff: handedOff}
	}

	cases := []struct {
		name    string
		kickoff Kickoff
		records []GateRecord
		want    bool
	}{
		{"solo spec cedida", kick, []GateRecord{approve(GateSpec, true)}, true},
		{"design cedida con roles sin aprobar", kick, []GateRecord{approve(GateSpec, false), approve(GateDesign, true)}, true},
		{
			"todos los role-apply aprobados con design cedida (A-7)",
			kick,
			[]GateRecord{approve(GateDesign, true), approve(RoleApplyGate("core"), false), approve(RoleApplyGate("web"), false)},
			false,
		},
		{
			"role-apply cedido con otro rol pendiente",
			kick,
			[]GateRecord{approve(RoleApplyGate("core"), true)},
			true,
		},
		{
			"ultimo role-apply cedido ya no es intermedia",
			kick,
			[]GateRecord{approve(RoleApplyGate("core"), false), approve(RoleApplyGate("web"), true)},
			false,
		},
		{"sin ninguna marca (A-8)", kick, []GateRecord{approve(GateSpec, false), approve(GateDesign, false)}, false},
		{"sin registros", kick, nil, false},
		{"un rejected con marca no cuenta", kick, []GateRecord{{Gate: GateSpec, Decision: DecisionRejected, HandedOff: true}}, false},
		{"roster vacio con marca", Kickoff{}, []GateRecord{approve(GateSpec, true)}, true},
		{"roster vacio sin marca", Kickoff{}, []GateRecord{approve(GateSpec, false)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IntermediateMerge(tc.kickoff, GateLedger{Schema: GateLedgerSchemaV1, Records: tc.records})
			if got != tc.want {
				t.Errorf("IntermediateMerge() = %v, se esperaba %v", got, tc.want)
			}
		})
	}
}

var errGateAlreadyApproved = errors.New("la puerta ya esta resuelta")

func notApprovedCheck(gate GateKey) func(GateLedger) error {
	return func(l GateLedger) error {
		if _, ok := l.Approval(gate); ok {
			return errGateAlreadyApproved
		}
		return nil
	}
}

// TestAppendGateCheckedRunsTheCheckUnderTheLock simula la salida 2 y la salida
// 3 de la misma puerta pendiente a la vez: la comprobacion corre bajo el
// cerrojo, asi que solo uno anexa su approved y el otro recibe el error del
// check sin escribir nada (REQ-26.3, A-1).
func TestAppendGateCheckedRunsTheCheckUnderTheLock(t *testing.T) {
	changeRoot := t.TempDir()
	records := []GateRecord{
		{Gate: GateSpec, Decision: DecisionApproved, Reason: "salida 2", HandedOff: true, Actor: "maintainer"},
		{Gate: GateSpec, Decision: DecisionApproved, Reason: "salida 3", Actor: "maintainer"},
	}

	var wg sync.WaitGroup
	errs := make([]error, len(records))
	for i := range records {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = AppendGateChecked(changeRoot, records[i], notApprovedCheck(GateSpec))
		}(i)
	}
	wg.Wait()

	failed := 0
	for _, err := range errs {
		if err == nil {
			continue
		}
		failed++
		if !errors.Is(err, errGateAlreadyApproved) {
			t.Fatalf("error = %v, se esperaba el del check", err)
		}
	}
	if failed != 1 {
		t.Fatalf("errores = %v, se esperaba exactamente un rechazo", errs)
	}
	ledger, err := LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("LoadGates() error = %v", err)
	}
	if len(ledger.Records) != 1 {
		t.Fatalf("Records = %+v, se esperaba un unico approved", ledger.Records)
	}
}

func TestAppendGateCheckedRejectionWritesNothing(t *testing.T) {
	changeRoot := t.TempDir()
	if err := AppendGate(changeRoot, GateRecord{Gate: GateSpec, Decision: DecisionApproved, Actor: "maintainer"}); err != nil {
		t.Fatalf("AppendGate() error = %v", err)
	}
	err := AppendGateChecked(changeRoot, GateRecord{Gate: GateSpec, Decision: DecisionApproved, HandedOff: true, Actor: "maintainer"}, notApprovedCheck(GateSpec))
	if !errors.Is(err, errGateAlreadyApproved) {
		t.Fatalf("error = %v, se esperaba el del check", err)
	}
	ledger, loadErr := LoadGates(changeRoot)
	if loadErr != nil {
		t.Fatalf("LoadGates() error = %v", loadErr)
	}
	if len(ledger.Records) != 1 || ledger.Records[0].HandedOff {
		t.Fatalf("Records = %+v, la aprobacion rechazada no debia escribirse", ledger.Records)
	}
}

func TestAppendGateWithoutCheckKeepsItsBehaviour(t *testing.T) {
	changeRoot := t.TempDir()
	for i := 0; i < 2; i++ {
		if err := AppendGateChecked(changeRoot, GateRecord{Gate: GateSpec, Decision: DecisionApproved, Actor: "maintainer"}, nil); err != nil {
			t.Fatalf("AppendGateChecked(nil) error = %v", err)
		}
	}
	ledger, err := LoadGates(changeRoot)
	if err != nil {
		t.Fatalf("LoadGates() error = %v", err)
	}
	if len(ledger.Records) != 2 {
		t.Fatalf("Records = %+v, sin check se anexan ambos como antes", ledger.Records)
	}
}
