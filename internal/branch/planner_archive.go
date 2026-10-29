package branch

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// El plan de archivado (REQ-25.45, design D-25): `ensure --change <c> --archive` planifica
// aparte la rama de archivado `<prefijo>/<cambio>-archive` del repositorio del planning home.
// No es una rama de rol ni sustituye a la del cambio: el plan ordinario no la incluye y este no
// incluye ninguna otra, de modo que archivar nunca mueve ni crea las ramas del cambio ni las
// de rol.

// archiveSuffix es lo que la rama de archivado añade al nombre de la rama del cambio.
const archiveSuffix = "-" + archiveRole

// ensureOperation es la orden de `axiom branch` cuyas continuaciones deben repetir `--archive`
// cuando lo que se planifica es el archivado.
const ensureOperation = "ensure"

// archiveOperation devuelve la orden que repiten las continuaciones del plan de archivado: la
// de `ensure` lleva `--archive` para que repetirla no cree la rama del cambio. Las demás
// órdenes (`status`, que solo lee) y la que ya la lleva quedan como están.
func archiveOperation(operation string) string {
	if operation == ensureOperation {
		return operation + " --" + archiveRole
	}
	return operation
}

// planArchive calcula el plan de archivado de un cambio (design D-25, pasos 1 a 4):
//
//  1. aplicabilidad: el carril ODD no tiene archive y en `multirepo` el archive se hace in
//     situ en el repositorio de specs, así que ninguno tiene rama de archivado
//     (`archive_branch_not_applicable`);
//  2. banderas: con `--archive` solo se admite `--prefix`; `--branch` y `--base` son
//     `plan_conflict` (supuesto S-16); ambos pasos son CheckArchiveRequest;
//  3. nombre: con prefijo, `<prefijo>/<cambio>-archive`; sin él, `<rama del cambio
//     registrada>-archive`, y sin rama registrada la compone la política;
//  4. ruta: `<raíz>/<cambio>-archive/<repo>`, con las mismas comprobaciones de colocación y de
//     longitud que el plan ordinario.
//
// La rama sale de la rama de integración (lo resuelve el motor, design D-06): el plan no lleva
// `BaseRef`, la excepción `--base` no existe para el archivado.
func planArchive(in PlanInput) (ChangePlan, *Refusal) {
	in.Operation = archiveOperation(in.Operation)
	if refusal := ValidateChangeName(in.Operation, in.Master, in.Change); refusal != nil {
		return ChangePlan{}, refusal
	}
	if refusal := CheckArchiveRequest(in); refusal != nil {
		return ChangePlan{}, refusal
	}
	if refusal := validateExceptions(in); refusal != nil {
		return ChangePlan{}, refusal
	}

	facts := in.PlanningRepo
	name := repoName(facts)
	effective := repoPolicy(in.Policy, facts)
	if refusal := validatePolicyValue(in, name, "integration_branch", effective.IntegrationBranch, gitref.ValidRefName); refusal != nil {
		return ChangePlan{}, refusal
	}
	archiveBranch, refusal := archiveBranchName(in, name, effective)
	if refusal != nil {
		return ChangePlan{}, refusal
	}
	if registered := in.ArchiveBranch; registered != "" && registered != archiveBranch {
		return ChangePlan{}, ArchiveBranchConflict(in.Operation, in.Change, in.Master, in.Lane, name, facts.GitRoot, registered, archiveBranch)
	}

	// Un rol `archive` colisionaría con esta rama (S-15): se rechaza igual que al planificar las
	// ramas de rol, nombrando la rama de archivado que se pide.
	roles := touchingRoles(facts, sealedRoles(in))
	if len(roles) >= 2 {
		for _, role := range roles {
			if refusal := validateRoleName(in, role, strings.TrimSuffix(archiveBranch, archiveSuffix)); refusal != nil {
				return ChangePlan{}, refusal
			}
		}
	}

	repo := RepoPlan{
		Name:              name,
		GitRoot:           facts.GitRoot,
		CommonDir:         facts.CommonDir,
		RelPath:           facts.RelPath,
		Role:              repositoryRole(in.Topology, true),
		Roles:             roles,
		IntegrationBranch: effective.IntegrationBranch,
		Branches: []BranchPlan{{
			Name:     archiveBranch,
			Kind:     KindArchive,
			PRTarget: effective.IntegrationBranch,
			Worktree: plannedWorktree(in, in.Change+archiveSuffix, name),
		}},
	}
	repos := []RepoPlan{repo}
	if refusal := checkPlacement(in, repos); refusal != nil {
		return ChangePlan{}, refusal
	}
	if refusal := checkPathLength(in, repos); refusal != nil {
		return ChangePlan{}, refusal
	}
	plan := ChangePlan{
		Lane:         in.Lane,
		Change:       in.Change,
		WorktreeRoot: in.WorktreeRoot,
		Repositories: repos,
	}
	plan.WorkspaceRoot, plan.PlanningHome = plannedWorkspace(in, repo)
	return plan, nil
}

// CheckArchiveRequest comprueba, sin leer nada, lo que decide si la petición de archivado tiene
// sentido (design D-25, pasos 1 y 2): la aplicabilidad, primero (`archive_branch_not_applicable`
// en el carril ODD, que no tiene archive, y en `multirepo`, donde el archive se hace in situ en
// la rama de integración del repositorio de specs, REQ-25.5), y las banderas después: con
// `--archive` solo se admite `--prefix`, y `--branch` o `--base` son `plan_conflict` (S-16).
// El motor lo llama antes de resolver el modo de trabajo, de modo que una petición que no se
// puede atender no pregunta por el modo; el planificador lo repite.
func CheckArchiveRequest(in PlanInput) *Refusal {
	in.Operation = archiveOperation(in.Operation)
	switch {
	case in.Lane == LaneODD:
		return ArchiveBranchNotApplicable(in, "el carril ODD no tiene archive y, por tanto, tampoco rama de archivado")
	case in.Topology == workspace.TopologyMultirepo:
		integration := repoPolicy(in.Policy, in.PlanningRepo).IntegrationBranch
		return ArchiveBranchNotApplicable(in, fmt.Sprintf("en un multirepo no existe rama de archivado: el archive se hace in situ en la rama de integración «%s» del repositorio de specs", integration))
	case in.Exceptions.Branch != "" || in.Exceptions.Base != "":
		return ArchiveOptionsConflict(in)
	}
	return nil
}

// archiveBranchName compone el nombre de la rama de archivado (design D-25, paso 3): con el
// prefijo de la invocación, `<prefijo>/<cambio>-archive`, sin tocar el prefijo del cambio; sin
// él, la rama de archivado que el registro ya guarda y, sin ella, la rama del cambio registrada
// más `-archive` o, si no hay ninguna, la que compondría la política (`<prefijo>/<cambio>`). El
// prefijo de la política solo se valida cuando se usa.
func archiveBranchName(in PlanInput, repository string, effective RepoPolicy) (string, *Refusal) {
	var archive string
	switch {
	case in.Exceptions.Prefix != "":
		archive = in.Exceptions.Prefix + "/" + in.Change + archiveSuffix
	case in.ArchiveBranch != "":
		archive = in.ArchiveBranch
	case in.ChangeBranch != "":
		archive = in.ChangeBranch + archiveSuffix
	default:
		if refusal := validatePolicyValue(in, repository, "branch_prefix", effective.BranchPrefix, gitref.ValidPrefix); refusal != nil {
			return "", refusal
		}
		archive = effective.BranchPrefix + "/" + in.Change + archiveSuffix
	}
	if err := gitref.ValidRefName(archive); err != nil {
		return "", invalidName("branch", "el nombre de la rama de archivado", archive, err,
			fmt.Sprintf("repite la orden con un prefijo válido para la rama de archivado: `axiom branch %s --change %s --prefix <prefijo> --cwd %s`",
				in.Operation, in.Change, pathquote.Quote(in.Master)))
	}
	return archive, nil
}
