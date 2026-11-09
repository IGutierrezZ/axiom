package engine

import (
	"context"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

// HandOff calcula, solo con lecturas, la entrega de la salida 2 de una parada («aprobar, publicar y
// parar», REQ-26.4, REQ-26.5, REQ-27.12, DD-12): el destino de la publicación de los documentos
// (rama, `prTarget` e invocación de `axiom sdd publish`), el push de código de los demás
// repositorios del rol y lo que ejecuta quien recoja el cambio. Se construye sobre el mismo camino
// de lectura que `Status` (`gather` y la planificación pura): no toma el cerrojo del cambio, no
// escribe el registro ni el ledger y no hace `fetch`, y el puerto de git no tiene ningún método que
// publique, así que nunca ejecuta `commit` ni `push` (REQ-25.1).
//
// No comprueba qué está confirmado ni lee el estado del árbol de trabajo: REQ-26.4 prohíbe que
// `gate record` inspeccione el trabajo del rol. Los documentos pendientes los confirma
// `axiom sdd publish`, y el código sin subir de un role-apply lo informa él como
// `code_commits_unpublished`, sin deshacer la aprobación. Un rechazo no ha tocado nada.
func (e *Engine) HandOff(ctx context.Context, req branch.HandOffRequest) (branch.HandOffPlan, *branch.Refusal) {
	// Las continuaciones de los rechazos del plan repiten `axiom branch <operation>`: `status` es la
	// orden real que describe el mismo plan.
	const operation = "status"

	facts, refusal := e.gather(ctx, operation, req.CWD, req.Change, branch.LaneSDD)
	if refusal != nil {
		return branch.HandOffPlan{}, refusal
	}
	plan, refusal := branch.Plan(facts.input)
	if refusal != nil {
		return branch.HandOffPlan{}, refusal
	}

	index, _ := planningRepository(plan, req.ChangeRoot)
	if index < 0 {
		return branch.HandOffPlan{}, branch.PlanningHomeOutsideRepository(facts.input.Master, plan.PlanningHome, plan.Repositories[0])
	}
	planning := plan.Repositories[index]

	publish, refusal := planningPublish(req, planning, facts.input.Master)
	if refusal != nil {
		return branch.HandOffPlan{}, refusal
	}

	var deliveries []branch.HandOffDelivery
	if req.Role != "" {
		for i, repo := range plan.Repositories {
			if i == index || len(repo.Branches) == 0 || !touchesRole(repo.Roles, req.Role) {
				continue
			}
			other, refusal := e.codeDelivery(ctx, operation, req, repo, facts.worktrees[repo.GitRoot])
			if refusal != nil {
				return branch.HandOffPlan{}, refusal
			}
			if other != nil {
				deliveries = append(deliveries, *other)
			}
		}
	}

	return branch.HandOffPlan{
		Topology:   facts.input.Topology,
		Publish:    publish,
		Deliveries: deliveries,
		Resume:     branch.HandOffResume(req.Change),
	}, nil
}

// planningRepository devuelve la posición, en el plan, del repositorio cuya raíz git contiene la
// carpeta del cambio, y la ruta de esa carpeta relativa a la raíz, con «/». Es -1 si ninguno la
// contiene.
func planningRepository(plan branch.ChangePlan, changeRoot string) (int, string) {
	for i, repo := range plan.Repositories {
		if rel, inside := relativeWithin(repo.GitRoot, changeRoot); inside {
			return i, rel
		}
	}
	return -1, ""
}

// touchesRole dice si el rol está entre los que tocan un repositorio, sin distinguir mayúsculas.
func touchesRole(roles []string, role string) bool {
	return slices.ContainsFunc(roles, func(candidate string) bool { return strings.EqualFold(candidate, role) })
}

// planningPublish compone el destino de la publicación del repositorio del planning home. En el
// repositorio de specs de un multirepo (se trabaja in situ) se publica la rama de integración y no
// hay `prTarget`; en cualquier otro, la rama actual, que debe ser la del cambio o una de sus ramas
// de rol, con el `prTarget` que el plan le da.
func planningPublish(req branch.HandOffRequest, repo branch.RepoPlan, master string) (branch.HandOffPublish, *branch.Refusal) {
	current, detached, err := gitcommon.ReadHead(repo.GitRoot)
	if err != nil {
		detached, current = true, ""
	}
	if detached {
		current = ""
	}
	publish := branch.HandOffPublish{Remote: branch.PublishRemote}

	if repo.Role == branch.RepoSpecsInPlace {
		if current != repo.IntegrationBranch {
			return branch.HandOffPublish{}, branch.HandOffWrongBranch(req.Change, master, repo, current, []string{repo.IntegrationBranch})
		}
		publish.Branch, publish.InPlace = repo.IntegrationBranch, true
	} else {
		var expected []string
		for _, planned := range repo.Branches {
			expected = append(expected, planned.Name)
			if current != "" && planned.Name == current {
				publish.Branch, publish.PRTarget = planned.Name, planned.PRTarget
			}
		}
		if publish.Branch == "" {
			return branch.HandOffPublish{}, branch.HandOffWrongBranch(req.Change, master, repo, current, expected)
		}
	}
	publish.Invocation = branch.PublishInvocation(req.CWD, req.Change, publish.Branch)
	return publish, nil
}

// codeDelivery compone la entrega de código de un repositorio del rol distinto del planning home
// (en un multirepo, un repositorio de código): su rama de rol si el plan la tiene y, si no, la del
// cambio. Solo queda publicarla, y es de la persona: Axiom no sube código. No comprueba si el
// trabajo está confirmado (REQ-26.4). Devuelve nil si la rama no existe: sin rama no hay trabajo que
// entregar.
func (e *Engine) codeDelivery(ctx context.Context, operation string, req branch.HandOffRequest, repo branch.RepoPlan, worktrees []branch.WorktreeEntry) (*branch.HandOffDelivery, *branch.Refusal) {
	planned := repo.Branches[0]
	for _, candidate := range repo.Branches {
		if candidate.Kind == branch.KindRole && strings.EqualFold(candidate.RoleID, req.Role) {
			planned = candidate
			break
		}
	}
	exists, err := e.git.BranchExists(ctx, repo.GitRoot, planned.Name)
	if err != nil {
		return nil, gitRefusal(operation, req.CWD, "comprobar si existe la rama "+planned.Name, err)
	}
	if !exists {
		return nil, nil
	}

	dir := repo.GitRoot
	for _, entry := range worktrees {
		if entry.Branch == planned.Name && !entry.Detached {
			dir = entry.Path
			break
		}
	}
	return &branch.HandOffDelivery{
		Repository: repo.Name,
		GitRoot:    dir,
		Branch:     planned.Name,
		PRTarget:   planned.PRTarget,
		Steps:      []string{branch.HandOffPushStep(dir, planned.Name, true)},
	}, nil
}
