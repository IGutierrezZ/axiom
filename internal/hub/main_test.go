package hub

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// TestMain fija el techo del descubrimiento ascendente en el directorio temporal
// del sistema: un marcador (`.axiom-workspace`, `axiom.yaml` u `openspec/`) en un
// ancestro de ese directorio, por ejemplo en el directorio personal de un
// desarrollador, no puede alterar el resultado de las pruebas (R29).
//
// También deja a los procesos git de las pruebas sin acceso a la configuración del
// usuario, con un autor fijo y sin red (tasks.md, regla 9): las pruebas de
// REQ-25.31 crean repositorios y worktrees reales.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	workspace.DiscoveryCeilingOverride = []string{os.TempDir()}

	root, err := os.MkdirTemp("", "hub-pruebas-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "hub: crear la carpeta temporal de las pruebas: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(root) }()

	home := filepath.Join(root, "home")
	templateDir := filepath.Join(root, "git-template")
	for _, dir := range []string{home, templateDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "hub: crear %s: %v\n", dir, err)
			return 1
		}
	}
	gitConfig := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(gitConfig, nil, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "hub: escribir el gitconfig aislado: %v\n", err)
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
			fmt.Fprintf(os.Stderr, "hub: fijar %s: %v\n", key, err)
			return 1
		}
	}
	return m.Run()
}
