package uninstall

import (
	"fmt"
	"os"
	"testing"
)

// TestMain aísla %APPDATA% para todo el paquete: una prueba que desinstala sin
// t.Setenv("APPDATA", ...) no debe leer ni borrar la configuración real de quien
// ejecuta las pruebas. Las pruebas que fijan su propio APPDATA con t.Setenv
// siguen mandando.
func TestMain(m *testing.M) {
	appData, err := os.MkdirTemp("", "axiom-uninstall-appdata-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "TestMain: crear APPDATA aislado:", err)
		os.Exit(1)
	}
	if err := os.Setenv("APPDATA", appData); err != nil {
		fmt.Fprintln(os.Stderr, "TestMain: fijar APPDATA aislado:", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(appData)
	os.Exit(code)
}
