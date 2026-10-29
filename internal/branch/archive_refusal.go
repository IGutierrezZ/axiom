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

// ChangeNotIntegrated es el rechazo `change_not_integrated` de `ensure --archive` (REQ-25.45,
// design D-25 paso 5): el árbol de la base evaluada no contiene la carpeta `changes/<cambio>/`
// del planning home, de modo que el PR del cambio no consta fusionado en las referencias
// locales y no se crea ninguna rama ni worktree. Nombra la base evaluada (`base`, como
// `origin/main` o `main`) y la carpeta que falta (`changePath`, relativa a la raíz del
// repositorio, con «/»). `fetched` dice si la invocación ya traía el remoto: sin `--fetch` la
// continuación lo sugiere (no hay fetch implícito, REQ-25.10); con él, la base sigue sin el
// cambio y lo que falta es fusionar su PR. `archivedEntry` es, si la base ya contiene el cambio
// archivado (G-13), la entrada `changes/archive/<fecha>-<cambio>/` que lo prueba: el mensaje la
// nombra y dice que no hay nada que archivar. prefix es el `--prefix` de la invocación, que la
// continuación conserva.
func ChangeNotIntegrated(change, master, prefix string, repo RepoPlan, base, changePath, archivedEntry string, fetched bool) *Refusal {
	retry := fmt.Sprintf("axiom branch ensure --cwd %s --change %s --archive --fetch", pathquote.Quote(master), change)
	if prefix != "" {
		retry += " --prefix " + prefix
	}
	details := map[string]string{"change": change, "base": base, "path": changePath, "repository": repo.GitRoot}
	if archivedEntry != "" {
		details["archivedEntry"] = archivedEntry
		return &Refusal{
			Code: CodeChangeNotIntegrated,
			Message: fmt.Sprintf("el cambio %s ya está archivado en la base %s del repositorio %s: contiene %s y no %s, así que no hay nada que archivar; no se ha creado ninguna rama ni worktree",
				change, base, repo.Name, archivedEntry, changePath),
			Continuation: fmt.Sprintf("no hace falta archivarlo: consulta su estado con `axiom branch status --change %s --cwd %s`", change, pathquote.Quote(master)),
			Details:      details,
		}
	}
	message := fmt.Sprintf("el cambio %s no está integrado en la base %s del repositorio %s: su carpeta %s no existe en ese árbol, así que el PR del cambio no consta fusionado en las referencias locales; no se ha creado ninguna rama ni worktree",
		change, base, repo.Name, changePath)
	continuation := fmt.Sprintf("trae la referencia remota y repite con `%s` (no hay fetch implícito), o fusiona antes el PR del cambio en %s", retry, repo.IntegrationBranch)
	if fetched {
		message = fmt.Sprintf("el cambio %s no está integrado en la base %s del repositorio %s ni tras el --fetch: su carpeta %s no existe en ese árbol; no se ha creado ninguna rama ni worktree",
			change, base, repo.Name, changePath)
		continuation = fmt.Sprintf("fusiona el PR del cambio en %s y repite `%s`", repo.IntegrationBranch, retry)
	}
	return &Refusal{Code: CodeChangeNotIntegrated, Message: message, Continuation: continuation, Details: details}
}

// ArchiveBranchConflict es el rechazo `plan_conflict` de un `--prefix` que daría a la rama de
// archivado un nombre distinto del de la que el registro ya guarda para el cambio (REQ-25.12):
// una rama de archivado no cambia de nombre y la repetición idéntica es idempotente. No se ha
// tocado nada; la continuación repite `ensure --archive` sin `--prefix`, que vuelve a la rama
// registrada, y explica cómo retirarla para elegir otro nombre. command es la orden que repite
// la continuación (`ensure --archive`).
func ArchiveBranchConflict(command, change, master string, lane Lane, repository, gitRoot, registered, planned string) *Refusal {
	return &Refusal{
		Code: CodePlanConflict,
		Message: fmt.Sprintf("el cambio %s ya tiene la rama de archivado %s en el repositorio %s y el prefijo pedido daría %s: una rama de archivado no cambia de nombre; no se ha tocado nada",
			change, registered, repository, planned),
		Continuation: fmt.Sprintf("repite sin --prefix, `%s`; para usar otro nombre, retira antes la rama registrada con `git -C %s branch -D %s` (y su worktree con `git -C %s worktree remove <ruta>`)",
			ensureInvocation(command, change, master, lane, Exceptions{}), pathquote.Quote(gitRoot), registered, pathquote.Quote(gitRoot)),
		Details: map[string]string{"change": change, "registered": registered, "planned": planned},
	}
}

// PlanningHomeOutsideRepository es el rechazo `config_invalid` de `ensure --archive` cuando el
// planning home queda fuera del repositorio en el que se crearía la rama de archivado: no se
// puede comprobar que la base contiene `changes/<cambio>/` porque su ruta no es relativa a la
// raíz del repositorio. No se ha creado nada; la continuación corrige la configuración.
func PlanningHomeOutsideRepository(master, planningHome string, repo RepoPlan) *Refusal {
	return &Refusal{
		Code:         CodeConfigInvalid,
		Message:      fmt.Sprintf("el planning home %s queda fuera del repositorio %s (%s): no se puede comprobar que la base contiene el cambio ya fusionado; no se ha creado ninguna rama ni worktree", planningHome, repo.Name, repo.GitRoot),
		Continuation: validateConfigContinuation(master),
		Details:      map[string]string{"planningHome": planningHome, "repository": repo.GitRoot},
	}
}
