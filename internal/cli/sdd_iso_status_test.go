package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
)

// REQ-29.16: `axiom sdd continue` refleja los mismos motivos tipados que
// `axiom sdd status` y no autoriza la fase siguiente.
func TestSDDContinueReflectsISOReasons(t *testing.T) {
	root := t.TempDir()
	changeRoot := seedSDDStatusReadyChange(t, root, "iso-x", "- [ ] 1.1 Trabajo\n")
	writeSDDStatusFile(t, filepath.Join(changeRoot, "specs", "pagos", "spec.md"),
		"## ADDED Requirements\n\n### Requirement: Cobro con IVA\n\nTexto.\n\n#### Scenario: Cobro general\n\n- **CUANDO** se cobra\n")
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: "iso-x", SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionContinuous, ExecutionStyleSource: "explicit",
			HandoffPolicy: kickoff.HandoffNone,
			Roles:         []kickoff.KickoffRole{{Role: "fullstack", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
		SDD:       &kickoff.SDDSnapshot{ISO: true},
	})
	if err != nil || !ok {
		t.Fatalf("kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}

	decode := func(name string, run func([]string, *bytes.Buffer) error) sddstatus.Status {
		t.Helper()
		var stdout bytes.Buffer
		if err := run([]string{"iso-x", "--cwd", root, "--json"}, &stdout); err != nil {
			t.Fatalf("%s error = %v", name, err)
		}
		var status sddstatus.Status
		if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
			t.Fatalf("decode %s JSON: %v\n%s", name, err, stdout.String())
		}
		return status
	}
	status := decode("status", func(args []string, out *bytes.Buffer) error { return RunSDDStatus(args, out) })
	continued := decode("continue", func(args []string, out *bytes.Buffer) error { return RunSDDContinue(args, out) })

	if !strings.Contains(strings.Join(status.BlockedReasons, "\n"), "blocked(iso_requirement_id)") {
		t.Fatalf("status no proyecta el motivo tipado: %v", status.BlockedReasons)
	}
	if strings.Join(continued.BlockedReasons, "\n") != strings.Join(status.BlockedReasons, "\n") {
		t.Errorf("continue = %v, status = %v: deben reflejar los mismos motivos", continued.BlockedReasons, status.BlockedReasons)
	}
	if continued.NextRecommended != "spec" || continued.ApplyState != sddstatus.ApplyBlocked {
		t.Errorf("continue autoriza la fase siguiente: nextRecommended %q, applyState %q", continued.NextRecommended, continued.ApplyState)
	}
}
