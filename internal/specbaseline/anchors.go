package specbaseline

import (
	"fmt"
	"strconv"
	"strings"
)

// anchorSeparator separa los elementos de la ruta de un ancla (REQ-24.32, D-16).
const anchorSeparator = " > "

// Anchor es el ancla de una entrada del delta de contexto: la ruta de títulos
// que identifica una sección de un documento, con un ordinal opcional en el
// último elemento (D-16). La sintaxis es `Título`, `Padre > Título` o
// `Abuelo > Padre > Título`, y el último elemento admite el sufijo ` #N`.
type Anchor struct {
	Path    []string
	Ordinal int // 0 = sin ordinal; N elige la N-ésima coincidencia en orden del documento
}

// String devuelve el ancla en su forma de texto, con el ordinal si lo tiene.
func (a Anchor) String() string {
	text := strings.Join(a.Path, anchorSeparator)
	if a.Ordinal > 0 {
		text += " #" + strconv.Itoa(a.Ordinal)
	}
	return text
}

// ParseAnchor lee un ancla. Cada elemento de la ruta se recorta de espacios y
// ninguno puede quedar vacío. Un sufijo ` #N` del último elemento es el ordinal
// de la coincidencia, con N entero mayor que cero; un título que termina en
// ` #N` no se puede anclar sin ordinal (limitación documentada, R33).
func ParseAnchor(text string) (Anchor, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Anchor{}, fmt.Errorf("el ancla está vacía: indica el título de la sección, o su ruta de encabezados ancestros separada por %q", strings.TrimSpace(anchorSeparator))
	}
	parts := strings.Split(text, anchorSeparator)
	anchor := Anchor{Path: make([]string, len(parts))}
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return Anchor{}, fmt.Errorf("el ancla %q tiene un elemento vacío: cada elemento de la ruta debe ser un título", text)
		}
		anchor.Path[i] = part
	}
	last := len(anchor.Path) - 1
	title, ordinal, hasOrdinal := splitOrdinal(anchor.Path[last])
	if hasOrdinal {
		if ordinal < 1 {
			return Anchor{}, fmt.Errorf("el ordinal del ancla %q debe ser un entero mayor que cero: ` #1` elige la primera coincidencia", text)
		}
		if title == "" {
			return Anchor{}, fmt.Errorf("el ancla %q solo lleva el ordinal: añade antes el título de la sección", text)
		}
		anchor.Path[last], anchor.Ordinal = title, ordinal
	}
	return anchor, nil
}

// splitOrdinal separa el sufijo ` #N` de un título. Un número que no cabe en un
// entero se devuelve como cero, que ParseAnchor rechaza.
func splitOrdinal(text string) (title string, ordinal int, ok bool) {
	at := strings.LastIndex(text, " #")
	if at < 0 {
		return text, 0, false
	}
	digits := text[at+2:]
	if digits == "" {
		return text, 0, false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return text, 0, false
		}
	}
	number, err := strconv.Atoi(digits)
	if err != nil {
		number = 0
	}
	return strings.TrimSpace(text[:at]), number, true
}

// HeadingRef identifica un encabezado ATX del documento.
type HeadingRef struct {
	Line  int // desde 1
	Level int
	Path  []string // títulos desde el encabezado raíz hasta este
}

// FullPath devuelve la ruta completa del encabezado como texto.
func (h HeadingRef) FullPath() string { return strings.Join(h.Path, anchorSeparator) }

// documentHeadingRefs devuelve los encabezados ATX del documento, fuera de
// vallas, cada uno con la cadena de títulos de sus ancestros: un encabezado es
// descendiente del anterior de nivel menor.
func documentHeadingRefs(data []byte) []HeadingRef {
	type frame struct {
		level int
		title string
	}
	var stack []frame
	var refs []HeadingRef
	for _, heading := range ParseDocument(data).Headings() {
		for len(stack) > 0 && stack[len(stack)-1].level >= heading.Level {
			stack = stack[:len(stack)-1]
		}
		stack = append(stack, frame{level: heading.Level, title: heading.Title})
		path := make([]string, len(stack))
		for i, f := range stack {
			path[i] = f.title
		}
		refs = append(refs, HeadingRef{Line: heading.Index + 1, Level: heading.Level, Path: path})
	}
	return refs
}

// ResolveAnchor devuelve los encabezados del documento a los que resuelve el
// ancla. Un encabezado coincide si su título es igual al último elemento y los
// elementos anteriores coinciden con su cadena de ancestros inmediatos, contigua
// desde el padre hacia arriba (D-16). Con ordinal devuelve, como mucho, la
// N-ésima coincidencia en orden del documento. Resolver no valida: que no haya
// coincidencias o haya varias lo decide quien llama según la acción.
func ResolveAnchor(doc []byte, anchor Anchor) []HeadingRef {
	if len(anchor.Path) == 0 {
		return nil
	}
	last := len(anchor.Path) - 1
	var matches []HeadingRef
	for _, ref := range documentHeadingRefs(doc) {
		if ref.Path[len(ref.Path)-1] != anchor.Path[last] {
			continue
		}
		ancestors := ref.Path[:len(ref.Path)-1]
		wanted := anchor.Path[:last]
		if len(ancestors) < len(wanted) || !equalStrings(ancestors[len(ancestors)-len(wanted):], wanted) {
			continue
		}
		matches = append(matches, ref)
	}
	if anchor.Ordinal > 0 {
		if anchor.Ordinal > len(matches) {
			return nil
		}
		return matches[anchor.Ordinal-1 : anchor.Ordinal]
	}
	return matches
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// unaddressable son las apariciones de un encabezado que el ancla describe pero
// que el formato no puede direccionar: un título con ` > ` en un encabezado ATX
// o un encabezado setext (R33). Se nombran en el rechazo para que quien redacta
// el delta entienda por qué un título visible no resuelve.
func unaddressable(doc []byte, anchor Anchor) []string {
	var found []string
	joined := strings.Join(anchor.Path, anchorSeparator)
	last := anchor.Path[len(anchor.Path)-1]
	for _, ref := range documentHeadingRefs(doc) {
		title := ref.Path[len(ref.Path)-1]
		if strings.Contains(title, anchorSeparator) && title == joined {
			found = append(found, fmt.Sprintf("línea %d %q: su título contiene %q y no se puede anclar", ref.Line, title, strings.TrimSpace(anchorSeparator)))
		}
	}
	for _, line := range setextHeadings(doc) {
		if line.title == last || line.title == joined {
			found = append(found, fmt.Sprintf("línea %d %q: es un encabezado setext y solo se reconocen los encabezados ATX (`#`)", line.number, line.title))
		}
	}
	return found
}

type setextHeading struct {
	number int // línea del texto, desde 1
	title  string
}

// setextHeadings localiza, fuera de vallas, las líneas de texto subrayadas con
// `===` o `---`, que otros renderizadores muestran como encabezados.
func setextHeadings(data []byte) []setextHeading {
	doc := ParseDocument(data)
	fenced := fencedLines(doc.Lines)
	var found []setextHeading
	for i := 0; i+1 < len(doc.Lines); i++ {
		if fenced[i] || fenced[i+1] {
			continue
		}
		text := strings.TrimSpace(doc.Lines[i].Text)
		if text == "" || isSetextUnderline(doc.Lines[i].Text) {
			continue
		}
		if _, _, atx := parseATX(doc.Lines[i].Text); atx {
			continue
		}
		if isSetextUnderline(doc.Lines[i+1].Text) {
			found = append(found, setextHeading{number: i + 1, title: text})
		}
	}
	return found
}

func isSetextUnderline(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || len(line)-len(strings.TrimLeft(line, " ")) > 3 {
		return false
	}
	return strings.Trim(trimmed, "=") == "" || strings.Trim(trimmed, "-") == ""
}

// CheckAnchor valida la acción de una entrada contra el estado del documento
// (REQ-24.32, REQ-24.55). exists dice si el documento existe; si no existe, solo
// `add` es válido, y archive lo crea. Con el documento existente: más de una
// coincidencia se rechaza, con la línea y la ruta completa de cada una; ninguna
// coincidencia solo es válida con `add`; y con una, las tres acciones son
// válidas. Devuelve si la entrada `add` crea el documento.
func CheckAnchor(entry ContextEntry, doc []byte, exists bool) (create bool, err error) {
	if !exists {
		if entry.Action != ContextAdd {
			return false, fmt.Errorf("la entrada %s (línea %d) con la acción %q apunta a %s, que no existe: solo la acción 'add' puede crear un documento mantenido", entry.ID, entry.Line, entry.Action, entry.Document)
		}
		return true, nil
	}
	matches := ResolveAnchor(doc, entry.Anchor)
	switch {
	case len(matches) > 1:
		found := make([]string, len(matches))
		for i, match := range matches {
			found[i] = fmt.Sprintf("línea %d %q", match.Line, match.FullPath())
		}
		return false, fmt.Errorf("la entrada %s (línea %d): el ancla %q es ambigua en %s y resuelve a %d secciones (%s); desambigua con la ruta de encabezados ancestros o con un ordinal ` #N`",
			entry.ID, entry.Line, entry.Anchor, entry.Document, len(matches), strings.Join(found, "; "))
	case len(matches) == 0:
		if hidden := unaddressable(doc, entry.Anchor); len(hidden) > 0 {
			return false, fmt.Errorf("la entrada %s (línea %d): el ancla %q no resuelve a ninguna sección de %s, pero el documento contiene encabezados que el formato no puede anclar (R33): %s",
				entry.ID, entry.Line, entry.Anchor, entry.Document, strings.Join(hidden, "; "))
		}
		if entry.Action != ContextAdd {
			return false, fmt.Errorf("la entrada %s (línea %d): el ancla %q no resuelve a ninguna sección de %s y la acción %q exige una sección existente; solo 'add' admite un ancla inexistente",
				entry.ID, entry.Line, entry.Anchor, entry.Document, entry.Action)
		}
	}
	return false, nil
}
