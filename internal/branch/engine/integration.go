package engine

import (
	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// PlanningIntegration devuelve la rama de integración resuelta del repositorio del planning home
// (design D-22, REQ-25.23): la de `git.integration_branch`, o la de su entrada de
// `repositories:` si la declara, o `main` solo como valor por defecto de branch.ResolvePolicy.
// Es la rama contra la que se comprueba que un commit llegó a integración (`sdd gate record
// --evidence-kind pr_merged`) y la que informa el estado de sincronización de especificaciones.
//
// Solo lee la configuración ya cargada en loc, sin lanzar procesos: el repositorio del planning
// home es el que contiene `loc.WorkspaceRoot` (la raíz del espacio de trabajo es donde vive
// `openspec/`: la carpeta maestra en un monorepo, el repositorio de especificaciones en el
// multirepo). Fuera de la carpeta maestra solo rige la política del workspace, como en el
// planificador.
func PlanningIntegration(loc workspace.Location) string {
	policy := branch.ResolvePolicy(loc.Config)
	if relative, inside := relativeWithin(loc.Master, loc.WorkspaceRoot); inside {
		return policy.For(relative).IntegrationBranch
	}
	return policy.IntegrationBranch
}
