package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Las pruebas de `axiom sdd archive-integrate` (REQ-47.8, REQ-21.17, REQ-47.3). Los nombres
// empiezan por SDDArchiveIntegrate. Todos los textos son inventados.

const (
	sddIntPagos = "# Especificación: pagos\n\n## Requirements\n\n### Requirement: Cobro (REQ-30.1)\n\nEl sistema DEBE cobrar.\n\n### Requirement: Reembolso (REQ-30.2)\n\nEl sistema DEBE reembolsar.\n"
	sddIntDelta = "## ADDED Requirements\n\n### Requirement: Cobro con IVA (prov:30:1)\n\nEl sistema DEBE cobrar con IVA.\n"
	sddIntNuevo = "# Especificación: nuevo\n\n## Requirements\n\n### Requirement: Uno (prov:a:1)\n\nA.\n\n### Requirement: Dos (prov:a:2)\n\nB.\n"
)

// sddIntegrateProject monta el proyecto con el cambio `x` sellado y los ficheros dados (rutas
// bajo openspec/), y deja el checkout en la rama de archivado.
func sddIntegrateProject(t *testing.T, files map[string]string) workspacetest.Workspace {
	t.Helper()
	ws := guardProject(t)
	for rel, content := range files {
		workspacetest.WriteFile(t, filepath.Join(ws.Home, filepath.FromSlash(rel)), content)
	}
	integrateGit(t, ws.Master, "switch", "-q", "-c", guardArchiveBranch)
	return ws
}

func sddIntegrateFiles() map[string]string {
	return map[string]string{
		"specs/pagos/spec.md":           sddIntPagos,
		"changes/x/specs/pagos/spec.md": sddIntDelta,
		"changes/x/specs/nuevo/spec.md": sddIntNuevo,
		"changes/x/tasks.md":            "- [ ] 1.1 Hacer\n  Cubre: prov:30:1, prov:a:2\n",
		"changes/x/verify-report.md":    "# V\n\n## Matriz de verificación\n\n" + archiveCheckMatrixHeader + "| prov:30:1 | Cobro | unitario | `x_test.go` | `go test` | 0 | ana | 2026-11-26 | n/a | pass | abc1234 |\n",
	}
}

func runSDDIntegrate(t *testing.T, ws workspacetest.Workspace, extra ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := RunSDDArchiveIntegrate(append([]string{"--cwd", ws.Master, "--change", sealChange}, extra...), &out)
	return out.String(), err
}

func decodeSDDReport(t *testing.T, out string) livespec.ChangeReport {
	t.Helper()
	var report livespec.ChangeReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("la salida no es un informe JSON: %v\n%s", err, out)
	}
	return report
}

func TestSDDArchiveIntegrateFullPassEmitsTheCompleteSchema(t *testing.T) {
	ws := sddIntegrateProject(t, sddIntegrateFiles())
	out, err := runSDDIntegrate(t, ws, "--json")
	if err != nil {
		t.Fatalf("RunSDDArchiveIntegrate: %v", err)
	}
	for _, key := range []string{`"schema": "axiom.sdd-archive-integrate/v1"`, `"capabilities"`, `"requirements"`, `"assignments"`, `"duplicates"`, `"written"`, `"nothingToIntegrate"`, `"check": false`} {
		if !strings.Contains(out, key) {
			t.Errorf("el informe debe contener %s:\n%s", key, out)
		}
	}
	report := decodeSDDReport(t, out)
	if report.Change != sealChange || report.Check || report.NothingToIntegrate || report.Error != nil {
		t.Errorf("informe = %+v", report)
	}
	outcomes := map[string]livespec.Outcome{}
	for _, c := range report.Capabilities {
		outcomes[c.Name] = c.Outcome
	}
	if outcomes["pagos"] != livespec.OutcomeComposed || outcomes["nuevo"] != livespec.OutcomeCreated {
		t.Errorf("capabilities = %+v", report.Capabilities)
	}
	got := map[string]string{}
	for _, a := range report.Assignments {
		got[a.Provisional] = a.Definitive
		if a.State != livespec.StateAssigned || a.Capability == "" || a.Requirement == "" {
			t.Errorf("assignment = %+v", a)
		}
	}
	if got["prov:30:1"] != "REQ-30.3" || got["prov:a:1"] != "REQ-31.1" || got["prov:a:2"] != "REQ-31.2" {
		t.Errorf("assignments = %v", got)
	}
	if len(report.Requirements.Creates) != 3 || len(report.Duplicates) != 0 || len(report.Written) < 6 {
		t.Errorf("requirements %+v, duplicates %+v, written %v", report.Requirements, report.Duplicates, report.Written)
	}
	if spec := readODDFile(t, filepath.Join(ws.Home, "specs", "nuevo", "spec.md")); strings.Contains(spec, "prov:") || !strings.Contains(spec, "Uno (REQ-31.1)") {
		t.Errorf("la capacidad nueva se escribe con definitivos:\n%s", spec)
	}
	if tasks := readODDFile(t, filepath.Join(ws.Home, "changes", "x", "tasks.md")); !strings.Contains(tasks, "Cubre: REQ-30.3, REQ-31.2") {
		t.Errorf("tasks.md:\n%s", tasks)
	}
	if trace := readODDFile(t, filepath.Join(ws.Home, "changes", "x", "ids.yaml")); !strings.Contains(trace, "prov:30:1") || !strings.Contains(trace, "REQ-30.3") {
		t.Errorf("ids.yaml:\n%s", trace)
	}
}

func TestSDDArchiveIntegrateTextOutputNamesTheResultAndTheMap(t *testing.T) {
	ws := sddIntegrateProject(t, sddIntegrateFiles())
	out, err := runSDDIntegrate(t, ws)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"pagos", "composed", "nuevo", "created", "prov:30:1", "REQ-30.3", "Escrito"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida debe contener %q:\n%s", want, out)
		}
	}
}

func TestSDDArchiveIntegrateRerunIsIdempotent(t *testing.T) {
	ws := sddIntegrateProject(t, sddIntegrateFiles())
	if _, err := runSDDIntegrate(t, ws); err != nil {
		t.Fatal(err)
	}
	before := oddSnapshot(t, ws.Master)
	out, err := runSDDIntegrate(t, ws, "--json")
	if err != nil {
		t.Fatalf("reejecución: %v", err)
	}
	report := decodeSDDReport(t, out)
	if !report.NothingToIntegrate || len(report.Written) != 0 || len(report.Assignments) != 3 {
		t.Errorf("informe = %+v", report)
	}
	for _, a := range report.Assignments {
		if a.State != livespec.StateKept {
			t.Errorf("assignment = %+v", a)
		}
	}
	for _, c := range report.Capabilities {
		if c.Outcome != livespec.OutcomeAlready {
			t.Errorf("capacidad %+v debe ser already_integrated", c)
		}
	}
	if !oddSameSnapshot(before, oddSnapshot(t, ws.Master)) {
		t.Error("repetir el paso no cambia ningún byte")
	}
}

func TestSDDArchiveIntegrateCheckWritesNothingAndSkipsOnlyTheBranchGuard(t *testing.T) {
	ws := sddIntegrateProject(t, sddIntegrateFiles())
	integrateGit(t, ws.Master, "switch", "-q", "main") // fuera de la rama de archivado
	before := oddSnapshot(t, ws.Master)

	if _, err := runSDDIntegrate(t, ws); err == nil {
		t.Fatal("sin --check la guarda A rechaza fuera de la rama de archivado")
	}
	out, err := runSDDIntegrate(t, ws, "--check", "--json")
	if err != nil {
		t.Fatalf("--check: %v", err)
	}
	report := decodeSDDReport(t, out)
	if !report.Check || len(report.Written) != 0 || len(report.Assignments) != 3 {
		t.Errorf("informe = %+v", report)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, ws.Master)) {
		t.Error("--check y el rechazo no escriben nada")
	}
}

func TestSDDArchiveIntegrateGuardARefusesOutsideTheArchiveBranchWithoutWriting(t *testing.T) {
	ws := sddIntegrateProject(t, sddIntegrateFiles())
	integrateGit(t, ws.Master, "switch", "-q", "feat/x")
	before := oddSnapshot(t, ws.Master)

	_, err := runSDDIntegrate(t, ws)
	var refusal *branch.RefusalError
	if !errors.As(err, &refusal) || refusal.Code != branch.CodeWrongBranch {
		t.Fatalf("error = %v, quería wrong_branch", err)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, ws.Master)) {
		t.Error("el rechazo deja el árbol intacto")
	}
}

func TestSDDArchiveIntegrateGuardsRunInOrder(t *testing.T) {
	t.Run("baseline residual antes que la guarda de rama", func(t *testing.T) {
		ws := sddIntegrateProject(t, sddIntegrateFiles())
		integrateGit(t, ws.Master, "switch", "-q", "main")
		config := readODDFile(t, filepath.Join(ws.Master, "axiom.yaml")) + contextCLIBaseline
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), config)
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "03_Modelo_Datos.md"), "# Modelo\n")
		_, err := runSDDIntegrate(t, ws)
		var retired *workspace.BaselineRetiredError
		if !errors.As(err, &retired) {
			t.Fatalf("error = %v, quería *workspace.BaselineRetiredError", err)
		}
	})
	t.Run("patrón inaccesible antes que la guarda de rama", func(t *testing.T) {
		ws := sddIntegrateProject(t, sddIntegrateFiles())
		integrateGit(t, ws.Master, "switch", "-q", "main")
		config := readODDFile(t, filepath.Join(ws.Master, "axiom.yaml")) + "sdd:\n  id_pattern: \"sin-marcadores\"\n"
		workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), config)
		before := oddSnapshot(t, ws.Master)
		_, err := runSDDIntegrate(t, ws)
		if err == nil || !strings.Contains(err.Error(), "axiom workspace validate") {
			t.Fatalf("error = %v, quería la continuación de validate", err)
		}
		var refusal *branch.RefusalError
		if errors.As(err, &refusal) {
			t.Errorf("el patrón precede a la guarda de rama: %v", err)
		}
		if !oddSameSnapshot(before, oddSnapshot(t, ws.Master)) {
			t.Error("el rechazo no escribe nada")
		}
	})
	t.Run("negativa ISO", func(t *testing.T) {
		root, changeRoot := archiveCheckWorkspace(t, archiveCheckChange{
			name: "demo", proposal: "# P\n", deltas: map[string]string{"pagos": archiveCheckDeltaNoScene}, tasks: archiveCheckTasksDone, verify: archiveCheckVerify(), iso: true,
		})
		before := treeSnapshot(t, root)
		var out bytes.Buffer
		err := RunSDDArchiveIntegrate([]string{"--cwd", root, "--change", "demo"}, &out)
		if err == nil || !strings.Contains(err.Error(), "iso_requirement_scenarios") || !strings.Contains(err.Error(), "archive-check") {
			t.Fatalf("error = %v, quería la negativa ISO con la continuación", err)
		}
		if after := treeSnapshot(t, root); len(after) != len(before) {
			t.Errorf("la negativa no escribe nada (%s)", changeRoot)
		}
	})
}

func TestSDDArchiveIntegrateAbsoluteCwdFromAnotherDirectoryOnlyChangesTheWorkspace(t *testing.T) {
	ws := sddIntegrateProject(t, sddIntegrateFiles())
	elsewhere := t.TempDir()
	t.Chdir(elsewhere)

	if _, err := runSDDIntegrate(t, ws); err != nil {
		t.Fatalf("RunSDDArchiveIntegrate: %v", err)
	}
	entries, err := os.ReadDir(elsewhere)
	if err != nil || len(entries) != 0 {
		t.Errorf("el directorio de trabajo no recibe nada (%v, %v)", entries, err)
	}
	if spec := readODDFile(t, filepath.Join(ws.Home, "specs", "pagos", "spec.md")); !strings.Contains(spec, "REQ-30.3") {
		t.Errorf("la spec viva del workspace cambia:\n%s", spec)
	}
}

func TestSDDArchiveIntegrateFlatDeltaIsRejectedWithATypedErrorInTheReport(t *testing.T) {
	files := sddIntegrateFiles()
	files["changes/x/specs/spec.md"] = sddIntDelta
	ws := sddIntegrateProject(t, files)
	before := oddSnapshot(t, ws.Master)

	out, err := runSDDIntegrate(t, ws, "--json")
	if err == nil {
		t.Fatal("una delta plana se rechaza")
	}
	report := decodeSDDReport(t, out)
	if report.Error == nil || report.Error.Code != livespec.CodeFlatDeltaUnsupported || report.Error.Continuation == "" || report.Error.File == "" {
		t.Fatalf("error del informe = %+v", report.Error)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, ws.Master)) {
		t.Error("el rechazo no escribe nada")
	}
}

func TestSDDArchiveIntegrateNewCapabilityWithModifiedIsRejected(t *testing.T) {
	files := sddIntegrateFiles()
	files["changes/x/tasks.md"] = "- [ ] 1.1 Hacer\n"
	files["changes/x/specs/nuevo/spec.md"] = "## MODIFIED Requirements\n\n### Requirement: Uno (REQ-31.1)\n\nA.\n"
	ws := sddIntegrateProject(t, files)
	out, err := runSDDIntegrate(t, ws, "--json")
	var missing *livespec.MissingCapabilityError
	if !errors.As(err, &missing) || missing.Capability != "nuevo" {
		t.Fatalf("error = %v, quería *MissingCapabilityError de nuevo", err)
	}
	if report := decodeSDDReport(t, out); report.Error == nil || report.Error.Code != "capability_missing" {
		t.Errorf("error del informe = %+v", report.Error)
	}
}

func TestSDDArchiveIntegrateSupersedeFlagsReachEveryCapability(t *testing.T) {
	files := map[string]string{
		"specs/pagos/spec.md":           sddIntPagos,
		"changes/x/specs/pagos/spec.md": "## REMOVED Requirements\n\n### Requirement: Reembolso (REQ-30.2)\n\n(Reason: x)\n",
	}
	ws := sddIntegrateProject(t, files)
	if _, err := runSDDIntegrate(t, ws, "--supersede", "--superseded-requirements", "pagos/Cobro (REQ-30.1)"); err != nil {
		t.Fatalf("RunSDDArchiveIntegrate: %v", err)
	}
	spec := readODDFile(t, filepath.Join(ws.Home, "specs", "pagos", "spec.md"))
	if strings.Count(spec, "[SUPERSEDED / DEPRECADO]") < 2 {
		t.Errorf("--supersede y --superseded-requirements marcan los dos requisitos:\n%s", spec)
	}
	_, err := runSDDIntegrate(t, ws, "--superseded-requirements", "Nada")
	var planErr *livespec.IDPlanError
	if !errors.As(err, &planErr) || planErr.Code != livespec.CodeSupersededRequirementUnknown {
		t.Errorf("error = %v, quería superseded_requirement_unknown", err)
	}
}

func TestSDDArchiveIntegrateArgumentErrorsNameTheContinuation(t *testing.T) {
	for name, args := range map[string][]string{
		"sin --change":         {"--cwd", "."},
		"cambio reservado":     {"--cwd", ".", "--change", "archive"},
		"argumento posicional": {"--cwd", ".", "--change", "x", "sobra"},
		"bandera desconocida":  {"--cwd", ".", "--change", "x", "--nada"},
	} {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			err := RunSDDArchiveIntegrate(args, &out)
			if err == nil || !strings.Contains(err.Error(), "sdd archive-integrate") {
				t.Errorf("error = %v, quería la continuación del verbo", err)
			}
		})
	}
}

func TestSDDArchiveIntegrateHelp(t *testing.T) {
	var out bytes.Buffer
	if err := RunSDDArchiveIntegrate([]string{"--help"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"axiom sdd archive-integrate", "--check", "--supersede", "--superseded-requirements", "axiom.sdd-archive-integrate/v1"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("la ayuda debe contener %q:\n%s", want, out.String())
		}
	}
}
