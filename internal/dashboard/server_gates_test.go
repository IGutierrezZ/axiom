package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/consentenvelope"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Sin t.Parallel en estas pruebas: montan proyectos en disco y comparten el estado del motor de ramas.

// fakeGatesGateway es la pasarela doble: cuenta las llamadas y devuelve lo configurado. Ninguna prueba
// llega a git ni hace push.
type fakeGatesGateway struct {
	mu                                  sync.Mutex
	stop                                cli.GateStop
	stopErr                             error
	outcome                             cli.GateRecordOutcome
	recordErr                           error
	published                           cli.ChangePublishResult
	publishErr                          error
	stopCalls, recordCalls, publishCall int
	lastRecord                          cli.GateRecordInput
	lastPublish                         cli.ChangePublishRequest
	stopCWD                             string
	enter                               chan struct{} // si no es nil, se cierra al entrar en Record o Publish
	release                             chan struct{} // si no es nil, Record y Publish esperan a que se cierre
}

func (f *fakeGatesGateway) Stop(_ context.Context, cwd, _ string) (cli.GateStop, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls++
	f.stopCWD = cwd
	return f.stop, f.stopErr
}

func (f *fakeGatesGateway) hold() {
	f.mu.Lock()
	enter, release := f.enter, f.release
	f.mu.Unlock()
	if enter != nil {
		close(enter)
	}
	if release != nil {
		<-release
	}
}

func (f *fakeGatesGateway) Record(_ context.Context, in cli.GateRecordInput) (cli.GateRecordOutcome, error) {
	f.mu.Lock()
	f.recordCalls++
	f.lastRecord = in
	outcome, err := f.outcome, f.recordErr
	f.mu.Unlock()
	f.hold()
	return outcome, err
}

func (f *fakeGatesGateway) Publish(_ context.Context, req cli.ChangePublishRequest) (cli.ChangePublishResult, error) {
	f.mu.Lock()
	f.publishCall++
	f.lastPublish = req
	result, err := f.published, f.publishErr
	f.mu.Unlock()
	f.hold()
	return result, err
}

func (f *fakeGatesGateway) counts() (stop, record, publish int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stopCalls, f.recordCalls, f.publishCall
}

func gateChoices(answers ...string) []consentenvelope.Choice {
	out := make([]consentenvelope.Choice, 0, len(answers))
	for _, answer := range answers {
		out = append(out, consentenvelope.Choice{Answer: answer, Label: "etiqueta " + answer, Effect: "efecto " + answer,
			Invocation: "axiom sdd gate record --decision " + answer})
	}
	return out
}

// gateStopFor es una parada `checkpointed` de la puerta dada con tres salidas.
func gateStopFor(root, gate string) cli.GateStop {
	return cli.GateStop{
		Change: "c", WorkspaceRoot: root, RecordRoot: root, ExecutionStyle: "checkpointed", ArtifactStore: "hybrid",
		NextRecommended: "await-gate", Digest: "sha256:visto",
		Question: &sddstatus.SDDGovernanceGateResult{
			Gate: gate, Headline: "titular", Reason: "motivo", Value: "valor",
			Criteria: []string{"criterio"}, Evidence: []string{"evidencia"},
			Choices:     gateChoices("rejected", "hand_off", "approved"),
			Publication: &sddstatus.GatePublication{Remote: "origin", Branch: "inc/c", Invocation: "axiom sdd publish --branch inc/c"},
		},
	}
}

type gatesEnv struct {
	t       *testing.T
	ws      workspacetest.Workspace
	svc     *Service
	handler http.Handler
	gw      *fakeGatesGateway
	root    string
}

// newGatesEnv monta un embebido con el cambio activo `c` y la pasarela doble con la parada de `spec`.
func newGatesEnv(t *testing.T) *gatesEnv {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "c", "proposal.md"), "# c\n")
	svc := NewServiceWithHub(ws.Master, nil)
	gw := &fakeGatesGateway{stop: gateStopFor(ws.Master, "spec")}
	svc.SetGatesGateway(gw)
	return &gatesEnv{t: t, ws: ws, svc: svc, handler: NewServer(svc).handler(), gw: gw, root: ws.Master}
}

type gateResp struct {
	Code int
	Body map[string]any
	Raw  string
}

func (e *gatesEnv) do(method, path, contentType string, body []byte, mutate ...func(*http.Request)) gateResp {
	e.t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Host = "127.0.0.1"
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	for _, m := range mutate {
		m(request)
	}
	recorder := httptest.NewRecorder()
	e.handler.ServeHTTP(recorder, request)
	out := gateResp{Code: recorder.Code, Raw: recorder.Body.String()}
	_ = json.Unmarshal(recorder.Body.Bytes(), &out.Body)
	return out
}

func (e *gatesEnv) post(path string, body any) gateResp {
	e.t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		e.t.Fatal(err)
	}
	return e.do(http.MethodPost, path, "application/json", data)
}

func (e *gatesEnv) get(path string) gateResp {
	e.t.Helper()
	return e.do(http.MethodGet, path, "", nil)
}

func (r gateResp) code() string { s, _ := r.Body["code"].(string); return s }

func (r gateResp) details() map[string]any { d, _ := r.Body["details"].(map[string]any); return d }

func (e *gatesEnv) wantRefusal(r gateResp, status int, code string) {
	e.t.Helper()
	if r.Code != status || r.code() != code {
		e.t.Fatalf("respuesta = %d %q, se esperaba %d %q: %s", r.Code, r.code(), status, code, r.Raw)
	}
	if cont, _ := r.Body["continuation"].(string); strings.TrimSpace(cont) == "" {
		e.t.Fatalf("el rechazo %s no lleva continuación: %s", code, r.Raw)
	}
}

func (e *gatesEnv) wantNoCalls() {
	e.t.Helper()
	if stop, record, publish := e.gw.counts(); stop+record+publish != 0 {
		e.t.Fatalf("la pasarela no debía llamarse: stop=%d record=%d publish=%d", stop, record, publish)
	}
}

func approveSpecBody() map[string]any {
	return map[string]any{"change": "c", "gate": "spec", "digest": "sha256:visto", "decision": "approved", "confirm": true}
}

func gatePublishBody() map[string]any {
	return map[string]any{"change": "c", "branch": "inc/c", "confirm": true}
}

var gatePOSTRoutes = []string{"/api/increments/gate", "/api/increments/gate/publish"}

// D11: el preflight compartido da los mismos rechazos en las dos rutas POST, sin llamar a la pasarela.
func TestGatePostRoutesPreflight(t *testing.T) {
	e := newGatesEnv(t)
	for _, path := range gatePOSTRoutes {
		if r := e.get(path); r.Code != http.StatusMethodNotAllowed {
			t.Errorf("GET %s = %d, se esperaba 405", path, r.Code)
		}
		foreign := e.do(http.MethodPost, path, "application/json", []byte(`{}`), func(r *http.Request) { r.Header.Set("Origin", "http://evil.example") })
		if foreign.Code != http.StatusForbidden || foreign.code() != "forbidden_origin" {
			t.Errorf("POST %s con Origin ajeno = %d %s", path, foreign.Code, foreign.Raw)
		}
		badHost := e.do(http.MethodPost, path, "application/json", []byte(`{}`), func(r *http.Request) { r.Host = "evil.example" })
		if badHost.Code != http.StatusForbidden || badHost.code() != "forbidden_origin" {
			t.Errorf("POST %s con Host ajeno = %d %s", path, badHost.Code, badHost.Raw)
		}
		e.wantRefusal(e.do(http.MethodPost, path, "text/plain", []byte(`{}`)), http.StatusUnsupportedMediaType, codeGateContentType)
		e.wantRefusal(e.do(http.MethodPost, path, "application/json", []byte(`{"change":"c","actor":"x"}`)), http.StatusBadRequest, codeGateBodyInvalid)
		e.wantRefusal(e.do(http.MethodPost, path, "application/json", []byte(`{"change":"c"}{"change":"c"}`)), http.StatusBadRequest, codeGateBodyInvalid)
		big := `{"change":"c","branch":"` + strings.Repeat("x", maxMarksBody) + `"}`
		e.wantRefusal(e.do(http.MethodPost, path, "application/json", []byte(big)), http.StatusBadRequest, codeGateBodyInvalid)
		for _, change := range []string{"", "..", "../x", "a/b", `a\b`, "/etc/passwd", "C:\\x"} {
			body := gateBodyFor(path)
			body["change"] = change
			e.wantRefusal(e.post(path, body), http.StatusBadRequest, codeGateBodyInvalid)
		}
		missing := gateBodyFor(path)
		delete(missing, "confirm")
		e.wantRefusal(e.post(path, missing), http.StatusUnprocessableEntity, codeGateConfirmMissing)
	}
	e.wantNoCalls()
}

func TestGateStopRouteRejectsPostAndBadChange(t *testing.T) {
	e := newGatesEnv(t)
	if r := e.do(http.MethodPost, "/api/increments/gate/stop?change=c", "application/json", []byte(`{}`)); r.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST stop = %d, se esperaba 405", r.Code)
	}
	for _, change := range []string{"", "..", "a%2Fb", "%2Fetc"} {
		e.wantRefusal(e.get("/api/increments/gate/stop?change="+change), http.StatusBadRequest, codeGateBodyInvalid)
	}
	e.wantNoCalls()
}

// Nota 2: `digest` es obligatorio al registrar; el 422 llega antes de leer nada.
func TestGateRecordRequiresDigest(t *testing.T) {
	e := newGatesEnv(t)
	body := approveSpecBody()
	delete(body, "digest")
	r := e.post("/api/increments/gate", body)
	e.wantRefusal(r, http.StatusUnprocessableEntity, cli.GateCodeArgsInvalid)
	e.wantNoCalls()
}

func TestGateRoutesWithoutGatewayAre503(t *testing.T) {
	e := newGatesEnv(t)
	e.svc.SetGatesGateway(nil)
	e.wantRefusal(e.post("/api/increments/gate", approveSpecBody()), http.StatusServiceUnavailable, codeGatesGatewayMissing)
}

// D8 paso 4: cambio inexistente, feature ODD y archivado; la pasarela no llega a consultarse.
func TestGateRoutesLocateTheChange(t *testing.T) {
	e := newGatesEnv(t)
	workspacetest.WriteFile(t, filepath.Join(e.ws.Specs, "odd", "tasks", "feat.md"), "# feat\n\n- [ ] 1.1 algo\n")
	workspacetest.WriteFile(t, filepath.Join(e.ws.Home, "changes", "archive", "2026-01-01-viejo", "proposal.md"), "# viejo\n")
	cases := map[string]struct {
		status int
		code   string
	}{
		"inexistente": {http.StatusNotFound, codeChangeNotFound},
		"feat":        {http.StatusConflict, codeGateNotSDD},
		"viejo":       {http.StatusConflict, codeGateChangeArchived},
	}
	for change, want := range cases {
		record := approveSpecBody()
		record["change"] = change
		e.wantRefusal(e.post("/api/increments/gate", record), want.status, want.code)
		publish := gatePublishBody()
		publish["change"] = change
		e.wantRefusal(e.post("/api/increments/gate/publish", publish), want.status, want.code)
	}
	e.wantNoCalls()
}

// D8 paso 5: el cerrojo es de la carpeta del cambio y lo comparten registrar y publicar.
func TestGateLockIsSharedByRecordAndPublish(t *testing.T) {
	cases := map[string]struct {
		first, second string
		firstBody     map[string]any
		secondBody    map[string]any
	}{
		"registrar y luego publicar": {"/api/increments/gate", "/api/increments/gate/publish", approveSpecBody(), gatePublishBody()},
		"publicar y luego registrar": {"/api/increments/gate/publish", "/api/increments/gate", gatePublishBody(), approveSpecBody()},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := newGatesEnv(t)
			e.gw.stop.Publication = &sddstatus.PublicationProjection{State: "pending", Remote: "origin", Branch: "inc/c"}
			e.gw.enter, e.gw.release = make(chan struct{}), make(chan struct{})
			done := make(chan gateResp, 1)
			go func() { done <- e.post(c.first, c.firstBody) }()
			select {
			case <-e.gw.enter:
			case <-time.After(5 * time.Second):
				t.Fatal("la primera petición no llegó a la pasarela")
			}
			e.wantRefusal(e.post(c.second, c.secondBody), http.StatusConflict, codeGateBusy)
			close(e.gw.release)
			if first := <-done; first.Code >= 400 {
				t.Fatalf("la primera petición debía completarse: %d %s", first.Code, first.Raw)
			}
			_, record, publish := e.gw.counts()
			if record+publish != 1 {
				t.Fatalf("solo la primera petición opera: record=%d publish=%d", record, publish)
			}
		})
	}
}

// D8 paso 6: la puerta pedida no es la proyectada, o no hay parada: 409 sin escribir.
func TestGateRecordNotPending(t *testing.T) {
	e := newGatesEnv(t)
	body := approveSpecBody()
	body["gate"] = "design"
	r := e.post("/api/increments/gate", body)
	e.wantRefusal(r, http.StatusConflict, codeGateNotPending)
	if d := r.details(); d["seenGate"] != "design" || d["projectedGate"] != "spec" {
		t.Fatalf("details = %v", d)
	}
	e.gw.stop.Question = nil
	r = e.post("/api/increments/gate", approveSpecBody())
	e.wantRefusal(r, http.StatusConflict, codeGateNotPending)
	if d := r.details(); d["seenGate"] != "spec" || d["projectedGate"] != "" {
		t.Fatalf("details sin parada = %v", d)
	}
	if _, record, _ := e.gw.counts(); record != 0 {
		t.Fatalf("no se debía registrar: %d", record)
	}
}

func TestGateRoutesStatusUnreadable(t *testing.T) {
	e := newGatesEnv(t)
	e.gw.stopErr = errors.New("status roto")
	e.wantRefusal(e.post("/api/increments/gate", approveSpecBody()), http.StatusConflict, codeStatusUnreadable)
	e.wantRefusal(e.post("/api/increments/gate/publish", gatePublishBody()), http.StatusConflict, codeStatusUnreadable)
	e.wantRefusal(e.get("/api/increments/gate/stop?change=c"), http.StatusConflict, codeStatusUnreadable)
	if _, record, publish := e.gw.counts(); record+publish != 0 {
		t.Fatal("con el status ilegible no se escribe ni se publica")
	}
}

// D11: cada negativa tipada del núcleo conserva su mensaje y se proyecta con su estado.
func TestGateRecordProjectsCoreRefusals(t *testing.T) {
	e := newGatesEnv(t)
	cases := []struct {
		code   string
		status int
	}{
		{cli.GateCodeArgsInvalid, http.StatusUnprocessableEntity},
		{cli.GateCodeChangeNotFound, http.StatusNotFound},
		{cli.GateCodeChangeArchived, http.StatusConflict},
		{cli.GateCodeNoStops, http.StatusConflict},
		{cli.GateCodeKickoffMissing, http.StatusConflict},
		{cli.GateCodeRoleNotInRoster, http.StatusConflict},
		{cli.GateCodeHandOffNA, http.StatusConflict},
		{cli.GateCodeArtifactMissing, http.StatusConflict},
		{cli.GateCodeAlreadyResolved, http.StatusConflict},
		{cli.GateCodeDigestChanged, http.StatusConflict},
		{cli.GateCodeEvidenceRefused, http.StatusConflict},
	}
	for _, c := range cases {
		e.gw.recordErr = &cli.GateRecordError{Code: c.code, Err: errors.New("mensaje de la CLI para " + c.code)}
		r := e.post("/api/increments/gate", approveSpecBody())
		e.wantRefusal(r, c.status, c.code)
		if got, _ := r.Body["error"].(string); got != "mensaje de la CLI para "+c.code {
			t.Errorf("%s: mensaje = %q", c.code, got)
		}
	}

	e.gw.recordErr = &branch.RefusalError{Refusal: branch.Refusal{Code: branch.CodeWrongBranch, Message: "rama equivocada",
		Continuation: "pon la rama correcta", Details: map[string]string{"repository": "r"}}}
	r := e.post("/api/increments/gate", approveSpecBody())
	e.wantRefusal(r, http.StatusConflict, string(branch.CodeWrongBranch))
	if cont, _ := r.Body["continuation"].(string); cont != "pon la rama correcta" || r.details()["repository"] != "r" {
		t.Errorf("negativa del motor de ramas = %s", r.Raw)
	}

	e.gw.recordErr = errors.New("disco lleno")
	e.wantRefusal(e.post("/api/increments/gate", approveSpecBody()), http.StatusInternalServerError, codeGateRecordFailed)
}

// Notas 2, 3 y 5: el digest viaja como ExpectedDigest, la evidencia se mapea campo a campo, el actor
// nunca viaja y un `commitReview` ausente llega vacío (equivale a notify).
func TestGateRecordMapsTheRequestToTheCore(t *testing.T) {
	e := newGatesEnv(t)
	e.gw.outcome = cli.GateRecordOutcome{Change: "c", Record: kickoff.GateRecord{Gate: kickoff.GateSpec, Decision: kickoff.DecisionApproved},
		Notices: []cli.GateNotice{{Source: "branch_guard", Text: "aviso"}}}
	r := e.post("/api/increments/gate", approveSpecBody())
	if r.Code != http.StatusCreated {
		t.Fatalf("registro = %d %s", r.Code, r.Raw)
	}
	in := e.gw.lastRecord
	if in.CWD != e.root || in.Change != "c" || in.Gate != kickoff.GateSpec || in.Decision != kickoff.DecisionApproved ||
		in.ExpectedDigest != "sha256:visto" || in.Actor != "" || in.CommitReview != "" || in.HandOff {
		t.Fatalf("orden al núcleo = %+v", in)
	}
	record, _ := r.Body["record"].(map[string]any)
	if record["schema"] != "axiom.sdd-gate-record/v1" || record["gate"] != "spec" || record["decision"] != "approved" {
		t.Fatalf("record = %v", record)
	}
	if notices, _ := r.Body["notices"].([]any); len(notices) != 1 {
		t.Fatalf("notices = %v", r.Body["notices"])
	}
	if strings.Contains(strings.ToLower(r.Raw), "actor") {
		t.Fatalf("la respuesta no expone el actor: %s", r.Raw)
	}

	e.gw.stop = gateStopFor(e.root, "integration")
	integration := map[string]any{"change": "c", "gate": "integration", "digest": "sha256:visto", "decision": "approved", "confirm": true,
		"evidence": map[string]any{"kind": "pr_merged", "commit": "abc123", "baseRef": "main", "text": "texto"}}
	if r := e.post("/api/increments/gate", integration); r.Code != http.StatusCreated {
		t.Fatalf("integration = %d %s", r.Code, r.Raw)
	}
	in = e.gw.lastRecord
	if in.EvidenceKind != kickoff.EvidencePRMerged || in.Commit != "abc123" || in.BaseRef != "main" || in.Evidence != "texto" {
		t.Fatalf("evidencia mapeada = %+v", in)
	}

	// Nota 1: evidencia malformada (`pr_merged` sin commit) se rechaza como la CLI, sin llamar.
	_, before, _ := e.gw.counts()
	integration["evidence"] = map[string]any{"kind": "pr_merged"}
	r = e.post("/api/increments/gate", integration)
	e.wantRefusal(r, http.StatusUnprocessableEntity, cli.GateCodeArgsInvalid)
	if got, _ := r.Body["error"].(string); !strings.Contains(got, "pr_merged requiere --commit") {
		t.Errorf("el mensaje debe ser el de la CLI: %q", got)
	}
	if _, after, _ := e.gw.counts(); after != before {
		t.Fatal("la evidencia malformada no debe llegar al núcleo")
	}
}

// Nota 5: `commitReview` con valor válido llega al núcleo y uno fuera de los tres se rechaza 422.
func TestGateRecordCommitReviewMode(t *testing.T) {
	e := newGatesEnv(t)
	e.gw.stop = gateStopFor(e.root, "tasks")
	body := map[string]any{"change": "c", "gate": "tasks", "digest": "sha256:visto", "decision": "approved", "confirm": true, "commitReview": "stop"}
	if r := e.post("/api/increments/gate", body); r.Code != http.StatusCreated {
		t.Fatalf("registro = %d %s", r.Code, r.Raw)
	}
	if e.gw.lastRecord.CommitReview != kickoff.CommitReviewStop {
		t.Fatalf("commitReview = %q", e.gw.lastRecord.CommitReview)
	}
	_, before, _ := e.gw.counts()
	body["commitReview"] = "siempre"
	e.wantRefusal(e.post("/api/increments/gate", body), http.StatusUnprocessableEntity, cli.GateCodeArgsInvalid)
	if _, after, _ := e.gw.counts(); after != before {
		t.Fatal("un modo desconocido no llega al núcleo")
	}
}

func pendingPublication() *sddstatus.PublicationProjection {
	return &sddstatus.PublicationProjection{State: "pending", Remote: "origin", Branch: "inc/c", Invocation: "axiom sdd publish --branch inc/c"}
}

// D8 y nota 6: la publicación solo exige `checkpointed` y `publication.state == pending`.
func TestGatePublishPreconditions(t *testing.T) {
	e := newGatesEnv(t)

	e.gw.stop.ExecutionStyle = "continuous"
	e.wantRefusal(e.post("/api/increments/gate/publish", gatePublishBody()), http.StatusConflict, codeGateNotCheckpointed)

	e.gw.stop.ExecutionStyle, e.gw.stop.ArtifactStore = "checkpointed", "engram"
	r := e.post("/api/increments/gate/publish", gatePublishBody())
	e.wantRefusal(r, http.StatusConflict, codePublishUnavailable)
	if r.details()["store"] != "engram" {
		t.Fatalf("details = %v", r.details())
	}

	e.gw.stop.ArtifactStore = "hybrid"
	e.gw.stop.Publication = nil
	r = e.post("/api/increments/gate/publish", gatePublishBody())
	if r.Code != http.StatusOK || r.Body["status"] != "nothing_to_publish" {
		t.Fatalf("sin publication = %d %s", r.Code, r.Raw)
	}
	if r2 := e.post("/api/increments/gate/publish", gatePublishBody()); r2.Code != http.StatusOK || r2.Body["status"] != "nothing_to_publish" {
		t.Fatalf("el segundo clic es idempotente: %d %s", r2.Code, r2.Raw)
	}

	e.gw.stop.Publication = &sddstatus.PublicationProjection{State: "code_unpublished", PushCommand: "git push origin inc/c", CodeCommits: []string{"abc1234 feat: x"}}
	r = e.post("/api/increments/gate/publish", gatePublishBody())
	e.wantRefusal(r, http.StatusConflict, codePublicationCodeUnpub)
	if r.details()["pushCommand"] != "git push origin inc/c" {
		t.Fatalf("details = %v", r.details())
	}

	e.gw.stop.Publication = &sddstatus.PublicationProjection{State: "unavailable", Unavailable: "wrong_branch"}
	r = e.post("/api/increments/gate/publish", gatePublishBody())
	e.wantRefusal(r, http.StatusConflict, codePublicationNA)
	if r.details()["unavailable"] != "wrong_branch" {
		t.Fatalf("details = %v", r.details())
	}

	e.gw.stop.Publication = pendingPublication()
	body := gatePublishBody()
	body["branch"] = "otra"
	r = e.post("/api/increments/gate/publish", body)
	e.wantRefusal(r, http.StatusConflict, codePublicationBranch)
	if d := r.details(); d["requested"] != "otra" || d["expected"] != "inc/c" {
		t.Fatalf("details = %v", d)
	}
	if _, _, publish := e.gw.counts(); publish != 0 {
		t.Fatalf("ninguna precondición fallida publica: %d", publish)
	}
}

// Nota 6: sin exigir que el último registro sea una aprobación; con `pending` publica la rama atada.
func TestGatePublishSucceedsWhateverTheLastDecision(t *testing.T) {
	for name, last := range map[string]*kickoff.GateRecord{
		"sin registros": nil,
		"rechazo":       {Gate: kickoff.GateSpec, Decision: kickoff.DecisionRejected},
		"aprobación":    {Gate: kickoff.GateSpec, Decision: kickoff.DecisionApproved},
	} {
		t.Run(name, func(t *testing.T) {
			e := newGatesEnv(t)
			e.gw.stop.Question, e.gw.stop.Publication, e.gw.stop.LastDecision = nil, pendingPublication(), last
			e.gw.published = cli.ChangePublishResult{Status: "published", Remote: "origin", Branch: "inc/c", Commit: "abc1234", PushedSHA: "abc1234",
				Resume: []string{"git fetch origin", "axiom branch ensure --change c --json"}, Notices: []cli.GateNotice{{Source: "branch_guard", Text: "aviso"}}}
			r := e.post("/api/increments/gate/publish", gatePublishBody())
			if r.Code != http.StatusOK || r.Body["status"] != "published" || r.Body["commit"] != "abc1234" || r.Body["pushed"] != "abc1234" {
				t.Fatalf("publicación = %d %s", r.Code, r.Raw)
			}
			if resume, _ := r.Body["resume"].([]any); len(resume) != 2 {
				t.Fatalf("resume = %v", r.Body["resume"])
			}
			req := e.gw.lastPublish
			if req.CWD != e.root || req.Change != "c" || req.Branch != "inc/c" {
				t.Fatalf("orden de publicación = %+v", req)
			}
		})
	}
}

// D8: una negativa de sddpublish responde 409 con su código, etiqueta, mensaje, continuación y detalles.
func TestGatePublishProjectsPublishRefusals(t *testing.T) {
	e := newGatesEnv(t)
	e.gw.stop.Publication = pendingPublication()
	e.gw.published = cli.ChangePublishResult{Status: "refused", Refusal: &cli.ChangePublishRefusal{
		Code: "non_fast_forward", Label: "remoto adelantado", Message: "el remoto rechaza el push", Continuation: "trae lo publicado y repite",
		Details: map[string]any{"remote": "origin"}}}
	r := e.post("/api/increments/gate/publish", gatePublishBody())
	e.wantRefusal(r, http.StatusConflict, "non_fast_forward")
	if r.Body["label"] != "remoto adelantado" || r.details()["remote"] != "origin" {
		t.Fatalf("negativa = %s", r.Raw)
	}

	e.gw.published = cli.ChangePublishResult{}
	e.gw.publishErr = &branch.RefusalError{Refusal: branch.Refusal{Code: branch.CodeWrongBranch, Message: "rama", Continuation: "cámbiala"}}
	e.wantRefusal(e.post("/api/increments/gate/publish", gatePublishBody()), http.StatusConflict, string(branch.CodeWrongBranch))
	e.gw.publishErr = errors.New("sin red")
	e.wantRefusal(e.post("/api/increments/gate/publish", gatePublishBody()), http.StatusInternalServerError, codeGatePublishFailed)
}

// D9 y D11: el GET devuelve la parada con sus opciones en orden, o su ausencia con el motivo, y
// siempre la publicación y el último registro sin el actor.
func TestGateStopResponse(t *testing.T) {
	e := newGatesEnv(t)
	e.gw.stop.Publication = pendingPublication()
	e.gw.stop.LastDecision = &kickoff.GateRecord{Gate: kickoff.GateDesign, Decision: kickoff.DecisionApproved, Actor: "Ana <ana@example.org>",
		HandedOff: true, HandedOffTo: "tasks", RecordedAt: time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)}
	r := e.get("/api/increments/gate/stop?change=c")
	if r.Code != http.StatusOK || r.Body["state"] != "await_gate" || r.Body["executionStyle"] != "checkpointed" {
		t.Fatalf("GET = %d %s", r.Code, r.Raw)
	}
	stop, _ := r.Body["stop"].(map[string]any)
	choices, _ := stop["choices"].([]any)
	if stop["gate"] != "spec" || stop["digest"] != "sha256:visto" || len(choices) != 3 {
		t.Fatalf("stop = %v", stop)
	}
	for i, answer := range []string{"rejected", "hand_off", "approved"} {
		if choice, _ := choices[i].(map[string]any); choice["answer"] != answer || choice["effect"] != "efecto "+answer {
			t.Errorf("choices[%d] = %v", i, choice)
		}
	}
	if commitReview, _ := stop["commitReview"].(map[string]any); commitReview["offered"] != false || commitReview["default"] != "notify" {
		t.Errorf("commitReview de spec = %v", stop["commitReview"])
	}
	if publication, _ := r.Body["publication"].(map[string]any); publication["state"] != "pending" || publication["branch"] != "inc/c" {
		t.Errorf("publication = %v", r.Body["publication"])
	}
	last, _ := r.Body["lastDecision"].(map[string]any)
	if last["gate"] != "design" || last["handedOffTo"] != "tasks" || last["recordedAt"] != "2026-11-27T10:00:00Z" {
		t.Errorf("lastDecision = %v", last)
	}
	if strings.Contains(r.Raw, "ana@example.org") {
		t.Errorf("el GET no expone el actor: %s", r.Raw)
	}
	if stopCalls, _, _ := e.gw.counts(); stopCalls != 1 || e.gw.stopCWD != e.root {
		t.Errorf("Stop llamado %d veces con cwd %q", stopCalls, e.gw.stopCWD)
	}

	e.gw.stop = gateStopFor(e.root, "tasks")
	r = e.get("/api/increments/gate/stop?change=c")
	stop, _ = r.Body["stop"].(map[string]any)
	if commitReview, _ := stop["commitReview"].(map[string]any); commitReview["offered"] != true {
		t.Errorf("tasks ofrece el modo de revisión: %v", stop["commitReview"])
	}
}

func TestGateStopAbsenceReasons(t *testing.T) {
	e := newGatesEnv(t)
	workspacetest.WriteFile(t, filepath.Join(e.ws.Specs, "odd", "tasks", "feat.md"), "# feat\n\n- [ ] 1.1 algo\n")
	workspacetest.WriteFile(t, filepath.Join(e.ws.Home, "changes", "archive", "2026-01-01-viejo", "proposal.md"), "# viejo\n")

	cases := []struct {
		name, change, reason string
		stop                 cli.GateStop
	}{
		{"continuo", "c", "continuous", cli.GateStop{ExecutionStyle: "continuous"}},
		{"sin sellar", "c", "not_sealed", cli.GateStop{}},
		{"bloqueado", "c", "blocked", cli.GateStop{ExecutionStyle: "checkpointed", NextRecommended: "blocked", BlockedReasons: []string{"falta algo"}}},
		{"sin puerta", "c", "no_gate", cli.GateStop{ExecutionStyle: "checkpointed", NextRecommended: "apply"}},
		{"ODD", "feat", "odd", cli.GateStop{}},
		{"archivado", "viejo", "archived", cli.GateStop{}},
	}
	for _, c := range cases {
		e.gw.stop = c.stop
		r := e.get("/api/increments/gate/stop?change=" + c.change)
		if r.Code != http.StatusOK || r.Body["state"] != "none" || r.Body["reason"] != c.reason {
			t.Errorf("%s: %d %s", c.name, r.Code, r.Raw)
		}
		if _, ok := r.Body["stop"]; ok {
			t.Errorf("%s: sin parada no hay objeto stop", c.name)
		}
	}
	if r := e.get("/api/increments/gate/stop?change=inexistente"); r.Code != http.StatusNotFound || r.code() != codeChangeNotFound {
		t.Errorf("inexistente: %d %s", r.Code, r.Raw)
	}
	e.svc.SetGatesGateway(nil)
	e.wantRefusal(e.get("/api/increments/gate/stop?change=c"), http.StatusServiceUnavailable, codeGatesGatewayMissing)
}

// gateBodyFor es un cuerpo válido de la ruta POST dada.
func gateBodyFor(path string) map[string]any {
	if path == "/api/increments/gate/publish" {
		return gatePublishBody()
	}
	return approveSpecBody()
}

// RF-INT-16: las rutas de puertas no devuelven el secreto de integración ni cuando el despacho del
// evento de aprobación falla con el valor del token en su mensaje. La pasarela es la real, sobre un
// cambio sellado `checkpointed`.
func TestGateRoutesNeverEchoTheSecret(t *testing.T) {
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"),
		"workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: .\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\nintegrations:\n  fake:\n    project: p\n")
	changeRoot := filepath.Join(ws.Home, "changes", "c")
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "proposal.md"), "# Propuesta\n")
	workspacetest.WriteFile(t, filepath.Join(changeRoot, "spec.md"), "# Spec\n\nContenido real.\n")
	if err := cli.RunSDDKickoff([]string{"seal", "--cwd", ws.Master, "--change", "c", "--execution-style", "checkpointed", "--handoff-policy", "none"}, io.Discard); err != nil {
		t.Fatalf("sellar el cambio: %v", err)
	}

	fake := integrationstest.NewFake().LeakTokenInError()
	memory := integrationstest.Install(t, fake)
	if err := memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(dashboardCanary)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_INTEGRATION_FAKE_TOKEN", dashboardCanary)

	svc := NewServiceWithHub(ws.Master, nil)
	svc.SetGatesGateway(cli.NewGatesGateway())
	e := &gatesEnv{t: t, ws: ws, svc: svc, handler: NewServer(svc).handler(), root: ws.Master}

	stop := e.get("/api/increments/gate/stop?change=c")
	digest, _ := stop.Body["stop"].(map[string]any)["digest"].(string)
	if stop.Code != http.StatusOK || digest == "" {
		t.Fatalf("GET de la parada = %d %s", stop.Code, stop.Raw)
	}
	responses := map[string]gateResp{
		"GET parada":       stop,
		"POST sin confirm": e.post("/api/increments/gate", map[string]any{"change": "c", "gate": "spec", "digest": digest, "decision": "approved"}),
		"POST con digest":  e.post("/api/increments/gate", map[string]any{"change": "c", "gate": "spec", "digest": "viejo", "decision": "approved", "confirm": true}),
		"POST aprobación":  e.post("/api/increments/gate", map[string]any{"change": "c", "gate": "spec", "digest": digest, "decision": "approved", "confirm": true}),
		"POST publicar":    e.post("/api/increments/gate/publish", map[string]any{"change": "c", "branch": "inc/c", "confirm": true}),
		"POST campo ajeno": e.post("/api/increments/gate", map[string]any{"change": "c", "secret": dashboardCanary}),
		"GET tras aprobar": e.get("/api/increments/gate/stop?change=c"),
		"POST host ajeno":  e.do(http.MethodPost, "/api/increments/gate", "application/json", []byte(`{}`), func(r *http.Request) { r.Host = "evil.example" }),
	}
	if recorded := responses["POST aprobación"]; recorded.Code != http.StatusCreated {
		t.Fatalf("la aprobación debía registrarse: %d %s", recorded.Code, recorded.Raw)
	}
	if fake.Calls() == 0 {
		t.Fatal("la aprobación debía despachar el evento de integración que falla")
	}
	for label, r := range responses {
		if strings.Contains(r.Raw, dashboardCanary) {
			t.Errorf("%s devuelve el valor del secreto de integración: %s", label, r.Raw)
		}
	}
}
