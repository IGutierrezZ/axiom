package changestart

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/fileprim"
)

const (
	defaultLockWait = 5 * time.Second
	lockInterval    = 100 * time.Millisecond
)

// lockPath es la ruta del cerrojo <v1>/.locks/<tipo>-<nombre>.lock (D-12). Los nombres que empiezan
// por punto no pueden colisionar con un cambio.
func (e Env) lockPath(kind, name string) string {
	return filepath.Join(e.v1(), ".locks", kind+"-"+name+".lock")
}

// lockItem toma el cerrojo de la identidad del work item (solo la fase 1).
func (e Env) lockItem(plugin, id string) (func(), *Refusal) {
	return e.acquire(e.lockPath("item", plugin+"-"+id))
}

// lockChange toma el cerrojo del cambio (fase 1, fase 2 y descarte).
func (e Env) lockChange(change string) (func(), *Refusal) {
	return e.acquire(e.lockPath("change", change))
}

// acquire toma el cerrojo consultivo con intentos cada 100 ms durante, como mucho, la espera
// configurada (5 s). Ocupado tras la espera es change_start_busy; no poder crear o abrir el fichero es
// write_failed. El sistema operativo libera el bloqueo de un proceso que murió y el fichero queda
// vacío: un cerrojo huérfano se recupera solo y los ficheros de cerrojo no se borran (D-24).
func (e Env) acquire(path string) (func(), *Refusal) {
	wait := e.LockWait
	if wait <= 0 {
		wait = defaultLockWait
	}
	deadline := time.Now().Add(wait)
	for {
		lock, err := fileprim.AcquireLock(path)
		if err == nil {
			return func() { _ = lock.Release() }, nil
		}
		if !errors.Is(err, fileprim.ErrLockContended) {
			return nil, refuse(CodeWriteFailed, err, "step", "lock")
		}
		if !time.Now().Before(deadline) {
			return nil, refuse(CodeChangeStartBusy, err)
		}
		time.Sleep(lockInterval)
	}
}
