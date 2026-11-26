package branch

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Rechazos de `axiom branch ensure --role <rol>` (RF-BRA-33, design D-13). Todos se devuelven
// antes de llamar al motor: no se ha creado ninguna rama, worktree ni registro. El motor de
// `ensure` no conoce el rol.

// RoleArchiveConflict es el `plan_conflict` de `--role` con `--archive`: la rama de archivado no
// pertenece a ningún rol.
func RoleArchiveConflict(change, master string) *Refusal {
	return &Refusal{
		Code:         CodePlanConflict,
		Message:      fmt.Sprintf("--role no se admite con --archive: la rama de archivado del cambio %s no pertenece a ningún rol; no se ha creado ninguna rama ni worktree", change),
		Continuation: fmt.Sprintf("repite sin --role: `axiom branch ensure --change %s --archive --cwd %s`", change, pathquote.Quote(master)),
		Details:      map[string]string{"change": change, "flags": "--role,--archive"},
	}
}

// RoleNotApplicable es el rechazo `role_not_applicable` de `--role` en el carril odd, que no tiene
// roles.
func RoleNotApplicable(change, role, master string) *Refusal {
	return &Refusal{
		Code:         CodeRoleNotApplicable,
		Message:      fmt.Sprintf("--role %s no es aplicable al carril odd del cambio %s: el carril odd no tiene roles ni kickoff; no se ha creado ninguna rama ni worktree", role, change),
		Continuation: fmt.Sprintf("repite sin --role: `axiom branch ensure --change %s --lane odd --cwd %s`", change, pathquote.Quote(master)),
		Details:      map[string]string{"change": change, "role": role, "lane": string(LaneODD)},
	}
}

// KickoffNotSealed es el rechazo `kickoff_not_sealed` de `--role` en un cambio sin `kickoff.yaml`
// sellado: sin roster no hay roles que arrancar.
func KickoffNotSealed(change, role, master string) *Refusal {
	return &Refusal{
		Code:         CodeKickoffNotSealed,
		Message:      fmt.Sprintf("--role %s exige el kickoff sellado del cambio %s y no hay ningún kickoff.yaml sellado; no se ha creado ninguna rama ni worktree", role, change),
		Continuation: fmt.Sprintf("sella el kickoff con `axiom sdd kickoff seal --cwd %s --change %s --execution-style checkpointed` y repite", pathquote.Quote(master), change),
		Details:      map[string]string{"change": change, "role": role},
	}
}

// TaskNotBound es el rechazo `task_not_bound` de `--role` con un tracker activo: el cambio tiene su
// work item vinculado y el rol no tiene Task. La continuación nombra las dos salidas reales.
func TaskNotBound(change, role, master string) *Refusal {
	cwd := pathquote.Quote(master)
	return &Refusal{
		Code:         CodeTaskNotBound,
		Message:      fmt.Sprintf("el rol %s del cambio %s no tiene Task vinculada en el tracker y no se puede arrancar; no se ha creado ninguna rama ni worktree", role, change),
		Continuation: fmt.Sprintf("crea las Tasks con `axiom integrations tasks create --change %s --cwd %s` o vincula una existente con `axiom integrations tasks bind --change %s --role %s=<id> --cwd %s`, y repite", change, cwd, change, role, cwd),
		Details:      map[string]string{"change": change, "role": role},
	}
}

// IntegrationsLinksInvalid es el rechazo `integrations_links_invalid` de `--role` con un tracker
// activo cuyo integrations.yaml no se puede leer: la barrera falla cerrado y no se esquiva con un
// fichero roto. cause es el motivo de la lectura fallida.
func IntegrationsLinksInvalid(change, role, master string, cause error) *Refusal {
	return &Refusal{
		Code:         CodeIntegrationsLinksInvalid,
		Message:      fmt.Sprintf("no se puede comprobar la Task del rol %s: integrations.yaml del cambio %s no se puede leer (%v); no se ha creado ninguna rama ni worktree", role, change, cause),
		Continuation: fmt.Sprintf("revisa integrations.yaml del cambio y repite, o consulta su estado con `axiom integrations sync --change %s --cwd %s`", change, pathquote.Quote(master)),
		Details:      map[string]string{"change": change, "role": role},
	}
}

// RoleNotInRoster es el rechazo `role_not_in_roster` de `--role` con un rol que no pertenece al
// roster sellado; roster lista los roles válidos (detalle `roster`, separados por coma).
func RoleNotInRoster(change, role string, roster []string) *Refusal {
	list := strings.Join(roster, ", ")
	return &Refusal{
		Code:         CodeRoleNotInRoster,
		Message:      fmt.Sprintf("el rol %s no pertenece al roster sellado del cambio %s (roster: %s); no se ha creado ninguna rama ni worktree", role, change, list),
		Continuation: fmt.Sprintf("repite con --role de uno de los roles del roster: %s", list),
		Details:      map[string]string{"change": change, "role": role, "roster": strings.Join(roster, ",")},
	}
}
