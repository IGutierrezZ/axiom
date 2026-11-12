package openspecdoc

import (
	"strings"
	"testing"
)

func TestRenderNewSpecWritesTheDocumentSkeleton(t *testing.T) {
	added := ParseDelta("## ADDED Requirements\n\n### Requirement: Alta (REQ-31.1)\n\nEl sistema DEBE dar de alta.\n\n#### Scenario: Alta\n\n- **DADO** un formulario\n\n### Requirement: Baja (REQ-31.2)\n\nEl sistema DEBE dar de baja.\n").Added

	got := RenderNewSpec("facturacion", "mi-feature", added)

	want := "# Especificación: facturacion\n\n## Propósito\n\nIntegrada desde el documento ODD `mi-feature`.\n\n## Requirements\n\n" +
		"### Requirement: Alta (REQ-31.1)\n\nEl sistema DEBE dar de alta.\n\n#### Scenario: Alta\n\n- **DADO** un formulario\n\n" +
		"### Requirement: Baja (REQ-31.2)\n\nEl sistema DEBE dar de baja.\n"
	if got != want {
		t.Errorf("RenderNewSpec:\n%q\nwant\n%q", got, want)
	}
}

func TestRenderNewSpecIsComposableAndItsRequirementsAreAlreadyPresent(t *testing.T) {
	added := ParseDelta("## ADDED Requirements\n\n### Requirement: Alta (REQ-31.1)\n\nCuerpo.\n\n").Added
	rendered := RenderNewSpec("facturacion", "mi-feature", added)

	parsed := ParseCanonical(rendered)
	if len(parsed) != 1 || parsed[0].ID != "REQ-31.1" {
		t.Fatalf("ParseCanonical(rendered) = %+v", parsed)
	}
	if NormalizeBody(parsed[0].Body) != NormalizeBody(added[0].Body) {
		t.Errorf("el cuerpo integrado difiere del de la delta:\n%q\n%q", parsed[0].Body, added[0].Body)
	}
	composed, err := Compose(rendered, "## ADDED Requirements\n\n### Requirement: Baja (REQ-31.2)\n\nOtro.\n", ComposeOptions{})
	if err != nil || !strings.Contains(composed, "Baja (REQ-31.2)") {
		t.Errorf("la especificación nueva debe poder componerse después: %v", err)
	}
}

func TestRenderNewSpecEndsWithASingleNewline(t *testing.T) {
	added := ParseDelta("## ADDED Requirements\n\n### Requirement: A (REQ-1.1)\n\nx\n\n\n\n").Added
	got := RenderNewSpec("c", "f", added)
	if !strings.HasSuffix(got, "x\n") || strings.HasSuffix(got, "\n\n") {
		t.Errorf("el documento termina con un solo salto de línea: %q", got)
	}
}
