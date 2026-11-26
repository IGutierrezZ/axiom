// Package openspecdoc es el paquete hoja que analiza y compone documentos OpenSpec
// (especificaciones canónicas y deltas con ADDED, MODIFIED, REMOVED y RENAMED). Lo
// comparten el archivado SDD (sddstatus), la integración ODD (livespec) y la adopción
// sin que ninguno dependa de otro; solo importa la biblioteca estándar y
// internal/reqid, que define la forma de los identificadores de requisito.
package openspecdoc

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// UnappliedDeltaError names exactly one delta requirement that
// Compose could not apply. Callers MUST refuse — never
// report success on a partial composition (#4119: archive previously wrote
// drift or reported success while a delta went unapplied).
type UnappliedDeltaError struct {
	// Section is ADDED, MODIFIED, REMOVED, RENAMED, DELTA, or CANONICAL (the
	// last two describe a malformed input, not one named requirement).
	Section     string
	Requirement string
	Reason      string
}

func (e *UnappliedDeltaError) Error() string {
	if e.Requirement == "" {
		return fmt.Sprintf("sdd-archive-compose: %s: %s", e.Section, e.Reason)
	}
	return fmt.Sprintf("sdd-archive-compose: unapplied %s delta for requirement %q: %s", e.Section, e.Requirement, e.Reason)
}

// UnknownRequirementIDError es el rechazo de un MODIFIED, un REMOVED o el nombre antiguo de un
// RENAMED cuyo nombre lleva un identificador válido que ningún requisito de la canónica
// lleva (REQ-47.3). El paquete no conoce la capacidad: la añade quien la sabe (livespec o
// la orden `archive-compose`, por la carpeta).
type UnknownRequirementIDError struct {
	Section     string
	Requirement string
	ID          string

	// name es el nombre que se buscó en la canónica (el antiguo, en un RENAMED).
	name string
}

func (e *UnknownRequirementIDError) Error() string { return e.Unwrap().Error() }

// Unwrap devuelve el *UnappliedDeltaError equivalente, de modo que quien ya lo trata con
// errors.As siga viéndolo.
func (e *UnknownRequirementIDError) Unwrap() error {
	return &UnappliedDeltaError{Section: e.Section, Requirement: e.Requirement, Reason: fmt.Sprintf("no canonical requirement named %q", e.searched())}
}

func (e *UnknownRequirementIDError) searched() string {
	if e.name != "" {
		return e.name
	}
	return e.Requirement
}

// ProvisionalIDError es el rechazo de un requisito cuyo nombre o cuerpo contiene un
// identificador provisional: la composición produce la spec viva, y un provisional nunca
// llega a ella (REQ-47.1, REQ-30.22).
type ProvisionalIDError struct {
	Section     string
	Requirement string
	Token       string
}

func (e *ProvisionalIDError) Error() string {
	return fmt.Sprintf("el requisito %q de %s contiene el identificador provisional %s, que no puede llegar a la spec viva; asigna los números definitivos antes de componer", e.Requirement, e.Section, e.Token)
}

var openSpecReqHeadingLine = regexp.MustCompile(`(?m)^### Requirement:[ \t]*(.+?)[ \t]*$`)
var openSpecH2Heading = regexp.MustCompile(`(?m)^## \S`)
var openSpecDeltaSectionHeading = regexp.MustCompile(`(?m)^## (ADDED|MODIFIED|REMOVED|RENAMED) Requirements[ \t]*$`)
var openSpecRenameArrow = regexp.MustCompile(`[ \t]*(?:\x{2192}|->)[ \t]*`)
var openSpecReasonNote = regexp.MustCompile(`(?m)^\(Reason:.+\)[ \t]*$`)

type specRequirementBlock struct {
	Name, Text string
	// Offset es la posición en bytes del bloque dentro del texto analizado.
	Offset int
}

// specSegment is one contiguous slice of a canonical spec: either a
// "### Requirement:" block or a non-requirement span (preamble, an
// interstitial "## Section", or trailing content). Segments partition the
// whole document with no gaps, so re-emitting every segment in order
// reproduces the input byte-for-byte (#4119 follow-up: a preamble/
// requirements/trailing model silently dropped a "## Section" that sat
// between two requirement blocks, since it belonged to none of the three).
type specSegment struct {
	IsRequirement bool
	Name, Text    string
}

type specDocument struct {
	Segments []specSegment
}

func (d specDocument) hasRequirement() bool {
	for _, s := range d.Segments {
		if s.IsRequirement {
			return true
		}
	}
	return false
}

// FenceMarker reports whether line opens or closes a ``` / ~~~ fence: the
// fence character, its run length, and whether trailing content follows the
// run (an info string, which only a valid opener may carry).
func FenceMarker(line string) (ch byte, length int, hasInfo, ok bool) {
	if len(line) < 3 || (line[0] != '`' && line[0] != '~') {
		return 0, 0, false, false
	}
	ch = line[0]
	for length < len(line) && line[length] == ch {
		length++
	}
	if length < 3 {
		return 0, 0, false, false
	}
	return ch, length, strings.TrimSpace(line[length:]) != "", true
}

// fencedRanges scans text line-by-line and returns the byte ranges that lie
// inside a fenced code block, so a "## " or "### Requirement:" line found
// only inside a fence (e.g. as sample content) is never treated as a
// document boundary. A close only matches the same fence character with a
// run at least as long as the opener, and carries no info string. An
// unterminated fence is treated as fenced through end of document.
func fencedRanges(text string) [][2]int {
	var ranges [][2]int
	inFence, fenceChar, fenceLen, fenceStart := false, byte(0), 0, 0
	for pos := 0; pos <= len(text); {
		nl := strings.IndexByte(text[pos:], '\n')
		lineEnd := len(text)
		next := len(text) + 1
		if nl != -1 {
			lineEnd, next = pos+nl, pos+nl+1
		}
		line := text[pos:lineEnd]
		if !inFence {
			if ch, n, _, ok := FenceMarker(line); ok {
				inFence, fenceChar, fenceLen, fenceStart = true, ch, n, pos
			}
		} else if ch, n, hasInfo, ok := FenceMarker(line); ok && ch == fenceChar && n >= fenceLen && !hasInfo {
			ranges, inFence = append(ranges, [2]int{fenceStart, lineEnd}), false
		}
		pos = next
	}
	if inFence {
		ranges = append(ranges, [2]int{fenceStart, len(text)})
	}
	return ranges
}

func insideAnyRange(ranges [][2]int, pos int) bool {
	for _, r := range ranges {
		if pos >= r[0] && pos < r[1] {
			return true
		}
	}
	return false
}

// parseSpecDocument partitions a spec at every "### Requirement:" and "## "
// heading boundary that sits outside a fenced code block. Every byte of the
// input belongs to exactly one segment.
func parseSpecDocument(text string) specDocument {
	fences := fencedRanges(text)

	var reqMatches [][]int
	for _, m := range openSpecReqHeadingLine.FindAllStringSubmatchIndex(text, -1) {
		if !insideAnyRange(fences, m[0]) {
			reqMatches = append(reqMatches, m)
		}
	}
	if len(reqMatches) == 0 {
		if text == "" {
			return specDocument{}
		}
		return specDocument{Segments: []specSegment{{Text: text}}}
	}

	reqAt := make(map[int]int, len(reqMatches))
	boundaries := make([]int, 0, len(reqMatches)+2)
	for i, m := range reqMatches {
		reqAt[m[0]] = i
		boundaries = append(boundaries, m[0])
	}
	for _, m := range openSpecH2Heading.FindAllStringIndex(text, -1) {
		if !insideAnyRange(fences, m[0]) {
			boundaries = append(boundaries, m[0])
		}
	}
	boundaries = append(boundaries, 0, len(text))
	sort.Ints(boundaries)

	var doc specDocument
	for i := 0; i < len(boundaries)-1; i++ {
		start, end := boundaries[i], boundaries[i+1]
		if start == end {
			continue
		}
		if m, ok := reqAt[start]; ok {
			name := strings.TrimSpace(text[reqMatches[m][2]:reqMatches[m][3]])
			doc.Segments = append(doc.Segments, specSegment{IsRequirement: true, Name: name, Text: text[start:end]})
			continue
		}
		doc.Segments = append(doc.Segments, specSegment{Text: text[start:end]})
	}
	return doc
}

type openSpecDelta struct {
	Added, Modified, Removed, Renamed []specRequirementBlock
}

// parseOpenSpecDelta splits a delta spec into its ADDED/MODIFIED/REMOVED/
// RENAMED requirement blocks (skills/_shared/openspec-convention.md).
func parseOpenSpecDelta(text string) openSpecDelta {
	sections := openSpecDeltaSectionHeading.FindAllStringSubmatchIndex(text, -1)
	var delta openSpecDelta
	for i, m := range sections {
		kind, start, end := text[m[2]:m[3]], m[1], len(text)
		if i+1 < len(sections) {
			end = sections[i+1][0]
		}
		blocks := parseDeltaRequirementBlocks(text[start:end])
		for j := range blocks {
			blocks[j].Offset += start
		}
		switch kind {
		case "ADDED":
			delta.Added = append(delta.Added, blocks...)
		case "MODIFIED":
			delta.Modified = append(delta.Modified, blocks...)
		case "REMOVED":
			delta.Removed = append(delta.Removed, blocks...)
		case "RENAMED":
			delta.Renamed = append(delta.Renamed, blocks...)
		}
	}
	return delta
}

func parseDeltaRequirementBlocks(text string) []specRequirementBlock {
	matches := openSpecReqHeadingLine.FindAllStringSubmatchIndex(text, -1)
	blocks := make([]specRequirementBlock, 0, len(matches))
	for i, m := range matches {
		end := len(text)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		name := strings.TrimSpace(text[m[2]:m[3]])
		blocks = append(blocks, specRequirementBlock{Name: name, Text: text[m[0]:end], Offset: m[0]})
	}
	return blocks
}

func splitRenamePair(name string) (oldName, newName string, ok bool) {
	parts := openSpecRenameArrow.Split(name, 2)
	if len(parts) != 2 {
		return "", "", false
	}
	oldName, newName = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	return oldName, newName, oldName != "" && newName != ""
}

func renameRequirementHeading(text, newName string) string {
	replaced := false
	return openSpecReqHeadingLine.ReplaceAllStringFunc(text, func(line string) string {
		if replaced {
			return line
		}
		replaced = true
		return "### Requirement: " + newName
	})
}

func ensureTrailingNewline(text string) string {
	if strings.HasSuffix(text, "\n") {
		return text
	}
	return text + "\n"
}

// ComposeOptions configura el comportamiento opcional de poda progresiva y archivado (ODD-4.1).
type ComposeOptions struct {
	// SupersedeRemoved marca los requisitos de REMOVED con [SUPERSEDED / DEPRECADO] en lugar de borrarlos físicamente.
	SupersedeRemoved bool
	// ExplicitSuperseded lista nombres de requisitos adicionales que deben marcarse como superados.
	ExplicitSuperseded []string
	// IDPattern es el patrón de identificadores del proyecto con el que MODIFIED, REMOVED y el
	// nombre antiguo de RENAMED se resuelven por identificador (REQ-30.22). El valor cero es el
	// patrón por defecto, `REQ-<n>.<m>`.
	IDPattern reqid.Pattern
}

// ambiguousReason es el motivo, en castellano, de un identificador que llevan varios
// requisitos de la canónica sin que ninguno tenga el título exacto de la delta.
func ambiguousReason(segments []specSegment, name string, p reqid.Pattern) string {
	id := p.InTitle(name)
	var titles []string
	for _, s := range segments {
		if s.IsRequirement && !reqid.IsSuperseded(s.Name) && p.InTitle(s.Name) == id {
			titles = append(titles, fmt.Sprintf("%q", s.Name))
		}
	}
	return fmt.Sprintf("el identificador %s lo llevan %d requisitos de la canónica (%s) y ninguno se llama %q; deja en la delta el título exacto de uno de ellos", id, len(titles), strings.Join(titles, ", "), name)
}

func extractReason(text string) string {
	match := openSpecReasonNote.FindString(text)
	if match != "" {
		match = strings.TrimSpace(match)
		match = strings.TrimPrefix(match, "(Reason:")
		match = strings.TrimPrefix(match, "(Reason :")
		match = strings.TrimSuffix(match, ")")
		return strings.TrimSpace(match)
	}
	return "Requisito superado durante la consolidación de especificaciones"
}

func annotateRequirementSuperseded(text, name, reason string) string {
	replaced := false
	out := openSpecReqHeadingLine.ReplaceAllStringFunc(text, func(line string) string {
		if replaced {
			return line
		}
		replaced = true
		note := fmt.Sprintf("\n> [!CAUTION]\n> **REQUISITO HISTÓRICO SUPERADO / DEPRECADO:** Este requisito ha sido superado o deprecado.\n> Motivo: %s\n", reason)
		return "### Requirement: " + name + reqid.SupersededSuffix + note
	})
	return ensureTrailingNewline(out)
}

// Compose merges an OpenSpec delta spec into a canonical
// spec's byte content. Every unrelated canonical requirement is preserved
// verbatim; RENAMED, MODIFIED, REMOVED, then ADDED delta requirements are
// applied in that order, so a rename is visible to a same-change MODIFIED
// for its new name.
//
// It never returns a partial composition: an unmatched, duplicate, or
// malformed delta requirement returns *UnappliedDeltaError instead (#4119 —
// archive previously reported success on exactly this kind of drift).
//
// opts permite además la preservación de requisitos deprecados
// ([SUPERSEDED / DEPRECADO]) para poda progresiva.
func Compose(canonical, delta string, opts ComposeOptions) (string, error) {
	canonicalDoc := parseSpecDocument(canonical)
	if !canonicalDoc.hasRequirement() {
		return "", &UnappliedDeltaError{Section: "CANONICAL", Reason: `canonical spec has no "### Requirement:" headings to compose against`}
	}

	deltaDoc := parseOpenSpecDelta(delta)
	if len(deltaDoc.Added) == 0 && len(deltaDoc.Modified) == 0 && len(deltaDoc.Removed) == 0 && len(deltaDoc.Renamed) == 0 {
		return "", &UnappliedDeltaError{Section: "DELTA", Reason: "delta spec declares no ADDED, MODIFIED, REMOVED, or RENAMED requirements"}
	}

	// Un provisional en el nombre o el cuerpo de cualquier requisito se rechaza antes de
	// tocar nada (REQ-30.22). Se mira en el orden en que se aplican las secciones.
	for _, group := range []struct {
		section string
		blocks  []specRequirementBlock
	}{{"RENAMED", deltaDoc.Renamed}, {"MODIFIED", deltaDoc.Modified}, {"REMOVED", deltaDoc.Removed}, {"ADDED", deltaDoc.Added}} {
		for _, r := range group.blocks {
			if spans := reqid.ScanProvisional(r.Text); len(spans) > 0 {
				return "", &ProvisionalIDError{Section: group.section, Requirement: r.Name, Token: spans[0].Token}
			}
		}
	}

	// segments re-emits the whole document, including non-requirement spans
	// between requirement blocks, so nothing but a targeted delta edit ever
	// changes a byte of it.
	segments := append([]specSegment(nil), canonicalDoc.Segments...)
	indexOf := func(name string) int {
		for i, s := range segments {
			if s.IsRequirement && s.Name == name {
				return i
			}
		}
		return -1
	}
	// resolve empareja el nombre de una delta con un requisito de la canónica por
	// identificador, con vuelta al título exacto (REQ-30.22). Devuelve el índice del
	// segmento y, si no hay emparejamiento, la causa para el rechazo.
	resolve := func(section, name, display string) (int, error) {
		var names []string
		var at []int
		for i, s := range segments {
			if s.IsRequirement {
				names = append(names, s.Name)
				at = append(at, i)
			}
		}
		k, match := opts.IDPattern.Resolve(names, name)
		switch match {
		case reqid.MatchFound:
			return at[k], nil
		case reqid.MatchAmbiguous:
			return -1, &UnappliedDeltaError{Section: section, Requirement: display, Reason: ambiguousReason(segments, name, opts.IDPattern)}
		}
		if id := opts.IDPattern.InTitle(name); id != "" {
			return -1, &UnknownRequirementIDError{Section: section, Requirement: display, ID: id, name: name}
		}
		return -1, &UnappliedDeltaError{Section: section, Requirement: display, Reason: fmt.Sprintf("no canonical requirement named %q", name)}
	}
	lastRequirementIndex := func() int {
		last := -1
		for i, s := range segments {
			if s.IsRequirement {
				last = i
			}
		}
		return last
	}

	for _, r := range deltaDoc.Renamed {
		oldName, newName, ok := splitRenamePair(r.Name)
		if !ok {
			return "", &UnappliedDeltaError{Section: "RENAMED", Requirement: r.Name, Reason: `heading must state "Old Name → New Name"`}
		}
		i, err := resolve("RENAMED", oldName, r.Name)
		if err != nil {
			return "", err
		}
		if !openSpecReasonNote.MatchString(r.Text) {
			return "", &UnappliedDeltaError{Section: "RENAMED", Requirement: r.Name, Reason: `missing required "(Reason: ...)" note`}
		}
		if j := indexOf(newName); j != -1 && j != i {
			return "", &UnappliedDeltaError{Section: "RENAMED", Requirement: newName, Reason: "target name already exists"}
		}
		segments[i].Text = renameRequirementHeading(segments[i].Text, newName)
		segments[i].Name = newName
	}

	for _, r := range deltaDoc.Modified {
		i, err := resolve("MODIFIED", r.Name, r.Name)
		if err != nil {
			return "", err
		}
		segments[i].Text = ensureTrailingNewline(r.Text)
		// El segmento toma el nombre de la delta: las operaciones siguientes de la misma
		// delta lo ven con el título nuevo (REQ-30.22).
		segments[i].Name = r.Name
	}

	// addAnchor is "insert the next ADDED requirement right after this
	// index". It starts at the last requirement segment and is adjusted as
	// REMOVED deletes segments, so a REMOVE-then-ADD on the last remaining
	// requirement falls back to the non-requirement segment that preceded
	// it (e.g. the "## Requirements" heading) instead of index 0.
	addAnchor := lastRequirementIndex()

	for _, r := range deltaDoc.Removed {
		if !openSpecReasonNote.MatchString(r.Text) {
			return "", &UnappliedDeltaError{Section: "REMOVED", Requirement: r.Name, Reason: `missing required "(Reason: ...)" note`}
		}
		i, err := resolve("REMOVED", r.Name, r.Name)
		if err != nil {
			return "", err
		}
		if opts.SupersedeRemoved {
			// La anotación usa el nombre del segmento de la canónica, que puede diferir del
			// de la delta cuando el requisito se resolvió por identificador.
			reason := extractReason(r.Text)
			canonicalName := segments[i].Name
			segments[i].Text = annotateRequirementSuperseded(segments[i].Text, canonicalName, reason)
			segments[i].Name = canonicalName + reqid.SupersededSuffix
			continue
		}
		switch {
		case i == addAnchor:
			addAnchor = i - 1
		case i < addAnchor:
			addAnchor--
		}
		segments = append(segments[:i], segments[i+1:]...)
	}

	for _, name := range opts.ExplicitSuperseded {
		i := indexOf(name)
		if i != -1 {
			segments[i].Text = annotateRequirementSuperseded(segments[i].Text, name, "Superado por consolidación de arquitectura")
			segments[i].Name = name + reqid.SupersededSuffix
		}
	}

	for _, r := range deltaDoc.Added {
		if indexOf(r.Name) != -1 {
			return "", &UnappliedDeltaError{Section: "ADDED", Requirement: r.Name, Reason: fmt.Sprintf("a requirement named %q already exists in the canonical spec", r.Name)}
		}
		insertAt := addAnchor + 1
		newSegment := specSegment{IsRequirement: true, Name: r.Name, Text: ensureTrailingNewline(r.Text)}
		segments = append(segments[:insertAt], append([]specSegment{newSegment}, segments[insertAt:]...)...)
		addAnchor = insertAt
	}

	var b strings.Builder
	for _, s := range segments {
		if s.IsRequirement {
			b.WriteString(ensureTrailingNewline(s.Text))
			continue
		}
		b.WriteString(s.Text)
	}
	return b.String(), nil
}
