package reqtrace

// Lane es el carril de trabajo de un incremento.
type Lane string

const (
	// LaneSDD es un cambio archivado en `changes/archive/<fecha>-<cambio>/`.
	LaneSDD Lane = "sdd"
	// LaneODD es un documento `odd/tasks/<feature>.md`.
	LaneODD Lane = "odd"
)

// Kind es el tipo de un incremento.
type Kind string

const (
	// KindIncrement es el tipo por defecto.
	KindIncrement Kind = "increment"
	// KindBug es un cambio con declaración de bug (SDD) o un documento con `Tipo: bug` (ODD).
	KindBug Kind = "bug"
)

// Role es lo que un incremento hace con el requisito.
type Role string

const (
	RoleCreated  Role = "created"
	RoleModified Role = "modified"
	RoleRemoved  Role = "removed"
	RoleRenamed  Role = "renamed"
	RoleCovered  Role = "covered"
)

// Anchor es el hecho de git que fija el commit de integración de un incremento.
type Anchor string

const (
	// AnchorCode es el commit que integró el código del cambio SDD.
	AnchorCode Anchor = "code"
	// AnchorArchive es el commit que archivó el cambio SDD, cuando no hay commit de código.
	AnchorArchive Anchor = "archive"
	// AnchorDocument es el primer commit que trae el documento ODD.
	AnchorDocument Anchor = "document"
)

// Códigos estables de las advertencias (design, decisión 5).
const (
	WarnODDUnparseable       = "odd_unparseable"
	WarnUnclassifiedMention  = "unclassified_mention"
	WarnTraceInvalid         = "trace_invalid"
	WarnProvisionalAmbiguous = "provisional_ambiguous"
	WarnODDNotIntegrated     = "odd_not_integrated"
	WarnODDRepoUnavailable   = "odd_repository_unavailable"
	WarnUnsafeName           = "unsafe_name"
	WarnUnreadable           = "unreadable"
)

// Commit es un commit de la línea `--first-parent`, o la constancia de que no se encontró.
type Commit struct {
	Commit string `json:"commit,omitempty"`
	Found  bool   `json:"found"`
	Anchor Anchor `json:"anchor,omitempty"`
}

// Task es una línea `Cubre:` que cita el requisito; File es relativo a la raíz git.
type Task struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// Assignment es una entrada de la traza de identificadores que toca el cierre de equivalentes.
type Assignment struct {
	Provisional string   `json:"provisional"`
	Definitive  string   `json:"definitive"`
	Capability  string   `json:"capability,omitempty"`
	Previous    []string `json:"previous,omitempty"`
}

// Warning es un dato que no se pudo usar; nunca aborta la consulta.
type Warning struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// Increment es un cambio SDD archivado o un documento ODD que toca el requisito con un rol.
// Hay un incremento por (fuente, rol).
type Increment struct {
	Lane         Lane         `json:"lane"`
	Name         string       `json:"name"`
	Path         string       `json:"path"`
	Type         Kind         `json:"type"`
	Role         Role         `json:"role"`
	IDs          []string     `json:"ids"`
	Capabilities []string     `json:"capabilities,omitempty"`
	Tasks        []Task       `json:"tasks"`
	Integration  Commit       `json:"integrationCommit"`
	Archive      *Commit      `json:"archiveCommit,omitempty"`
	Assignments  []Assignment `json:"assignments,omitempty"`
}

// Result es el documento `axiom.trace/v1`.
type Result struct {
	Schema            string      `json:"schema"`
	ID                string      `json:"id"`
	ResolvedIDs       []string    `json:"resolvedIds"`
	IntegrationBranch string      `json:"integrationBranch"`
	Increments        []Increment `json:"increments"`
	Warnings          []Warning   `json:"warnings"`
}

// normalize garantiza que ninguna lista del resultado sea nil, para que el JSON emita `[]`.
func (r *Result) normalize() {
	r.Schema = Schema
	if r.ResolvedIDs == nil {
		r.ResolvedIDs = []string{}
	}
	if r.Increments == nil {
		r.Increments = []Increment{}
	}
	if r.Warnings == nil {
		r.Warnings = []Warning{}
	}
	for i := range r.Increments {
		inc := &r.Increments[i]
		if inc.IDs == nil {
			inc.IDs = []string{}
		}
		if inc.Tasks == nil {
			inc.Tasks = []Task{}
		}
	}
}
