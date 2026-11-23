package azuredevops_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

const (
	workItemsBase = "/contoso/Fabrikam/_apis/wit/workitems/"
	batchRoute    = "/contoso/Fabrikam/_apis/wit/workitemsbatch"
)

func TestGetWorkItemExpandsRelations(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 7, map[string]any{"System.Title": "Arreglar el login"},
		map[string]any{"rel": "System.LinkTypes.Hierarchy-Reverse", "url": "https://adotest.invalid/workItems/1"})
	client, _ := newTestClient(t, srv, nil)

	item, err := client.GetWorkItem(context.Background(), 42)

	if err != nil {
		t.Fatalf("GetWorkItem: %v", err)
	}
	if item.ID != 42 || item.Rev != 7 || item.Fields["System.Title"] != "Arreglar el login" {
		t.Fatalf("work item = %+v", item)
	}
	if len(item.Relations) != 1 || item.Relations[0].Rel != "System.LinkTypes.Hierarchy-Reverse" {
		t.Fatalf("relaciones = %+v", item.Relations)
	}
	reqs := srv.Requests()
	if len(reqs) != 1 || reqs[0].Method != "GET" || reqs[0].Path != workItemsBase+"42" {
		t.Fatalf("peticiones = %+v", reqs)
	}
	if got := reqs[0].Query.Get("$expand"); got != "relations" {
		t.Fatalf("$expand = %q", got)
	}
}

func TestGetWorkItemNotFoundNamesTheIDAndIsNotRetried(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.Reply("GET", workItemsBase+"999", adotest.Response{Status: 404, JSON: map[string]any{"message": "no existe"}})
	client, _ := newTestClient(t, srv, nil)

	_, err := client.GetWorkItem(context.Background(), 999)

	var nf *azuredevops.NotFoundError
	if !errors.As(err, &nf) || !errors.Is(err, azuredevops.ErrNotFound) || nf.WorkItemID != 999 {
		t.Fatalf("esperaba NotFoundError del work item 999: %v", err)
	}
	if !strings.Contains(err.Error(), "999") {
		t.Fatalf("el mensaje debe nombrar el 999: %v", err)
	}
	if got := srv.Count("GET", workItemsBase+"999"); got != 1 {
		t.Fatalf("peticiones = %d, esperaba 1 (sin reintento)", got)
	}
}

func TestInvalidWorkItemIDsSendNothing(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	client, _ := newTestClient(t, srv, nil)
	if _, err := client.GetWorkItem(context.Background(), 0); !errors.Is(err, azuredevops.ErrBadRequest) {
		t.Errorf("GetWorkItem(0): %v", err)
	}
	if _, err := client.GetWorkItems(context.Background(), []int{1, -2}); !errors.Is(err, azuredevops.ErrBadRequest) {
		t.Errorf("GetWorkItems con un id negativo: %v", err)
	}
	if len(srv.Requests()) != 0 {
		t.Fatal("un identificador inválido no envía nada")
	}
}

func TestGetWorkItemsEmptyListSendsNothing(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	client, _ := newTestClient(t, srv, nil)

	for _, ids := range [][]int{nil, {}} {
		items, err := client.GetWorkItems(context.Background(), ids)
		if err != nil || items == nil || len(items) != 0 {
			t.Fatalf("GetWorkItems(%v) = %v, %v; esperaba un resultado vacío y no nulo", ids, items, err)
		}
	}
	if len(srv.Requests()) != 0 {
		t.Fatal("una lista vacía no envía ninguna petición")
	}
}

func TestGetWorkItemsSplitsIntoBatchesAndKeepsTheRequestedOrder(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	const total = 450
	ids := make([]int, 0, total)
	for id := total; id >= 1; id-- { // orden pedido: descendente, no el natural de la API
		srv.SeedWorkItem(id, id, map[string]any{"System.Title": "t"})
		ids = append(ids, id)
	}
	srv.ScrambleBatch(true) // la API devuelve cada lote al revés
	client, _ := newTestClient(t, srv, nil)

	items, err := client.GetWorkItems(context.Background(), ids)

	if err != nil {
		t.Fatalf("GetWorkItems: %v", err)
	}
	reqs := srv.Requests()
	if len(reqs) != 3 {
		t.Fatalf("POST enviados = %d, esperaba 3 (450 ids en lotes de 200)", len(reqs))
	}
	var sent []int
	for i, r := range reqs {
		if r.Method != "POST" || r.Path != batchRoute {
			t.Fatalf("petición %d = %s %s", i, r.Method, r.Path)
		}
		var body struct {
			IDs         []int  `json:"ids"`
			Expand      string `json:"$expand"`
			ErrorPolicy string `json:"errorPolicy"`
		}
		if err := json.Unmarshal(r.Body, &body); err != nil {
			t.Fatalf("cuerpo %d: %v", i, err)
		}
		if len(body.IDs) > 200 {
			t.Fatalf("el lote %d lleva %d identificadores, el máximo es 200", i, len(body.IDs))
		}
		if body.Expand != "Relations" || body.ErrorPolicy != "Fail" {
			t.Fatalf("lote %d: $expand=%q errorPolicy=%q", i, body.Expand, body.ErrorPolicy)
		}
		sent = append(sent, body.IDs...)
	}
	if !reflect.DeepEqual(sent, ids) {
		t.Fatal("los lotes deben cubrir los identificadores pedidos, en orden")
	}
	got := make([]int, len(items))
	for i, item := range items {
		got[i] = item.ID
	}
	if !reflect.DeepEqual(got, ids) {
		t.Fatalf("el resultado no conserva el orden de los identificadores pedidos: %v...", got[:5])
	}
}

func TestGetWorkItemsMissingIDWithFailPolicyIsNotFound(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(1, 1, map[string]any{"System.Title": "t"})
	client, _ := newTestClient(t, srv, nil)

	_, err := client.GetWorkItems(context.Background(), []int{1, 999})

	var nf *azuredevops.NotFoundError
	if !errors.As(err, &nf) || !errors.Is(err, azuredevops.ErrNotFound) {
		t.Fatalf("esperaba NotFoundError: %v", err)
	}
	if nf.WorkItemID != 999 {
		t.Fatalf("el error debe nombrar el work item que falta (999): %+v", nf)
	}
	if got := srv.Count("POST", batchRoute); got != 1 {
		t.Fatalf("peticiones = %d, esperaba 1 (sin reintento)", got)
	}
}
