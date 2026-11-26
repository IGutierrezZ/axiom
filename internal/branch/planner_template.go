package branch

import (
	"errors"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch/nametemplate"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Plantillas de nombre de rama en el planificador (C-07-f, design D-04, D-05, D-07 y D-15). Todo
// es puro: los valores de las variables los reúne el motor solo con lecturas de ficheros y llegan
// en PlanInput.TemplateValues. Sin ninguna plantilla en juego, ninguna de estas rutas evalúa nada
// y el plan es el de siempre.

// repoNaming resuelve el nombre de las ramas de un repositorio.
type repoNaming struct {
	in         PlanInput
	selection  repoSelection
	effective  RepoPolicy
	repository string // `<repo>` del plan
	prefix     string // `--prefix` o, sin él, el de la política
	snapshot   kickoff.GitSnapshotRepo
	hasSnap    bool
}

// namedBranch es el resultado de nombrar una rama: su nombre y la plantilla que lo produjo, o la
// entrada pendiente cuando todavía no tiene nombre.
type namedBranch struct {
	name, template string
	pending        *PendingBranch
}

// activeTemplate es la plantilla que rige una rama: su texto, la clave de la que sale, el
// `{prefix}` que congeló la instantánea (vacío si no) y el rol atribuido (vacío si ninguno).
type activeTemplate struct {
	raw, key, prefix, role string
}

func newRepoNaming(in PlanInput, selection repoSelection, effective RepoPolicy, repository, prefix string) repoNaming {
	snapshot, hasSnap := snapshotRepo(in, selection.facts)
	return repoNaming{in: in, selection: selection, effective: effective, repository: repository, prefix: prefix, snapshot: snapshot, hasSnap: hasSnap}
}

// sealedChange dice que el cambio ya está sellado y su plantilla solo puede salir de la instantánea:
// uno sellado sin plantilla en su instantánea nunca aplica la de la política (D5, prov:KOF:1).
func sealedChange(in PlanInput) bool {
	return in.Lane != LaneODD && in.HasKickoff && !in.Sealing
}

func (n repoNaming) frozen(kind BranchKind, role string) (FrozenBranch, bool) {
	for _, frozen := range n.selection.facts.Frozen {
		if frozen.Name != "" && frozen.Template != "" && frozen.Kind == kind && strings.EqualFold(frozen.RoleID, role) {
			return frozen, true
		}
	}
	return FrozenBranch{}, false
}

// changeBranch nombra la rama del cambio del repositorio.
func (n repoNaming) changeBranch() (namedBranch, *Refusal) {
	snapshotName := ""
	if n.hasSnap {
		snapshotName = snapshotBranch(n.snapshot, KindChange, "")
	}
	frozen, _ := n.frozen(KindChange, "")
	named := ResolveChangeBranch(n.in.Change, "",
		NamedBranch("--branch", n.in.Exceptions.Branch),
		NamedBranch("instantánea", snapshotName),
		NamedBranch("registro", frozen.Name))
	switch {
	case named == "":
	case n.in.Exceptions.Branch != "":
		return namedBranch{name: named}, nil
	case snapshotName != "":
		return namedBranch{name: named, template: snapshotBranchTemplate(n.snapshot, KindChange, "")}, nil
	default:
		return namedBranch{name: named, template: frozen.Template}, nil
	}

	role, reason := n.changeAttribution()
	tpl, ok := n.changeTemplate(role)
	if !ok {
		return namedBranch{name: ResolveChangeBranch(n.in.Change, n.prefix)}, nil
	}
	return n.render(tpl, KindChange, reason)
}

// roleBranch nombra la rama de un rol de un repositorio compartido: solo cuenta su plantilla de rol;
// sin ella, `<rama de seguimiento>-<rol>`.
func (n repoNaming) roleBranch(role, tracking string) (namedBranch, *Refusal) {
	if n.hasSnap {
		if name := snapshotBranch(n.snapshot, KindRole, role); name != "" {
			return namedBranch{name: name, template: snapshotBranchTemplate(n.snapshot, KindRole, role)}, nil
		}
	}
	if frozen, ok := n.frozen(KindRole, role); ok {
		return namedBranch{name: frozen.Name, template: frozen.Template}, nil
	}
	tpl, ok := n.roleTemplate(role)
	if !ok {
		return namedBranch{name: tracking + "-" + role}, nil
	}
	return n.render(tpl, KindRole, "")
}

// changeAttribution es el rol al que se atribuye la rama del cambio (design D-04) o, sin rol, el
// motivo por el que {role} y {task_ref} no aplican.
func (n repoNaming) changeAttribution() (role, reason string) {
	switch {
	case n.in.Lane == LaneODD:
		return "", TemplateReasonODDLane
	case n.selection.planning:
		return "", TemplateReasonOpenspecBranch
	case len(n.selection.roles) == 1:
		return n.selection.roles[0], ""
	default:
		return "", TemplateReasonSharedBranch
	}
}

// changeTemplate es la plantilla de la rama del cambio: la de la entrada pendiente de la instantánea
// en un cambio sellado y, en uno sin sellar, en el sellado o en ODD, rol > repositorio > `git:`.
func (n repoNaming) changeTemplate(role string) (activeTemplate, bool) {
	if sealedChange(n.in) {
		if !n.hasSnap {
			return activeTemplate{}, false
		}
		pending, ok := snapshotPending(n.snapshot, KindChange, "")
		if !ok {
			return activeTemplate{}, false
		}
		return activeTemplate{raw: pending.Template, key: sealedTemplateKey(n.snapshot), prefix: pending.Prefix, role: pending.Role}, true
	}
	if role != "" && n.in.Topology == workspace.TopologyMultirepo {
		if raw := n.in.Policy.RoleTemplates[strings.ToLower(role)]; raw != "" {
			return activeTemplate{raw: raw, key: "roles." + role + ".branch_template", role: role}, true
		}
	}
	if n.effective.BranchTemplate == "" {
		return activeTemplate{}, false
	}
	return activeTemplate{raw: n.effective.BranchTemplate, key: n.effective.BranchTemplateKey, role: role}, true
}

// roleTemplate es la plantilla de la rama de un rol de un repositorio compartido: solo
// `roles.<id>.branch_template` (multirepo) o la entrada pendiente de rol de la instantánea.
func (n repoNaming) roleTemplate(role string) (activeTemplate, bool) {
	if sealedChange(n.in) {
		if !n.hasSnap {
			return activeTemplate{}, false
		}
		pending, ok := snapshotPending(n.snapshot, KindRole, role)
		if !ok {
			return activeTemplate{}, false
		}
		return activeTemplate{raw: pending.Template, key: sealedTemplateKey(n.snapshot), prefix: pending.Prefix, role: role}, true
	}
	if n.in.Topology != workspace.TopologyMultirepo {
		return activeTemplate{}, false
	}
	raw := n.in.Policy.RoleTemplates[strings.ToLower(role)]
	if raw == "" {
		return activeTemplate{}, false
	}
	return activeTemplate{raw: raw, key: "roles." + role + ".branch_template", role: role}, true
}

func sealedTemplateKey(repo kickoff.GitSnapshotRepo) string {
	return "kickoff.yaml git.repositories[" + repo.Path + "].pending_branches"
}

// render renderiza la plantilla de una rama: la deja pendiente si usa `{task_ref}` y su rol no
// entra, y si no sustituye las variables con los valores del motor. noRoleReason es el motivo por el
// que la rama no tiene rol, para rechazar `{role}` y `{task_ref}`.
func (n repoNaming) render(tpl activeTemplate, kind BranchKind, noRoleReason string) (namedBranch, *Refusal) {
	in := n.in
	parsed, err := nametemplate.Parse(tpl.raw)
	if err != nil {
		return namedBranch{}, TemplateInvalidName(in.Change, in.Master, tpl.key, "branch_template", tpl.raw, err)
	}
	if parsed.IsZero() {
		return namedBranch{}, TemplateInvalidName(in.Change, in.Master, tpl.key, "branch_template", "", errors.New("la plantilla está vacía"))
	}
	if (parsed.Uses(nametemplate.VarRole) || parsed.Uses(nametemplate.VarTaskRef)) && tpl.role == "" {
		return namedBranch{}, TemplateNotApplicable(in.Change, in.Master, tpl.key, noRoleReason, in.Lane)
	}
	prefix := in.Exceptions.Prefix
	if prefix == "" {
		prefix = tpl.prefix
	}
	if prefix == "" {
		prefix = n.effective.BranchPrefix
	}
	values := in.TemplateValues
	pending := PendingBranch{Kind: kind, RoleID: tpl.role, Template: parsed.Raw(), Key: tpl.key, Prefix: prefix}

	if parsed.Uses(nametemplate.VarTaskRef) && !strings.EqualFold(in.EnteringRole, tpl.role) {
		if values.LinksErr == nil {
			if id := values.TaskRefs[strings.ToLower(tpl.role)]; id != "" {
				if candidate, err := parsed.Render(n.valuesFor(prefix, tpl.role, id)); err == nil {
					pending.Candidate = candidate
				}
			}
		} else if in.Describe {
			pending.Refusal = IntegrationsLinksUnreadable(in.Operation, in.Change, in.Master, values.LinksErr)
		}
		return namedBranch{pending: &pending}, nil
	}

	if refusal := n.unresolved(parsed, tpl); refusal != nil {
		if in.Describe && (refusal.Code == CodeBranchTemplateUnresolved || refusal.Code == CodeIntegrationsLinksInvalid) {
			pending.Refusal = refusal
			return namedBranch{pending: &pending}, nil
		}
		return namedBranch{}, refusal
	}
	name, err := parsed.Render(n.valuesFor(prefix, tpl.role, values.TaskRefs[strings.ToLower(tpl.role)]))
	if err != nil {
		return namedBranch{}, n.renderRefusal(tpl, err)
	}
	return namedBranch{name: name, template: parsed.Raw()}, nil
}

// unresolved comprueba lo que impide renderizar: un integrations.yaml ilegible con una variable que
// no está congelada (G4) o una variable sin valor.
func (n repoNaming) unresolved(parsed nametemplate.Template, tpl activeTemplate) *Refusal {
	in, values := n.in, n.in.TemplateValues
	if values.LinksErr != nil {
		needsLinks := (parsed.Uses(nametemplate.VarRef) || parsed.Uses(nametemplate.VarSlug)) && !values.RefFrozen ||
			parsed.Uses(nametemplate.VarKind) && !values.KindFrozen ||
			parsed.Uses(nametemplate.VarTaskRef)
		if needsLinks {
			return IntegrationsLinksUnreadable(in.Operation, in.Change, in.Master, values.LinksErr)
		}
	}
	if parsed.Uses(nametemplate.VarRef) && values.Ref == "" {
		return TemplateUnresolved(in.Change, in.Master, tpl.key, string(nametemplate.VarRef), "", in.Lane)
	}
	if parsed.Uses(nametemplate.VarTaskRef) && values.TaskRefs[strings.ToLower(tpl.role)] == "" {
		return TemplateUnresolved(in.Change, in.Master, tpl.key, string(nametemplate.VarTaskRef), tpl.role, in.Lane)
	}
	return nil
}

// renderRefusal traduce el fallo de render a su rechazo tipado: un valor no seguro nombra su variable
// y un nombre inválido, la plantilla; nunca se sanea nada.
func (n repoNaming) renderRefusal(tpl activeTemplate, err error) *Refusal {
	in := n.in
	var unsafe *nametemplate.UnsafeValueError
	var invalid *nametemplate.InvalidNameError
	var missing *nametemplate.MissingValueError
	switch {
	case errors.As(err, &unsafe):
		return TemplateInvalidName(in.Change, in.Master, tpl.key, string(unsafe.Variable), unsafe.Value, unsafe.Cause)
	case errors.As(err, &invalid):
		return TemplateInvalidName(in.Change, in.Master, tpl.key, "branch_template", invalid.Name, invalid.Cause)
	case errors.As(err, &missing):
		return TemplateUnresolved(in.Change, in.Master, tpl.key, string(missing.Variable), tpl.role, in.Lane)
	}
	return TemplateInvalidName(in.Change, in.Master, tpl.key, "branch_template", tpl.raw, err)
}

// valuesFor compone los valores de render de una rama: `{kind}` vale feature salvo `fix`, `{slug}`
// se deriva de `{ref}` y `{role}` y `{task_ref}` solo existen con rol.
func (n repoNaming) valuesFor(prefix, role, taskRef string) nametemplate.Values {
	values := n.in.TemplateValues
	kind := values.Kind
	if kind == "" {
		kind = "feature"
	}
	rendered := nametemplate.Values{
		nametemplate.VarPrefix: prefix,
		nametemplate.VarKind:   kind,
		nametemplate.VarChange: n.in.Change,
		nametemplate.VarSlug:   nametemplate.Slug(n.in.Change, values.Ref),
	}
	if values.Ref != "" {
		rendered[nametemplate.VarRef] = values.Ref
	}
	if role != "" {
		rendered[nametemplate.VarRole] = role
	}
	if taskRef != "" {
		rendered[nametemplate.VarTaskRef] = taskRef
	}
	return rendered
}

// checkTemplateCollisions rechaza con `invalid_name`, sin crear nada, un repositorio cuyas ramas se
// nombran con plantilla y chocan entre sí o con la rama de integración o `main_branch` (design D-05).
// Sin ninguna rama con plantilla no evalúa nada.
func checkTemplateCollisions(in PlanInput, repo RepoPlan, effective RepoPolicy) *Refusal {
	templated := false
	for _, planned := range repo.Branches {
		templated = templated || planned.Template != ""
	}
	if !templated {
		return nil
	}
	seen := map[string]string{}
	for _, planned := range repo.Branches {
		label := "cambio"
		if planned.Kind == KindRole {
			label = planned.RoleID
		}
		if previous, duplicated := seen[planned.Name]; duplicated {
			return TemplateCollision(in.Change, in.Master, repo.Name, planned.Name, []string{previous, label})
		}
		seen[planned.Name] = label
	}
	for _, planned := range repo.Branches {
		switch {
		case planned.Template == "":
		case planned.Name == effective.IntegrationBranch:
			return TemplateReservedName(in.Change, in.Master, repo.Name, planned.Name, "integration_branch")
		case effective.MainBranch != "" && planned.Name == effective.MainBranch:
			return TemplateReservedName(in.Change, in.Master, repo.Name, planned.Name, "main_branch")
		}
	}
	return nil
}

// archiveChangeTemplated dice si la rama del cambio del planning home se nombra con plantilla: la
// que la instantánea dejó anotada en un cambio sellado o, en uno sin sellar, la de la política.
func archiveChangeTemplated(in PlanInput, effective RepoPolicy) bool {
	if sealedChange(in) {
		if repo, ok := snapshotRepo(in, in.PlanningRepo); ok {
			return snapshotBranchTemplate(repo, KindChange, "") != ""
		}
		return false
	}
	return effective.BranchTemplate != ""
}
