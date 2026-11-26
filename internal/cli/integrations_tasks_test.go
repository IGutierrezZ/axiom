package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

const (
	hoursOriginalField  = "Microsoft.VSTS.Scheduling.OriginalEstimate"
	hoursRemainingField = "Microsoft.VSTS.Scheduling.RemainingWork"
	hoursCompletedField = "Microsoft.VSTS.Scheduling.CompletedWork"

	tasksParentID = 100
	tasksDesign   = "# Diseño\n\n## Estimación por rol\n\n- fullstack: 6 h\n"
)

// tasksEnv es el escenario de los verbos `tasks`: el cambio x sellado en el estilo pedido, el work
// item 100 vinculado y el doble de Azure DevOps con el proceso de las Tasks sembrado.
type tasksEnv struct {
	*adoEnv
	changeRoot string
	folder     integrations.Folder
}

func newTasksEnv(t *testing.T, style kickoff.ExecutionStyle, design string, roles ...string) *tasksEnv {
	t.Helper()
	return newTasksEnvWith(t, adoBlock, style, design, roles...)
}

func newTasksEnvWith(t *testing.T, block string, style kickoff.ExecutionStyle, design string, roles ...string) *tasksEnv {
	t.Helper()
	e := newADOEnv(t, block)
	changeRoot := filepath.Join(e.root, "openspec", "changes", "x")
	if len(roles) == 0 {
		roles = []string{"fullstack"}
	}
	var sealed []kickoff.KickoffRole
	for _, role := range roles {
		sealed = append(sealed, kickoff.KickoffRole{Role: role, GatePolicy: "blocking", TasksFile: "tasks.md", VerifyFile: "verify-report.md"})
	}
	if _, ok, err := kickoff.Seal(changeRoot, kickoff.Kickoff{
		Schema: kickoff.KickoffSchemaV1, Change: "x", SealedBy: "prueba",
		Config:    kickoff.FlowConfig{FlowMode: kickoff.FlowSDD, ExecutionStyle: style, ExecutionStyleSource: "explicit", HandoffPolicy: kickoff.HandoffNone, Roles: sealed},
		Lifecycle: kickoff.Lifecycle{DeploymentTarget: "local", PostArchivePolicy: "bug_only"},
	}); err != nil || !ok {
		t.Fatalf("sellar el cambio x: %v (publicado: %v)", err, ok)
	}
	if design != "" {
		writeCLIFixture(t, filepath.Join(changeRoot, "design.md"), design)
	}

	te := &tasksEnv{adoEnv: e, changeRoot: changeRoot, folder: integrations.Folder{WorkspaceRoot: e.root, Root: changeRoot}}
	link := integrations.Link{Plugin: "azure-devops", Kind: "work_item", ID: "100", URL: e.srv.URL() + "/contoso/_apis/wit/workItems/100", Source: integrations.WorkItemLinkSource}
	if _, err := integrations.LinkWorkItem(te.folder, "azure-devops", link, time.Now().UTC()); err != nil {
		t.Fatalf("vincular el work item: %v", err)
	}

	e.srv.SeedWorkItem(tasksParentID, 3, map[string]any{
		"System.WorkItemType": "User Story", "System.Title": "Historia de origen",
		"System.AreaPath": `Fabrikam\Area`, "System.IterationPath": `Fabrikam\Sprint 1`,
	})
	e.srv.SeedStates("Task",
		adotest.State{Name: "To Do", Category: "Proposed"}, adotest.State{Name: "Active", Category: "InProgress"},
		adotest.State{Name: "Done", Category: "Completed"}, adotest.State{Name: "Removed", Category: "Removed"})
	e.srv.SeedTypeFields("Task", hoursOriginalField, hoursRemainingField, hoursCompletedField)

	if style == kickoff.ExecutionCheckpointed {
		var out bytes.Buffer
		if err := RunSDDGate([]string{"record", "--cwd", e.root, "--change", "x", "--gate", "design", "--decision", "approved", "--reason", "ok", "--actor", "Ana"}, &out); err != nil {
			t.Fatalf("aprobar la compuerta design: %v", err)
		}
	}
	return te
}

func (e *tasksEnv) workItemAPI(id int) string {
	return e.srv.URL() + "/contoso/_apis/wit/workItems/" + itoaCLI(id)
}

func itoaCLI(n int) string { b, _ := json.Marshal(n); return string(b) }

// seedTask siembra una Task hija (o de otro padre) con etiquetas y campos extra; devuelve nada: se
// lee con e.srv.WorkItem.
func (e *tasksEnv) seedTask(id int, tags string, parent int, extra map[string]any) {
	fields := map[string]any{"System.WorkItemType": "Task", "System.Title": "otra", "System.Tags": tags, "System.State": "To Do"}
	for k, v := range extra {
		fields[k] = v
	}
	var rels []map[string]any
	if parent != 0 {
		rels = append(rels, map[string]any{"rel": "System.LinkTypes.Hierarchy-Reverse", "url": e.workItemAPI(parent)})
	}
	e.srv.SeedWorkItem(id, 1, fields, rels...)
}

// bindLink escribe un vínculo de Task del rol sin pasar por el verbo.
func (e *tasksEnv) bindLink(t *testing.T, role, id string, estimate *float64, startedAt *time.Time) {
	t.Helper()
	link := integrations.Link{Plugin: "azure-devops", Kind: integrations.TaskLinkKind, ID: id, URL: e.workItemAPI(mustAtoi(t, id)), Role: role,
		Source: integrations.TasksBindSource, EstimateHours: estimate, StartedAt: startedAt}
	if _, err := integrations.BindTask(e.folder, "azure-devops", link, time.Now().UTC()); err != nil {
		t.Fatalf("BindTask: %v", err)
	}
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	var n int
	if err := json.Unmarshal([]byte(s), &n); err != nil {
		t.Fatalf("%q no es un número: %v", s, err)
	}
	return n
}

func (e *tasksEnv) tasksCLI(t *testing.T, args ...string) (int, tasksJSON, string, string) {
	t.Helper()
	full := append([]string{"tasks"}, args...)
	full = append(full, "--cwd", e.root, "--json")
	code, out, errOut := runIntegrationsCLI(t, "", full...)
	var got tasksJSON
	if strings.TrimSpace(out) != "" {
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("la salida --json no es JSON válido: %v\n%s", err, out)
		}
	}
	return code, got, out, errOut
}

func (e *tasksEnv) links(t *testing.T) []integrations.Link {
	t.Helper()
	links, err := integrations.TaskLinks(e.folder)
	if err != nil {
		t.Fatalf("TaskLinks: %v", err)
	}
	return links
}

func (e *tasksEnv) linksFile(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(e.changeRoot, integrations.LinksFile))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return data
}

func (e *tasksEnv) requests(method string) int {
	count := 0
	for _, r := range e.srv.Requests() {
		if r.Method == method {
			count++
		}
	}
	return count
}

func fieldNumber(t *testing.T, wi adotest.WorkItem, field string) float64 {
	t.Helper()
	switch v := wi.Fields[field].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case nil:
		t.Fatalf("el campo %s no está", field)
	}
	t.Fatalf("el campo %s no es numérico: %v", field, wi.Fields[field])
	return 0
}

func roleResult(t *testing.T, got tasksJSON, role string) tasksRoleJSON {
	t.Helper()
	for _, r := range got.Roles {
		if r.Role == role {
			return r
		}
	}
	t.Fatalf("la salida no trae el rol %s: %+v", role, got.Roles)
	return tasksRoleJSON{}
}

func notesOf(t *testing.T, e *tasksEnv) []integrations.TrackerNote {
	t.Helper()
	notes, err := integrations.TrackerNotes(e.folder)
	if err != nil {
		t.Fatalf("TrackerNotes: %v", err)
	}
	return notes
}

// ---------------------------------------------------------------- create

// prov:INT:2: crea la Task con la marca, las horas y el vínculo con estimate_hours y source tasks.create.
func TestIntegrationsTasksCreateCreatesTaskWithMarkHoursAndLink(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, tasksDesign)

	code, got, _, errOut := e.tasksCLI(t, "create", "--change", "x")

	if code != 0 || got.Status != trackerStatusDone || got.Schema != integrationsTasksSchema || got.Verb != "create" {
		t.Fatalf("código %d, salida %+v, stderr %q", code, got, errOut)
	}
	role := roleResult(t, got, "fullstack")
	if role.Result != tasksResultCreated || role.Task == nil || role.EstimateHours == nil || *role.EstimateHours != 6 || role.EstimateSource != estimateSourceDesign {
		t.Fatalf("resultado del rol: %+v", role)
	}
	wi, ok := e.srv.WorkItem(mustAtoi(t, role.Task.ID))
	if !ok {
		t.Fatalf("la Task %s no existe en el doble", role.Task.ID)
	}
	if tags, _ := wi.Fields["System.Tags"].(string); !strings.Contains(tags, "axiom-task:x:fullstack") {
		t.Errorf("la Task debe llevar la marca: %v", wi.Fields["System.Tags"])
	}
	if fieldNumber(t, wi, hoursOriginalField) != 6 || fieldNumber(t, wi, hoursRemainingField) != 6 {
		t.Errorf("horas de la Task: %v", wi.Fields)
	}
	links := e.links(t)
	if len(links) != 1 || links[0].Role != "fullstack" || links[0].Source != integrations.TasksCreateSource || links[0].EstimateHours == nil || *links[0].EstimateHours != 6 || links[0].ID != role.Task.ID {
		t.Fatalf("vínculo de Task: %+v", links)
	}
}

// `--estimate` prevalece sobre design.md.
func TestIntegrationsTasksCreateEstimateFlagOverridesDesign(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, tasksDesign)

	code, got, _, errOut := e.tasksCLI(t, "create", "--change", "x", "--estimate", "fullstack=4")

	role := roleResult(t, got, "fullstack")
	if code != 0 || role.EstimateSource != estimateSourceFlag || *role.EstimateHours != 4 {
		t.Fatalf("código %d, rol %+v, stderr %q", code, role, errOut)
	}
	if links := e.links(t); len(links) != 1 || *links[0].EstimateHours != 4 {
		t.Fatalf("el vínculo guarda 4 h: %+v", links)
	}
}

// Un vínculo ya registrado con la misma estimación no hace ninguna petición ni escritura.
func TestIntegrationsTasksCreateReusesLinkWithoutRequestsOrWrites(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, tasksDesign)
	e.seedTask(205, "", tasksParentID, nil)
	e.bindLink(t, "fullstack", "205", hoursPtr(6), nil)
	before := e.linksFile(t)

	code, got, _, errOut := e.tasksCLI(t, "create", "--change", "x")

	if code != 0 || roleResult(t, got, "fullstack").Result != tasksResultReusedLink {
		t.Fatalf("código %d, salida %+v, stderr %q", code, got, errOut)
	}
	if n := len(e.srv.Requests()); n != 0 {
		t.Errorf("no debe haber ninguna petición, hubo %d", n)
	}
	if !bytes.Equal(before, e.linksFile(t)) {
		t.Errorf("integrations.yaml no debe cambiar")
	}
}

// D17: con el vínculo y una estimación confirmada distinta, un único PATCH con test /rev propaga la
// estimación y solo después se actualiza el vínculo.
func TestIntegrationsTasksCreateEstimateDifferentPropagatesWithOnePatch(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, strings.Replace(tasksDesign, "6 h", "10 h", 1))
	e.srv.SeedWorkItem(205, 7, map[string]any{
		"System.WorkItemType": "Task", "System.State": "To Do",
		hoursOriginalField: 6.0, hoursRemainingField: 4.0, hoursCompletedField: 2.0,
	}, map[string]any{"rel": "System.LinkTypes.Hierarchy-Reverse", "url": e.workItemAPI(tasksParentID)})
	e.bindLink(t, "fullstack", "205", hoursPtr(6), nil)

	code, got, _, errOut := e.tasksCLI(t, "create", "--change", "x")

	role := roleResult(t, got, "fullstack")
	if code != 0 || role.Result != tasksResultEstimateUpdated || *role.EstimateHours != 10 {
		t.Fatalf("código %d, rol %+v, stderr %q", code, role, errOut)
	}
	if n := e.requests("PATCH"); n != 1 || e.srv.Created() != 0 {
		t.Fatalf("un único PATCH y ninguna creación; PATCH=%d creadas=%d", n, e.srv.Created())
	}
	wi, _ := e.srv.WorkItem(205)
	if fieldNumber(t, wi, hoursOriginalField) != 10 || fieldNumber(t, wi, hoursRemainingField) != 8 {
		t.Errorf("OriginalEstimate 10 y RemainingWork 8: %v", wi.Fields)
	}
	if links := e.links(t); *links[0].EstimateHours != 10 {
		t.Errorf("el vínculo queda con 10 h: %+v", links[0])
	}
}

// D17: un conflicto de rev rechaza el rol con rev_conflict, sin reaplicar ni tocar el vínculo.
func TestIntegrationsTasksCreateRevConflictKeepsLink(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, tasksDesign)
	e.srv.SeedWorkItem(205, 7, map[string]any{
		"System.WorkItemType": "Task", "System.State": "To Do",
		hoursOriginalField: 6.0, hoursRemainingField: 4.0, hoursCompletedField: 2.0,
	}, map[string]any{"rel": "System.LinkTypes.Hierarchy-Reverse", "url": e.workItemAPI(tasksParentID)})
	e.bindLink(t, "fullstack", "205", hoursPtr(6), nil)
	e.srv.EditAfterNextRead(205, map[string]any{"System.Title": "editada a mano"})

	code, got, _, _ := e.tasksCLI(t, "create", "--change", "x", "--estimate", "fullstack=10")

	role := roleResult(t, got, "fullstack")
	if code != 1 || role.Refusal == nil || role.Refusal.Code != "rev_conflict" || !strings.Contains(role.Refusal.Next, "axiom integrations tasks create --change x") {
		t.Fatalf("código %d, rol %+v", code, role)
	}
	if links := e.links(t); *links[0].EstimateHours != 6 {
		t.Errorf("el vínculo conserva 6 h: %+v", links[0])
	}
}

// Un fallo de creación deja la compuerta aprobada, nombra los dos caminos y, tras la respuesta
// perdida, el reintento encuentra la Task por la marca sin un segundo POST.
func TestIntegrationsTasksCreateLostResponseKeepsGateAndRetryFindsByMark(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionCheckpointed, tasksDesign)
	gatesBefore, err := os.ReadFile(filepath.Join(e.changeRoot, kickoff.GatesFileName))
	if err != nil {
		t.Fatal(err)
	}
	e.srv.LoseNextCreateResponse()

	code, got, out, _ := e.tasksCLI(t, "create", "--change", "x")

	role := roleResult(t, got, "fullstack")
	if code != 1 || role.Refusal == nil {
		t.Fatalf("código %d, salida %s", code, out)
	}
	for _, want := range []string{"axiom integrations tasks create --change x", "axiom integrations tasks bind --change x --role fullstack=<id>"} {
		if !strings.Contains(role.Refusal.Next, want) {
			t.Errorf("la continuación debe nombrar %q: %q", want, role.Refusal.Next)
		}
	}
	gatesAfter, _ := os.ReadFile(filepath.Join(e.changeRoot, kickoff.GatesFileName))
	if !bytes.Equal(gatesBefore, gatesAfter) {
		t.Errorf("gates.yaml debe quedar intacto")
	}

	code, got, _, errOut := e.tasksCLI(t, "create", "--change", "x")
	if code != 0 || roleResult(t, got, "fullstack").Result != tasksResultFoundByMark || e.srv.Created() != 1 {
		t.Fatalf("reintento: código %d, salida %+v, creadas %d, stderr %q", code, got, e.srv.Created(), errOut)
	}
}

// Dos marcas iguales: task_ambiguous nombrando `tasks bind`, sin crear ninguna Task.
func TestIntegrationsTasksCreateAmbiguousMarkNamesBind(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, tasksDesign)
	e.seedTask(201, "axiom-task:x:fullstack", tasksParentID, nil)
	e.seedTask(202, "axiom-task:x:fullstack", tasksParentID, nil)
	parent, _ := e.srv.WorkItem(tasksParentID)
	parent.Relations = append(parent.Relations,
		map[string]any{"rel": "System.LinkTypes.Hierarchy-Forward", "url": e.workItemAPI(201)},
		map[string]any{"rel": "System.LinkTypes.Hierarchy-Forward", "url": e.workItemAPI(202)})
	e.srv.SeedWorkItem(tasksParentID, 3, parent.Fields, parent.Relations...)

	code, got, _, _ := e.tasksCLI(t, "create", "--change", "x")

	role := roleResult(t, got, "fullstack")
	if code != 1 || role.Refusal == nil || role.Refusal.Code != "task_ambiguous" || !strings.Contains(role.Refusal.Next, "axiom integrations tasks bind") {
		t.Fatalf("código %d, rol %+v", code, role)
	}
	if e.srv.Created() != 0 || len(e.links(t)) != 0 {
		t.Errorf("no se crea ni se vincula nada")
	}
}

// Sin estimación para un rol: estimate_missing nombrando --estimate y ninguna creación.
func TestIntegrationsTasksCreateEstimateMissing(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "# Diseño\n\nsin sección\n")

	code, got, _, _ := e.tasksCLI(t, "create", "--change", "x")

	role := roleResult(t, got, "fullstack")
	if code != 1 || role.Refusal == nil || role.Refusal.Code != "estimate_missing" || !strings.Contains(role.Refusal.Next, "--estimate") {
		t.Fatalf("código %d, rol %+v", code, role)
	}
	if e.srv.Created() != 0 {
		t.Errorf("no se crea ninguna Task")
	}
}

// Una línea malformada de la sección de estimaciones se rechaza con estimate_invalid nombrando la línea.
func TestIntegrationsTasksCreateEstimateInvalidNamesTheLine(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "## Estimación por rol\n- fullstack: mucho\n")

	code, got, _, _ := e.tasksCLI(t, "create", "--change", "x")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "estimate_invalid" || !strings.Contains(got.Refusal.Text, "- fullstack: mucho") {
		t.Fatalf("código %d, salida %+v", code, got)
	}
	if len(e.srv.Requests()) != 0 {
		t.Errorf("ninguna petición")
	}
}

// Un fallo en un rol no impide los demás: el segundo queda vinculado y gates.yaml intacto.
func TestIntegrationsTasksCreatePartialWhenOnlyTheFirstRoleFails(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionCheckpointed, "## Estimación por rol\n- backend: 3 h\n- frontend: 5 h\n", "backend", "frontend")
	e.seedTask(201, "axiom-task:x:backend", tasksParentID, nil)
	e.seedTask(202, "axiom-task:x:backend", tasksParentID, nil)
	parent, _ := e.srv.WorkItem(tasksParentID)
	e.srv.SeedWorkItem(tasksParentID, 3, parent.Fields,
		map[string]any{"rel": "System.LinkTypes.Hierarchy-Forward", "url": e.workItemAPI(201)},
		map[string]any{"rel": "System.LinkTypes.Hierarchy-Forward", "url": e.workItemAPI(202)})
	gatesBefore, _ := os.ReadFile(filepath.Join(e.changeRoot, kickoff.GatesFileName))

	code, got, _, _ := e.tasksCLI(t, "create", "--change", "x")

	if code != 1 || got.Status != trackerStatusPartial {
		t.Fatalf("código %d, estado %s", code, got.Status)
	}
	if roleResult(t, got, "backend").Refusal == nil || roleResult(t, got, "frontend").Result != tasksResultCreated {
		t.Fatalf("roles: %+v", got.Roles)
	}
	if links := e.links(t); len(links) != 1 || links[0].Role != "frontend" {
		t.Errorf("solo el segundo rol queda vinculado: %+v", links)
	}
	if gatesAfter, _ := os.ReadFile(filepath.Join(e.changeRoot, kickoff.GatesFileName)); !bytes.Equal(gatesBefore, gatesAfter) {
		t.Errorf("gates.yaml intacto")
	}
}

// Si BindTask falla tras un POST con éxito, write_failed nombra el ID creado y el reintento
// `tasks create`, que encuentra la Task por la marca sin un segundo POST.
func TestIntegrationsTasksCreateBindFailureAfterPostNamesCreatedTask(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, tasksDesign)
	real := trackerBindTask
	trackerBindTask = func(integrations.Folder, string, integrations.Link, time.Time) (bool, error) {
		return false, errors.New("disco lleno")
	}
	t.Cleanup(func() { trackerBindTask = real })

	code, got, _, _ := e.tasksCLI(t, "create", "--change", "x")

	role := roleResult(t, got, "fullstack")
	if code != 1 || role.Refusal == nil || role.Refusal.Code != "write_failed" || role.Task == nil ||
		!strings.Contains(role.Refusal.Text, role.Task.ID) || !strings.Contains(role.Refusal.Next, "axiom integrations tasks create --change x") {
		t.Fatalf("código %d, rol %+v", code, role)
	}

	trackerBindTask = real
	code, got, _, errOut := e.tasksCLI(t, "create", "--change", "x")
	if code != 0 || roleResult(t, got, "fullstack").Result != tasksResultFoundByMark || e.srv.Created() != 1 {
		t.Fatalf("reintento: código %d, salida %+v, creadas %d, stderr %q", code, got, e.srv.Created(), errOut)
	}
}

// Una estimación inválida o un rol fuera del roster es error de uso (2) antes de cualquier petición.
func TestIntegrationsTasksCreateUsageErrorsMakeNoRequests(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, tasksDesign)
	for _, args := range [][]string{
		{"--estimate", "fullstack=0"}, {"--estimate", "fullstack=-1"}, {"--estimate", "fullstack=abc"},
		{"--estimate", "fullstack=1.255"}, {"--estimate", "fullstack=1001"},
		{"--estimate", "fullstack=2", "--estimate", "fullstack=3"}, {"--estimate", "qa=2"},
	} {
		code, _, out, errOut := e.tasksCLI(t, append([]string{"create", "--change", "x"}, args...)...)
		if code != 2 || !strings.Contains(errOut, "axiom integrations tasks create --help") {
			t.Errorf("%v: código %d, stdout %q, stderr %q", args, code, out, errOut)
		}
	}
	if n := len(e.srv.Requests()); n != 0 || e.srv.Created() != 0 || len(e.links(t)) != 0 {
		t.Errorf("ni peticiones ni vínculos: %d peticiones", n)
	}
}

// ---------------------------------------------------------------- precondiciones (D14)

func TestIntegrationsTasksNotApplicableWithoutActivePlugin(t *testing.T) {
	e := newTasksEnvWith(t, "", kickoff.ExecutionContinuous, tasksDesign)

	for _, verb := range [][]string{{"create"}, {"bind", "--role", "fullstack=205"}} {
		code, got, _, errOut := e.tasksCLI(t, append(verb, "--change", "x")...)
		if code != 0 || got.Status != trackerStatusNotApplicable || errOut != "" {
			t.Fatalf("%v: código %d, salida %+v, stderr %q", verb, code, got, errOut)
		}
	}
	if n := len(e.srv.Requests()); n != 0 {
		t.Errorf("sin red: %d peticiones", n)
	}
	if fileExistsCLI(filepath.Join(e.changeRoot, integrations.TrackerNotesFile)) {
		t.Errorf("no se crea el diario de avisos")
	}
}

func TestIntegrationsTasksODDLaneRejectedWithoutFiles(t *testing.T) {
	e := newADOEnv(t, adoBlock)
	writeCLIFixture(t, filepath.Join(e.root, "odd", "tasks", "f.md"), "# f\n")

	code, out, errOut := runIntegrationsCLI(t, "", "tasks", "create", "--change", "f", "--cwd", e.root, "--json")

	var got tasksJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
	if code != 1 || got.Refusal == nil || got.Refusal.Code != "odd_lane_not_supported" {
		t.Fatalf("código %d, salida %+v, stderr %q", code, got, errOut)
	}
	if n := len(e.srv.Requests()); n != 0 || fileExistsCLI(filepath.Join(e.root, "openspec", "changes", "f")) {
		t.Errorf("ninguna petición ni carpeta: %d peticiones", n)
	}
}

func TestIntegrationsTasksChangeNotFound(t *testing.T) {
	e := newADOEnv(t, adoBlock)

	code, out, _ := runIntegrationsCLI(t, "", "tasks", "create", "--change", "nada", "--cwd", e.root, "--json")

	var got tasksJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil || code != 1 || got.Refusal == nil || got.Refusal.Code != "change_not_found" {
		t.Fatalf("código %d, salida %s (%v)", code, out, err)
	}
}

func TestIntegrationsTasksWorkItemNotLinked(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, tasksDesign)
	if err := os.Remove(filepath.Join(e.changeRoot, integrations.LinksFile)); err != nil {
		t.Fatal(err)
	}

	code, got, _, _ := e.tasksCLI(t, "create", "--change", "x")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "work_item_not_linked" ||
		!strings.Contains(got.Refusal.Next, "axiom change start --from") || !strings.Contains(got.Refusal.Next, "axiom change start --adopt --change x") {
		t.Fatalf("código %d, salida %+v", code, got)
	}
	if len(e.srv.Requests()) != 0 {
		t.Errorf("ninguna petición")
	}
}

// En checkpointed hace falta la compuerta design aprobada con el digest vigente; una edición de
// design.md posterior la deja caducada.
func TestIntegrationsTasksCheckpointedRequiresApprovedGateWithCurrentDigest(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionCheckpointed, tasksDesign)
	writeCLIFixture(t, filepath.Join(e.changeRoot, "design.md"), strings.Replace(tasksDesign, "6 h", "7 h", 1))

	code, got, _, _ := e.tasksCLI(t, "create", "--change", "x")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "gate_not_approved" ||
		!strings.Contains(got.Refusal.Next, "axiom sdd gate show --change x") || !strings.Contains(got.Refusal.Text, "design") {
		t.Fatalf("código %d, salida %+v", code, got)
	}
	if len(e.srv.Requests()) != 0 || fileExistsCLI(filepath.Join(e.changeRoot, integrations.TrackerNotesFile)) {
		t.Errorf("ninguna petición y ningún diario en checkpointed")
	}
}

func TestIntegrationsTasksCheckpointedRejectedGateIsNotApproved(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionCheckpointed, tasksDesign)
	var out bytes.Buffer
	if err := RunSDDGate([]string{"record", "--cwd", e.root, "--change", "x", "--gate", "design", "--decision", "rejected", "--reason", "no", "--actor", "Ana"}, &out); err != nil {
		t.Fatal(err)
	}

	code, got, _, _ := e.tasksCLI(t, "create", "--change", "x")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "gate_not_approved" {
		t.Fatalf("código %d, salida %+v", code, got)
	}
}

func TestIntegrationsTasksContinuousWithoutDesignIsArtifactMissing(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")

	code, got, _, _ := e.tasksCLI(t, "create", "--change", "x")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "artifact_missing" || len(e.srv.Requests()) != 0 {
		t.Fatalf("código %d, salida %+v", code, got)
	}
}

// El diario .tracker-notes.yaml se escribe solo en continuous: rechazos, avisos y éxitos.
func TestIntegrationsTasksTrackerNotesOnlyInContinuous(t *testing.T) {
	cont := newTasksEnv(t, kickoff.ExecutionContinuous, "# Diseño\n")
	cont.tasksCLI(t, "create", "--change", "x")
	notes := notesOf(t, cont)
	if len(notes) != 1 || notes[0].Verb != "tasks create" || notes[0].Role != "fullstack" || notes[0].Code != "estimate_missing" || !strings.Contains(notes[0].Continuation, "--estimate") {
		t.Fatalf("el rechazo se anota en continuous: %+v", notes)
	}
	cont.tasksCLI(t, "create", "--change", "x", "--estimate", "fullstack=3")
	notes = notesOf(t, cont)
	if len(notes) != 2 || notes[1].Code != "done" || notes[1].Role != "fullstack" {
		t.Fatalf("el éxito se anota con code done: %+v", notes)
	}

	chk := newTasksEnv(t, kickoff.ExecutionCheckpointed, tasksDesign)
	chk.tasksCLI(t, "create", "--change", "x")
	if fileExistsCLI(filepath.Join(chk.changeRoot, integrations.TrackerNotesFile)) {
		t.Errorf("en checkpointed no se escribe el diario")
	}
}

// ---------------------------------------------------------------- bind

func TestIntegrationsTasksBindWritesLinkWithSourceAndEstimate(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")
	e.seedTask(205, "", tasksParentID, nil)

	code, got, _, errOut := e.tasksCLI(t, "bind", "--change", "x", "--role", "fullstack=205", "--estimate", "fullstack=5")

	role := roleResult(t, got, "fullstack")
	if code != 0 || role.Result != tasksResultBound || *role.EstimateHours != 5 || role.EstimateSource != estimateSourceFlag {
		t.Fatalf("código %d, rol %+v, stderr %q", code, role, errOut)
	}
	links := e.links(t)
	if len(links) != 1 || links[0].ID != "205" || links[0].Source != integrations.TasksBindSource || *links[0].EstimateHours != 5 {
		t.Fatalf("vínculo: %+v", links)
	}
	wi, _ := e.srv.WorkItem(205)
	if fieldNumber(t, wi, hoursOriginalField) != 5 || fieldNumber(t, wi, hoursRemainingField) != 5 {
		t.Errorf("una Task sin horas recibe OriginalEstimate y RemainingWork 5: %v", wi.Fields)
	}
}

func TestIntegrationsTasksBindDoesNotTouchHoursOfATaskWithEstimate(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")
	e.seedTask(205, "", tasksParentID, map[string]any{hoursOriginalField: 8.0})

	code, got, _, _ := e.tasksCLI(t, "bind", "--change", "x", "--role", "fullstack=205", "--estimate", "fullstack=5")

	if code != 0 || e.requests("PATCH") != 0 {
		t.Fatalf("código %d, PATCH=%d", code, e.requests("PATCH"))
	}
	if links := e.links(t); *links[0].EstimateHours != 5 || *roleResult(t, got, "fullstack").EstimateHours != 5 {
		t.Errorf("el vínculo guarda 5 h: %+v", links[0])
	}
}

// Sin ninguna estimación se vincula igualmente, con el aviso estimate_not_recorded.
func TestIntegrationsTasksBindWithoutEstimateWarns(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")
	e.seedTask(205, "", tasksParentID, nil)

	code, got, out, _ := e.tasksCLI(t, "bind", "--change", "x", "--role", "fullstack=205")

	role := roleResult(t, got, "fullstack")
	if code != 0 || role.EstimateHours != nil || role.EstimateSource != estimateSourceNone {
		t.Fatalf("código %d, rol %+v", code, role)
	}
	if len(got.Warnings) != 1 || got.Warnings[0].Code != "estimate_not_recorded" || !strings.Contains(got.Warnings[0].Next, "axiom integrations tasks create --change x --estimate fullstack=") {
		t.Fatalf("avisos: %+v", got.Warnings)
	}
	if !strings.Contains(out, `"estimate_hours": null`) {
		t.Errorf("el JSON lleva estimate_hours null: %s", out)
	}
	if links := e.links(t); len(links) != 1 || links[0].EstimateHours != nil {
		t.Errorf("el vínculo se escribe sin estimate_hours: %+v", links)
	}
	if notes := notesOf(t, e); len(notes) < 2 || notes[1].Code != "estimate_not_recorded" || notes[1].Role != "fullstack" {
		t.Errorf("el aviso se anota en continuous: %+v", notes)
	}
}

// bind no exige compuerta ni artefacto en ningún estilo.
func TestIntegrationsTasksBindNeedsNoGateInCheckpointed(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionCheckpointed, "")
	e.seedTask(205, "", tasksParentID, nil)
	var out bytes.Buffer
	if err := RunSDDGate([]string{"record", "--cwd", e.root, "--change", "x", "--gate", "design", "--decision", "rejected", "--reason", "no", "--actor", "Ana"}, &out); err != nil {
		t.Fatal(err)
	}

	code, got, _, errOut := e.tasksCLI(t, "bind", "--change", "x", "--role", "fullstack=205")

	if code != 0 || roleResult(t, got, "fullstack").Result != tasksResultBound {
		t.Fatalf("código %d, salida %+v, stderr %q", code, got, errOut)
	}
}

func TestIntegrationsTasksBindRejectionsWriteNothing(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")
	e.seedTask(206, "", 555, nil) // otro padre
	e.srv.SeedWorkItem(300, 1, map[string]any{"System.WorkItemType": "Bug"})

	before := e.linksFile(t)
	cases := []struct{ role, code string }{
		{"fullstack=999", "task_not_found"},
		{"fullstack=300", "not_a_task"},
		{"fullstack=206", "not_child_of_work_item"},
	}
	for _, tc := range cases {
		code, got, _, _ := e.tasksCLI(t, "bind", "--change", "x", "--role", tc.role)
		role := roleResult(t, got, "fullstack")
		if code != 1 || role.Refusal == nil || role.Refusal.Code != tc.code || !strings.Contains(role.Refusal.Next, "axiom integrations tasks bind --change x --role "+tc.role) {
			t.Errorf("%s: código %d, rol %+v", tc.role, code, role)
		}
	}
	if !bytes.Equal(before, e.linksFile(t)) {
		t.Errorf("integrations.yaml no cambia")
	}
	if len(e.links(t)) != 0 || e.requests("PATCH") != 0 {
		t.Errorf("ni vínculo ni escritura en el tracker")
	}
}

func TestIntegrationsTasksBindRoleNotInRosterMakesNoRequest(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")

	code, got, _, _ := e.tasksCLI(t, "bind", "--change", "x", "--role", "backend=205")

	if code != 1 || got.Refusal == nil || got.Refusal.Code != "role_not_in_roster" || !strings.Contains(got.Refusal.Next, "fullstack") || len(e.srv.Requests()) != 0 {
		t.Fatalf("código %d, salida %+v, peticiones %d", code, got, len(e.srv.Requests()))
	}
}

func TestIntegrationsTasksBindUsageErrors(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")
	for _, args := range [][]string{
		{},                          // sin --role
		{"--role", "fullstack"},     // sin id
		{"--role", "fullstack=2 3"}, // id con espacio
		{"--role", "fullstack=205", "--estimate", "backend=2"}, // estimación de otro rol
		{"--role", "fullstack=205", "--estimate", "fullstack=0"},
	} {
		code, _, _, errOut := e.tasksCLI(t, append([]string{"bind", "--change", "x"}, args...)...)
		if code != 2 || !strings.Contains(errOut, "axiom integrations tasks bind --help") {
			t.Errorf("%v: código %d, stderr %q", args, code, errOut)
		}
	}
	if len(e.srv.Requests()) != 0 {
		t.Errorf("ninguna petición")
	}
}

// D18: otra Task sobre un rol ya vinculado se rechaza sin --replace nombrando el comando con --replace.
func TestIntegrationsTasksBindAlreadyBoundNamesReplace(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")
	e.seedTask(205, "", tasksParentID, nil)
	e.seedTask(206, "", tasksParentID, nil)
	e.bindLink(t, "fullstack", "205", nil, nil)
	before := e.linksFile(t)

	code, got, _, _ := e.tasksCLI(t, "bind", "--change", "x", "--role", "fullstack=206")

	role := roleResult(t, got, "fullstack")
	if code != 1 || role.Refusal == nil || role.Refusal.Code != "task_already_bound" ||
		!strings.Contains(role.Refusal.Next, "axiom integrations tasks bind --change x --role fullstack=206 --replace") {
		t.Fatalf("código %d, rol %+v", code, role)
	}
	if !bytes.Equal(before, e.linksFile(t)) || len(e.srv.Requests()) != 0 {
		t.Errorf("no cambia nada y no hay peticiones")
	}
}

// D18: --replace conserva started_at e imputaciones, y con la misma Task no cambia el fichero.
func TestIntegrationsTasksBindReplaceKeepsStartedAtAndTimeEntries(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")
	e.seedTask(205, "", tasksParentID, nil)
	e.seedTask(206, "", tasksParentID, nil)
	started := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	e.bindLink(t, "fullstack", "205", hoursPtr(6), &started)
	entry := integrations.TimeEntry{Plugin: "azure-devops", Role: "fullstack", ItemID: "205", Minutes: 120, Date: "2026-11-24",
		Source: "role.close/fullstack/1", Actor: "Ana <ana@example.test>"}
	if _, err := integrations.RecordTimeEntry(e.folder, "azure-devops", entry, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	code, got, _, errOut := e.tasksCLI(t, "bind", "--change", "x", "--role", "fullstack=206", "--replace")

	role := roleResult(t, got, "fullstack")
	if code != 0 || role.Result != tasksResultReplaced {
		t.Fatalf("código %d, rol %+v, stderr %q", code, role, errOut)
	}
	links := e.links(t)
	if len(links) != 1 || links[0].ID != "206" || links[0].Source != integrations.TasksBindSource || links[0].StartedAt == nil || !links[0].StartedAt.Equal(started) {
		t.Fatalf("vínculo sustituido: %+v", links)
	}
	if links[0].EstimateHours == nil || *links[0].EstimateHours != 6 {
		t.Errorf("sin --estimate ni design.md se conserva la estimación anterior: %+v", links[0])
	}
	entries, err := integrations.TimeEntries(e.folder)
	if err != nil || len(entries) != 1 || entries[0].ItemID != "205" || entries[0].Source != "role.close/fullstack/1" {
		t.Fatalf("la imputación conserva su item_id: %+v (%v)", entries, err)
	}

	before := e.linksFile(t)
	code, got, _, _ = e.tasksCLI(t, "bind", "--change", "x", "--role", "fullstack=206", "--replace")
	if code != 0 || roleResult(t, got, "fullstack").Result != tasksResultUnchanged || !bytes.Equal(before, e.linksFile(t)) {
		t.Errorf("--replace con la misma Task no cambia el fichero: código %d, %+v", code, got)
	}
}

func TestIntegrationsTasksBindReplaceWithInvalidTaskKeepsLink(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")
	e.seedTask(205, "", tasksParentID, nil)
	e.seedTask(206, "", 555, nil)
	e.bindLink(t, "fullstack", "205", nil, nil)
	before := e.linksFile(t)

	code, got, _, _ := e.tasksCLI(t, "bind", "--change", "x", "--role", "fullstack=206", "--replace")

	if role := roleResult(t, got, "fullstack"); code != 1 || role.Refusal == nil || role.Refusal.Code != "not_child_of_work_item" {
		t.Fatalf("código %d, rol %+v", code, role)
	}
	if !bytes.Equal(before, e.linksFile(t)) {
		t.Errorf("el vínculo existente no cambia")
	}
}

// `--replace` sin vínculo previo equivale a un bind normal.
func TestIntegrationsTasksBindReplaceWithoutPreviousLinkIsBind(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, "")
	e.seedTask(205, "", tasksParentID, nil)

	code, got, _, _ := e.tasksCLI(t, "bind", "--change", "x", "--role", "fullstack=205", "--replace")

	if role := roleResult(t, got, "fullstack"); code != 0 || role.Result != tasksResultBound {
		t.Fatalf("código %d, salida %+v", code, got)
	}
}

// ---------------------------------------------------------------- ayuda y secretos

func TestIntegrationsHelpListsTasksFamilyAndVerbs(t *testing.T) {
	code, out, _ := runIntegrationsCLI(t, "", "--help")
	if code != 0 || !strings.Contains(out, "tasks") {
		t.Fatalf("la ayuda de la familia lista tasks: código %d\n%s", code, out)
	}
	code, out, _ = runIntegrationsCLI(t, "", "inspect")
	if code == 0 || !strings.Contains(out, "tasks") {
		t.Fatalf("un subcomando desconocido muestra la ayuda con tasks: código %d\n%s", code, out)
	}
	code, out, _ = runIntegrationsCLI(t, "", "tasks", "--help")
	if code != 0 || !strings.Contains(out, "create") || !strings.Contains(out, "bind") || !strings.Contains(out, "--json") {
		t.Fatalf("ayuda de tasks: código %d\n%s", code, out)
	}
	for verb, flags := range map[string][]string{
		"create": {"--change", "--estimate", "--cwd", "--json"},
		"bind":   {"--change", "--role", "--estimate", "--replace", "--cwd", "--json"},
	} {
		code, out, _ = runIntegrationsCLI(t, "", "tasks", verb, "--help")
		if code != 0 {
			t.Fatalf("tasks %s --help: código %d", verb, code)
		}
		for _, flag := range flags {
			if !strings.Contains(out, flag) {
				t.Errorf("la ayuda de tasks %s debe describir %s", verb, flag)
			}
		}
	}
	if code, _, _ = runIntegrationsCLI(t, "", "tasks", "inspect"); code != 2 {
		t.Errorf("un verbo desconocido de tasks es error de uso: %d", code)
	}
}

// El PAT nunca aparece en la salida ni en stderr aunque el tracker falle la autenticación.
func TestIntegrationsTasksNeverLeakThePATOnAuthFailure(t *testing.T) {
	e := newTasksEnv(t, kickoff.ExecutionContinuous, tasksDesign)
	e.srv.Reply("POST", "/contoso/Fabrikam/_apis/wit/workitems/$Task", adotest.Response{Status: 401, Body: "Unauthorized " + adoPAT})

	for _, asJSON := range []bool{false, true} {
		args := []string{"tasks", "create", "--change", "x", "--cwd", e.root}
		if asJSON {
			args = append(args, "--json")
		}
		code, out, errOut := runIntegrationsCLI(t, "", args...)
		if code != 1 || strings.Contains(out, adoPAT) || strings.Contains(errOut, adoPAT) {
			t.Fatalf("json=%v: código %d, el PAT no debe filtrarse\n%s\n%s", asJSON, code, out, errOut)
		}
	}
	if strings.Contains(string(e.linksFile(t)), adoPAT) {
		t.Errorf("integrations.yaml no contiene el PAT")
	}
}
