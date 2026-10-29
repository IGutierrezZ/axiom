package main

import (
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// validationTarget devuelve la carpeta maestra contra la que `workspace validate`
// evalúa las rutas, el axiom.yaml que carga y los avisos de la resolución.
//
// `--path` conserva su papel de carpeta maestra (REQ-25.28, D-15): solo cuando no
// tiene marcador propio y está en un worktree enlazado, por ejemplo el de un
// repositorio de código de un multirepo creado fuera de la carpeta maestra, toma la
// carpeta maestra y la configuración de workspace.Locate(--path), de modo que los
// repositorios de los roles no se reportan como inexistentes solo porque el
// worktree esté fuera de ella. LocateAt, que evalúa únicamente el directorio dado,
// no cambia; cualquier error de resolución deja el comportamiento de siempre.
func validationTarget(baseDir string) (master, configFile string, notes []string) {
	master, configFile = baseDir, filepath.Join(baseDir, workspace.DefaultConfigFilename)
	if gitcommon.LinkedWorktreeRoot(baseDir) == "" {
		return master, configFile, nil
	}
	if own, err := workspace.LocateAt(baseDir); err != nil || own.Origin != workspace.OriginStart {
		return master, configFile, nil
	}
	loc, err := workspace.Locate(baseDir)
	if err != nil || loc.Origin == workspace.OriginStart {
		return master, configFile, nil
	}
	master = loc.Master
	configFile = loc.ConfigPath
	if configFile == "" {
		configFile = filepath.Join(master, workspace.DefaultConfigFilename)
	}
	return master, configFile, loc.Notes
}
