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

// ChangeLocked es el rechazo `plan_conflict` de un cambio que otra ejecución de `axiom
// branch` tiene bloqueado (design D-05): el bloqueo exclusivo por cambio impide dos `ensure`
// a la vez. No se ha tocado nada; la continuación es esperar y repetir.
func ChangeLocked(operation, change, master string) *Refusal {
	return &Refusal{
		Code:         CodePlanConflict,
		Message:      fmt.Sprintf("otra ejecución de `axiom branch` está modificando el cambio %s: no se ha tocado nada", change),
		Continuation: fmt.Sprintf("espera a que termine esa ejecución y repite `%s`", repeatContinuation(operation, change, master)),
		Details:      map[string]string{"change": change},
	}
}

// RegistryWriteFailed es el rechazo `config_invalid` de un registro local que no se puede
// bloquear o escribir (design D-07: es estado local derivado en el git common dir). path es
// el fichero o la carpeta que falló. Nada del plan queda a medias: si ya se aplicó, repetir
// `ensure` reconstruye el registro desde el estado de git.
func RegistryWriteFailed(operation, change, master, path string, err error) *Refusal {
	return &Refusal{
		Code:         CodeConfigInvalid,
		Message:      err.Error(),
		Continuation: fmt.Sprintf("comprueba que puedes escribir en %s (el git common dir del repositorio) y repite `%s`: reconstruye el registro desde el estado de git", pathquote.Quote(path), repeatContinuation(operation, change, master)),
		Details:      map[string]string{"path": path},
	}
}

// FetchFailed es el rechazo `fetch_failed` de `--fetch` (REQ-25.10, supuesto S-8): el
// `git fetch <remote>` de un repositorio del plan falló. Se lanza durante la comprobación
// sin efectos, de modo que no se ha creado ninguna rama, worktree ni registro. Nombra el
// repositorio y el remoto, resume el error de git y ofrece las dos salidas: reintentar con
// `--fetch` tras revisar el remoto, o seguir sin red con las referencias locales.
func FetchFailed(operation, change, master string, repo RepoPlan, remote string, err error) *Refusal {
	return &Refusal{
		Code: CodeFetchFailed,
		Message: fmt.Sprintf("`git fetch %s` falló en el repositorio %s (%s): %v; no se ha creado ninguna rama, worktree ni registro",
			remote, repo.Name, repo.GitRoot, err),
		Continuation: fmt.Sprintf("comprueba el remoto con `git -C %s remote -v` y repite `axiom branch %s --change %s --fetch --cwd %s`, o repite `%s` sin --fetch para trabajar con las referencias que ya hay en local",
			pathquote.Quote(repo.GitRoot), operation, change, pathquote.Quote(master), repeatContinuation(operation, change, master)),
		Details: map[string]string{"repository": repo.GitRoot, "remote": remote, "summary": err.Error()},
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
