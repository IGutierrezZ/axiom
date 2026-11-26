package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
)

// Las pruebas de `axiom sdd archive-check` con provisionales y de la traza tras archivar
// (REQ-47.5, REQ-47.6, REQ-47.8). Todos los textos son inventados.

const sddCheckTrace = `schema: axiom.requirement-ids/v1
pattern: REQ-<n>.<m>
assignments:
  - provisional: prov:30:1
    definitive: REQ-30.3
    capability: pagos
    requirement: Cobro con IVA (REQ-30.3)
`

type archiveCheckIDsReport struct {
	OK                  bool                        `json:"ok"`
	Error               string                      `json:"error"`
	Assignments         []livespec.AssignmentReport `json:"assignments"`
	PendingProvisionals []string                    `json:"pendingProvisionals"`
}

func decodeArchiveCheckIDs(t *testing.T, out string) archiveCheckIDsReport {
	t.Helper()
	var report archiveCheckIDsReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("la salida no es un informe JSON: %v\n%s", err, out)
	}
	return report
}

func TestSDDArchiveCheckReportsPendingProvisionalsAndNoAssignmentsBeforeTheStep(t *testing.T) {
	root, _ := archiveCheckWorkspace(t, archiveCheckChange{
		name: "demo", proposal: "# P\n", tasks: archiveCheckTasksDone,
		deltas: map[string]string{
			"pagos": "## ADDED Requirements\n\n### Requirement: Cobro con IVA (prov:30:1)\n\nCuerpo.\n\n### Requirement: Otro (prov:30:2)\n\nCuerpo.\n",
			"nuevo": "# Especificación: nuevo\n\n## Requirements\n\n### Requirement: Uno (prov:a:1)\n\nA.\n",
		},
	})
	out, err := runArchiveCheck(root, "demo", "--json")
	if err != nil {
		t.Fatalf("archive-check: %v", err)
	}
	report := decodeArchiveCheckIDs(t, out)
	if len(report.Assignments) != 0 || strings.Join(report.PendingProvisionals, ",") != "prov:a:1,prov:30:1,prov:30:2" {
		t.Errorf("assignments %+v, pendientes %v", report.Assignments, report.PendingProvisionals)
	}
}

func TestSDDArchiveCheckRecognisesAnAssignmentAlreadyMadeAfterAnInterruptedArchive(t *testing.T) {
	root, changeRoot := archiveCheckWorkspace(t, archiveCheckChange{
		name: "demo", proposal: "# P\n", tasks: archiveCheckTasksDone,
		deltas: map[string]string{"pagos": "## ADDED Requirements\n\n### Requirement: Cobro con IVA (REQ-30.3)\n\nCuerpo.\n"},
	})
	if err := os.WriteFile(filepath.Join(changeRoot, "ids.yaml"), []byte(sddCheckTrace), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runArchiveCheck(root, "demo", "--json")
	if err != nil {
		t.Fatalf("archive-check: %v", err)
	}
	report := decodeArchiveCheckIDs(t, out)
	if len(report.PendingProvisionals) != 0 || len(report.Assignments) != 1 ||
		report.Assignments[0].Provisional != "prov:30:1" || report.Assignments[0].Definitive != "REQ-30.3" || report.Assignments[0].Capability != "pagos" {
		t.Errorf("assignments %+v, pendientes %v", report.Assignments, report.PendingProvisionals)
	}
	text, err := runArchiveCheck(root, "demo")
	if err != nil || !strings.Contains(text, "prov:30:1 → REQ-30.3") {
		t.Errorf("la salida de texto nombra la asignación guardada (err %v):\n%s", err, text)
	}
}

func TestSDDArchiveCheckWithoutProvisionalsKeepsTheReportUnchanged(t *testing.T) {
	root, _ := archiveCheckWorkspace(t, archiveCheckChange{
		name: "demo", proposal: "# P\n", tasks: archiveCheckTasksDone,
		deltas: map[string]string{"pagos": "## ADDED Requirements\n\n### Requirement: Cobro con IVA (REQ-30.3)\n\nCuerpo.\n"},
	})
	out, err := runArchiveCheck(root, "demo", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "assignments") || strings.Contains(out, "pendingProvisionals") {
		t.Errorf("sin provisionales ni traza el informe no gana campos:\n%s", out)
	}
}

func TestSDDArchiveCheckRejectsAnUnreadableTrace(t *testing.T) {
	root, changeRoot := archiveCheckWorkspace(t, archiveCheckChange{name: "demo", proposal: "# P\n", tasks: archiveCheckTasksDone})
	if err := os.WriteFile(filepath.Join(changeRoot, "ids.yaml"), []byte("schema: otro/v9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runArchiveCheck(root, "demo")
	if err == nil || !strings.Contains(err.Error(), "ids.yaml") || !strings.Contains(err.Error(), "la traza nunca se regenera en silencio") {
		t.Fatalf("error = %v, quería el rechazo de la traza ilegible con su continuación", err)
	}
}

// Traza legible tras archivar: el mapa del cambio archivado se lee con `--check` y todas sus
// entradas constan como kept.
func TestSDDArchiveCheckArchivedTraceIsReadableAfterArchiving(t *testing.T) {
	ws := sddIntegrateProject(t, sddIntegrateFiles())
	if _, err := runSDDIntegrate(t, ws); err != nil {
		t.Fatal(err)
	}
	archived := filepath.Join(ws.Home, "changes", "archive", "2026-11-26-x")
	if err := os.MkdirAll(filepath.Dir(archived), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(ws.Home, "changes", "x"), archived); err != nil {
		t.Fatal(err)
	}
	before := oddSnapshot(t, ws.Master)

	out, err := runSDDIntegrate(t, ws, "--check", "--json")
	if err != nil {
		t.Fatalf("--check sobre el cambio archivado: %v", err)
	}
	report := decodeSDDReport(t, out)
	if !report.Check || report.Error != nil || len(report.Assignments) != 3 {
		t.Fatalf("informe = %+v", report)
	}
	for _, a := range report.Assignments {
		if a.State != livespec.StateKept {
			t.Errorf("assignment = %+v", a)
		}
	}
	if !oddSameSnapshot(before, oddSnapshot(t, ws.Master)) {
		t.Error("--check no escribe nada")
	}
}

// Una segunda rama de archivo rebasada sobre una que ya contiene el mismo número: `--check`
// sobre el cambio archivado da duplicate_assigned_id con la continuación de descartar la rama.
func TestSDDArchiveCheckArchivedTraceReportsADuplicateWithTheDiscardContinuation(t *testing.T) {
	ws := sddIntegrateProject(t, sddIntegrateFiles())
	if _, err := runSDDIntegrate(t, ws); err != nil {
		t.Fatal(err)
	}
	archived := filepath.Join(ws.Home, "changes", "archive", "2026-11-26-x")
	if err := os.MkdirAll(filepath.Dir(archived), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(ws.Home, "changes", "x"), archived); err != nil {
		t.Fatal(err)
	}
	// La rama de integración ya trajo otro cambio con REQ-30.3 en otra capacidad.
	if err := os.MkdirAll(filepath.Join(ws.Home, "specs", "facturacion"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Home, "specs", "facturacion", "spec.md"), []byte("# E\n\n## Requirements\n\n### Requirement: Ajeno (REQ-30.3)\n\nB.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := oddSnapshot(t, ws.Master)

	out, err := runSDDIntegrate(t, ws, "--check", "--json")
	var planErr *livespec.IDPlanError
	if !errors.As(err, &planErr) || planErr.Code != livespec.CodeDuplicateAssignedID || planErr.Token != "REQ-30.3" {
		t.Fatalf("error = %v, quería duplicate_assigned_id de REQ-30.3", err)
	}
	if !strings.Contains(err.Error(), "axiom branch done --force") {
		t.Errorf("la continuación descarta la rama de archivo: %v", err)
	}
	if report := decodeSDDReport(t, out); report.Error == nil || report.Error.Code != livespec.CodeDuplicateAssignedID {
		t.Errorf("error del informe = %+v", report.Error)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, ws.Master)) {
		t.Error("--check no escribe nada")
	}
}
