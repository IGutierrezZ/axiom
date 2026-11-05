package dashboard

import (
	"os"
	"path/filepath"
	"strings"
)

// ServerWorkDir devuelve el directorio de trabajo del servidor de `axiom ui`:
// el home del usuario o, en su defecto, el temporal. Nunca el workspace: en
// Windows un proceso cuyo cwd está dentro de un directorio impide borrarlo, y un
// servidor lanzado desde un worktree haría fallar `git worktree remove`.
func ServerWorkDir(baseDir string) string {
	var candidates []string
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, home)
	}
	candidates = append(candidates, os.TempDir())
	return neutralWorkDir(baseDir, candidates...)
}

// neutralWorkDir elige el primer candidato no vacío que no esté dentro de
// baseDir (ni sea baseDir). Si ninguno sirve, devuelve la raíz del volumen de
// baseDir, que solo coincide con el workspace si este es la propia raíz.
func neutralWorkDir(baseDir string, candidates ...string) string {
	for _, c := range candidates {
		if c != "" && !isWithinDir(baseDir, c) {
			return c
		}
	}
	return filepath.VolumeName(baseDir) + string(filepath.Separator)
}

// isWithinDir indica si path es dir o está dentro de dir.
func isWithinDir(dir, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
