package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Sin t.Parallel en estas pruebas: montan proyectos git completos.

const atDelta = "## ADDED Requirements\n\n### Requirement: Uno (REQ-1.1)\n\nTexto.\n\n#### Scenario: Alfa\n\n- **DADO** algo\n"

const atHeader = "| ID | Cubre | Rol | Tipo | Prioridad | Entornos | Precondiciones | Pasos | Resultado esperado |\n" +
	"|---|---|---|---|---|---|---|---|---|\n"

func atRow(id, cubre, steps string) string {
	return "| " + id + " | " + cubre + " | fullstack | manual | alta | dev | ninguna | " + steps + " | ok |\n"
}

func atFiles(pruebas string) map[string]string {
	files := map[string]string{"specs/dom/spec.md": atDelta, "tasks.md": "- [ ] T1 algo\n"}
	if pruebas != "" {
		files["pruebas.md"] = pruebas
	}
	return files
}

func TestIncrementDetailAcceptanceTestsMatchesTheShared(t *testing.T) {
	svc, root := explainerChange(t, "", atFiles(atHeader+atRow("P-01", "REQ-1.1#Alfa", "abrir")))
	dto, err := svc.GetIncrementDetail("c")
	if err != nil {
		t.Fatal(err)
	}
	at := dto.AcceptanceTests
	if at == nil || !at.Present || at.Error != "" {
		t.Fatalf("AcceptanceTests = %+v", at)
	}
	if at.CaseCount != 1 || len(at.Cases) != 1 || at.Cases[0].ID != "P-01" || at.Cases[0].Steps != "abrir" || at.Cases[0].Cubre != "REQ-1.1#Alfa" {
		t.Errorf("casos = %+v (count %d)", at.Cases, at.CaseCount)
	}
	if at.Scenarios.Covered != 1 || at.Scenarios.Total != 1 || at.Requirements.Covered != 1 || at.Requirements.Total != 1 || len(at.Findings) != 0 {
		t.Errorf("cobertura = %+v %+v hallazgos %+v", at.Scenarios, at.Requirements, at.Findings)
	}
	loc, err := workspace.Locate(svc.getRootPath())
	if err != nil {
		t.Fatal(err)
	}
	want := sddstatus.AcceptanceTestsFor(loc, root)
	if want.Report.Cases != at.CaseCount || len(want.Findings) != len(at.Findings) {
		t.Errorf("AcceptanceTestsFor = %+v, DTO = %+v", want.Report, at)
	}
}

func TestIncrementDetailAcceptanceTestsFindingsAndHostileText(t *testing.T) {
	hostile := "<script>alert(1)</script> \\| x"
	svc, _ := explainerChange(t, "", atFiles(atHeader+atRow("P-01", "REQ-9.9#Nada", hostile)))
	dto, err := svc.GetIncrementDetail("c")
	if err != nil {
		t.Fatal(err)
	}
	at := dto.AcceptanceTests
	if at == nil || !at.Present || len(at.Findings) == 0 {
		t.Fatalf("se esperaban hallazgos: %+v", at)
	}
	if len(at.Cases) != 1 || !strings.Contains(at.Cases[0].Steps, "<script>") {
		t.Errorf("el texto hostil debe viajar literal en el DTO: %+v", at.Cases)
	}
	if _, err := json.Marshal(dto); err != nil {
		t.Fatal(err)
	}
}

func TestIncrementDetailAcceptanceTestsInactiveWithoutFile(t *testing.T) {
	svc, _ := explainerChange(t, "", atFiles(""))
	dto, err := svc.GetIncrementDetail("c")
	if err != nil {
		t.Fatal(err)
	}
	at := dto.AcceptanceTests
	if at == nil || at.Active || at.Present || at.Warning != "" || len(at.Cases) != 0 {
		t.Errorf("función no activada: %+v", at)
	}
}

func TestIncrementDetailAcceptanceTestsActiveWithoutFileWarns(t *testing.T) {
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), `workspace:
  name: ejemplo-canonico
  topology: multirepo
  specs_repository: repo-specs
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
  frontend:
    name: Frontend
    repositories:
      - path: repo-frontend
acceptance_tests:
  enabled: true
  environments: [dev]
`)
	root := filepath.Join(ws.Home, "changes", "c")
	workspacetest.WriteFile(t, filepath.Join(root, "proposal.md"), "# c\n")
	workspacetest.WriteFile(t, filepath.Join(root, "tasks.md"), "- [ ] T1 algo\n")
	workspacetest.WriteFile(t, filepath.Join(root, "specs", "dom", "spec.md"), atDelta)
	svc := NewServiceWithHub(ws.Master, nil)

	dto, err := svc.GetIncrementDetail("c")
	if err != nil {
		t.Fatal(err)
	}
	at := dto.AcceptanceTests
	if at == nil || !at.Active || at.Present || !strings.Contains(at.Warning, "pruebas_ausente") || len(at.Findings) != 1 {
		t.Errorf("activa sin fichero: %+v", at)
	}
	if len(at.Environments) != 1 || at.Environments[0] != "dev" {
		t.Errorf("entornos = %v", at.Environments)
	}
}

func TestIncrementDetailAcceptanceTestsOmittedWhenArchived(t *testing.T) {
	svc, root := explainerChange(t, "", atFiles(atHeader+atRow("P-01", "REQ-1.1#Alfa", "abrir")))
	archived := filepath.Join(filepath.Dir(root), "archive", "2026-01-01-c")
	if err := os.MkdirAll(filepath.Dir(archived), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, archived); err != nil {
		t.Fatal(err)
	}
	dto, err := svc.GetIncrementDetail("c")
	if err != nil {
		t.Fatal(err)
	}
	if dto.AcceptanceTests != nil {
		t.Errorf("archivado: AcceptanceTests = %+v, se esperaba nil", dto.AcceptanceTests)
	}
	body, _ := json.Marshal(dto)
	if strings.Contains(string(body), "acceptance_tests") {
		t.Error("el campo debe omitirse del JSON")
	}
}

func TestIncrementAcceptanceTestsReportsLocateError(t *testing.T) {
	ws := workspacetest.Canonical(t, workspacetest.WithBrokenPointer())
	svc := NewServiceWithHub(ws.Master, nil)
	dto := svc.incrementAcceptanceTests(filepath.Join(ws.Home, "changes", "c"))
	if dto == nil || dto.Error == "" || dto.Cases == nil || dto.Findings == nil {
		t.Errorf("sin workspace: %+v", dto)
	}
}
