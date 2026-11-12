package workspace

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// SamePlanningHome compara dos planning homes por identidad
// (pathidentity.SameDirectory) y, cuando alguno no existe, por ruta absoluta
// limpia, sin distinguir mayúsculas en Windows. REQ-2.2 compara por ruta y no
// exige que existan: axiom init es quien crea <ruta>/openspec.
//
// Ese respaldo léxico queda fuera de la política de internal/pathidentity, que
// pide declararlo donde se use. Aquí es aceptable porque las dos rutas se derivan
// léxicamente de la misma <ruta> y de su configuración antes de que exista nada
// que comparar por identidad.
func SamePlanningHome(a, b string) bool {
	if pathidentity.SameDirectory(a, b) {
		return true
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	absA, absB = filepath.Clean(absA), filepath.Clean(absB)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(absA, absB)
	}
	return absA == absB
}
