package mcp

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/filemerge"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/versions"
)

// AzureDevOpsServerID es el nombre de la entrada MCP del servidor oficial de
// Azure DevOps en la configuración de cada agente.
const AzureDevOpsServerID = "azure-devops"

// azureDevOpsDomains limita las herramientas del servidor a lo que el agente
// necesita para leer: proyectos (`core`), iteraciones y equipos (`work`), work
// items y búsqueda de duplicados (`work-items`, `search`).
var azureDevOpsDomains = []string{"core", "work", "work-items", "search"}

// azureDevOpsOrganization acepta el nombre de una organización de
// dev.azure.com: letras, dígitos y guiones, sin empezar por guion. Así un valor
// de axiom.yaml no puede colarse como opción del servidor ni como otra orden.
var azureDevOpsOrganization = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,49}$`)

// ValidAzureDevOpsOrganization informa de si organization es un nombre de
// organización que Axiom puede pasar como argumento del servidor MCP.
func ValidAzureDevOpsOrganization(organization string) bool {
	return azureDevOpsOrganization.MatchString(organization)
}

// AzureDevOpsArgs devuelve los argumentos de `npx` del servidor MCP local de
// Azure DevOps con la versión fijada. No lleva `--authentication`: el valor por
// defecto del servidor es `interactive`, el inicio de sesión de Microsoft
// (Entra), así que la configuración nunca contiene un PAT ni otro secreto.
func AzureDevOpsArgs(organization string) []string {
	args := []string{"-y", "@azure-devops/mcp@" + versions.AzureDevOpsMCP, organization, "-d"}
	return append(args, azureDevOpsDomains...)
}

// azureDevOpsEntry es la forma común `command` + `args` (Gemini, Cursor, Qwen,
// Pi, Trae, Antigravity y los ficheros por servidor).
func azureDevOpsEntry(organization string) map[string]any {
	return map[string]any{"command": "npx", "args": AzureDevOpsArgs(organization)}
}

// azureDevOpsStdioEntry añade `type` y `tools`, que espera el arnés de Copilot
// (Copilot CLI y VS Code) y acepta Claude Code. <proyecto>/.mcp.json lo
// comparten Claude Code y VS Code Copilot: los dos escriben esta misma forma para
// que un `sync` no reescriba la del otro.
func azureDevOpsStdioEntry(organization string) map[string]any {
	entry := azureDevOpsEntry(organization)
	entry["type"] = "stdio"
	entry["tools"] = []string{"*"}
	return entry
}

// azureDevOpsOpenCodeEntry es la forma `local` de OpenCode: la orden va entera
// en `command`.
func azureDevOpsOpenCodeEntry(organization string) map[string]any {
	return map[string]any{
		"type":    "local",
		"command": append([]string{"npx"}, AzureDevOpsArgs(organization)...),
		"enabled": true,
	}
}

// azureDevOpsOverlay envuelve la entrada en `<root>.azure-devops` con
// `__replace__`, para que un cambio de organización o de versión sustituya la
// entrada entera y no deje argumentos viejos tras la fusión profunda.
func azureDevOpsOverlay(root string, entry map[string]any) ([]byte, error) {
	overlay, err := json.Marshal(map[string]any{
		root: map[string]any{AzureDevOpsServerID: map[string]any{"__replace__": entry}},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal azure-devops overlay: %w", err)
	}
	return overlay, nil
}

// InjectAzureDevOps registra el servidor MCP oficial de Azure DevOps para el
// adaptador. Solo escribe en el ámbito de proyecto (targetDir distinto de
// homeDir): la organización es del proyecto que activa el plugin azure-devops y
// no debe filtrarse a la configuración de usuario de todos los proyectos. En
// ámbito de usuario no hace nada.
func InjectAzureDevOps(homeDir, targetDir string, adapter agents.Adapter, organization string) (InjectionResult, error) {
	if !adapter.SupportsMCP() || filepath.Clean(targetDir) == filepath.Clean(homeDir) {
		return InjectionResult{}, nil
	}
	if !ValidAzureDevOpsOrganization(organization) {
		return InjectionResult{}, fmt.Errorf("integrations.azure-devops.organization %q no es un nombre de organización válido de dev.azure.com", organization)
	}

	switch adapter.MCPStrategy() {
	case model.StrategySeparateMCPFiles:
		if adapter.Agent() == model.AgentClaudeCode {
			return mergeAzureDevOps(filepath.Join(targetDir, ".mcp.json"), "mcpServers", azureDevOpsStdioEntry(organization))
		}
		path := adapter.MCPConfigPath(targetDir, AzureDevOpsServerID)
		content, err := json.MarshalIndent(azureDevOpsEntry(organization), "", "  ")
		if err != nil {
			return InjectionResult{}, fmt.Errorf("marshal azure-devops server: %w", err)
		}
		writeResult, err := filemerge.WriteFileAtomic(path, append(content, '\n'), 0o644)
		if err != nil {
			return InjectionResult{}, err
		}
		return InjectionResult{Changed: writeResult.Changed, Files: []string{path}}, nil
	case model.StrategyMergeIntoSettings:
		settingsPath := adapter.SettingsPath(targetDir)
		if settingsPath == "" {
			return InjectionResult{}, nil
		}
		if adapter.Agent() == model.AgentOpenCode {
			return mergeAzureDevOps(settingsPath, "mcp", azureDevOpsOpenCodeEntry(organization))
		}
		return mergeAzureDevOps(settingsPath, "mcpServers", azureDevOpsEntry(organization))
	case model.StrategyMCPConfigFile:
		path := adapter.MCPConfigPath(targetDir, AzureDevOpsServerID)
		if path == "" {
			return InjectionResult{}, nil
		}
		entry := azureDevOpsEntry(organization)
		if adapter.Agent() == model.AgentVSCodeCopilot || adapter.Agent() == model.AgentGitHubCopilotCLI {
			entry = azureDevOpsStdioEntry(organization)
		}
		return mergeAzureDevOps(path, "mcpServers", entry)
	case model.StrategyTOMLFile:
		configPath := adapter.MCPConfigPath(targetDir, AzureDevOpsServerID)
		existing, err := osReadFile(configPath)
		if err != nil {
			return InjectionResult{}, fmt.Errorf("read TOML config %q: %w", configPath, err)
		}
		updated := filemerge.UpsertCodexMCPServerBlock(string(existing), AzureDevOpsServerID, "npx", AzureDevOpsArgs(organization))
		writeResult, err := filemerge.WriteFileAtomic(configPath, []byte(updated), 0o644)
		if err != nil {
			return InjectionResult{}, fmt.Errorf("write TOML config %q: %w", configPath, err)
		}
		return InjectionResult{Changed: writeResult.Changed, Files: []string{configPath}}, nil
	default:
		return InjectionResult{}, fmt.Errorf("mcp injector does not support MCP strategy %d for agent %q", adapter.MCPStrategy(), adapter.Agent())
	}
}

func mergeAzureDevOps(path, root string, entry map[string]any) (InjectionResult, error) {
	overlay, err := azureDevOpsOverlay(root, entry)
	if err != nil {
		return InjectionResult{}, err
	}
	writeResult, err := mergeJSONFile(path, overlay)
	if err != nil {
		return InjectionResult{}, fmt.Errorf("merge azure-devops into %q: %w", path, err)
	}
	return InjectionResult{Changed: writeResult.Changed, Files: []string{path}}, nil
}
