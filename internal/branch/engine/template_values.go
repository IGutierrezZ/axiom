package engine

import (
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/nametemplate"
	"github.com/IGutierrezZ/axiom/v3/internal/changestart"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Valores de las variables de plantilla (C-07-f, design D-13). El resolutor solo lee ficheros
// (staging de `change start`, work-item.json adoptado, `integrations.yaml` y la instantánea del
// kickoff) y solo actúa cuando alguna plantilla del plan usa `ref`, `kind`, `slug` o `task_ref`:
// sin plantilla no hay lecturas nuevas ni modos de fallo nuevos.

// templateNeeds son las variables cuyos valores hay que resolver.
type templateNeeds struct{ kind, ref, taskRef bool }

func (n templateNeeds) any() bool { return n.kind || n.ref || n.taskRef }

// templatesInPlay dice qué variables que no salen del propio plan usan las plantillas que pueden
// renderizarse: en un cambio sellado, las de las entradas pendientes de su instantánea; en uno sin
// sellar, en el sellado o en ODD, las de la política (las de rol solo rigen en multirepo).
func templatesInPlay(in branch.PlanInput) templateNeeds {
	var raws []string
	if in.Lane != branch.LaneODD && in.HasKickoff && !in.Sealing {
		if in.Snapshot != nil {
			for _, repo := range in.Snapshot.Repositories {
				for _, pending := range repo.PendingBranches {
					raws = append(raws, pending.Template)
				}
			}
		}
	} else {
		raws = append(raws, in.Policy.BranchTemplate)
		for _, repo := range in.Policy.PerRepository {
			raws = append(raws, repo.BranchTemplate)
		}
		if in.Topology == workspace.TopologyMultirepo {
			for _, raw := range in.Policy.RoleTemplates {
				raws = append(raws, raw)
			}
		}
	}
	var needs templateNeeds
	for _, raw := range raws {
		if raw == "" {
			continue
		}
		parsed, err := nametemplate.Parse(raw)
		if err != nil {
			continue
		}
		needs.kind = needs.kind || parsed.Uses(nametemplate.VarKind)
		needs.ref = needs.ref || parsed.Uses(nametemplate.VarRef) || parsed.Uses(nametemplate.VarSlug)
		needs.taskRef = needs.taskRef || parsed.Uses(nametemplate.VarTaskRef)
	}
	return needs
}

// resolveTemplateValues resuelve los valores de las variables que usan las plantillas del plan. Con
// un plugin activo e `integrations.yaml` ilegible no rechaza: entrega LinksErr y el planificador o la
// guarda rechazan solo si una plantilla lo necesita (D-11).
func resolveTemplateValues(in branch.PlanInput, location workspace.Location) branch.TemplateValues {
	needs := templatesInPlay(in)
	if !needs.any() {
		return branch.TemplateValues{}
	}
	sealed := in.Lane != branch.LaneODD && in.HasKickoff && !in.Sealing
	values := branch.TemplateValues{Kind: "feature"}

	adoptedDir := filepath.Join(location.PlanningHome, "changes", in.Change, "context")
	if in.Lane == branch.LaneODD {
		adoptedDir = filepath.Join(location.WorkspaceRoot, "odd", "context", in.Change)
	}
	origin, hasOrigin := changestart.WorkItemFacts(in.PlanningRepo.CommonDir, in.Change, adoptedDir)

	if sealed {
		// Un cambio sellado solo lee de su instantánea; sin `template_values`, `{kind}` es feature (G9).
		values.KindFrozen = true
		if in.Snapshot != nil && in.Snapshot.TemplateValues != nil {
			frozen := in.Snapshot.TemplateValues
			if frozen.Kind != "" {
				values.Kind = frozen.Kind
			}
			if frozen.Ref != "" {
				values.Ref, values.RefFrozen = frozen.Ref, true
			}
		}
	} else if hasOrigin && origin.Profile == string(changestart.ProfileBug) {
		values.Kind = "fix"
	}

	needsLinks := needs.ref && !values.RefFrozen || needs.kind && !values.KindFrozen || needs.taskRef
	if active := activePlugins(location); needsLinks && len(active) > 0 && in.Lane != branch.LaneODD {
		ref, taskRefs, err := changestart.LinkedIDs(location.WorkspaceRoot, filepath.Join(location.PlanningHome, "changes", in.Change), active)
		if err != nil {
			values.LinksErr = err
		} else {
			if values.Ref == "" {
				values.Ref = ref
			}
			values.TaskRefs = taskRefs
		}
	}
	if values.Ref == "" && hasOrigin {
		values.Ref = origin.ID
	}
	return values
}

// activePlugins son los identificadores de los plugins de integraciones activos, en orden de
// declaración.
func activePlugins(location workspace.Location) []string {
	if location.Declarations.Integrations == nil {
		return nil
	}
	var ids []string
	for _, plugin := range location.Declarations.Integrations.Active() {
		ids = append(ids, plugin.ID)
	}
	return ids
}
