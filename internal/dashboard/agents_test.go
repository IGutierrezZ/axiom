package dashboard

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// stubDetectAgents fija los agentes detectados en el equipo.
func stubDetectAgents(t *testing.T, ids ...model.AgentID) {
	t.Helper()
	restore := detectAgentsFn
	detectAgentsFn = func(string) map[model.AgentID]bool {
		out := map[model.AgentID]bool{}
		for _, id := range ids {
			out[id] = true
		}
		return out
	}
	t.Cleanup(func() { detectAgentsFn = restore })
}

func getProjectAgents(t *testing.T, router http.Handler) (ProjectAgentsDTO, map[string]ProjectAgentDTO) {
	t.Helper()
	rr := request(router, http.MethodGet, "/api/agents", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/agents = %d: %s", rr.Code, rr.Body.String())
	}
	var got ProjectAgentsDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	byID := map[string]ProjectAgentDTO{}
	for _, a := range got.Agents {
		byID[a.ID] = a
	}
	return got, byID
}

// Las tres marcas: admitido, detectado en el equipo e instalado en el proyecto.
func TestProjectAgentsListsTheThreeMarks(t *testing.T) {
	root := newProjectDir(t, "agentes")
	if err := state.ProjectStore(sandboxHome, root).Write(state.InstallState{InstalledAgents: []string{"claude-code"}}); err != nil {
		t.Fatal(err)
	}
	stubDetectAgents(t, model.AgentClaudeCode, model.AgentCursor)
	router, _, _ := newPrefsServer(t, root)

	got, byID := getProjectAgents(t, router)
	if !got.Recorded || !got.Project || got.ProjectRoot != root || got.StateError != "" {
		t.Fatalf("cabecera = %+v", got)
	}
	if a := byID["claude-code"]; !a.Supported || !a.Detected || !a.Installed || !a.Workspace || a.Reason != "" || a.Name != "Claude Code" {
		t.Errorf("claude-code = %+v", a)
	}
	if a := byID["cursor"]; !a.Supported || !a.Detected || a.Installed {
		t.Errorf("cursor = %+v", a)
	}
	if a := byID["codex"]; !a.Supported || a.Detected || a.Installed {
		t.Errorf("codex = %+v", a)
	}
	if got.Agents[0].ID != "claude-code" {
		t.Errorf("el primer agente debe ser el primero del catálogo: %q", got.Agents[0].ID)
	}
}

// Sin lista registrada en el proyecto no se hereda la del estado global.
func TestProjectAgentsDoNotInheritTheGlobalList(t *testing.T) {
	if err := state.GlobalStore(sandboxHome).Write(state.InstallState{InstalledAgents: []string{"claude-code"}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(state.Path(sandboxHome)) })
	stubDetectAgents(t)
	router, _, _ := newPrefsServer(t, newProjectDir(t, "sin-lista"))

	got, byID := getProjectAgents(t, router)
	if got.Recorded {
		t.Errorf("recorded debe ser falso sin lista del proyecto")
	}
	if byID["claude-code"].Installed {
		t.Errorf("claude-code no debe figurar instalado: la lista global no cuenta")
	}
}

// Los agentes sin modo workspace y los no seleccionables llevan el motivo.
func TestProjectAgentsExplainWhyAnAgentCannotBeInstalled(t *testing.T) {
	stubDetectAgents(t)
	router, _, _ := newPrefsServer(t, newProjectDir(t, "motivos"))

	_, byID := getProjectAgents(t, router)
	for _, id := range []string{"github-copilot-cli", "pi", "antigravity", "trae-ide"} {
		if a, ok := byID[id]; !ok || !a.Supported || a.Workspace || a.Reason == "" {
			t.Errorf("%s debe ser admitido, sin modo workspace y con motivo: %+v (ok=%v)", id, a, ok)
		}
	}
	if a := byID["qwen-code"]; a.Supported || a.Reason == "" {
		t.Errorf("qwen-code debe figurar como no admitido y con motivo: %+v", a)
	}
}

// Instalar delega en el núcleo de install con la raíz del proyecto y un solo agente.
func TestInstallProjectAgentUsesTheWorkspaceInstall(t *testing.T) {
	root := newProjectDir(t, "instalar")
	var gotDir string
	var gotIDs []string
	stubInstallAdapters(t, func(dir string, ids []string) ([]string, error) {
		gotDir, gotIDs = dir, ids
		return ids, nil
	})
	router, _, _ := newPrefsServer(t, root)

	rr := request(router, http.MethodPost, "/api/agents/install", `{"agent":"cursor"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/agents/install = %d: %s", rr.Code, rr.Body.String())
	}
	var resp AgentInstallResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if !resp.Success || resp.Agent != "cursor" || len(resp.Agents) != 1 {
		t.Errorf("respuesta = %+v", resp)
	}
	if gotDir != root || len(gotIDs) != 1 || gotIDs[0] != "cursor" {
		t.Errorf("installAdapters(%q, %v)", gotDir, gotIDs)
	}
}

// Los rechazos se deciden antes de escribir nada.
func TestInstallProjectAgentRejectsBeforeWriting(t *testing.T) {
	cases := []struct {
		name, root, body string
		want             int
	}{
		{"sin modo workspace", "proyecto", `{"agent":"github-copilot-cli"}`, http.StatusBadRequest},
		{"no seleccionable", "proyecto", `{"agent":"qwen-code"}`, http.StatusBadRequest},
		{"desconocido", "proyecto", `{"agent":"kiro"}`, http.StatusBadRequest},
		{"vacío", "proyecto", `{}`, http.StatusBadRequest},
		{"JSON inválido", "proyecto", `{`, http.StatusBadRequest},
		{"sin proyecto activo", "", `{"agent":"cursor"}`, http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			stubInstallAdapters(t, func(string, []string) ([]string, error) {
				called = true
				return nil, nil
			})
			// Sin proyecto activo: una carpeta sin marcador de workspace.
			root := t.TempDir()
			if tc.root != "" {
				root = newProjectDir(t, tc.root)
			}
			router, _, _ := newPrefsServer(t, root)
			rr := request(router, http.MethodPost, "/api/agents/install", tc.body)
			if rr.Code != tc.want {
				t.Fatalf("código = %d, quiero %d: %s", rr.Code, tc.want, rr.Body.String())
			}
			if called {
				t.Errorf("no debe ejecutarse ninguna instalación")
			}
		})
	}
}

// Un fallo de la instalación responde 500 con el error.
func TestInstallProjectAgentReportsAFailedInstall(t *testing.T) {
	stubInstallAdapters(t, func(string, []string) ([]string, error) {
		return nil, errors.New("verificación fallida")
	})
	router, _, _ := newPrefsServer(t, newProjectDir(t, "fallo"))

	rr := request(router, http.MethodPost, "/api/agents/install", `{"agent":"cursor"}`)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("código = %d: %s", rr.Code, rr.Body.String())
	}
	var resp AgentInstallResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if resp.Success || resp.Error != "verificación fallida" {
		t.Errorf("respuesta = %+v", resp)
	}
}

// Solo el propio dashboard puede instalar.
func TestInstallProjectAgentRejectsForeignOrigins(t *testing.T) {
	stubInstallAdapters(t, func(string, []string) ([]string, error) {
		t.Errorf("no debe ejecutarse ninguna instalación")
		return nil, nil
	})
	router, _, _ := newPrefsServer(t, newProjectDir(t, "origen"))
	rr := requestFrom(router, http.MethodPost, "/api/agents/install", `{"agent":"cursor"}`, "127.0.0.1", "http://evil.example")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("código = %d", rr.Code)
	}
}

// El bloque «Agentes» de Ecosistema: marcado, rutas, confirmación previa y motivo en
// lugar del botón para los agentes que no se pueden instalar en el proyecto.
func TestEcosystemAgentsBlock(t *testing.T) {
	html := readAsset(t, "index.html")
	for _, marker := range []string{`id="eco-agents-body"`, `id="eco-agents-count"`, `id="eco-agents-h"`} {
		if !strings.Contains(html, marker) {
			t.Errorf("index.html no contiene %s", marker)
		}
	}
	js := readAsset(t, "js/screens/ecosystem.js")
	for _, route := range []string{"'/api/agents'", "'/api/agents/install'"} {
		if strings.Count(js, route) != 1 {
			t.Errorf("ecosystem.js debe usar %s en un único punto", route)
		}
	}
	if !strings.Contains(functionBody(t, js, "export async function loadEcosystem("), "loadAgents()") {
		t.Errorf("loadEcosystem debe cargar los agentes")
	}
	load := functionBody(t, js, "async function loadAgents(")
	if !strings.Contains(load, "catch (err)") || !strings.Contains(load, "stateError(") {
		t.Errorf("loadAgents debe capturar su error y mostrarlo con stateError")
	}
	// Las tres marcas son columnas de la tabla.
	for _, col := range []string{"<th>Admitido</th>", "<th>Detectado</th>", "<th>En el proyecto</th>"} {
		if !strings.Contains(load, col) {
			t.Errorf("la tabla de agentes no tiene la columna %s", col)
		}
	}
	// La cabecera de fila (el agente) no hereda las mayúsculas ni el no-salto de las de columna:
	// con ellas la tabla desbordaba su caja a unos 1024 px.
	css := readAsset(t, "style.css")
	rowHead := regexp.MustCompile(`\.eco-agents th\[scope="row"\] \{[^}]*text-transform: none;[^}]*white-space: normal;`)
	if !rowHead.MatchString(css) {
		t.Errorf("style.css debe quitar mayúsculas y no-salto a la cabecera de fila de .eco-agents")
	}
	// Sin modo workspace (o no admitido): el motivo, antes y en lugar del botón.
	action := functionBody(t, js, "function agentAction(")
	reason, button := strings.Index(action, "a.reason"), strings.Index(action, "data-install-agent")
	if !strings.Contains(action, "!a.supported || !a.workspace") || reason < 0 || button < 0 || reason > button {
		t.Errorf("agentAction debe devolver el motivo antes de ofrecer el botón a un agente sin modo workspace")
	}
	// Instalar confirma antes de llamar a la API y avisa de dónde escribe.
	install := functionBody(t, js, "async function installAgent(")
	confirm, call := strings.Index(install, "confirmDialog("), strings.Index(install, "'/api/agents/install'")
	if confirm < 0 || call < 0 || confirm > call || !strings.Contains(install, "if (!ok) return;") {
		t.Errorf("installAgent debe pedir confirmación antes de llamar a la API")
	}
	msg := functionBody(t, js, "function installMessage(")
	for _, want := range []string{"repositorio", ".axiom/state.json", "carpeta de usuario"} {
		if !strings.Contains(msg, want) {
			t.Errorf("el aviso de instalar debe mencionar %q", want)
		}
	}
	// Tras instalar se refrescan la lista y el Doctor.
	if !strings.Contains(install, "loadAgents()") || !strings.Contains(install, "refreshAfterChange()") {
		t.Errorf("installAgent debe refrescar los agentes y el Doctor")
	}
	if !strings.Contains(readAsset(t, "js/core/ecosystem.js"), "install: 'install'") {
		t.Errorf("OPERATION_LABEL debe nombrar la operación install")
	}
}
