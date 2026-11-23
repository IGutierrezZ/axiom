package adotest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/azuredevops"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
)

// recorderTB recoge los Errorf sin fallar la prueba, para comprobar la guarda de hosts locales.
type recorderTB struct {
	testing.TB
	errs []string
}

func (r *recorderTB) Helper()                   {}
func (r *recorderTB) Cleanup(fn func())         { r.TB.Cleanup(fn) }
func (r *recorderTB) Errorf(f string, a ...any) { r.errs = append(r.errs, f) }
func (r *recorderTB) Fatalf(f string, a ...any) { r.TB.Fatalf(f, a...) }

func do(t *testing.T, client *http.Client, method, url, contentType string, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestServerRecordsRequestsWithVirtualTime(t *testing.T) {
	srv := NewServer(t, nil)
	srv.Reply("GET", "/org/_apis/ping", Response{JSON: map[string]any{"ok": true}})
	client := srv.HTTPClient()

	do(t, client, "GET", srv.URL()+"/org/_apis/ping?api-version=7.1", "", "")
	srv.Clock().Advance(3 * time.Second)
	resp := do(t, client, "GET", srv.URL()+"/org/_apis/ping?api-version=7.1", "", "")

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	reqs := srv.Requests()
	if len(reqs) != 2 || reqs[0].Query.Get("api-version") != "7.1" {
		t.Fatalf("peticiones = %+v", reqs)
	}
	if got := reqs[1].At.Sub(reqs[0].At); got != 3*time.Second {
		t.Fatalf("tiempo virtual entre peticiones = %v, esperaba 3s", got)
	}
	if srv.Count("GET", "/org/_apis/ping") != 2 {
		t.Fatalf("Count = %d", srv.Count("GET", "/org/_apis/ping"))
	}
}

func TestServerUnmatchedRouteIs404AndRecorded(t *testing.T) {
	srv := NewServer(t, nil)
	resp := do(t, srv.HTTPClient(), "GET", srv.URL()+"/nada", "", "")
	if resp.StatusCode != 404 || len(srv.Requests()) != 1 {
		t.Fatalf("status = %d, peticiones = %d", resp.StatusCode, len(srv.Requests()))
	}
}

func TestServerSequenceRepeatsTheLastResponse(t *testing.T) {
	srv := NewServer(t, nil)
	srv.Sequence("GET", "/x", Response{Status: 503}, Response{Status: 200})
	client := srv.HTTPClient()
	var got []int
	for i := 0; i < 3; i++ {
		got = append(got, do(t, client, "GET", srv.URL()+"/x", "", "").StatusCode)
	}
	if got[0] != 503 || got[1] != 200 || got[2] != 200 {
		t.Fatalf("estados = %v", got)
	}
}

func TestHTTPClientRejectsNonLocalHosts(t *testing.T) {
	rec := &recorderTB{TB: t}
	srv := NewServer(rec, nil)
	_, err := srv.HTTPClient().Get("https://dev.azure.com/contoso/_apis/projects")
	if err == nil {
		t.Fatal("una petición a dev.azure.com debe rechazarse")
	}
	if len(rec.errs) != 1 || !strings.Contains(rec.errs[0], "host no local") {
		t.Fatalf("la guarda debe marcar la prueba como fallida: %v", rec.errs)
	}
	if n := len(srv.Requests()); n != 0 {
		t.Fatalf("el servidor local no debe recibir nada: %d", n)
	}
}

func TestClockSleepDoesNotSleepAndAdvancesVirtualTime(t *testing.T) {
	c := NewClock()
	start := c.Now()
	began := time.Now()
	if err := c.Sleep(context.Background(), 30*time.Second); err != nil {
		t.Fatal(err)
	}
	if time.Since(began) > time.Second {
		t.Fatal("Sleep no debe dormir")
	}
	if c.Now().Sub(start) != 30*time.Second || c.Slept() != 30*time.Second {
		t.Fatalf("now avanzó %v, durmió %v", c.Now().Sub(start), c.Slept())
	}
}

func TestClockSleepRunsHookAndReturnsContextError(t *testing.T) {
	c := NewClock()
	ctx, cancel := context.WithCancel(context.Background())
	var hooked time.Duration
	c.OnSleep(func(d time.Duration) { hooked = d; cancel() })
	start := c.Now()

	err := c.Sleep(ctx, 30*time.Second)

	if !errors.Is(err, context.Canceled) || hooked != 30*time.Second {
		t.Fatalf("err = %v, gancho = %v", err, hooked)
	}
	if c.Now() != start {
		t.Fatal("una espera cancelada no avanza el tiempo")
	}
	if got := c.Sleeps(); len(got) != 1 || got[0] != 30*time.Second {
		t.Fatalf("Sleeps = %v", got)
	}
}

func TestWorkItemSimulationHonoursTestRev(t *testing.T) {
	srv := NewServer(t, nil)
	srv.SeedWorkItem(42, 7, map[string]any{"System.Title": "viejo"}, map[string]any{"rel": "ArtifactLink", "url": "vstfs:///x"})
	client := srv.HTTPClient()
	base := srv.URL() + "/org/proj/_apis/wit/workitems/42"

	patch := func(rev int, title string) *http.Response {
		body, _ := json.Marshal([]map[string]any{
			{"op": "test", "path": "/rev", "value": rev},
			{"op": "replace", "path": "/fields/System.Title", "value": title},
		})
		return do(t, client, "PATCH", base, "application/json-patch+json", string(body))
	}

	if resp := patch(7, "nuevo"); resp.StatusCode != 200 {
		t.Fatalf("PATCH vigente = %d", resp.StatusCode)
	}
	item, _ := srv.WorkItem(42)
	if item.Rev != 8 || item.Fields["System.Title"] != "nuevo" || srv.AppliedPatches(42) != 1 {
		t.Fatalf("estado = %+v, aplicados = %d", item, srv.AppliedPatches(42))
	}
	if resp := patch(7, "otro"); resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("PATCH con rev obsoleto = %d, esperaba 412", resp.StatusCode)
	}
	srv.RejectRevWith(409)
	if resp := patch(7, "otro"); resp.StatusCode != 409 {
		t.Fatalf("PATCH con rev obsoleto = %d, esperaba 409", resp.StatusCode)
	}
	if item, _ = srv.WorkItem(42); item.Rev != 8 || srv.AppliedPatches(42) != 1 {
		t.Fatal("un rechazo no modifica el work item")
	}
}

func TestWorkItemSimulationGetAndBatch(t *testing.T) {
	srv := NewServer(t, nil)
	for id := 1; id <= 3; id++ {
		srv.SeedWorkItem(id, id, map[string]any{"System.Title": "t"})
	}
	client := srv.HTTPClient()

	resp := do(t, client, "GET", srv.URL()+"/o/p/_apis/wit/workitems/1?$expand=relations", "", "")
	if resp.StatusCode != 200 {
		t.Fatalf("GET = %d", resp.StatusCode)
	}
	if resp = do(t, client, "GET", srv.URL()+"/o/p/_apis/wit/workitems/99", "", ""); resp.StatusCode != 404 {
		t.Fatalf("GET inexistente = %d", resp.StatusCode)
	}

	srv.ScrambleBatch(true)
	resp = do(t, client, "POST", srv.URL()+"/o/p/_apis/wit/workitemsbatch", "application/json", `{"ids":[1,2,3],"errorPolicy":"Fail"}`)
	var out struct {
		Value []struct{ ID int } `json:"value"`
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil || len(out.Value) != 3 || out.Value[0].ID != 3 {
		t.Fatalf("lote desordenado = %s (%v)", buf.String(), err)
	}
	if resp = do(t, client, "POST", srv.URL()+"/o/p/_apis/wit/workitemsbatch", "application/json", `{"ids":[1,999],"errorPolicy":"Fail"}`); resp.StatusCode != 404 {
		t.Fatalf("lote con inexistente y Fail = %d", resp.StatusCode)
	}
}

// Install instala un registro cuyo plugin habla con el servidor local y un almacén en memoria; el
// endpoint efectivo es el servidor local (guarda de REQ-45.16).
func TestInstallPointsThePluginAtTheLocalServer(t *testing.T) {
	srv := NewServer(t, nil)
	memory := Install(t, srv)

	reg, err := integrations.ResolveRegistry(nil)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := reg.Get("azure-devops")
	if !ok {
		t.Fatal("el registro debe contener azure-devops")
	}
	plugin, ok := p.(*azuredevops.Plugin)
	if !ok {
		t.Fatalf("plugin = %T", p)
	}
	ref := secrets.Ref{Plugin: "azure-devops", Name: "pat"}
	if err := memory.Set(context.Background(), ref, secrets.NewSecret("pat-ficticio-de-prueba")); err != nil {
		t.Fatal(err)
	}
	src := fixedSource{secrets.NewSecret("pat-ficticio-de-prueba")}
	client, err := plugin.NewClient(map[string]any{"organization": "contoso", "project": "p", "team": "t"}, src)
	if err != nil {
		t.Fatal(err)
	}
	if client.Endpoint() != srv.URL() {
		t.Fatalf("el endpoint efectivo es %q, esperaba el servidor local %q", client.Endpoint(), srv.URL())
	}
	if strings.Contains(client.Endpoint(), "dev.azure.com") {
		t.Fatal("el endpoint de las pruebas no puede ser dev.azure.com")
	}
	if got, _, err := secrets.Default().Get(context.Background(), ref); err != nil || got.Reveal() != "pat-ficticio-de-prueba" {
		t.Fatalf("el almacén en memoria debe estar instalado: %v", err)
	}
}

type fixedSource struct{ secret secrets.Secret }

func (f fixedSource) Secret(context.Context, string) (secrets.Secret, error) { return f.secret, nil }
