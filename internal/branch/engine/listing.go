package engine

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RegisteredChange es un cambio SDD que vive en un worktree registrado del planning home (REQ-25.44,
// design D-16): la rama del cambio o la de archivado desplegada en un worktree por `axiom branch ensure`.
type RegisteredChange struct {
	// Name es el nombre del cambio.
	Name string
	// Branch es la rama que el worktree tiene desplegada.
	Branch string
	// Kind es `change` o `archive`: las ramas de rol no son un cambio aparte, son parte del trabajo de este.
	Kind branch.BranchKind
	// WorktreeRoot es la raíz del worktree.
	WorktreeRoot string
	// PlanningHome es el planning home dentro del worktree: el mismo camino relativo a la raíz git que el
	// del proyecto desde el que se consulta.
	PlanningHome string
	// ChangeRoot es `<PlanningHome>/changes/<cambio>`, o vacío si esa carpeta no existe en el worktree.
	ChangeRoot string
}

// RegisteredChanges lista los cambios SDD que viven en worktrees registrados del planning home de loc
// (REQ-25.44, design D-16). Lee el registro local del git common dir del repositorio del planning home y
// solo ficheros: nunca lanza procesos de git. Por cada rama de cambio o de archivado con worktree devuelve
// una entrada; una rama de rol, una rama sin worktree (también en `in_place`) o un repositorio que no es el
// del planning home no cuentan. El resultado sale por nombre y, dentro de cada cambio, con la rama de
// archivado primero: es el orden de precedencia (worktree de archivado, después el del cambio).
//
// Una entrada cuyo worktree ya no existe se omite y deja un aviso en castellano que nombra su ruta y la
// continuación. Nunca falla: sin repositorio git, sin registro o sin planning home dentro del repositorio
// no hay nada que listar; un registro ilegible no se lista y deja un aviso con la continuación de
// reconstruirlo.
func (e *Engine) RegisteredChanges(ctx context.Context, loc workspace.Location) ([]RegisteredChange, []string) {
	if ctx.Err() != nil {
		return nil, nil
	}
	planningRoot, ok := gitcommon.GitRootFromFiles(loc.WorkspaceRoot)
	if !ok {
		return nil, nil
	}
	common, err := gitcommon.CommonDirFromFiles(planningRoot)
	if err != nil {
		return nil, nil
	}
	relative, inside := relativeWithin(planningRoot, loc.PlanningHome)
	if !inside {
		return nil, nil
	}
	records, err := e.registry.List(common, branch.LaneSDD)
	if err != nil {
		return nil, []string{fmt.Sprintf(
			"no se evalúan los cambios de los worktrees registrados: %v; borra ese fichero (es estado local derivado) y reconstrúyelo con `axiom branch ensure --change <cambio> --cwd %s`",
			err, pathquote.Quote(loc.WorkspaceRoot))}
	}

	var changes []RegisteredChange
	var warnings []string
	for _, record := range records {
		for _, repo := range record.Repositories {
			if !pathidentity.SameDirectory(repo.CommonDir, common) {
				continue
			}
			for _, registered := range repo.Branches {
				if registered.Worktree == "" || (registered.Kind != branch.KindChange && registered.Kind != branch.KindArchive) {
					continue
				}
				if info, err := os.Stat(registered.Worktree); err != nil || !info.IsDir() {
					warnings = append(warnings, missingWorktreeWarning(record.Change, registered, loc.WorkspaceRoot))
					continue
				}
				home := filepath.Join(registered.Worktree, filepath.FromSlash(relative))
				change := RegisteredChange{
					Name: record.Change, Branch: registered.Name, Kind: registered.Kind,
					WorktreeRoot: registered.Worktree, PlanningHome: home,
				}
				if root := filepath.Join(home, "changes", record.Change); isDirectory(root) {
					change.ChangeRoot = root
				}
				changes = append(changes, change)
			}
		}
	}
	slices.SortStableFunc(changes, func(a, b RegisteredChange) int {
		if byName := strings.Compare(a.Name, b.Name); byName != 0 {
			return byName
		}
		return cmp.Compare(kindPrecedence(a.Kind), kindPrecedence(b.Kind))
	})
	return changes, warnings
}

// RegisteredChangesByName devuelve, por nombre, el cambio de worktree registrado que prevalece
// (REQ-25.34, design D-16): el del worktree de archivado antes que el del worktree del cambio. Solo
// cuentan las entradas cuyo worktree tiene la carpeta del cambio. Es la agregación que comparten el
// dashboard y la TUI, de modo que listan el mismo conjunto; los avisos son los de RegisteredChanges.
func (e *Engine) RegisteredChangesByName(ctx context.Context, loc workspace.Location) (map[string]RegisteredChange, []string) {
	changes, warnings := e.RegisteredChanges(ctx, loc)
	byName := make(map[string]RegisteredChange, len(changes))
	for _, change := range changes {
		if change.ChangeRoot == "" {
			continue
		}
		// RegisteredChanges sale por nombre y, dentro de cada uno, en orden de precedencia: el
		// primero que se ve de un nombre es el que prevalece.
		if _, taken := byName[change.Name]; !taken {
			byName[change.Name] = change
		}
	}
	return byName, warnings
}

// kindPrecedence ordena las ramas de un mismo cambio: la de archivado, que ya está fusionada y solo falta
// archivarla, antes que la del cambio (REQ-25.44, REQ-25.45).
func kindPrecedence(kind branch.BranchKind) int {
	if kind == branch.KindArchive {
		return 0
	}
	return 1
}

// missingWorktreeWarning es el aviso de una entrada del registro cuyo worktree ya no existe: nombra la
// ruta ausente y la orden que lo restaura.
func missingWorktreeWarning(change string, registered branch.RecordBranch, workspaceRoot string) string {
	restore := fmt.Sprintf("axiom branch ensure --change %s --cwd %s", pathquote.ShellWord(change), pathquote.Quote(workspaceRoot))
	if registered.Kind == branch.KindArchive {
		restore += " --archive"
	}
	return fmt.Sprintf(
		"el worktree %s de la rama %s del cambio %s figura en el registro local pero ya no existe o no es accesible; se omite (restáuralo con `%s`)",
		registered.Worktree, registered.Name, change, restore)
}

func isDirectory(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}
