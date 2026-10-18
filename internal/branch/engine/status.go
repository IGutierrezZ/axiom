package engine

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// originRemote es el remoto cuya rama de integración, ya traída a local, sirve de base
// (design D-06). `status` nunca consulta el remoto: solo lee las referencias locales.
const originRemote = "origin"

// Status describe, solo con lecturas, el estado de las ramas y los worktrees de un
// cambio (REQ-25.15): por repositorio, su papel, la rama actual y la base; por rama, si
// existe, su worktree, si tiene cambios sin confirmar, su adelanto y su atraso respecto
// de su base y el destino propuesto para su PR. Un cambio sin ramas creadas es
// `not_started`, no un error. cwd debe ser un directorio absoluto; el cambio es
// obligatorio en esta rebanada. Nunca ejecuta operaciones remotas (REQ-25.1).
func (e *Engine) Status(ctx context.Context, cwd, change string, lane branch.Lane) branch.Result {
	const operation = "status"
	input, refusal := e.planInput(ctx, operation, cwd, change, lane)
	if refusal != nil {
		return branch.NewResult(operation).Refused(refusal)
	}
	plan, refusal := branch.Plan(input)
	if refusal != nil {
		return branch.NewResult(operation).Refused(refusal)
	}
	result, refusal := e.describe(ctx, operation, cwd, input, plan)
	if refusal != nil {
		return branch.NewResult(operation).Refused(refusal)
	}
	return result
}

// describe lee el estado de git de cada rama del plan y compone el resultado.
func (e *Engine) describe(ctx context.Context, operation, cwd string, input branch.PlanInput, plan branch.ChangePlan) (branch.Result, *branch.Refusal) {
	result := branch.NewResult(operation)
	result.Change, result.Lane, result.Topology = plan.Change, plan.Lane, input.Topology
	result.WorkspaceRoot, result.PlanningHome, result.OddDocument = plan.WorkspaceRoot, plan.PlanningHome, plan.OddDocument
	result.Skipped = plan.Skipped
	result.Warnings = slices.Clone(plan.Warnings)

	planned, existing := 0, 0
	for _, repo := range plan.Repositories {
		repository, warnings, refusal := e.describeRepository(ctx, operation, cwd, repo)
		if refusal != nil {
			return branch.Result{}, refusal
		}
		result.Repositories = append(result.Repositories, repository)
		result.Warnings = append(result.Warnings, warnings...)
		for _, described := range repository.Branches {
			planned++
			if described.Exists {
				existing++
			}
		}
	}
	if planned > 0 && existing == 0 {
		result = result.NotStarted()
	}
	return result, nil
}

// baseReference es la base de un repositorio: la referencia con la que se mide el
// adelanto y el atraso de sus ramas.
type baseReference struct {
	ref    string // como se muestra: `origin/main` o `main`
	full   string // referencia completa que se resuelve: `refs/remotes/origin/main`
	source string // `remote-tracking` o `local`
	sha    string
}

// resolveBase resuelve la base sin red (design D-06): `refs/remotes/origin/<integración>`
// si existe en local y, si no, `refs/heads/<integración>`. La excepción `--base`
// (RepoPlan.BaseRef) la aplicará `status` cuando el registro local aporte las
// excepciones del cambio; hasta entonces no llega ninguna. ok es falso si no existe
// ninguna de las dos.
func (e *Engine) resolveBase(ctx context.Context, repo branch.RepoPlan) (base baseReference, ok bool, err error) {
	for _, candidate := range []baseReference{
		{ref: originRemote + "/" + repo.IntegrationBranch, full: "refs/remotes/" + originRemote + "/" + repo.IntegrationBranch, source: "remote-tracking"},
		{ref: repo.IntegrationBranch, full: "refs/heads/" + repo.IntegrationBranch, source: "local"},
	} {
		sha, found, err := e.git.ResolveCommit(ctx, repo.GitRoot, candidate.full)
		if err != nil {
			return baseReference{}, false, err
		}
		if found {
			candidate.sha = sha
			return candidate, true, nil
		}
	}
	return baseReference{}, false, nil
}

// describeRepository lee el estado de git de un repositorio del plan: la rama actual
// (de ficheros, sin procesos, REQ-25.21), la base y, por rama planificada, si existe,
// dónde está desplegada, si su worktree está limpio y cuánto se ha alejado de su base.
func (e *Engine) describeRepository(ctx context.Context, operation, cwd string, repo branch.RepoPlan) (branch.RepositoryResult, []string, *branch.Refusal) {
	repository := branch.NewRepositoryResult(repo)
	var warnings []string
	switch name, detached, err := gitcommon.ReadHead(repo.GitRoot); {
	case err != nil:
		warnings = append(warnings, fmt.Sprintf("no se pudo determinar la rama actual de %s: %v", repo.Name, err))
	case detached:
		repository.Detached = true
	default:
		repository.CurrentBranch = name
	}
	if len(repo.Branches) == 0 {
		return repository, warnings, nil
	}

	worktrees, err := e.git.Worktrees(ctx, repo.GitRoot)
	if err != nil {
		return branch.RepositoryResult{}, nil, gitRefusal(operation, cwd, "listar los worktrees de "+repo.GitRoot, err)
	}
	deployed := map[string]string{}
	for _, entry := range worktrees {
		if entry.Branch != "" && !entry.Detached {
			deployed[entry.Branch] = entry.Path
		}
	}

	base, haveBase, err := e.resolveBase(ctx, repo)
	if err != nil {
		return branch.RepositoryResult{}, nil, gitRefusal(operation, cwd, "resolver la rama de integración de "+repo.GitRoot, err)
	}
	if haveBase {
		repository.Base = &branch.BaseResult{Ref: base.ref, SHA: base.sha, Source: base.source}
	} else {
		warnings = append(warnings, fmt.Sprintf("no existe la rama de integración «%s» de %s, ni en local ni como referencia remota local: no se calculan adelanto ni atraso", repo.IntegrationBranch, repo.Name))
	}

	exists := map[string]bool{}
	for i := range repository.Branches {
		described := &repository.Branches[i]
		described.Exists, err = e.git.BranchExists(ctx, repo.GitRoot, described.Name)
		if err != nil {
			return branch.RepositoryResult{}, nil, gitRefusal(operation, cwd, "comprobar si existe la rama "+described.Name, err)
		}
		exists[described.Name] = described.Exists
		if !described.Exists {
			continue
		}

		if worktree, isDeployed := deployed[described.Name]; isDeployed {
			if _, statErr := os.Stat(worktree); statErr != nil {
				warnings = append(warnings, fmt.Sprintf("el worktree %s de la rama %s ya no existe en disco; elimina su registro con `git -C %s worktree prune`", worktree, described.Name, pathquote.Quote(repo.GitRoot)))
			} else {
				described.Worktree = worktree
				dirty, err := e.git.DirtyPaths(ctx, worktree)
				if err != nil {
					return branch.RepositoryResult{}, nil, gitRefusal(operation, cwd, "leer el estado del worktree "+worktree, err)
				}
				described.Dirty = len(dirty) > 0
			}
		}

		// Una rama de rol se mide contra la de seguimiento, de la que sale; las demás,
		// contra la base del repositorio.
		compared := comparisonFor(repo.Branches[i], base, haveBase, exists)
		if compared.full == "" {
			continue
		}
		ahead, behind, err := e.git.AheadBehind(ctx, repo.GitRoot, compared.full, "refs/heads/"+described.Name)
		if err != nil {
			return branch.RepositoryResult{}, nil, gitRefusal(operation, cwd, "contar los commits de la rama "+described.Name, err)
		}
		described.ComparedTo, described.Ahead, described.Behind = compared.ref, &ahead, &behind
	}
	return repository, warnings, nil
}

// comparisonFor elige la referencia con la que se mide el adelanto y el atraso de una
// rama: la de seguimiento para una rama de rol cuando esta existe y, en cualquier otro
// caso, la base del repositorio. Sin ninguna de las dos devuelve una referencia vacía.
func comparisonFor(planned branch.BranchPlan, base baseReference, haveBase bool, exists map[string]bool) baseReference {
	if planned.Kind == branch.KindRole && exists[planned.From] {
		return baseReference{ref: planned.From, full: "refs/heads/" + planned.From}
	}
	if haveBase {
		return base
	}
	return baseReference{}
}
