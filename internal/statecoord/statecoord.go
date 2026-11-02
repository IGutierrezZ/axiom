// Package statecoord serializes read-modify-write mutations of the shared install
// state. All such operations use one lock identity derived from the resolved
// home directory, so each mutation reads the latest state and writes it before
// releasing the shared lock. El estado de un proyecto (state.ProjectStore) tiene
// su propio cerrojo junto a su fichero, con la misma disciplina.
package statecoord

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// LockPath returns the canonical install-state lock path for homeDir.
func LockPath(homeDir string) (string, error) {
	resolvedHome, err := filepath.EvalSymlinks(homeDir)
	if err != nil {
		return "", fmt.Errorf("resolve install state home: %w", err)
	}
	return state.Path(resolvedHome) + ".lock", nil
}

// StoreLockPath devuelve la ruta del cerrojo del almacén. En el global es la de
// LockPath; en el de proyecto es `<raíz>/.axiom/state.json.lock`, con la raíz
// resuelta por enlaces simbólicos para que dos grafías de la misma carpeta
// compartan cerrojo.
func StoreLockPath(store state.Store) (string, error) {
	if !store.IsProject() {
		return LockPath(store.Home())
	}
	resolvedRoot, err := filepath.EvalSymlinks(store.Root())
	if err != nil {
		return "", fmt.Errorf("resolve project state root: %w", err)
	}
	return state.ProjectPath(resolvedRoot) + ".lock", nil
}

// WithLock runs operation while holding the canonical install-state lock shared
// by every install-state read-modify-write operation for the resolved home.
func WithLock(homeDir string, operation func() error) (err error) {
	return WithStoreLock(state.GlobalStore(homeDir), operation)
}

// WithStoreLock ejecuta operation con el cerrojo del almacén: el global comparte el
// de WithLock y cada proyecto tiene el suyo. Los cerrojos de almacenes distintos no
// se anidan: quien lee la semilla global desde un almacén de proyecto lo hace sin
// cerrojo, porque el estado se publica con escritura atómica.
func WithStoreLock(store state.Store, operation func() error) (err error) {
	// El cerrojo de proyecto vive en `.axiom/`: se prepara antes para que el
	// directorio exista y git ya excluya fichero y cerrojo cuando este se cree.
	store.Prepare()
	lockPath, err := StoreLockPath(store)
	if err != nil {
		return fmt.Errorf("acquire install state lock: %w", err)
	}
	lock, err := reviewtransaction.AcquireAuthorityFileLock(lockPath)
	if err != nil {
		return fmt.Errorf("acquire install state lock: %w", err)
	}
	defer func() {
		if releaseErr := lock.Release(); releaseErr != nil {
			err = errors.Join(err, fmt.Errorf("release install state lock: %w", releaseErr))
		}
	}()
	return operation()
}
