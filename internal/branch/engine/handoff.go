package engine

import (
	"context"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// HandOff calcula, solo con lecturas, la entrega de la salida 2 de una parada («aprobar, subir y
// parar», REQ-26.4, REQ-26.5, DD-07): comprueba que lo que viaja en la PR está confirmado y compone,
// por repositorio, los pasos exactos que la persona ejecuta. Se construye sobre el mismo camino de
// lectura que `Status` (`gather` y la planificación pura): no toma el cerrojo del cambio, no escribe
// el registro ni el ledger y no hace `fetch`, y el puerto de git no tiene ningún método que
// publique, así que nunca ejecuta `commit` ni `push` (REQ-25.1).
//
// Qué cuenta como «confirmado» (DD-03). En spec, design y tasks, nada sucio bajo la carpeta del
// cambio en el repositorio del planning home, con la exención de `req.Exempt` (`gates.yaml`, que la
// orden modifica y se confirma después) y de las entradas de la raíz de la carpeta cuyo nombre
// empieza por punto (estado local que nunca se versiona: `.gates.lock`, `.axiom-instance`, los
// temporales `.gates-*.yaml` y `.kickoff-*.yaml`). Una carpeta entera sin seguimiento aparece como
// una sola entrada con «/» final y se rechaza. En `role-apply:<rol>`, además, el árbol de trabajo
// entero de cada repositorio del plan que toca el rol.
//
// El resultado lleva siempre MergeRequired: la fase siguiente solo empieza cuando lo entregado está
// integrado. Un rechazo no ha tocado nada.
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

	index, rel := planningRepository(plan, req.ChangeRoot)
	if index < 0 {
		return branch.HandOffPlan{}, branch.PlanningHomeOutsideRepository(facts.input.Master, plan.PlanningHome, plan.Repositories[0])
	}
	planning := plan.Repositories[index]

	// Los artefactos de la carpeta del cambio, y en role-apply el árbol entero del repositorio si el
	// rol lo toca.
	dirty, err := e.git.DirtyPaths(ctx, planning.GitRoot)
	if err != nil {
		return branch.HandOffPlan{}, gitRefusal(operation, req.CWD, "leer el estado del repositorio "+planning.GitRoot, err)
	}
	wholeTree := req.Role != "" && touchesRole(planning.Roles, req.Role)
	if pending := uncommittedArtifacts(dirty, rel, req.Exempt, wholeTree); len(pending) > 0 {
		return branch.HandOffPlan{}, branch.UncommittedArtifacts(req.Change, planning, pending)
	}

	gates := path.Join(rel, kickoff.GatesFileName)
	message := branch.HandOffCommitMessage(req.Gate, req.Change, req.HandedOffTo)
	delivery, refusal := planningDelivery(req, planning, gates, message, facts.input.Master)
	if refusal != nil {
		return branch.HandOffPlan{}, refusal
	}
	deliveries := []branch.HandOffDelivery{delivery}

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

	return branch.HandOffPlan{Topology: facts.input.Topology, MergeRequired: true, Deliveries: deliveries}, nil
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

// uncommittedArtifacts devuelve las rutas sin confirmar que impiden la entrega. changeRel es la
// carpeta del cambio relativa a la raíz git; con wholeTree cuenta todo el árbol de trabajo y no solo
// esa carpeta. Las rutas salen como las da git, con «/» y relativas a la raíz git.
func uncommittedArtifacts(dirty []string, changeRel string, exempt []string, wholeTree bool) []string {
	prefix := ""
	if changeRel != "." && changeRel != "" {
		prefix = changeRel + "/"
	}
	var pending []string
	for _, entry := range dirty {
		insideChange := strings.HasPrefix(entry, prefix)
		// Una carpeta sin seguimiento que contiene la del cambio cubre el prefijo.
		coversChange := strings.HasSuffix(entry, "/") && strings.HasPrefix(prefix, entry)
		if !wholeTree && !insideChange && !coversChange {
			continue
		}
		if insideChange && exemptEntry(strings.TrimPrefix(entry, prefix), exempt) {
			continue
		}
		pending = append(pending, entry)
	}
	return pending
}

// exemptEntry dice si una entrada sucia, relativa a la carpeta del cambio, no cuenta como sin
// confirmar: una ruta que el llamador declara exenta, o un fichero de la raíz de la carpeta cuyo
// nombre empieza por punto (estado local que nunca se versiona).
func exemptEntry(within string, exempt []string) bool {
	if strings.HasSuffix(within, "/") {
		return false
	}
	if !strings.Contains(within, "/") && strings.HasPrefix(within, ".") {
		return true
	}
	return slices.ContainsFunc(exempt, func(candidate string) bool {
		return path.Clean(filepath.ToSlash(candidate)) == within
	})
}

// planningDelivery compone la entrega del repositorio del planning home: confirmar `gates.yaml` y
// publicar. En el repositorio de specs de un multirepo (se trabaja in situ) se publica la rama de
// integración y no hay PR; en cualquier otro, la rama actual, que debe ser la del cambio o una de sus
// ramas de rol, con su destino.
func planningDelivery(req branch.HandOffRequest, repo branch.RepoPlan, gates, message, master string) (branch.HandOffDelivery, *branch.Refusal) {
	current, detached, err := gitcommon.ReadHead(repo.GitRoot)
	if err != nil {
		detached, current = true, ""
	}
	if detached {
		current = ""
	}
	delivery := branch.HandOffDelivery{Repository: repo.Name, GitRoot: repo.GitRoot}

	if repo.Role == branch.RepoSpecsInPlace {
		if current != repo.IntegrationBranch {
			return branch.HandOffDelivery{}, branch.HandOffWrongBranch(req.Change, master, repo, current, []string{repo.IntegrationBranch})
		}
		delivery.Branch, delivery.InPlace = repo.IntegrationBranch, true
		delivery.Steps = []string{
			branch.HandOffAddStep(repo.GitRoot, gates),
			branch.HandOffCommitStep(repo.GitRoot, message, gates),
			branch.HandOffPushStep(repo.GitRoot, repo.IntegrationBranch, false),
		}
		return delivery, nil
	}

	var expected []string
	for _, planned := range repo.Branches {
		expected = append(expected, planned.Name)
		if current != "" && planned.Name == current {
			delivery.Branch, delivery.PRTarget = planned.Name, planned.PRTarget
		}
	}
	if delivery.Branch == "" {
		return branch.HandOffDelivery{}, branch.HandOffWrongBranch(req.Change, master, repo, current, expected)
	}
	delivery.Steps = []string{
		branch.HandOffAddStep(repo.GitRoot, gates),
		branch.HandOffCommitStep(repo.GitRoot, message, gates),
		branch.HandOffPushStep(repo.GitRoot, delivery.Branch, true),
	}
	delivery.PR = &branch.HandOffPR{Head: delivery.Branch, Base: delivery.PRTarget}
	return delivery, nil
}

// codeDelivery compone la entrega de un repositorio del rol distinto del planning home (en un
// multirepo, un repositorio de código): su rama de rol si el plan la tiene y, si no, la del cambio.
// El trabajo del rol debe estar confirmado en el worktree donde está desplegada la rama; solo queda
// publicarla. Devuelve nil si la rama no existe: sin rama no hay trabajo que entregar.
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
	if dir != repo.GitRoot || currentBranchOf(repo.GitRoot) == planned.Name {
		dirty, err := e.git.DirtyPaths(ctx, dir)
		if err != nil {
			return nil, gitRefusal(operation, req.CWD, "leer el estado del worktree "+dir, err)
		}
		if len(dirty) > 0 {
			inspected := repo
			inspected.GitRoot = dir
			return nil, branch.UncommittedArtifacts(req.Change, inspected, dirty)
		}
	}
	return &branch.HandOffDelivery{
		Repository: repo.Name,
		GitRoot:    dir,
		Branch:     planned.Name,
		PRTarget:   planned.PRTarget,
		Steps:      []string{branch.HandOffPushStep(dir, planned.Name, true)},
		PR:         &branch.HandOffPR{Head: planned.Name, Base: planned.PRTarget},
	}, nil
}

// currentBranchOf es la rama actual de un repositorio, leída de ficheros; vacía con HEAD desacoplado
// o ilegible.
func currentBranchOf(gitRoot string) string {
	name, detached, err := gitcommon.ReadHead(gitRoot)
	if err != nil || detached {
		return ""
	}
	return name
}
