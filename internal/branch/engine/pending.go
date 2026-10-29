package engine

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// cumulativeSink reconoce, por nombre, los sumideros `cumulative-<rol>` de los
// verificadores acumulativos: viven en `changes/` pero no son cambios (REQ-25.20).
var cumulativeSink = regexp.MustCompile(`^cumulative-[A-Za-z0-9._-]+$`)

// isPendingChange dice si una carpeta de `changes/` cuenta como cambio: no lo son
// `archive`, `e2e-cumulative` ni los `cumulative-<rol>`, excluidos por nombre (REQ-25.20).
func isPendingChange(name string) bool {
	if name == "archive" || name == "e2e-cumulative" {
		return false
	}
	return !cumulativeSink.MatchString(name)
}

// PendingArchive lista los cambios pendientes de archivar (REQ-25.20, design D-17): las
// carpetas de `<planning home>/changes` presentes en el árbol de la rama de integración
// del repositorio del planning home, es decir, ya fusionadas y aún sin archivar. Es
// información, no una comprobación: nunca falla, nunca bloquea y no tiene código de fallo.
//
// La evaluación se hace sobre la referencia de integración local de REQ-25.10
// (`origin/<integración>` si existe en local y, si no, la rama local), leyendo su árbol
// con `git ls-tree` sin cambiar de rama ni usar la red. Los pendientes salen ordenados
// por nombre, cada uno con la orden para archivarlo.
//
// El resultado distingue tres casos: no nil y vacío, se evaluó y no hay pendientes (también
// el repositorio de specs de un `multirepo`, donde los cambios activos viven en la rama de
// integración de forma normal y la lista no aplica); no nil con elementos; y nil, no se
// pudo evaluar (sin referencia de integración o ante un fallo de git), con un aviso que
// dice por qué. Los avisos son siempre una frase en castellano.
func (e *Engine) PendingArchive(ctx context.Context, loc workspace.Location) ([]branch.PendingChange, []string) {
	if topologyOf(loc) == workspace.TopologyMultirepo {
		return []branch.PendingChange{}, nil
	}
	omitted := func(format string, args ...any) ([]branch.PendingChange, []string) {
		return nil, []string{"no se evalúan los cambios pendientes de archivar: " + fmt.Sprintf(format, args...)}
	}

	root, err := e.git.TopLevel(ctx, loc.WorkspaceRoot)
	if err != nil {
		return omitted("no se pudo localizar la raíz del repositorio de %s: %v", loc.WorkspaceRoot, err)
	}
	relative, inside := relativeWithin(root, loc.PlanningHome)
	if !inside {
		return omitted("el planning home %s queda fuera del repositorio %s", loc.PlanningHome, root)
	}
	integration := integrationBranchOf(loc, root)
	base, found, err := e.resolveBase(ctx, branch.RepoPlan{GitRoot: root, IntegrationBranch: integration})
	if err != nil {
		return omitted("no se pudo resolver la rama de integración «%s» de %s: %v", integration, filepath.Base(root), err)
	}
	if !found {
		return omitted("no existe la rama de integración «%s» de %s, ni en local ni como referencia remota local", integration, filepath.Base(root))
	}

	// Se lee el commit ya resuelto, no el nombre de la rama: es la misma referencia aunque
	// el nombre sea ambiguo.
	changesPath := path.Join(relative, "changes")
	dirs, _, err := e.git.TreeDirs(ctx, root, base.sha, changesPath)
	if err != nil {
		return omitted("no se pudo leer %s en %s: %v", changesPath, base.ref, err)
	}
	pending := []branch.PendingChange{}
	for _, dir := range dirs {
		if isPendingChange(dir) {
			pending = append(pending, branch.NewPendingChange(dir))
		}
	}
	slices.SortFunc(pending, func(a, b branch.PendingChange) int { return strings.Compare(a.Change, b.Change) })
	return pending, nil
}

// integrationBranchOf es la rama de integración del repositorio de raíz git root según la
// política del proyecto (REQ-25.3): la entrada de `repositories:` o, sin ella o con el
// repositorio fuera de la carpeta maestra, la del workspace.
func integrationBranchOf(loc workspace.Location, root string) string {
	policy := branch.ResolvePolicy(loc.Config)
	if relative, inside := relativeWithin(loc.Master, root); inside {
		return policy.For(relative).IntegrationBranch
	}
	return policy.IntegrationBranch
}

// reportPendingArchive añade a un resultado de `status --change` si el cambio pedido está
// pendiente de archivar y, si lo está, la orden para archivarlo (REQ-25.15, REQ-25.20). El
// carril ODD no archiva, así que no lo informa. Si no se pudo evaluar, deja el campo
// omitido y añade el aviso.
func (e *Engine) reportPendingArchive(ctx context.Context, result *branch.Result, location workspace.Location, change string, lane branch.Lane) {
	if lane != branch.LaneSDD {
		return
	}
	pending, warnings := e.PendingArchive(ctx, location)
	result.Warnings = append(result.Warnings, warnings...)
	if pending == nil {
		return
	}
	index := slices.IndexFunc(pending, func(candidate branch.PendingChange) bool { return candidate.Change == change })
	isPending := index >= 0
	result.PendingArchive = &branch.PendingArchive{Requested: &isPending}
	if isPending {
		result.PendingArchiveContinuation = pending[index].Continuation
	}
}
