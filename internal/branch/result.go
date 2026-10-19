package branch

import (
	"encoding/json"
	"fmt"
	"io"
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
// Se construye con NewResult y Refused, que mantienen el estado y su etiqueta
// emparejados. Los campos de cada operación (por ahora, los de `status`) los rellena
// quien la ejecuta.
type Result struct {
	Schema       string             `json:"schema"`
	Operation    string             `json:"operation"`
	Status       Status             `json:"status"`
	StatusLabel  string             `json:"statusLabel"`
	Code         Code               `json:"code,omitempty"`
	Message      string             `json:"message,omitempty"`
	Continuation string             `json:"continuation,omitempty"`
	Details      map[string]string  `json:"details,omitempty"`
	Repositories []RepositoryResult `json:"repositories,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
}

// RepositoryResult describe un repositorio del resultado de `status`. GitRoot y
// CommonDir son absolutas; IntegrationBranch es la de la política resuelta para ese
// repositorio (REQ-25.3). CurrentBranch queda vacía con HEAD desacoplado o de forma
// desconocida (esta última se acompaña de un aviso en Result.Warnings).
type RepositoryResult struct {
	GitRoot           string `json:"gitRoot"`
	CommonDir         string `json:"commonDir"`
	IntegrationBranch string `json:"integrationBranch"`
	CurrentBranch     string `json:"currentBranch,omitempty"`
	Detached          bool   `json:"detached,omitempty"`
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

// Refused devuelve el resultado marcado con el rechazo: su estado sale del código
// (`decision_required` y `error` tienen estado propio; el resto es `refused`) y
// recibe el código, el mensaje, la continuación y los detalles.
func (r Result) Refused(refusal *Refusal) Result {
	r.setStatus(refusal.Code.status())
	r.Code, r.Message, r.Continuation, r.Details = refusal.Code, refusal.Message, refusal.Continuation, refusal.Details
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
	for _, repository := range r.Repositories {
		fmt.Fprintf(stdout, "Repositorio git: %s\n", repository.GitRoot)
		fmt.Fprintf(stdout, "Directorio común de git: %s\n", repository.CommonDir)
		switch {
		case repository.Detached:
			fmt.Fprintln(stdout, "Rama actual: ninguna (HEAD desacoplado)")
		case repository.CurrentBranch != "":
			fmt.Fprintf(stdout, "Rama actual: %s\n", repository.CurrentBranch)
		}
		fmt.Fprintf(stdout, "Rama de integración: %s\n", repository.IntegrationBranch)
	}
	for _, warning := range r.Warnings {
		fmt.Fprintf(stdout, "Aviso: %s\n", warning)
	}
}

// Refusal es la negativa tipada de una operación: el código estable, el mensaje en
// castellano, la continuación (siempre una orden ejecutable o una acción concreta,
// regla 11 de tasks.md) y los detalles legibles por máquina.
type Refusal struct {
	Code         Code
	Message      string
	Continuation string
	Details      map[string]string
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
