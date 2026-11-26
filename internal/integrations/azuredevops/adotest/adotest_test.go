package adotest

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func noRedirect(srv *Server) *http.Client {
	c := srv.HTTPClient()
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return c
}

func TestAdotestRelationsTagsAttachments(t *testing.T) {
	srv := NewServer(t, nil)
	srv.SeedWorkItem(42, 3, map[string]any{"System.Title": "t"},
		map[string]any{"rel": "System.LinkTypes.Hierarchy-Reverse", "url": srv.URL() + "/org/_apis/wit/workItems/7"})
	srv.SeedAttachment("/org/p/_apis/wit/attachments/a1", Blob{Data: []byte("hola"), ContentType: "image/png"})
	client := srv.HTTPClient()

	resp := do(t, client, "GET", srv.URL()+"/org/p/_apis/wit/workitems/42?$expand=relations", "", "")
	var item struct {
		Rev       int
		Relations []map[string]any
	}
	_ = json.NewDecoder(resp.Body).Decode(&item)
	if item.Rev != 3 || len(item.Relations) != 1 {
		t.Fatalf("lectura = %+v", item)
	}

	// System.Tags no se siembra y aun así admite un PATCH.
	body, _ := json.Marshal([]map[string]any{
		{"op": "test", "path": "/rev", "value": 3},
		{"op": "add", "path": "/fields/System.Tags", "value": "a; axiom:c"},
	})
	if resp := do(t, client, "PATCH", srv.URL()+"/org/p/_apis/wit/workitems/42", "application/json-patch+json", string(body)); resp.StatusCode != 200 {
		t.Fatalf("PATCH de System.Tags = %d", resp.StatusCode)
	}
	if got, _ := srv.WorkItem(42); got.Fields["System.Tags"] != "a; axiom:c" || got.Rev != 4 {
		t.Fatalf("estado = %+v", got)
	}

	att := do(t, client, "GET", srv.URL()+"/org/p/_apis/wit/attachments/a1", "", "")
	data, _ := io.ReadAll(att.Body)
	if string(data) != "hola" || att.Header.Get("Content-Type") != "image/png" || srv.Count("GET", "/org/p/_apis/wit/attachments/a1") != 1 {
		t.Fatalf("adjunto = %q (%s), lecturas = %d", data, att.Header.Get("Content-Type"), srv.Count("GET", "/org/p/_apis/wit/attachments/a1"))
	}
}

func TestAdotestAttachmentNoContentLength(t *testing.T) {
	srv := NewServer(t, nil)
	srv.SeedAttachment("/o/p/_apis/wit/attachments/x", Blob{Data: []byte("abc")})
	resp := do(t, srv.HTTPClient(), "GET", srv.URL()+"/o/p/_apis/wit/attachments/x", "", "")
	if resp.ContentLength != -1 || resp.Header.Get("Content-Length") != "" {
		t.Fatalf("el adjunto no debe llevar Content-Length: %d %q", resp.ContentLength, resp.Header.Get("Content-Length"))
	}
	if data, _ := io.ReadAll(resp.Body); string(data) != "abc" {
		t.Fatalf("cuerpo = %q", data)
	}
}

func TestAdotestAttachmentNoContentType(t *testing.T) {
	srv := NewServer(t, nil)
	srv.SeedAttachment("/o/p/_apis/wit/attachments/x", Blob{Data: []byte("%PDF-1.7 x"), NoContentType: true})
	resp := do(t, srv.HTTPClient(), "GET", srv.URL()+"/o/p/_apis/wit/attachments/x", "", "")
	if got := resp.Header.Values("Content-Type"); len(got) != 0 {
		t.Fatalf("no debe haber Content-Type, hay %v", got)
	}
	srv.SeedAttachment("/o/p/_apis/wit/attachments/y", Blob{Data: []byte("zz")})
	resp = do(t, srv.HTTPClient(), "GET", srv.URL()+"/o/p/_apis/wit/attachments/y", "", "")
	if resp.Header.Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("por defecto application/octet-stream, hay %q", resp.Header.Get("Content-Type"))
	}
}

func TestAdotestRedirect(t *testing.T) {
	srv := NewServer(t, nil)
	srv.SeedAttachment("/o/p/_apis/wit/attachments/r", Blob{RedirectTo: "http://127.0.0.1:1/login"})
	srv.SeedAttachment("/o/p/_apis/wit/attachments/e", Blob{Status: 500, Data: []byte("boom")})
	resp := do(t, noRedirect(srv), "GET", srv.URL()+"/o/p/_apis/wit/attachments/r", "", "")
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "http://127.0.0.1:1/login" {
		t.Fatalf("redirección = %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if resp := do(t, srv.HTTPClient(), "GET", srv.URL()+"/o/p/_apis/wit/attachments/e", "", ""); resp.StatusCode != 500 {
		t.Fatalf("estado = %d", resp.StatusCode)
	}
}

func TestAdotestAddRelations(t *testing.T) {
	srv := NewServer(t, nil)
	srv.SeedWorkItem(5, 1, map[string]any{"System.Title": "t"})
	body, _ := json.Marshal([]map[string]any{
		{"op": "test", "path": "/rev", "value": 1},
		{"op": "add", "path": "/relations/-", "value": map[string]any{"rel": "Hyperlink", "url": "https://x/y"}},
	})
	if resp := do(t, srv.HTTPClient(), "PATCH", srv.URL()+"/o/p/_apis/wit/workitems/5", "application/json-patch+json", string(body)); resp.StatusCode != 200 {
		t.Fatalf("PATCH = %d", resp.StatusCode)
	}
	got, _ := srv.WorkItem(5)
	if len(got.Relations) != 1 || got.Relations[0]["rel"] != "Hyperlink" || got.Rev != 2 {
		t.Fatalf("estado = %+v", got)
	}
}

func TestAdotestStatesFailure(t *testing.T) {
	srv := NewServer(t, nil)
	srv.SeedStates("User Story", State{"New", "Proposed"}, State{"Closed", "Completed"})
	srv.FailStates("Bug", 503)
	client := srv.HTTPClient()

	resp := do(t, client, "GET", srv.URL()+"/o/p/_apis/wit/workitemtypes/User%20Story/states?api-version=7.1", "", "")
	var got struct {
		Value []struct{ Name, Category string }
	}
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if resp.StatusCode != 200 || len(got.Value) != 2 || got.Value[1].Category != "Completed" {
		t.Fatalf("estados = %d %+v", resp.StatusCode, got)
	}
	if resp := do(t, client, "GET", srv.URL()+"/o/p/_apis/wit/workitemtypes/Bug/states", "", ""); resp.StatusCode != 503 {
		t.Fatalf("el fallo programado debe responder 503: %d", resp.StatusCode)
	}
	if resp := do(t, client, "GET", srv.URL()+"/o/p/_apis/wit/workitemtypes/Task/states", "", ""); resp.StatusCode != 404 {
		t.Fatalf("un tipo sin sembrar responde 404: %d", resp.StatusCode)
	}
}

func TestAdotestEditAfterNextRead(t *testing.T) {
	srv := NewServer(t, nil)
	srv.SeedWorkItem(9, 1, map[string]any{"System.Tags": "a"})
	srv.EditAfterNextRead(9, map[string]any{"System.Tags": "a; otra"})
	client := srv.HTTPClient()
	for i := 0; i < 2; i++ {
		resp := do(t, client, "GET", srv.URL()+"/o/p/_apis/wit/workitems/9", "", "")
		var got struct {
			Rev    int
			Fields map[string]any
		}
		_ = json.NewDecoder(resp.Body).Decode(&got)
		want, wantRev := "a", 1
		if i == 1 {
			want, wantRev = "a; otra", 2
		}
		if got.Fields["System.Tags"] != want || got.Rev != wantRev {
			t.Fatalf("lectura %d = %+v", i, got)
		}
	}
}
