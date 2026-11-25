package tui

import (
	"os"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// TestMain fija el techo del descubrimiento ascendente de workspace.Locate en el
// directorio temporal del sistema: un marcador (`.axiom-workspace`, `axiom.yaml`
// u `openspec/`) en un ancestro de ese directorio, por ejemplo en el directorio
// personal de un desarrollador, no puede alterar el resultado de las pruebas
// (inc-24, R29). Las pruebas de gobernanza usan t.Chdir sobre espacios
// temporales y la TUI resuelve el planning home desde el directorio actual.
func TestMain(m *testing.M) {
	workspace.DiscoveryCeilingOverride = []string{os.TempDir()}
	os.Exit(m.Run())
}
