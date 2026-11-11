package workspace

import (
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// ProjectMaster devuelve la carpeta maestra del proyecto Axiom que contiene start, vista
// desde el checkout principal de su repositorio, y si start está dentro de un proyecto Axiom
// (IsAxiomProject). Es la clave con la que el hub guarda las preferencias personales del
// proyecto: dentro de un worktree enlazado la maestra se traslada al mismo lugar relativo del
// checkout principal, de modo que todos los worktrees del proyecto comparten la entrada.
//
// Es de solo lectura y no lanza procesos. Si el checkout principal no se puede deducir por
// ficheros, o la maestra queda fuera del repositorio (multirepo), devuelve la maestra que
// resolvió Locate.
func ProjectMaster(start string) (string, bool) {
	root, ok := IsAxiomProject(start)
	if !ok {
		return "", false
	}
	master := root
	if loc, err := Locate(start); err == nil && loc.Master != "" {
		master = loc.Master
	}
	return mainCheckoutMaster(master), true
}

// mainCheckoutMaster traslada master al checkout principal de su clon cuando está en un
// worktree enlazado. La maestra se mantiene si no hay traslado posible.
func mainCheckoutMaster(master string) string {
	gitRoot, ok := gitcommon.GitRootFromFiles(master)
	if !ok {
		return master
	}
	commonDir, err := gitcommon.CommonDirFromFiles(gitRoot)
	if err != nil {
		return master
	}
	main, known := gitcommon.MainWorktreeFromFiles(commonDir)
	if !known || pathidentity.SameDirectory(main, gitRoot) {
		return master
	}
	relative, err := filepath.Rel(resolveDir(gitRoot), resolveDir(master))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return master
	}
	return filepath.Join(main, relative)
}

func resolveDir(dir string) string {
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		return resolved
	}
	return dir
}
