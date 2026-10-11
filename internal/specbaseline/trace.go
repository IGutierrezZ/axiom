package specbaseline

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

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
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

// MarshalPlan serializa el plan de integración (`integration.yaml`). La salida es
// determinista: la misma entrada produce los mismos bytes. No valida el plan:
// DecodePlan y WritePlan sí lo hacen.
func MarshalPlan(plan Plan) ([]byte, error) {
	if !plan.PreparedAt.IsZero() {
		plan.PreparedAt = plan.PreparedAt.UTC()
	}
	if plan.VerifiedAt != nil {
		verified := plan.VerifiedAt.UTC()
		plan.VerifiedAt = &verified
	}
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(plan); err != nil {
		return nil, fmt.Errorf("serializar %s: %w", PlanFileName, err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("serializar %s: %w", PlanFileName, err)
	}
	return buffer.Bytes(), nil
}

// DecodePlan lee `integration.yaml` de forma estricta: sin claves desconocidas,
// con un solo documento YAML, con el esquema `axiom.sdd-integration/v1` y con
// rutas relativas con `/` (R35). Reúne todos los defectos del contenido.
func DecodePlan(data []byte) (Plan, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return Plan{}, fmt.Errorf("%s está vacío", PlanFileName)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var plan Plan
	if err := decoder.Decode(&plan); err != nil {
		return Plan{}, fmt.Errorf("%s no se puede leer: %w", PlanFileName, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Plan{}, fmt.Errorf("%s debe contener un solo documento YAML", PlanFileName)
	}
	if problems := plan.problems(); len(problems) > 0 {
		return Plan{}, fmt.Errorf("%s no es válido: %s", PlanFileName, strings.Join(problems, "; "))
	}
	return plan, nil
}

// ReadPlan lee y valida el plan de la ruta. Si el fichero no existe, el error
// envuelve fs.ErrNotExist.
func ReadPlan(file string) (Plan, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return Plan{}, fmt.Errorf("leer %s: %w", file, err)
	}
	plan, err := DecodePlan(data)
	if err != nil {
		return Plan{}, fmt.Errorf("%s: %w", file, err)
	}
	return plan, nil
}

// WritePlan valida el plan y lo escribe con un fichero temporal en el mismo
// directorio y un renombrado, de modo que nunca queda un plan a medio escribir.
func WritePlan(file string, plan Plan) error {
	if problems := plan.problems(); len(problems) > 0 {
		return fmt.Errorf("%s no es válido: %s", PlanFileName, strings.Join(problems, "; "))
	}
	data, err := MarshalPlan(plan)
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

// problems valida el plan y devuelve todos sus defectos.
func (p Plan) problems() []string {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	if p.Schema != PlanSchema {
		add("schema debe ser %q (es %q)", PlanSchema, p.Schema)
	}
	if strings.TrimSpace(p.Change) == "" {
		add("change es obligatorio")
	}
	if p.PreparedAt.IsZero() {
		add("prepared_at es obligatorio")
	}
	if !hexSHA256.MatchString(p.InputsSHA256) {
		add("inputs_sha256 debe ser un SHA-256 en hexadecimal (es %q)", p.InputsSHA256)
	}

	repos := map[string]bool{}
	for i, repo := range p.Repositories {
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

	for i, assignment := range p.IDMap {
		if assignment.Provisional == "" || assignment.Definitive == "" || assignment.Domain == "" {
			add("id_map[%d] debe indicar provisional, definitive y domain", i)
		}
	}

	for i, document := range p.Documents {
		label := fmt.Sprintf("documents[%d]", i)
		if document.Domain == "" {
			add("%s.domain es obligatorio", label)
		}
		if problem := checkPlanPath(label+".path", document.Path, false); problem != "" {
			add("%s", problem)
		}
		switch document.Integration {
		case workspace.IntegrationCompose, workspace.IntegrationDirected, workspace.IntegrationRegister:
		default:
			add("%s.integration debe ser compose, directed o register (es %q)", label, document.Integration)
		}
		if !hexSHA256.MatchString(document.BeforeSHA256) {
			add("%s.before_sha256 debe ser un SHA-256 en hexadecimal (es %q)", label, document.BeforeSHA256)
		}
		if document.AfterSHA256 != "" && !hexSHA256.MatchString(document.AfterSHA256) {
			add("%s.after_sha256 debe ser un SHA-256 en hexadecimal o estar vacío (es %q)", label, document.AfterSHA256)
		}
		switch document.State {
		case "", StatePending, StateIntegrated:
		default:
			add("%s.state debe ser pending o integrated (es %q)", label, document.State)
		}
	}

	switch p.Context.Decision {
	case ContextEntries, ContextNoImpact, ContextNone:
	default:
		add("context.decision debe ser entries, no_impact o none (es %q)", p.Context.Decision)
	}

	for i, entry := range p.Entries {
		label := fmt.Sprintf("entries[%d]", i)
		if entry.ID == "" {
			add("%s.id es obligatorio", label)
		}
		switch entry.Kind {
		case EntryCompose, EntryDirected, EntryRegister, EntryContext:
		default:
			add("%s.kind debe ser compose, directed, register o context (es %q)", label, entry.Kind)
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

// checkPlanPath comprueba que una ruta del plan es relativa, usa `/` y, salvo
// que allowParent lo permita (repositorios y documentos de contexto, que pueden
// estar en un repositorio hermano), no sale de su base. El plan no guarda rutas
// de la máquina (R35).
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

// Formatos de `integrate show` (§4.4).
const (
	// TraceFormatMarkdown es la traza como tabla Markdown para el informe de archive.
	TraceFormatMarkdown = "markdown"
	// TraceFormatJSON es la traza como JSON.
	TraceFormatJSON = "json"
	// TraceSchema es el esquema del JSON de `integrate show`.
	TraceSchema = "axiom.sdd-integration-trace/v1"
)

// TraceView es el JSON de `integrate show --format json`: la traza del plan sin la
// línea base de git, que no es parte de la auditoría (REQ-24.52).
type TraceView struct {
	Schema     string         `json:"schema"`
	Change     string         `json:"change"`
	PreparedAt time.Time      `json:"preparedAt"`
	VerifiedAt *time.Time     `json:"verifiedAt,omitempty"`
	Context    TraceContext   `json:"context"`
	IDMap      []IDAssignment `json:"idMap"`
	Entries    []TraceEntry   `json:"entries"`
	Warnings   []string       `json:"warnings"`
}

// TraceContext es la decisión del cambio sobre el contexto técnico mantenido.
type TraceContext struct {
	Decision string `json:"decision"` // entries | no_impact | none
	Reason   string `json:"reason,omitempty"`
}

// traceView proyecta el plan a la vista de la traza. Los avisos son los de cada
// entrada, precedidos por su identificador.
func traceView(plan Plan) TraceView {
	view := TraceView{
		Schema:     TraceSchema,
		Change:     plan.Change,
		PreparedAt: plan.PreparedAt.UTC(),
		Context:    TraceContext{Decision: plan.Context.Decision, Reason: plan.Context.Reason},
		IDMap:      append([]IDAssignment{}, plan.IDMap...),
		Entries:    append([]TraceEntry{}, plan.Entries...),
		Warnings:   []string{},
	}
	if plan.VerifiedAt != nil {
		verified := plan.VerifiedAt.UTC()
		view.VerifiedAt = &verified
	}
	for _, entry := range plan.Entries {
		for _, warning := range entry.Warnings {
			view.Warnings = append(view.Warnings, entry.ID+": "+warning)
		}
	}
	return view
}

// RenderTrace da forma a la traza de integración (REQ-24.52, D-13 paso 6) como
// Markdown o JSON. El Markdown es la sección que se añade al informe de archive con
// `axiom sdd integrate show --format markdown >> archive-report.md`: empieza por una
// línea en blanco que la separa del informe, lleva la tabla «Identificador |
// Documento | Ancla | Acción | Estado» con el identificador de un ADDED con perfil
// como `RF-9001 → RF-008`, y después la decisión «Sin impacto» con su motivo, los
// escenarios omitidos y los avisos. El formato es `markdown` o `json`.
func RenderTrace(plan Plan, format string) ([]byte, error) {
	view := traceView(plan)
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
			identifier := entry.ID
			if entry.Provisional != "" && entry.Provisional != entry.ID {
				identifier = entry.Provisional + " → " + entry.ID
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", markdownCell(identifier), markdownCell(entry.Document), markdownCell(entry.Anchor), entry.Action, entry.State)
		}
	}

	if view.Context.Decision == ContextNoImpact {
		fmt.Fprintf(&b, "\n### Decisión sobre el contexto técnico\n\nSin impacto: %s\n", view.Context.Reason)
	}

	var omitted []string
	for _, entry := range view.Entries {
		if entry.ScenariosOmitted {
			label := entry.ID
			if entry.Domain != "" {
				label += " (dominio " + entry.Domain + ")"
			}
			omitted = append(omitted, label)
		}
	}
	if len(omitted) > 0 {
		b.WriteString("\n### Escenarios omitidos\n\n")
		for _, label := range omitted {
			fmt.Fprintf(&b, "- %s\n", label)
		}
	}

	if len(view.Warnings) > 0 {
		b.WriteString("\n### Avisos\n\n")
		for _, warning := range view.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}
	return b.String()
}
