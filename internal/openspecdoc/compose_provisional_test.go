package openspecdoc

import (
	"errors"
	"strings"
	"testing"
)

const addedBody = "\nEl sistema MUST algo.\n\n#### Scenario: S\n\n- GIVEN a\n- WHEN b\n- THEN c\n"

// REQ-47.3: un MODIFIED, un REMOVED o el nombre antiguo de un RENAMED con un ID válido
// que la canónica no lleva se rechazan con el error tipado, que sigue siendo un
// *UnappliedDeltaError para quien ya lo trata con errors.As.
func TestComposeUnknownRequirementID(t *testing.T) {
	cases := []struct {
		name, delta, section, requirement string
	}{
		{"MODIFIED", "## MODIFIED Requirements\n\n### Requirement: Cobro fantasma (REQ-9.9)\n" + addedBody, "MODIFIED", "Cobro fantasma (REQ-9.9)"},
		{"REMOVED", "## REMOVED Requirements\n\n### Requirement: Cobro fantasma (REQ-9.9)\n\n(Reason: obsoleto)\n", "REMOVED", "Cobro fantasma (REQ-9.9)"},
		{"RENAMED", "## RENAMED Requirements\n\n### Requirement: Cobro fantasma (REQ-9.9) → Cobro nuevo (REQ-9.9)\n\n(Reason: x)\n", "RENAMED", "Cobro fantasma (REQ-9.9) → Cobro nuevo (REQ-9.9)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compose(composeByIDCanonical, tc.delta, ComposeOptions{})
			var unknown *UnknownRequirementIDError
			if !errors.As(err, &unknown) {
				t.Fatalf("error = %v, quería *UnknownRequirementIDError", err)
			}
			if unknown.Section != tc.section || unknown.Requirement != tc.requirement || unknown.ID != "REQ-9.9" {
				t.Errorf("error = %+v", unknown)
			}
			var unapplied *UnappliedDeltaError
			if !errors.As(err, &unapplied) || unapplied.Section != tc.section || unapplied.Requirement != tc.requirement {
				t.Errorf("errors.As al *UnappliedDeltaError: %v", err)
			}
			if !strings.Contains(err.Error(), "REQ-9.9") {
				t.Errorf("el mensaje nombra el identificador: %v", err)
			}
		})
	}
}

// REQ-47.3 regla 2: sin ID, un MODIFIED se resuelve por título y conserva el rechazo
// vigente (un *UnappliedDeltaError simple) cuando no casa.
func TestComposeModifiedWithoutIDKeepsTheCurrentRejection(t *testing.T) {
	delta := "## MODIFIED Requirements\n\n### Requirement: Cobro fantasma\n" + addedBody
	_, err := Compose(composeByIDCanonical, delta, ComposeOptions{})
	var unknown *UnknownRequirementIDError
	if errors.As(err, &unknown) {
		t.Fatalf("sin ID no es un identificador inexistente: %v", err)
	}
	var unapplied *UnappliedDeltaError
	if !errors.As(err, &unapplied) || unapplied.Section != "MODIFIED" {
		t.Fatalf("error = %v", err)
	}
	ok := "## MODIFIED Requirements\n\n### Requirement: Alta de cobro (REQ-1.1)\n" + addedBody
	if _, err := Compose(composeByIDCanonical, ok, ComposeOptions{}); err != nil {
		t.Errorf("con ID que existe se compone: %v", err)
	}
}

// REQ-30.22: Compose rechaza un provisional en el nombre y en el cuerpo de un requisito
// de cualquier sección, antes de aplicar nada.
func TestComposeRejectsProvisionals(t *testing.T) {
	cases := []struct {
		name, delta, section, requirement, token string
	}{
		{"nombre en ADDED", "## ADDED Requirements\n\n### Requirement: Nuevo (prov:1:1)\n" + addedBody, "ADDED", "Nuevo (prov:1:1)", "prov:1:1"},
		{"cuerpo en ADDED", "## ADDED Requirements\n\n### Requirement: Nuevo (REQ-1.4)\n\nEl sistema MUST cumplir prov:1:2.\n", "ADDED", "Nuevo (REQ-1.4)", "prov:1:2"},
		{"cuerpo en MODIFIED", "## MODIFIED Requirements\n\n### Requirement: Alta de cobro (REQ-1.1)\n\nVéase prov:a:3.\n", "MODIFIED", "Alta de cobro (REQ-1.1)", "prov:a:3"},
		{"nombre en MODIFIED", "## MODIFIED Requirements\n\n### Requirement: Alta (prov:1:1)\n" + addedBody, "MODIFIED", "Alta (prov:1:1)", "prov:1:1"},
		{"cuerpo en REMOVED", "## REMOVED Requirements\n\n### Requirement: Baja de cobro (REQ-1.3)\n\n(Reason: sustituido por prov:1:1)\n", "REMOVED", "Baja de cobro (REQ-1.3)", "prov:1:1"},
		{"nombre en RENAMED", "## RENAMED Requirements\n\n### Requirement: Alta de cobro (REQ-1.1) → Alta (prov:1:1)\n\n(Reason: x)\n", "RENAMED", "Alta de cobro (REQ-1.1) → Alta (prov:1:1)", "prov:1:1"},
		{"forma laxa mal formada", "## ADDED Requirements\n\n### Requirement: Nuevo (REQ-1.4)\n\nEl sistema MUST ver prov:a1B:1.\n", "ADDED", "Nuevo (REQ-1.4)", "prov:a1B:1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Compose(composeByIDCanonical, tc.delta, ComposeOptions{})
			var pe *ProvisionalIDError
			if !errors.As(err, &pe) {
				t.Fatalf("error = %v, quería *ProvisionalIDError", err)
			}
			if pe.Section != tc.section || pe.Requirement != tc.requirement || pe.Token != tc.token {
				t.Errorf("error = %+v", pe)
			}
			if out != "" {
				t.Errorf("no compone nada: %q", out)
			}
			if !strings.Contains(err.Error(), tc.token) {
				t.Errorf("el mensaje nombra el token: %v", err)
			}
		})
	}
}

func TestComposeProvisionalTextNotATokenIsAccepted(t *testing.T) {
	delta := "## ADDED Requirements\n\n### Requirement: Nuevo (REQ-1.4)\n\nEl formato es prov:<serie>:<ordinal> y un provider:30:1 no cuenta.\n"
	if _, err := Compose(composeByIDCanonical, delta, ComposeOptions{}); err != nil {
		t.Errorf("la forma con ángulos y una palabra distinta no son un provisional: %v", err)
	}
}

// Un nombre con dos definitivos y ningún provisional conserva la regla actual (último ID).
func TestComposeNameWithTwoDefinitivesKeepsTheCurrentRule(t *testing.T) {
	delta := "## MODIFIED Requirements\n\n### Requirement: Alta (REQ-9.9) (REQ-1.1)\n" + addedBody
	if _, err := Compose(composeByIDCanonical, delta, ComposeOptions{}); err != nil {
		t.Errorf("el último ID del título resuelve: %v", err)
	}
}
