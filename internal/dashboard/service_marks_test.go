package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests/marks"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Sin t.Parallel en estas pruebas: montan proyectos git y servidores httptest.

const marksAxiomYAML = `workspace:
  name: ejemplo-canonico
  topology: multirepo
  specs_repository: repo-specs
roles:
  backend:
    name: Backend
    repositories:
      - path: repo-backend
acceptance_tests:
  enabled: true
  environments: [dev, pre]
`

func marksRow(id, envs string) string {
	return "| " + id + " | REQ-1.1#Alfa | fullstack | manual | alta | " + envs + " | ninguna | abrir | ok |\n"
}

// fakeMarksPublisher es el MarksPublisher falso: cuenta las llamadas y devuelve lo configurado.
type fakeMarksPublisher struct {
	mu            sync.Mutex
	actor         string
	missing       []string
	result        cli.MarksPublishResult
	identityCalls int
	publishCalls  int
	lastRequest   cli.MarksPublishRequest
	enter         chan struct{} // si no es nil, se cierra al entrar en PublishMarks
	release       chan struct{} // si no es nil, PublishMarks espera a que se cierre
}

func (f *fakeMarksPublisher) MarksIdentity(_ context.Context, _ string) (string, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.identityCalls++
	return f.actor, f.missing
}

func (f *fakeMarksPublisher) PublishMarks(_ context.Context, req cli.MarksPublishRequest) cli.MarksPublishResult {
	f.mu.Lock()
	f.publishCalls++
	f.lastRequest = req
	enter, release, result := f.enter, f.release, f.result
	f.mu.Unlock()
	if enter != nil {
		close(enter)
	}
	if release != nil {
		<-release
	}
	return result
}

func (f *fakeMarksPublisher) calls() (identity, publish int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.identityCalls, f.publishCalls
}

type marksEnv struct {
	t    *testing.T
	ws   workspacetest.Workspace
	svc  *Service
	srv  *httptest.Server
	fake *fakeMarksPublisher
	root string // carpeta del cambio `c` en el repo de specs in situ
}

// newMarksEnv monta un multirepo con la batería activa (entornos dev y pre), el cambio `c` con
// dos casos y un publicador falso inyectado.
func newMarksEnv(t *testing.T) *marksEnv {
	t.Helper()
	ws := workspacetest.Canonical(t, workspacetest.WithGit())
	workspacetest.WriteFile(t, filepath.Join(ws.Specs, "axiom.yaml"), marksAxiomYAML)
	root := filepath.Join(ws.Home, "changes", "c")
	workspacetest.WriteFile(t, filepath.Join(root, "proposal.md"), "# c\n")
	workspacetest.WriteFile(t, filepath.Join(root, "tasks.md"), "- [ ] T1 algo\n")
	workspacetest.WriteFile(t, filepath.Join(root, "specs", "dom", "spec.md"), atDelta)
	workspacetest.WriteFile(t, filepath.Join(root, "pruebas.md"), atHeader+marksRow("P-01", "dev; pre")+marksRow("P-02", "dev"))
	svc := NewServiceWithHub(ws.Master, nil)
	fake := &fakeMarksPublisher{actor: "Ana Pérez <ana@example.org>",
		result: cli.MarksPublishResult{Status: "published", Remote: "origin", Branch: "main", Commit: "abc1234", PushedSHA: "abc1234"}}
	svc.SetMarksPublisher(fake)
	srv := httptest.NewServer(NewServer(svc).Router())
	t.Cleanup(srv.Close)
	return &marksEnv{t: t, ws: ws, svc: svc, srv: srv, fake: fake, root: root}
}

func (e *marksEnv) post(path string, body any) (*http.Response, map[string]any) {
	e.t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		e.t.Fatal(err)
	}
	return e.postRaw(path, "application/json", data)
}

func (e *marksEnv) postRaw(path, contentType string, data []byte) (*http.Response, map[string]any) {
	e.t.Helper()
	req, err := http.NewRequest(http.MethodPost, e.srv.URL+path, bytes.NewReader(data))
	if err != nil {
		e.t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func (e *marksEnv) mark(change, caseID, env, result, note string) (*http.Response, map[string]any) {
	return e.post("/api/increments/marks", map[string]any{"change": change, "case": caseID, "environment": env, "result": result, "note": note})
}

func (e *marksEnv) marksFile() string {
	data, err := os.ReadFile(filepath.Join(e.root, marks.FileName))
	if err != nil && !os.IsNotExist(err) {
		e.t.Fatal(err)
	}
	return string(data)
}

func (e *marksEnv) detailMarks() *AcceptanceMarksDTO {
	e.t.Helper()
	dto, err := e.svc.GetIncrementDetail("c")
	if err != nil {
		e.t.Fatal(err)
	}
	if dto.AcceptanceTests == nil || dto.AcceptanceTests.Marks == nil {
		e.t.Fatalf("falta el DTO de marcas: %+v", dto.AcceptanceTests)
	}
	return dto.AcceptanceTests.Marks
}

func wantMarksRefusal(t *testing.T, label string, resp *http.Response, body map[string]any, status int, code string) {
	t.Helper()
	if resp.StatusCode != status || body["code"] != code {
		t.Fatalf("%s: %d %v, quería %d %s", label, resp.StatusCode, body, status, code)
	}
	if s, _ := body["error"].(string); s == "" {
		t.Errorf("%s: falta el mensaje: %v", label, body)
	}
	if s, _ := body["continuation"].(string); s == "" {
		t.Errorf("%s: falta la continuación: %v", label, body)
	}
}
