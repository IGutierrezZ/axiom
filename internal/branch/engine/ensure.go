package engine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// EnsureRequest es la petición de `ensure`. CWD debe ser un directorio absoluto. Fetch pide
// `git fetch origin` en cada repositorio del plan antes de resolver la base (`--fetch`,
// REQ-25.10): sin él, `ensure` no hace ninguna operación remota. Mode es `--mode`: la
// respuesta a `ask` o un modo explícito (`worktree` o `in_place`); vacío si no se indicó
// (REQ-25.8). Exceptions son `--prefix`, `--branch` y `--base`, que se registran por cambio
// (REQ-25.12); Repos, los `--repo` (REQ-25.12, REQ-25.13): en el carril ODD incluyen
// repositorios y en el SDD acotan los del plan.
type EnsureRequest struct {
	CWD        string
	Change     string
	Lane       branch.Lane
	Fetch      bool
	Mode       branch.Mode
	Exceptions branch.Exceptions
	Repos      []string
}

// Ensure crea lo que falta de las ramas del cambio y las despliega según el modo de trabajo
// (REQ-25.7, REQ-25.8, design D-05): en `worktree`, un worktree por rama; en `in_place`, la
// rama de seguimiento en el propio repositorio con `git switch`. La base se resuelve sin red
// (D-06):
//
//  1. bloquea el cambio en exclusiva (un bloqueo por cambio en el git common dir: con otra
//     ejecución en marcha, `plan_conflict`), reúne los hechos de solo lectura (gather),
//     resuelve el modo de trabajo (registro > `--mode` > hub > defecto, D-08: con `ask` y sin
//     respuesta, `decision_required` antes de cualquier otra cosa, también antes del fetch) y
//     planifica (branch.Plan);
//  2. comprueba, sin efectos, todos los repositorios del plan antes de mutar ninguno
//     (checkPlan): con `--fetch`, el `git fetch origin` de cada repositorio, antes de
//     resolver la base; que la base existe, que ninguna rama está desplegada en un
//     worktree desconocido y que las rutas planificadas están libres; en `in_place`, que
//     el repositorio tiene el árbol limpio si va a cambiar de rama (`dirty_worktree`, sin
//     modificar nada: ya en la rama esperada es un no-op, REQ-25.9);
//  3. aplica solo lo que falta (apply): crea las ramas que no existen, desde la base o,
//     las de rol, desde la de seguimiento, y los worktrees que no existen (en `in_place`,
//     el `git switch` a la rama de seguimiento), siempre con cmd.Dir en el checkout
//     principal. Nunca mueve ni reinicia una rama existente; en `worktree` tampoco toca el
//     checkout principal;
//  4. guarda el registro local, compuesto desde el plan y el estado final de git: solo se
//     escribe si cambian sus bytes, y un registro borrado se recompone idéntico;
//  5. describe el estado final, sin acciones ni marcas de tiempo, de modo que repetir la
//     orden con el mismo estado da el mismo resultado byte a byte.
//
// Un fallo de git durante la aplicación es `git_failed` y repetir `ensure` completa lo
// pendiente; un fallo de la comprobación (incluido `fetch_failed`) no deja ni ramas, ni
// worktrees, ni registro. La única operación remota es `git fetch origin` con `--fetch`.
//
// Las excepciones por cambio (`--prefix`, `--branch`, `--base`, REQ-25.12) se validan antes de
// bloquear nada (`invalid_name`), se contrastan con las del registro (una contradicción de lo
// materializado es `plan_conflict`, la repetición idéntica es idempotente) y se registran. En un
// multirepo, el repositorio de specs se trabaja in situ: nunca recibe ramas, worktrees ni
// checkout, y fuera de su integración se rechaza (`specs_repo_not_on_integration`, REQ-25.5). El
// carril ODD (REQ-25.13) no tiene roles ni kickoff y `--repo` incluye repositorios de código.
func (e *Engine) Ensure(ctx context.Context, r EnsureRequest) branch.Result {
	const operation = "ensure"
	refused := func(refusal *branch.Refusal) branch.Result {
		if refusal.Code == branch.CodeGitFailed {
			// Un fallo de git no deja un estado inconsistente: repetir `ensure` completa
			// lo que falte, de modo que la continuación es la orden con el cambio.
			refusal.Continuation = fmt.Sprintf("comprueba que git está instalado y que puedes escribir en el repositorio, y repite `axiom branch %s --change %s --cwd %s`: completa lo que falte", operation, r.Change, pathquote.Quote(r.CWD))
		}
		return branch.NewResult(operation).Refused(refusal)
	}
	// Solo `worktree` e `in_place` son respuestas: `ask` es una preferencia, no un modo que
	// se materialice.
	if r.Mode != "" && r.Mode != branch.ModeWorktree && r.Mode != branch.ModeInPlace {
		return refused(branch.InvalidMode(operation, r.Change, r.CWD, r.Lane, r.Mode))
	}
	// Un nombre inválido de la invocación se rechaza antes de bloquear el cambio ni leer nada.
	if refusal := branch.ValidateExceptions(operation, r.CWD, r.Change, r.Exceptions); refusal != nil {
		return refused(refusal)
	}
	options := gatherOptions{exceptions: r.Exceptions, repos: r.Repos}

	// El bloqueo se toma en cuanto se conoce el git common dir y se suelta al terminar,
	// también cuando se rechaza.
	var unlock func()
	defer func() {
		if unlock != nil {
			unlock()
		}
	}()
	acquire := func(commonDir string) *branch.Refusal {
		release, err := e.registry.Lock(ctx, commonDir, r.Lane, r.Change)
		if err != nil {
			return lockRefusal(err, operation, r.Change, r.CWD)
		}
		unlock = release
		return nil
	}

	facts, refusal := e.gatherWith(ctx, operation, r.CWD, r.Change, r.Lane, acquire, options)
	if refusal != nil {
		return refused(refusal)
	}
	mode, refusal := resolveMode(operation, r, facts)
	if refusal != nil {
		return refused(refusal)
	}
	// En `in_place` las ramas se despliegan en el propio repositorio: el plan no lleva rutas
	// de worktree (ni se comprueba su colocación ni su longitud).
	facts.input.InPlace = mode.mode == branch.ModeInPlace
	plan, refusal := branch.Plan(facts.input)
	if refusal != nil {
		return refused(refusal)
	}

	pending, fetched, refusal := e.checkPlan(ctx, operation, r.CWD, plan, facts, r.Fetch)
	if refusal != nil {
		return refused(refusal)
	}
	if len(pending) > 0 {
		if refusal := e.apply(ctx, operation, r.CWD, pending); refusal != nil {
			return refused(refusal)
		}
		// El estado final se lee de git otra vez: es el mismo camino que sigue la
		// ejecución que no tiene nada que crear, de modo que ambas describen lo mismo. El
		// cambio ya está bloqueado: esta lectura no lo vuelve a bloquear.
		if facts, refusal = e.gatherWith(ctx, operation, r.CWD, r.Change, r.Lane, nil, options); refusal != nil {
			return refused(refusal)
		}
		facts.input.InPlace = mode.mode == branch.ModeInPlace
		if plan, refusal = branch.Plan(facts.input); refusal != nil {
			return refused(refusal)
		}
	}

	record := newRecord(facts, plan, mode)
	if err := e.registry.Save(facts.input.PlanningRepo.CommonDir, record); err != nil {
		return refused(writeRefusal(err, operation, r.Change, r.CWD))
	}
	facts.record = &record

	result, refusal := e.describe(ctx, operation, r.CWD, facts, plan)
	if refusal != nil {
		return refused(refusal)
	}
	// `fetched` dice qué repositorios se trajeron en esta ejecución: describe solo lee
	// referencias locales y deja `base.fetched` en falso.
	for i := range result.Repositories {
		if base := result.Repositories[i].Base; base != nil && fetched[result.Repositories[i].GitRoot] {
			base.Fetched = true
		}
	}
	// En `in_place` no hay worktrees: la raíz de worktrees no se informa.
	if !facts.input.InPlace {
		result.WorktreeRoot = plan.WorktreeRoot
	}
	result.Warnings = append(result.Warnings, facts.warnings...)
	if facts.prefs.warning != "" {
		result.Warnings = append(result.Warnings, facts.prefs.warning)
	}
	return result
}

// ensureStep es lo que falta por crear de una rama del plan.
type ensureStep struct {
	branch string
	// createFrom es el punto de partida de la rama, o vacío si la rama ya existe.
	createFrom string
	// worktree es la ruta del worktree que falta, o vacía si la rama ya está desplegada.
	worktree string
	// switchTo dice que, en `in_place`, la rama aún no está desplegada en el propio
	// repositorio y hay que cambiar a ella con `git switch`.
	switchTo bool
}

// repositorySteps son los pasos pendientes de un repositorio del plan, con su checkout
// principal: toda creación se lanza desde él, también cuando cwd era un worktree enlazado
// (matriz de amenazas, «Selección de repositorio git»).
type repositorySteps struct {
	name         string
	mainCheckout string
	steps        []ensureStep
	// fetched dice que la comprobación ejecutó `git fetch origin` en este repositorio.
	fetched bool
}

// checkPlan es la fase 3 de D-05: comprueba, sin efectos, todos los repositorios del plan
// antes de mutar ninguno y devuelve lo que falta por crear en cada uno, y los repositorios
// (por raíz git) en los que se ejecutó `git fetch origin`. Con fetch, cada repositorio con
// ramas se trae justo antes de resolver su base. Los repositorios sin ramas (el de specs de
// un multirepo, que se trabaja in situ) no tienen nada que traer ni que crear, pero sí que
// estar en su rama de integración (checkSpecsRepository, REQ-25.5).
func (e *Engine) checkPlan(ctx context.Context, operation, cwd string, plan branch.ChangePlan, facts changeFacts, fetch bool) ([]repositorySteps, map[string]bool, *branch.Refusal) {
	var pending []repositorySteps
	fetched := map[string]bool{}
	for _, repo := range plan.Repositories {
		if len(repo.Branches) == 0 {
			if repo.Role == branch.RepoSpecsInPlace {
				if refusal := checkSpecsRepository(operation, cwd, repo, facts); refusal != nil {
					return nil, nil, refusal
				}
			}
			continue
		}
		steps, refusal := e.checkRepository(ctx, operation, cwd, repo, facts, fetch)
		if refusal != nil {
			return nil, nil, refusal
		}
		fetched[repo.GitRoot] = steps.fetched
		if len(steps.steps) > 0 {
			pending = append(pending, steps)
		}
	}
	return pending, fetched, nil
}

// checkRepository comprueba un repositorio del plan: con fetch, trae `origin`; después, que
// tiene base y, por rama, si existe, si está desplegada y dónde, y si su ruta planificada
// está libre. Devuelve el rechazo que impide aplicar el plan, o los pasos que faltan.
func (e *Engine) checkRepository(ctx context.Context, operation, cwd string, repo branch.RepoPlan, facts changeFacts, fetch bool) (repositorySteps, *branch.Refusal) {
	change, master := facts.input.Change, facts.input.Master
	listed := facts.worktrees[repo.GitRoot]
	if len(listed) == 0 {
		return repositorySteps{}, gitRefusal(operation, cwd, "listar los worktrees de "+repo.GitRoot, errors.New("git no devolvió ningún worktree"))
	}
	pending := repositorySteps{name: repo.Name, mainCheckout: listed[0].Path}

	// El fetch va antes de resolver la base (D-06): es lo único que cambia el resultado de
	// resolveBase. Solo trae referencias remotas locales; no crea ramas ni worktrees.
	if fetch {
		if err := e.writer.Fetch(ctx, pending.mainCheckout, originRemote); err != nil {
			return repositorySteps{}, branch.FetchFailed(operation, change, master, repo, originRemote, err)
		}
		pending.fetched = true
	}

	base, haveBase, err := e.resolveBase(ctx, repo)
	if err != nil {
		return repositorySteps{}, gitRefusal(operation, cwd, "resolver la rama de integración de "+repo.GitRoot, err)
	}
	if !haveBase {
		return repositorySteps{}, branch.BaseNotFound(operation, change, master, repo)
	}

	registered := registeredWorktrees(facts.record, repo)
	deployed := map[string]branch.WorktreeEntry{}
	for _, entry := range listed {
		// Las vistas de candidato de RDD son worktrees desacoplados: no cuentan (D-07).
		if entry.Branch != "" && !entry.Detached {
			deployed[entry.Branch] = entry
		}
	}
	if facts.input.InPlace {
		return e.checkInPlaceRepository(ctx, operation, cwd, repo, facts, pending, base, deployed)
	}

	for _, planned := range repo.Branches {
		exists, err := e.git.BranchExists(ctx, repo.GitRoot, planned.Name)
		if err != nil {
			return repositorySteps{}, gitRefusal(operation, cwd, "comprobar si existe la rama "+planned.Name, err)
		}
		step := ensureStep{branch: planned.Name}
		if !exists {
			// La rama del cambio sale de la base; las de rol, de la de seguimiento, que el
			// plan crea antes (va la primera de la lista).
			step.createFrom = base.sha
			if planned.Kind == branch.KindRole {
				step.createFrom = "refs/heads/" + planned.From
			}
		}

		entry, isDeployed := deployed[planned.Name]
		switch {
		case isDeployed:
			known := e.samePlace(entry.Path, planned.Worktree) || (registered[planned.Name] != "" && e.samePlace(entry.Path, registered[planned.Name]))
			switch {
			case !known:
				inMain := e.samePlace(entry.Path, pending.mainCheckout)
				return repositorySteps{}, branch.BranchCheckedOutElsewhere(operation, change, master, repo, planned.Name, entry.Path, pending.mainCheckout, inMain)
			case !directoryExists(entry.Path):
				return repositorySteps{}, branch.PlannedPathConflict(operation, change, master, planned.Name, entry.Path,
					fmt.Sprintf("el worktree %s de la rama %s está registrado en git pero su directorio ya no existe", entry.Path, planned.Name),
					fmt.Sprintf("elimina su registro con `git -C %s worktree prune`", pathquote.Quote(pending.mainCheckout)))
			}
		case worktreeAt(listed, planned.Worktree, e.samePlace) != nil:
			other := worktreeAt(listed, planned.Worktree, e.samePlace)
			held := "HEAD desacoplado"
			if other.Branch != "" {
				held = "la rama " + other.Branch
			}
			return repositorySteps{}, branch.PlannedPathConflict(operation, change, master, planned.Name, planned.Worktree,
				fmt.Sprintf("la ruta %s planificada para la rama %s ya es un worktree con %s", planned.Worktree, planned.Name, held),
				fmt.Sprintf("retira ese worktree con `git -C %s worktree remove %s` o elige otro nombre de cambio", pathquote.Quote(pending.mainCheckout), pathquote.Quote(planned.Worktree)))
		default:
			if reason := occupied(planned.Worktree); reason != "" {
				return repositorySteps{}, branch.PlannedPathConflict(operation, change, master, planned.Name, planned.Worktree,
					fmt.Sprintf("la ruta %s planificada para el worktree de la rama %s %s", planned.Worktree, planned.Name, reason),
					fmt.Sprintf("mueve o borra %s, o elige otro nombre de cambio", pathquote.Quote(planned.Worktree)))
			}
			step.worktree = planned.Worktree
		}
		if step.createFrom != "" || step.worktree != "" {
			pending.steps = append(pending.steps, step)
		}
	}
	return pending, nil
}

// checkInPlaceRepository es checkRepository en modo `in_place` (REQ-25.8, REQ-25.9): las
// ramas del plan se crean igual (la del cambio desde la base, las de rol desde ella), pero
// solo la de seguimiento, la primera, se despliega, y en el propio repositorio, que es su
// checkout principal (limitación documentada, R26: las ramas de rol no se despliegan). Si ya
// es la rama actual del checkout principal, no hay nada que cambiar y el árbol puede estar
// sucio (no-op); si está desplegada en otro worktree, git no admite desplegarla también aquí
// (`branch_checked_out_elsewhere`); si no, hay que cambiar de rama y el árbol debe estar
// limpio: `dirty_worktree` sin efectos (git status --porcelain=v1 -z --untracked-files=normal,
// con los ficheros ignorados como limpios). pending ya lleva el checkout principal.
func (e *Engine) checkInPlaceRepository(ctx context.Context, operation, cwd string, repo branch.RepoPlan, facts changeFacts, pending repositorySteps, base baseReference, deployed map[string]branch.WorktreeEntry) (repositorySteps, *branch.Refusal) {
	change, master := facts.input.Change, facts.input.Master
	switching := false
	for i, planned := range repo.Branches {
		exists, err := e.git.BranchExists(ctx, repo.GitRoot, planned.Name)
		if err != nil {
			return repositorySteps{}, gitRefusal(operation, cwd, "comprobar si existe la rama "+planned.Name, err)
		}
		step := ensureStep{branch: planned.Name}
		if !exists {
			step.createFrom = base.sha
			if planned.Kind == branch.KindRole {
				step.createFrom = "refs/heads/" + planned.From
			}
		}
		if i == 0 {
			entry, isDeployed := deployed[planned.Name]
			switch {
			case !isDeployed:
				step.switchTo = true
				switching = true
			case !e.samePlace(entry.Path, pending.mainCheckout):
				return repositorySteps{}, branch.BranchCheckedOutElsewhere(operation, change, master, repo, planned.Name, entry.Path, pending.mainCheckout, false)
			}
		}
		if step.createFrom != "" || step.switchTo {
			pending.steps = append(pending.steps, step)
		}
	}
	if !switching {
		return pending, nil
	}
	dirty, err := e.git.DirtyPaths(ctx, pending.mainCheckout)
	if err != nil {
		return repositorySteps{}, gitRefusal(operation, cwd, "leer el estado del repositorio "+pending.mainCheckout, err)
	}
	if len(dirty) > 0 {
		return repositorySteps{}, branch.DirtyWorktree(operation, change, cwd, facts.input.Lane, repo, pending.mainCheckout, dirty, facts.record != nil)
	}
	return pending, nil
}

// apply es la fase 4 de D-05: ejecuta solo lo que falta, repositorio a repositorio y rama
// a rama, en el orden del plan (la rama de seguimiento antes que las de rol). Un fallo de
// git para la aplicación con `git_failed`; lo ya creado se conserva y repetir `ensure`
// completa el resto.
func (e *Engine) apply(ctx context.Context, operation, cwd string, pending []repositorySteps) *branch.Refusal {
	for _, repo := range pending {
		for _, step := range repo.steps {
			if step.createFrom != "" {
				if err := e.writer.CreateBranch(ctx, repo.mainCheckout, step.branch, step.createFrom); err != nil {
					return branch.GitFailed(operation, cwd, fmt.Sprintf("crear la rama %s en %s", step.branch, repo.name), err)
				}
			}
			if step.worktree != "" {
				if err := e.writer.AddWorktree(ctx, repo.mainCheckout, step.worktree, step.branch); err != nil {
					return branch.GitFailed(operation, cwd, fmt.Sprintf("crear el worktree %s de la rama %s", step.worktree, step.branch), err)
				}
			}
			if step.switchTo {
				if err := e.writer.Switch(ctx, repo.mainCheckout, step.branch); err != nil {
					return branch.GitFailed(operation, cwd, fmt.Sprintf("cambiar el repositorio %s a la rama %s", repo.name, step.branch), err)
				}
			}
		}
	}
	return nil
}

// samePlace dice si dos rutas son el mismo lugar: el mismo directorio existente (por
// identidad, que cubre los nombres cortos 8.3 y los enlaces) o, si alguno no existe, la
// misma ruta limpia, sin distinguir mayúsculas en Windows (regla 10 de tasks.md).
func (e *Engine) samePlace(a, b string) bool {
	if pathidentity.SameDirectory(a, b) {
		return true
	}
	a, b = filepath.Clean(a), filepath.Clean(b)
	if e.goos == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// worktreeAt devuelve el worktree de la lista que está en path, o nil.
func worktreeAt(listed []branch.WorktreeEntry, path string, same func(a, b string) bool) *branch.WorktreeEntry {
	for i := range listed {
		if same(listed[i].Path, path) {
			return &listed[i]
		}
	}
	return nil
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// occupied explica por qué una ruta no está libre para un worktree nuevo, o devuelve la
// cadena vacía: no existe o es un directorio vacío, que git admite.
func occupied(path string) string {
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ""
	case err != nil:
		return fmt.Sprintf("no es accesible (%v)", err)
	case !info.IsDir():
		return "ya existe y no es un directorio"
	}
	entries, err := os.ReadDir(path)
	switch {
	case err != nil:
		return fmt.Sprintf("no se puede leer (%v)", err)
	case len(entries) > 0:
		return "ya existe y es un directorio con contenido que no es un worktree"
	}
	return ""
}
