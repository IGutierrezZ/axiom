package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/gitidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

const (
	timeTaskID  = 205
	timeActor   = "Ana <ana@x.test>"
	closeToDone = "    task_state_close: Done\n"
)

// timeZoneTest es la zona local fija de las pruebas (UTC+1): el inicio es el lunes 09:00 y la consulta
// el martes 12:00, así que la propuesta es de 11 h.
var timeZoneTest = time.FixedZone("UTC+1", 3600)

// timeEnv es el escenario de los verbos `time`: el cambio x con el rol fullstack vinculado a la Task
// 205 (rev 4, original 8, restante 5, imputado 3) e iniciado el lunes a las 09:00 hora local.
type timeEnv struct{ *tasksEnv }

func newTimeEnv(t *testing.T, style kickoff.ExecutionStyle, extraBlock string) *timeEnv {
	t.Helper()
	block := adoBlock + extraBlock
	if style == kickoff.ExecutionCheckpointed {
		// La aprobación de role-apply entrega role.apply.approved: que no mueva la Task de To Do.
		block += "    task_state_apply_approved: To Do\n"
	}
	e := &timeEnv{newTasksEnvWith(t, block, style, tasksDesign)}
	e.seedTimeTask(timeTaskID, 4, 8, 5, 3)
	started := time.Date(2026, 11, 23, 8, 0, 0, 0, time.UTC)
	e.bindLink(t, "fullstack", "205", hoursPtr(8), &started)

	previousNow, previousLocation, previousRecord := trackerNow, trackerLocation, trackerRecordTimeEntry
	trackerNow = func() time.Time { return time.Date(2026, 11, 24, 11, 0, 0, 0, time.UTC) }
	trackerLocation = func() *time.Location { return timeZoneTest }
	t.Cleanup(func() {
		trackerNow, trackerLocation, trackerRecordTimeEntry = previousNow, previousLocation, previousRecord
	})
	stubActorIdentity(t, gitidentity.Identity{Name: "Ana", Email: "ana@x.test"})

	if style == kickoff.ExecutionCheckpointed {
		writeCLIFixture(t, filepath.Join(e.changeRoot, "tasks.md"), "# Tareas\n\n- [x] 1.1 Hecho\n")
		e.approveRoleApply(t)
	}
	return e
}

func (e *timeEnv) approveRoleApply(t *testing.T) {
	t.Helper()
	var out bytes.Buffer
	if err := RunSDDGate([]string{"record", "--cwd", e.root, "--change", "x", "--gate", "role-apply:fullstack", "--decision", "approved", "--reason", "ok", "--actor", "Ana"}, &out); err != nil {
		t.Fatalf("aprobar role-apply:fullstack: %v", err)
	}
}

// seedTimeTask siembra una Task hija del work item 100 con las horas dadas.
func (e *timeEnv) seedTimeTask(id, rev int, original, remaining, completed float64) {
	e.srv.SeedWorkItem(id, rev, map[string]any{
		"System.WorkItemType": "Task", "System.State": "To Do", "System.Title": "x / fullstack",
		hoursOriginalField: original, hoursRemainingField: remaining, hoursCompletedField: completed,
	}, map[string]any{"rel": "System.LinkTypes.Hierarchy-Reverse", "url": e.workItemAPI(tasksParentID)})
}

func (e *timeEnv) timeCLI(t *testing.T, verb string, args ...string) (int, timeJSON, string, string) {
	t.Helper()
	full := append([]string{"time", verb}, args...)
	full = append(full, "--cwd", e.root, "--json")
	code, out, errOut := runIntegrationsCLI(t, "", full...)
	var got timeJSON
	if strings.TrimSpace(out) != "" {
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("la salida --json no es JSON válido: %v\n%s", err, out)
		}
	}
	return code, got, out, errOut
}

func (e *timeEnv) record(t *testing.T, args ...string) (int, timeJSON, string, string) {
	t.Helper()
	return e.timeCLI(t, "record", append([]string{"--change", "x", "--role", "fullstack"}, args...)...)
}

func (e *timeEnv) propose(t *testing.T, args ...string) (int, timeJSON, string, string) {
	t.Helper()
	return e.timeCLI(t, "propose", append([]string{"--change", "x", "--role", "fullstack"}, args...)...)
}

func (e *timeEnv) entries(t *testing.T) []integrations.TimeEntry {
	t.Helper()
	entries, err := integrations.TimeEntries(e.folder)
	if err != nil {
		t.Fatalf("TimeEntries: %v", err)
	}
	return entries
}

func (e *timeEnv) pendingFile() string {
	return filepath.Join(e.changeRoot, integrations.PendingTimeFile)
}

func (e *timeEnv) pending(t *testing.T) (integrations.PendingTime, bool) {
	t.Helper()
	p, ok, err := integrations.PendingTimeFor(e.folder, "fullstack")
	if err != nil {
		t.Fatalf("PendingTimeFor: %v", err)
	}
	return p, ok
}

func (e *timeEnv) putPending(t *testing.T, p integrations.PendingTime) {
	t.Helper()
	if err := integrations.PutPendingTime(e.folder, p); err != nil {
		t.Fatalf("PutPendingTime: %v", err)
	}
}

// intent es la intención de imputar 2 h a la Task 205 leída en la revisión 4 (3 y 5 → 5 y 3).
func intent() integrations.PendingTime {
	return integrations.PendingTime{Role: "fullstack", ItemID: "205", Hours: 2, Minutes: 120, Actor: timeActor, Date: "2026-11-24",
		Source: "role.close/fullstack/1", BaseRev: 4, ExpectedCompleted: 5, ExpectedRemaining: 3}
}

func (e *timeEnv) task(t *testing.T, id int) adotest.WorkItem {
	t.Helper()
	wi, ok := e.srv.WorkItem(id)
	if !ok {
		t.Fatalf("la Task %d no existe", id)
	}
	return wi
}

func (e *timeEnv) state(t *testing.T, id int) string {
	t.Helper()
	s, _ := e.task(t, id).Fields["System.State"].(string)
	return s
}

// workitemPath devuelve la ruta decodificada del work item 205 tal como la pide el cliente.
func (e *timeEnv) workitemPath(t *testing.T) string {
	t.Helper()
	e.propose(t)
	for _, r := range e.srv.Requests() {
		if r.Method == http.MethodGet && strings.HasSuffix(r.Path, "/workitems/205") {
			return r.Path
		}
	}
	t.Fatalf("no consta ninguna lectura del work item 205: %v", e.srv.Requests())
	return ""
}

func warningCodes(w []trackerWarning) string {
	codes := make([]string, 0, len(w))
	for _, x := range w {
		codes = append(codes, x.Code)
	}
	return strings.Join(codes, ",")
}

func (e *timeEnv) noRequests(t *testing.T, before int) {
	t.Helper()
	if got := len(e.srv.Requests()); got != before {
		t.Errorf("no debía haber ninguna petición al doble: %d antes, %d después", before, got)
	}
}

// ---------------------------------------------------------------- propose

// prov:INT:7: la propuesta se recorta a la jornada, muestra los valores del tracker y no escribe nada.
func TestIntegrationsTimeProposeClipsToTheWorkdayAndShowsTrackerValues(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, "")
	linksBefore := e.linksFile(t)

	code, got, out, errOut := e.propose(t)

	if code != 0 || got.Status != trackerStatusDone || got.Schema != integrationsTimeSchema || got.Verb != "propose" {
		t.Fatalf("código %d, salida %s, stderr %q", code, out, errOut)
	}
	if got.Proposal == nil || got.Proposal.Hours != 11 || got.Proposal.Minutes != 660 {
		t.Errorf("propuesta de 11 h (8 del lunes y 3 del martes): %+v", got.Proposal)
	}
	if got.Timezone == nil || got.Timezone.Offset != "+01:00" || got.Timezone.Name == "" {
		t.Errorf("la zona horaria se nombra con su desfase: %+v", got.Timezone)
	}
	if got.Clock == nil || got.Clock.WorkdayHours != 8 || strings.Join(got.Clock.Workdays, ",") != "mon,tue,wed,thu,fri" || got.Clock.Seconds != 11*3600 {
		t.Errorf("reloj: %+v", got.Clock)
	}
	if got.EstimateHours == nil || *got.EstimateHours != 8 {
		t.Errorf("la estimación del vínculo es 8 h: %v", got.EstimateHours)
	}
	if got.Tracker == nil || got.Tracker.OriginalEstimate == nil || *got.Tracker.OriginalEstimate != 8 || got.Tracker.RemainingWork != 5 || got.Tracker.CompletedWork != 3 || got.Tracker.Rev != 4 {
		t.Errorf("valores leídos del tracker: %+v", got.Tracker)
	}
	if e.requests("PATCH") != 0 || !bytes.Equal(e.linksFile(t), linksBefore) {
		t.Errorf("propose no escribe en el tracker ni en integrations.yaml")
	}
	if _, err := os.Stat(e.pendingFile()); !os.IsNotExist(err) {
		t.Errorf("propose no crea el diario pendiente: %v", err)
	}
}

// El segundo cierre cuenta desde la última imputación role.close del rol.
func TestIntegrationsTimeProposeCountsFromTheLastClose(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, "")
	last := time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC) // martes 09:00 local
	if _, err := integrations.RecordTimeEntry(e.folder, "azure-devops", integrations.TimeEntry{
		Plugin: "azure-devops", Role: "fullstack", ItemID: "205", Minutes: 480, Date: "2026-11-24", Source: "role.close/fullstack/1", Actor: timeActor,
	}, last); err != nil {
		t.Fatal(err)
	}

	_, got, out, _ := e.propose(t)

	if got.Proposal == nil || got.Proposal.Hours != 3 {
		t.Fatalf("solo cuenta el martes de 09:00 a 12:00: %s", out)
	}
}

func TestIntegrationsTimeRefusalsNameTheirResolution(t *testing.T) {
	// Sin vínculo de Task.
	noLink := newTasksEnvWith(t, adoBlock, kickoff.ExecutionContinuous, tasksDesign)
	noLinkEnv := &timeEnv{noLink}
	for _, verb := range []string{"propose", "record"} {
		args := []string{"--change", "x", "--role", "fullstack"}
		if verb == "record" {
			args = append(args, "--hours", "2")
		}
		before := len(noLink.srv.Requests())
		code, got, _, _ := noLinkEnv.timeCLI(t, verb, args...)
		if code != 1 || got.Refusal == nil || got.Refusal.Code != "task_not_bound" || !strings.Contains(got.Refusal.Next, "axiom integrations tasks create --change x") {
			t.Errorf("%s sin vínculo: código %d, %+v", verb, code, got.Refusal)
		}
		noLinkEnv.noRequests(t, before)
	}

	// Con vínculo pero sin started_at.
	noStart := newTasksEnvWith(t, adoBlock, kickoff.ExecutionContinuous, tasksDesign)
	noStart.bindLink(t, "fullstack", "205", hoursPtr(8), nil)
	noStartEnv := &timeEnv{noStart}
	for _, verb := range []string{"propose", "record"} {
		args := []string{"--change", "x", "--role", "fullstack"}
		if verb == "record" {
			args = append(args, "--hours", "2")
		}
		code, got, _, _ := noStartEnv.timeCLI(t, verb, args...)
		if code != 1 || got.Refusal == nil || got.Refusal.Code != "role_not_started" || !strings.Contains(got.Refusal.Next, "axiom branch ensure --change x --role fullstack") {
			t.Errorf("%s sin started_at: código %d, %+v", verb, code, got.Refusal)
		}
	}
}

// time propose con una intención pendiente solo avisa: ni la reconcilia ni la toca.
func TestIntegrationsTimeProposeWithPendingOnlyWarns(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, "")
	e.putPending(t, intent())
	pendingBefore, linksBefore := mustRead(t, e.pendingFile()), e.linksFile(t)

	code, got, out, _ := e.propose(t)

	if code != 0 || got.Status != trackerStatusDone || warningCodes(got.Warnings) != "time_pending" {
		t.Fatalf("código %d, salida %s", code, out)
	}
	if next := got.Warnings[0].Next; next != "axiom integrations time record --change x --role fullstack" {
		t.Errorf("la resolución del aviso: %q", next)
	}
	if !bytes.Equal(mustRead(t, e.pendingFile()), pendingBefore) || !bytes.Equal(e.linksFile(t), linksBefore) || e.requests("PATCH") != 0 {
		t.Errorf("propose no cambia .time-pending.yaml, integrations.yaml ni el tracker")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// ---------------------------------------------------------------- record

// prov:INT:8: imputación correcta con test /rev, registro con actor, minutos y fecha solo tras el éxito.
func TestIntegrationsTimeRecordWritesHoursAndRegistersTheEntry(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionCheckpointed, "")

	code, got, out, errOut := e.record(t, "--hours", "2")

	if code != 0 || got.Status != trackerStatusDone || got.Recorded == nil {
		t.Fatalf("código %d, salida %s, stderr %q", code, out, errOut)
	}
	task := e.task(t, timeTaskID)
	if task.Rev != 5 || fieldNumber(t, task, hoursCompletedField) != 5 || fieldNumber(t, task, hoursRemainingField) != 3 || e.requests("PATCH") != 1 {
		t.Errorf("PATCH con test /rev 4: rev %d, %+v, %d PATCH", task.Rev, task.Fields, e.requests("PATCH"))
	}
	entries := e.entries(t)
	if len(entries) != 1 || entries[0].Actor != timeActor || entries[0].Minutes != 120 || entries[0].Date != "2026-11-24" ||
		entries[0].Role != "fullstack" || entries[0].ItemID != "205" || entries[0].Source != "role.close/fullstack/1" {
		t.Errorf("imputación registrada: %+v", entries)
	}
	if _, ok := e.pending(t); ok {
		t.Errorf("la intención se borra tras el registro")
	}
	if got.Recorded.Source != "role.close/fullstack/1" || got.Recorded.CompletedWork != 5 || got.Recorded.RemainingWork != 3 {
		t.Errorf("salida de la imputación: %+v", got.Recorded)
	}
	if e.state(t, timeTaskID) != "To Do" || got.Transition != nil {
		t.Errorf("en checkpointed no se toca el estado de la Task: %q", e.state(t, timeTaskID))
	}
}

func TestIntegrationsTimeRecordRemainingNeverGoesBelowZero(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
	e.seedTimeTask(timeTaskID, 4, 8, 1, 3)

	code, _, out, _ := e.record(t, "--hours", "2.5")

	task := e.task(t, timeTaskID)
	if code != 0 || fieldNumber(t, task, hoursRemainingField) != 0 || fieldNumber(t, task, hoursCompletedField) != 5.5 {
		t.Fatalf("código %d, salida %s, tarea %+v", code, out, task.Fields)
	}
	if entries := e.entries(t); len(entries) != 1 || entries[0].Minutes != 150 {
		t.Errorf("imputación: %+v", entries)
	}
}

// Un conflicto de rev no reaplica, no registra, muestra los valores releídos y nombra time propose.
func TestIntegrationsTimeRecordRevConflictNeverReappliesOrRegisters(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
	e.srv.EditAfterNextRead(timeTaskID, map[string]any{hoursCompletedField: 4.0})

	code, got, out, _ := e.record(t, "--hours", "2")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "rev_conflict" || got.Refusal.Next != "axiom integrations time propose --change x --role fullstack" {
		t.Fatalf("código %d, salida %s", code, out)
	}
	if got.Tracker == nil || got.Tracker.Rev != 5 || got.Tracker.CompletedWork != 4 || !strings.Contains(got.Refusal.Text, "CompletedWork 4") {
		t.Errorf("valores releídos: %+v / %s", got.Tracker, got.Refusal.Text)
	}
	if len(e.entries(t)) != 0 || e.srv.AppliedPatches(timeTaskID) != 0 || e.state(t, timeTaskID) != "To Do" {
		t.Errorf("sin imputación ni reaplicación ni transición: %d parches, estado %q", e.srv.AppliedPatches(timeTaskID), e.state(t, timeTaskID))
	}
	if _, ok := e.pending(t); ok {
		t.Errorf("un conflicto en el primer intento borra la intención: consta que no se aplicó")
	}
}

func TestIntegrationsTimeRecordSecondCloseUsesTheNextSource(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
	if _, err := integrations.RecordTimeEntry(e.folder, "azure-devops", integrations.TimeEntry{
		Plugin: "azure-devops", Role: "fullstack", ItemID: "205", Minutes: 60, Date: "2026-11-23", Source: "role.close/fullstack/1", Actor: timeActor,
	}, time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	code, got, out, _ := e.record(t, "--hours", "1")

	if code != 0 || got.Recorded == nil || got.Recorded.Source != "role.close/fullstack/2" {
		t.Fatalf("código %d, salida %s", code, out)
	}
	if entries := e.entries(t); len(entries) != 2 || entries[0].Source != "role.close/fullstack/1" {
		t.Errorf("no choca con la anterior: %+v", entries)
	}
}

func TestIntegrationsTimeRecordUsageErrorsMakeNoRequestsAndKeepFiles(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, "")
	linksBefore := e.linksFile(t)
	before := len(e.srv.Requests())

	cases := map[string][]string{
		"horas cero":                    {"--hours", "0"},
		"horas negativas":               {"--hours", "-1"},
		"horas no numéricas":            {"--hours", "mucho"},
		"más de dos decimales":          {"--hours", "1.234"},
		"por encima del tope":           {"--hours", "1001"},
		"sin modo":                      {},
		"hours con resolve-pending":     {"--hours", "2", "--resolve-pending", "applied"},
		"close-only con hours":          {"--close-only", "--hours", "2"},
		"close-only con resolve":        {"--close-only", "--resolve-pending", "discarded"},
		"resolve-pending desconocido":   {"--resolve-pending", "quizas"},
		"resolve-pending sin intención": {"--resolve-pending", "applied"},
		"rol fuera del roster":          {"--role", "otro", "--hours", "2"},
	}
	for name, args := range cases {
		full := append([]string{"record", "--change", "x", "--role", "fullstack"}, args...)
		full = append(full, "--cwd", e.root, "--json")
		if code, _, errOut := runIntegrationsCLI(t, "", full...); code != 2 || !strings.Contains(errOut, "Error:") {
			t.Errorf("%s: código %d, stderr %q", name, code, errOut)
		}
	}
	e.noRequests(t, before)
	if !bytes.Equal(e.linksFile(t), linksBefore) {
		t.Errorf("integrations.yaml no cambia")
	}
	if code, _, _ := runIntegrationsCLI(t, "", "time", "record", "--change", "x", "--hours", "2", "--cwd", e.root); code != 2 {
		t.Errorf("--role es obligatorio: %d", code)
	}
}

func TestIntegrationsTimeCheckpointedNeedsTheRoleApplyGate(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionCheckpointed, "")
	// Una edición posterior a la aprobación deja la compuerta en pending.
	writeCLIFixture(t, filepath.Join(e.changeRoot, "tasks.md"), "# Tareas\n\n- [x] 1.1 Hecho\n- [x] 1.2 Más\n")
	before := len(e.srv.Requests())

	for _, verb := range []string{"propose", "record"} {
		args := []string{"--change", "x", "--role", "fullstack"}
		if verb == "record" {
			args = append(args, "--hours", "2")
		}
		code, got, _, _ := e.timeCLI(t, verb, args...)
		if code != 1 || got.Refusal == nil || got.Refusal.Code != "gate_not_approved" || !strings.Contains(got.Refusal.Next, "axiom sdd gate show --change x") {
			t.Errorf("%s: código %d, %+v", verb, code, got.Refusal)
		}
	}
	e.noRequests(t, before)
}

func TestIntegrationsTimeNotApplicableWithoutAnActivePlugin(t *testing.T) {
	root, _, _ := integrationsWorkspace(t, "")
	_ = root
	code, out, errOut := runIntegrationsCLI(t, "", "time", "propose", "--change", "x", "--role", "fullstack", "--cwd", root, "--json")
	var got timeJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("salida: %v\n%s", err, out)
	}
	if code != 0 || got.Status != trackerStatusNotApplicable {
		t.Fatalf("código %d, salida %s, stderr %q", code, out, errOut)
	}
}

func TestIntegrationsTimeODDLaneIsRejectedWithoutFiles(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, "")
	writeCLIFixture(t, filepath.Join(e.root, "odd", "tasks", "f.md"), "# f\n")
	linksBefore := e.linksFile(t)
	before := len(e.srv.Requests())

	for _, verb := range []string{"propose", "record"} {
		args := []string{"--change", "f", "--role", "fullstack"}
		if verb == "record" {
			args = append(args, "--hours", "2")
		}
		code, got, _, _ := e.timeCLI(t, verb, args...)
		if code != 1 || got.Refusal == nil || got.Refusal.Code != "odd_lane_not_supported" {
			t.Errorf("%s: código %d, %+v", verb, code, got.Refusal)
		}
	}
	e.noRequests(t, before)
	if !bytes.Equal(e.linksFile(t), linksBefore) {
		t.Errorf("integrations.yaml no cambia")
	}
}

// ---------------------------------------------------------------- estado de cierre

func TestIntegrationsTimeRecordContinuousMovesTheTaskToTheCloseState(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)

	code, got, out, _ := e.record(t, "--hours", "2")

	if code != 0 || got.Transition == nil || got.Transition.State != "Done" || !got.Transition.Changed {
		t.Fatalf("código %d, salida %s", code, out)
	}
	if e.state(t, timeTaskID) != "Done" || len(e.entries(t)) != 1 {
		t.Errorf("la Task queda en Done y la imputación registrada: %q", e.state(t, timeTaskID))
	}
}

func TestIntegrationsTimeRecordTransitionFailureKeepsTheEntryAsAWarning(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, "    task_state_close: Inexistente\n")

	code, got, out, _ := e.record(t, "--hours", "2")

	if code != 0 || got.Status != trackerStatusDone || warningCodes(got.Warnings) != "state_transition_failed" {
		t.Fatalf("código %d, salida %s", code, out)
	}
	if !strings.Contains(got.Warnings[0].Text, "To Do") || got.Warnings[0].Next != "axiom integrations time record --change x --role fullstack --close-only" {
		t.Errorf("el aviso lista los estados válidos y nombra --close-only: %+v", got.Warnings[0])
	}
	if len(e.entries(t)) != 1 || e.state(t, timeTaskID) != "To Do" {
		t.Errorf("la imputación sigue registrada")
	}
}

// D21: cierre sin horas en continuous, sin imputación ni PATCH de horas.
func TestIntegrationsTimeCloseOnlyContinuousTransitionsWithoutRecording(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
	code, got, out, _ := e.record(t, "--close-only")

	if code != 0 || got.Transition == nil || got.Transition.State != "Done" || got.Recorded != nil {
		t.Fatalf("código %d, salida %s", code, out)
	}
	task := e.task(t, timeTaskID)
	if e.state(t, timeTaskID) != "Done" || fieldNumber(t, task, hoursCompletedField) != 3 || len(e.entries(t)) != 0 {
		t.Errorf("sin imputación: estado %q, %+v", e.state(t, timeTaskID), task.Fields)
	}
}

func TestIntegrationsTimeCloseOnlyDoesNotNeedStartedAtButNeedsTheLink(t *testing.T) {
	e := newTasksEnvWith(t, adoBlock+closeToDone, kickoff.ExecutionContinuous, tasksDesign)
	te := &timeEnv{e}
	te.seedTimeTask(timeTaskID, 4, 8, 5, 3)

	code, got, _, _ := te.timeCLI(t, "record", "--change", "x", "--role", "fullstack", "--close-only")
	if code != 1 || got.Refusal == nil || got.Refusal.Code != "task_not_bound" {
		t.Fatalf("sin vínculo: código %d, %+v", code, got.Refusal)
	}
	e.bindLink(t, "fullstack", "205", hoursPtr(8), nil)
	code, got, out, _ := te.timeCLI(t, "record", "--change", "x", "--role", "fullstack", "--close-only")
	if code != 0 || te.state(t, timeTaskID) != "Done" {
		t.Fatalf("con vínculo y sin started_at procede: código %d, salida %s", code, out)
	}
}

func TestIntegrationsTimeCloseOnlyInCheckpointedIsAUsageErrorWithoutRequests(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionCheckpointed, "")
	before := len(e.srv.Requests())

	code, _, errOut := runIntegrationsCLI(t, "", "time", "record", "--change", "x", "--role", "fullstack", "--close-only", "--cwd", e.root)

	if code != 2 || !strings.Contains(errOut, "--close-only") {
		t.Fatalf("código %d, stderr %q", code, errOut)
	}
	e.noRequests(t, before)
}

// ---------------------------------------------------------------- diario pendiente (D9, D10, D21)

// El PATCH se aplicó pero el registro local falló: la intención sobrevive y el siguiente time record
// la reconcilia sin ningún PATCH más.
func TestIntegrationsTimeRecordLocalFailureAfterPatchIsRecoveredByTheNextRecord(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
	trackerRecordTimeEntry = func(integrations.Folder, string, integrations.TimeEntry, time.Time) (bool, error) {
		return false, errors.New("disco lleno")
	}

	code, got, _, _ := e.record(t, "--hours", "2")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "write_failed" || len(e.entries(t)) != 0 {
		t.Fatalf("código %d, %+v", code, got.Refusal)
	}
	if p, ok := e.pending(t); !ok || p.Source != "role.close/fullstack/1" || p.BaseRev != 4 || p.ExpectedCompleted != 5 || p.ExpectedRemaining != 3 || p.Actor != timeActor {
		t.Fatalf("la intención se conserva: %+v", p)
	}
	patches := e.requests("PATCH")
	trackerRecordTimeEntry = integrations.RecordTimeEntry

	code, got, out, _ := e.record(t, "--hours", "2")

	if code != 0 || got.Pending != "recovered" || warningCodes(got.Warnings) != "time_record_recovered" {
		t.Fatalf("código %d, salida %s", code, out)
	}
	if !strings.Contains(got.Warnings[0].Next, "axiom integrations time propose --change x --role fullstack") {
		t.Errorf("la ejecución recuperada nombra time propose: %q", got.Warnings[0].Next)
	}
	if e.requests("PATCH") != patches+1 { // solo el PATCH de la transición de cierre
		t.Errorf("no hay ningún PATCH de horas más: %d antes, %d después", patches, e.requests("PATCH"))
	}
	if entries := e.entries(t); len(entries) != 1 || entries[0].Source != "role.close/fullstack/1" || entries[0].Minutes != 120 {
		t.Errorf("imputación de la intención: %+v", entries)
	}
	if _, ok := e.pending(t); ok {
		t.Errorf("la intención ya no está")
	}
	if got.Transition == nil || e.state(t, timeTaskID) != "Done" {
		t.Errorf("en continuous la transición de cierre se aplica tras recuperar: %+v", got.Transition)
	}
	if e.task(t, timeTaskID).Rev != 6 || fieldNumber(t, e.task(t, timeTaskID), hoursCompletedField) != 5 {
		t.Errorf("las horas no se reaplican: %+v", e.task(t, timeTaskID))
	}
}

func TestIntegrationsTimeRecordNotAppliedIntentIsDroppedAndTheFlowContinues(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
	e.putPending(t, intent())

	code, got, out, _ := e.record(t, "--hours", "2")

	if code != 0 || got.Recorded == nil || got.Recorded.Source != "role.close/fullstack/1" {
		t.Fatalf("código %d, salida %s", code, out)
	}
	if fieldNumber(t, e.task(t, timeTaskID), hoursCompletedField) != 5 || e.srv.AppliedPatches(timeTaskID) < 1 || len(e.entries(t)) != 1 {
		t.Errorf("la imputación sigue el flujo normal desde la revisión 4")
	}
	if _, ok := e.pending(t); ok {
		t.Errorf("la intención se borra")
	}
}

func TestIntegrationsTimeRecordAmbiguousIntentIsRefusedWithoutWriting(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
	e.seedTimeTask(timeTaskID, 6, 8, 5, 7)
	e.putPending(t, intent())
	linksBefore := e.linksFile(t)

	code, got, out, _ := e.record(t, "--hours", "2")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "time_pending_ambiguous" {
		t.Fatalf("código %d, salida %s", code, out)
	}
	if !strings.Contains(got.Refusal.Text, "revisión 6") || !strings.Contains(got.Refusal.Text, "CompletedWork 7") ||
		got.Refusal.Next != "axiom integrations time record --change x --role fullstack --resolve-pending applied|discarded" {
		t.Errorf("valores releídos y resolución: %+v", got.Refusal)
	}
	if e.requests("PATCH") != 0 || !bytes.Equal(e.linksFile(t), linksBefore) {
		t.Errorf("no se escribe nada en el tracker ni en integrations.yaml")
	}
	if _, ok := e.pending(t); !ok {
		t.Errorf("la intención se conserva")
	}
}

func TestIntegrationsTimeResolvePending(t *testing.T) {
	t.Run("applied registra sin tocar el horario del tracker y cierra en continuous", func(t *testing.T) {
		e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
		e.putPending(t, intent())

		code, got, out, _ := e.record(t, "--resolve-pending", "applied")

		if code != 0 || got.Pending != "applied" || got.Recorded == nil {
			t.Fatalf("código %d, salida %s", code, out)
		}
		if entries := e.entries(t); len(entries) != 1 || entries[0].Source != "role.close/fullstack/1" || entries[0].Minutes != 120 {
			t.Errorf("imputación registrada: %+v", entries)
		}
		if _, ok := e.pending(t); ok {
			t.Errorf("la intención se borra")
		}
		if fieldNumber(t, e.task(t, timeTaskID), hoursCompletedField) != 3 {
			t.Errorf("no toca las horas del tracker")
		}
		if e.state(t, timeTaskID) != "Done" || got.Transition == nil {
			t.Errorf("la transición de cierre se aplica tras applied en continuous")
		}
	})
	t.Run("applied en checkpointed no toca el tracker", func(t *testing.T) {
		e := newTimeEnv(t, kickoff.ExecutionCheckpointed, "")
		e.putPending(t, intent())
		before := len(e.srv.Requests())

		code, _, out, _ := e.record(t, "--resolve-pending", "applied")

		if code != 0 || len(e.entries(t)) != 1 {
			t.Fatalf("código %d, salida %s", code, out)
		}
		e.noRequests(t, before)
	})
	t.Run("discarded borra sin registrar", func(t *testing.T) {
		e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
		e.putPending(t, intent())
		before := len(e.srv.Requests())

		code, got, out, _ := e.record(t, "--resolve-pending", "discarded")

		if code != 0 || got.Pending != "discarded" || len(e.entries(t)) != 0 || got.Recorded != nil {
			t.Fatalf("código %d, salida %s", code, out)
		}
		if _, ok := e.pending(t); ok {
			t.Errorf("la intención se borra")
		}
		e.noRequests(t, before)
	})
}

// La reconciliación relee la Task de la intención (205) aunque el vínculo ya sea el de la 206.
func TestIntegrationsTimeRecordReconcilesTheIntentTaskAfterReplace(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
	e.putPending(t, intent())
	e.seedTimeTask(206, 2, 4, 4, 0)
	if _, err := integrations.ReplaceTaskLink(e.folder, "azure-devops", "fullstack", integrations.Link{
		Plugin: "azure-devops", Kind: integrations.TaskLinkKind, ID: "206", URL: e.workItemAPI(206), Role: "fullstack", Source: integrations.TasksBindSource, EstimateHours: hoursPtr(4),
	}, time.Now().UTC()); err != nil {
		t.Fatalf("ReplaceTaskLink: %v", err)
	}

	code, _, out, _ := e.record(t, "--hours", "1")

	if code != 0 {
		t.Fatalf("código %d, salida %s", code, out)
	}
	first := ""
	for _, r := range e.srv.Requests() {
		if r.Method == http.MethodGet && strings.Contains(r.Path, "/workitems/20") {
			first = r.Path
			break
		}
	}
	if !strings.HasSuffix(first, "/workitems/205") {
		t.Errorf("la primera lectura es la Task de la intención (205): %q", first)
	}
	if fieldNumber(t, e.task(t, 206), hoursCompletedField) != 1 || fieldNumber(t, e.task(t, timeTaskID), hoursCompletedField) != 3 {
		t.Errorf("las horas nuevas van a la Task vigente (206)")
	}
	if entries := e.entries(t); len(entries) != 1 || entries[0].ItemID != "206" {
		t.Errorf("imputación: %+v", entries)
	}
}

// Un corte de conexión sin respuesta conserva la intención; el siguiente time record la reconcilia.
func TestIntegrationsTimeRecordConnectionLossKeepsTheIntentAndTheNextRecordReconciles(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionCheckpointed, "")
	path := e.workitemPath(t)
	e.srv.Sequence(http.MethodPatch, path, adotest.Response{Drop: true})

	code, got, out, _ := e.record(t, "--hours", "2")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "tracker_unavailable" || warningCodes(got.Warnings) != "time_pending" || got.Pending != "kept" {
		t.Fatalf("código %d, salida %s", code, out)
	}
	if len(e.entries(t)) != 0 {
		t.Errorf("no se registra ninguna imputación")
	}
	p, ok := e.pending(t)
	if !ok || p.BaseRev != 4 {
		t.Fatalf("la intención se conserva: %+v", p)
	}

	// El PATCH sí se había aplicado en el tracker.
	e.seedTimeTask(timeTaskID, 5, 8, 3, 5)
	patches := e.requests("PATCH")

	code, got, out, _ = e.record(t, "--hours", "2")

	if code != 0 || got.Pending != "recovered" || len(e.entries(t)) != 1 {
		t.Fatalf("la reconciliación registra la imputación: código %d, salida %s", code, out)
	}
	if e.task(t, timeTaskID).Rev != 5 || fieldNumber(t, e.task(t, timeTaskID), hoursCompletedField) != 5 || e.requests("PATCH") != patches {
		t.Errorf("no se reaplican horas: %+v", e.task(t, timeTaskID))
	}
}

func TestIntegrationsTimeRecordUncertainAndDefinitiveFailuresOfThePatch(t *testing.T) {
	cases := []struct {
		name        string
		response    adotest.Response
		wantCode    string
		wantPending bool
	}{
		{"5xx tras el presupuesto: incierto", adotest.Response{Status: http.StatusInternalServerError, JSON: map[string]any{"message": "boom"}}, "tracker_unavailable", true},
		{"4xx que no es conflicto: no se aplicó", adotest.Response{Status: http.StatusBadRequest, JSON: map[string]any{"message": "TF401320: regla", "typeKey": "RuleValidationException"}}, "tracker_rejected", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
			e.srv.Reply(http.MethodPatch, e.workitemPath(t), tc.response)

			code, got, out, _ := e.record(t, "--hours", "2")

			if code != 1 || got.Refusal == nil || got.Refusal.Code != tc.wantCode {
				t.Fatalf("código %d, salida %s", code, out)
			}
			if _, ok := e.pending(t); ok != tc.wantPending {
				t.Errorf("intención conservada: %v, quería %v", ok, tc.wantPending)
			}
			if tc.wantPending && warningCodes(got.Warnings) != "time_pending" {
				t.Errorf("el resultado incierto avisa time_pending: %+v", got.Warnings)
			}
			if len(e.entries(t)) != 0 {
				t.Errorf("un fallo del tracker no registra la imputación")
			}
		})
	}
}

// Una respuesta perdida dentro de la misma ejecución se verifica (D10) y cuenta como éxito.
func TestIntegrationsTimeRecordLostResponseIsVerifiedWithinTheSameRun(t *testing.T) {
	e := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
	e.srv.LoseNextPatchResponse()

	code, got, out, _ := e.record(t, "--hours", "2")

	if code != 0 || got.Recorded == nil || !got.Recorded.Recovered {
		t.Fatalf("código %d, salida %s", code, out)
	}
	if fieldNumber(t, e.task(t, timeTaskID), hoursCompletedField) != 5 || len(e.entries(t)) != 1 {
		t.Errorf("las horas se aplican una sola vez")
	}
}

// ---------------------------------------------------------------- diario de continuous y ayuda

func TestIntegrationsTimeTrackerNotesOnlyInContinuous(t *testing.T) {
	cont := newTimeEnv(t, kickoff.ExecutionContinuous, closeToDone)
	cont.propose(t)
	cont.record(t, "--hours", "2")
	cont.record(t, "--close-only")
	notes := notesOf(t, cont.tasksEnv)
	var verbs []string
	for _, n := range notes {
		verbs = append(verbs, n.Verb+":"+n.Code)
	}
	if got := strings.Join(verbs, ","); !strings.Contains(got, "time propose:done") || !strings.Contains(got, "time record:done") {
		t.Errorf("continuous anota propose y record con done: %v", verbs)
	}

	check := newTimeEnv(t, kickoff.ExecutionCheckpointed, "")
	check.propose(t)
	check.record(t, "--hours", "2")
	if _, err := os.Stat(filepath.Join(check.changeRoot, integrations.TrackerNotesFile)); !os.IsNotExist(err) {
		t.Errorf("checkpointed no escribe el diario: %v", err)
	}
}

func TestIntegrationsTimeHelpListsTheFamilyAndVerbs(t *testing.T) {
	code, out, _ := runIntegrationsCLI(t, "", "time", "--help")
	if code != 0 || !strings.Contains(out, "propose") || !strings.Contains(out, "record") || !strings.Contains(out, "--json") {
		t.Fatalf("ayuda de time: código %d\n%s", code, out)
	}
	for verb, flags := range map[string][]string{
		"propose": {"--change", "--role", "--cwd", "--json"},
		"record":  {"--change", "--role", "--hours", "--resolve-pending", "--close-only", "--cwd", "--json"},
	} {
		code, out, _ = runIntegrationsCLI(t, "", "time", verb, "--help")
		if code != 0 {
			t.Fatalf("time %s --help: código %d", verb, code)
		}
		for _, flag := range flags {
			if !strings.Contains(out, flag) {
				t.Errorf("la ayuda de time %s debe describir %s", verb, flag)
			}
		}
	}
	if code, _, _ = runIntegrationsCLI(t, "", "time", "inspect"); code != 2 {
		t.Errorf("un verbo desconocido de time es error de uso: %d", code)
	}
}
