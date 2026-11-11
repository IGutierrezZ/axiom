package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/copilotcli"
)

func copilotCLIHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("COPILOT_HOME", "")
	return home
}

func readMCPJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	err = json.Unmarshal(raw, &root)
	if err != nil {
		t.Fatalf("%s no es JSON válido: %v", path, err)
	}
	return root
}

// Context7 en Copilot CLI: <raíz>/mcp-config.json, raíz `mcpServers`, entrada http con
// `type` y `tools: ["*"]`.
func TestInjectCopilotCLIWritesContext7ToMCPConfig(t *testing.T) {
	home := copilotCLIHome(t)
	adapter := copilotcli.NewAdapter()

	first, err := Inject(home, home, adapter)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed {
		t.Fatal("el primer Inject no cambió nada")
	}
	second, err := Inject(home, home, adapter)
	if err != nil || second.Changed {
		t.Fatalf("segundo Inject = %+v, %v; want sin cambios", second, err)
	}

	path := filepath.Join(home, ".copilot", "mcp-config.json")
	root := readMCPJSON(t, path)
	if _, vscodeShape := root["servers"]; vscodeShape {
		t.Fatal("mcp-config.json no debe usar la raíz `servers` de VS Code")
	}
	entry := root["mcpServers"].(map[string]any)["context7"].(map[string]any)
	if entry["type"] != "http" || entry["url"] != "https://mcp.context7.com/mcp" {
		t.Errorf("entrada context7 = %v", entry)
	}
	if !reflect.DeepEqual(entry["tools"], []any{"*"}) {
		t.Errorf("tools = %v, want [*]", entry["tools"])
	}
	if _, leaked := entry["__replace__"]; leaked {
		t.Error("el marcador __replace__ llegó al fichero")
	}
}

// Una definición local anterior de context7 no sobrevive a la fusión, y los demás
// servidores del usuario se conservan.
func TestInjectCopilotCLIReplacesLegacyContext7AndKeepsUserServers(t *testing.T) {
	home := copilotCLIHome(t)
	path := filepath.Join(home, ".copilot", "mcp-config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := `{"mcpServers":{"context7":{"type":"local","command":"npx","args":["-y","@upstash/context7-mcp"],"tools":["*"]},"mine":{"type":"http","url":"https://example.test/mcp","tools":["a"]}}}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Inject(home, home, copilotcli.NewAdapter()); err != nil {
		t.Fatal(err)
	}
	servers := readMCPJSON(t, path)["mcpServers"].(map[string]any)
	ctx7 := servers["context7"].(map[string]any)
	if ctx7["type"] != "http" || ctx7["command"] != nil || ctx7["args"] != nil {
		t.Errorf("context7 conserva campos de la definición local: %v", ctx7)
	}
	mine := servers["mine"].(map[string]any)
	if mine["url"] != "https://example.test/mcp" || !reflect.DeepEqual(mine["tools"], []any{"a"}) {
		t.Errorf("se alteró el servidor del usuario: %v", mine)
	}
}
