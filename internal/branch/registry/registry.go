// Package registry es el registro local de `axiom branch` (INC-25, design D-07 y §4.4):
// un fichero JSON por cambio y carril, `<git common dir>/axiom/branch/v1/<carril>/<cambio>.json`,
// con el plan materializado del cambio (modo de trabajo, excepciones y, por repositorio,
// sus ramas y worktrees con rutas absolutas). Es estado local no versionado: vive en el
// git common dir del repositorio del planning home, de modo que el checkout principal y
// cualquier worktree enlazado lo ven igual, y borrarlo no daña el repositorio.
//
// Trae la lectura (`Load` y `List`) y, desde `ensure`, la escritura (`Save`, que no toca el
// fichero si los bytes no cambian) y el bloqueo exclusivo por cambio (`Lock`). El borrado
// llega con `done`, su primer llamador. El paquete no ejecuta procesos: el motor obtiene el
// common dir con git y se lo entrega, y reconstruye el registro desde el estado de git.
package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/reviewtransaction"
)

// Store lee y escribe el registro local.
type Store struct{}

// New devuelve el almacén del registro local.
func New() *Store { return &Store{} }

// ErrBusy dice que otra ejecución de `axiom branch` tiene bloqueado el cambio (`Lock`). El
// motor lo traduce a `plan_conflict` con este mismo texto (design D-05).
var ErrBusy = errors.New("otra ejecución de `axiom branch` está modificando el cambio")

// folder es la carpeta del registro de un carril: `<common dir>/axiom/branch/v1/<carril>`.
// El registro y su bloqueo viven aquí y nunca bajo `<common dir>/gentle-ai/` (design D-07).
func folder(commonDir string, lane branch.Lane) string {
	return filepath.Join(commonDir, "axiom", "branch", "v1", string(lane))
}

// Lock bloquea en exclusiva el cambio y el carril dados, con `<cambio>.lock` junto al
// registro, mediante el bloqueo de ficheros de reviewtransaction. No espera: si otra
// ejecución lo tiene, devuelve ErrBusy. unlock lo libera y puede llamarse más de una vez.
// Una clave que no puede componer la ruta de un registro, o un contexto ya cancelado,
// devuelven un error corriente sin crear nada.
func (*Store) Lock(ctx context.Context, commonDir string, lane branch.Lane, change string) (unlock func(), err error) {
	if err := validateKey(commonDir, lane, change); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path := filepath.Join(folder(commonDir, lane), change+".lock")
	lock, err := reviewtransaction.AcquireAuthorityFileLock(path)
	switch {
	case errors.Is(err, reviewtransaction.ErrStoreLockContended):
		return nil, ErrBusy
	case err != nil:
		return nil, &WriteError{Path: path, Cause: fmt.Errorf("no se pudo bloquear el cambio: %w", err)}
	}
	var once sync.Once
	return func() { once.Do(func() { _ = lock.Release() }) }, nil
}

// WriteError es el fallo de escribir o bloquear un fichero del registro: el directorio no se
// puede crear o el fichero no se puede escribir ni publicar. El motor lo traduce a
// `config_invalid`: el registro es estado local derivado y sin él `ensure` lo reconstruye.
type WriteError struct {
	Path  string // absoluta
	Cause error
}

func (e *WriteError) Error() string {
	return fmt.Sprintf("no se pudo escribir el registro local %s: %v", e.Path, e.Cause)
}

func (e *WriteError) Unwrap() error { return e.Cause }

// Save guarda el registro del cambio en `<common dir>/axiom/branch/v1/<carril>/<cambio>.json`
// con una serialización estable (JSON con sangrado de dos espacios, `\n` y sin marcas de
// tiempo). Si el fichero ya tiene esos mismos bytes no lo toca; si no, lo publica de forma
// atómica con un temporal de la misma carpeta (las primitivas de kickoff.AppendGate:
// ReplaceFileAtomic y SyncReviewDirectory). Rechaza, sin crear nada, un registro que Load
// no aceptaría. Quien llama tiene el cambio bloqueado con Lock.
func (*Store) Save(commonDir string, record branch.Record) error {
	if err := validateKey(commonDir, record.Lane, record.Change); err != nil {
		return err
	}
	if err := validate(record, record.Lane, record.Change); err != nil {
		return fmt.Errorf("el registro de %q no es válido: %w", record.Change, err)
	}
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar el registro de %q: %w", record.Change, err)
	}
	payload = append(payload, '\n')

	dir := folder(commonDir, record.Lane)
	path := filepath.Join(dir, record.Change+".json")
	switch current, err := os.ReadFile(path); {
	case err == nil && bytes.Equal(current, payload):
		return nil
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		return &WriteError{Path: path, Cause: err}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &WriteError{Path: dir, Cause: err}
	}
	// El temporal no acaba en `.json`: List solo cuenta los `<cambio>.json`.
	temporary, err := os.CreateTemp(dir, "."+record.Change+"-*.tmp")
	if err != nil {
		return &WriteError{Path: path, Cause: err}
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if _, err := temporary.Write(payload); err != nil {
		_ = temporary.Close()
		return &WriteError{Path: path, Cause: err}
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return &WriteError{Path: path, Cause: err}
	}
	if err := temporary.Close(); err != nil {
		return &WriteError{Path: path, Cause: err}
	}
	if err := reviewtransaction.ReplaceFileAtomic(temporaryPath, path); err != nil {
		return &WriteError{Path: path, Cause: err}
	}
	if err := reviewtransaction.SyncReviewDirectory(dir); err != nil {
		return &WriteError{Path: dir, Cause: err}
	}
	return nil
}

// FileError es el fallo de un fichero del registro que existe: no se puede leer, no es
// un JSON válido o su contenido no es coherente con el esquema o con el cambio al que
// pertenece. El motor lo traduce a `config_invalid` con la continuación de borrarlo: el
// registro es estado derivado y `ensure` lo reconstruye.
type FileError struct {
	Path  string // absoluta
	Cause error
}

func (e *FileError) Error() string {
	return fmt.Sprintf("el registro local %s no se puede leer o no es válido: %v", e.Path, e.Cause)
}

func (e *FileError) Unwrap() error { return e.Cause }

// Load lee el registro del cambio en el git common dir dado. Un registro que no existe
// no es un error: devuelve nil, nil. Un fichero que existe pero no se puede leer o no es
// válido devuelve un *FileError. Un carril, un nombre de cambio o un common dir que no
// pueden componer la ruta de un registro devuelven un error corriente: el motor los
// valida antes y la lectura nunca sale de la carpeta del registro. No escribe nada.
func (*Store) Load(commonDir string, lane branch.Lane, change string) (*branch.Record, error) {
	if err := validateKey(commonDir, lane, change); err != nil {
		return nil, err
	}
	path := filepath.Join(folder(commonDir, lane), change+".json")
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, &FileError{Path: path, Cause: err}
	}
	var record branch.Record
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, &FileError{Path: path, Cause: fmt.Errorf("no es un JSON válido: %w", err)}
	}
	if err := validate(record, lane, change); err != nil {
		return nil, &FileError{Path: path, Cause: err}
	}
	return &record, nil
}

// List lee los registros de todos los cambios del carril en el git common dir dado,
// ordenados por nombre de cambio (no por nombre de fichero: `a-b.json` precede a
// `a.json`). Sin carpeta del carril no hay registros y devuelve una lista vacía, no nil.
// Solo cuentan los ficheros `<cambio>.json`; lo demás de la carpeta se ignora. Un
// registro que existe y no se puede leer o no es válido devuelve un *FileError que lo
// nombra, y ninguna lista a medias: el registro es estado derivado y no se interpreta con
// indulgencia (como en Load). Un carril o un common dir que no pueden componer la carpeta
// del registro devuelven un error corriente. No escribe nada.
func (s *Store) List(commonDir string, lane branch.Lane) ([]branch.Record, error) {
	if err := validateScope(commonDir, lane); err != nil {
		return nil, err
	}
	dir := folder(commonDir, lane)
	entries, err := os.ReadDir(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return []branch.Record{}, nil
	case err != nil:
		return nil, &FileError{Path: dir, Cause: err}
	}
	records := []branch.Record{}
	for _, entry := range entries {
		change, isRecord := strings.CutSuffix(entry.Name(), ".json")
		if !isRecord {
			continue
		}
		record, err := s.Load(commonDir, lane, change)
		if err != nil {
			// Un nombre de fichero que no puede ser el de un cambio no es un registro que
			// escribiera `ensure`: se nombra el fichero como en cualquier otro fallo.
			var failure *FileError
			if !errors.As(err, &failure) {
				err = &FileError{Path: filepath.Join(dir, entry.Name()), Cause: err}
			}
			return nil, err
		}
		if record != nil {
			records = append(records, *record)
		}
	}
	slices.SortFunc(records, func(a, b branch.Record) int { return strings.Compare(a.Change, b.Change) })
	return records, nil
}

// validateScope comprueba que el common dir y el carril componen la carpeta de un
// registro sin salir de ella.
func validateScope(commonDir string, lane branch.Lane) error {
	if !filepath.IsAbs(commonDir) {
		return fmt.Errorf("el git common dir %q no es una ruta absoluta", commonDir)
	}
	if lane != branch.LaneSDD && lane != branch.LaneODD {
		return fmt.Errorf("el carril %q no es válido: usa %s u %s", lane, branch.LaneSDD, branch.LaneODD)
	}
	return nil
}

// validateKey comprueba que los tres datos componen la ruta de un registro sin salir de
// su carpeta.
func validateKey(commonDir string, lane branch.Lane, change string) error {
	if err := validateScope(commonDir, lane); err != nil {
		return err
	}
	if err := gitref.ValidBranchComponent(change); err != nil {
		return fmt.Errorf("el nombre de cambio %q no puede nombrar un registro: %w", change, err)
	}
	return nil
}

// validate comprueba que el contenido de un registro es coherente: el esquema que este
// código entiende, el carril y el cambio a los que pertenece el fichero, un modo
// materializado y su origen, rutas absolutas y excepciones con nombres válidos. Los
// campos que no conoce se ignoran: el esquema v1 puede crecer sin romper a un binario
// anterior.
func validate(record branch.Record, lane branch.Lane, change string) error {
	switch {
	case record.Schema != branch.RegistrySchema:
		return fmt.Errorf("el esquema %q no es %s", record.Schema, branch.RegistrySchema)
	case record.Lane != lane:
		return fmt.Errorf("el carril del registro es %q y se pidió %q", record.Lane, lane)
	case record.Change != change:
		return fmt.Errorf("el cambio del registro es %q y se pidió %q", record.Change, change)
	case record.Mode != branch.ModeWorktree && record.Mode != branch.ModeInPlace:
		return fmt.Errorf("el modo %q no está materializado: debe ser %s o %s", record.Mode, branch.ModeWorktree, branch.ModeInPlace)
	case !knownModeSource(record.ModeSource):
		return fmt.Errorf("el origen del modo %q no es válido: debe ser %s, %s, %s o %s", record.ModeSource,
			branch.ModeSourceAnswer, branch.ModeSourceFlag, branch.ModeSourceHub, branch.ModeSourceDefault)
	}
	if err := absolute("planningRepo", record.PlanningRepo); err != nil {
		return err
	}
	for i, repo := range record.Repositories {
		if err := absolute(fmt.Sprintf("repositories[%d].gitRoot", i), repo.GitRoot); err != nil {
			return err
		}
		if err := absolute(fmt.Sprintf("repositories[%d].commonDir", i), repo.CommonDir); err != nil {
			return err
		}
		for j, registered := range repo.Branches {
			if registered.Worktree == "" {
				continue
			}
			if err := absolute(fmt.Sprintf("repositories[%d].branches[%d].worktree", i, j), registered.Worktree); err != nil {
				return err
			}
		}
	}
	if prefix := record.Exceptions.Prefix; prefix != "" {
		if err := gitref.ValidPrefix(prefix); err != nil {
			return fmt.Errorf("exceptions.prefix %q no es válido: %w", prefix, err)
		}
	}
	if name := record.Exceptions.Branch; name != "" {
		if err := gitref.ValidRefName(name); err != nil {
			return fmt.Errorf("exceptions.branch %q no es válido: %w", name, err)
		}
	}
	// El prefijo de la rama de archivado (`ensure --archive --prefix`, REQ-25.12) es un prefijo
	// de rama como el de las excepciones.
	if prefix := record.ArchivePrefix; prefix != "" {
		if err := gitref.ValidPrefix(prefix); err != nil {
			return fmt.Errorf("archivePrefix %q no es válido: %w", prefix, err)
		}
	}
	return nil
}

func knownModeSource(source string) bool {
	switch source {
	case branch.ModeSourceAnswer, branch.ModeSourceFlag, branch.ModeSourceHub, branch.ModeSourceDefault:
		return true
	}
	return false
}

// absolute exige que una ruta del registro sea absoluta (REQ-25.2, REQ-25.14).
func absolute(field, path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s %q no es una ruta absoluta", field, path)
	}
	return nil
}
