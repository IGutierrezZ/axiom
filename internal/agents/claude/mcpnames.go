package claude

import (
	"encoding/json"
	"os"
	"sort"
)

// UserMCPServerNames devuelve, ordenados, los nombres de los servidores MCP de alcance de
// usuario que Claude Code tiene en ~/.claude.json (las claves de `mcpServers`). Solo lee
// los nombres: ese fichero guarda también la sesión OAuth y nada más de él sale de esta
// función. Un fichero ausente, ilegible o sin esa clave da una lista vacía y ningún error:
// los nombres son una ayuda para elegir, no un requisito.
func UserMCPServerNames(homeDir string) []string {
	raw, err := os.ReadFile(UserConfigPath(homeDir))
	if err != nil {
		return nil
	}
	var doc struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	names := make([]string, 0, len(doc.MCPServers))
	for name := range doc.MCPServers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
