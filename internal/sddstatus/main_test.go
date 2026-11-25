package sddstatus

import (
	"os"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// TestMain fija el techo del descubrimiento ascendente en el directorio temporal
// del sistema: un marcador (`.axiom-workspace`, `axiom.yaml` u `openspec/`) en un
// ancestro de ese directorio, por ejemplo en el directorio personal de un
// desarrollador, no puede alterar el resultado de las pruebas (R29).
func TestMain(m *testing.M) {
	workspace.DiscoveryCeilingOverride = []string{os.TempDir()}
	// La guarda B (rama del cambio) queda desactivada por defecto: los fixtures de las pruebas
	// anteriores montan repositorios git en master o main sin kickoff.yaml y no tratan de la rama.
	// branch_context_test.go la reactiva con useBranchGuard.
	branchEngine = func() *engine.Engine { return nil }
	os.Exit(m.Run())
}
