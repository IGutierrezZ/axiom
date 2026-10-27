package testenv

import (
	"os"
	"strings"
	"testing"
)

// BuildBinariesEnv es la variable de entorno que habilita las pruebas que
// compilan y ejecutan binarios propios (axiom.exe o ayudantes en %TEMP%). Vale
// "1" o "true" (sin distinguir mayúsculas). La CI la fija de forma explícita en
// sus workflows; no se activa por la variable CI: en los equipos de desarrollo
// el EDR corporativo bloquea esos ejecutables sin firmar y cada ejecución
// local generaría una alerta de seguridad.
const BuildBinariesEnv = "AXIOM_TEST_BUILD_BINARIES"

// binaryBuildsSkipMessage nombra la variable que habilita la prueba.
const binaryBuildsSkipMessage = "this test builds and runs a Go binary, which endpoint security tools " +
	"may block on developer machines; set " + BuildBinariesEnv + "=1 to run it (CI does)"

// binaryBuildsEnabled interpreta el valor crudo de BuildBinariesEnv.
func binaryBuildsEnabled(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true":
		return true
	}
	return false
}

// BinaryBuildsEnabled informa de si BuildBinariesEnv habilita compilar y
// ejecutar binarios propios. Es para TestMain, que decide antes de que exista
// ninguna prueba; las pruebas usan RequireBinaryBuilds.
func BinaryBuildsEnabled() bool {
	return binaryBuildsEnabled(os.Getenv(BuildBinariesEnv))
}

// RequireBinaryBuilds omite la prueba salvo que BuildBinariesEnv valga "1" o
// "true". Debe llamarse antes de cualquier `go build` que produzca un
// ejecutable, de modo que sin la variable no se compile ni se lance nada.
func RequireBinaryBuilds(t testing.TB) {
	t.Helper()
	if !BinaryBuildsEnabled() {
		t.Skip(binaryBuildsSkipMessage)
	}
}
