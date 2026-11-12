package sddstatus

import (
	"errors"
	"strings"
	"testing"
)

// El análisis y la composición viven en internal/openspecdoc, con sus pruebas. Aquí
// solo se fija que el envoltorio conserva la API y el tipo de error de sddstatus
// (REQ-30.22 a REQ-30.24): la CLI `sdd archive-compose` depende de ellos.

const composeWrapperCanonical = `# Widgets Specification

## Requirements

### Requirement: Widget Expiration

The system MUST expire widgets after 30 days.

#### Scenario: Expire after 30 days

- GIVEN a widget older than 30 days
- THEN the widget is removed
`

func TestComposeOpenSpecCanonicalSpecWrapperDelegatesToTheLeafPackage(t *testing.T) {
	delta := "## ADDED Requirements\n\n### Requirement: Widget Tagging\n\nThe system MUST tag widgets.\n"

	plain, err := ComposeOpenSpecCanonicalSpec(composeWrapperCanonical, delta)
	if err != nil {
		t.Fatalf("ComposeOpenSpecCanonicalSpec error = %v", err)
	}
	withOptions, err := ComposeOpenSpecCanonicalSpecWithOptions(composeWrapperCanonical, delta, ComposeOptions{})
	if err != nil {
		t.Fatalf("ComposeOpenSpecCanonicalSpecWithOptions error = %v", err)
	}
	if plain != withOptions {
		t.Errorf("las dos formas del envoltorio difieren:\n%s\n---\n%s", plain, withOptions)
	}
	for _, want := range []string{"### Requirement: Widget Expiration", "### Requirement: Widget Tagging"} {
		if !strings.Contains(plain, want) {
			t.Errorf("la composición debe contener %q:\n%s", want, plain)
		}
	}
}

func TestComposeOpenSpecCanonicalSpecWrapperKeepsTheUnappliedDeltaErrorType(t *testing.T) {
	_, err := ComposeOpenSpecCanonicalSpec(composeWrapperCanonical, "## MODIFIED Requirements\n\n### Requirement: Missing\n\nBody.\n")
	var unapplied *UnappliedDeltaError
	if !errors.As(err, &unapplied) {
		t.Fatalf("error = %v, want *UnappliedDeltaError", err)
	}
	if unapplied.Section != "MODIFIED" || unapplied.Requirement != "Missing" {
		t.Fatalf("unapplied = %+v, want MODIFIED/Missing", unapplied)
	}
}

func TestComposeOpenSpecCanonicalSpecWrapperSupersedesRemovedRequirements(t *testing.T) {
	delta := "## REMOVED Requirements\n\n### Requirement: Widget Expiration\n\n(Reason: replaced)\n"
	composed, err := ComposeOpenSpecCanonicalSpecWithOptions(composeWrapperCanonical, delta, ComposeOptions{SupersedeRemoved: true})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(composed, "[SUPERSEDED / DEPRECADO]") {
		t.Errorf("SupersedeRemoved debe marcar el requisito:\n%s", composed)
	}
}
