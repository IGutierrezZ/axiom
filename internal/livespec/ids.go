package livespec

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// Códigos estables de IDPlanError.Code (D7); aparecen tal cual en el JSON de `--check`.
const (
	CodeProvisionalMalformed            = reqid.CodeProvisionalMalformed
	CodeProvisionalNewSeriesUnsupported = reqid.CodeProvisionalNewSeriesUnsupported
	CodeProvisionalOutsideAdded         = "provisional_outside_added"
	CodeProvisionalDuplicate            = "provisional_duplicate"
	CodeProvisionalAmbiguous            = "provisional_ambiguous"
	CodeProvisionalUndeclared           = "provisional_undeclared"
	CodeUnknownRequirementID            = "unknown_requirement_id"
	CodeIDAllocationFailed              = "id_allocation_failed"
	CodeIDTraceInvalid                  = "id_trace_invalid"
	CodeIDTracePatternMismatch          = "id_trace_pattern_mismatch"
	CodeIDTraceStale                    = "id_trace_stale"
	CodeDuplicateAssignedID             = "duplicate_assigned_id"
	CodeSupersededRequirementUnknown    = "superseded_requirement_unknown"
	CodeSupersededRequirementAmbiguous  = "superseded_requirement_ambiguous"
	CodeFlatDeltaUnsupported            = "flat_delta_unsupported"
)

// Estados de una asignación en el informe (D5).
const (
	StateAssigned   = "assigned"
	StateKept       = "kept"
	StateReassigned = "reassigned"
)

// IDPlanError es el rechazo tipado de la validación previa o de la asignación (D7). El
// plan no escribe nada: ningún error deja el árbol cambiado.
type IDPlanError struct {
	Code         string
	Capability   string
	Section      string
	Requirement  string
	Token        string
	File         string
	Line         int
	Reason       string
	Continuation string
}

func (e *IDPlanError) Error() string {
	var where []string
	if e.Capability != "" {
		where = append(where, "capacidad "+e.Capability)
	}
	if e.File != "" {
		if e.Line > 0 {
			where = append(where, fmt.Sprintf("%s:%d", e.File, e.Line))
		} else {
			where = append(where, e.File)
		}
	}
	text := e.Reason
	if len(where) > 0 {
		text = strings.Join(where, ", ") + ": " + text
	}
	if e.Continuation != "" {
		text += "; " + e.Continuation
	}
	return text
}

// Unwrap devuelve, para unknown_requirement_id, el *openspecdoc.UnknownRequirementIDError
// equivalente, de modo que quien ya trata el error del compositor lo siga viendo.
func (e *IDPlanError) Unwrap() error {
	if e.Code != CodeUnknownRequirementID {
		return nil
	}
	return &openspecdoc.UnknownRequirementIDError{Section: e.Section, Requirement: e.Requirement, ID: e.Token}
}

// CapabilitySource es una capacidad del cambio o del documento con su canónica actual.
type CapabilitySource struct {
	CapabilityDelta
	// File es la ruta de la delta, solo para los motivos (en ODD, el documento).
	File string
	// LineBase es la línea del documento anterior a la primera de Text (0 en SDD).
	LineBase int
	// Path es la ruta de openspec/specs/<capacidad>/spec.md.
	Path string
	// Canonical es el texto actual de la canónica; Existed dice si la capacidad existe.
	Canonical string
	Existed   bool
}

// Citation es un token provisional citado en un texto que se valida (D6): una línea
// `Cubre:`, un fichero de tareas, la matriz de verify o el documento ODD.
type Citation struct {
	File  string
	Line  int
	Token string
}

// IDPlanInput es todo lo que el plan necesita; PlanIDs no lee ficheros.
type IDPlanInput struct {
	Pattern reqid.Pattern
	// Capabilities en el orden del carril: SDD por nombre de carpeta, ODD el del documento.
	Capabilities []CapabilitySource
	// Registry es el registro de especificaciones y cambios activos sin el propio cambio.
	Registry  *Registry
	Trace     Trace
	Citations []Citation
	// Superseded son los nombres de `--superseded-requirements` (solo SDD), con el prefijo
	// opcional `<capacidad>/`.
	Superseded []string
	// FlatDelta es la ruta de un `specs/spec.md` plano, si el cambio lo tiene: no se admite.
	FlatDelta string
}

// PlannedEntry es una entrada de la traza con el estado de su asignación.
type PlannedEntry struct {
	TraceEntry
	State string
}

// IDPlan es el resultado del plan: el mapa de sustitución, las entradas de la traza en el
// orden de aparición, la canónica editada de las capacidades donde un bloque propio se
// reasignó, los nombres superados por capacidad y todos los identificadores definitivos
// que la integración escribe.
type IDPlan struct {
	Map        reqid.IDMap
	Entries    []PlannedEntry
	Canonical  map[string]string
	Superseded map[string][]string
	Written    []string
}

// Trace devuelve la traza resultante del plan para el patrón dado.
func (p IDPlan) Trace(pattern reqid.Pattern) Trace {
	trace := Trace{Schema: traceSchema, Pattern: pattern.Template()}
	for _, e := range p.Entries {
		trace.Assignments = append(trace.Assignments, e.TraceEntry)
	}
	return trace
}

// addedItem es un requisito ADDED con identidad (un provisional o un definitivo) y su
// resolución.
type addedItem struct {
	capIdx int
	name   string
	req    openspecdoc.Requirement

	token string // provisional del título
	pv    reqid.Provisional
	def   string // definitivo del título

	entry int // índice en la traza; -1 si no tiene entrada
	state string
	final string
	own   *openspecdoc.Requirement // bloque propio en la canónica
	prev  []string
}

type tokenRef struct {
	token, capability, file string
	line                    int
}

// PlanIDs es el plan único de asignación de identificadores (D3): valida todo antes de
// asignar, sin escribir, y de la misma manera con o sin `--check`. El primer error rechaza
// el plan completo.
func PlanIDs(in IDPlanInput) (IDPlan, error) {
	p := in.Pattern
	caps := in.Capabilities

	if in.FlatDelta != "" {
		return IDPlan{}, &IDPlanError{Code: CodeFlatDeltaUnsupported, File: in.FlatDelta,
			Reason:       "el cambio tiene una delta plana `specs/spec.md` y no se sabe a qué capacidad pertenece",
			Continuation: "mueve la delta a `specs/<capacidad>/spec.md`"}
	}

	// a. La traza: legible, de este patrón y coherente con los ADDED.
	if err := in.Trace.validate(p, "la traza de identificadores"); err != nil {
		return IDPlan{}, err
	}
	if err := checkTraceFresh(in); err != nil {
		return IDPlan{}, err
	}

	// b. Los tokens provisionales de los textos validados: gramática y patrón. Un token
	// dentro de código de la delta es un ejemplo y no cuenta (REQ-47.4).
	var refs []tokenRef
	for _, c := range caps {
		for _, span := range reqid.ScanProvisionalRefs(c.Text) {
			line := c.LineBase + lineAt(c.Text, span.Start)
			if err := checkToken(p, span.Token, c.Name, c.File, line); err != nil {
				return IDPlan{}, err
			}
			refs = append(refs, tokenRef{token: span.Token, capability: c.Name, file: c.File, line: line})
		}
	}
	for _, ct := range in.Citations {
		if err := checkToken(p, ct.Token, "", ct.File, ct.Line); err != nil {
			return IDPlan{}, err
		}
		refs = append(refs, tokenRef{token: ct.Token, file: ct.File, line: ct.Line})
	}

	// c. Los nombres: ambiguos, fuera de ADDED y duplicados.
	declared := map[string]bool{}
	seen := map[string]string{}
	for _, c := range caps {
		for _, section := range deltaSections(c.Delta) {
			for _, r := range section.reqs {
				spans := reqid.ScanProvisional(r.Name)
				if len(spans) == 0 {
					continue
				}
				fail := func(code, reason, continuation string) error {
					return &IDPlanError{Code: code, Capability: c.Name, Section: section.name, Requirement: r.Name, Token: spans[0].Token,
						File: c.File, Line: c.LineBase + r.Line, Reason: reason, Continuation: continuation}
				}
				distinct := map[string]bool{}
				for _, s := range spans {
					distinct[s.Token] = true
				}
				valid, _ := p.Scan(r.Name)
				if len(valid)+len(distinct) > 1 {
					return IDPlan{}, fail(CodeProvisionalAmbiguous, fmt.Sprintf("el nombre %q lleva un identificador provisional y otro identificador", r.Name),
						"deja en el nombre un solo identificador: el provisional, si el requisito es nuevo")
				}
				if section.name != "ADDED" {
					return IDPlan{}, fail(CodeProvisionalOutsideAdded, fmt.Sprintf("el provisional %s está en un requisito %s y solo un ADDED lo declara", spans[0].Token, section.name),
						"usa el identificador definitivo del requisito en MODIFIED, REMOVED y RENAMED")
				}
				if title, ok := reqid.ProvisionalInTitle(r.Name); !ok || title != spans[0].Token {
					return IDPlan{}, fail(CodeProvisionalMalformed, fmt.Sprintf("el provisional %s no está entre paréntesis al final del título", spans[0].Token),
						"escríbelo como `### Requirement: Título ("+reqid.ProvisionalText("<serie>", "<ordinal>")+")`")
				}
				if other, dup := seen[spans[0].Token]; dup {
					return IDPlan{}, fail(CodeProvisionalDuplicate, fmt.Sprintf("el provisional %s ya lo declara un ADDED de la capacidad %s", spans[0].Token, other),
						"usa un ordinal distinto para cada requisito nuevo de la misma serie")
				}
				seen[spans[0].Token] = c.Name
				declared[spans[0].Token] = true
			}
		}
	}
	for _, e := range in.Trace.Assignments {
		declared[e.Provisional] = true
	}

	// d. Las citas: todo provisional citado está declarado por un ADDED o por la traza.
	for _, ref := range refs {
		if !declared[ref.token] {
			return IDPlan{}, &IDPlanError{Code: CodeProvisionalUndeclared, Capability: ref.capability, Token: ref.token, File: ref.file, Line: ref.line,
				Reason:       fmt.Sprintf("se cita el provisional %s y ningún requisito ADDED del cambio ni la traza lo declara", ref.token),
				Continuation: "declara el requisito con `### Requirement: Título (" + ref.token + ")` en un ADDED o corrige la cita"}
		}
	}

	// e. La clasificación contra la canónica, la misma que usa el carril ODD.
	items := make([]classified, len(caps))
	for i, c := range caps {
		item, err := classifyIn(c.CapabilityDelta, c.Path, c.Existed, c.Canonical, p)
		if err != nil {
			return IDPlan{}, err
		}
		if err := checkKnownIDs(c, item, p); err != nil {
			return IDPlan{}, err
		}
		items[i] = item
	}

	// f. Las colisiones de los ADDED con identificador definitivo propio (A2).
	if err := checkDefinitiveAdded(in, items); err != nil {
		return IDPlan{}, err
	}

	// g. Los nombres de `--superseded-requirements`.
	superseded, err := resolveSuperseded(in.Superseded, caps, items)
	if err != nil {
		return IDPlan{}, err
	}

	plan, err := assign(in, items)
	if err != nil {
		return IDPlan{}, err
	}
	plan.Superseded = superseded
	return plan, nil
}

// deltaSections son las operaciones de una delta en el orden de la composición.
func deltaSections(d openspecdoc.Delta) []struct {
	name string
	reqs []openspecdoc.Requirement
} {
	return []struct {
		name string
		reqs []openspecdoc.Requirement
	}{{"ADDED", d.Added}, {"MODIFIED", d.Modified}, {"REMOVED", d.Removed}, {"RENAMED", d.Renamed}}
}

// lineAt es la línea (desde 1) de un desplazamiento en bytes dentro de un texto.
func lineAt(text string, offset int) int {
	if offset > len(text) {
		offset = len(text)
	}
	return strings.Count(text[:offset], "\n") + 1
}

// checkToken valida un token provisional contra el patrón y lo proyecta como error tipado.
func checkToken(p reqid.Pattern, token, capability, file string, line int) error {
	_, err := p.ParseProvisional(token)
	if err == nil {
		return nil
	}
	code, reason, continuation := reqid.CodeProvisionalMalformed, err.Error(), "escribe el provisional como `"+reqid.ProvisionalText("<clave>", "<ordinal>")+"` con los componentes que pide el patrón del proyecto"
	var perr *reqid.ProvisionalError
	if errors.As(err, &perr) {
		code, reason = perr.Code, perr.Error()
		if perr.Code == reqid.CodeProvisionalNewSeriesUnsupported {
			continuation = "fija el dominio en el provisional (por ejemplo `" + reqid.ProvisionalText("USU", "1") + "`): un dominio nuevo se escribe literal"
		}
	}
	return &IDPlanError{Code: code, Capability: capability, Token: token, File: file, Line: line, Reason: reason, Continuation: continuation}
}

// checkTraceFresh comprueba que cada entrada de la traza case con exactamente un ADDED de
// su capacidad, por su provisional o por su definitivo (id_trace_stale).
func checkTraceFresh(in IDPlanInput) error {
	for _, e := range in.Trace.Assignments {
		matches := 0
		for _, c := range in.Capabilities {
			if c.Name != e.Capability {
				continue
			}
			for _, r := range c.Delta.Added {
				if titleCarries(in.Pattern, r.Name, e) {
					matches++
				}
			}
		}
		if matches != 1 {
			return traceError(CodeIDTraceStale, "la traza de identificadores",
				fmt.Sprintf("la entrada %s → %s de la capacidad %s no casa con exactamente un requisito ADDED (casa con %d)", e.Provisional, e.Definitive, e.Capability, matches))
		}
	}
	return nil
}

// titleCarries dice si el título de un ADDED lleva el provisional o el definitivo de la entrada.
func titleCarries(p reqid.Pattern, name string, e TraceEntry) bool {
	if token, ok := reqid.ProvisionalInTitle(name); ok {
		return token == e.Provisional
	}
	return p.InTitle(name) == e.Definitive
}

// checkKnownIDs rechaza un MODIFIED o un RENAMED pendiente cuyo nombre lleva un
// identificador válido que ningún requisito de la canónica lleva (unknown_requirement_id).
func checkKnownIDs(c CapabilitySource, item classified, p reqid.Pattern) error {
	names := make([]string, len(item.canonical))
	for i, r := range item.canonical {
		names[i] = r.Name
	}
	unknown := func(section, name string, line int) error {
		id := p.InTitle(name)
		if id == "" {
			return nil
		}
		if _, match := openspecdoc.Resolve(names, name, p); match != openspecdoc.MatchNone {
			return nil
		}
		return &IDPlanError{Code: CodeUnknownRequirementID, Capability: c.Name, Section: section, Requirement: name, Token: id,
			File: c.File, Line: c.LineBase + line,
			Reason:       fmt.Sprintf("el requisito %q de %s lleva el identificador %s y la capacidad %s no tiene ningún requisito con él", name, section, id, c.Name),
			Continuation: "corrige el identificador o usa el título exacto del requisito de la spec viva"}
	}
	for _, r := range item.pending.Modified {
		if err := unknown("MODIFIED", r.Name, r.Line); err != nil {
			return err
		}
	}
	for _, r := range item.pending.Renamed {
		if oldName, _, ok := openspecdoc.SplitRename(r.Name); ok {
			if err := unknown("RENAMED", oldName, r.Line); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkDefinitiveAdded comprueba los ADDED pendientes con identificador definitivo y sin
// entrada en la traza: no existen en el registro ni se repiten en el propio cambio (A2,
// REQ-30.4). Los que la traza ya conoce los trata la reintegración (D5). Se compara por
// identidad (serie y número), no por cadena: con un patrón de ancho libre (`<m>`), `RF-1` y
// `RF-001` son el mismo identificador.
func checkDefinitiveAdded(in IDPlanInput, items []classified) error {
	reg, p := in.Registry, in.Pattern
	type tracedKey struct {
		capability string
		key        identityKey
	}
	traced := map[tracedKey]bool{}
	for _, e := range in.Trace.Assignments {
		if k, ok := reg.keyOf(e.Definitive); ok {
			traced[tracedKey{e.Capability, k}] = true
		}
	}
	taken := map[identityKey]bool{}
	for _, item := range items {
		for _, r := range item.pending.Added {
			if _, prov := reqid.ProvisionalInTitle(r.Name); !prov {
				if k, ok := reg.keyOf(p.InTitle(r.Name)); ok {
					taken[k] = true
				}
			}
		}
	}
	firstSeen := map[identityKey]string{}
	for _, item := range items {
		for _, r := range item.pending.Added {
			if _, prov := reqid.ProvisionalInTitle(r.Name); prov {
				continue
			}
			id := p.InTitle(r.Name)
			k, ok := reg.keyOf(id)
			if !ok || traced[tracedKey{item.cap.Name, k}] {
				continue
			}
			if found := reg.LookupIdentity(id); len(found) > 0 {
				return &CollisionError{ID: id, Where: found[0].Where(), NextFree: nextFree(reg, id, taken)}
			}
			if other, dup := firstSeen[k]; dup {
				return &CollisionError{
					ID:       id,
					Where:    fmt.Sprintf("el propio cambio o documento, en las capacidades %s y %s", other, item.cap.Name),
					NextFree: nextFree(reg, id, taken),
				}
			}
			firstSeen[k] = item.cap.Name
		}
	}
	return nil
}

// resolveSuperseded resuelve los nombres de `--superseded-requirements` (D4): con el prefijo
// `<capacidad>/` (solo si es una capacidad del cambio) o por título exacto en la canónica
// de exactamente una capacidad.
func resolveSuperseded(names []string, caps []CapabilitySource, items []classified) (map[string][]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	isCapability := map[string]bool{}
	for _, c := range caps {
		isCapability[c.Name] = true
	}
	out := map[string][]string{}
	for _, raw := range names {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		prefix, title := "", raw
		if i := strings.Index(raw, "/"); i > 0 && isCapability[raw[:i]] {
			prefix, title = raw[:i], raw[i+1:]
		}
		var holders []string
		for _, item := range items {
			if prefix != "" && item.cap.Name != prefix {
				continue
			}
			for _, r := range item.canonical {
				if r.Name == title {
					holders = append(holders, item.cap.Name)
					break
				}
			}
		}
		switch {
		case len(holders) == 0:
			return nil, &IDPlanError{Code: CodeSupersededRequirementUnknown, Capability: prefix, Requirement: title,
				Reason:       fmt.Sprintf("el requisito superado %q no existe en la spec viva de las capacidades del cambio", raw),
				Continuation: "usa el título exacto de un requisito de la canónica, con el prefijo `<capacidad>/` si hace falta"}
		case len(holders) > 1:
			options := make([]string, len(holders))
			for i, h := range holders {
				options[i] = h + "/" + title
			}
			return nil, &IDPlanError{Code: CodeSupersededRequirementAmbiguous, Requirement: title,
				Reason:       fmt.Sprintf("el requisito superado %q existe en varias capacidades (%s)", raw, strings.Join(holders, ", ")),
				Continuation: "repite el nombre con el prefijo de su capacidad: " + strings.Join(options, " o ")}
		}
		out[holders[0]] = append(out[holders[0]], title)
	}
	return out, nil
}

// assign ocupa los identificadores existentes y asigna los números (D3 pasos 3 y 4, D5).
func assign(in IDPlanInput, items []classified) (IDPlan, error) {
	p, reg := in.Pattern, in.Registry
	allocFail := func(err error) error {
		return &IDPlanError{Code: CodeIDAllocationFailed, Reason: err.Error(),
			Continuation: "corrige el identificador que desborda o no es válido para el patrón y repite la integración"}
	}

	alloc := reg.alloc.Clone()
	var adds []*addedItem
	for ci, c := range in.Capabilities {
		for _, r := range c.Delta.Added {
			it := &addedItem{capIdx: ci, name: c.Name, req: r, entry: -1}
			if token, ok := reqid.ProvisionalInTitle(r.Name); ok {
				it.token = token
				it.pv, _ = p.ParseProvisional(token) // validado en el paso b
			} else {
				it.def = p.InTitle(r.Name)
			}
			if it.token == "" && it.def == "" {
				continue
			}
			for ei, e := range in.Trace.Assignments {
				if e.Capability == c.Name && titleCarries(p, r.Name, e) {
					it.entry = ei
				}
			}
			adds = append(adds, it)
		}
	}
	for _, it := range adds {
		if it.def != "" {
			if err := alloc.Occupy(it.def); err != nil {
				return IDPlan{}, allocFail(err)
			}
		}
	}
	for _, e := range in.Trace.Assignments {
		if err := alloc.Occupy(e.Definitive); err != nil {
			return IDPlan{}, allocFail(err)
		}
	}

	// Reintegración (D5): qué hacer con cada entrada de la traza.
	var pairs []reqid.Assignment
	for _, e := range in.Trace.Assignments {
		pairs = append(pairs, reqid.Assignment{Provisional: e.Provisional, Definitive: e.Definitive})
	}
	traceMap := reqid.NewIDMap(pairs)
	reassignLabels := map[string]bool{}
	for _, it := range adds {
		if it.entry < 0 {
			continue
		}
		e := in.Trace.Assignments[it.entry]
		if err := resolveTraced(it, e, items[it.capIdx], reg, traceMap, p); err != nil {
			return IDPlan{}, err
		}
		if it.state == StateReassigned {
			if label, ok := labelOf(p, e.Provisional); ok {
				reassignLabels[label] = true
			}
		}
	}
	for _, it := range adds {
		if it.entry >= 0 && it.state == StateKept {
			e := in.Trace.Assignments[it.entry]
			if label, ok := labelOf(p, e.Provisional); ok && reassignLabels[label] {
				// Toda la etiqueta se reasigna a una serie nueva, aunque este definitivo esté libre.
				it.state, it.final = StateReassigned, ""
				it.prev = append(append([]string(nil), e.Previous...), e.Definitive)
			}
		}
	}

	// Series fijadas primero y etiquetas después: una serie nueva no recae en una fijada.
	for _, it := range adds {
		switch {
		case it.entry < 0 && it.token != "":
			it.state = StateAssigned
			if series, ok := it.pv.Series(); ok {
				id, err := alloc.Next(series)
				if err != nil {
					return IDPlan{}, allocFail(err)
				}
				it.final = id
			}
		case it.entry >= 0 && it.state == StateReassigned:
			e := in.Trace.Assignments[it.entry]
			if _, labelled := labelOf(p, e.Provisional); !labelled {
				series, ok := p.Series(e.Definitive)
				if !ok {
					return IDPlan{}, allocFail(fmt.Errorf("el identificador %s no tiene serie en el patrón %s", e.Definitive, p.Template()))
				}
				id, err := alloc.Next(series)
				if err != nil {
					return IDPlan{}, allocFail(err)
				}
				it.final = id
			}
		}
	}
	labelSeries := map[string]reqid.Series{}
	for _, it := range adds {
		if it.entry >= 0 && it.state == StateKept {
			e := in.Trace.Assignments[it.entry]
			if label, ok := labelOf(p, e.Provisional); ok {
				if series, ok := p.Series(e.Definitive); ok {
					if _, set := labelSeries[label]; !set {
						labelSeries[label] = series
					}
				}
			}
		}
	}
	for _, it := range adds {
		if it.final != "" || it.state == StateKept || it.entry < 0 && it.token == "" {
			continue
		}
		provisional := it.token
		if it.entry >= 0 {
			provisional = in.Trace.Assignments[it.entry].Provisional
		}
		pv, err := p.ParseProvisional(provisional)
		if err != nil {
			return IDPlan{}, allocFail(err)
		}
		label, ok := pv.Label()
		if !ok {
			continue
		}
		series, set := labelSeries[label]
		if !set {
			if series, err = alloc.NewSeries(pv); err != nil {
				return IDPlan{}, allocFail(err)
			}
			labelSeries[label] = series
		}
		id, err := alloc.Next(series)
		if err != nil {
			return IDPlan{}, allocFail(err)
		}
		it.final = id
	}

	return buildPlan(in, adds, items)
}

// labelOf devuelve la etiqueta de serie nueva de un provisional.
func labelOf(p reqid.Pattern, provisional string) (string, bool) {
	pv, err := p.ParseProvisional(provisional)
	if err != nil {
		return "", false
	}
	return pv.Label()
}

// resolveTraced decide el estado de un ADDED con entrada en la traza (tabla de D5).
func resolveTraced(it *addedItem, e TraceEntry, item classified, reg *Registry, traceMap reqid.IDMap, p reqid.Pattern) error {
	refs := reg.LookupIdentity(e.Definitive)
	want := openspecdoc.NormalizeBody(afterHeading(traceMap.Substitute(it.req.Body)))
	var own *openspecdoc.Requirement
	bodyEqual := false
	for i := range item.canonical {
		r := &item.canonical[i]
		if p.InTitle(r.Name) != e.Definitive {
			continue
		}
		if openspecdoc.NormalizeBody(afterHeading(r.Body)) == want {
			own, bodyEqual = r, true
			break
		}
		if own == nil && r.Name == e.Requirement {
			own = r
		}
	}
	it.own = own
	if own != nil && !bodyEqual {
		return &DivergentRequirementError{Capability: it.name, Requirement: it.req.Name, ID: e.Definitive, Path: item.path}
	}
	foreign := false
	for _, ref := range refs {
		if own == nil || ref.Kind != RefSpec || ref.Capability != it.name || ref.Requirement != own.Name {
			foreign = true
		}
	}
	if foreign {
		it.state = StateReassigned
		it.prev = append(append([]string(nil), e.Previous...), e.Definitive)
		return nil
	}
	it.state, it.final = StateKept, e.Definitive
	return nil
}

// afterHeading es el bloque de un requisito sin su línea de título.
func afterHeading(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	if i := strings.Index(body, "\n"); i >= 0 {
		return body[i+1:]
	}
	return ""
}

// buildPlan reúne el mapa, las entradas, la canónica editada y los identificadores escritos.
func buildPlan(in IDPlanInput, adds []*addedItem, items []classified) (IDPlan, error) {
	var pairs []reqid.Assignment
	for _, it := range adds {
		if it.def != "" && it.entry < 0 {
			continue
		}
		var e TraceEntry
		provisional := it.token
		if it.entry >= 0 {
			e = in.Trace.Assignments[it.entry]
			provisional = e.Provisional
		}
		if it.state == StateReassigned && it.final == "" {
			return IDPlan{}, &IDPlanError{Code: CodeIDAllocationFailed, Capability: it.name, Requirement: it.req.Name,
				Reason: "no se pudo asignar el identificador de " + provisional, Continuation: "repite la integración"}
		}
		pairs = append(pairs, reqid.Assignment{Provisional: provisional, Definitive: it.final})
		if it.token != "" && it.token != provisional {
			pairs = append(pairs, reqid.Assignment{Provisional: it.token, Definitive: it.final})
		}
		if it.state == StateReassigned {
			pairs = append(pairs, reqid.Assignment{Provisional: e.Definitive, Definitive: it.final})
		}
	}
	plan := IDPlan{Map: reqid.NewIDMap(pairs)}

	written := map[string]bool{}
	for _, it := range adds {
		if it.entry < 0 && it.token == "" {
			written[it.def] = true
			continue
		}
		var entry TraceEntry
		provisional := it.token
		if it.entry >= 0 {
			entry = in.Trace.Assignments[it.entry]
			provisional = entry.Provisional
		}
		requirement := plan.Map.Substitute(it.req.Name)
		if it.state == StateKept && entry.Requirement != "" {
			requirement = entry.Requirement
		}
		plan.Entries = append(plan.Entries, PlannedEntry{
			TraceEntry: TraceEntry{Provisional: provisional, Definitive: it.final, Capability: it.name, Requirement: requirement, Previous: it.prev},
			State:      it.state,
		})
		written[it.final] = true

		if it.state == StateReassigned && it.own != nil {
			text, ok := plan.Canonical[it.name]
			if !ok {
				text = items[it.capIdx].previous
			}
			if strings.Contains(text, it.own.Body) {
				if plan.Canonical == nil {
					plan.Canonical = map[string]string{}
				}
				plan.Canonical[it.name] = strings.Replace(text, it.own.Body, plan.Map.Substitute(it.own.Body), 1)
			}
		}
	}
	for id := range written {
		plan.Written = append(plan.Written, id)
	}
	sort.Strings(plan.Written)
	return plan, nil
}

// Duplicate es un identificador que aparece más de una vez en la spec viva resultante.
// Blocking es verdadero si lo escribe la integración; los históricos ajenos solo se
// informan (REQ-47.5).
type Duplicate struct {
	ID           string   `json:"id"`
	Capabilities []string `json:"capabilities"`
	Blocking     bool     `json:"blocking"`
}

// checkDuplicates cuenta las apariciones de cada identificador en la spec viva resultante:
// las capacidades que la integración reescribe (resulting, capacidad → texto) y el resto
// del registro. Más de una aparición de un identificador que escribe la integración
// (written) da duplicate_assigned_id; los demás duplicados se devuelven con Blocking falso.
func checkDuplicates(reg *Registry, resulting map[string]string, written []string) ([]Duplicate, error) {
	where := map[string][]string{}
	for id, refs := range reg.byID {
		for _, ref := range refs {
			if _, replaced := resulting[ref.Capability]; ref.Kind != RefSpec || replaced {
				continue
			}
			where[id] = append(where[id], ref.Capability)
		}
	}
	for capability, text := range resulting {
		for _, r := range openspecdoc.ParseCanonical(text) {
			if id := reg.pattern.InTitle(r.Name); id != "" {
				where[id] = append(where[id], capability)
			}
		}
	}
	writtenSet := map[string]bool{}
	for _, id := range written {
		writtenSet[id] = true
	}
	var ids []string
	for id, list := range where {
		if len(list) > 1 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	var out []Duplicate
	var first *IDPlanError
	for _, id := range ids {
		list := append([]string(nil), where[id]...)
		sort.Strings(list)
		d := Duplicate{ID: id, Capabilities: list, Blocking: writtenSet[id]}
		out = append(out, d)
		if d.Blocking && first == nil {
			first = &IDPlanError{Code: CodeDuplicateAssignedID, Capability: list[0], Token: id,
				Reason:       fmt.Sprintf("el identificador %s que escribe la integración aparece %d veces en la spec viva (%s)", id, len(list), strings.Join(list, ", ")),
				Continuation: "restaura `openspec/specs/<capacidad>/spec.md` desde la base y reintegra: la traza conserva el definitivo y la reintegración reasigna el que choca"}
		}
	}
	if first != nil {
		return out, first
	}
	return out, nil
}
