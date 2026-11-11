package dashboard

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
)

// reviewModeServer aísla el HOME (el interruptor global es estado de usuario) y monta un
// servidor sobre un proyecto abierto y registrado en un hub temporal.
func reviewModeServer(t *testing.T, register bool) (router http.Handler, root string, mgr *hub.Manager) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AXIOM_STATE_DIR", "")
	root = newProjectDir(t, "abierto")
	router, mgr, _ = newPrefsServer(t, root)
	if register {
		if _, err := mgr.Register(root, "Abierto", ""); err != nil {
			t.Fatal(err)
		}
	}
	return router, root, mgr
}

func decodeReviewMode(t *testing.T, body []byte) ReviewModeDTO {
	t.Helper()
	var dto ReviewModeDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		t.Fatalf("JSON inválido: %v (%s)", err, body)
	}
	return dto
}

// G4.5: el dashboard muestra el modo efectivo y su origen, y fija la preferencia del
// proyecto abierto (on, off, inherit) sin tocar el repositorio.
func TestProjectReviewModeEndpointReadsAndSetsTheProjectPreference(t *testing.T) {
	router, root, mgr := reviewModeServer(t, true)
	yamlBefore := readOrEmpty(t, filepath.Join(root, "axiom.yaml"))

	rr := request(router, http.MethodGet, "/api/project/review-mode", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET = %d: %s", rr.Code, rr.Body.String())
	}
	if dto := decodeReviewMode(t, rr.Body.Bytes()); dto.Effective != "off" || dto.Source != "default" || !dto.Registered || dto.Project != "" {
		t.Fatalf("estado inicial = %+v", dto)
	}

	for _, tc := range []struct {
		mode, effective, source, project string
	}{
		{"on", "on", "project", "on"},
		{"off", "off", "project", "off"},
		{"inherit", "off", "default", ""},
	} {
		rr = request(router, http.MethodPost, "/api/project/review-mode", `{"mode":"`+tc.mode+`"}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("POST %s = %d: %s", tc.mode, rr.Code, rr.Body.String())
		}
		dto := decodeReviewMode(t, rr.Body.Bytes())
		if dto.Effective != tc.effective || dto.Source != tc.source || dto.Project != tc.project {
			t.Errorf("tras %s: %+v", tc.mode, dto)
		}
		if saved, _, err := mgr.LookupProjectRDD(root); err != nil || saved != tc.project {
			t.Errorf("tras %s el hub guarda %q (err=%v)", tc.mode, saved, err)
		}
	}

	if got := readOrEmpty(t, filepath.Join(root, "axiom.yaml")); string(got) != string(yamlBefore) {
		t.Errorf("la preferencia no debe cambiar axiom.yaml:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".axiom", "state.json")); !os.IsNotExist(err) {
		t.Errorf("la preferencia no debe escribir .axiom/state.json (err=%v)", err)
	}
}

func TestProjectReviewModeEndpointRejectsBadInput(t *testing.T) {
	router, _, _ := reviewModeServer(t, true)
	if rr := request(router, http.MethodPost, "/api/project/review-mode", `{"mode":"quizás"}`); rr.Code != http.StatusBadRequest {
		t.Errorf("modo inválido = %d, quiero 400: %s", rr.Code, rr.Body.String())
	}
	if rr := request(router, http.MethodPost, "/api/project/review-mode", `no es json`); rr.Code != http.StatusBadRequest {
		t.Errorf("JSON inválido = %d, quiero 400", rr.Code)
	}
	if rr := request(router, http.MethodPut, "/api/project/review-mode", `{}`); rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("PUT = %d, quiero 405", rr.Code)
	}
}

func TestProjectReviewModeEndpointWithAnUnregisteredProject(t *testing.T) {
	router, _, _ := reviewModeServer(t, false)
	rr := request(router, http.MethodGet, "/api/project/review-mode", "")
	if rr.Code != http.StatusOK || decodeReviewMode(t, rr.Body.Bytes()).Registered {
		t.Fatalf("GET = %d: %s", rr.Code, rr.Body.String())
	}
	rr = request(router, http.MethodPost, "/api/project/review-mode", `{"mode":"on"}`)
	if rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "no está registrado") {
		t.Fatalf("POST sin registrar = %d, quiero 409 claro: %s", rr.Code, rr.Body.String())
	}
}

// La raíz es la del proyecto abierto, no el directorio actual del proceso.
func TestProjectReviewModeEndpointIgnoresTheProcessCwd(t *testing.T) {
	router, _, _ := reviewModeServer(t, true)
	t.Chdir(t.TempDir())
	rr := request(router, http.MethodPost, "/api/project/review-mode", `{"mode":"on"}`)
	if rr.Code != http.StatusOK || decodeReviewMode(t, rr.Body.Bytes()).Source != "project" {
		t.Fatalf("POST = %d: %s", rr.Code, rr.Body.String())
	}
}

// Las rutas que cambian estado solo las acepta el propio dashboard: Host y Origin ajenos
// reciben 403 y no se escribe nada.
func TestStateChangingProjectRoutesRejectForeignOriginAndHost(t *testing.T) {
	router, _, mgr := reviewModeServer(t, true)
	newDir := filepath.Join(t.TempDir(), "proyecto-nuevo")
	routes := []struct{ name, target, body string }{
		{"alta", "/api/projects/init", `{"path":` + quoteJSON(newDir) + `}`},
		{"modo de revisión", "/api/project/review-mode", `{"mode":"on"}`},
	}
	cases := []struct {
		name, host, origin string
		want               int
	}{
		{"Origin ajeno", "127.0.0.1:8123", "https://evil.example", http.StatusForbidden},
		{"Origin null", "127.0.0.1:8123", "null", http.StatusForbidden},
		{"Host ajeno sin Origin", "evil.example:8123", "", http.StatusForbidden},
		{"Host ajeno con Origin local", "evil.example", "http://127.0.0.1:8123", http.StatusForbidden},
	}
	for _, route := range routes {
		for _, tc := range cases {
			t.Run(route.name+"/"+tc.name, func(t *testing.T) {
				rr := requestFrom(router, http.MethodPost, route.target, route.body, tc.host, tc.origin)
				if rr.Code != tc.want {
					t.Fatalf("%s %s = %d, quiero %d: %s", tc.host, tc.origin, rr.Code, tc.want, rr.Body.String())
				}
			})
		}
	}
	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		t.Errorf("el alta rechazada no debe crear %s (err=%v)", newDir, err)
	}
	if mode, _, _ := mgr.LookupProjectRDD(filepath.Join(newDir)); mode != "" {
		t.Errorf("no debe guardarse nada: %q", mode)
	}
	if saved, _, _ := mgr.LookupProjectRDD(t.TempDir()); saved != "" {
		t.Errorf("no debe guardarse nada: %q", saved)
	}

	// Con Host y Origin del propio dashboard sí se acepta.
	rr := requestFrom(router, http.MethodPost, "/api/project/review-mode", `{"mode":"on"}`, "localhost:8123", "http://localhost:8123")
	if rr.Code != http.StatusOK {
		t.Errorf("petición local = %d: %s", rr.Code, rr.Body.String())
	}
}

// El GET es de solo lectura y no se protege: lo ve cualquier página del dashboard.
func TestProjectReviewModeGetIsReadOnly(t *testing.T) {
	router, _, mgr := reviewModeServer(t, true)
	rr := requestFrom(router, http.MethodGet, "/api/project/review-mode", "", "127.0.0.1:8123", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET = %d", rr.Code)
	}
	if saved, _, _ := mgr.LookupProjectRDD(t.TempDir()); saved != "" {
		t.Errorf("GET no debe modificar nada: %q", saved)
	}
}

// En el alta, la preferencia se guarda en el hub después de escribir y registrar el
// proyecto; un valor inválido se rechaza antes de escribir nada.
func TestProjectsInitStoresTheChosenReviewModeInTheHub(t *testing.T) {
	_, _, _ = reviewModeServer(t, true)
	router, mgr, _ := newPrefsServer(t, t.TempDir())
	dir := filepath.Join(t.TempDir(), "proyecto-nuevo")

	rr := request(router, http.MethodPost, "/api/projects/init", `{"path":`+quoteJSON(dir)+`,"review_mode":"on"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/projects/init = %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		ReviewModeError string `json:"review_mode_error"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body.ReviewModeError != "" {
		t.Errorf("review_mode_error = %q", body.ReviewModeError)
	}
	if mode, record, err := mgr.LookupProjectRDD(dir); err != nil || mode != "on" || record == nil {
		t.Fatalf("el hub debe guardar on: %q %v %v", mode, record, err)
	}
	if got := readOrEmpty(t, filepath.Join(dir, "axiom.yaml")); strings.Contains(string(got), "review") || strings.Contains(string(got), "rdd") {
		t.Errorf("axiom.yaml no debe llevar la preferencia:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(dir, ".axiom", "state.json")); !os.IsNotExist(err) {
		t.Errorf("el alta no debe escribir .axiom/state.json con la preferencia (err=%v)", err)
	}

	// Heredar (o no indicar nada) no guarda preferencia.
	other := filepath.Join(t.TempDir(), "otro")
	rr = request(router, http.MethodPost, "/api/projects/init", `{"path":`+quoteJSON(other)+`,"review_mode":"inherit"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("inherit = %d: %s", rr.Code, rr.Body.String())
	}
	if mode, _, _ := mgr.LookupProjectRDD(other); mode != "" {
		t.Errorf("inherit no guarda preferencia: %q", mode)
	}
}

func TestProjectsInitRejectsAnInvalidReviewModeBeforeWriting(t *testing.T) {
	router, mgr, _ := newPrefsServer(t, t.TempDir())
	dir := filepath.Join(t.TempDir(), "proyecto-nuevo")
	rr := request(router, http.MethodPost, "/api/projects/init", `{"path":`+quoteJSON(dir)+`,"review_mode":"quizás","roles":[{"key":"a","name":"A","repositories":["../fuera"]}]}`)
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "no es válido") || !strings.Contains(rr.Body.String(), "sale de la carpeta del proyecto") {
		t.Fatalf("POST = %d, quiero 400 con ambas violaciones: %s", rr.Code, rr.Body.String())
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("no debe crear %s (err=%v)", dir, err)
	}
	if list, _ := mgr.List(); len(list) != 0 {
		t.Errorf("no debe registrar nada: %v", list)
	}
}
