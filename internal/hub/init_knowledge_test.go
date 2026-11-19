package hub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitKnowledgeProfile(t *testing.T) {
	tempDir := t.TempDir()

	ini := NewInitializer(nil, nil)
	res, err := ini.Init(InitOptions{
		Path:    tempDir,
		Name:    "KnowledgeProject",
		Profile: "knowledge",
	})
	if err != nil {
		t.Fatalf("Init fallo con perfil knowledge: %v", err)
	}

	// 1. Verificar axiom.yaml
	axiomContent, err := os.ReadFile(res.ConfigPath)
	if err != nil {
		t.Fatalf("error leyendo axiom.yaml: %v", err)
	}
	contentStr := string(axiomContent)

	if !strings.Contains(contentStr, `topology: "multirepo"`) {
		t.Errorf("se esperaba topology: multirepo, obtenido:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `specs_repository: "openspec"`) {
		t.Errorf("se esperaba specs_repository: openspec, obtenido:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `shared_memory: "engram"`) {
		t.Errorf("se esperaba shared_memory: engram, obtenido:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `language: "es"`) {
		t.Errorf("se esperaba language: es, obtenido:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `gate_policy: "deferred"`) {
		t.Errorf("se esperaba gate_policy: deferred en el rol knowledge, obtenido:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "knowledge:") {
		t.Errorf("se esperaba rol knowledge declarado, obtenido:\n%s", contentStr)
	}

	// 2. Verificar que NO se crearon skills de testeo ni ejecutores
	skillsDir := filepath.Join(tempDir, ".axiom", "inbox", "skills")
	if _, err := os.Stat(skillsDir); !os.IsNotExist(err) {
		t.Errorf("no se debió crear el directorio de skills %s en perfil knowledge", skillsDir)
	}

	// 3. Verificar directorios canónicos OpenSpec
	if _, err := os.Stat(filepath.Join(tempDir, "openspec", "specs")); os.IsNotExist(err) {
		t.Errorf("se esperaba directorio openspec/specs")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "openspec", "changes")); os.IsNotExist(err) {
		t.Errorf("se esperaba directorio openspec/changes")
	}

	// 4. Verificar openspec/INDEX.md
	indexPath := filepath.Join(tempDir, "openspec", "INDEX.md")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("error leyendo openspec/INDEX.md: %v", err)
	}
	if !strings.Contains(string(indexBytes), "Catálogo Maestro de Especificaciones Vivas") {
		t.Errorf("contenido inesperado en INDEX.md: %s", string(indexBytes))
	}

	// 5. Verificar .mcp.json
	mcpPath := filepath.Join(tempDir, ".mcp.json")
	mcpBytes, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("error leyendo .mcp.json: %v", err)
	}
	var mcpData map[string]any
	if err := json.Unmarshal(mcpBytes, &mcpData); err != nil {
		t.Fatalf("error parseando .mcp.json: %v", err)
	}
	servers, ok := mcpData["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf(".mcp.json no contiene mcpServers")
	}
	for _, expected := range []string{"engram", "codegraph"} {
		if _, exists := servers[expected]; !exists {
			t.Errorf("servidor %s no encontrado en .mcp.json", expected)
		}
	}
	if _, exists := servers["serena"]; exists {
		t.Errorf("el perfil knowledge ya no debe escribir la entrada serena")
	}
}

func TestInitKnowledgeProfileAliasSpecOnly(t *testing.T) {
	tempDir := t.TempDir()

	ini := NewInitializer(nil, nil)
	res, err := ini.Init(InitOptions{
		Path:    tempDir,
		Name:    "SpecOnlyProject",
		Profile: "spec-only",
	})
	if err != nil {
		t.Fatalf("Init fallo con perfil spec-only: %v", err)
	}

	axiomBytes, err := os.ReadFile(res.ConfigPath)
	if err != nil {
		t.Fatalf("error leyendo axiom.yaml: %v", err)
	}
	if !strings.Contains(string(axiomBytes), `topology: "multirepo"`) {
		t.Errorf("se esperaba topology: multirepo con spec-only")
	}

	if _, err := os.Stat(filepath.Join(tempDir, ".mcp.json")); os.IsNotExist(err) {
		t.Errorf("se esperaba creación de .mcp.json con spec-only")
	}
}

func TestInitKnowledgeProfileMergeMCP(t *testing.T) {
	tempDir := t.TempDir()

	// Pre-crear .mcp.json con un servidor preexistente
	mcpPath := filepath.Join(tempDir, ".mcp.json")
	initialMCP := `{
		"mcpServers": {
			"custom-db": {
				"command": "db-server",
				"args": ["--port", "5432"]
			}
		}
	}`
	if err := os.WriteFile(mcpPath, []byte(initialMCP), 0644); err != nil {
		t.Fatalf("error pre-creando .mcp.json: %v", err)
	}

	ini := NewInitializer(nil, nil)
	_, err := ini.Init(InitOptions{
		Path:    tempDir,
		Name:    "MergedProject",
		Profile: "knowledge",
	})
	if err != nil {
		t.Fatalf("Init fallo: %v", err)
	}

	mcpBytes, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("error leyendo .mcp.json resultante: %v", err)
	}

	var mcpData map[string]any
	if err := json.Unmarshal(mcpBytes, &mcpData); err != nil {
		t.Fatalf("error parseando .mcp.json resultante: %v", err)
	}

	servers, ok := mcpData["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers no encontrado")
	}

	// Verificar que el servidor previo se conserva
	if _, exists := servers["custom-db"]; !exists {
		t.Errorf("se perdió el servidor preexistente 'custom-db'")
	}

	// Verificar que los nuevos servidores fueron añadidos
	for _, expected := range []string{"engram", "codegraph"} {
		if _, exists := servers[expected]; !exists {
			t.Errorf("servidor %s no encontrado tras merge", expected)
		}
	}
}

func injectAndReadServers(t *testing.T, initialMCP string) map[string]any {
	t.Helper()
	dir := t.TempDir()
	mcpPath := filepath.Join(dir, ".mcp.json")
	if err := os.WriteFile(mcpPath, []byte(initialMCP), 0644); err != nil {
		t.Fatalf("error pre-creando .mcp.json: %v", err)
	}
	if err := InjectKnowledgeMCPServers(dir); err != nil {
		t.Fatalf("InjectKnowledgeMCPServers fallo: %v", err)
	}
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("error leyendo .mcp.json: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("error parseando .mcp.json: %v", err)
	}
	servers, ok := parsed["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers no encontrado")
	}
	return servers
}

func TestInjectKnowledgeMCPServersRemovesManagedSerena(t *testing.T) {
	servers := injectAndReadServers(t, `{
		"mcpServers": {
			"serena": {"command": "serena", "args": ["serve"]},
			"custom-db": {"command": "db-server", "args": ["--port", "5432"]}
		}
	}`)
	if _, exists := servers["serena"]; exists {
		t.Errorf("la entrada serena gestionada debía retirarse")
	}
	for _, expected := range []string{"engram", "codegraph", "custom-db"} {
		if _, exists := servers[expected]; !exists {
			t.Errorf("servidor %s no encontrado tras la retirada", expected)
		}
	}
}

func TestInjectKnowledgeMCPServersPreservesModifiedSerena(t *testing.T) {
	for name, entry := range map[string]string{
		"otros argumentos": `{"command": "serena", "args": ["serve", "--context", "ide"]}`,
		"otro comando":     `{"command": "uvx", "args": ["serve"]}`,
		"clave extra env":  `{"command": "serena", "args": ["serve"], "env": {"A": "1"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			servers := injectAndReadServers(t, `{"mcpServers": {"serena": `+entry+`, "custom-db": {"command": "db-server"}}}`)
			got, exists := servers["serena"]
			if !exists {
				t.Fatalf("la entrada serena modificada debía conservarse")
			}
			var want any
			if err := json.Unmarshal([]byte(entry), &want); err != nil {
				t.Fatalf("entrada de prueba inválida: %v", err)
			}
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("serena = %s, se esperaba intacta %s", gotJSON, wantJSON)
			}
			if _, exists := servers["custom-db"]; !exists {
				t.Errorf("se perdió el servidor preexistente 'custom-db'")
			}
		})
	}
}

func TestInjectKnowledgeMCPServersDoesNotRewriteWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	mcpPath := filepath.Join(dir, ".mcp.json")
	// Formato deliberadamente distinto al que serializa Axiom: si se reescribiera, cambiaría.
	initial := `{"mcpServers":{"engram":{"command":"engram","args":["serve"]},"codegraph":{"command":"codegraph","args":["serve"]}}}`
	if err := os.WriteFile(mcpPath, []byte(initial), 0644); err != nil {
		t.Fatalf("error pre-creando .mcp.json: %v", err)
	}
	if err := InjectKnowledgeMCPServers(dir); err != nil {
		t.Fatalf("InjectKnowledgeMCPServers fallo: %v", err)
	}
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("error leyendo .mcp.json: %v", err)
	}
	if string(data) != initial {
		t.Errorf(".mcp.json se reescribió sin cambios: %s", data)
	}
}
