package sddpublish

import (
	"fmt"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
)

// Schema es el esquema del JSON de `axiom sdd publish --json` (DD-10, §5.3).
const Schema = "axiom.sdd-publish/v1"

// Remote es el único remoto al que publica `axiom sdd publish`.
const Remote = "origin"

// maxUnpublishedCommits es el límite de commits sin subir que se examinan (DD-09, paso 8); es una
// variable de paquete para que las pruebas lo reduzcan, y ningún código de producción la cambia. Con
// más, la negativa es `unpublished_range_too_large`.
var maxUnpublishedCommits = 500

// Request es la petición de `Publisher.Run`.
type Request struct {
	CWD     string // workspaceRoot resuelto por la CLI, nunca el directorio del proceso
	Change  string
	Archive bool   // plan de archivado (rama `-archive` o integración in situ)
	Branch  string // opcional: la rama que autorizó la persona
}

// Status es el estado de una publicación.
type Status string

const (
	StatusPublished Status = "published"
	StatusUpToDate  Status = "up_to_date"
	StatusRefused   Status = "refused"
)

// Context dice si la publicación es la del cambio o la de su archivado.
type Context string

const (
	ContextChange  Context = "change"
	ContextArchive Context = "archive"
)

// Code es el código tipado de una negativa: un token estable en inglés (DD-09, §5.3).
type Code string

// Los que reenvía `internal/branch` (`config_invalid`, `not_a_git_repository`, `git_failed`,
// `detached_head`, `wrong_branch`, `integration_branch` y los demás de la planificación) conservan
// su código y su etiqueta.
const (
	CodeStoreWithoutFiles       Code = "store_without_files"
	CodeChangeNotFound          Code = "change_not_found"
	CodeRemoteMissing           Code = "remote_missing"
	CodeUnpublishedRangeUnknown Code = "unpublished_range_unknown"
	CodeIdentityMissing         Code = "identity_missing"
	CodeCommitHookRejected      Code = "commit_hook_rejected"
	CodeCommitFailed            Code = "commit_failed"
	CodeUnpublishedRangeTooBig  Code = "unpublished_range_too_large"
	CodeCodeCommitsUnpublished  Code = "code_commits_unpublished"
	CodePushNonFastForward      Code = "push_non_fast_forward"
	CodePushHookRejected        Code = "push_hook_rejected"
	CodeRemoteUnavailable       Code = "remote_unavailable"
	CodePushFailed              Code = "push_failed"
)

// labels son las etiquetas en castellano de los códigos propios de la publicación.
var labels = map[Code]string{
	CodeStoreWithoutFiles:       "almacén sin ficheros",
	CodeChangeNotFound:          "cambio inexistente",
	CodeRemoteMissing:           "no existe el remoto origin",
	CodeUnpublishedRangeUnknown: "sin referencia para calcular lo no subido",
	CodeIdentityMissing:         "identidad de git ausente",
	CodeCommitHookRejected:      "un hook rechazó el commit",
	CodeCommitFailed:            "el commit falló",
	CodeUnpublishedRangeTooBig:  "más de 500 commits sin subir",
	CodeCodeCommitsUnpublished:  "commits de código sin subir",
	CodePushNonFastForward:      "el remoto avanzó",
	CodePushHookRejected:        "un hook (local o remoto) rechazó el push",
	CodeRemoteUnavailable:       "remoto inalcanzable o sin credenciales",
	CodePushFailed:              "el push falló",
}

// Label devuelve la etiqueta en castellano de un código: la propia de la publicación o, si es
// uno de los que reenvía `internal/branch`, la de ese paquete.
func Label(code Code) string {
	if label, known := labels[code]; known {
		return label
	}
	return branch.Label(string(code))
}

// Refusal es la negativa tipada de una publicación. Details lleva datos estructurados de la
// negativa: `commits`, `paths`, `timeout`, el recuento de commits.
type Refusal struct {
	Code         Code           `json:"code"`
	Label        string         `json:"label"`
	Message      string         `json:"message"`
	Continuation string         `json:"continuation"`
	Details      map[string]any `json:"details,omitempty"`
}

func refuse(code Code, message, continuation string, details map[string]any) *Refusal {
	return &Refusal{Code: code, Label: Label(code), Message: message, Continuation: continuation, Details: details}
}

// fromBranch convierte un rechazo de `internal/branch` en una negativa de la publicación, con su
// código, su mensaje y su continuación. Los detalles de texto de branch pasan tal cual.
func fromBranch(refusal *branch.Refusal) *Refusal {
	var details map[string]any
	if len(refusal.Details) > 0 {
		details = make(map[string]any, len(refusal.Details))
		for key, value := range refusal.Details {
			details[key] = value
		}
	}
	return refuse(Code(refusal.Code), refusal.Message, refusal.Continuation, details)
}

// RepositoryRef es el repositorio donde se publica.
type RepositoryRef struct {
	Name    string `json:"name"`
	GitRoot string `json:"gitRoot"`
}

// CommitResult es el commit de documentación que creó la publicación.
type CommitResult struct {
	SHA     string   `json:"sha"`
	Message string   `json:"message"`
	Paths   []string `json:"paths"`
}

// PushResult es el push de la rama: la refspec que se subió, el SHA publicado y si la rama
// remota se creó con ese push.
type PushResult struct {
	Refspec string `json:"refspec"`
	SHA     string `json:"sha"`
	Created bool   `json:"created"`
}

// Result es el resultado de `Publisher.Run` y el documento `axiom.sdd-publish/v1`. Commit y Push
// se omiten cuando no ocurrieron; con una negativa posterior al commit, Commit sigue presente
// porque el commit local se conserva. Refusal es null salvo con status `refused`.
type Result struct {
	Schema     string         `json:"schema"`
	Change     string         `json:"change"`
	Context    Context        `json:"context"`
	Status     Status         `json:"status"`
	Repository *RepositoryRef `json:"repository,omitempty"`
	Remote     string         `json:"remote"`
	Branch     string         `json:"branch,omitempty"`
	InPlace    bool           `json:"inPlace"`
	Commit     *CommitResult  `json:"commit,omitempty"`
	Push       *PushResult    `json:"push,omitempty"`
	// UnpublishedCommits son los commits que no estaban en el remoto antes de la publicación, del
	// más antiguo al más reciente: los que sube el push o, con `code_commits_unpublished`, los
	// que lo impiden.
	UnpublishedCommits []string `json:"unpublishedCommits,omitempty"`
	// Resume es lo que ejecuta quien siga con el cambio publicado.
	Resume  []string `json:"resume,omitempty"`
	Refusal *Refusal `json:"refusal"`
}

// ExitCode es el código de salida de la orden: 0 con `published` y `up_to_date`, 1 con `refused`.
func (r Result) ExitCode() int {
	if r.Status == StatusRefused {
		return 1
	}
	return 0
}

// ResumeSteps son las órdenes de quien continúa un cambio publicado: traer la rama (decisión de la
// persona) y dejar que `ensure` la adopte (REQ-27.15). No llevan `-C`: la ruta de quien siga no se
// conoce.
func ResumeSteps(change string) []string {
	return []string{"git fetch origin", fmt.Sprintf("axiom branch ensure --change %s --json", change)}
}
