package livespec

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Las pruebas del plan de integración del cambio SDD entero (REQ-47.8, REQ-30.22, REQ-21.17).
// Todos los textos son inventados.

const changeName = "mi-cambio"

// changeFixture monta el planning home con los ficheros dados (rutas bajo openspec/) y
// devuelve su Location y el registro sin el propio cambio.
func changeFixture(t *testing.T, pattern reqid.Pattern, files map[string]string) (workspace.Location, *Registry) {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "openspec")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTree(t, home, files)
	reg, err := LoadRegistry(home, RegistryOptions{Pattern: pattern, ExcludeChange: changeName})
	if err != nil {
		t.Fatal(err)
	}
	return workspace.Location{WorkspaceRoot: root, PlanningHome: home}, reg
}

func changeFile(capability string) string {
	return "changes/" + changeName + "/specs/" + capability + "/spec.md"
}

func planChange(t *testing.T, loc workspace.Location, reg *Registry, opts ChangeOptions) ChangeIntegration {
	t.Helper()
	plan, err := PlanChange(loc, changeName, opts, reg)
	if err != nil {
		t.Fatalf("PlanChange: %v", err)
	}
	return plan
}

func capabilityOf(t *testing.T, plan ChangeIntegration, name string) CapabilityPlan {
	t.Helper()
	for _, c := range plan.Capabilities {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("la capacidad %s no está en %+v", name, plan.Capabilities)
	return CapabilityPlan{}
}

func TestPlanChangeComposesAnExistingCapabilityAssigningTheNumberAndRewritingTheArtifacts(t *testing.T) {
	matrix := "# Informe\n\n## Matriz de verificación\n\n| Requisito | Escenario | Tipo | Referencia | Comando | Salida | Responsable | Fecha | Justificación | Resultado | Commit |\n|---|---|---|---|---|---|---|---|---|---|---|\n" +
		"| prov:30:1 | Cobro | unitario | `x_test.go` | `go test` | 0 | ana | 2026-11-26 | n/a | pass | abc1234 |\n\nTexto con prov:30:1 fuera de la matriz.\n"
	files := map[string]string{
		"specs/pagos/spec.md":                pagosCanon,
		changeFile("pagos"):                  added(block("Cobro con IVA (prov:30:1)", "El sistema DEBE cobrar con IVA.")),
		"changes/mi-cambio/tasks.md":         "- [ ] 1.1 Hacer\n  Cubre: prov:30:1\n\nMención libre de prov:30:1.\n",
		"changes/mi-cambio/verify-report.md": matrix,
	}
	loc, reg := changeFixture(t, reqid.Default(), files)
	plan := planChange(t, loc, reg, ChangeOptions{})

	got := capabilityOf(t, plan, "pagos")
	if got.Outcome != OutcomeComposed || !strings.Contains(got.Content, "### Requirement: Cobro con IVA (REQ-30.3)") || strings.Contains(got.Content, "prov:") {
		t.Fatalf("pagos = %+v", got)
	}
	if len(plan.Assignments) != 1 || plan.Assignments[0].Definitive != "REQ-30.3" || plan.Assignments[0].State != StateAssigned {
		t.Fatalf("assignments = %+v", plan.Assignments)
	}

	written, err := plan.Apply()
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	home := loc.PlanningHome
	if delta := readFile(t, filepath.Join(home, filepath.FromSlash(changeFile("pagos")))); !strings.Contains(delta, "(REQ-30.3)") || strings.Contains(delta, "prov:") {
		t.Errorf("la delta cita el definitivo:\n%s", delta)
	}
	tasks := readFile(t, filepath.Join(home, "changes", changeName, "tasks.md"))
	if !strings.Contains(tasks, "Cubre: REQ-30.3\n") || !strings.Contains(tasks, "Mención libre de prov:30:1.") {
		t.Errorf("solo la línea Cubre: se reescribe:\n%s", tasks)
	}
	report := readFile(t, filepath.Join(home, "changes", changeName, "verify-report.md"))
	if !strings.Contains(report, "| REQ-30.3 | Cobro |") || !strings.Contains(report, "Texto con prov:30:1 fuera de la matriz.") {
		t.Errorf("solo la columna Requisito de la matriz se reescribe:\n%s", report)
	}
	trace, err := ReadTraceFile(filepath.Join(home, "changes", changeName, "ids.yaml"))
	if err != nil || len(trace.Assignments) != 1 || trace.Assignments[0].Provisional != "prov:30:1" || trace.Assignments[0].Definitive != "REQ-30.3" || trace.Assignments[0].Capability != "pagos" {
		t.Errorf("traza = %+v, err %v", trace, err)
	}
	if len(written) != 5 {
		t.Errorf("written = %v (spec, delta, tasks, verify-report e ids.yaml)", written)
	}
	if spec := readFile(t, filepath.Join(home, "specs", "pagos", "spec.md")); !strings.Contains(spec, "(REQ-30.3)") {
		t.Errorf("la spec viva lleva el requisito:\n%s", spec)
	}
}

func TestPlanChangeNewCapabilityWithoutProvisionalsIsAByteForByteCopyAndWritesNoTrace(t *testing.T) {
	text := "# Especificación: nuevo\n\n## Requirements\n\n### Requirement: Alta (REQ-50.1)\n\nCuerpo.\n"
	loc, reg := changeFixture(t, reqid.Default(), map[string]string{changeFile("nuevo"): text})
	plan := planChange(t, loc, reg, ChangeOptions{})

	got := capabilityOf(t, plan, "nuevo")
	if got.Outcome != OutcomeCreated || got.Existed || got.Content != text {
		t.Fatalf("nuevo = %+v", got)
	}
	if _, err := plan.Apply(); err != nil {
		t.Fatal(err)
	}
	if readFile(t, filepath.Join(loc.PlanningHome, "specs", "nuevo", "spec.md")) != text {
		t.Error("la copia es la misma que hacía `cp`")
	}
	if _, err := os.Stat(filepath.Join(loc.PlanningHome, "changes", changeName, "ids.yaml")); !os.IsNotExist(err) {
		t.Errorf("sin provisionales no hay ids.yaml (%v)", err)
	}
	if !readSame(t, filepath.Join(loc.PlanningHome, "changes", changeName, filepath.FromSlash("specs/nuevo/spec.md")), text) {
		t.Error("la delta del cambio no cambia")
	}
}

func readSame(t *testing.T, path, want string) bool {
	t.Helper()
	return readFile(t, path) == want
}

func TestPlanChangeNewCapabilityOpensANewSeriesWithBothPatterns(t *testing.T) {
	t.Run("REQ-<n>.<m>", func(t *testing.T) {
		files := map[string]string{
			"specs/pagos/spec.md": "# E\n\n## Requirements\n\n### Requirement: Último (REQ-46.3)\n\nA.\n",
			changeFile("nuevo"):   "# Especificación: nuevo\n\n## Requirements\n\n### Requirement: Uno (prov:a:1)\n\nA.\n\n### Requirement: Dos (prov:a:2)\n\nB.\n",
		}
		loc, reg := changeFixture(t, reqid.Default(), files)
		got := capabilityOf(t, planChange(t, loc, reg, ChangeOptions{}), "nuevo")
		if got.Outcome != OutcomeCreated || !strings.Contains(got.Content, "Uno (REQ-47.1)") || !strings.Contains(got.Content, "Dos (REQ-47.2)") || strings.Contains(got.Content, "prov:") {
			t.Fatalf("nuevo = %+v", got)
		}
	})
	t.Run("RF-<DOM>-<NN>", func(t *testing.T) {
		pattern, err := reqid.Parse("RF-<DOM>-<NN>")
		if err != nil {
			t.Fatal(err)
		}
		files := map[string]string{changeFile("facturacion"): "# Especificación: facturacion\n\n## Requirements\n\n### Requirement: Alta (prov:FAC:1)\n\nA.\n"}
		loc, reg := changeFixture(t, pattern, files)
		got := capabilityOf(t, planChange(t, loc, reg, ChangeOptions{}), "facturacion")
		if !strings.Contains(got.Content, "Alta (RF-FAC-01)") {
			t.Fatalf("facturacion = %+v", got)
		}
	})
}

func TestPlanChangeResumesAPartiallyIntegratedCapability(t *testing.T) {
	canon := pagosCanon + "\n### Requirement: Alta (REQ-30.3)\n\nEl sistema DEBE dar de alta.\n"
	delta := added(block("Alta (REQ-30.3)", "El sistema DEBE dar de alta."), block("Baja (prov:30:1)", "El sistema DEBE dar de baja."))
	loc, reg := changeFixture(t, reqid.Default(), map[string]string{"specs/pagos/spec.md": canon, changeFile("pagos"): delta})
	got := capabilityOf(t, planChange(t, loc, reg, ChangeOptions{}), "pagos")
	if got.Outcome != OutcomeResumed || !strings.Contains(got.Content, "Baja (REQ-30.4)") || strings.Count(got.Content, "(REQ-30.3)") != 1 {
		t.Fatalf("pagos = %+v", got)
	}
	if len(got.Skipped) != 1 || got.Skipped[0].Section != "ADDED" || got.Skipped[0].Requirement != "Alta (REQ-30.3)" {
		t.Errorf("skipped = %+v", got.Skipped)
	}
}

func TestPlanChangeAnAlreadyIntegratedCapabilityIsNotWritten(t *testing.T) {
	canon := pagosCanon + "\n### Requirement: Alta (REQ-30.3)\n\nEl sistema DEBE dar de alta.\n"
	loc, reg := changeFixture(t, reqid.Default(), map[string]string{"specs/pagos/spec.md": canon, changeFile("pagos"): added(block("Alta (REQ-30.3)", "El sistema DEBE dar de alta."))})
	plan := planChange(t, loc, reg, ChangeOptions{})
	got := capabilityOf(t, plan, "pagos")
	if got.Outcome != OutcomeAlready || got.Content != "" || !plan.NothingToIntegrate() {
		t.Fatalf("pagos = %+v, nada que integrar = %v", got, plan.NothingToIntegrate())
	}
}

func TestPlanChangeRejectsANewCapabilityWithModifiedRemovedOrRenamed(t *testing.T) {
	delta := "## MODIFIED Requirements\n\n" + block("Cobro (REQ-30.1)", "Cuerpo.")
	loc, reg := changeFixture(t, reqid.Default(), map[string]string{changeFile("pagos"): delta})
	_, err := PlanChange(loc, changeName, ChangeOptions{}, reg)
	var missing *MissingCapabilityError
	if !errors.As(err, &missing) || missing.Capability != "pagos" || missing.Section != "MODIFIED" {
		t.Fatalf("error = %v, quería *MissingCapabilityError", err)
	}
}

func TestPlanChangeRejectsAFlatDelta(t *testing.T) {
	loc, reg := changeFixture(t, reqid.Default(), map[string]string{"changes/mi-cambio/specs/spec.md": added(block("Alta (REQ-30.3)", "A."))})
	_, err := PlanChange(loc, changeName, ChangeOptions{}, reg)
	wantPlanError(t, err, CodeFlatDeltaUnsupported)
}

func TestPlanChangeRejectsAnUndeclaredCitationInTheTasks(t *testing.T) {
	files := map[string]string{
		"specs/pagos/spec.md":        pagosCanon,
		changeFile("pagos"):          added(block("Alta (prov:30:1)", "A.")),
		"changes/mi-cambio/tasks.md": "- [ ] 1.1\n  Cubre: prov:30:7\n",
	}
	loc, reg := changeFixture(t, reqid.Default(), files)
	_, err := PlanChange(loc, changeName, ChangeOptions{}, reg)
	var planErr *IDPlanError
	if !errors.As(err, &planErr) || planErr.Code != CodeProvisionalUndeclared || planErr.Token != "prov:30:7" || !strings.HasSuffix(planErr.File, "tasks.md") || planErr.Line != 2 {
		t.Fatalf("error = %v", err)
	}
}

// La paridad con `archive-compose`: para toda composición aceptada hasta hoy, el contenido
// es el de openspecdoc.Compose con la delta entera.
func TestPlanChangeParityWithArchiveComposeForAcceptedDeltas(t *testing.T) {
	deltas := map[string]string{
		"MODIFIED":   "## MODIFIED Requirements\n\n" + block("Cobro (REQ-30.1)", "El sistema DEBE cobrar con IVA."),
		"REMOVED":    "## REMOVED Requirements\n\n### Requirement: Reembolso (REQ-30.2)\n\n(Reason: obsoleto)\n",
		"RENAMED":    "## RENAMED Requirements\n\n### Requirement: Cobro (REQ-30.1) → Cobro total (REQ-30.1)\n\n(Reason: nombre)\n",
		"ADDED+MOD":  added(block("Anulación (REQ-30.9)", "El sistema DEBE anular.")) + "\n## MODIFIED Requirements\n\n" + block("Cobro (REQ-30.1)", "Nuevo cuerpo."),
		"sin cambio": "## MODIFIED Requirements\n\n" + block("Cobro (REQ-30.1)", "El sistema DEBE cobrar."),
	}
	for name, delta := range deltas {
		for _, supersede := range []bool{false, true} {
			t.Run(name, func(t *testing.T) {
				loc, reg := changeFixture(t, reqid.Default(), map[string]string{"specs/pagos/spec.md": pagosCanon, changeFile("pagos"): delta})
				want, composeErr := openspecdoc.Compose(pagosCanon, delta, openspecdoc.ComposeOptions{SupersedeRemoved: supersede, IDPattern: reqid.Default()})
				plan, err := PlanChange(loc, changeName, ChangeOptions{Supersede: supersede}, reg)
				if composeErr != nil {
					t.Skipf("Compose rechaza esta delta (%v): no es una composición aceptada hoy", composeErr)
				}
				if err != nil {
					t.Fatalf("PlanChange: %v", err)
				}
				got := capabilityOf(t, plan, "pagos")
				if got.Outcome == OutcomeAlready {
					if name != "sin cambio" {
						t.Errorf("outcome = %s", got.Outcome)
					}
					return
				}
				if got.Outcome != OutcomeComposed || got.Content != want {
					t.Errorf("la composición difiere de archive-compose:\n%s\nquería\n%s", got.Content, want)
				}
			})
		}
	}
}

func TestPlanChangeSupersedeAppliesToTheRemovedOfEveryCapability(t *testing.T) {
	files := map[string]string{
		"specs/pagos/spec.md":       pagosCanon,
		"specs/facturacion/spec.md": "# E\n\n## Requirements\n\n### Requirement: Alta de factura (REQ-31.1)\n\nA.\n",
		changeFile("pagos"):         "## REMOVED Requirements\n\n### Requirement: Reembolso (REQ-30.2)\n\n(Reason: x)\n",
		changeFile("facturacion"):   "## REMOVED Requirements\n\n### Requirement: Alta de factura (REQ-31.1)\n\n(Reason: x)\n",
	}
	loc, reg := changeFixture(t, reqid.Default(), files)
	plan := planChange(t, loc, reg, ChangeOptions{Supersede: true})
	for _, name := range []string{"pagos", "facturacion"} {
		if got := capabilityOf(t, plan, name); !strings.Contains(got.Content, "[SUPERSEDED / DEPRECADO]") {
			t.Errorf("%s debe marcar el requisito retirado como superado:\n%s", name, got.Content)
		}
	}
	plain := planChange(t, loc, reg, ChangeOptions{})
	if got := capabilityOf(t, plain, "pagos"); strings.Contains(got.Content, "SUPERSEDED") || strings.Contains(got.Content, "Reembolso") {
		t.Errorf("sin --supersede el requisito se borra:\n%s", got.Content)
	}
}

func TestPlanChangeSupersededRequirementNamesWithAndWithoutPrefix(t *testing.T) {
	files := map[string]string{
		"specs/pagos/spec.md":       pagosCanon,
		"specs/facturacion/spec.md": "# E\n\n## Requirements\n\n### Requirement: Alta de factura (REQ-31.1)\n\nA.\n\n### Requirement: Cobro (REQ-31.2)\n\nB.\n",
		changeFile("pagos"):         added(block("Anulación (REQ-30.9)", "X.")),
		changeFile("facturacion"):   added(block("Baja (REQ-31.9)", "Y.")),
	}
	loc, reg := changeFixture(t, reqid.Default(), files)

	t.Run("sin prefijo, en una sola capacidad", func(t *testing.T) {
		plan := planChange(t, loc, reg, ChangeOptions{Superseded: []string{"Reembolso (REQ-30.2)"}})
		if got := capabilityOf(t, plan, "pagos"); !strings.Contains(got.Content, "[SUPERSEDED / DEPRECADO]") {
			t.Errorf("pagos:\n%s", got.Content)
		}
	})
	t.Run("con prefijo", func(t *testing.T) {
		plan := planChange(t, loc, reg, ChangeOptions{Superseded: []string{"facturacion/Cobro (REQ-31.2)"}})
		if got := capabilityOf(t, plan, "facturacion"); !strings.Contains(got.Content, "[SUPERSEDED / DEPRECADO]") {
			t.Errorf("facturacion:\n%s", got.Content)
		}
	})
	t.Run("inexistente", func(t *testing.T) {
		_, err := PlanChange(loc, changeName, ChangeOptions{Superseded: []string{"Nada"}}, reg)
		wantPlanError(t, err, CodeSupersededRequirementUnknown)
	})
	t.Run("ambiguo con la continuación del prefijo", func(t *testing.T) {
		both := changeFixtureWith(t, map[string]string{
			"specs/pagos/spec.md":       "# E\n\n## Requirements\n\n### Requirement: Cobro\n\nA.\n",
			"specs/facturacion/spec.md": "# E\n\n## Requirements\n\n### Requirement: Cobro\n\nB.\n",
			changeFile("pagos"):         added(block("Anulación (REQ-30.9)", "X.")),
			changeFile("facturacion"):   added(block("Baja (REQ-31.9)", "Y.")),
		})
		_, err := PlanChange(both.loc, changeName, ChangeOptions{Superseded: []string{"Cobro"}}, both.reg)
		wantPlanError(t, err, CodeSupersededRequirementAmbiguous)
		if !strings.Contains(err.Error(), "facturacion/Cobro") || !strings.Contains(err.Error(), "pagos/Cobro") {
			t.Errorf("la continuación nombra el prefijo: %v", err)
		}
	})
}

type fixtureResult struct {
	loc workspace.Location
	reg *Registry
}

func changeFixtureWith(t *testing.T, files map[string]string) fixtureResult {
	t.Helper()
	loc, reg := changeFixture(t, reqid.Default(), files)
	return fixtureResult{loc: loc, reg: reg}
}

func TestPlanChangeApplyIsAtomicAndIdempotent(t *testing.T) {
	files := map[string]string{
		"specs/pagos/spec.md":       pagosCanon,
		"specs/facturacion/spec.md": "# E\n\n## Requirements\n\n### Requirement: Alta de factura (REQ-31.1)\n\nA.\n",
		changeFile("pagos"):         added(block("Cobro con IVA (prov:30:1)", "El sistema DEBE cobrar con IVA.")),
		changeFile("facturacion"):   added(block("Nota (prov:31:1)", "El sistema DEBE emitir notas.")),
	}
	loc, reg := changeFixture(t, reqid.Default(), files)
	before := snapshotTree(t, loc.WorkspaceRoot)
	plan := planChange(t, loc, reg, ChangeOptions{})

	original := renameFile
	t.Cleanup(func() { renameFile = original })
	renames := 0
	renameFile = func(from, to string) error {
		renames++
		if renames == 4 {
			return errors.New("fallo inyectado")
		}
		return os.Rename(from, to)
	}
	if _, err := plan.Apply(); err == nil {
		t.Fatal("Apply debe fallar con el rename inyectado")
	}
	if !sameTree(before, snapshotTree(t, loc.WorkspaceRoot)) {
		t.Error("un fallo en cualquier escritura no deja nada a medias")
	}
	renameFile = original

	if _, err := plan.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	after := snapshotTree(t, loc.WorkspaceRoot)
	reg2, err := LoadRegistry(loc.PlanningHome, RegistryOptions{ExcludeChange: changeName})
	if err != nil {
		t.Fatal(err)
	}
	again := planChange(t, loc, reg2, ChangeOptions{})
	if !again.NothingToIntegrate() {
		t.Fatalf("la reejecución no tiene nada que integrar: %+v", again.Capabilities)
	}
	for _, e := range again.Assignments {
		if e.State != StateKept {
			t.Errorf("entrada %+v debe ser kept", e)
		}
	}
	if _, err := again.Apply(); err != nil || !sameTree(after, snapshotTree(t, loc.WorkspaceRoot)) {
		t.Errorf("repetir no cambia ningún byte (err %v)", err)
	}
}

func TestPlanChangeTwoChangesInARowGetDistinctNumbers(t *testing.T) {
	loc, reg := changeFixture(t, reqid.Default(), map[string]string{
		"specs/pagos/spec.md": pagosCanon,
		changeFile("pagos"):   added(block("Uno (prov:30:1)", "A.")),
	})
	plan := planChange(t, loc, reg, ChangeOptions{})
	if _, err := plan.Apply(); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(loc.PlanningHome, "changes", "otro-cambio", "specs", "pagos", "spec.md")
	writeTree(t, loc.PlanningHome, map[string]string{"changes/otro-cambio/specs/pagos/spec.md": added(block("Dos (prov:30:1)", "B."))})
	_ = other
	reg2, err := LoadRegistry(loc.PlanningHome, RegistryOptions{ExcludeChange: "otro-cambio"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanChange(loc, "otro-cambio", ChangeOptions{}, reg2)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Assignments) != 1 || second.Assignments[0].Definitive != "REQ-30.4" {
		t.Errorf("el segundo cambio recibe REQ-30.4: %+v", second.Assignments)
	}
}

func TestCheckArchivedTraceReportsADefinitiveThatAppearsMoreThanOnce(t *testing.T) {
	trace := Trace{Schema: traceSchema, Pattern: "REQ-<n>.<m>", Assignments: []TraceEntry{{Provisional: "prov:30:1", Definitive: "REQ-30.3", Capability: "pagos", Requirement: "Cobro (REQ-30.3)"}}}
	t.Run("una sola vez en su capacidad", func(t *testing.T) {
		loc, _ := changeFixture(t, reqid.Default(), map[string]string{"specs/pagos/spec.md": pagosCanon + "\n### Requirement: Cobro (REQ-30.3)\n\nA.\n"})
		reg, _ := LoadRegistry(loc.PlanningHome, RegistryOptions{SpecsOnly: true})
		if err := CheckArchivedTrace(reg, trace); err != nil {
			t.Errorf("CheckArchivedTrace: %v", err)
		}
	})
	t.Run("repetido en otra capacidad", func(t *testing.T) {
		loc, _ := changeFixture(t, reqid.Default(), map[string]string{
			"specs/pagos/spec.md":       pagosCanon + "\n### Requirement: Cobro (REQ-30.3)\n\nA.\n",
			"specs/facturacion/spec.md": "# E\n\n## Requirements\n\n### Requirement: Ajeno (REQ-30.3)\n\nB.\n",
		})
		reg, _ := LoadRegistry(loc.PlanningHome, RegistryOptions{SpecsOnly: true})
		err := CheckArchivedTrace(reg, trace)
		wantPlanError(t, err, CodeDuplicateAssignedID)
		if !strings.Contains(err.Error(), "axiom branch done --force") {
			t.Errorf("la continuación de D5: %v", err)
		}
	})
	t.Run("ausente de la spec viva", func(t *testing.T) {
		loc, _ := changeFixture(t, reqid.Default(), map[string]string{"specs/pagos/spec.md": pagosCanon})
		reg, _ := LoadRegistry(loc.PlanningHome, RegistryOptions{SpecsOnly: true})
		wantPlanError(t, CheckArchivedTrace(reg, trace), CodeDuplicateAssignedID)
	})
}
