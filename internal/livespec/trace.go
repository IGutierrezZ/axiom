package livespec

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
)

// traceSchema es el esquema de la traza de identificadores asignados (D12).
const traceSchema = "axiom.requirement-ids/v1"

// traceSectionHeading es la sección del documento ODD que guarda la traza.
const traceSectionHeading = "Identificadores asignados"

// traceContinuation es la continuación común de los errores de la traza (D12).
const traceContinuation = "restaura la traza con git (`git checkout -- <ruta>`) o rehaz la integración desde la base; la traza nunca se regenera en silencio"

// TraceEntry es una asignación de la traza: el provisional que escribió el autor, el
// definitivo que recibió y el requisito (con el título que tenía al asignarse). Previous
// lista los definitivos anteriores si la entrada se reasignó.
type TraceEntry struct {
	Provisional string   `yaml:"provisional"`
	Definitive  string   `yaml:"definitive"`
	Capability  string   `yaml:"capability"`
	Requirement string   `yaml:"requirement"`
	Previous    []string `yaml:"previous,omitempty"`
}

// Trace es la fuente de identidad de lo ya asignado: SDD la guarda en
// `openspec/changes/<cambio>/ids.yaml` y ODD en la sección `## Identificadores asignados`
// del documento.
type Trace struct {
	Schema      string       `yaml:"schema"`
	Pattern     string       `yaml:"pattern"`
	Assignments []TraceEntry `yaml:"assignments"`
}

// traceError construye un *IDPlanError de la traza.
func traceError(code, file, reason string) *IDPlanError {
	return &IDPlanError{Code: code, File: file, Reason: reason, Continuation: traceContinuation}
}

// ParseTrace lee una traza YAML. Un YAML ilegible o un esquema desconocido dan
// id_trace_invalid; file solo sirve para los motivos.
func ParseTrace(data []byte, file string) (Trace, error) {
	var trace Trace
	if err := yaml.Unmarshal(data, &trace); err != nil {
		return Trace{}, traceError(CodeIDTraceInvalid, file, "la traza no es un YAML legible: "+err.Error())
	}
	if trace.Schema != traceSchema {
		return Trace{}, traceError(CodeIDTraceInvalid, file, fmt.Sprintf("el esquema %q de la traza no es %s", trace.Schema, traceSchema))
	}
	return trace, nil
}

// ReadTraceFile lee la traza de un fichero; uno que no existe da la traza vacía, que no
// asigna nada todavía.
func ReadTraceFile(path string) (Trace, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Trace{}, nil
	}
	if err != nil {
		return Trace{}, fmt.Errorf("no se puede leer %s: %w; comprueba los permisos del fichero", path, err)
	}
	return ParseTrace(data, path)
}

// validate comprueba la traza contra el patrón efectivo: el patrón de la traza debe ser el
// mismo, y los definitivos (también los anteriores) deben ser válidos y los provisionales
// legibles con el patrón. La traza vacía (sin esquema) es válida: no asigna nada.
func (t Trace) validate(p reqid.Pattern, file string) error {
	if t.Schema == "" && len(t.Assignments) == 0 {
		return nil
	}
	if t.Pattern != p.Template() {
		return traceError(CodeIDTracePatternMismatch, file, fmt.Sprintf("la traza se hizo con el patrón %q y el efectivo es %q", t.Pattern, p.Template()))
	}
	for _, e := range t.Assignments {
		if _, err := p.ParseProvisional(e.Provisional); err != nil {
			return traceError(CodeIDTraceInvalid, file, fmt.Sprintf("el provisional %q de la traza no es válido: %v", e.Provisional, err))
		}
		for _, id := range append([]string{e.Definitive}, e.Previous...) {
			if !p.Valid(id) {
				return traceError(CodeIDTraceInvalid, file, fmt.Sprintf("el definitivo %q de la traza no es un identificador válido para el patrón %s", id, p.Template()))
			}
		}
	}
	return nil
}

// Render devuelve el YAML de la traza; sin asignaciones no se escribe nada (nil).
func (t Trace) Render() []byte {
	if len(t.Assignments) == 0 {
		return nil
	}
	if t.Schema == "" {
		t.Schema = traceSchema
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	// La traza solo contiene cadenas: la serialización no puede fallar.
	if err := enc.Encode(t); err != nil {
		return nil
	}
	if err := enc.Close(); err != nil {
		return nil
	}
	return buf.Bytes()
}

// oddTraceSection ubica la sección de traza del documento: el índice de su encabezado, el
// del siguiente encabezado de nivel 2 (o el final) y las líneas. found es falso sin
// sección; dos secciones dan id_trace_invalid.
func oddTraceSection(doc, file string) (lines []docLine, start, end int, found bool, err error) {
	lines = scanLines(doc)
	start = -1
	for i, l := range lines {
		if l.fenced || !strings.HasPrefix(l.text, "## ") || strings.TrimSpace(l.text[3:]) != traceSectionHeading {
			continue
		}
		if start >= 0 {
			return nil, 0, 0, false, traceError(CodeIDTraceInvalid, file, "el documento tiene dos secciones `## "+traceSectionHeading+"`; deja una sola")
		}
		start = i
	}
	if start < 0 {
		return lines, 0, 0, false, nil
	}
	end = len(lines)
	for i := start + 1; i < len(lines); i++ {
		if !lines[i].fenced && strings.HasPrefix(lines[i].text, "## ") {
			end = i
			break
		}
	}
	return lines, start, end, true, nil
}

// ExtractODDTrace lee la traza de la sección `## Identificadores asignados` de un
// documento ODD: found es falso si no hay sección. La sección lleva un único bloque
// vallado `yaml`; un bloque ausente, repetido o ilegible da id_trace_invalid.
func ExtractODDTrace(doc, file string) (Trace, bool, error) {
	lines, start, end, found, err := oddTraceSection(doc, file)
	if err != nil || !found {
		return Trace{}, false, err
	}
	var blocks []string
	var current []string
	inYAML := false
	for _, l := range lines[start+1 : end] {
		switch {
		case l.open:
			fields := strings.Fields(l.info)
			inYAML = len(fields) > 0 && fields[0] == "yaml"
			current = nil
		case l.close:
			if inYAML {
				blocks = append(blocks, strings.Join(current, "\n")+"\n")
			}
			inYAML = false
		case inYAML:
			current = append(current, l.text)
		}
	}
	if len(blocks) != 1 {
		return Trace{}, true, traceError(CodeIDTraceInvalid, file, fmt.Sprintf("la sección `## %s` debe tener exactamente un bloque `yaml` y tiene %d", traceSectionHeading, len(blocks)))
	}
	trace, err := ParseTrace([]byte(blocks[0]), file)
	return trace, true, err
}

// SetODDTrace devuelve el documento con su sección de traza: la añade al final o
// reemplaza la existente, sin tocar el resto. Sin asignaciones devuelve el documento
// igual. Conserva el final de línea del documento.
func SetODDTrace(doc string, t Trace) (string, error) {
	rendered := t.Render()
	if rendered == nil {
		return doc, nil
	}
	eol := "\n"
	if strings.Contains(doc, "\r\n") {
		eol = "\r\n"
	}
	section := []string{"## " + traceSectionHeading, "", "```yaml"}
	section = append(section, strings.Split(strings.TrimRight(string(bytes.ReplaceAll(rendered, []byte("\r\n"), []byte("\n"))), "\n"), "\n")...)
	section = append(section, "```")

	lines, start, end, found, err := oddTraceSection(doc, "")
	if err != nil {
		return "", err
	}
	texts := make([]string, len(lines))
	for i, l := range lines {
		texts[i] = l.text
	}
	var out []string
	if found {
		out = append(out, texts[:start]...)
		out = append(out, section...)
		if end < len(texts) {
			out = append(out, "")
			out = append(out, texts[end:]...)
		} else {
			out = append(out, "")
		}
		return strings.Join(out, eol), nil
	}
	// scanLines deja una línea vacía final cuando el documento acaba en salto de línea.
	body := texts
	if n := len(body); n > 0 && body[n-1] == "" {
		body = body[:n-1]
	}
	out = append(out, body...)
	out = append(out, "")
	out = append(out, section...)
	out = append(out, "")
	return strings.Join(out, eol), nil
}
