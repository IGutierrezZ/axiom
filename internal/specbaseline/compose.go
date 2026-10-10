package specbaseline

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// supersededMarker es la marca que el motor nativo añade al título de un
// requisito superado.
const supersededMarker = " [SUPERSEDED / DEPRECADO]"

// explicitSupersededReason es el motivo que el motor nativo registra para los
// requisitos indicados con --superseded-requirements.
const explicitSupersededReason = "Superado por consolidación de arquitectura"

type (
	// EntryKind es el tipo de una entrada de la traza.
	EntryKind string
	// TraceAction es la acción que registra una entrada de la traza.
	TraceAction string
	// TraceState es el estado de una entrada de la traza.
	TraceState string
)

const (
	// EntryCompose es una entrada de composición con perfil.
	EntryCompose EntryKind = "compose"
	// EntryDirected es una entrada de edición dirigida de un documento de prosa.
	EntryDirected EntryKind = "directed"
	// EntryRegister es una entrada de solo registro.
	EntryRegister EntryKind = "register"
	// EntryContext es una entrada del contexto técnico mantenido.
	EntryContext EntryKind = "context"

	// ActionAdded registra un requisito añadido.
	ActionAdded TraceAction = "added"
	// ActionModified registra un requisito modificado (o renombrado).
	ActionModified TraceAction = "modified"
	// ActionRemoved registra un requisito eliminado o superado.
	ActionRemoved TraceAction = "removed"
	// ActionCreated registra un documento creado.
	ActionCreated TraceAction = "created"

	// StatePending marca una entrada aún no integrada.
	StatePending TraceState = "pending"
	// StateIntegrated marca una entrada ya integrada.
	StateIntegrated TraceState = "integrated"
)

// TraceEntry es una entrada de la traza de integración (REQ-24.52). La
// composición produce las de tipo `compose`, en estado `pending`: pasarlas a
// `integrated` corresponde a quien escribe el documento.
type TraceEntry struct {
	ID               string      `yaml:"id" json:"id"`
	Provisional      string      `yaml:"provisional,omitempty" json:"provisional,omitempty"`
	Kind             EntryKind   `yaml:"kind" json:"kind"`
	Domain           string      `yaml:"domain,omitempty" json:"domain,omitempty"`
	Document         string      `yaml:"document" json:"document"` // relativo a workspaceRoot, con '/'
	Anchor           string      `yaml:"anchor" json:"anchor"`
	Action           TraceAction `yaml:"action" json:"action"`
	State            TraceState  `yaml:"state" json:"state"`
	Reason           string      `yaml:"reason,omitempty" json:"reason,omitempty"`
	ScenariosOmitted bool        `yaml:"scenarios_omitted,omitempty" json:"scenariosOmitted,omitempty"`
	Warnings         []string    `yaml:"warnings,omitempty" json:"warnings,omitempty"`
}

// PlannedDocument es un documento con perfil que participa en la composición
// del cambio.
type PlannedDocument struct {
	Domain   string
	Path     string // absoluta
	RelPath  string // relativa a workspaceRoot, con '/'
	Profile  Profile
	Original []byte // bytes del árbol de trabajo
	Delta    []byte
}

// ComposeInput reúne lo necesario para componer un documento con perfil. IDs es
// la correspondencia de IDs provisionales a definitivos de todo el cambio, que
// calcula PlanIDs una sola vez (REQ-24.59): sin ella no se puede componer un ADDED
// con identidad `id`, y con ella se sustituyen los IDs provisionales en el texto
// que se integra.
type ComposeInput struct {
	Document           PlannedDocument
	IDs                IDMap
	Supersede          bool
	ExplicitSuperseded []string
}

// ComposeResult es el resultado de componer un documento. Output son los bytes
// del documento compuesto; la composición nunca escribe nada.
type ComposeResult struct {
	Output   []byte
	Entries  []TraceEntry // una por requisito del delta
	Warnings []string     // colas de secciones contiguas (REQ-24.41) y avisos no bloqueantes
}

// ComposeError describe por qué un delta no se puede componer (REQ-24.46): el
// dominio, la sección del delta, el requisito, el motivo y la continuación. Quien
// llama debe rechazar sin escribir: nunca hay una composición parcial.
type ComposeError struct {
	Domain       string
	Section      string // ADDED, MODIFIED, REMOVED, RENAMED, DELTA o DOCUMENTO
	Requirement  string
	Reason       string
	Continuation string
}

func (e *ComposeError) Error() string {
	var b strings.Builder
	b.WriteString("sdd-archive-compose: ")
	if e.Domain != "" {
		fmt.Fprintf(&b, "dominio %q: ", e.Domain)
	}
	if e.Section != "" {
		fmt.Fprintf(&b, "sección %s", e.Section)
		if e.Requirement != "" {
			fmt.Fprintf(&b, ", requisito %q", e.Requirement)
		}
		b.WriteString(": ")
	}
	b.WriteString(e.Reason)
	if e.Continuation != "" {
		b.WriteString(". Continuación: ")
		b.WriteString(e.Continuation)
	}
	return b.String()
}

// segment es un tramo contiguo del documento: el bloque de un requisito o el
// texto que no pertenece a ningún bloque (preámbulo, secciones entre requisitos,
// cola y final). Los tramos particionan el documento sin huecos, de modo que
// volver a emitirlos en orden reproduce sus bytes.
type segment struct {
	lines      []Line
	requisite  *Requirement // nil en el texto ajeno a todo bloque
	id         string
	title      string
	key        string
	superseded bool
	// added marca el bloque de un requisito ADDED recién insertado; anchor es la
	// ubicación que declaró, para que los ADDED con la misma ancla queden en el
	// orden del delta (D-11).
	added  bool
	anchor string
}

// composer lleva el estado de una composición en memoria.
type composer struct {
	input    ComposeInput
	profile  Profile
	domain   string
	eol      string
	segments []segment
	warnings []string
	entries  []TraceEntry
	// removed son los identificadores que un REMOVED del mismo delta eliminó sin
	// --supersede: no pueden ser el ancla de un ADDED.
	removed map[string]bool
}

// ComposeDocument compone el delta sobre el documento con el perfil de
// requisito, en memoria y sin escribir (D-08, D-12). Aplica RENAMED, MODIFIED y
// REMOVED por identificador, en el mismo orden que el motor nativo; todo lo que no
// pertenece a un bloque editado conserva sus bytes (REQ-24.45). Devuelve
// *ComposeError ante cualquier delta no aplicable: nunca una composición parcial.
//
// Después de ellas inserta los ADDED en el orden del delta, cada uno en la
// ubicación que declara, y sustituye los IDs provisionales del cambio por los
// definitivos en el texto que integra (D-11). Con `scenarios: omit` integra los
// requisitos sin sus escenarios (D-10).
func ComposeDocument(in ComposeInput) (ComposeResult, error) {
	document := in.Document
	c := &composer{input: in, profile: document.Profile, domain: document.Domain, removed: map[string]bool{}}
	if c.profile.Heading == nil {
		return ComposeResult{}, missingProfileError(c.domain)
	}

	parsed := ParseDocument(document.Original)
	c.eol = parsed.EOL()
	c.segments = splitSegments(parsed, c.profile)

	deltaDoc := parseDelta(document.Delta)
	if deltaDoc.empty() {
		return ComposeResult{}, c.fail("DELTA", "", "el delta no declara requisitos ADDED, MODIFIED, REMOVED ni RENAMED", "escribe el delta en el formato nativo: secciones '## ADDED Requirements', '## MODIFIED Requirements', '## REMOVED Requirements' o '## RENAMED Requirements' con un bloque '### Requirement: <nombre>' por requisito")
	}

	for _, block := range deltaDoc.Renamed {
		if err := c.rename(block); err != nil {
			return ComposeResult{}, err
		}
	}
	for _, block := range deltaDoc.Modified {
		if err := c.modify(block); err != nil {
			return ComposeResult{}, err
		}
	}
	for _, block := range deltaDoc.Removed {
		if err := c.remove(block); err != nil {
			return ComposeResult{}, err
		}
	}
	if err := c.supersedeExplicit(); err != nil {
		return ComposeResult{}, err
	}
	for _, block := range deltaDoc.Added {
		if err := c.add(block); err != nil {
			return ComposeResult{}, err
		}
	}

	var lines []Line
	for _, seg := range c.segments {
		lines = append(lines, seg.lines...)
	}
	return ComposeResult{
		Output:   Document{BOM: parsed.BOM, Lines: lines}.Bytes(),
		Entries:  c.entries,
		Warnings: c.warnings,
	}, nil
}

// splitSegments particiona el documento en tramos: un tramo por bloque de
// requisito y un tramo de texto ajeno entre ellos.
func splitSegments(doc Document, profile Profile) []segment {
	requirements := doc.Requirements(profile)
	var segments []segment
	cursor := 0
	for i := range requirements {
		requirement := &requirements[i]
		if cursor < requirement.Start {
			segments = append(segments, segment{lines: doc.Lines[cursor:requirement.Start]})
		}
		segments = append(segments, segment{
			lines:     doc.Lines[requirement.Start:requirement.End],
			requisite: requirement,
			id:        requirement.ID,
			title:     requirement.Title,
			key:       requirement.Key,
		})
		cursor = requirement.End
	}
	if cursor < len(doc.Lines) {
		segments = append(segments, segment{lines: doc.Lines[cursor:]})
	}
	return segments
}

func (c *composer) fail(section, requirement, reason, continuation string) *ComposeError {
	return &ComposeError{Domain: c.domain, Section: section, Requirement: requirement, Reason: reason, Continuation: continuation}
}

// parseName reconstruye el encabezado de un requisito del delta y extrae su id,
// su título y su identificador (REQ-24.39). Rechaza un nombre que no cumple el
// patrón `heading`.
func (c *composer) parseName(section, name string) (id, title, key string, err error) {
	return parseRequirementName(c.domain, c.profile, section, name)
}

// parseRequirementName es parseName sin el estado de una composición: lo
// comparten la composición y PlanIDs, que leen el nombre de un requisito del
// delta con las mismas reglas.
func parseRequirementName(domain string, profile Profile, section, name string) (id, title, key string, err error) {
	fail := func(reason, continuation string) error {
		return &ComposeError{Domain: domain, Section: section, Requirement: name, Reason: reason, Continuation: continuation}
	}
	line := profile.HeadingLine(name)
	id, title, ok := profile.Match(line)
	if !ok {
		return "", "", "", fail(
			fmt.Sprintf("el nombre del requisito no cumple el patrón 'heading' %q (se evalúa la línea %q)", profile.Heading.String(), line),
			"escribe el nombre como el encabezado del proyecto sin los '#' iniciales ni el espacio que los sigue, de modo que los '#' del nivel, un espacio y el nombre cumplan 'heading'")
	}
	key = profile.Key(id, title)
	if strings.TrimSpace(key) == "" {
		return "", "", "", fail(
			fmt.Sprintf("el nombre del requisito no define el %s con el que se identifica", identityLabel(profile)),
			"incluye en el nombre el identificador del requisito, según el patrón 'heading' del perfil")
	}
	return id, title, key, nil
}

// missingProfileError es el rechazo de un documento sin perfil compilado.
func missingProfileError(domain string) error {
	return &ComposeError{Domain: domain, Section: "DOCUMENTO", Reason: "el documento no tiene un perfil de requisito compilado", Continuation: "declara 'requirement' en baseline.documents del dominio en axiom.yaml"}
}

func identityLabel(p Profile) string {
	if p.Identity == workspace.IdentityText {
		return "título"
	}
	return "id"
}

// find localiza el único tramo de requisito con ese identificador. Un
// identificador inexistente o duplicado se rechaza (REQ-24.40): el rechazo de un
// duplicado nombra todas sus apariciones y no afecta a las operaciones sobre
// identificadores no ambiguos.
func (c *composer) find(section, name, key string) (int, error) {
	var found []int
	for i, seg := range c.segments {
		if seg.requisite != nil && !seg.superseded && seg.key == key {
			found = append(found, i)
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return -1, c.fail(section, name,
			fmt.Sprintf("el documento no contiene ningún requisito con %s %q", identityLabel(c.profile), key),
			"comprueba el identificador en el documento de la baseline o retira la operación del delta")
	}
	return -1, c.ambiguous(section, name, key, found)
}

// ambiguous es el rechazo de un identificador que aparece más de una vez: nombra
// todas sus apariciones, con su línea y su encabezado (REQ-24.40).
func (c *composer) ambiguous(section, name, key string, found []int) error {
	occurrences := make([]string, len(found))
	for i, at := range found {
		requisite := c.segments[at].requisite
		occurrences[i] = fmt.Sprintf("línea %d %q", requisite.LineNumber(), requisite.Line)
	}
	return c.fail(section, name,
		fmt.Sprintf("el identificador %q es ambiguo: aparece %d veces en el documento (%s)", key, len(found), strings.Join(occurrences, "; ")),
		"normaliza los identificadores duplicados en el repositorio del proyecto, en un commit propio, y vuelve a ejecutar la composición")
}

// tailWarnings emite el aviso no bloqueante de REQ-24.41 por cada sección de la
// cola contigua de un requisito sobre el que actúa MODIFIED o REMOVED.
func (c *composer) tailWarnings(section string, seg segment) []string {
	var warnings []string
	for _, tail := range seg.requisite.Tail {
		location := fmt.Sprintf("línea %d", tail.LineNumber())
		if c.input.Document.RelPath != "" {
			location += " de " + c.input.Document.RelPath
		}
		warnings = append(warnings, fmt.Sprintf("dominio %q: la operación %s sobre %q deja intacta la sección contigua %q (%s); comprueba si es una subsección del requisito o una sección independiente",
			c.domain, section, seg.key, tail.Line, location))
	}
	c.warnings = append(c.warnings, warnings...)
	return warnings
}

// render emite el encabezado de un requisito sustituyendo en él los IDs
// provisionales del cambio: el texto del delta que se integra nunca conserva uno.
func (c *composer) render(id, title string) string {
	return c.input.IDs.Substitute(c.profile.RenderHeading(id, title))
}

func (c *composer) entry(key, anchor string, action TraceAction, reason string, warnings []string) {
	c.entries = append(c.entries, TraceEntry{
		ID:       key,
		Kind:     EntryCompose,
		Domain:   c.domain,
		Document: c.input.Document.RelPath,
		Anchor:   anchor,
		Action:   action,
		State:    StatePending,
		Reason:   reason,
		Warnings: warnings,
	})
}

// rename aplica un RENAMED. Con identidad `id` se rechaza: cambiar el título es
// un MODIFIED y renumerar no está soportado. Con identidad `text` se comporta
// como el motor nativo, tomando el título como nombre.
func (c *composer) rename(block deltaBlock) error {
	if c.profile.Identity != workspace.IdentityText {
		return c.fail(sectionRenamed, block.Name,
			"con identidad 'id' no se admite RENAMED: renumerar un requisito no está soportado",
			"usa MODIFIED para cambiar el título del requisito, que conserva su id")
	}
	oldName, newName, ok := splitRenamePair(block.Name)
	if !ok {
		return c.fail(sectionRenamed, block.Name, `el encabezado debe indicar "Nombre antiguo → Nombre nuevo"`, `escribe el nombre como "<antiguo> → <nuevo>"`)
	}
	_, _, oldKey, err := c.parseName(sectionRenamed, oldName)
	if err != nil {
		return err
	}
	newID, newTitle, newKey, err := c.parseName(sectionRenamed, newName)
	if err != nil {
		return err
	}
	at, err := c.find(sectionRenamed, block.Name, oldKey)
	if err != nil {
		return err
	}
	if !block.HasReason {
		return c.fail(sectionRenamed, block.Name, `falta la nota obligatoria "(Reason: ...)"`, `añade al bloque una línea "(Reason: <motivo>)"`)
	}
	for i, seg := range c.segments {
		if i != at && seg.requisite != nil && !seg.superseded && seg.key == newKey {
			return c.fail(sectionRenamed, newName, fmt.Sprintf("el nombre de destino %q ya existe en el documento", newKey), "elige un nombre que no exista o usa MODIFIED sobre el requisito existente")
		}
	}

	seg := &c.segments[at]
	heading := c.render(newID, newTitle)
	seg.lines = append([]Line{{Text: heading, EOL: seg.lines[0].EOL}}, seg.lines[1:]...)
	seg.id, seg.title, seg.key = newID, newTitle, newKey
	c.entry(newKey, heading, ActionModified, c.input.IDs.Substitute(fmt.Sprintf("renombrado desde %q: %s", oldKey, block.Reason)), nil)
	return nil
}

// modify sustituye el bloque de un requisito por el cuerpo del delta (REQ-24.44).
func (c *composer) modify(block deltaBlock) error {
	id, title, key, err := c.parseName(sectionModified, block.Name)
	if err != nil {
		return err
	}
	at, err := c.find(sectionModified, block.Name, key)
	if err != nil {
		return err
	}
	body, omitted, err := c.integratedBody(sectionModified, block)
	if err != nil {
		return err
	}

	seg := &c.segments[at]
	heading := c.render(id, title)
	warnings := c.tailWarnings(sectionModified, *seg)
	seg.lines = c.replacement(seg.lines, heading, body)
	seg.id, seg.title = id, title
	c.entry(key, heading, ActionModified, "", warnings)
	c.entries[len(c.entries)-1].ScenariosOmitted = omitted
	return nil
}

// integratedBody devuelve el cuerpo que se integra de un requisito ADDED o
// MODIFIED: sin los escenarios con `scenarios: omit` (D-10), comprobado contra los
// encabezados que cerrarían el bloque (D-09: la regla se aplica al texto que se
// integra) y con los IDs provisionales ya sustituidos. Informa de si omitió algún
// escenario.
func (c *composer) integratedBody(section string, block deltaBlock) ([]deltaLine, bool, error) {
	body, omitted := block.Body, false
	if c.profile.Scenarios == workspace.ScenariosOmit {
		body, omitted = omitScenarios(body)
	}
	if err := c.checkBody(section, block.Name, body); err != nil {
		return nil, false, err
	}
	integrated := make([]deltaLine, len(body))
	for i, line := range body {
		integrated[i] = deltaLine{Text: c.input.IDs.Substitute(line.Text), Number: line.Number}
	}
	return integrated, omitted, nil
}

// checkBody rechaza el cuerpo de un delta que contiene, fuera de vallas, un
// encabezado de nivel menor o igual que el de los requisitos (D-09, REQ-24.42).
func (c *composer) checkBody(section, name string, body []deltaLine) error {
	violations := bodyViolations(body, c.profile.Level)
	if len(violations) == 0 {
		return nil
	}
	found := make([]string, len(violations))
	for i, violation := range violations {
		found[i] = fmt.Sprintf("%q (línea %d del delta, nivel %d)", violation.Text, violation.Number, violation.Level)
	}
	return c.fail(section, name,
		fmt.Sprintf("el cuerpo contiene, fuera de bloques de código, un encabezado de nivel menor o igual que el nivel %d de los requisitos: %s; cerraría el bloque del requisito y dejaría el resto del cuerpo fuera de él", c.profile.Level, strings.Join(found, ", ")),
		fmt.Sprintf("rebaja el encabezado a un nivel mayor que %d o conviértelo en texto", c.profile.Level))
}

// replacement construye las líneas de un bloque sustituido: el encabezado
// renderizado, el cuerpo del delta y la serie de líneas en blanco finales del
// bloque original (D-12). Las líneas nuevas usan el final de línea del
// encabezado que sustituyen; si el bloque original terminaba el documento sin
// salto de línea, el nuevo también.
func (c *composer) replacement(original []Line, heading string, body []deltaLine) []Line {
	eol := original[0].EOL
	if eol == "" {
		eol = c.eol
	}
	lines := make([]Line, 0, len(body)+2)
	lines = append(lines, Line{Text: heading, EOL: eol})
	for _, line := range body {
		lines = append(lines, Line{Text: line.Text, EOL: eol})
	}
	blanks := trailingBlank(original)
	lines = append(lines, original[len(original)-blanks:]...)
	if last := original[len(original)-1]; last.EOL == "" {
		lines[len(lines)-1].EOL = ""
	}
	return lines
}

// trailingBlank cuenta las líneas en blanco (solo espacios o tabuladores) al
// final de un bloque, sin contar su primera línea, que es el encabezado.
func trailingBlank(lines []Line) int {
	count := 0
	for i := len(lines) - 1; i >= 1; i-- {
		if strings.TrimSpace(lines[i].Text) != "" {
			break
		}
		count++
	}
	return count
}

// remove elimina el bloque de un requisito o, con --supersede, lo conserva
// marcado como superado (REQ-24.44). La cola contigua no se toca.
func (c *composer) remove(block deltaBlock) error {
	_, _, key, err := c.parseName(sectionRemoved, block.Name)
	if err != nil {
		return err
	}
	at, err := c.find(sectionRemoved, block.Name, key)
	if err != nil {
		return err
	}
	if !block.HasReason {
		return c.fail(sectionRemoved, block.Name, `falta la nota obligatoria "(Reason: ...)"`, `añade al bloque una línea "(Reason: <motivo>)"`)
	}

	seg := &c.segments[at]
	warnings := c.tailWarnings(sectionRemoved, *seg)
	reason := c.input.IDs.Substitute(block.Reason)
	if c.input.Supersede {
		anchor := c.supersede(seg, reason)
		c.entry(key, anchor, ActionRemoved, reason, warnings)
		return nil
	}
	anchor := seg.requisite.Line
	c.segments = append(c.segments[:at], c.segments[at+1:]...)
	c.removed[key] = true
	c.entry(key, anchor, ActionRemoved, reason, warnings)
	return nil
}

// supersede deja el bloque marcado como superado con la misma nota que el motor
// nativo: encabezado renderizado más la marca, la nota de advertencia con el
// motivo y el resto del bloque original. Devuelve el encabezado resultante.
func (c *composer) supersede(seg *segment, reason string) string {
	eol := seg.lines[0].EOL
	if eol == "" {
		eol = c.eol
	}
	heading := c.profile.RenderHeading(seg.id, seg.title) + supersededMarker
	lines := []Line{
		{Text: heading, EOL: eol},
		{Text: "> [!CAUTION]", EOL: eol},
		{Text: "> **REQUISITO HISTÓRICO SUPERADO / DEPRECADO:** Este requisito ha sido superado o deprecado.", EOL: eol},
		{Text: "> Motivo: " + reason, EOL: eol},
		{Text: "", EOL: eol},
	}
	lines = append(lines, seg.lines[1:]...)
	if last := seg.lines[len(seg.lines)-1]; last.EOL == "" {
		lines[len(lines)-1].EOL = ""
	}
	seg.lines = lines
	seg.superseded = true
	return heading
}

// supersedeExplicit marca los requisitos indicados con --superseded-requirements:
// IDs con identidad `id` o títulos con identidad `text`. Como en el motor nativo,
// un requisito que no existe no detiene la composición; aquí se avisa, para que
// un identificador mal escrito no pase desapercibido. Uno ya marcado por un
// REMOVED con --supersede del mismo delta no se vuelve a marcar.
func (c *composer) supersedeExplicit() error {
	for _, raw := range c.input.ExplicitSuperseded {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		var alreadyMarked bool
		var found []int
		for i, seg := range c.segments {
			if seg.requisite == nil || seg.key != key {
				continue
			}
			if seg.superseded {
				alreadyMarked = true
				continue
			}
			found = append(found, i)
		}
		switch {
		case len(found) > 1:
			_, err := c.find("--superseded-requirements", key, key)
			return err
		case len(found) == 1:
			c.supersede(&c.segments[found[0]], explicitSupersededReason)
		case !alreadyMarked:
			c.warnings = append(c.warnings, fmt.Sprintf("dominio %q: el requisito %q indicado en --superseded-requirements no existe en el documento y no se marca", c.domain, key))
		}
	}
	return nil
}

var (
	// placementAfter y placementSection son las dos formas de la nota
	// (Placement: ...) de un requisito ADDED (D-11).
	placementAfter   = regexp.MustCompile(`^after[ \t]+(\S.*)$`)
	placementSection = regexp.MustCompile(`^section[ \t]+"(.+)"$`)
)

type placementKind int

const (
	// placeDefault es la ausencia de nota: tras el último bloque de requisito y su
	// cola (solo con identidad `text`).
	placeDefault placementKind = iota
	placeAfter
	placeSection
)

// placement es la ubicación declarada de un requisito ADDED.
type placement struct {
	kind  placementKind
	value string // el identificador del ancla o el título de la sección
}

// insertionPoint es una posición entre líneas del documento: antes de la línea
// `line` del tramo `seg`. Con seg igual al número de tramos es el final del
// documento.
type insertionPoint struct {
	seg, line int
}

func (c *composer) isText(seg segment) bool { return seg.requisite == nil && !seg.added }

// add inserta un requisito ADDED (REQ-24.43, D-11): su ubicación se evalúa sobre
// el documento después de RENAMED, MODIFIED y REMOVED, y su encabezado se emite
// con el ID definitivo que PlanIDs le asignó.
func (c *composer) add(block deltaBlock) error {
	id, title, key, err := c.parseName(sectionAdded, block.Name)
	if err != nil {
		return err
	}
	provisional := ""
	if c.profile.Identity == workspace.IdentityID {
		assignment, ok := c.input.IDs.Lookup(id)
		if !ok || assignment.Domain != c.domain {
			return c.fail(sectionAdded, block.Name,
				fmt.Sprintf("la correspondencia de IDs del cambio no asigna un ID definitivo al ID provisional %q de este dominio", id),
				"calcula la correspondencia de todo el cambio con PlanIDs y pásala en ComposeInput.IDs antes de componer")
		}
		provisional, id, key = id, assignment.Definitive, assignment.Definitive
	} else if c.exists(key) {
		return c.fail(sectionAdded, block.Name,
			fmt.Sprintf("ya existe un requisito con %s %q en el documento", identityLabel(c.profile), key),
			"elige un nombre que no exista o usa MODIFIED sobre el requisito existente")
	}

	place, err := c.placement(block)
	if err != nil {
		return err
	}
	var (
		point  insertionPoint
		refEOL string
		marker string
	)
	switch place.kind {
	case placeAfter:
		at, err := c.anchorBlock(block, place.value)
		if err != nil {
			return err
		}
		point, refEOL, marker = c.afterBlock(at), c.eolOf(c.segments[at].lines[0]), "after:"+place.value
	case placeSection:
		if point, refEOL, err = c.sectionEnd(block, place.value); err != nil {
			return err
		}
		marker = "section:" + place.value
	default:
		point, refEOL = c.afterLastBlock()
		marker = "last"
	}
	// Los ADDED con la misma ancla se insertan en el orden del delta, cada uno
	// tras el anterior.
	for point.line == 0 && point.seg < len(c.segments) && c.segments[point.seg].added && c.segments[point.seg].anchor == marker {
		point.seg++
	}

	body, omitted, err := c.integratedBody(sectionAdded, block)
	if err != nil {
		return err
	}
	heading := c.render(id, title)
	c.insert(point, segment{
		lines: c.newBlock(point, refEOL, heading, body),
		id:    id, title: title, key: key,
		added: true, anchor: marker,
	})
	c.entry(key, heading, ActionAdded, "", nil)
	last := &c.entries[len(c.entries)-1]
	last.Provisional, last.ScenariosOmitted = provisional, omitted
	return nil
}

// exists dice si el documento, con las operaciones ya aplicadas y los ADDED ya
// insertados, tiene un requisito con ese identificador.
func (c *composer) exists(key string) bool {
	for _, seg := range c.segments {
		if (seg.requisite != nil || seg.added) && !seg.superseded && seg.key == key {
			return true
		}
	}
	return false
}

func (c *composer) eolOf(line Line) string {
	if line.EOL != "" {
		return line.EOL
	}
	return c.eol
}

// placement lee la nota (Placement: ...) del bloque. Con identidad `id` es
// obligatoria; con identidad `text` es opcional.
func (c *composer) placement(block deltaBlock) (placement, error) {
	switch len(block.Placements) {
	case 0:
		if c.profile.Identity == workspace.IdentityID {
			return placement{}, c.fail(sectionAdded, block.Name,
				"falta la nota (Placement: ...): con identidad 'id' la ubicación de un requisito ADDED debe ser explícita y no se deduce del orden de los IDs",
				`añade al bloque una línea "(Placement: after <ID>)" o "(Placement: section "<título>")"`)
		}
		return placement{}, nil
	case 1:
	default:
		return placement{}, c.fail(sectionAdded, block.Name,
			fmt.Sprintf("el bloque declara %d notas (Placement: ...) y solo puede declarar una", len(block.Placements)),
			"deja una sola nota (Placement: ...) en el bloque")
	}
	raw := block.Placements[0]
	if match := placementAfter.FindStringSubmatch(raw); match != nil {
		return placement{kind: placeAfter, value: strings.TrimSpace(match[1])}, nil
	}
	if match := placementSection.FindStringSubmatch(raw); match != nil {
		return placement{kind: placeSection, value: strings.TrimSpace(match[1])}, nil
	}
	return placement{}, c.fail(sectionAdded, block.Name,
		fmt.Sprintf("la nota (Placement: %s) no tiene un formato reconocido", raw),
		`escribe "(Placement: after <ID>)" o "(Placement: section "<título>")"`)
}

// anchorBlock localiza el tramo del único requisito `<ID>` de `(Placement: after
// <ID>)`. Un identificador duplicado, inexistente o eliminado por el mismo delta
// sin --supersede se rechaza (REQ-24.40, REQ-24.43).
func (c *composer) anchorBlock(block deltaBlock, key string) (int, error) {
	var found []int
	for i, seg := range c.segments {
		if seg.requisite != nil && seg.key == key {
			found = append(found, i)
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		if c.removed[key] {
			return -1, c.fail(sectionAdded, block.Name,
				fmt.Sprintf("la ubicación (Placement: after %s) apunta a un requisito que el mismo delta elimina sin --supersede", key),
				"elige como ubicación otro requisito del documento o conserva el bloque con --supersede")
		}
		return -1, c.fail(sectionAdded, block.Name,
			fmt.Sprintf("la ubicación (Placement: after %s) apunta a un requisito que el documento no contiene", key),
			"comprueba el identificador en el documento de la baseline o elige otra ubicación")
	}
	return -1, c.ambiguous(sectionAdded, block.Name, key, found)
}

// afterBlock es la posición inmediatamente posterior al bloque del tramo `at` y a
// su cola contigua de secciones no requisito (D-09).
func (c *composer) afterBlock(at int) insertionPoint {
	next := at + 1
	if next < len(c.segments) && c.isText(c.segments[next]) {
		return c.normalize(insertionPoint{seg: next, line: tailExtent(c.segments[next].lines, c.profile)})
	}
	return insertionPoint{seg: next}
}

// afterLastBlock es la ubicación por defecto de un ADDED con identidad `text`:
// tras el último bloque de requisito y su cola, o al final del documento si no hay
// ninguno.
func (c *composer) afterLastBlock() (insertionPoint, string) {
	for i := len(c.segments) - 1; i >= 0; i-- {
		if c.segments[i].requisite != nil {
			return c.afterBlock(i), c.eolOf(c.segments[i].lines[0])
		}
	}
	return insertionPoint{seg: len(c.segments)}, c.eol
}

// sectionEnd localiza el final de la sección de nivel `level-1` con ese título,
// que debe ser única entre los encabezados de ese nivel: el punto anterior al
// siguiente encabezado de nivel menor o igual, o el final del documento (D-11).
func (c *composer) sectionEnd(block deltaBlock, title string) (insertionPoint, string, error) {
	level := c.profile.Level - 1
	if level < 1 {
		return insertionPoint{}, "", c.fail(sectionAdded, block.Name,
			fmt.Sprintf("la ubicación (Placement: section %q) no es posible con requisitos de nivel %d: no hay un nivel de sección superior", title, c.profile.Level),
			"usa (Placement: after <ID>)")
	}
	type hit struct{ seg, line int }
	var hits []hit
	for i, seg := range c.segments {
		if !c.isText(seg) {
			continue
		}
		for _, heading := range scanHeadings(seg.lines) {
			if heading.Level == level && strings.TrimSpace(heading.Title) == title {
				hits = append(hits, hit{seg: i, line: heading.Index})
			}
		}
	}
	if len(hits) != 1 {
		return insertionPoint{}, "", c.fail(sectionAdded, block.Name,
			fmt.Sprintf("el título de sección %q de (Placement: section ...) aparece %d veces entre los encabezados de nivel %d del documento y debe aparecer exactamente una", title, len(hits), level),
			"usa un título de sección que aparezca una sola vez o (Placement: after <ID>)")
	}
	start := hits[0]
	eol := c.eolOf(c.segments[start.seg].lines[start.line])
	for i := start.seg; i < len(c.segments); i++ {
		if !c.isText(c.segments[i]) {
			continue
		}
		for _, heading := range scanHeadings(c.segments[i].lines) {
			if (i > start.seg || heading.Index > start.line) && heading.Level <= level {
				return c.normalize(insertionPoint{seg: i, line: heading.Index}), eol, nil
			}
		}
	}
	return insertionPoint{seg: len(c.segments)}, eol, nil
}

// normalize convierte «al final de un tramo» en «al principio del siguiente».
func (c *composer) normalize(p insertionPoint) insertionPoint {
	if p.seg < len(c.segments) && p.line >= len(c.segments[p.seg].lines) {
		return insertionPoint{seg: p.seg + 1}
	}
	return p
}

// tailExtent devuelve cuántas líneas del tramo de texto que sigue a un bloque
// pertenecen a su cola contigua: las secciones no requisito del nivel de los
// requisitos, hasta el siguiente encabezado de nivel menor (D-09).
func tailExtent(lines []Line, profile Profile) int {
	headings := scanHeadings(lines)
	position := 0
	for position < len(lines) {
		at := -1
		for i, heading := range headings {
			if heading.Index == position {
				at = i
				break
			}
		}
		if at < 0 || headings[at].Level != profile.Level {
			return position
		}
		position = blockEnd(headings, at, profile.Level, len(lines))
	}
	return position
}

// blanksBefore cuenta las líneas en blanco que preceden a la posición: la serie de
// líneas en blanco finales del bloque de referencia (D-12). El segundo resultado
// indica si esa serie la delimita una línea que no está en blanco; si no hay
// ninguna línea anterior, o solo hay líneas en blanco, es falso.
func (c *composer) blanksBefore(p insertionPoint) (int, bool) {
	count := 0
	for i := p.seg; i >= 0; i-- {
		if i >= len(c.segments) {
			continue
		}
		lines := c.segments[i].lines
		if i == p.seg {
			lines = lines[:p.line]
		}
		for j := len(lines) - 1; j >= 0; j-- {
			if strings.TrimSpace(lines[j].Text) != "" {
				return count, true
			}
			count++
		}
	}
	return count, false
}

// newBlock construye las líneas de un bloque insertado: una línea en blanco
// delante si la línea anterior al punto de inserción no lo está, el encabezado, el
// cuerpo y la serie de líneas en blanco finales que hereda del bloque de
// referencia, con un mínimo de una salvo al final del documento. Usa el final de
// línea del encabezado de referencia. Si se inserta tras una última línea sin
// salto, esa línea recibe el final de línea del documento y el bloque nuevo
// termina sin él, de modo que el documento conserva su final. Solo inserta líneas:
// no modifica ningún byte existente salvo ese final de línea (D-12).
func (c *composer) newBlock(p insertionPoint, eol, heading string, body []deltaLine) []Line {
	atEnd := p.seg >= len(c.segments)
	blanks, afterText := c.blanksBefore(p)
	separate := afterText && blanks == 0
	if !atEnd && blanks < 1 {
		blanks = 1
	}
	lines := make([]Line, 0, len(body)+2+blanks)
	if separate {
		lines = append(lines, Line{EOL: eol})
	}
	lines = append(lines, Line{Text: heading, EOL: eol})
	for _, line := range body {
		lines = append(lines, Line{Text: line.Text, EOL: eol})
	}
	for i := 0; i < blanks; i++ {
		lines = append(lines, Line{EOL: eol})
	}
	if atEnd {
		if last := c.lastLine(); last != nil && last.EOL == "" {
			last.EOL = c.eol
			lines[len(lines)-1].EOL = ""
		}
	}
	return lines
}

// lastLine devuelve la última línea del documento, copiando su tramo para no
// modificar el documento original, o nil si no tiene líneas.
func (c *composer) lastLine() *Line {
	for i := len(c.segments) - 1; i >= 0; i-- {
		if len(c.segments[i].lines) == 0 {
			continue
		}
		c.segments[i].lines = append([]Line(nil), c.segments[i].lines...)
		return &c.segments[i].lines[len(c.segments[i].lines)-1]
	}
	return nil
}

// insert coloca el tramo en la posición, partiendo en dos el tramo de texto que
// la contiene si cae en medio de él.
func (c *composer) insert(p insertionPoint, s segment) {
	if p.seg < len(c.segments) && p.line > 0 {
		whole := c.segments[p.seg]
		head := segment{lines: whole.lines[:p.line:p.line]}
		tail := segment{lines: whole.lines[p.line:]}
		c.segments = append(c.segments[:p.seg], append([]segment{head, s, tail}, c.segments[p.seg+1:]...)...)
		return
	}
	c.segments = append(c.segments[:p.seg], append([]segment{s}, c.segments[p.seg:]...)...)
}
