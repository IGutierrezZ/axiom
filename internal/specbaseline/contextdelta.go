package specbaseline

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// ContextAction es la acción de una entrada del delta de contexto (REQ-24.31).
type ContextAction string

const (
	// ContextAdd añade una sección, o la crea junto con el documento si no existe.
	ContextAdd ContextAction = "add"
	// ContextModify modifica una sección existente.
	ContextModify ContextAction = "modify"
	// ContextRemove elimina una sección existente.
	ContextRemove ContextAction = "remove"
)

// ContextEntry es una entrada del delta de contexto: el documento, el ancla, la
// acción y el motivo, con un cuerpo opcional que describe el texto que integrar.
type ContextEntry struct {
	ID       string // C1..Cn por orden de aparición
	Document string // relativo a workspaceRoot, con '/'
	Anchor   Anchor
	Action   ContextAction
	Reason   string
	Body     string // sin líneas en blanco al principio ni al final
	Line     int    // línea del encabezado `## Entrada:`, desde 1
}

// ContextDelta es el contenido de `context-impact.md`: o bien entradas, o bien
// la declaración explícita «Sin impacto» con su motivo, nunca ambas (REQ-24.31).
type ContextDelta struct {
	NoImpact bool
	Reason   string // motivo de «Sin impacto»
	Entries  []ContextEntry
}

// ContextProblem es un defecto del formato del delta de contexto.
type ContextProblem struct {
	Line    int // desde 1; 0 si afecta al fichero entero
	Message string
}

// ContextDeltaError reúne todos los defectos del formato, no solo el primero.
type ContextDeltaError struct {
	Problems []ContextProblem
}

func (e *ContextDeltaError) Error() string {
	parts := make([]string, len(e.Problems))
	for i, problem := range e.Problems {
		if problem.Line > 0 {
			parts[i] = fmt.Sprintf("línea %d: %s", problem.Line, problem.Message)
		} else {
			parts[i] = problem.Message
		}
	}
	return "el delta de contexto no es válido: " + strings.Join(parts, "; ")
}

var (
	contextEntryHeading = regexp.MustCompile(`^## Entrada:[ \t]*(.*?)[ \t]*$`)
	contextNoImpact     = regexp.MustCompile(`^Sin impacto[ \t]*(?::[ \t]*(.*?))?[ \t]*$`)
	contextField        = regexp.MustCompile(`^- ([^:]*?)[ \t]*:[ \t]*(.*?)[ \t]*$`)
	driveLetter         = regexp.MustCompile(`^[A-Za-z]:`)
)

// Nombres exactos de los campos de una entrada (D-16).
const (
	fieldAnchor = "Ancla"
	fieldAction = "Acción"
	fieldReason = "Motivo"
)

// ParseContextDelta lee `context-impact.md` de forma determinista (D-16). Fuera de
// bloques de código con vallas, el fichero es un encabezado `# ...` opcional y o
// bien una línea `Sin impacto: <motivo>`, o bien una o más entradas. Cada entrada
// es `## Entrada: <ruta>`, los campos exactos `- Ancla:`, `- Acción:` y
// `- Motivo:` seguidos de una línea en blanco, y un cuerpo opcional hasta la
// siguiente entrada. Devuelve *ContextDeltaError con todos los defectos; las
// entradas legibles se devuelven aunque haya defectos.
func ParseContextDelta(data []byte) (ContextDelta, error) {
	doc := ParseDocument(data)
	fenced := fencedLines(doc.Lines)

	var delta ContextDelta
	var problems []ContextProblem
	problem := func(line int, format string, args ...any) {
		problems = append(problems, ContextProblem{Line: line, Message: fmt.Sprintf(format, args...)})
	}

	var headings []int // índices de los encabezados de entrada
	for i, line := range doc.Lines {
		if !fenced[i] && contextEntryHeading.MatchString(line.Text) {
			headings = append(headings, i)
		}
	}
	preambleEnd := len(doc.Lines)
	if len(headings) > 0 {
		preambleEnd = headings[0]
	}

	noImpactLine := 0
	for i := 0; i < preambleEnd; i++ {
		text := doc.Lines[i].Text
		if fenced[i] {
			if i == 0 || !fenced[i-1] {
				problem(i+1, "un bloque de código fuera de una entrada no se interpreta: colócalo en el cuerpo de una entrada")
			}
			continue
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		if level, _, ok := parseATX(text); ok && level == 1 {
			continue
		}
		if match := contextNoImpact.FindStringSubmatch(strings.TrimSpace(text)); match != nil {
			if noImpactLine != 0 {
				problem(i+1, "«Sin impacto» aparece más de una vez (también en la línea %d)", noImpactLine)
				continue
			}
			noImpactLine = i + 1
			delta.NoImpact, delta.Reason = true, strings.TrimSpace(match[1])
			if delta.Reason == "" {
				problem(i+1, "«Sin impacto» requiere un motivo: escribe `Sin impacto: <motivo>`")
			}
			continue
		}
		problem(i+1, "texto no reconocido fuera de una entrada %q: el delta solo contiene un encabezado `# ...`, una línea `Sin impacto: <motivo>` o entradas `## Entrada: <ruta>`", strings.TrimSpace(text))
	}

	for n, start := range headings {
		end := len(doc.Lines)
		if n+1 < len(headings) {
			end = headings[n+1]
		}
		entry, entryProblems := parseContextEntry(doc, fenced, start, end, fmt.Sprintf("C%d", n+1))
		problems = append(problems, entryProblems...)
		delta.Entries = append(delta.Entries, entry)
	}

	switch {
	case delta.NoImpact && len(delta.Entries) > 0:
		problem(noImpactLine, "«Sin impacto» y las entradas `## Entrada:` no pueden coexistir: o el cambio no afecta al contexto, o describe sus entradas")
	case !delta.NoImpact && len(delta.Entries) == 0 && len(problems) == 0:
		problem(0, "el fichero no contiene ninguna entrada `## Entrada: <ruta>` ni la declaración `Sin impacto: <motivo>`")
	}

	seen := map[string]ContextEntry{}
	for _, entry := range delta.Entries {
		if entry.Document == "" || len(entry.Anchor.Path) == 0 {
			continue
		}
		key := strings.ToLower(entry.Document) + "\x00" + entry.Anchor.String()
		if first, ok := seen[key]; ok {
			problem(entry.Line, "la entrada %s repite el documento %q y el ancla %q de la entrada %s (línea %d)", entry.ID, entry.Document, entry.Anchor, first.ID, first.Line)
			continue
		}
		seen[key] = entry
	}

	if len(problems) > 0 {
		return delta, &ContextDeltaError{Problems: problems}
	}
	return delta, nil
}

// parseContextEntry lee la entrada cuyo encabezado está en doc.Lines[start] y
// llega hasta end.
func parseContextEntry(doc Document, fenced []bool, start, end int, id string) (ContextEntry, []ContextProblem) {
	entry := ContextEntry{ID: id, Line: start + 1}
	var problems []ContextProblem
	problem := func(line int, format string, args ...any) {
		problems = append(problems, ContextProblem{Line: line, Message: fmt.Sprintf(format, args...)})
	}

	match := contextEntryHeading.FindStringSubmatch(doc.Lines[start].Text)
	document, err := normalizeContextDocument(match[1])
	if err != nil {
		problem(start+1, "la entrada %s: %v", id, err)
	}
	entry.Document = document

	at := start + 1
	for at < end && strings.TrimSpace(doc.Lines[at].Text) == "" {
		at++
	}
	fields := map[string]string{}
	for ; at < end && !fenced[at] && strings.HasPrefix(doc.Lines[at].Text, "- "); at++ {
		number := at + 1
		parts := contextField.FindStringSubmatch(doc.Lines[at].Text)
		if parts == nil {
			problem(number, "la entrada %s: la línea %q no es un campo `- <Nombre>: <valor>`; los campos son `- %s:`, `- %s:` y `- %s:`", id, strings.TrimSpace(doc.Lines[at].Text), fieldAnchor, fieldAction, fieldReason)
			continue
		}
		name, value := parts[1], parts[2]
		switch name {
		case fieldAnchor, fieldAction, fieldReason:
		default:
			problem(number, "la entrada %s: campo desconocido %q; los campos exactos son `- %s:`, `- %s:` y `- %s:`", id, name, fieldAnchor, fieldAction, fieldReason)
			continue
		}
		if _, dup := fields[name]; dup {
			problem(number, "la entrada %s: el campo %q está repetido", id, name)
			continue
		}
		fields[name] = value
		if value == "" {
			problem(number, "la entrada %s: el campo %q no puede estar vacío", id, name)
		}
	}
	for _, name := range []string{fieldAnchor, fieldAction, fieldReason} {
		if _, ok := fields[name]; !ok {
			problem(start+1, "la entrada %s: falta el campo `- %s:`", id, name)
		}
	}

	if value := fields[fieldAnchor]; value != "" {
		anchor, err := ParseAnchor(value)
		if err != nil {
			problem(start+1, "la entrada %s: %v", id, err)
		}
		entry.Anchor = anchor
	}
	switch action := ContextAction(fields[fieldAction]); action {
	case ContextAdd, ContextModify, ContextRemove:
		entry.Action = action
	case "":
	default:
		problem(start+1, "la entrada %s: la acción %q no es válida; usa add, modify o remove", id, action)
	}
	entry.Reason = fields[fieldReason]

	body := doc.Lines[at:end]
	for len(body) > 0 && strings.TrimSpace(body[0].Text) == "" {
		body = body[1:]
	}
	for len(body) > 0 && strings.TrimSpace(body[len(body)-1].Text) == "" {
		body = body[:len(body)-1]
	}
	texts := make([]string, len(body))
	for i, line := range body {
		texts[i] = line.Text
	}
	entry.Body = strings.Join(texts, "\n")
	return entry, problems
}

// normalizeContextDocument lee la ruta de un encabezado `## Entrada:`: relativa
// a workspaceRoot, con `/` o `\` y, opcionalmente, entre comillas invertidas.
// Devuelve la ruta con `/`.
func normalizeContextDocument(raw string) (string, error) {
	text := strings.TrimSpace(raw)
	if len(text) >= 2 && strings.HasPrefix(text, "`") && strings.HasSuffix(text, "`") {
		text = strings.TrimSpace(text[1 : len(text)-1])
	}
	if text == "" {
		return "", fmt.Errorf("el encabezado no indica la ruta del documento: escribe `## Entrada: <ruta relativa a workspaceRoot>`")
	}
	slashed := strings.ReplaceAll(text, `\`, "/")
	if strings.HasPrefix(slashed, "/") || driveLetter.MatchString(slashed) {
		return "", fmt.Errorf("la ruta %q debe ser relativa a workspaceRoot, sin barra inicial ni unidad de disco", text)
	}
	clean := path.Clean(slashed)
	if clean == "." || strings.HasSuffix(slashed, "/") {
		return "", fmt.Errorf("la ruta %q debe apuntar a un documento, no a un directorio", text)
	}
	return clean, nil
}

// ContextScope reúne las rutas que deciden si el documento de una entrada puede
// editarse (REQ-24.30, D-16). Todas son relativas a WorkspaceRoot, con `/`; las
// de contexto pueden salir de él hacia un repositorio hermano (`../repo/docs/`).
type ContextScope struct {
	WorkspaceRoot     string   // absoluta
	Maintained        []string // technical_context.maintained.paths
	Reference         []string // technical_context.reference.paths
	BaselineDocuments []string // baseline.documents[].path
}

// Resolve devuelve la ruta absoluta de un documento del delta.
func (s ContextScope) Resolve(document string) string {
	return filepath.Join(s.WorkspaceRoot, filepath.FromSlash(document))
}

// CheckDocument comprueba, por identidad y no por texto, que el documento de la
// entrada está bajo alguna ruta mantenida y que no está bajo una ruta de
// referencia ni es un documento de la baseline (REQ-24.30, REQ-24.31, D-16).
func (s ContextScope) CheckDocument(entry ContextEntry) error {
	target := resolveDeepest(s.Resolve(entry.Document))
	for _, declared := range s.Reference {
		if underPath(s.Resolve(declared), target) {
			return fmt.Errorf("la entrada %s (línea %d) apunta a %s, que está bajo la ruta de referencia %q (modo reference): las rutas de referencia nunca se modifican", entry.ID, entry.Line, entry.Document, declared)
		}
	}
	for _, document := range s.BaselineDocuments {
		if underPath(s.Resolve(document), target) {
			return fmt.Errorf("la entrada %s (línea %d) apunta a %s, que es un documento de 'baseline.documents': la baseline se integra por composición o edición dirigida de su dominio, no por el delta de contexto", entry.ID, entry.Line, entry.Document)
		}
	}
	for _, declared := range s.Maintained {
		if underPath(s.Resolve(declared), target) {
			return nil
		}
	}
	return fmt.Errorf("la entrada %s (línea %d) apunta a %s, que no está bajo ninguna ruta mantenida; las rutas mantenidas declaradas son: %s", entry.ID, entry.Line, entry.Document, quoteList(s.Maintained))
}

func quoteList(items []string) string {
	if len(items) == 0 {
		return "(ninguna)"
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return strings.Join(quoted, ", ")
}

// underPath dice si target es declared o está dentro de él. Decide por identidad
// (pathidentity.Contains para un directorio, os.SameFile para un fichero), nunca
// comparando cadenas, para tolerar mayúsculas y enlaces simbólicos (H14). Una
// ruta declarada que aún no existe no tiene identidad: solo entonces se compara
// de forma léxica, porque no hay nada que identificar (workspace validate solo
// avisa de las rutas de contexto inexistentes).
func underPath(declared, target string) bool {
	info, err := os.Stat(declared)
	switch {
	case err == nil && info.IsDir():
		return pathidentity.Contains(declared, target)
	case err == nil:
		other, statErr := os.Stat(target)
		return statErr == nil && os.SameFile(info, other)
	default:
		return lexicallyUnder(resolveDeepest(declared), target)
	}
}

// lexicallyUnder es la comparación léxica de respaldo de underPath, sin
// distinguir mayúsculas en Windows.
func lexicallyUnder(declared, target string) bool {
	declared, target = filepath.Clean(declared), filepath.Clean(target)
	if runtime.GOOS == "windows" {
		declared, target = strings.ToLower(declared), strings.ToLower(target)
	}
	return target == declared || strings.HasPrefix(target, declared+string(filepath.Separator))
}

// resolveDeepest resuelve los enlaces simbólicos del tramo existente de una ruta
// y conserva tal cual el resto, de modo que un documento que aún no existe se
// juzga por el lugar real de su directorio.
func resolveDeepest(p string) string {
	p = filepath.Clean(p)
	rest := ""
	for {
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			return filepath.Join(resolved, rest)
		}
		parent := filepath.Dir(p)
		if parent == p {
			return filepath.Join(p, rest)
		}
		rest = filepath.Join(filepath.Base(p), rest)
		p = parent
	}
}
