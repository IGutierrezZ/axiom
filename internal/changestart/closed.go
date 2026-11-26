package changestart

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/registry"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// localState es la vista local, solo de ficheros, de los cambios del registro de ramas (D-11). Se
// carga una vez por operación. Un registro ilegible no bloquea: queda en warning y el escaneo local
// se da por no realizado.
type localState struct {
	env     Env
	records []branch.Record
	warning string
}

func newLocalState(env Env) *localState {
	l := &localState{env: env}
	store := registry.New()
	for _, lane := range []branch.Lane{branch.LaneSDD, branch.LaneODD} {
		recs, err := store.List(env.CommonDir, lane)
		if err != nil {
			l.warning = "no se pudo leer el registro local de ramas: el escaneo de vínculos locales no fue posible"
			continue
		}
		l.records = append(l.records, recs...)
	}
	return l
}

// planningRel y workspaceRel son las rutas del planning home y del workspace relativas a la raíz git.
func (e Env) planningRel() string  { return relOrDot(e.GitRoot, e.PlanningHome) }
func (e Env) workspaceRel() string { return relOrDot(e.GitRoot, e.WorkspaceRoot) }

func relOrDot(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "."
	}
	return rel
}

// recordsOf devuelve los registros del cambio, en cualquier carril, sin distinguir mayúsculas.
func (l *localState) recordsOf(change string) []branch.Record {
	var out []branch.Record
	for _, r := range l.records {
		if strings.EqualFold(r.Change, change) {
			out = append(out, r)
		}
	}
	return out
}

// planningRepo devuelve el repositorio de especificaciones de un registro.
func planningRepo(r branch.Record) (branch.RecordRepo, bool) {
	for _, repo := range r.Repositories {
		if pathidentity.SameDirectory(repo.GitRoot, r.PlanningRepo) {
			return repo, true
		}
	}
	return branch.RecordRepo{}, false
}

// checkouts devuelve las raíces de checkout donde vive la rama del cambio del repositorio de
// especificaciones: su worktree si existe o el propio repositorio si está desplegada en sitio.
func checkouts(r branch.Record) []string {
	repo, ok := planningRepo(r)
	if !ok {
		return nil
	}
	var out []string
	for _, b := range repo.Branches {
		if b.Kind != branch.KindChange {
			continue
		}
		switch {
		case b.Worktree != "" && dirExists(b.Worktree):
			out = append(out, b.Worktree)
		case b.InPlace && dirExists(repo.GitRoot):
			out = append(out, repo.GitRoot)
		}
	}
	return out
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// closedLocally dice si el cambio other está cerrado según D-11. other ya superó ValidChangeName.
//   - SDD archivado: existe changes/archive/AAAA-MM-DD-<other>/ en el workspace de --cwd, en el
//     worktree principal o en el worktree de la rama archive del registro. Manda aunque el registro
//     siga presente.
//   - ODD terminado: no hay registro de other en ningún carril y existe odd/tasks/<other>.md en el
//     worktree principal o en el workspace de --cwd.
//
// En cualquier otro caso other cuenta como abierto.
func (l *localState) closedLocally(other string) bool {
	mainRoot, hasMain := gitcommon.MainWorktreeFromFiles(l.env.CommonDir)
	roots := []string{l.env.PlanningHome}
	if hasMain {
		roots = append(roots, filepath.Join(mainRoot, l.env.planningRel()))
	}
	recs := l.recordsOf(other)
	for _, r := range recs {
		if repo, ok := planningRepo(r); ok {
			for _, b := range repo.Branches {
				if b.Kind == branch.KindArchive && b.Worktree != "" {
					roots = append(roots, filepath.Join(b.Worktree, l.env.planningRel()))
				}
			}
		}
	}
	for _, root := range roots {
		if archivedUnder(root, other) {
			return true
		}
	}
	if len(recs) > 0 {
		return false
	}
	docs := []string{filepath.Join(l.env.WorkspaceRoot, "odd", "tasks", other+".md")}
	if hasMain {
		docs = append(docs, filepath.Join(mainRoot, l.env.workspaceRel(), "odd", "tasks", other+".md"))
	}
	for _, doc := range docs {
		if info, err := os.Stat(doc); err == nil && info.Mode().IsRegular() {
			return true
		}
	}
	return false
}

// archivedUnder busca <planning>/changes/archive/AAAA-MM-DD-<other> con la fecha y el nombre exactos.
func archivedUnder(planning, other string) bool {
	entries, err := os.ReadDir(filepath.Join(planning, "changes", "archive"))
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() && len(name) == 11+len(other) && name[11:] == other && datePrefix(name[:11]) {
			return true
		}
	}
	return false
}

// datePrefix comprueba el prefijo AAAA-MM-DD- (con el guion final).
func datePrefix(s string) bool {
	if len(s) != 11 || s[10] != '-' {
		return false
	}
	for i := 0; i < 10; i++ {
		switch i {
		case 4, 7:
			if s[i] != '-' {
				return false
			}
		default:
			if s[i] < '0' || s[i] > '9' {
				return false
			}
		}
	}
	return true
}
