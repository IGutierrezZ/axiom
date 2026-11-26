package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/integrations/integrationstest"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

const settingsProjectYAML = `# Proyecto de pruebas de Ajustes.
workspace:
  name: "Demo"
  topology: "monorepo-embedded"
  specs_repository: "openspec"

workflow:
  lane: "ask"

roles:
  fullstack:
    name: "Fullstack"
    repositories:
      - path: "."

`

type settingsJSON struct {
	ConfigPath         string   `json:"configPath"`
	Revision           string   `json:"revision"`
	Topology           string   `json:"topology"`
	ISO                bool     `json:"iso"`
	Writable           bool     `json:"writable"`
	WriteBlockedReason string   `json:"writeBlockedReason"`
	DocumentErrors     []string `json:"documentErrors"`
	Entries            []struct {
		Key      string   `json:"key"`
		Editable bool     `json:"editable"`
		Allowed  []string `json:"allowed"`
		Declared bool     `json:"declared"`
		Value    any      `json:"value"`
	} `json:"entries"`
}

type settingsPutJSON struct {
	Settings settingsJSON `json:"settings"`
	Changed  bool         `json:"changed"`
	Warnings []string     `json:"warnings"`
}

type settingsErrJSON struct {
	Code       string   `json:"code"`
	Error      string   `json:"error"`
	Violations []string `json:"violations"`
}

// newSettingsProject crea un proyecto embebido con openspec/ y el axiom.yaml indicado.
func newSettingsProject(t *testing.T, yaml string) string {
	t.Helper()
	ws := workspacetest.Embedded(t, "openspec")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), yaml)
	return ws.Master
}

func getSettings(t *testing.T, router http.Handler) settingsJSON {
	t.Helper()
	rr := request(router, http.MethodGet, "/api/settings", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/settings = %d: %s", rr.Code, rr.Body.String())
	}
	var dto settingsJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &dto); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	return dto
}

func putSettings(router http.Handler, revision string, changes string) *httptest.ResponseRecorder {
	return request(router, http.MethodPut, "/api/settings", `{"revision":"`+revision+`","changes":`+changes+`}`)
}

func (s settingsJSON) value(key string) (any, bool) {
	for _, e := range s.Entries {
		if e.Key == key {
			return e.Value, true
		}
	}
	return nil, false
}

// RF-WEB-18: el GET no devuelve el bloque integrations ni ningún valor que contenga. No usa
// t.Parallel: registra un plugin de integraciones de prueba en el registro del proceso.
func TestSettingsGetDoesNotExposeIntegrations(t *testing.T) {
	integrationstest.Install(t, integrationstest.NewFake())
	root := newSettingsProject(t, settingsProjectYAML+"\nintegrations:\n  fake:\n    project: \"p-secreto-no-visible\"\n")
	router, _, _ := newPrefsServer(t, root)
	rr := request(router, http.MethodGet, "/api/settings", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET = %d: %s", rr.Code, rr.Body.String())
	}
	for _, banned := range []string{"integrations", "p-secreto-no-visible"} {
		if strings.Contains(rr.Body.String(), banned) {
			t.Errorf("el cuerpo no debe contener %q:\n%s", banned, rr.Body.String())
		}
	}
	// Escribir otra clave conserva el bloque integrations intacto en el fichero.
	dto := getSettings(t, router)
	if rr := putSettings(router, dto.Revision, `{"workflow.lane":"sdd"}`); rr.Code != http.StatusOK {
		t.Fatalf("PUT = %d: %s", rr.Code, rr.Body.String())
	}
	data, _ := os.ReadFile(filepath.Join(root, "axiom.yaml"))
	if !strings.Contains(string(data), "project: \"p-secreto-no-visible\"") {
		t.Errorf("integrations debe conservarse:\n%s", data)
	}
}

// RF-WEB-18: el GET devuelve el catálogo con la revisión y la ruta.
func TestSettingsGetReturnsCatalogRevisionAndPath(t *testing.T) {
	t.Parallel()
	root := newSettingsProject(t, settingsProjectYAML)
	router, _, _ := newPrefsServer(t, root)
	rr := request(router, http.MethodGet, "/api/settings", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET = %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	dto := getSettings(t, router)
	if dto.ConfigPath != filepath.Join(root, "axiom.yaml") || len(dto.Revision) != 64 || !dto.Writable || dto.Topology != "monorepo-embedded" {
		t.Fatalf("dto = %+v", dto)
	}
	if v, ok := dto.value("workflow.lane"); !ok || v != "ask" {
		t.Errorf("workflow.lane = %v (%v)", v, ok)
	}
	if v, _ := dto.value("git.promotion"); v != nil {
		t.Errorf("una clave sin declarar vale null: %v", v)
	}
	if !strings.Contains(body, `"documentErrors":[`) {
		t.Errorf("documentErrors es siempre un array: %s", body)
	}
}

// RF-WEB-19: un PUT correcto escribe, devuelve la revisión nueva y el catálogo actualizado.
func TestSettingsPutWritesAndReturnsTheNewCatalog(t *testing.T) {
	t.Parallel()
	root := newSettingsProject(t, settingsProjectYAML)
	router, _, _ := newPrefsServer(t, root)
	before := getSettings(t, router)
	rr := putSettings(router, before.Revision, `{"workflow.lane":"sdd","git.branch_prefix":"inc"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT = %d: %s", rr.Code, rr.Body.String())
	}
	var resp settingsPutJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Changed || resp.Settings.Revision == before.Revision || resp.Warnings == nil {
		t.Fatalf("respuesta = %+v", resp)
	}
	if v, _ := resp.Settings.value("workflow.lane"); v != "sdd" {
		t.Errorf("lane = %v", v)
	}
	data, _ := os.ReadFile(filepath.Join(root, "axiom.yaml"))
	for _, want := range []string{`lane: "sdd"`, `branch_prefix: "inc"`, "# Proyecto de pruebas de Ajustes."} {
		if !strings.Contains(string(data), want) {
			t.Errorf("falta %q en axiom.yaml:\n%s", want, data)
		}
	}
	// Un PUT sin efecto devuelve changed=false.
	rr = putSettings(router, resp.Settings.Revision, `{"workflow.lane":"sdd"}`)
	var noop settingsPutJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &noop); err != nil || rr.Code != http.StatusOK || noop.Changed {
		t.Fatalf("PUT sin efecto = %d %s", rr.Code, rr.Body.String())
	}
}

// RF-WEB-19: 400, 404, 409 y 422 con el cuerpo `{code, error, violations?}`.
func TestSettingsPutErrorStatuses(t *testing.T) {
	t.Parallel()
	root := newSettingsProject(t, settingsProjectYAML)
	router, _, _ := newPrefsServer(t, root)
	rev := getSettings(t, router).Revision
	path := filepath.Join(root, "axiom.yaml")
	original, _ := os.ReadFile(path)

	cases := []struct {
		name   string
		body   string
		status int
		code   string
	}{
		{"JSON inválido", `{`, http.StatusBadRequest, "invalid_request"},
		{"sin revisión", `{"changes":{"workflow.lane":"sdd"}}`, http.StatusBadRequest, "invalid_request"},
		{"lote vacío", `{"revision":"` + rev + `","changes":{}}`, http.StatusBadRequest, "invalid_request"},
		{"revisión obsoleta", `{"revision":"` + strings.Repeat("0", 64) + `","changes":{"workflow.lane":"sdd"}}`, http.StatusConflict, "revision_mismatch"},
		{"clave desconocida", `{"revision":"` + rev + `","changes":{"git.inventada":"x"}}`, http.StatusUnprocessableEntity, "unknown_key"},
		{"integrations", `{"revision":"` + rev + `","changes":{"integrations.fake.project":"x"}}`, http.StatusUnprocessableEntity, "unknown_key"},
		{"solo lectura", `{"revision":"` + rev + `","changes":{"workspace.topology":"multirepo"}}`, http.StatusUnprocessableEntity, "key_not_editable"},
		{"no aplicable", `{"revision":"` + rev + `","changes":{"acceptance_tests.enabled":true}}`, http.StatusUnprocessableEntity, "key_not_applicable"},
		{"valor inválido", `{"revision":"` + rev + `","changes":{"sdd.iso":"si"}}`, http.StatusUnprocessableEntity, "invalid_value"},
		{"validación", `{"revision":"` + rev + `","changes":{"git.promotion":"siempre"}}`, http.StatusUnprocessableEntity, "validation_failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := request(router, http.MethodPut, "/api/settings", tc.body)
			if rr.Code != tc.status {
				t.Fatalf("estado = %d, quiero %d: %s", rr.Code, tc.status, rr.Body.String())
			}
			var body settingsErrJSON
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil || body.Code != tc.code || body.Error == "" {
				t.Fatalf("cuerpo = %s (%v)", rr.Body.String(), err)
			}
			if tc.code == "validation_failed" && (len(body.Violations) == 0 || !strings.Contains(strings.Join(body.Violations, "\n"), "debe ser 'branch', 'release' o 'ask'")) {
				t.Errorf("validation_failed lleva las violaciones: %v", body.Violations)
			}
			if now, _ := os.ReadFile(path); string(now) != string(original) {
				t.Fatal("un rechazo no debe modificar axiom.yaml")
			}
		})
	}
}

// RF-WEB-19: un axiom.yaml con construcciones no soportadas se informa y se rechaza con 422.
func TestSettingsUnsupportedDocumentIsReportedAndRejected(t *testing.T) {
	t.Parallel()
	root := newSettingsProject(t, settingsProjectYAML+"---\nworkspace:\n  name: otro\n")
	router, _, _ := newPrefsServer(t, root)
	dto := getSettings(t, router)
	if dto.Writable || !strings.Contains(dto.WriteBlockedReason, "más de un documento") {
		t.Fatalf("dto = %+v", dto)
	}
	rr := putSettings(router, dto.Revision, `{"workflow.lane":"sdd"}`)
	if rr.Code != http.StatusUnprocessableEntity || !strings.Contains(rr.Body.String(), "unsupported_document") {
		t.Fatalf("PUT = %d: %s", rr.Code, rr.Body.String())
	}
}

// RF-WEB-19: sin axiom.yaml, GET y PUT responden 404 config_missing.
func TestSettingsWithoutAxiomYamlIsNotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	router, _, _ := newPrefsServer(t, dir)
	for _, rr := range []*httptest.ResponseRecorder{
		request(router, http.MethodGet, "/api/settings", ""),
		putSettings(router, strings.Repeat("0", 64), `{"workflow.lane":"sdd"}`),
	} {
		if rr.Code != http.StatusNotFound || !strings.Contains(rr.Body.String(), `"code":"config_missing"`) {
			t.Fatalf("respuesta = %d: %s", rr.Code, rr.Body.String())
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "axiom.yaml")); err == nil {
		t.Fatal("no debe crearse ningún axiom.yaml")
	}
}

// RF-WEB-10, RF-WEB-19: el PUT de un host o un origen ajenos responde 403 sin leer el cuerpo ni
// escribir; el GET sigue abierto.
func TestSettingsPutFromAForeignHostIsForbidden(t *testing.T) {
	t.Parallel()
	root := newSettingsProject(t, settingsProjectYAML)
	router, _, _ := newPrefsServer(t, root)
	rev := getSettings(t, router).Revision
	path := filepath.Join(root, "axiom.yaml")
	original, _ := os.ReadFile(path)
	body := `{"revision":"` + rev + `","changes":{"workflow.lane":"sdd"}}`
	for name, rr := range map[string]*httptest.ResponseRecorder{
		"host ajeno":   requestFrom(router, http.MethodPut, "/api/settings", body, "evil.example.com", ""),
		"origen ajeno": requestFrom(router, http.MethodPut, "/api/settings", body, "127.0.0.1", "http://evil.example.com"),
		"cuerpo roto":  requestFrom(router, http.MethodPut, "/api/settings", `{`, "evil.example.com", ""),
	} {
		if rr.Code != http.StatusForbidden {
			t.Errorf("%s: estado = %d, quiero 403: %s", name, rr.Code, rr.Body.String())
		}
	}
	if now, _ := os.ReadFile(path); string(now) != string(original) {
		t.Fatal("un PUT ajeno no debe escribir")
	}
	if rr := requestFrom(router, http.MethodGet, "/api/settings", "", "evil.example.com", ""); rr.Code != http.StatusOK {
		t.Errorf("el GET sigue abierto: %d", rr.Code)
	}
}

// RF-WEB-19: POST, PATCH y DELETE responden 405.
func TestSettingsOtherMethodsAreNotAllowed(t *testing.T) {
	t.Parallel()
	router, _, _ := newPrefsServer(t, newSettingsProject(t, settingsProjectYAML))
	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodDelete} {
		if rr := request(router, method, "/api/settings", `{}`); rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s = %d, quiero 405", method, rr.Code)
		}
	}
}

// RF-WEB-19: en multirepo se escribe el axiom.yaml del repositorio de specs apuntado y no se
// crea ninguno en la carpeta maestra ni en los repositorios de código.
func TestSettingsMultirepoWritesTheSpecsRepositoryFile(t *testing.T) {
	t.Parallel()
	ws := workspacetest.Canonical(t)
	router, _, _ := newPrefsServer(t, ws.Master)
	dto := getSettings(t, router)
	specsYAML := filepath.Join(ws.Specs, "axiom.yaml")
	if dto.ConfigPath != specsYAML || dto.Topology != "multirepo" {
		t.Fatalf("dto = %+v, quiero la ruta %s", dto, specsYAML)
	}
	if _, ok := dto.value("roles[backend].gate_policy"); !ok {
		t.Error("multirepo incluye roles[].gate_policy")
	}
	rr := putSettings(router, dto.Revision, `{"workflow.lane":"sdd","roles[backend].gate_policy":"deferred"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT = %d: %s", rr.Code, rr.Body.String())
	}
	data, _ := os.ReadFile(specsYAML)
	if !strings.Contains(string(data), `lane: "sdd"`) || !strings.Contains(string(data), `gate_policy: "deferred"`) {
		t.Errorf("repo-specs/axiom.yaml:\n%s", data)
	}
	for _, dir := range []string{ws.Master, ws.Backend, ws.Frontend} {
		if _, err := os.Stat(filepath.Join(dir, "axiom.yaml")); err == nil {
			t.Errorf("no debe existir un axiom.yaml en %s", dir)
		}
	}
}

// RF-WEB-19: un PUT solo cambia axiom.yaml: un kickoff.yaml sellado y el resto del árbol no cambian.
func TestSettingsPutOnlyChangesAxiomYaml(t *testing.T) {
	t.Parallel()
	ws := workspacetest.Embedded(t, "openspec")
	workspacetest.WriteFile(t, filepath.Join(ws.Master, "axiom.yaml"), settingsProjectYAML)
	workspacetest.WriteFile(t, filepath.Join(ws.Home, "changes", "x", "kickoff.yaml"), "executionStyle: continuous\nsealed: true\n")
	router, _, _ := newPrefsServer(t, ws.Master)
	before := snapshotTree(t, ws.Master)
	rr := putSettings(router, getSettings(t, router).Revision, `{"workflow.lane":"sdd"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT = %d: %s", rr.Code, rr.Body.String())
	}
	after := snapshotTree(t, ws.Master)
	for name, content := range before {
		if name == "axiom.yaml" {
			if after[name] == content {
				t.Error("axiom.yaml debía cambiar")
			}
			continue
		}
		if after[name] != content {
			t.Errorf("%s cambió", name)
		}
	}
	if len(after) != len(before) {
		t.Errorf("el PUT creó o borró ficheros: %v frente a %v", before, after)
	}
}

// RF-WEB-10: PUT worktree-prefs desde un host ajeno responde 403 sin escribir el hub; el GET
// sigue siendo normal.
func TestWorktreePrefsPutFromAForeignHostIsForbidden(t *testing.T) {
	t.Parallel()
	dir := newProjectDir(t, "activo")
	router, _, hubPath := newPrefsServer(t, dir)
	if rr := request(router, http.MethodPost, "/api/projects/add", `{"path":`+quoteJSON(dir)+`}`); rr.Code != http.StatusOK {
		t.Fatalf("alta = %d: %s", rr.Code, rr.Body.String())
	}
	hubBefore := string(readOrEmpty(t, hubPath))
	rr := requestFrom(router, http.MethodPut, "/api/projects/active/worktree-prefs", `{"mode":"ask","root":""}`, "evil.example.com", "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("PUT ajeno = %d: %s", rr.Code, rr.Body.String())
	}
	if hubAfter := string(readOrEmpty(t, hubPath)); hubAfter != hubBefore {
		t.Errorf("el hub cambió:\n%s", hubAfter)
	}
	if rr := requestFrom(router, http.MethodGet, "/api/projects/active/worktree-prefs", "", "evil.example.com", ""); rr.Code != http.StatusOK {
		t.Errorf("el GET sigue siendo normal: %d %s", rr.Code, rr.Body.String())
	}
}
