package livespec

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/livingdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Costuras de las pruebas: inyectan un fallo en cada punto de escritura del camino de
// ida y en la restauración.
var (
	writeTempFile = os.WriteFile
	renameFile    = os.Rename
	restoreFile   = os.WriteFile
	removeFile    = os.Remove
	syncIndex     = syncIndexOf
)

// tempNameOf es el temporal determinista de un destino: `.<base>.axiom-tmp` junto a él. Un
// huérfano de un proceso matado se sobrescribe y se consume en la siguiente ejecución
// real que escribe ese destino.
func tempNameOf(path string) string {
	return filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".axiom-tmp")
}

// applied es lo necesario para deshacer la escritura de un fichero.
type applied struct {
	path        string
	previous    []byte
	existed     bool
	createdDirs []string // de la más interior a la más exterior
}

// fileWrite es un fichero que escribe writeSet. Name solo sirve para nombrar el destino
// en los motivos (la capacidad, normalmente).
type fileWrite struct {
	Path    string
	Content []byte
	Name    string
}

// writeOptions configura writeSet. Retry es la orden que el motivo propone repetir y
// Extra un paso posterior a los renombrados (la regeneración del índice) que devuelve lo
// que escribe y cómo deshacerlo, y que si falla se deshace junto con el resto.
type writeOptions struct {
	Retry string
	Extra func() (written []string, undo func() (failed []string), err error)
}

// writeSet escribe todos los ficheros o ninguno (REQ-47.4): primero un temporal por fichero
// junto a su destino y, solo si todos se escriben, los renombra; ante cualquier fallo
// restaura lo ya renombrado y quita lo creado. Si la restauración falla conserva el
// rollbackError, que nombra cada ruta no restaurada. Sin ficheros no toca el disco.
// Devuelve las rutas absolutas escritas, con las del paso extra al final.
func writeSet(files []fileWrite, opts writeOptions) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}

	staged := make([]string, 0, len(files))
	cleanTemps := func() {
		for _, tmp := range staged {
			_ = removeFile(tmp)
		}
	}
	var undo []applied
	for _, f := range files {
		entry := applied{path: f.Path}
		if data, err := os.ReadFile(f.Path); err == nil {
			entry.previous, entry.existed = data, true
		} else if !errors.Is(err, fs.ErrNotExist) {
			cleanTemps()
			rollbackDirs(undo)
			return nil, fmt.Errorf("no se puede leer %s: %w; no se ha escrito nada; comprueba los permisos y vuelve a ejecutar `%s`", f.Path, err, opts.Retry)
		}
		dirs, err := makeDirs(filepath.Dir(f.Path))
		entry.createdDirs = dirs
		undo = append(undo, entry)
		if err != nil {
			cleanTemps()
			rollbackDirs(undo)
			return nil, fmt.Errorf("no se puede crear la carpeta de la capacidad %s: %w; no se ha escrito nada; comprueba los permisos y vuelve a ejecutar `%s`", f.Name, err, opts.Retry)
		}
		tmp := tempNameOf(f.Path)
		staged = append(staged, tmp)
		if err := writeTempFile(tmp, f.Content, 0o644); err != nil {
			cleanTemps()
			rollbackDirs(undo)
			return nil, fmt.Errorf("no se puede escribir el temporal de %s: %w; no se ha escrito nada; libera espacio o permisos y vuelve a ejecutar `%s`", f.Path, err, opts.Retry)
		}
	}

	done := 0
	for n, f := range files {
		if err := renameFile(staged[n], f.Path); err != nil {
			cleanTemps()
			failures := restore(undo[:done])
			rollbackDirs(undo)
			return nil, rollbackError(fmt.Errorf("no se puede escribir %s: %w", f.Path, err), failures, opts.Retry)
		}
		done++
	}

	written := make([]string, 0, len(files)+1)
	for _, f := range files {
		written = append(written, f.Path)
	}
	if opts.Extra != nil {
		extra, undoExtra, err := opts.Extra()
		if err != nil {
			failures := restore(undo)
			rollbackDirs(undo)
			if undoExtra != nil {
				failures = append(failures, undoExtra()...)
			}
			return nil, rollbackError(err, failures, opts.Retry)
		}
		written = append(written, extra...)
	}
	return written, nil
}

// Apply escribe todas las especificaciones del plan o ninguna (REQ-30.2) con writeSet. Con
// una integración sin cambios no toca el disco. Si el proyecto ya tiene openspec/INDEX.md
// lo regenera con livingdoc.Service.Sync y, si eso falla, también lo deshace. Devuelve las
// rutas absolutas escritas, con el índice al final si cambió.
func (i Integration) Apply() ([]string, error) {
	var files []fileWrite
	for _, c := range i.Capabilities {
		if c.Outcome != OutcomeAlready {
			files = append(files, fileWrite{Path: c.Path, Content: []byte(c.Content), Name: c.Name})
		}
	}
	return writeSet(files, writeOptions{
		Retry: "axiom archive integrate --odd " + i.Feature,
		Extra: func() ([]string, func() []string, error) {
			indexPath := filepath.Join(i.loc.PlanningHome, "INDEX.md")
			previousIndex, err := os.ReadFile(indexPath)
			if err != nil {
				return nil, nil, nil
			}
			restoreIndex := func() []string {
				if werr := restoreFile(indexPath, previousIndex, 0o644); werr != nil {
					return []string{indexPath}
				}
				return nil
			}
			if err := syncIndex(i.loc); err != nil {
				return nil, restoreIndex, fmt.Errorf("no se puede regenerar %s: %w", indexPath, err)
			}
			return []string{indexPath}, restoreIndex, nil
		},
	})
}

// syncIndexOf regenera openspec/INDEX.md con el servicio de documentación viva.
func syncIndexOf(loc workspace.Location) error {
	_, err := livingdoc.NewService(loc.WorkspaceRoot, nil, nil).Sync(context.Background())
	return err
}

// makeDirs crea dir y devuelve las carpetas que no existían, de la más interior a la
// más exterior, para poder quitarlas al deshacer.
func makeDirs(dir string) ([]string, error) {
	var missing []string
	for current := dir; ; current = filepath.Dir(current) {
		if _, err := os.Stat(current); err == nil {
			break
		}
		missing = append(missing, current)
		if filepath.Dir(current) == current {
			break
		}
	}
	return missing, os.MkdirAll(dir, 0o755)
}

// restore devuelve cada fichero escrito a su contenido anterior o lo quita si no
// existía; devuelve las rutas que no pudo restaurar.
func restore(entries []applied) []string {
	var failed []string
	for n := len(entries) - 1; n >= 0; n-- {
		entry := entries[n]
		var err error
		if entry.existed {
			err = restoreFile(entry.path, entry.previous, 0o644)
		} else if rmErr := removeFile(entry.path); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			err = rmErr
		}
		if err != nil {
			failed = append(failed, entry.path)
		}
	}
	return failed
}

// rollbackDirs quita las carpetas que writeSet creó, si están vacías.
func rollbackDirs(entries []applied) {
	for n := len(entries) - 1; n >= 0; n-- {
		for _, dir := range entries[n].createdDirs {
			_ = os.Remove(dir)
		}
	}
}

func rollbackError(cause error, failed []string, retry string) error {
	if len(failed) > 0 {
		return fmt.Errorf("%w; no se pudo restaurar %s: revísalo con `git status` y recupéralo con `git checkout -- <ruta>` antes de repetir la integración", cause, strings.Join(failed, ", "))
	}
	return fmt.Errorf("%w; se ha deshecho lo escrito y no se ha cambiado ninguna especificación; corrige la causa y vuelve a ejecutar `%s`", cause, retry)
}
