package fileprim

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LockSchema es el esquema del registro de propietario que se escribe en el
// fichero de cerrojo. El valor conserva el nombre histórico del almacén de
// revisión porque es compatibilidad de formato en disco: cambiarlo dejaría
// ilegibles los cerrojos ya existentes.
const LockSchema = "gentle-ai.review-store-lock/v1"

// LockCoordinationBytes is the byte range the Windows lock primitive
// coordinates. POSIX flock remains whole-file advisory.
const LockCoordinationBytes int64 = 1

// LockOwner es el registro de propietario que el cerrojo deja en el fichero.
// Es solo una pista: la verdad la dan los cerrojos consultivos del núcleo.
type LockOwner struct {
	Schema     string    `json:"schema"`
	OwnerID    string    `json:"owner_id"`
	PID        int       `json:"pid"`
	Host       string    `json:"host"`
	AcquiredAt time.Time `json:"acquired_at"`
}

// NewLockOwner genera un registro de propietario con un identificador aleatorio.
func NewLockOwner() (LockOwner, error) {
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return LockOwner{}, fmt.Errorf("generate review store lock owner: %w", err)
	}
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}
	return LockOwner{
		Schema: LockSchema, OwnerID: hex.EncodeToString(token[:]), PID: os.Getpid(),
		Host: host, AcquiredAt: time.Now().UTC(),
	}, nil
}

// ErrLockContended reports the one refusal that proves, from where it is
// produced rather than from its text, that nothing was mutated: the advisory
// lock was already held, so the non-blocking acquisition syscall refused and
// the guarded body never ran.
var ErrLockContended = errors.New("authoritative review store advisory lock is held")

// concurrentUpdateText es el texto de ErrConcurrentUpdate de reviewtransaction.
// El error de cerrojo ocupado debe renderizar el mismo mensaje que ese paquete
// ofrecía, sin importarlo.
const concurrentUpdateText = "review transaction changed concurrently"

type lockBusyError struct{}

func (lockBusyError) Error() string {
	return concurrentUpdateText + ": authoritative review store advisory lock is held; persisted PID and host metadata are not current-holder proof"
}

func (lockBusyError) Unwrap() error { return ErrLockContended }

// PreAcquisitionError reports a failure that happened before the lock was
// acquired: directory creation, the secure open walk, or the advisory lock
// syscall itself all fail before any guarded mutation, so callers must
// classify this as not-started rather than an unknown-mutation outcome.
type PreAcquisitionError struct {
	Err error
}

func (err *PreAcquisitionError) Error() string {
	if err == nil || err.Err == nil {
		// refusal:by-design world-action: this sentence renders only for a degenerately constructed error with no cause, which no operator command can reach; the exit is a code fix at the construction site.
		return "review store lock could not be acquired"
	}
	return fmt.Sprintf("review store lock could not be acquired: %v", err.Err)
}

func (err *PreAcquisitionError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Err
}

// AnchorResolver devuelve, para la ruta absoluta de un cerrojo, la raíz desde
// la que el recorrido seguro de directorios debe empezar. Si informa false, el
// recorrido parte de la raíz del sistema de ficheros.
type AnchorResolver func(absPath string) (root string, ok bool)

// Lock es un cerrojo consultivo exclusivo sobre un fichero privado, abierto sin
// seguir enlaces.
type Lock struct {
	file  *os.File
	owner LockOwner
}

// File devuelve el descriptor del fichero bloqueado.
func (lock *Lock) File() *os.File { return lock.file }

// Owner devuelve el registro de propietario escrito al adquirir.
func (lock *Lock) Owner() LockOwner { return lock.owner }

// AcquireLock abre y bloquea de forma exclusiva el fichero de cerrojo en path,
// con el recorrido seguro anclado en la raíz del sistema de ficheros.
func AcquireLock(path string) (*Lock, error) { return AcquireLockAnchored(path, nil) }

// AcquireLockAnchored es AcquireLock con un recorrido seguro anclado donde
// indique anchor.
func AcquireLockAnchored(path string, anchor AnchorResolver) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, &PreAcquisitionError{Err: err}
	}
	file, err := SecureOpenLock(path, anchor)
	if err != nil {
		return nil, &PreAcquisitionError{Err: err}
	}
	locked, err := TryLock(file)
	if err != nil {
		_ = file.Close()
		return nil, &PreAcquisitionError{Err: fmt.Errorf("acquire bounded review store lock: %w", err)}
	}
	if !locked {
		_ = file.Close()
		return nil, lockBusyError{}
	}

	owner, err := NewLockOwner()
	if err != nil {
		_ = Unlock(file)
		_ = file.Close()
		return nil, err
	}
	payload, err := json.Marshal(owner)
	if err != nil {
		_ = Unlock(file)
		_ = file.Close()
		return nil, err
	}
	payload = append(payload, '\n')
	if err := file.Truncate(0); err == nil {
		_, err = file.Seek(0, 0)
	}
	if err == nil {
		_, err = file.Write(payload)
	}
	if err == nil {
		err = file.Sync()
	}
	if err != nil {
		_ = Unlock(file)
		_ = file.Close()
		return nil, fmt.Errorf("write review store lock owner: %w", err)
	}
	return &Lock{file: file, owner: owner}, nil
}

// Release relinquishes the lock. It is safe on a nil or already released Lock.
func (lock *Lock) Release() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	err := ReleaseLockFile(lock.file)
	lock.file = nil
	return err
}

// ReleaseLockFile clears the owner payload while the advisory lock is still
// held, so no observer ever reads an exited process as the recorded holder
// (#2504), then unlocks and closes the file. Kernel ownership stays the truth;
// the payload is only ever a hint.
func ReleaseLockFile(file *os.File) error {
	truncateErr := file.Truncate(0)
	unlockErr := Unlock(file)
	closeErr := file.Close()
	return errors.Join(truncateErr, unlockErr, closeErr)
}

// TryLock intenta bloquear file de forma exclusiva sin esperar.
func TryLock(file *os.File) (bool, error) { return TryLockMode(file, true) }
