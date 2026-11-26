package livespec

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// Las pruebas del plan único de asignación de identificadores (D3, D5, D7; REQ-47.2 a
// REQ-47.5). Todos los textos son inventados.

const pagosCanon = "# Especificación: pagos\n\n## Requirements\n\n### Requirement: Cobro (REQ-30.1)\n\nEl sistema DEBE cobrar.\n\n### Requirement: Reembolso (REQ-30.2)\n\nEl sistema DEBE reembolsar.\n"

// source construye la capacidad de un cambio a partir del texto de su delta.
func source(name, delta string) CapabilitySource {
	return CapabilitySource{
		CapabilityDelta: CapabilityDelta{Name: name, Delta: openspecdoc.ParseDelta(delta), Text: delta},
		File:            "specs/" + name + "/spec.md",
	}
}

// idInput monta un registro con los ficheros dados (rutas bajo openspec/) y completa la
// canónica de cada capacidad con la del árbol.
func idInput(t *testing.T, pattern reqid.Pattern, files map[string]string, caps ...CapabilitySource) IDPlanInput {
	t.Helper()
	home := filepath.Join(t.TempDir(), "openspec")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTree(t, home, files)
	reg, err := LoadRegistry(home, RegistryOptions{Pattern: pattern, ExcludeChange: "propio"})
	if err != nil {
		t.Fatal(err)
	}
	for i := range caps {
		rel := "specs/" + caps[i].Name + "/spec.md"
		caps[i].Path = filepath.Join(home, filepath.FromSlash(rel))
		caps[i].Canonical, caps[i].Existed = files[rel], files[rel] != ""
	}
	return IDPlanInput{Pattern: pattern, Capabilities: caps, Registry: reg}
}

func added(blocks ...string) string {
	return "## ADDED Requirements\n\n" + strings.Join(blocks, "\n")
}

func block(title, body string) string {
	return "### Requirement: " + title + "\n\n" + body + "\n"
}

func mustPlan(t *testing.T, in IDPlanInput) IDPlan {
	t.Helper()
	plan, err := PlanIDs(in)
	if err != nil {
		t.Fatalf("PlanIDs: %v", err)
	}
	return plan
}

func definitiveOf(t *testing.T, plan IDPlan, provisional string) string {
	t.Helper()
	for _, e := range plan.Entries {
		if e.Provisional == provisional {
			return e.Definitive
		}
	}
	t.Fatalf("%s no tiene entrada en %+v", provisional, plan.Entries)
	return ""
}

func TestPlanIDsRejectsEachDocumentedCodeWithoutWriting(t *testing.T) {
	rfDom, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	pagos := map[string]string{"specs/pagos/spec.md": pagosCanon}
	type tc struct {
		name    string
		pattern reqid.Pattern
		files   map[string]string
		caps    []CapabilitySource
		tweak   func(*IDPlanInput)
		code    string
		check   func(*testing.T, *IDPlanError)
	}
	cases := []tc{
		{name: "malformado", pattern: reqid.Default(), files: pagos,
			caps: []CapabilitySource{source("pagos", added(block("Alta (prov:30:0)", "Cuerpo.")))},
			code: CodeProvisionalMalformed, check: func(t *testing.T, e *IDPlanError) {
				if e.Token != "prov:30:0" || e.File != "specs/pagos/spec.md" || e.Line == 0 {
					t.Errorf("campos = %+v", e)
				}
			}},
		{name: "serie nueva no soportada", pattern: rfDom, files: map[string]string{},
			caps: []CapabilitySource{source("usuarios", added(block("Alta (prov:usu:1)", "Cuerpo.")))},
			code: CodeProvisionalNewSeriesUnsupported},
		{name: "fuera de ADDED", pattern: reqid.Default(), files: pagos,
			caps: []CapabilitySource{source("pagos", "## MODIFIED Requirements\n\n"+block("Cobro (prov:30:1)", "Cuerpo."))},
			code: CodeProvisionalOutsideAdded, check: func(t *testing.T, e *IDPlanError) {
				if e.Section != "MODIFIED" || e.Capability != "pagos" || e.Token != "prov:30:1" {
					t.Errorf("campos = %+v", e)
				}
			}},
		{name: "duplicado", pattern: reqid.Default(), files: pagos,
			caps: []CapabilitySource{source("pagos", added(block("Alta (prov:30:1)", "Uno."), block("Baja (prov:30:1)", "Dos.")))},
			code: CodeProvisionalDuplicate, check: func(t *testing.T, e *IDPlanError) {
				if e.Token != "prov:30:1" || e.Requirement != "Baja (prov:30:1)" {
					t.Errorf("campos = %+v", e)
				}
			}},
		{name: "ambiguo", pattern: reqid.Default(), files: pagos,
			caps: []CapabilitySource{source("pagos", added(block("Alta (prov:30:1) (REQ-30.9)", "Cuerpo.")))},
			code: CodeProvisionalAmbiguous},
		{name: "sin declarar en la delta", pattern: reqid.Default(), files: pagos,
			caps: []CapabilitySource{source("pagos", added(block("Alta (prov:30:1)", "Cuerpo que cita prov:30:5.")))},
			code: CodeProvisionalUndeclared, check: func(t *testing.T, e *IDPlanError) {
				if e.Token != "prov:30:5" {
					t.Errorf("Token = %q", e.Token)
				}
			}},
		{name: "sin declarar en una cita", pattern: reqid.Default(), files: pagos,
			caps:  []CapabilitySource{source("pagos", added(block("Alta (prov:30:1)", "Cuerpo.")))},
			tweak: func(in *IDPlanInput) { in.Citations = []Citation{{File: "tasks.md", Line: 7, Token: "prov:30:8"}} },
			code:  CodeProvisionalUndeclared, check: func(t *testing.T, e *IDPlanError) {
				if e.File != "tasks.md" || e.Line != 7 {
					t.Errorf("campos = %+v", e)
				}
			}},
		{name: "ID inexistente", pattern: reqid.Default(), files: pagos,
			caps: []CapabilitySource{source("pagos", "## MODIFIED Requirements\n\n"+block("Cobro (REQ-30.99)", "Nuevo."))},
			code: CodeUnknownRequirementID, check: func(t *testing.T, e *IDPlanError) {
				var unknown *openspecdoc.UnknownRequirementIDError
				if e.Capability != "pagos" || e.Token != "REQ-30.99" || !errors.As(e, &unknown) {
					t.Errorf("campos = %+v", e)
				}
			}},
		{name: "asignación imposible", pattern: reqid.Default(),
			files: map[string]string{"specs/pagos/spec.md": "# E\n\n## Requirements\n\n### Requirement: Tope (REQ-30.9223372036854775807)\n\nCuerpo.\n"},
			caps:  []CapabilitySource{source("pagos", added(block("Alta (prov:30:1)", "Cuerpo.")))},
			code:  CodeIDAllocationFailed},
		{name: "superado inexistente", pattern: reqid.Default(), files: pagos,
			caps:  []CapabilitySource{source("pagos", added(block("Alta (prov:30:1)", "Cuerpo.")))},
			tweak: func(in *IDPlanInput) { in.Superseded = []string{"No existe"} },
			code:  CodeSupersededRequirementUnknown},
		{name: "superado ambiguo", pattern: reqid.Default(),
			files: map[string]string{
				"specs/pagos/spec.md":       "# E\n\n## Requirements\n\n### Requirement: Resumen\n\nUno.\n",
				"specs/facturacion/spec.md": "# E\n\n## Requirements\n\n### Requirement: Resumen\n\nDos.\n",
			},
			caps:  []CapabilitySource{source("pagos", added(block("Alta (prov:30:1)", "Cuerpo."))), source("facturacion", added(block("Baja (prov:31:1)", "Cuerpo.")))},
			tweak: func(in *IDPlanInput) { in.Superseded = []string{"Resumen"} },
			code:  CodeSupersededRequirementAmbiguous, check: func(t *testing.T, e *IDPlanError) {
				if !strings.Contains(e.Continuation, "pagos/Resumen") || !strings.Contains(e.Continuation, "facturacion/Resumen") {
					t.Errorf("la continuación no nombra el prefijo: %q", e.Continuation)
				}
			}},
		{name: "delta plana", pattern: reqid.Default(), files: pagos,
			caps:  []CapabilitySource{source("pagos", added(block("Alta (prov:30:1)", "Cuerpo.")))},
			tweak: func(in *IDPlanInput) { in.FlatDelta = "specs/spec.md" },
			code:  CodeFlatDeltaUnsupported, check: func(t *testing.T, e *IDPlanError) {
				if e.File != "specs/spec.md" {
					t.Errorf("File = %q", e.File)
				}
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := idInput(t, c.pattern, c.files, c.caps...)
			if c.tweak != nil {
				c.tweak(&in)
			}
			before := snapshotTree(t, treeRoot(in))
			plan, err := PlanIDs(in)
			planErr := wantPlanError(t, err, c.code)
			if plan.Map.Count() != 0 || len(plan.Entries) != 0 {
				t.Errorf("un plan rechazado devolvió asignaciones: %+v", plan)
			}
			if c.check != nil {
				c.check(t, planErr)
			}
			if !sameTree(before, snapshotTree(t, treeRoot(in))) {
				t.Error("el plan escribió en el árbol")
			}
		})
	}
}

func TestPlanIDsRejectionsKeepTheValidationOrder(t *testing.T) {
	// Un nombre ambiguo (c) se reporta antes que una cita sin declarar (d).
	in := idInput(t, reqid.Default(), map[string]string{"specs/pagos/spec.md": pagosCanon},
		source("pagos", added(block("Alta (prov:30:1) (REQ-30.9)", "Cita prov:30:7."))))
	_, err := PlanIDs(in)
	wantPlanError(t, err, CodeProvisionalAmbiguous)
}

func TestPlanIDsRejectionsOnTheSameInputAreIdenticalWithAndWithoutCheckSemantics(t *testing.T) {
	in := idInput(t, reqid.Default(), map[string]string{"specs/pagos/spec.md": pagosCanon},
		source("pagos", added(block("Alta (prov:30:1)", "Cita prov:30:7."))))
	_, first := PlanIDs(in)
	_, second := PlanIDs(in)
	if first == nil || second == nil || first.Error() != second.Error() {
		t.Fatalf("el rechazo no es estable: %v / %v", first, second)
	}
}

func TestPlanIDsAssignsTheNextFreeNumberAcrossCapabilitiesWithoutRepeating(t *testing.T) {
	files := map[string]string{"specs/pagos/spec.md": pagosCanon}
	in := idInput(t, reqid.Default(), files,
		source("facturacion", added(block("Alta (prov:30:1)", "Uno."))),
		source("pagos", added(block("Baja (prov:30:2)", "Dos."))))
	plan := mustPlan(t, in)
	if got := definitiveOf(t, plan, "prov:30:1"); got != "REQ-30.3" {
		t.Errorf("primera capacidad = %s, quería REQ-30.3", got)
	}
	if got := definitiveOf(t, plan, "prov:30:2"); got != "REQ-30.4" {
		t.Errorf("segunda capacidad = %s, quería REQ-30.4", got)
	}
	for _, e := range plan.Entries {
		if e.State != StateAssigned {
			t.Errorf("estado de %s = %q", e.Provisional, e.State)
		}
	}
	if plan.Map.Substitute("Cubre: prov:30:1 y prov:30:2") != "Cubre: REQ-30.3 y REQ-30.4" {
		t.Errorf("el mapa no sustituye por token completo")
	}
}

func TestPlanIDsOrderFollowsAppearanceNotTheOrdinal(t *testing.T) {
	in := idInput(t, reqid.Default(), map[string]string{"specs/pagos/spec.md": pagosCanon},
		source("pagos", added(block("Alta (prov:30:2)", "Uno."), block("Baja (prov:30:1)", "Dos."))))
	plan := mustPlan(t, in)
	if definitiveOf(t, plan, "prov:30:2") != "REQ-30.3" || definitiveOf(t, plan, "prov:30:1") != "REQ-30.4" {
		t.Fatalf("entradas = %+v", plan.Entries)
	}
}

func TestPlanIDsOccupationCountsRegistryOwnDefinitiveActiveChangesAndTrace(t *testing.T) {
	files := map[string]string{
		"specs/pagos/spec.md":                   pagosCanon,
		"changes/otro/specs/pagos/spec.md":      "## ADDED Requirements\n\n### Requirement: Ajeno (REQ-30.9)\n\nCuerpo.\n",
		"changes/propio/specs/pagos/spec.md":    "## ADDED Requirements\n\n### Requirement: Propio (REQ-30.20)\n\nCuerpo.\n",
		"specs/facturacion/spec.md":             "# E\n\n## Requirements\n\n### Requirement: Hist (REQ-31.4)\n\nCuerpo.\n",
		"changes/archive/viejo/specs/x/spec.md": "## ADDED Requirements\n\n### Requirement: Arch (REQ-30.50)\n\nCuerpo.\n",
	}
	t.Run("registro y cambios activos ajenos, sin el propio ni el archivo", func(t *testing.T) {
		in := idInput(t, reqid.Default(), files, source("pagos", added(block("Alta (prov:30:1)", "Cuerpo."))))
		if got := definitiveOf(t, mustPlan(t, in), "prov:30:1"); got != "REQ-30.10" {
			t.Errorf("= %s, quería REQ-30.10 (ocupado hasta 30.9 por un cambio activo)", got)
		}
	})
	t.Run("definitivos propios", func(t *testing.T) {
		in := idInput(t, reqid.Default(), files, source("pagos", added(block("Propio (REQ-30.15)", "Uno."), block("Alta (prov:30:1)", "Dos."))))
		if got := definitiveOf(t, mustPlan(t, in), "prov:30:1"); got != "REQ-30.16" {
			t.Errorf("= %s, quería REQ-30.16", got)
		}
	})
	t.Run("definitivos de la traza", func(t *testing.T) {
		in := idInput(t, reqid.Default(), files,
			source("pagos", added(block("Alta (prov:30:1)", "Uno."), block("Baja (prov:30:2)", "Dos."))),
			source("facturacion", added(block("Otra (prov:30:3)", "Tres."))))
		in.Trace = Trace{Schema: traceSchema, Pattern: "REQ-<n>.<m>", Assignments: []TraceEntry{
			{Provisional: "prov:30:3", Definitive: "REQ-30.12", Capability: "facturacion", Requirement: "Otra (REQ-30.12)"},
		}}
		plan := mustPlan(t, in)
		if definitiveOf(t, plan, "prov:30:1") != "REQ-30.13" || definitiveOf(t, plan, "prov:30:2") != "REQ-30.14" {
			t.Errorf("entradas = %+v", plan.Entries)
		}
	})
}

func TestPlanIDsAnEmptySeriesStartsAtOneAndANewSeriesNeverLandsOnAFixedOne(t *testing.T) {
	files := map[string]string{"specs/pagos/spec.md": pagosCanon}
	in := idInput(t, reqid.Default(), files,
		source("pagos", added(
			block("Con etiqueta (prov:a:1)", "Uno."),
			block("Con serie fijada (prov:31:1)", "Dos."),
			block("Otra etiqueta (prov:b:1)", "Tres."),
			block("Misma etiqueta (prov:a:2)", "Cuatro."),
			block("Serie vacía (prov:40:1)", "Cinco."))))
	plan := mustPlan(t, in)
	want := map[string]string{
		"prov:31:1": "REQ-31.1", // serie fijada: asignada antes que las etiquetas
		"prov:40:1": "REQ-40.1", // serie vacía
		"prov:a:1":  "REQ-41.1", // la serie nueva no recae en 31 ni en 40
		"prov:a:2":  "REQ-41.2", // la misma etiqueta comparte serie
		"prov:b:1":  "REQ-42.1",
	}
	for provisional, definitive := range want {
		if got := definitiveOf(t, plan, provisional); got != definitive {
			t.Errorf("%s = %s, quería %s", provisional, got, definitive)
		}
	}
}

func TestPlanIDsSeriesWorkWithTheDomainPattern(t *testing.T) {
	p, err := reqid.Parse("RF-<DOM>-<NN>")
	if err != nil {
		t.Fatal(err)
	}
	in := idInput(t, p, map[string]string{"specs/usuarios/spec.md": "# E\n\n## Requirements\n\n### Requirement: Alta (RF-USU-02)\n\nCuerpo.\n"},
		source("usuarios", added(block("Baja (prov:USU:1)", "Uno."), block("Cobro (prov:FAC:1)", "Dos."))))
	plan := mustPlan(t, in)
	if definitiveOf(t, plan, "prov:USU:1") != "RF-USU-03" || definitiveOf(t, plan, "prov:FAC:1") != "RF-FAC-01" {
		t.Fatalf("entradas = %+v", plan.Entries)
	}
}

func TestPlanIDsTraceEntryKeepsItsDefinitiveWhenTheOrderChanges(t *testing.T) {
	in := idInput(t, reqid.Default(), map[string]string{"specs/pagos/spec.md": pagosCanon},
		source("aaa", added(block("Nueva (prov:30:9)", "Uno."))),
		source("pagos", added(block("Alta (prov:30:1)", "Dos."))))
	in.Trace = Trace{Schema: traceSchema, Pattern: "REQ-<n>.<m>", Assignments: []TraceEntry{
		{Provisional: "prov:30:1", Definitive: "REQ-30.3", Capability: "pagos", Requirement: "Alta (REQ-30.3)"},
	}}
	plan := mustPlan(t, in)
	if definitiveOf(t, plan, "prov:30:1") != "REQ-30.3" {
		t.Errorf("la entrada de la traza perdió su definitivo: %+v", plan.Entries)
	}
	if definitiveOf(t, plan, "prov:30:9") != "REQ-30.4" {
		t.Errorf("la capacidad nueva debía tomar el siguiente libre: %+v", plan.Entries)
	}
}

func TestPlanIDsDeletedTraceDegradesToDefinitivesChosenByTheAuthor(t *testing.T) {
	canon := pagosCanon + "\n### Requirement: Alta (REQ-30.3)\n\nEl sistema DEBE dar de alta.\n"
	in := idInput(t, reqid.Default(), map[string]string{"specs/pagos/spec.md": canon},
		source("pagos", added(block("Alta (REQ-30.3)", "El sistema DEBE dar de alta."))))
	plan := mustPlan(t, in)
	if len(plan.Entries) != 0 || plan.Map.Count() != 0 {
		t.Fatalf("sin traza y con definitivo no hay asignaciones: %+v", plan)
	}
}

func TestPlanIDsAddedWithAnotherBodyAndMissingCapabilityKeepTheirTypedErrors(t *testing.T) {
	files := map[string]string{"specs/pagos/spec.md": pagosCanon}
	in := idInput(t, reqid.Default(), files, source("pagos", added(block("Cobro (REQ-30.1)", "Otro cuerpo."))))
	_, err := PlanIDs(in)
	var diverged *DivergentRequirementError
	if !errors.As(err, &diverged) {
		t.Fatalf("error = %v, quería *DivergentRequirementError", err)
	}
	in = idInput(t, reqid.Default(), files, source("nueva", "## MODIFIED Requirements\n\n"+block("Algo (REQ-50.1)", "Cuerpo.")))
	_, err = PlanIDs(in)
	var missing *MissingCapabilityError
	if !errors.As(err, &missing) {
		t.Fatalf("error = %v, quería *MissingCapabilityError", err)
	}
}

func TestPlanIDsDefinitiveAddedCollisionsKeepTheUsualRejection(t *testing.T) {
	files := map[string]string{"specs/pagos/spec.md": pagosCanon}
	in := idInput(t, reqid.Default(), files, source("facturacion", added(block("Nuevo (REQ-30.2)", "Cuerpo."))))
	_, err := PlanIDs(in)
	var collision *CollisionError
	if !errors.As(err, &collision) || collision.ID != "REQ-30.2" || collision.NextFree != "REQ-30.3" {
		t.Fatalf("error = %v", err)
	}
	// Dos capacidades con el mismo definitivo en el propio cambio.
	in = idInput(t, reqid.Default(), nil,
		source("a", added(block("Uno (REQ-60.1)", "Cuerpo."))), source("b", added(block("Dos (REQ-60.1)", "Cuerpo."))))
	_, err = PlanIDs(in)
	if !errors.As(err, &collision) || collision.ID != "REQ-60.1" || !strings.Contains(collision.Where, "a y b") {
		t.Fatalf("duplicado propio = %v", err)
	}
}

func TestPlanIDsMissingTitlelessMODIFIEDStaysPendingForTheComposition(t *testing.T) {
	// Un MODIFIED sin ID que no casa no lo rechaza el plan: sigue pendiente y lo rechaza la
	// composición con el motivo vigente (REQ-47.3 regla 2).
	in := idInput(t, reqid.Default(), map[string]string{"specs/pagos/spec.md": pagosCanon},
		source("pagos", "## MODIFIED Requirements\n\n"+block("Sin coincidencia", "Cuerpo.")))
	if _, err := PlanIDs(in); err != nil {
		t.Fatalf("PlanIDs: %v", err)
	}
}

func TestPlanIDsResolvesSupersededNamesWithAndWithoutPrefix(t *testing.T) {
	files := map[string]string{
		"specs/pagos/spec.md":       "# E\n\n## Requirements\n\n### Requirement: Resumen\n\nUno.\n\n### Requirement: Único\n\nDos.\n",
		"specs/facturacion/spec.md": "# E\n\n## Requirements\n\n### Requirement: Resumen\n\nTres.\n",
	}
	in := idInput(t, reqid.Default(), files,
		source("pagos", added(block("Alta (prov:30:1)", "Cuerpo."))), source("facturacion", added(block("Baja (prov:31:1)", "Cuerpo."))))
	in.Superseded = []string{"facturacion/Resumen", "Único"}
	plan := mustPlan(t, in)
	if got := plan.Superseded["facturacion"]; len(got) != 1 || got[0] != "Resumen" {
		t.Errorf("facturacion = %v", got)
	}
	if got := plan.Superseded["pagos"]; len(got) != 1 || got[0] != "Único" {
		t.Errorf("pagos = %v", got)
	}
}

// reintegration monta el caso de reintegración: la traza da P→D para una ADDED de pagos.
func reintegration(t *testing.T, files map[string]string, delta string, entry TraceEntry, extra ...CapabilitySource) IDPlanInput {
	t.Helper()
	caps := append([]CapabilitySource{source("pagos", delta)}, extra...)
	in := idInput(t, reqid.Default(), files, caps...)
	entry.Capability = "pagos"
	in.Trace = Trace{Schema: traceSchema, Pattern: "REQ-<n>.<m>", Assignments: []TraceEntry{entry}}
	return in
}

func TestPlanIDsKeepsATraceDefinitiveThatIsFreeAndOneAlreadyIntegratedWithTheSameBody(t *testing.T) {
	entry := TraceEntry{Provisional: "prov:30:1", Definitive: "REQ-30.5", Requirement: "Alta (REQ-30.5)"}
	t.Run("D libre", func(t *testing.T) {
		in := reintegration(t, map[string]string{"specs/pagos/spec.md": pagosCanon}, added(block("Alta (prov:30:1)", "Cuerpo.")), entry)
		plan := mustPlan(t, in)
		if plan.Entries[0].State != StateKept || plan.Entries[0].Definitive != "REQ-30.5" {
			t.Fatalf("entradas = %+v", plan.Entries)
		}
		if plan.Map.Substitute("prov:30:1") != "REQ-30.5" {
			t.Error("el mapa no lleva P→D")
		}
	})
	t.Run("ya integrado con el mismo cuerpo", func(t *testing.T) {
		canon := pagosCanon + "\n### Requirement: Alta (REQ-30.5)\n\nCuerpo.\n"
		in := reintegration(t, map[string]string{"specs/pagos/spec.md": canon}, added(block("Alta (prov:30:1)", "Cuerpo.")), entry)
		plan := mustPlan(t, in)
		if plan.Entries[0].State != StateKept || plan.Entries[0].Definitive != "REQ-30.5" || len(plan.Canonical) != 0 {
			t.Fatalf("plan = %+v", plan)
		}
	})
	t.Run("rescrita con el definitivo", func(t *testing.T) {
		canon := pagosCanon + "\n### Requirement: Alta (REQ-30.5)\n\nCuerpo.\n"
		in := reintegration(t, map[string]string{"specs/pagos/spec.md": canon}, added(block("Alta (REQ-30.5)", "Cuerpo.")), entry)
		if plan := mustPlan(t, in); plan.Entries[0].State != StateKept {
			t.Fatalf("entradas = %+v", plan.Entries)
		}
	})
}

func TestPlanIDsRejectsTheEditedOwnBlockInsteadOfReassigningIt(t *testing.T) {
	canon := pagosCanon + "\n### Requirement: Alta (REQ-30.5)\n\nCuerpo original.\n"
	entry := TraceEntry{Provisional: "prov:30:1", Definitive: "REQ-30.5", Requirement: "Alta (REQ-30.5)"}
	for name, delta := range map[string]string{
		"con el provisional": added(block("Alta (prov:30:1)", "Cuerpo editado.")),
		"con el definitivo":  added(block("Alta (REQ-30.5)", "Cuerpo editado.")),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := PlanIDs(reintegration(t, map[string]string{"specs/pagos/spec.md": canon}, delta, entry))
			var diverged *DivergentRequirementError
			if !errors.As(err, &diverged) || diverged.Capability != "pagos" || diverged.ID != "REQ-30.5" {
				t.Fatalf("error = %v, quería *DivergentRequirementError", err)
			}
		})
	}
}

func TestPlanIDsReassignsAnIndividualEntryWhenItsDefinitiveIsTakenByAnotherRequirement(t *testing.T) {
	files := map[string]string{
		"specs/pagos/spec.md":       pagosCanon,
		"specs/facturacion/spec.md": "# E\n\n## Requirements\n\n### Requirement: Ajeno (REQ-30.5)\n\nOtro.\n",
	}
	entry := TraceEntry{Provisional: "prov:30:1", Definitive: "REQ-30.5", Requirement: "Alta (REQ-30.5)"}
	plan := mustPlan(t, reintegration(t, files, added(block("Alta (prov:30:1)", "Cuerpo que cita REQ-30.5.")), entry))
	got := plan.Entries[0]
	if got.State != StateReassigned || got.Definitive != "REQ-30.6" || len(got.Previous) != 1 || got.Previous[0] != "REQ-30.5" {
		t.Fatalf("entrada = %+v", got)
	}
	if plan.Map.Substitute("REQ-30.5 y prov:30:1") != "REQ-30.6 y REQ-30.6" {
		t.Errorf("el mapa debe llevar D→D' y P→D': %q", plan.Map.Substitute("REQ-30.5 y prov:30:1"))
	}
}

func TestPlanIDsReassignsOurBlockInTheLivingSpecWhenItsDefinitiveIsAlsoElsewhere(t *testing.T) {
	canon := pagosCanon + "\n### Requirement: Alta (REQ-30.5)\n\nCuerpo.\n"
	files := map[string]string{
		"specs/pagos/spec.md":       canon,
		"specs/facturacion/spec.md": "# E\n\n## Requirements\n\n### Requirement: Ajeno (REQ-30.5)\n\nOtro.\n",
	}
	entry := TraceEntry{Provisional: "prov:30:1", Definitive: "REQ-30.5", Requirement: "Alta (REQ-30.5)"}
	plan := mustPlan(t, reintegration(t, files, added(block("Alta (REQ-30.5)", "Cuerpo.")), entry))
	if plan.Entries[0].State != StateReassigned || plan.Entries[0].Definitive != "REQ-30.6" {
		t.Fatalf("entrada = %+v", plan.Entries[0])
	}
	edited := plan.Canonical["pagos"]
	if !strings.Contains(edited, "### Requirement: Alta (REQ-30.6)") || strings.Contains(edited, "REQ-30.5") || !strings.Contains(edited, "Cobro (REQ-30.1)") {
		t.Fatalf("la canónica editada no es la esperada:\n%s", edited)
	}
}

func TestPlanIDsReassignsAWholeNewSeriesLabelOnCollision(t *testing.T) {
	files := map[string]string{
		"specs/pagos/spec.md":       pagosCanon,
		"specs/facturacion/spec.md": "# E\n\n## Requirements\n\n### Requirement: Ajeno (REQ-32.1)\n\nOtro.\n",
	}
	delta := added(block("Alta (prov:a:1)", "Uno."), block("Baja (prov:a:2)", "Dos."))
	in := idInput(t, reqid.Default(), files, source("pagos", delta))
	in.Trace = Trace{Schema: traceSchema, Pattern: "REQ-<n>.<m>", Assignments: []TraceEntry{
		{Provisional: "prov:a:1", Definitive: "REQ-32.1", Capability: "pagos", Requirement: "Alta (REQ-32.1)"},
		{Provisional: "prov:a:2", Definitive: "REQ-32.2", Capability: "pagos", Requirement: "Baja (REQ-32.2)"},
	}}
	plan := mustPlan(t, in)
	if len(plan.Entries) != 2 {
		t.Fatalf("entradas = %+v", plan.Entries)
	}
	for i, want := range []struct{ definitive, previous string }{{"REQ-33.1", "REQ-32.1"}, {"REQ-33.2", "REQ-32.2"}} {
		got := plan.Entries[i]
		if got.State != StateReassigned || got.Definitive != want.definitive || len(got.Previous) != 1 || got.Previous[0] != want.previous {
			t.Errorf("entrada %d = %+v, quería %+v", i, got, want)
		}
	}
}

func TestPlanIDsTraceEntryWithoutExactlyOneAddedIsStale(t *testing.T) {
	entry := TraceEntry{Provisional: "prov:30:1", Definitive: "REQ-30.5", Requirement: "Alta (REQ-30.5)"}
	files := map[string]string{"specs/pagos/spec.md": pagosCanon}
	t.Run("ninguna", func(t *testing.T) {
		_, err := PlanIDs(reintegration(t, files, added(block("Otra (prov:30:2)", "Cuerpo.")), entry))
		wantPlanError(t, err, CodeIDTraceStale)
	})
	t.Run("dos", func(t *testing.T) {
		_, err := PlanIDs(reintegration(t, files, added(block("Alta (REQ-30.5)", "Uno."), block("Alta bis (prov:30:1)", "Dos.")), entry))
		wantPlanError(t, err, CodeIDTraceStale)
	})
	t.Run("capacidad ausente", func(t *testing.T) {
		in := reintegration(t, files, added(block("Alta (prov:30:1)", "Cuerpo.")), entry)
		in.Trace.Assignments[0].Capability = "otra"
		_, err := PlanIDs(in)
		wantPlanError(t, err, CodeIDTraceStale)
	})
	t.Run("patrón distinto", func(t *testing.T) {
		in := reintegration(t, files, added(block("Alta (prov:30:1)", "Cuerpo.")), entry)
		in.Trace.Pattern = "RF-<DOM>-<NN>"
		_, err := PlanIDs(in)
		wantPlanError(t, err, CodeIDTracePatternMismatch)
	})
}

func TestCheckDuplicatesCountsWrittenIDsAcrossTheResultingSpecs(t *testing.T) {
	files := map[string]string{
		"specs/pagos/spec.md":       pagosCanon,
		"specs/facturacion/spec.md": "# E\n\n## Requirements\n\n### Requirement: Ajeno (REQ-30.5)\n\nOtro.\n",
		"specs/hist-a/spec.md":      "# E\n\n## Requirements\n\n### Requirement: A (REQ-1.1)\n\nA.\n",
		"specs/hist-b/spec.md":      "# E\n\n## Requirements\n\n### Requirement: B (REQ-1.1)\n\nB.\n",
	}
	in := idInput(t, reqid.Default(), files)
	resulting := map[string]string{"pagos": pagosCanon + "\n### Requirement: Alta (REQ-30.5)\n\nCuerpo.\n"}

	dups, err := checkDuplicates(in.Registry, resulting, []string{"REQ-30.5"})
	planErr := wantPlanError(t, err, CodeDuplicateAssignedID)
	if planErr.Token != "REQ-30.5" {
		t.Errorf("Token = %q", planErr.Token)
	}
	var blocking, historical *Duplicate
	for i := range dups {
		if dups[i].ID == "REQ-30.5" {
			blocking = &dups[i]
		}
		if dups[i].ID == "REQ-1.1" {
			historical = &dups[i]
		}
	}
	if blocking == nil || !blocking.Blocking || strings.Join(blocking.Capabilities, ",") != "facturacion,pagos" {
		t.Errorf("bloqueante = %+v", blocking)
	}
	if historical == nil || historical.Blocking {
		t.Errorf("el histórico ajeno debe informarse con blocking false: %+v", historical)
	}

	dups, err = checkDuplicates(in.Registry, map[string]string{"pagos": pagosCanon + "\n### Requirement: Alta (REQ-30.6)\n\nCuerpo.\n"}, []string{"REQ-30.6"})
	if err != nil {
		t.Fatalf("sin duplicados propios no hay error: %v", err)
	}
	if len(dups) != 1 || dups[0].ID != "REQ-1.1" || dups[0].Blocking {
		t.Errorf("duplicados = %+v", dups)
	}
}

// treeRoot es el planning home del registro de la prueba: tres niveles sobre `spec.md`.
func treeRoot(in IDPlanInput) string {
	return filepath.Dir(filepath.Dir(filepath.Dir(in.Capabilities[0].Path)))
}
