package livingdoc

import (
	"os"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// TestMain fija el techo del descubrimiento ascendente en el directorio temporal
// del sistema: un marcador (`.axiom-workspace`, `axiom.yaml` u `openspec/`) en un
// ancestro de ese directorio, por ejemplo en el directorio personal de un
// desarrollador, no puede alterar el resultado de las pruebas (R29).
func TestMain(m *testing.M) {
	workspace.DiscoveryCeilingOverride = []string{os.TempDir()}
	os.Exit(m.Run())
}
