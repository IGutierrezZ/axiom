// Package branch es el dominio de `axiom branch` (INC-25, D-01, design §4.1): los
// tipos, la política de ramas, el resultado `axiom.branch-result/v1` y los puertos
// con los que el motor lee y escribe git, el registro local y las preferencias
// personales.
//
// No ejecuta nada: toda invocación de git pasa por gitcommon.Run (REQ-25.1) y una
// prueba de arquitectura (internal/gitcommon/architecture_test.go) fija que ningún
// fichero de este árbol lanza procesos ni abre la red. Los textos para el usuario y
// las etiquetas están en castellano; los identificadores y los tokens estables del
// JSON, en inglés, cada token con su campo hermano `<campo>Label` (REQ-25.2, G-4).
package branch

// Lane es el carril del cambio: `sdd` (por defecto) u `odd` (REQ-25.13).
type Lane string

const (
	LaneSDD Lane = "sdd"
	LaneODD Lane = "odd"
)

// Mode es el modo de trabajo, una preferencia personal que vive en el hub
// (REQ-25.8): un worktree por rama, la propia carpeta del proyecto, o preguntar.
type Mode string

const (
	ModeWorktree Mode = "worktree"
	ModeInPlace  Mode = "in_place"
	ModeAsk      Mode = "ask"
)

// RepoRole es el papel de un repositorio según la topología (REQ-25.5).
type RepoRole string

const (
	RepoChangeBranch RepoRole = "change_branch"
	RepoSpecsInPlace RepoRole = "specs_in_place"
	RepoCode         RepoRole = "code"
)

// BranchKind distingue la rama del cambio, las ramas de rol y la de archivado
// (REQ-25.4, REQ-25.45).
type BranchKind string

const (
	KindChange  BranchKind = "change"
	KindRole    BranchKind = "role"
	KindArchive BranchKind = "archive"
)

// Context dice si una consulta de rama es la ordinaria o la del archivado.
type Context string

const (
	ContextOrdinary Context = "ordinary"
	ContextArchive  Context = "archive"
)

// Consultant es quien consulta el veredicto de rama (REQ-25.18).
type Consultant string

const (
	ConsultantWrite  Consultant = "write"
	ConsultantStatus Consultant = "status"
	ConsultantApply  Consultant = "apply"
	ConsultantGuard  Consultant = "guard"
)

// VerdictKind es la respuesta a «¿esta rama es la que corresponde?» (REQ-25.18).
type VerdictKind string

const (
	VerdictCorrect       VerdictKind = "correct"
	VerdictWrongBranch   VerdictKind = "wrong_branch"
	VerdictNotApplicable VerdictKind = "not_applicable"
)

// MergeLayer es la capa de la detección de fusión de `done` que confirmó la
// integración de una rama (REQ-25.16, REQ-25.17).
type MergeLayer string

const (
	LayerAncestry       MergeLayer = "ancestry"
	LayerEquivalentTree MergeLayer = "equivalent_tree"
	LayerPRMerged       MergeLayer = "pr_merged"
	LayerForced         MergeLayer = "forced"
)

// Status es el estado tipado de un resultado (design D-02). Es un token estable en
// inglés y su etiqueta en castellano se obtiene con Label.
type Status string

const (
	StatusReady            Status = "ready"
	StatusDecisionRequired Status = "decision_required"
	StatusRefused          Status = "refused"
	StatusPartial          Status = "partial" // solo `done`
	StatusNotStarted       Status = "not_started"
	StatusError            Status = "error"
)

// Code es el código tipado con el que se identifica un resultado que no es «listo»:
// nunca se identifica por el texto del mensaje (spec, «Códigos tipados»).
type Code string

const (
	CodeDecisionRequired           Code = "decision_required"
	CodeDirtyWorktree              Code = "dirty_worktree"
	CodeWrongBranch                Code = "wrong_branch"
	CodeSpecsRepoNotOnIntegration  Code = "specs_repo_not_on_integration"
	CodeBaseNotFound               Code = "base_not_found"
	CodeFetchFailed                Code = "fetch_failed"
	CodePathTooLong                Code = "path_too_long"
	CodeInvalidName                Code = "invalid_name"
	CodeConfigInvalid              Code = "config_invalid"
	CodeGitFailed                  Code = "git_failed"
	CodePlanConflict               Code = "plan_conflict"
	CodeBranchCheckedOutElsewhere  Code = "branch_checked_out_elsewhere"
	CodeNotAGitRepository          Code = "not_a_git_repository"
	CodeMergeNotConfirmed          Code = "merge_not_confirmed"
	CodeChangeNotIntegrated        Code = "change_not_integrated"
	CodeArchiveBranchNotApplicable Code = "archive_branch_not_applicable"
)

// Exceptions son las excepciones por cambio de `ensure` (`--prefix`, `--branch`,
// `--base`, REQ-25.12). Se registran en el registro local (Record).
type Exceptions struct {
	Prefix string `json:"prefix"`
	Branch string `json:"branch"`
	Base   string `json:"base"`
}

// WorktreeEntry es un worktree que git conoce (`git worktree list --porcelain`).
type WorktreeEntry struct {
	Path     string // absoluta, limpia y con el separador del sistema
	Head     string // SHA al que apunta HEAD
	Branch   string // nombre corto de la rama; vacío con HEAD desacoplado
	Detached bool
}

// RegistrySchema es el esquema del registro local (design §4.4).
const RegistrySchema = "axiom.branch-registry/v1"

// Record es el plan materializado de un cambio, tal como lo guarda el registro
// local en el git common dir del repositorio del planning home (REQ-25.14, design
// §4.4). Es estado local no versionado: borrarlo no daña el repositorio y `ensure`
// lo reconstruye.
type Record struct {
	Schema        string       `json:"schema"`
	Lane          Lane         `json:"lane"`
	Change        string       `json:"change"`
	Mode          Mode         `json:"mode"`
	ModeSource    string       `json:"modeSource"`
	Exceptions    Exceptions   `json:"exceptions"`
	ArchivePrefix string       `json:"archivePrefix"`
	PlanningRepo  string       `json:"planningRepo"`
	Repositories  []RecordRepo `json:"repositories"`
}

// RecordRepo es un repositorio del plan registrado.
type RecordRepo struct {
	Name      string         `json:"name"`
	GitRoot   string         `json:"gitRoot"`
	CommonDir string         `json:"commonDir"`
	Role      RepoRole       `json:"role"`
	Branches  []RecordBranch `json:"branches"`
}

// RecordBranch es una rama del plan registrada, con su worktree si lo hay.
type RecordBranch struct {
	Name     string     `json:"name"`
	Kind     BranchKind `json:"kind"`
	RoleID   string     `json:"role,omitempty"`
	Worktree string     `json:"worktree,omitempty"`
}
