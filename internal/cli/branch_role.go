package cli

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// `axiom branch ensure --role <rol>` (RF-BRA-33, D-13): el adaptador valida el rol ANTES de llamar
// al motor, con rechazos tipados y sin efectos, y tras un `ensure` listo emite role.start a las
// integraciones. El motor de ensure no conoce el rol.

// canonicalEnsureRole deja el rol en la forma canónica de la compuerta role-apply:<rol>
// (minúsculas), que es también la que usa la clave de idempotencia role.start/<cambio>/<rol>.
func canonicalEnsureRole(role string) string { return strings.ToLower(strings.TrimSpace(role)) }

// validateEnsureRole devuelve el rechazo de `--role`, o nil si el rol es válido. El orden es el de
// D-13: --archive, carril odd, kickoff sellado y pertenencia al roster. El kickoff se busca, en solo
// lectura, en el planning home que devuelve el estado del cambio cuando está registrado (puede
// estar en su worktree) y, si no, en el de workspace.Locate.
func validateEnsureRole(dir, change string, lane branch.Lane, archive bool, role string) *branch.Refusal {
	switch {
	case archive:
		return branch.RoleArchiveConflict(change, dir)
	case lane == branch.LaneODD:
		return branch.RoleNotApplicable(change, role, dir)
	}
	// Candidatos en orden: el planning home del estado del cambio (puede ser el de su worktree, que
	// aún no existe si ensure no se ha ejecutado) y el de workspace.Locate.
	var homes []string
	if status := engine.Default().Status(context.Background(), dir, change, branch.LaneSDD); status.Status != branch.StatusError && status.Status != branch.StatusRefused && status.PlanningHome != "" {
		homes = append(homes, status.PlanningHome)
	}
	if loc, err := workspace.Locate(dir); err == nil {
		homes = append(homes, loc.PlanningHome)
	}
	var sealed *kickoff.Kickoff
	for _, home := range homes {
		if loaded, err := kickoff.Load(filepath.Join(home, "changes", change)); err == nil && loaded != nil {
			sealed = loaded
			break
		}
	}
	if sealed == nil {
		return branch.KickoffNotSealed(change, role, dir)
	}
	roster := rosterFromKickoff(*sealed)
	if !gateKeyInRoster(kickoff.RoleApplyGate(role), roster) {
		return branch.RoleNotInRoster(change, role, rosterRoleNames(roster))
	}
	return nil
}

// emitEnsureRoleStart emite role.start tras un `ensure` listo, sobre la carpeta del cambio del
// planning home que devolvió el motor (la del worktree si lo hay).
func emitEnsureRoleStart(dir string, result branch.Result, change, role string) {
	if result.Status != branch.StatusReady || result.PlanningHome == "" {
		return
	}
	loc, err := workspace.Locate(result.PlanningHome)
	if err != nil {
		loc, err = workspace.Locate(dir)
		if err != nil {
			return
		}
	}
	emitRoleStartEvent(loc, filepath.Join(result.PlanningHome, "changes", change), role)
}
