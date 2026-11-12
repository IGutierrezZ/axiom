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
// ida. La restauración usa siempre las funciones reales del sistema.
var (
	writeTempFile = os.WriteFile
	renameFile    = os.Rename
	syncIndex     = syncIndexOf
)

const tempName = ".spec.md.axiom-tmp"

// applied es lo necesario para deshacer la escritura de una especificación.
type applied struct {
	path        string
	previous    []byte
	existed     bool
	createdDirs []string // de la más interior a la más exterior
}

// Apply escribe todas las especificaciones del plan o ninguna (REQ-30.2): primero
// escribe un temporal por capacidad junto a su destino y, solo si todos se escriben,
// los renombra; ante cualquier fallo restaura lo ya renombrado y quita lo creado. Con
// una integración sin cambios no toca el disco. Si el proyecto ya tiene
// openspec/INDEX.md lo regenera con livingdoc.Service.Sync y, si eso falla, también lo
// deshace. Devuelve las rutas absolutas escritas, con el índice al final si cambió.
func (i Integration) Apply() ([]string, error) {
	var todo []CapabilityPlan
	for _, c := range i.Capabilities {
		if c.Outcome != OutcomeAlready {
			todo = append(todo, c)
		}
	}
	if len(todo) == 0 {
		return nil, nil
	}

	staged := make([]string, 0, len(todo))
	cleanTemps := func() {
		for _, tmp := range staged {
			_ = os.Remove(tmp)
		}
	}
	var undo []applied
	for _, c := range todo {
		entry := applied{path: c.Path}
		if data, err := os.ReadFile(c.Path); err == nil {
			entry.previous, entry.existed = data, true
		} else if !errors.Is(err, fs.ErrNotExist) {
			cleanTemps()
			rollbackDirs(undo)
			return nil, fmt.Errorf("no se puede leer %s: %w; no se ha escrito nada; comprueba los permisos y vuelve a ejecutar `axiom archive integrate --odd %s`", c.Path, err, i.Feature)
		}
		dirs, err := makeDirs(filepath.Dir(c.Path))
		entry.createdDirs = dirs
		undo = append(undo, entry)
		if err != nil {
			cleanTemps()
			rollbackDirs(undo)
			return nil, fmt.Errorf("no se puede crear la carpeta de la capacidad %s: %w; no se ha escrito nada; comprueba los permisos y vuelve a ejecutar `axiom archive integrate --odd %s`", c.Name, err, i.Feature)
		}
		tmp := filepath.Join(filepath.Dir(c.Path), tempName)
		staged = append(staged, tmp)
		if err := writeTempFile(tmp, []byte(c.Content), 0o644); err != nil {
			cleanTemps()
			rollbackDirs(undo)
			return nil, fmt.Errorf("no se puede escribir el temporal de %s: %w; no se ha escrito nada; libera espacio o permisos y vuelve a ejecutar `axiom archive integrate --odd %s`", c.Path, err, i.Feature)
		}
	}

	done := 0
	for n, c := range todo {
		if err := renameFile(staged[n], c.Path); err != nil {
			cleanTemps()
			failures := restore(undo[:done])
			rollbackDirs(undo)
			return nil, rollbackError(fmt.Errorf("no se puede escribir %s: %w", c.Path, err), failures, i.Feature)
		}
		done++
	}

	written := make([]string, 0, len(todo)+1)
	for _, c := range todo {
		written = append(written, c.Path)
	}

	indexPath := filepath.Join(i.loc.PlanningHome, "INDEX.md")
	if previousIndex, err := os.ReadFile(indexPath); err == nil {
		if err := syncIndex(i.loc); err != nil {
			failures := restore(undo)
			rollbackDirs(undo)
			if werr := os.WriteFile(indexPath, previousIndex, 0o644); werr != nil {
				failures = append(failures, indexPath)
			}
			return nil, rollbackError(fmt.Errorf("no se puede regenerar %s: %w", indexPath, err), failures, i.Feature)
		}
		written = append(written, indexPath)
	}
	return written, nil
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

// restore devuelve cada especificación escrita a su contenido anterior o la quita si no
// existía; devuelve las rutas que no pudo restaurar.
func restore(entries []applied) []string {
	var failed []string
	for n := len(entries) - 1; n >= 0; n-- {
		entry := entries[n]
		var err error
		if entry.existed {
			err = os.WriteFile(entry.path, entry.previous, 0o644)
		} else if rmErr := os.Remove(entry.path); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
			err = rmErr
		}
		if err != nil {
			failed = append(failed, entry.path)
		}
	}
	return failed
}

// rollbackDirs quita las carpetas que Apply creó, si están vacías.
func rollbackDirs(entries []applied) {
	for n := len(entries) - 1; n >= 0; n-- {
		for _, dir := range entries[n].createdDirs {
			_ = os.Remove(dir)
		}
	}
}

func rollbackError(cause error, failed []string, feature string) error {
	if len(failed) > 0 {
		return fmt.Errorf("%w; no se pudo restaurar %s: revísalo con `git status` y recupéralo con `git checkout -- <ruta>` antes de repetir la integración", cause, strings.Join(failed, ", "))
	}
	return fmt.Errorf("%w; se ha deshecho lo escrito y no se ha cambiado ninguna especificación; corrige la causa y vuelve a ejecutar `axiom archive integrate --odd %s`", cause, feature)
}
