package techcontext

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// PlanSchema es el esquema de `integration.yaml`. Es el mismo que escribe
	// `axiom sdd integrate`: el plan y la traza del contexto siguen en un solo
	// fichero (D3 de spec-viva-openspec).
	PlanSchema = "axiom.sdd-integration/v1"
	// TraceFileName es el nombre del plan y de la traza dentro de la carpeta del cambio.
	TraceFileName = "integration.yaml"
	// DeltaFileName es el nombre de trabajo del delta de contexto (D-16).
	DeltaFileName = "context-impact.md"
	// MaxDirtyEntries es el máximo de entradas sucias que el plan registra por
	// repositorio: acota el tamaño de integration.yaml.
	MaxDirtyEntries = 1000
)

// Decisiones del delta de contexto que el plan registra.
const (
	// ContextEntries: el delta de contexto describe entradas.
	ContextEntries = "entries"
	// ContextNoImpact: el delta de contexto declara «Sin impacto».
	ContextNoImpact = "no_impact"
	// ContextNone: no hay contexto mantenido, o no hay delta.
	ContextNone = "none"
)

type (
	// EntryKind es el tipo de una entrada de la traza.
	EntryKind string
	// TraceAction es la acción que registra una entrada de la traza.
	TraceAction string
	// TraceState es el estado de una entrada de la traza.
	TraceState string
)

const (
	// EntryContext es una entrada del contexto técnico mantenido, la única que
	// admite esta traza.
	EntryContext EntryKind = "context"

	// ActionAdded registra una sección añadida.
	ActionAdded TraceAction = "added"
	// ActionModified registra una sección modificada.
	ActionModified TraceAction = "modified"
	// ActionRemoved registra una sección eliminada.
	ActionRemoved TraceAction = "removed"
	// ActionCreated registra un documento creado.
	ActionCreated TraceAction = "created"

	// StatePending marca una entrada aún no integrada.
	StatePending TraceState = "pending"
	// StateIntegrated marca una entrada ya integrada.
	StateIntegrated TraceState = "integrated"
)

// TraceEntry es una entrada de la traza (REQ-24.52): una sección del contexto
// técnico mantenido.
type TraceEntry struct {
	ID       string      `yaml:"id" json:"id"`
	Kind     EntryKind   `yaml:"kind" json:"kind"`
	Document string      `yaml:"document" json:"document"` // relativo a workspaceRoot, con '/'
	Anchor   string      `yaml:"anchor" json:"anchor"`
	Action   TraceAction `yaml:"action" json:"action"`
	State    TraceState  `yaml:"state" json:"state"`
	Reason   string      `yaml:"reason,omitempty" json:"reason,omitempty"`
	Warnings []string    `yaml:"warnings,omitempty" json:"warnings,omitempty"`
}

// traceOptions son las opciones de supersede de un plan con baseline. El plan
// las escribe siempre; en una traza de contexto están vacías.
type traceOptions struct {
	Supersede              bool     `yaml:"supersede"`
	SupersededRequirements []string `yaml:"superseded_requirements"`
}

// TraceRepository es la línea base de un repositorio afectado: su HEAD y la foto
// de su suciedad previa.
type TraceRepository struct {
	Repo  string       `yaml:"repo"`  // relativo a la carpeta maestra, con '/'
	Head  string       `yaml:"head"`  // commit en prepare; "" si no hay commits
	Dirty []DirtyEntry `yaml:"dirty"` // como máximo MaxDirtyEntries
}

// DirtyEntry es un fichero con cambios sin confirmar en prepare.
type DirtyEntry struct {
	Path   string `yaml:"path"`   // relativa al repositorio, con '/'
	SHA256 string `yaml:"sha256"` // huella del árbol de trabajo; "" si el fichero estaba borrado
}

// ContextDecision es la decisión del cambio sobre el contexto técnico.
type ContextDecision struct {
	Decision string `yaml:"decision" json:"decision"` // entries | no_impact | none
	Reason   string `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// Trace es un subconjunto estricto de `specbaseline.Plan` (`integration.yaml`,
// esquema axiom.sdd-integration/v1): lo que `prepare` fija y `verify` comprueba
// del contexto técnico. Las claves de baseline (`options`, `id_map`, `documents`)
// se leen y se escriben vacías, de modo que `axiom sdd integrate` y `axiom
// archive context` se leen mutuamente el plan sin ventana rota.
type Trace struct {
	Schema       string            `yaml:"schema"`
	Change       string            `yaml:"change"`
	PreparedAt   time.Time         `yaml:"prepared_at"`
	VerifiedAt   *time.Time        `yaml:"verified_at,omitempty"`
	InputsSHA256 string            `yaml:"inputs_sha256"`
	Options      traceOptions      `yaml:"options"`
	Repositories []TraceRepository `yaml:"repositories"`
	IDMap        []any             `yaml:"id_map"`
	Documents    []any             `yaml:"documents"`
	Context      ContextDecision   `yaml:"context"`
	Entries      []TraceEntry      `yaml:"entries"`
}

// ForeignPlanError es el rechazo de un plan que no es solo de contexto técnico:
// lleva documentos de una baseline, correspondencia de IDs, opciones de supersede
// o entradas que no son de contexto. Lo escribió `axiom sdd integrate` con una
// baseline declarada.
type ForeignPlanError struct {
	File    string
	Reasons []string
}

func (e *ForeignPlanError) Error() string {
	file := e.File
	if file == "" {
		file = TraceFileName
	}
	return fmt.Sprintf("%s no es un plan de contexto técnico (%s): lo escribió `axiom sdd integrate` con una baseline y `axiom archive context` solo integra el contexto técnico. Continuación: continúa con `axiom sdd integrate verify` o consulta la traza con `axiom sdd integrate show`", file, strings.Join(e.Reasons, "; "))
}

// foreign devuelve las razones por las que el plan no es solo de contexto.
func (t Trace) foreign() []string {
	var reasons []string
	if t.Options.Supersede || len(t.Options.SupersededRequirements) > 0 {
		reasons = append(reasons, "options.supersede está activo o lista requisitos superados")
	}
	if len(t.IDMap) > 0 {
		reasons = append(reasons, fmt.Sprintf("id_map tiene %d entrada(s)", len(t.IDMap)))
	}
	if len(t.Documents) > 0 {
		reasons = append(reasons, fmt.Sprintf("documents tiene %d documento(s) de una baseline", len(t.Documents)))
	}
	for i, entry := range t.Entries {
		switch entry.Kind {
		case "compose", "directed", "register":
			reasons = append(reasons, fmt.Sprintf("entries[%d].kind es %s", i, entry.Kind))
		}
	}
	return reasons
}

// MarshalTrace serializa la traza (`integration.yaml`). La salida es
// determinista: la misma entrada produce los mismos bytes. No valida: DecodeTrace
// y WriteTrace sí lo hacen.
func MarshalTrace(trace Trace) ([]byte, error) {
	if !trace.PreparedAt.IsZero() {
		trace.PreparedAt = trace.PreparedAt.UTC()
	}
	if trace.VerifiedAt != nil {
		verified := trace.VerifiedAt.UTC()
		trace.VerifiedAt = &verified
	}
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(trace); err != nil {
		return nil, fmt.Errorf("serializar %s: %w", TraceFileName, err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("serializar %s: %w", TraceFileName, err)
	}
	return buffer.Bytes(), nil
}

// DecodeTrace lee `integration.yaml` de forma estricta: sin claves desconocidas,
// con un solo documento YAML, con el esquema axiom.sdd-integration/v1 y con
// rutas relativas con `/`. Un plan con documentos, correspondencia de IDs,
// supersede o entradas que no son de contexto devuelve *ForeignPlanError; el
// resto de defectos del contenido se reúnen en un solo error.
func DecodeTrace(data []byte) (Trace, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return Trace{}, fmt.Errorf("%s está vacío", TraceFileName)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var trace Trace
	if err := decoder.Decode(&trace); err != nil {
		return Trace{}, fmt.Errorf("%s no se puede leer: %w", TraceFileName, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Trace{}, fmt.Errorf("%s debe contener un solo documento YAML", TraceFileName)
	}
	if reasons := trace.foreign(); len(reasons) > 0 {
		return Trace{}, &ForeignPlanError{Reasons: reasons}
	}
	if problems := trace.problems(); len(problems) > 0 {
		return Trace{}, fmt.Errorf("%s no es válido: %s", TraceFileName, strings.Join(problems, "; "))
	}
	return trace, nil
}

// ReadTrace lee y valida la traza de la ruta. Si el fichero no existe, el error
// envuelve fs.ErrNotExist.
func ReadTrace(file string) (Trace, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return Trace{}, fmt.Errorf("leer %s: %w", file, err)
	}
	trace, err := DecodeTrace(data)
	if err != nil {
		var foreign *ForeignPlanError
		if errors.As(err, &foreign) {
			foreign.File = file
			return Trace{}, foreign
		}
		return Trace{}, fmt.Errorf("%s: %w", file, err)
	}
	return trace, nil
}

// WriteTrace valida la traza y la escribe con un fichero temporal en el mismo
// directorio y un renombrado, de modo que nunca queda un plan a medio escribir.
func WriteTrace(file string, trace Trace) error {
	if problems := trace.problems(); len(problems) > 0 {
		return fmt.Errorf("%s no es válido: %s", TraceFileName, strings.Join(problems, "; "))
	}
	data, err := MarshalTrace(trace)
	if err != nil {
		return err
	}
	return writeFileAtomic(file, data, 0o644)
}

// renameFile es el renombrado con el que se sustituyen los ficheros; las pruebas
// lo sustituyen para provocar un fallo a mitad de la escritura.
var renameFile = os.Rename

// writeFileAtomic escribe data en un temporal del mismo directorio y lo renombra
// sobre file: o queda el contenido anterior o queda el nuevo, nunca uno parcial.
func writeFileAtomic(file string, data []byte, perm os.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(file), "."+filepath.Base(file)+".tmp-*")
	if err != nil {
		return fmt.Errorf("crear el temporal de %s: %w", file, err)
	}
	name := temp.Name()
	cleanup := func(cause error) error {
		_ = temp.Close()
		_ = os.Remove(name)
		return cause
	}
	if _, err := temp.Write(data); err != nil {
		return cleanup(fmt.Errorf("escribir %s: %w", file, err))
	}
	if err := temp.Sync(); err != nil {
		return cleanup(fmt.Errorf("sincronizar %s: %w", file, err))
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("cerrar %s: %w", file, err)
	}
	if err := os.Chmod(name, perm); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("fijar los permisos de %s: %w", file, err)
	}
	if err := renameFile(name, file); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("sustituir %s: %w", file, err)
	}
	return nil
}

var (
	hexSHA256 = regexp.MustCompile(`^[0-9a-f]{64}$`)
	hexCommit = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
)

// problems valida la traza y devuelve todos sus defectos.
func (t Trace) problems() []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	if t.Schema != PlanSchema {
		add("schema debe ser %q (es %q)", PlanSchema, t.Schema)
	}
	if strings.TrimSpace(t.Change) == "" {
		add("change es obligatorio")
	}
	if t.PreparedAt.IsZero() {
		add("prepared_at es obligatorio")
	}
	if !hexSHA256.MatchString(t.InputsSHA256) {
		add("inputs_sha256 debe ser un SHA-256 en hexadecimal (es %q)", t.InputsSHA256)
	}

	repos := map[string]bool{}
	for i, repo := range t.Repositories {
		label := fmt.Sprintf("repositories[%d]", i)
		if problem := checkPlanPath(label+".repo", repo.Repo, true); problem != "" {
			add("%s", problem)
		} else if repos[repo.Repo] {
			add("%s.repo %q está repetido", label, repo.Repo)
		}
		repos[repo.Repo] = true
		if repo.Head != "" && !hexCommit.MatchString(repo.Head) {
			add("%s.head debe ser un commit en hexadecimal o estar vacío (es %q)", label, repo.Head)
		}
		if len(repo.Dirty) > MaxDirtyEntries {
			add("%s tiene %d entradas sucias, por encima del límite de %d", label, len(repo.Dirty), MaxDirtyEntries)
		}
		for j, dirty := range repo.Dirty {
			dirtyLabel := fmt.Sprintf("%s.dirty[%d]", label, j)
			if problem := checkPlanPath(dirtyLabel+".path", dirty.Path, false); problem != "" {
				add("%s", problem)
			}
			if dirty.SHA256 != "" && !hexSHA256.MatchString(dirty.SHA256) {
				add("%s.sha256 debe ser un SHA-256 en hexadecimal o estar vacío (es %q)", dirtyLabel, dirty.SHA256)
			}
		}
	}

	switch t.Context.Decision {
	case ContextEntries, ContextNoImpact, ContextNone:
	default:
		add("context.decision debe ser entries, no_impact o none (es %q)", t.Context.Decision)
	}

	for i, entry := range t.Entries {
		label := fmt.Sprintf("entries[%d]", i)
		if entry.ID == "" {
			add("%s.id es obligatorio", label)
		}
		if entry.Kind != EntryContext {
			add("%s.kind debe ser context (es %q)", label, entry.Kind)
		}
		switch entry.Action {
		case ActionAdded, ActionModified, ActionRemoved, ActionCreated:
		default:
			add("%s.action debe ser added, modified, removed o created (es %q)", label, entry.Action)
		}
		switch entry.State {
		case StatePending, StateIntegrated:
		default:
			add("%s.state debe ser pending o integrated (es %q)", label, entry.State)
		}
		if problem := checkPlanPath(label+".document", entry.Document, true); problem != "" {
			add("%s", problem)
		}
	}
	return problems
}

// checkPlanPath comprueba que una ruta del plan es relativa, usa `/` y, salvo que
// allowParent lo permita (repositorios y documentos de contexto, que pueden estar
// en un repositorio hermano), no sale de su base. El plan no guarda rutas de la
// máquina.
func checkPlanPath(label, value string, allowParent bool) string {
	switch {
	case value == "":
		return label + " es obligatorio"
	case strings.Contains(value, `\`):
		return fmt.Sprintf("%s %q debe usar '/' como separador", label, value)
	case strings.HasPrefix(value, "/") || driveLetter.MatchString(value):
		return fmt.Sprintf("%s %q debe ser una ruta relativa", label, value)
	}
	if clean := path.Clean(value); !allowParent && (clean == ".." || strings.HasPrefix(clean, "../")) {
		return fmt.Sprintf("%s %q no puede salir de su base", label, value)
	}
	return ""
}

// Formatos de `show`.
const (
	// TraceFormatMarkdown es la traza como tabla Markdown para el informe de archive.
	TraceFormatMarkdown = "markdown"
	// TraceFormatJSON es la traza como JSON.
	TraceFormatJSON = "json"
	// TraceSchema es el esquema del JSON de `show`.
	TraceSchema = "axiom.sdd-integration-trace/v1"
)

// TraceView es el JSON de `show --format json`: la traza sin la línea base de
// git, que no es parte de la auditoría. `idMap` está siempre vacío: la forma es la
// de `axiom sdd integrate show`.
type TraceView struct {
	Schema     string          `json:"schema"`
	Change     string          `json:"change"`
	PreparedAt time.Time       `json:"preparedAt"`
	VerifiedAt *time.Time      `json:"verifiedAt,omitempty"`
	Context    ContextDecision `json:"context"`
	IDMap      []any           `json:"idMap"`
	Entries    []TraceEntry    `json:"entries"`
	Warnings   []string        `json:"warnings"`
}

func traceView(trace Trace) TraceView {
	view := TraceView{
		Schema:     TraceSchema,
		Change:     trace.Change,
		PreparedAt: trace.PreparedAt.UTC(),
		Context:    trace.Context,
		IDMap:      []any{},
		Entries:    append([]TraceEntry{}, trace.Entries...),
		Warnings:   []string{},
	}
	if trace.VerifiedAt != nil {
		verified := trace.VerifiedAt.UTC()
		view.VerifiedAt = &verified
	}
	for _, entry := range trace.Entries {
		for _, warning := range entry.Warnings {
			view.Warnings = append(view.Warnings, entry.ID+": "+warning)
		}
	}
	return view
}

// RenderTrace da forma a la traza (REQ-24.52) como Markdown o JSON. El Markdown
// es la sección que se añade al informe de archive con `axiom archive context
// show --format markdown >> archive-report.md`: empieza por una línea en blanco
// que la separa del informe, lleva la tabla «Identificador | Documento | Ancla |
// Acción | Estado» y después la decisión «Sin impacto» con su motivo y los avisos.
func RenderTrace(trace Trace, format string) ([]byte, error) {
	view := traceView(trace)
	switch format {
	case TraceFormatJSON:
		data, err := json.MarshalIndent(view, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("serializar la traza: %w", err)
		}
		return append(data, '\n'), nil
	case TraceFormatMarkdown:
		return []byte(traceMarkdown(view)), nil
	default:
		return nil, fmt.Errorf("el formato %q de la traza no se reconoce: usa %q o %q", format, TraceFormatMarkdown, TraceFormatJSON)
	}
}

// markdownCell escapa un texto para una celda de tabla Markdown.
func markdownCell(text string) string {
	text = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(text)
	return strings.ReplaceAll(text, "|", `\|`)
}

func traceMarkdown(view TraceView) string {
	var b strings.Builder
	b.WriteString("\n## Traza de integración\n\n")
	fmt.Fprintf(&b, "Cambio: `%s`. Preparada: %s. Verificada: ", view.Change, view.PreparedAt.Format(time.RFC3339))
	if view.VerifiedAt != nil {
		b.WriteString(view.VerifiedAt.Format(time.RFC3339))
	} else {
		b.WriteString("pendiente")
	}
	b.WriteString(".\n\n")

	if len(view.Entries) == 0 {
		b.WriteString("La integración no registra entradas.\n")
	} else {
		b.WriteString("| Identificador | Documento | Ancla | Acción | Estado |\n|---|---|---|---|---|\n")
		for _, entry := range view.Entries {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", markdownCell(entry.ID), markdownCell(entry.Document), markdownCell(entry.Anchor), entry.Action, entry.State)
		}
	}

	if view.Context.Decision == ContextNoImpact {
		fmt.Fprintf(&b, "\n### Decisión sobre el contexto técnico\n\nSin impacto: %s\n", view.Context.Reason)
	}

	if len(view.Warnings) > 0 {
		b.WriteString("\n### Avisos\n\n")
		for _, warning := range view.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}
	return b.String()
}
