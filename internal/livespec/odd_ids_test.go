package livespec

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Las pruebas de la integración ODD con identificadores provisionales (REQ-30.2, REQ-30.3,
// REQ-47.4, REQ-47.5, REQ-47.6). Todos los textos son inventados.

const oddIDFacturacion = "# Especificación: facturacion\n\n## Requirements\n\n### Requirement: Alta de factura (REQ-31.1)\n\nEl sistema DEBE dar de alta una factura.\n"

// oddIDDoc es un documento ODD con un ADDED provisional con la serie 31 fijada, una línea
// `Cubre:` fuera de vallas y una cita en el resto del texto.
func oddIDDoc(provisional, title, body string) string {
	return "# ODD · mi-feature\n\n## Objetivo\n\nCita de " + provisional + " solo en el texto.\n\n## Tareas\n\n- [x] 1.1 Hecho\n  Cubre: " + provisional + "\n\n" +
		"## Spec viva\n\n" + capability("facturacion", fence("## ADDED Requirements\n\n### Requirement: "+title+" ("+provisional+")\n\n"+body+"\n"))
}

// oddIDProject monta el proyecto con la spec de facturación y el documento ODD.
func oddIDProject(t *testing.T, doc string) (loc workspace.Location, docPath string) {
	t.Helper()
	l := applyProject(t, map[string]string{"specs/facturacion/spec.md": oddIDFacturacion})
	path := filepath.Join(l.WorkspaceRoot, "odd", "tasks", "mi-feature.md")
	writeTree(t, l.WorkspaceRoot, map[string]string{"odd/tasks/mi-feature.md": doc})
	return l, path
}

// oddIDPlan lee el documento del disco y planifica la integración.
func oddIDPlan(t *testing.T, loc workspace.Location, docPath string) (Integration, error) {
	t.Helper()
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := ParseFeatureDoc("mi-feature", data)
	if err != nil {
		t.Fatalf("ParseFeatureDoc: %v", err)
	}
	reg, err := LoadRegistry(loc.PlanningHome, RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return PlanIntegration(loc, doc, reg)
}

func oddIDIntegrate(t *testing.T, loc workspace.Location, docPath string) Integration {
	t.Helper()
	plan, err := oddIDPlan(t, loc, docPath)
	if err != nil {
		t.Fatalf("PlanIntegration: %v", err)
	}
	if _, err := plan.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return plan
}

func TestODDDocRewriteAssignsTheNumberAndRewritesTheDocumentInTheSameSet(t *testing.T) {
	original := oddIDDoc("prov:31:1", "Nota de abono", "El sistema DEBE emitir notas de abono.")
	loc, docPath := oddIDProject(t, original)

	plan := oddIDIntegrate(t, loc, docPath)

	spec := readFile(t, filepath.Join(loc.PlanningHome, "specs", "facturacion", "spec.md"))
	if !strings.Contains(spec, "### Requirement: Nota de abono (REQ-31.2)") || strings.Contains(spec, "prov:") {
		t.Errorf("la spec viva debe llevar REQ-31.2 y ningún provisional:\n%s", spec)
	}
	got := readFile(t, docPath)
	if !strings.Contains(got, "### Requirement: Nota de abono (REQ-31.2)") || !strings.Contains(got, "  Cubre: REQ-31.2\n") {
		t.Errorf("la valla y la línea Cubre: deben citar el definitivo:\n%s", got)
	}
	if !strings.Contains(got, "Cita de prov:31:1 solo en el texto.") {
		t.Errorf("una cita fuera de la valla y de Cubre: no se reescribe:\n%s", got)
	}
	// El resto del documento conserva sus bytes: lo único que cambia es la valla, la línea
	// Cubre: y la sección de traza.
	rest := strings.ReplaceAll(strings.ReplaceAll(got, "REQ-31.2", "prov:31:1"), "Nota de abono (prov:31:1)", "Nota de abono (prov:31:1)")
	if !strings.HasPrefix(rest, original) {
		t.Errorf("el resto del documento cambió:\n%s", got)
	}
	trace, found, err := ExtractODDTrace(got, "doc")
	if err != nil || !found || len(trace.Assignments) != 1 {
		t.Fatalf("traza = %+v, found %v, err %v", trace, found, err)
	}
	if e := trace.Assignments[0]; e.Provisional != "prov:31:1" || e.Definitive != "REQ-31.2" || e.Capability != "facturacion" {
		t.Errorf("entrada = %+v", e)
	}
	if len(plan.Assignments) != 1 || plan.Assignments[0].State != StateAssigned {
		t.Errorf("assignments = %+v", plan.Assignments)
	}
	if plan.NothingToIntegrate() {
		t.Error("hay una spec que escribir")
	}
}

func TestODDDocRewriteLeavesADocumentWithoutProvisionalsUntouched(t *testing.T) {
	original := oddIDDoc("REQ-31.9", "Nota de abono", "El sistema DEBE emitir notas de abono.")
	loc, docPath := oddIDProject(t, original)
	oddIDIntegrate(t, loc, docPath)
	if got := readFile(t, docPath); got != original {
		t.Errorf("sin provisionales el documento conserva sus bytes:\n%s", got)
	}
}

func TestODDDocIgnoresProvisionalExamplesOutsideIdentifierPositions(t *testing.T) {
	doc := oddIDDoc("prov:31:1", "Nota de abono", "Cuerpo con el ejemplo `prov:40:1`.") +
		"\n## Notas\n\nVéase prov:31:7 y `prov::1`.\n\n```text\nCubre: prov:40:2\n```\n"
	loc, docPath := oddIDProject(t, doc)
	plan, err := oddIDPlan(t, loc, docPath)
	if err != nil {
		t.Fatalf("los ejemplos fuera de posiciones de identificador no se validan: %v", err)
	}
	if len(plan.Assignments) != 1 || plan.Assignments[0].Definitive != "REQ-31.2" {
		t.Fatalf("assignments = %+v", plan.Assignments)
	}
}

func TestODDDocRewriteRejectsAnUndeclaredProvisionalInACubreLine(t *testing.T) {
	doc := oddIDDoc("prov:31:1", "Nota de abono", "Cuerpo.") + "\n## Notas\n\n- [ ] 2.1 Otra\n  Cubre: prov:31:7\n"
	loc, docPath := oddIDProject(t, doc)
	_, err := oddIDPlan(t, loc, docPath)
	var planErr *IDPlanError
	if !errors.As(err, &planErr) || planErr.Code != CodeProvisionalUndeclared || planErr.Token != "prov:31:7" {
		t.Fatalf("error = %v, quería provisional_undeclared de prov:31:7", err)
	}
	if !strings.HasSuffix(planErr.File, "mi-feature.md") || planErr.Line == 0 {
		t.Errorf("el error debe nombrar el documento y la línea: %+v", planErr)
	}
	if got := readFile(t, docPath); got != doc {
		t.Error("un rechazo no escribe el documento")
	}
}

func TestODDDocRewriteKeepsTheLineOfTheProblemInsideTheFence(t *testing.T) {
	doc := oddIDDoc("prov:31:1", "Nota de abono", "Cuerpo.")
	doc = strings.Replace(doc, "### Requirement: Nota de abono (prov:31:1)", "### Requirement: Nota de abono (prov:31:1)\n\nTexto con prov:31:3 sin declarar.\n\n### Requirement: Otra (prov:31:2)", 1)
	loc, docPath := oddIDProject(t, doc)
	_, err := oddIDPlan(t, loc, docPath)
	var planErr *IDPlanError
	if !errors.As(err, &planErr) || planErr.Code != CodeProvisionalUndeclared {
		t.Fatalf("error = %v", err)
	}
	wantLine := 1
	for _, l := range strings.Split(doc, "\n") {
		if strings.Contains(l, "prov:31:3") {
			break
		}
		wantLine++
	}
	if planErr.Line != wantLine {
		t.Errorf("línea = %d, quería %d (la del documento, no la de la valla)", planErr.Line, wantLine)
	}
}

func TestODDDocRewriteIsAtomicWithTheDocumentAsLastDestination(t *testing.T) {
	loc, docPath := oddIDProject(t, oddIDDoc("prov:31:1", "Nota de abono", "Cuerpo."))
	before := snapshotTree(t, loc.WorkspaceRoot)
	plan, err := oddIDPlan(t, loc, docPath)
	if err != nil {
		t.Fatal(err)
	}
	original := renameFile
	t.Cleanup(func() { renameFile = original })
	renames := 0
	renameFile = func(from, to string) error {
		renames++
		if strings.HasSuffix(to, "mi-feature.md") {
			return errors.New("fallo inyectado")
		}
		return os.Rename(from, to)
	}
	if _, err := plan.Apply(); err == nil {
		t.Fatal("Apply debe fallar con el rename inyectado")
	}
	if renames < 2 {
		t.Fatalf("el documento debe escribirse en el mismo conjunto que la spec (renames %d)", renames)
	}
	if !sameTree(before, snapshotTree(t, loc.WorkspaceRoot)) {
		t.Error("el árbol debe quedar como estaba: spec y documento")
	}
}

// Dos ramas concurrentes con un ADDED provisional en la misma capacidad reciben números
// distintos al integrarse una tras otra (REQ-30.3, REQ-47.5).
func TestODDDocRewriteConcurrentBranchesGetDistinctNumbers(t *testing.T) {
	loc, docA := oddIDProject(t, oddIDDoc("prov:31:1", "Nota de abono", "Cuerpo de A."))
	oddIDIntegrate(t, loc, docA)

	docB := filepath.Join(loc.WorkspaceRoot, "odd", "tasks", "otra-feature.md")
	writeTree(t, loc.WorkspaceRoot, map[string]string{"odd/tasks/otra-feature.md": strings.ReplaceAll(oddIDDoc("prov:31:1", "Recargo", "Cuerpo de B."), "mi-feature", "otra-feature")})
	data, _ := os.ReadFile(docB)
	doc, err := ParseFeatureDoc("otra-feature", data)
	if err != nil {
		t.Fatal(err)
	}
	reg, _ := LoadRegistry(loc.PlanningHome, RegistryOptions{})
	plan, err := PlanIntegration(loc, doc, reg)
	if err != nil {
		t.Fatalf("PlanIntegration: %v", err)
	}
	if len(plan.Assignments) != 1 || plan.Assignments[0].Definitive != "REQ-31.3" {
		t.Fatalf("la segunda rama recibe REQ-31.3: %+v", plan.Assignments)
	}
}

// Tras un rebase en el que la base ocupó el definitivo con otro cuerpo, la reejecución
// reasigna con previous y reescribe documento y spec.
func TestODDDocRewriteRebaseReassignsWithPreviousAndRewritesEverything(t *testing.T) {
	loc, docPath := oddIDProject(t, oddIDDoc("prov:31:1", "Nota de abono", "Cuerpo de A."))
	oddIDIntegrate(t, loc, docPath)

	// La base ya traía otro requisito con REQ-31.2 y no tiene el bloque de esta rama.
	specPath := filepath.Join(loc.PlanningHome, "specs", "facturacion", "spec.md")
	rebased := oddIDFacturacion + "\n### Requirement: Recargo (REQ-31.2)\n\nCuerpo ajeno.\n"
	writeTree(t, loc.PlanningHome, map[string]string{"specs/facturacion/spec.md": rebased})

	plan, err := oddIDPlan(t, loc, docPath)
	if err != nil {
		t.Fatalf("PlanIntegration: %v", err)
	}
	if len(plan.Assignments) != 1 {
		t.Fatalf("assignments = %+v", plan.Assignments)
	}
	got := plan.Assignments[0]
	if got.State != StateReassigned || got.Definitive != "REQ-31.3" || len(got.Previous) != 1 || got.Previous[0] != "REQ-31.2" {
		t.Fatalf("entrada = %+v", got)
	}
	if _, err := plan.Apply(); err != nil {
		t.Fatal(err)
	}
	spec := readFile(t, specPath)
	if !strings.Contains(spec, "Recargo (REQ-31.2)") || !strings.Contains(spec, "Nota de abono (REQ-31.3)") {
		t.Errorf("la spec conserva lo ajeno y añade lo propio con el nuevo número:\n%s", spec)
	}
	doc := readFile(t, docPath)
	if !strings.Contains(doc, "Nota de abono (REQ-31.3)") || !strings.Contains(doc, "Cubre: REQ-31.3") || strings.Contains(doc, "Nota de abono (REQ-31.2)") {
		t.Errorf("el documento cita el nuevo número:\n%s", doc)
	}
	if !strings.Contains(doc, "previous:") || !strings.Contains(doc, "- REQ-31.2") {
		t.Errorf("la traza guarda el anterior:\n%s", doc)
	}
}

// Un documento ya integrado devuelve kept y no hay nada que integrar.
func TestODDDocRewriteAlreadyIntegratedIsKeptAndNothingToIntegrate(t *testing.T) {
	loc, docPath := oddIDProject(t, oddIDDoc("prov:31:1", "Nota de abono", "Cuerpo."))
	oddIDIntegrate(t, loc, docPath)
	before := snapshotTree(t, loc.WorkspaceRoot)

	plan := oddIDIntegrate(t, loc, docPath)
	if !plan.NothingToIntegrate() {
		t.Errorf("no había nada que integrar: %+v", plan)
	}
	if len(plan.Assignments) != 1 || plan.Assignments[0].State != StateKept || plan.Assignments[0].Definitive != "REQ-31.2" {
		t.Errorf("assignments = %+v", plan.Assignments)
	}
	if !sameTree(before, snapshotTree(t, loc.WorkspaceRoot)) {
		t.Error("repetir la integración no cambia ningún byte")
	}
}

// El título del bloque ya integrado cambió con el mismo cuerpo: se reconoce como integrado
// (REQ-30.3) y no vuelve a escribirse un segundo bloque.
func TestODDDocRewriteAChangedTitleWithTheSameBodyIsRecognisedAsIntegrated(t *testing.T) {
	loc, docPath := oddIDProject(t, oddIDDoc("prov:31:1", "Nota de abono", "Cuerpo."))
	oddIDIntegrate(t, loc, docPath)
	edited := strings.ReplaceAll(readFile(t, docPath), "Nota de abono (REQ-31.2)", "Nota de abono renombrada (REQ-31.2)")
	writeTree(t, loc.WorkspaceRoot, map[string]string{"odd/tasks/mi-feature.md": edited})
	specBefore := readFile(t, filepath.Join(loc.PlanningHome, "specs", "facturacion", "spec.md"))

	plan := oddIDIntegrate(t, loc, docPath)
	if got := readFile(t, filepath.Join(loc.PlanningHome, "specs", "facturacion", "spec.md")); got != specBefore {
		t.Errorf("la spec no cambia, y no se duplica el requisito:\n%s", got)
	}
	if len(plan.Assignments) != 1 || plan.Assignments[0].State != StateKept {
		t.Errorf("assignments = %+v", plan.Assignments)
	}
}

// Un duplicado entre capacidades que la integración escribe se rechaza sin escribir
// (REQ-47.5); uno ajeno solo se informa.
func TestODDDocRewriteDuplicatesBlockOnlyWhenTheIntegrationWritesThem(t *testing.T) {
	loc := applyProject(t, map[string]string{
		"specs/facturacion/spec.md": oddIDFacturacion,
		"specs/pagos/spec.md":       "# E\n\n## Requirements\n\n### Requirement: Uno (REQ-40.1)\n\nA.\n\n### Requirement: Dos (REQ-40.1)\n\nB.\n",
	})
	docPath := filepath.Join(loc.WorkspaceRoot, "odd", "tasks", "mi-feature.md")
	writeTree(t, loc.WorkspaceRoot, map[string]string{"odd/tasks/mi-feature.md": oddIDDoc("prov:31:1", "Nota de abono", "Cuerpo.")})
	plan, err := oddIDPlan(t, loc, docPath)
	if err != nil {
		t.Fatalf("un duplicado ajeno no bloquea: %v", err)
	}
	if len(plan.Duplicates) != 1 || plan.Duplicates[0].Blocking || plan.Duplicates[0].ID != "REQ-40.1" {
		t.Errorf("duplicates = %+v", plan.Duplicates)
	}
}
