package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const (
	testsCheckBase = "workspace:\n  name: ejemplo-canonico\n  topology: multirepo\n  specs_repository: repo-specs\nroles:\n  backend:\n    name: Backend\n    repositories:\n      - path: repo-backend\n"
	testsCheckOn   = "acceptance_tests:\n  enabled: true\n  environments: [dev, pre]\n"
	testsCheckOff  = "acceptance_tests:\n  enabled: false\n"
	testsCheckHead = "| ID | Cubre | Rol | Tipo | Prioridad | Entornos | Precondiciones | Pasos | Resultado esperado |\n|---|---|---|---|---|---|---|---|---|\n"
	testsCheckSpec = "## ADDED Requirements\n\n### Requirement: Uno REQ-1.1\n\nTexto.\n\n#### Scenario: A\n- **DADO** x\n\n#### Scenario: B\n- **DADO** x\n"
)

func testsCheckRow(id, cubre, envs string) string {
	return "| " + id + " | " + cubre + " | backend | manual | alta | " + envs + " |  | abrir | ok |\n"
}

// testsCheckChange monta un cambio multirepo con una delta de dos escenarios y las tareas hechas.
func testsCheckChange(t *testing.T, block string, withSpec bool) (master, changeRoot string) {
	t.Helper()
	ws := workspacetest.Canonical(t)
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), testsCheckBase+block)
	changeRoot = filepath.Join(ws.Specs, "openspec", "changes", "inc-demo")
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "proposal.md"), "# Proposal\n")
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "design.md"), "# Design\n")
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "tasks.md"), "- [ ] 1.1 Trabajo\n")
	if withSpec {
		workspacetest.WriteFile(t, filepath.Join(changeRoot, "specs", "auth", "spec.md"), testsCheckSpec)
	}
	_, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: "inc-demo", SealedBy: "test",
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionCheckpointed, ExecutionStyleSource: "explicit",
			Roles: []kickoff.KickoffRole{{Role: "backend", GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"}},
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	})
	if err != nil || !ok {
		t.Fatalf("kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}
	return ws.Master, changeRoot
}

func runTestsCheck(t *testing.T, master string, extra ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := RunSDDTests(append([]string{"check", "--cwd", master, "--change", "inc-demo"}, extra...), &out)
	return out.String(), err
}

func TestSDDTestsCheckValidBatteryTextAndJSON(t *testing.T) {
	master, changeRoot := testsCheckChange(t, testsCheckOn, true)
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "pruebas.md"),
		testsCheckHead+testsCheckRow("P-01", "REQ-1.1#A", "dev")+testsCheckRow("P-02", "REQ-1.1#B", "dev;pre"))

	text, err := runTestsCheck(t, master)
	if err != nil {
		t.Fatalf("texto: %v\n%s", err, text)
	}
	for _, want := range []string{"Escenarios: 2/2", "Requisitos: 1/1", "Casos: 2"} {
		if !strings.Contains(text, want) {
			t.Errorf("el texto no contiene %q:\n%s", want, text)
		}
	}
	raw, err := runTestsCheck(t, master, "--json")
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	var got testsCheckReport
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if got.Schema != testsCheckSchema || !got.Active || !got.Present || got.Cases != 2 ||
		got.Scenarios.Covered != 2 || got.Scenarios.Total != 2 || got.Requirements.Covered != 1 || len(got.Findings) != 0 ||
		strings.Join(got.Environments, ",") != "dev,pre" {
		t.Fatalf("informe inesperado: %+v", got)
	}
}

func TestSDDTestsCheckFindingsExitWithErrorAfterPrintingTheReport(t *testing.T) {
	master, changeRoot := testsCheckChange(t, testsCheckOn, true)
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "pruebas.md"), testsCheckHead+testsCheckRow("P-01", "REQ-1.1#A", "dev"))

	text, err := runTestsCheck(t, master)
	if err == nil {
		t.Fatalf("se esperaba error con un escenario sin cubrir:\n%s", text)
	}
	if !strings.Contains(text, "escenario_sin_cubrir") || !strings.Contains(text, "Escenarios: 1/2") {
		t.Errorf("el informe completo debe imprimirse antes del error:\n%s", text)
	}
	raw, _ := runTestsCheck(t, master, "--json")
	var got testsCheckReport
	if err := json.Unmarshal([]byte(raw), &got); err != nil || len(got.Findings) == 0 || got.Findings[0].Code != "escenario_sin_cubrir" {
		t.Fatalf("json: %v %+v", err, got)
	}
}

func TestSDDTestsCheckMissingBatteryWhenActive(t *testing.T) {
	master, _ := testsCheckChange(t, testsCheckOn, true)
	text, err := runTestsCheck(t, master)
	if err == nil || !strings.Contains(text, "pruebas_ausente") {
		t.Fatalf("se esperaba error con pruebas_ausente: err=%v\n%s", err, text)
	}
}

func TestSDDTestsCheckInactiveWithoutFileSucceeds(t *testing.T) {
	for name, block := range map[string]string{"sin bloque": "", "enabled false": testsCheckOff} {
		t.Run(name, func(t *testing.T) {
			master, _ := testsCheckChange(t, block, true)
			text, err := runTestsCheck(t, master)
			if err != nil || !strings.Contains(text, "no están activadas") {
				t.Fatalf("err=%v\n%s", err, text)
			}
			raw, err := runTestsCheck(t, master, "--json")
			var got testsCheckReport
			if err != nil || json.Unmarshal([]byte(raw), &got) != nil || got.Active || got.Present || len(got.Findings) != 0 {
				t.Fatalf("json: err=%v %s", err, raw)
			}
		})
	}
}

func TestSDDTestsCheckInactiveWithFileIgnoresEnvironments(t *testing.T) {
	master, changeRoot := testsCheckChange(t, "", true)
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "pruebas.md"),
		testsCheckHead+testsCheckRow("P-01", "REQ-1.1#A", "produccion")+testsCheckRow("P-02", "REQ-1.1#B", "produccion"))
	if text, err := runTestsCheck(t, master); err != nil {
		t.Fatalf("no debe validar Entornos sin dominio declarado: %v\n%s", err, text)
	}
	// El resto de comprobaciones sí se aplican.
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "pruebas.md"), testsCheckHead+testsCheckRow("P-01", "REQ-1.1#A", "produccion"))
	text, err := runTestsCheck(t, master)
	if err == nil || !strings.Contains(text, "escenario_sin_cubrir") || strings.Contains(text, "entorno_invalido") {
		t.Fatalf("err=%v\n%s", err, text)
	}
}

func TestSDDTestsCheckBeforeTasksWithoutBatteryIsNotAnError(t *testing.T) {
	master, changeRoot := testsCheckChange(t, testsCheckOn, true)
	if err := os.Remove(filepath.Join(changeRoot, "tasks.md")); err != nil {
		t.Fatal(err)
	}
	text, err := runTestsCheck(t, master)
	if err != nil || !strings.Contains(text, "todavía no hay batería") {
		t.Fatalf("err=%v\n%s", err, text)
	}
}

func TestSDDTestsCheckUsageErrors(t *testing.T) {
	master, _ := testsCheckChange(t, testsCheckOn, false)
	if text, err := runTestsCheck(t, master); err == nil || !strings.Contains(err.Error(), "specs delta") {
		t.Errorf("sin specs delta: err=%v\n%s", err, text)
	}
	var out bytes.Buffer
	if err := RunSDDTests([]string{"check", "--cwd", master, "--change", "no-existe"}, &out); err == nil || !strings.Contains(err.Error(), "no existe") {
		t.Errorf("cambio inexistente: %v", err)
	}
	if err := RunSDDTests([]string{"check", "--cwd", master}, &out); err == nil || !strings.Contains(err.Error(), "requiere --change") {
		t.Errorf("sin --change: %v", err)
	}
	if err := RunSDDTests([]string{"otro"}, &out); err == nil || !strings.Contains(err.Error(), "no reconocido") {
		t.Errorf("subcomando: %v", err)
	}
	if err := RunSDDTests(nil, &out); err == nil {
		t.Error("sin subcomando debe fallar")
	}
	out.Reset()
	if err := RunSDDTests([]string{"--help"}, &out); err != nil || !strings.Contains(out.String(), "axiom sdd tests check") {
		t.Errorf("ayuda: %v %s", err, out.String())
	}
}

func TestSDDTestsCheckInvalidDeclarationFails(t *testing.T) {
	master, _ := testsCheckChange(t, "acceptance_tests:\n  enabled: true\n", true)
	text, err := runTestsCheck(t, master)
	if err == nil || !strings.Contains(err.Error(), "workspace validate") {
		t.Fatalf("err=%v\n%s", err, text)
	}
}

func TestSDDTestsCheckIsReadOnly(t *testing.T) {
	master, changeRoot := testsCheckChange(t, testsCheckOn, true)
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "pruebas.md"), testsCheckHead+testsCheckRow("P-01", "REQ-1.1#A", "dev"))
	snapshot := func() map[string]string {
		files := map[string]string{}
		_ = filepath.Walk(filepath.Dir(changeRoot), func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				data, _ := os.ReadFile(path)
				files[path] = string(data)
			}
			return nil
		})
		return files
	}
	before := snapshot()
	_, _ = runTestsCheck(t, master)
	_, _ = runTestsCheck(t, master, "--json")
	after := snapshot()
	if len(before) != len(after) {
		t.Fatalf("el verbo creó o borró ficheros: %d -> %d", len(before), len(after))
	}
	for path, content := range before {
		if after[path] != content {
			t.Errorf("el verbo modificó %s", path)
		}
	}
}
