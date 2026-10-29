package sddstatus

import (
	"os"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
	"github.com/IGutierrezZ/axiom/v3/internal/testenv"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// TestMain fija el techo del descubrimiento ascendente en el directorio temporal
// del sistema: un marcador (`.axiom-workspace`, `axiom.yaml` u `openspec/`) en un
// ancestro de ese directorio, por ejemplo en el directorio personal de un
// desarrollador, no puede alterar el resultado de las pruebas (R29).
//
// También escala los techos por comando de git (AXIOM_TEST_TIMEOUT_SCALE; por
// defecto 2 en Windows): con paquetes en paralelo los procesos git superan los
// 2 minutos de producción. Solo cambia el valor dentro de este binario de
// pruebas, nunca el valor declarado en reviewtransaction.
func TestMain(m *testing.M) {
	workspace.DiscoveryCeilingOverride = []string{os.TempDir()}
	// La guarda B (rama del cambio) queda desactivada por defecto: los fixtures de las pruebas
	// anteriores montan repositorios git en master o main sin kickoff.yaml y no tratan de la rama.
	// branch_context_test.go la reactiva con useBranchGuard.
	branchEngine = func() *engine.Engine { return nil }
	reviewtransaction.LocalGitCommandTimeout = testenv.MustScaled(reviewtransaction.LocalGitCommandTimeout)
	reviewtransaction.RemoteGitCommandTimeout = testenv.MustScaled(reviewtransaction.RemoteGitCommandTimeout)
	os.Exit(m.Run())
}
