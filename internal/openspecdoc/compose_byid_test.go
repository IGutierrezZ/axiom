package openspecdoc

import (
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

const composeByIDCanonical = `# Cobros

## Requirements

### Requirement: Alta de cobro (REQ-1.1)

El sistema MUST dar de alta cobros.

#### Scenario: Alta

- GIVEN un cobro
- WHEN se da de alta
- THEN queda registrado

### Requirement: Cobro con tarjeta (REQ-1.2)

El sistema MUST cobrar con tarjeta.

#### Scenario: Tarjeta

- GIVEN una tarjeta
- WHEN se cobra
- THEN se cobra

### Requirement: Baja de cobro (REQ-1.3)

El sistema MUST dar de baja cobros.

#### Scenario: Baja

- GIVEN un cobro
- WHEN se da de baja
- THEN desaparece
`

// RF-ARC-11: un MODIFIED con el título cambiado y el mismo ID reemplaza el
// requisito, el segmento toma el nombre de la delta y la siguiente operación de la
// misma delta lo ve.
func TestComposeModifiedByIDWithChangedTitle(t *testing.T) {
	delta := `## MODIFIED Requirements

### Requirement: Cobro con tarjeta o bizum (REQ-1.2)

El sistema MUST cobrar con tarjeta o bizum.

#### Scenario: Bizum

- GIVEN un bizum
- WHEN se cobra
- THEN se cobra
`
	out, err := Compose(composeByIDCanonical, delta, ComposeOptions{})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if strings.Contains(out, "### Requirement: Cobro con tarjeta (REQ-1.2)") || !strings.Contains(out, "### Requirement: Cobro con tarjeta o bizum (REQ-1.2)") {
		t.Errorf("el requisito debe tomar el título de la delta:\n%s", out)
	}
	if !strings.Contains(out, "### Requirement: Alta de cobro (REQ-1.1)") || !strings.Contains(out, "### Requirement: Baja de cobro (REQ-1.3)") {
		t.Errorf("los demás requisitos se conservan:\n%s", out)
	}

	// Un segundo MODIFIED del mismo ID dentro de la misma delta ve el segmento ya
	// renombrado por el primero.
	twice := delta + `
### Requirement: Cobro con tarjeta, bizum o transferencia (REQ-1.2)

El sistema MUST cobrar con tarjeta, bizum o transferencia.

#### Scenario: Transferencia

- GIVEN una transferencia
- WHEN se cobra
- THEN se cobra
`
	out, err = Compose(composeByIDCanonical, twice, ComposeOptions{})
	if err != nil {
		t.Fatalf("Compose con dos MODIFIED del mismo ID: %v", err)
	}
	if !strings.Contains(out, "bizum o transferencia (REQ-1.2)") || strings.Contains(out, "tarjeta o bizum (REQ-1.2)") {
		t.Errorf("el segundo MODIFIED reemplaza al primero:\n%s", out)
	}
}

// RF-ARC-11: REMOVED por ID con el título obsoleto, con y sin SupersedeRemoved; la
// anotación usa el nombre del segmento de la canónica.
func TestComposeRemovedByIDWithObsoleteTitle(t *testing.T) {
	delta := `## REMOVED Requirements

### Requirement: Baja antigua (REQ-1.3)

(Reason: ya no se usa)
`
	out, err := Compose(composeByIDCanonical, delta, ComposeOptions{})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if strings.Contains(out, "(REQ-1.3)") || !strings.Contains(out, "(REQ-1.1)") || !strings.Contains(out, "(REQ-1.2)") {
		t.Errorf("solo se borra el requisito REQ-1.3:\n%s", out)
	}

	out, err = Compose(composeByIDCanonical, delta, ComposeOptions{SupersedeRemoved: true})
	if err != nil {
		t.Fatalf("Compose con SupersedeRemoved: %v", err)
	}
	if !strings.Contains(out, "### Requirement: Baja de cobro (REQ-1.3)"+SupersededSuffix) || strings.Contains(out, "Baja antigua") {
		t.Errorf("la anotación usa el nombre del segmento de la canónica:\n%s", out)
	}
}

// RF-ARC-11: el nombre antiguo de un RENAMED se resuelve por ID y sigue exigiendo
// `Antiguo → Nuevo` y la nota (Reason).
func TestComposeRenamedOldNameByID(t *testing.T) {
	delta := `## RENAMED Requirements

### Requirement: Alta vieja (REQ-1.1) → Alta de cobro nueva (REQ-1.1)

(Reason: renombrado)
`
	out, err := Compose(composeByIDCanonical, delta, ComposeOptions{})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if !strings.Contains(out, "### Requirement: Alta de cobro nueva (REQ-1.1)") || strings.Contains(out, "### Requirement: Alta de cobro (REQ-1.1)") {
		t.Errorf("el requisito se renombra:\n%s", out)
	}

	var unapplied *UnappliedDeltaError
	noArrow := "## RENAMED Requirements\n\n### Requirement: Alta vieja (REQ-1.1)\n\n(Reason: x)\n"
	if _, err := Compose(composeByIDCanonical, noArrow, ComposeOptions{}); !errors.As(err, &unapplied) || unapplied.Section != "RENAMED" {
		t.Errorf("sin flecha se rechaza: %v", err)
	}
	noReason := "## RENAMED Requirements\n\n### Requirement: Alta vieja (REQ-1.1) → Alta nueva (REQ-1.1)\n"
	if _, err := Compose(composeByIDCanonical, noReason, ComposeOptions{}); !errors.As(err, &unapplied) || !strings.Contains(unapplied.Reason, "Reason") {
		t.Errorf("sin nota (Reason) se rechaza: %v", err)
	}
}

// RF-ARC-11: ADDED, la colisión del nombre nuevo de un RENAMED y ExplicitSuperseded
// siguen emparejando por título exacto.
func TestComposeAddedAndRenameCollisionStayByTitle(t *testing.T) {
	var unapplied *UnappliedDeltaError
	added := "## ADDED Requirements\n\n### Requirement: Otra cosa (REQ-1.1)\n\nEl sistema MUST algo.\n\n#### Scenario: S\n\n- GIVEN a\n- WHEN b\n- THEN c\n"
	if out, err := Compose(composeByIDCanonical, added, ComposeOptions{}); err != nil || !strings.Contains(out, "Otra cosa (REQ-1.1)") {
		t.Errorf("un ADDED con un ID ya presente bajo otro título sigue aceptado (la unicidad es aguas arriba): %v", err)
	}
	dup := "## ADDED Requirements\n\n### Requirement: Alta de cobro (REQ-1.1)\n\nEl sistema MUST algo.\n"
	if _, err := Compose(composeByIDCanonical, dup, ComposeOptions{}); !errors.As(err, &unapplied) || unapplied.Section != "ADDED" {
		t.Errorf("ADDED con título exacto existente se rechaza: %v", err)
	}
	collide := "## RENAMED Requirements\n\n### Requirement: Alta de cobro (REQ-1.1) → Baja de cobro (REQ-1.3)\n\n(Reason: x)\n"
	if _, err := Compose(composeByIDCanonical, collide, ComposeOptions{}); !errors.As(err, &unapplied) || !strings.Contains(unapplied.Reason, "already exists") {
		t.Errorf("colisión del nombre nuevo: %v", err)
	}
	superseded, err := Compose(composeByIDCanonical, "## ADDED Requirements\n\n### Requirement: Nuevo (REQ-1.9)\n\nEl sistema MUST.\n", ComposeOptions{ExplicitSuperseded: []string{"Alta de cobro (REQ-1.1)"}})
	if err != nil || !strings.Contains(superseded, "Alta de cobro (REQ-1.1)"+SupersededSuffix) {
		t.Errorf("ExplicitSuperseded por título exacto: %v\n%s", err, superseded)
	}
}

const composeRepeatedIDCanonical = `# Cobros

## Requirements

### Requirement: Cobro A (REQ-1.2)

Texto A.

### Requirement: Cobro B (REQ-1.2)

Texto B.
`

// RF-ARC-11: un ID repetido en la canónica se acepta si el título exacto desempata y
// se rechaza, nombrando el identificador y los títulos afectados, si no.
func TestComposeRepeatedIDInCanonical(t *testing.T) {
	exact := "## MODIFIED Requirements\n\n### Requirement: Cobro B (REQ-1.2)\n\nTexto B nuevo.\n"
	out, err := Compose(composeRepeatedIDCanonical, exact, ComposeOptions{})
	if err != nil || !strings.Contains(out, "Texto B nuevo.") || !strings.Contains(out, "Texto A.") {
		t.Fatalf("el título exacto desempata: %v\n%s", err, out)
	}

	ambiguous := "## MODIFIED Requirements\n\n### Requirement: Cobro C (REQ-1.2)\n\nTexto C.\n"
	_, err = Compose(composeRepeatedIDCanonical, ambiguous, ComposeOptions{})
	var unapplied *UnappliedDeltaError
	if !errors.As(err, &unapplied) || unapplied.Section != "MODIFIED" {
		t.Fatalf("sin título exacto se rechaza: %v", err)
	}
	for _, want := range []string{"REQ-1.2", "Cobro A (REQ-1.2)", "Cobro B (REQ-1.2)", "Cobro C (REQ-1.2)"} {
		if !strings.Contains(unapplied.Reason, want) {
			t.Errorf("el motivo %q debe nombrar %q", unapplied.Reason, want)
		}
	}
}

// RF-ARC-11: los títulos sin ID siguen por título exacto, y con otro patrón la
// resolución por ID usa ese patrón.
func TestComposeWithoutIDsAndWithAnotherPattern(t *testing.T) {
	canonical := "# C\n\n## Requirements\n\n### Requirement: Sin identificador\n\nTexto.\n"
	delta := "## MODIFIED Requirements\n\n### Requirement: Sin identificador\n\nTexto nuevo.\n"
	if out, err := Compose(canonical, delta, ComposeOptions{}); err != nil || !strings.Contains(out, "Texto nuevo.") {
		t.Fatalf("título sin ID: %v", err)
	}
	if _, err := Compose(canonical, "## MODIFIED Requirements\n\n### Requirement: Otro\n\nX.\n", ComposeOptions{}); err == nil {
		t.Fatal("un título sin ID que no existe se rechaza")
	}

	rf := mustPattern(t, "RF-<DOM>-<NN>")
	rfCanonical := "# U\n\n## Requirements\n\n### Requirement: Alta (RF-USU-01)\n\nTexto.\n\n### Requirement: Alta masiva (RF-USU-02)\n\nTexto.\n"
	rfDelta := "## MODIFIED Requirements\n\n### Requirement: Alta masiva y por lotes (RF-USU-02)\n\nTexto nuevo.\n"
	out, err := Compose(rfCanonical, rfDelta, ComposeOptions{IDPattern: rf})
	if err != nil || !strings.Contains(out, "Alta masiva y por lotes (RF-USU-02)") {
		t.Fatalf("RF-<DOM>-<NN> resuelve por ID: %v\n%s", err, out)
	}
	// Con el patrón por defecto, RF-USU-02 no es un ID: solo vale el título exacto.
	if _, err := Compose(rfCanonical, rfDelta, ComposeOptions{IDPattern: reqid.Default()}); err == nil {
		t.Fatal("con el defecto el título cambiado no se resuelve")
	}
}
