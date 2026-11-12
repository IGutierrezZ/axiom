// Package livespec integra en la spec viva de OpenSpec (openspec/specs/) el trabajo del
// carril ODD: analiza las secciones `## Spec viva` e `## Impacto en la spec` del
// documento odd/tasks/<feature>.md, planifica la integración de cada delta con la misma
// composición que el archivado SDD y la aplica todo o nada. No tiene estado propio, no
// ejecuta procesos (git pasa por gitcommon) y no importa ningún paquete de integración de documentos.
package livespec

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/isoevidence"
	"github.com/IGutierrezZ/axiom/v3/internal/openspecdoc"
)

// Kind es el tipo de trabajo que declara la cabecera del documento ODD.
type Kind string

const (
	// KindBug es un documento con la línea `Tipo: bug`: debe declarar su impacto en la spec.
	KindBug Kind = "bug"
	// KindIncrement es el tipo por defecto (`Tipo: incremento` o sin la línea).
	KindIncrement Kind = "increment"
)

// Defectos del marcador `## Impacto en la spec` (Impact.Defect).
const (
	// ImpactMarkerMissing: la sección no contiene la línea exacta `Sin impacto en la spec`.
	ImpactMarkerMissing = "marker_missing"
	// ImpactReasonMissing: hay marcador pero falta la línea `Motivo:` o está vacía.
	ImpactReasonMissing = "reason_missing"
)

const (
	specSectionHeading   = "Spec viva"
	impactSectionHeading = "Impacto en la spec"
	// impactMarkerLine es la misma línea que fija el perfil de bug del carril SDD
	// (REQ-29.15): un único contrato para los dos carriles.
	impactMarkerLine   = isoevidence.NoImpactMark
	impactReasonPrefix = "Motivo:"
	capabilityPrefix   = "### Capacidad:"
	openspecFenceLabel = "openspec"
)

var (
	capabilityName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	kindLine       = regexp.MustCompile(`^Tipo:[ \t]*(.*?)[ \t]*$`)
	checkboxInText = regexp.MustCompile(`^\s*- \[[ xX]\]|^#{2,6}\s+\[[ xX]\]`)
)

// FeatureDoc es el contenido de un documento ODD que le importa a la spec viva.
type FeatureDoc struct {
	Feature string
	Kind    Kind
	// Capabilities son las capacidades de `## Spec viva`, en el orden del documento.
	Capabilities []CapabilityDelta
	// Impact es la sección `## Impacto en la spec`; nil si el documento no la tiene.
	Impact *Impact
	// SpecSection dice si el documento tiene la sección `## Spec viva`, aunque esté vacía.
	SpecSection bool
}

// HasDelta dice si el documento aporta alguna capacidad con delta.
func (d FeatureDoc) HasDelta() bool { return len(d.Capabilities) > 0 }

// CapabilityDelta es la delta OpenSpec de una capacidad.
type CapabilityDelta struct {
	Name string
	// Line es la línea (desde 1) del `### Capacidad:` en el documento.
	Line  int
	Delta openspecdoc.Delta
	// Text es el contenido de la valla `openspec`, sin las vallas, con LF.
	Text string
}

// Impact es la sección `## Impacto en la spec` y el resultado de leer su marcador.
type Impact struct {
	// NoImpact es verdadero solo con el marcador válido: la línea exacta
	// `Sin impacto en la spec` y una línea `Motivo:` con texto.
	NoImpact bool
	Reason   string
	// Line es la línea (desde 1) del encabezado de la sección.
	Line int
	// Defect es "" con el marcador válido; si no, ImpactMarkerMissing o ImpactReasonMissing.
	Defect string
}

// Problem es un defecto del documento con su código estable, su línea, el sujeto que
// nombra (la capacidad, la sección) y el mensaje con lo que hay que corregir.
type Problem struct {
	Code    string `json:"code"`
	Line    int    `json:"line"`
	Subject string `json:"subject,omitempty"`
	Message string `json:"message"`
}

// DocError agrupa todos los defectos de un documento ODD; ParseFeatureDoc no se detiene
// en el primero.
type DocError struct {
	Feature  string
	Problems []Problem
}

func (e *DocError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "odd/tasks/%s.md no es válido para la spec viva (%d problema(s))", e.Feature, len(e.Problems))
	for _, p := range e.Problems {
		fmt.Fprintf(&b, "\n  - [%s] línea %d", p.Code, p.Line)
		if p.Subject != "" {
			fmt.Fprintf(&b, " (%s)", p.Subject)
		}
		fmt.Fprintf(&b, ": %s", p.Message)
	}
	return b.String()
}

// docLine es una línea del documento con su estado de valla.
type docLine struct {
	n      int
	text   string
	fenced bool // dentro de una valla, incluidas las líneas de apertura y de cierre
	open   bool // línea de apertura de una valla
	close  bool // línea de cierre de una valla
	info   string
}

// scanLines recorre el documento marcando las vallas con la misma regla que la
// composición: abre cualquier ``` o ~~~ (con o sin etiqueta) y cierra la misma clase de
// valla, con una longitud no menor y sin etiqueta. Una valla sin cerrar llega al final.
func scanLines(text string) []docLine {
	raw := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	lines := make([]docLine, 0, len(raw))
	inFence, fenceChar, fenceLen := false, byte(0), 0
	for i, line := range raw {
		l := docLine{n: i + 1, text: line}
		ch, n, hasInfo, ok := openspecdoc.FenceMarker(line)
		switch {
		case !inFence && ok:
			inFence, fenceChar, fenceLen = true, ch, n
			l.fenced, l.open = true, true
			l.info = strings.TrimSpace(line[n:])
		case inFence && ok && ch == fenceChar && n >= fenceLen && !hasInfo:
			inFence = false
			l.fenced, l.close = true, true
		default:
			l.fenced = inFence
		}
		lines = append(lines, l)
	}
	return lines
}

type docSection struct {
	name  string
	line  int
	lines []docLine // las líneas del cuerpo, sin el encabezado
}

// splitSections devuelve las líneas de la cabecera (antes del primer `## `) y las
// secciones de nivel 2, ignorando los encabezados dentro de vallas.
func splitSections(lines []docLine) (header []docLine, sections []docSection) {
	var current *docSection
	for _, l := range lines {
		if !l.fenced && strings.HasPrefix(l.text, "## ") {
			sections = append(sections, docSection{name: strings.TrimSpace(l.text[3:]), line: l.n})
			current = &sections[len(sections)-1]
			continue
		}
		if current == nil {
			header = append(header, l)
		} else {
			current.lines = append(current.lines, l)
		}
	}
	return header, sections
}

// ParseFeatureDoc analiza el documento ODD odd/tasks/<feature>.md (REQ-30.1, REQ-30.5):
// el tipo de la cabecera, la sección `## Spec viva` con una delta por capacidad y la
// sección `## Impacto en la spec`. Devuelve un *DocError con todos los defectos del
// documento. No decide si un bug sin delta ni marcador se rechaza (eso es CheckImpact)
// ni mira la spec viva del proyecto (eso es PlanIntegration).
func ParseFeatureDoc(feature string, data []byte) (FeatureDoc, error) {
	doc := FeatureDoc{Feature: feature, Kind: KindIncrement}
	var problems []Problem
	add := func(code string, line int, subject, message string) {
		problems = append(problems, Problem{Code: code, Line: line, Subject: subject, Message: message})
	}

	lines := scanLines(string(data))
	header, sections := splitSections(lines)

	for _, l := range header {
		if l.fenced {
			continue
		}
		m := kindLine.FindStringSubmatch(l.text)
		if m == nil {
			continue
		}
		switch m[1] {
		case string(KindBug):
			doc.Kind = KindBug
		case "incremento":
			doc.Kind = KindIncrement
		default:
			add("kind_invalid", l.n, m[1], fmt.Sprintf("`Tipo: %s` no es un tipo válido; usa `Tipo: bug` o `Tipo: incremento` (o quita la línea: es incremento)", m[1]))
		}
		break
	}

	seenSpec, seenImpact := false, false
	for _, section := range sections {
		switch section.name {
		case specSectionHeading:
			if seenSpec {
				add("section_duplicate", section.line, "## "+specSectionHeading, "el documento tiene dos secciones `## Spec viva`; deja una sola con una `### Capacidad:` por capacidad")
				continue
			}
			seenSpec, doc.SpecSection = true, true
			doc.Capabilities = parseCapabilities(section, add)
		case impactSectionHeading:
			if seenImpact {
				add("section_duplicate", section.line, "## "+impactSectionHeading, "el documento tiene dos secciones `## Impacto en la spec`; deja una sola")
				continue
			}
			seenImpact = true
			doc.Impact = parseImpact(section)
		}
	}

	if len(problems) > 0 {
		return FeatureDoc{}, &DocError{Feature: feature, Problems: problems}
	}
	return doc, nil
}

// parseCapabilities lee los subapartados `### Capacidad:` de la sección `## Spec viva`.
func parseCapabilities(section docSection, add func(code string, line int, subject, message string)) []CapabilityDelta {
	type group struct {
		name  string
		line  int
		lines []docLine
	}
	var groups []*group
	for _, l := range section.lines {
		if !l.fenced && strings.HasPrefix(l.text, capabilityPrefix) {
			groups = append(groups, &group{name: strings.TrimSpace(l.text[len(capabilityPrefix):]), line: l.n})
			continue
		}
		if len(groups) > 0 {
			last := groups[len(groups)-1]
			last.lines = append(last.lines, l)
		}
	}

	var capabilities []CapabilityDelta
	seen := map[string]bool{}
	for _, g := range groups {
		if !capabilityName.MatchString(g.name) {
			add("capability_invalid", g.line, g.name, fmt.Sprintf("el dominio %q de `### Capacidad:` no es válido: usa minúsculas, dígitos y guiones, empezando por una letra o un dígito (por ejemplo `facturacion`)", g.name))
			continue
		}
		if seen[g.name] {
			add("capability_duplicate", g.line, g.name, fmt.Sprintf("la capacidad %s aparece en dos subapartados; reúne sus requisitos en un solo bloque `openspec`", g.name))
			continue
		}
		seen[g.name] = true

		// Los bloques vallados de primer nivel con etiqueta `openspec`; el resto se ignora.
		var blocks [][]docLine
		var inBlock bool
		var current []docLine
		unclosed := false
		for _, l := range g.lines {
			switch {
			case l.open:
				inBlock, current = false, nil
				if fields := strings.Fields(l.info); len(fields) > 0 && fields[0] == openspecFenceLabel {
					inBlock = true
					unclosed = true
				}
			case l.close:
				if inBlock {
					blocks = append(blocks, current)
					inBlock, unclosed = false, false
				}
			case inBlock:
				current = append(current, l)
			}
		}
		switch {
		case unclosed:
			add("unclosed_fence", g.line, g.name, fmt.Sprintf("el bloque `openspec` de la capacidad %s no se cierra; cierra la valla con la misma clase de marca y una longitud no menor", g.name))
			continue
		case len(blocks) == 0:
			add("no_fence", g.line, g.name, fmt.Sprintf("la capacidad %s no tiene un bloque de código vallado con la etiqueta `openspec` (```openspec); la delta va dentro de la valla", g.name))
			continue
		case len(blocks) > 1:
			add("multiple_fences", g.line, g.name, fmt.Sprintf("la capacidad %s tiene %d bloques `openspec`; debe tener exactamente uno", g.name, len(blocks)))
			continue
		}

		var text strings.Builder
		for _, l := range blocks[0] {
			text.WriteString(l.text)
			text.WriteString("\n")
		}
		delta := openspecdoc.ParseDelta(text.String())
		if delta.Empty() {
			add("delta_unparseable", g.line, g.name, fmt.Sprintf("el bloque `openspec` de la capacidad %s no se puede analizar como delta: necesita `## ADDED Requirements`, `## MODIFIED Requirements`, `## REMOVED Requirements` o `## RENAMED Requirements` con `### Requirement:`", g.name))
			continue
		}
		if len(delta.Unknown) > 0 {
			add("delta_unknown_section", g.line, g.name, fmt.Sprintf("la delta de la capacidad %s tiene secciones que no son una operación (%s) y se perderían; quítalas o muévelas fuera de la valla", g.name, strings.Join(delta.Unknown, ", ")))
			continue
		}
		checkbox := false
		for _, l := range blocks[0] {
			if checkboxInText.MatchString(l.text) {
				checkbox = true
				add("checkbox_in_delta", l.n, g.name, fmt.Sprintf("la delta de la capacidad %s contiene una casilla (%s) y el listado del documento la contaría como tarea; reescríbela sin `- [ ]` ni `- [x]`", g.name, strings.TrimSpace(l.text)))
				break
			}
		}
		if checkbox {
			continue
		}
		capabilities = append(capabilities, CapabilityDelta{Name: g.name, Line: g.line, Delta: delta, Text: text.String()})
	}
	return capabilities
}

// parseImpact lee el marcador de la sección `## Impacto en la spec`: la línea exacta
// `Sin impacto en la spec` (se ignoran solo los espacios iniciales y finales) y una
// línea `Motivo:` con texto. Las líneas dentro de vallas son ejemplos y no cuentan.
func parseImpact(section docSection) *Impact {
	impact := &Impact{Line: section.line}
	marker, reasonSeen := false, false
	for _, l := range section.lines {
		if l.fenced {
			continue
		}
		trimmed := strings.TrimSpace(l.text)
		switch {
		case trimmed == impactMarkerLine:
			marker = true
		case strings.HasPrefix(trimmed, impactReasonPrefix) && !reasonSeen:
			// Solo cuenta la primera línea `Motivo:`, como en el carril SDD: si está
			// vacía, el marcador queda incompleto aunque haya otra con texto.
			reasonSeen = true
			impact.Reason = strings.TrimSpace(trimmed[len(impactReasonPrefix):])
		}
	}
	switch {
	case !marker:
		impact.Defect, impact.Reason = ImpactMarkerMissing, ""
	case impact.Reason == "":
		impact.Defect = ImpactReasonMissing
	default:
		impact.NoImpact = true
	}
	return impact
}
