package sddstatus

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// Raíces registradas y reubicación de la autoridad de edición (REQ-25.43, REQ-24.9, design D-14).
//
// `axiom branch ensure` deja en el registro local (el git common dir del planning home) los worktrees
// del cambio. Este fichero los lee, solo con ficheros y sin procesos, y de ellos salen tres hechos:
// las raíces que entran en allowedEditRoots, la base contra la que se resuelven las rutas de cada
// fuente de tareas y la reubicación de las rutas del checkout principal de un repositorio registrado
// en su worktree. Sin registro, o con uno ilegible, los tres son los de antes del incremento.

// identityRebase es la reubicación que no mueve ninguna ruta: con ella y una única fuente `tasks.md`
// los detectores de la autoridad de edición se comportan como antes del incremento (design D-14).
func identityRebase(path string) string { return path }

// registeredFacts son los hechos que el registro local aporta al status.
type registeredFacts struct {
	roots  []string
	rebase func(string) string
	plans  []planText
	notes  []string
}

// relocation reescribe lo que cuelga del checkout principal from de un repositorio registrado en el
// worktree to del cambio.
type relocation struct{ from, to string }

// resolveRegisteredFacts lee el registro del cambio y devuelve sus raíces registradas, las fuentes de
// tareas con su base y la reubicación. Nunca falla: sin git, sin registro o con un registro ilegible
// (con aviso en este último caso) devuelve la identidad y las fuentes de la consulta tal como llegan.
func resolveRegisteredFacts(q branchQuery) registeredFacts {
	facts := registeredFacts{rebase: identityRebase, plans: q.plans}
	if q.change == "" {
		return facts
	}
	planningRoot, ok := gitcommon.GitRootFromFiles(q.workspaceRoot)
	if !ok {
		return facts
	}
	commonDir, err := gitcommon.CommonDirFromFiles(planningRoot)
	if err != nil {
		return facts
	}
	record, err := registry.New().Load(commonDir, branch.LaneSDD, q.change)
	var unreadable *registry.FileError
	if errors.As(err, &unreadable) {
		facts.notes = append(facts.notes, fmt.Sprintf("note(registered_roots_unchecked): el registro local de ramas del cambio %s no se puede leer: %v; no se proyectan raíces registradas", q.change, err))
		return facts
	}
	if err != nil || record == nil {
		return facts
	}

	var moves []relocation
	roleBases := map[string]string{}
	changeBase := ""
	addRoot := func(path string) {
		if !isDirectory(path) || pathidentity.SameDirectory(path, q.workspaceRoot) {
			return
		}
		for _, existing := range facts.roots {
			if pathidentity.SameDirectory(existing, path) {
				return
			}
		}
		facts.roots = append(facts.roots, path)
	}
	for _, repo := range record.Repositories {
		planning := pathidentity.SameDirectory(repo.GitRoot, planningRoot)
		// En `in_place` el repositorio de código se trabaja en su propia raíz; el de specs ya es el workspace.
		if record.Mode == branch.ModeInPlace && repo.Role != branch.RepoSpecsInPlace {
			addRoot(repo.GitRoot)
		}
		changeWorktree := ""
		for _, registered := range repo.Branches {
			if registered.Worktree == "" || !isDirectory(registered.Worktree) {
				continue
			}
			addRoot(registered.Worktree)
			switch registered.Kind {
			case branch.KindChange:
				changeWorktree = registered.Worktree
			case branch.KindRole:
				if planning {
					roleBases[registered.RoleID] = registered.Worktree
				}
			}
		}
		if changeWorktree != "" {
			moves = append(moves, relocation{from: repo.GitRoot, to: changeWorktree})
			if planning {
				changeBase = changeWorktree
			}
		}
	}
	if len(moves) != 0 {
		facts.rebase = func(path string) string { return relocate(path, moves) }
	}
	facts.plans = plansWithBases(q, changeBase, roleBases)
	return facts
}

// plansWithBases da a cada fuente de tareas la base contra la que se resuelven sus rutas relativas
// (design D-14): `tasks.<rol>.md`, el worktree de su rama de rol si existe; cualquier otra fuente, o
// un rol sin worktree propio, el worktree del cambio; sin él, la base queda vacía (el workspace).
func plansWithBases(q branchQuery, changeBase string, roleBases map[string]string) []planText {
	roleOf := map[string]string{}
	if sealed, err := kickoff.Load(q.changeRoot); err == nil && sealed != nil {
		for _, role := range sealed.Config.Roles {
			roleOf[role.TasksFile] = role.Role
		}
	}
	plans := make([]planText, 0, len(q.plans))
	for _, plan := range q.plans {
		plan.base = changeBase
		if role := roleOf[plan.name]; role != "" {
			if worktree, ok := roleBases[role]; ok {
				plan.base = worktree
			}
		}
		plans = append(plans, plan)
	}
	return plans
}

// relocate reescribe path con el mismo sufijo en el worktree del cambio cuando cae dentro del
// checkout principal de un repositorio registrado. Una ruta que ya está en un worktree registrado, o
// en un repositorio que no lo está, queda igual. Comparar por identidad (no por texto) tolera las
// grafías distintas de una misma carpeta (nombres cortos de Windows, enlaces simbólicos); la ruta
// puede no existir todavía: se pregunta por su ancestro más profundo que existe.
func relocate(path string, moves []relocation) string {
	for _, move := range moves {
		if pathidentity.Contains(move.to, path) {
			return path
		}
	}
	var suffix []string
	for current := path; ; {
		for _, move := range moves {
			if pathidentity.SameDirectory(current, move.from) {
				return filepath.Join(append([]string{move.to}, suffix...)...)
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return path
		}
		suffix = append([]string{filepath.Base(current)}, suffix...)
		current = parent
	}
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
