package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/identitystore"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/secrets"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// Estas pruebas comparten estado de proceso (catálogo de manifiestos, almacén de secretos por
// defecto, AXIOM_STATE_DIR): ninguna llama a t.Parallel. Usan handler() (la cadena de seguridad
// completa) con Host 127.0.0.1 y un doble de IntegrationsGateway: nunca llaman a un tracker real.

const (
	apiSentinel = "PAT-CENTINELA-7d41c9e2"
	apiEnvVar   = "AXIOM_INTEGRATION_FAKE_TOKEN"
)

// fakeGateway es el doble del puerto: cada método delega en una función sustituible y cuenta las
// llamadas.
type fakeGateway struct {
	connect  func(ctx context.Context, cwd, plugin, user string) (cli.IntegrationConnectOutcome, error)
	discover func(ctx context.Context, cwd, plugin string) (integrations.Discovery, error)
	applic   cli.WorkItemApplicability

	// Alta desde un work item (6.1): cada paso delega en una función sustituible y cuenta las llamadas.
	startWI func(ctx context.Context, cwd, ref string) (cli.WorkItemStartOutcome, error)
	staged  func(cwd, change string) (cli.WorkItemStagedOutcome, error)
	adopt   func(ctx context.Context, cwd, change string) (cli.WorkItemAdoptOutcome, error)
	discard func(cwd, change string) (cli.WorkItemDiscardOutcome, error)

	starts, stageds, adopts, discards atomic.Int32
	startCwd, adoptCwd                atomic.Value

	connects, discovers atomic.Int32
	lastCwd             atomic.Value
}

func (g *fakeGateway) Connect(ctx context.Context, cwd, plugin, user string) (cli.IntegrationConnectOutcome, error) {
	g.connects.Add(1)
	g.lastCwd.Store(cwd)
	if g.connect != nil {
		return g.connect(ctx, cwd, plugin, user)
	}
	return cli.IntegrationConnectOutcome{
		Identity:   integrations.TrackerIdentity{Account: "contoso", ID: "guid-1", DisplayName: "Ana Pérez", Source: "connection_data"},
		ResolvedAt: "2026-07-01T10:00:00Z",
	}, nil
}

func (g *fakeGateway) Discover(ctx context.Context, cwd, plugin string) (integrations.Discovery, error) {
	g.discovers.Add(1)
	g.lastCwd.Store(cwd)
	if g.discover != nil {
		return g.discover(ctx, cwd, plugin)
	}
	return integrations.Discovery{
		Schema: integrations.DiscoverySchemaV1, Plugin: plugin, Scope: map[string]string{"project": "p"},
		ItemTypes: []integrations.ItemType{{Name: "Task", ReferenceName: "System.Task"}},
	}, nil
}

func (g *fakeGateway) WorkItemApplicability(string) cli.WorkItemApplicability { return g.applic }

// failingStore es un almacén persistente que falla siempre con un error que incluye el valor.
type failingStore struct{ err func(value string) error }

func (failingStore) Get(context.Context, secrets.Ref) (secrets.Secret, secrets.Source, error) {
	return secrets.Secret{}, secrets.SourceMissing, secrets.ErrNotFound
}

func (f failingStore) Set(_ context.Context, _ secrets.Ref, v secrets.Secret) error {
	return f.err(v.Reveal())
}
func (failingStore) Delete(context.Context, secrets.Ref) error {
	return errors.New("borrado imposible")
}

// brokenGetStore falla en Get con una causa distinta de «no existe» (almacén no disponible).
type brokenGetStore struct{}

func (brokenGetStore) Get(context.Context, secrets.Ref) (secrets.Secret, secrets.Source, error) {
	return secrets.Secret{}, secrets.SourceKeyringUnavailable, errors.New("sin servicio de claves")
}
func (brokenGetStore) Set(context.Context, secrets.Ref, secrets.Secret) error { return nil }
func (brokenGetStore) Delete(context.Context, secrets.Ref) error              { return nil }

type apiEnv struct {
	ws     workspacetest.Workspace
	svc    *Service
	srv    *Server
	h      http.Handler
	gw     *fakeGateway
	memory *secrets.Memory
	state  string
}

const apiConfig = "workspace:\n  name: ejemplo\n  topology: monorepo-embedded\n  specs_repository: .\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\nintegrations:\n  fake:\n    project: p\n"

func newAPIEnv(t *testing.T) *apiEnv {
	t.Helper()
	ws := workspacetest.Embedded(t, ".")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), apiConfig)
	state := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", state)
	t.Setenv(apiEnvVar, "")
	memory := integrationstest.Install(t, integrationstest.NewFake())
	svc := NewService(ws.Master)
	gw := &fakeGateway{applic: cli.WorkItemApplicability{Applicable: true, Plugin: "fake", RefAlias: "fk"}}
	svc.SetIntegrationsGateway(gw)
	srv := NewServer(svc)
	return &apiEnv{ws: ws, svc: svc, srv: srv, h: srv.handler(), gw: gw, memory: memory, state: state}
}

type apiResp struct {
	*httptest.ResponseRecorder
}

func (r apiResp) body() string { return r.Body.String() }

func (r apiResp) json(t *testing.T) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &out); err != nil {
		t.Fatalf("la respuesta no es JSON: %v\n%s", err, r.body())
	}
	return out
}

func (r apiResp) code(t *testing.T) string {
	t.Helper()
	v, _ := r.json(t)["code"].(string)
	return v
}

// request lanza una petición local (Host 127.0.0.1) con cuerpo JSON en los POST.
func (e *apiEnv) request(method, target, body string, mods ...func(*http.Request)) apiResp {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Host = "127.0.0.1"
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, m := range mods {
		m(req)
	}
	rr := httptest.NewRecorder()
	e.h.ServeHTTP(rr, req)
	return apiResp{rr}
}

func (e *apiEnv) post(target, body string) apiResp { return e.request(http.MethodPost, target, body) }

func (e *apiEnv) saveSecret(value string) apiResp {
	return e.post("/api/integrations/secret", fmt.Sprintf(`{"plugin":"fake","name":"token","value":%q}`, value))
}

func (e *apiEnv) gatewayCalls() int { return int(e.gw.connects.Load() + e.gw.discovers.Load()) }

func assertNoStore(t *testing.T, label string, r apiResp) {
	t.Helper()
	if got := r.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("%s: Cache-Control = %q, se esperaba no-store", label, got)
	}
}

func assertNoSentinel(t *testing.T, label string, r apiResp) {
	t.Helper()
	if strings.Contains(r.body(), apiSentinel) {
		t.Errorf("%s: el cuerpo contiene el centinela del PAT: %s", label, r.body())
	}
	for key, values := range r.Header() {
		for _, v := range values {
			if strings.Contains(v, apiSentinel) {
				t.Errorf("%s: la cabecera %s contiene el centinela", label, key)
			}
		}
	}
}

// ------------------------------------------------------------------ GET /api/integrations

func TestIntegrationsAPIStatusServesManifestEntriesSecretsIdentityAndWorkItems(t *testing.T) {
	e := newAPIEnv(t)
	if err := e.memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(apiSentinel)); err != nil {
		t.Fatal(err)
	}
	path, err := identitystore.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	if err := (identitystore.Store{Path: path}).Put("fake", "Contoso", identitystore.Entry{ID: "guid-1", DisplayName: "Ana Pérez", Source: "member_selection", ResolvedAt: when}); err != nil {
		t.Fatal(err)
	}

	r := e.request(http.MethodGet, "/api/integrations", "")

	if r.Code != http.StatusOK {
		t.Fatalf("estado = %d: %s", r.Code, r.body())
	}
	assertNoStore(t, "GET", r)
	assertNoSentinel(t, "GET", r)
	var dto IntegrationsDTO
	if err := json.Unmarshal(r.Body.Bytes(), &dto); err != nil {
		t.Fatal(err)
	}
	if dto.ConfigPath == "" || dto.Revision == "" || !dto.Writable || dto.DocumentErrors == nil {
		t.Errorf("cabecera del DTO = %+v", dto)
	}
	if len(dto.Plugins) != 1 || dto.Plugins[0].Manifest.ID != "fake" {
		t.Fatalf("plugins = %+v", dto.Plugins)
	}
	p := dto.Plugins[0]
	if !p.Declared || !p.Enabled {
		t.Errorf("declared/enabled = %v/%v", p.Declared, p.Enabled)
	}
	keys := map[string]any{}
	for _, entry := range p.Entries {
		if entry.Plugin != "fake" {
			t.Errorf("entrada de otro plugin: %+v", entry)
		}
		keys[entry.Key] = entry.Value
	}
	if keys["integrations[fake].project"] != "p" {
		t.Errorf("entradas = %v", keys)
	}
	if len(p.Secrets) != 1 || p.Secrets[0].Name != "token" || p.Secrets[0].Source != "keyring" || p.Secrets[0].EnvVar != apiEnvVar || !p.Secrets[0].Required {
		t.Errorf("secretos = %+v", p.Secrets)
	}
	if len(p.Identities) != 1 || p.Identities[0].ID != "guid-1" || p.Identities[0].Account != "contoso" || p.Identities[0].DisplayName != "Ana Pérez" {
		t.Errorf("identidades = %+v", p.Identities)
	}
	if !dto.WorkItems.Applicable || dto.WorkItems.Plugin != "fake" || dto.WorkItems.RefAlias != "fk" {
		t.Errorf("workItems = %+v", dto.WorkItems)
	}
	if strings.Contains(r.body(), `"value":"`+apiSentinel) {
		t.Error("el estado no puede llevar el valor del secreto")
	}
}

func TestIntegrationsAPIStatusDegradesWithoutGateway(t *testing.T) {
	e := newAPIEnv(t)
	e.svc.SetIntegrationsGateway(nil)

	r := e.request(http.MethodGet, "/api/integrations", "")

	if r.Code != http.StatusOK {
		t.Fatalf("sin pasarela debe degradarse, no responder %d: %s", r.Code, r.body())
	}
	var dto IntegrationsDTO
	if err := json.Unmarshal(r.Body.Bytes(), &dto); err != nil {
		t.Fatal(err)
	}
	if len(dto.Plugins) != 1 || len(dto.Plugins[0].Secrets) != 1 || len(dto.Plugins[0].Entries) == 0 {
		t.Errorf("el manifiesto, los valores y el estado del secreto se sirven igualmente: %+v", dto.Plugins)
	}
	if dto.WorkItems.Applicable || dto.WorkItems.Reason == "" {
		t.Errorf("workItems debe ser no aplicable con motivo: %+v", dto.WorkItems)
	}
}

func TestIntegrationsAPIStatusSecretSources(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		e := newAPIEnv(t)
		if got := statusSource(t, e); got != "missing" {
			t.Errorf("source = %q", got)
		}
	})
	t.Run("env tapa el almacén", func(t *testing.T) {
		e := newAPIEnv(t)
		_ = e.memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret("guardado"))
		t.Setenv(apiEnvVar, apiSentinel)
		r := e.request(http.MethodGet, "/api/integrations", "")
		assertNoSentinel(t, "GET", r)
		if got := statusSource(t, e); got != "env" {
			t.Errorf("source = %q", got)
		}
	})
	t.Run("almacén no disponible", func(t *testing.T) {
		e := newAPIEnv(t)
		secrets.SetDefaultForTest(t, secrets.NewChain(secrets.Env{}, brokenGetStore{}))
		if got := statusSource(t, e); got != "keyring_unavailable" {
			t.Errorf("source = %q", got)
		}
	})
}

func statusSource(t *testing.T, e *apiEnv) string {
	t.Helper()
	r := e.request(http.MethodGet, "/api/integrations", "")
	var dto IntegrationsDTO
	if err := json.Unmarshal(r.Body.Bytes(), &dto); err != nil || len(dto.Plugins) != 1 || len(dto.Plugins[0].Secrets) != 1 {
		t.Fatalf("respuesta inesperada (%v): %s", err, r.body())
	}
	return dto.Plugins[0].Secrets[0].Source
}

func TestIntegrationsAPIStatusReportsAnUnreadableIdentityFile(t *testing.T) {
	e := newAPIEnv(t)
	path, _ := identitystore.DefaultPath()
	if err := os.WriteFile(path, []byte("{no es json"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := e.request(http.MethodGet, "/api/integrations", "")

	var dto IntegrationsDTO
	if err := json.Unmarshal(r.Body.Bytes(), &dto); err != nil {
		t.Fatal(err)
	}
	if r.Code != http.StatusOK || dto.Plugins[0].IdentityError == "" || len(dto.Plugins[0].Identities) != 0 {
		t.Errorf("estado %d, identityError=%q identidades=%v", r.Code, dto.Plugins[0].IdentityError, dto.Plugins[0].Identities)
	}
}

func TestIntegrationsAPIStatusWithoutConfigIsNotFound(t *testing.T) {
	e := newAPIEnv(t)
	if err := os.Remove(filepath.Join(e.ws.Master, "axiom.yaml")); err != nil {
		t.Fatal(err)
	}

	r := e.request(http.MethodGet, "/api/integrations", "")

	if r.Code != http.StatusNotFound || r.code(t) != "config_missing" {
		t.Errorf("estado %d código %q: %s", r.Code, r.code(t), r.body())
	}
	assertNoStore(t, "404", r)
}

func TestIntegrationsAPIStatusRejectsOtherMethods(t *testing.T) {
	e := newAPIEnv(t)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		if r := e.request(method, "/api/integrations", `{}`); r.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: estado %d", method, r.Code)
		}
	}
}

func TestIntegrationsAPIRejectsForeignHostAndOriginWithoutTouchingAnything(t *testing.T) {
	e := newAPIEnv(t)
	routes := []string{"/api/integrations/secret", "/api/integrations/secret/delete", "/api/integrations/connect", "/api/integrations/discover"}
	foreign := map[string]func(*http.Request){
		"host ajeno":   func(r *http.Request) { r.Host = "evil.example" },
		"origin ajeno": func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") },
		"origin null":  func(r *http.Request) { r.Header.Set("Origin", "null") },
	}
	for name, mod := range foreign {
		for _, route := range routes {
			r := e.request(http.MethodPost, route, fmt.Sprintf(`{"plugin":"fake","name":"token","value":%q,"confirm":true}`, apiSentinel), mod)
			if r.Code != http.StatusForbidden || r.code(t) != "forbidden_origin" {
				t.Errorf("%s %s: estado %d: %s", name, route, r.Code, r.body())
			}
			assertNoStore(t, name+" "+route, r)
		}
	}
	if e.gatewayCalls() != 0 {
		t.Error("una petición rechazada no puede llamar a la pasarela")
	}
	if _, _, err := e.memory.Get(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}); err == nil {
		t.Error("una petición rechazada no puede guardar el PAT")
	}
}

// ------------------------------------------------------------------ secreto

func TestIntegrationsAPISecretSavesAndRespondsOnlyTheSource(t *testing.T) {
	e := newAPIEnv(t)

	r := e.saveSecret(apiSentinel)

	if r.Code != http.StatusOK {
		t.Fatalf("estado %d: %s", r.Code, r.body())
	}
	assertNoSentinel(t, "guardar", r)
	assertNoStore(t, "guardar", r)
	var resp IntegrationSecretResponse
	if err := json.Unmarshal(r.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Plugin != "fake" || resp.Name != "token" || resp.Source != "keyring" || resp.EnvVar != apiEnvVar || resp.ShadowedByEnv {
		t.Errorf("respuesta = %+v", resp)
	}
	got, _, err := e.memory.Get(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"})
	if err != nil || got.Reveal() != apiSentinel {
		t.Errorf("el almacén debe tener el PAT: %v", err)
	}
	yaml, _ := os.ReadFile(filepath.Join(e.ws.Master, "axiom.yaml"))
	if strings.Contains(string(yaml), apiSentinel) {
		t.Error("el PAT no puede llegar a axiom.yaml")
	}
}

func TestIntegrationsAPISecretTrimsSurroundingBlanks(t *testing.T) {
	e := newAPIEnv(t)
	if r := e.saveSecret("  " + apiSentinel + "  "); r.Code != http.StatusOK {
		t.Fatalf("estado %d: %s", r.Code, r.body())
	}
	got, _, _ := e.memory.Get(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"})
	if got.Reveal() != apiSentinel {
		t.Errorf("el PAT se guarda sin espacios laterales")
	}
}

func TestIntegrationsAPISecretRejectsInvalidRequestsWithoutTouchingTheStore(t *testing.T) {
	e := newAPIEnv(t)
	tooBig := strings.Repeat("a", secrets.MaxValueBytes+1)
	huge := fmt.Sprintf(`{"plugin":"fake","name":"token","value":%q}`, strings.Repeat("b", 9<<10))
	tests := []struct {
		name, target, body, contentType string
		wantStatus                      int
		wantCode                        string
	}{
		{"no JSON (text/plain)", "/api/integrations/secret", `{"plugin":"fake","name":"token","value":"x"}`, "text/plain", 400, "invalid_request"},
		{"sin tipo de contenido", "/api/integrations/secret", `{"plugin":"fake","name":"token","value":"x"}`, "-", 400, "invalid_request"},
		{"JSON roto", "/api/integrations/secret", `{"plugin":`, "", 400, "invalid_request"},
		{"cuerpo vacío", "/api/integrations/secret", ``, "", 400, "invalid_request"},
		{"campo desconocido", "/api/integrations/secret", `{"plugin":"fake","name":"token","value":"x","extra":1}`, "", 400, "invalid_request"},
		{"valor ausente", "/api/integrations/secret", `{"plugin":"fake","name":"token"}`, "", 400, "invalid_request"},
		{"valor vacío", "/api/integrations/secret", `{"plugin":"fake","name":"token","value":""}`, "", 400, "invalid_request"},
		{"valor solo blancos", "/api/integrations/secret", `{"plugin":"fake","name":"token","value":"   "}`, "", 400, "invalid_request"},
		{"salto de línea", "/api/integrations/secret", `{"plugin":"fake","name":"token","value":"ab\ncd"}`, "", 400, "invalid_request"},
		{"carácter de control", "/api/integrations/secret", `{"plugin":"fake","name":"token","value":"ab\u0001cd"}`, "", 400, "invalid_request"},
		{"DEL", "/api/integrations/secret", `{"plugin":"fake","name":"token","value":"ab\u007fcd"}`, "", 400, "invalid_request"},
		{"demasiado largo", "/api/integrations/secret", fmt.Sprintf(`{"plugin":"fake","name":"token","value":%q}`, tooBig), "", 400, "value_too_large"},
		{"cuerpo por encima del límite", "/api/integrations/secret", huge, "", 400, "invalid_request"},
		{"query rechazada", "/api/integrations/secret?value=x", `{"plugin":"fake","name":"token","value":"x"}`, "", 400, "invalid_request"},
		{"plugin desconocido", "/api/integrations/secret", `{"plugin":"jira","name":"token","value":"x"}`, "", 404, "unknown_plugin"},
		{"secreto desconocido", "/api/integrations/secret", `{"plugin":"fake","name":"otro","value":"x"}`, "", 422, "unknown_secret"},
		{"borrado sin confirm", "/api/integrations/secret/delete", `{"plugin":"fake","name":"token"}`, "", 400, "invalid_request"},
		{"borrado confirm falso", "/api/integrations/secret/delete", `{"plugin":"fake","name":"token","confirm":false}`, "", 400, "invalid_request"},
		{"borrado plugin desconocido", "/api/integrations/secret/delete", `{"plugin":"jira","name":"token","confirm":true}`, "", 404, "unknown_plugin"},
		{"borrado secreto desconocido", "/api/integrations/secret/delete", `{"plugin":"fake","name":"otro","confirm":true}`, "", 422, "unknown_secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := e.request(http.MethodPost, tt.target, tt.body, func(req *http.Request) {
				switch tt.contentType {
				case "":
				case "-":
					req.Header.Del("Content-Type")
				default:
					req.Header.Set("Content-Type", tt.contentType)
				}
			})
			if r.Code != tt.wantStatus || r.code(t) != tt.wantCode {
				t.Fatalf("estado %d código %q, se esperaba %d %q: %s", r.Code, r.code(t), tt.wantStatus, tt.wantCode, r.body())
			}
			assertNoStore(t, tt.name, r)
			if msg, _ := r.json(t)["error"].(string); strings.TrimSpace(msg) == "" {
				t.Error("el error debe llevar un mensaje en castellano")
			}
		})
	}
	if _, _, err := e.memory.Get(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}); err == nil {
		t.Error("ninguna petición inválida puede guardar el PAT")
	}
}

func TestIntegrationsAPISecretRejectsOtherMethods(t *testing.T) {
	e := newAPIEnv(t)
	for _, route := range []string{"/api/integrations/secret", "/api/integrations/secret/delete", "/api/integrations/connect", "/api/integrations/discover"} {
		for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
			if r := e.request(method, route, ""); r.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s: estado %d", method, route, r.Code)
			}
		}
	}
}

func TestIntegrationsAPISecretValueTooLargeFromTheStoreGuidesToTheEnvVar(t *testing.T) {
	e := newAPIEnv(t)
	secrets.SetDefaultForTest(t, failingStoreChain(func(string) error { return nil }))
	big := apiSentinel + strings.Repeat("x", secrets.MaxValueBytes)

	r := e.saveSecret(big)

	if r.Code != http.StatusBadRequest || r.code(t) != "value_too_large" {
		t.Fatalf("estado %d código %q: %s", r.Code, r.code(t), r.body())
	}
	assertNoSentinel(t, "valor largo", r)
	if env, _ := r.json(t)["envVar"].(string); env != apiEnvVar {
		t.Errorf("envVar = %q", env)
	}
	if msg, _ := r.json(t)["error"].(string); !strings.Contains(msg, apiEnvVar) {
		t.Errorf("el mensaje debe guiar a la variable de entorno: %q", msg)
	}
}

func failingStoreChain(err func(string) error) secrets.WritableStore {
	return secrets.NewChain(secrets.Env{}, failingStore{err: err})
}

func TestIntegrationsAPISecretWithoutWritableStoreGuidesToTheEnvVar(t *testing.T) {
	e := newAPIEnv(t)
	secrets.SetDefaultForTest(t, secrets.NewChain(secrets.Env{}, nil))

	r := e.saveSecret(apiSentinel)

	if r.Code != http.StatusUnprocessableEntity || r.code(t) != "no_writable_store" {
		t.Fatalf("estado %d código %q: %s", r.Code, r.code(t), r.body())
	}
	assertNoSentinel(t, "sin almacén", r)
	body := r.json(t)
	if body["envVar"] != apiEnvVar || !strings.Contains(body["error"].(string), apiEnvVar) {
		t.Errorf("la guía debe nombrar %s: %v", apiEnvVar, body)
	}
}

func TestIntegrationsAPISecretShadowedByEnv(t *testing.T) {
	e := newAPIEnv(t)
	t.Setenv(apiEnvVar, "valor-de-entorno")

	r := e.saveSecret(apiSentinel)

	if r.Code != http.StatusOK {
		t.Fatalf("estado %d: %s", r.Code, r.body())
	}
	var resp IntegrationSecretResponse
	_ = json.Unmarshal(r.Body.Bytes(), &resp)
	if resp.Source != "env" || !resp.ShadowedByEnv {
		t.Errorf("la variable tapa el almacén: %+v", resp)
	}
	assertNoSentinel(t, "env", r)
}

func TestIntegrationsAPISecretStoreFailureIsTypedAndRedacted(t *testing.T) {
	e := newAPIEnv(t)
	secrets.SetDefaultForTest(t, failingStoreChain(func(value string) error {
		return fmt.Errorf("el almacén rechazó %q", value)
	}))

	r := e.saveSecret(apiSentinel)

	if r.Code != http.StatusInternalServerError || r.code(t) != "secret_store_failed" {
		t.Fatalf("estado %d código %q: %s", r.Code, r.code(t), r.body())
	}
	assertNoSentinel(t, "fallo del almacén", r)
	if !strings.Contains(r.body(), "[redactado]") {
		t.Errorf("el valor debe sustituirse por [redactado]: %s", r.body())
	}
	if env, _ := r.json(t)["envVar"].(string); env != apiEnvVar {
		t.Errorf("el fallo guía a la variable de entorno: %q", env)
	}
}

func TestIntegrationsAPISecretDeleteIsConfirmedAndIdempotent(t *testing.T) {
	e := newAPIEnv(t)
	ref := secrets.Ref{Plugin: "fake", Name: "token"}
	_ = e.memory.Set(context.Background(), ref, secrets.NewSecret(apiSentinel))
	body := `{"plugin":"fake","name":"token","confirm":true}`

	for attempt := 1; attempt <= 2; attempt++ {
		r := e.post("/api/integrations/secret/delete", body)
		if r.Code != http.StatusOK {
			t.Fatalf("intento %d: estado %d: %s", attempt, r.Code, r.body())
		}
		var resp IntegrationSecretResponse
		_ = json.Unmarshal(r.Body.Bytes(), &resp)
		if resp.Source != "missing" || resp.ShadowedByEnv {
			t.Errorf("intento %d: %+v", attempt, resp)
		}
		assertNoSentinel(t, "borrar", r)
	}
	if _, _, err := e.memory.Get(context.Background(), ref); err == nil {
		t.Error("el PAT debe haberse borrado")
	}

	t.Setenv(apiEnvVar, "de-entorno")
	r := e.post("/api/integrations/secret/delete", body)
	var resp IntegrationSecretResponse
	_ = json.Unmarshal(r.Body.Bytes(), &resp)
	if r.Code != http.StatusOK || resp.Source != "env" || !resp.ShadowedByEnv {
		t.Errorf("con la variable definida el origen sigue siendo env: %d %+v", r.Code, resp)
	}
}

func TestIntegrationsAPISecretDeleteFailuresAreTyped(t *testing.T) {
	e := newAPIEnv(t)
	body := `{"plugin":"fake","name":"token","confirm":true}`

	secrets.SetDefaultForTest(t, secrets.NewChain(secrets.Env{}, nil))
	if r := e.post("/api/integrations/secret/delete", body); r.Code != 422 || r.code(t) != "no_writable_store" {
		t.Errorf("sin almacén: estado %d código %q", r.Code, r.code(t))
	}
	secrets.SetDefaultForTest(t, failingStoreChain(nil))
	if r := e.post("/api/integrations/secret/delete", body); r.Code != 500 || r.code(t) != "secret_store_failed" {
		t.Errorf("almacén roto: estado %d código %q", r.Code, r.code(t))
	}
}

// ------------------------------------------------------------------ connect

func TestIntegrationsAPIConnectReturnsTheResolvedIdentity(t *testing.T) {
	e := newAPIEnv(t)
	_ = e.memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(apiSentinel))
	e.gw.connect = func(_ context.Context, _, _, user string) (cli.IntegrationConnectOutcome, error) {
		if user != "guid-7" {
			t.Errorf("la elección se reenvía como user: %q", user)
		}
		return cli.IntegrationConnectOutcome{
			Identity:   integrations.TrackerIdentity{Account: "contoso", ID: "guid-7", DisplayName: "Eva", Source: "member_selection"},
			ResolvedAt: "2026-07-01T10:00:00Z",
			Replaced:   &identitystore.Entry{ID: "guid-1", DisplayName: "Ana", Source: "connection_data", ResolvedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
		}, nil
	}

	r := e.post("/api/integrations/connect", `{"plugin":"fake","user":"guid-7"}`)

	if r.Code != http.StatusOK {
		t.Fatalf("estado %d: %s", r.Code, r.body())
	}
	assertNoStore(t, "connect", r)
	var resp IntegrationConnectResponse
	_ = json.Unmarshal(r.Body.Bytes(), &resp)
	if resp.Identity.ID != "guid-7" || resp.Identity.Account != "contoso" || resp.ResolvedAt == "" || resp.Replaced == nil || resp.Replaced.ID != "guid-1" {
		t.Errorf("respuesta = %+v", resp)
	}
	if cwd, _ := e.gw.lastCwd.Load().(string); cwd != e.ws.Master {
		t.Errorf("la pasarela recibe la raíz del proyecto activo: %q", cwd)
	}
}

func TestIntegrationsAPIConnectSelectionRequiredListsCandidates(t *testing.T) {
	e := newAPIEnv(t)
	_ = e.memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(apiSentinel))
	members := []integrations.Member{{ID: "a", DisplayName: "Ana"}, {ID: "b", DisplayName: "Beto"}}
	e.gw.connect = func(_ context.Context, _, _, user string) (cli.IntegrationConnectOutcome, error) {
		return cli.IntegrationConnectOutcome{}, fmt.Errorf("envuelto: %w", &integrations.IdentitySelectionError{Plugin: "fake", Account: "contoso", User: user, Candidates: members})
	}

	r := e.post("/api/integrations/connect", `{"plugin":"fake"}`)
	if r.Code != http.StatusConflict || r.code(t) != "identity_selection_required" {
		t.Fatalf("estado %d código %q: %s", r.Code, r.code(t), r.body())
	}
	if c, _ := r.json(t)["candidates"].([]any); len(c) != 2 {
		t.Errorf("candidatos = %v", r.json(t)["candidates"])
	}

	r = e.post("/api/integrations/connect", `{"plugin":"fake","user":"forjado"}`)
	if r.Code != http.StatusUnprocessableEntity || r.code(t) != "identity_user_unknown" {
		t.Fatalf("estado %d código %q: %s", r.Code, r.code(t), r.body())
	}
	if c, _ := r.json(t)["candidates"].([]any); len(c) != 2 {
		t.Errorf("candidatos = %v", r.json(t)["candidates"])
	}
}

func TestIntegrationsAPIConnectAndDiscoverWithoutSecretNeverCallTheGateway(t *testing.T) {
	e := newAPIEnv(t)
	for _, route := range []string{"/api/integrations/connect", "/api/integrations/discover"} {
		r := e.post(route, `{"plugin":"fake"}`)
		if r.Code != http.StatusUnprocessableEntity || r.code(t) != "secret_missing" {
			t.Fatalf("%s: estado %d código %q: %s", route, r.Code, r.code(t), r.body())
		}
		if r.json(t)["envVar"] != apiEnvVar || !strings.Contains(r.json(t)["error"].(string), apiEnvVar) {
			t.Errorf("%s: debe guiar a guardar el PAT o definir la variable: %v", route, r.json(t))
		}
	}
	if e.gatewayCalls() != 0 {
		t.Error("sin PAT no se llama al tracker")
	}
}

func TestIntegrationsAPIConnectAndDiscoverRejectBadRequests(t *testing.T) {
	e := newAPIEnv(t)
	_ = e.memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(apiSentinel))
	for _, route := range []string{"/api/integrations/connect", "/api/integrations/discover"} {
		cases := []struct {
			name, body, query string
			status            int
			code              string
		}{
			{"plugin desconocido", `{"plugin":"jira"}`, "", 404, "unknown_plugin"},
			{"plugin vacío", `{"plugin":""}`, "", 404, "unknown_plugin"},
			{"JSON roto", `{`, "", 400, "invalid_request"},
			{"campo desconocido", `{"plugin":"fake","pat":"x"}`, "", 400, "invalid_request"},
			{"query", `{"plugin":"fake"}`, "?x=1", 400, "invalid_request"},
		}
		for _, c := range cases {
			r := e.post(route+c.query, c.body)
			if r.Code != c.status || r.code(t) != c.code {
				t.Errorf("%s %s: estado %d código %q: %s", route, c.name, r.Code, r.code(t), r.body())
			}
		}
	}
	if e.gatewayCalls() != 0 {
		t.Error("una petición inválida no llega a la pasarela")
	}
}

func TestIntegrationsAPIConnectWithoutGatewayIs503(t *testing.T) {
	e := newAPIEnv(t)
	e.svc.SetIntegrationsGateway(nil)
	for _, route := range []string{"/api/integrations/connect", "/api/integrations/discover"} {
		r := e.post(route, `{"plugin":"fake"}`)
		if r.Code != http.StatusServiceUnavailable || r.code(t) != "integrations_gateway_missing" {
			t.Errorf("%s: estado %d código %q", route, r.Code, r.code(t))
		}
	}
}

func TestIntegrationsAPIGatewayErrorsMapToTypedCodes(t *testing.T) {
	e := newAPIEnv(t)
	_ = e.memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(apiSentinel))
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"plugin desconocido", fmt.Errorf("x: %w", cli.ErrIntegrationPluginUnknown), 404, "unknown_plugin"},
		{"no configurado", fmt.Errorf("x: %w", cli.ErrIntegrationNotConfigured), 422, "plugin_not_configured"},
		{"desactivado", fmt.Errorf("x: %w", cli.ErrIntegrationDisabled), 422, "plugin_disabled"},
		{"configuración inválida", fmt.Errorf("x: %w", cli.ErrIntegrationConfigInvalid), 422, "config_invalid"},
		{"no compilado", fmt.Errorf("x: %w", cli.ErrIntegrationNotRegistered), 422, "plugin_not_compiled"},
		{"puerto no ofrecido", &integrations.PortUnsupportedError{Plugin: "fake", Port: integrations.PortIdentity}, 422, "port_unsupported"},
		{"secreto ausente", &secrets.MissingError{Ref: secrets.Ref{Plugin: "fake", Name: "token"}}, 422, "secret_missing"},
		{"fallo del plugin", &integrations.PortError{Plugin: "fake", Port: integrations.PortIdentity, Message: "credencial rechazada por el tracker"}, 502, "integration_failed"},
		{"fichero de identidad", &identitystore.CorruptError{Path: "x", Cause: errors.New("roto")}, 500, "identity_store_failed"},
		{"otro error", errors.New("algo inesperado"), 502, "integration_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e.gw.connect = func(context.Context, string, string, string) (cli.IntegrationConnectOutcome, error) {
				return cli.IntegrationConnectOutcome{}, tt.err
			}
			r := e.post("/api/integrations/connect", `{"plugin":"fake"}`)
			if r.Code != tt.status || r.code(t) != tt.code {
				t.Errorf("connect: estado %d código %q: %s", r.Code, r.code(t), r.body())
			}
			e.gw.discover = func(context.Context, string, string) (integrations.Discovery, error) {
				return integrations.Discovery{}, tt.err
			}
			wantDiscover := tt.code
			if tt.code == "identity_store_failed" {
				wantDiscover = "integration_failed"
			}
			r = e.post("/api/integrations/discover", `{"plugin":"fake"}`)
			if r.Code == 0 || r.code(t) != wantDiscover {
				t.Errorf("discover: estado %d código %q, se esperaba %q: %s", r.Code, r.code(t), wantDiscover, r.body())
			}
		})
	}
}

func TestIntegrationsAPIDiscoverReturnsTheDiscoveryWithoutWritingConfig(t *testing.T) {
	e := newAPIEnv(t)
	_ = e.memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(apiSentinel))
	before, _ := os.ReadFile(filepath.Join(e.ws.Master, "axiom.yaml"))

	r := e.post("/api/integrations/discover", `{"plugin":"fake"}`)

	if r.Code != http.StatusOK {
		t.Fatalf("estado %d: %s", r.Code, r.body())
	}
	assertNoStore(t, "discover", r)
	var d integrations.Discovery
	if err := json.Unmarshal(r.Body.Bytes(), &d); err != nil || d.Plugin != "fake" || len(d.ItemTypes) != 1 {
		t.Errorf("descubrimiento = %+v (%v)", d, err)
	}
	after, _ := os.ReadFile(filepath.Join(e.ws.Master, "axiom.yaml"))
	if !bytes.Equal(before, after) {
		t.Error("descubrir no escribe axiom.yaml")
	}
}

func TestIntegrationsAPITimeoutAnswers504BeforeTheWriteDeadline(t *testing.T) {
	e := newAPIEnv(t)
	_ = e.memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(apiSentinel))
	e.srv.integrationBudget = 40 * time.Millisecond
	block := func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
	e.gw.connect = func(ctx context.Context, _, _, _ string) (cli.IntegrationConnectOutcome, error) {
		return cli.IntegrationConnectOutcome{}, block(ctx)
	}
	e.gw.discover = func(ctx context.Context, _, _ string) (integrations.Discovery, error) {
		return integrations.Discovery{}, block(ctx)
	}

	for _, route := range []string{"/api/integrations/connect", "/api/integrations/discover"} {
		started := time.Now()
		r := e.post(route, `{"plugin":"fake"}`)
		if r.Code != http.StatusGatewayTimeout || r.code(t) != "integration_timeout" {
			t.Errorf("%s: estado %d código %q: %s", route, r.Code, r.code(t), r.body())
		}
		if elapsed := time.Since(started); elapsed > 2*time.Second {
			t.Errorf("%s: tardó %v", route, elapsed)
		}
	}
	// El cerrojo se libera tras el vencimiento: una nueva petición vuelve a pasar.
	e.gw.connect = nil
	if r := e.post("/api/integrations/connect", `{"plugin":"fake"}`); r.Code != http.StatusOK {
		t.Errorf("tras el vencimiento el cerrojo debe estar libre: estado %d", r.Code)
	}
}

func TestIntegrationsAPIOverlappingCallsAnswer409(t *testing.T) {
	e := newAPIEnv(t)
	_ = e.memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(apiSentinel))
	entered, release := make(chan struct{}), make(chan struct{})
	e.gw.connect = func(context.Context, string, string, string) (cli.IntegrationConnectOutcome, error) {
		close(entered)
		<-release
		return cli.IntegrationConnectOutcome{Identity: integrations.TrackerIdentity{Account: "c", ID: "1"}}, nil
	}
	var wg sync.WaitGroup
	wg.Add(1)
	var first apiResp
	go func() {
		defer wg.Done()
		first = e.post("/api/integrations/connect", `{"plugin":"fake"}`)
	}()
	<-entered

	second := e.post("/api/integrations/discover", `{"plugin":"fake"}`)
	close(release)
	wg.Wait()

	if second.Code != http.StatusConflict || second.code(t) != "integration_busy" {
		t.Errorf("la segunda petición solapada: estado %d código %q", second.Code, second.code(t))
	}
	if first.Code != http.StatusOK {
		t.Errorf("la primera termina bien: %d", first.Code)
	}
}

// ------------------------------------------------------------------ canario

// El centinela del PAT vive en el almacén, en el entorno del doble y en los errores del doble de la
// pasarela. Ninguna respuesta (de éxito o de error), cabecera ni línea de log puede contenerlo.
func TestIntegrationsAPINeverEchoesTheSecret(t *testing.T) {
	e := newAPIEnv(t)
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previous) })

	_ = e.memory.Set(context.Background(), secrets.Ref{Plugin: "fake", Name: "token"}, secrets.NewSecret(apiSentinel))
	leak := &integrations.PortError{Plugin: "fake", Port: integrations.PortIdentity, Message: "el tracker rechazó el token " + apiSentinel}
	e.gw.connect = func(context.Context, string, string, string) (cli.IntegrationConnectOutcome, error) {
		return cli.IntegrationConnectOutcome{}, leak
	}
	e.gw.discover = func(context.Context, string, string) (integrations.Discovery, error) {
		return integrations.Discovery{}, errors.New("fallo con " + apiSentinel)
	}

	responses := map[string]apiResp{
		"GET estado":      e.request(http.MethodGet, "/api/integrations", ""),
		"POST connect":    e.post("/api/integrations/connect", `{"plugin":"fake"}`),
		"POST discover":   e.post("/api/integrations/discover", `{"plugin":"fake"}`),
		"POST guardar":    e.saveSecret(apiSentinel),
		"POST borrar":     e.post("/api/integrations/secret/delete", `{"plugin":"fake","name":"token","confirm":true}`),
		"POST inválido":   e.post("/api/integrations/secret", fmt.Sprintf(`{"plugin":"fake","name":"token","value":%q,"extra":1}`, apiSentinel)),
		"POST con query":  e.post("/api/integrations/secret?value="+apiSentinel, fmt.Sprintf(`{"plugin":"fake","name":"token","value":%q}`, apiSentinel)),
		"POST host ajeno": e.request(http.MethodPost, "/api/integrations/secret", fmt.Sprintf(`{"value":%q}`, apiSentinel), func(r *http.Request) { r.Host = "evil.example" }),
		"POST largo":      e.saveSecret(apiSentinel + strings.Repeat("z", secrets.MaxValueBytes)),
	}
	secrets.SetDefaultForTest(t, failingStoreChain(func(v string) error { return errors.New("rechazado " + v) }))
	responses["POST fallo del almacén"] = e.saveSecret(apiSentinel)
	secrets.SetDefaultForTest(t, secrets.NewChain(secrets.Env{}, nil))
	responses["POST sin almacén"] = e.saveSecret(apiSentinel)

	for label, r := range responses {
		assertNoSentinel(t, label, r)
		assertNoStore(t, label, r)
	}
	if strings.Contains(logs.String(), apiSentinel) {
		t.Errorf("el log del servidor contiene el centinela: %s", logs.String())
	}
	if got := responses["POST connect"]; got.Code != http.StatusBadGateway || !strings.Contains(got.body(), "[redactado]") {
		t.Errorf("el error del plugin debe salir redactado: %d %s", got.Code, got.body())
	}
}

func (g *fakeGateway) StartWorkItem(ctx context.Context, cwd, ref string) (cli.WorkItemStartOutcome, error) {
	g.starts.Add(1)
	g.startCwd.Store(cwd)
	if g.startWI != nil {
		return g.startWI(ctx, cwd, ref)
	}
	return cli.WorkItemStartOutcome{Change: "us-12-alta", ExternalRef: "fk:12", Status: "staged"}, nil
}

func (g *fakeGateway) WorkItemStaged(cwd, change string) (cli.WorkItemStagedOutcome, error) {
	g.stageds.Add(1)
	if g.staged != nil {
		return g.staged(cwd, change)
	}
	return cli.WorkItemStagedOutcome{Change: change, ExternalRef: "fk:12"}, nil
}

func (g *fakeGateway) AdoptWorkItem(ctx context.Context, cwd, change string) (cli.WorkItemAdoptOutcome, error) {
	g.adopts.Add(1)
	g.adoptCwd.Store(cwd)
	if g.adopt != nil {
		return g.adopt(ctx, cwd, change)
	}
	return cli.WorkItemAdoptOutcome{Change: change, Status: "adopted"}, nil
}

func (g *fakeGateway) DiscardWorkItem(cwd, change string) (cli.WorkItemDiscardOutcome, error) {
	g.discards.Add(1)
	if g.discard != nil {
		return g.discard(cwd, change)
	}
	return cli.WorkItemDiscardOutcome{Change: change, Status: "discarded"}, nil
}
