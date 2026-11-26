package livespec

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

const acceptanceHeader = "| ID | Cubre | Rol | Tipo | Prioridad | Entornos | Precondiciones | Pasos | Resultado esperado |\r\n" +
	"|---|---|---|---|---|---|---|---|---|\r\n"

func acceptanceBattery(rows ...string) string {
	return "# Pruebas\n\nTexto con prov:30:1 fuera de la tabla.\n\n" + acceptanceHeader + strings.Join(rows, "") + "\nPie con prov:30:1.\n"
}

func acceptanceRow(id, cubre string) string {
	return "| " + id + " | " + cubre + " | fullstack | manual | alta | dev | prov:30:1 | paso prov:30:1 | ok |\r\n"
}

func acceptanceFiles(battery string) map[string]string {
	return map[string]string{
		"specs/pagos/spec.md":               pagosCanon,
		changeFile("pagos"):                 added(block("Cobro con IVA (prov:30:1)", "El sistema DEBE cobrar con IVA.")),
		"changes/mi-cambio/" + "pruebas.md": battery,
	}
}

func TestPlanChangeAcceptanceTestsRewritesOnlyTheCubreColumn(t *testing.T) {
	battery := acceptanceBattery(
		acceptanceRow("P-01", "prov:30:1#Cobro"),
		acceptanceRow("P-02", "prov:30:1"),
		acceptanceRow("P-03", "REQ-30.1#Alta ; prov:30:1#Otro"),
	)
	loc, reg := changeFixture(t, reqid.Default(), acceptanceFiles(battery))
	plan := planChange(t, loc, reg, ChangeOptions{})
	if _, err := plan.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got := readFile(t, filepath.Join(loc.PlanningHome, "changes", changeName, "pruebas.md"))
	want := strings.NewReplacer(
		"| P-01 | prov:30:1#Cobro |", "| P-01 | REQ-30.3#Cobro |",
		"| P-02 | prov:30:1 |", "| P-02 | REQ-30.3 |",
		"| P-03 | REQ-30.1#Alta ; prov:30:1#Otro |", "| P-03 | REQ-30.1#Alta ; REQ-30.3#Otro |",
	).Replace(battery)
	if got != want {
		t.Fatalf("pruebas.md:\n%q\nesperado:\n%q", got, want)
	}
	if !strings.Contains(got, "Texto con prov:30:1 fuera de la tabla.") || !strings.Contains(got, "Pie con prov:30:1.") || !strings.Contains(got, "\r\n") {
		t.Errorf("el texto fuera de la columna Cubre y los finales de línea no cambian:\n%q", got)
	}

	again, err := PlanChange(loc, changeName, ChangeOptions{}, reg)
	if err != nil {
		t.Fatalf("segunda ejecución: %v", err)
	}
	for _, f := range again.files {
		if filepath.Base(f.Path) == "pruebas.md" {
			t.Errorf("la reescritura debe ser idempotente: %s", f.Path)
		}
	}
}

func TestPlanChangeAcceptanceTestsLeavesDefinitiveIDsAndAbsentFileAlone(t *testing.T) {
	definitive := acceptanceBattery(acceptanceRow("P-01", "REQ-30.1#Alta"))
	loc, reg := changeFixture(t, reqid.Default(), acceptanceFiles(definitive))
	plan := planChange(t, loc, reg, ChangeOptions{})
	for _, f := range plan.files {
		if filepath.Base(f.Path) == "pruebas.md" {
			t.Errorf("un identificador definitivo no se toca: %s", f.Path)
		}
	}

	files := acceptanceFiles("")
	delete(files, "changes/mi-cambio/pruebas.md")
	loc, reg = changeFixture(t, reqid.Default(), files)
	plan = planChange(t, loc, reg, ChangeOptions{})
	for _, f := range plan.files {
		if filepath.Base(f.Path) == "pruebas.md" {
			t.Errorf("sin pruebas.md no hay escritura: %s", f.Path)
		}
	}
}

func TestPlanChangeAcceptanceTestsRejectsAnUndeclaredCitation(t *testing.T) {
	battery := acceptanceBattery(acceptanceRow("P-01", "prov:30:7#Cobro"))
	loc, reg := changeFixture(t, reqid.Default(), acceptanceFiles(battery))
	_, err := PlanChange(loc, changeName, ChangeOptions{}, reg)
	var planErr *IDPlanError
	if !errors.As(err, &planErr) || planErr.Code != CodeProvisionalUndeclared || planErr.Token != "prov:30:7" || !strings.HasSuffix(planErr.File, "pruebas.md") {
		t.Fatalf("error = %v", err)
	}
	if planErr.Line != 7 {
		t.Errorf("línea = %d, quiero la de la fila (7)", planErr.Line)
	}
}

func TestPlanChangeAcceptanceTestsInvalidTableProducesNoCitationsNorRewrite(t *testing.T) {
	invalid := "# Pruebas\n\n| ID | Cubre |\n|---|---|\n| P-01 | prov:30:7#X |\n"
	loc, reg := changeFixture(t, reqid.Default(), acceptanceFiles(invalid))
	plan := planChange(t, loc, reg, ChangeOptions{})
	for _, f := range plan.files {
		if filepath.Base(f.Path) == "pruebas.md" {
			t.Errorf("una tabla inválida no se reescribe: %s", f.Path)
		}
	}
}
