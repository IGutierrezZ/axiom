package changestart

import (
	"errors"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

// ErrNoRepository indica que la raíz del workspace no pertenece a ningún repositorio git: sin él no
// hay common dir donde dejar el staging ni registro de ramas que leer.
var ErrNoRepository = errors.New("la raíz del workspace no está en un repositorio git: repite `axiom change start … --cwd <repositorio>`")

// ResolveRepository devuelve la raíz git y el common dir del repositorio que contiene la raíz del
// workspace (D-23), leídos de ficheros y sin ejecutar git. Es el repositorio cuyo common dir usa el
// motor de ramas para su registro, de modo que staging, cerrojos y registro conviven.
func ResolveRepository(workspaceRoot string) (gitRoot, commonDir string, err error) {
	root, ok := gitcommon.GitRootFromFiles(workspaceRoot)
	if !ok {
		return "", "", ErrNoRepository
	}
	common, err := gitcommon.CommonDirFromFiles(root)
	if err != nil {
		return "", "", errors.Join(ErrNoRepository, err)
	}
	return root, common, nil
}
