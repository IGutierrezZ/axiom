package gitcommon

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// RepositoryRoot devuelve la raíz canónica del árbol de trabajo que contiene
// path: la ruta pedida se hace absoluta y se resuelven sus enlaces, git da la
// raíz (`rev-parse --show-toplevel`) y esa raíz también se resuelve. La
// contención la decide el sistema de ficheros (pathidentity), no la grafía:
// en un volumen que no distingue mayúsculas la misma carpeta escrita de dos
// formas es el mismo repositorio. Fuera de un repositorio devuelve un error que
// envuelve ErrNotGitRepository.
func RepositoryRoot(ctx context.Context, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("la ruta del repositorio es obligatoria")
	}
	requested, err := canonicalPath(path)
	if err != nil {
		return "", err
	}
	top, err := TopLevel(ctx, requested)
	if err != nil {
		return "", err
	}
	root, err := canonicalPath(top)
	if err != nil {
		return "", err
	}
	if !pathidentity.Contains(root, requested) {
		return "", fmt.Errorf("la raíz %s que da git no contiene la ruta pedida %s", root, requested)
	}
	return root, nil
}

// IsAncestor responde si ancestor es antecesor de descendant en el repositorio
// que contiene dir (`git merge-base --is-ancestor`). git termina con 0 cuando lo
// es y con 1 cuando no; cualquier otro código (una referencia que no existe, un
// directorio que no es un repositorio) es un error con el mensaje de git.
func IsAncestor(ctx context.Context, dir, ancestor, descendant string) (bool, error) {
	root, err := RepositoryRoot(ctx, dir)
	if err != nil {
		return false, err
	}
	result, err := Run(ctx, root, "merge-base", "--is-ancestor", ancestor, descendant)
	if err != nil {
		return false, err
	}
	switch result.ExitCode {
	case 0:
		return true, nil
	case 1:
		return false, nil
	default:
		return false, fmt.Errorf("git merge-base --is-ancestor %s %s en %s terminó con el código %d: %s", ancestor, descendant, root, result.ExitCode, result.Stderr)
	}
}

func canonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}
