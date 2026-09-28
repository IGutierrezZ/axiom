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
		"serena": map[string]any{
			"command": "serena",
			"args":    []any{"serve"},
		},
		"codegraph": map[string]any{
			"command": "codegraph",
			"args":    []any{"serve"},
		},
	}
}

// InjectKnowledgeMCPServers inyecta y fusiona de forma no destructiva los servidores engram,
// serena y codegraph dentro de <workspaceRoot>/.mcp.json.
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

	defaultServers := DefaultKnowledgeMCPServers()
	for name, def := range defaultServers {
		if _, exists := servers[name]; !exists {
			servers[name] = def
		}
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
