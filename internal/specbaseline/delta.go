package specbaseline

import (
	"regexp"
	"strings"
)

// Secciones de un delta en formato nativo (D-10, REQ-24.42).
const (
	sectionAdded    = "ADDED"
	sectionModified = "MODIFIED"
	sectionRemoved  = "REMOVED"
	sectionRenamed  = "RENAMED"
)

var (
	// deltaSectionHeading y deltaRequirementHeading son los encabezados del
	// formato nativo del delta, con la misma forma que reconoce el motor nativo
	// (sddstatus.openSpecDeltaSectionHeading y openSpecReqHeadingLine).
	deltaSectionHeading     = regexp.MustCompile(`^## (ADDED|MODIFIED|REMOVED|RENAMED) Requirements[ \t]*$`)
	deltaRequirementHeading = regexp.MustCompile(`^### Requirement:[ \t]*(.+?)[ \t]*$`)
	// deltaNote reconoce las líneas completas de nota, que son metadatos del
	// delta y no se integran en el documento.
	deltaNote = regexp.MustCompile(`^\((Placement|Previously|Reason|Migration):(.+)\)[ \t]*$`)
	// renameArrow separa el nombre antiguo y el nuevo de un RENAMED.
	renameArrow = regexp.MustCompile(`[ \t]*(?:\x{2192}|->)[ \t]*`)
)

// deltaLine es una línea del cuerpo de un requisito del delta con su número de
// línea (desde 1) en el fichero del delta.
type deltaLine struct {
	Text   string
	Number int
}

// deltaBlock es un requisito del delta: el nombre tras `### Requirement:` y su
// cuerpo, sin las líneas de nota ni las líneas en blanco finales.
type deltaBlock struct {
	Section   string
	Name      string
	Line      int // línea del encabezado, desde 1
	Body      []deltaLine
	Reason    string // valor de la nota (Reason: ...)
	HasReason bool
	// Placements son los valores de las notas (Placement: ...), en orden: la
	// ubicación de un requisito ADDED (D-11).
	Placements []string
}

// delta agrupa los requisitos de un delta por sección, en el orden del fichero.
type delta struct {
	Added, Modified, Removed, Renamed []deltaBlock
}

func (d delta) empty() bool {
	return len(d.Added)+len(d.Modified)+len(d.Removed)+len(d.Renamed) == 0
}

// parseDelta lee un delta en formato nativo. Los encabezados de sección y de
// requisito dentro de un bloque de código con vallas no cuentan, y las líneas de
// nota solo se retiran fuera de vallas. Lo que precede a la primera sección, o a
// un requisito dentro de ella, se ignora, como hace el motor nativo.
func parseDelta(data []byte) delta {
	doc := ParseDocument(data)
	fenced := fencedLines(doc.Lines)

	var parsed delta
	section := ""
	var current *deltaBlock
	flush := func() {
		if current == nil {
			return
		}
		for len(current.Body) > 0 && strings.TrimSpace(current.Body[len(current.Body)-1].Text) == "" {
			current.Body = current.Body[:len(current.Body)-1]
		}
		switch current.Section {
		case sectionAdded:
			parsed.Added = append(parsed.Added, *current)
		case sectionModified:
			parsed.Modified = append(parsed.Modified, *current)
		case sectionRemoved:
			parsed.Removed = append(parsed.Removed, *current)
		case sectionRenamed:
			parsed.Renamed = append(parsed.Renamed, *current)
		}
		current = nil
	}

	for i, line := range doc.Lines {
		number := i + 1
		if !fenced[i] {
			if match := deltaSectionHeading.FindStringSubmatch(line.Text); match != nil {
				flush()
				section = match[1]
				continue
			}
			if match := deltaRequirementHeading.FindStringSubmatch(line.Text); match != nil {
				flush()
				if section != "" {
					current = &deltaBlock{Section: section, Name: match[1], Line: number}
				}
				continue
			}
		}
		if current == nil {
			continue
		}
		if !fenced[i] {
			if note := deltaNote.FindStringSubmatch(line.Text); note != nil {
				switch note[1] {
				case "Reason":
					current.Reason, current.HasReason = strings.TrimSpace(note[2]), true
				case "Placement":
					current.Placements = append(current.Placements, strings.TrimSpace(note[2]))
				}
				continue
			}
		}
		current.Body = append(current.Body, deltaLine{Text: line.Text, Number: number})
	}
	flush()
	return parsed
}

// splitRenamePair separa el nombre de un RENAMED, «Antiguo → Nuevo» o
// «Antiguo -> Nuevo», en sus dos nombres.
func splitRenamePair(name string) (oldName, newName string, ok bool) {
	parts := renameArrow.Split(name, 2)
	if len(parts) != 2 {
		return "", "", false
	}
	oldName, newName = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	return oldName, newName, oldName != "" && newName != ""
}

// headingViolation es un encabezado del cuerpo de un delta que cerraría el
// bloque del requisito (D-09).
type headingViolation struct {
	Text   string // línea del encabezado, sin final de línea
	Level  int
	Number int // línea en el delta, desde 1
}

// bodyViolations devuelve los encabezados ATX del cuerpo, fuera de bloques de
// código con vallas, cuyo nivel es menor o igual que level: integrarlos tal cual
// cortaría el bloque y la siguiente composición leería el resto como cola
// (REQ-24.42). La integración nunca rebaja ni reescribe esos encabezados.
func bodyViolations(body []deltaLine, level int) []headingViolation {
	lines := make([]Line, len(body))
	for i, line := range body {
		lines[i] = Line{Text: line.Text}
	}
	fenced := fencedLines(lines)
	var violations []headingViolation
	for i, line := range body {
		if fenced[i] {
			continue
		}
		if headingLevel, _, ok := parseATX(line.Text); ok && headingLevel <= level {
			violations = append(violations, headingViolation{Text: line.Text, Level: headingLevel, Number: line.Number})
		}
	}
	return violations
}

// scenarioLevel es el nivel de los encabezados `#### Scenario:` del formato
// nativo del delta.
const scenarioLevel = 4

// omitScenarios elimina del cuerpo cada escenario: el fragmento que comienza en
// una línea `#### Scenario:` y termina antes del siguiente encabezado ATX de nivel
// menor o igual que 4, o al final del cuerpo, siempre fuera de bloques de código
// con vallas (D-10, REQ-24.61). Un encabezado de nivel 4 que no es un escenario,
// como `#### Notas`, y su texto se conservan. Devuelve el cuerpo sin las líneas
// en blanco finales y si había algún escenario.
func omitScenarios(body []deltaLine) (kept []deltaLine, omitted bool) {
	lines := make([]Line, len(body))
	for i, line := range body {
		lines[i] = Line{Text: line.Text}
	}
	fenced := fencedLines(lines)
	inScenario := false
	for i, line := range body {
		if !fenced[i] {
			if level, title, ok := parseATX(line.Text); ok && level <= scenarioLevel {
				inScenario = level == scenarioLevel && strings.HasPrefix(title, "Scenario:")
				omitted = omitted || inScenario
			}
		}
		if !inScenario {
			kept = append(kept, line)
		}
	}
	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1].Text) == "" {
		kept = kept[:len(kept)-1]
	}
	return kept, omitted
}
