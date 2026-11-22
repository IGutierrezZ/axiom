package azuredevops_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops/adotest"
)

var retitle = []azuredevops.PatchOp{{Op: "replace", Path: "/fields/System.Title", Value: "título nuevo"}}

func seededServer(t *testing.T, rev int) (*adotest.Server, *azuredevops.Client) {
	t.Helper()
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, rev, map[string]any{"System.Title": "título viejo"})
	client, _ := newTestClient(t, srv, nil)
	return srv, client
}

func TestUpdateWorkItemWithCurrentRev(t *testing.T) {
	srv, client := seededServer(t, 7)

	item, err := client.UpdateWorkItem(context.Background(), 42, 7, retitle)

	if err != nil {
		t.Fatalf("UpdateWorkItem: %v", err)
	}
	if item.Rev != 8 || item.Fields["System.Title"] != "título nuevo" {
		t.Fatalf("resultado = %+v", item)
	}
	state, _ := srv.WorkItem(42)
	if state.Rev != 8 || state.Fields["System.Title"] != "título nuevo" || srv.AppliedPatches(42) != 1 {
		t.Fatalf("el servidor debe quedar en rev 8 con el título nuevo: %+v", state)
	}
	reqs := srv.Requests()
	if len(reqs) != 1 || reqs[0].Method != "PATCH" || reqs[0].Path != workItemsBase+"42" {
		t.Fatalf("peticiones = %+v", reqs)
	}
	if got := reqs[0].Header.Get("Content-Type"); got != "application/json-patch+json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var ops []map[string]any
	if err := json.Unmarshal(reqs[0].Body, &ops); err != nil || len(ops) != 2 {
		t.Fatalf("cuerpo = %s (%v)", reqs[0].Body, err)
	}
	if ops[0]["op"] != "test" || ops[0]["path"] != "/rev" || ops[0]["value"] != float64(7) {
		t.Fatalf("la primera operación debe ser test /rev con valor 7: %v", ops[0])
	}
}

func TestUpdateWorkItemWithStaleRevMakesOneRequestAndChangesNothing(t *testing.T) {
	srv, client := seededServer(t, 9)

	_, err := client.UpdateWorkItem(context.Background(), 42, 7, retitle)

	var conflict *azuredevops.RevConflictError
	if !errors.As(err, &conflict) || !errors.Is(err, azuredevops.ErrRevConflict) {
		t.Fatalf("esperaba RevConflictError: %v", err)
	}
	if conflict.WorkItemID != 42 || conflict.Rev != 7 || conflict.AfterRetry {
		t.Fatalf("conflicto = %+v", conflict)
	}
	if got := srv.Count("PATCH", workItemsBase+"42"); got != 1 || len(srv.Requests()) != 1 {
		t.Fatalf("peticiones = %d, esperaba una sola (sin reintento ni relectura)", len(srv.Requests()))
	}
	state, _ := srv.WorkItem(42)
	if state.Rev != 9 || state.Fields["System.Title"] != "título viejo" || srv.AppliedPatches(42) != 0 {
		t.Fatalf("el work item no debe modificarse: %+v", state)
	}
	if len(srv.Clock().Sleeps()) != 0 {
		t.Fatal("un conflicto no se reintenta: sin esperas")
	}
}

func TestEveryRejectionOfTheRevTestIsAConflict(t *testing.T) {
	for _, status := range []int{412, 400, 409} {
		t.Run(statusName(status), func(t *testing.T) {
			srv, client := seededServer(t, 9)
			srv.RejectRevWith(status)

			_, err := client.UpdateWorkItem(context.Background(), 42, 7, retitle)

			if !errors.Is(err, azuredevops.ErrRevConflict) {
				t.Fatalf("un %d de test debe ser ErrRevConflict: %v", status, err)
			}
			if got := len(srv.Requests()); got != 1 {
				t.Fatalf("peticiones = %d, esperaba 1 (sin reintento)", got)
			}
		})
	}
}

func statusName(status int) string {
	switch status {
	case 412:
		return "412"
	case 400:
		return "400"
	}
	return "409"
}

func TestAnUnrelated400IsNotAConflict(t *testing.T) {
	srv, client := seededServer(t, 7)

	_, err := client.UpdateWorkItem(context.Background(), 42, 7, []azuredevops.PatchOp{{Op: "replace", Path: "/fields/System.Inexistente", Value: "x"}})

	var api *azuredevops.APIError
	if !errors.As(err, &api) || !errors.Is(err, azuredevops.ErrBadRequest) || errors.Is(err, azuredevops.ErrRevConflict) {
		t.Fatalf("esperaba APIError ErrBadRequest distinto de ErrRevConflict: %v", err)
	}
	if api.Status != 400 || srv.Count("PATCH", workItemsBase+"42") != 1 {
		t.Fatalf("un 4xx nunca se reintenta: %+v", api)
	}
}

func TestLostResponseThenRetryIsAConflictAndAppliesOnce(t *testing.T) {
	srv, client := seededServer(t, 7)
	srv.LoseNextPatchResponse()

	_, err := client.UpdateWorkItem(context.Background(), 42, 7, retitle)

	var conflict *azuredevops.RevConflictError
	if !errors.As(err, &conflict) || !conflict.AfterRetry {
		t.Fatalf("esperaba RevConflictError con AfterRetry: %v", err)
	}
	if got := srv.AppliedPatches(42); got != 1 {
		t.Fatalf("el servidor aplicó la modificación %d veces, esperaba una sola", got)
	}
	if got := srv.Count("PATCH", workItemsBase+"42"); got != 2 {
		t.Fatalf("PATCH enviados = %d, esperaba 2 (el original y el reintento)", got)
	}
	if conflict.WorkItemID != 42 || conflict.Rev != 7 {
		t.Fatalf("conflicto = %+v", conflict)
	}
	if !errors.Is(err, azuredevops.ErrRevConflict) {
		t.Fatal("debe ser ErrRevConflict")
	}
	if msg := err.Error(); !strings.Contains(msg, "pudo aplicarse") {
		t.Fatalf("el mensaje debe advertir de que pudo aplicarse: %s", msg)
	}
}

func TestUpdateWorkItemWithoutRevOrWithCallerTestSendsNothing(t *testing.T) {
	srv, client := seededServer(t, 7)

	if _, err := client.UpdateWorkItem(context.Background(), 42, 0, retitle); !errors.Is(err, azuredevops.ErrMissingRev) {
		t.Errorf("sin rev: esperaba ErrMissingRev, obtuvo %v", err)
	}
	if _, err := client.UpdateWorkItem(context.Background(), 42, 7, []azuredevops.PatchOp{{Op: "test", Path: "/rev", Value: 1}}); !errors.Is(err, azuredevops.ErrBadRequest) {
		t.Errorf("test /rev aportado: esperaba ErrBadRequest, obtuvo %v", err)
	}
	if _, err := client.UpdateWorkItem(context.Background(), 42, 7, nil); !errors.Is(err, azuredevops.ErrBadRequest) {
		t.Errorf("sin operaciones: esperaba ErrBadRequest, obtuvo %v", err)
	}
	if _, err := client.UpdateWorkItem(context.Background(), 0, 7, retitle); !errors.Is(err, azuredevops.ErrBadRequest) {
		t.Errorf("id inválido: esperaba ErrBadRequest, obtuvo %v", err)
	}
	if len(srv.Requests()) != 0 {
		t.Fatalf("una modificación mal construida no envía nada: %d peticiones", len(srv.Requests()))
	}
}

func TestPatchIsRetriedOn503ButNeverAppliedTwice(t *testing.T) {
	srv := adotest.NewServer(t, nil)
	srv.SeedWorkItem(42, 7, map[string]any{"System.Title": "título viejo"})
	failedOnce := false
	// El primer PATCH recibe 503 antes de llegar al simulacro; el reintento lo aplica.
	srv.Handle("PATCH", workItemsBase+"42", func(r *adotest.Request) adotest.Response {
		if !failedOnce {
			failedOnce = true
			return adotest.Response{Status: 503}
		}
		return adotest.Response{Status: 200, JSON: map[string]any{"id": 42, "rev": 8, "fields": map[string]any{"System.Title": "título nuevo"}}}
	})
	client, _ := newTestClient(t, srv, nil)

	item, err := client.UpdateWorkItem(context.Background(), 42, 7, retitle)

	if err != nil || item.Rev != 8 {
		t.Fatalf("UpdateWorkItem = %+v, %v", item, err)
	}
	if got := srv.Count("PATCH", workItemsBase+"42"); got != 2 {
		t.Fatalf("PATCH enviados = %d, esperaba 2", got)
	}
}
