package workspace

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// locateFromMainCheckout resuelve start, que está dentro del worktree enlazado de
// raíz worktreeRoot y no halló ningún marcador dentro de él, como lo haría desde el
// checkout principal del mismo clon (REQ-25.28.2, supuesto S-11, D-15). Así, un
// worktree de un repositorio de código de un multirepo, creado fuera de la carpeta
// maestra, resuelve el mismo planning home que su checkout principal.
//
// Solo lee ficheros (gitcommon.CommonDirFromFiles y MainWorktreeFromFiles): este
// paquete nunca lanza procesos. Si el checkout principal no se puede determinar por
// ficheros (un common dir que no se llama `.git`, un repositorio desnudo o ilegible)
// o tampoco tiene marcador, la resolución es la del directorio de partida con una
// nota que lo explica, nunca un rechazo (REQ-24.2).
//
// La ubicación del worktree nunca interviene en la deducción de la carpeta maestra
// (REQ-25.28.3): la ascensión y deduceMaster trabajan sobre rutas del checkout
// principal.
func locateFromMainCheckout(start, worktreeRoot string) (Location, error) {
	commonDir, err := gitcommon.CommonDirFromFiles(worktreeRoot)
	if err != nil {
		return unresolvedWorktree(start, fmt.Sprintf(
			"El directorio %s está en un worktree enlazado sin marcador propio y no se pudo leer su common dir de git (%v); se resuelve desde el directorio de partida.",
			pathquote.Quote(start), err)), nil
	}
	main, ok := gitcommon.MainWorktreeFromFiles(commonDir)
	if !ok {
		return unresolvedWorktree(start, fmt.Sprintf(
			"El directorio %s está en un worktree enlazado sin marcador propio y su common dir %s no se llama `.git`, así que no se deduce su checkout principal; se resuelve desde el directorio de partida.",
			pathquote.Quote(start), pathquote.Quote(commonDir))), nil
	}
	if !isDir(main) {
		return unresolvedWorktree(start, fmt.Sprintf(
			"El directorio %s está en un worktree enlazado sin marcador propio y su checkout principal %s no existe; se resuelve desde el directorio de partida.",
			pathquote.Quote(start), pathquote.Quote(main))), nil
	}
	dir, origin, found := findMarker(main, true, "")
	if !found {
		return unresolvedWorktree(start, fmt.Sprintf(
			"El directorio %s está en un worktree enlazado sin marcador propio y su checkout principal %s tampoco tiene marcador ascendente; se resuelve desde el directorio de partida.",
			pathquote.Quote(start), pathquote.Quote(main))), nil
	}
	loc, err := locateFromMarker(start, dir, origin)
	if err != nil {
		return Location{}, err
	}
	loc.Notes = append(loc.Notes, fmt.Sprintf(
		"El directorio %s está en un worktree enlazado sin marcador propio; se resuelve como desde su checkout principal %s.",
		pathquote.Quote(start), pathquote.Quote(main)))
	return loc, nil
}

// unresolvedWorktree es la resolución por directorio de partida con la nota que
// explica por qué un worktree enlazado no se resolvió desde su checkout principal.
func unresolvedWorktree(start, note string) Location {
	loc := startLocation(start)
	loc.Notes = []string{note}
	return loc
}
