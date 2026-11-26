package branch

import (
	"fmt"
	"io"
)

// Resultado tipado del ciclo `axiom autofetch run` (RF-BRA-39, design D-09). Es el contrato
// que consumen la CLI, el hook de inicio de sesión y, más adelante, el resto de órdenes: el
// ciclo no escribe texto, devuelve estos valores.

// AutofetchResultSchema es el esquema del resultado de `axiom autofetch run --json`.
const AutofetchResultSchema = "axiom.autofetch-result/v1"

// AutofetchOperation es el valor de AutofetchResult.Operation.
const AutofetchOperation = "autofetch run"

// Valores de AutofetchResult.Autofetch.
const (
	AutofetchStateEnabled  = "enabled"
	AutofetchStateDisabled = "disabled"
)

// Valores de AutofetchResult.Fetch.
const (
	AutofetchFetchDone            = "done"
	AutofetchFetchSkippedInterval = "skipped_interval"
	AutofetchFetchFailed          = "failed"
	AutofetchFetchTimeout         = "timeout"
	AutofetchFetchNoOrigin        = "no_origin"
	AutofetchFetchNotAttempted    = "not_attempted"
)

// Valores de AutofetchNotice.Kind (design D-09). Sin `origin` no hay aviso: `Fetch` vale
// `no_origin`, como en `ensure`.
const (
	AutofetchNoticeIncomingRelevant       = "incoming_relevant"
	AutofetchNoticeChangeBranchBehind     = "change_branch_behind"
	AutofetchNoticeChangeBranchNotAdopted = "change_branch_not_adopted"
	AutofetchNoticeIntegrationDiverged    = "integration_diverged"
	AutofetchNoticeOperationInProgress    = "integration_operation_in_progress"
	AutofetchNoticeIntegrationDetached    = "integration_detached"
	AutofetchNoticeNotDeployed            = "integration_not_deployed"
	AutofetchNoticeIntegrationDirty       = "integration_dirty"
	AutofetchNoticeFastForwardFailed      = "integration_ff_failed"
	AutofetchNoticeFetchFailed            = "fetch_failed"
	AutofetchNoticeFetchTimeout           = "fetch_timeout"
)

// AutofetchResult es el resultado de un ciclo de autofetch. Un ciclo nunca falla: lo que no
// pudo hacer queda como aviso o como `Fetch` distinto de `done`.
type AutofetchResult struct {
	Schema      string                `json:"schema"`
	Operation   string                `json:"operation"`
	Autofetch   string                `json:"autofetch"`            // enabled | disabled
	Project     string                `json:"project,omitempty"`    // carpeta maestra
	Repository  string                `json:"repository,omitempty"` // raíz git del repositorio de specs
	Fetch       string                `json:"fetch"`
	LastFetch   string                `json:"lastFetch,omitempty"` // RFC 3339 del sello
	Integration *AutofetchIntegration `json:"integration,omitempty"`
	Change      *AutofetchChange      `json:"change,omitempty"`
	Notices     []AutofetchNotice     `json:"notices"`
	Warnings    []string              `json:"warnings,omitempty"`
}

// AutofetchIntegration es lo que el ciclo hizo con la rama de integración del repositorio de
// specs.
type AutofetchIntegration struct {
	Branch   string `json:"branch"`
	Checkout string `json:"checkout,omitempty"`
	Before   string `json:"before,omitempty"`
	After    string `json:"after,omitempty"`
	State    string `json:"state"` // up_to_date | advanced | ahead | not_advanced | no_remote_ref | no_local_branch
	Advanced bool   `json:"advanced"`
}

// AutofetchChange es el cambio actual resuelto por la relevancia, si lo hay.
type AutofetchChange struct {
	Lane Lane   `json:"lane"`
	Name string `json:"name"`
}

// AutofetchNotice es un aviso del ciclo, con la orden exacta que la persona puede ejecutar.
type AutofetchNotice struct {
	Kind    string   `json:"kind"`
	Branch  string   `json:"branch"`
	Commits int      `json:"commits,omitempty"`
	Scope   []string `json:"scope,omitempty"`
	Message string   `json:"message"`
	Command string   `json:"command,omitempty"`
}

// NewAutofetchResult es el resultado de partida de un ciclo: con el esquema y la operación,
// el interruptor apagado, sin red intentada y con la lista de avisos vacía (nunca nula, para
// que el JSON sea `[]`).
func NewAutofetchResult() AutofetchResult {
	return AutofetchResult{
		Schema:    AutofetchResultSchema,
		Operation: AutofetchOperation,
		Autofetch: AutofetchStateDisabled,
		Fetch:     AutofetchFetchNotAttempted,
		Notices:   []AutofetchNotice{},
	}
}

// WriteText escribe el resultado como texto en castellano: el estado del interruptor, el
// fetch, la integración y los avisos con su orden (design D-09).
func (r AutofetchResult) WriteText(w io.Writer) {
	if r.Autofetch == AutofetchStateEnabled {
		_, _ = fmt.Fprintln(w, "Autofetch: encendido")
	} else {
		_, _ = fmt.Fprintln(w, "Autofetch: apagado")
	}
	if r.Repository != "" {
		_, _ = fmt.Fprintf(w, "Repositorio de specs: %s\n", r.Repository)
	}
	if r.Autofetch == AutofetchStateEnabled {
		_, _ = fmt.Fprintf(w, "Fetch: %s\n", fetchLabel(r.Fetch))
		if r.LastFetch != "" {
			_, _ = fmt.Fprintf(w, "Último fetch correcto: %s\n", r.LastFetch)
		}
	}
	if integration := r.Integration; integration != nil {
		_, _ = fmt.Fprintf(w, "Integración %s: %s\n", integration.Branch, integration.State)
	}
	if r.Change != nil {
		_, _ = fmt.Fprintf(w, "Cambio actual: %s\n", r.Change.Name)
	}
	for _, notice := range r.Notices {
		_, _ = fmt.Fprintf(w, "Aviso: %s\n", notice.Message)
		if notice.Command != "" {
			_, _ = fmt.Fprintf(w, "  %s\n", notice.Command)
		}
	}
	for _, warning := range r.Warnings {
		_, _ = fmt.Fprintf(w, "Aviso: %s\n", warning)
	}
}

// fetchLabel es la etiqueta en castellano de un valor de AutofetchResult.Fetch.
func fetchLabel(fetch string) string {
	switch fetch {
	case AutofetchFetchDone:
		return "traído"
	case AutofetchFetchSkippedInterval:
		return "omitido (dentro del intervalo mínimo)"
	case AutofetchFetchFailed:
		return "fallido"
	case AutofetchFetchTimeout:
		return "tiempo agotado"
	case AutofetchFetchNoOrigin:
		return "sin origin configurado"
	}
	return "no intentado"
}
