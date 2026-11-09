package engine

import (
	"context"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// PublicationTarget calcula, solo con lecturas, el destino de la publicación de los documentos de un
// cambio (REQ-27.7, DD-07): el repositorio del planning home, su raíz git, la rama que se publica y
// la de integración. Se construye sobre el mismo camino de lectura que `Status` y `HandOff`
// (`gather` y la planificación pura): no toma el cerrojo del cambio, no escribe el registro ni el
// ledger y no hace `fetch`, y el puerto de git no tiene ningún método que publique, así que nunca
// ejecuta `commit` ni `push` (REQ-25.1). Lo consultan la pregunta de la puerta, `gate record` y
// `axiom sdd publish`, de modo que la rama que se nombra es exactamente la que se sube.
//
// Reglas de rama (S9), con la rama actual leída del repositorio del planning home:
//
//  1. HEAD desacoplado: `detached_head`.
//  2. Repositorio de specs de un `multirepo` (se trabaja in situ, también tras archivar): la rama
//     permitida es la de integración; otra es `wrong_branch`.
//  3. Si no, la rama de integración es `integration_branch`; una rama que no es ninguna de las
//     planificadas del repositorio (la del cambio y las de rol; con Archive, solo la de archivado) es
//     `wrong_branch`, y su continuación nombra el flag `--archive` cuando es lo que falta o sobra.
//  4. Con Branch no vacío y distinto de la rama permitida: `wrong_branch`. Así la publicación queda
//     atada a la rama que nombró la autorización.
func (e *Engine) PublicationTarget(ctx context.Context, req branch.PublicationRequest) (branch.PublicationTarget, *branch.Refusal) {
	// Las continuaciones de los rechazos del plan repiten `axiom branch <operation>`: `status` es la
	// orden real que describe el mismo plan.
	const operation = "status"

	facts, refusal := e.gather(ctx, operation, req.CWD, req.Change, branch.LaneSDD)
	if refusal != nil {
		return branch.PublicationTarget{}, refusal
	}
	ordinary, refusal := branch.Plan(facts.input)
	if refusal != nil {
		return branch.PublicationTarget{}, refusal
	}

	// Un multirepo no tiene rama de archivado: el archivado se hace in situ, en la integración del
	// repositorio de specs, y el plan ordinario ya lo describe.
	archiveApplies := facts.input.Topology != workspace.TopologyMultirepo
	plan, usesArchive := ordinary, false
	if req.Archive && archiveApplies {
		archiveFacts := facts
		if refusal := useArchivePlan(&archiveFacts, EnsureRequest{}); refusal != nil {
			return branch.PublicationTarget{}, refusal
		}
		archived, refusal := branch.Plan(archiveFacts.input)
		if refusal != nil {
			return branch.PublicationTarget{}, refusal
		}
		plan, usesArchive = archived, true
	}

	index, planningRel := planningHomeRepository(plan, facts.location.PlanningHome)
	if index < 0 {
		return branch.PublicationTarget{}, branch.PlanningHomeOutsideRepository(facts.input.Master, plan.PlanningHome, plan.Repositories[0])
	}
	repo := plan.Repositories[index]

	current, detached, err := gitcommon.ReadHead(repo.GitRoot)
	if err != nil || detached || current == "" {
		return branch.PublicationTarget{}, branch.PublicationDetachedHead(req.Change, facts.input.Master, repo)
	}

	target := branch.PublicationTarget{
		Repository:        repo.Name,
		GitRoot:           repo.GitRoot,
		Branch:            current,
		IntegrationBranch: repo.IntegrationBranch,
		Topology:          facts.input.Topology,
		PlanningRel:       planningRel,
	}

	if repo.Role == branch.RepoSpecsInPlace {
		if current != repo.IntegrationBranch {
			return branch.PublicationTarget{}, branch.PublicationWrongBranch(req.Change, facts.input.Master, repo, current, []string{repo.IntegrationBranch}, req.Archive, "")
		}
		target.InPlace = true
	} else {
		if current == repo.IntegrationBranch {
			return branch.PublicationTarget{}, branch.PublicationIntegrationBranch(req.Change, facts.input.Master, repo, current)
		}
		planned, expected := plannedBranch(repo, current)
		if planned == nil {
			hint := e.archiveFlagHint(facts, ordinary, usesArchive, archiveApplies, current)
			return branch.PublicationTarget{}, branch.PublicationWrongBranch(req.Change, facts.input.Master, repo, current, expected, usesArchive, hint)
		}
		target.PRTarget = planned.PRTarget
	}

	if req.Branch != "" && req.Branch != target.Branch {
		return branch.PublicationTarget{}, branch.PublicationWrongBranch(req.Change, facts.input.Master, repo, current, []string{req.Branch}, usesArchive, "")
	}
	return target, nil
}

// planningHomeRepository devuelve la posición, en el plan, del repositorio cuya raíz git contiene el
// planning home, y la ruta de éste relativa a la raíz, con «/» y vacía si es la propia raíz. Es -1 si
// ninguno lo contiene.
func planningHomeRepository(plan branch.ChangePlan, planningHome string) (int, string) {
	for i, repo := range plan.Repositories {
		if rel, inside := relativeWithin(repo.GitRoot, planningHome); inside {
			if rel == "." {
				rel = ""
			}
			return i, rel
		}
	}
	return -1, ""
}

// plannedBranch devuelve la rama planificada del repositorio que se llama current y, si no hay
// ninguna, los nombres de las que el plan espera.
func plannedBranch(repo branch.RepoPlan, current string) (*branch.BranchPlan, []string) {
	var expected []string
	for i := range repo.Branches {
		if repo.Branches[i].Name == current {
			return &repo.Branches[i], nil
		}
		expected = append(expected, repo.Branches[i].Name)
	}
	return nil, expected
}

// archiveFlagHint nombra el flag `--archive` cuando la rama actual es una rama del plan contrario:
// la de archivado sin `--archive`, o una del cambio con él. Vacío en cualquier otro caso, también si
// el plan contrario no se puede calcular.
func (e *Engine) archiveFlagHint(facts changeFacts, ordinary branch.ChangePlan, usesArchive, archiveApplies bool, current string) string {
	if usesArchive {
		for _, repo := range ordinary.Repositories {
			if slices.ContainsFunc(repo.Branches, func(planned branch.BranchPlan) bool { return planned.Name == current }) {
				return "esta es una rama del cambio y no la de archivado: quita `--archive`"
			}
		}
		return ""
	}
	if !archiveApplies {
		return ""
	}
	archiveFacts := facts
	if useArchivePlan(&archiveFacts, EnsureRequest{}) != nil {
		return ""
	}
	archived, refusal := branch.Plan(archiveFacts.input)
	if refusal != nil {
		return ""
	}
	for _, repo := range archived.Repositories {
		if slices.ContainsFunc(repo.Branches, func(planned branch.BranchPlan) bool { return planned.Name == current }) {
			return "esta es la rama de archivado: añade `--archive`"
		}
	}
	return ""
}
