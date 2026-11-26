package sddstatus

import (
	"fmt"
	"os"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/testenv"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// TestMain fija el techo del descubrimiento ascendente en el directorio temporal
// del sistema: un marcador (`.axiom-workspace`, `axiom.yaml` u `openspec/`) en un
// ancestro de ese directorio, por ejemplo en el directorio personal de un
// desarrollador, no puede alterar el resultado de las pruebas (R29). También fija
// una sola vez el entorno de git (testenv.IsolateGit), de modo que las pruebas con
// git no dependen de la configuración de la máquina y pueden marcarse con
// t.Parallel().
//
// Siguen en serie las pruebas que fijan entorno (t.Setenv), cambian de directorio
// o sustituyen una variable global: branchEngine (useBranchGuard), engramExport
// (stubEngramExport), publishChangeInstanceMarker y gitcommon.Observe.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	_, cleanup, err := testenv.IsolateGit("sddstatus")
	if err != nil {
		fmt.Fprintf(os.Stderr, "sddstatus: %v\n", err)
		return 1
	}
	defer cleanup()
	workspace.DiscoveryCeilingOverride = []string{os.TempDir()}
	// La guarda B (rama del cambio) queda desactivada por defecto: los fixtures de las pruebas
	// anteriores montan repositorios git en master o main sin kickoff.yaml y no tratan de la rama.
	// branch_context_test.go la reactiva con useBranchGuard.
	branchEngine = func() *engine.Engine { return nil }
	return m.Run()
}
