package branch

import (
	"fmt"
	"strings"

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

// El modo de trabajo de `ensure` (REQ-25.8, design D-08): la decisión de `ask` y los rechazos
// del modo. La continuación de todos ellos es una orden `axiom branch ensure --mode ...` con
// el cambio, el carril y la carpeta de partida.

// modeInvocation es la orden de `ensure` que da una respuesta de modo. cwd es el directorio de
// partida; el carril `sdd`, el de por defecto, no se repite.
func modeInvocation(operation, change, cwd string, lane Lane, mode Mode) string {
	laneFlag := ""
	if lane == LaneODD {
		laneFlag = " --lane " + string(lane)
	}
	return fmt.Sprintf("axiom branch %s --change %s%s --mode %s --cwd %s", operation, change, laneFlag, mode, pathquote.Quote(cwd))
}

// DecisionRequired es el resultado `decision_required` de un proyecto en modo `ask` cuyo
// cambio no tiene respuesta registrada (REQ-25.8): no se ha creado nada, ni siquiera el
// registro, y `ensure` no espera entrada. La decisión lleva las dos respuestas posibles, cada
// una con la orden que la da; quien invoca (una skill, el dashboard) presenta la pregunta y
// repite `ensure` con `--mode`.
func DecisionRequired(operation, change, cwd string, lane Lane) *Refusal {
	worktree := modeInvocation(operation, change, cwd, lane, ModeWorktree)
	inPlace := modeInvocation(operation, change, cwd, lane, ModeInPlace)
	return &Refusal{
		Code:    CodeDecisionRequired,
		Message: fmt.Sprintf("el modo de trabajo del proyecto es «%s» y el cambio %s no tiene respuesta registrada: no se ha creado nada ni se ha escrito el registro", Label(string(ModeAsk)), change),
		Continuation: fmt.Sprintf("elige un modo y repite `%s` (%s) o `%s` (%s): la respuesta se registra para este cambio y no modifica tus preferencias del hub",
			worktree, Label(string(ModeWorktree)), inPlace, Label(string(ModeInPlace))),
		Details: map[string]string{"change": change, "mode": string(ModeAsk)},
		Decision: &Decision{
			Question: fmt.Sprintf("El modo de trabajo del proyecto es «%s» y el cambio %s no tiene respuesta registrada: ¿%s o %s?", Label(string(ModeAsk)), change, Label(string(ModeWorktree)), Label(string(ModeInPlace))),
			Choices: []DecisionChoice{
				{
					Answer:     string(ModeWorktree),
					Label:      Label(string(ModeWorktree)),
					Effect:     "crea la rama del cambio desde la base y un worktree por rama en la raíz de worktrees; el checkout principal no se toca",
					Invocation: worktree,
				},
				{
					Answer:     string(ModeInPlace),
					Label:      Label(string(ModeInPlace)),
					Effect:     "crea (o selecciona) la rama del cambio y hace checkout de ella en el propio repositorio; exige el árbol de trabajo limpio",
					Invocation: inPlace,
				},
			},
		},
	}
}

// ModeConflict es el rechazo `plan_conflict` de un `--mode` distinto del modo con el que el
// cambio ya está materializado (REQ-25.8): un cambio no cambia de modo. No se ha tocado nada;
// la continuación repite `ensure` con el modo materializado.
func ModeConflict(operation, change, cwd string, lane Lane, materialized, requested Mode) *Refusal {
	return &Refusal{
		Code:         CodePlanConflict,
		Message:      fmt.Sprintf("el cambio %s ya está materializado en modo %s y se pidió --mode %s: un cambio no cambia de modo", change, materialized, requested),
		Continuation: fmt.Sprintf("repite con el modo materializado, `%s`, o sin --mode", modeInvocation(operation, change, cwd, lane, materialized)),
		Details:      map[string]string{"change": change, "mode": string(materialized), "requested": string(requested)},
	}
}

// InvalidMode es el rechazo `config_invalid` de una petición cuyo modo no es una respuesta
// válida: solo `worktree` e `in_place` se materializan (`ask` es una preferencia, no una
// respuesta). No se ha tocado nada.
func InvalidMode(operation, change, cwd string, lane Lane, mode Mode) *Refusal {
	return &Refusal{
		Code:    CodeConfigInvalid,
		Message: fmt.Sprintf("el modo «%s» no es una respuesta válida: usa %s o %s", mode, ModeWorktree, ModeInPlace),
		Continuation: fmt.Sprintf("repite `%s` o `%s`", modeInvocation(operation, change, cwd, lane, ModeWorktree),
			modeInvocation(operation, change, cwd, lane, ModeInPlace)),
		Details: map[string]string{"change": change, "mode": string(mode)},
	}
}

// maxDirtyPathsListed es cuántos ficheros sin confirmar nombra el mensaje de `dirty_worktree`
// antes de resumir el resto: la lista completa va en Details.
const maxDirtyPathsListed = 10

// DirtyWorktree es el rechazo `dirty_worktree` de `ensure` en modo `in_place` (REQ-25.9, matriz
// de amenazas «Estado del commit»): el repositorio tiene cambios sin confirmar (ficheros
// rastreados modificados, con o sin stage, o ficheros sin seguimiento no ignorados) y cambiar
// de rama los pondría en riesgo. Se decide en la comprobación sin efectos, de modo que no se ha
// modificado nada: ni HEAD, ni el índice, ni el árbol de trabajo, ni la rama del cambio ni el
// registro. Nombra el repositorio, su ruta y los ficheros, y ofrece la continuación: confirmar
// o guardar los cambios y repetir, o, mientras el cambio no esté materializado en `in_place`
// (changeMaterialized falso), usar un worktree por rama. paths son las rutas que devuelve git,
// con «/».
func DirtyWorktree(operation, change, cwd string, lane Lane, repo RepoPlan, dir string, paths []string, changeMaterialized bool) *Refusal {
	listed := paths
	more := ""
	if len(listed) > maxDirtyPathsListed {
		listed = listed[:maxDirtyPathsListed]
		more = fmt.Sprintf(" y %d más", len(paths)-maxDirtyPathsListed)
	}
	target := repo.IntegrationBranch
	if len(repo.Branches) > 0 {
		target = repo.Branches[0].Name
	}
	continuation := fmt.Sprintf("confirma esos cambios o guárdalos (por ejemplo, `git -C %s stash push --include-untracked`) y repite `%s`",
		pathquote.Quote(dir), modeInvocation(operation, change, cwd, lane, ModeInPlace))
	if !changeMaterialized {
		continuation += fmt.Sprintf(", o trabaja en un worktree por rama con `%s`", modeInvocation(operation, change, cwd, lane, ModeWorktree))
	}
	return &Refusal{
		Code: CodeDirtyWorktree,
		Message: fmt.Sprintf("el repositorio %s (%s) tiene cambios sin confirmar y cambiar a la rama %s los pondría en riesgo: %s%s; no se ha modificado nada",
			repo.Name, dir, target, strings.Join(listed, ", "), more),
		Continuation: continuation,
		Details:      map[string]string{"repository": dir, "branch": target, "paths": strings.Join(paths, "\n")},
	}
}
