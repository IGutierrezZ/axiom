package engine

import (
	"fmt"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Preferencias personales de worktree y modo de trabajo efectivo de `ensure` (REQ-25.8,
// REQ-25.29, design D-08). Las preferencias salen del hub, nunca de `axiom.yaml`; el modo
// efectivo de un cambio se resuelve con la prioridad registro > `--mode` > hub > defecto.

// hubPrefs implementa el puerto branch.Prefs sobre `~/.axiom/workspaces.json`.
type hubPrefs struct{}

var _ branch.Prefs = hubPrefs{}

// WorktreePrefs devuelve el modo de trabajo de las preferencias del proyecto cuya carpeta
// maestra es projectMaster, la raíz de worktrees, de dónde sale el modo (`hub` si el hub lo
// guarda, `default` si no) y un aviso. Un hub que no se puede leer, o con un valor guardado
// inválido, no impide trabajar: devuelve los valores por defecto (modo `worktree` y la raíz
// de por defecto) con un aviso que lo explica (design D-08).
func (hubPrefs) WorktreePrefs(projectMaster string) (branch.Mode, string, string, string) {
	manager, err := hub.NewManager("")
	if err != nil {
		return defaultPrefs(projectMaster, err)
	}
	prefs, stored, err := manager.LookupWorktreePrefs(projectMaster)
	if err != nil {
		return defaultPrefs(projectMaster, err)
	}
	source := branch.ModeSourceDefault
	if stored {
		source = branch.ModeSourceHub
	}
	return branch.Mode(prefs.Mode), prefs.Root, source, ""
}

// defaultPrefs son los valores por defecto cuando el hub no se pudo leer: la causa va en un
// aviso.
func defaultPrefs(projectMaster string, cause error) (branch.Mode, string, string, string) {
	warning := fmt.Sprintf("no se pudieron leer las preferencias personales del hub (%v): se usan los valores por defecto (modo %s y la raíz de worktrees por defecto)", cause, branch.ModeWorktree)
	return branch.ModeWorktree, branch.DefaultWorktreeRoot(projectMaster), branch.ModeSourceDefault, warning
}

// workPrefs son las preferencias de worktree del proyecto que lee `gather`: la raíz de
// worktrees entra en el plan y el modo, con su origen, en la resolución de `ensure`.
type workPrefs struct {
	mode    branch.Mode
	root    string
	source  string
	warning string
}

// workPrefs lee las preferencias del proyecto. La identidad del proyecto es la carpeta maestra
// del checkout principal del repositorio del planning home, nunca la de un worktree enlazado
// (design D-08); sin puerto (las pruebas que construyen el motor a mano) rigen los valores
// por defecto sin leer el hub.
func (e *Engine) workPrefs(location workspace.Location, planning branch.RepoFacts) workPrefs {
	master := resolveDirectory(projectMaster(location, planning))
	if e.prefs == nil {
		return workPrefs{mode: branch.ModeWorktree, root: branch.DefaultWorktreeRoot(master), source: branch.ModeSourceDefault}
	}
	mode, root, source, warning := e.prefs.WorktreePrefs(master)
	return workPrefs{mode: mode, root: root, source: source, warning: warning}
}

// projectMaster es la carpeta maestra del proyecto vista desde su checkout principal: la
// que resolvió `Locate` si cwd ya está en él, y, si estaba en un worktree enlazado, la misma
// ruta relativa dentro del checkout principal (`gitcommon.MainWorktreeFromFiles` sobre el git
// common dir). Si el checkout principal no se puede deducir por ficheros, o la maestra queda
// fuera del repositorio del planning home (multirepo), es la que resolvió `Locate`.
func projectMaster(location workspace.Location, planning branch.RepoFacts) string {
	main, known := gitcommon.MainWorktreeFromFiles(planning.CommonDir)
	if !known || pathidentity.SameDirectory(main, planning.GitRoot) {
		return location.Master
	}
	relative, inside := relativeWithin(planning.GitRoot, location.Master)
	if !inside {
		return location.Master
	}
	return filepath.Join(main, filepath.FromSlash(relative))
}

// workMode es el modo de trabajo efectivo de un `ensure` y de dónde sale (branch.ModeSource*).
type workMode struct {
	mode   branch.Mode
	source string
}

// resolveMode resuelve el modo efectivo del cambio con la prioridad registro > `--mode` > hub
// > defecto (design D-08):
//
//   - un cambio con registro conserva su modo y su origen, y un `--mode` distinto es
//     `plan_conflict` (uno igual es idempotente);
//   - sin registro, `--mode` fija el modo: su origen es `answer` si el hub pregunta (es la
//     respuesta a `ask`) y `flag` si no;
//   - sin `--mode`, rige el hub (`hub` si guarda un modo, `default` si no) y, si es `ask`,
//     `decision_required` sin crear nada ni leer la entrada estándar.
//
// El motor nunca escribe la preferencia del hub: la respuesta solo se registra por cambio.
func resolveMode(operation string, r EnsureRequest, facts changeFacts) (workMode, *branch.Refusal) {
	if previous := facts.record; previous != nil {
		if r.Mode != "" && r.Mode != previous.Mode {
			return workMode{}, branch.ModeConflict(operation, r.Change, r.CWD, r.Lane, previous.Mode, r.Mode)
		}
		return workMode{mode: previous.Mode, source: previous.ModeSource}, nil
	}
	prefs := facts.prefs
	if r.Mode != "" {
		source := branch.ModeSourceFlag
		if prefs.mode == branch.ModeAsk {
			source = branch.ModeSourceAnswer
		}
		return workMode{mode: r.Mode, source: source}, nil
	}
	if prefs.mode == branch.ModeAsk {
		return workMode{}, branch.DecisionRequired(operation, r.Change, r.CWD, r.Lane)
	}
	return workMode{mode: prefs.mode, source: prefs.source}, nil
}
