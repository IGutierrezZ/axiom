package cli

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
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
	var changeRoot string
	for _, home := range homes {
		root := filepath.Join(home, "changes", change)
		if loaded, err := kickoff.Load(root); err == nil && loaded != nil {
			sealed, changeRoot = loaded, root
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
	return ensureRoleTaskBarrier(dir, change, role, changeRoot)
}

// ensureRoleTaskBarrier es la barrera D5 de `ensure --role`: con un plugin activo cuyo manifiesto
// declara items.create y el work item del cambio vinculado, el rol necesita su Task vinculada. No hace
// red ni muta nada. Sin plugin activo con items.create o sin vínculo work_item no cambia nada; un
// integrations.yaml ilegible con el plugin activo falla cerrado.
func ensureRoleTaskBarrier(dir, change, role, changeRoot string) *branch.Refusal {
	loc, err := workspace.Locate(dir)
	if err != nil || loc.Declarations.Integrations == nil {
		return nil
	}
	pluginID := activeItemsCreatePlugin(loc)
	if pluginID == "" {
		return nil
	}
	folder := integrations.Folder{WorkspaceRoot: loc.WorkspaceRoot, Root: changeRoot}
	workItems, err := integrations.WorkItemLinks(folder)
	if err != nil {
		return branch.IntegrationsLinksInvalid(change, role, dir, err)
	}
	if !hasPluginLink(workItems, pluginID, "") {
		return nil
	}
	tasks, err := integrations.TaskLinks(folder)
	if err != nil {
		return branch.IntegrationsLinksInvalid(change, role, dir, err)
	}
	if !hasPluginLink(tasks, pluginID, role) {
		return branch.TaskNotBound(change, role, dir)
	}
	return nil
}

// activeItemsCreatePlugin devuelve el primer plugin activo, por orden de declaración, cuyo manifiesto
// del catálogo declara items.create, o "" si no hay ninguno.
func activeItemsCreatePlugin(loc workspace.Location) string {
	for _, cfg := range loc.Declarations.Integrations.Active() {
		if m, ok := manifest.Lookup(cfg.ID); ok && manifestHasCapability(m, manifest.CapItemsCreate) {
			return cfg.ID
		}
	}
	return ""
}

// hasPluginLink dice si hay un vínculo del plugin y, si role no está vacío, de ese rol.
func hasPluginLink(links []integrations.Link, pluginID, role string) bool {
	for _, l := range links {
		if l.Plugin == pluginID && (role == "" || l.Role == role) {
			return true
		}
	}
	return false
}

// recordEnsureRoleStarted registra started_at del rol (D6) tras un `ensure --role` listo y antes de
// despachar role.start, para que la instantánea de vínculos del evento ya lo lleve. Solo actúa con un
// plugin activo con el puerto de Tasks y el vínculo de Task del rol; un fallo es un aviso
// `started_at_not_recorded` y no altera la salida ni el código de `ensure`.
func recordEnsureRoleStarted(dir string, result branch.Result, change, role string) {
	if result.Status != branch.StatusReady || result.PlanningHome == "" {
		return
	}
	loc, err := workspace.Locate(result.PlanningHome)
	if err != nil {
		if loc, err = workspace.Locate(dir); err != nil {
			return
		}
	}
	ctx := &trackerContext{Loc: loc, Change: change}
	if resolveTrackerPlugin(ctx, trackerOptions{Verb: "tasks bind"}) != nil || ctx.Plugin == nil {
		return
	}
	folder := integrations.Folder{WorkspaceRoot: loc.WorkspaceRoot, Root: filepath.Join(result.PlanningHome, "changes", change)}
	warn := func(cause error) {
		integrationWarn("started_at_not_recorded: no se ha podido registrar el inicio del rol %s del cambio %s (%v); `axiom integrations time propose` lo rechazará con role_not_started hasta que conste. Repite `axiom branch ensure --change %s --role %s`.",
			role, change, cause, change, role)
	}
	tasks, err := integrations.TaskLinks(folder)
	if err != nil {
		warn(err)
		return
	}
	if !hasPluginLink(tasks, ctx.PluginID, role) {
		return
	}
	if err := integrationWriterGuard(folder.Root, change, "branch ensure --role", branch.ContextOrdinary); err != nil {
		warn(err)
		return
	}
	if _, err := integrations.MarkRoleStarted(folder, ctx.PluginID, role, trackerNow().UTC()); err != nil {
		warn(err)
	}
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
