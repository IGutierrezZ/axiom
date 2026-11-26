package livespec

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Outcome es lo que la integración hace con una capacidad.
type Outcome string

const (
	// OutcomeCreated: la capacidad no existe y se crea con sus requisitos ADDED.
	OutcomeCreated Outcome = "created"
	// OutcomeComposed: la delta se compone sobre la especificación existente.
	OutcomeComposed Outcome = "composed"
	// OutcomeResumed: la capacidad estaba parcialmente integrada y se completa con lo pendiente.
	OutcomeResumed Outcome = "resumed"
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
	// Skipped son los requisitos ya integrados que una capacidad reanudada no vuelve a
	// componer (solo con OutcomeResumed).
	Skipped []SkippedRequirement
}

// SkippedRequirement es un requisito que una reanudación omite por estar ya integrado.
type SkippedRequirement struct {
	Section     string `json:"section"`
	Requirement string `json:"requirement"`
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
	// Assignments es el mapa de provisional a definitivo con el estado de cada entrada
	// (REQ-47.6); vacío sin provisionales.
	Assignments []PlannedEntry
	// Requirements son los requisitos que las deltas del documento tocan y crean (REQ-47.7).
	Requirements openspecdoc.Declaration
	// Duplicates son los identificadores repetidos en la spec viva resultante; los que no
	// escribe la integración se informan sin bloquear (REQ-47.5).
	Duplicates []Duplicate
	// Check dice que la integración se planificó solo para informar (`--check`): no se aplica.
	Check bool

	loc workspace.Location
	// docPath y docText son el documento ODD reescrito con el mapa y su traza; docText es
	// vacío si el documento no cambia.
	docPath, docText string
}

// NothingToIntegrate dice si la integración no cambia ningún fichero: el documento no
// tiene delta o todas sus capacidades ya estaban integradas y el documento no cambia.
func (i Integration) NothingToIntegrate() bool {
	for _, c := range i.Capabilities {
		if c.Outcome != OutcomeAlready {
			return false
		}
	}
	return i.docText == ""
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
	// canonical son los requisitos de la especificación canónica, en orden; vacío si la
	// capacidad no existe.
	canonical []openspecdoc.Requirement
}

// PlanIntegration calcula en memoria la integración de cada capacidad del documento
// sobre <planning home>/specs, sin escribir nada (REQ-30.2). Antes comprueba la obligación
// de impacto de los bugs (CheckImpact). Después valida y asigna los identificadores de todas
// las capacidades con el plan único (PlanIDs: provisionales, traza del documento,
// colisiones, REQ-30.4), sustituye el mapa en las deltas, clasifica cada requisito contra
// la especificación canónica (D5: `already` si ya está con el mismo cuerpo, rechazo si un
// ADDED está con otro cuerpo, `pending` en el resto), compone solo lo pendiente con la
// composición del archivado SDD, comprueba los duplicados entre capacidades (REQ-47.5),
// reescribe el documento con el mapa y su traza y, al final, comprueba el contexto técnico
// de la rama (REQ-30.6). El primer error rechaza todo el plan.
func PlanIntegration(loc workspace.Location, doc FeatureDoc, reg *Registry) (Integration, error) {
	if err := CheckImpact(doc); err != nil {
		return Integration{}, err
	}
	plan := Integration{Feature: doc.Feature, Kind: doc.Kind, loc: loc, Impact: impactOf(doc)}
	if plan.Impact == ImpactNone {
		plan.Reason = doc.Impact.Reason
	}

	pattern := reg.Pattern()
	docRel := oddDocRel(doc.Feature)
	var trace Trace
	var citations []Citation
	if doc.text != "" {
		var err error
		if trace, _, err = ExtractODDTrace(doc.text, docRel); err != nil {
			return Integration{}, err
		}
		if citations, err = docCitations(doc, docRel); err != nil {
			return Integration{}, err
		}
	}
	sources := make([]CapabilitySource, 0, len(doc.Capabilities))
	for _, c := range doc.Capabilities {
		path, canonical, existed, err := readCanonical(loc, c.Name, "axiom archive integrate --odd "+doc.Feature)
		if err != nil {
			return Integration{}, err
		}
		source := CapabilitySource{CapabilityDelta: c, File: docRel, Path: path, Canonical: canonical, Existed: existed}
		if c.TextLine > 0 {
			source.LineBase = c.TextLine - 1
		}
		sources = append(sources, source)
		declared := openspecdoc.Declared(c.Name, c.Delta, pattern)
		plan.Requirements.Modifies = append(plan.Requirements.Modifies, declared.Modifies...)
		plan.Requirements.Creates = append(plan.Requirements.Creates, declared.Creates...)
	}
	idPlan, err := PlanIDs(IDPlanInput{Pattern: pattern, Capabilities: sources, Registry: reg, Trace: trace, Citations: citations})
	if err != nil {
		return Integration{}, err
	}
	plan.Assignments = idPlan.Entries

	items := make([]classified, 0, len(sources))
	for _, s := range sources {
		cd, canonical := s.CapabilityDelta, s.Canonical
		if idPlan.Map.Count() > 0 {
			cd.Text = idPlan.Map.Substitute(cd.Text)
			cd.Delta = openspecdoc.ParseDelta(cd.Text)
		}
		if edited, ok := idPlan.Canonical[s.Name]; ok {
			canonical = edited
		}
		item, err := classifyIn(cd, s.Path, s.Existed, canonical, pattern)
		if err != nil {
			return Integration{}, err
		}
		dropTracedAdded(&item, idPlan.Entries, pattern)
		items = append(items, item)
	}
	if err := checkAddedWithoutID(doc.Feature, items, pattern); err != nil {
		return Integration{}, err
	}

	resulting := map[string]string{}
	for _, item := range items {
		step, err := compose(doc.Feature, item, pattern)
		if err != nil {
			return Integration{}, err
		}
		if edited, ok := idPlan.Canonical[item.cap.Name]; ok && step.Outcome == OutcomeAlready {
			// Solo cambió el número de un bloque propio de la spec viva (reasignación).
			step.Outcome, step.Content = OutcomeComposed, edited
		}
		if step.Outcome != OutcomeAlready {
			resulting[step.Name] = step.Content
		}
		plan.Capabilities = append(plan.Capabilities, step)
	}
	if plan.Duplicates, err = checkDuplicates(reg, resulting, idPlan.Written); err != nil {
		return Integration{}, err
	}

	if doc.text != "" {
		rewritten, err := rewriteODDDoc(doc, idPlan, pattern)
		if err != nil {
			return Integration{}, err
		}
		if rewritten != doc.text {
			plan.docPath, plan.docText = filepath.Join(loc.WorkspaceRoot, "odd", "tasks", doc.Feature+".md"), rewritten
		}
	}

	// La comprobación del contexto es lo último que se calcula: Apply es la única
	// escritura y solo se alcanza si todo el plan, contexto incluido, es válido (REQ-30.6).
	report, err := checkContext(context.Background(), loc, doc.Feature)
	if err != nil {
		return Integration{}, err
	}
	plan.Context = report
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

// classifyIn separa lo pendiente de lo ya integrado en la delta de una capacidad. MODIFIED y
// REMOVED se emparejan con la canónica como el compositor, por identificador con vuelta al
// título exacto (REQ-30.22), de modo que la segunda ejecución de una integración es
// `already`; ADDED y RENAMED siguen por título exacto. Opera sobre una especificación canónica ya leída,
// compartida por el carril ODD y el SDD (D3 paso 2.e). previous es el texto de la canónica
// y existed dice si la capacidad existe; path solo sirve para los motivos.
func classifyIn(c CapabilityDelta, path string, existed bool, previous string, pattern reqid.Pattern) (classified, error) {
	item := classified{cap: c, path: path, existed: existed, previous: previous}

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
	var canonicalList []openspecdoc.Requirement
	var canonicalNames []string
	for _, r := range openspecdoc.ParseCanonical(item.previous) {
		canonical[r.Name] = r
		canonicalList = append(canonicalList, r)
		canonicalNames = append(canonicalNames, r.Name)
	}
	item.canonical = canonicalList
	same := func(a, b openspecdoc.Requirement) bool {
		return openspecdoc.NormalizeBody(a.Body) == openspecdoc.NormalizeBody(b.Body)
	}

	for _, r := range c.Delta.Added {
		present, ok := canonical[r.Name]
		switch {
		case !ok:
			item.pending.Added = append(item.pending.Added, r)
		case !same(present, r):
			return classified{}, &DivergentRequirementError{Capability: c.Name, Requirement: r.Name, ID: pattern.InTitle(r.Name), Path: item.path}
		}
	}
	for _, r := range c.Delta.Modified {
		// Pendiente salvo que el requisito resuelto ya tenga el mismo cuerpo normalizado; un
		// nombre que no se resuelve (o es ambiguo) queda pendiente y la composición lo rechaza.
		if i, match := openspecdoc.Resolve(canonicalNames, r.Name, pattern); match != openspecdoc.MatchFound || !same(canonicalList[i], r) {
			item.pending.Modified = append(item.pending.Modified, r)
		}
	}
	for _, r := range c.Delta.Removed {
		// Pendiente si el requisito se resuelve; si no, ya estaba retirado.
		if _, match := openspecdoc.Resolve(canonicalNames, r.Name, pattern); match != openspecdoc.MatchNone {
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

// patternHint nombra la plantilla del proyecto en los motivos cuando no es la de por
// defecto; con el defecto no añade nada y el texto es el de siempre.
func patternHint(pattern reqid.Pattern) string {
	if pattern == reqid.Default() {
		return ""
	}
	return fmt.Sprintf("; el patrón de identificadores del proyecto es %s (sdd.id_pattern de axiom.yaml)", pattern.Template())
}

// nextFree es el siguiente número libre de la serie de id: el del registro, saltando los
// que el propio documento ya usa.
func nextFree(reg *Registry, id string, taken map[string]bool) string {
	pattern := reg.Pattern()
	series, ok := pattern.Series(id)
	if !ok {
		return id
	}
	candidate := reg.NextFree(series)
	for taken[candidate] {
		number, _ := pattern.Number(candidate)
		candidate = pattern.Format(series, number+1)
	}
	return candidate
}

// compose calcula la especificación resultante de una capacidad.
func compose(feature string, item classified, pattern reqid.Pattern) (CapabilityPlan, error) {
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
	composed, err := openspecdoc.Compose(item.previous, reducedDelta(pending), openspecdoc.ComposeOptions{IDPattern: pattern})
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
