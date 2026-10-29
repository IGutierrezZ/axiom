package engine

import (
	"context"
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Snapshot calcula la instantánea de política git de un cambio (REQ-25.36, design D-12 y §4.5)
// que `kickoff seal` escribe una sola vez junto al resto de la configuración sellada: la
// integración y el prefijo del workspace, las excepciones por cambio que ya registró `ensure`
// (REQ-25.12) y, por repositorio, su ruta relativa a la carpeta maestra, su nombre, su papel, su
// integración, su base (un nombre de rama, nunca un SHA) y los nombres de rama derivados del
// cambio: la del cambio, las de rol de los repositorios que tocan dos o más de los roles que se
// sellan y, en el repositorio del planning home de un repositorio con rama de cambio, la de
// archivado (`<rama del cambio>-archive`).
//
// Es el plan de nombres del planificador de `ensure` aplicado a los roles que se van a sellar,
// sin kickoff aún, de modo que `ensure` y la instantánea nunca discrepan. Contiene política y
// ningún resultado: ni SHAs, ni rutas absolutas, ni rutas de worktree. No escribe nada: solo lee
// git y el registro local. Un nombre inválido o un fallo de lectura son un *branch.RefusalError
// tipado, con su continuación; con él no se escribe ninguna instantánea.
//
// Un repositorio fuera de la carpeta maestra no tiene ruta relativa que escribir ni entrada propia
// en `repositories:`: no figura, y sus nombres siguen saliendo de la política del workspace.
func (e *Engine) Snapshot(ctx context.Context, loc workspace.Location, change string, roles []string) (*kickoff.GitSnapshot, error) {
	facts, refusal := e.gatherWith(ctx, "status", loc.Start, change, branch.LaneSDD, nil, gatherOptions{sealing: true, sealRoles: roles})
	if refusal != nil {
		return nil, &branch.RefusalError{Refusal: *refusal}
	}
	// Solo se necesitan los nombres: sin rutas de worktree que colocar ni medir.
	facts.input.InPlace = true
	plan, refusal := branch.Plan(facts.input)
	if refusal != nil {
		return nil, &branch.RefusalError{Refusal: *refusal}
	}

	policy := branch.ResolvePolicy(loc.Config)
	snapshot := &kickoff.GitSnapshot{IntegrationBranch: policy.IntegrationBranch, BranchPrefix: policy.BranchPrefix}
	if exceptions := facts.input.Exceptions; exceptions != (branch.Exceptions{}) {
		snapshot.Exceptions = &kickoff.GitExceptions{Prefix: exceptions.Prefix, Branch: exceptions.Branch, Base: exceptions.Base}
	}

	archive, refusal := archiveBranchOf(facts, plan)
	if refusal != nil {
		return nil, &branch.RefusalError{Refusal: *refusal}
	}
	for index, repo := range plan.Repositories {
		if repo.RelPath == "" {
			continue
		}
		entry := kickoff.GitSnapshotRepo{
			Path:              repo.RelPath,
			Name:              repo.Name,
			Role:              string(repo.Role),
			IntegrationBranch: repo.IntegrationBranch,
			Base:              repo.IntegrationBranch,
		}
		if repo.BaseRef != "" {
			entry.Base = repo.BaseRef
		}
		for _, planned := range repo.Branches {
			entry.Branches = append(entry.Branches, kickoff.GitSnapshotBranch{Name: planned.Name, Kind: string(planned.Kind), Role: planned.RoleID})
			if snapshot.ChangeBranch == "" && planned.Kind == branch.KindChange {
				snapshot.ChangeBranch = planned.Name
			}
		}
		if index == 0 && archive != "" {
			entry.Branches = append(entry.Branches, kickoff.GitSnapshotBranch{Name: archive, Kind: string(branch.KindArchive)})
		}
		snapshot.Repositories = append(snapshot.Repositories, entry)
	}
	return snapshot, nil
}

// archiveBranchOf es el nombre de la rama de archivado del repositorio del planning home, o la
// cadena vacía donde no existe (un multirepo, donde el archive se hace in situ, REQ-25.5). Es el
// plan de archivado de `ensure --archive`, que comparte con `status` (archivePlanInput): el nombre
// deriva de la rama del cambio registrada o, sin ella, de la política.
func archiveBranchOf(facts changeFacts, plan branch.ChangePlan) (string, *branch.Refusal) {
	if len(plan.Repositories) == 0 || plan.Repositories[0].Role != branch.RepoChangeBranch {
		return "", nil
	}
	archive, refusal := branch.Plan(archivePlanInput(facts, branch.Exceptions{}))
	switch {
	case refusal != nil && refusal.Code == branch.CodeArchiveBranchNotApplicable:
		return "", nil
	case refusal != nil:
		return "", refusal
	case len(archive.Repositories) != 1 || len(archive.Repositories[0].Branches) != 1:
		return "", &branch.Refusal{
			Code:         branch.CodeConfigInvalid,
			Message:      "el plan de archivado no tiene exactamente una rama: no se puede escribir la instantánea",
			Continuation: fmt.Sprintf("comprueba el estado del cambio con `axiom branch status --change %s --cwd %s` y repite la orden", facts.input.Change, pathquote.Quote(facts.input.Master)),
		}
	}
	return archive.Repositories[0].Branches[0].Name, nil
}
