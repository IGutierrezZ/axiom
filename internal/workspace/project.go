package workspace

import (
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// IsAxiomProject informa de si start pertenece a un proyecto Axiom y devuelve el
// directorio del marcador que lo demuestra. Es el criterio único para que los
// hooks y las guardas instalados en el equipo del usuario sean inertes fuera de
// los proyectos de Axiom.
//
// Un proyecto Axiom es el que tiene un axiom.yaml en start o en un ancestro, o un
// puntero .axiom-workspace que resuelve a un axiom.yaml existente (la vía canónica
// del multirepo hasta el repositorio de especificaciones). Un directorio openspec/
// suelto, .git, .atl o un directorio de skills no bastan: los usan repositorios
// que no son de Axiom. Un puntero roto tampoco cuenta por sí solo, porque no
// apunta a ningún axiom.yaml; se sigue ascendiendo como si no existiera.
//
// Es de solo lectura y no lanza procesos. Aplica las mismas cotas que Locate: el
// techo de pruebas, la raíz de un worktree enlazado y, si este no tiene marcador
// propio, el checkout principal de su clon (REQ-25.28). Un start inexistente, que
// no es un directorio o es la raíz del sistema de ficheros nunca es un proyecto.
// No cambia la semántica de markerAt ni de Locate, que siguen aceptando openspec/.
func IsAxiomProject(start string) (root string, ok bool) {
	abs, err := normalizeStart(start)
	if err != nil {
		return "", false
	}
	bound := gitcommon.LinkedWorktreeRoot(abs)
	if dir, found := findProjectMarker(abs, bound); found {
		return dir, true
	}
	if bound == "" {
		return "", false
	}
	// Worktree enlazado sin marcador propio: se resuelve como desde su checkout
	// principal, igual que locateFromMainCheckout, pero sin avisos.
	commonDir, err := gitcommon.CommonDirFromFiles(bound)
	if err != nil {
		return "", false
	}
	main, found := gitcommon.MainWorktreeFromFiles(commonDir)
	if !found || !isDir(main) {
		return "", false
	}
	return findProjectMarker(main, "")
}

// findProjectMarker recorre from y sus ancestros como findMarker, pero solo
// acepta los marcadores propios de un proyecto Axiom (projectMarkerAt).
func findProjectMarker(from, bound string) (string, bool) {
	dir := from
	for {
		if coversCeiling(dir) {
			break
		}
		if projectMarkerAt(dir) {
			return dir, true
		}
		if bound != "" && pathidentity.SameDirectory(dir, bound) {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// projectMarkerAt informa de si dir contiene un axiom.yaml o un puntero
// .axiom-workspace que resuelve a uno. A diferencia de markerAt, excluye openspec/.
func projectMarkerAt(dir string) bool {
	if fileExists(filepath.Join(dir, DefaultConfigFilename)) {
		return true
	}
	pointerPath := filepath.Join(dir, WorkspacePointerFilename)
	if !fileExists(pointerPath) {
		return false
	}
	_, err := resolveFromPointerFile(pointerPath)
	return err == nil
}
