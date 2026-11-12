package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
)

// TestRunSDDGateRecordRejectionPersistsAcrossStatusReads reproduces the
// user-visible defect confirmed live against the compiled binary
// (apply-progress.md, Fases 11-14, Issues Found #8): `axiom sdd gate
// record --decision rejected` on a fixed gate judged against a REAL,
// non-empty artifact must survive the very next `sdd status` read as
// "rejected", carrying its exact reason — not silently reopen to
// "pending" the moment status is re-resolved.
//
// Root cause (design.md D-08, machine.go's resolveGateStatus): a rejection
// only stays rejected while its recorded ArtifactDigest still matches the
// artifact's CURRENT digest. runSDDGateRecord never computed that digest at
// all, so every recorded GateRecord carried ArtifactDigest == "" while
// sddstatus.loadGovernance always recomputes a real, non-empty digest for
// an artifact that actually has content — the two can never match, so
// resolveGateStatus's "digest unchanged" branch is never taken and the
// gate is reported "reopened" on every subsequent read, even though
// nothing was ever remediated.
func TestRunSDDGateRecordRejectionPersistsAcrossStatusReads(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
	sealForGateTest(t, root, "inc-99-example")

	// A real, non-empty design.md: the digest recomputed at read time must
	// be non-empty for this defect to reproduce (an absent or empty
	// artifact already digests to "", which the pre-fix writer's empty
	// ArtifactDigest would spuriously "match").
	if err := os.WriteFile(filepath.Join(changeRoot, "design.md"), []byte("# Diseno\n\nContenido real.\n"), 0644); err != nil {
		t.Fatalf("preparar design.md: %v", err)
	}

	const rejectionReason = "falta cubrir un caso limite"
	var out bytes.Buffer
	if err := RunSDDGate([]string{
		"record", "--cwd", root, "--change", "inc-99-example",
		"--gate", "design", "--decision", "rejected", "--reason", rejectionReason, "--actor", "reviewer",
	}, &out); err != nil {
		t.Fatalf("RunSDDGate() error = %v", err)
	}

	status, err := sddstatus.Resolve(sddstatus.ResolveOptions{CWD: root, ChangeName: "inc-99-example"})
	if err != nil {
		t.Fatalf("sddstatus.Resolve() error = %v", err)
	}
	if status.Governance == nil {
		t.Fatal("status.Governance = nil, se esperaba gobernanza proyectada (kickoff sellado en checkpointed)")
	}

	found := false
	for _, gate := range status.Governance.Gates {
		if string(gate.Key) != "design" {
			continue
		}
		found = true
		if gate.Status != "rejected" {
			t.Fatalf(`gate "design".Status = %q, se esperaba "rejected" (el rechazo no debe reabrirse solo porque se releyo el estado)`, gate.Status)
		}
		if gate.Reason != rejectionReason {
			t.Fatalf("gate \"design\".Reason = %q, se esperaba %q", gate.Reason, rejectionReason)
		}
		if gate.Reopened {
			t.Fatal(`gate "design".Reopened = true, se esperaba false: el artefacto no cambio desde el rechazo`)
		}
	}
	if !found {
		t.Fatal(`no se encontro la compuerta "design" en status.Governance.Gates`)
	}

	if status.NextRecommended != "design" {
		t.Fatalf("NextRecommended = %q, se esperaba \"design\" (la fase duena del rechazo, D-09) en vez de await-gate", status.NextRecommended)
	}
	if !strings.Contains(strings.Join(status.BlockedReasons, "\n"), rejectionReason) {
		t.Fatalf("BlockedReasons = %v, se esperaba el motivo exacto del rechazo", status.BlockedReasons)
	}
}

// TestRunSDDGateRecordRejectionReopensAfterRealRemediation triangulates the
// fix from the opposite direction: D-08's whole point is that a rejection
// DOES reopen once the artifact genuinely changes. Fixing the digest gap
// must not turn every rejection into a permanent one — remediation must
// still work exactly as machine.go already proves it does at the unit
// level.
func TestRunSDDGateRecordRejectionReopensAfterRealRemediation(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
	sealForGateTest(t, root, "inc-99-example")

	designPath := filepath.Join(changeRoot, "design.md")
	if err := os.WriteFile(designPath, []byte("# Diseno\n\nVersion original.\n"), 0644); err != nil {
		t.Fatalf("preparar design.md: %v", err)
	}
	if err := RunSDDGate([]string{
		"record", "--cwd", root, "--change", "inc-99-example",
		"--gate", "design", "--decision", "rejected", "--reason", "falta X", "--actor", "reviewer",
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunSDDGate() error = %v", err)
	}

	// Remediate: the artifact's content genuinely changes.
	if err := os.WriteFile(designPath, []byte("# Diseno\n\nVersion remediada con X cubierto.\n"), 0644); err != nil {
		t.Fatalf("remediar design.md: %v", err)
	}

	status, err := sddstatus.Resolve(sddstatus.ResolveOptions{CWD: root, ChangeName: "inc-99-example"})
	if err != nil {
		t.Fatalf("sddstatus.Resolve() error = %v", err)
	}
	for _, gate := range status.Governance.Gates {
		if string(gate.Key) != "design" {
			continue
		}
		if gate.Status != "pending" || !gate.Reopened {
			t.Fatalf(`gate "design" = {Status:%q Reopened:%v}, se esperaba pending+Reopened tras remediar de verdad`, gate.Status, gate.Reopened)
		}
	}
}

// TestRunSDDGateRecordRoleApplyRejectionPersistsAcrossStatusReads
// triangulates the fix for a role-apply:<rol> gate, not only the four
// fixed gates: the same digest gap applied there too
// (roleApplyArtifactDigestForRecord), judged against the role's own
// sealed TasksFile.
func TestRunSDDGateRecordRoleApplyRejectionPersistsAcrossStatusReads(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
	sealForGateTest(t, root, "inc-99-example")

	if err := os.WriteFile(filepath.Join(changeRoot, "tasks.md"), []byte("- [x] 1.1 hecho\n"), 0644); err != nil {
		t.Fatalf("preparar tasks.md: %v", err)
	}

	const rejectionReason = "no cumple la spec asignada"
	if err := RunSDDGate([]string{
		"record", "--cwd", root, "--change", "inc-99-example",
		"--gate", "role-apply:fullstack", "--decision", "rejected", "--reason", rejectionReason, "--actor", "reviewer",
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunSDDGate() error = %v", err)
	}

	status, err := sddstatus.Resolve(sddstatus.ResolveOptions{CWD: root, ChangeName: "inc-99-example"})
	if err != nil {
		t.Fatalf("sddstatus.Resolve() error = %v", err)
	}
	found := false
	for _, gate := range status.Governance.Gates {
		if string(gate.Key) != "role-apply:fullstack" {
			continue
		}
		found = true
		if gate.Status != "rejected" || gate.Reason != rejectionReason {
			t.Fatalf(`gate "role-apply:fullstack" = {Status:%q Reason:%q}, se esperaba rejected/%q sin remediacion`, gate.Status, gate.Reason, rejectionReason)
		}
	}
	if !found {
		t.Fatal(`no se encontro la compuerta "role-apply:fullstack" en status.Governance.Gates`)
	}
}

// TestRunSDDGateRecordSpecGateDigestCoversMultiFileSpecs confirms the fix
// resolves the SAME multi-file "spec" artifact set
// internal/sddstatus/status.go's findSpecFiles does (every
// specs/**/spec.md, sorted), not only a flat spec.md — a rejection
// judged against a real per-capability spec file must persist too.
func TestRunSDDGateRecordSpecGateDigestCoversMultiFileSpecs(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
	sealForGateTest(t, root, "inc-99-example")

	specPath := filepath.Join(changeRoot, "specs", "auth", "spec.md")
	if err := os.MkdirAll(filepath.Dir(specPath), 0755); err != nil {
		t.Fatalf("preparar specs/auth: %v", err)
	}
	if err := os.WriteFile(specPath, []byte("### Requirement: Auth\n#### Scenario: x\n"), 0644); err != nil {
		t.Fatalf("preparar specs/auth/spec.md: %v", err)
	}

	const rejectionReason = "falta cubrir sesion expirada"
	if err := RunSDDGate([]string{
		"record", "--cwd", root, "--change", "inc-99-example",
		"--gate", "spec", "--decision", "rejected", "--reason", rejectionReason, "--actor", "reviewer",
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunSDDGate() error = %v", err)
	}

	status, err := sddstatus.Resolve(sddstatus.ResolveOptions{CWD: root, ChangeName: "inc-99-example"})
	if err != nil {
		t.Fatalf("sddstatus.Resolve() error = %v", err)
	}
	found := false
	for _, gate := range status.Governance.Gates {
		if string(gate.Key) != "spec" {
			continue
		}
		found = true
		if gate.Status != "rejected" || gate.Reason != rejectionReason {
			t.Fatalf(`gate "spec" = {Status:%q Reason:%q}, se esperaba rejected/%q sin remediacion`, gate.Status, gate.Reason, rejectionReason)
		}
	}
	if !found {
		t.Fatal(`no se encontro la compuerta "spec" en status.Governance.Gates`)
	}
}

// TestRunSDDGateRecordTasksGateDigestCoversEveryRoleFile pins the read/write
// agreement for a change sealed with several roles: the tasks gate is judged
// against the role task files (there is no shared tasks.md), the record digests
// that same set, and the very next status read keeps the approval instead of
// reopening it.
func TestRunSDDGateRecordTasksGateDigestCoversEveryRoleFile(t *testing.T) {
	root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
	sealForGateTest(t, root, "inc-99-example", "--role", "backend", "--role", "frontend")

	if got, err := digestIfAllPresent(tasksArtifactPathsForDigest(changeRoot, mustLoadKickoff(t, changeRoot))); err != nil || got != "" {
		t.Fatalf("digest before any role file exists = (%q, %v), want none", got, err)
	}
	for role, text := range map[string]string{"backend": "- [ ] 1.1 api\n", "frontend": "- [ ] 1.1 vista\n"} {
		if err := os.WriteFile(filepath.Join(changeRoot, "tasks."+role+".md"), []byte(text), 0644); err != nil {
			t.Fatalf("preparar tasks.%s.md: %v", role, err)
		}
	}

	if err := RunSDDGate([]string{
		"record", "--cwd", root, "--change", "inc-99-example",
		"--gate", "tasks", "--decision", "approved", "--reason", "plan por rol coherente", "--actor", "reviewer",
	}, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunSDDGate() error = %v", err)
	}

	status, err := sddstatus.Resolve(sddstatus.ResolveOptions{CWD: root, ChangeName: "inc-99-example"})
	if err != nil || status.Governance == nil {
		t.Fatalf("sddstatus.Resolve() = (governance=%v, err=%v)", status.Governance != nil, err)
	}
	found := false
	for _, gate := range status.Governance.Gates {
		if gate.Key != kickoff.GateTasks {
			continue
		}
		found = true
		if gate.Status != "approved" || gate.Reopened {
			t.Fatalf(`gate "tasks" = {Status:%q Reopened:%v}, se esperaba approved sin reabrir: el digest del registro y el de status deben coincidir`, gate.Status, gate.Reopened)
		}
	}
	if !found {
		t.Fatal(`no se encontro la compuerta "tasks" en status.Governance.Gates`)
	}
}

func mustLoadKickoff(t *testing.T, changeRoot string) *kickoff.Kickoff {
	t.Helper()
	sealed, err := kickoff.Load(changeRoot)
	if err != nil || sealed == nil {
		t.Fatalf("kickoff.Load() = (%v, %v), se esperaba un kickoff sellado", sealed, err)
	}
	return sealed
}

// TestGateDigestBugWithoutDelta fija el acuerdo de D20 entre `gate record` y el
// status: sin delta, un bug declarado en proposal.md hace que la compuerta
// `spec` juzgue proposal.md, y el rechazo registrado sobrevive a la siguiente
// lectura del status; una declaración incompleta no produce huella.
func TestGateDigestBugWithoutDelta(t *testing.T) {
	const report = "## Informe de bug\n\nObservado: el total omite el IVA.\nEsperado: el total lo incluye.\n"

	t.Run("declaración completa", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		sealForGateTest(t, root, "inc-99-example")
		proposal := filepath.Join(changeRoot, "proposal.md")
		if err := os.WriteFile(proposal, []byte("# Propuesta\n\n"+report), 0644); err != nil {
			t.Fatal(err)
		}

		want, err := kickoff.ArtifactDigest([]string{proposal})
		if err != nil {
			t.Fatal(err)
		}
		got, err := gateArtifactDigest(changeRoot, kickoff.GateSpec, mustLoadKickoff(t, changeRoot))
		if err != nil || got != want {
			t.Fatalf("gateArtifactDigest(spec) = (%q, %v), se esperaba la huella de proposal.md %q", got, err, want)
		}

		const reason = "falta la referencia externa"
		if err := RunSDDGate([]string{
			"record", "--cwd", root, "--change", "inc-99-example",
			"--gate", "spec", "--decision", "rejected", "--reason", reason, "--actor", "reviewer",
		}, &bytes.Buffer{}); err != nil {
			t.Fatalf("RunSDDGate() error = %v", err)
		}
		status, err := sddstatus.Resolve(sddstatus.ResolveOptions{CWD: root, ChangeName: "inc-99-example"})
		if err != nil || status.Governance == nil {
			t.Fatalf("sddstatus.Resolve() = (governance=%v, err=%v)", status.Governance != nil, err)
		}
		var found bool
		for _, gate := range status.Governance.Gates {
			if gate.Key != kickoff.GateSpec {
				continue
			}
			found = true
			if gate.Status != "rejected" || gate.Reason != reason || gate.Reopened {
				t.Fatalf(`gate "spec" = %+v, se esperaba rejected sin reabrir: la huella del registro y la del status deben coincidir`, gate)
			}
		}
		if !found {
			t.Fatal(`no se encontro la compuerta "spec" en status.Governance.Gates`)
		}
	})

	t.Run("declaración incompleta", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		sealForGateTest(t, root, "inc-99-example")
		if err := os.WriteFile(filepath.Join(changeRoot, "proposal.md"), []byte("## Informe de bug\n\nObservado: algo.\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if got, err := gateArtifactDigest(changeRoot, kickoff.GateSpec, mustLoadKickoff(t, changeRoot)); err != nil || got != "" {
			t.Fatalf("gateArtifactDigest(spec) = (%q, %v), una declaración incompleta no produce huella", got, err)
		}
	})

	t.Run("con delta juzga las specs", func(t *testing.T) {
		root, changeRoot := newGovernanceWorkspace(t, "inc-99-example")
		sealForGateTest(t, root, "inc-99-example")
		if err := os.WriteFile(filepath.Join(changeRoot, "proposal.md"), []byte(report), 0644); err != nil {
			t.Fatal(err)
		}
		specPath := filepath.Join(changeRoot, "specs", "auth", "spec.md")
		if err := os.MkdirAll(filepath.Dir(specPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(specPath, []byte("### Requirement: Auth\n#### Scenario: x\n"), 0644); err != nil {
			t.Fatal(err)
		}
		want, err := kickoff.ArtifactDigest([]string{specPath})
		if err != nil {
			t.Fatal(err)
		}
		if got, err := gateArtifactDigest(changeRoot, kickoff.GateSpec, mustLoadKickoff(t, changeRoot)); err != nil || got != want {
			t.Fatalf("gateArtifactDigest(spec) = (%q, %v), con delta se esperaba la huella de las specs %q", got, err, want)
		}
	})
}
