package azuredevops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
)

const (
	itemTypeTask    = "Task"
	relChild        = "System.LinkTypes.Hierarchy-Forward"
	categoryRemoved = "Removed"
)

// neutralError traduce un fallo del adaptador a un error neutral del puerto de Tasks sin perder el
// texto ni la causa: errors.Is reconoce el centinela neutral y la causa sigue en Unwrap.
type neutralError struct{ sentinel, cause error }

func (e *neutralError) Error() string        { return e.cause.Error() }
func (e *neutralError) Unwrap() error        { return e.cause }
func (e *neutralError) Is(target error) bool { return target == e.sentinel }

// mapWriteError clasifica el fallo de una petición del puerto de Tasks. Un 404 es task_not_found; un
// 4xx que no es conflicto de revisión (validación, reglas, autorización) es tracker_rejected: consta
// que la petición no se aplicó. Cualquier otro fallo (red, tiempo agotado, 5xx, límite de tasa) se
// devuelve tal cual: su resultado es incierto y no se confunde con un rechazo.
func mapWriteError(err error) error {
	var notFound *NotFoundError
	var api *APIError
	var auth *AuthError
	switch {
	case errors.As(err, &notFound):
		return &neutralError{sentinel: integrations.ErrTaskNotFound, cause: err}
	case errors.As(err, &api) && api.Status >= 400 && api.Status < 500:
		return &neutralError{sentinel: integrations.ErrTrackerRejected, cause: err}
	case errors.As(err, &auth):
		return &neutralError{sentinel: integrations.ErrTrackerRejected, cause: err}
	}
	return err
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// numberField lee un campo numérico de un work item. ok es false si falta o no es un número.
func numberField(fields map[string]any, name string) (float64, bool) {
	switch n := fields[name].(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

// hoursOf lee las horas de una Task: un campo ausente se lee como 0 y HasOriginal distingue la
// estimación original ausente de una estimación de 0.
func hoursOf(fields map[string]any) integrations.TaskHours {
	var h integrations.TaskHours
	h.Original, h.HasOriginal = numberField(fields, fieldOriginalEstimate)
	h.Remaining, _ = numberField(fields, fieldRemainingWork)
	h.Completed, _ = numberField(fields, fieldCompletedWork)
	return h
}

// assignedID devuelve el id del asignado. Nunca lee uniqueName ni el nombre visible: un valor que no
// es una identidad con id se ignora.
func assignedID(fields map[string]any) string {
	m, ok := fields[fieldAssignedTo].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := m["id"].(string)
	return id
}

// parentOf devuelve el identificador del padre del work item (relación Hierarchy-Reverse del host y la
// organización configurados), o "" si no tiene.
func (c *Client) parentOf(wi WorkItem) string {
	for _, rel := range wi.Relations {
		if rel.Rel != relParent {
			continue
		}
		if id, ok := c.relatedItemID(rel.URL); ok {
			return id
		}
	}
	return ""
}

// childIDs devuelve los identificadores de las relaciones Hierarchy-Forward del work item, sin
// repetidos y en el orden de las relaciones.
func (c *Client) childIDs(wi WorkItem) []int {
	var ids []int
	seen := map[int]bool{}
	for _, rel := range wi.Relations {
		if rel.Rel != relChild {
			continue
		}
		raw, ok := c.relatedItemID(rel.URL)
		if !ok {
			continue
		}
		if id, err := strconv.Atoi(raw); err == nil && id > 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

// taskCategories devuelve la categoría de cada estado del tipo Task (clave en minúsculas).
func (c *Client) taskCategories(ctx context.Context) (map[string]string, error) {
	states, err := c.TypeStates(ctx, itemTypeTask)
	if err != nil {
		return nil, err
	}
	cats := make(map[string]string, len(states))
	for _, s := range states {
		cats[strings.ToLower(s.Name)] = s.Category
	}
	return cats, nil
}

// softCategories es taskCategories sin propagar el fallo: la categoría de una lectura suelta es
// informativa, y su ausencia no debe impedir leer la Task.
func (c *Client) softCategories(ctx context.Context) map[string]string {
	cats, err := c.taskCategories(ctx)
	if err != nil {
		return map[string]string{}
	}
	return cats
}

// taskItem convierte un work item en la lectura neutral de una Task.
func (c *Client) taskItem(wi WorkItem, cats map[string]string) integrations.TaskItem {
	state := fieldString(wi.Fields, fieldState)
	return integrations.TaskItem{
		ID:            strconv.Itoa(wi.ID),
		URL:           c.workItemURL(wi.ID),
		Type:          fieldString(wi.Fields, fieldType),
		State:         state,
		StateCategory: cats[strings.ToLower(state)],
		ParentID:      c.parentOf(wi),
		AssignedToID:  assignedID(wi.Fields),
		Rev:           wi.Rev,
		Tags:          splitTags(fieldString(wi.Fields, fieldTags)),
		Hours:         hoursOf(wi.Fields),
	}
}

func hasMark(wi WorkItem, mark string) bool {
	for _, tag := range splitTags(fieldString(wi.Fields, fieldTags)) {
		if strings.EqualFold(tag, mark) {
			return true
		}
	}
	return false
}

// isLiveTask indica si wi es una Task cuyo estado no está en la categoría Removed.
func isLiveTask(wi WorkItem, cats map[string]string) bool {
	if !strings.EqualFold(fieldString(wi.Fields, fieldType), itemTypeTask) {
		return false
	}
	return cats[strings.ToLower(fieldString(wi.Fields, fieldState))] != categoryRemoved
}

func parseTaskID(raw string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || id <= 0 {
		return 0, &neutralError{sentinel: integrations.ErrTaskNotFound,
			cause: fmt.Errorf("el identificador de Task %q no es un número positivo", raw)}
	}
	return id, nil
}

// childTasks devuelve las Tasks hijas vivas de parent: lee las relaciones Hierarchy-Forward y los
// hijos en lotes de 200, y descarta lo que no es Task o está en la categoría Removed.
func (c *Client) childTasks(ctx context.Context, parent WorkItem) ([]WorkItem, map[string]string, error) {
	ids := c.childIDs(parent)
	if len(ids) == 0 {
		return nil, nil, nil
	}
	items, err := c.GetWorkItems(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	cats, err := c.taskCategories(ctx)
	if err != nil {
		return nil, nil, err
	}
	var tasks []WorkItem
	for _, wi := range items {
		if isLiveTask(wi, cats) {
			tasks = append(tasks, wi)
		}
	}
	return tasks, cats, nil
}

// ChildTasks implementa integrations.TaskTracker: lista las Tasks hijas vivas del work item.
func (p *Plugin) ChildTasks(ctx context.Context, req integrations.ChildTasksRequest) ([]integrations.TaskItem, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return nil, err
	}
	parentID, err := strconv.Atoi(strings.TrimSpace(req.ParentID))
	if err != nil || parentID <= 0 {
		return nil, invalidWorkItemID(parentID)
	}
	parent, err := client.GetWorkItem(ctx, parentID)
	if err != nil {
		return nil, err
	}
	tasks, cats, err := client.childTasks(ctx, parent)
	if err != nil {
		return nil, err
	}
	out := make([]integrations.TaskItem, 0, len(tasks))
	for _, wi := range tasks {
		out = append(out, client.taskItem(wi, cats))
	}
	return out, nil
}

// apiItemURL es la URL API de un work item, la que se escribe en una relación: {endpoint}/{org}/_apis/
// wit/workItems/{id}. Sale de la configuración, nunca del contenido del tracker.
func (c *Client) apiItemURL(id int) string {
	return strings.TrimRight(c.endpoint.String(), "/") + "/" + url.PathEscape(c.target.Organization) +
		"/_apis/wit/workItems/" + strconv.Itoa(id)
}

func parentRelationOp(parentURL string) PatchOp {
	return PatchOp{Op: "add", Path: "/relations/-", Value: map[string]any{"rel": relParent, "url": parentURL}}
}

// requireFields comprueba que el proceso define los campos pedidos para el tipo Task. Si falta alguno
// devuelve *integrations.FieldMissingError sin haber escrito nada.
func (c *Client) requireFields(ctx context.Context, refs ...string) error {
	defined, err := c.TypeFields(ctx, itemTypeTask)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		if !defined[strings.ToLower(ref)] {
			return &integrations.FieldMissingError{Field: ref}
		}
	}
	return nil
}

func wiqlQuote(s string) string { return strings.ReplaceAll(s, "'", "''") }

func idStrings(items []WorkItem) []string {
	ids := make([]string, 0, len(items))
	for _, wi := range items {
		ids = append(ids, strconv.Itoa(wi.ID))
	}
	sort.Strings(ids)
	return ids
}

// EnsureChildTask implementa integrations.TaskTracker (D11): busca la Task del rol por su marca,
// primero entre las hijas del work item y después con WIQL, y solo crea con cero coincidencias.
func (p *Plugin) EnsureChildTask(ctx context.Context, req integrations.CreateTaskRequest) (integrations.CreateTaskResult, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.CreateTaskResult{}, err
	}
	parentID, err := strconv.Atoi(strings.TrimSpace(req.ParentID))
	if err != nil || parentID <= 0 {
		return integrations.CreateTaskResult{}, invalidWorkItemID(parentID)
	}
	return client.ensureChildTask(ctx, parentID, req)
}

func (c *Client) ensureChildTask(ctx context.Context, parentID int, req integrations.CreateTaskRequest) (integrations.CreateTaskResult, error) {
	mark := strings.ToLower(strings.TrimSpace(req.Mark))
	if mark == "" {
		return integrations.CreateTaskResult{}, fmt.Errorf("%w: falta la marca de la Task", ErrBadRequest)
	}
	parent, err := c.GetWorkItem(ctx, parentID)
	if err != nil {
		return integrations.CreateTaskResult{}, err
	}
	children, cats, err := c.childTasks(ctx, parent)
	if err != nil {
		return integrations.CreateTaskResult{}, err
	}
	var marked []WorkItem
	for _, wi := range children {
		if hasMark(wi, mark) {
			marked = append(marked, wi)
		}
	}
	switch {
	case len(marked) == 1:
		return c.foundResult(marked[0], cats, parentID, integrations.TaskFoundByMark), nil
	case len(marked) > 1:
		return integrations.CreateTaskResult{}, &integrations.TaskAmbiguousError{Mark: mark, IDs: idStrings(marked)}
	}

	// Sin hijas con la marca: una Task huérfana de un camino alternativo anterior solo aparece por WIQL.
	// Si la consulta falla no se crea nada: crearía un duplicado de una Task que quizá existe.
	ids, err := c.QueryWIQL(ctx, fmt.Sprintf(
		"SELECT [System.Id] FROM WorkItems WHERE [System.WorkItemType] = 'Task' AND [System.Tags] CONTAINS '%s'", wiqlQuote(mark)))
	if err != nil {
		return integrations.CreateTaskResult{}, err
	}
	var candidates []WorkItem
	if len(ids) > 0 {
		found, err := c.GetWorkItems(ctx, ids)
		if err != nil {
			return integrations.CreateTaskResult{}, err
		}
		if cats, err = c.taskCategories(ctx); err != nil {
			return integrations.CreateTaskResult{}, err
		}
		for _, wi := range found {
			if isLiveTask(wi, cats) && hasMark(wi, mark) {
				candidates = append(candidates, wi)
			}
		}
	}
	switch {
	case len(candidates) > 1:
		return integrations.CreateTaskResult{}, &integrations.TaskAmbiguousError{Mark: mark, IDs: idStrings(candidates)}
	case len(candidates) == 1:
		return c.adoptCandidate(ctx, candidates[0], cats, parentID, mark)
	}
	return c.createTask(ctx, parent, req, mark)
}

func (c *Client) foundResult(wi WorkItem, cats map[string]string, parentID int, outcome integrations.TaskOutcome) integrations.CreateTaskResult {
	item := c.taskItem(wi, cats)
	item.ParentID = strconv.Itoa(parentID)
	return integrations.CreateTaskResult{Task: item, Outcome: outcome}
}

// adoptCandidate decide sobre la única Task con la marca que WIQL halló fuera de las hijas: huérfana
// se repara relacionándola con el padre (con test /rev); bajo otro padre es ambigua.
func (c *Client) adoptCandidate(ctx context.Context, wi WorkItem, cats map[string]string, parentID int, mark string) (integrations.CreateTaskResult, error) {
	switch owner := c.parentOf(wi); owner {
	case "":
		updated, err := c.UpdateWorkItem(ctx, wi.ID, wi.Rev, []PatchOp{parentRelationOp(c.apiItemURL(parentID))})
		if err != nil {
			return integrations.CreateTaskResult{}, mapWriteError(err)
		}
		return c.foundResult(updated, cats, parentID, integrations.TaskRepaired), nil
	case strconv.Itoa(parentID):
		return c.foundResult(wi, cats, parentID, integrations.TaskFoundByMark), nil
	}
	return integrations.CreateTaskResult{}, &integrations.TaskAmbiguousError{Mark: mark, IDs: []string{strconv.Itoa(wi.ID)}}
}

// createTask crea la Task con un único POST. Antes de escribir una estimación comprueba que el proceso
// define los campos de horas. Un 400 al crear con la relación activa el camino alternativo: la Task no
// se creó, se repite sin relaciones y se relaciona con un PATCH con test /rev.
func (c *Client) createTask(ctx context.Context, parent WorkItem, req integrations.CreateTaskRequest, mark string) (integrations.CreateTaskResult, error) {
	if req.EstimateHours > 0 {
		if err := c.requireFields(ctx, fieldOriginalEstimate, fieldRemainingWork); err != nil {
			return integrations.CreateTaskResult{}, err
		}
	}
	ops := []PatchOp{
		{Op: "add", Path: "/fields/" + fieldTitle, Value: req.Title},
		{Op: "add", Path: "/fields/" + fieldTags, Value: mark},
	}
	for _, field := range []string{fieldAreaPath, fieldIterationPath} {
		if v := fieldString(parent.Fields, field); v != "" {
			ops = append(ops, PatchOp{Op: "add", Path: "/fields/" + field, Value: v})
		}
	}
	if h := round2(req.EstimateHours); h > 0 {
		ops = append(ops,
			PatchOp{Op: "add", Path: "/fields/" + fieldOriginalEstimate, Value: h},
			PatchOp{Op: "add", Path: "/fields/" + fieldRemainingWork, Value: h})
	}
	relOp := parentRelationOp(c.apiItemURL(parent.ID))

	created, err := c.CreateWorkItem(ctx, itemTypeTask, append(append([]PatchOp(nil), ops...), relOp))
	if err != nil {
		if !errors.Is(err, ErrBadRequest) {
			return integrations.CreateTaskResult{}, mapWriteError(err)
		}
		if created, err = c.CreateWorkItem(ctx, itemTypeTask, ops); err != nil {
			return integrations.CreateTaskResult{}, mapWriteError(err)
		}
		linked, linkErr := c.UpdateWorkItem(ctx, created.ID, max(created.Rev, 1), []PatchOp{relOp})
		if linkErr != nil {
			return integrations.CreateTaskResult{}, &neutralError{sentinel: integrations.ErrTaskParentLinkFailed,
				cause: fmt.Errorf("la Task %d se creó pero no se pudo relacionar con el work item %d: %w", created.ID, parent.ID, linkErr)}
		}
		created = linked
	}

	cats := c.softCategories(ctx)
	result := c.foundResult(created, cats, parent.ID, integrations.TaskCreated)
	result.Warnings = c.duplicateWarnings(ctx, parent.ID, mark)
	return result, nil
}

// duplicateWarnings relee las hijas tras crear: si más de una lleva la marca (dos máquinas crearon a la
// vez, sin exclusión posible en Azure DevOps) avisa con task_duplicated. Un fallo de la relectura no
// convierte en error una creación que tuvo éxito.
func (c *Client) duplicateWarnings(ctx context.Context, parentID int, mark string) []integrations.Warning {
	parent, err := c.GetWorkItem(ctx, parentID)
	if err != nil {
		return nil
	}
	children, _, err := c.childTasks(ctx, parent)
	if err != nil {
		return nil
	}
	var marked []WorkItem
	for _, wi := range children {
		if hasMark(wi, mark) {
			marked = append(marked, wi)
		}
	}
	if len(marked) < 2 {
		return nil
	}
	return []integrations.Warning{{
		Code:         "task_duplicated",
		Message:      fmt.Sprintf("varias Tasks llevan la marca %s (%s)", mark, strings.Join(idStrings(marked), ", ")),
		Continuation: "axiom integrations tasks bind",
	}}
}

// ReadTask implementa integrations.TaskTracker: lee una Task por su identificador.
func (p *Plugin) ReadTask(ctx context.Context, req integrations.TaskRequest) (integrations.TaskItem, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.TaskItem{}, err
	}
	return client.readTask(ctx, req.TaskID)
}

func (c *Client) readTask(ctx context.Context, raw string) (integrations.TaskItem, error) {
	id, err := parseTaskID(raw)
	if err != nil {
		return integrations.TaskItem{}, err
	}
	wi, err := c.GetWorkItem(ctx, id)
	if err != nil {
		return integrations.TaskItem{}, mapReadError(err)
	}
	return c.taskItem(wi, c.softCategories(ctx)), nil
}

// mapReadError traduce solo el 404 de una lectura: el resto de fallos de lectura se devuelven tal cual.
func mapReadError(err error) error {
	var notFound *NotFoundError
	if errors.As(err, &notFound) {
		return &neutralError{sentinel: integrations.ErrTaskNotFound, cause: err}
	}
	return err
}

// ValidateTask implementa integrations.TaskTracker: comprueba, en este orden, que la Task existe, que
// es de tipo Task y que es hija de req.ParentID.
func (p *Plugin) ValidateTask(ctx context.Context, req integrations.TaskRequest) (integrations.TaskItem, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.TaskItem{}, err
	}
	id, err := parseTaskID(req.TaskID)
	if err != nil {
		return integrations.TaskItem{}, err
	}
	wi, err := client.GetWorkItem(ctx, id)
	if err != nil {
		return integrations.TaskItem{}, mapReadError(err)
	}
	if typ := fieldString(wi.Fields, fieldType); !strings.EqualFold(typ, itemTypeTask) {
		return integrations.TaskItem{}, &neutralError{sentinel: integrations.ErrNotATask,
			cause: fmt.Errorf("el work item %d es de tipo %q, no una Task", id, typ)}
	}
	item := client.taskItem(wi, client.softCategories(ctx))
	if item.ParentID == "" || item.ParentID != strings.TrimSpace(req.ParentID) {
		return integrations.TaskItem{}, &neutralError{sentinel: integrations.ErrNotChildOfWorkItem,
			cause: fmt.Errorf("la Task %d no es hija del work item %s", id, req.ParentID)}
	}
	return item, nil
}

// patchHours aplica un PATCH de horas con test /rev (D10). Un conflicto tras un reintento de red se
// da por aplicado (recovered) solo si la relectura está en rev+1 y verify confirma los valores
// esperados; cualquier otro conflicto devuelve *integrations.RevConflictError con los valores releídos
// y sin reaplicar. Un fallo de red sin respuesta, un 5xx o un límite de tasa se devuelven tal cual:
// son resultados inciertos y no un rechazo.
func (c *Client) patchHours(ctx context.Context, id, rev int, ops []PatchOp, verify func(WorkItem) bool) (WorkItem, bool, error) {
	item, err := c.UpdateWorkItem(ctx, id, rev, ops)
	if err == nil {
		return item, false, nil
	}
	var conflict *RevConflictError
	if !errors.As(err, &conflict) {
		return WorkItem{}, false, mapWriteError(err)
	}
	current, readErr := c.GetWorkItem(ctx, id)
	if readErr != nil {
		return WorkItem{}, false, readErr
	}
	if conflict.AfterRetry && current.Rev == rev+1 && verify(current) {
		return current, true, nil
	}
	return WorkItem{}, false, &integrations.RevConflictError{Rev: current.Rev, Hours: hoursOf(current.Fields), AfterRetry: conflict.AfterRetry}
}

func sameNumber(fields map[string]any, name string, want float64) bool {
	got, ok := numberField(fields, name)
	return ok && round2(got) == round2(want)
}

// SetEstimate implementa integrations.TaskTracker: fija la estimación original de la Task y su trabajo
// restante, RemainingWork = max(0, estimación − CompletedWork), con test /rev. En if_missing no escribe
// si la Task ya tiene estimación original; en replace usa req.Base, que es obligatoria.
func (p *Plugin) SetEstimate(ctx context.Context, req integrations.EstimateRequest) (integrations.EstimateResult, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.EstimateResult{}, err
	}
	hours := round2(req.Hours)
	if hours <= 0 {
		return integrations.EstimateResult{}, fmt.Errorf("%w: la estimación debe ser positiva", ErrBadRequest)
	}
	base := req.Base
	switch {
	case req.Mode == integrations.EstimateReplace && base.Rev <= 0:
		return integrations.EstimateResult{}, ErrMissingRev
	case base.Rev <= 0:
		if base, err = client.readTask(ctx, req.TaskID); err != nil {
			return integrations.EstimateResult{}, err
		}
	}
	if req.Mode == integrations.EstimateIfMissing && base.Hours.HasOriginal {
		return integrations.EstimateResult{Task: base}, nil
	}
	if err := client.requireFields(ctx, fieldOriginalEstimate, fieldRemainingWork); err != nil {
		return integrations.EstimateResult{}, err
	}
	id, err := parseTaskID(req.TaskID)
	if err != nil {
		return integrations.EstimateResult{}, err
	}
	remaining := round2(math.Max(0, hours-base.Hours.Completed))
	ops := []PatchOp{
		{Op: "add", Path: "/fields/" + fieldOriginalEstimate, Value: hours},
		{Op: "add", Path: "/fields/" + fieldRemainingWork, Value: remaining},
	}
	item, recovered, err := client.patchHours(ctx, id, base.Rev, ops, func(wi WorkItem) bool {
		return sameNumber(wi.Fields, fieldOriginalEstimate, hours) && sameNumber(wi.Fields, fieldRemainingWork, remaining)
	})
	if err != nil {
		return integrations.EstimateResult{}, err
	}
	task := client.taskItem(item, client.softCategories(ctx))
	if task.ParentID == "" {
		task.ParentID = base.ParentID
	}
	return integrations.EstimateResult{Task: task, Written: true, Recovered: recovered}, nil
}

// RecordHours implementa integrations.TaskTracker (D8, D10): imputa req.Hours con
// CompletedWork = round2(C + h) y RemainingWork = round2(max(0, R − h)) y test /rev sobre la lectura
// previa req.Base. Nunca reaplica tras un conflicto.
func (p *Plugin) RecordHours(ctx context.Context, req integrations.HoursRequest) (integrations.HoursResult, error) {
	client, err := p.NewClient(req.Config, req.Secrets)
	if err != nil {
		return integrations.HoursResult{}, err
	}
	hours := round2(req.Hours)
	if hours <= 0 {
		return integrations.HoursResult{}, fmt.Errorf("%w: las horas a imputar deben ser positivas", ErrBadRequest)
	}
	base := req.Base
	if base.Rev <= 0 {
		return integrations.HoursResult{}, ErrMissingRev
	}
	id, err := parseTaskID(base.ID)
	if err != nil {
		return integrations.HoursResult{}, err
	}
	if err := client.requireFields(ctx, fieldCompletedWork, fieldRemainingWork); err != nil {
		return integrations.HoursResult{}, err
	}
	completed := round2(base.Hours.Completed + hours)
	remaining := round2(math.Max(0, base.Hours.Remaining-hours))
	ops := []PatchOp{
		{Op: "add", Path: "/fields/" + fieldCompletedWork, Value: completed},
		{Op: "add", Path: "/fields/" + fieldRemainingWork, Value: remaining},
	}
	item, recovered, err := client.patchHours(ctx, id, base.Rev, ops, func(wi WorkItem) bool {
		return sameNumber(wi.Fields, fieldCompletedWork, completed) && sameNumber(wi.Fields, fieldRemainingWork, remaining)
	})
	if err != nil {
		return integrations.HoursResult{}, err
	}
	task := client.taskItem(item, client.softCategories(ctx))
	if task.ParentID == "" {
		task.ParentID = base.ParentID
	}
	return integrations.HoursResult{Task: task, Recovered: recovered}, nil
}
