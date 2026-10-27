package sddstatus

import (
	"context"
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// wrongBranchPrefix es el token estable con el que empieza el motivo tipado de la guarda B
// (REQ-25.42): el segundo motivo tipado junto a `blocked(edit_authority_missing)`.
const wrongBranchPrefix = "blocked(wrong_branch)"

// branchEngine es la variable de prueba del motor de ramas (design D-13): nula en producción, donde
// se usa engine.Default(). Es una variable y no una función, para que el ratchet de deadcode no la
// cuente. Una prueba que devuelve un motor nulo desactiva la guarda: el TestMain de este paquete lo
// hace para las pruebas anteriores a la guarda B, cuyos fixtures montan repositorios git en
// `master` o `main` sin kickoff.yaml (un cambio sin kickoff no está exento, supuesto S-7) y no tratan
// de la rama; las pruebas de la guarda la reactivan con engine.Default.
var branchEngine func() *engine.Engine

// branchQuery son los hechos del status que la guarda B necesita para consultar el veredicto de rama.
type branchQuery struct {
	workspaceRoot string
	changeRoot    string
	change        string
	// archivePhase es verdadero en la fase archive: todas las tareas cerradas y el verify-report hecho
	// (design D-13). Entonces el contexto del veredicto es `archive`.
	archivePhase bool
	// apply es verdadero cuando el estado de aplicación evalúa unidades de trabajo (apply): es la única
	// fase en la que un repositorio de código de un multirepo se comprueba.
	apply bool
	// plans son las fuentes de tareas del cambio: de ellas salen los repositorios de código que toca la
	// unidad de trabajo actual.
	plans []planText
}

// branchContext es el resultado de la guarda B: el veredicto desfavorable, si lo hay, y los avisos no
// bloqueantes (git no disponible, ruta fuera de git, `HEAD` ilegible).
type branchContext struct {
	wrongBranch *branch.Verdict
	notes       []string
}

// resolveBranchContext es la guarda B (REQ-25.42, design D-13): consulta el veredicto único de S8
// (`VerdictFor`) para el repositorio del planning home y, en la fase apply, para cada repositorio de
// código que toca la unidad de trabajo actual. Solo lee ficheros y nunca falla: lo que no se puede
// comprobar no bloquea y deja un aviso. Un repositorio de código no se evalúa fuera de apply porque el
// veredicto lo declara `not_applicable` con el consultante `status`.
func resolveBranchContext(q branchQuery) branchContext {
	eng := engine.Default()
	if branchEngine != nil {
		eng = branchEngine()
	}
	if eng == nil {
		return branchContext{}
	}
	phase := branch.ContextOrdinary
	if q.archivePhase {
		phase = branch.ContextArchive
	}
	consultant := branch.ConsultantStatus
	if q.apply {
		consultant = branch.ConsultantApply
	}

	var result branchContext
	seen := map[string]bool{}
	note := func(text string) {
		if !seen[text] {
			seen[text] = true
			result.notes = append(result.notes, text)
		}
	}
	consult := func(path string) {
		verdict, err := eng.VerdictFor(context.Background(), engine.VerdictQuery{
			Path:       path,
			Change:     q.change,
			Lane:       branch.LaneSDD,
			Context:    phase,
			Consultant: consultant,
		})
		switch {
		case err != nil:
			// El veredicto solo falla por configuración, kickoff o registro ilegibles; el status es
			// una lectura y no se rompe por ello. Los escritores sí fallan en cerrado (guarda A).
			note(fmt.Sprintf("note(branch_unchecked): no se pudo comprobar la rama del cambio %s: %v; no se bloquea", q.change, err))
		case verdict.Warning != "":
			note("note(branch_unchecked): " + verdict.Warning)
		}
		if err == nil && verdict.Kind == branch.VerdictWrongBranch && result.wrongBranch == nil {
			result.wrongBranch = &verdict
		}
	}

	consult(q.changeRoot)
	if q.apply && result.wrongBranch == nil {
		for _, root := range touchedCodeRepositories(q) {
			consult(root)
		}
	}
	return result
}

// touchedCodeRepositories son las raíces git de los repositorios de código del multirepo que tocan las
// rutas de la unidad de trabajo actual de cada fuente de tareas (REQ-25.42: un repositorio de código
// solo se evalúa en apply). Reutiliza el detector de la autoridad de edición sin raíces permitidas, que
// devuelve todo destino fuera del planning home, y se queda con las raíces de repositorios que
// pertenecen al mismo workspace: un repositorio ajeno o anidado en un monorepo no tiene rama del cambio
// que comprobar. Solo lee ficheros.
func touchedCodeRepositories(q branchQuery) []string {
	specs, err := workspace.Locate(q.changeRoot)
	if err != nil || topologyOf(specs) != workspace.TopologyMultirepo {
		return nil
	}
	planningRoot := gitRootOf(resolveExistingPath(q.workspaceRoot))
	seen := map[string]bool{}
	var roots []string
	for _, plan := range q.plans {
		blocking, _ := detectUnauthorizedEditRootsForCurrentWorkUnit(plan.text, q.workspaceRoot, nil)
		for _, root := range blocking {
			if seen[root] || gitRootOf(root) != root {
				continue
			}
			if planningRoot != "" && pathidentity.SameDirectory(root, planningRoot) {
				continue
			}
			if code, err := workspace.Locate(root); err != nil || !pathidentity.SameDirectory(code.PlanningHome, specs.PlanningHome) {
				continue
			}
			seen[root] = true
			roots = append(roots, root)
		}
	}
	return roots
}

// topologyOf es la topología del workspace; sin configuración rige la embebida.
func topologyOf(location workspace.Location) workspace.TopologyType {
	if location.Config != nil {
		return location.Config.Workspace.Topology
	}
	return workspace.TopologyMonorepoEmbedded
}

// wrongBranchReason es el motivo tipado `blocked(wrong_branch)`: nombra el repositorio, la rama actual
// y la esperada (Detail) e incluye la continuación exacta. Sin envoltorio de consentimiento: la
// continuación es una orden, no una concesión.
func wrongBranchReason(verdict branch.Verdict) string {
	return fmt.Sprintf("%s: %s; ejecuta `%s` y repite la orden", wrongBranchPrefix, verdict.Detail, verdict.Continuation)
}

// BlockedByWrongBranch dice si el status proyecta `blocked(wrong_branch)`. `sdd continue` lo consulta
// para no preparar el marcador de instancia con la rama equivocada (REQ-25.42, design D-11).
func BlockedByWrongBranch(status Status) bool {
	for _, reason := range status.BlockedReasons {
		if strings.HasPrefix(reason, wrongBranchPrefix) {
			return true
		}
	}
	return false
}
