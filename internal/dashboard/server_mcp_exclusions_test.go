package dashboard

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/mcpexclusion"
)

func decodeMCPExclusions(t *testing.T, body []byte) MCPExclusionsDTO {
	t.Helper()
	var dto MCPExclusionsDTO
	if err := json.Unmarshal(body, &dto); err != nil {
		t.Fatalf("JSON inválido: %v (%s)", err, body)
	}
	return dto
}

func deniedServerNames(t *testing.T, dir string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.local.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Denied []struct {
			ServerName string `json:"serverName"`
		} `json:"deniedMcpServers"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range doc.Denied {
		names = append(names, entry.ServerName)
	}
	return names
}

// El GET ofrece como candidatos los MCP de usuario de ~/.claude.json (solo nombres) y los
// ya excluidos; el POST guarda en el hub, escribe .claude/settings.local.json en el
// proyecto e informa de cada directorio.
func TestActiveMCPExclusionsEndpointReadsAndAppliesThePreference(t *testing.T) {
	router, root, mgr := reviewModeServer(t, true)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	userConfig := `{"oauthAccount":{"token":"canario-oauth"},"mcpServers":{"work-copilot":{"command":"x"},"engram":{"command":"y"}}}`
	if err := os.WriteFile(claude.UserConfigPath(home), []byte(userConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	rr := request(router, http.MethodGet, "/api/projects/active/mcp-exclusions", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET = %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "canario-oauth") {
		t.Errorf("el GET no debe devolver nada de ~/.claude.json salvo los nombres: %s", rr.Body.String())
	}
	dto := decodeMCPExclusions(t, rr.Body.Bytes())
	if !reflect.DeepEqual(dto.Candidates, []string{"engram", "work-copilot"}) || len(dto.Exclusions) != 0 || len(dto.Dirs) == 0 {
		t.Fatalf("estado inicial = %+v", dto)
	}
	if len(dto.Agents) != 1 || dto.Agents[0].ID != "claude-code" || !dto.Agents[0].Supported {
		t.Errorf("agentes = %+v", dto.Agents)
	}

	body := `{"exclusions":{"claude-code":["claude.ai Microsoft 365","work-copilot"]}}`
	rr = request(router, http.MethodPost, "/api/projects/active/mcp-exclusions", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST = %d: %s", rr.Code, rr.Body.String())
	}
	dto = decodeMCPExclusions(t, rr.Body.Bytes())
	if want := []string{"claude.ai Microsoft 365", "work-copilot"}; !reflect.DeepEqual(dto.Exclusions["claude-code"], want) {
		t.Errorf("exclusiones = %v", dto.Exclusions)
	}
	// El nombre excluido a mano aparece como candidato aunque no sea un MCP de usuario.
	if !reflect.DeepEqual(dto.Candidates, []string{"claude.ai Microsoft 365", "engram", "work-copilot"}) {
		t.Errorf("candidatos = %v", dto.Candidates)
	}
	if dto.Report == nil || len(dto.Report.Targets) == 0 || dto.Report.Targets[0].Status != "written" || len(dto.Report.Failed()) != 0 {
		t.Fatalf("informe = %+v", dto.Report)
	}
	if got := deniedServerNames(t, root); !reflect.DeepEqual(got, []string{"claude.ai Microsoft 365", "work-copilot"}) {
		t.Errorf("deniedMcpServers = %v", got)
	}
	if stored, _, err := mgr.LookupProjectMCPExclusions(root); err != nil || len(stored["claude-code"]) != 2 {
		t.Errorf("el hub guarda %v (err=%v)", stored, err)
	}

	// Quitar un nombre lo retira del fichero del proyecto.
	rr = request(router, http.MethodPost, "/api/projects/active/mcp-exclusions", `{"exclusions":{"claude-code":["work-copilot"]}}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST (quitar) = %d: %s", rr.Code, rr.Body.String())
	}
	if got := deniedServerNames(t, root); !reflect.DeepEqual(got, []string{"work-copilot"}) {
		t.Errorf("tras quitar, deniedMcpServers = %v", got)
	}
}

func TestActiveMCPExclusionsPostReportsUnsupportedAgents(t *testing.T) {
	router, _, _ := reviewModeServer(t, true)
	rr := request(router, http.MethodPost, "/api/projects/active/mcp-exclusions", `{"exclusions":{"codex":["x"]}}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST = %d: %s", rr.Code, rr.Body.String())
	}
	dto := decodeMCPExclusions(t, rr.Body.Bytes())
	if dto.Report == nil || !reflect.DeepEqual(dto.Report.Unsupported, []string{"codex"}) {
		t.Errorf("informe = %+v", dto.Report)
	}
}

func TestActiveMCPExclusionsRejectsBadInput(t *testing.T) {
	router, root, _ := reviewModeServer(t, true)
	for name, body := range map[string]string{
		"JSON inválido":          `no es json`,
		"nombre con control":     `{"exclusions":{"claude-code":["a\u0007b"]}}`,
		"nombre demasiado largo": `{"exclusions":{"claude-code":["` + strings.Repeat("x", 300) + `"]}}`,
		"agente vacío":           `{"exclusions":{"":["x"]}}`,
	} {
		rr := request(router, http.MethodPost, "/api/projects/active/mcp-exclusions", body)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s = %d, quiero 400: %s", name, rr.Code, rr.Body.String())
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Errorf("una petición inválida no debe escribir en el proyecto (err=%v)", err)
	}
	if rr := request(router, http.MethodDelete, "/api/projects/active/mcp-exclusions", ""); rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE = %d, quiero 405", rr.Code)
	}
}

func TestActiveMCPExclusionsWithAnUnregisteredProjectIs409(t *testing.T) {
	router, root, _ := reviewModeServer(t, false)
	for _, tc := range []struct{ method, body string }{
		{http.MethodGet, ""},
		{http.MethodPost, `{"exclusions":{"claude-code":["x"]}}`},
	} {
		rr := request(router, tc.method, "/api/projects/active/mcp-exclusions", tc.body)
		if rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "axiom project add") {
			t.Errorf("%s = %d, quiero 409 con la orden que lo registra: %s", tc.method, rr.Code, rr.Body.String())
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Errorf("no debe escribirse nada en un proyecto sin registrar (err=%v)", err)
	}
}

// El GET añade el uso de cada candidato en 30 días a partir de las transcripciones de
// Claude Code del proyecto: sin transcripciones es «sin datos»; con ellas, un candidato
// sin llamadas sale como sin uso y uno con llamadas, con su cuenta.
func TestActiveMCPExclusionsEndpointReportsUsage(t *testing.T) {
	router, root, _ := reviewModeServer(t, true)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	userConfig := `{"mcpServers":{"work-copilot":{"command":"x"},"engram":{"command":"y"}}}`
	if err := os.WriteFile(claude.UserConfigPath(home), []byte(userConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	rr := request(router, http.MethodGet, "/api/projects/active/mcp-exclusions", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("GET = %d: %s", rr.Code, rr.Body.String())
	}
	if dto := decodeMCPExclusions(t, rr.Body.Bytes()); dto.UsageData {
		t.Fatalf("sin transcripciones debe ser sin datos: %+v", dto)
	}

	folder := filepath.Join(home, ".claude", "projects", mcpexclusion.EncodeProjectDir(root))
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"assistant","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `","message":{"content":[{"type":"tool_use","name":"mcp__engram__mem_save","input":{}}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(folder, "s.jsonl"), []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}

	rr = request(router, http.MethodGet, "/api/projects/active/mcp-exclusions", "")
	dto := decodeMCPExclusions(t, rr.Body.Bytes())
	if !dto.UsageData || len(dto.Usage) != 2 {
		t.Fatalf("uso = %+v", dto)
	}
	byName := map[string]mcpexclusion.ServerUsage{}
	for _, server := range dto.Usage {
		byName[server.Name] = server
	}
	if s := byName["engram"]; s.Uses != 1 || s.Unused || s.LastUsed == nil {
		t.Errorf("engram = %+v", s)
	}
	if s := byName["work-copilot"]; s.Uses != 0 || !s.Unused {
		t.Errorf("work-copilot = %+v", s)
	}
}
