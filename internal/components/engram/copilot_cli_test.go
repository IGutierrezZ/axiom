package engram

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/copilotcli"
)

// copilotCLIHome aísla un home que es además el home real del proceso (COPILOT_HOME
// solo se respeta en ese caso).
func copilotCLIHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("COPILOT_HOME", "")
	return home
}

// El MCP de Engram en Copilot CLI va en <raíz>/mcp-config.json, bajo `mcpServers`
// (no `servers`, la raíz de VS Code), con `type: "local"` y `tools: ["*"]`.
func TestInjectCopilotCLIWritesEngramMCPConfigShape(t *testing.T) {
	home := copilotCLIHome(t)
	mockEngramLookPath(t, "", "not found")
	adapter := copilotcli.NewAdapter()

	result, err := Inject(home, adapter)
	if err != nil {
		t.Fatalf("Inject(copilot-cli) error = %v", err)
	}
	if !result.Changed {
		t.Fatal("Inject(copilot-cli) changed = false")
	}

	path := filepath.Join(home, ".copilot", "mcp-config.json")
	root := readJSONFile(t, path)
	if _, vscodeShape := root["servers"]; vscodeShape {
		t.Fatal("mcp-config.json no debe usar la raíz `servers` de VS Code")
	}
	servers, ok := root["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcp-config.json sin raíz mcpServers: %v", root)
	}
	entry, ok := servers["engram"].(map[string]any)
	if !ok {
		t.Fatalf("falta la entrada engram: %v", servers)
	}
	if entry["type"] != "local" {
		t.Errorf("type = %v, want local", entry["type"])
	}
	if !reflect.DeepEqual(entry["tools"], []any{"*"}) {
		t.Errorf("tools = %v, want [*]", entry["tools"])
	}
	if !reflect.DeepEqual(entry["args"], []any{"mcp", "--tools=agent"}) {
		t.Errorf("args = %v, want [mcp --tools=agent]", entry["args"])
	}
	if command, _ := entry["command"].(string); command != "engram" {
		t.Errorf("command = %q, want engram (Copilot CLI hereda el PATH de la terminal)", command)
	}
	assertArgsHaveToolsAgent(t, path)

	// No escribe nada de VS Code ni usa `engram setup`.
	if _, err := os.Stat(filepath.Join(home, ".copilot", "mcp.json")); !os.IsNotExist(err) {
		t.Fatalf("Copilot CLI escribió un mcp.json (err=%v)", err)
	}
}

func TestInjectCopilotCLIIsIdempotent(t *testing.T) {
	home := copilotCLIHome(t)
	mockEngramLookPath(t, "", "not found")
	adapter := copilotcli.NewAdapter()

	if _, err := Inject(home, adapter); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".copilot", "mcp-config.json")
	first, _ := os.ReadFile(path)

	second, err := Inject(home, adapter)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed {
		t.Fatal("el segundo Inject cambió mcp-config.json")
	}
	if again, _ := os.ReadFile(path); string(again) != string(first) {
		t.Fatal("mcp-config.json cambió entre syncs")
	}
}

// Los servidores del usuario y los demás ajustes del fichero se conservan, y una ruta
// absoluta de Engram ya escrita no se pisa con el nombre relativo.
func TestInjectCopilotCLIPreservesUserServersAndAbsoluteEngramCommand(t *testing.T) {
	home := copilotCLIHome(t)
	mockEngramLookPath(t, "", "not found")
	path := filepath.Join(home, ".copilot", "mcp-config.json")
	writeFile(t, path, `{
  "mcpServers": {
    "mine": {"type": "http", "url": "https://example.test/mcp", "tools": ["only_this"]},
    "engram": {"type": "local", "command": "/opt/custom/engram", "args": ["mcp"], "tools": ["*"]}
  },
  "other": true
}`)

	if _, err := Inject(home, copilotcli.NewAdapter()); err != nil {
		t.Fatal(err)
	}
	root := readJSONFile(t, path)
	if root["other"] != true {
		t.Error("se perdió un ajuste del usuario")
	}
	servers := root["mcpServers"].(map[string]any)
	mine, ok := servers["mine"].(map[string]any)
	if !ok || !reflect.DeepEqual(mine["tools"], []any{"only_this"}) {
		t.Errorf("se alteró el servidor del usuario: %v", servers["mine"])
	}
	engramEntry := servers["engram"].(map[string]any)
	if got := engramEntry["command"]; got != "/opt/custom/engram" {
		t.Errorf("command = %v, want la ruta absoluta existente", got)
	}
}

// La sonda de salud lee el mismo mcp-config.json que escribe el adaptador.
func TestReadPersistedStdioCommandsReadsCopilotCLIMCPConfig(t *testing.T) {
	home := copilotCLIHome(t)
	writeFile(t, filepath.Join(home, ".copilot", "mcp-config.json"),
		`{"mcpServers":{"engram":{"type":"local","command":"/configured/engram","args":["mcp","--tools=agent"],"tools":["*"]}}}`)

	commands, err := ReadPersistedStdioCommands(home, []string{"github-copilot-cli"})
	if err != nil {
		t.Fatalf("ReadPersistedStdioCommands() error = %v", err)
	}
	if len(commands) != 1 || commands[0].Command != "/configured/engram" || !reflect.DeepEqual(commands[0].Args, []string{"mcp", "--tools=agent"}) {
		t.Fatalf("commands = %+v", commands)
	}
	if !strings.HasSuffix(filepath.ToSlash(commands[0].Source), ".copilot/mcp-config.json") {
		t.Fatalf("Source = %q, want mcp-config.json de Copilot CLI", commands[0].Source)
	}
}

// Copilot CLI no usa `engram setup`: el MCP se inyecta directamente.
func TestCopilotCLIDoesNotAttemptEngramSetup(t *testing.T) {
	if slug, ok := SetupAgentSlug(copilotcli.NewAdapter().Agent()); ok || slug != "" {
		t.Fatalf("SetupAgentSlug = (%q, %v), want sin slug", slug, ok)
	}
	for _, mode := range []SetupMode{SetupModeOff, SetupModeOpenCode, SetupModeSupported} {
		if ShouldAttemptSetup(mode, copilotcli.NewAdapter().Agent()) {
			t.Fatalf("ShouldAttemptSetup(%v) = true para Copilot CLI", mode)
		}
	}
}
