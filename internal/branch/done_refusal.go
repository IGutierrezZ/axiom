package branch

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Rechazos de `axiom branch done` (REQ-25.16, REQ-25.17, design D-09). Cada uno se refiere a una
// rama del plan (los de git y los de la fusión) o a toda la operación (el fetch), nombra una orden
// ejecutable con las rutas citadas por pathquote.Quote (tasks.md, regla 11) y, cuando `--force`
// podría resolverlo, lo dice. Nacen con su primer llamador: `Engine.Done`.

// doneOperation es la orden de `axiom branch` que repiten las continuaciones de estos rechazos.
const doneOperation = "done"

// DoneInvocation es la orden de `done` con el cambio, el carril y la carpeta de partida; flags son
// las banderas que se añaden al final (por ejemplo, `--force`).
func DoneInvocation(change, cwd string, lane Lane, flags ...string) string {
	laneFlag := ""
	if lane == LaneODD {
		laneFlag = " --lane " + string(lane)
	}
	command := fmt.Sprintf("axiom branch %s --change %s%s --cwd %s", doneOperation, change, laneFlag, pathquote.Quote(cwd))
	if len(flags) > 0 {
		command += " " + strings.Join(flags, " ")
	}
	return command
}

// MergeNotConfirmed es el rechazo `merge_not_confirmed` de una rama cuya fusión ninguna capa
// confirma (REQ-25.16): no se elimina ni su worktree ni su rama. targets son las referencias en
// las que se buscó (vacías si el destino no existe) y repo, el repositorio de la rama; la
// continuación ofrece comprobar qué le falta a la rama y, si se quiere eliminar igualmente,
// `--force`, que deja en el informe la punta de la rama para recuperarla.
func MergeNotConfirmed(change, cwd string, lane Lane, repo RepoPlan, planned BranchPlan, targets []string) *Refusal {
	destination := planned.PRTarget
	message := fmt.Sprintf("la rama %s del repositorio %s no se puede confirmar como fusionada: no existe %s (ni en local ni como referencia remota local) donde buscar su fusión; no se elimina ni su worktree ni la rama", planned.Name, repo.Name, destination)
	if len(targets) > 0 {
		message = fmt.Sprintf("la rama %s del repositorio %s no está fusionada en %s: ninguna capa de confirmación de la fusión lo acredita; no se elimina ni su worktree ni la rama", planned.Name, repo.Name, strings.Join(targets, " ni en "))
	}
	return &Refusal{
		Code:    CodeMergeNotConfirmed,
		Message: message,
		Continuation: fmt.Sprintf("comprueba qué le falta a la rama con `git -C %s log --oneline %s..%s` y repite `%s` cuando esté fusionada, o repite `%s` para eliminarla igualmente (la punta de la rama queda en el informe para recuperarla)",
			pathquote.Quote(repo.GitRoot), destination, planned.Name, DoneInvocation(change, cwd, lane), DoneInvocation(change, cwd, lane, "--force")),
		Details: map[string]string{"branch": planned.Name, "repository": repo.GitRoot, "target": destination},
	}
}

// DoneDirtyWorktree es el rechazo `dirty_worktree` de una rama cuyo worktree (o, en `in_place`, el
// propio repositorio) tiene cambios sin confirmar (REQ-25.17): no se elimina nada. dir es donde
// están los cambios y paths las rutas que devuelve git, con «/». En `in_place`, `--force` no salta
// esta comprobación, de modo que la continuación solo ofrece confirmar o guardar los cambios.
func DoneDirtyWorktree(change, cwd string, lane Lane, repoName, name, dir string, paths []string, inPlace bool) *Refusal {
	listed, more := paths, ""
	if len(listed) > maxDirtyPathsListed {
		listed, more = listed[:maxDirtyPathsListed], fmt.Sprintf(" y %d más", len(paths)-maxDirtyPathsListed)
	}
	continuation := fmt.Sprintf("confirma esos cambios o guárdalos (por ejemplo, `git -C %s stash push --include-untracked`) y repite `%s`", pathquote.Quote(dir), DoneInvocation(change, cwd, lane))
	if inPlace {
		continuation += "; en el modo in_place `--force` no salta esta comprobación, porque el cambio de rama pondría en riesgo esos cambios"
	} else {
		continuation += fmt.Sprintf(", o repite `%s` para descartarlos (el informe los lista)", DoneInvocation(change, cwd, lane, "--force"))
	}
	return &Refusal{
		Code: CodeDirtyWorktree,
		Message: fmt.Sprintf("la rama %s del repositorio %s (%s) tiene cambios sin confirmar: %s%s; no se elimina ni su worktree ni la rama",
			name, repoName, dir, strings.Join(listed, ", "), more),
		Continuation: continuation,
		Details:      map[string]string{"branch": name, "repository": dir, "paths": strings.Join(paths, "\n")},
	}
}

// DoneCheckedOutInMainCheckout es el rechazo `branch_checked_out_elsewhere` de una rama del plan
// desplegada en el checkout principal de un cambio que no está en `in_place`: un worktree
// principal no se retira, de modo que se conserva la rama hasta que el checkout cambie de rama.
func DoneCheckedOutInMainCheckout(change, cwd string, lane Lane, repo RepoPlan, name, mainCheckout string) *Refusal {
	return &Refusal{
		Code:    CodeBranchCheckedOutElsewhere,
		Message: fmt.Sprintf("la rama %s del repositorio %s está desplegada en el checkout principal %s, que no es un worktree del plan: no se elimina", name, repo.Name, mainCheckout),
		Continuation: fmt.Sprintf("cambia ese checkout a otra rama con `git -C %s switch %s` y repite `%s`",
			pathquote.Quote(mainCheckout), repo.IntegrationBranch, DoneInvocation(change, cwd, lane)),
		Details: map[string]string{"branch": name, "repository": repo.GitRoot, "worktree": mainCheckout},
	}
}

// DoneGitFailed es el error `git_failed` de una operación de `done` sobre una rama: nombra la
// operación (gitOperation, en castellano) y resume el error de git (REQ-25.2). fix es la acción
// concreta que lo resuelve y la continuación la repite con la orden de `done`; la rama que
// falló se conserva.
func DoneGitFailed(change, cwd string, lane Lane, dir, gitOperation, fix string, err error) *Refusal {
	refusal := GitFailed(doneOperation, dir, gitOperation, err)
	refusal.Continuation = fmt.Sprintf("%s y repite `%s`: completa lo que falte", fix, DoneInvocation(change, cwd, lane))
	return refusal
}

// DoneFetchFailed es el rechazo `fetch_failed` de `done --fetch` (REQ-25.10): el `git fetch
// <remote>` de un repositorio del plan falló antes de evaluar ninguna rama, de modo que no se ha
// eliminado ninguna rama ni ningún worktree. La continuación ofrece repetir `--fetch` tras
// revisar el remoto o seguir sin red con las referencias locales.
func DoneFetchFailed(change, cwd string, lane Lane, repo RepoPlan, remote string, err error) *Refusal {
	return &Refusal{
		Code: CodeFetchFailed,
		Message: fmt.Sprintf("`git fetch %s` falló en el repositorio %s (%s): %v; no se ha eliminado ninguna rama ni ningún worktree",
			remote, repo.Name, repo.GitRoot, err),
		Continuation: fmt.Sprintf("comprueba el remoto con `git -C %s remote -v` y repite `%s`, o repite `%s` sin --fetch para trabajar con las referencias que ya hay en local",
			pathquote.Quote(repo.GitRoot), DoneInvocation(change, cwd, lane, "--fetch"), DoneInvocation(change, cwd, lane)),
		Details: map[string]string{"repository": repo.GitRoot, "remote": remote, "summary": err.Error()},
	}
}
