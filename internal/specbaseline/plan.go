package specbaseline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

const (
	// PlanSchema es el esquema de `integration.yaml` (D-13, §4.5).
	PlanSchema = "axiom.sdd-integration/v1"
	// PlanFileName es el nombre del plan dentro de la carpeta del cambio.
	PlanFileName = "integration.yaml"
	// MaxDirtyEntries es el máximo de entradas sucias que el plan registra por
	// repositorio: acota el tamaño de integration.yaml (D-13).
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

// Plan es el contenido de integration.yaml (axiom.sdd-integration/v1): lo que
// `prepare` fija y lo que `verify` comprueba (D-13, D-15).
type Plan struct {
	Schema       string           `yaml:"schema"`
	Change       string           `yaml:"change"`
	PreparedAt   time.Time        `yaml:"prepared_at"`
	VerifiedAt   *time.Time       `yaml:"verified_at,omitempty"`
	InputsSHA256 string           `yaml:"inputs_sha256"`
	Options      PlanOptions      `yaml:"options"`
	Repositories []PlanRepository `yaml:"repositories"`
	IDMap        []IDAssignment   `yaml:"id_map"`
	Documents    []PlanDocument   `yaml:"documents"`
	Context      ContextDecision  `yaml:"context"`
	Entries      []TraceEntry     `yaml:"entries"`
}

// PlanOptions son las opciones de supersede que el plan fija para el cambio.
type PlanOptions struct {
	Supersede              bool     `yaml:"supersede"`
	SupersededRequirements []string `yaml:"superseded_requirements"`
}

// PlanRepository es la línea base de un repositorio afectado: su HEAD y la foto
// de su suciedad previa.
type PlanRepository struct {
	Repo  string       `yaml:"repo"`  // relativo a la carpeta maestra, con '/'
	Head  string       `yaml:"head"`  // commit en prepare; "" si no hay commits
	Dirty []DirtyEntry `yaml:"dirty"` // como máximo MaxDirtyEntries
}

// DirtyEntry es un fichero con cambios sin confirmar en prepare.
type DirtyEntry struct {
	Path   string `yaml:"path"`   // relativa al repositorio, con '/'
	SHA256 string `yaml:"sha256"` // huella del árbol de trabajo; "" si el fichero estaba borrado
}

// PlanDocument es un documento de un dominio del cambio, con las huellas de su
// árbol de trabajo.
type PlanDocument struct {
	Domain       string                    `yaml:"domain"`
	Path         string                    `yaml:"path"` // relativo a workspaceRoot, con '/'
	Integration  workspace.IntegrationMode `yaml:"integration"`
	BeforeSHA256 string                    `yaml:"before_sha256"`
	AfterSHA256  string                    `yaml:"after_sha256,omitempty"` // solo compose
	State        TraceState                `yaml:"state,omitempty"`        // solo compose
}

// ContextDecision es la decisión del cambio sobre el contexto técnico.
type ContextDecision struct {
	Decision string `yaml:"decision"` // entries | no_impact | none
	Reason   string `yaml:"reason,omitempty"`
}

// sha256Hex devuelve el SHA-256 de data en hexadecimal.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// workingTreeFingerprint es la huella de un fichero del árbol de trabajo: el
// SHA-256 de su contenido, vacía si no existe (borrado). Un enlace simbólico o un
// directorio, como un repositorio anidado sin seguimiento, se identifican por su
// destino o por su naturaleza. Se calcula sobre el árbol de trabajo y nunca
// contra los bytes de un commit, que `core.autocrlf` puede normalizar (D-12).
func workingTreeFingerprint(file string) (string, error) {
	info, err := os.Lstat(file)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", nil
	case err != nil:
		return "", err
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(file)
		if err != nil {
			return "", err
		}
		return sha256Hex([]byte("symlink:" + target)), nil
	case info.IsDir():
		return sha256Hex([]byte("dir")), nil
	}
	handle, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer handle.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, handle); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// DeltaInput es el delta de un dominio tal como lo lee `prepare`.
type DeltaInput struct {
	Domain string
	Data   []byte
}

// inputsDigest calcula `inputs_sha256`: la huella de los deltas, del delta de
// contexto y de la instantánea de la declaración (D-13). Es independiente del
// orden en que se reciben los deltas. Un delta de contexto nil (ausente) y uno
// vacío producen huellas distintas.
func inputsDigest(deltas []DeltaInput, contextDelta, declaration []byte) string {
	sorted := append([]DeltaInput(nil), deltas...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Domain < sorted[j].Domain })
	hash := sha256.New()
	frame := func(tag string, data []byte) {
		fmt.Fprintf(hash, "%s\x00%d\x00", tag, len(data))
		hash.Write(data)
	}
	for _, delta := range sorted {
		frame("delta-domain", []byte(delta.Domain))
		frame("delta", delta.Data)
	}
	if contextDelta == nil {
		frame("context-absent", nil)
	} else {
		frame("context", contextDelta)
	}
	frame("declaration", declaration)
	return hex.EncodeToString(hash.Sum(nil))
}

// declarationSnapshot es la instantánea determinista de las declaraciones de
// axiom.yaml que influyen en la integración: la baseline y el contexto técnico.
func declarationSnapshot(declarations workspace.Declarations) []byte {
	data, err := yaml.Marshal(struct {
		Baseline         *workspace.BaselineConfig         `yaml:"baseline"`
		TechnicalContext *workspace.TechnicalContextConfig `yaml:"technical_context"`
	}{declarations.Baseline, declarations.TechnicalContext})
	if err != nil {
		return []byte("declaración no serializable: " + err.Error())
	}
	return data
}

// AlreadyIntegratedError es el rechazo de preparar o aplicar de nuevo un cambio
// cuyas composiciones ya figuran como integradas (REQ-24.43, D-13). Nombra la
// correspondencia de IDs y propone verificar o consultar la traza.
type AlreadyIntegratedError struct {
	Change string
	IDs    []IDAssignment
}

func (e *AlreadyIntegratedError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "el cambio %q figura ya integrado: todas sus composiciones están marcadas como integradas en %s y no se vuelven a aplicar", e.Change, PlanFileName)
	if len(e.IDs) > 0 {
		pairs := make([]string, len(e.IDs))
		for i, id := range e.IDs {
			pairs[i] = fmt.Sprintf("%s -> %s (dominio %q)", id.Provisional, id.Definitive, id.Domain)
		}
		fmt.Fprintf(&b, "; correspondencia de IDs: %s", strings.Join(pairs, ", "))
	}
	b.WriteString(". Continuación: comprueba la integración con `axiom sdd integrate verify` o consulta la traza con `axiom sdd integrate show`")
	return b.String()
}

// PlanChangedError es el rechazo de reemitir un plan cuyas entradas ya no son las
// actuales (D-13, R35): los deltas, el delta de contexto, la declaración o un
// documento compuesto cambiaron desde `prepare`.
type PlanChangedError struct {
	Change    string
	Documents []string // documentos compuestos del plan, relativos a workspaceRoot
}

func (e *PlanChangedError) Error() string {
	documents := "<documentos>"
	if len(e.Documents) > 0 {
		documents = strings.Join(e.Documents, " ")
	}
	return fmt.Sprintf("el plan de integración del cambio %q (%s) ya no corresponde a las entradas actuales: cambiaron los deltas, el delta de contexto, la declaración o un documento de la baseline desde `prepare`. Continuación: devolver los documentos a su estado anterior con `git checkout -- %s`, retirar %s de la carpeta del cambio y volver a preparar con `axiom sdd integrate prepare`", e.Change, PlanFileName, documents, PlanFileName)
}

// reconcilePlan decide qué hace `prepare` cuando ya existe un plan (D-13). Si todas
// las composiciones del plan existente figuran como integradas, rechaza como «ya
// integrado», con precedencia sobre cualquier otro cambio. Si los deltas, el delta
// de contexto, la declaración o la huella `before` de un documento compuesto
// difieren de las actuales (fresh), rechaza con la continuación. En otro caso
// reemite el plan existente, con su `HEAD` y su foto previa: un commit posterior
// no lo invalida, y conservarlos mantiene la protección de `verify` (D-15).
func reconcilePlan(existing, fresh Plan) (Plan, error) {
	var composed []PlanDocument
	for _, document := range existing.Documents {
		if document.Integration == workspace.IntegrationCompose {
			composed = append(composed, document)
		}
	}
	integrated := len(composed) > 0
	for _, document := range composed {
		integrated = integrated && document.State == StateIntegrated
	}
	if integrated {
		return Plan{}, &AlreadyIntegratedError{Change: existing.Change, IDs: existing.IDMap}
	}

	changed := existing.InputsSHA256 != fresh.InputsSHA256
	current := map[string]PlanDocument{}
	for _, document := range fresh.Documents {
		current[document.Domain] = document
	}
	paths := make([]string, len(composed))
	for i, document := range composed {
		paths[i] = document.Path
		if now, ok := current[document.Domain]; !ok || now.BeforeSHA256 != document.BeforeSHA256 {
			changed = true
		}
	}
	if changed {
		return Plan{}, &PlanChangedError{Change: existing.Change, Documents: paths}
	}
	return existing, nil
}

// affectedPath es un documento que la integración edita. Creates indica que
// archive lo crea (REQ-24.55): queda excluido de la precondición de cambios sin
// confirmar (REQ-24.49).
type affectedPath struct {
	Path    string // absoluta
	Creates bool
}

// PreconditionError es la parada de REQ-24.49: un documento afectado no está en
// un repositorio git, o tiene cambios sin confirmar o sin seguimiento, o hay
// demasiada suciedad ajena. Cada problema nombra los ficheros y la continuación.
type PreconditionError struct {
	Problems []string
}

func (e *PreconditionError) Error() string { return strings.Join(e.Problems, "; ") }

// repositoryCapture es la línea base de los repositorios afectados y los avisos
// no bloqueantes de su captura.
type repositoryCapture struct {
	Repositories []PlanRepository
	Warnings     []string
}

const preparePreconditionContinuation = "confirma, guarda (`git stash -u`) o descarta los cambios y vuelve a ejecutar `axiom sdd integrate prepare`"

// repoWork reúne un repositorio con los documentos afectados que contiene.
type repoWork struct {
	repo  gitRepository
	files []affectedPath // con la ruta resuelta
}

// captureRepositories registra la línea base de cada repositorio afectado
// (REQ-24.49, D-15): su HEAD (vacío si no tiene commits) y la foto acotada de su
// suciedad previa, con la huella del árbol de trabajo de cada fichero. Los
// repositorios son los que contienen los documentos de affected, que además deben
// estar dentro de la carpeta maestra una vez resueltos los enlaces simbólicos, y
// los de also, que solo se vigilan: uno que no existe o no es git se avisa y nunca
// detiene. Devuelve *PreconditionError si un documento afectado está en un
// directorio que no es git, o con cambios en el índice, sin stage o sin
// seguimiento, o si un repositorio supera MaxDirtyEntries. Solo lee: nunca
// modifica el repositorio.
func captureRepositories(ctx context.Context, master string, affected []affectedPath, also []string) (repositoryCapture, error) {
	master = resolveDeepest(filepath.Clean(master))
	var works []*repoWork
	var problems, warnings []string

	workOf := func(repo gitRepository) *repoWork {
		for _, work := range works {
			if pathidentity.SameDirectory(work.repo.root, repo.root) {
				return work
			}
		}
		work := &repoWork{repo: repo}
		works = append(works, work)
		return work
	}

	for _, document := range affected {
		resolved := resolveDeepest(filepath.Clean(document.Path))
		if !pathidentity.Contains(master, resolved) {
			return repositoryCapture{}, fmt.Errorf("la ruta %s resuelve a %s, fuera de la carpeta maestra %s: un enlace simbólico no puede sacar la integración de ella", document.Path, resolved, master)
		}
		repo, err := openRepository(ctx, resolved)
		var notGit *NotGitRepositoryError
		if errors.As(err, &notGit) {
			problems = append(problems, fmt.Sprintf("el documento %s no está dentro de un repositorio git: la verificación por diff de la integración requiere un repositorio git; versiona su directorio con `git init` y confirma su estado actual, o declara el documento en un repositorio versionado", displayPath(master, resolved)))
			continue
		}
		if err != nil {
			return repositoryCapture{}, err
		}
		work := workOf(repo)
		work.files = append(work.files, affectedPath{Path: resolved, Creates: document.Creates})
	}

	for _, dir := range also {
		resolved := resolveDeepest(filepath.Clean(dir))
		if info, err := os.Stat(resolved); err != nil || !info.IsDir() {
			warnings = append(warnings, fmt.Sprintf("el repositorio %s que se vigila no existe; se omite", displayPath(master, resolved)))
			continue
		}
		repo, err := openRepository(ctx, resolved)
		var notGit *NotGitRepositoryError
		if errors.As(err, &notGit) {
			warnings = append(warnings, fmt.Sprintf("el directorio %s que se vigila no es un repositorio git; se omite", displayPath(master, resolved)))
			continue
		}
		if err != nil {
			return repositoryCapture{}, err
		}
		workOf(repo)
	}

	type captured struct {
		repository PlanRepository
		warnings   []string
	}
	var captures []captured
	for _, work := range works {
		name := repositoryName(master, work.repo.root)
		head, err := work.repo.head(ctx)
		if err != nil {
			return repositoryCapture{}, err
		}
		entries, err := work.repo.status(ctx)
		if err != nil {
			return repositoryCapture{}, err
		}
		if len(entries) > MaxDirtyEntries {
			problems = append(problems, fmt.Sprintf("el repositorio %s tiene %d entradas sin confirmar, por encima del límite de %d que el plan registra: %s", name, len(entries), MaxDirtyEntries, preparePreconditionContinuation))
			continue
		}
		if dirty := dirtyAffected(work, entries); len(dirty) > 0 {
			problems = append(problems, fmt.Sprintf("el repositorio %s tiene cambios sin confirmar en documentos que la integración modifica: %s; %s", name, strings.Join(dirty, ", "), preparePreconditionContinuation))
		}

		photo := make([]DirtyEntry, 0, len(entries))
		for _, entry := range entries {
			entryPath := strings.TrimSuffix(entry.Path, "/")
			fingerprint, err := workingTreeFingerprint(filepath.Join(work.repo.root, filepath.FromSlash(entryPath)))
			if err != nil {
				return repositoryCapture{}, fmt.Errorf("calcular la huella de %s en el repositorio %s: %w", entryPath, name, err)
			}
			photo = append(photo, DirtyEntry{Path: entryPath, SHA256: fingerprint})
		}
		state := work.repo.upstream(ctx)
		var notes []string
		if state.Behind > 0 {
			notes = append(notes, fmt.Sprintf("el repositorio %s va %s por detrás de su upstream (el dato procede de las referencias locales: Axiom nunca ejecuta fetch); si otros han integrado cambios, actualiza el repositorio antes de asignar IDs definitivos", name, pluralCommits(state.Behind)))
		}
		for _, note := range state.Notes {
			notes = append(notes, fmt.Sprintf("repositorio %s: %s", name, note))
		}
		captures = append(captures, captured{
			repository: PlanRepository{Repo: name, Head: head, Dirty: photo},
			warnings:   notes,
		})
	}

	if len(problems) > 0 {
		return repositoryCapture{}, &PreconditionError{Problems: problems}
	}
	sort.SliceStable(captures, func(i, j int) bool { return captures[i].repository.Repo < captures[j].repository.Repo })
	result := repositoryCapture{Warnings: warnings}
	for _, capture := range captures {
		result.Repositories = append(result.Repositories, capture.repository)
		result.Warnings = append(result.Warnings, capture.warnings...)
	}
	return result, nil
}

func pluralCommits(n int) string {
	if n == 1 {
		return "1 commit"
	}
	return fmt.Sprintf("%d commits", n)
}

// dirtyAffected devuelve, en orden y con su estado, los documentos afectados que
// tienen cualquier estado en `git status`: índice, árbol de trabajo o sin
// seguimiento. Los documentos que archive crea quedan excluidos.
func dirtyAffected(work *repoWork, entries []statusEntry) []string {
	var dirty []string
	for _, file := range work.files {
		if file.Creates {
			continue
		}
		relative, err := filepath.Rel(work.repo.root, file.Path)
		if err != nil {
			continue
		}
		relative = filepath.ToSlash(relative)
		for _, entry := range entries {
			if samePath(entry.Path, relative) {
				dirty = append(dirty, fmt.Sprintf("%s (%s)", relative, describeStatus(entry.Code)))
			}
		}
	}
	sort.Strings(dirty)
	return dirty
}

// samePath compara dos rutas relativas de git, sin distinguir mayúsculas donde el
// sistema de ficheros no las distingue.
func samePath(a, b string) bool {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// describeStatus describe en castellano un código XY de `git status`.
func describeStatus(code string) string {
	if code == "??" {
		return "sin seguimiento"
	}
	staged := code[0] != ' ' && code[0] != '?'
	worktree := code[1] != ' ' && code[1] != '?'
	switch {
	case staged && worktree:
		return "cambios en el índice y en el árbol de trabajo"
	case staged:
		return "cambios en el índice"
	default:
		return "cambios en el árbol de trabajo"
	}
}

// repositoryName es el nombre de un repositorio en el plan: su ruta relativa a la
// carpeta maestra, con `/`, y `.` si es la propia carpeta maestra.
func repositoryName(master, root string) string {
	relative, err := filepath.Rel(master, root)
	if err != nil {
		return filepath.ToSlash(root)
	}
	return filepath.ToSlash(relative)
}

// displayPath muestra una ruta relativa a la carpeta maestra si está dentro de ella.
func displayPath(master, p string) string {
	if relative, err := filepath.Rel(master, p); err == nil && !strings.HasPrefix(filepath.ToSlash(relative), "../") && relative != ".." {
		return filepath.ToSlash(relative)
	}
	return p
}

// Esquemas de los informes de `check` y de `prepare` (§4.4).
const (
	// CheckSchema es el esquema del informe de `integrate check`.
	CheckSchema = "axiom.sdd-integration-check/v1"
	// PrepareSchema es el esquema del informe de `integrate prepare`.
	PrepareSchema = "axiom.sdd-integration-plan/v1"
	// ContextDeltaFileName es el nombre de trabajo del delta de contexto (D-16).
	ContextDeltaFileName = "context-impact.md"

	// BaselineKindProject y BaselineKindDefault son los valores de Report.Baseline.
	BaselineKindProject = "project"
	BaselineKindDefault = "default"
)

// now es el reloj de `prepare`; las pruebas lo fijan.
var now = time.Now

// ChangeTarget es el cambio sobre el que actúan `check` y `prepare`: lo resuelve
// el adaptador de la CLI a partir de la ubicación del workspace y del status.
type ChangeTarget struct {
	Location      workspace.Location
	Change        string
	ChangeRoot    string   // carpeta del cambio, absoluta
	AllowedRoots  []string // actionContext.allowedEditRoots del status
	ArtifactStore string   // almacén efectivo que el adaptador toma del status (m8)
}

// OperationReport son las operaciones de un tipo (ADDED, MODIFIED, REMOVED o
// RENAMED) de la delta de un dominio, con los identificadores de sus requisitos.
type OperationReport struct {
	Section string   `json:"section"`
	IDs     []string `json:"ids"`
}

// BodyConflict es un cuerpo de delta con un encabezado que cerraría el bloque del
// requisito (REQ-24.42): el requisito, el encabezado y su línea en el delta.
type BodyConflict struct {
	Section     string `json:"section"`
	Requirement string `json:"requirement"`
	Heading     string `json:"heading"`
	Line        int    `json:"line"`
	Level       int    `json:"level"`
}

// FamilyReport describe una familia de IDs con requisitos ADDED en un documento:
// su máximo, el rango definitivo que recibirán y el inicio provisional sugerido
// para los IDs provisionales, que han de quedar fuera de ese rango (D-11).
type FamilyReport struct {
	Family          string `json:"family"`
	Max             string `json:"max"`
	FirstDefinitive string `json:"firstDefinitive"`
	LastDefinitive  string `json:"lastDefinitive"`
	SuggestedStart  string `json:"suggestedStart,omitempty"`
	Note            string `json:"note,omitempty"`
}

// DocumentReport es lo que `check` y `prepare` dicen de un documento de la baseline
// con delta en el cambio.
type DocumentReport struct {
	Domain      string                    `json:"domain"`
	Path        string                    `json:"path"` // relativa a workspaceRoot, con '/'
	Integration workspace.IntegrationMode `json:"integration"`
	Operations  []OperationReport         `json:"operations"`
	Conflicts   []BodyConflict            `json:"conflicts,omitempty"`
	Families    []FamilyReport            `json:"families,omitempty"`
	Warnings    []string                  `json:"warnings,omitempty"` // colas de secciones contiguas (REQ-24.41)
}

// ContextEntryReport es una entrada del delta de contexto tal como la ve `check`.
type ContextEntryReport struct {
	ID       string `json:"id"` // C1..Cn
	Document string `json:"document"`
	Anchor   string `json:"anchor"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
	Create   bool   `json:"create,omitempty"` // la entrada `add` crea el documento
}

// ContextReport es el estado del delta de contexto: si el contexto mantenido está
// declarado, si el fichero existe, la decisión y sus entradas.
type ContextReport struct {
	Declared bool                 `json:"declared"`
	Present  bool                 `json:"present"`
	Decision string               `json:"decision"` // entries | no_impact | none
	Reason   string               `json:"reason,omitempty"`
	Entries  []ContextEntryReport `json:"entries,omitempty"`
}

// DirectedEntry es lo que hay que integrar de un requisito de la delta de un
// dominio de prosa, con los IDs provisionales ya sustituidos.
type DirectedEntry struct {
	Section string `json:"section"` // ADDED, MODIFIED, REMOVED o RENAMED
	Name    string `json:"name"`
	Text    string `json:"text"`
}

// DirectedItem es el trabajo de edición dirigida de un documento de prosa: la
// sección titulada con el identificador del cambio y sus entradas (REQ-24.50).
type DirectedItem struct {
	Domain   string          `json:"domain"`
	Document string          `json:"document"`
	Section  string          `json:"section"` // título esperado de la sección: el identificador del cambio
	Entries  []DirectedEntry `json:"entries"`
}

// ContextItem es el trabajo de edición de una entrada del delta de contexto, con
// los IDs provisionales ya sustituidos en el motivo y el texto (REQ-24.59).
type ContextItem struct {
	ID       string `json:"id"`
	Document string `json:"document"`
	Anchor   string `json:"anchor"`
	Action   string `json:"action"`
	Reason   string `json:"reason"`
	Text     string `json:"text,omitempty"`
	Create   bool   `json:"create,omitempty"`
}

// Report es la salida de `check` (axiom.sdd-integration-check/v1) y de `prepare`
// (axiom.sdd-integration-plan/v1). Los errores impiden integrar; con errores quien
// llama sale con un código distinto de cero.
type Report struct {
	Schema         string           `json:"schema"`
	Change         string           `json:"change"`
	Baseline       string           `json:"baseline"` // project | default
	Documents      []DocumentReport `json:"documents"`
	IDMap          []IDAssignment   `json:"idMap"`
	Context        ContextReport    `json:"context"`
	ComposeCommand string           `json:"composeCommand,omitempty"`
	Directed       []DirectedItem   `json:"directed,omitempty"`
	ContextItems   []ContextItem    `json:"contextItems,omitempty"`
	Register       []TraceEntry     `json:"register,omitempty"`
	Warnings       []string         `json:"warnings"`
	Errors         []string         `json:"errors"`
}

func newReport(schema, change string) Report {
	return Report{
		Schema:    schema,
		Change:    change,
		Baseline:  BaselineKindDefault,
		Documents: []DocumentReport{},
		IDMap:     []IDAssignment{},
		Context:   ContextReport{Decision: ContextNone},
		Warnings:  []string{},
		Errors:    []string{},
	}
}

func (r *Report) fail(format string, args ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

func (r *Report) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// gatheredDocument es un documento de la baseline con delta en el cambio.
type gatheredDocument struct {
	declared workspace.BaselineDocument
	mode     workspace.IntegrationMode
	relPath  string
	abs      string
	original []byte
	delta    []byte
	profile  Profile // solo con integración compose
}

// gathered es lo que `check` y `prepare` leen del disco antes de evaluar nada.
type gathered struct {
	project        bool
	maintained     bool
	documents      []gatheredDocument // dominios con delta, en el orden de baseline.documents
	contextData    []byte             // nil si no hay delta de contexto
	contextPresent bool
}

// gather lee la declaración, los deltas, los documentos de los dominios con delta y
// el delta de contexto (REQ-24.18, REQ-24.62, REQ-24.48). Devuelve false si el
// resto de comprobaciones no tiene sentido: declaración ilegible o inválida, o un
// almacén sin ficheros.
func gather(t ChangeTarget, report *Report) (*gathered, bool) {
	loc := t.Location
	declarations := loc.Declarations
	g := &gathered{}
	switch {
	case declarations.ParseErr != nil:
		report.fail("%s no se puede leer como YAML: %v. Continuación: corrige el YAML y vuelve a ejecutar el comando (`axiom workspace validate --path %s` comprueba la declaración)", pathquote.Quote(loc.ConfigPath), declarations.ParseErr, pathquote.Quote(loc.Master))
		return g, false
	case declarations.BaselineErr != nil || declarations.TechnicalContextErr != nil:
		for _, err := range []error{declarations.BaselineErr, declarations.TechnicalContextErr} {
			if err != nil {
				report.fail("la declaración de %s no es válida: %v. Continuación: corrígela con `axiom workspace validate --path %s` y vuelve a ejecutar el comando", pathquote.Quote(loc.ConfigPath), err, pathquote.Quote(loc.Master))
			}
		}
		return g, false
	}

	g.project = declarations.Baseline != nil
	g.maintained = declarations.TechnicalContext != nil && declarations.TechnicalContext.Maintained != nil
	if g.project {
		report.Baseline = BaselineKindProject
	}
	report.Context.Declared = g.maintained

	if (g.project || g.maintained) && (t.ArtifactStore == "engram" || t.ArtifactStore == "none") {
		var declared []string
		if g.project {
			declared = append(declared, "la baseline del proyecto ('baseline')")
		}
		if g.maintained {
			declared = append(declared, "el contexto técnico mantenido ('technical_context.maintained')")
		}
		report.fail("%s exige un almacén de artefactos de ficheros y el almacén de este cambio es %q: sin la carpeta del cambio en ficheros no hay delta de contexto ni traza que integrar (REQ-24.62). Continuación: declara `artifact_store: openspec` o `hybrid` en `openspec/config.yaml`, o retira la declaración", strings.Join(declared, " y "), t.ArtifactStore)
		return g, false
	}

	if g.project {
		g.gatherDocuments(t, report)
	}
	if g.maintained {
		data, err := os.ReadFile(filepath.Join(t.ChangeRoot, ContextDeltaFileName))
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			report.fail("no se puede leer %s: %v", ContextDeltaFileName, err)
		default:
			g.contextData, g.contextPresent = data, true
		}
	}
	report.Context.Present = g.contextPresent
	return g, true
}

// gatherDocuments lee los deltas de <cambio>/specs/<dominio>/spec.md y los
// documentos declarados de los dominios que tienen delta. Un dominio no
// declarado se rechaza antes de tocar ningún documento (REQ-24.48).
func (g *gathered) gatherDocuments(t ChangeTarget, report *Report) {
	loc := t.Location
	deltas := map[string][]byte{}
	var domains []string
	entries, err := os.ReadDir(filepath.Join(t.ChangeRoot, "specs"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		report.fail("no se pueden leer los deltas del cambio en %s: %v", filepath.Join(t.ChangeRoot, "specs"), err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(t.ChangeRoot, "specs", entry.Name(), "spec.md"))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			report.fail("no se puede leer el delta del dominio %q: %v", entry.Name(), err)
			continue
		}
		deltas[entry.Name()] = data
		domains = append(domains, entry.Name())
	}

	declared := map[string]bool{}
	for _, document := range loc.Declarations.Baseline.Documents {
		declared[document.Domain] = true
	}
	for _, domain := range domains {
		if !declared[domain] {
			report.fail("el dominio %q del cambio no figura en 'baseline.documents' de %s: no se integra ni se promueve, y nunca se crea %s. Continuación: declara el dominio en 'baseline.documents' de axiom.yaml (`axiom workspace validate --path %s` comprueba la declaración) o retira la delta del cambio",
				domain, pathquote.Quote(loc.ConfigPath), pathquote.Quote(filepath.Join(loc.PlanningHome, "specs", domain)), pathquote.Quote(loc.Master))
		}
	}

	for _, document := range loc.Declarations.Baseline.Documents {
		delta, ok := deltas[document.Domain]
		if !ok {
			continue
		}
		item := gatheredDocument{
			declared: document,
			mode:     document.EffectiveIntegration(),
			relPath:  path.Clean(strings.ReplaceAll(document.Path, `\`, "/")),
			delta:    delta,
		}
		item.abs = filepath.Join(loc.WorkspaceRoot, filepath.FromSlash(item.relPath))
		original, err := os.ReadFile(item.abs)
		if err != nil {
			report.fail("no se puede leer el documento %s del dominio %q: %v. Continuación: comprueba la declaración con `axiom workspace validate --path %s`", item.relPath, document.Domain, err, pathquote.Quote(loc.Master))
			continue
		}
		item.original = original
		if item.mode == workspace.IntegrationCompose {
			profile, err := CompileProfile(document)
			if err != nil {
				report.fail("%v", err)
				continue
			}
			item.profile = profile
		}
		g.documents = append(g.documents, item)
	}
}

// composeDocuments son los documentos con perfil de requisito, en el orden de
// `baseline.documents`, en la forma que consumen PlanIDs y ComposeDocument.
func (g *gathered) composeDocuments() []PlannedDocument {
	var planned []PlannedDocument
	for _, document := range g.documents {
		if document.mode == workspace.IntegrationCompose {
			planned = append(planned, PlannedDocument{
				Domain:   document.declared.Domain,
				Path:     document.abs,
				RelPath:  document.relPath,
				Profile:  document.profile,
				Original: document.original,
				Delta:    document.delta,
			})
		}
	}
	return planned
}

// evaluation es el resultado de validar y componer en memoria todo el cambio
// antes de escribir nada (REQ-24.46).
type evaluation struct {
	ids          IDMap
	outputs      map[string][]byte // documento compuesto por dominio
	planDocs     []PlanDocument
	entries      []TraceEntry
	directed     []DirectedItem
	contextItems []ContextItem
	decision     ContextDecision
	creates      map[string]bool // documentos de contexto que archive crea, por ruta relativa
}

// evaluate valida los deltas, calcula la correspondencia de IDs, compone en
// memoria cada documento con perfil, prepara las entradas dirigidas, de solo
// registro y de contexto, y valida el delta de contexto. No escribe nada. Reúne
// todos los errores que puede en el informe.
func evaluate(t ChangeTarget, g *gathered, opts PlanOptions, report *Report) evaluation {
	ev := evaluation{outputs: map[string][]byte{}, creates: map[string]bool{}, decision: ContextDecision{Decision: ContextNone}}

	planned := g.composeDocuments()
	ids, idsErr := PlanIDs(planned)
	if idsErr != nil {
		report.fail("%v", idsErr)
	}
	ev.ids = ids
	report.IDMap = append([]IDAssignment{}, ids.Assignments()...)

	for _, document := range g.documents {
		item := DocumentReport{
			Domain:      document.declared.Domain,
			Path:        document.relPath,
			Integration: document.mode,
			Operations:  []OperationReport{},
		}
		parsed := parseDelta(document.delta)
		item.Operations = operationReports(document, parsed)
		planDocument := PlanDocument{
			Domain:       document.declared.Domain,
			Path:         document.relPath,
			Integration:  document.mode,
			BeforeSHA256: sha256Hex(document.original),
		}
		if parsed.empty() {
			report.fail("dominio %q: el delta no declara requisitos ADDED, MODIFIED, REMOVED ni RENAMED. Continuación: escribe el delta en el formato nativo, con una sección '## ADDED Requirements', '## MODIFIED Requirements', '## REMOVED Requirements' o '## RENAMED Requirements' y un bloque '### Requirement: <nombre>' por requisito", document.declared.Domain)
		}

		switch document.mode {
		case workspace.IntegrationCompose:
			item.Conflicts = bodyConflicts(document, parsed)
			for _, conflict := range item.Conflicts {
				report.fail("dominio %q, sección %s, requisito %q: el cuerpo contiene el encabezado %q (línea %d del delta, nivel %d), de nivel menor o igual que el nivel %d de los requisitos: cerraría el bloque del requisito. Continuación: rebaja el encabezado a un nivel mayor que %d o conviértelo en texto",
					document.declared.Domain, conflict.Section, conflict.Requirement, conflict.Heading, conflict.Line, conflict.Level, document.profile.Level, document.profile.Level)
			}
			if idsErr == nil {
				item.Families = planFamilies(planned[composeIndex(planned, document.declared.Domain)], parsed)
			}
			if idsErr == nil && len(item.Conflicts) == 0 && !parsed.empty() {
				result, err := ComposeDocument(ComposeInput{
					Document:           planned[composeIndex(planned, document.declared.Domain)],
					IDs:                ids,
					Supersede:          opts.Supersede,
					ExplicitSuperseded: opts.SupersededRequirements,
				})
				if err != nil {
					report.fail("%v", err)
				} else {
					ev.outputs[document.declared.Domain] = result.Output
					planDocument.AfterSHA256 = sha256Hex(result.Output)
					planDocument.State = StatePending
					ev.entries = append(ev.entries, result.Entries...)
					item.Warnings = result.Warnings
					for _, warning := range result.Warnings {
						report.warn("%s", warning)
					}
				}
			}
		default:
			ev.entries = append(ev.entries, directedEntries(document, parsed, ids, document.mode, t.Change)...)
			if document.mode == workspace.IntegrationDirected {
				ev.directed = append(ev.directed, directedItem(document, parsed, ids, t.Change))
			}
		}
		report.Documents = append(report.Documents, item)
		ev.planDocs = append(ev.planDocs, planDocument)
	}
	ev.evaluateContext(t, g, report)
	return ev
}

// composeIndex devuelve la posición del dominio entre los documentos con perfil.
func composeIndex(planned []PlannedDocument, domain string) int {
	for i, document := range planned {
		if document.Domain == domain {
			return i
		}
	}
	return -1
}

var deltaSections = []string{sectionAdded, sectionModified, sectionRemoved, sectionRenamed}

func deltaBlocks(parsed delta, section string) []deltaBlock {
	switch section {
	case sectionAdded:
		return parsed.Added
	case sectionModified:
		return parsed.Modified
	case sectionRemoved:
		return parsed.Removed
	default:
		return parsed.Renamed
	}
}

// operationReports lista las operaciones del delta por sección, con el identificador
// de cada requisito: su ID con identidad `id` o su nombre en otro caso.
func operationReports(document gatheredDocument, parsed delta) []OperationReport {
	var reports []OperationReport
	for _, section := range deltaSections {
		blocks := deltaBlocks(parsed, section)
		if len(blocks) == 0 {
			continue
		}
		ids := make([]string, len(blocks))
		for i, block := range blocks {
			ids[i] = block.Name
			if document.mode == workspace.IntegrationCompose && section != sectionRenamed {
				if id, _, key, err := parseRequirementName(document.declared.Domain, document.profile, section, block.Name); err == nil {
					ids[i] = key
					if document.profile.Identity == workspace.IdentityID {
						ids[i] = id
					}
				}
			}
		}
		reports = append(reports, OperationReport{Section: section, IDs: ids})
	}
	return reports
}

// bodyConflicts devuelve, para cada requisito ADDED o MODIFIED, los encabezados del
// cuerpo que se integraría que cerrarían el bloque (REQ-24.42, D-09). Con
// `scenarios: omit` se evalúa el cuerpo ya sin sus escenarios.
func bodyConflicts(document gatheredDocument, parsed delta) []BodyConflict {
	var conflicts []BodyConflict
	for _, section := range []string{sectionAdded, sectionModified} {
		for _, block := range deltaBlocks(parsed, section) {
			body := block.Body
			if document.profile.Scenarios == workspace.ScenariosOmit {
				body, _ = omitScenarios(body)
			}
			for _, violation := range bodyViolations(body, document.profile.Level) {
				conflicts = append(conflicts, BodyConflict{Section: section, Requirement: block.Name, Heading: violation.Text, Line: violation.Number, Level: violation.Level})
			}
		}
	}
	return conflicts
}

// planFamilies describe las familias de IDs con requisitos ADDED de un documento
// con identidad `id`: el máximo del documento original, el rango definitivo y el
// inicio provisional sugerido (D-11). El inicio es `9·10^d + 1` con
// `d = max(3, dígitos(máximo + K))`, siendo K el número de ADDED de la familia, y
// se prueba también `9·10^(ancho-1)` por si el patrón fija el ancho; se sugiere el
// primero que cumple el patrón `heading` y queda fuera del rango definitivo.
func planFamilies(document PlannedDocument, parsed delta) []FamilyReport {
	if document.Profile.Identity != workspace.IdentityID || len(parsed.Added) == 0 {
		return nil
	}
	scanned := scanDocIDs(document)
	counts := map[string]int64{}
	var order []string
	for _, block := range parsed.Added {
		id, _, _, err := parseRequirementName(document.Domain, document.Profile, sectionAdded, block.Name)
		if err != nil {
			continue
		}
		family, digits := splitID(id)
		if digits == "" || scanned.families[family] == nil {
			continue
		}
		if counts[family] == 0 {
			order = append(order, family)
		}
		counts[family]++
	}

	var reports []FamilyReport
	for _, family := range order {
		stats := scanned.families[family]
		count := big.NewInt(counts[family])
		first := new(big.Int).Add(stats.max, big.NewInt(1))
		last := new(big.Int).Add(stats.max, count)
		report := FamilyReport{
			Family:          family,
			Max:             family + padNumber(stats.max, stats.width),
			FirstDefinitive: family + padNumber(first, stats.width),
			LastDefinitive:  family + padNumber(last, stats.width),
		}

		d := len(last.String())
		if d < 3 {
			d = 3
		}
		first9 := new(big.Int).Mul(big.NewInt(9), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d)), nil))
		candidates := []*big.Int{first9.Add(first9, big.NewInt(1))}
		if stats.width > 1 {
			candidates = append(candidates, new(big.Int).Mul(big.NewInt(9), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(stats.width-1)), nil)))
		}
		var tried []string
		for _, candidate := range candidates {
			id := family + candidate.String()
			tried = append(tried, id)
			if candidate.Cmp(last) <= 0 {
				continue
			}
			if _, exists := scanned.existing[id]; exists {
				continue
			}
			if matched, _, ok := document.Profile.Match(document.Profile.RenderHeading(id, "Título provisional")); ok && matched == id {
				report.SuggestedStart = id
				break
			}
		}
		if report.SuggestedStart == "" {
			report.Note = fmt.Sprintf("ningún candidato (%s) cumple el patrón 'heading' del perfil y queda fuera del rango definitivo %s..%s: elige un ID provisional que cumpla el patrón", strings.Join(tried, ", "), report.FirstDefinitive, report.LastDefinitive)
		}
		reports = append(reports, report)
	}
	return reports
}

func padNumber(n *big.Int, width int) string {
	text := n.String()
	if pad := width - len(text); pad > 0 {
		text = strings.Repeat("0", pad) + text
	}
	return text
}

// directedEntries son las entradas de la traza de un dominio de prosa: una por
// requisito de la delta, anclada a la sección titulada con el identificador del
// cambio. Las de un dominio `register` quedan pendientes y no editan el documento
// (REQ-24.50, REQ-24.53).
func directedEntries(document gatheredDocument, parsed delta, ids IDMap, mode workspace.IntegrationMode, change string) []TraceEntry {
	kind := EntryDirected
	if mode == workspace.IntegrationRegister {
		kind = EntryRegister
	}
	var entries []TraceEntry
	for _, section := range deltaSections {
		for _, block := range deltaBlocks(parsed, section) {
			entries = append(entries, TraceEntry{
				ID:       ids.Substitute(block.Name),
				Kind:     kind,
				Domain:   document.declared.Domain,
				Document: document.relPath,
				Anchor:   change,
				Action:   sectionAction(section),
				State:    StatePending,
				Reason:   ids.Substitute(block.Reason),
			})
		}
	}
	return entries
}

func sectionAction(section string) TraceAction {
	switch section {
	case sectionAdded:
		return ActionAdded
	case sectionRemoved:
		return ActionRemoved
	default:
		return ActionModified
	}
}

// directedItem es el trabajo de edición dirigida de un documento de prosa, con los
// IDs provisionales del cambio ya sustituidos por los definitivos.
func directedItem(document gatheredDocument, parsed delta, ids IDMap, change string) DirectedItem {
	item := DirectedItem{Domain: document.declared.Domain, Document: document.relPath, Section: change, Entries: []DirectedEntry{}}
	for _, section := range deltaSections {
		for _, block := range deltaBlocks(parsed, section) {
			body := block.Body
			for len(body) > 0 && strings.TrimSpace(body[0].Text) == "" {
				body = body[1:]
			}
			lines := make([]string, 0, len(body)+1)
			for _, line := range body {
				lines = append(lines, ids.Substitute(line.Text))
			}
			if block.HasReason {
				lines = append(lines, "(Reason: "+ids.Substitute(block.Reason)+")")
			}
			item.Entries = append(item.Entries, DirectedEntry{Section: section, Name: ids.Substitute(block.Name), Text: strings.Join(lines, "\n")})
		}
	}
	return item
}

// evaluateContext valida el delta de contexto (REQ-24.31, REQ-24.32, REQ-24.30):
// su formato, que cada documento esté bajo una ruta mantenida y no bajo una de
// referencia, y que cada ancla resuelva según su acción. Prepara los trabajos de
// edición con los IDs provisionales sustituidos.
func (ev *evaluation) evaluateContext(t ChangeTarget, g *gathered, report *Report) {
	if !g.maintained || !g.contextPresent {
		return
	}
	delta, err := ParseContextDelta(g.contextData)
	if err != nil {
		report.fail("%s (REQ-24.31): %v. Continuación: corrige el formato del delta de contexto (`## Entrada: <ruta>` con `- Ancla:`, `- Acción:` y `- Motivo:`, o `Sin impacto: <motivo>`)", ContextDeltaFileName, err)
	}
	report.Context.Reason = delta.Reason
	switch {
	case err != nil:
		return
	case delta.NoImpact:
		report.Context.Decision = ContextNoImpact
		ev.decision = ContextDecision{Decision: ContextNoImpact, Reason: delta.Reason}
		return
	}
	report.Context.Decision = ContextEntries
	ev.decision = ContextDecision{Decision: ContextEntries}

	scope := contextScope(t.Location)
	for _, entry := range delta.Entries {
		report.Context.Entries = append(report.Context.Entries, ContextEntryReport{ID: entry.ID, Document: entry.Document, Anchor: entry.Anchor.String(), Action: string(entry.Action), Reason: entry.Reason})
		if err := scope.CheckDocument(entry); err != nil {
			report.fail("%v", err)
			continue
		}
		data, readErr := os.ReadFile(scope.Resolve(entry.Document))
		exists := readErr == nil
		if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
			report.fail("la entrada %s (línea %d): no se puede leer %s: %v", entry.ID, entry.Line, entry.Document, readErr)
			continue
		}
		create, err := CheckAnchor(entry, data, exists)
		if err != nil {
			report.fail("%v", err)
			continue
		}
		if create {
			ev.creates[entry.Document] = true
			report.Context.Entries[len(report.Context.Entries)-1].Create = true
		}
		ev.contextItems = append(ev.contextItems, ContextItem{
			ID:       entry.ID,
			Document: entry.Document,
			Anchor:   entry.Anchor.String(),
			Action:   string(entry.Action),
			Reason:   ev.ids.Substitute(entry.Reason),
			Text:     ev.ids.Substitute(entry.Body),
			Create:   create,
		})
	}
}

// contextScope reúne las rutas declaradas que deciden si una entrada del delta de
// contexto puede editar un documento.
func contextScope(loc workspace.Location) ContextScope {
	scope := ContextScope{WorkspaceRoot: loc.WorkspaceRoot}
	normalize := func(paths []string) []string {
		out := make([]string, 0, len(paths))
		for _, p := range paths {
			out = append(out, strings.ReplaceAll(strings.TrimSpace(p), `\`, "/"))
		}
		return out
	}
	if technical := loc.Declarations.TechnicalContext; technical != nil {
		if technical.Maintained != nil {
			scope.Maintained = normalize(technical.Maintained.Paths)
		}
		if technical.Reference != nil {
			scope.Reference = normalize(technical.Reference.Paths)
		}
	}
	if baseline := loc.Declarations.Baseline; baseline != nil {
		for _, document := range baseline.Documents {
			scope.BaselineDocuments = append(scope.BaselineDocuments, path.Clean(strings.ReplaceAll(document.Path, `\`, "/")))
		}
	}
	return scope
}

// composeCommand es el comando literal que imprime `prepare` para componer todos
// los documentos con perfil del cambio (D-13, paso 2).
func composeCommand(t ChangeTarget, planDocs []PlanDocument) string {
	var domains []string
	for _, document := range planDocs {
		if document.Integration == workspace.IntegrationCompose {
			domains = append(domains, "--document "+pathquote.ShellWord(document.Domain))
		}
	}
	if len(domains) == 0 {
		return ""
	}
	return fmt.Sprintf("axiom sdd archive-compose --cwd %s --change %s %s --in-place", pathquote.Quote(t.Location.WorkspaceRoot), pathquote.ShellWord(t.Change), strings.Join(domains, " "))
}

// Check hace las validaciones de la integración sin exigir git ni autorización de
// edición, y no escribe nada, tampoco en la carpeta del cambio (REQ-24.42, D-13).
// Un delta de contexto ausente es un aviso. Quien llama sale con un código distinto
// de cero si Report.Errors no está vacío; el error devuelto es solo de ejecución.
func Check(ctx context.Context, t ChangeTarget) (Report, error) {
	report := newReport(CheckSchema, t.Change)
	g, ok := gather(t, &report)
	if !ok {
		return report, ctx.Err()
	}
	evaluate(t, g, PlanOptions{}, &report)
	if g.maintained && !g.contextPresent {
		report.warn("el cambio no lleva delta de contexto %s y el contexto técnico mantenido está declarado: registra el impacto previsto o «Sin impacto: <motivo>» antes de archivar; la ausencia nunca se asume como «Sin impacto» (REQ-24.36)", ContextDeltaFileName)
	}
	return report, ctx.Err()
}

// Prepare valida y compone en memoria todo el cambio, comprueba las precondiciones
// de la integración (REQ-24.49) y la autorización de edición de las rutas
// mantenidas fuera del repositorio de especificaciones (REQ-24.37), registra el
// HEAD y la foto de suciedad de cada repositorio afectado y escribe
// `<carpeta del cambio>/integration.yaml`: nunca escribe en la baseline ni en las
// rutas mantenidas (D-13, paso 1). Con un plan existente lo reemite si las entradas
// y las huellas coinciden, rechaza como «ya integrado» si todas las composiciones
// lo están, y rechaza con la continuación si las entradas cambiaron. Los
// rechazos de validación van en Report.Errors, sin plan; el error devuelto es solo
// de ejecución.
func Prepare(ctx context.Context, t ChangeTarget, opts PlanOptions) (Plan, Report, error) {
	report := newReport(PrepareSchema, t.Change)
	opts.SupersededRequirements = cleanList(opts.SupersededRequirements)
	g, ok := gather(t, &report)
	if !ok || len(report.Errors) > 0 {
		return Plan{}, report, ctx.Err()
	}

	digest := inputsDigest(g.deltaInputs(), g.contextData, declarationSnapshot(t.Location.Declarations))
	planPath := filepath.Join(t.ChangeRoot, PlanFileName)
	existing, readErr := ReadPlan(planPath)
	reuse := readErr == nil
	if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
		report.fail("no se puede leer el plan existente: %v. Continuación: retira %s de la carpeta del cambio y vuelve a preparar con `axiom sdd integrate prepare`", readErr, PlanFileName)
		return Plan{}, report, ctx.Err()
	}
	if reuse {
		current := Plan{InputsSHA256: digest}
		for _, document := range g.documents {
			current.Documents = append(current.Documents, PlanDocument{Domain: document.declared.Domain, Integration: document.mode, BeforeSHA256: sha256Hex(document.original)})
		}
		kept, err := reconcilePlan(existing, current)
		if err != nil {
			report.fail("%v", err)
			return Plan{}, report, ctx.Err()
		}
		existing = kept
	}

	ev := evaluate(t, g, opts, &report)
	if g.maintained && !g.contextPresent {
		report.fail("el cambio no lleva delta de contexto %s y el contexto técnico mantenido está declarado: archive se detiene hasta que se registre el impacto previsto o «Sin impacto: <motivo>»; la ausencia nunca se asume como «Sin impacto» (REQ-24.36, REQ-24.56). Continuación: redacta %s en la carpeta del cambio", ContextDeltaFileName, ContextDeltaFileName)
	}
	if len(report.Errors) > 0 {
		return Plan{}, report, ctx.Err()
	}
	report.Directed, report.ContextItems, report.ComposeCommand = ev.directed, ev.contextItems, composeCommand(t, ev.planDocs)
	report.Register = ev.registered()
	if reuse {
		return existing, report, ctx.Err()
	}

	authorizeContextEdits(ctx, t, ev, &report)
	if len(report.Errors) > 0 {
		return Plan{}, report, ctx.Err()
	}

	capture, err := captureRepositories(ctx, t.Location.Master, ev.affected(g, t.Location), watchedDirectories(t.Location))
	var stop *PreconditionError
	if errors.As(err, &stop) {
		for _, problem := range stop.Problems {
			report.fail("%s", problem)
		}
		return Plan{}, report, ctx.Err()
	}
	if err != nil {
		report.fail("%v", err)
		return Plan{}, report, ctx.Err()
	}
	for _, warning := range capture.Warnings {
		report.warn("%s", warning)
	}

	plan := Plan{
		Schema:       PlanSchema,
		Change:       t.Change,
		PreparedAt:   now().UTC().Truncate(time.Second),
		InputsSHA256: digest,
		Options:      PlanOptions{Supersede: opts.Supersede, SupersededRequirements: opts.SupersededRequirements},
		Repositories: capture.Repositories,
		IDMap:        report.IDMap,
		Documents:    ev.planDocs,
		Context:      ev.decision,
		Entries:      append(append([]TraceEntry(nil), ev.entries...), ev.contextEntries()...),
	}
	if err := WritePlan(planPath, plan); err != nil {
		return Plan{}, report, err
	}
	return plan, report, ctx.Err()
}

func cleanList(items []string) []string {
	cleaned := []string{}
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

// deltaInputs son los deltas de los dominios del cambio, para la huella de entradas.
func (g *gathered) deltaInputs() []DeltaInput {
	inputs := make([]DeltaInput, 0, len(g.documents))
	for _, document := range g.documents {
		inputs = append(inputs, DeltaInput{Domain: document.declared.Domain, Data: document.delta})
	}
	return inputs
}

// registered son las entradas de solo registro de la traza.
func (ev evaluation) registered() []TraceEntry {
	var entries []TraceEntry
	for _, entry := range ev.entries {
		if entry.Kind == EntryRegister {
			entries = append(entries, entry)
		}
	}
	return entries
}

// contextEntries son las entradas de la traza del delta de contexto: una por
// entrada con los IDs provisionales sustituidos en el motivo (REQ-24.59).
func (ev evaluation) contextEntries() []TraceEntry {
	var entries []TraceEntry
	for _, item := range ev.contextItems {
		action := ActionModified
		switch ContextAction(item.Action) {
		case ContextAdd:
			action = ActionAdded
		case ContextRemove:
			action = ActionRemoved
		}
		entries = append(entries, TraceEntry{ID: item.ID, Kind: EntryContext, Document: item.Document, Anchor: item.Anchor, Action: action, State: StatePending, Reason: item.Reason})
	}
	return entries
}

// affected son los documentos que la integración edita: los de composición y los
// dirigidos, y los del delta de contexto. Los de solo registro no se editan. Los
// documentos que archive crea quedan exentos de la precondición (REQ-24.49).
func (ev evaluation) affected(g *gathered, loc workspace.Location) []affectedPath {
	var paths []affectedPath
	for _, document := range g.documents {
		if document.mode != workspace.IntegrationRegister {
			paths = append(paths, affectedPath{Path: document.abs})
		}
	}
	scope := contextScope(loc)
	for _, item := range ev.contextItems {
		paths = append(paths, affectedPath{Path: scope.Resolve(item.Document), Creates: item.Create})
	}
	return paths
}

// watchedDirectories son los directorios cuyos repositorios se vigilan sin
// detener la integración: el de especificaciones, los de las rutas mantenidas y de
// referencia y los de los roles declarados (D-15). Solo los que existen.
func watchedDirectories(loc workspace.Location) []string {
	candidates := []string{loc.WorkspaceRoot}
	scope := contextScope(loc)
	for _, declared := range append(append([]string(nil), scope.Maintained...), scope.Reference...) {
		candidates = append(candidates, scope.Resolve(declared))
	}
	if loc.Config != nil {
		roles := make([]string, 0, len(loc.Config.Roles))
		for name := range loc.Config.Roles {
			roles = append(roles, name)
		}
		sort.Strings(roles)
		for _, name := range roles {
			for _, repo := range loc.Config.Roles[name].Repositories {
				location := filepath.Clean(strings.TrimSpace(repo.Path))
				if !filepath.IsAbs(location) {
					location = filepath.Join(loc.Master, location)
				}
				candidates = append(candidates, location)
			}
		}
	}
	var watched []string
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			candidate = filepath.Dir(candidate)
		}
		watched = append(watched, candidate)
	}
	return watched
}

// authorizeContextEdits comprueba que cada documento que edita el delta de
// contexto está bajo alguna raíz de edición autorizada del cambio
// (actionContext.allowedEditRoots). El repositorio de especificaciones siempre lo
// está; los de código hermanos solo con una autorización por cambio, que nunca se
// concede de forma implícita (REQ-24.37).
func authorizeContextEdits(ctx context.Context, t ChangeTarget, ev evaluation, report *Report) {
	allowed := t.AllowedRoots
	if len(allowed) == 0 {
		allowed = []string{t.Location.WorkspaceRoot}
	}
	scope := contextScope(t.Location)
	for _, item := range ev.contextItems {
		target := resolveDeepest(scope.Resolve(item.Document))
		authorized := false
		for _, root := range allowed {
			if pathidentity.Contains(root, target) {
				authorized = true
				break
			}
		}
		if authorized {
			continue
		}
		root := deepestExistingDir(target)
		if repo, err := openRepository(ctx, target); err == nil {
			root = repo.root
		}
		report.fail("la entrada %s edita %s, que no está bajo ninguna raíz de edición autorizada del cambio (actionContext.allowedEditRoots): falta la autorización de edición de %s. Continuación: concédela por cambio con `axiom sdd-attempt grant --cwd %s --change %s --root %s` (más `--change-instance`, `--request-id`, `--actor` y `--reason`); nunca se concede de forma implícita (REQ-24.37)",
			item.ID, item.Document, pathquote.Quote(root), pathquote.Quote(t.Location.WorkspaceRoot), pathquote.ShellWord(t.Change), pathquote.Quote(root))
	}
}
