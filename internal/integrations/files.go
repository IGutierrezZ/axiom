package integrations

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// Costuras de prueba sobre las primitivas de reviewtransaction, con el mismo patrón que
// internal/kickoff/ledger.go. C-01-e consolidará ambas copias al extraer las primitivas.
var (
	acquireIntegrationsLock = reviewtransaction.AcquireAuthorityFileLock
	replaceIntegrationsFile = reviewtransaction.ReplaceFileAtomic
	syncIntegrationsDir     = reviewtransaction.SyncReviewDirectory
	sleepBeforeLockRetry    = time.Sleep
	// randomInt63n es una costura para fijar el jitter del retroceso en las pruebas.
	randomInt63n = rand.Int63n
)

// lockAcquireAttempts, lockRetryDelay y lockMaxRetryDelay acotan cuánto espera una escritura ante un
// cerrojo ocupado. AcquireAuthorityFileLock no bloquea: quien llama reintenta. El retroceso es
// exponencial con jitter para que los contendientes no reintenten al unísono (copia de
// internal/kickoff/ledger.go).
const (
	lockAcquireAttempts = 200
	lockRetryDelay      = 2 * time.Millisecond
	lockMaxRetryDelay   = 100 * time.Millisecond
)

const (
	// lockFallbackName es el cerrojo de una carpeta ordinaria sin git: sin git no hay árbol sucio
	// que proteger (D-08).
	lockFallbackName = ".integrations.lock"

	linksTempPattern  = ".integrations-*.yaml"
	outboxTempPattern = ".integrations-outbox-*.yaml"
)

// ErrArchivedWithoutGit indica que la carpeta archivada no está en un repositorio git: allí el único
// nombre autorizado es la cola (D-18) y el cerrojo no puede caer a la carpeta, así que no se encola.
var ErrArchivedWithoutGit = errors.New("la carpeta archivada no está en un repositorio git: no se puede encolar la entrega")

var archivedFolderName = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-(.+)$`)

// Folder es una carpeta de cambio donde se encolan operaciones y se guardan vínculos: la ordinaria
// (<workspace>/openspec/changes/<cambio>) o la archivada
// (<workspace>/openspec/changes/archive/<fecha>-<cambio>).
type Folder struct {
	WorkspaceRoot string
	Root          string
}

func (f Folder) relative() string {
	rel, err := filepath.Rel(f.WorkspaceRoot, f.Root)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

// Archived indica si la carpeta es una carpeta archivada.
func (f Folder) Archived() bool {
	rel := f.relative()
	return rel == "openspec/changes/archive" || strings.HasPrefix(rel, "openspec/changes/archive/")
}

// Change devuelve el nombre del cambio: el de la carpeta ordinaria o, en la archivada, el nombre
// sin el prefijo de fecha.
func (f Folder) Change() string {
	base := filepath.Base(f.Root)
	if f.Archived() {
		if m := archivedFolderName.FindStringSubmatch(base); m != nil {
			return m[1]
		}
	}
	return base
}

// guard aplica la guarda de inmutabilidad (D-18) antes de escribir name en la carpeta.
func (f Folder) guard(name string) error {
	return kickoff.RefuseArchivedWriteExceptLocalState(f.WorkspaceRoot, f.Root, name)
}

func (f Folder) path(name string) string { return filepath.Join(f.Root, name) }

// lockPath resuelve el cerrojo sin lanzar procesos: <common dir de git>/axiom/integrations/v1/locks/
// <cambio>.lock. Sin git, una carpeta ordinaria cae a <carpeta>/.integrations.lock; una archivada
// no cae (ErrArchivedWithoutGit).
func (f Folder) lockPath() (string, error) {
	if gitRoot, ok := gitcommon.GitRootFromFiles(f.Root); ok {
		common, err := gitcommon.CommonDirFromFiles(gitRoot)
		if err != nil {
			return "", fmt.Errorf("resolver el directorio común de git de %q: %w", f.Root, err)
		}
		return filepath.Join(common, "axiom", "integrations", "v1", "locks", f.Change()+".lock"), nil
	}
	if f.Archived() {
		return "", ErrArchivedWithoutGit
	}
	return filepath.Join(f.Root, lockFallbackName), nil
}

// acquire toma el cerrojo del cambio con reintento acotado y devuelve su liberación.
func (f Folder) acquire() (release func(), err error) {
	lockPath, err := f.lockPath()
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt < lockAcquireAttempts; attempt++ {
		lock, err := acquireIntegrationsLock(lockPath)
		if err == nil {
			return func() { _ = lock.Release() }, nil
		}
		if !errors.Is(err, reviewtransaction.ErrStoreLockContended) {
			return nil, fmt.Errorf("adquirir el cerrojo de integraciones %q: %w", lockPath, err)
		}
		lastErr = err
		sleepBeforeLockRetry(lockRetryBackoff(attempt))
	}
	return nil, fmt.Errorf("cerrojo de integraciones %q ocupado tras %d intentos: %w", lockPath, lockAcquireAttempts, lastErr)
}

// lockRetryBackoff devuelve cuánto esperar antes del reintento attempt (desde cero): el retardo
// base duplicado por intento, con tope, menos una fracción aleatoria de sí mismo. El jitter resta
// para que el tope sea un techo real, y el resultado nunca baja del retardo base.
func lockRetryBackoff(attempt int) time.Duration {
	delay := lockRetryDelay
	for i := 0; i < attempt && delay < lockMaxRetryDelay; i++ {
		delay *= 2
	}
	if delay > lockMaxRetryDelay {
		delay = lockMaxRetryDelay
	}
	if delay <= lockRetryDelay {
		return lockRetryDelay
	}
	jittered := delay - time.Duration(randomInt63n(int64(delay-lockRetryDelay)))
	if jittered < lockRetryDelay {
		return lockRetryDelay
	}
	return jittered
}

// writeAtomic escribe data en name con un temporal en la misma carpeta y reemplazo atómico: si algo
// falla tras crear el temporal, el fichero conserva su contenido anterior íntegro. La guarda D-18
// se aplica al fichero final y al temporal antes de crear nada.
func (f Folder) writeAtomic(name, tempPattern string, data []byte) error {
	if err := f.guard(name); err != nil {
		return err
	}
	if err := f.guard(strings.Replace(tempPattern, "*", "0", 1)); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(f.Root, tempPattern)
	if err != nil {
		return fmt.Errorf("crear el temporal de %s: %w", name, err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }() // tras el reemplazo ya no existe: es inocuo

	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("escribir el temporal de %s: %w", name, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("cerrar el temporal de %s: %w", name, err)
	}
	if err := os.Chmod(temporaryPath, 0o644); err != nil {
		return fmt.Errorf("fijar los permisos del temporal de %s: %w", name, err)
	}
	if err := replaceIntegrationsFile(temporaryPath, f.path(name)); err != nil {
		return fmt.Errorf("publicar %s: %w", name, err)
	}
	if err := syncIntegrationsDir(f.Root); err != nil {
		return fmt.Errorf("sincronizar el directorio de %s: %w", name, err)
	}
	return nil
}

// remove borra name de la carpeta (la ausencia no es un error), con la guarda D-18.
func (f Folder) remove(name string) error {
	if err := f.guard(name); err != nil {
		return err
	}
	if err := os.Remove(f.path(name)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("borrar %s: %w", name, err)
	}
	if err := syncIntegrationsDir(f.Root); err != nil {
		return fmt.Errorf("sincronizar el directorio de %s: %w", name, err)
	}
	return nil
}
