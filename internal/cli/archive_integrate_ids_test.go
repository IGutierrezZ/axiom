package cli

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Las pruebas de `axiom archive integrate --odd` con identificadores provisionales y de su
// opción `--check` (REQ-30.2, REQ-30.3, RF-RID-14, RF-OLS-10). Todos los textos son inventados.

const oddProvDelta = `## ADDED Requirements

### Requirement: Nota de abono (prov:31:1)

El sistema DEBE emitir notas de abono.
`

// oddProvDoc es un documento con un ADDED provisional con la serie 31 fijada y una línea
// Cubre: que lo cita.
func oddProvDoc(delta string) string {
	return oddDoc("Cubre: prov:31:1\n\n", oddSpecSection([2]string{"facturacion", delta}))
}

func TestArchiveIntegrateAssignsTheNumberRewritesTheDocumentAndReportsTheMap(t *testing.T) {
	f := newODDFixture(t, true)
	f.writeDoc(t, oddProvDoc(oddProvDelta))

	out, err := f.run(t, "--json")
	if err != nil {
		t.Fatalf("RunArchiveIntegrate: %v", err)
	}
	report := decodeODDReport(t, out)
	if report.Check || report.Requirements != nil {
		t.Errorf("sin --check el informe no trae check ni requirements: %+v", report)
	}
	if len(report.Assignments) != 1 {
		t.Fatalf("assignments = %+v", report.Assignments)
	}
	if a := report.Assignments[0]; a.Provisional != "prov:31:1" || a.Definitive != "REQ-31.2" || a.Capability != "facturacion" || a.State != livespec.StateAssigned {
		t.Errorf("assignment = %+v", a)
	}
	if want := []string{"openspec/specs/facturacion/spec.md", "odd/tasks/mi-feature.md"}; strings.Join(report.Written, ",") != strings.Join(want, ",") {
		t.Errorf("written = %v, quería %v", report.Written, want)
	}
	doc := readODDFile(t, f.docPath(oddFeature))
	if !strings.Contains(doc, "Cubre: REQ-31.2\n") || !strings.Contains(doc, "Nota de abono (REQ-31.2)") || !strings.Contains(doc, "## Identificadores asignados") {
		t.Errorf("el documento debe citar el definitivo y llevar la traza:\n%s", doc)
	}
	if spec := readODDFile(t, f.specPath("facturacion")); !strings.Contains(spec, "Nota de abono (REQ-31.2)") || strings.Contains(spec, "prov:") {
		t.Errorf("la spec viva lleva el definitivo y ningún provisional:\n%s", spec)
	}
}

func TestArchiveIntegrateIdempotentWithProvisionalsKeepsEveryEntry(t *testing.T) {
	f := newODDFixture(t, true)
	f.writeDoc(t, oddProvDoc(oddProvDelta))
	if _, err := f.run(t); err != nil {
		t.Fatalf("primera integración: %v", err)
	}
	before := oddSnapshot(t, f.root)

	out, err := f.run(t, "--json")
	if err != nil {
		t.Fatalf("segunda integración: %v", err)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("repetir la integración no cambia ningún byte")
	}
	report := decodeODDReport(t, out)
	if !report.NothingToIntegrate || len(report.Assignments) != 1 || report.Assignments[0].State != livespec.StateKept || len(report.Written) != 0 {
		t.Errorf("informe = %+v", report)
	}
}

// Otra rama ya fusionó su número: la integración de esta rama, que aún no había asignado,
// recibe el siguiente.
func TestArchiveIntegrateConcurrentBranchTakesTheNextNumberAfterTheOtherMerged(t *testing.T) {
	f := newODDFixture(t, true)
	workspacetest.WriteFile(t, f.specPath("facturacion"), oddFacturacionSpec+"\n### Requirement: Recargo (REQ-31.2)\n\nEl sistema DEBE recargar.\n")
	f.writeDoc(t, oddProvDoc(oddProvDelta))

	out, err := f.run(t, "--json")
	if err != nil {
		t.Fatalf("RunArchiveIntegrate: %v", err)
	}
	if a := decodeODDReport(t, out).Assignments; len(a) != 1 || a[0].Definitive != "REQ-31.3" || a[0].State != livespec.StateAssigned {
		t.Errorf("assignments = %+v", a)
	}
}

// Tras un rebase en el que la base ocupó el definitivo con otro cuerpo, la reejecución
// reasigna con previous y reescribe el documento.
func TestArchiveIntegrateRebaseReassignsTheNumberTakenByAnotherRequirement(t *testing.T) {
	f := newODDFixture(t, true)
	f.writeDoc(t, oddProvDoc(oddProvDelta))
	if _, err := f.run(t); err != nil {
		t.Fatalf("primera integración: %v", err)
	}
	workspacetest.WriteFile(t, f.specPath("facturacion"), oddFacturacionSpec+"\n### Requirement: Recargo (REQ-31.2)\n\nEl sistema DEBE recargar.\n")

	out, err := f.run(t, "--json")
	if err != nil {
		t.Fatalf("reintegración: %v", err)
	}
	a := decodeODDReport(t, out).Assignments
	if len(a) != 1 || a[0].State != livespec.StateReassigned || a[0].Definitive != "REQ-31.3" || len(a[0].Previous) != 1 || a[0].Previous[0] != "REQ-31.2" {
		t.Fatalf("assignments = %+v", a)
	}
	doc := readODDFile(t, f.docPath(oddFeature))
	if !strings.Contains(doc, "Cubre: REQ-31.3\n") || strings.Contains(doc, "Nota de abono (REQ-31.2)") {
		t.Errorf("el documento cita el nuevo número:\n%s", doc)
	}
}

func TestArchiveIntegrateCheckWritesNothingAndProjectsRequirementsAndTheMap(t *testing.T) {
	f := newODDFixture(t, true)
	workspacetest.WriteFile(t, filepath.Join(f.ws.Home, "INDEX.md"), "# Índice viejo\n")
	delta := oddProvDelta + "\n## MODIFIED Requirements\n\n### Requirement: Alta de factura (REQ-31.1)\n\nEl sistema DEBE dar de alta y validar una factura.\n"
	f.writeDoc(t, oddProvDoc(delta))
	before := oddSnapshot(t, f.root)

	out, err := f.run(t, "--check", "--json")
	if err != nil {
		t.Fatalf("--check: %v", err)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("--check no escribe nada: ni la spec, ni INDEX.md, ni el documento")
	}
	report := decodeODDReport(t, out)
	if !report.Check || report.Schema != livespec.ReportSchema || len(report.Written) != 0 {
		t.Errorf("informe = %+v", report)
	}
	if report.Requirements == nil || len(report.Requirements.Modifies) != 1 || report.Requirements.Modifies[0].ID != "REQ-31.1" ||
		len(report.Requirements.Creates) != 1 || report.Requirements.Creates[0].ID != "prov:31:1" || !report.Requirements.Creates[0].Provisional {
		t.Errorf("requirements = %+v", report.Requirements)
	}
	if len(report.Assignments) != 1 || report.Assignments[0].Provisional != "prov:31:1" || report.Assignments[0].Definitive != "REQ-31.2" ||
		report.Assignments[0].Capability != "facturacion" || report.Assignments[0].State != livespec.StateAssigned {
		t.Errorf("assignments = %+v", report.Assignments)
	}
}

func TestArchiveIntegrateCheckRejectsWithTheSameReasonAsTheRealRun(t *testing.T) {
	f := newODDFixture(t, true)
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", "## MODIFIED Requirements\n\n### Requirement: Inexistente (REQ-31.99)\n\nCuerpo.\n"})))
	before := oddSnapshot(t, f.root)

	_, checkErr := f.run(t, "--check")
	_, realErr := f.run(t)
	if checkErr == nil || realErr == nil || checkErr.Error() != realErr.Error() {
		t.Fatalf("--check %v\nreal %v\ndeben rechazar con el mismo motivo", checkErr, realErr)
	}
	var planErr *livespec.IDPlanError
	if !errors.As(checkErr, &planErr) || planErr.Code != livespec.CodeUnknownRequirementID {
		t.Errorf("error = %v, quería unknown_requirement_id", checkErr)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("un rechazo no escribe nada")
	}
}

func TestArchiveIntegrateCheckAppliesTheContextCheckAndLeavesTheIndexAlone(t *testing.T) {
	f := newODDContextFixture(t)
	workspacetest.WriteFile(t, filepath.Join(f.ws.Home, "INDEX.md"), "# Índice viejo\n")
	f.writeDoc(t, oddProvDoc(oddProvDelta))
	workspacetest.WriteFile(t, filepath.Join(f.root, "referencia", "ADR.md"), "# ADR\n\nEditado.\n")
	before := oddSnapshot(t, f.root)

	_, err := f.run(t, "--check")
	var refErr *livespec.ReferencePathError
	if !errors.As(err, &refErr) {
		t.Fatalf("error = %v, se esperaba *livespec.ReferencePathError", err)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("--check conserva los bytes de INDEX.md y de todo el árbol")
	}
}

func TestArchiveIntegrateCheckOnAnIntegratedDocumentSucceedsWithKeptEntries(t *testing.T) {
	f := newODDFixture(t, true)
	f.writeDoc(t, oddProvDoc(oddProvDelta))
	if _, err := f.run(t); err != nil {
		t.Fatalf("integración: %v", err)
	}
	before := oddSnapshot(t, f.root)

	out, err := f.run(t, "--check", "--json")
	if err != nil {
		t.Fatalf("--check: %v", err)
	}
	report := decodeODDReport(t, out)
	if !report.NothingToIntegrate || len(report.Assignments) != 1 || report.Assignments[0].State != livespec.StateKept ||
		report.Requirements == nil || len(report.Requirements.Creates) != 1 || report.Requirements.Creates[0].ID != "REQ-31.2" {
		t.Errorf("informe = %+v", report)
	}
	if !oddSameSnapshot(before, oddSnapshot(t, f.root)) {
		t.Error("--check no escribe nada")
	}
}

func TestArchiveIntegrateCheckSkipsOnlyTheBranchGuard(t *testing.T) {
	f := newODDFixture(t, false) // en main, donde la guarda de rama rechaza la integración real
	f.writeDoc(t, oddProvDoc(oddProvDelta))
	if _, err := f.run(t); err == nil {
		t.Fatal("la integración real debe negarse fuera de la rama de la feature")
	}
	if _, err := f.run(t, "--check"); err != nil {
		t.Errorf("--check omite la guarda de rama: %v", err)
	}
}

func TestArchiveIntegrateCheckDetectsADuplicateBetweenCapabilities(t *testing.T) {
	f := newODDFixture(t, true)
	workspacetest.WriteFile(t, f.specPath("pagos"), "# Especificación: pagos\n\n## Requirements\n\n### Requirement: Otro (REQ-31.1)\n\nOtro cuerpo.\n")
	f.writeDoc(t, oddDoc("", oddSpecSection([2]string{"facturacion", "## ADDED Requirements\n\n### Requirement: Alta de factura (REQ-31.1)\n\nEl sistema DEBE dar de alta una factura.\n"})))

	_, err := f.run(t, "--check")
	var planErr *livespec.IDPlanError
	if !errors.As(err, &planErr) || planErr.Code != livespec.CodeDuplicateAssignedID || planErr.Token != "REQ-31.1" {
		t.Fatalf("error = %v, quería duplicate_assigned_id de REQ-31.1", err)
	}
}

func TestArchiveIntegrateHelpDocumentsCheckAndTheDocumentRewrite(t *testing.T) {
	out, err := runODDIntegrate(t, "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--check", "prov:", "reescribe"} {
		if !strings.Contains(out, want) {
			t.Errorf("la ayuda debe contener %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "no modifica el documento ODD") {
		t.Errorf("la ayuda ya no dice que no modifica el documento ODD:\n%s", out)
	}
}
