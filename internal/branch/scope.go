package branch

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// ScopeSchema es el contrato de salida de `axiom branch scope` (REQ-25.51): propio y distinto del
// `axiom.branch-result/v1` de `status`.
const ScopeSchema = "axiom.branch-scope/v1"

// Resultado tipado de cada repositorio (REQ-25.53): dominio cerrado.
const (
	ScopeOutcomeOK           = "ok"
	ScopeOutcomeEmpty        = "empty"
	ScopeOutcomeUnverifiable = "unverifiable"
	ScopeOutcomeExempt       = "exempt"
)

// Motivos de `unverifiable` y de un pendiente (REQ-25.53, REQ-25.54): los tres primeros son el
// dominio cerrado de `unverifiable`; `not_merged` solo aparece en la lista `pending` del resumen,
// para una rama `ok` que aún no está fusionada en la integración.
const (
	ScopeReasonBranchMissing         = "branch_missing"
	ScopeReasonBaseMissing           = "base_missing"
	ScopeReasonForkPointUndetermined = "fork_point_undetermined"
	ScopeReasonNotMerged             = "not_merged"
)

// Oferta de verificación conjunta del resumen (REQ-25.54).
const (
	ScopeOfferUnverifiable    = "unverifiable"
	ScopeOfferPendingMerge    = "pending_merge"
	ScopeOfferNothingToVerify = "nothing_to_verify"
	ScopeOfferOfferable       = "offerable"
)

// Fuentes del punto de partida (D5, D6) y de las referencias de la rama (D4), y el único aviso.
const (
	ScopeStartFirstParent              = "first_parent"
	ScopeStartReflog                   = "reflog"
	ScopeSourceLocal                   = "local"
	ScopeSourceRemoteTracking          = "remote-tracking"
	ScopeWarningForeignCommitsPossible = "foreign_commits_possible"
)

// ScopeResult es el `axiom.branch-scope/v1`. Comparte con `axiom.branch-result/v1` la envoltura de
// estado, código, mensaje, continuación y detalles de un rechazo; las claves van en camelCase, cada
// valor enumerado lleva su token en inglés y, en un campo hermano, su etiqueta en castellano, las
// rutas son absolutas y la serialización es estable y sin marcas de tiempo (D10).
type ScopeResult struct {
	Schema        string                 `json:"schema"`
	Operation     string                 `json:"operation"`
	Status        Status                 `json:"status"`
	StatusLabel   string                 `json:"statusLabel"`
	Code          Code                   `json:"code,omitempty"`
	Message       string                 `json:"message,omitempty"`
	Continuation  string                 `json:"continuation,omitempty"`
	Details       map[string]string      `json:"details,omitempty"`
	Change        string                 `json:"change,omitempty"`
	Topology      workspace.TopologyType `json:"topology,omitempty"`
	WorkspaceRoot string                 `json:"workspaceRoot,omitempty"`
	Summary       *ScopeSummary          `json:"summary,omitempty"`
	Repositories  []RepositoryScope      `json:"repositories,omitempty"`
}

// ScopeSummary es el resumen a nivel de cambio con la oferta de verificación conjunta (REQ-25.54).
type ScopeSummary struct {
	Offer      string         `json:"offer"`
	OfferLabel string         `json:"offerLabel"`
	Pending    []ScopePending `json:"pending"` // siempre una lista, nunca null
}

// ScopePending es un repositorio que motiva una oferta distinta de `offerable`, con su rama y su
// motivo.
type ScopePending struct {
	Repository  string `json:"repository"`
	Branch      string `json:"branch,omitempty"`
	Reason      string `json:"reason"`
	ReasonLabel string `json:"reasonLabel"`
}

// RepositoryScope es la entrada de un repositorio de la instantánea (REQ-25.51, REQ-25.53).
type RepositoryScope struct {
	Name          string            `json:"name"`
	GitRoot       string            `json:"gitRoot"`
	Role          RepoRole          `json:"role"`
	RoleLabel     string            `json:"roleLabel"`
	Promotion     Promotion         `json:"promotion,omitempty"`
	Outcome       string            `json:"outcome"`
	OutcomeLabel  string            `json:"outcomeLabel"`
	Reason        string            `json:"reason,omitempty"` // solo con unverifiable
	ReasonLabel   string            `json:"reasonLabel,omitempty"`
	Message       string            `json:"message,omitempty"`
	Continuation  string            `json:"continuation,omitempty"` // el fetch que la persona debe autorizar
	Details       map[string]string `json:"details,omitempty"`
	Branch        *ScopeRef         `json:"branch,omitempty"`
	Base          *ScopeRef         `json:"base,omitempty"`
	Integration   *ScopeRef         `json:"integration,omitempty"`
	Start         *ScopeStart       `json:"start,omitempty"`
	Tip           string            `json:"tip,omitempty"`
	Range         string            `json:"range,omitempty"`
	Revisions     []string          `json:"revisions,omitempty"`
	Commits       []ScopeCommit     `json:"commits,omitempty"`
	Excluded      []string          `json:"excluded,omitempty"`
	Files         []string          `json:"files,omitempty"`
	Merged        *bool             `json:"merged,omitempty"` // solo en las entradas ok
	MergedBy      MergeLayer        `json:"mergedBy,omitempty"`
	MergedByLabel string            `json:"mergedByLabel,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
}

// ScopeRef es una referencia resuelta: el nombre de la rama, la referencia completa, de dónde salió
// (`local` o `remote-tracking`) y el commit al que apunta.
type ScopeRef struct {
	Name        string `json:"name"`
	Ref         string `json:"ref,omitempty"`
	Source      string `json:"source,omitempty"`
	SourceLabel string `json:"sourceLabel,omitempty"`
	SHA         string `json:"sha,omitempty"`
}

// NewScopeRef describe una referencia resuelta con la etiqueta de su fuente.
func NewScopeRef(name, ref, source, sha string) *ScopeRef {
	scopeRef := &ScopeRef{Name: name, Ref: ref, Source: source, SHA: sha}
	if source != "" {
		scopeRef.SourceLabel = scopeSourceLabels[source]
	}
	return scopeRef
}

// ScopeStart es el punto de partida del alcance y cómo se fijó (`first_parent` o `reflog`).
type ScopeStart struct {
	SHA         string `json:"sha"`
	Source      string `json:"source"`
	SourceLabel string `json:"sourceLabel"`
}

// NewScopeStart describe un punto de partida con la etiqueta de su fuente.
func NewScopeStart(sha, source string) *ScopeStart {
	return &ScopeStart{SHA: sha, Source: source, SourceLabel: Label(source)}
}

// ScopeCommit es un commit propio; Merge dice si es de fusión (más de un padre).
type ScopeCommit struct {
	SHA   string `json:"sha"`
	Merge bool   `json:"merge"`
}

// NewScopeResult devuelve un resultado `ready` y vacío de `axiom branch scope`.
func NewScopeResult() ScopeResult {
	return ScopeResult{Schema: ScopeSchema, Operation: "scope", Status: StatusReady, StatusLabel: Label(string(StatusReady))}
}

// Refused devuelve el resultado marcado con el rechazo: el estado sale del código (`git_failed` es
// `error`; el resto, `refused`) y recibe el código, el mensaje, la continuación y los detalles.
func (r ScopeResult) Refused(refusal *Refusal) ScopeResult {
	r.Status = refusal.Code.status()
	r.StatusLabel = Label(string(r.Status))
	r.Code, r.Message, r.Continuation, r.Details = refusal.Code, refusal.Message, refusal.Continuation, refusal.Details
	return r
}

// SetOutcome fija el resultado de la entrada con su etiqueta.
func (s *RepositoryScope) SetOutcome(outcome string) {
	s.Outcome, s.OutcomeLabel = outcome, Label(outcome)
}

// SetUnverifiable marca la entrada como `unverifiable` con su motivo, su mensaje, la continuación
// (el `fetch` que la persona debe autorizar, si procede) y los detalles.
func (s *RepositoryScope) SetUnverifiable(reason, message, continuation string, details map[string]string) {
	s.SetOutcome(ScopeOutcomeUnverifiable)
	s.Reason, s.ReasonLabel = reason, Label(reason)
	s.Message, s.Continuation, s.Details = message, continuation, details
}

// SetMerged fija el indicador de fusión en la integración y la capa que lo confirmó.
func (s *RepositoryScope) SetMerged(merged bool, by MergeLayer) {
	s.Merged = &merged
	s.MergedBy = by
	if by != "" {
		s.MergedByLabel = Label(string(by))
	}
}

// SummarizeScope calcula el resumen del cambio (REQ-25.54, D8): `pending` lista cada repositorio
// `unverifiable` con su motivo y cada `ok` sin fusionar con `not_merged`; `offer` sigue la
// precedencia `unverifiable` > `pending_merge` > `nothing_to_verify` > `offerable`. Los repositorios
// `empty` y `exempt` no son pendientes.
func SummarizeScope(repos []RepositoryScope) ScopeSummary {
	pending := []ScopePending{}
	var unverifiable, notMerged, ok bool
	for _, repo := range repos {
		branchName := ""
		if repo.Branch != nil {
			branchName = repo.Branch.Name
		}
		switch repo.Outcome {
		case ScopeOutcomeUnverifiable:
			unverifiable = true
			pending = append(pending, ScopePending{Repository: repo.Name, Branch: branchName, Reason: repo.Reason, ReasonLabel: Label(repo.Reason)})
		case ScopeOutcomeOK:
			ok = true
			if repo.Merged == nil || !*repo.Merged {
				notMerged = true
				pending = append(pending, ScopePending{Repository: repo.Name, Branch: branchName, Reason: ScopeReasonNotMerged, ReasonLabel: Label(ScopeReasonNotMerged)})
			}
		}
	}
	offer := ScopeOfferOfferable
	switch {
	case unverifiable:
		offer = ScopeOfferUnverifiable
	case notMerged:
		offer = ScopeOfferPendingMerge
	case !ok:
		offer = ScopeOfferNothingToVerify
	}
	return ScopeSummary{Offer: offer, OfferLabel: Label(offer), Pending: pending}
}

// ExitCode es la salida del proceso: 0 cuando el alcance se calculó (aunque haya repositorios
// `unverifiable`, REQ-25.2) y 1 ante un rechazo tipado o un error. El 2 es de los errores de uso,
// que no producen un resultado.
func (r ScopeResult) ExitCode() int {
	if r.Status == StatusReady {
		return ExitReady
	}
	return ExitRefused
}

// WriteJSON escribe el resultado como JSON con sangrado de dos espacios y un salto de línea final,
// sin escapar HTML. Las claves de Details salen ordenadas: dos escrituras del mismo resultado dan
// los mismos bytes.
func (r ScopeResult) WriteJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

// WriteText escribe el resultado en castellano: un rechazo va a stderr con su código y su
// continuación; un alcance calculado va a stdout, una línea por repositorio y la oferta.
func (r ScopeResult) WriteText(stdout, stderr io.Writer) {
	if r.Status != StatusReady {
		fmt.Fprintf(stderr, "Error [%s]: %s\nContinuación: %s\n", r.Code, r.Message, r.Continuation)
		return
	}
	fmt.Fprintf(stdout, "Cambio: %s\n", r.Change)
	for _, repo := range r.Repositories {
		line := fmt.Sprintf("- %s: %s", repo.Name, repo.OutcomeLabel)
		if repo.Reason != "" {
			line += " (" + repo.ReasonLabel + ")"
		}
		if repo.Branch != nil && repo.Branch.Name != "" {
			line += " · rama " + repo.Branch.Name
		}
		if repo.Range != "" {
			line += " · rango " + repo.Range
		}
		if repo.Merged != nil {
			if *repo.Merged {
				line += " · fusionada en la integración"
			} else {
				line += " · sin fusionar en la integración"
			}
		}
		fmt.Fprintln(stdout, line)
		if repo.Message != "" {
			fmt.Fprintf(stdout, "  %s\n", repo.Message)
		}
		if repo.Continuation != "" {
			fmt.Fprintf(stdout, "  Continuación: %s\n", repo.Continuation)
		}
		for _, warning := range repo.Warnings {
			fmt.Fprintf(stdout, "  Aviso: %s\n", Label(warning))
		}
	}
	if r.Summary != nil {
		fmt.Fprintf(stdout, "Oferta: %s\n", r.Summary.OfferLabel)
	}
}

// scopeSourceLabels son las etiquetas de la fuente de una referencia. `remote-tracking` es el token
// que ya usa el puerto Git (RefKind) y lleva un guion, de modo que no cabe en el mapa de etiquetas,
// que solo admite identificadores con «_».
var scopeSourceLabels = map[string]string{
	ScopeSourceLocal:          "rama local",
	ScopeSourceRemoteTracking: "referencia remota ya descargada",
}
