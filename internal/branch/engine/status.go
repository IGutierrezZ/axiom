package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// originRemote es el remoto cuya rama de integración, ya traída a local, sirve de base
// (design D-06). `status` nunca consulta el remoto: solo lee las referencias locales.
const originRemote = "origin"

// Status describe, solo con lecturas, el estado de las ramas y los worktrees de un
// cambio (REQ-25.15): por repositorio, su papel, la rama actual y la base; por rama, si
// existe, su worktree, si tiene cambios sin confirmar, su adelanto y su atraso respecto
// de su base y el destino propuesto para su PR. Un cambio sin ramas creadas es
// `not_started`, no un error. Con registro local, informa además del modo de trabajo
// materializado y de los worktrees registrados, y aplica las excepciones que el registro
// guarda (`--prefix`, `--branch`, `--base`); sin él, `registry.present` es falso.
// cwd debe ser un directorio absoluto; el cambio es obligatorio en esta rebanada. Nunca
// ejecuta operaciones remotas (REQ-25.1) ni escribe el registro.
func (e *Engine) Status(ctx context.Context, cwd, change string, lane branch.Lane) branch.Result {
	const operation = "status"
	facts, refusal := e.gather(ctx, operation, cwd, change, lane)
	if refusal != nil {
		return branch.NewResult(operation).Refused(refusal)
	}
	plan, refusal := branch.Plan(facts.input)
	if refusal != nil {
		return branch.NewResult(operation).Refused(refusal)
	}
	result, refusal := e.describe(ctx, operation, cwd, facts, plan)
	if refusal != nil {
		return branch.NewResult(operation).Refused(refusal)
	}
	return result
}

// describe lee el estado de git de cada rama del plan y compone el resultado.
func (e *Engine) describe(ctx context.Context, operation, cwd string, facts changeFacts, plan branch.ChangePlan) (branch.Result, *branch.Refusal) {
	result := branch.NewResult(operation)
	result.Change, result.Lane, result.Topology = plan.Change, plan.Lane, facts.input.Topology
	result.WorkspaceRoot, result.PlanningHome, result.OddDocument = plan.WorkspaceRoot, plan.PlanningHome, plan.OddDocument
	result.Registry = &branch.RegistryResult{Present: facts.record != nil}
	if facts.record != nil {
		result.SetMode(facts.record.Mode, facts.record.ModeSource)
	}
	result.Skipped = plan.Skipped
	result.Warnings = slices.Clone(plan.Warnings)

	planned, existing := 0, 0
	for _, repo := range plan.Repositories {
		repository, warnings, refusal := e.describeRepository(ctx, operation, cwd, repo, facts)
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

// baseCandidates son las referencias que pueden ser la base de un repositorio, por orden
// de preferencia (design D-06). Sin excepción `--base`: `refs/remotes/origin/<integración>`
// y, si no existe, `refs/heads/<integración>`. Con `--base <ref>` (RepoPlan.BaseRef, que
// el registro aporta): la rama local, la remota ya traída a local y, por último, la
// referencia tal cual (una etiqueta o un commit), en el orden en que git resuelve un
// nombre; la excepción sustituye a la rama de integración, no se combina con ella.
func baseCandidates(repo branch.RepoPlan) []baseReference {
	if ref := repo.BaseRef; ref != "" {
		return []baseReference{
			{ref: ref, full: "refs/heads/" + ref, source: "local"},
			{ref: ref, full: "refs/remotes/" + ref, source: "remote-tracking"},
			{ref: ref, full: ref, source: "local"},
		}
	}
	return []baseReference{
		{ref: originRemote + "/" + repo.IntegrationBranch, full: "refs/remotes/" + originRemote + "/" + repo.IntegrationBranch, source: "remote-tracking"},
		{ref: repo.IntegrationBranch, full: "refs/heads/" + repo.IntegrationBranch, source: "local"},
	}
}

// resolveBase resuelve la base sin red (design D-06): la primera de baseCandidates que
// existe en local. ok es falso si no existe ninguna.
func (e *Engine) resolveBase(ctx context.Context, repo branch.RepoPlan) (base baseReference, ok bool, err error) {
	for _, candidate := range baseCandidates(repo) {
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

// missingBaseWarning es el aviso de un repositorio sin base: no se calculan adelanto ni
// atraso.
func missingBaseWarning(repo branch.RepoPlan) string {
	if repo.BaseRef != "" {
		return fmt.Sprintf("la base «%s» fijada para el cambio no existe en %s: no se calculan adelanto ni atraso", repo.BaseRef, repo.Name)
	}
	return fmt.Sprintf("no existe la rama de integración «%s» de %s, ni en local ni como referencia remota local: no se calculan adelanto ni atraso", repo.IntegrationBranch, repo.Name)
}

// registeredWorktrees son los worktrees que el registro local guarda para las ramas de un
// repositorio del plan, por nombre de rama y con la ruta limpia, como las de git. El
// repositorio se reconoce por su git common dir, que es el mismo desde el checkout
// principal y desde cualquier worktree enlazado.
func registeredWorktrees(record *branch.Record, repo branch.RepoPlan) map[string]string {
	if record == nil {
		return nil
	}
	for _, registered := range record.Repositories {
		if !pathidentity.SameDirectory(registered.CommonDir, repo.CommonDir) {
			continue
		}
		worktrees := map[string]string{}
		for _, recorded := range registered.Branches {
			if recorded.Worktree != "" {
				worktrees[recorded.Name] = filepath.Clean(recorded.Worktree)
			}
		}
		return worktrees
	}
	return nil
}

// describeRepository lee el estado de git de un repositorio del plan: la rama actual
// (de ficheros, sin procesos, REQ-25.21), la base y, por rama planificada, si existe,
// dónde está desplegada, si su worktree está limpio y cuánto se ha alejado de su base.
// Los worktrees que git conoce ya los reunió gather.
func (e *Engine) describeRepository(ctx context.Context, operation, cwd string, repo branch.RepoPlan, facts changeFacts) (branch.RepositoryResult, []string, *branch.Refusal) {
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

	deployed := map[string]string{}
	for _, entry := range facts.worktrees[repo.GitRoot] {
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
		warnings = append(warnings, missingBaseWarning(repo))
	}

	registered := registeredWorktrees(facts.record, repo)
	exists := map[string]bool{}
	for i := range repository.Branches {
		described := &repository.Branches[i]
		described.RegisteredWorktree = registered[described.Name]
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
