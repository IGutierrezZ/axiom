package livespec

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestReportCarriesTheContractFields(t *testing.T) {
	loc := applyProject(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
	plan, written, err := planAndApply(t, loc, twoCapabilityDoc(t))
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(plan.Report(written))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	if got["schema"] != "axiom.odd-integration/v1" || got["feature"] != "mi-feature" || got["kind"] != "increment" || got["impact"] != "delta" || got["nothingToIntegrate"] != false {
		t.Errorf("informe = %v", got)
	}
	capabilities, _ := got["capabilities"].([]any)
	if len(capabilities) != 2 {
		t.Fatalf("capabilities = %v", got["capabilities"])
	}
	first, _ := capabilities[0].(map[string]any)
	if first["name"] != "facturacion" || first["path"] != "openspec/specs/facturacion/spec.md" || first["outcome"] != "composed" {
		t.Errorf("capacidad = %v", first)
	}
	second, _ := capabilities[1].(map[string]any)
	if second["outcome"] != "created" {
		t.Errorf("capacidad = %v", second)
	}
	if want := []any{"openspec/specs/facturacion/spec.md", "openspec/specs/pagos/spec.md"}; !reflect.DeepEqual(got["written"], want) {
		t.Errorf("written = %v, want %v", got["written"], want)
	}
	if _, present := got["context"]; present {
		t.Errorf("sin contexto técnico declarado el informe no lleva `context`: %v", got["context"])
	}
}

func TestReportNothingToIntegrateHasEmptyArraysNotNull(t *testing.T) {
	loc := applyProject(t, nil)
	reg, _ := LoadRegistry(loc.PlanningHome, RegistryOptions{})
	plan, err := PlanIntegration(loc, parseOK(t, featureDoc("Tipo: bug\n\n", "## Impacto en la spec\n\nSin impacto en la spec\nMotivo: error tipográfico\n")), reg)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(plan.Report(nil))
	var got map[string]any
	_ = json.Unmarshal(data, &got)

	if got["nothingToIntegrate"] != true || got["impact"] != "no_impact" || got["reason"] != "error tipográfico" || got["kind"] != "bug" {
		t.Errorf("informe = %v", got)
	}
	if capabilities, ok := got["capabilities"].([]any); !ok || len(capabilities) != 0 {
		t.Errorf("capabilities debe ser un arreglo vacío: %v", got["capabilities"])
	}
	if written, ok := got["written"].([]any); !ok || len(written) != 0 {
		t.Errorf("written debe ser un arreglo vacío: %v", got["written"])
	}
}

func TestReportKeepsAbsolutePathsOutsideTheWorkspaceUnchanged(t *testing.T) {
	plan := Integration{}
	outside := filepath.Join(t.TempDir(), "x", "spec.md")
	if got := plan.Report([]string{outside}).Written[0]; got != filepath.ToSlash(outside) {
		t.Errorf("Written = %q", got)
	}
}
