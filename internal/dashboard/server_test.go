package dashboard

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Pruebas de las preferencias personales de worktree en el dashboard (INC-25, REQ-3.1,
// REQ-3.2, REQ-25.29, REQ-25.33). Decodifican el JSON con tipos propios de la prueba, no con
// los DTO de producción: así fijan los nombres de los campos del contrato HTTP.

// prefsJSON y projectJSON son la forma observable de `worktree_prefs` y de cada proyecto de
// `GET /api/projects`. Un campo vacío es «sin preferencia guardada».
type prefsJSON struct {
	Mode string `json:"mode"`
	Root string `json:"root"`
}

type projectJSON struct {
	ID                     string    `json:"id"`
	Path                   string    `json:"path"`
	WorktreePrefs          prefsJSON `json:"worktree_prefs"`
	EffectiveWorktreePrefs prefsJSON `json:"effective_worktree_prefs"`
}

// activePrefsJSON es la respuesta de `GET|PUT /api/projects/active/worktree-prefs`.
type activePrefsJSON struct {
	ID                     string    `json:"id"`
	Path                   string    `json:"path"`
	WorktreePrefs          prefsJSON `json:"worktree_prefs"`
	EffectiveWorktreePrefs prefsJSON `json:"effective_worktree_prefs"`
}

// newPrefsServer crea un servidor sobre un hub temporal cuyo proyecto de trabajo es root.
func newPrefsServer(t *testing.T, root string) (http.Handler, *hub.Manager, string) {
	t.Helper()
	hubPath := filepath.Join(t.TempDir(), "workspaces.json")
	mgr, err := hub.NewManager(hubPath)
	if err != nil {
		t.Fatalf("hub.NewManager: %v", err)
	}
	return NewServer(NewServiceWithHub(root, mgr)).Router(), mgr, hubPath
}

// newProjectDir crea un proyecto temporal con un axiom.yaml mínimo.
func newProjectDir(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "workspace:\n  name: " + name + "\n  topology: monorepo-embedded\n"
	if err := os.WriteFile(filepath.Join(dir, "axiom.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// request lanza una petición como la haría el propio dashboard: Host 127.0.0.1, que es lo
// que aceptan las rutas que cambian estado (isLocalRequest). httptest usa example.com por
// defecto.
func request(router http.Handler, method, target, body string) *httptest.ResponseRecorder {
	return requestFrom(router, method, target, body, "127.0.0.1", "")
}

// requestFrom es request con el Host y el Origin indicados.
func requestFrom(router http.Handler, method, target, body, host, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, bytes.NewReader([]byte(body)))
	req.Host = host
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func listProjects(t *testing.T, router http.Handler) []projectJSON {
	t.Helper()
	rr := request(router, http.MethodGet, "/api/projects", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/projects = %d: %s", rr.Code, rr.Body.String())
	}
	var list struct {
		Projects []projectJSON `json:"projects"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("JSON inválido en /api/projects: %v", err)
	}
	return list.Projects
}

// readOrEmpty devuelve los bytes del fichero, o nil si no existe.
func readOrEmpty(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// snapshotTree devuelve el contenido de todos los ficheros bajo dir por ruta relativa.
func snapshotTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		data, err := os.ReadFile(path)
		out[rel] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func errorOf(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("la respuesta no es un JSON con error: %v (%s)", err, rr.Body.String())
	}
	return body.Error
}

func TestProjectsAddPersistsWorktreePrefs(t *testing.T) {
	dir := newProjectDir(t, "con-modo")
	router, _, hubPath := newPrefsServer(t, dir)

	rr := request(router, http.MethodPost, "/api/projects/add", `{"path":`+quoteJSON(dir)+`,"name":"Con modo","worktree_mode":"in_place"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/projects/add = %d: %s", rr.Code, rr.Body.String())
	}

	projects := listProjects(t, router)
	if len(projects) != 1 {
		t.Fatalf("proyectos = %d, quiero 1", len(projects))
	}
	got := projects[0]
	if got.WorktreePrefs != (prefsJSON{Mode: "in_place"}) {
		t.Errorf("worktree_prefs = %+v, quiero solo el modo in_place", got.WorktreePrefs)
	}
	if want := (prefsJSON{Mode: "in_place", Root: branch.DefaultWorktreeRoot(dir)}); got.EffectiveWorktreePrefs != want {
		t.Errorf("effective_worktree_prefs = %+v, quiero %+v", got.EffectiveWorktreePrefs, want)
	}
	if !strings.Contains(string(readOrEmpty(t, hubPath)), `"worktree_mode": "in_place"`) {
		t.Errorf("workspaces.json no guarda el modo:\n%s", readOrEmpty(t, hubPath))
	}
}

func TestProjectsAddWithoutPrefsUsesDefaults(t *testing.T) {
	dir := newProjectDir(t, "sin-modo")
	router, _, hubPath := newPrefsServer(t, dir)

	rr := request(router, http.MethodPost, "/api/projects/add", `{"path":`+quoteJSON(dir)+`}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/projects/add = %d: %s", rr.Code, rr.Body.String())
	}

	got := listProjects(t, router)[0]
	if got.WorktreePrefs != (prefsJSON{}) {
		t.Errorf("worktree_prefs = %+v, quiero ninguna preferencia guardada", got.WorktreePrefs)
	}
	if want := (prefsJSON{Mode: "worktree", Root: branch.DefaultWorktreeRoot(dir)}); got.EffectiveWorktreePrefs != want {
		t.Errorf("effective_worktree_prefs = %+v, quiero %+v", got.EffectiveWorktreePrefs, want)
	}
	if strings.Contains(string(readOrEmpty(t, hubPath)), "worktree_") {
		t.Errorf("workspaces.json no debe guardar preferencias:\n%s", readOrEmpty(t, hubPath))
	}
}

func TestProjectsInitPersistsWorktreePrefs(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "recien-iniciado")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	router, _, _ := newPrefsServer(t, dir)
	root := filepath.Join(t.TempDir(), "mis-worktrees")

	rr := request(router, http.MethodPost, "/api/projects/init", `{"path":`+quoteJSON(dir)+`,"name":"Recién iniciado","worktree_mode":"ask","worktree_root":`+quoteJSON(root)+`}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/projects/init = %d: %s", rr.Code, rr.Body.String())
	}

	got := listProjects(t, router)[0]
	if want := (prefsJSON{Mode: "ask", Root: root}); got.WorktreePrefs != want || got.EffectiveWorktreePrefs != want {
		t.Errorf("preferencias = %+v / %+v, quiero ambas %+v", got.WorktreePrefs, got.EffectiveWorktreePrefs, want)
	}
}

func TestProjectsRegistrationRejectsInvalidPrefsWithoutTouchingHub(t *testing.T) {
	dir := newProjectDir(t, "invalido")
	router, _, hubPath := newPrefsServer(t, dir)
	before := readOrEmpty(t, hubPath)

	cases := []struct {
		name, target, body, want string
	}{
		{"add con modo inválido", "/api/projects/add", `{"path":` + quoteJSON(dir) + `,"worktree_mode":"siempre"}`, "in_place"},
		{"add con raíz relativa", "/api/projects/add", `{"path":` + quoteJSON(dir) + `,"worktree_root":"../wt"}`, "absoluta"},
		{"init con modo inválido", "/api/projects/init", `{"path":` + quoteJSON(dir) + `,"worktree_mode":"siempre"}`, "in_place"},
		{"init con raíz relativa", "/api/projects/init", `{"path":` + quoteJSON(dir) + `,"worktree_root":"../wt"}`, "absoluta"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := request(router, http.MethodPost, tc.target, tc.body)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("código = %d, quiero 400: %s", rr.Code, rr.Body.String())
			}
			if msg := errorOf(t, rr); !strings.Contains(msg, tc.want) {
				t.Errorf("el mensaje %q debe contener %q", msg, tc.want)
			}
			if after := readOrEmpty(t, hubPath); !bytes.Equal(before, after) {
				t.Errorf("el hub cambió:\n%s", after)
			}
		})
	}
}

func TestProjectsAddLinkedWorktreeNamesMainCheckout(t *testing.T) {
	base := t.TempDir()
	main := filepath.Join(base, "repo-principal")
	gitDir := filepath.Join(main, ".git", "worktrees", "feature")
	linked := filepath.Join(base, "repo-principal-wt", "feature")
	for _, dir := range []string{gitDir, linked} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(gitDir, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linked, ".git"), []byte("gitdir: "+filepath.ToSlash(gitDir)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	router, _, hubPath := newPrefsServer(t, linked)

	rr := request(router, http.MethodPost, "/api/projects/add", `{"path":`+quoteJSON(linked)+`}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("código = %d, quiero 400: %s", rr.Code, rr.Body.String())
	}
	msg := errorOf(t, rr)
	if !strings.Contains(msg, "axiom project add") || !strings.Contains(msg, "repo-principal") {
		t.Errorf("el rechazo debe nombrar el checkout principal con su orden de registro: %s", msg)
	}
	if data := readOrEmpty(t, hubPath); data != nil {
		t.Errorf("el hub no debe crearse con un worktree enlazado:\n%s", data)
	}
}

func TestActiveWorktreePrefsRoundTrip(t *testing.T) {
	dir := newProjectDir(t, "activo")
	router, _, hubPath := newPrefsServer(t, dir)
	if rr := request(router, http.MethodPost, "/api/projects/add", `{"path":`+quoteJSON(dir)+`}`); rr.Code != http.StatusOK {
		t.Fatalf("alta = %d: %s", rr.Code, rr.Body.String())
	}

	var initial activePrefsJSON
	rr := request(router, http.MethodGet, "/api/projects/active/worktree-prefs", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET = %d: %s", rr.Code, rr.Body.String())
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if want := (prefsJSON{Mode: "worktree", Root: branch.DefaultWorktreeRoot(dir)}); initial.WorktreePrefs != (prefsJSON{}) || initial.EffectiveWorktreePrefs != want {
		t.Errorf("GET inicial = %+v, quiero sin guardadas y efectivas %+v", initial, want)
	}

	projectBefore := snapshotTree(t, dir)
	root := filepath.Join(t.TempDir(), "worktrees-del-activo")
	rr = request(router, http.MethodPut, "/api/projects/active/worktree-prefs", `{"mode":"ask","root":`+quoteJSON(root)+`}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT = %d: %s", rr.Code, rr.Body.String())
	}
	var saved activePrefsJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if want := (prefsJSON{Mode: "ask", Root: root}); saved.WorktreePrefs != want || saved.EffectiveWorktreePrefs != want {
		t.Errorf("respuesta del PUT = %+v, quiero ambas %+v", saved, want)
	}
	hubJSON := string(readOrEmpty(t, hubPath))
	if !strings.Contains(hubJSON, `"worktree_mode": "ask"`) || !strings.Contains(hubJSON, `"worktree_root"`) {
		t.Errorf("workspaces.json no guarda las preferencias:\n%s", hubJSON)
	}
	// La edición es personal: ningún fichero del proyecto cambia, tampoco axiom.yaml.
	if after := snapshotTree(t, dir); !equalTrees(projectBefore, after) {
		t.Errorf("el PUT modificó ficheros del proyecto:\nantes %v\ndespués %v", projectBefore, after)
	}

	// Un GET posterior devuelve lo guardado, y un PUT vacío retira las preferencias.
	rr = request(router, http.MethodGet, "/api/projects/active/worktree-prefs", "")
	var again activePrefsJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &again); err != nil || again.WorktreePrefs.Mode != "ask" {
		t.Errorf("GET tras el PUT = %s (%v)", rr.Body.String(), err)
	}
	if rr = request(router, http.MethodPut, "/api/projects/active/worktree-prefs", `{"mode":"","root":""}`); rr.Code != http.StatusOK {
		t.Fatalf("PUT vacío = %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(string(readOrEmpty(t, hubPath)), "worktree_") {
		t.Errorf("el PUT vacío debe retirar las preferencias:\n%s", readOrEmpty(t, hubPath))
	}
}

func TestActiveWorktreePrefsRejectsInvalidValues(t *testing.T) {
	dir := newProjectDir(t, "rechazo")
	router, _, hubPath := newPrefsServer(t, dir)
	if rr := request(router, http.MethodPost, "/api/projects/add", `{"path":`+quoteJSON(dir)+`}`); rr.Code != http.StatusOK {
		t.Fatalf("alta = %d: %s", rr.Code, rr.Body.String())
	}
	before := readOrEmpty(t, hubPath)

	for _, tc := range []struct{ name, body, want string }{
		{"modo inválido", `{"mode":"siempre"}`, "worktree, in_place o ask"},
		{"raíz relativa", `{"root":"../wt"}`, "ruta absoluta"},
		{"JSON inválido", `{`, "JSON inválido"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := request(router, http.MethodPut, "/api/projects/active/worktree-prefs", tc.body)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("código = %d, quiero 400: %s", rr.Code, rr.Body.String())
			}
			if msg := errorOf(t, rr); !strings.Contains(msg, tc.want) {
				t.Errorf("el mensaje %q debe contener %q", msg, tc.want)
			}
			if after := readOrEmpty(t, hubPath); !bytes.Equal(before, after) {
				t.Errorf("el hub cambió:\n%s", after)
			}
		})
	}
}

func TestActiveWorktreePrefsRequiresRegisteredProject(t *testing.T) {
	dir := newProjectDir(t, "sin-registrar")
	router, _, hubPath := newPrefsServer(t, dir)

	for _, tc := range []struct{ method, body string }{
		{http.MethodGet, ""},
		{http.MethodPut, `{"mode":"ask"}`},
	} {
		rr := request(router, tc.method, "/api/projects/active/worktree-prefs", tc.body)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("%s = %d, quiero 404: %s", tc.method, rr.Code, rr.Body.String())
		}
		if msg := errorOf(t, rr); !strings.Contains(msg, "axiom project add") {
			t.Errorf("%s: el mensaje %q debe nombrar `axiom project add`", tc.method, msg)
		}
	}
	if data := readOrEmpty(t, hubPath); data != nil {
		t.Errorf("el hub no debe crearse sin proyecto registrado:\n%s", data)
	}
}

func TestActiveWorktreePrefsMethodNotAllowed(t *testing.T) {
	router, _, _ := newPrefsServer(t, newProjectDir(t, "metodos"))
	for _, method := range []string{http.MethodPost, http.MethodDelete, http.MethodPatch} {
		if rr := request(router, method, "/api/projects/active/worktree-prefs", `{}`); rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s = %d, quiero 405", method, rr.Code)
		}
	}
}

// El arnés de S7.11: GET /api/projects y PUT /api/projects/active/worktree-prefs sobre un hub
// temporal, mostrando el cuerpo observable. `axiom ui` bloquea y no se ejecuta.
func TestProjectsWorktreePrefsHarness(t *testing.T) {
	dir := newProjectDir(t, "arnes")
	router, _, _ := newPrefsServer(t, dir)
	request(router, http.MethodPost, "/api/projects/add", `{"path":`+quoteJSON(dir)+`}`)
	root := filepath.Join(t.TempDir(), "wts")
	if rr := request(router, http.MethodPut, "/api/projects/active/worktree-prefs", `{"mode":"in_place","root":`+quoteJSON(root)+`}`); rr.Code != http.StatusOK {
		t.Fatalf("PUT = %d: %s", rr.Code, rr.Body.String())
	}
	rr := request(router, http.MethodGet, "/api/projects", "")
	t.Logf("GET /api/projects -> %d %s", rr.Code, strings.TrimSpace(rr.Body.String()))
	got := listProjects(t, router)[0]
	if want := (prefsJSON{Mode: "in_place", Root: root}); got.WorktreePrefs != want {
		t.Errorf("worktree_prefs = %+v, quiero %+v", got.WorktreePrefs, want)
	}
}

// El servidor sirve la página con los campos de worktree del modal de registro y el módulo que los
// envía. El detalle de los campos y de la lógica (rediseño) lo cubre assets_branch_test.go; aquí se
// fija solo que lo que el navegador recibe por HTTP es lo que ofrece las preferencias (REQ-25.29,
// REQ-25.33) y que la interfaz no deja editar el bloque `git:`.
func TestRegistrationModalAsksWorktreePrefs(t *testing.T) {
	router, _, _ := newPrefsServer(t, newProjectDir(t, "assets"))

	html := request(router, http.MethodGet, "/", "").Body.String()
	selectBlock := regexp.MustCompile(`(?s)<select id="select-project-worktree-mode".*?</select>`).FindString(html)
	if selectBlock == "" {
		t.Fatal("index.html: el modal de registro no tiene el selector de modo `select-project-worktree-mode`")
	}
	if got := strings.Count(selectBlock, "<option"); got != 3 {
		t.Errorf("el selector de modo tiene %d opciones, quiero 3", got)
	}
	if !regexp.MustCompile(`<input[^>]*id="input-project-worktree-root"[^>]*>`).MatchString(html) {
		t.Fatal("index.html: el modal de registro no tiene el campo `input-project-worktree-root`")
	}
	for _, marker := range []string{"input-worktree-prefs-root", "select-worktree-prefs-mode", "btn-save-worktree-prefs"} {
		if !strings.Contains(html, marker) {
			t.Errorf("index.html debe contener el editor de preferencias del proyecto activo (%s)", marker)
		}
	}
	if strings.Contains(html, "integration_branch") || strings.Contains(html, "branch_prefix") {
		t.Error("index.html no debe ofrecer controles de edición de `git:` (REQ-25.33)")
	}

	// Los módulos que envían y guardan las preferencias se sirven como JavaScript.
	for target, markers := range map[string][]string{
		"/js/projects.js":          {"worktree"},
		"/js/screens/ecosystem.js": {"/api/projects/active/worktree-prefs"},
		"/js/core/branches.js":     {"worktree_mode", "worktree_root", "defaultWorktreeRoot"},
	} {
		rr := request(router, http.MethodGet, target, "")
		if rr.Code != http.StatusOK || !strings.HasPrefix(rr.Header().Get("Content-Type"), "text/javascript") {
			t.Errorf("GET %s = %d (%s), se esperaba un módulo JavaScript", target, rr.Code, rr.Header().Get("Content-Type"))
		}
		for _, marker := range markers {
			if !strings.Contains(rr.Body.String(), marker) {
				t.Errorf("%s debe contener %q", target, marker)
			}
		}
	}
}

func quoteJSON(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}

func equalTrees(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}

// Pruebas de la política de ramas en solo lectura de `GET /api/workspace` (INC-25, REQ-1.1,
// REQ-25.33, design D-20). Decodifican el JSON con tipos propios de la prueba, no con los DTO de
// producción: así fijan los nombres de los campos del contrato HTTP.

// branchPolicyJSON y repositoryJSON son la forma observable de `branchPolicy` y de cada entrada
// de `repositories[]`. Un campo vacío en repositoryJSON es «hereda el valor del bloque `git:`».
type branchPolicyJSON struct {
	IntegrationBranch string `json:"integrationBranch"`
	BranchPrefix      string `json:"branchPrefix"`
}

type repositoryJSON struct {
	Path              string `json:"path"`
	IntegrationBranch string `json:"integrationBranch"`
	BranchPrefix      string `json:"branchPrefix"`
}

type workspacePolicyJSON struct {
	IsConfigured bool             `json:"is_configured"`
	BranchPolicy branchPolicyJSON `json:"branchPolicy"`
	Repositories []repositoryJSON `json:"repositories"`
}

// newPolicyProjectDir crea un proyecto temporal cuyo axiom.yaml es el texto dado.
func newPolicyProjectDir(t *testing.T, axiomYAML string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "axiom.yaml"), []byte(axiomYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func getWorkspacePolicy(t *testing.T, router http.Handler) (workspacePolicyJSON, map[string]json.RawMessage) {
	t.Helper()
	rr := request(router, http.MethodGet, "/api/workspace", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/workspace = %d, quiero 200: %s", rr.Code, rr.Body.String())
	}
	var got workspacePolicyJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON inválido en /api/workspace: %v (%s)", err, rr.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	return got, raw
}

// policyHeader es el axiom.yaml mínimo que carga como configurado (la carga exige roles).
const policyHeader = "workspace:\n  name: politica\n  topology: monorepo-embedded\n  specs_repository: openspec\nroles:\n  core:\n    name: Core\n    repositories:\n      - path: .\n"

func TestWorkspaceBranchPolicy(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string // vacío: el proyecto no tiene axiom.yaml
		wantPolicy branchPolicyJSON
		wantRepos  []repositoryJSON
	}{
		{
			name:       "git declara solo la rama de integración y el prefijo toma su valor por defecto",
			yaml:       policyHeader + "git:\n  integration_branch: develop\n",
			wantPolicy: branchPolicyJSON{IntegrationBranch: "develop", BranchPrefix: "feat"},
		},
		{
			name:       "sin bloque git rigen los valores por defecto",
			yaml:       policyHeader,
			wantPolicy: branchPolicyJSON{IntegrationBranch: "main", BranchPrefix: "feat"},
		},
		{
			name:       "git declara ambos valores",
			yaml:       policyHeader + "git:\n  integration_branch: trunk\n  branch_prefix: inc\n",
			wantPolicy: branchPolicyJSON{IntegrationBranch: "trunk", BranchPrefix: "inc"},
		},
		{
			name:       "un repositorio declara su rama de integración y hereda el prefijo",
			yaml:       policyHeader + "repositories:\n  - path: repo-backend\n    integration_branch: release\n",
			wantPolicy: branchPolicyJSON{IntegrationBranch: "main", BranchPrefix: "feat"},
			wantRepos:  []repositoryJSON{{Path: "repo-backend", IntegrationBranch: "release"}},
		},
		{
			name: "varios repositorios con git: la política del workspace no se sustituye por la de un repositorio",
			yaml: policyHeader + "git:\n  integration_branch: develop\n  branch_prefix: inc\n" +
				"repositories:\n  - path: repo-backend\n    integration_branch: release\n    branch_prefix: fix\n  - path: repo-web\n",
			wantPolicy: branchPolicyJSON{IntegrationBranch: "develop", BranchPrefix: "inc"},
			wantRepos: []repositoryJSON{
				{Path: "repo-backend", IntegrationBranch: "release", BranchPrefix: "fix"},
				{Path: "repo-web"},
			},
		},
		{
			name:       "sin axiom.yaml rigen los valores por defecto",
			yaml:       "",
			wantPolicy: branchPolicyJSON{IntegrationBranch: "main", BranchPrefix: "feat"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.yaml != "" {
				dir = newPolicyProjectDir(t, tt.yaml)
			}
			router, _, _ := newPrefsServer(t, dir)
			got, raw := getWorkspacePolicy(t, router)

			if want := tt.yaml != ""; got.IsConfigured != want {
				t.Fatalf("is_configured = %v, quiero %v: el axiom.yaml de la prueba no se cargó como esperaba", got.IsConfigured, want)
			}
			if got.BranchPolicy != tt.wantPolicy {
				t.Errorf("branchPolicy = %+v, quiero %+v", got.BranchPolicy, tt.wantPolicy)
			}
			if len(got.Repositories) != len(tt.wantRepos) {
				t.Fatalf("repositories = %+v, quiero %+v", got.Repositories, tt.wantRepos)
			}
			for i, want := range tt.wantRepos {
				if got.Repositories[i] != want {
					t.Errorf("repositories[%d] = %+v, quiero %+v", i, got.Repositories[i], want)
				}
			}
			// Sin entradas, `repositories` es un array vacío y no `null`: la SPA itera sin guardas.
			if string(raw["repositories"]) == "null" || raw["repositories"] == nil {
				t.Errorf("repositories debe ser un array JSON, es %s", raw["repositories"])
			}
		})
	}
}

// Ninguna ruta del servidor escribe axiom.yaml (REQ-25.33): ni la lectura de la política ni
// los métodos no permitidos sobre `/api/workspace` cambian el árbol del proyecto.
func TestWorkspaceIsReadOnly(t *testing.T) {
	dir := newPolicyProjectDir(t, policyHeader+"git:\n  integration_branch: develop\nrepositories:\n  - path: repo-backend\n    integration_branch: release\n")
	router, _, _ := newPrefsServer(t, dir)
	before := snapshotTree(t, dir)

	getWorkspacePolicy(t, router)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		body := `{"branchPolicy":{"integrationBranch":"hack","branchPrefix":"hack"}}`
		if rr := request(router, method, "/api/workspace", body); rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/workspace = %d, quiero 405", method, rr.Code)
		}
	}

	if after := snapshotTree(t, dir); !equalTrees(before, after) {
		t.Errorf("el árbol del proyecto cambió:\nantes:   %v\ndespués: %v", before, after)
	}
	got, _ := getWorkspacePolicy(t, router)
	if !got.IsConfigured {
		t.Fatal("is_configured = false: el axiom.yaml de la prueba no se cargó")
	}
	if got.BranchPolicy.IntegrationBranch != "develop" {
		t.Errorf("integrationBranch = %q tras los métodos rechazados, quiero develop", got.BranchPolicy.IntegrationBranch)
	}
}

// El arnés de S10.6: GET /api/workspace con y sin `git:` y `repositories:`, mostrando el cuerpo
// observable. `axiom ui` bloquea y no se ejecuta.
func TestWorkspaceBranchPolicyHarness(t *testing.T) {
	for name, yaml := range map[string]string{
		"sin git ni repositories": policyHeader,
		"con git y repositories":  policyHeader + "git:\n  integration_branch: develop\nrepositories:\n  - path: repo-backend\n    integration_branch: release\n",
	} {
		router, _, _ := newPrefsServer(t, newPolicyProjectDir(t, yaml))
		rr := request(router, http.MethodGet, "/api/workspace", "")
		t.Logf("%s: GET /api/workspace -> %d %s", name, rr.Code, strings.TrimSpace(rr.Body.String()))
		if rr.Code != http.StatusOK {
			t.Errorf("%s: GET /api/workspace = %d, quiero 200", name, rr.Code)
		}
	}
}

// G4.1: una configuración inválida es un 400 con la lista de violaciones y no
// escribe nada en disco; ya no es un 500 opaco.
func TestProjectsInitRejectsInvalidConfigurationWith400AndViolations(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proyecto-nuevo")
	router, _, hubPath := newPrefsServer(t, t.TempDir())
	body := `{"path":` + quoteJSON(dir) + `,"name":"Mal","roles":[` +
		`{"key":"web","name":"Web","repositories":["../fuera"]},` +
		`{"key":"web","name":"Otra","repositories":["."]}]}`

	rr := request(router, http.MethodPost, "/api/projects/init", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/projects/init = %d, quiero 400: %s", rr.Code, rr.Body.String())
	}
	var got struct {
		Error      string   `json:"error"`
		Violations []string `json:"violations"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON inválido: %v (%s)", err, rr.Body.String())
	}
	if got.Error == "" || len(got.Violations) != 2 {
		t.Fatalf("se esperaban error y 2 violaciones (repetida y fuera del proyecto): %+v", got)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("el alta rechazada no debe crear %s (err=%v)", dir, err)
	}
	if strings.Contains(string(readOrEmpty(t, hubPath)), "proyecto-nuevo") {
		t.Errorf("el alta rechazada no debe registrar el proyecto en el hub:\n%s", readOrEmpty(t, hubPath))
	}
}

// G4.3: el campo `tech` de cada rol viaja hasta el axiom.yaml.
func TestProjectsInitWritesRoleTech(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proyecto-nuevo")
	router, _, _ := newPrefsServer(t, t.TempDir())
	body := `{"path":` + quoteJSON(dir) + `,"name":"Con tech","roles":[{"key":"web","name":"Web","repositories":["."],"tech":["go","react"]}]}`

	rr := request(router, http.MethodPost, "/api/projects/init", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/projects/init = %d: %s", rr.Code, rr.Body.String())
	}
	cfg, err := workspace.LoadConfig(filepath.Join(dir, "axiom.yaml"))
	if err != nil {
		t.Fatalf("axiom.yaml no carga: %v", err)
	}
	if tech := cfg.Roles["web"].Tech; len(tech) != 2 || tech[0] != "go" || tech[1] != "react" {
		t.Errorf("tech del rol web = %v, quiero [go react]", tech)
	}
}

// G4.2: el alta multirepo desde la API deja el axiom.yaml en el repositorio de specs
// y el puntero en la carpeta del proyecto, y el dashboard lo carga a través del puntero.
func TestProjectsInitMultirepoCreatesSpecsRepoAndPointer(t *testing.T) {
	master := filepath.Join(t.TempDir(), "maestra")
	router, _, _ := newPrefsServer(t, t.TempDir())
	body := `{"path":` + quoteJSON(master) + `,"name":"Multi","topology":"multirepo","specs_repository":"repo-specs","roles":[{"key":"api","name":"API","repositories":["repo-api"]}]}`

	rr := request(router, http.MethodPost, "/api/projects/init", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/projects/init = %d: %s", rr.Code, rr.Body.String())
	}
	if _, err := os.Stat(filepath.Join(master, "repo-specs", "axiom.yaml")); err != nil {
		t.Errorf("falta el axiom.yaml en el repositorio de specs: %v", err)
	}
	if pointer := readOrEmpty(t, filepath.Join(master, ".axiom-workspace")); !strings.Contains(string(pointer), "repo-specs/axiom.yaml") {
		t.Errorf("puntero = %q", pointer)
	}
	loc, err := workspace.LocateAt(master)
	if err != nil || loc.Config.Workspace.SpecsRepository != "repo-specs" {
		t.Fatalf("LocateAt = %+v (err=%v)", loc, err)
	}
}

func TestProjectsInitRejectsBadSpecsRepositoryWith400(t *testing.T) {
	master := filepath.Join(t.TempDir(), "maestra")
	router, _, _ := newPrefsServer(t, t.TempDir())
	body := `{"path":` + quoteJSON(master) + `,"topology":"multirepo","specs_repository":"../fuera"}`

	rr := request(router, http.MethodPost, "/api/projects/init", body)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "violations") || !strings.Contains(rr.Body.String(), "sale de la carpeta del proyecto") {
		t.Fatalf("POST /api/projects/init = %d, quiero 400 con violations: %s", rr.Code, rr.Body.String())
	}
	if _, err := os.Stat(master); !os.IsNotExist(err) {
		t.Errorf("el alta rechazada no debe crear %s (err=%v)", master, err)
	}
}

// stubInstallAdapters sustituye la instalación real y devuelve lo que se pidió.
func stubInstallAdapters(t *testing.T, fn func(dir string, ids []string) ([]string, error)) {
	t.Helper()
	restore := installAdapters
	installAdapters = fn
	t.Cleanup(func() { installAdapters = restore })
}

// G4.6: el catálogo lista los adaptadores seleccionables y dice cuáles admiten workspace.
func TestAdaptersCatalogListsSelectableAgentsWithWorkspaceSupport(t *testing.T) {
	router, _, _ := newPrefsServer(t, t.TempDir())
	rr := request(router, http.MethodGet, "/api/adapters", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/adapters = %d: %s", rr.Code, rr.Body.String())
	}
	var got struct {
		Adapters []AdapterDTO `json:"adapters"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	byID := map[string]AdapterDTO{}
	for _, a := range got.Adapters {
		byID[a.ID] = a
	}
	if a := byID["claude-code"]; a.Name != "Claude Code" || !a.Workspace || a.Reason != "" {
		t.Errorf("claude-code = %+v", a)
	}
	for _, id := range []string{"pi", "openclaw", "windsurf", "antigravity", "trae-ide"} {
		if a, ok := byID[id]; !ok || a.Workspace || a.Reason == "" {
			t.Errorf("%s debe aparecer sin modo workspace y con motivo: %+v (ok=%v)", id, a, ok)
		}
	}
	for _, id := range []string{"kimi", "qwen-code", "hermes"} {
		if _, ok := byID[id]; ok {
			t.Errorf("%s no es seleccionable y no debe listarse", id)
		}
	}
	if rr := request(router, http.MethodPost, "/api/adapters", "{}"); rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/adapters = %d, quiero 405", rr.Code)
	}
}

func TestProjectsInitRejectsInvalidAdaptersBeforeWriting(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proyecto-nuevo")
	router, _, hubPath := newPrefsServer(t, t.TempDir())
	called := false
	stubInstallAdapters(t, func(string, []string) ([]string, error) { called = true; return nil, nil })

	body := `{"path":` + quoteJSON(dir) + `,"adapters":["claude-code","no-existe","kimi","pi"],"roles":[{"key":"a","name":"A","repositories":["../fuera"]}]}`
	rr := request(router, http.MethodPost, "/api/projects/init", body)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/projects/init = %d, quiero 400: %s", rr.Code, rr.Body.String())
	}
	var got struct {
		Violations []string `json:"violations"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got.Violations, "\n")
	for _, want := range []string{`"no-existe" no existe`, `"kimi" no está disponible`, `"pi" no se puede instalar`, "sale de la carpeta del proyecto"} {
		if !strings.Contains(joined, want) {
			t.Errorf("falta la violación %q en:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, `"claude-code"`) {
		t.Errorf("claude-code es válido y no debe aparecer:\n%s", joined)
	}
	if called {
		t.Error("no debe instalarse nada con adaptadores inválidos")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("el alta rechazada no debe crear %s (err=%v)", dir, err)
	}
	if strings.Contains(string(readOrEmpty(t, hubPath)), "proyecto-nuevo") {
		t.Error("el alta rechazada no debe registrar el proyecto")
	}
}

func TestProjectsInitInstallsTheChosenAdaptersInTheNewProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proyecto-nuevo")
	router, _, _ := newPrefsServer(t, t.TempDir())
	var gotDir string
	var gotIDs []string
	stubInstallAdapters(t, func(d string, ids []string) ([]string, error) {
		gotDir, gotIDs = d, ids
		return ids, nil
	})

	rr := request(router, http.MethodPost, "/api/projects/init", `{"path":`+quoteJSON(dir)+`,"adapters":["claude-code","cursor","claude-code"]}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/projects/init = %d: %s", rr.Code, rr.Body.String())
	}
	if gotDir != dir || strings.Join(gotIDs, ",") != "claude-code,cursor" {
		t.Errorf("instalación en %q con %v, quiero %q con [claude-code cursor]", gotDir, gotIDs, dir)
	}
	var body struct {
		ConfigPath string          `json:"ConfigPath"`
		Adapters   AdaptersOutcome `json:"adapters"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ConfigPath == "" || body.Adapters.Error != "" || strings.Join(body.Adapters.Installed, ",") != "claude-code,cursor" {
		t.Errorf("respuesta inesperada: %s", rr.Body.String())
	}
}

// Un fallo de la instalación después de escribir el proyecto es parcial: el proyecto
// queda creado, la respuesta es 200 y lleva el error.
func TestProjectsInitKeepsTheProjectWhenAdapterInstallFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proyecto-nuevo")
	router, _, _ := newPrefsServer(t, t.TempDir())
	stubInstallAdapters(t, func(string, []string) ([]string, error) { return nil, errors.New("sin red") })

	rr := request(router, http.MethodPost, "/api/projects/init", `{"path":`+quoteJSON(dir)+`,"adapters":["claude-code"]}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/projects/init = %d, quiero 200 con fallo parcial: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Adapters AdaptersOutcome `json:"adapters"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Adapters.Error != "sin red" || len(body.Adapters.Installed) != 0 || len(body.Adapters.Requested) != 1 {
		t.Errorf("fallo parcial mal informado: %+v", body.Adapters)
	}
	if _, err := os.Stat(filepath.Join(dir, "axiom.yaml")); err != nil {
		t.Errorf("el proyecto debe quedar creado: %v", err)
	}
}

func TestProjectsInitWithoutAdaptersInstallsNothing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proyecto-nuevo")
	router, _, _ := newPrefsServer(t, t.TempDir())
	stubInstallAdapters(t, func(string, []string) ([]string, error) {
		t.Error("sin adaptadores no debe instalarse nada")
		return nil, nil
	})
	rr := request(router, http.MethodPost, "/api/projects/init", `{"path":`+quoteJSON(dir)+`}`)
	if rr.Code != http.StatusOK || strings.Contains(rr.Body.String(), `"adapters"`) {
		t.Fatalf("POST /api/projects/init = %d: %s", rr.Code, rr.Body.String())
	}
}
