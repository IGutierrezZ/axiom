package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/opencode"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// modelsProject monta un proyecto abierto con el estado de instalación de agents y un
// HOME aislado, y sustituye el catálogo de OpenCode por uno fijo.
func modelsProject(t *testing.T, st state.InstallState) (router http.Handler, root, home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("AXIOM_STATE_DIR", "")
	root = newProjectDir(t, "abierto")
	if err := state.ProjectStore(home, root).Write(st); err != nil {
		t.Fatal(err)
	}
	previous := discoverOpenCodeCatalog
	discoverOpenCodeCatalog = func(context.Context, string) (map[string]opencode.Provider, error) {
		return map[string]opencode.Provider{
			"anthropic": {ID: "anthropic", Name: "Anthropic", Models: map[string]opencode.Model{
				"claude-sonnet-4":  {ID: "claude-sonnet-4", Name: "Claude Sonnet 4", ToolCall: true, Variants: []string{"low", "high"}},
				"sin-herramientas": {ID: "sin-herramientas", Name: "Sin herramientas"},
			}},
			"openai": {ID: "openai", Models: map[string]opencode.Model{
				"gpt-5": {ID: "gpt-5", Name: "GPT-5", ToolCall: true},
			}},
		}, nil
	}
	t.Cleanup(func() { discoverOpenCodeCatalog = previous })
	router, _, _ = newPrefsServer(t, root)
	return router, root, home
}

func getModels(t *testing.T, router http.Handler) ModelAssignmentsDTO {
	t.Helper()
	rr := request(router, http.MethodGet, "/api/ecosystem/models", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET models = %d: %s", rr.Code, rr.Body.String())
	}
	var dto ModelAssignmentsDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &dto); err != nil {
		t.Fatalf("JSON inválido: %v (%s)", err, rr.Body.String())
	}
	return dto
}

func blockByID(t *testing.T, dto ModelAssignmentsDTO, id string) AdapterModelsDTO {
	t.Helper()
	for _, block := range dto.Adapters {
		if block.ID == id {
			return block
		}
	}
	t.Fatalf("no hay bloque %q en %+v", id, dto.Adapters)
	return AdapterModelsDTO{}
}

func TestEcosystemModelsGetReturnsOneBlockPerInstalledAdapter(t *testing.T) {
	router, _, _ := modelsProject(t, state.InstallState{
		InstalledAgents: []string{"opencode", "claude-code", "vscode-copilot", "github-copilot-cli", "cursor"},
		SDDMode:         model.SDDModeMulti,
		ModelAssignments: map[string]state.ModelAssignmentState{
			"sdd-apply": {ProviderID: "anthropic", ModelID: "claude-sonnet-4", Effort: "high"},
		},
		ClaudePhaseAssignments:  map[string]state.ClaudePhaseAssignmentState{"sdd-design": {Model: "opus", Effort: "max"}},
		CopilotPhaseAssignments: map[string]state.CopilotPhaseAssignmentState{"sdd-verify": {Model: "GPT-5.2 (copilot)", Effort: "high"}},
	})
	dto := getModels(t, router)
	if dto.ActivePersona == "" || dto.Assignments == nil {
		t.Errorf("los campos de siempre deben seguir: %+v", dto)
	}
	if len(dto.Adapters) != 3 {
		t.Fatalf("quiero 3 bloques (OpenCode, Claude, Copilot), hay %d: %+v", len(dto.Adapters), dto.Adapters)
	}
	for i, id := range []string{"opencode", "claude-code", "copilot"} {
		if dto.Adapters[i].ID != id {
			t.Errorf("bloque %d = %q, quiero %q", i, dto.Adapters[i].ID, id)
		}
	}

	oc := blockByID(t, dto, "opencode")
	if got := oc.Assignments["sdd-apply"]; got.Model != "anthropic/claude-sonnet-4" || got.Effort != "high" {
		t.Errorf("OpenCode sdd-apply = %+v", got)
	}
	if !slices.Contains(oc.Phases, "sdd-init") || !slices.Contains(oc.Phases, "jd-judge-a") {
		t.Errorf("fases de OpenCode = %v", oc.Phases)
	}
	if len(oc.Options.Providers) != 2 || oc.Options.CatalogError != "" {
		t.Fatalf("catálogo = %+v", oc.Options)
	}
	anthropic := oc.Options.Providers[0]
	if anthropic.ID != "anthropic" || len(anthropic.Models) != 1 || anthropic.Models[0].ID != "claude-sonnet-4" || strings.Join(anthropic.Models[0].Variants, ",") != "low,high" {
		t.Errorf("proveedor anthropic = %+v (solo modelos con herramientas)", anthropic)
	}
	if len(oc.Notes) != 1 {
		t.Errorf("en modo multi solo debe avisar de que quitar no borra opencode.json: %v", oc.Notes)
	}

	cl := blockByID(t, dto, "claude-code")
	if got := cl.Assignments["sdd-design"]; got.Model != "opus" || got.Effort != "max" {
		t.Errorf("Claude sdd-design = %+v", got)
	}
	if cl.Defaults["sdd-propose"].Model != "opus" || cl.Defaults["default"].Model != "sonnet" {
		t.Errorf("valores por defecto de Claude = %+v", cl.Defaults)
	}
	if slices.Contains(cl.Phases, "orchestrator") || !slices.Contains(cl.Phases, "default") {
		t.Errorf("fases de Claude = %v", cl.Phases)
	}
	var haiku, opus *EffortModelOptionDTO
	for i := range cl.Options.Models {
		switch cl.Options.Models[i].ID {
		case "haiku":
			haiku = &cl.Options.Models[i]
		case "opus":
			opus = &cl.Options.Models[i]
		}
	}
	if haiku == nil || len(haiku.Efforts) != 0 || opus == nil || !slices.Contains(opus.Efforts, "max") {
		t.Errorf("esfuerzos por modelo = %+v", cl.Options.Models)
	}

	cp := blockByID(t, dto, "copilot")
	if got := cp.Assignments["sdd-verify"]; got.Model != "GPT-5.2 (copilot)" || got.Effort != "high" {
		t.Errorf("Copilot sdd-verify = %+v", got)
	}
	want := model.CopilotPhaseNames()
	if strings.Join(cp.Phases, ",") != strings.Join(want, ",") {
		t.Errorf("fases de Copilot = %v, quiero %v", cp.Phases, want)
	}
	if !cp.Options.FreeText || len(cp.Options.Suggestions) == 0 {
		t.Errorf("Copilot debe admitir texto libre con sugerencias: %+v", cp.Options)
	}
	if len(cp.Destinations) != 1 || cp.Destinations[0] != ".github/agents" || len(cp.Notes) != 1 {
		t.Errorf("destino y avisos de Copilot = %v %v", cp.Destinations, cp.Notes)
	}
}

func TestEcosystemModelsGetOmitsAdaptersThatAreNotInstalledAndDegradesWithoutCatalog(t *testing.T) {
	router, _, _ := modelsProject(t, state.InstallState{InstalledAgents: []string{"opencode", "cursor"}})
	previous := discoverOpenCodeCatalog
	discoverOpenCodeCatalog = func(context.Context, string) (map[string]opencode.Provider, error) {
		return nil, &opencode.CatalogError{Kind: "unavailable"}
	}
	t.Cleanup(func() { discoverOpenCodeCatalog = previous })

	dto := getModels(t, router)
	if len(dto.Adapters) != 1 || dto.Adapters[0].ID != "opencode" {
		t.Fatalf("solo debe haber OpenCode: %+v", dto.Adapters)
	}
	oc := dto.Adapters[0]
	if oc.Options.CatalogError == "" || len(oc.Options.Providers) != 0 {
		t.Errorf("sin catálogo debe llevar catalog_error: %+v", oc.Options)
	}
	if len(oc.Notes) != 2 || !strings.Contains(oc.Notes[0], "multi") {
		t.Errorf("en modo single debe avisar de que no aplica: %v", oc.Notes)
	}
	if oc.Assignments == nil {
		t.Error("assignments debe ser un objeto, no null")
	}
}

func TestEcosystemModelsGetWithoutProjectStateHasNoBlocks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	router, _, _ := newPrefsServer(t, newProjectDir(t, "vacio"))
	dto := getModels(t, router)
	if len(dto.Adapters) != 0 || dto.AdaptersError != "" {
		t.Errorf("sin estado no hay bloques ni error: %+v", dto)
	}
	if !strings.Contains(request(router, http.MethodGet, "/api/ecosystem/models", "").Body.String(), `"adapters":[]`) {
		t.Error("adapters debe ser [] y no null")
	}
}

func postModels(router http.Handler, body string) (int, map[string]json.RawMessage) {
	rr := request(router, http.MethodPost, "/api/ecosystem/models", body)
	out := map[string]json.RawMessage{}
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	return rr.Code, out
}

func readProjectState(t *testing.T, home, root string) state.InstallState {
	t.Helper()
	st, err := state.ProjectStore(home, root).Read()
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestEcosystemModelsPostClaudeSavesStateAndRegeneratesAgentFiles(t *testing.T) {
	router, root, home := modelsProject(t, state.InstallState{InstalledAgents: []string{"claude-code"}})
	// La raíz es la del proyecto abierto, no la carpeta actual del proceso.
	t.Chdir(t.TempDir())

	code, body := postModels(router, `{"adapter":"claude-code","assignments":{"sdd-apply":{"model":"opus","effort":"high"},"sdd-archive":{"model":"haiku"}}}`)
	if code != http.StatusOK {
		t.Fatalf("POST = %d: %s", code, body)
	}
	st := readProjectState(t, home, root)
	if got := st.ClaudePhaseAssignments["sdd-apply"]; got.Model != "opus" || got.Effort != "high" {
		t.Errorf("estado sdd-apply = %+v", got)
	}
	var regenerated []string
	_ = json.Unmarshal(body["regenerated"], &regenerated)
	if len(regenerated) == 0 {
		t.Fatalf("debe listar los ficheros regenerados: %s", body["regenerated"])
	}
	apply, err := os.ReadFile(filepath.Join(root, ".claude", "agents", "sdd-apply.md"))
	if err != nil {
		t.Fatalf("el agente de Claude debe regenerarse en el proyecto: %v", err)
	}
	if !strings.Contains(string(apply), "model: opus") || !strings.Contains(string(apply), "effort: high") {
		t.Errorf("sdd-apply.md sin el modelo elegido:\n%s", apply)
	}
	for _, rel := range regenerated {
		if filepath.IsAbs(rel) || strings.Contains(rel, `\`) {
			t.Errorf("las rutas regeneradas son relativas con barras normales: %q", rel)
		}
	}

	// Quitar una fase la vuelve a su valor por defecto y no toca las demás.
	code, body = postModels(router, `{"adapter":"claude-code","assignments":{"sdd-apply":{"model":""}}}`)
	if code != http.StatusOK {
		t.Fatalf("POST de quitar = %d: %s", code, body)
	}
	st = readProjectState(t, home, root)
	if _, ok := st.ClaudePhaseAssignments["sdd-apply"]; ok {
		t.Errorf("sdd-apply debe quitarse del estado: %+v", st.ClaudePhaseAssignments)
	}
	if st.ClaudePhaseAssignments["sdd-archive"].Model != "haiku" {
		t.Errorf("las demás fases se conservan: %+v", st.ClaudePhaseAssignments)
	}
	apply, _ = os.ReadFile(filepath.Join(root, ".claude", "agents", "sdd-apply.md"))
	if strings.Contains(string(apply), "model: opus") || strings.Contains(string(apply), "effort: high") {
		t.Errorf("sdd-apply.md debe volver al valor por defecto:\n%s", apply)
	}
}

func TestEcosystemModelsPostCopilotWritesTheSharedAgentFiles(t *testing.T) {
	router, root, home := modelsProject(t, state.InstallState{InstalledAgents: []string{"vscode-copilot", "github-copilot-cli"}})

	code, body := postModels(router, `{"adapter":"copilot","assignments":{"sdd-apply":{"model":"Claude Sonnet 4.6 (copilot)","effort":"high"}}}`)
	if code != http.StatusOK {
		t.Fatalf("POST = %d: %s", code, body)
	}
	st := readProjectState(t, home, root)
	if got := st.CopilotPhaseAssignments["sdd-apply"]; got.Model != "Claude Sonnet 4.6 (copilot)" || got.Effort != "high" {
		t.Errorf("estado = %+v", st.CopilotPhaseAssignments)
	}
	agent, err := os.ReadFile(filepath.Join(root, ".github", "agents", "sdd-apply.agent.md"))
	if err != nil {
		t.Fatalf("el agente de Copilot debe regenerarse en el proyecto: %v", err)
	}
	if !strings.Contains(string(agent), "Claude Sonnet 4.6 (copilot)") || !strings.Contains(string(agent), "reasoning-effort") {
		t.Errorf("sdd-apply.agent.md sin el modelo elegido:\n%s", agent)
	}
	var notes []string
	_ = json.Unmarshal(body["notes"], &notes)
	if len(notes) != 0 {
		t.Errorf("con VS Code Copilot y Copilot CLI no hay nada que avisar: los agentes viven solo en el proyecto: %v", notes)
	}

	// Quitar la asignación es explícito y la elimina del estado y del fichero.
	code, body = postModels(router, `{"adapter":"copilot","assignments":{"sdd-apply":{"model":" "}}}`)
	if code != http.StatusOK {
		t.Fatalf("POST de quitar = %d: %s", code, body)
	}
	if got := readProjectState(t, home, root).CopilotPhaseAssignments; len(got) != 0 {
		t.Errorf("el estado debe quedar sin asignaciones: %+v", got)
	}
	agent, _ = os.ReadFile(filepath.Join(root, ".github", "agents", "sdd-apply.agent.md"))
	if strings.Contains(string(agent), "Claude Sonnet 4.6") {
		t.Errorf("el agente debe quedar sin modelo:\n%s", agent)
	}
}

func TestEcosystemModelsPostCopilotCLIOnlyRegeneratesTheProjectAgents(t *testing.T) {
	router, root, home := modelsProject(t, state.InstallState{InstalledAgents: []string{"github-copilot-cli"}})
	code, body := postModels(router, `{"adapter":"copilot","assignments":{"sdd-spec":{"model":"GPT-5.2 (copilot)"}}}`)
	if code != http.StatusOK {
		t.Fatalf("POST = %d: %s", code, body)
	}
	if got := readProjectState(t, home, root).CopilotPhaseAssignments["sdd-spec"].Model; got != "GPT-5.2 (copilot)" {
		t.Errorf("el estado debe guardarse: %q", got)
	}
	// Copilot CLI lee .github/agents del proyecto y no tiene modo workspace: el sync del
	// proyecto los escribe igualmente, y no se toca la carpeta de usuario.
	agent, err := os.ReadFile(filepath.Join(root, ".github", "agents", "sdd-spec.agent.md"))
	if err != nil {
		t.Fatalf("con solo Copilot CLI debe regenerarse .github/agents: %v", err)
	}
	if !strings.Contains(string(agent), "model: GPT-5.2 (copilot)") {
		t.Errorf("sdd-spec.agent.md sin el modelo elegido:\n%s", agent)
	}
	if !strings.Contains(string(body["regenerated"]), ".github/agents/sdd-spec.agent.md") {
		t.Errorf("la respuesta debe listar los agentes regenerados: %s", body["regenerated"])
	}
	if _, err := os.Stat(filepath.Join(home, ".copilot", "agents")); !os.IsNotExist(err) {
		t.Errorf("no se debe escribir en ~/.copilot/agents (err=%v)", err)
	}
}

func TestEcosystemModelsPostOpenCodeSavesStateAndConfig(t *testing.T) {
	// El sync de OpenCode sondea su versión: se simula una 1.x instalada.
	previousRunner := opencode.VersionRunnerOverride
	opencode.VersionRunnerOverride = func(context.Context, opencode.Command) (opencode.CommandOutput, error) {
		return opencode.CommandOutput{Stdout: []byte("1.4.0")}, nil
	}
	t.Cleanup(func() { opencode.VersionRunnerOverride = previousRunner })
	router, root, home := modelsProject(t, state.InstallState{InstalledAgents: []string{"opencode"}, SelectionConfigured: true, Components: []model.ComponentID{model.ComponentPersona, model.ComponentSDD}, Preset: model.PresetFullGentleman, SDDMode: model.SDDModeMulti})
	code, body := postModels(router, `{"adapter":"opencode","assignments":{"sdd-apply":{"model":"anthropic/claude-sonnet-4","effort":"low"}}}`)
	if code != http.StatusOK {
		t.Fatalf("POST = %d: %s", code, body)
	}
	if got := readProjectState(t, home, root).ModelAssignments["sdd-apply"]; got.ProviderID != "anthropic" || got.ModelID != "claude-sonnet-4" || got.Effort != "low" {
		t.Errorf("estado = %+v", got)
	}
	config, err := os.ReadFile(filepath.Join(root, "opencode.json"))
	if err != nil {
		t.Fatalf("opencode.json debe regenerarse en el proyecto: %v", err)
	}
	if !strings.Contains(string(config), "anthropic/claude-sonnet-4") {
		t.Errorf("opencode.json sin el modelo elegido:\n%s", config)
	}
	if dto := getModels(t, router); blockByID(t, dto, "opencode").Assignments["sdd-apply"].Effort != "low" {
		t.Errorf("el GET debe reflejar lo guardado: %+v", dto.Adapters)
	}

	// Quitar la fase la olvida en el estado; opencode.json conserva el modelo (el aviso lo dice).
	if code, body = postModels(router, `{"adapter":"opencode","assignments":{"sdd-apply":{"model":""}}}`); code != http.StatusOK {
		t.Fatalf("POST de quitar = %d: %s", code, body)
	}
	if got := readProjectState(t, home, root).ModelAssignments; len(got) != 0 {
		t.Errorf("el estado debe quedar sin asignaciones: %+v", got)
	}
}

func TestEcosystemModelsPostRejectsInvalidInputWithViolationsAndWritesNothing(t *testing.T) {
	router, root, home := modelsProject(t, state.InstallState{InstalledAgents: []string{"opencode", "claude-code", "vscode-copilot"}, SDDMode: model.SDDModeMulti})
	before := readProjectState(t, home, root)

	cases := []struct {
		name, body string
		want       []string
	}{
		{"adaptador desconocido", `{"adapter":"kiro","assignments":{}}`, []string{"no existe"}},
		{"alias de Claude inválido", `{"adapter":"claude-code","assignments":{"sdd-apply":{"model":"gpt"},"nope":{"model":"opus"}}}`, []string{"sdd-apply", "fases desconocidas: nope"}},
		{"esfuerzo no admitido por el modelo", `{"adapter":"claude-code","assignments":{"sdd-apply":{"model":"haiku","effort":"high"}}}`, []string{"esfuerzo"}},
		{"Copilot con salto de línea", `{"adapter":"copilot","assignments":{"sdd-apply":{"model":"GPT\ntools: []"},"orchestrator":{"model":"x"}}}`, []string{"caracteres de control", "orchestrator"}},
		{"OpenCode sin proveedor", `{"adapter":"opencode","assignments":{"sdd-apply":{"model":"claude"}}}`, []string{"proveedor/modelo"}},
		{"OpenCode fuera del catálogo", `{"adapter":"opencode","assignments":{"sdd-apply":{"model":"anthropic/no-existe"},"sdd-spec":{"model":"nube/x"}}}`, []string{"no-existe", "nube"}},
		{"OpenCode con variante inválida", `{"adapter":"opencode","assignments":{"sdd-apply":{"model":"anthropic/claude-sonnet-4","effort":"ultra"},"sdd-spec":{"model":"openai/gpt-5","effort":"low"}}}`, []string{"ultra", "no admite variantes"}},
		{"OpenCode con fase desconocida", `{"adapter":"opencode","assignments":{"fase":{"model":"openai/gpt-5"}}}`, []string{"fases desconocidas: fase"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := request(router, http.MethodPost, "/api/ecosystem/models", tc.body)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("POST = %d, quiero 400: %s", rr.Code, rr.Body.String())
			}
			var resp struct {
				Error      string   `json:"error"`
				Violations []string `json:"violations"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil || resp.Error == "" {
				t.Fatalf("cuerpo del 400 = %s (%v)", rr.Body.String(), err)
			}
			joined := strings.Join(resp.Violations, "\n")
			for _, want := range tc.want {
				if !strings.Contains(joined, want) {
					t.Errorf("las violaciones deben mencionar %q:\n%s", want, joined)
				}
			}
			if len(resp.Violations) < len(tc.want) {
				t.Errorf("deben llegar todas las violaciones a la vez: %v", resp.Violations)
			}
		})
	}

	after := readProjectState(t, home, root)
	if len(after.ClaudePhaseAssignments)+len(after.CopilotPhaseAssignments)+len(after.ModelAssignments) != 0 || len(before.InstalledAgents) != len(after.InstalledAgents) {
		t.Errorf("una petición inválida no debe cambiar el estado: %+v", after)
	}
	for _, dir := range []string{".claude", ".github", "opencode.json"} {
		if _, err := os.Stat(filepath.Join(root, dir)); !os.IsNotExist(err) {
			t.Errorf("una petición inválida no debe escribir %s (err=%v)", dir, err)
		}
	}
	if rr := request(router, http.MethodPost, "/api/ecosystem/models", `no es json`); rr.Code != http.StatusBadRequest {
		t.Errorf("JSON inválido = %d, quiero 400", rr.Code)
	}
	if rr := request(router, http.MethodPut, "/api/ecosystem/models", `{}`); rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("PUT = %d, quiero 405", rr.Code)
	}
}

func TestEcosystemModelsPostRejectsForeignOriginAndHost(t *testing.T) {
	router, root, home := modelsProject(t, state.InstallState{InstalledAgents: []string{"claude-code"}})
	body := `{"adapter":"claude-code","assignments":{"sdd-apply":{"model":"opus"}}}`
	for _, tc := range []struct{ name, host, origin string }{
		{"Origin ajeno", "127.0.0.1:8123", "https://evil.example"},
		{"Origin null", "127.0.0.1:8123", "null"},
		{"Host ajeno sin Origin", "evil.example:8123", ""},
		{"Host ajeno con Origin local", "evil.example", "http://127.0.0.1:8123"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if rr := requestFrom(router, http.MethodPost, "/api/ecosystem/models", body, tc.host, tc.origin); rr.Code != http.StatusForbidden {
				t.Fatalf("%s %s = %d, quiero 403: %s", tc.host, tc.origin, rr.Code, rr.Body.String())
			}
		})
	}
	if got := readProjectState(t, home, root).ClaudePhaseAssignments; len(got) != 0 {
		t.Errorf("no debe guardarse nada: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Errorf("no debe escribirse nada en el proyecto (err=%v)", err)
	}
	// El GET es de solo lectura y no se protege.
	if rr := requestFrom(router, http.MethodGet, "/api/ecosystem/models", "", "127.0.0.1:8123", ""); rr.Code != http.StatusOK {
		t.Errorf("GET = %d", rr.Code)
	}
}

func TestEcosystemModelsPostWithoutAnOpenProjectIs409(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	missing := filepath.Join(t.TempDir(), "cerrado")
	// El servicio apunta a una carpeta que ya no existe: no hay proyecto abierto.
	router := NewServer(NewService(missing)).Router()
	rr := request(router, http.MethodPost, "/api/ecosystem/models", `{"adapter":"claude-code","assignments":{}}`)
	if rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "no hay un proyecto abierto") {
		t.Fatalf("POST sin proyecto = %d, quiero 409: %s", rr.Code, rr.Body.String())
	}
}

func TestEcosystemModelsPostRequiresTheAdapterToBeInstalledInTheProject(t *testing.T) {
	router, _, _ := modelsProject(t, state.InstallState{InstalledAgents: []string{"opencode"}})
	for _, adapter := range []string{"claude-code", "copilot"} {
		rr := request(router, http.MethodPost, "/api/ecosystem/models", `{"adapter":"`+adapter+`","assignments":{"sdd-apply":{"model":"opus"}}}`)
		if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "no está instalado") {
			t.Errorf("%s: POST = %d, quiero 400 «no está instalado»: %s", adapter, rr.Code, rr.Body.String())
		}
	}
}
