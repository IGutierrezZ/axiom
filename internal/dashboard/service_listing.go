package dashboard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Listados que agregan los cambios de los worktrees registrados y los pendientes de archivar (INC-25,
// REQ-25.34 y REQ-25.20, design D-16, D-17 y D-20). El dashboard no reimplementa ni la lectura del
// registro local ni la precedencia ni la evaluación de pendientes: las toma de engine.RegisteredChanges
// y engine.PendingArchive, los mismos que usan `axiom sdd status` y `axiom branch status`.

// incrementLocation es dónde vive un incremento y de dónde se leen sus artefactos. WorktreePath y
// Branch solo están cuando el cambio se lee de un worktree registrado y no del checkout principal.
type incrementLocation struct {
	path         string
	kind         string // "active" o "archived"
	worktreePath string
	branch       string
}

// registeredIncrements devuelve, por nombre, el cambio activo que gana la precedencia entre los de los
// worktrees registrados (REQ-25.34, D-16): el del worktree de archivado antes que el del worktree del
// cambio. Solo cuentan los worktrees que tienen la carpeta del cambio. Los avisos son los del motor
// (worktree desaparecido, registro ilegible). La agregación es la del motor, la misma que usa la TUI.
func registeredIncrements(loc workspace.Location) (map[string]engine.RegisteredChange, []string) {
	return engine.Default().RegisteredChangesByName(context.Background(), loc)
}

// locateIncrement busca el incremento por nombre: primero el cambio activo de un worktree registrado
// (prevalece sobre el del checkout principal), después el activo del checkout principal, el archivado
// exacto y, por último, el archivado por prefijo o sufijo (ej. `inc-01` en `2026-09-14-inc-01-...`).
// Un fallo de Locate se propaga tal cual, con su causa y su continuación.
func (s *Service) locateIncrement(name string) (incrementLocation, error) {
	loc, err := workspace.Locate(s.getRootPath())
	if err != nil {
		return incrementLocation{}, err
	}
	root := loc.PlanningHome

	// 1. Cambio activo en un worktree registrado
	registered, _ := registeredIncrements(loc)
	if change, ok := registered[name]; ok {
		return incrementLocation{path: change.ChangeRoot, kind: "active", worktreePath: change.WorktreeRoot, branch: change.Branch}, nil
	}

	// 2. Probar en activos
	activePath := filepath.Join(root, "changes", name)
	if dirExists(activePath) {
		return incrementLocation{path: activePath, kind: "active"}, nil
	}

	// 3. Probar en archivados exacto
	archivePath := filepath.Join(root, "changes", "archive", name)
	if dirExists(archivePath) {
		return incrementLocation{path: archivePath, kind: "archived"}, nil
	}

	// 4. Probar en archivados buscando por sufijo
	archiveDir := filepath.Join(root, "changes", "archive")
	if entries, err := os.ReadDir(archiveDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && (strings.Contains(e.Name(), name) || strings.HasSuffix(e.Name(), name)) {
				return incrementLocation{path: filepath.Join(archiveDir, e.Name()), kind: "archived"}, nil
			}
		}
	}

	return incrementLocation{}, fmt.Errorf("incremento '%s' no encontrado", name)
}

// pendingArchiveNote es el aviso de los cambios pendientes de archivar (REQ-25.20): nombra cada uno con
// la orden que lo archiva. Es informativo, no un error.
func pendingArchiveNote(pending []branch.PendingChange) string {
	items := make([]string, 0, len(pending))
	for _, change := range pending {
		items = append(items, fmt.Sprintf("%s (%s)", change.Change, change.Continuation))
	}
	return fmt.Sprintf("cambios pendientes de archivar (%d): %s", len(pending), strings.Join(items, "; "))
}
