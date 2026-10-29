package gitcommon

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestMain deja a los procesos git de las pruebas sin acceso a la configuración
// del usuario, con un autor fijo y sin red (tasks.md, regla 9). Se fija una sola
// vez para todo el binario, con el mismo patrón que internal/specbaseline.
//
// A diferencia de otros paquetes, aquí no se fija workspace.DiscoveryCeilingOverride:
// gitcommon no llama a workspace.Locate, y workspace importa gitcommon (desde
// INC-25), así que importarlo desde una prueba de este paquete sería un ciclo.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

// isolatedGitConfig es el gitconfig global de las pruebas: sin gc ni mantenimiento
// automático en segundo plano (que compite con la limpieza del directorio temporal) y
// sin fsmonitor, caché de ficheros sin seguimiento ni fsync, que solo encarecen cada
// proceso git en Windows.
const isolatedGitConfig = `[gc]
	auto = 0
	autoDetach = false
[maintenance]
	auto = false
[core]
	fsmonitor = false
	untrackedCache = false
	fsync = none
`

func runTests(m *testing.M) int {
	root, err := os.MkdirTemp("", "gitcommon-pruebas-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gitcommon: crear la carpeta temporal de las pruebas: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(root) }()

	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "gitcommon: crear el HOME aislado: %v\n", err)
		return 1
	}
	gitConfig := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(gitConfig, []byte(isolatedGitConfig), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "gitcommon: escribir el gitconfig aislado: %v\n", err)
		return 1
	}
	templateDir := filepath.Join(root, "git-template")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "gitcommon: crear la plantilla vacía de git: %v\n", err)
		return 1
	}
	for key, value := range map[string]string{
		"HOME":                home,
		"USERPROFILE":         home,
		"GIT_CONFIG_GLOBAL":   gitConfig,
		"GIT_TEMPLATE_DIR":    templateDir,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME":     "Autor de Prueba",
		"GIT_AUTHOR_EMAIL":    "autor@ejemplo.invalid",
		"GIT_COMMITTER_NAME":  "Autor de Prueba",
		"GIT_COMMITTER_EMAIL": "autor@ejemplo.invalid",
		"GIT_TERMINAL_PROMPT": "0",
	} {
		if err := os.Setenv(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "gitcommon: fijar %s: %v\n", key, err)
			return 1
		}
	}
	return m.Run()
}
