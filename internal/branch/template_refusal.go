package branch

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Rechazos de la plantilla de nombre de rama (design D-11). Todos se devuelven antes de crear
// nada: no se ha creado ninguna rama, worktree ni registro. Cada uno lleva una `Continuation`
// no vacía con las rutas citadas por pathquote.Quote.

// Motivos de `Details["reason"]` de `branch_template_not_applicable`.
const (
	TemplateReasonSharedBranch   = "shared_branch"
	TemplateReasonOpenspecBranch = "openspec_branch"
	TemplateReasonODDLane        = "odd_lane"
)

// TemplateUnresolved es el rechazo `branch_template_unresolved`: la variable de la plantilla key no
// tiene valor. variable es el nombre de la variable (`ref` o `task_ref`); role, el rol que entra
// cuando la variable es `task_ref`. La continuación de `{ref}` inicia el cambio desde su work item,
// retira la variable o fija el nombre con `--branch`; la de `{task_ref}` vincula la Task del rol.
func TemplateUnresolved(change, master, key, variable, role string, lane Lane) *Refusal {
	cwd := pathquote.Quote(master)
	details := map[string]string{"change": change, "key": key, "variable": variable}
	if variable == "task_ref" {
		if role == "" {
			role = "<rol>"
		}
		details["role"] = role
		return &Refusal{
			Code:    CodeBranchTemplateUnresolved,
			Message: fmt.Sprintf("la plantilla %s del cambio %s usa {task_ref} y el rol %s no tiene Task vinculada; no se ha creado ninguna rama ni worktree", key, change, role),
			Continuation: fmt.Sprintf("vincula la Task con `axiom integrations tasks bind --change %s --role %s=<id> --cwd %s` o créala con `axiom integrations tasks create --change %s --cwd %s`, y repite `axiom branch ensure --change %s --role %s --cwd %s`",
				change, role, cwd, change, cwd, change, role, cwd),
			Details: details,
		}
	}
	return &Refusal{
		Code:    CodeBranchTemplateUnresolved,
		Message: fmt.Sprintf("la plantilla %s del cambio %s usa {%s} y no hay work item vinculado, staging ni instantánea de la que tomar su valor; no se ha creado ninguna rama ni worktree", key, change, variable),
		Continuation: fmt.Sprintf("inicia el cambio desde su work item con `axiom change start --from <id|url> --cwd %s`, retira {%s} de %s, o fija el nombre con `%s`",
			cwd, variable, key, ensureInvocation("ensure", change, master, lane, Exceptions{Branch: "<rama>"})),
		Details: details,
	}
}

// TemplateNotApplicable es el rechazo `branch_template_not_applicable`: la plantilla key es válida
// pero no aplica a esta rama. reason es `shared_branch` (`{role}` o `{task_ref}` en la rama de
// seguimiento de un repositorio compartido), `openspec_branch` (la rama con `openspec/`) o
// `odd_lane` (el carril ODD no tiene roles). En ODD la continuación añade el `--branch`.
func TemplateNotApplicable(change, master, key, reason string, lane Lane) *Refusal {
	cwd := pathquote.Quote(master)
	what := map[string]string{
		TemplateReasonSharedBranch:   "la rama de seguimiento de un repositorio compartido no tiene un rol concreto",
		TemplateReasonOpenspecBranch: "la rama que lleva openspec/ no tiene un rol concreto",
		TemplateReasonODDLane:        "el carril odd no tiene roles",
	}[reason]
	if what == "" {
		what = "la rama no tiene un rol concreto"
	}
	continuation := fmt.Sprintf("quita {role} y {task_ref} de %s (las ramas de rol usan roles.<id>.branch_template) y comprueba con `axiom workspace validate --cwd %s`", key, cwd)
	if reason == TemplateReasonODDLane || lane == LaneODD {
		continuation += fmt.Sprintf("; o fija el nombre con `%s`", ensureInvocation("ensure", change, master, LaneODD, Exceptions{Branch: "<rama>"}))
	}
	return &Refusal{
		Code:         CodeBranchTemplateNotApplicable,
		Message:      fmt.Sprintf("la plantilla %s del cambio %s usa {role} o {task_ref} y no aplica: %s; no se ha creado ninguna rama ni worktree", key, change, what),
		Continuation: continuation,
		Details:      map[string]string{"change": change, "key": key, "reason": reason},
	}
}

// TemplateInvalidName es el rechazo `invalid_name` de un nombre renderizado que no es una
// referencia git válida o de un valor de variable no seguro. field es `branch_template` o el nombre
// de la variable; value, el nombre o el valor rechazado; cause, el porqué.
func TemplateInvalidName(change, master, key, field, value string, cause error) *Refusal {
	return &Refusal{
		Code:         CodeInvalidName,
		Message:      fmt.Sprintf("la plantilla %s del cambio %s produce un valor no válido en %s, «%s»: %v; no se ha creado ninguna rama ni worktree", key, change, field, value, cause),
		Continuation: fmt.Sprintf("corrige la plantilla o el valor de la variable y comprueba con `axiom workspace validate --cwd %s`, o fija el nombre con `axiom branch ensure --change %s --branch <rama> --cwd %s`", pathquote.Quote(master), change, pathquote.Quote(master)),
		Details:      map[string]string{"change": change, "key": key, "field": field, "value": value},
	}
}

// TemplateCollision es el rechazo `invalid_name` de dos ramas del mismo repositorio que
// renderizan el mismo nombre. roles nombra las ramas que chocan (la del cambio se cita como
// «cambio»).
func TemplateCollision(change, master, repository, name string, roles []string) *Refusal {
	list := strings.Join(roles, ", ")
	return &Refusal{
		Code:         CodeInvalidName,
		Message:      fmt.Sprintf("las ramas %s del repositorio %s renderizan el mismo nombre «%s» en el cambio %s; no se ha creado ninguna rama ni worktree", list, repository, name, change),
		Continuation: fmt.Sprintf("haz distintas las plantillas con una variable distintiva ({role}, {task_ref}, {ref}) y comprueba con `axiom workspace validate --cwd %s`", pathquote.Quote(master)),
		Details:      map[string]string{"change": change, "field": "branch_template", "value": name, "repository": repository, "roles": strings.Join(roles, ",")},
	}
}

// TemplateReservedName es el rechazo `invalid_name` de un nombre renderizado igual a la rama de
// integración o a `main_branch` del repositorio. reserved dice cuál: `integration_branch` o
// `main_branch`.
func TemplateReservedName(change, master, repository, name, reserved string) *Refusal {
	return &Refusal{
		Code:         CodeInvalidName,
		Message:      fmt.Sprintf("la plantilla del cambio %s renderiza «%s», que es la %s del repositorio %s; no se ha creado ninguna rama ni worktree", change, name, reserved, repository),
		Continuation: fmt.Sprintf("cambia la plantilla para que no coincida con una rama reservada y comprueba con `axiom workspace validate --cwd %s`", pathquote.Quote(master)),
		Details:      map[string]string{"change": change, "field": "branch_template", "value": name, "repository": repository, "reserved": reserved},
	}
}

// TemplateCandidateCollision es el rechazo `invalid_name` de dos roles del mismo repositorio cuya
// rama pendiente tendría el mismo nombre candidato (D-09): la recuperación no podría decidir a qué
// rol pertenece la rama existente.
func TemplateCandidateCollision(change, master, repository, name string, roles []string) *Refusal {
	list := strings.Join(roles, ", ")
	return &Refusal{
		Code:         CodeInvalidName,
		Message:      fmt.Sprintf("los roles %s del repositorio %s darían el mismo nombre de rama «%s» en el cambio %s; no se ha creado ninguna rama ni worktree", list, repository, name, change),
		Continuation: fmt.Sprintf("haz distinto el {task_ref} de cada rol con `axiom integrations tasks bind --change %s --role <rol>=<id> --cwd %s`, o distingue sus plantillas", change, pathquote.Quote(master)),
		Details:      map[string]string{"change": change, "field": "branch_template", "value": name, "repository": repository, "roles": strings.Join(roles, ",")},
	}
}

// IntegrationsLinksUnreadable es el rechazo `integrations_links_invalid` sin rol (G4): con un
// plugin activo e `integrations.yaml` ilegible, una plantilla que hay que renderizar usa una
// variable cuyo valor no está congelado. operation es la orden (`ensure`, `kickoff seal`…) y cause,
// el motivo de la lectura fallida.
func IntegrationsLinksUnreadable(operation, change, master string, cause error) *Refusal {
	return &Refusal{
		Code:         CodeIntegrationsLinksInvalid,
		Message:      fmt.Sprintf("no se puede renderizar la plantilla de rama de %s: integrations.yaml del cambio %s no se puede leer (%v); no se ha creado ninguna rama ni worktree", operation, change, cause),
		Continuation: fmt.Sprintf("revisa integrations.yaml del cambio y repite, o consulta su estado con `axiom integrations sync --change %s --cwd %s`", change, pathquote.Quote(master)),
		Details:      map[string]string{"change": change, "operation": operation},
	}
}
