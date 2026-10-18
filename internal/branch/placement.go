package branch

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Colocación de los worktrees (REQ-25.11, design D-04): el planificador comprueba,
// antes de crear nada, que el nombre `<repo>` de cada worktree no se repite, que la
// raíz de worktrees y las rutas planificadas no se solapan con árboles de trabajo
// existentes y que ninguna ruta supera el límite de la plataforma. Son comparaciones
// de texto sobre rutas ya normalizadas: el planificador no lee el disco, y es el motor
// quien las entrega con la grafía que usa git.

// windowsPathLimit es la longitud máxima de la ruta de un worktree en Windows sin rutas
// largas habilitadas: MAX_PATH (260) menos un margen de 40 caracteres para los ficheros
// que contiene (A-1).
const windowsPathLimit = 260 - 40

// PathRules son las reglas de rutas de la plataforma que entran al planificador, de modo
// que la regla de Windows se prueba en cualquier sistema (design D-04).
type PathRules struct {
	// GOOS es el sistema operativo de las rutas del plan (`windows`, `linux`, `darwin`).
	// En Windows rige el límite de longitud y las rutas se comparan sin distinguir
	// mayúsculas; en los demás sistemas no hay límite propio.
	GOOS string
}

func (r PathRules) windows() bool { return r.GOOS == "windows" }

// pathLength mide una ruta como lo hace MAX_PATH: en unidades UTF-16.
func pathLength(path string) int {
	return len(utf16.Encode([]rune(path)))
}

// samePath dice si dos rutas son la misma, por texto: limpias y, si foldCase, sin
// distinguir mayúsculas.
func samePath(a, b string, foldCase bool) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if foldCase {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// within dice si child es parent o queda dentro de él, por segmentos de ruta: `axiom-wt`
// no está dentro de `axiom`.
func within(parent, child string, foldCase bool) bool {
	parent, child = filepath.Clean(parent), filepath.Clean(child)
	if foldCase {
		parent, child = strings.ToLower(parent), strings.ToLower(child)
	}
	if child == parent {
		return true
	}
	prefix := parent
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(child, prefix)
}

// checkRepositoryNames rechaza con `invalid_name` dos repositorios del plan que darían
// el mismo `<repo>` a la ruta de su worktree (REQ-25.11, S-17), nombrando ambas rutas.
// El repositorio de specs de un multirepo no tiene worktree y no entra en la
// comprobación: su nombre no compone ninguna ruta.
func checkRepositoryNames(in PlanInput, selections []repoSelection) *Refusal {
	owners := map[string]RepoFacts{}
	for _, selection := range selections {
		if repositoryRole(in.Topology, selection.planning) == RepoSpecsInPlace {
			continue
		}
		name := repoName(selection.facts)
		key := name
		if in.PathRules.windows() {
			key = strings.ToLower(name)
		}
		first, taken := owners[key]
		if !taken {
			owners[key] = selection.facts
			continue
		}
		second := selection.facts
		return &Refusal{
			Code: CodeInvalidName,
			Message: fmt.Sprintf("los repositorios «%s» (%s) y «%s» (%s) comparten el nombre «%s», con el que se compone la ruta de sus worktrees",
				declaredLabel(first), first.GitRoot, declaredLabel(second), second.GitRoot, name),
			Continuation: fmt.Sprintf("declara en axiom.yaml rutas de repositorio con distinto último segmento (roles.<rol>.repositories[].path) y comprueba con `axiom branch %s --change %s --cwd %s`",
				in.Operation, in.Change, pathquote.Quote(in.Master)),
			Details: map[string]string{"field": "repository", "value": name, "paths": first.GitRoot + ", " + second.GitRoot},
		}
	}
	return nil
}

// declaredLabel es la ruta de un repositorio tal como la declara axiom.yaml, o `.`.
func declaredLabel(facts RepoFacts) string {
	if declared := strings.TrimSpace(facts.DeclaredPath); declared != "" {
		return declared
	}
	return "."
}

// placedWorktree es el worktree de una rama del plan, con el repositorio al que
// pertenece.
type placedWorktree struct {
	repo   RepoPlan
	branch BranchPlan
}

// placedWorktrees son los worktrees que el plan crearía, en el orden del plan.
func placedWorktrees(repos []RepoPlan) []placedWorktree {
	var placed []placedWorktree
	for _, repo := range repos {
		for _, branch := range repo.Branches {
			if branch.Worktree != "" {
				placed = append(placed, placedWorktree{repo: repo, branch: branch})
			}
		}
	}
	return placed
}

// checkPlacement rechaza con `plan_conflict` un plan cuyos worktrees se solaparían con
// otro árbol de trabajo (design D-04): la raíz de worktrees dentro de un repositorio del
// plan, o una ruta planificada dentro de un worktree existente que no es el suyo (por
// ejemplo, uno antiguo de los scripts locales). El worktree que ya está exactamente en
// la ruta planificada es el suyo y `ensure` lo reutiliza. Sin ningún worktree que
// colocar no hay nada que comprobar.
func checkPlacement(in PlanInput, repos []RepoPlan) *Refusal {
	placed := placedWorktrees(repos)
	if len(placed) == 0 {
		return nil
	}
	fold := in.PathRules.windows()
	recheck := fmt.Sprintf("axiom branch %s --change %s --cwd %s", in.Operation, in.Change, pathquote.Quote(in.Master))

	for _, repo := range repos {
		if within(repo.GitRoot, in.WorktreeRoot, fold) {
			return &Refusal{
				Code: CodePlanConflict,
				Message: fmt.Sprintf("la raíz de worktrees %s queda dentro del repositorio %s (%s), que entra al plan: los worktrees quedarían dentro de otro árbol de trabajo",
					in.WorktreeRoot, repo.Name, repo.GitRoot),
				Continuation: fmt.Sprintf("fija una raíz de worktrees fuera de ese repositorio en las preferencias personales del hub, o mueve el proyecto a la raíz de su repositorio, y comprueba con `%s`", recheck),
				Details:      map[string]string{"worktreeRoot": in.WorktreeRoot, "repository": repo.GitRoot},
			}
		}
	}
	for _, worktree := range placed {
		for _, existing := range in.ExistingWorktrees {
			planned := worktree.branch.Worktree
			if samePath(existing.Path, planned, fold) || !within(existing.Path, planned, fold) {
				continue
			}
			return &Refusal{
				Code: CodePlanConflict,
				Message: fmt.Sprintf("la ruta %s planificada para la rama %s queda dentro del worktree existente %s, que no es el suyo",
					planned, worktree.branch.Name, existing.Path),
				Continuation: fmt.Sprintf("retira ese worktree con `git -C %s worktree remove %s` o elige otro nombre de cambio con `axiom branch %s --change <cambio> --cwd %s`",
					pathquote.Quote(worktree.repo.GitRoot), pathquote.Quote(existing.Path), in.Operation, pathquote.Quote(in.Master)),
				Details: map[string]string{"worktree": planned, "existing": existing.Path},
			}
		}
	}
	return nil
}

// checkPathLength rechaza con `path_too_long` la primera ruta de worktree que supera el
// límite de Windows (A-1), nombrando la ruta, su longitud y el máximo. En los demás
// sistemas no hay límite propio.
func checkPathLength(in PlanInput, repos []RepoPlan) *Refusal {
	if !in.PathRules.windows() {
		return nil
	}
	for _, worktree := range placedWorktrees(repos) {
		path := worktree.branch.Worktree
		length := pathLength(path)
		if length <= windowsPathLimit {
			continue
		}
		return &Refusal{
			Code: CodePathTooLong,
			Message: fmt.Sprintf("la ruta del worktree %s tiene %d caracteres y el máximo admitido en Windows es %d (260 menos un margen de 40 para los ficheros que contiene)",
				path, length, windowsPathLimit),
			Continuation: fmt.Sprintf("usa un nombre de cambio más corto con `axiom branch %s --change <cambio> --cwd %s` o fija una raíz de worktrees más corta en las preferencias personales del hub",
				in.Operation, pathquote.Quote(in.Master)),
			Details: map[string]string{"path": path, "length": fmt.Sprint(length), "max": fmt.Sprint(windowsPathLimit)},
		}
	}
	return nil
}
