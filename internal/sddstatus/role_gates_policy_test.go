package sddstatus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
)

// REQ-21.15: solo los roles con política blocking bloquean la fase verify global;
// los deferred y optional se informan en notes con su política y el estado de su
// compuerta, sin bloquear. Los nombres empiezan por TestRoleGatesDependency y
// TestVerifyDependencyFromRoleGates para entrar en el mismo patrón de `-run`.

// resolveMixedPolicyChange monta un cambio con checkpointed, las tareas de todos
// los roles hechas y las compuertas de planificación aprobadas, sellado con la
// política de cada rol ("rol:política"), y lo resuelve.
func resolveMixedPolicyChange(t *testing.T, name string, mutate func(changeRoot string), roles ...string) Status {
	t.Helper()
	root := t.TempDir()
	names := make([]string, 0, len(roles))
	tasks := map[string]string{}
	for _, entry := range roles {
		role, _, _ := strings.Cut(entry, ":")
		names = append(names, role)
		tasks[role] = "- [x] 1.1 Trabajo\n"
	}
	changeRoot := seedMultiRoleChange(t, root, name, kickoff.ExecutionCheckpointed, tasks, names...)

	// Vuelve a sellar con las políticas pedidas.
	if err := os.Remove(filepath.Join(changeRoot, kickoff.KickoffFileName)); err != nil {
		t.Fatal(err)
	}
	sealed := make([]kickoff.KickoffRole, 0, len(roles))
	for _, entry := range roles {
		role, policy, _ := strings.Cut(entry, ":")
		sealed = append(sealed, kickoff.KickoffRole{
			Role: role, GatePolicy: multirole.GatePolicy(policy),
			TasksFile: "tasks." + role + ".md", VerifyFile: "verify-report." + role + ".md",
		})
	}
	if _, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: name, SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionCheckpointed, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone, Roles: sealed,
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	}); err != nil || !ok {
		t.Fatalf("kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}
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

func TestRoleGatesDependencyBlockingApprovedAndDeferredPending(t *testing.T) {
	status := resolveMixedPolicyChange(t, "mixed-open", func(changeRoot string) {
		approveGates(t, changeRoot, kickoff.RoleApplyGate("core"))
	}, "core:blocking", "e2e:deferred")

	if status.Dependencies.Verify != DependencyReady {
		t.Fatalf("Dependencies.Verify = %q, se esperaba ready: el rol deferred no bloquea (REQ-21.15)", status.Dependencies.Verify)
	}
	if reasons := strings.Join(status.BlockedReasons, "\n"); strings.Contains(reasons, "REQ-21.15") {
		t.Errorf("BlockedReasons = %v, no debe haber motivo de verify", status.BlockedReasons)
	}
	notes := strings.Join(status.Notes, "\n")
	if !strings.Contains(notes, `"e2e"`) || !strings.Contains(notes, "deferred") || !strings.Contains(notes, "pendiente") || strings.Contains(notes, `"core"`) {
		t.Errorf("Notes = %v, se esperaba una nota con el rol e2e, su política deferred y su estado pendiente", status.Notes)
	}
}

func TestRoleGatesDependencyBlockingPendingStaysBlocked(t *testing.T) {
	status := resolveMixedPolicyChange(t, "mixed-blocked", func(changeRoot string) {
		approveGates(t, changeRoot, kickoff.RoleApplyGate("e2e"))
	}, "core:blocking", "e2e:deferred")

	if status.Dependencies.Verify != DependencyBlocked {
		t.Fatalf("Dependencies.Verify = %q, se esperaba blocked: core (blocking) no está aprobado", status.Dependencies.Verify)
	}
	reasons := strings.Join(status.BlockedReasons, "\n")
	if !strings.Contains(reasons, "core") || strings.Contains(reasons, "e2e") {
		t.Errorf("BlockedReasons = %v, el motivo nombra solo el rol blocking pendiente", status.BlockedReasons)
	}
}

func TestRoleGatesDependencyRejectedNonBlockingIsReportedNotBlocking(t *testing.T) {
	status := resolveMixedPolicyChange(t, "mixed-rejected", func(changeRoot string) {
		approveGates(t, changeRoot, kickoff.RoleApplyGate("core"))
		// Un rechazo solo se mantiene mientras la huella siga siendo la del
		// artefacto que juzgó: se registra con la de tasks.docs.md.
		digest, err := kickoff.ArtifactDigest([]string{filepath.Join(changeRoot, "tasks.docs.md")})
		if err != nil {
			t.Fatal(err)
		}
		if err := kickoff.AppendGate(changeRoot, kickoff.GateRecord{
			Gate: kickoff.RoleApplyGate("docs"), Decision: kickoff.DecisionRejected, Reason: "falta cobertura",
			ArtifactDigest: digest, Actor: "reviewer",
		}); err != nil {
			t.Fatal(err)
		}
	}, "core:blocking", "docs:optional")

	if status.Dependencies.Verify != DependencyReady {
		t.Fatalf("Dependencies.Verify = %q, un optional rechazado no bloquea", status.Dependencies.Verify)
	}
	notes := strings.Join(status.Notes, "\n")
	if !strings.Contains(notes, `"docs"`) || !strings.Contains(notes, "optional") || !strings.Contains(notes, "rechazado") {
		t.Errorf("Notes = %v, se esperaba el rol docs optional rechazado", status.Notes)
	}
}

// Con todos los roles blocking el resultado es el de siempre y no hay notas nuevas.
func TestRoleGatesDependencyAllBlockingProjectsNoNonBlockingNotes(t *testing.T) {
	status := resolveMixedPolicyChange(t, "all-blocking", func(changeRoot string) {
		approveGates(t, changeRoot, kickoff.RoleApplyGate("core"))
	}, "core:blocking", "web:blocking")
	if status.Dependencies.Verify != DependencyBlocked {
		t.Fatalf("Dependencies.Verify = %q, se esperaba blocked con web pendiente", status.Dependencies.Verify)
	}
	for _, note := range status.Notes {
		if strings.Contains(note, "no bloquea la fase verify global") {
			t.Errorf("Notes = %v, con todos los roles blocking no hay notas de roles no bloqueantes", status.Notes)
		}
	}
}

// verifyDependencyFromRoleGates directamente: roster con políticas.
func TestVerifyDependencyFromRoleGatesByPolicy(t *testing.T) {
	build := func(coreStatus, e2eStatus string) *Governance {
		return &Governance{
			Kickoff: kickoff.Kickoff{Config: kickoff.FlowConfig{
				ExecutionStyle: kickoff.ExecutionCheckpointed,
				Roles: []kickoff.KickoffRole{
					{Role: "core", GatePolicy: multirole.GatePolicy("blocking")},
					{Role: "e2e", GatePolicy: multirole.GatePolicy("deferred")},
				},
			}},
			Gates: []kickoff.GateState{
				{Key: kickoff.RoleApplyGate("core"), Status: coreStatus, Blocks: "core"},
				{Key: kickoff.RoleApplyGate("e2e"), Status: e2eStatus, Blocks: "e2e"},
			},
			Roster: GovernanceRoster{Source: governanceRosterSourceKickoff, Roles: []string{"core", "e2e"}},
		}
	}
	tests := []struct {
		name        string
		core, e2e   string
		wantState   DependencyState
		wantReason  bool
		wantNoteFor string
	}{
		{"blocking aprobado, deferred pendiente", "approved", "pending", DependencyReady, false, "e2e"},
		{"blocking aprobado, deferred rechazado", "approved", "rejected", DependencyReady, false, "e2e"},
		{"blocking aprobado, deferred aprobado", "approved", "approved", DependencyReady, false, ""},
		{"blocking pendiente, deferred aprobado", "pending", "approved", DependencyBlocked, true, ""},
		{"blocking rechazado, deferred pendiente", "rejected", "pending", DependencyBlocked, true, "e2e"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			governance := build(tt.core, tt.e2e)
			state, reason := verifyDependencyFromRoleGates(governance, DependencyReady)
			if state != tt.wantState || (reason != "") != tt.wantReason {
				t.Fatalf("verifyDependencyFromRoleGates() = (%q, %q), se esperaba (%q, motivo=%v)", state, reason, tt.wantState, tt.wantReason)
			}
			if tt.wantReason && (!strings.Contains(reason, "core") || strings.Contains(reason, "e2e")) {
				t.Errorf("el motivo %q debe nombrar solo el rol blocking pendiente", reason)
			}
			notes := strings.Join(nonBlockingRoleNotes(governance), "\n")
			if tt.wantNoteFor == "" && notes != "" {
				t.Errorf("notas = %q, no se esperaba ninguna", notes)
			}
			if tt.wantNoteFor != "" && !strings.Contains(notes, `"`+tt.wantNoteFor+`"`) {
				t.Errorf("notas = %q, se esperaba nombrar a %s", notes, tt.wantNoteFor)
			}
		})
	}

	t.Run("sin ningún rol blocking no hay nada que esperar", func(t *testing.T) {
		governance := &Governance{
			Kickoff: kickoff.Kickoff{Config: kickoff.FlowConfig{
				ExecutionStyle: kickoff.ExecutionCheckpointed,
				Roles: []kickoff.KickoffRole{
					{Role: "e2e", GatePolicy: multirole.GatePolicy("deferred")},
					{Role: "docs", GatePolicy: multirole.GatePolicy("optional")},
				},
			}},
			Roster: GovernanceRoster{Source: governanceRosterSourceKickoff, Roles: []string{"e2e", "docs"}},
		}
		if state, reason := verifyDependencyFromRoleGates(governance, DependencyReady); state != DependencyReady || reason != "" {
			t.Fatalf("verifyDependencyFromRoleGates() = (%q, %q), se esperaba ready sin motivo", state, reason)
		}
	})
}
