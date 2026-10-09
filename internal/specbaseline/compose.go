package specbaseline

import (
	"fmt"
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

// ComposeInput reúne lo necesario para componer un documento con perfil.
type ComposeInput struct {
	Document           PlannedDocument
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
}

// ComposeDocument compone el delta sobre el documento con el perfil de
// requisito, en memoria y sin escribir (D-08, D-12). Aplica RENAMED, MODIFIED y
// REMOVED por identificador, en el mismo orden que el motor nativo; todo lo que no
// pertenece a un bloque editado conserva sus bytes (REQ-24.45). Devuelve
// *ComposeError ante cualquier delta no aplicable: nunca una composición parcial.
//
// En este tramo no se componen todavía los ADDED ni `scenarios: omit` (se
// incorporan con la asignación de IDs): se rechazan con un *ComposeError.
func ComposeDocument(in ComposeInput) (ComposeResult, error) {
	document := in.Document
	c := &composer{input: in, profile: document.Profile, domain: document.Domain}
	if c.profile.Heading == nil {
		return ComposeResult{}, c.fail("DOCUMENTO", "", "el documento no tiene un perfil de requisito compilado", "declara 'requirement' en baseline.documents del dominio en axiom.yaml")
	}
	if c.profile.Scenarios == workspace.ScenariosOmit {
		return ComposeResult{}, c.fail("DOCUMENTO", "", "la composición con 'scenarios: omit' todavía no está disponible", "declara 'scenarios: include' o espera a la composición de ADDED y escenarios")
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
	if len(deltaDoc.Added) > 0 {
		block := deltaDoc.Added[0]
		return ComposeResult{}, c.fail(sectionAdded, block.Name, "la composición de ADDED con perfil de requisito (ubicación explícita e ID definitivo) todavía no está disponible", "retira el ADDED del delta o espera a que la composición lo admita")
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
	line := c.profile.HeadingLine(name)
	id, title, ok := c.profile.Match(line)
	if !ok {
		return "", "", "", c.fail(section, name,
			fmt.Sprintf("el nombre del requisito no cumple el patrón 'heading' %q (se evalúa la línea %q)", c.profile.Heading.String(), line),
			"escribe el nombre como el encabezado del proyecto sin los '#' iniciales ni el espacio que los sigue, de modo que los '#' del nivel, un espacio y el nombre cumplan 'heading'")
	}
	key = c.profile.Key(id, title)
	if strings.TrimSpace(key) == "" {
		return "", "", "", c.fail(section, name,
			fmt.Sprintf("el nombre del requisito no define el %s con el que se identifica", identityLabel(c.profile)),
			"incluye en el nombre el identificador del requisito, según el patrón 'heading' del perfil")
	}
	return id, title, key, nil
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
	occurrences := make([]string, len(found))
	for i, at := range found {
		requisite := c.segments[at].requisite
		occurrences[i] = fmt.Sprintf("línea %d %q", requisite.LineNumber(), requisite.Line)
	}
	return -1, c.fail(section, name,
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
	heading := c.profile.RenderHeading(newID, newTitle)
	seg.lines = append([]Line{{Text: heading, EOL: seg.lines[0].EOL}}, seg.lines[1:]...)
	seg.id, seg.title, seg.key = newID, newTitle, newKey
	c.entry(newKey, heading, ActionModified, fmt.Sprintf("renombrado desde %q: %s", oldKey, block.Reason), nil)
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
	if err := c.checkBody(sectionModified, block); err != nil {
		return err
	}

	seg := &c.segments[at]
	heading := c.profile.RenderHeading(id, title)
	warnings := c.tailWarnings(sectionModified, *seg)
	seg.lines = c.replacement(seg.lines, heading, block.Body)
	seg.id, seg.title = id, title
	c.entry(key, heading, ActionModified, "", warnings)
	return nil
}

// checkBody rechaza el cuerpo de un delta que contiene, fuera de vallas, un
// encabezado de nivel menor o igual que el de los requisitos (D-09, REQ-24.42).
func (c *composer) checkBody(section string, block deltaBlock) error {
	violations := bodyViolations(block.Body, c.profile.Level)
	if len(violations) == 0 {
		return nil
	}
	found := make([]string, len(violations))
	for i, violation := range violations {
		found[i] = fmt.Sprintf("%q (línea %d del delta, nivel %d)", violation.Text, violation.Number, violation.Level)
	}
	return c.fail(section, block.Name,
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
	if c.input.Supersede {
		anchor := c.supersede(seg, block.Reason)
		c.entry(key, anchor, ActionRemoved, block.Reason, warnings)
		return nil
	}
	anchor := seg.requisite.Line
	c.segments = append(c.segments[:at], c.segments[at+1:]...)
	c.entry(key, anchor, ActionRemoved, block.Reason, warnings)
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
