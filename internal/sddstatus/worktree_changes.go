package sddstatus

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Cambios que viven en worktrees registrados (REQ-25.44, design D-16).
//
// `axiom branch ensure` deja registrados los worktrees de la rama del cambio y de la de archivado. Desde el
// checkout principal, u otro directorio del proyecto, el cambio no está en `openspec/changes`: Resolve lo
// busca antes de elegir cambio y, si gana un worktree, reenraíza el status en él. La precedencia es la de
// D-16: el worktree de archivado, después el del cambio y, por último, el checkout principal. Todo el
// listado sale de ficheros del registro local; sin registro, o sin worktrees con la carpeta del cambio, el
// status es el de antes del incremento.

// registeredWorktreesNote es el prefijo estable de los avisos de este fichero.
const registeredWorktreesNote = "note(registered_worktrees): "

// rerootOnRegisteredWorktrees decide, antes de resolveByPreferenceOrder, en qué árbol se resuelve el cambio
// y devuelve las opciones, la ubicación y los avisos que usa el resto de Resolve.
//
// Con nombre, gana el worktree registrado que tiene la carpeta del cambio (archivo antes que cambio) y el
// status se reenraíza en él. Sin nombre, los candidatos son la unión de los del checkout actual y los de los
// worktrees: con uno solo, se selecciona (y, si vive en un worktree, se reenraíza); con varios, el
// `select-change` de siempre actúa sobre la unión. Con `--cwd` dentro de un worktree enlazado el cambio se
// resuelve en su propio árbol (REQ-25.28) y no se mira el registro. Nunca falla: lo que no se puede leer deja
// un aviso y el status se resuelve donde estaba.
func rerootOnRegisteredWorktrees(options ResolveOptions, location workspace.Location) (ResolveOptions, workspace.Location) {
	if gitcommon.LinkedWorktreeRoot(location.WorkspaceRoot) != "" {
		return options, location
	}
	changes, warnings := engine.Default().RegisteredChanges(context.Background(), location)
	for _, warning := range warnings {
		location.Notes = appendNote(location.Notes, registeredWorktreesNote+warning)
	}
	// Solo cuenta un worktree que tiene la carpeta del cambio: sin ella no hay nada que resolver en él.
	present := slices.DeleteFunc(slices.Clone(changes), func(change engine.RegisteredChange) bool {
		return change.ChangeRoot == ""
	})
	if len(present) == 0 {
		return options, location
	}

	name := strings.TrimSpace(options.ChangeName)
	if name == "" {
		// Sin nombre, un contenedor de cambios o una carpeta solo de exploración no es candidato (#2317, #3278).
		selectable := slices.DeleteFunc(slices.Clone(present), func(change engine.RegisteredChange) bool {
			return !activeChangeDir(change.ChangeRoot)
		})
		candidates, ok := unionCandidates(location.WorkspaceRoot, selectable)
		if !ok || len(selectable) == 0 {
			return options, location
		}
		if len(candidates) != 1 {
			options.worktreeCandidates = namesOf(selectable)
			return options, location
		}
		name = candidates[0]
	}

	winner, found := winningWorktree(present, name)
	if !found {
		return options, location
	}
	root := filepath.Dir(winner.PlanningHome)
	relocated, err := workspace.Locate(root)
	if err != nil || !pathidentity.SameDirectory(relocated.WorkspaceRoot, root) {
		location.Notes = appendNote(location.Notes, fmt.Sprintf(
			"%sel cambio %s está en el worktree %s de la rama %s, pero no se pudo resolver su workspace; se resuelve en %s",
			registeredWorktreesNote, name, winner.WorktreeRoot, winner.Branch, location.WorkspaceRoot))
		return options, location
	}
	for _, note := range location.Notes {
		relocated.Notes = appendNote(relocated.Notes, note)
	}
	relocated.Notes = appendNote(relocated.Notes, fmt.Sprintf(
		"%sel cambio %s se resuelve en el worktree %s de la rama %s (REQ-25.44), no en %s",
		registeredWorktreesNote, name, winner.WorktreeRoot, winner.Branch, location.WorkspaceRoot))
	options.ChangeName = name
	options.WorkspaceRoot = relocated.WorkspaceRoot
	return options, relocated
}

// winningWorktree es la entrada que prevalece para el cambio dado: la primera de present con su nombre,
// que el listado ordena con la rama de archivado delante de la del cambio.
func winningWorktree(present []engine.RegisteredChange, name string) (engine.RegisteredChange, bool) {
	index := slices.IndexFunc(present, func(change engine.RegisteredChange) bool { return change.Name == name })
	if index < 0 {
		return engine.RegisteredChange{}, false
	}
	return present[index], true
}

// unionCandidates son los cambios seleccionables sin nombre: los del checkout actual y los de los
// worktrees registrados, sin repetidos y por orden alfabético. false si el checkout actual no se puede leer.
func unionCandidates(workspaceRoot string, present []engine.RegisteredChange) ([]string, bool) {
	active, err := listActiveOpenSpecChanges(workspaceRoot)
	if err != nil {
		return nil, false
	}
	candidates := selectableOpenSpecChanges(filepath.Join(workspaceRoot, "openspec", "changes"), active)
	return unionNames(candidates, namesOf(present)), true
}

// namesOf son los nombres de los cambios de present, sin repetidos y por orden alfabético.
func namesOf(present []engine.RegisteredChange) []string {
	names := make([]string, 0, len(present))
	for _, change := range present {
		names = append(names, change.Name)
	}
	return unionNames(names, nil)
}

// unionNames une dos listas de nombres, sin repetidos y por orden alfabético.
func unionNames(first, second []string) []string {
	union := slices.Concat(first, second)
	slices.Sort(union)
	return slices.Compact(union)
}

// activeChangeDir dice si la carpeta de un cambio registrado cuenta como cambio activo seleccionable: no es
// un contenedor de cambios ni una carpeta solo de exploración (#2317, #3278).
func activeChangeDir(changeRoot string) bool {
	if openSpecChangeContainer(changeRoot) {
		return false
	}
	return len(selectableOpenSpecChanges(filepath.Dir(changeRoot), []string{filepath.Base(changeRoot)})) == 1
}

func appendNote(notes []string, note string) []string {
	if slices.Contains(notes, note) {
		return notes
	}
	return append(notes, note)
}
