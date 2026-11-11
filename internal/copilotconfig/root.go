// Package copilotconfig resuelve la raíz de configuración de GitHub Copilot CLI.
// Vive fuera de internal/agents para que internal/system pueda usarla sin ciclo
// de importaciones (agents importa system).
package copilotconfig

import (
	"os"
	"path/filepath"
	"strings"
)

// EnvVar es la variable con la que Copilot CLI mueve su carpeta de configuración.
const EnvVar = "COPILOT_HOME"

// Root devuelve la carpeta de configuración de Copilot CLI: COPILOT_HOME si está
// definida y es una ruta absoluta, o <homeDir>/.copilot.
//
// COPILOT_HOME solo se respeta cuando homeDir es el directorio personal real del
// usuario. Quien pasa otra raíz de instalación (instalaciones aisladas, tests)
// debe quedarse contenido dentro de ella: el entorno ambiente nunca puede
// redirigir una escritura fuera. Es el mismo criterio que usa el adaptador de VS
// Code con APPDATA y XDG_CONFIG_HOME.
func Root(homeDir string) string {
	if root := strings.TrimSpace(os.Getenv(EnvVar)); filepath.IsAbs(root) && isRealUserHome(homeDir) {
		return filepath.Clean(root)
	}
	return filepath.Join(homeDir, ".copilot")
}

func isRealUserHome(homeDir string) bool {
	userHome, err := os.UserHomeDir()
	return err == nil && filepath.Clean(homeDir) == filepath.Clean(userHome)
}

// cliStateEntries son las entradas de la raíz que solo existen si se ha usado
// Copilot CLI o si Axiom lo ha configurado. Fuente: la referencia del directorio
// de configuración de Copilot CLI
// (docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference).
//
// Quedan fuera a propósito las entradas que comparte con VS Code Copilot, que lee
// ~/.copilot/skills y ~/.copilot/agents (code.visualstudio.com/docs/copilot): su
// presencia no distingue a un agente del otro. También queda fuera cualquier
// entrada desconocida, para que un fichero ajeno no convierta a un usuario de VS
// Code en usuario del CLI.
var cliStateEntries = map[string]bool{
	// Las crea el propio CLI al usarlo.
	"config.json":             true,
	"session-state":           true,
	"session-store.db":        true,
	"command-history-state":   true,
	"logs":                    true,
	"permissions-config.json": true,
	"installed-plugins":       true,
	"plugin-data":             true,
	"ide":                     true,
	// Solo las lee el CLI: las escribe el usuario, Axiom o CodeGraph para él.
	"settings.json":           true,
	"copilot-instructions.md": true,
	"mcp-config.json":         true,
}

// HasCLIState dice si root contiene alguna entrada propia de Copilot CLI (su
// estado, sus ajustes, su MCP o su prompt global).
//
// Existe porque la sola presencia de ~/.copilot no distingue al CLI de VS Code
// Copilot: Axiom escribe ahí las skills de VS Code. Sin esta comprobación, quien
// solo usa VS Code Copilot aparecería como usuario de Copilot CLI cuando la
// detección cae al sistema de ficheros.
func HasCLIState(root string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if cliStateEntries[entry.Name()] {
			return true
		}
	}
	return false
}
