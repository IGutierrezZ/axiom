package engine

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// ODDTasksDir es la carpeta, relativa a la raíz git, donde viven los documentos vivos de las
// features ODD (`odd/tasks/<feature>.md`).
const ODDTasksDir = "odd/tasks"

// ODDDocChange es un documento ODD que difiere de la base de integración en un worktree.
type ODDDocChange struct {
	// Name es el nombre de la feature: el fichero sin la extensión `.md`.
	Name string
	// Path es la ruta absoluta del documento en el worktree.
	Path string
	// Added es cierto si el documento no existía en la base (añadido en la rama o sin seguimiento).
	Added bool
}

// ODDWorktree describe un worktree registrado visto desde el carril ODD: el trabajo ODD que
// contiene frente a la base de integración y si ya está integrado en ella.
type ODDWorktree struct {
	// Change es el nombre con el que figura en el registro local (el slug del worktree).
	Change string
	// Branch es la rama que el worktree tiene desplegada.
	Branch string
	// Root es la raíz del worktree.
	Root string
	// Base es la referencia de integración con la que se compara (`origin/main` o `main`).
	Base string
	// Docs son los documentos `odd/tasks/*.md` que ya seleccionó SelectODDDocs; vacío si el
	// worktree no aporta ninguna feature en curso.
	Docs []ODDDocChange
	// Integrated es cierto cuando la rama ya está contenida en la base, no tiene cambios sin
	// confirmar y su punta difiere de la de la base (una rama recién creada también es ancestro de
	// la base, pero su punta es la misma y no cuenta como integrada).
	Integrated bool
}

// ODDWorktrees lista, para cada worktree registrado que exista en disco, las features ODD en curso
// y si el worktree ya está integrado en la base (sin red: la base es la referencia local
// `origin/<integración>` o la rama local, como `resolveBase`). Enumera los dos carriles del
// registro (`sdd` y `odd`) porque el carril no distingue el trabajo ODD del SDD: se detecta por el
// contenido. Un mismo worktree que figure varias veces cuenta una. Nunca falla: lo que no se puede
// evaluar se omite y deja un aviso. Los worktrees ausentes del carril `sdd` ya los avisa
// RegisteredChanges, de modo que solo se avisan los del carril `odd`.
func (e *Engine) ODDWorktrees(ctx context.Context, loc workspace.Location) ([]ODDWorktree, []string) {
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

	var result []ODDWorktree
	var warnings []string
	var seen []string
	for _, lane := range []branch.Lane{branch.LaneSDD, branch.LaneODD} {
		records, err := e.registry.List(common, lane)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("no se evalúan las features ODD de los worktrees del carril %s: %v", lane, err))
			continue
		}
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
						if lane == branch.LaneODD {
							warnings = append(warnings, missingWorktreeWarning(record.Change, registered, loc.WorkspaceRoot))
						}
						continue
					}
					if slices.ContainsFunc(seen, func(root string) bool { return pathidentity.SameDirectory(root, registered.Worktree) }) {
						continue
					}
					seen = append(seen, registered.Worktree)
					state, warning, ok := e.oddWorktreeState(ctx, loc, planningRoot, record, registered)
					if warning != "" {
						warnings = append(warnings, warning)
					}
					if ok {
						result = append(result, state)
					}
				}
			}
		}
	}
	slices.SortStableFunc(result, func(a, b ODDWorktree) int { return strings.Compare(a.Change, b.Change) })
	return result, warnings
}

// oddWorktreeState evalúa un worktree. ok es falso si no se pudo evaluar (sin base o git falló).
func (e *Engine) oddWorktreeState(ctx context.Context, loc workspace.Location, planningRoot string, record branch.Record, registered branch.RecordBranch) (state ODDWorktree, warning string, ok bool) {
	root := registered.Worktree
	repo := branch.RepoPlan{
		Name: filepath.Base(planningRoot), GitRoot: planningRoot,
		IntegrationBranch: integrationBranchOf(loc, planningRoot), BaseRef: record.Exceptions.Base,
	}
	base, found, err := e.resolveBase(ctx, repo)
	if err != nil {
		return ODDWorktree{}, fmt.Sprintf("no se evalúa el worktree %s: %v", root, err), false
	}
	if !found {
		return ODDWorktree{}, missingBaseWarning(repo), false
	}
	head, found, err := e.git.ResolveCommit(ctx, root, "HEAD")
	if err != nil || !found {
		return ODDWorktree{}, fmt.Sprintf("no se evalúa el worktree %s: no se pudo resolver su HEAD", root), false
	}
	ancestor, err := e.git.IsAncestor(ctx, root, "HEAD", base.full)
	if err != nil {
		return ODDWorktree{}, fmt.Sprintf("no se evalúa el worktree %s: %v", root, err), false
	}
	dirty, err := e.git.DirtyPaths(ctx, root)
	if err != nil {
		return ODDWorktree{}, fmt.Sprintf("no se evalúa el worktree %s: %v", root, err), false
	}
	changes, err := oddDocChanges(ctx, root, base.full)
	if err != nil {
		return ODDWorktree{}, fmt.Sprintf("no se evalúa el worktree %s: %v", root, err), false
	}

	state = ODDWorktree{Change: record.Change, Branch: registered.Name, Root: root, Base: base.ref}
	state.Integrated = ancestor && len(dirty) == 0 && head != base.sha
	if !state.Integrated {
		state.Docs = SelectODDDocs(record.Change, changes)
	}
	return state, "", true
}

// oddDocChanges devuelve los documentos `odd/tasks/*.md` que difieren de la base en el worktree:
// los de `git diff --name-status <base>...HEAD` más los que tienen cambios sin confirmar o no
// tienen seguimiento. Los borrados no cuentan. Salen ordenados por nombre.
func oddDocChanges(ctx context.Context, root, baseRef string) ([]ODDDocChange, error) {
	byName := map[string]ODDDocChange{}
	record := func(relative string, added bool) {
		if path.Dir(relative) != ODDTasksDir || path.Ext(relative) != ".md" {
			return
		}
		name := strings.TrimSuffix(path.Base(relative), ".md")
		change := byName[name]
		change.Name, change.Path = name, filepath.Join(root, filepath.FromSlash(relative))
		change.Added = change.Added || added
		byName[name] = change
	}

	diffArgs := []string{"diff", "--name-status", "-z", baseRef + "...HEAD", "--", ODDTasksDir}
	diff, err := gitcommon.Run(ctx, root, diffArgs...)
	if err != nil {
		return nil, err
	}
	if diff.ExitCode != 0 {
		return nil, fmt.Errorf("git %s en %s falló: %s", strings.Join(diffArgs, " "), root, strings.TrimSpace(diff.Stderr))
	}
	fields := strings.Split(string(diff.Stdout), "\x00")
	for i := 0; i+1 < len(fields); i++ {
		status := fields[i]
		if status == "" {
			continue
		}
		switch status[0] {
		case 'R', 'C':
			if i+2 < len(fields) {
				record(fields[i+2], true)
			}
			i += 2
		case 'D':
			i++
		default:
			record(fields[i+1], status[0] == 'A')
			i++
		}
	}

	statusArgs := []string{"status", "--porcelain=v1", "-z", "--untracked-files=all", "--", ODDTasksDir}
	status, err := gitcommon.Run(ctx, root, statusArgs...)
	if err != nil {
		return nil, err
	}
	if status.ExitCode != 0 {
		return nil, fmt.Errorf("git %s en %s falló: %s", strings.Join(statusArgs, " "), root, strings.TrimSpace(status.Stderr))
	}
	records := strings.Split(string(status.Stdout), "\x00")
	for i := 0; i < len(records); i++ {
		entry := records[i]
		if len(entry) < 4 || entry[2] != ' ' {
			continue
		}
		code := entry[:2]
		if strings.ContainsAny(code, "RC") {
			i++
		}
		if strings.Contains(code, "D") {
			continue
		}
		record(entry[3:], code == "??" || strings.Contains(code, "A") || strings.ContainsAny(code, "RC"))
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	slices.Sort(names)
	changes := make([]ODDDocChange, 0, len(names))
	for _, name := range names {
		changes = append(changes, byName[name])
	}
	return changes, nil
}

// SelectODDDocs elige, entre los documentos ODD que difieren de la base, los de la feature del
// worktree: si uno se llama como el slug del registro, solo ese; si no, los añadidos en la rama o
// sin seguimiento; si no hay ninguno añadido, los modificados.
func SelectODDDocs(slug string, changes []ODDDocChange) []ODDDocChange {
	if index := slices.IndexFunc(changes, func(c ODDDocChange) bool { return c.Name == slug }); index >= 0 {
		return []ODDDocChange{changes[index]}
	}
	var added []ODDDocChange
	for _, change := range changes {
		if change.Added {
			added = append(added, change)
		}
	}
	if len(added) > 0 {
		return added
	}
	return slices.Clone(changes)
}
