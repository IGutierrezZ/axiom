package branch

import "strings"

// Nombres con plantilla para la guarda (C-07-f, design D-10) y para el archivado (S9). La guarda
// calcula el nombre esperado con el mismo `repoNaming` que el plan, de modo que ambos nunca
// discrepan; no lee nada: los valores de las variables llegan en PlanInput.TemplateValues.

// GuardRepository son los hechos de un repositorio con los que la guarda nombra sus ramas.
type GuardRepository struct {
	Facts     RepoFacts // GitRoot, CommonDir, RelPath y Frozen (el registro)
	Planning  bool      // el repositorio aloja `openspec/`
	Roles     []string  // roles del cambio que tocan el repositorio (D-04)
	Effective RepoPolicy
	Prefix    string // `--prefix` registrado o, sin él, el de la política
}

// GuardBranches son los nombres que el plan daría a las ramas de un repositorio.
type GuardBranches struct {
	// Change es la rama del cambio; vacía si espera a su rol (PendingRoles).
	Change string
	// Templated dice que la rama del cambio se nombra con plantilla.
	Templated bool
	// Roles son los nombres de las ramas de rol pedidas, con su plantilla o `<cambio>-<rol>`.
	Roles []string
	// PendingRoles son los roles cuya rama aún no tiene nombre (su Task no existe).
	PendingRoles []string
}

// RenderGuardBranches nombra la rama del cambio del repositorio y las de los roles de roleNames con
// el mismo criterio que el plan (D-04, D-05, D-07). Un rechazo (variable sin valor, rol sin sentido
// con la plantilla, nombre inválido) se devuelve tal cual: la guarda falla cerrado.
func RenderGuardBranches(in PlanInput, repo GuardRepository, roleNames []string) (GuardBranches, *Refusal) {
	selection := repoSelection{facts: repo.Facts, planning: repo.Planning, roles: repo.Roles}
	naming := newRepoNaming(in, selection, repo.Effective, repoName(repo.Facts), repo.Prefix)
	change, refusal := naming.changeBranch()
	if refusal != nil {
		return GuardBranches{}, refusal
	}
	var out GuardBranches
	switch {
	case change.pending != nil:
		out.Templated = true
		if change.pending.Candidate != "" {
			out.Change = change.pending.Candidate
		} else if change.pending.RoleID != "" {
			out.PendingRoles = append(out.PendingRoles, change.pending.RoleID)
		}
	default:
		out.Change, out.Templated = change.name, change.template != ""
	}
	for _, role := range roleNames {
		named, refusal := naming.roleBranch(role, out.Change)
		if refusal != nil {
			return GuardBranches{}, refusal
		}
		switch {
		case named.pending == nil:
			out.Roles = append(out.Roles, named.name)
		case named.pending.Candidate != "":
			out.Roles = append(out.Roles, named.pending.Candidate)
		default:
			out.PendingRoles = append(out.PendingRoles, role)
		}
	}
	return out, nil
}

// ChangeBranchTemplated dice si la rama del cambio del repositorio del planning home se nombra con
// plantilla: la de su entrada en la instantánea o, en un cambio sin sellar, la de la política. Con
// plantilla, `--prefix` no interviene en el nombre de la rama de archivado (design D-14, S9).
func ChangeBranchTemplated(in PlanInput) bool {
	return archiveChangeTemplated(in, repoPolicy(in.Policy, in.PlanningRepo))
}

// FrozenFrom reduce a RecordBranch las entradas con plantilla de un repositorio del registro: solo
// ellas son nombres congelados (design D-09).
func FrozenFrom(branches []RecordBranch) []FrozenBranch {
	var frozen []FrozenBranch
	for _, recorded := range branches {
		if recorded.Template == "" || recorded.Name == "" {
			continue
		}
		roleID := recorded.RoleID
		if recorded.Kind != KindRole {
			roleID = ""
		}
		frozen = append(frozen, FrozenBranch{Kind: recorded.Kind, RoleID: strings.ToLower(roleID), Name: recorded.Name, Template: recorded.Template})
	}
	return frozen
}
