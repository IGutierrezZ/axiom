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

// sharedSkillsDir es la única carpeta de la raíz que Copilot CLI comparte con la
// extensión de VS Code y que Axiom crea para los dos.
const sharedSkillsDir = "skills"

// HasCLIState dice si root contiene algo propio de Copilot CLI (su config,
// sesiones, ajustes o el prompt global), es decir, cualquier entrada que no sea la
// carpeta de skills compartida.
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
		if entry.Name() != sharedSkillsDir {
			return true
		}
	}
	return false
}
