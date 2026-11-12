package workspace

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace/workspacetest"
)

// REQ-2.2: la comparación de planning homes es por ruta y no exige que existan.
func TestSamePlanningHome(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "uno")
	other := filepath.Join(root, "dos")
	workspacetest.MkdirAll(t, existing)
	workspacetest.MkdirAll(t, other)

	t.Run("misma ruta existente", func(t *testing.T) {
		if !SamePlanningHome(existing, filepath.Join(existing, "..", "uno")) {
			t.Fatal("dos grafias del mismo directorio existente deben ser el mismo planning home")
		}
	})
	t.Run("directorios existentes distintos", func(t *testing.T) {
		if SamePlanningHome(existing, other) {
			t.Fatal("dos directorios existentes distintos no son el mismo planning home")
		}
	})
	t.Run("ninguno existe, mismo texto", func(t *testing.T) {
		missing := filepath.Join(root, "M", "openspec")
		if !SamePlanningHome(missing, filepath.Join(root, "M", "x", "..", "openspec")) {
			t.Fatal("el respaldo léxico debe igualar rutas limpias idénticas aunque no existan")
		}
	})
	t.Run("ninguno existe, distintos", func(t *testing.T) {
		if SamePlanningHome(filepath.Join(root, "M", "openspec"), filepath.Join(root, "M", "repo-specs", "openspec")) {
			t.Fatal("rutas distintas que no existen no son el mismo planning home")
		}
	})
	t.Run("solo uno existe", func(t *testing.T) {
		if SamePlanningHome(existing, filepath.Join(root, "inexistente")) {
			t.Fatal("un directorio existente y una ruta distinta que no existe no son el mismo planning home")
		}
		if !SamePlanningHome(filepath.Join(root, "inexistente", "openspec"), filepath.Join(root, "inexistente", ".", "openspec")) {
			t.Fatal("el respaldo léxico debe cubrir el caso en que alguna ruta no existe")
		}
	})
	t.Run("mayusculas", func(t *testing.T) {
		missingLower := filepath.Join(root, "m", "openspec")
		missingUpper := filepath.Join(root, "M", "OpenSpec")
		got := SamePlanningHome(missingLower, missingUpper)
		if want := runtime.GOOS == "windows"; got != want {
			t.Fatalf("SamePlanningHome sin distinguir mayúsculas = %v en %s, se esperaba %v", got, runtime.GOOS, want)
		}
	})
}
