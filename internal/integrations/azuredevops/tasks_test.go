package azuredevops_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

const (
	taskParent = 42
	taskMark   = "axiom-task:cambio:fullstack"

	createRoute = "/contoso/Fabrikam/_apis/wit/workitems/$Task"
	wiqlRoute   = "/contoso/Fabrikam/_apis/wit/wiql"

	hoursOriginal  = "Microsoft.VSTS.Scheduling.OriginalEstimate"
	hoursRemaining = "Microsoft.VSTS.Scheduling.RemainingWork"
	hoursCompleted = "Microsoft.VSTS.Scheduling.CompletedWork"
)

// taskEnv es el escenario de las pruebas del puerto de Tasks: un work item padre, los estados y los
// campos del tipo Task, y el plugin apuntado al doble.
type taskEnv struct {
	srv  *adotest.Server
	p    *azuredevops.Plugin
	port integrations.PortRequest
}

func newTaskEnv(t *testing.T, fields ...string) *taskEnv {
	t.Helper()
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(taskParent, 3, map[string]any{
		"System.WorkItemType":  "User Story",
		"System.Title":         "Historia de origen",
		"System.AreaPath":      `Fabrikam\Area`,
		"System.IterationPath": `Fabrikam\Sprint 1`,
	})
	srv.SeedStates("Task",
		adotest.State{Name: "To Do", Category: "Proposed"},
		adotest.State{Name: "Active", Category: "InProgress"},
		adotest.State{Name: "Tech Review", Category: "InProgress"},
		adotest.State{Name: "Done", Category: "Completed"},
		adotest.State{Name: "Removed", Category: "Removed"})
	if len(fields) == 0 {
		fields = []string{hoursOriginal, hoursRemaining, hoursCompleted}
	}
	srv.SeedTypeFields("Task", fields...)
	return &taskEnv{
		srv: srv,
		p:   azuredevops.New(srv.Options()),
		port: integrations.PortRequest{
			Config:  map[string]any{"organization": "contoso", "project": "Fabrikam", "team": "Fabrikam Team"},
			Secrets: &patSource{pat: testPAT},
		},
	}
}

// seedTask siembra una Task con la marca, el estado y, si parent es distinto de 0, la relación
// Hierarchy-Reverse hacia ese padre.
func (e *taskEnv) seedTask(id, rev int, tags, state string, parent int, extra map[string]any) {
	fields := map[string]any{"System.WorkItemType": "Task", "System.Title": "otra", "System.Tags": tags, "System.State": state}
	for k, v := range extra {
		fields[k] = v
	}
	var rels []map[string]any
	if parent != 0 {
		rels = append(rels, map[string]any{"rel": "System.LinkTypes.Hierarchy-Reverse", "url": e.workItemAPI(parent)})
	}
	e.srv.SeedWorkItem(id, rev, fields, rels...)
}

func (e *taskEnv) workItemAPI(id int) string {
	return e.srv.URL() + "/contoso/_apis/wit/workItems/" + itoa(id)
}

// linkChild añade al padre la relación Hierarchy-Forward hacia un hijo sembrado a mano.
func (e *taskEnv) linkChild(parent *adotest.WorkItem, child int) {
	parent.Relations = append(parent.Relations, map[string]any{"rel": "System.LinkTypes.Hierarchy-Forward", "url": e.workItemAPI(child)})
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func (e *taskEnv) ensure(t *testing.T, hours float64) (integrations.CreateTaskResult, error) {
	t.Helper()
	return e.p.EnsureChildTask(context.Background(), integrations.CreateTaskRequest{
		PortRequest: e.port, ParentID: itoa(taskParent), Title: "cambio / fullstack", Mark: taskMark, EstimateHours: hours,
	})
}

func (e *taskEnv) read(t *testing.T, id int) integrations.TaskItem {
	t.Helper()
	item, err := e.p.ReadTask(context.Background(), integrations.TaskRequest{PortRequest: e.port, TaskID: itoa(id)})
	if err != nil {
		t.Fatalf("ReadTask(%d): %v", id, err)
	}
	return item
}

func patchOps(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var ops []map[string]any
	if err := json.Unmarshal(body, &ops); err != nil {
		t.Fatalf("cuerpo JSON Patch ilegible: %v", err)
	}
	return ops
}

func opValue(ops []map[string]any, path string) any {
	for _, op := range ops {
		if op["path"] == path {
			return op["value"]
		}
	}
	return nil
}

func TestEnsureChildTaskCreatesWithMarkRelationHoursAndPaths(t *testing.T) {
	e := newTaskEnv(t)

	res, err := e.ensure(t, 8)

	if err != nil {
		t.Fatalf("EnsureChildTask: %v", err)
	}
	if res.Outcome != integrations.TaskCreated || res.Task.ID == "" || res.Task.ParentID != "42" {
		t.Fatalf("resultado = %+v", res)
	}
	if e.srv.Created() != 1 || e.srv.Count("POST", createRoute) != 1 {
		t.Fatalf("creaciones = %d, POST = %d; esperaba 1", e.srv.Created(), e.srv.Count("POST", createRoute))
	}
	var post adotest.Request
	for _, r := range e.srv.Requests() {
		if r.Method == "POST" && r.Path == createRoute {
			post = r
		}
	}
	if ct := post.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json-patch+json") {
		t.Errorf("Content-Type = %q", ct)
	}
	ops := patchOps(t, post.Body)
	for path, want := range map[string]any{
		"/fields/System.Title":         "cambio / fullstack",
		"/fields/System.Tags":          taskMark,
		"/fields/System.AreaPath":      `Fabrikam\Area`,
		"/fields/System.IterationPath": `Fabrikam\Sprint 1`,
		"/fields/" + hoursOriginal:     8.0,
		"/fields/" + hoursRemaining:    8.0,
	} {
		if got := opValue(ops, path); got != want {
			t.Errorf("%s = %v, esperaba %v", path, got, want)
		}
	}
	rel, _ := opValue(ops, "/relations/-").(map[string]any)
	if rel["rel"] != "System.LinkTypes.Hierarchy-Reverse" || !strings.HasSuffix(rel["url"].(string), "/contoso/_apis/wit/workItems/42") {
		t.Errorf("relación = %v", rel)
	}
	for _, r := range e.srv.Requests() {
		if strings.Contains(string(r.Body), "/rev") && r.Method == "POST" && r.Path == createRoute {
			t.Errorf("la creación no lleva test /rev: %s", r.Body)
		}
	}
	children, err := e.p.ChildTasks(context.Background(), integrations.ChildTasksRequest{PortRequest: e.port, ParentID: "42"})
	if err != nil || len(children) != 1 || children[0].ID != res.Task.ID || children[0].Hours.Original != 8 {
		t.Fatalf("hijas = %+v, %v", children, err)
	}
}

func TestEnsureChildTaskWithoutEstimateWritesNoHours(t *testing.T) {
	e := newTaskEnv(t)

	if _, err := e.ensure(t, 0); err != nil {
		t.Fatalf("EnsureChildTask: %v", err)
	}

	for _, r := range e.srv.Requests() {
		if r.Method == "POST" && r.Path == createRoute && strings.Contains(string(r.Body), "Scheduling") {
			t.Fatalf("sin estimación no se escriben horas: %s", r.Body)
		}
	}
}

func TestEnsureChildTaskReusesByMarkIgnoringCase(t *testing.T) {
	e := newTaskEnv(t)
	first, err := e.ensure(t, 8)
	if err != nil {
		t.Fatalf("primera: %v", err)
	}

	second, err := e.ensure(t, 8)

	if err != nil || second.Outcome != integrations.TaskFoundByMark || second.Task.ID != first.Task.ID {
		t.Fatalf("segunda = %+v, %v", second, err)
	}
	if e.srv.Created() != 1 {
		t.Fatalf("creaciones = %d, esperaba 1", e.srv.Created())
	}
	// Azure DevOps puede reescribir la grafía de la etiqueta: la coincidencia ignora mayúsculas.
	e2 := newTaskEnv(t)
	parent, _ := e2.srv.WorkItem(taskParent)
	e2.seedTask(50, 1, strings.ToUpper(taskMark), "To Do", taskParent, nil)
	e2.linkChild(&parent, 50)
	e2.srv.SeedWorkItem(taskParent, 3, parent.Fields, parent.Relations...)
	got, err := e2.ensure(t, 8)
	if err != nil || got.Outcome != integrations.TaskFoundByMark || got.Task.ID != "50" || e2.srv.Created() != 0 {
		t.Fatalf("mayúsculas: %+v, %v, creadas %d", got, err, e2.srv.Created())
	}
}

func TestEnsureChildTaskLostPostResponseIsNotRetriedAndNextAttemptFindsIt(t *testing.T) {
	e := newTaskEnv(t)
	e.srv.LoseNextCreateResponse()

	_, err := e.ensure(t, 8)

	var transport *azuredevops.TransportError
	if !errors.As(err, &transport) || !errors.Is(err, azuredevops.ErrTransport) {
		t.Fatalf("esperaba TransportError: %v", err)
	}
	if got := e.srv.Count("POST", createRoute); got != 1 {
		t.Fatalf("POST = %d, esperaba 1 (sin reintento)", got)
	}
	again, err := e.ensure(t, 8)
	if err != nil || again.Outcome != integrations.TaskFoundByMark {
		t.Fatalf("siguiente intento = %+v, %v", again, err)
	}
	if got := e.srv.Count("POST", createRoute); got != 1 {
		t.Fatalf("POST = %d: el siguiente intento no debe crear otra Task", got)
	}
}

func TestEnsureChildTaskTwoMarksIsAmbiguous(t *testing.T) {
	e := newTaskEnv(t)
	parent, _ := e.srv.WorkItem(taskParent)
	e.seedTask(60, 1, taskMark, "To Do", taskParent, nil)
	e.seedTask(61, 1, taskMark, "Active", taskParent, nil)
	e.linkChild(&parent, 60)
	e.linkChild(&parent, 61)
	e.srv.SeedWorkItem(taskParent, 3, parent.Fields, parent.Relations...)

	_, err := e.ensure(t, 8)

	var amb *integrations.TaskAmbiguousError
	if !errors.Is(err, integrations.ErrTaskAmbiguous) || !errors.As(err, &amb) || strings.Join(amb.IDs, ",") != "60,61" {
		t.Fatalf("esperaba task_ambiguous con 60 y 61: %v", err)
	}
	if e.srv.Created() != 0 {
		t.Fatal("una marca ambigua no crea")
	}
}

func TestEnsureChildTaskMarkUnderAnotherParentIsAmbiguous(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(70, 1, taskMark, "To Do", 7, nil)

	_, err := e.ensure(t, 8)

	if !errors.Is(err, integrations.ErrTaskAmbiguous) {
		t.Fatalf("esperaba task_ambiguous: %v", err)
	}
	if e.srv.Created() != 0 {
		t.Fatal("no se crea junto a una Task de otro padre")
	}
}

func TestEnsureChildTaskTitleDoesNotDecideAndRemovedIsIgnored(t *testing.T) {
	e := newTaskEnv(t)
	parent, _ := e.srv.WorkItem(taskParent)
	// Mismo título, sin etiqueta: el título no decide.
	e.seedTask(80, 1, "otra", "To Do", taskParent, map[string]any{"System.Title": "cambio / fullstack"})
	// Con la marca pero retirada: se ignora.
	e.seedTask(81, 1, taskMark, "Removed", taskParent, nil)
	e.linkChild(&parent, 80)
	e.linkChild(&parent, 81)
	e.srv.SeedWorkItem(taskParent, 3, parent.Fields, parent.Relations...)

	res, err := e.ensure(t, 8)

	if err != nil || res.Outcome != integrations.TaskCreated {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
}

func TestEnsureChildTaskRepairsOrphanFoundByWIQL(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(90, 4, taskMark, "To Do", 0, nil)

	res, err := e.ensure(t, 8)

	if err != nil || res.Outcome != integrations.TaskRepaired || res.Task.ID != "90" || res.Task.ParentID != "42" {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	if e.srv.Count("POST", createRoute) != 0 {
		t.Fatal("reparar no crea")
	}
	var patch adotest.Request
	for _, r := range e.srv.Requests() {
		if r.Method == "PATCH" {
			patch = r
		}
	}
	ops := patchOps(t, patch.Body)
	if ops[0]["op"] != "test" || ops[0]["path"] != "/rev" || ops[0]["value"] != 4.0 {
		t.Fatalf("el PATCH de reparación empieza por test /rev = 4: %v", ops)
	}
}

func TestEnsureChildTaskWIQLDownDoesNotCreate(t *testing.T) {
	e := newTaskEnv(t)
	e.srv.FailWIQL(500)

	_, err := e.ensure(t, 8)

	if err == nil {
		t.Fatal("esperaba el error del tracker")
	}
	if e.srv.Created() != 0 || e.srv.Count("POST", createRoute) != 0 {
		t.Fatalf("con WIQL caído no se crea: creadas %d", e.srv.Created())
	}
}

func TestEnsureChildTaskFallbackWhenCreateWithRelationIsRejected(t *testing.T) {
	e := newTaskEnv(t)
	e.srv.RejectCreateRelations(true)

	res, err := e.ensure(t, 8)

	if err != nil || res.Outcome != integrations.TaskCreated {
		t.Fatalf("resultado = %+v, %v", res, err)
	}
	if got := e.srv.Count("POST", createRoute); got != 2 {
		t.Fatalf("POST = %d, esperaba 2 (con relación y sin ella)", got)
	}
	if e.srv.Created() != 1 {
		t.Fatalf("creadas = %d: el 400 no crea nada", e.srv.Created())
	}
	children, err := e.p.ChildTasks(context.Background(), integrations.ChildTasksRequest{PortRequest: e.port, ParentID: "42"})
	if err != nil || len(children) != 1 || children[0].ID != res.Task.ID {
		t.Fatalf("hijas = %+v, %v", children, err)
	}
}

func TestEnsureChildTaskFallbackLinkFailureNamesTheTask(t *testing.T) {
	e := newTaskEnv(t)
	e.srv.RejectCreateRelations(true)
	e.srv.Reply("PATCH", "/contoso/Fabrikam/_apis/wit/workitems/43", adotest.Response{Status: 400, JSON: map[string]any{"message": "regla del proceso"}})

	_, err := e.ensure(t, 8)

	if !errors.Is(err, integrations.ErrTaskParentLinkFailed) || !strings.Contains(err.Error(), "43") {
		t.Fatalf("esperaba task_parent_link_failed nombrando la Task 43: %v", err)
	}
}

func TestEnsureChildTaskMissingHoursFieldDoesNotPost(t *testing.T) {
	e := newTaskEnv(t, hoursOriginal, hoursCompleted)

	_, err := e.ensure(t, 8)

	var missing *integrations.FieldMissingError
	if !errors.As(err, &missing) || missing.Field != hoursRemaining || !errors.Is(err, integrations.ErrFieldMissing) {
		t.Fatalf("esperaba field_missing de %s: %v", hoursRemaining, err)
	}
	if e.srv.Count("POST", createRoute) != 0 {
		t.Fatal("sin campo de horas no hay POST")
	}
}

func TestEnsureChildTaskWarnsWhenTwoTasksCarryTheMarkAfterCreating(t *testing.T) {
	e := newTaskEnv(t)
	// Otra máquina creó una Task con la marca justo antes: aparece como hija tras nuestra creación.
	e.srv.Handle("POST", createRoute, func(*adotest.Request) adotest.Response {
		e.seedTask(95, 1, taskMark, "To Do", taskParent, nil)
		e.seedTask(96, 1, taskMark, "To Do", taskParent, nil)
		parent, _ := e.srv.WorkItem(taskParent)
		e.linkChild(&parent, 95)
		e.linkChild(&parent, 96)
		e.srv.SeedWorkItem(taskParent, 3, parent.Fields, parent.Relations...)
		return adotest.Response{JSON: map[string]any{
			"id": 96, "rev": 1, "fields": map[string]any{"System.WorkItemType": "Task", "System.Tags": taskMark, "System.State": "To Do"},
		}}
	})

	res, err := e.ensure(t, 8)

	if err != nil || res.Outcome != integrations.TaskCreated || res.Task.ID != "96" {
		t.Fatalf("EnsureChildTask: %+v, %v", res, err)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Code != "task_duplicated" ||
		res.Warnings[0].Continuation != "axiom integrations tasks bind" || !strings.Contains(res.Warnings[0].Message, "95, 96") {
		t.Fatalf("avisos = %+v", res.Warnings)
	}
}

func TestValidateTask(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(100, 2, taskMark, "To Do", taskParent, nil)
	e.srv.SeedWorkItem(101, 1, map[string]any{"System.WorkItemType": "Bug", "System.State": "New"})
	e.seedTask(102, 1, "x", "To Do", 7, nil)
	e.seedTask(103, 1, "x", "To Do", 0, nil)
	validate := func(id string) (integrations.TaskItem, error) {
		return e.p.ValidateTask(context.Background(), integrations.TaskRequest{PortRequest: e.port, TaskID: id, ParentID: "42"})
	}

	item, err := validate("100")
	if err != nil || item.ID != "100" || item.ParentID != "42" || item.Rev != 2 {
		t.Fatalf("válida: %+v, %v", item, err)
	}
	for id, want := range map[string]error{
		"999": integrations.ErrTaskNotFound,
		"101": integrations.ErrNotATask,
		"102": integrations.ErrNotChildOfWorkItem,
		"103": integrations.ErrNotChildOfWorkItem,
	} {
		if _, err := validate(id); !errors.Is(err, want) {
			t.Errorf("ValidateTask(%s) = %v, esperaba %v", id, err, want)
		}
	}
}

func TestReadTaskReadsHoursAsZeroWhenAbsentAndAssignedIDOnly(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(110, 5, taskMark, "Active", taskParent, map[string]any{
		hoursOriginal:       8.0,
		"System.AssignedTo": map[string]any{"id": "guid-1", "displayName": "Ana", "uniqueName": "ana.unique.fixture"},
	})

	item := e.read(t, 110)

	if item.Hours.Original != 8 || !item.Hours.HasOriginal || item.Hours.Completed != 0 || item.Hours.Remaining != 0 {
		t.Fatalf("horas = %+v", item.Hours)
	}
	if item.AssignedToID != "guid-1" || item.StateCategory != "InProgress" || item.State != "Active" {
		t.Fatalf("item = %+v", item)
	}
	if strings.Contains(strings.ToLower(jsonOf(t, item)), "ana.unique") {
		t.Fatal("el uniqueName nunca sale en la lectura neutral")
	}
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func (e *taskEnv) record(t *testing.T, base integrations.TaskItem, hours float64) (integrations.HoursResult, error) {
	t.Helper()
	return e.p.RecordHours(context.Background(), integrations.HoursRequest{PortRequest: e.port, Base: base, Hours: hours})
}

func lastPatch(t *testing.T, e *taskEnv) adotest.Request {
	t.Helper()
	var last adotest.Request
	for _, r := range e.srv.Requests() {
		if r.Method == "PATCH" {
			last = r
		}
	}
	if last.Method == "" {
		t.Fatal("no hubo ningún PATCH")
	}
	return last
}

func TestRecordHoursWritesCompletedAndRemainingWithTestRev(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(120, 6, taskMark, "Active", taskParent, map[string]any{hoursOriginal: 8.0, hoursRemaining: 5.0, hoursCompleted: 3.0})
	base := e.read(t, 120)

	res, err := e.record(t, base, 1.25)

	if err != nil || res.Recovered {
		t.Fatalf("RecordHours: %+v, %v", res, err)
	}
	ops := patchOps(t, lastPatch(t, e).Body)
	if ops[0]["op"] != "test" || ops[0]["path"] != "/rev" || ops[0]["value"] != 6.0 {
		t.Fatalf("test /rev = %v", ops[0])
	}
	if got := opValue(ops, "/fields/"+hoursCompleted); got != 4.25 {
		t.Errorf("CompletedWork = %v, esperaba 4.25", got)
	}
	if got := opValue(ops, "/fields/"+hoursRemaining); got != 3.75 {
		t.Errorf("RemainingWork = %v, esperaba 3.75", got)
	}
	if res.Task.Hours.Completed != 4.25 || res.Task.Rev != 7 {
		t.Fatalf("Task = %+v", res.Task)
	}
}

func TestRecordHoursRemainingFloorsAtZero(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(121, 1, taskMark, "Active", taskParent, map[string]any{hoursRemaining: 2.0, hoursCompleted: 0.0})

	if _, err := e.record(t, e.read(t, 121), 6); err != nil {
		t.Fatalf("RecordHours: %v", err)
	}

	if got := opValue(patchOps(t, lastPatch(t, e).Body), "/fields/"+hoursRemaining); got != 0.0 {
		t.Fatalf("RemainingWork = %v, esperaba 0", got)
	}
}

func TestRecordHoursReadsAbsentCompletedAsZero(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(122, 1, taskMark, "Active", taskParent, map[string]any{hoursOriginal: 4.0})

	if _, err := e.record(t, e.read(t, 122), 2); err != nil {
		t.Fatalf("RecordHours: %v", err)
	}

	if got := opValue(patchOps(t, lastPatch(t, e).Body), "/fields/"+hoursCompleted); got != 2.0 {
		t.Fatalf("CompletedWork = %v, esperaba 2", got)
	}
}

func TestRecordHoursMissingProcessFieldWritesNothing(t *testing.T) {
	e := newTaskEnv(t, hoursOriginal, hoursRemaining) // sin CompletedWork en el proceso
	e.seedTask(123, 1, taskMark, "Active", taskParent, nil)

	_, err := e.record(t, e.read(t, 123), 2)

	var missing *integrations.FieldMissingError
	if !errors.As(err, &missing) || missing.Field != hoursCompleted {
		t.Fatalf("esperaba field_missing de %s: %v", hoursCompleted, err)
	}
	if e.srv.AppliedPatches(123) != 0 {
		t.Fatal("no se escribe")
	}
}

func TestRecordHoursStaleRevIsAConflictWithRereadValuesAndNoReapply(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(124, 9, taskMark, "Active", taskParent, map[string]any{hoursRemaining: 7.0, hoursCompleted: 1.0})
	base := e.read(t, 124)
	base.Rev = 8 // la lectura previa está obsoleta

	_, err := e.record(t, base, 1)

	var conflict *integrations.RevConflictError
	if !errors.Is(err, integrations.ErrRevConflict) || !errors.As(err, &conflict) {
		t.Fatalf("esperaba rev_conflict: %v", err)
	}
	if conflict.Rev != 9 || conflict.Hours.Completed != 1 || conflict.Hours.Remaining != 7 || conflict.AfterRetry {
		t.Fatalf("conflicto = %+v", conflict)
	}
	if e.srv.AppliedPatches(124) != 0 {
		t.Fatal("el conflicto no reaplica")
	}
	if n := e.srv.Count("PATCH", "/contoso/Fabrikam/_apis/wit/workitems/124"); n != 1 {
		t.Fatalf("PATCH = %d, esperaba 1 (sin reaplicar)", n)
	}
}

func TestRecordHoursLostResponseIsRecoveredWhenTheRereadVerifies(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(125, 2, taskMark, "Active", taskParent, map[string]any{hoursRemaining: 5.0, hoursCompleted: 0.0})
	base := e.read(t, 125)
	e.srv.LoseNextPatchResponse()

	res, err := e.record(t, base, 2)

	if err != nil || !res.Recovered {
		t.Fatalf("esperaba recuperada: %+v, %v", res, err)
	}
	if got := e.srv.AppliedPatches(125); got != 1 {
		t.Fatalf("PATCH aplicados = %d, esperaba 1", got)
	}
	if res.Task.Hours.Completed != 2 || res.Task.Rev != 3 {
		t.Fatalf("Task = %+v", res.Task)
	}
}

func TestRecordHoursConflictAfterRetryThatDoesNotVerifyIsUncertain(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(126, 2, taskMark, "Active", taskParent, map[string]any{hoursRemaining: 5.0, hoursCompleted: 0.0})
	base := e.read(t, 126)
	e.srv.Sequence("PATCH", "/contoso/Fabrikam/_apis/wit/workitems/126",
		adotest.Response{Drop: true},
		adotest.Response{Status: 412, JSON: map[string]any{"message": "TF26071", "typeKey": "WorkItemRevisionMismatchException"}})

	_, err := e.record(t, base, 2)

	var conflict *integrations.RevConflictError
	if !errors.As(err, &conflict) || !conflict.AfterRetry {
		t.Fatalf("esperaba rev_conflict con AfterRetry: %v", err)
	}
}

func TestRecordHoursTransportErrorOnBothAttemptsIsUncertainNotARejection(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(127, 2, taskMark, "Active", taskParent, map[string]any{hoursRemaining: 5.0, hoursCompleted: 0.0})
	base := e.read(t, 127)
	e.srv.Reply("PATCH", "/contoso/Fabrikam/_apis/wit/workitems/127", adotest.Response{Drop: true})

	_, err := e.record(t, base, 2)

	var transport *azuredevops.TransportError
	if !errors.As(err, &transport) || transport.Attempts < 2 {
		t.Fatalf("esperaba TransportError tras reintentar: %v", err)
	}
	if errors.Is(err, integrations.ErrRevConflict) || errors.Is(err, integrations.ErrTrackerRejected) {
		t.Fatalf("un fallo de red sin respuesta no es conflicto ni rechazo: %v", err)
	}
}

func TestRecordHoursRejectionIs4xxAndDistinguishableFromConflict(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(128, 2, taskMark, "Active", taskParent, map[string]any{hoursRemaining: 5.0, hoursCompleted: 0.0})
	base := e.read(t, 128)
	e.srv.Reply("PATCH", "/contoso/Fabrikam/_apis/wit/workitems/128",
		adotest.Response{Status: 400, JSON: map[string]any{"message": "TF401320: regla incumplida", "typeKey": "RuleValidationException"}})

	_, err := e.record(t, base, 2)

	if !errors.Is(err, integrations.ErrTrackerRejected) || errors.Is(err, integrations.ErrRevConflict) {
		t.Fatalf("esperaba tracker_rejected: %v", err)
	}
}

func TestSetEstimateIfMissing(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(130, 1, taskMark, "Active", taskParent, nil)
	e.seedTask(131, 1, taskMark, "Active", taskParent, map[string]any{hoursOriginal: 3.0, hoursRemaining: 3.0})
	set := func(id int) integrations.EstimateResult {
		res, err := e.p.SetEstimate(context.Background(), integrations.EstimateRequest{
			PortRequest: e.port, TaskID: itoa(id), Hours: 6, Mode: integrations.EstimateIfMissing,
		})
		if err != nil {
			t.Fatalf("SetEstimate(%d): %v", id, err)
		}
		return res
	}

	without := set(130)
	with := set(131)

	if !without.Written || without.Task.Hours.Original != 6 || without.Task.Hours.Remaining != 6 {
		t.Fatalf("sin estimación: %+v", without)
	}
	if with.Written || with.Task.Hours.Original != 3 || e.srv.AppliedPatches(131) != 0 {
		t.Fatalf("con estimación no se escribe: %+v", with)
	}
}

func TestSetEstimateReplaceRecomputesRemainingFromCompleted(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(132, 4, taskMark, "Active", taskParent, map[string]any{hoursOriginal: 4.0, hoursRemaining: 1.0, hoursCompleted: 3.0})
	base := e.read(t, 132)

	res, err := e.p.SetEstimate(context.Background(), integrations.EstimateRequest{
		PortRequest: e.port, TaskID: "132", Hours: 10, Mode: integrations.EstimateReplace, Base: base,
	})

	if err != nil || !res.Written || res.Task.Hours.Original != 10 || res.Task.Hours.Remaining != 7 {
		t.Fatalf("SetEstimate: %+v, %v", res, err)
	}
	if ops := patchOps(t, lastPatch(t, e).Body); ops[0]["value"] != 4.0 {
		t.Fatalf("test /rev = %v, esperaba 4", ops[0])
	}
}

func TestSetEstimateReplaceNeedsBaseAndReportsConflict(t *testing.T) {
	e := newTaskEnv(t)
	e.seedTask(133, 4, taskMark, "Active", taskParent, map[string]any{hoursOriginal: 4.0})
	req := integrations.EstimateRequest{PortRequest: e.port, TaskID: "133", Hours: 10, Mode: integrations.EstimateReplace}

	if _, err := e.p.SetEstimate(context.Background(), req); !errors.Is(err, azuredevops.ErrMissingRev) {
		t.Fatalf("sin Base esperaba ErrMissingRev: %v", err)
	}
	req.Base = e.read(t, 133)
	req.Base.Rev = 3
	if _, err := e.p.SetEstimate(context.Background(), req); !errors.Is(err, integrations.ErrRevConflict) {
		t.Fatalf("con Base obsoleta esperaba rev_conflict: %v", err)
	}
}

func TestTaskPortErrorsNeverCarryThePAT(t *testing.T) {
	e := newTaskEnv(t)
	e.srv.Reply("GET", "/contoso/Fabrikam/_apis/wit/workitems/140", adotest.Response{
		Status: 500, JSON: map[string]any{"message": "fallo interno con " + testPAT + " incluido"}})

	_, err := e.p.ReadTask(context.Background(), integrations.TaskRequest{PortRequest: e.port, TaskID: "140"})

	if err == nil || strings.Contains(err.Error(), testPAT) {
		t.Fatalf("el error debe existir y no contener el PAT: %v", err)
	}
	// Una lectura sin PAT configurado no sale a la red.
	empty := integrations.PortRequest{Config: e.port.Config, Secrets: &patSource{pat: ""}}
	before := len(e.srv.Requests())
	if _, err := e.p.ReadTask(context.Background(), integrations.TaskRequest{PortRequest: empty, TaskID: "140"}); err == nil {
		t.Fatal("sin PAT esperaba error")
	}
	if len(e.srv.Requests()) != before {
		t.Fatal("sin PAT no hay petición")
	}
}
