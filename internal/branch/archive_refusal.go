package branch

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Rechazos del plan de archivado (REQ-25.45, REQ-25.12, design D-25). Cada uno nombra una orden
// ejecutable con las rutas citadas por pathquote.Quote (tasks.md, regla 11).

// ArchiveBranchNotApplicable es el rechazo `archive_branch_not_applicable` de `ensure
// --archive` en un carril o una topología sin rama de archivado (REQ-25.45): el carril ODD, que
// no tiene archive, y `multirepo`, donde el archive se hace in situ en la rama de integración
// del repositorio de specs. reason explica cuál de los dos es. No se ha creado nada; la
// continuación es la orden que sí describe el estado del cambio en esa topología o carril.
func ArchiveBranchNotApplicable(in PlanInput, reason string) *Refusal {
	continuation := fmt.Sprintf("comprueba el estado del cambio con `axiom sdd status %s --cwd %s` y archívalo in situ en la rama de integración del repositorio de specs, sin rama de archivado",
		in.Change, pathquote.Quote(in.Master))
	if in.Lane == LaneODD {
		continuation = fmt.Sprintf("el archive solo existe en el carril sdd: consulta la rama de la feature con `axiom branch status --change %s --lane odd --cwd %s`",
			in.Change, pathquote.Quote(in.Master))
	}
	return &Refusal{
		Code:         CodeArchiveBranchNotApplicable,
		Message:      fmt.Sprintf("no se puede crear la rama de archivado del cambio %s: %s; no se ha creado ninguna rama ni worktree", in.Change, reason),
		Continuation: continuation,
		Details:      map[string]string{"change": in.Change, "lane": string(in.Lane), "topology": string(in.Topology)},
	}
}

// ArchiveOptionsConflict es el rechazo `plan_conflict` de `--branch` o `--base` con `--archive`
// (REQ-25.12, supuesto S-16): con `--archive` solo se admite `--prefix`, porque la rama de
// archivado sale siempre de la rama de integración y su nombre deriva de la rama del cambio
// registrada o del prefijo. Nombra las banderas que sobran; no se ha tocado nada y la
// continuación repite `ensure --archive` con el `--prefix` pedido, si lo hay.
func ArchiveOptionsConflict(in PlanInput) *Refusal {
	var flags []string
	if in.Exceptions.Branch != "" {
		flags = append(flags, "--branch")
	}
	if in.Exceptions.Base != "" {
		flags = append(flags, "--base")
	}
	return &Refusal{
		Code: CodePlanConflict,
		Message: fmt.Sprintf("con --archive solo se admite --prefix y se pidió %s: la rama de archivado sale siempre de la rama de integración y su nombre deriva de la rama del cambio registrada o de --prefix; no se ha creado ninguna rama ni worktree",
			strings.Join(flags, " y ")),
		Continuation: fmt.Sprintf("repite solo con las banderas que se admiten, `%s`", ensureInvocation(in.Operation, in.Change, in.Master, in.Lane, Exceptions{Prefix: in.Exceptions.Prefix})),
		Details:      map[string]string{"change": in.Change, "flags": strings.Join(flags, ",")},
	}
}
