package adotest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// WorkItem es un work item del servidor simulado.
type WorkItem struct {
	ID        int
	Rev       int
	Fields    map[string]any
	Relations []map[string]any
}

// MaxBatch es el máximo de identificadores que admite el simulacro de workitemsbatch, como la API.
const MaxBatch = 200

type workItemStore struct {
	mu           sync.Mutex
	items        map[int]*WorkItem
	applied      map[int]int
	rejectStatus int
	loseNext     bool
	scramble     bool
	afterRead    map[int][]map[string]any // ediciones ajenas que se aplican tras cada lectura
}

var (
	itemPath  = regexp.MustCompile(`/_apis/wit/workitems/(\d+)$`)
	batchPath = regexp.MustCompile(`/_apis/wit/workitemsbatch$`)
)

// store devuelve el almacén de work items, creándolo la primera vez. Se llama con s.mu tomado.
func (s *Server) store() *workItemStore {
	if s.items == nil {
		s.items = &workItemStore{items: map[int]*WorkItem{}, applied: map[int]int{}, rejectStatus: http.StatusPreconditionFailed}
	}
	return s.items
}

// SeedWorkItem da de alta un work item. Los campos sembrados son los únicos que admite un PATCH. Activa
// el simulacro de GET .../_apis/wit/workitems/{id}, PATCH sobre la misma ruta y POST
// .../_apis/wit/workitemsbatch (con cualquier prefijo de organización y proyecto). Las rutas
// registradas con Handle tienen prioridad.
func (s *Server) SeedWorkItem(id, rev int, fields map[string]any, relations ...map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.store()
	st.mu.Lock()
	defer st.mu.Unlock()
	copied := map[string]any{}
	for k, v := range fields {
		copied[k] = v
	}
	st.items[id] = &WorkItem{ID: id, Rev: rev, Fields: copied, Relations: relations}
}

// WorkItem devuelve una copia del work item del servidor.
func (s *Server) WorkItem(id int) (WorkItem, bool) {
	s.mu.Lock()
	st := s.items
	s.mu.Unlock()
	if st == nil {
		return WorkItem{}, false
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	it, ok := st.items[id]
	if !ok {
		return WorkItem{}, false
	}
	copied := *it
	copied.Fields = map[string]any{}
	for k, v := range it.Fields {
		copied.Fields[k] = v
	}
	copied.Relations = append([]map[string]any(nil), it.Relations...)
	return copied, true
}

// AppliedPatches devuelve cuántos PATCH se aplicaron al work item.
func (s *Server) AppliedPatches(id int) int {
	s.mu.Lock()
	st := s.items
	s.mu.Unlock()
	if st == nil {
		return 0
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.applied[id]
}

// RejectRevWith fija el código con el que el simulacro rechaza la prueba test /rev: 412 (por
// defecto), 400 o 409. El contrato real no está verificado: el simulacro cubre las tres variantes.
func (s *Server) RejectRevWith(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.store()
	st.mu.Lock()
	defer st.mu.Unlock()
	st.rejectStatus = status
}

// LoseNextPatchResponse hace que el siguiente PATCH se aplique pero la conexión se corte antes de
// responder.
func (s *Server) LoseNextPatchResponse() {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.store()
	st.mu.Lock()
	defer st.mu.Unlock()
	st.loseNext = true
}

// ScrambleBatch hace que workitemsbatch devuelva los work items en orden inverso al pedido.
func (s *Server) ScrambleBatch(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.store()
	st.mu.Lock()
	defer st.mu.Unlock()
	st.scramble = on
}

func (st *workItemStore) match(_ *Server, req *Request) Handler {
	switch {
	case req.Method == http.MethodGet && itemPath.MatchString(req.Path):
		return st.get
	case req.Method == http.MethodPatch && itemPath.MatchString(req.Path):
		return st.patch
	case req.Method == http.MethodPost && batchPath.MatchString(req.Path):
		return st.batch
	}
	return nil
}

func notFoundResponse(id int) Response {
	return Response{Status: http.StatusNotFound, JSON: map[string]any{
		"message": fmt.Sprintf("TF401232: Work item %d does not exist, or you do not have permissions to read it.", id),
		"typeKey": "WorkItemDoesNotExistException",
	}}
}

func render(it *WorkItem, withRelations bool) map[string]any {
	out := map[string]any{"id": it.ID, "rev": it.Rev, "fields": it.Fields, "url": fmt.Sprintf("https://adotest.invalid/workItems/%d", it.ID)}
	if withRelations && len(it.Relations) > 0 {
		out["relations"] = it.Relations
	}
	return out
}

func (st *workItemStore) get(req *Request) Response {
	id, _ := strconv.Atoi(itemPath.FindStringSubmatch(req.Path)[1])
	st.mu.Lock()
	defer st.mu.Unlock()
	it, ok := st.items[id]
	if !ok {
		return notFoundResponse(id)
	}
	expand := strings.EqualFold(req.Query.Get("$expand"), "relations")
	resp := Response{JSON: render(it, expand)}
	if edits := st.afterRead[id]; len(edits) > 0 {
		st.afterRead[id] = edits[1:]
		updated := map[string]any{}
		for k, v := range it.Fields {
			updated[k] = v
		}
		for k, v := range edits[0] {
			updated[k] = v
		}
		it.Fields = updated
		it.Rev++
	}
	return resp
}

func (st *workItemStore) batch(req *Request) Response {
	var body struct {
		IDs         []int  `json:"ids"`
		Expand      string `json:"$expand"`
		ErrorPolicy string `json:"errorPolicy"`
	}
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return Response{Status: http.StatusBadRequest, JSON: map[string]any{"message": "adotest: cuerpo de lote inválido"}}
	}
	if len(body.IDs) > MaxBatch {
		return Response{Status: http.StatusBadRequest, JSON: map[string]any{"message": fmt.Sprintf("adotest: el lote supera los %d identificadores", MaxBatch)}}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	var value []any
	for _, id := range body.IDs {
		it, ok := st.items[id]
		if !ok {
			if strings.EqualFold(body.ErrorPolicy, "Fail") {
				return notFoundResponse(id)
			}
			continue
		}
		value = append(value, render(it, strings.EqualFold(body.Expand, "relations")))
	}
	if st.scramble {
		for i, j := 0, len(value)-1; i < j; i, j = i+1, j-1 {
			value[i], value[j] = value[j], value[i]
		}
	}
	return Response{JSON: map[string]any{"count": len(value), "value": value}}
}

type patchOp struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value"`
}

func (st *workItemStore) rejection() Response {
	if st.rejectStatus == http.StatusPreconditionFailed {
		return Response{Status: http.StatusPreconditionFailed, JSON: map[string]any{
			"message": "TF26071: This work item has been changed by someone else since you opened it.",
			"typeKey": "WorkItemRevisionMismatchException",
		}}
	}
	return Response{Status: st.rejectStatus, JSON: map[string]any{
		"message": "The test operation at path '/rev' failed: the value does not match.",
		"typeKey": "TestOperationFailedException",
	}}
}

func (st *workItemStore) patch(req *Request) Response {
	id, _ := strconv.Atoi(itemPath.FindStringSubmatch(req.Path)[1])
	if !strings.HasPrefix(req.Header.Get("Content-Type"), "application/json-patch+json") {
		return Response{Status: http.StatusUnsupportedMediaType, JSON: map[string]any{"message": "adotest: se espera application/json-patch+json"}}
	}
	var ops []patchOp
	if err := json.Unmarshal(req.Body, &ops); err != nil || len(ops) == 0 {
		return Response{Status: http.StatusBadRequest, JSON: map[string]any{"message": "adotest: documento JSON Patch inválido"}}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	it, ok := st.items[id]
	if !ok {
		return notFoundResponse(id)
	}
	updated := map[string]any{}
	for k, v := range it.Fields {
		updated[k] = v
	}
	relations := append([]map[string]any(nil), it.Relations...)
	for _, op := range ops {
		switch {
		case op.Op == "test" && op.Path == "/rev":
			var want int
			if err := json.Unmarshal(op.Value, &want); err != nil || want != it.Rev {
				return st.rejection()
			}
		case (op.Op == "add" || op.Op == "replace") && strings.HasPrefix(op.Path, "/fields/"):
			name := strings.TrimPrefix(op.Path, "/fields/")
			// System.Tags existe en todo work item aunque la prueba no lo siembre.
			if _, known := it.Fields[name]; !known && name != "System.Tags" {
				return Response{Status: http.StatusBadRequest, JSON: map[string]any{
					"message": "TF51535: Cannot find field " + name + ".",
					"typeKey": "RuleValidationException",
				}}
			}
			var value any
			_ = json.Unmarshal(op.Value, &value)
			updated[name] = value
		case op.Op == "add" && op.Path == "/relations/-":
			var rel map[string]any
			if err := json.Unmarshal(op.Value, &rel); err != nil || rel["rel"] == nil || rel["url"] == nil {
				return Response{Status: http.StatusBadRequest, JSON: map[string]any{"message": "adotest: relación inválida en add /relations/-"}}
			}
			relations = append(relations, rel)
		default:
			return Response{Status: http.StatusBadRequest, JSON: map[string]any{"message": "adotest: operación no soportada " + op.Op + " " + op.Path}}
		}
	}
	it.Fields = updated
	it.Relations = relations
	it.Rev++
	st.applied[id]++
	if st.loseNext {
		st.loseNext = false
		return Response{Drop: true}
	}
	return Response{JSON: render(it, false)}
}

// EditAfterNextRead simula que otra persona edita el work item id justo después de la próxima
// lectura (GET) que se haga de él: aplica fields y sube la revisión. Cada llamada se consume en una
// lectura distinta, en orden.
func (s *Server) EditAfterNextRead(id int, fields map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.store()
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.afterRead == nil {
		st.afterRead = map[int][]map[string]any{}
	}
	st.afterRead[id] = append(st.afterRead[id], fields)
}
