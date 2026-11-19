package hub

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultKnowledgeMCPServers devuelve el mapa estándar de servidores MCP para el perfil knowledge.
func DefaultKnowledgeMCPServers() map[string]any {
	return map[string]any{
		"engram": map[string]any{
			"command": "engram",
			"args":    []any{"serve"},
		},
		"codegraph": map[string]any{
			"command": "codegraph",
			"args":    []any{"serve"},
		},
	}
}

// isManagedSerenaEntry indica si la entrada es exactamente la que escribían versiones
// anteriores de Axiom para Serena: command "serena", args ["serve"] y ninguna otra clave.
func isManagedSerenaEntry(entry any) bool {
	m, ok := entry.(map[string]any)
	if !ok || len(m) != 2 {
		return false
	}
	if cmd, ok := m["command"].(string); !ok || cmd != "serena" {
		return false
	}
	args, ok := m["args"].([]any)
	if !ok || len(args) != 1 {
		return false
	}
	arg, ok := args[0].(string)
	return ok && arg == "serve"
}

// InjectKnowledgeMCPServers inyecta y fusiona de forma no destructiva los servidores engram
// y codegraph dentro de <workspaceRoot>/.mcp.json. Además retira la entrada serena solo si es
// exactamente la que escribían versiones anteriores de Axiom. Solo escribe si algo cambia.
func InjectKnowledgeMCPServers(workspaceRoot string) error {
	mcpPath := filepath.Join(workspaceRoot, ".mcp.json")

	var root map[string]any
	if data, err := os.ReadFile(mcpPath); err == nil {
		if err := json.Unmarshal(data, &root); err != nil {
			// Si el JSON estaba malformado, inicializamos uno nuevo
			root = make(map[string]any)
		}
	} else if os.IsNotExist(err) {
		root = make(map[string]any)
	} else {
		return fmt.Errorf("leer %s: %w", mcpPath, err)
	}

	if root == nil {
		root = make(map[string]any)
	}

	servers, ok := root["mcpServers"].(map[string]any)
	if !ok || servers == nil {
		servers = make(map[string]any)
		root["mcpServers"] = servers
	}

	changed := false
	if entry, exists := servers["serena"]; exists && isManagedSerenaEntry(entry) {
		delete(servers, "serena")
		changed = true
	}

	defaultServers := DefaultKnowledgeMCPServers()
	for name, def := range defaultServers {
		if _, exists := servers[name]; !exists {
			servers[name] = def
			changed = true
		}
	}
	if !changed {
		return nil
	}

	output, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar .mcp.json: %w", err)
	}

	output = append(output, '\n')
	if err := os.WriteFile(mcpPath, output, 0644); err != nil {
		return fmt.Errorf("escribir %s: %w", mcpPath, err)
	}

	return nil
}
