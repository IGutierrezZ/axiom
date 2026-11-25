package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/components/engram"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// claudeEngramServerName es el nombre con el que Axiom declara Engram en el
// .mcp.json del proyecto y el que Claude Code guarda en sus listas de aprobación.
const claudeEngramServerName = "engram"

// claudeProjectEngramPendingApproval devuelve la ruta de <raíz>/.mcp.json cuando el
// Engram que usaría Claude Code solo está declarado ahí y Claude Code no lo ha
// aprobado: Claude Code no carga un servidor de .mcp.json sin aprobación, así que
// sus sesiones arrancan sin Engram aunque el comando funcione.
//
// Aprobado es: `enableAllProjectMcpServers: true` o `engram` en
// `enabledMcpjsonServers` en algún settings (proyecto `.claude/settings.local.json`,
// donde Claude Code guarda la aprobación, y `.claude/settings.json`; usuario
// `~/.claude/settings.json`), salvo que `engram` figure en `disabledMcpjsonServers`.
// No lee aprobaciones antiguas de ~/.claude.json. Solo lee: nunca aprueba nada.
func claudeProjectEngramPendingApproval(homeDir, workspaceDir string, installedAgents []string) (string, bool) {
	if strings.TrimSpace(workspaceDir) == "" || !slices.Contains(installedAgents, string(model.AgentClaudeCode)) {
		return "", false
	}
	// Con un registro de usuario (~/.claude.json) Claude Code tiene Engram sin
	// depender de la aprobación del proyecto.
	if commands, err := engram.ReadPersistedStdioCommands(homeDir, []string{string(model.AgentClaudeCode)}); err != nil || len(commands) > 0 {
		return "", false
	}
	mcpPath := filepath.Join(workspaceDir, ".mcp.json")
	project, ok := readJSONObjectFile(mcpPath)
	if !ok {
		return "", false
	}
	servers, _ := project["mcpServers"].(map[string]any)
	if _, declared := servers[claudeEngramServerName].(map[string]any); !declared {
		return "", false
	}

	approved, rejected := false, false
	for _, path := range []string{
		filepath.Join(workspaceDir, ".claude", "settings.local.json"),
		filepath.Join(workspaceDir, ".claude", "settings.json"),
		filepath.Join(homeDir, ".claude", "settings.json"),
	} {
		settings, ok := readJSONObjectFile(path)
		if !ok {
			continue
		}
		if enableAll, _ := settings["enableAllProjectMcpServers"].(bool); enableAll {
			approved = true
		}
		if jsonStringListContains(settings["enabledMcpjsonServers"], claudeEngramServerName) {
			approved = true
		}
		if jsonStringListContains(settings["disabledMcpjsonServers"], claudeEngramServerName) {
			rejected = true
		}
	}
	if approved && !rejected {
		return "", false
	}
	return mcpPath, true
}

// readJSONObjectFile lee un objeto JSON; un fichero ausente o ilegible da false.
func readJSONObjectFile(path string) (map[string]any, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil || root == nil {
		return nil, false
	}
	return root, true
}

func jsonStringListContains(value any, want string) bool {
	items, _ := value.([]any)
	for _, item := range items {
		if text, ok := item.(string); ok && text == want {
			return true
		}
	}
	return false
}
