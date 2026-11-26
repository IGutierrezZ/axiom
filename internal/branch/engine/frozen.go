package engine

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// Nombres congelados y recuperación (C-07-f, design D-09). El registro local es la fuente de verdad
// del nombre de una rama renderizada con plantilla: el motor se lo entrega al planificador en
// RepoFacts.Frozen y, si el registro se perdió, recupera las ramas pendientes que ya existen en git.

// frozenFor son las ramas con plantilla que el registro guarda para el repositorio cuyo git common
// dir es commonDir (se reconoce por identidad de ruta, como recordedBranches).
func frozenFor(record *branch.Record, commonDir string) []branch.FrozenBranch {
	if record == nil {
		return nil
	}
	var frozen []branch.FrozenBranch
	for _, repo := range record.Repositories {
		if pathidentity.SameDirectory(repo.CommonDir, commonDir) {
			frozen = append(frozen, branch.FrozenFrom(repo.Branches)...)
		}
	}
	return frozen
}

// freezeFromRecord rellena RepoFacts.Frozen de los repositorios de la entrada con lo que guarda el
// registro. Sin plantilla en el registro no cambia nada.
func freezeFromRecord(record *branch.Record, planning *branch.RepoFacts, groups ...[]branch.RepoFacts) {
	planning.Frozen = frozenFor(record, planning.CommonDir)
	for _, group := range groups {
		for i := range group {
			group[i].Frozen = frozenFor(record, group[i].CommonDir)
		}
	}
}

// anyTemplate dice si alguna plantilla puede intervenir en el plan: la de la política (`git:`,
// repositorios y roles) o las entradas pendientes de la instantánea.
func anyTemplate(in branch.PlanInput) bool {
	if in.Policy.BranchTemplate != "" || len(in.Policy.RoleTemplates) > 0 {
		return true
	}
	for _, repo := range in.Policy.PerRepository {
		if repo.BranchTemplate != "" {
			return true
		}
	}
	if in.Snapshot != nil {
		for _, repo := range in.Snapshot.Repositories {
			if len(repo.PendingBranches) > 0 {
				return true
			}
		}
	}
	return false
}

// recoverPending recupera, sin registro, el nombre de las ramas pendientes cuyo candidato ya existe
// en git (design D-09, S3, G5). Planifica con los hechos actuales y, para cada rama pendiente con
// `Candidate`, mira con BranchExists solo en `refs/heads` (nunca `refs/remotes`: no adopta la rama de
// un rol que no entra) si existe: un error distinto de «no existe» se propaga, dos roles del mismo
// repositorio con el mismo candidato son `invalid_name` y, si existe, el nombre pasa a
// RepoFacts.Frozen para que el siguiente plan lo trate como congelado. Sin plantillas no hace nada.
func (e *Engine) recoverPending(ctx context.Context, operation, cwd string, facts *changeFacts) *branch.Refusal {
	if !anyTemplate(facts.input) {
		return nil
	}
	// Solo interesan los nombres: sin worktrees, para no rechazar por una colocación que `in_place` no usa.
	probe := facts.input
	probe.InPlace = true
	plan, refusal := branch.Plan(probe)
	if refusal != nil {
		return refusal
	}
	for _, repo := range plan.Repositories {
		candidates := map[string][]string{}
		for _, pending := range repo.Pending {
			if pending.Candidate != "" {
				candidates[pending.Candidate] = append(candidates[pending.Candidate], pendingLabel(pending))
			}
		}
		for _, name := range slices.Sorted(maps.Keys(candidates)) {
			if roles := candidates[name]; len(roles) > 1 {
				slices.Sort(roles)
				return branch.TemplateCandidateCollision(facts.input.Change, facts.input.Master, repo.Name, name, roles)
			}
		}
		for _, pending := range repo.Pending {
			if pending.Candidate == "" {
				continue
			}
			exists, err := e.git.BranchExists(ctx, repo.GitRoot, pending.Candidate)
			if err != nil {
				return gitRefusal(operation, cwd, "comprobar si existe la rama "+pending.Candidate, err)
			}
			if !exists {
				continue
			}
			frozen := branch.FrozenBranch{Kind: pending.Kind, Name: pending.Candidate, Template: pending.Template}
			if pending.Kind == branch.KindRole {
				frozen.RoleID = pending.RoleID
			}
			addFrozen(&facts.input, repo.GitRoot, frozen)
		}
	}
	return nil
}

// pendingLabel nombra el rol de una rama pendiente en un mensaje.
func pendingLabel(pending branch.PendingBranch) string {
	if pending.RoleID == "" {
		return "cambio"
	}
	return strings.ToLower(pending.RoleID)
}

// addFrozen añade una rama congelada a los hechos del repositorio de raíz gitRoot.
func addFrozen(in *branch.PlanInput, gitRoot string, frozen branch.FrozenBranch) {
	if in.PlanningRepo.GitRoot == gitRoot {
		in.PlanningRepo.Frozen = append(slices.Clone(in.PlanningRepo.Frozen), frozen)
	}
	for _, group := range [][]branch.RepoFacts{in.CodeRepos, in.IncludedRepos} {
		for i := range group {
			if group[i].GitRoot == gitRoot {
				group[i].Frozen = append(slices.Clone(group[i].Frozen), frozen)
			}
		}
	}
}
