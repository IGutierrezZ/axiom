// Package registry es el registro local de `axiom branch` (INC-25, design D-07 y §4.4):
// un fichero JSON por cambio y carril, `<git common dir>/axiom/branch/v1/<carril>/<cambio>.json`,
// con el plan materializado del cambio (modo de trabajo, excepciones y, por repositorio,
// sus ramas y worktrees con rutas absolutas). Es estado local no versionado: vive en el
// git common dir del repositorio del planning home, de modo que el checkout principal y
// cualquier worktree enlazado lo ven igual, y borrarlo no daña el repositorio.
//
// Esta rebanada trae solo la lectura (`Load`); quien escribe el registro es `ensure`,
// que llega con las rebanadas siguientes. El paquete no ejecuta procesos: el motor
// obtiene el common dir con git y se lo entrega.
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
)

// Store lee el registro local.
type Store struct{}

// New devuelve el lector del registro local.
func New() *Store { return &Store{} }

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
	path := filepath.Join(commonDir, "axiom", "branch", "v1", string(lane), change+".json")
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

// validateKey comprueba que los tres datos componen la ruta de un registro sin salir de
// su carpeta.
func validateKey(commonDir string, lane branch.Lane, change string) error {
	if !filepath.IsAbs(commonDir) {
		return fmt.Errorf("el git common dir %q no es una ruta absoluta", commonDir)
	}
	if lane != branch.LaneSDD && lane != branch.LaneODD {
		return fmt.Errorf("el carril %q no es válido: usa %s u %s", lane, branch.LaneSDD, branch.LaneODD)
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
