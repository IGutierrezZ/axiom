package dashboard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/changestart"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/integrations/manifest"
)

// Alta de un cambio desde un work item (D-08, prov:WEB:9). Secuencia: validación, fase 1 (o el
// staging pendiente de un reintento), `ensure`, carpeta del cambio, `proposal.md` y fase 2. El
// dashboard no sella `kickoff.yaml` (H-20): lo sella después el orquestador con `--infer`. Nunca se
// copia texto del work item (dato de terceros) a `proposal.md`: la intención es la tecleada o un texto
// genérico que cita la referencia externa. El descarte es siempre una decisión de la persona.

const (
	maxWorkItemRef    = 512
	maxWorkItemIntent = 4 << 10

	stageStart  = "start"
	stageEnsure = "ensure"
	stageFolder = "folder"
	stageAdopt  = "adopt"

	actionRetry   = "retry"
	actionDiscard = "discard"
)

// workItemFailure es un fallo del alta con su estado HTTP y su cuerpo tipado.
type workItemFailure struct {
	Status int
	Body   workItemErrorBody
}

func workItemFail(status int, code, message string) *workItemFailure {
	return &workItemFailure{Status: status, Body: workItemErrorBody{Code: code, Error: message}}
}

func invalidWorkItemRequest(message string) *workItemFailure {
	return workItemFail(http.StatusBadRequest, "invalid_request", message)
}

// validWorkItemType son los tipos de cambio que ofrece el alta (los de CreateIncrementRequest.Type).
var validWorkItemType = map[string]bool{"": true, "feature": true, "fix": true, "refactor": true, "architecture": true}

// validateStartRequest comprueba la entrada del alta (D-08, paso 1) sin tocar nada.
func validateStartRequest(req WorkItemStartRequest) (ref string, bad *workItemFailure) {
	ref = strings.TrimSpace(req.Ref)
	hasRef, hasChange := req.Ref != "", req.Change != ""
	switch {
	case hasRef == hasChange:
		return "", invalidWorkItemRequest("indica exactamente uno de ref (el work item, para empezar) o change (para reintentar un alta ya preparada)")
	case hasRef:
		if ref == "" || len(ref) > maxWorkItemRef || !validPlainText(ref) {
			return "", invalidWorkItemRequest("la referencia del work item debe tener entre 1 y 512 caracteres, sin saltos de línea ni caracteres de control")
		}
	default:
		if bad := validWorkItemChange(req.Change); bad != nil {
			return "", bad
		}
	}
	if len(req.Intent) > maxWorkItemIntent || !utf8.ValidString(req.Intent) {
		return "", invalidWorkItemRequest("la intención no puede superar 4 KiB")
	}
	if !validWorkItemType[strings.TrimSpace(req.Type)] {
		return "", invalidWorkItemRequest("el tipo de cambio debe ser feature, fix, refactor o architecture")
	}
	return ref, nil
}

func validPlainText(text string) bool {
	if !utf8.ValidString(text) {
		return false
	}
	for _, r := range text {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// validWorkItemChange acepta solo un nombre de cambio válido (un componente de rama, nunca una ruta).
func validWorkItemChange(change string) *workItemFailure {
	if err := changestart.ValidChangeName(change); err != nil {
		return invalidWorkItemRequest("el nombre del cambio no es válido: " + err.Error())
	}
	return nil
}

// StartFromWorkItem ejecuta el alta (D-08). Un fallo posterior a la fase 1 lleva `change` y las
// acciones `retry` y `discard`; la fase 1 no deja rastro cuando rechaza.
func (s *Service) StartFromWorkItem(ctx context.Context, req WorkItemStartRequest) (*WorkItemStartResponse, *workItemFailure) {
	ref, bad := validateStartRequest(req)
	if bad != nil {
		return nil, bad
	}
	gw := s.integrationsGateway()
	if gw == nil {
		return nil, workItemGatewayMissing()
	}
	cwd, err := filepath.Abs(s.getRootPath())
	if err != nil {
		return nil, workItemFail(http.StatusInternalServerError, "project_unresolved", "no se pudo resolver la ruta del proyecto: "+err.Error())
	}
	app := gw.WorkItemApplicability(cwd)
	if !app.Applicable {
		return nil, workItemFail(http.StatusUnprocessableEntity, "work_items_not_applicable",
			"el alta desde un work item no aplica: "+app.Reason)
	}
	m, known := manifest.Lookup(app.Plugin)
	if known {
		if bad := s.requireIntegrationSecrets(ctx, m); bad != nil {
			failure := workItemFail(bad.Status, bad.Code, bad.Message)
			failure.Body.EnvVar = bad.EnvVar
			return nil, failure
		}
	}
	fail := func(stage, change string, err error) *workItemFailure {
		return s.workItemGatewayFailure(ctx, m, err, stage, change)
	}

	var change, externalRef string
	var warnings []string
	fresh := false
	if ref != "" {
		out, err := gw.StartWorkItem(ctx, cwd, ref)
		if err != nil {
			return nil, fail(stageStart, "", err)
		}
		change, externalRef, warnings = out.Change, out.ExternalRef, append(warnings, out.Warnings...)
		if bad := validWorkItemChange(change); bad != nil {
			// La fase 1 devolvió un nombre que no puede llegar a una ruta: no se usa para nada más.
			return nil, workItemFail(http.StatusUnprocessableEntity, "invalid_change_name",
				"el nombre de cambio que derivó el work item no es válido: revisa el título y el tipo del work item")
		}
		fresh = out.Status == "staged"
	} else {
		change = req.Change
		out, err := gw.WorkItemStaged(cwd, change)
		if err != nil {
			return nil, fail(stageStart, "", err)
		}
		externalRef = out.ExternalRef
	}

	// Un cambio activo o archivado ajeno con el mismo nombre derivado no se reutiliza en silencio: el
	// staging que acaba de crear esta petición se retira para que «Reintentar» no lo adopte.
	if fresh {
		if exists, err := s.changeNameTaken(change); err != nil {
			return nil, fail(stageFolder, change, err)
		} else if exists {
			return nil, s.foreignChangeFailure(gw, cwd, change)
		}
	}

	planningHome, err := s.ensureIncrementBranch(ctx, change, req.Mode)
	if err != nil {
		return nil, fail(stageEnsure, change, err)
	}
	activePath, err := prepareIncrementFolder(planningHome, change, !fresh)
	if err != nil {
		return nil, fail(stageFolder, change, err)
	}
	proposalPath, err := writeWorkItemProposal(activePath, change, externalRef, req)
	if err != nil {
		return nil, fail(stageFolder, change, err)
	}

	adopted, err := gw.AdoptWorkItem(ctx, filepath.Dir(planningHome), change)
	if err != nil {
		return nil, fail(stageAdopt, change, err)
	}
	warnings = append(warnings, adopted.Warnings...)
	if warnings == nil {
		warnings = []string{}
	}
	return &WorkItemStartResponse{Success: true, Name: change, Path: proposalPath, ExternalRef: externalRef, Warnings: warnings}, nil
}

// DiscardWorkItemStart retira solo el staging de un alta fallida (como `axiom change start --discard`).
func (s *Service) DiscardWorkItemStart(ctx context.Context, req WorkItemDiscardRequest) (*WorkItemDiscardResponse, *workItemFailure) {
	if bad := validWorkItemChange(req.Change); bad != nil {
		return nil, bad
	}
	gw := s.integrationsGateway()
	if gw == nil {
		return nil, workItemGatewayMissing()
	}
	cwd, err := filepath.Abs(s.getRootPath())
	if err != nil {
		return nil, workItemFail(http.StatusInternalServerError, "project_unresolved", "no se pudo resolver la ruta del proyecto: "+err.Error())
	}
	app := gw.WorkItemApplicability(cwd)
	if !app.Applicable {
		return nil, workItemFail(http.StatusUnprocessableEntity, "work_items_not_applicable",
			"el alta desde un work item no aplica: "+app.Reason)
	}
	m, _ := manifest.Lookup(app.Plugin)
	out, err := gw.DiscardWorkItem(cwd, req.Change)
	if err != nil {
		return nil, s.workItemGatewayFailure(ctx, m, err, "", "")
	}
	warnings := out.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return &WorkItemDiscardResponse{Change: out.Change, Status: out.Status, Warnings: warnings}, nil
}

func workItemGatewayMissing() *workItemFailure {
	return workItemFail(http.StatusServiceUnavailable, "integrations_gateway_missing",
		"el servidor no tiene configurada la pasarela de integraciones: reinicia Axiom con una versión completa")
}

// changeNameTaken dice si ya hay un cambio activo o archivado con ese nombre en el planning home del
// proyecto (la misma comprobación que el alta manual).
func (s *Service) changeNameTaken(change string) (bool, error) {
	root, err := s.planningRoot()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(filepath.Join(root, "changes", change)); err == nil {
		return true, nil
	}
	entries, err := os.ReadDir(filepath.Join(root, "changes", "archive"))
	if err != nil {
		return false, nil
	}
	for _, e := range entries {
		if e.IsDir() && (e.Name() == change || strings.HasSuffix(e.Name(), "-"+change)) {
			return true, nil
		}
	}
	return false, nil
}

// foreignChangeFailure es el rechazo `change_exists`: retira el staging recién creado y, si no puede,
// ofrece descartarlo a mano.
func (s *Service) foreignChangeFailure(gw IntegrationsGateway, cwd, change string) *workItemFailure {
	failure := workItemFail(http.StatusConflict, "change_exists",
		fmt.Sprintf("ya existe un cambio con el nombre %s: no se reutiliza para este work item; renómbralo o archívalo y vuelve a empezar", change))
	failure.Body.Change = change
	failure.Body.Stage = stageFolder
	if _, err := gw.DiscardWorkItem(cwd, change); err != nil {
		failure.Body.Actions = []string{actionDiscard}
	}
	return failure
}

// writeWorkItemProposal escribe `proposal.md` con la plantilla del alta manual solo si aún no existe
// (un reintento no pisa lo que la persona haya editado). La intención es la tecleada o un texto
// genérico que cita la referencia externa; nunca texto del work item.
func writeWorkItemProposal(activePath, change, externalRef string, req WorkItemStartRequest) (string, error) {
	proposalPath := filepath.Join(activePath, "proposal.md")
	if _, err := os.Stat(proposalPath); err == nil {
		return proposalPath, nil
	}
	intent := strings.TrimSpace(req.Intent)
	if intent == "" {
		intent = fmt.Sprintf("Cambio creado desde el work item %s: completa aquí el propósito.", externalRef)
	}
	res, err := writeIncrementProposal(activePath, change, CreateIncrementRequest{Name: change, Intent: intent, Type: req.Type})
	if err != nil {
		return "", err
	}
	return res.Path, nil
}

// workItemGatewayFailure traduce un error de la pasarela, de `ensure` o de la carpeta al fallo tipado
// de la etapa. Tras la fase 1 (cualquier etapa distinta de start) ofrece `retry` y `discard`, salvo
// cuando ya no hay staging (`staging_not_found`). Todo texto pasa por la redacción de secretos.
func (s *Service) workItemGatewayFailure(ctx context.Context, m manifest.Manifest, err error, stage, change string) *workItemFailure {
	redact := func(text string) string {
		return redactSecrets(text, secretValues(context.WithoutCancel(ctx), m)...)
	}
	actions := func(f *workItemFailure) *workItemFailure {
		f.Body.Stage, f.Body.Change = stage, change
		if stage != "" && stage != stageStart && f.Body.Code != "staging_not_found" {
			f.Body.Actions = []string{actionRetry, actionDiscard}
		}
		return f
	}

	var refusal *cli.WorkItemRefusal
	var ensureRefusal *branch.RefusalError
	var exists *incrementExistsError
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded):
		return actions(workItemFail(http.StatusGatewayTimeout, "integration_timeout",
			"el servicio externo no respondió a tiempo: comprueba la red y vuelve a intentarlo"))
	case errors.As(err, &refusal):
		f := workItemFail(workItemRefusalStatus(refusal.Code, stage), refusal.Code, redact(refusal.Message))
		f.Body.Continuation = redact(refusal.Continuation)
		f.Body.Reason = refusal.Fields["reason"]
		return actions(f)
	case errors.As(err, &ensureRefusal):
		if ensureRefusal.Code == branch.CodeDecisionRequired && ensureRefusal.Decision != nil {
			status := string(branch.StatusDecisionRequired)
			f := workItemFail(http.StatusConflict, string(ensureRefusal.Code), ensureRefusal.Message)
			f.Body.Status, f.Body.StatusLabel, f.Body.Decision = status, branch.Label(status), ensureRefusal.Decision
			f.Body.Stage, f.Body.Change = stage, change
			return f
		}
		f := workItemFail(http.StatusConflict, string(ensureRefusal.Code), ensureRefusal.Message)
		f.Body.Continuation = ensureRefusal.Continuation
		return actions(f)
	case errors.As(err, &exists):
		return actions(workItemFail(http.StatusConflict, "change_exists", exists.Error()))
	case stage == stageEnsure:
		return actions(workItemFail(http.StatusInternalServerError, "ensure_failed", redact(err.Error())))
	case stage == stageFolder:
		return actions(workItemFail(http.StatusInternalServerError, "folder_write_failed", redact(err.Error())))
	}
	// Incluye cualquier error de la pasarela que no es un rechazo: su texto ya sale saneado del plugin.
	return actions(workItemFail(http.StatusBadGateway, "integration_failed", redact(err.Error())))
}

// workItemRefusalStatus elige el estado HTTP de un rechazo de las fases del alta: los que dependen
// del estado local (staging, cerrojo, preparación del cambio) son 409; el resto, 422.
func workItemRefusalStatus(code, stage string) int {
	switch code {
	case "staging_not_found", "staging_conflict", "change_start_busy", "change_not_prepared":
		return http.StatusConflict
	}
	if stage == stageAdopt || stage == stageEnsure || stage == stageFolder {
		return http.StatusConflict
	}
	return http.StatusUnprocessableEntity
}
