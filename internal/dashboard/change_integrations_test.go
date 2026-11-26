package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Estas pruebas leen ficheros de integraciones de un cambio de un proyecto temporal por handler()
// (la cadena de seguridad completa) con Host local. Ninguna llama a un tracker: la pestaña solo lee.
// Sin t.Parallel: comparten el catálogo de manifiestos del proceso.

const changeSentinel = "CENTINELA-OUTBOX-5b2e"

const changeLinksYAML = `schema: axiom.integrations/v2
change: c
links:
  - plugin: fake
    kind: work_item
    id: "4521"
    url: https://dev.example.test/wi/4521
    source: change.start/c
    recorded_at: 2026-11-18T10:00:00Z
  - plugin: fake
    kind: work_item
    id: "4522"
    url: https://dev.example.test/wi/4522 con espacio
    source: change.start/c2
    recorded_at: 2026-11-18T10:00:00Z
  - plugin: viejo
    kind: work_item
    id: "77"
    url: https://viejo.example.test/77
    source: change.start/c3
    recorded_at: 2026-11-18T10:00:00Z
  - plugin: fake
    kind: task
    id: "9001"
    role: web
    estimate_hours: 6.5
    source: tasks.create
    recorded_at: 2026-11-18T10:05:00Z
time_entries:
  - plugin: fake
    role: web
    item_id: "9001"
    minutes: 90
    date: 2026-11-18
    source: role.close/web
    recorded_at: 2026-11-18T12:00:00Z
    actor: Ana <ana@example.test>
  - plugin: fake
    role: web
    item_id: "9001"
    minutes: 30
    date: 2026-11-19
    source: time.record/1
    recorded_at: 2026-11-19T12:00:00Z
`

const changePendingYAML = `schema: axiom.integrations-time-pending/v1
change: c
pending:
  - role: web
    item_id: "9001"
    hours: 1
    minutes: 60
    actor: PENDIENTE-ACTOR
    date: 2026-11-20
    source: PENDIENTE-FUENTE
    base_rev: 3
    expected_completed: 1
    expected_remaining: 2
`

const changeOutboxYAML = `schema: axiom.integrations-outbox/v1
change: c
operations:
  - key: ` + changeSentinel + `/1
    plugin: fake
    state: pending
    attempts: 2
    enqueued_at: 2026-11-18T22:30:00Z
    last_error: ` + changeSentinel + ` error crudo
    event:
      type: gate.spec.approved
      change: c
      gate: spec
      actor: ` + changeSentinel + `
      occurred_at: 2026-11-18T22:30:00Z
      artifact_digest: ` + changeSentinel + `
      idempotency_key: ` + changeSentinel + `/1
`

// changeTabEnv monta un proyecto con el cambio activo c y los ficheros dados (ruta relativa a la
// carpeta del cambio) y devuelve el manejador completo y la carpeta del cambio.
func changeTabEnv(t *testing.T, files map[string]string) (http.Handler, string, workspacetest.Workspace) {
	t.Helper()
	integrationstest.Install(t, integrationstest.NewFake())
	ws := embeddedProject(t, hub.WorktreeModeInPlace)
	root := filepath.Join(ws.Home, "changes", "c")
	workspacetest.WriteFile(t, filepath.Join(root, "proposal.md"), "# c\n")
	for rel, content := range files {
		workspacetest.WriteFile(t, filepath.Join(root, filepath.FromSlash(rel)), content)
	}
	return NewServer(NewService(ws.Master)).handler(), root, ws
}

func getChangeTab(t *testing.T, h http.Handler, target string) (int, ChangeIntegrationsDTO, string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request.Host = "127.0.0.1"
	h.ServeHTTP(recorder, request)
	var dto ChangeIntegrationsDTO
	if recorder.Code == http.StatusOK {
		if err := json.Unmarshal(recorder.Body.Bytes(), &dto); err != nil {
			t.Fatalf("respuesta no es JSON: %v", err)
		}
	}
	if cache := recorder.Header().Get("Cache-Control"); cache != "no-store" {
		t.Errorf("Cache-Control = %q, quiero no-store", cache)
	}
	return recorder.Code, dto, recorder.Body.String()
}

func TestChangeIntegrationsShowsLinksTasksTimeAndPending(t *testing.T) {
	h, _, _ := changeTabEnv(t, map[string]string{
		"integrations.yaml":           changeLinksYAML,
		integrationsPendingFileForTab: changePendingYAML,
	})
	code, dto, _ := getChangeTab(t, h, "/api/integrations/change?name=c")
	if code != http.StatusOK {
		t.Fatalf("estado = %d", code)
	}
	if len(dto.Errors) != 0 || dto.Empty || dto.Archived {
		t.Fatalf("errores/empty/archived inesperados: %+v", dto)
	}
	if len(dto.WorkItems) != 3 {
		t.Fatalf("work items = %d, quiero 3", len(dto.WorkItems))
	}
	if first := dto.WorkItems[0]; first.ID != "4521" || first.URL != "https://dev.example.test/wi/4521" || first.PluginTitle == "fake" {
		t.Errorf("primer work item: %+v", first)
	}
	if len(dto.Roles) != 1 || dto.Roles[0].Role != "web" || dto.Roles[0].Task == nil {
		t.Fatalf("roles: %+v", dto.Roles)
	}
	role := dto.Roles[0]
	if role.Task.EstimateHours == nil || *role.Task.EstimateHours != 6.5 || len(role.Entries) != 2 {
		t.Errorf("task o imputaciones: %+v", role)
	}
	if role.Entries[0].Actor == "" || role.Entries[1].Actor != "" {
		t.Errorf("actores de imputación: %+v", role.Entries)
	}
	if role.Pending == nil || role.Pending.Minutes != 60 || role.Pending.Date != "2026-11-20" || role.Pending.ItemID != "9001" {
		t.Errorf("pendiente: %+v", role.Pending)
	}
}

func TestChangeIntegrationsOmitsUnsafeURLAndKeepsHistoricLinks(t *testing.T) {
	h, _, _ := changeTabEnv(t, map[string]string{"integrations.yaml": changeLinksYAML})
	_, dto, _ := getChangeTab(t, h, "/api/integrations/change?name=c")
	byID := map[string]ChangeLinkDTO{}
	for _, l := range dto.WorkItems {
		byID[l.ID] = l
	}
	if unsafe := byID["4522"]; unsafe.URL != "" || unsafe.ID != "4522" {
		t.Errorf("la URL con espacio debe omitirse y conservar el vínculo: %+v", unsafe)
	}
	if old := byID["77"]; old.PluginTitle != "viejo" || old.URL == "" {
		t.Errorf("vínculo histórico sin manifiesto: %+v", old)
	}
}

func TestSafeExternalURL(t *testing.T) {
	good := []string{"https://a.b/x", "http://a.b", "HTTPS://A.B/x", "https://a.b:8080/x?y=1#z"}
	bad := []string{
		"", "javascript:alert(1)", "data:text/html,x", "//a.b/x", "/ruta", "ftp://a.b/x",
		"https://u:p@a.b/x", "https://u@a.b/x", " https://a.b", "https://a.b ", "https://a.b/x y",
		"https://a.b/\tx", "https://a.b/\nx", "https:///x", "https://", "http://%zz",
	}
	for _, raw := range good {
		if !safeExternalURL(raw) {
			t.Errorf("safeExternalURL(%q) = false, quiero true", raw)
		}
	}
	for _, raw := range bad {
		if safeExternalURL(raw) {
			t.Errorf("safeExternalURL(%q) = true, quiero false", raw)
		}
	}
}

func TestChangeIntegrationsRoleWithoutTaskAndWithoutKickoff(t *testing.T) {
	// Roster con dos roles; solo web tiene Task. Sin kickoff.yaml no hay roster: la pestaña lo
	// tolera («sin sellar»); aquí se sella para comprobar «sin Task» y luego se compara.
	h, root, _ := changeTabEnv(t, map[string]string{"integrations.yaml": changeLinksYAML})
	_, dto, _ := getChangeTab(t, h, "/api/integrations/change?name=c")
	if dto.Kickoff.Sealed || len(dto.Errors) != 0 {
		t.Fatalf("un cambio sin kickoff.yaml no es un error: %+v", dto)
	}
	sealChange(t, root, "Ana <ana@example.test>", "api", "web")
	_, dto, _ = getChangeTab(t, h, "/api/integrations/change?name=c")
	if !dto.Kickoff.Sealed || dto.Kickoff.SealedBy != "Ana <ana@example.test>" {
		t.Errorf("kickoff: %+v", dto.Kickoff)
	}
	var api *ChangeRoleDTO
	for i := range dto.Roles {
		if dto.Roles[i].Role == "api" {
			api = &dto.Roles[i]
		}
	}
	if api == nil || api.Task != nil || len(api.Entries) != 0 || api.Pending != nil {
		t.Errorf("el rol api debe aparecer sin Task: %+v", dto.Roles)
	}
}

func TestChangeIntegrationsSealedWithoutActorIsNotAnError(t *testing.T) {
	h, root, _ := changeTabEnv(t, nil)
	sealChange(t, root, "", "web")
	_, dto, _ := getChangeTab(t, h, "/api/integrations/change?name=c")
	if !dto.Kickoff.Sealed || dto.Kickoff.SealedBy != "" || len(dto.Errors) != 0 {
		t.Errorf("kickoff sin sealed_by: %+v", dto)
	}
}

func TestChangeIntegrationsGateActors(t *testing.T) {
	h, root, _ := changeTabEnv(t, nil)
	for _, rec := range []kickoff.GateRecord{
		{Gate: kickoff.GateKey("spec"), Decision: kickoff.GateDecision("approved"), Actor: "Bea <bea@example.test>", RecordedAt: time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)},
		{Gate: kickoff.GateKey("design"), Decision: kickoff.GateDecision("approved"), RecordedAt: time.Date(2026, 11, 18, 10, 0, 0, 0, time.UTC)},
	} {
		if err := kickoff.AppendGate(root, rec); err != nil {
			t.Fatal(err)
		}
	}
	_, dto, _ := getChangeTab(t, h, "/api/integrations/change?name=c")
	if len(dto.Gates) != 2 || dto.Gates[0].Gate != "spec" || dto.Gates[0].Actor != "Bea <bea@example.test>" || dto.Gates[1].Actor != "" {
		t.Errorf("puertas: %+v", dto.Gates)
	}
}

func TestChangeIntegrationsOutboxIsRedactedAndReadOnly(t *testing.T) {
	h, _, _ := changeTabEnv(t, map[string]string{integrationsOutboxFileForTab: changeOutboxYAML})
	code, dto, body := getChangeTab(t, h, "/api/integrations/change?name=c")
	if code != http.StatusOK || len(dto.Outbox) != 1 || dto.Empty {
		t.Fatalf("outbox: %d %+v", code, dto)
	}
	if op := dto.Outbox[0]; op.Plugin != "fake" || op.Event != "gate.spec.approved" || op.Gate != "spec" || op.State != "pending" || op.Attempts != 2 {
		t.Errorf("operación: %+v", op)
	}
	if strings.Contains(body, changeSentinel) {
		t.Errorf("la respuesta filtra el centinela de la outbox: %s", body)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(method, "/api/integrations/change?name=c", nil)
		request.Host = "127.0.0.1"
		h.ServeHTTP(recorder, request)
		if recorder.Code == http.StatusOK {
			t.Errorf("%s no debe estar permitido", method)
		}
	}
}

func TestChangeIntegrationsCapsTheOutboxAtOneHundred(t *testing.T) {
	var b strings.Builder
	b.WriteString("schema: axiom.integrations-outbox/v1\nchange: c\noperations:\n")
	for i := 0; i < 130; i++ {
		b.WriteString("  - key: k" + itoa(i) + "\n    plugin: fake\n    state: pending\n    attempts: 0\n")
		b.WriteString("    enqueued_at: 2026-11-18T22:30:00Z\n    event:\n      type: gate.spec.approved\n      change: c\n      gate: spec\n      actor: cli\n      occurred_at: 2026-11-18T22:30:00Z\n      idempotency_key: k" + itoa(i) + "\n")
	}
	h, _, _ := changeTabEnv(t, map[string]string{integrationsOutboxFileForTab: b.String()})
	_, dto, _ := getChangeTab(t, h, "/api/integrations/change?name=c")
	if len(dto.Outbox) != 100 || !dto.OutboxTruncated || dto.OutboxTotal != 130 {
		t.Errorf("tope: %d truncated=%v total=%d", len(dto.Outbox), dto.OutboxTruncated, dto.OutboxTotal)
	}
}

func TestChangeIntegrationsPartialErrorsDoNotBreakTheTab(t *testing.T) {
	h, root, _ := changeTabEnv(t, map[string]string{
		"integrations.yaml":          "schema: otra/v9\n",
		integrationsOutboxFileForTab: changeOutboxYAML,
	})
	workspacetest.WriteFile(t, filepath.Join(root, "kickoff.yaml"), "esto: no es un kickoff\n")
	code, dto, body := getChangeTab(t, h, "/api/integrations/change?name=c")
	if code != http.StatusOK {
		t.Fatalf("estado = %d", code)
	}
	if len(dto.Errors) < 2 || len(dto.Outbox) != 1 {
		t.Errorf("errores parciales: %+v", dto)
	}
	if strings.Contains(body, root) {
		t.Errorf("los errores no deben llevar rutas locales: %s", body)
	}
}

func TestChangeIntegrationsCorruptOutboxLeavesTheRestIntact(t *testing.T) {
	h, _, _ := changeTabEnv(t, map[string]string{
		"integrations.yaml":          changeLinksYAML,
		integrationsOutboxFileForTab: "schema: roto\n",
	})
	_, dto, _ := getChangeTab(t, h, "/api/integrations/change?name=c")
	if len(dto.Errors) != 1 || len(dto.Outbox) != 0 || len(dto.WorkItems) != 3 {
		t.Errorf("outbox corrupta: %+v", dto)
	}
}

func TestChangeIntegrationsChangeNotFoundAndTraversal(t *testing.T) {
	h, _, _ := changeTabEnv(t, nil)
	cases := map[string]int{
		"/api/integrations/change?name=inexistente": http.StatusNotFound,
		"/api/integrations/change?name=":            http.StatusBadRequest,
		"/api/integrations/change":                  http.StatusBadRequest,
		"/api/integrations/change?name=..":          http.StatusBadRequest,
		"/api/integrations/change?name=..%2Fx":      http.StatusBadRequest,
		"/api/integrations/change?name=a%5Cb":       http.StatusBadRequest,
		"/api/integrations/change?name=a:b":         http.StatusBadRequest,
		"/api/integrations/change?name=%2Fetc":      http.StatusBadRequest,
	}
	for target, want := range cases {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, target, nil)
		request.Host = "127.0.0.1"
		h.ServeHTTP(recorder, request)
		if recorder.Code != want {
			t.Errorf("GET %s = %d, quiero %d", target, recorder.Code, want)
		}
		if want == http.StatusNotFound && !strings.Contains(recorder.Body.String(), "change_not_found") {
			t.Errorf("GET %s no responde change_not_found: %s", target, recorder.Body.String())
		}
	}
}

func TestChangeIntegrationsEmptyChangeAndForeignHost(t *testing.T) {
	h, _, _ := changeTabEnv(t, nil)
	code, dto, _ := getChangeTab(t, h, "/api/integrations/change?name=c")
	if code != http.StatusOK || !dto.Empty || len(dto.Errors) != 0 || dto.WorkItems == nil || dto.Roles == nil || dto.Outbox == nil || dto.Gates == nil {
		t.Errorf("cambio sin integración: %d %+v", code, dto)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/integrations/change?name=c", nil)
	request.Host = "evil.example.test"
	h.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Errorf("Host ajeno = %d, quiero 403", recorder.Code)
	}
}

func TestChangeIntegrationsReadsAnArchivedChange(t *testing.T) {
	integrationstest.Install(t, integrationstest.NewFake())
	ws := embeddedProject(t, hub.WorktreeModeInPlace)
	archived := filepath.Join(ws.Home, "changes", "archive", "2026-11-01-viejo")
	workspacetest.WriteFile(t, filepath.Join(archived, "proposal.md"), "# viejo\n")
	workspacetest.WriteFile(t, filepath.Join(archived, "integrations.yaml"), strings.ReplaceAll(changeLinksYAML, "change: c", "change: viejo"))
	h := NewServer(NewService(ws.Master)).handler()
	code, dto, _ := getChangeTab(t, h, "/api/integrations/change?name=viejo")
	if code != http.StatusOK || !dto.Archived || len(dto.WorkItems) != 3 || len(dto.Errors) != 0 {
		t.Errorf("archivado: %d %+v", code, dto)
	}
}

const (
	integrationsPendingFileForTab = ".time-pending.yaml"
	integrationsOutboxFileForTab  = ".integrations-outbox.yaml"
)

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}

// sealChange sella un kickoff con los roles dados y el actor indicado.
func sealChange(t *testing.T, root, sealedBy string, roles ...string) {
	t.Helper()
	assigned := make([]kickoff.KickoffRole, 0, len(roles))
	for _, r := range roles {
		assigned = append(assigned, kickoff.KickoffRole{Role: r, GatePolicy: multirole.PolicyBlocking, TasksFile: "tasks." + r + ".md", VerifyFile: "verify-report." + r + ".md"})
	}
	_, ok, err := kickoff.Seal(root, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: filepath.Base(root), SealedBy: sealedBy,
		Config: kickoff.FlowConfig{
			FlowMode: kickoff.FlowSDD, ExecutionStyle: kickoff.ExecutionCheckpointed, ExecutionStyleSource: "explicit",
			Roles: assigned,
		},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	})
	if err != nil || !ok {
		t.Fatalf("kickoff.Seal() = (ok=%v, err=%v)", ok, err)
	}
}

// La pestaña es de solo lectura y todo texto externo entra como texto: se fijan los invariantes del
// módulo por texto (no hay ejecutor de JS en las pruebas).
func TestChangeIntegrationsTabAssets(t *testing.T) {
	html := readAsset(t, "index.html")
	if !strings.Contains(html, `data-tab="integraciones"`) || !strings.Contains(html, `data-panel="integraciones"`) {
		t.Error("index.html no declara la pestaña ni el panel de integraciones del cambio")
	}
	if !strings.Contains(readAsset(t, "js/core/router.js"), "'integraciones'") {
		t.Error("router.js no incluye la pestaña integraciones en INCREMENT_TABS")
	}
	if !strings.Contains(readAsset(t, "js/screens/detalle.js"), "renderChangeIntegrations") {
		t.Error("detalle.js no pinta la pestaña integraciones")
	}
	module := readAsset(t, "js/screens/detalle-integraciones.js")
	for _, marker := range []string{"textContent", "externalLink", "/api/integrations/change?name=", "sin Task", "sin registrar", "sin sellar"} {
		if !strings.Contains(module, marker) {
			t.Errorf("detalle-integraciones.js no contiene %q", marker)
		}
	}
	for _, banned := range []string{"localStorage", "outerHTML", "insertAdjacentHTML", "method: 'POST'", "method: \"POST\"", "/retry", "/start"} {
		if strings.Contains(module, banned) {
			t.Errorf("detalle-integraciones.js no debe contener %q: la pestaña es de solo lectura", banned)
		}
	}
	// innerHTML solo con las plantillas fijas de estado (cargando, vacío, error), nunca con datos.
	for _, line := range strings.Split(module, "\n") {
		if strings.Contains(line, "innerHTML") && !strings.Contains(line, "stateLoading(") && !strings.Contains(line, "stateEmpty(") && !strings.Contains(line, "stateError(") {
			t.Errorf("innerHTML con contenido no fijo: %s", strings.TrimSpace(line))
		}
	}
}
