package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/vscode"
)

func writeLegacyVSCodeMCP(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".vscode", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// Retira la entrada de Axiom y conserva comentarios, servidores e inputs del usuario.
func TestRetireLegacyVSCodeServerKeepsUserContent(t *testing.T) {
	path := writeLegacyVSCodeMCP(t, "// equipo\n{\n  \"servers\": {\n    \"propio\": {\"command\": \"propio\"},\n    \"context7\": {\"type\": \"http\", \"url\": \"https://mcp.context7.com/mcp/\"}\n  },\n  \"inputs\": []\n}\n")

	retired, err := RetireLegacyVSCodeServer(path, "context7")
	if err != nil || !retired {
		t.Fatalf("RetireLegacyVSCodeServer() = %v, %v; quiero true, nil", retired, err)
	}
	raw, _ := os.ReadFile(path)
	text := string(raw)
	if strings.Contains(text, "context7") || !strings.Contains(text, "// equipo") || !strings.Contains(text, "propio") || !strings.Contains(text, "\"inputs\"") {
		t.Errorf("el fichero debe conservar lo del usuario y perder solo context7:\n%s", text)
	}

	again, err := RetireLegacyVSCodeServer(path, "context7")
	if err != nil || again {
		t.Errorf("una segunda retirada no cambia nada: %v, %v", again, err)
	}
}

// Una entrada con el nombre de Axiom pero con claves del usuario, o con otra orden, no
// se toca.
func TestRetireLegacyVSCodeServerKeepsUserDefinitions(t *testing.T) {
	for name, content := range map[string]string{
		"engram con env":         `{"servers": {"engram": {"command": "engram", "args": ["mcp"], "env": {"ENGRAM_DIR": "x"}}}}`,
		"engram de otro binario": `{"servers": {"engram": {"command": "mi-engram", "args": ["mcp"]}}}`,
		"context7 con cabeceras": `{"servers": {"context7": {"type": "http", "url": "https://mcp.context7.com/mcp", "headers": {"X": "y"}}}}`,
		"context7 de otra url":   `{"servers": {"context7": {"type": "http", "url": "https://context7.example/mcp"}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := writeLegacyVSCodeMCP(t, content)
			server := "engram"
			if strings.Contains(name, "context7") {
				server = "context7"
			}
			retired, err := RetireLegacyVSCodeServer(path, server)
			if err != nil || retired {
				t.Errorf("RetireLegacyVSCodeServer() = %v, %v; quiero false, nil", retired, err)
			}
			if raw, _ := os.ReadFile(path); string(raw) != content {
				t.Errorf("el fichero cambió:\n%s", raw)
			}
		})
	}
}

// Si el documento queda sin miembros, se borra con la carpeta .vscode vacía; un
// fichero ilegible o ausente no se toca.
func TestRetireLegacyVSCodeServerRemovesEmptiedFileAndSkipsUnreadable(t *testing.T) {
	path := writeLegacyVSCodeMCP(t, `{"servers": {"engram": {"command": "C:\\tools\\engram.exe", "args": ["mcp", "--tools=agent"]}}}`)
	if retired, err := RetireLegacyVSCodeServer(path, "engram"); err != nil || !retired {
		t.Fatalf("RetireLegacyVSCodeServer() = %v, %v; quiero true, nil", retired, err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("la carpeta .vscode vacía debe desaparecer: %v", err)
	}

	broken := writeLegacyVSCodeMCP(t, "{\"servers\": [no es json")
	if retired, err := RetireLegacyVSCodeServer(broken, "engram"); err != nil || retired {
		t.Errorf("un fichero ilegible no se toca: %v, %v", retired, err)
	}
	if retired, err := RetireLegacyVSCodeServer(filepath.Join(t.TempDir(), "mcp.json"), "engram"); err != nil || retired {
		t.Errorf("un fichero ausente no es un error: %v, %v", retired, err)
	}
}

// En el proyecto, Context7 de VS Code va a .mcp.json con la forma de Claude Code
// (npx por stdio) más type y tools, y se retira del .vscode/mcp.json anterior.
func TestInjectVSCodeWorkspaceWritesContext7ToProjectMCPJSON(t *testing.T) {
	root := t.TempDir()
	adapter := vscode.NewAdapter().WithWorkspace(root)
	legacy := filepath.Join(root, ".vscode", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte(`{"servers": {"context7": {"type": "http", "url": "https://mcp.context7.com/mcp"}, "propio": {"command": "propio"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Inject(root, root, adapter)
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}
	if !result.Changed || len(result.Files) != 2 {
		t.Errorf("Inject() = %+v; quiero cambios en .mcp.json y .vscode/mcp.json", result)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".mcp.json"))
	if err != nil {
		t.Fatalf("falta .mcp.json: %v", err)
	}
	for _, want := range []string{`"mcpServers"`, `"type": "stdio"`, `"command": "npx"`, `"context7-mcp"`, `"tools"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf(".mcp.json no contiene %s:\n%s", want, raw)
		}
	}
	if legacyRaw, _ := os.ReadFile(legacy); strings.Contains(string(legacyRaw), "context7") || !strings.Contains(string(legacyRaw), "propio") {
		t.Errorf(".vscode/mcp.json debe conservar solo el servidor del usuario:\n%s", legacyRaw)
	}

	again, err := Inject(root, root, adapter)
	if err != nil || again.Changed {
		t.Errorf("una segunda inyección no cambia nada: %+v, %v", again, err)
	}
}
