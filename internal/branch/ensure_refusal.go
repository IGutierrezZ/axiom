package branch

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Rechazos de la comprobación sin efectos de `ensure` (design D-05, fase 3). Cada uno
// nombra una orden ejecutable con las rutas citadas por pathquote.Quote (tasks.md,
// regla 11) y repite la propia `ensure` con el cambio, que completa lo pendiente. Nacen
// con su primer llamador: `Engine.Ensure`.

// repeatContinuation es la orden que repite la operación con el cambio y la carpeta
// maestra.
func repeatContinuation(operation, change, master string) string {
	return fmt.Sprintf("axiom branch %s --change %s --cwd %s", operation, change, pathquote.Quote(master))
}

// BaseNotFound es el rechazo `base_not_found` (REQ-25.10): el repositorio no tiene la
// referencia de la que saldrían las ramas del cambio. Con `--base` fijada, nombra esa
// referencia; si no, la rama de integración que resuelve la política, que no existe ni
// en local ni como referencia remota local. Un repositorio sin commits cae aquí.
func BaseNotFound(operation, change, master string, repo RepoPlan) *Refusal {
	expected := repo.IntegrationBranch
	message := fmt.Sprintf("no existe la rama de integración «%s» del repositorio %s (%s) ni en local ni como referencia remota local origin/%s: no hay base desde la que crear las ramas del cambio",
		expected, repo.Name, repo.GitRoot, expected)
	if repo.BaseRef != "" {
		expected = repo.BaseRef
		message = fmt.Sprintf("la base «%s» fijada para el cambio no existe en el repositorio %s (%s): no hay base desde la que crear las ramas del cambio",
			expected, repo.Name, repo.GitRoot)
	}
	return &Refusal{
		Code:    CodeBaseNotFound,
		Message: message,
		Continuation: fmt.Sprintf("crea la rama «%s» en el repositorio (por ejemplo, `git -C %s branch %s <commit>`; un repositorio sin commits necesita antes el primero) o corrige git.integration_branch en axiom.yaml, y repite `%s`",
			expected, pathquote.Quote(repo.GitRoot), expected, repeatContinuation(operation, change, master)),
		Details: map[string]string{"repository": repo.GitRoot, "branch": expected},
	}
}

// BranchCheckedOutElsewhere es el rechazo `branch_checked_out_elsewhere` (REQ-25.7): la
// rama planificada ya está desplegada en un worktree que el plan y el registro local no
// conocen. Nombra la ruta de ese worktree. inMainCheckout dice si es el propio checkout
// principal, que no se retira: se cambia de rama; mainCheckout es el checkout principal
// del repositorio, desde el que se retiran los demás.
func BranchCheckedOutElsewhere(operation, change, master string, repo RepoPlan, name, worktree, mainCheckout string, inMainCheckout bool) *Refusal {
	fix := fmt.Sprintf("retira ese worktree con `git -C %s worktree remove %s` (o sigue trabajando allí)", pathquote.Quote(mainCheckout), pathquote.Quote(worktree))
	if inMainCheckout {
		fix = fmt.Sprintf("cambia ese checkout a otra rama con `git -C %s switch %s`", pathquote.Quote(worktree), repo.IntegrationBranch)
	}
	return &Refusal{
		Code: CodeBranchCheckedOutElsewhere,
		Message: fmt.Sprintf("la rama %s del repositorio %s ya está desplegada en %s, un worktree que el plan no conoce: no se crea otro worktree para ella",
			name, repo.Name, worktree),
		Continuation: fmt.Sprintf("%s y repite `%s`", fix, repeatContinuation(operation, change, master)),
		Details:      map[string]string{"branch": name, "worktree": worktree, "repository": repo.GitRoot},
	}
}

// PlannedPathConflict es el rechazo `plan_conflict` de una ruta de worktree planificada
// que no está libre para su rama (design D-05, fase 3): la ocupa un worktree con otra
// rama, un worktree cuyo directorio ya no existe o un directorio con contenido. message
// explica el motivo y fix, la acción que lo resuelve; ambos los compone quien comprueba.
func PlannedPathConflict(operation, change, master, name, worktree, message, fix string) *Refusal {
	return &Refusal{
		Code:         CodePlanConflict,
		Message:      message,
		Continuation: fmt.Sprintf("%s y repite `%s`", fix, repeatContinuation(operation, change, master)),
		Details:      map[string]string{"branch": name, "worktree": worktree},
	}
}
