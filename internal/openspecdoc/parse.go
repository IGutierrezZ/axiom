package openspecdoc

import (
	"regexp"
	"strings"
)

// Requirement es un bloque `### Requirement:` de una especificación o de una delta.
type Requirement struct {
	// Name es el título tras `### Requirement:`, sin espacios en los extremos.
	Name string
	// ID es el identificador `REQ-n.m` del título; "" si el título no lleva `(REQ-n.m)`.
	ID string
	// Body es el bloque completo, con su línea de título, tal como está en el texto.
	Body string
	// Line es la línea (desde 1) del título dentro del texto analizado.
	Line int
}

// Delta son los requisitos de una delta OpenSpec por operación y los encabezados de
// nivel 2 que no son ninguna operación (fuera de vallas), que el llamante decide si
// rechaza.
type Delta struct {
	Added, Modified, Removed, Renamed []Requirement
	Unknown                           []string
}

// Empty dice si la delta no declara ningún requisito.
func (d Delta) Empty() bool {
	return len(d.Added)+len(d.Modified)+len(d.Removed)+len(d.Renamed) == 0
}

var requirementIDInTitle = regexp.MustCompile(`\((REQ-\d+\.\d+)\)`)

// RequirementID devuelve el último `(REQ-n.m)` del título de un requisito, o "".
func RequirementID(title string) string {
	matches := requirementIDInTitle.FindAllStringSubmatch(title, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1][1]
}

// SplitRename separa el título de un requisito RENAMED, `Antiguo → Nuevo` (o `->`).
func SplitRename(name string) (oldName, newName string, ok bool) {
	return splitRenamePair(name)
}

func lineOf(text string, offset int) int {
	return strings.Count(text[:offset], "\n") + 1
}

func toRequirements(text string, blocks []specRequirementBlock) []Requirement {
	reqs := make([]Requirement, 0, len(blocks))
	for _, b := range blocks {
		reqs = append(reqs, Requirement{Name: b.Name, ID: RequirementID(b.Name), Body: b.Text, Line: lineOf(text, b.Offset)})
	}
	return reqs
}

// ParseDelta divide una delta OpenSpec en sus requisitos ADDED, MODIFIED, REMOVED y
// RENAMED con el mismo análisis que usa la composición.
func ParseDelta(text string) Delta {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	parsed := parseOpenSpecDelta(text)
	delta := Delta{
		Added:    toRequirements(text, parsed.Added),
		Modified: toRequirements(text, parsed.Modified),
		Removed:  toRequirements(text, parsed.Removed),
		Renamed:  toRequirements(text, parsed.Renamed),
	}
	fences := fencedRanges(text)
	for _, m := range openSpecH2Heading.FindAllStringIndex(text, -1) {
		if insideAnyRange(fences, m[0]) {
			continue
		}
		end := strings.IndexByte(text[m[0]:], '\n')
		line := text[m[0]:]
		if end >= 0 {
			line = text[m[0] : m[0]+end]
		}
		line = strings.TrimRight(line, " \t")
		if !openSpecDeltaSectionHeading.MatchString(line) {
			delta.Unknown = append(delta.Unknown, line)
		}
	}
	return delta
}

// ParseCanonical devuelve los requisitos de una especificación canónica, en orden. Un
// encabezado dentro de una valla no cuenta.
func ParseCanonical(text string) []Requirement {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var reqs []Requirement
	offset := 0
	for _, segment := range parseSpecDocument(text).Segments {
		if segment.IsRequirement {
			reqs = append(reqs, Requirement{Name: segment.Name, ID: RequirementID(segment.Name), Body: segment.Text, Line: lineOf(text, offset)})
		}
		offset += len(segment.Text)
	}
	return reqs
}

// NormalizeBody normaliza un cuerpo para compararlo: LF, sin espacios ni tabuladores
// finales en cada línea y sin líneas en blanco al principio ni al final.
func NormalizeBody(body string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	start, end := 0, len(lines)
	for start < end && lines[start] == "" {
		start++
	}
	for end > start && lines[end-1] == "" {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}
