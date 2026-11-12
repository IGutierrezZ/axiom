package dashboard

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
)

// REQ-30.1: la sección `## Spec viva` NO DEBE alterar el listado ni el recuento de
// casillas del documento ODD. El listado cuenta las casillas también dentro de vallas, y
// por eso ParseFeatureDoc rechaza una casilla dentro de la delta.

const oddWithoutSpecSection = `# ODD · facturacion

## Objetivo

Emitir facturas.

## Tareas

- [x] 1.1 Modelo
- [ ] 1.2 Servicio
- [ ] 1.3 Pantalla

## Cierre

Pendiente.
`

const oddSpecSection = "\n## Spec viva\n\n### Capacidad: facturacion\n\n```openspec\n" +
	"## ADDED Requirements\n\n### Requirement: Alta de factura (REQ-31.1)\n\nEl sistema DEBE dar de alta una factura.\n\n" +
	"#### Scenario: Alta correcta\n\n- **DADO** un formulario válido\n- **CUANDO** se envía\n- **ENTONCES** se crea la factura\n```\n"

func TestParseODDDocCountsTheSameWithAndWithoutTheSpecSection(t *testing.T) {
	without := parseODDDoc(oddWithoutSpecSection)
	with := parseODDDoc(oddWithoutSpecSection + oddSpecSection)

	if without.Total != 3 || without.Completed != 1 {
		t.Fatalf("sin sección: Total=%d Completed=%d, se esperaba 3 y 1", without.Total, without.Completed)
	}
	if with.Total != without.Total || with.Completed != without.Completed || with.Title != without.Title {
		t.Errorf("con sección = %+v, sin sección = %+v: el recuento no debe cambiar", with, without)
	}
	if _, err := livespec.ParseFeatureDoc("facturacion", []byte(oddWithoutSpecSection+oddSpecSection)); err != nil {
		t.Errorf("el documento de la prueba debe ser válido para la spec viva: %v", err)
	}
}

func TestParseODDDocCountsCheckboxesInsideFencesSoTheDeltaMayNotHaveThem(t *testing.T) {
	withCheckbox := oddWithoutSpecSection + strings.Replace(oddSpecSection, "- **DADO** un formulario válido", "- [ ] un formulario válido", 1)

	if got := parseODDDoc(withCheckbox); got.Total != 4 {
		t.Fatalf("Total = %d: el listado cuenta la casilla de la valla (premisa de la regla checkbox_in_delta)", got.Total)
	}
	_, err := livespec.ParseFeatureDoc("facturacion", []byte(withCheckbox))
	if err == nil || !strings.Contains(err.Error(), "checkbox_in_delta") {
		t.Errorf("ParseFeatureDoc debe rechazar la casilla de la delta: %v", err)
	}
}
