package specbaseline

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Las pruebas de RenderTrace (inc-24, REQ-24.52, D-13 paso 6, design §4.5). Todos
// los textos son inventados.

func tracePlan() Plan {
	plan := samplePlan()
	plan.Entries = append(plan.Entries, TraceEntry{
		ID: "Flujo | de devolución", Kind: EntryRegister, Domain: "flujos-de-negocio", Document: "04_Flujos_Negocio.md",
		Anchor: "inc-99-ejemplo", Action: ActionAdded, State: StatePending,
	})
	plan.Entries[0].Warnings = []string{"la sección «Anexos» sigue al requisito como cola contigua"}
	return plan
}

func TestRenderTraceMarkdownHasTheEntryTableWithTheProvisionalToDefinitiveID(t *testing.T) {
	data, err := RenderTrace(tracePlan(), TraceFormatMarkdown)
	if err != nil {
		t.Fatalf("RenderTrace: %v", err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "\n## Traza de integración\n\n") {
		t.Errorf("empieza por una línea en blanco y el encabezado de la sección:\n%q", text[:40])
	}
	for _, want := range []string{
		"Cambio: `inc-99-ejemplo`. Preparada: 2026-10-04T10:00:00Z. Verificada: pendiente.",
		"| Identificador | Documento | Ancla | Acción | Estado |\n|---|---|---|---|---|\n",
		"| RF-9001 → RF-008 | 01_Requisitos_Funcionales.md | RF-008 | added | pending |",
		"| Reintento de pago | 03_Modelo_Datos.md | inc-99-ejemplo | added | pending |",
		"| C1 | ../repo-backend/docs/API.md | Errores > Reintentos | modified | pending |",
		`| Flujo \| de devolución | 04_Flujos_Negocio.md | inc-99-ejemplo | added | pending |`,
		"### Escenarios omitidos\n\n- RF-008 (dominio requisitos-funcionales)\n",
		"### Avisos\n\n- RF-008: la sección «Anexos» sigue al requisito como cola contigua\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("la traza debe contener %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Decisión sobre el contexto técnico") {
		t.Errorf("sin «Sin impacto» no hay sección de decisión:\n%s", text)
	}
}

func TestRenderTraceMarkdownShowsTheVerificationDate(t *testing.T) {
	plan := tracePlan()
	verified := time.Date(2026, 10, 5, 8, 30, 0, 0, time.FixedZone("CEST", 2*3600))
	plan.VerifiedAt = &verified
	data, err := RenderTrace(plan, TraceFormatMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Verificada: 2026-10-05T06:30:00Z.") {
		t.Errorf("las fechas se muestran en UTC:\n%s", data)
	}
}

func TestRenderTraceMarkdownShowsTheNoImpactDecisionWithItsReason(t *testing.T) {
	plan := tracePlan()
	plan.Entries = nil
	plan.IDMap = nil
	plan.Context = ContextDecision{Decision: ContextNoImpact, Reason: "el cambio solo corrige una errata en un mensaje de error"}
	data, err := RenderTrace(plan, TraceFormatMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"La integración no registra entradas.",
		"### Decisión sobre el contexto técnico\n\nSin impacto: el cambio solo corrige una errata en un mensaje de error\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("la traza debe contener %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "| Identificador |") || strings.Contains(text, "Escenarios omitidos") || strings.Contains(text, "Avisos") {
		t.Errorf("sin entradas no hay tabla ni secciones vacías:\n%s", text)
	}
}

func TestRenderTraceKeepsAnEntryWhoseProvisionalEqualsItsDefinitiveWithoutAnArrow(t *testing.T) {
	plan := tracePlan()
	plan.Entries = []TraceEntry{{ID: "RF-008", Provisional: "RF-008", Kind: EntryCompose, Document: "01_Requisitos_Funcionales.md", Anchor: "RF-008", Action: ActionAdded, State: StateIntegrated}}
	data, err := RenderTrace(plan, TraceFormatMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	if text := string(data); strings.Contains(text, "→") || !strings.Contains(text, "| RF-008 | 01_Requisitos_Funcionales.md | RF-008 | added | integrated |") {
		t.Errorf("un provisional igual a su definitivo se muestra sin flecha:\n%s", text)
	}
}

func TestRenderTraceJSON(t *testing.T) {
	plan := tracePlan()
	verified := time.Date(2026, 10, 5, 8, 30, 0, 0, time.UTC)
	plan.VerifiedAt = &verified
	plan.Context = ContextDecision{Decision: ContextNoImpact, Reason: "sin cambios"}
	data, err := RenderTrace(plan, TraceFormatJSON)
	if err != nil {
		t.Fatalf("RenderTrace: %v", err)
	}
	if !strings.HasSuffix(string(data), "}\n") {
		t.Errorf("el JSON termina en un salto de línea: %q", data[len(data)-5:])
	}
	var view TraceView
	if err := json.Unmarshal(data, &view); err != nil {
		t.Fatalf("la salida no es JSON: %v\n%s", err, data)
	}
	if TraceSchema != "axiom.sdd-integration-trace/v1" || view.Schema != TraceSchema || view.Change != "inc-99-ejemplo" {
		t.Errorf("vista = %+v", view)
	}
	if view.VerifiedAt == nil || !view.VerifiedAt.Equal(verified) || !view.PreparedAt.Equal(plan.PreparedAt) {
		t.Errorf("fechas = %v %v", view.PreparedAt, view.VerifiedAt)
	}
	if view.Context != (TraceContext{Decision: ContextNoImpact, Reason: "sin cambios"}) {
		t.Errorf("contexto = %+v", view.Context)
	}
	if len(view.IDMap) != 1 || view.IDMap[0].Definitive != "RF-008" || len(view.Entries) != 4 {
		t.Errorf("correspondencia y entradas = %+v %+v", view.IDMap, view.Entries)
	}
	if len(view.Warnings) != 1 || !strings.HasPrefix(view.Warnings[0], "RF-008: ") {
		t.Errorf("avisos = %v", view.Warnings)
	}
	if strings.Contains(string(data), "repositories") || strings.Contains(string(data), "sha256") {
		t.Errorf("la traza no incluye la línea base de git ni las huellas:\n%s", data)
	}
}

func TestRenderTraceJSONListsAnEmptyTraceAsEmptyArrays(t *testing.T) {
	plan := tracePlan()
	plan.Entries, plan.IDMap = nil, nil
	data, err := RenderTrace(plan, TraceFormatJSON)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"idMap": []`, `"entries": []`, `"warnings": []`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("el JSON debe contener %s:\n%s", want, data)
		}
	}
}

func TestRenderTraceRejectsAnUnknownFormat(t *testing.T) {
	if _, err := RenderTrace(tracePlan(), "yaml"); err == nil || !strings.Contains(err.Error(), `"yaml"`) || !strings.Contains(err.Error(), "markdown") || !strings.Contains(err.Error(), "json") {
		t.Errorf("error = %v", err)
	}
}

// La traza de un plan verificado recorre el ciclo completo: lo que Verify marca es
// lo que RenderTrace muestra.
func TestRenderTraceOfAVerifiedPlanShowsTheIntegratedStates(t *testing.T) {
	f, target, _ := integratedFixture(t)
	mustPass(t, verifyOf(t, target))
	data, err := RenderTrace(loadPlan(t, f), TraceFormatMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"| Entidad Reintento | 03_Modelo_Datos.md | inc-99-ejemplo | added | integrated |",
		"| C1 | context/ARQUITECTURA.md | Componentes > Servicio de pedidos | modified | integrated |",
		"| C2 | ../repo-backend/docs/API.md | Errores | added | integrated |",
		"| Flujo de devolución | 04_Flujos_Negocio.md | inc-99-ejemplo | added | pending |",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("la traza debe contener %q:\n%s", want, text)
		}
	}
	var composeRow bool
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "| RF-9001 → RF-008 | 01_Requisitos_Funcionales.md | ") && strings.HasSuffix(line, " | added | integrated |") {
			composeRow = true
		}
	}
	if !composeRow {
		t.Errorf("la traza lleva la correspondencia RF-9001 → RF-008 del ADDED integrado:\n%s", text)
	}
	if strings.Contains(text, "Verificada: pendiente") {
		t.Errorf("verify escribe verified_at:\n%s", text)
	}
}
