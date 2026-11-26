package openspecdoc

import (
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

const declaredDelta = `# Cobros

## ADDED Requirements

### Requirement: Cobro con IVA (prov:30:1)

El sistema MUST aplicar IVA.

### Requirement: Cobro en efectivo (REQ-30.9)

El sistema MUST cobrar en efectivo.

### Requirement: Cobro sin identificador

El sistema MUST cobrar.

## MODIFIED Requirements

### Requirement: Alta de cobro

El sistema MUST dar de alta.

### Requirement: Cobro con tarjeta (REQ-1.2)

El sistema MUST cobrar con tarjeta.

## REMOVED Requirements

### Requirement: Baja de cobro (REQ-1.3)

(Reason: obsoleto)

## RENAMED Requirements

### Requirement: Cobro viejo (REQ-1.4) → Cobro nuevo (REQ-1.4)

(Reason: claridad)
`

func TestDeclared(t *testing.T) {
	delta, fullSpec := ParseChangeSpec(declaredDelta)
	if fullSpec {
		t.Fatal("una delta con secciones no es una spec completa")
	}
	got := Declared("pagos", delta, reqid.Default())
	want := Declaration{
		Modifies: []DeclaredRequirement{
			{Capability: "pagos", Operation: "MODIFIED", Requirement: "Alta de cobro"},
			{Capability: "pagos", ID: "REQ-1.2", Operation: "MODIFIED", Requirement: "Cobro con tarjeta (REQ-1.2)"},
			{Capability: "pagos", ID: "REQ-1.3", Operation: "REMOVED", Requirement: "Baja de cobro (REQ-1.3)"},
			{Capability: "pagos", ID: "REQ-1.4", Operation: "RENAMED", Requirement: "Cobro viejo (REQ-1.4)"},
		},
		Creates: []DeclaredRequirement{
			{Capability: "pagos", ID: "prov:30:1", Provisional: true, Operation: "ADDED", Requirement: "Cobro con IVA (prov:30:1)"},
			{Capability: "pagos", ID: "REQ-30.9", Operation: "ADDED", Requirement: "Cobro en efectivo (REQ-30.9)"},
			{Capability: "pagos", Operation: "ADDED", Requirement: "Cobro sin identificador"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Declared =\n%+v\nquería\n%+v", got, want)
	}
	if again := Declared("pagos", delta, reqid.Default()); !reflect.DeepEqual(again, got) {
		t.Error("el orden no es estable")
	}
}

func TestDeclaredFullSpecCountsAsAdded(t *testing.T) {
	text := "# Nueva\n\n## Requirements\n\n### Requirement: Primero (prov:a:1)\n\nEl sistema MUST uno.\n\n### Requirement: Segundo (REQ-5.1)\n\nEl sistema MUST dos.\n"
	delta, fullSpec := ParseChangeSpec(text)
	if !fullSpec {
		t.Fatal("una spec sin secciones de delta es una spec completa")
	}
	got := Declared("nueva", delta, reqid.Default())
	if len(got.Modifies) != 0 || len(got.Creates) != 2 {
		t.Fatalf("Declared = %+v", got)
	}
	if got.Creates[0].ID != "prov:a:1" || !got.Creates[0].Provisional || got.Creates[1].ID != "REQ-5.1" || got.Creates[1].Operation != "ADDED" {
		t.Errorf("Creates = %+v", got.Creates)
	}
}

func TestParseChangeSpecEmpty(t *testing.T) {
	if d, full := ParseChangeSpec("# Solo texto\n\nsin requisitos\n"); !d.Empty() || full {
		t.Errorf("= %+v, %v", d, full)
	}
	if got := Declared("x", Delta{}, reqid.Default()); len(got.Modifies)+len(got.Creates) != 0 {
		t.Errorf("Declared vacío = %+v", got)
	}
}

func TestDeclaredIDPerPattern(t *testing.T) {
	p, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	text := "## ADDED Requirements\n\n### Requirement: A (prov:USU:1)\n\nx\n\n### Requirement: B (prov:30:1)\n\nx\n\n### Requirement: C (RF-USU-01)\n\nx\n"
	delta, _ := ParseChangeSpec(text)
	got := Declared("usu", delta, p).Creates
	if got[0].ID != "prov:USU:1" || !got[0].Provisional {
		t.Errorf("provisional válido para el patrón: %+v", got[0])
	}
	if got[1].ID != "" || got[1].Provisional {
		t.Errorf("un provisional que el patrón no admite no se proyecta como id: %+v", got[1])
	}
	if got[2].ID != "RF-USU-01" || got[2].Provisional {
		t.Errorf("definitivo del patrón: %+v", got[2])
	}
}
