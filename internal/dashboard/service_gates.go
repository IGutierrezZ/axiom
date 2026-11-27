package dashboard

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/cli"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// Decisiones de puerta desde el dashboard (C-06-f, D8, D9 y D11). Este fichero no escribe
// `gates.yaml` ni ejecuta git: consulta, registra y publica por GatesGateway, que delega en el mismo
// núcleo que `axiom sdd gate record` y `axiom sdd publish`. Todas las negativas usan MarksRefusal.

// Códigos de rechazo propios de las rutas de puertas.
const (
	codeGateContentType      = "gate_content_type"
	codeGateBodyInvalid      = "gate_body_invalid"
	codeGateConfirmMissing   = "gate_confirm_missing"
	codeGatesGatewayMissing  = "gates_gateway_missing"
	codeGateBusy             = "gate_busy"
	codeGateNotPending       = "gate_not_pending"
	codeGateNotSDD           = "gate_not_sdd"
	codeGateChangeArchived   = "gate_change_archived"
	codeStatusUnreadable     = "status_unreadable"
	codeGateNotCheckpointed  = "gate_not_checkpointed"
	codePublishUnavailable   = "publish_unavailable"
	codePublicationCodeUnpub = "publication_code_unpublished"
	codePublicationNA        = "publication_unavailable"
	codePublicationBranch    = "publication_branch_mismatch"
	codeGateRecordFailed     = "gate_record_failed"
	codeGatePublishFailed    = "gate_publish_failed"
)

var gatePreflightCodes = preflightCodes{contentType: codeGateContentType, bodyInvalid: codeGateBodyInvalid}

// Estados y motivos del `GET` de la parada (D11).
const (
	gateStateAwait = "await_gate"
	gateStateNone  = "none"

	gateReasonContinuous = "continuous"
	gateReasonNoGate     = "no_gate"
	gateReasonNotSealed  = "not_sealed"
	gateReasonBlocked    = "blocked"
	gateReasonArchived   = "archived"
	gateReasonODD        = "odd"

	gateRecordSchema = "axiom.sdd-gate-record/v1"
)

// GateRecordRequest es el cuerpo de `POST /api/increments/gate`. Digest es el del artefacto que la
// persona vio; el actor no se admite: lo resuelve el núcleo con la identidad git de la raíz.
type GateRecordRequest struct {
	Change       string               `json:"change"`
	Gate         string               `json:"gate"`
	Digest       string               `json:"digest"`
	Decision     string               `json:"decision"`
	HandOff      bool                 `json:"handOff"`
	CommitReview string               `json:"commitReview"`
	Reason       string               `json:"reason"`
	Evidence     *GateEvidenceRequest `json:"evidence"`
	Confirm      bool                 `json:"confirm"`
}

// GateEvidenceRequest es la evidencia de la aprobación de `integration`: Kind es el tipo, Commit y
// BaseRef los de `pr_merged` y Text el texto de `deployment` o `attestation`.
type GateEvidenceRequest struct {
	Kind    string `json:"kind"`
	Commit  string `json:"commit"`
	BaseRef string `json:"baseRef"`
	Text    string `json:"text"`
}

// PublishGateRequest es el cuerpo de `POST /api/increments/gate/publish`: Branch es la rama que la
// pantalla mostró.
type PublishGateRequest struct {
	Change  string `json:"change"`
	Branch  string `json:"branch"`
	Confirm bool   `json:"confirm"`
}

// GateNoticeDTO es un aviso del registro o de la publicación.
type GateNoticeDTO struct {
	Source string `json:"source"`
	Text   string `json:"text"`
}

// GateRecordDTO es el documento `axiom.sdd-gate-record/v1` de un registro, más `verified` (la
// evidencia `pr_merged` comprobada frente a declarada).
type GateRecordDTO struct {
	Schema             string                 `json:"schema"`
	Change             string                 `json:"change"`
	Gate               string                 `json:"gate"`
	Decision           string                 `json:"decision"`
	Verified           bool                   `json:"verified"`
	HandedOff          bool                   `json:"handedOff,omitempty"`
	HandedOffTo        string                 `json:"handedOffTo,omitempty"`
	CommitReview       string                 `json:"commitReview,omitempty"`
	LastRoleClosed     *bool                  `json:"lastRoleClosed,omitempty"`
	PendingNonBlocking []GatePendingRole      `json:"pendingNonBlockingRoles,omitempty"`
	Publish            *cli.GatePublishTarget `json:"publish,omitempty"`
	PublishUnavailable string                 `json:"publishUnavailable,omitempty"`
	Delivery           *branch.HandOffPlan    `json:"delivery,omitempty"`
}

// GatePendingRole es un rol no bloqueante cuya puerta sigue abierta al cerrar el último bloqueante.
type GatePendingRole struct {
	Role   string `json:"role"`
	Policy string `json:"policy"`
	Status string `json:"status"`
}

// GateRecorded es la respuesta (201) de un registro.
type GateRecorded struct {
	Record  GateRecordDTO   `json:"record"`
	Notices []GateNoticeDTO `json:"notices"`
}

// GatePublished es la respuesta (200) de una publicación: `published`, `up_to_date` o
// `nothing_to_publish`.
type GatePublished struct {
	Status          string          `json:"status"`
	Remote          string          `json:"remote,omitempty"`
	Branch          string          `json:"branch,omitempty"`
	Commit          string          `json:"commit,omitempty"`
	Pushed          string          `json:"pushed,omitempty"`
	UncommittedDocs []string        `json:"uncommittedDocs"`
	Resume          []string        `json:"resume"`
	Notices         []GateNoticeDTO `json:"notices"`
}

// GateChoiceDTO es una salida de la parada, tal cual la proyecta la `GateQuestion`.
type GateChoiceDTO struct {
	Answer     string `json:"answer"`
	Label      string `json:"label"`
	Effect     string `json:"effect"`
	Invocation string `json:"invocation"`
}

// GatePublicationTargetDTO es el destino de publicación que la pregunta nombra.
type GatePublicationTargetDTO struct {
	Remote     string `json:"remote"`
	Branch     string `json:"branch"`
	PRTarget   string `json:"prTarget,omitempty"`
	Invocation string `json:"invocation"`
}

// GateCommitReviewDTO describe el modo de revisión por commit que ofrece la aprobación de `tasks`.
type GateCommitReviewDTO struct {
	Offered bool     `json:"offered"`
	Modes   []string `json:"modes"`
	Default string   `json:"default"`
}

// GateStopDTO es la parada abierta: la pregunta de la puerta con su digest.
type GateStopDTO struct {
	Gate                   string                    `json:"gate"`
	Digest                 string                    `json:"digest"`
	Headline               string                    `json:"headline"`
	Reason                 string                    `json:"reason"`
	Value                  string                    `json:"value"`
	Criteria               []string                  `json:"criteria"`
	Evidence               []string                  `json:"evidence"`
	Choices                []GateChoiceDTO           `json:"choices"`
	HandOffUnavailable     string                    `json:"handOffUnavailable,omitempty"`
	Publication            *GatePublicationTargetDTO `json:"publication,omitempty"`
	PublicationUnavailable string                    `json:"publicationUnavailable,omitempty"`
	CommitReview           GateCommitReviewDTO       `json:"commitReview"`
}

// GatePublicationDTO es la proyección `publication` del status.
type GatePublicationDTO struct {
	State       string   `json:"state"`
	Reasons     []string `json:"reasons,omitempty"`
	Remote      string   `json:"remote,omitempty"`
	Branch      string   `json:"branch,omitempty"`
	PRTarget    string   `json:"prTarget,omitempty"`
	Invocation  string   `json:"invocation,omitempty"`
	PushCommand string   `json:"pushCommand,omitempty"`
	CodeCommits []string `json:"codeCommits,omitempty"`
	Unavailable string   `json:"unavailable,omitempty"`
}

// GateLastDecisionDTO es el último registro del ledger, sin el actor.
type GateLastDecisionDTO struct {
	Gate        string `json:"gate"`
	Decision    string `json:"decision"`
	HandedOff   bool   `json:"handedOff"`
	HandedOffTo string `json:"handedOffTo,omitempty"`
	RecordedAt  string `json:"recordedAt"`
}

// GateStopResponse es la respuesta del `GET` de la parada: con o sin parada trae siempre el estilo,
// la publicación y el último registro, de los que la pantalla reconstruye «aprobada sin publicar».
type GateStopResponse struct {
	Change          string               `json:"change"`
	State           string               `json:"state"`
	Reason          string               `json:"reason,omitempty"`
	ExecutionStyle  string               `json:"executionStyle,omitempty"`
	NextRecommended string               `json:"nextRecommended,omitempty"`
	BlockedReasons  []string             `json:"blockedReasons"`
	Stop            *GateStopDTO         `json:"stop,omitempty"`
	Publication     *GatePublicationDTO  `json:"publication,omitempty"`
	LastDecision    *GateLastDecisionDTO `json:"lastDecision,omitempty"`
}

func gateBusy() *MarksRefusal {
	return marksRefuse(http.StatusConflict, codeGateBusy,
		"hay otra decisión o una publicación en curso sobre este cambio",
		"espera a que termine, recarga el detalle del incremento y repite")
}

func gatesGatewayMissing() *MarksRefusal {
	return &MarksRefusal{HTTPStatus: http.StatusServiceUnavailable, Status: marksStatusUnavailable, Code: codeGatesGatewayMissing,
		Message:      "este dashboard no tiene pasarela de puertas",
		Continuation: "arranca el dashboard con `axiom dashboard` desde el binario de Axiom"}
}

func gateConfirmMissing() *MarksRefusal {
	return marksRefuse(http.StatusUnprocessableEntity, codeGateConfirmMissing,
		"la acción exige confirmación explícita; no se ha tocado nada",
		"confirma la acción en el diálogo del detalle del incremento")
}

func gateArgsInvalid(message string) *MarksRefusal {
	return marksRefuse(http.StatusUnprocessableEntity, cli.GateCodeArgsInvalid, message,
		"corrige los datos de la decisión y repite la acción desde el detalle del incremento")
}

func statusUnreadable(err error, change, root string) *MarksRefusal {
	return marksRefuse(http.StatusConflict, codeStatusUnreadable,
		"no se puede leer el estado del cambio: "+err.Error(),
		"repáralo con `axiom sdd status --change "+change+" --cwd "+root+"` y recarga el detalle")
}

// gateTarget es dónde está el cambio: si está activo, su carpeta (la clave del cerrojo) y, si no, el
// motivo.
type gateTarget struct {
	path    string
	active  bool
	missing missingKind
}

// gateTargetFor localiza el cambio. Un workspace ilegible es un rechazo; un cambio que no está
// activo se devuelve con su motivo para que cada ruta lo proyecte a su manera.
func (s *Service) gateTargetFor(change string) (gateTarget, *MarksRefusal) {
	loc, err := workspace.Locate(s.getRootPath())
	if err != nil {
		return gateTarget{}, marksUnavailable(codeMarksWorkspace,
			"no se puede localizar el workspace: "+err.Error(),
			"corrígelo con `axiom workspace validate --path "+s.getRootPath()+"`")
	}
	if found, err := s.locateIncrement(change); err == nil && found.kind == "active" {
		return gateTarget{path: found.path, active: true}, nil
	}
	return gateTarget{missing: s.classifyMissingChange(loc, change)}, nil
}

// gateMissingRefusal proyecta el motivo de un cambio no activo en un rechazo de las rutas POST.
func gateMissingRefusal(kind missingKind, change string) *MarksRefusal {
	switch kind {
	case missingODD:
		return marksUnavailable(codeGateNotSDD,
			"«"+change+"» es una feature ODD: sus tareas no tienen puertas de decisión",
			"las puertas solo existen en cambios SDD de openspec/changes")
	case missingArchived:
		return marksUnavailable(codeGateChangeArchived,
			"el cambio «"+change+"» está archivado y ya no admite decisiones",
			"las puertas se deciden antes de archivar el cambio")
	}
	return changeNotFound(change)
}

func (s *Service) gatesLockFor(changePath string) *sync.Mutex {
	lock, _ := s.gatesLocks.LoadOrStore(changePath, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// gateSession es lo que comparten registrar y publicar tras los pasos 1 a 6 de D8: la pasarela, la
// parada consultada y el cerrojo ya tomado (se libera con unlock).
type gateSession struct {
	gw     GatesGateway
	stop   cli.GateStop
	unlock func()
}

// openGateSession aplica los pasos 3 a 6 de D8: pasarela, localización, cerrojo y consulta de la
// parada. Ninguno escribe.
func (s *Service) openGateSession(ctx context.Context, change string) (*gateSession, *MarksRefusal) {
	gw := s.gatesGateway()
	if gw == nil {
		return nil, gatesGatewayMissing()
	}
	target, refusal := s.gateTargetFor(change)
	if refusal != nil {
		return nil, refusal
	}
	if !target.active {
		return nil, gateMissingRefusal(target.missing, change)
	}
	lock := s.gatesLockFor(target.path)
	if !lock.TryLock() {
		return nil, gateBusy()
	}
	stop, err := gw.Stop(ctx, s.getRootPath(), change)
	if err != nil {
		lock.Unlock()
		return nil, statusUnreadable(err, change, s.getRootPath())
	}
	return &gateSession{gw: gw, stop: stop, unlock: lock.Unlock}, nil
}

// GateStop devuelve la parada del cambio, o su ausencia con el motivo (D11).
func (s *Service) GateStop(ctx context.Context, change string) (*GateStopResponse, *MarksRefusal) {
	if !validMarksChange(change) {
		return nil, jsonBodyInvalid(codeGateBodyInvalid, "el nombre del cambio no es válido")
	}
	gw := s.gatesGateway()
	if gw == nil {
		return nil, gatesGatewayMissing()
	}
	target, refusal := s.gateTargetFor(change)
	if refusal != nil {
		return nil, refusal
	}
	if !target.active {
		switch target.missing {
		case missingODD:
			return gateNoneResponse(change, gateReasonODD), nil
		case missingArchived:
			return gateNoneResponse(change, gateReasonArchived), nil
		}
		return nil, changeNotFound(change)
	}
	stop, err := gw.Stop(ctx, s.getRootPath(), change)
	if err != nil {
		return nil, statusUnreadable(err, change, s.getRootPath())
	}
	return gateStopResponse(change, stop), nil
}

func gateNoneResponse(change, reason string) *GateStopResponse {
	return &GateStopResponse{Change: change, State: gateStateNone, Reason: reason, BlockedReasons: []string{}}
}

// gateStopResponse proyecta lo que devuelve la pasarela sin recalcular nada.
func gateStopResponse(change string, stop cli.GateStop) *GateStopResponse {
	out := &GateStopResponse{
		Change: change, ExecutionStyle: stop.ExecutionStyle, NextRecommended: stop.NextRecommended,
		BlockedReasons: append([]string{}, stop.BlockedReasons...), Publication: gatePublicationDTO(stop.Publication),
	}
	if last := stop.LastDecision; last != nil {
		out.LastDecision = &GateLastDecisionDTO{Gate: string(last.Gate), Decision: string(last.Decision),
			HandedOff: last.HandedOff, HandedOffTo: last.HandedOffTo, RecordedAt: last.RecordedAt.UTC().Format("2006-01-02T15:04:05Z")}
	}
	if stop.Question != nil {
		out.State = gateStateAwait
		out.Stop = gateStopDTO(stop)
		return out
	}
	out.State = gateStateNone
	switch {
	case stop.ExecutionStyle == "":
		out.Reason = gateReasonNotSealed
	case stop.ExecutionStyle == string(kickoff.ExecutionContinuous):
		out.Reason = gateReasonContinuous
	case len(stop.BlockedReasons) > 0:
		out.Reason = gateReasonBlocked
	default:
		out.Reason = gateReasonNoGate
	}
	return out
}

func gateStopDTO(stop cli.GateStop) *GateStopDTO {
	q := stop.Question
	dto := &GateStopDTO{
		Gate: q.Gate, Digest: stop.Digest, Headline: q.Headline, Reason: q.Reason, Value: q.Value,
		Criteria: append([]string{}, q.Criteria...), Evidence: append([]string{}, q.Evidence...),
		Choices:            make([]GateChoiceDTO, 0, len(q.Choices)),
		HandOffUnavailable: q.HandOffUnavailable, PublicationUnavailable: q.PublicationUnavailable,
	}
	for _, choice := range q.Choices {
		dto.Choices = append(dto.Choices, GateChoiceDTO{Answer: choice.Answer, Label: choice.Label,
			Effect: choice.Effect, Invocation: choice.Invocation})
	}
	if q.Publication != nil {
		dto.Publication = &GatePublicationTargetDTO{Remote: q.Publication.Remote, Branch: q.Publication.Branch,
			PRTarget: q.Publication.PRTarget, Invocation: q.Publication.Invocation}
	}
	dto.CommitReview = GateCommitReviewDTO{Offered: kickoff.GateKey(q.Gate) == kickoff.GateTasks,
		Default: string(kickoff.CommitReviewNotify), Modes: []string{}}
	for _, mode := range kickoff.CommitReviewModes() {
		dto.CommitReview.Modes = append(dto.CommitReview.Modes, string(mode))
	}
	return dto
}

func gatePublicationDTO(p *sddstatus.PublicationProjection) *GatePublicationDTO {
	if p == nil {
		return nil
	}
	return &GatePublicationDTO{State: p.State, Reasons: p.Reasons, Remote: p.Remote, Branch: p.Branch,
		PRTarget: p.PRTarget, Invocation: p.Invocation, PushCommand: p.PushCommand, CodeCommits: p.CodeCommits,
		Unavailable: p.Unavailable}
}

// RecordGate registra la decisión de la puerta proyectada (D8). Cada paso responde y para; ninguno
// escribe antes del último, que delega en el núcleo con el digest que la persona vio.
func (s *Service) RecordGate(ctx context.Context, req GateRecordRequest) (*GateRecorded, *MarksRefusal) {
	if !validMarksChange(req.Change) {
		return nil, jsonBodyInvalid(codeGateBodyInvalid, "el nombre del cambio no es válido")
	}
	if !req.Confirm {
		return nil, gateConfirmMissing()
	}
	if req.Digest == "" {
		return nil, gateArgsInvalid("la decisión exige el digest del artefacto que se ha visto; no se ha escrito nada")
	}
	input, refusal := gateRecordInput(req)
	if refusal != nil {
		return nil, refusal
	}
	session, refusal := s.openGateSession(ctx, req.Change)
	if refusal != nil {
		return nil, refusal
	}
	defer session.unlock()

	projected := ""
	if session.stop.Question != nil {
		projected = session.stop.Question.Gate
	}
	if projected != req.Gate {
		refusal := marksRefuse(http.StatusConflict, codeGateNotPending,
			"la puerta «"+req.Gate+"» ya no es la que espera una decisión; no se ha escrito nada",
			"recarga el detalle del incremento para ver la parada vigente")
		refusal.Details = map[string]any{"seenGate": req.Gate, "projectedGate": projected}
		return nil, refusal
	}
	input.CWD = session.stop.RecordRoot
	outcome, err := session.gw.Record(ctx, input)
	if err != nil {
		return nil, gateRecordRefusal(err, req.Change, session.stop.RecordRoot)
	}
	return &GateRecorded{Record: gateRecordDTO(outcome), Notices: gateNotices(outcome.Notices)}, nil
}

// gateRecordInput traduce el cuerpo a la orden del núcleo y valida su forma con las mismas reglas que
// la CLI, antes de leer nada. La evidencia `{kind, commit, baseRef, text}` se mapea a los campos
// `EvidenceKind`, `Commit`, `BaseRef` y `Evidence`; un `commitReview` ausente queda vacío (notify).
func gateRecordInput(req GateRecordRequest) (cli.GateRecordInput, *MarksRefusal) {
	in := cli.GateRecordInput{
		Change: req.Change, Gate: kickoff.GateKey(req.Gate), Decision: kickoff.GateDecision(req.Decision),
		Reason: req.Reason, HandOff: req.HandOff, CommitReview: kickoff.CommitReviewMode(req.CommitReview),
		ExpectedDigest: req.Digest,
	}
	if e := req.Evidence; e != nil {
		in.EvidenceKind, in.Commit, in.BaseRef, in.Evidence = kickoff.EvidenceKind(e.Kind), e.Commit, e.BaseRef, e.Text
	}
	if err := kickoff.ValidateGateRecordArgs(kickoff.GateRecordArgs{Change: in.Change, Gate: in.Gate,
		Decision: in.Decision, Reason: in.Reason, HandOff: in.HandOff, EvidenceKind: in.EvidenceKind,
		Commit: in.Commit, BaseRef: in.BaseRef, Evidence: in.Evidence}); err != nil {
		return in, gateArgsInvalid(err.Error())
	}
	if in.CommitReview != "" && !validCommitReviewMode(in.CommitReview) {
		return in, gateArgsInvalid("`commitReview` desconocido «" + req.CommitReview + "»; usa notify, stop o stop-before-commit")
	}
	return in, nil
}

func validCommitReviewMode(mode kickoff.CommitReviewMode) bool {
	for _, candidate := range kickoff.CommitReviewModes() {
		if mode == candidate {
			return true
		}
	}
	return false
}

// gateRecordRefusal proyecta la negativa del núcleo con la tabla de D11: las tipadas con su código,
// las del motor de ramas con el suyo y su continuación, y el resto como `gate_record_failed`.
func gateRecordRefusal(err error, change, root string) *MarksRefusal {
	var coded *cli.GateRecordError
	if errors.As(err, &coded) {
		status := http.StatusConflict
		switch coded.Code {
		case cli.GateCodeArgsInvalid:
			status = http.StatusUnprocessableEntity
		case cli.GateCodeChangeNotFound:
			status = http.StatusNotFound
		}
		return marksRefuse(status, coded.Code, coded.Error(), gateCodedContinuation(coded.Code, change, root))
	}
	var refused *branch.RefusalError
	if errors.As(err, &refused) {
		return branchRefusal(refused.Refusal)
	}
	return marksRefuse(http.StatusInternalServerError, codeGateRecordFailed,
		"no se pudo registrar la decisión: "+err.Error(),
		"comprueba los permisos de la carpeta del cambio y repite la decisión")
}

func gateCodedContinuation(code, change, root string) string {
	switch code {
	case cli.GateCodeDigestChanged:
		return "reabre el detalle del incremento para ver el artefacto vigente o consulta `axiom sdd status --change " + change + " --cwd " + root + "`"
	case cli.GateCodeAlreadyResolved:
		return "recarga el detalle del incremento: la puerta ya tiene una aprobación"
	case cli.GateCodeArgsInvalid:
		return "corrige los datos de la decisión y repite la acción desde el detalle del incremento"
	}
	return "consulta `axiom sdd status --change " + change + " --cwd " + root + "` y repite la decisión"
}

// branchRefusal proyecta una negativa del motor de ramas (`wrong_branch`, etc.) con su continuación.
func branchRefusal(r branch.Refusal) *MarksRefusal {
	refusal := marksRefuse(http.StatusConflict, string(r.Code), r.Message, r.Continuation)
	if len(r.Details) > 0 {
		refusal.Details = map[string]any{}
		for key, value := range r.Details {
			refusal.Details[key] = value
		}
	}
	return refusal
}

func gateRecordDTO(o cli.GateRecordOutcome) GateRecordDTO {
	dto := GateRecordDTO{
		Schema: gateRecordSchema, Change: o.Change, Gate: string(o.Record.Gate), Decision: string(o.Record.Decision),
		Verified: o.Record.Verified, HandedOff: o.Delivery != nil, HandedOffTo: o.HandedOffTo,
		CommitReview: string(o.CommitReview), LastRoleClosed: o.LastRoleClosed, Publish: o.Publish,
		PublishUnavailable: o.PublishUnavailable, Delivery: o.Delivery,
	}
	for _, role := range o.PendingNonBlocking {
		dto.PendingNonBlocking = append(dto.PendingNonBlocking, GatePendingRole{Role: role.Role, Policy: string(role.Policy), Status: role.Status})
	}
	return dto
}

func gateNotices(notices []cli.GateNotice) []GateNoticeDTO {
	out := make([]GateNoticeDTO, 0, len(notices))
	for _, notice := range notices {
		out = append(out, GateNoticeDTO{Source: notice.Source, Text: notice.Text})
	}
	return out
}

// PublishGate publica los documentos del cambio en su rama atada (D8). Solo exige `checkpointed` y
// `publication.state == pending`: no mira el último registro, de modo que el reintento tras una
// aprobación sin publicar y la publicación tras una cesión usan la misma ruta. Nunca ejecuta
// `pushCommand`.
func (s *Service) PublishGate(ctx context.Context, req PublishGateRequest) (*GatePublished, *MarksRefusal) {
	if !validMarksChange(req.Change) {
		return nil, jsonBodyInvalid(codeGateBodyInvalid, "el nombre del cambio no es válido")
	}
	if !req.Confirm {
		return nil, gateConfirmMissing()
	}
	session, refusal := s.openGateSession(ctx, req.Change)
	if refusal != nil {
		return nil, refusal
	}
	defer session.unlock()
	stop := session.stop

	if stop.ExecutionStyle != string(kickoff.ExecutionCheckpointed) {
		return nil, marksRefuse(http.StatusConflict, codeGateNotCheckpointed,
			"la publicación desde el detalle solo existe en cambios con paradas de revisión (checkpointed)",
			"publica con `axiom sdd publish --change "+req.Change+" --cwd "+stop.WorkspaceRoot+"` cuando corresponda")
	}
	if stop.ArtifactStore == string(sddstatus.ArtifactStoreEngram) || stop.ArtifactStore == string(sddstatus.ArtifactStoreNone) {
		refusal := marksRefuse(http.StatusConflict, codePublishUnavailable,
			"el almacén declarado ("+stop.ArtifactStore+") no guarda ficheros: no hay documentos que publicar",
			"declara un almacén con ficheros (openspec o hybrid) para publicar los documentos del cambio")
		refusal.Details = map[string]any{"store": stop.ArtifactStore}
		return nil, refusal
	}
	publication := stop.Publication
	if publication == nil {
		return &GatePublished{Status: "nothing_to_publish", UncommittedDocs: []string{}, Resume: []string{}, Notices: []GateNoticeDTO{}}, nil
	}
	switch publication.State {
	case "pending":
	case "code_unpublished":
		refusal := marksRefuse(http.StatusConflict, codePublicationCodeUnpub,
			"hay commits de código sin subir: la publicación de documentos no los sube y no se ha ejecutado nada",
			"sube el código tú con `"+publication.PushCommand+"` y repite la publicación")
		refusal.Details = map[string]any{"pushCommand": publication.PushCommand, "codeCommits": publication.CodeCommits}
		return nil, refusal
	default:
		refusal := marksRefuse(http.StatusConflict, codePublicationNA,
			"la publicación no se puede calcular ahora ("+publication.Unavailable+"); no se ha tocado nada",
			"corrige el motivo con `axiom sdd status --change "+req.Change+" --cwd "+stop.WorkspaceRoot+"` y recarga el detalle")
		refusal.Details = map[string]any{"unavailable": publication.Unavailable}
		return nil, refusal
	}
	if req.Branch != publication.Branch {
		refusal := marksRefuse(http.StatusConflict, codePublicationBranch,
			"la rama que se mostró ("+req.Branch+") no es la que se publicaría ("+publication.Branch+"); no se ha tocado nada",
			"recarga el detalle del incremento, comprueba la rama que muestra y confirma de nuevo")
		refusal.Details = map[string]any{"requested": req.Branch, "expected": publication.Branch}
		return nil, refusal
	}

	result, err := session.gw.Publish(ctx, cli.ChangePublishRequest{CWD: stop.WorkspaceRoot, Change: req.Change, Branch: publication.Branch})
	if err != nil {
		var refused *branch.RefusalError
		if errors.As(err, &refused) {
			return nil, branchRefusal(refused.Refusal)
		}
		return nil, marksRefuse(http.StatusInternalServerError, codeGatePublishFailed,
			"no se pudo publicar: "+err.Error(),
			"comprueba el remoto y repite la publicación, o publica con `axiom sdd publish --change "+req.Change+" --cwd "+stop.WorkspaceRoot+"`")
	}
	if r := result.Refusal; r != nil {
		return nil, &MarksRefusal{HTTPStatus: http.StatusConflict, Status: marksStatusRefused, Code: r.Code, Label: r.Label,
			Message: r.Message, Continuation: r.Continuation, Details: r.Details}
	}
	docs, resume := result.UncommittedDocs, result.Resume
	if docs == nil {
		docs = []string{}
	}
	if resume == nil {
		resume = []string{}
	}
	return &GatePublished{Status: result.Status, Remote: result.Remote, Branch: result.Branch, Commit: result.Commit,
		Pushed: result.PushedSHA, UncommittedDocs: docs, Resume: resume, Notices: gateNotices(result.Notices)}, nil
}
