package engine

import (
	"context"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// El plan de archivado en el motor (REQ-25.45, design D-25): cómo se compone la entrada de
// `branch.Plan` para la rama de archivado a partir de los hechos del cambio y qué muestra
// `status` de ella. `ensure --archive` (archive.go) y `status` comparten esta composición, de
// modo que ambos nombran la misma rama.

// archivePlanInput es la entrada del plan de archivado de un cambio (branch.PlanInput.Archive):
// los hechos del cambio, con la rama del cambio registrada de la que deriva el nombre, la de
// archivado que el registro ya guarda y las excepciones de la invocación. requested.Prefix es el
// prefijo de la rama de archivado; sin él rige la rama de archivado registrada y, si no hay
// ninguna, el prefijo de archivado que guarda el registro. Branch y Base se pasan tal cual para
// que el planificador las rechace (`plan_conflict`, S-16).
func archivePlanInput(facts changeFacts, requested branch.Exceptions) branch.PlanInput {
	in := facts.input
	in.Archive = true
	// La rama del cambio sale de los hechos antes de sustituir las excepciones: las registradas
	// son las del cambio y las de la invocación son las del archivado.
	in.ChangeBranch = registeredChangeBranch(facts)
	recorded, isRecorded := recordedArchiveBranch(facts.record, in.PlanningRepo)
	if isRecorded {
		in.ArchiveBranch = recorded.Name
	}
	prefix := requested.Prefix
	if prefix == "" && !isRecorded && facts.record != nil {
		prefix = facts.record.ArchivePrefix
	}
	in.Exceptions = branch.Exceptions{Prefix: prefix, Branch: requested.Branch, Base: requested.Base}
	// El archivado solo planifica el repositorio del planning home.
	in.CodeRepos, in.IncludedRepos = nil, nil
	return in
}

// registeredChangeBranch es la rama del cambio de la que deriva la de archivado (design D-25,
// paso 3: «registro, instantánea o política, en ese orden»; la instantánea llega con S8): la
// rama de tipo `change` que el registro guarda para el repositorio del planning home (se reconoce
// por su git common dir, que es el mismo desde el checkout principal y desde un worktree enlazado)
// y, sin ella, la que fijan las excepciones registradas del cambio (`--branch`, o `--prefix` más el
// cambio). Vacía, la compone la política.
func registeredChangeBranch(facts changeFacts) string {
	planning := facts.input.PlanningRepo
	if facts.record != nil {
		for _, repo := range facts.record.Repositories {
			if !pathidentity.SameDirectory(repo.CommonDir, planning.CommonDir) {
				continue
			}
			for _, recorded := range repo.Branches {
				if recorded.Kind == branch.KindChange {
					return recorded.Name
				}
			}
		}
	}
	switch exceptions := facts.input.Exceptions; {
	case exceptions.Branch != "":
		return exceptions.Branch
	case exceptions.Prefix != "":
		return exceptions.Prefix + "/" + facts.input.Change
	}
	return ""
}

// recordedArchiveBranch devuelve la rama de archivado que el registro guarda para el repositorio
// del planning home, y false si no guarda ninguna.
func recordedArchiveBranch(record *branch.Record, planning branch.RepoFacts) (branch.RecordBranch, bool) {
	if record == nil {
		return branch.RecordBranch{}, false
	}
	for _, repo := range record.Repositories {
		if !pathidentity.SameDirectory(repo.CommonDir, planning.CommonDir) {
			continue
		}
		for _, recorded := range repo.Branches {
			if recorded.Kind == branch.KindArchive {
				return recorded, true
			}
		}
	}
	return branch.RecordBranch{}, false
}

// withArchiveBranch añade al plan ordinario de un cambio su rama de archivado cuando existe
// (REQ-25.15, REQ-25.4: la rama de archivado no pertenece al modelo de roles y su PR va a la
// integración): existe en git o consta en el registro. Sin ella, el plan queda como está. Solo
// el carril SDD de un monorepo tiene rama de archivado: en cualquier otro caso, y si el plan de
// archivado se rechaza (una ruta ocupada o demasiado larga, por ejemplo), `status` no la
// muestra y sigue informando de lo demás: es información y no una comprobación.
func (e *Engine) withArchiveBranch(ctx context.Context, operation, cwd string, facts changeFacts, plan branch.ChangePlan) (branch.ChangePlan, *branch.Refusal) {
	if facts.input.Lane != branch.LaneSDD || facts.input.Topology == workspace.TopologyMultirepo || len(plan.Repositories) == 0 {
		return plan, nil
	}
	archive, refusal := branch.Plan(archivePlanInput(facts, branch.Exceptions{}))
	if refusal != nil || len(archive.Repositories) != 1 || len(archive.Repositories[0].Branches) != 1 {
		return plan, nil
	}
	repo, planned := archive.Repositories[0], archive.Repositories[0].Branches[0]
	exists, err := e.git.BranchExists(ctx, repo.GitRoot, planned.Name)
	if err != nil {
		return branch.ChangePlan{}, gitRefusal(operation, cwd, "comprobar si existe la rama "+planned.Name, err)
	}
	// Una rama de archivado registrada figura aunque ya no exista en git: el registro es lo que
	// `ensure --archive` materializó y `status` informa de que falta.
	if _, isRecorded := recordedArchiveBranch(facts.record, facts.input.PlanningRepo); !exists && !isRecorded {
		return plan, nil
	}
	// El repositorio del planning home es siempre el primero del plan.
	plan.Repositories = slices.Clone(plan.Repositories)
	plan.Repositories[0].Branches = append(slices.Clone(plan.Repositories[0].Branches), planned)
	return plan, nil
}
