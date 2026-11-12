package livespec

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Las pruebas de la planificación de la integración (REQ-30.2, REQ-30.3, REQ-30.4, D5,
// D6). Todos los textos son inventados.

// planFixture crea un planning home con el árbol dado y devuelve su Location mínima.
func planFixture(t *testing.T, files map[string]string) (workspace.Location, *Registry) {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "openspec")
	writeTree(t, home, files)
	reg, err := LoadRegistry(home, RegistryOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return workspace.Location{WorkspaceRoot: root, PlanningHome: home}, reg
}

// docWith construye un documento ODD con una capacidad por entrada del mapa (orden
// alfabético de las claves que da el slice).
func docWith(t *testing.T, kind string, caps ...[2]string) FeatureDoc {
	t.Helper()
	body := "## Spec viva\n\n"
	for _, c := range caps {
		body += capability(c[0], fence(c[1]))
	}
	header := ""
	if kind != "" {
		header = "Tipo: " + kind + "\n\n"
	}
	return parseOK(t, featureDoc(header, body))
}

func addedBlock(name, id, body string) string {
	return "### Requirement: " + name + " (" + id + ")\n\n" + body + "\n"
}

func TestPlanIntegrationCreatesANewCapabilityFromAddedRequirements(t *testing.T) {
	loc, reg := planFixture(t, nil)
	delta := "## ADDED Requirements\n\n" + addedBlock("Alta", "REQ-31.1", "El sistema DEBE dar de alta.")
	plan, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", delta}), reg)
	if err != nil {
		t.Fatalf("PlanIntegration: %v", err)
	}
	if len(plan.Capabilities) != 1 {
		t.Fatalf("capacidades = %+v", plan.Capabilities)
	}
	got := plan.Capabilities[0]
	if got.Name != "facturacion" || got.Outcome != OutcomeCreated || got.Existed {
		t.Errorf("plan = %+v", got)
	}
	if want := filepath.Join(loc.PlanningHome, "specs", "facturacion", "spec.md"); got.Path != want {
		t.Errorf("Path = %q, want %q", got.Path, want)
	}
	for _, want := range []string{"# Especificación: facturacion", "Integrada desde el documento ODD `mi-feature`.", "### Requirement: Alta (REQ-31.1)"} {
		if !strings.Contains(got.Content, want) {
			t.Errorf("el contenido debe contener %q:\n%s", want, got.Content)
		}
	}
	if plan.Impact != ImpactDelta {
		t.Errorf("Impact = %q", plan.Impact)
	}
}

func TestPlanIntegrationComposesOntoAnExistingCapability(t *testing.T) {
	loc, reg := planFixture(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
	delta := "## ADDED Requirements\n\n" + addedBlock("Nota de abono", "REQ-31.5", "El sistema DEBE emitir notas de abono.")
	plan, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", delta}), reg)
	if err != nil {
		t.Fatalf("PlanIntegration: %v", err)
	}
	got := plan.Capabilities[0]
	if got.Outcome != OutcomeComposed || !got.Existed {
		t.Fatalf("plan = %+v", got)
	}
	if !strings.HasPrefix(got.Content, facturacionSpec) || !strings.Contains(got.Content, "Nota de abono (REQ-31.5)") {
		t.Errorf("la composición debe conservar los bytes existentes y añadir el requisito:\n%s", got.Content)
	}
}

func TestPlanIntegrationClassifiesEachRequirementAgainstTheCanonicalSpec(t *testing.T) {
	spec := facturacionSpec
	tests := []struct {
		name        string
		delta       string
		wantOutcome Outcome
	}{
		{"ADDED con el mismo cuerpo ya está integrado", "## ADDED Requirements\n\n" + addedBlock("Alta de factura", "REQ-31.1", "El sistema DEBE dar de alta una factura."), OutcomeAlready},
		{"el cuerpo se compara normalizado", "## ADDED Requirements\n\n### Requirement: Alta de factura (REQ-31.1)  \r\n\r\nEl sistema DEBE dar de alta una factura.   \r\n\r\n\r\n", OutcomeAlready},
		{"MODIFIED con el mismo cuerpo ya está aplicado", "## MODIFIED Requirements\n\n" + addedBlock("Alta de factura", "REQ-31.1", "El sistema DEBE dar de alta una factura."), OutcomeAlready},
		{"MODIFIED con otro cuerpo está pendiente", "## MODIFIED Requirements\n\n" + addedBlock("Alta de factura", "REQ-31.1", "El sistema DEBE dar de alta y validar."), OutcomeComposed},
		{"REMOVED ausente ya está aplicado", "## REMOVED Requirements\n\n### Requirement: Retirada (REQ-31.9)\n\n(Reason: obsoleta)\n", OutcomeAlready},
		{"REMOVED presente está pendiente", "## REMOVED Requirements\n\n### Requirement: Baja de factura (REQ-31.2)\n\n(Reason: obsoleta)\n", OutcomeComposed},
		{"RENAMED con el nombre nuevo presente y el viejo ausente ya está aplicado", "## RENAMED Requirements\n\n### Requirement: Antigua → Alta de factura (REQ-31.1)\n\n(Reason: nombre)\n", OutcomeAlready},
		{"RENAMED pendiente", "## RENAMED Requirements\n\n### Requirement: Alta de factura (REQ-31.1) → Emisión de factura (REQ-31.1)\n\n(Reason: nombre)\n", OutcomeComposed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, reg := planFixture(t, map[string]string{"specs/facturacion/spec.md": spec})
			plan, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", tt.delta}), reg)
			if err != nil {
				t.Fatalf("PlanIntegration: %v", err)
			}
			if got := plan.Capabilities[0].Outcome; got != tt.wantOutcome {
				t.Errorf("Outcome = %q, want %q", got, tt.wantOutcome)
			}
			if tt.wantOutcome == OutcomeAlready && plan.Capabilities[0].Content != "" {
				t.Errorf("una capacidad ya integrada no lleva contenido que escribir: %q", plan.Capabilities[0].Content)
			}
		})
	}
}

func TestPlanIntegrationComposesOnlyThePendingRequirements(t *testing.T) {
	loc, reg := planFixture(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
	delta := "## ADDED Requirements\n\n" +
		addedBlock("Alta de factura", "REQ-31.1", "El sistema DEBE dar de alta una factura.") + "\n" +
		addedBlock("Nota de abono", "REQ-31.5", "El sistema DEBE emitir notas de abono.")
	plan, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", delta}), reg)
	if err != nil {
		t.Fatalf("PlanIntegration: %v", err)
	}
	got := plan.Capabilities[0]
	if got.Outcome != OutcomeComposed {
		t.Fatalf("Outcome = %q", got.Outcome)
	}
	if n := strings.Count(got.Content, "Alta de factura (REQ-31.1)"); n != 1 {
		t.Errorf("el requisito ya integrado no se duplica: aparece %d veces\n%s", n, got.Content)
	}
	if !strings.Contains(got.Content, "Nota de abono (REQ-31.5)") {
		t.Errorf("falta el requisito pendiente:\n%s", got.Content)
	}
}

func TestPlanIntegrationRejectsAnAddedRequirementAlreadyPresentWithAnotherBody(t *testing.T) {
	loc, reg := planFixture(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
	delta := "## ADDED Requirements\n\n" + addedBlock("Alta de factura", "REQ-31.1", "El sistema DEBE dar de alta una factura y notificarla.")
	_, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", delta}), reg)
	var divergent *DivergentRequirementError
	if !errors.As(err, &divergent) {
		t.Fatalf("error = %v, se esperaba *DivergentRequirementError", err)
	}
	if divergent.Capability != "facturacion" || divergent.Requirement != "Alta de factura (REQ-31.1)" || divergent.ID != "REQ-31.1" {
		t.Errorf("divergent = %+v", divergent)
	}
	for _, want := range []string{"facturacion", "Alta de factura (REQ-31.1)"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error %q debe nombrar %q", err, want)
		}
	}
}

func TestPlanIntegrationFailsClosedWhenAModifiedNameNoLongerMatches(t *testing.T) {
	loc, reg := planFixture(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
	delta := "## MODIFIED Requirements\n\n" + addedBlock("Emisión de factura", "REQ-31.1", "Texto nuevo.")
	_, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", delta}), reg)
	var unapplied *openspecdoc.UnappliedDeltaError
	if !errors.As(err, &unapplied) {
		t.Fatalf("error = %v, se esperaba *UnappliedDeltaError", err)
	}
	if unapplied.Section != "MODIFIED" || unapplied.Requirement != "Emisión de factura (REQ-31.1)" {
		t.Errorf("unapplied = %+v", unapplied)
	}
	if !strings.Contains(err.Error(), "facturacion") {
		t.Errorf("el error %q debe nombrar la capacidad", err)
	}
}

func TestPlanIntegrationRejectsNonAddedOperationsOnAMissingCapability(t *testing.T) {
	tests := []struct {
		name, delta, section string
	}{
		{"MODIFIED", "## MODIFIED Requirements\n\n" + addedBlock("Alta", "REQ-31.1", "x"), "MODIFIED"},
		{"REMOVED", "## REMOVED Requirements\n\n### Requirement: Alta\n\n(Reason: r)\n", "REMOVED"},
		{"RENAMED", "## RENAMED Requirements\n\n### Requirement: A → B\n\n(Reason: r)\n", "RENAMED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, reg := planFixture(t, nil)
			_, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", tt.delta}), reg)
			var missing *MissingCapabilityError
			if !errors.As(err, &missing) {
				t.Fatalf("error = %v, se esperaba *MissingCapabilityError", err)
			}
			if missing.Capability != "facturacion" || missing.Section != tt.section {
				t.Errorf("missing = %+v", missing)
			}
		})
	}
}

func TestPlanIntegrationAddedRequirementsNeedAnIdentifier(t *testing.T) {
	loc, reg := planFixture(t, nil)
	delta := "## ADDED Requirements\n\n### Requirement: Sin identificador\n\nCuerpo.\n"
	_, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", delta}), reg)
	var docErr *DocError
	if !errors.As(err, &docErr) {
		t.Fatalf("error = %v, se esperaba *DocError", err)
	}
	wantProblem(t, docErr.Problems, "added_without_id", "Sin identificador")
}

func TestPlanIntegrationRejectsIdentifierCollisions(t *testing.T) {
	files := map[string]string{
		"specs/facturacion/spec.md":                        facturacionSpec,
		"specs/pagos/spec.md":                              "# pagos\n\n## Requirements\n\n### Requirement: Cobro (REQ-32.1)\n\nCuerpo.\n",
		"changes/inc-7/specs/pagos/spec.md":                "## ADDED Requirements\n\n### Requirement: Reembolso (REQ-32.2)\n\nCuerpo.\n",
		"changes/archive/2026-01-01-x/specs/pagos/spec.md": "## ADDED Requirements\n\n### Requirement: Antiguo (REQ-32.3)\n\nCuerpo.\n",
	}
	tests := []struct {
		name       string
		caps       [][2]string
		wantID     string
		wantWhere  string
		wantNext   string
		wantNoFail bool
	}{
		{
			name:      "con la spec viva",
			caps:      [][2]string{{"pagos", "## ADDED Requirements\n\n" + addedBlock("Otro cobro", "REQ-31.1", "Cuerpo.")}},
			wantID:    "REQ-31.1",
			wantWhere: "la capacidad facturacion",
			wantNext:  "REQ-31.3",
		},
		{
			name:      "con la delta de un cambio activo",
			caps:      [][2]string{{"pagos", "## ADDED Requirements\n\n" + addedBlock("Otro reembolso", "REQ-32.2", "Cuerpo.")}},
			wantID:    "REQ-32.2",
			wantWhere: "inc-7",
			wantNext:  "REQ-32.3",
		},
		{
			name: "duplicado dentro del documento",
			caps: [][2]string{
				{"alfa", "## ADDED Requirements\n\n" + addedBlock("Uno", "REQ-40.1", "Cuerpo.")},
				{"beta", "## ADDED Requirements\n\n" + addedBlock("Dos", "REQ-40.1", "Cuerpo.")},
			},
			wantID:    "REQ-40.1",
			wantWhere: "alfa",
			wantNext:  "REQ-40.2",
		},
		{
			name:       "los cambios archivados no cuentan",
			caps:       [][2]string{{"pagos", "## ADDED Requirements\n\n" + addedBlock("Resucitado", "REQ-32.3", "Cuerpo.")}},
			wantNoFail: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, reg := planFixture(t, files)
			_, err := PlanIntegration(loc, docWith(t, "", tt.caps...), reg)
			if tt.wantNoFail {
				if err != nil {
					t.Fatalf("PlanIntegration: %v", err)
				}
				return
			}
			var collision *CollisionError
			if !errors.As(err, &collision) {
				t.Fatalf("error = %v, se esperaba *CollisionError", err)
			}
			if collision.ID != tt.wantID || !strings.Contains(collision.Where, tt.wantWhere) || collision.NextFree != tt.wantNext {
				t.Errorf("collision = %+v", collision)
			}
			if tt.name == "duplicado dentro del documento" && !strings.Contains(collision.Where, "beta") {
				t.Errorf("el duplicado debe nombrar ambas capacidades: %q", collision.Where)
			}
			for _, want := range []string{collision.ID, collision.NextFree} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("el error %q debe nombrar %q", err, want)
				}
			}
		})
	}
}

func TestPlanIntegrationAnAlreadyIntegratedRequirementIsNotACollision(t *testing.T) {
	loc, reg := planFixture(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
	delta := "## ADDED Requirements\n\n" + addedBlock("Alta de factura", "REQ-31.1", "El sistema DEBE dar de alta una factura.")
	plan, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", delta}), reg)
	if err != nil {
		t.Fatalf("una integración ya aplicada no colisiona consigo misma: %v", err)
	}
	if plan.Capabilities[0].Outcome != OutcomeAlready {
		t.Errorf("Outcome = %q", plan.Capabilities[0].Outcome)
	}
}

func TestPlanIntegrationIsAllOrNothingAcrossCapabilities(t *testing.T) {
	loc, reg := planFixture(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
	good := "## ADDED Requirements\n\n" + addedBlock("Nota de abono", "REQ-31.5", "Cuerpo.")
	bad := "## MODIFIED Requirements\n\n" + addedBlock("Inexistente", "REQ-31.6", "Cuerpo.")
	plan, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", good}, [2]string{"facturacion-extra", bad}), reg)
	if err == nil {
		t.Fatal("un MODIFIED sobre una capacidad inexistente debe rechazar todo el plan")
	}
	if len(plan.Capabilities) != 0 {
		t.Errorf("un plan rechazado no devuelve capacidades: %+v", plan.Capabilities)
	}
}

func TestPlanIntegrationDocumentWithoutADeltaHasNothingToIntegrate(t *testing.T) {
	loc, reg := planFixture(t, nil)
	tests := []struct {
		name string
		text string
		want string
	}{
		{"sin secciones", featureDoc("", ""), ImpactAbsent},
		{"con marcador válido", featureDoc("Tipo: bug\n\n", "## Impacto en la spec\n\nSin impacto en la spec\nMotivo: razón\n"), ImpactNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := PlanIntegration(loc, parseOK(t, tt.text), reg)
			if err != nil {
				t.Fatalf("PlanIntegration: %v", err)
			}
			if len(plan.Capabilities) != 0 || plan.Impact != tt.want || !plan.NothingToIntegrate() {
				t.Errorf("plan = %+v", plan)
			}
		})
	}
}

func TestPlanIntegrationAMarkerTogetherWithADeltaIsIgnored(t *testing.T) {
	loc, reg := planFixture(t, nil)
	delta := "## ADDED Requirements\n\n" + addedBlock("Alta", "REQ-31.1", "Cuerpo.")
	body := "## Spec viva\n\n" + capability("facturacion", fence(delta)) + "## Impacto en la spec\n\nSin impacto en la spec\nMotivo: razón\n"
	plan, err := PlanIntegration(loc, parseOK(t, featureDoc("Tipo: bug\n\n", body)), reg)
	if err != nil {
		t.Fatalf("PlanIntegration: %v", err)
	}
	if plan.Impact != ImpactDelta || plan.Capabilities[0].Outcome != OutcomeCreated {
		t.Errorf("plan = %+v: el marcador junto con una delta se ignora y se integra la delta", plan)
	}
}

func TestPlanIntegrationNothingToIntegrateWhenEveryCapabilityIsAlreadyIntegrated(t *testing.T) {
	loc, reg := planFixture(t, map[string]string{"specs/facturacion/spec.md": facturacionSpec})
	delta := "## REMOVED Requirements\n\n### Requirement: Retirada\n\n(Reason: obsoleta)\n"
	plan, err := PlanIntegration(loc, docWith(t, "", [2]string{"facturacion", delta}), reg)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.NothingToIntegrate() {
		t.Errorf("plan = %+v, no había nada que integrar", plan)
	}
}
