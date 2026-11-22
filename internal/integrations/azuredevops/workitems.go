package azuredevops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// maxBatchSize es el máximo de identificadores que admite workitemsbatch.
const maxBatchSize = 200

// WorkItem es un work item de Azure DevOps: identificador, revisión (rev), campos y relaciones.
type WorkItem struct {
	ID        int            `json:"id"`
	Rev       int            `json:"rev"`
	Fields    map[string]any `json:"fields"`
	Relations []Relation     `json:"relations,omitempty"`
	URL       string         `json:"url,omitempty"`
}

// Relation es una relación de un work item con otro recurso.
type Relation struct {
	Rel        string         `json:"rel"`
	URL        string         `json:"url"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// PatchOp es una operación JSON Patch sobre un work item. La operación test sobre /rev la antepone
// siempre el cliente: quien llama no puede aportarla (newPatchBody la rechaza).
type PatchOp struct {
	Op    string `json:"op"` // add | replace | remove | copy | move
	Path  string `json:"path"`
	From  string `json:"from,omitempty"`
	Value any    `json:"value,omitempty"`
}

func invalidWorkItemID(id int) error {
	return fmt.Errorf("%w: el identificador de work item %d no es válido", ErrBadRequest, id)
}

// GetWorkItem lee un work item con sus relaciones: GET .../_apis/wit/workitems/{id}?$expand=relations.
// Un work item inexistente es *NotFoundError con su identificador.
func (c *Client) GetWorkItem(ctx context.Context, id int) (WorkItem, error) {
	if id <= 0 {
		return WorkItem{}, invalidWorkItemID(id)
	}
	var item WorkItem
	err := c.do(ctx, request{
		method:     "GET",
		segments:   []string{c.target.Project, "_apis", "wit", "workitems", strconv.Itoa(id)},
		query:      url.Values{"$expand": {"relations"}},
		scope:      scopeWorkWrite,
		operation:  fmt.Sprintf("leer el work item %d", id),
		retryable:  true,
		workItemID: id,
	}, &item)
	if err != nil {
		return WorkItem{}, err
	}
	return item, nil
}

// batchResponse es la respuesta de workitemsbatch.
type batchResponse struct {
	Value []WorkItem `json:"value"`
}

// GetWorkItems lee varios work items con POST .../_apis/wit/workitemsbatch, troceando la lista en
// lotes del máximo que admite la API (200). El resultado conserva el orden de los identificadores
// pedidos, no el de la API. Una lista vacía devuelve un resultado vacío sin enviar nada. Con
// errorPolicy Fail, un identificador inexistente es *NotFoundError, igual que la lectura individual.
func (c *Client) GetWorkItems(ctx context.Context, ids []int) ([]WorkItem, error) {
	if len(ids) == 0 {
		return []WorkItem{}, nil
	}
	for _, id := range ids {
		if id <= 0 {
			return nil, invalidWorkItemID(id)
		}
	}
	byID := make(map[int]WorkItem, len(ids))
	for start := 0; start < len(ids); start += maxBatchSize {
		chunk := ids[start:min(start+maxBatchSize, len(ids))]
		body, err := json.Marshal(map[string]any{"ids": chunk, "$expand": "Relations", "errorPolicy": "Fail"})
		if err != nil {
			return nil, fmt.Errorf("serializar el lote de work items: %w", err)
		}
		var resp batchResponse
		if err := c.do(ctx, request{
			method:      "POST",
			contentType: "application/json",
			segments:    []string{c.target.Project, "_apis", "wit", "workitemsbatch"},
			body:        body,
			scope:       scopeWorkWrite,
			operation:   fmt.Sprintf("leer un lote de %d work items", len(chunk)),
			retryable:   true,
			batchIDs:    chunk,
		}, &resp); err != nil {
			return nil, err
		}
		for _, item := range resp.Value {
			byID[item.ID] = item
		}
	}
	out := make([]WorkItem, 0, len(ids))
	for _, id := range ids {
		item, ok := byID[id]
		if !ok {
			return nil, &NotFoundError{Operation: "leer los work items", WorkItemID: id}
		}
		out = append(out, item)
	}
	return out, nil
}

// numberToken separa los números de un mensaje de error.
var numberToken = regexp.MustCompile(`\d+`)

// missingIDFromMessage devuelve, entre los identificadores pedidos, el primero que el mensaje de un
// 404 de lote nombra como número completo, o 0 si ninguno.
func missingIDFromMessage(message string, requested []int) int {
	want := make(map[int]bool, len(requested))
	for _, id := range requested {
		want[id] = true
	}
	for _, token := range numberToken.FindAllString(message, -1) {
		if n, err := strconv.Atoi(token); err == nil && want[n] {
			return n
		}
	}
	return 0
}

// newPatchBody es la única forma de construir el cuerpo de una modificación: antepone siempre
// {"op":"test","path":"/rev","value":rev}. Rechaza una revisión no positiva (ErrMissingRev), una lista
// de operaciones vacía y cualquier operación test sobre /rev aportada por quien llama.
func newPatchBody(rev int, ops []PatchOp) ([]byte, error) {
	if rev <= 0 {
		return nil, ErrMissingRev
	}
	if len(ops) == 0 {
		return nil, fmt.Errorf("%w: la modificación no trae operaciones", ErrBadRequest)
	}
	body := make([]PatchOp, 0, len(ops)+1)
	body = append(body, PatchOp{Op: "test", Path: "/rev", Value: rev})
	for _, op := range ops {
		if strings.TrimSpace(op.Op) == "" || strings.TrimSpace(op.Path) == "" {
			return nil, fmt.Errorf("%w: toda operación necesita op y path", ErrBadRequest)
		}
		if strings.EqualFold(op.Op, "test") && op.Path == "/rev" {
			return nil, fmt.Errorf("%w: la prueba test sobre /rev la antepone el cliente", ErrBadRequest)
		}
		body = append(body, op)
	}
	return json.Marshal(body)
}

// UpdateWorkItem modifica un work item con PATCH .../_apis/wit/workitems/{id} y concurrencia
// optimista: el cuerpo empieza siempre por test /rev con la revisión leída. Un rechazo de esa prueba
// (412, o 400/409 con el error de test) es *RevConflictError: el cliente no lo reintenta, no relee
// para reaplicar y el work item no se modifica. Un PATCH aplicado cuya respuesta se pierde y se
// reintenta falla por /rev y se informa como conflicto con AfterRetry: nunca se aplica dos veces. El
// manifiesto no declara capacidad de escritura: ningún evento invoca esta operación todavía.
func (c *Client) UpdateWorkItem(ctx context.Context, id, rev int, ops []PatchOp) (WorkItem, error) {
	if id <= 0 {
		return WorkItem{}, invalidWorkItemID(id)
	}
	body, err := newPatchBody(rev, ops)
	if err != nil {
		return WorkItem{}, err
	}
	var item WorkItem
	err = c.do(ctx, request{
		method:      "PATCH",
		contentType: "application/json-patch+json",
		segments:    []string{c.target.Project, "_apis", "wit", "workitems", strconv.Itoa(id)},
		body:        body,
		scope:       scopeWorkWrite,
		operation:   fmt.Sprintf("modificar el work item %d", id),
		retryable:   true,
		revPatch:    true,
		workItemID:  id,
		rev:         rev,
	}, &item)
	if err != nil {
		return WorkItem{}, err
	}
	return item, nil
}
