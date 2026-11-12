package livespec

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Outcome es lo que la integración hace con una capacidad.
type Outcome string

const (
	// OutcomeCreated: la capacidad no existe y se crea con sus requisitos ADDED.
	OutcomeCreated Outcome = "created"
	// OutcomeComposed: la delta se compone sobre la especificación existente.
	OutcomeComposed Outcome = "composed"
	// OutcomeAlready: la especificación ya contiene la delta; no se escribe nada.
	OutcomeAlready Outcome = "already_integrated"
)

// Valores de Integration.Impact (el campo `impact` del informe).
const (
	// ImpactDelta: el documento aporta una delta (aunque lleve un marcador, que se ignora).
	ImpactDelta = "delta"
	// ImpactNone: el documento declara el marcador válido y no tiene delta.
	ImpactNone = "no_impact"
	// ImpactAbsent: el documento no declara ni delta ni marcador válido.
	ImpactAbsent = "absent"
)

// CapabilityPlan es la integración calculada de una capacidad.
type CapabilityPlan struct {
	Name string
	// Path es la ruta absoluta de openspec/specs/<capacidad>/spec.md.
	Path    string
	Outcome Outcome
	// Content es la especificación resultante; vacío con OutcomeAlready.
	Content string
	// Existed dice si la especificación ya existía antes de la integración.
	Existed bool
}

// Integration es el resultado, calculado en memoria, de integrar un documento ODD. No se
// ha escrito nada hasta que se llama a Apply.
type Integration struct {
	Feature      string
	Kind         Kind
	Impact       string
	Reason       string // el motivo del marcador, cuando Impact es ImpactNone
	Capabilities []CapabilityPlan
	Context      *ContextReport

	loc workspace.Location
}

// NothingToIntegrate dice si la integración no cambia ningún fichero: el documento no
// tiene delta o todas sus capacidades ya estaban integradas.
func (i Integration) NothingToIntegrate() bool {
	for _, c := range i.Capabilities {
		if c.Outcome != OutcomeAlready {
			return false
		}
	}
	return true
}

// CollisionError es el rechazo de un requisito ADDED cuyo identificador ya existe (REQ-30.4).
type CollisionError struct {
	ID string
	// Where nombra dónde colisiona: la capacidad, el cambio activo o, si es un duplicado
	// del propio documento, las dos capacidades.
	Where string
	// NextFree es el siguiente número libre de la serie del identificador.
	NextFree string
}

func (e *CollisionError) Error() string {
	return fmt.Sprintf("el identificador %s de un requisito ADDED ya existe en %s; elige el siguiente número libre de su serie, %s, y vuelve a ejecutar la integración", e.ID, e.Where, e.NextFree)
}

// DivergentRequirementError es el rechazo de un requisito ADDED que la especificación ya
// contiene con un cuerpo distinto (REQ-30.3).
type DivergentRequirementError struct {
	Capability  string
	Requirement string
	ID          string
	Path        string
}

func (e *DivergentRequirementError) Error() string {
	return fmt.Sprintf("capacidad %s: el requisito %q ya está integrado en %s con un cuerpo distinto del de la delta; si quieres cambiarlo, muévelo a `## MODIFIED Requirements`; si el integrado es el correcto, alinea la delta con él", e.Capability, e.Requirement, e.Path)
}

// MissingCapabilityError es el rechazo de una operación MODIFIED, REMOVED o RENAMED sobre
// una capacidad que no existe en openspec/specs/ (REQ-30.1).
type MissingCapabilityError struct {
	Capability  string
	Section     string
	Requirement string
}

func (e *MissingCapabilityError) Error() string {
	return fmt.Sprintf("capacidad %s: la delta tiene una sección %s (requisito %q) pero la capacidad no existe en openspec/specs/; solo ADDED crea una capacidad: cambia la operación o corrige el nombre de la capacidad", e.Capability, e.Section, e.Requirement)
}

// CapabilityError sitúa un error de composición en su capacidad.
type CapabilityError struct {
	Capability string
	Err        error
}

func (e *CapabilityError) Error() string { return fmt.Sprintf("capacidad %s: %v", e.Capability, e.Err) }
func (e *CapabilityError) Unwrap() error { return e.Err }

// classified es la delta de una capacidad dividida en lo ya integrado y lo pendiente.
type classified struct {
	cap      CapabilityDelta
	path     string
	existed  bool
	previous string
	pending  openspecdoc.Delta
}

// PlanIntegration calcula en memoria la integración de cada capacidad del documento
// sobre <planning home>/specs, sin escribir nada (REQ-30.2). Cada requisito se clasifica
// contra la especificación canónica (D5): `already` si ya está con el mismo cuerpo,
// rechazo si un ADDED está con otro cuerpo, `pending` en el resto. Después comprueba los
// identificadores de los ADDED pendientes contra el registro (REQ-30.4) y compone solo
// lo pendiente con la composición del archivado SDD. El primer error rechaza todo el plan.
func PlanIntegration(loc workspace.Location, doc FeatureDoc, reg *Registry) (Integration, error) {
	plan := Integration{Feature: doc.Feature, Kind: doc.Kind, loc: loc, Impact: impactOf(doc)}
	if plan.Impact == ImpactNone {
		plan.Reason = doc.Impact.Reason
	}

	items := make([]classified, 0, len(doc.Capabilities))
	for _, c := range doc.Capabilities {
		item, err := classify(loc, c)
		if err != nil {
			return Integration{}, err
		}
		items = append(items, item)
	}

	if err := checkIdentifiers(doc.Feature, items, reg); err != nil {
		return Integration{}, err
	}

	for _, item := range items {
		step, err := compose(doc.Feature, item)
		if err != nil {
			return Integration{}, err
		}
		plan.Capabilities = append(plan.Capabilities, step)
	}
	return plan, nil
}

func impactOf(doc FeatureDoc) string {
	switch {
	case doc.HasDelta():
		return ImpactDelta
	case doc.Impact != nil && doc.Impact.NoImpact:
		return ImpactNone
	default:
		return ImpactAbsent
	}
}

func specPath(loc workspace.Location, capability string) string {
	return filepath.Join(loc.PlanningHome, "specs", capability, "spec.md")
}

// classify separa lo pendiente de lo ya integrado en la delta de una capacidad.
func classify(loc workspace.Location, c CapabilityDelta) (classified, error) {
	item := classified{cap: c, path: specPath(loc, c.Name)}
	data, err := os.ReadFile(item.path)
	switch {
	case err == nil:
		item.existed, item.previous = true, string(data)
	case os.IsNotExist(err):
	default:
		return classified{}, &CapabilityError{Capability: c.Name, Err: fmt.Errorf("no se puede leer %s: %w; comprueba los permisos del fichero y vuelve a ejecutar `axiom archive integrate --odd <feature>`", item.path, err)}
	}

	if !item.existed {
		for _, group := range []struct {
			section string
			reqs    []openspecdoc.Requirement
		}{{"MODIFIED", c.Delta.Modified}, {"REMOVED", c.Delta.Removed}, {"RENAMED", c.Delta.Renamed}} {
			if len(group.reqs) > 0 {
				return classified{}, &MissingCapabilityError{Capability: c.Name, Section: group.section, Requirement: group.reqs[0].Name}
			}
		}
		item.pending.Added = c.Delta.Added
		return item, nil
	}

	canonical := map[string]openspecdoc.Requirement{}
	for _, r := range openspecdoc.ParseCanonical(item.previous) {
		canonical[r.Name] = r
	}
	same := func(a, b openspecdoc.Requirement) bool {
		return openspecdoc.NormalizeBody(a.Body) == openspecdoc.NormalizeBody(b.Body)
	}

	for _, r := range c.Delta.Added {
		present, ok := canonical[r.Name]
		switch {
		case !ok:
			item.pending.Added = append(item.pending.Added, r)
		case !same(present, r):
			return classified{}, &DivergentRequirementError{Capability: c.Name, Requirement: r.Name, ID: r.ID, Path: item.path}
		}
	}
	for _, r := range c.Delta.Modified {
		if present, ok := canonical[r.Name]; !ok || !same(present, r) {
			item.pending.Modified = append(item.pending.Modified, r)
		}
	}
	for _, r := range c.Delta.Removed {
		if _, ok := canonical[r.Name]; ok {
			item.pending.Removed = append(item.pending.Removed, r)
		}
	}
	for _, r := range c.Delta.Renamed {
		oldName, newName, ok := openspecdoc.SplitRename(r.Name)
		_, oldPresent := canonical[oldName]
		_, newPresent := canonical[newName]
		if ok && newPresent && !oldPresent {
			continue
		}
		item.pending.Renamed = append(item.pending.Renamed, r)
	}
	return item, nil
}

// checkIdentifiers comprueba los ADDED pendientes: cada uno lleva `(REQ-n.m)`, no existe
// en el registro y no se repite en otra capacidad del documento.
func checkIdentifiers(feature string, items []classified, reg *Registry) error {
	taken := map[string]bool{}
	for _, item := range items {
		for _, r := range item.pending.Added {
			if r.ID != "" {
				taken[r.ID] = true
			}
		}
	}
	firstSeen := map[string]string{}
	for _, item := range items {
		for _, r := range item.pending.Added {
			if r.ID == "" {
				return &DocError{Feature: feature, Problems: []Problem{{
					Code:    "added_without_id",
					Line:    item.cap.Line + r.Line,
					Subject: r.Name,
					Message: fmt.Sprintf("el requisito ADDED %q de la capacidad %s no lleva un identificador `(REQ-n.m)` en el título; añade el siguiente número libre de la serie, p. ej. `### Requirement: %s (REQ-n.m)`", r.Name, item.cap.Name, r.Name),
				}}}
			}
			if refs := reg.Lookup(r.ID); len(refs) > 0 {
				return &CollisionError{ID: r.ID, Where: refs[0].Where(), NextFree: nextFree(reg, r.ID, taken)}
			}
			if other, dup := firstSeen[r.ID]; dup {
				return &CollisionError{
					ID:       r.ID,
					Where:    fmt.Sprintf("el propio documento, en las capacidades %s y %s", other, item.cap.Name),
					NextFree: nextFree(reg, r.ID, taken),
				}
			}
			firstSeen[r.ID] = item.cap.Name
		}
	}
	return nil
}

// nextFree es el siguiente número libre de la serie de id: el del registro, saltando los
// que el propio documento ya usa.
func nextFree(reg *Registry, id string, taken map[string]bool) string {
	series, ok := SeriesOf(id)
	if !ok {
		return id
	}
	candidate := reg.NextFree(series)
	for taken[candidate] {
		number, _ := strconv.Atoi(candidate[strings.LastIndexByte(candidate, '.')+1:])
		candidate = fmt.Sprintf("REQ-%d.%d", series, number+1)
	}
	return candidate
}

// compose calcula la especificación resultante de una capacidad.
func compose(feature string, item classified) (CapabilityPlan, error) {
	step := CapabilityPlan{Name: item.cap.Name, Path: item.path, Existed: item.existed}
	pending := item.pending
	if len(pending.Added)+len(pending.Modified)+len(pending.Removed)+len(pending.Renamed) == 0 {
		step.Outcome = OutcomeAlready
		return step, nil
	}
	if !item.existed {
		step.Outcome = OutcomeCreated
		step.Content = openspecdoc.RenderNewSpec(item.cap.Name, feature, pending.Added)
		return step, nil
	}
	composed, err := openspecdoc.Compose(item.previous, reducedDelta(pending), openspecdoc.ComposeOptions{})
	if err != nil {
		return CapabilityPlan{}, &CapabilityError{Capability: item.cap.Name, Err: err}
	}
	step.Outcome, step.Content = OutcomeComposed, composed
	return step, nil
}

// reducedDelta es la delta con solo lo pendiente, en el formato que consume Compose.
func reducedDelta(d openspecdoc.Delta) string {
	var b strings.Builder
	for _, section := range []struct {
		name string
		reqs []openspecdoc.Requirement
	}{{"ADDED", d.Added}, {"MODIFIED", d.Modified}, {"REMOVED", d.Removed}, {"RENAMED", d.Renamed}} {
		if len(section.reqs) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## %s Requirements\n\n", section.name)
		for _, r := range section.reqs {
			b.WriteString(strings.TrimRight(strings.ReplaceAll(r.Body, "\r\n", "\n"), " \t\n"))
			b.WriteString("\n\n")
		}
	}
	return b.String()
}
