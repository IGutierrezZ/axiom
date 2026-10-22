package branch

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// ResultSchema es el contrato de salida común de la familia `axiom branch`
// (design §4.3, REQ-25.2).
const ResultSchema = "axiom.branch-result/v1"

// Códigos de salida de `axiom branch` (design D-02): 0 únicamente cuando la
// operación ha dejado el estado pedido (`ready`, o `not_started` en `status`); 1 ante
// cualquier rechazo tipado, `decision_required`, `partial` o `error`; 2 ante un error
// de uso, que no produce un resultado.
const (
	ExitReady   = 0
	ExitRefused = 1
	ExitUsage   = 2
)

// Result es el `axiom.branch-result/v1`. Las claves van en camelCase; cada valor
// enumerado lleva su token estable en inglés y, en un campo hermano, su etiqueta en
// castellano; `code` aparece solo cuando el estado no es «listo»; todas las rutas son
// absolutas; la serialización es estable y no lleva marcas de tiempo (REQ-25.2).
//
// Se construye con NewResult, Refused y NotStarted, que mantienen el estado y su
// etiqueta emparejados (SetMode hace lo propio con el modo). Los campos de cada
// operación (por ahora, los de `status`) los rellena quien la ejecuta, y solo en un
// resultado que no es un rechazo.
type Result struct {
	Schema       string            `json:"schema"`
	Operation    string            `json:"operation"`
	Status       Status            `json:"status"`
	StatusLabel  string            `json:"statusLabel"`
	Code         Code              `json:"code,omitempty"`
	Message      string            `json:"message,omitempty"`
	Continuation string            `json:"continuation,omitempty"`
	Details      map[string]string `json:"details,omitempty"`
	// Decision es la pregunta tipada de un `decision_required` (REQ-25.8): quien invoca la
	// presenta y repite la orden con la respuesta. Solo la lleva ese estado.
	Decision *Decision `json:"decision,omitempty"`
	Change   string    `json:"change,omitempty"`
	Lane     Lane      `json:"lane,omitempty"`
	// Mode es el modo de trabajo materializado del cambio y ModeSource, de dónde salió
	// (ModeSource*); solo se conocen cuando el cambio tiene registro local.
	Mode            Mode                   `json:"mode,omitempty"`
	ModeLabel       string                 `json:"modeLabel,omitempty"`
	ModeSource      string                 `json:"modeSource,omitempty"`
	ModeSourceLabel string                 `json:"modeSourceLabel,omitempty"`
	Topology        workspace.TopologyType `json:"topology,omitempty"`
	// WorktreeRoot es la raíz de worktrees del plan (REQ-25.11); solo la informa `ensure`,
	// que es quien los crea.
	WorktreeRoot  string             `json:"worktreeRoot,omitempty"`
	WorkspaceRoot string             `json:"workspaceRoot,omitempty"`
	PlanningHome  string             `json:"planningHome,omitempty"`
	OddDocument   string             `json:"oddDocument,omitempty"`
	Registry      *RegistryResult    `json:"registry,omitempty"`
	Repositories  []RepositoryResult `json:"repositories,omitempty"`
	Skipped       []SkippedPath      `json:"skipped,omitempty"`
	// Registered son los cambios con registro local que `status` sin `--change` lista
	// (nil si no se listó; vacío pero no nil si se listó y no hay ninguno, que se imprime
	// como `[]`). PendingArchive son los cambios pendientes de archivar (REQ-25.20): una
	// lista sin `--change` y un booleano con él (nil si no se evaluó, con un aviso que lo
	// explica); PendingArchiveContinuation es la orden para archivar el cambio pedido,
	// solo cuando está pendiente.
	Registered                 []RegisteredChange `json:"registered,omitzero"`
	PendingArchive             *PendingArchive    `json:"pendingArchive,omitempty"`
	PendingArchiveContinuation string             `json:"pendingArchiveContinuation,omitempty"`
	Warnings                   []string           `json:"warnings,omitempty"`
}

// PendingChange es un cambio pendiente de archivar: ya está en el árbol de la rama de
// integración y aún no se ha archivado (REQ-25.20). Continuation es la orden que lo
// archiva.
type PendingChange struct {
	Change       string `json:"change"`
	Continuation string `json:"continuation"`
}

// NewPendingChange describe un cambio pendiente de archivar con la orden para archivarlo.
func NewPendingChange(change string) PendingChange {
	return PendingChange{Change: change, Continuation: archiveContinuation(change)}
}

// PendingArchive es el campo `pendingArchive` de `status` (design §4.3): sin `--change`,
// la lista de cambios pendientes de archivar de todo el proyecto (Changes, que se imprime
// `[]` si se evaluó y no hay ninguno); con él, un booleano que dice si el cambio pedido
// está pendiente (Requested).
type PendingArchive struct {
	Changes   []PendingChange
	Requested *bool
}

// MarshalJSON imprime el booleano de `--change` o, sin él, la lista.
func (p PendingArchive) MarshalJSON() ([]byte, error) {
	if p.Requested != nil {
		return json.Marshal(*p.Requested)
	}
	changes := p.Changes
	if changes == nil {
		changes = []PendingChange{}
	}
	return json.Marshal(changes)
}

// RegisteredChange es un cambio con registro local en la lista de `status` sin
// `--change` (REQ-25.14, REQ-25.15): su carril, su modo de trabajo materializado y su
// origen, cada token con su etiqueta, y las ramas del registro con su worktree.
type RegisteredChange struct {
	Change          string             `json:"change"`
	Lane            Lane               `json:"lane"`
	Mode            Mode               `json:"mode"`
	ModeLabel       string             `json:"modeLabel"`
	ModeSource      string             `json:"modeSource"`
	ModeSourceLabel string             `json:"modeSourceLabel"`
	Branches        []RegisteredBranch `json:"branches,omitempty"`
}

// RegisteredBranch es una rama del registro de un cambio, con el repositorio al que
// pertenece y, si lo tiene, su worktree (ruta absoluta).
type RegisteredBranch struct {
	Repository string     `json:"repository"`
	Name       string     `json:"name"`
	Kind       BranchKind `json:"kind"`
	KindLabel  string     `json:"kindLabel"`
	Role       string     `json:"role,omitempty"`
	Worktree   string     `json:"worktree,omitempty"`
}

// NewRegisteredChange resume un registro local para la lista de `status`: las ramas de
// todos sus repositorios, en el orden del registro.
func NewRegisteredChange(record Record) RegisteredChange {
	registered := RegisteredChange{
		Change:     record.Change,
		Lane:       record.Lane,
		Mode:       record.Mode,
		ModeLabel:  Label(string(record.Mode)),
		ModeSource: record.ModeSource, ModeSourceLabel: Label(record.ModeSource),
	}
	for _, repo := range record.Repositories {
		for _, recorded := range repo.Branches {
			registered.Branches = append(registered.Branches, RegisteredBranch{
				Repository: repo.Name,
				Name:       recorded.Name,
				Kind:       recorded.Kind,
				KindLabel:  Label(string(recorded.Kind)),
				Role:       recorded.RoleID,
				Worktree:   recorded.Worktree,
			})
		}
	}
	return registered
}

// RegistryResult informa del registro local del cambio (REQ-25.14): Present es falso
// cuando el cambio no tiene registro, que no es un error (`ensure` lo reconstruye).
type RegistryResult struct {
	Present bool `json:"present"`
}

// RepositoryResult describe un repositorio del plan de un cambio en el resultado de
// `status` (REQ-25.15). GitRoot y CommonDir son absolutas; IntegrationBranch es la de
// la política resuelta para ese repositorio (REQ-25.3). CurrentBranch queda vacía con
// HEAD desacoplado o de forma desconocida (esta última se acompaña de un aviso en
// Result.Warnings). Role lleva su etiqueta en RoleLabel; Roles son los roles del
// cambio que tocan el repositorio.
type RepositoryResult struct {
	Name              string         `json:"name"`
	GitRoot           string         `json:"gitRoot"`
	CommonDir         string         `json:"commonDir"`
	Role              RepoRole       `json:"role"`
	RoleLabel         string         `json:"roleLabel"`
	Roles             []string       `json:"roles,omitempty"`
	IntegrationBranch string         `json:"integrationBranch"`
	CurrentBranch     string         `json:"currentBranch,omitempty"`
	Detached          bool           `json:"detached,omitempty"`
	Base              *BaseResult    `json:"base,omitempty"`
	Branches          []BranchResult `json:"branches,omitempty"`
}

// BaseResult es la base de un repositorio: la referencia, su commit y de dónde sale
// (design D-06). Source es `remote-tracking` (la referencia remota que ya hay en
// local) o `local` (la rama de integración local); Fetched dice si esta ejecución
// hizo `fetch`, que `status` nunca hace.
type BaseResult struct {
	Ref     string `json:"ref"`
	SHA     string `json:"sha"`
	Source  string `json:"source"`
	Fetched bool   `json:"fetched"`
}

// BranchResult es una rama del plan y su estado en git (REQ-25.15). Name es la rama
// esperada; Worktree, la ruta absoluta del worktree donde está desplegada (vacía si no
// lo está); Ahead y Behind, los commits que lleva de adelanto y de atraso respecto de
// ComparedTo, calculados solo con referencias locales (nil cuando no se calcularon:
// la rama no existe o no se pudo resolver su base); PRTarget, la rama destino
// propuesta para su PR.
type BranchResult struct {
	Name      string     `json:"name"`
	Kind      BranchKind `json:"kind"`
	KindLabel string     `json:"kindLabel"`
	Role      string     `json:"role,omitempty"`
	Exists    bool       `json:"exists"`
	Worktree  string     `json:"worktree,omitempty"`
	// RegisteredWorktree es el worktree que el registro local guarda para la rama,
	// aparte del que git conoce (Worktree): así se ve cuando divergen. Vacío si el cambio
	// no tiene registro o este no lista worktree para la rama.
	RegisteredWorktree string `json:"registeredWorktree,omitempty"`
	Dirty              bool   `json:"dirty"`
	ComparedTo         string `json:"comparedTo,omitempty"`
	Ahead              *int   `json:"ahead,omitempty"`
	Behind             *int   `json:"behind,omitempty"`
	PRTarget           string `json:"prTarget"`
}

// NewRepositoryResult describe un repositorio del plan con sus ramas planificadas,
// todavía sin el estado de git: el motor rellena la rama actual, la base y, por rama,
// si existe, su worktree, su limpieza y su adelanto y atraso.
func NewRepositoryResult(repo RepoPlan) RepositoryResult {
	result := RepositoryResult{
		Name:              repo.Name,
		GitRoot:           repo.GitRoot,
		CommonDir:         repo.CommonDir,
		Role:              repo.Role,
		RoleLabel:         Label(string(repo.Role)),
		Roles:             repo.Roles,
		IntegrationBranch: repo.IntegrationBranch,
	}
	for _, planned := range repo.Branches {
		result.Branches = append(result.Branches, BranchResult{
			Name:      planned.Name,
			Kind:      planned.Kind,
			KindLabel: Label(string(planned.Kind)),
			Role:      planned.RoleID,
			PRTarget:  planned.PRTarget,
		})
	}
	return result
}

// NewResult devuelve un resultado listo de la operación dada, con su esquema.
func NewResult(operation string) Result {
	result := Result{Schema: ResultSchema, Operation: operation}
	result.setStatus(StatusReady)
	return result
}

func (r *Result) setStatus(status Status) {
	r.Status, r.StatusLabel = status, Label(string(status))
}

// SetMode fija el modo de trabajo materializado y su origen (ModeSource*), cada uno con
// su etiqueta en castellano.
func (r *Result) SetMode(mode Mode, source string) {
	r.Mode, r.ModeLabel = mode, Label(string(mode))
	r.ModeSource, r.ModeSourceLabel = source, Label(source)
}

// Refused devuelve el resultado marcado con el rechazo: su estado sale del código
// (`decision_required` y `error` tienen estado propio; el resto es `refused`) y
// recibe el código, el mensaje, la continuación y los detalles.
func (r Result) Refused(refusal *Refusal) Result {
	r.setStatus(refusal.Code.status())
	r.Code, r.Message, r.Continuation, r.Details = refusal.Code, refusal.Message, refusal.Continuation, refusal.Details
	r.Decision = refusal.Decision
	return r
}

// NotStarted devuelve el resultado marcado como no iniciado: un cambio sin ramas
// creadas no es un error (REQ-25.15) y la operación sale con ExitReady.
func (r Result) NotStarted() Result {
	r.setStatus(StatusNotStarted)
	return r
}

func (c Code) status() Status {
	switch c {
	case CodeDecisionRequired:
		return StatusDecisionRequired
	case CodeGitFailed:
		return StatusError
	default:
		return StatusRefused
	}
}

// ExitCode es el código de salida del resultado (ExitReady o ExitRefused).
func (r Result) ExitCode() int {
	switch r.Status {
	case StatusReady, StatusNotStarted:
		return ExitReady
	default:
		return ExitRefused
	}
}

// WriteJSON escribe el resultado como JSON con sangrado de dos espacios y un salto
// de línea final. No escapa HTML: una ruta o un mensaje con `<`, `>` o `&` sale tal
// cual. Las claves de Details salen ordenadas, de modo que dos escrituras del mismo
// resultado dan los mismos bytes.
func (r Result) WriteJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

// WriteText escribe el resultado como texto en castellano: un resultado que alcanzó
// el estado pedido va a stdout y todo lo demás a stderr, con la continuación en una
// línea aparte.
func (r Result) WriteText(stdout, stderr io.Writer) {
	if r.ExitCode() != ExitReady {
		fmt.Fprintf(stderr, "Error [%s]: %s\nContinuación: %s\n", r.Code, r.Message, r.Continuation)
		return
	}
	if r.Change != "" {
		fmt.Fprintf(stdout, "Cambio: %s (carril %s)\n", r.Change, r.Lane)
		fmt.Fprintf(stdout, "Estado: %s\n", r.StatusLabel)
	} else if r.Lane != "" {
		fmt.Fprintf(stdout, "Proyecto: %s (carril %s)\n", r.WorkspaceRoot, r.Lane)
	}
	if r.Mode != "" {
		fmt.Fprintf(stdout, "Modo de trabajo: %s (%s)\n", r.ModeLabel, r.ModeSourceLabel)
	}
	if r.WorktreeRoot != "" {
		fmt.Fprintf(stdout, "Raíz de worktrees: %s\n", r.WorktreeRoot)
	}
	if r.Registry != nil {
		if r.Registry.Present {
			fmt.Fprintln(stdout, "Registro local: presente")
		} else {
			fmt.Fprintln(stdout, "Registro local: ausente")
		}
	}
	for _, repository := range r.Repositories {
		repository.writeText(stdout)
	}
	for _, skipped := range r.Skipped {
		fmt.Fprintf(stdout, "Omitido: %s (rol %s): %s\n", skipped.Path, skipped.Role, skipped.Message)
	}
	r.writeChangeLists(stdout)
	for _, warning := range r.Warnings {
		fmt.Fprintf(stdout, "Aviso: %s\n", warning)
	}
}

// writeChangeLists escribe los cambios registrados y los pendientes de archivar de
// `status` sin `--change` y, con `--change`, si el cambio pedido está pendiente. Lo que no
// se evaluó no se menciona: el aviso del resultado dice por qué.
func (r Result) writeChangeLists(stdout io.Writer) {
	if r.Registered != nil {
		fmt.Fprintf(stdout, "Cambios registrados: %s\n", countOrNone(len(r.Registered)))
		for _, registered := range r.Registered {
			fmt.Fprintf(stdout, "  - %s: %s (%s)\n", registered.Change, registered.ModeLabel, registered.ModeSourceLabel)
			for _, recorded := range registered.Branches {
				line := fmt.Sprintf("    - %s: %s (%s)", recorded.Repository, recorded.Name, recorded.KindLabel)
				if recorded.Worktree != "" {
					line += ", worktree " + recorded.Worktree
				}
				fmt.Fprintln(stdout, line)
			}
		}
	}
	pending := r.PendingArchive
	switch {
	case pending == nil:
	case pending.Requested == nil:
		fmt.Fprintf(stdout, "Cambios pendientes de archivar: %s\n", countOrNone(len(pending.Changes)))
		for _, change := range pending.Changes {
			fmt.Fprintf(stdout, "  - %s: %s\n", change.Change, change.Continuation)
		}
	case *pending.Requested:
		fmt.Fprintf(stdout, "Pendiente de archivar: sí (%s)\n", r.PendingArchiveContinuation)
	}
}

// countOrNone es el recuento de una lista para el texto: «ninguno» si está vacía.
func countOrNone(count int) string {
	if count == 0 {
		return "ninguno"
	}
	return strconv.Itoa(count)
}

// writeText escribe un repositorio del resultado: con nombre, bajo su encabezado; sin
// él (un repositorio descrito solo por su raíz), sin sangrado.
func (r RepositoryResult) writeText(stdout io.Writer) {
	indent := ""
	if r.Name != "" {
		indent = "  "
		fmt.Fprintf(stdout, "%s (%s)\n", r.Name, r.RoleLabel)
	}
	fmt.Fprintf(stdout, "%sRepositorio git: %s\n", indent, r.GitRoot)
	fmt.Fprintf(stdout, "%sDirectorio común de git: %s\n", indent, r.CommonDir)
	switch {
	case r.Detached:
		fmt.Fprintf(stdout, "%sRama actual: ninguna (HEAD desacoplado)\n", indent)
	case r.CurrentBranch != "":
		fmt.Fprintf(stdout, "%sRama actual: %s\n", indent, r.CurrentBranch)
	}
	fmt.Fprintf(stdout, "%sRama de integración: %s\n", indent, r.IntegrationBranch)
	if len(r.Roles) > 0 {
		fmt.Fprintf(stdout, "%sRoles del cambio: %s\n", indent, strings.Join(r.Roles, ", "))
	}
	if r.Base != nil {
		fmt.Fprintf(stdout, "%sBase: %s (%s)\n", indent, r.Base.Ref, shortSHA(r.Base.SHA))
	}
	for _, branch := range r.Branches {
		fmt.Fprintf(stdout, "%s- %s (%s): %s\n", indent, branch.Name, branch.KindLabel, strings.Join(branch.textParts(), ", "))
	}
}

// textParts son los datos de una rama que salen en el texto: su estado en git y el
// destino de su PR.
func (b BranchResult) textParts() []string {
	var parts []string
	switch {
	case !b.Exists:
		parts = append(parts, "sin crear")
	case b.Worktree == "":
		parts = append(parts, "creada, sin worktree")
	case b.Dirty:
		parts = append(parts, "worktree "+b.Worktree, "con cambios sin confirmar")
	default:
		parts = append(parts, "worktree "+b.Worktree, "limpio")
	}
	if b.RegisteredWorktree != "" && b.RegisteredWorktree != b.Worktree {
		parts = append(parts, "worktree registrado "+b.RegisteredWorktree)
	}
	if b.Ahead != nil && b.Behind != nil {
		parts = append(parts, fmt.Sprintf("adelanto %d, atraso %d respecto de %s", *b.Ahead, *b.Behind, b.ComparedTo))
	}
	return append(parts, "PR hacia "+b.PRTarget)
}

// shortSHA abrevia un identificador de commit a los 12 primeros caracteres.
func shortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// Refusal es la negativa tipada de una operación: el código estable, el mensaje en
// castellano, la continuación (siempre una orden ejecutable o una acción concreta,
// regla 11 de tasks.md) y los detalles legibles por máquina.
type Refusal struct {
	Code         Code
	Message      string
	Continuation string
	Details      map[string]string
	// Decision solo la lleva `decision_required` (REQ-25.8).
	Decision *Decision
}

// Decision es la pregunta de un `decision_required` (design §4.3): el texto y las respuestas
// posibles, cada una con su efecto y la orden que la da. Los tokens de Answer son estables, en
// inglés; Label y Effect están en castellano.
type Decision struct {
	Question string           `json:"question"`
	Choices  []DecisionChoice `json:"choices"`
}

// DecisionChoice es una respuesta posible de una Decision: Answer es el valor de `--mode`,
// Label su etiqueta, Effect lo que hace y Invocation la orden completa que la da.
type DecisionChoice struct {
	Answer     string `json:"answer"`
	Label      string `json:"label"`
	Effect     string `json:"effect"`
	Invocation string `json:"invocation"`
}

// UnusableDirectory es el rechazo `not_a_git_repository` de un directorio de partida
// que no existe, no es un directorio o no se puede leer. reason es la frase que lo
// explica y ya nombra el directorio.
func UnusableDirectory(operation, reason string) *Refusal {
	return &Refusal{Code: CodeNotAGitRepository, Message: reason, Continuation: cwdContinuation(operation)}
}

// NotAGitRepository es el rechazo `not_a_git_repository` de un directorio que existe
// pero no pertenece a ningún repositorio git.
func NotAGitRepository(operation, dir string) *Refusal {
	return &Refusal{
		Code:         CodeNotAGitRepository,
		Message:      fmt.Sprintf("%s no pertenece a ningún repositorio git", dir),
		Continuation: initContinuation(operation, dir),
	}
}

// GitFailed es el error `git_failed`: una invocación de git falló por una causa que
// ningún otro código cubre. Nombra la operación que se intentaba (gitOperation, en
// castellano) y resume el error de git, también en Details para quien lee el JSON
// por máquina (REQ-25.2). operation es la de `axiom branch` que se repite en la
// continuación, y dir el directorio de partida.
func GitFailed(operation, dir, gitOperation string, err error) *Refusal {
	return &Refusal{
		Code:         CodeGitFailed,
		Message:      fmt.Sprintf("git falló al %s: %v", gitOperation, err),
		Continuation: retryContinuation(operation, dir),
		Details:      map[string]string{"operation": gitOperation, "summary": err.Error()},
	}
}
