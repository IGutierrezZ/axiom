package branch

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Rechazos de `axiom branch ensure --role <rol>` (REQ-43.24, design D-13). Todos se devuelven
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
