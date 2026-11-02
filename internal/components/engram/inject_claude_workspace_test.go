package engram

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

const claudeWorkspaceLegacyEngram = "{\n  \"command\": \"/usr/local/bin/engram\",\n  \"args\": [\"mcp\", \"--tools=agent\"]\n}\n"

// TestInjectWorkspaceClaudeDeclaresEngramInProjectMCPJSON fija que, en el ámbito
// workspace, Engram se declara en <proyecto>/.mcp.json (el fichero del que Claude
// Code carga los servidores MCP de proyecto), conserva lo que el usuario ya tenía
// y no crea el fichero independiente .claude/mcp/engram.json que Claude no lee.
func TestInjectWorkspaceClaudeDeclaresEngramInProjectMCPJSON(t *testing.T) {
	mockEngramLookPath(t, "", "not found")
	workspace := t.TempDir()
	home := t.TempDir()
	mcpPath := filepath.Join(workspace, ".mcp.json")
	writeFile(t, mcpPath, `{
  "mcpServers": {"propio": {"command": "propio-mcp", "args": ["serve"]}},
  "otraClave": {"x": 1}
}
`)

	result, err := InjectWorkspaceWithOptions(workspace, claudeAdapter(), InjectOptions{})
	if err != nil {
		t.Fatalf("InjectWorkspaceWithOptions() error = %v", err)
	}
	if !result.Changed {
		t.Fatal("InjectWorkspaceWithOptions() changed = false; want true")
	}

	registry := readJSONFile(t, mcpPath)
	assertNestedString(t, registry, "engram", "mcpServers", "engram", "command")
	assertNestedStrings(t, registry, []string{"mcp", "--tools=agent"}, "mcpServers", "engram", "args")
	assertNestedString(t, registry, "propio-mcp", "mcpServers", "propio", "command")
	assertNestedStrings(t, registry, []string{"serve"}, "mcpServers", "propio", "args")
	if _, ok := nestedValue(t, registry, "otraClave", "x"); !ok {
		t.Fatalf("la clave de nivel superior del usuario se perdió: %#v", registry)
	}
	if !containsFile(result.Files, mcpPath) {
		t.Fatalf("result.Files no incluye %q: %v", mcpPath, result.Files)
	}

	if _, statErr := os.Stat(filepath.Join(workspace, ".claude", "mcp", "engram.json")); !os.IsNotExist(statErr) {
		t.Fatalf("el ámbito workspace no debe crear .claude/mcp/engram.json; stat = %v", statErr)
	}
	if _, statErr := os.Stat(claudeUserRegistryPath(home)); !os.IsNotExist(statErr) {
		t.Fatalf("el ámbito workspace no debe tocar el registro de usuario; stat = %v", statErr)
	}
	if entries, readErr := os.ReadDir(home); readErr != nil || len(entries) != 0 {
		t.Fatalf("el HOME debe quedar vacío: entries=%v err=%v", entries, readErr)
	}
}

func TestInjectWorkspaceClaudeCreatesProjectMCPJSONWhenMissing(t *testing.T) {
	mockEngramLookPath(t, "", "not found")
	workspace := t.TempDir()

	if _, err := InjectWorkspaceWithOptions(workspace, claudeAdapter(), InjectOptions{}); err != nil {
		t.Fatalf("InjectWorkspaceWithOptions() error = %v", err)
	}

	registry := readJSONFile(t, filepath.Join(workspace, ".mcp.json"))
	assertNestedString(t, registry, "engram", "mcpServers", "engram", "command")
	assertNestedStrings(t, registry, []string{"mcp", "--tools=agent"}, "mcpServers", "engram", "args")
}

func TestInjectWorkspaceClaudeIsIdempotent(t *testing.T) {
	mockEngramLookPath(t, "", "not found")
	workspace := t.TempDir()
	mcpPath := filepath.Join(workspace, ".mcp.json")
	writeFile(t, mcpPath, `{"mcpServers":{"propio":{"command":"propio-mcp"}}}`)

	if _, err := InjectWorkspaceWithOptions(workspace, claudeAdapter(), InjectOptions{}); err != nil {
		t.Fatalf("primera inyección: %v", err)
	}
	first, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := InjectWorkspaceWithOptions(workspace, claudeAdapter(), InjectOptions{})
	if err != nil {
		t.Fatalf("segunda inyección: %v", err)
	}
	if second.Changed {
		t.Fatal("la segunda inyección debe ser idempotente (changed = false)")
	}
	after, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, after) {
		t.Fatalf(".mcp.json cambió en la segunda inyección:\n%s\n---\n%s", first, after)
	}
}

// TestInjectWorkspaceClaudeMigratesManagedLegacyFile cubre las instalaciones
// previas: el fichero independiente con la forma exacta que genera Axiom se
// migra a .mcp.json (conservando su orden absoluta) y se retira, junto con su
// carpeta si queda vacía.
func TestInjectWorkspaceClaudeMigratesManagedLegacyFile(t *testing.T) {
	mockEngramLookPath(t, "", "not found")
	workspace := t.TempDir()
	legacy := filepath.Join(workspace, ".claude", "mcp", "engram.json")
	writeFile(t, legacy, claudeWorkspaceLegacyEngram)

	result, err := InjectWorkspaceWithOptions(workspace, claudeAdapter(), InjectOptions{})
	if err != nil {
		t.Fatalf("InjectWorkspaceWithOptions() error = %v", err)
	}

	registry := readJSONFile(t, filepath.Join(workspace, ".mcp.json"))
	assertNestedString(t, registry, "/usr/local/bin/engram", "mcpServers", "engram", "command")
	assertNestedStrings(t, registry, []string{"mcp", "--tools=agent"}, "mcpServers", "engram", "args")
	if _, statErr := os.Stat(legacy); !os.IsNotExist(statErr) {
		t.Fatalf("el fichero heredado gestionado debe retirarse; stat = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Dir(legacy)); !os.IsNotExist(statErr) {
		t.Fatalf("la carpeta .claude/mcp vacía debe retirarse; stat = %v", statErr)
	}
	if !containsFile(result.Files, legacy) {
		t.Fatalf("result.Files debe incluir el fichero heredado retirado %q: %v", legacy, result.Files)
	}
}

func TestInjectWorkspaceClaudeProjectEntryWinsOverManagedLegacyCommand(t *testing.T) {
	mockEngramLookPath(t, "", "not found")
	workspace := t.TempDir()
	legacy := filepath.Join(workspace, ".claude", "mcp", "engram.json")
	writeFile(t, legacy, claudeWorkspaceLegacyEngram)
	writeFile(t, filepath.Join(workspace, ".mcp.json"), `{"mcpServers":{"engram":{"command":"/opt/propio/engram","args":["mcp","--tools=agent"]}}}`)

	if _, err := InjectWorkspaceWithOptions(workspace, claudeAdapter(), InjectOptions{}); err != nil {
		t.Fatalf("InjectWorkspaceWithOptions() error = %v", err)
	}

	registry := readJSONFile(t, filepath.Join(workspace, ".mcp.json"))
	assertNestedString(t, registry, "/opt/propio/engram", "mcpServers", "engram", "command")
	if _, statErr := os.Stat(legacy); !os.IsNotExist(statErr) {
		t.Fatalf("el fichero heredado gestionado debe retirarse; stat = %v", statErr)
	}
}

func TestInjectWorkspaceClaudePreservesUnmanagedLegacyFile(t *testing.T) {
	mockEngramLookPath(t, "", "not found")
	workspace := t.TempDir()
	legacy := filepath.Join(workspace, ".claude", "mcp", "engram.json")
	content := `{"command":"/usr/local/bin/engram","args":["mcp","--tools=agent"],"env":{"PROPIO":"1"}}`
	writeFile(t, legacy, content)

	if _, err := InjectWorkspaceWithOptions(workspace, claudeAdapter(), InjectOptions{}); err != nil {
		t.Fatalf("InjectWorkspaceWithOptions() error = %v", err)
	}

	got, err := os.ReadFile(legacy)
	if err != nil || string(got) != content {
		t.Fatalf("el fichero heredado con contenido ajeno no debe tocarse: got=%q err=%v", got, err)
	}
	registry := readJSONFile(t, filepath.Join(workspace, ".mcp.json"))
	assertNestedString(t, registry, "engram", "mcpServers", "engram", "command")
}

// TestInjectWorkspaceClaudeKeepsLegacyFileWhenProjectMCPJSONIsCorrupt asegura que
// un .mcp.json ilegible no se pisa y que el fichero heredado sobrevive: no se
// retira nada hasta que la entrada nueva está escrita.
func TestInjectWorkspaceClaudeKeepsLegacyFileWhenProjectMCPJSONIsCorrupt(t *testing.T) {
	mockEngramLookPath(t, "", "not found")
	workspace := t.TempDir()
	legacy := filepath.Join(workspace, ".claude", "mcp", "engram.json")
	writeFile(t, legacy, claudeWorkspaceLegacyEngram)
	mcpPath := filepath.Join(workspace, ".mcp.json")
	corrupt := []byte("{ no es json")
	if err := os.WriteFile(mcpPath, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := InjectWorkspaceWithOptions(workspace, claudeAdapter(), InjectOptions{}); err == nil {
		t.Fatal("InjectWorkspaceWithOptions() error = nil; want fallo por .mcp.json corrupto")
	}
	if got, err := os.ReadFile(mcpPath); err != nil || !bytes.Equal(got, corrupt) {
		t.Fatalf(".mcp.json corrupto modificado: got=%q err=%v", got, err)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatalf("el fichero heredado debe sobrevivir al fallo: %v", err)
	}
}

// TestInjectGlobalClaudeDoesNotWriteProjectMCPJSON mantiene el comportamiento
// del ámbito global: el registro de usuario es ~/.claude.json y no se crea ningún
// .mcp.json bajo el HOME.
func TestInjectGlobalClaudeDoesNotWriteProjectMCPJSON(t *testing.T) {
	mockEngramLookPath(t, "", "not found")
	home := t.TempDir()

	if _, err := Inject(home, claudeAdapter()); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	if _, err := os.Stat(claudeUserRegistryPath(home)); err != nil {
		t.Fatalf("el ámbito global debe escribir el registro de usuario: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".mcp.json")); !os.IsNotExist(statErr) {
		t.Fatalf("el ámbito global no debe crear .mcp.json; stat = %v", statErr)
	}
}

func claudeUserRegistryPath(home string) string {
	return filepath.Join(home, ".claude.json")
}

func containsFile(files []string, want string) bool {
	for _, file := range files {
		if file == want {
			return true
		}
	}
	return false
}
