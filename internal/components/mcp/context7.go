package mcp

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/versions"
)

var defaultContext7ServerJSON = []byte(fmt.Sprintf("{\n  \"command\": \"npx\",\n  \"args\": [\n    \"-y\",\n    \"--package=@upstash/context7-mcp@%s\",\n    \"--\",\n    \"context7-mcp\"\n  ]\n}\n", versions.Context7MCP))

var defaultContext7OverlayJSON = []byte(fmt.Sprintf("{\n  \"mcpServers\": {\n    \"context7\": {\n      \"command\": \"npx\",\n      \"args\": [\n        \"-y\",\n        \"--package=@upstash/context7-mcp@%s\",\n        \"--\",\n        \"context7-mcp\"\n      ]\n    }\n  }\n}\n", versions.Context7MCP))

// openCodeContext7OverlayJSON is the opencode.json overlay using the new MCP format.
// Context7 is a remote MCP server — no npx needed.
// The context7 entry must replace atomically so legacy local keys do not survive
// deep merge into OpenCode's strict MCP schema.
var openCodeContext7OverlayJSON = []byte("{\n  \"mcp\": {\n    \"context7\": {\n      \"__replace__\": {\n        \"type\": \"remote\",\n        \"url\": \"https://mcp.context7.com/mcp\",\n        \"enabled\": true\n      }\n    }\n  }\n}\n")

// antigravityContext7OverlayJSON is the Antigravity mcp_config.json overlay.
// Uses mcpServers key (same schema as Claude Code) with serverUrl for HTTP remote.
// The context7 entry must replace atomically so legacy local keys do not survive
// deep merge into this managed MCP server entry.
var antigravityContext7OverlayJSON = []byte("{\n  \"mcpServers\": {\n    \"context7\": {\n      \"__replace__\": {\n        \"serverUrl\": \"https://mcp.context7.com/mcp\"\n      }\n    }\n  }\n}\n")

func DefaultContext7ServerJSON() []byte {
	content := make([]byte, len(defaultContext7ServerJSON))
	copy(content, defaultContext7ServerJSON)
	return content
}

func DefaultContext7OverlayJSON() []byte {
	content := make([]byte, len(defaultContext7OverlayJSON))
	copy(content, defaultContext7OverlayJSON)
	return content
}

func OpenCodeContext7OverlayJSON() []byte {
	content := make([]byte, len(openCodeContext7OverlayJSON))
	copy(content, openCodeContext7OverlayJSON)
	return content
}

// vsCodeWorkspaceContext7OverlayJSON es la entrada de Context7 en <proyecto>/.mcp.json
// para VS Code Copilot. Ese fichero lo comparte con Claude Code, así que la orden es
// la misma que escribe Claude (npx con la versión fijada): si fueran distintas, la
// fusión profunda mezclaría las dos formas y cada `sync` reescribiría la del otro.
// Añade lo que el arnés de Copilot espera y Claude Code acepta: `type: "stdio"` y
// `tools: ["*"]`, que expone todas las herramientas del servidor.
var vsCodeWorkspaceContext7OverlayJSON = []byte(fmt.Sprintf("{\n  \"mcpServers\": {\n    \"context7\": {\n      \"type\": \"stdio\",\n      \"command\": \"npx\",\n      \"args\": [\n        \"-y\",\n        \"--package=@upstash/context7-mcp@%s\",\n        \"--\",\n        \"context7-mcp\"\n      ],\n      \"tools\": [\n        \"*\"\n      ]\n    }\n  }\n}\n", versions.Context7MCP))

// VSCodeWorkspaceContext7OverlayJSON devuelve la entrada de Context7 de
// <proyecto>/.mcp.json para VS Code Copilot.
func VSCodeWorkspaceContext7OverlayJSON() []byte {
	content := make([]byte, len(vsCodeWorkspaceContext7OverlayJSON))
	copy(content, vsCodeWorkspaceContext7OverlayJSON)
	return content
}

func AntigravityContext7OverlayJSON() []byte {
	content := make([]byte, len(antigravityContext7OverlayJSON))
	copy(content, antigravityContext7OverlayJSON)
	return content
}

// copilotCLIContext7OverlayJSON es el overlay de <COPILOT_HOME|~/.copilot>/mcp-config.json,
// el fichero de usuario del arnés de Copilot que comparten Copilot CLI y VS Code
// Copilot. La raíz es `mcpServers` y cada entrada
// declara `type` y `tools`: `tools: ["*"]` expone todas las herramientas del
// servidor, que es lo que el CLI exige para no dejarlo sin herramientas. Context7 es
// remoto (http), así que no hace falta npx. La entrada se reemplaza entera para que
// una definición local anterior no sobreviva a la fusión profunda.
var copilotCLIContext7OverlayJSON = []byte("{\n  \"mcpServers\": {\n    \"context7\": {\n      \"__replace__\": {\n        \"type\": \"http\",\n        \"url\": \"https://mcp.context7.com/mcp\",\n        \"tools\": [\n          \"*\"\n        ]\n      }\n    }\n  }\n}\n")

func CopilotCLIContext7OverlayJSON() []byte {
	content := make([]byte, len(copilotCLIContext7OverlayJSON))
	copy(content, copilotCLIContext7OverlayJSON)
	return content
}
