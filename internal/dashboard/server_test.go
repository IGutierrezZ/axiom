package dashboard

import (
	"bytes"
	"encoding/json"
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

func request(router http.Handler, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, bytes.NewReader([]byte(body)))
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

func TestRegistrationModalAsksWorktreePrefs(t *testing.T) {
	router, _, _ := newPrefsServer(t, newProjectDir(t, "assets"))

	html := request(router, http.MethodGet, "/", "").Body.String()
	selectBlock := regexp.MustCompile(`(?s)<select id="select-project-worktree-mode".*?</select>`).FindString(html)
	if selectBlock == "" {
		t.Fatal("index.html: el modal de registro no tiene el selector de modo `select-project-worktree-mode`")
	}
	for _, option := range []string{`value="worktree" selected`, `value="in_place"`, `value="ask"`} {
		if !strings.Contains(selectBlock, option) {
			t.Errorf("el selector de modo debe contener %q:\n%s", option, selectBlock)
		}
	}
	if got := strings.Count(selectBlock, "<option"); got != 3 {
		t.Errorf("el selector de modo tiene %d opciones, quiero 3", got)
	}
	rootInput := regexp.MustCompile(`<input[^>]*id="input-project-worktree-root"[^>]*>`).FindString(html)
	if rootInput == "" {
		t.Fatal("index.html: el modal de registro no tiene el campo `input-project-worktree-root`")
	}
	if !strings.Contains(rootInput, "placeholder=") || !strings.Contains(rootInput, "por defecto") || !strings.Contains(rootInput, "-wt") {
		t.Errorf("el campo de raíz debe mostrar su valor por defecto en el placeholder: %s", rootInput)
	}
	for _, marker := range []string{"input-worktree-prefs-root", "select-worktree-prefs-mode", "btn-save-worktree-prefs"} {
		if !strings.Contains(html, marker) {
			t.Errorf("index.html debe contener el editor de preferencias del proyecto activo (%s)", marker)
		}
	}
	if strings.Contains(html, "integration_branch") || strings.Contains(html, "branch_prefix") {
		t.Error("index.html no debe ofrecer controles de edición de `git:` (REQ-25.33)")
	}

	js := request(router, http.MethodGet, "/app.js", "").Body.String()
	for _, marker := range []string{"worktree_mode:", "worktree_root:", "/api/projects/active/worktree-prefs", "defaultWorktreeRoot"} {
		if !strings.Contains(js, marker) {
			t.Errorf("app.js debe contener %q", marker)
		}
	}
	// La petición de alta del modal envía las preferencias elegidas.
	submit := regexp.MustCompile(`(?s)fetch\('/api/projects/init',.*?roles: roles.*?\}\)`).FindString(js)
	if !strings.Contains(submit, "worktree_mode") || !strings.Contains(submit, "worktree_root") {
		t.Errorf("la petición de alta del modal debe enviar worktree_mode y worktree_root:\n%s", submit)
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
