package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/handoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
)

// keysOf devuelve las claves de un objeto JSON decodificado como mapa.
func keysOf(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("no es un objeto JSON: %v (%s)", err, raw)
	}
	return m
}

func requireKeys(t *testing.T, where string, m map[string]json.RawMessage, want ...string) {
	t.Helper()
	for _, k := range want {
		if _, ok := m[k]; !ok {
			t.Errorf("%s: falta la clave %q (claves: %v)", where, k, m)
		}
	}
	if len(m) != len(want) {
		t.Errorf("%s: %d claves, se esperaban %d (%v)", where, len(m), len(want), want)
	}
}

func TestBarrierDTOContract(t *testing.T) {
	report := &multirole.BarrierReport{
		Satisfied: false,
		Change:    "inc-x",
		Blockers:  []string{"rol backend sin verificar"},
		Warnings:  []string{"rol e2e diferido"},
		Roles: []multirole.RoleExecutionStatus{
			{
				Assignment: multirole.RoleAssignment{
					Role:         "backend",
					Name:         "Backend",
					GatePolicy:   multirole.PolicyBlocking,
					Repositories: []string{"api", "core"},
					Deliverables: []string{"endpoint"},
				},
				TasksFound: true,
				Tasks:      multirole.RoleTaskProgress{Total: 4, Completed: 3, Pending: 1, Percent: 75},
				ApplyDone:  true,
				VerifyDone: false,
				Verdict:    "missing",
				Compliant:  false,
			},
			{Assignment: multirole.RoleAssignment{Role: "e2e", GatePolicy: multirole.PolicyDeferred}},
		},
		DeferredTasks: []multirole.DeferredTask{
			{Change: "inc-x", Role: "e2e", TaskText: "- [ ] prueba", SpecRef: "REQ-1", ChangeDirRef: "openspec/changes/inc-x"},
		},
	}

	raw, err := json.Marshal(NewBarrierDTO(report))
	if err != nil {
		t.Fatal(err)
	}
	top := keysOf(t, raw)
	requireKeys(t, "barrera", top, "satisfied", "change", "blockers", "warnings", "roles", "deferred_tasks")

	var roles []json.RawMessage
	if err := json.Unmarshal(top["roles"], &roles); err != nil {
		t.Fatalf("roles debe ser un array JSON: %v (%s)", err, top["roles"])
	}
	if len(roles) != 2 {
		t.Fatalf("roles = %d entradas, se esperaban 2", len(roles))
	}
	role := keysOf(t, roles[0])
	requireKeys(t, "rol", role, "role", "name", "gate_policy", "repositories", "deliverables",
		"tasks_found", "tasks", "apply_done", "verify_done", "verdict", "compliant")
	requireKeys(t, "rol.tasks", keysOf(t, role["tasks"]), "total", "completed", "pending", "percent")

	var got struct {
		GatePolicy string `json:"gate_policy"`
		Tasks      struct {
			Percent float64 `json:"percent"`
		} `json:"tasks"`
		Repositories []string `json:"repositories"`
	}
	if err := json.Unmarshal(roles[0], &got); err != nil {
		t.Fatal(err)
	}
	if got.GatePolicy != "blocking" || got.Tasks.Percent != 75 || len(got.Repositories) != 2 {
		t.Errorf("valores inesperados: %+v", got)
	}

	var deferred []json.RawMessage
	if err := json.Unmarshal(top["deferred_tasks"], &deferred); err != nil || len(deferred) != 1 {
		t.Fatalf("deferred_tasks debe ser un array de 1 elemento: %v (%s)", err, top["deferred_tasks"])
	}
	requireKeys(t, "deferred_task", keysOf(t, deferred[0]), "change", "role", "task_text", "spec_ref", "change_dir_ref")
}

func TestBarrierDTOEmptySlicesAreArrays(t *testing.T) {
	raw, err := json.Marshal(NewBarrierDTO(&multirole.BarrierReport{
		Change: "vacio",
		Roles:  []multirole.RoleExecutionStatus{{Assignment: multirole.RoleAssignment{Role: "fullstack"}}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	top := keysOf(t, raw)
	for _, k := range []string{"blockers", "warnings", "deferred_tasks"} {
		if string(top[k]) != "[]" {
			t.Errorf("%s = %s, se esperaba []", k, top[k])
		}
	}
	var roles []json.RawMessage
	if err := json.Unmarshal(top["roles"], &roles); err != nil || len(roles) != 1 {
		t.Fatalf("roles debe ser un array de 1 elemento: %v (%s)", err, top["roles"])
	}
	role := keysOf(t, roles[0])
	for _, k := range []string{"repositories", "deliverables"} {
		if string(role[k]) != "[]" {
			t.Errorf("rol.%s = %s, se esperaba []", k, role[k])
		}
	}

	// Sin roles ni nada: roles también es [].
	raw, _ = json.Marshal(NewBarrierDTO(&multirole.BarrierReport{}))
	if got := string(keysOf(t, raw)["roles"]); got != "[]" {
		t.Errorf("roles vacío = %s, se esperaba []", got)
	}

	if NewBarrierDTO(nil) != nil {
		t.Error("NewBarrierDTO(nil) debe devolver nil")
	}
}

func TestHandoffDTOContract(t *testing.T) {
	ts := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	ho := &handoff.Handoff{
		Metadata: handoff.Metadata{
			Change: "inc-x", FromPhase: handoff.PhaseDesign, ToPhase: handoff.PhaseTasks,
			FromRole: "core", ToRole: "backend", Timestamp: ts, Status: handoff.StatusReady,
		},
		Sections: handoff.Sections{
			ExecutiveSummary: "resumen", Artifacts: "arts", Decisions: "decs",
			RisksAndBlockers: "riesgos", DirectInstructions: "instrucciones",
		},
	}
	raw, err := json.Marshal(NewHandoffDTO(ho))
	if err != nil {
		t.Fatal(err)
	}
	top := keysOf(t, raw)
	requireKeys(t, "handoff", top, "metadata", "sections")
	meta := keysOf(t, top["metadata"])
	requireKeys(t, "metadata", meta, "change", "from_phase", "to_phase", "from_role", "to_role", "timestamp", "status")
	requireKeys(t, "sections", keysOf(t, top["sections"]),
		"executive_summary", "artifacts", "decisions", "risks_and_blockers", "direct_instructions")

	var m struct {
		FromPhase string `json:"from_phase"`
		Timestamp string `json:"timestamp"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(top["metadata"], &m); err != nil {
		t.Fatal(err)
	}
	if m.FromPhase != "design" || m.Status != "ready" || m.Timestamp != "2026-09-14T12:00:00Z" {
		t.Errorf("metadata inesperada: %+v", m)
	}

	// Un handoff sin timestamp no debe emitir el año 1.
	raw, _ = json.Marshal(NewHandoffDTO(&handoff.Handoff{}))
	var zero struct {
		Metadata struct {
			Timestamp string `json:"timestamp"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &zero); err != nil || zero.Metadata.Timestamp != "" {
		t.Errorf("timestamp cero = %q (%v), se esperaba vacío", zero.Metadata.Timestamp, err)
	}
	if NewHandoffDTO(nil) != nil {
		t.Error("NewHandoffDTO(nil) debe devolver nil")
	}
}

// TestRolesAndHandoffEndpointsUseSnakeCase comprueba el contrato de extremo a extremo por HTTP.
func TestRolesAndHandoffEndpointsUseSnakeCase(t *testing.T) {
	// /api/roles sobre el incremento real inc-03 de este repositorio.
	srv := NewServer(NewService("../.."))
	rec := httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/roles?change=inc-03-multi-role-sdd-fan-out", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/roles = %d: %s", rec.Code, rec.Body.String())
	}
	top := keysOf(t, rec.Body.Bytes())
	requireKeys(t, "/api/roles", top, "satisfied", "change", "blockers", "warnings", "roles", "deferred_tasks")
	var roles []json.RawMessage
	if err := json.Unmarshal(top["roles"], &roles); err != nil || len(roles) == 0 {
		t.Fatalf("roles debe ser un array no vacío: %v (%s)", err, top["roles"])
	}

	// /api/handoffs sobre un handoff.md temporal.
	tmpDir := t.TempDir()
	changeDir := filepath.Join(tmpDir, "openspec", "changes", "test-ho")
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nchange: test-ho\nfrom_phase: design\nto_phase: tasks\nfrom_role: core\nto_role: core\n" +
		"timestamp: \"2026-09-14T12:00:00Z\"\nstatus: ready\n---\n\n" +
		"## 1. Resumen Ejecutivo\nResumen prueba.\n\n## 2. Artefactos Modificados y Creados\n- arch1\n\n" +
		"## 3. Decisiones Técnicas y Acuerdos\n- dec1\n\n## 4. Riesgos, Bloqueos y Preguntas Abiertas\n- ninguna\n\n" +
		"## 5. Instrucciones Directas para el Siguiente Rol\nProceder con tasks.\n"
	if err := os.WriteFile(filepath.Join(changeDir, "handoff.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	srv = NewServer(NewService(tmpDir))
	rec = httptest.NewRecorder()
	srv.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/handoffs?change=test-ho", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/handoffs = %d: %s", rec.Code, rec.Body.String())
	}
	top = keysOf(t, rec.Body.Bytes())
	requireKeys(t, "/api/handoffs", top, "metadata", "sections")
	var sec struct {
		ExecutiveSummary   string `json:"executive_summary"`
		DirectInstructions string `json:"direct_instructions"`
	}
	if err := json.Unmarshal(top["sections"], &sec); err != nil {
		t.Fatal(err)
	}
	if sec.ExecutiveSummary != "Resumen prueba." || sec.DirectInstructions != "Proceder con tasks." {
		t.Errorf("secciones inesperadas: %+v", sec)
	}
}
