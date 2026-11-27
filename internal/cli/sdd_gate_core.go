package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Fuentes de los avisos que RecordGate recoge como datos (D2 del diseño de puertas-desde-ui).
const (
	gateNoticeBranchGuard         = "branch_guard"
	gateNoticeActorIdentity       = "actor_identity"
	gateNoticeIntegrationDispatch = "integration_dispatch"
)

// GateRecordInput es la orden tipada de registrar la decisión de una puerta: los mismos campos que
// `axiom sdd gate record` (el dashboard la rellena sin pasar por un argv).
type GateRecordInput struct {
	CWD      string
	Change   string
	Gate     kickoff.GateKey
	Decision kickoff.GateDecision
	Reason   string
	// Actor vacío: la identidad git de la raíz del workspace, como la CLI sin --actor.
	Actor        string
	HandOff      bool
	CommitReview kickoff.CommitReviewMode // vacío: sin modo (equivale a notify)
	EvidenceKind kickoff.EvidenceKind
	Commit       string
	BaseRef      string
	Evidence     string
	// ExpectedDigest es el digest del artefacto que la persona vio al decidir (solo lo rellena el
	// dashboard). Con valor, un digest recalculado distinto se rechaza como gate_digest_changed y el
	// registro se anexa siempre bajo el cerrojo del ledger, negando una puerta ya aprobada. La CLI
	// lo deja vacío y conserva su comportamiento.
	ExpectedDigest string
}

// Códigos de GateRecordError (D3 del diseño de puertas-desde-ui).
const (
	GateCodeArgsInvalid     = "gate_args_invalid"
	GateCodeChangeNotFound  = "change_not_found"
	GateCodeChangeArchived  = "change_archived"
	GateCodeNoStops         = "gate_no_stops"
	GateCodeKickoffMissing  = "gate_kickoff_missing"
	GateCodeRoleNotInRoster = "gate_role_not_in_roster"
	GateCodeHandOffNA       = "hand_off_unavailable"
	GateCodeArtifactMissing = "gate_artifact_missing"
	GateCodeAlreadyResolved = "gate_already_resolved"
	GateCodeDigestChanged   = "gate_digest_changed"
	GateCodeEvidenceRefused = "gate_evidence_refused"
)

// GateRecordError es una negativa de RecordGate con su código estable. Error() devuelve exactamente
// el mensaje de la CLI, de modo que el envoltorio es transparente.
type GateRecordError struct {
	Code string
	Err  error
}

func (e *GateRecordError) Error() string { return e.Err.Error() }
func (e *GateRecordError) Unwrap() error { return e.Err }

func gateCoded(code string, err error) error {
	if err == nil {
		return nil
	}
	return &GateRecordError{Code: code, Err: err}
}

// GateNotice es un aviso del registro que la CLI vuelca a su sumidero de siempre y el dashboard
// devuelve en la respuesta.
type GateNotice struct {
	Source string // branch_guard | actor_identity | integration_dispatch
	Text   string
}

// GateRecordOutcome es el resultado de RecordGate. Notices se devuelve también con error.
type GateRecordOutcome struct {
	Change, WorkspaceRoot string
	Record                kickoff.GateRecord // tal como se anexó
	HandedOffTo           string
	Delivery              *branch.HandOffPlan // solo con la salida 2
	CommitReview          kickoff.CommitReviewMode
	Publish               *GatePublishTarget
	PublishUnavailable    string
	LastRoleClosed        *bool
	PendingNonBlocking    []kickoff.PendingRole
	Notices               []GateNotice
}

// RecordGate valida y registra la decisión de una puerta: es el único escritor de `gates.yaml` de
// `gate record`. Conserva el orden de comprobaciones de la CLI y emite el evento de integración una
// sola vez, tras anexar el registro.
func RecordGate(ctx context.Context, in GateRecordInput) (GateRecordOutcome, error) {
	outcome := GateRecordOutcome{Change: in.Change, CommitReview: in.CommitReview}
	fail := func(err error) (GateRecordOutcome, error) { return outcome, err }
	notice := func(source string) func(string) {
		return func(text string) { outcome.Notices = append(outcome.Notices, GateNotice{Source: source, Text: text}) }
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}

	parsed := kickoff.GateRecordArgs{
		CWD: in.CWD, Change: in.Change, Gate: in.Gate, Decision: in.Decision, Reason: in.Reason,
		Actor: in.Actor, HandOff: in.HandOff, EvidenceKind: in.EvidenceKind, Commit: in.Commit,
		BaseRef: in.BaseRef, Evidence: in.Evidence,
	}
	if err := kickoff.ValidateGateRecordArgs(parsed); err != nil {
		return fail(gateCoded(GateCodeArgsInvalid, err))
	}
	commitReview := in.CommitReview
	if commitReview != "" {
		if _, err := parseCommitReviewMode(string(commitReview)); err != nil {
			return fail(gateCoded(GateCodeArgsInvalid, err))
		}
	}
	if err := validateCommitReviewShape(parsed, commitReview); err != nil {
		return fail(gateCoded(GateCodeArgsInvalid, err))
	}
	loc, changeRoot, err := resolveGovernanceChangeRoot(parsed.CWD, parsed.Change)
	if err != nil {
		return fail(err)
	}
	workspaceRoot := loc.WorkspaceRoot
	outcome.WorkspaceRoot = workspaceRoot
	parsed = completeGateBaseRef(parsed, loc)
	// Guarda A (RF-BRA-20): una sola consulta cubre AppendGate de esta invocación; la compuerta
	// de integración se registra en la rama de archivado (G-10).
	writerContext := branch.ContextOrdinary
	if parsed.Gate == kickoff.GateIntegration {
		writerContext = branch.ContextArchive
	}
	if err := guardChangeWriterCollect(changeRoot, parsed.Change, "sdd gate record", writerContext, notice(gateNoticeBranchGuard)); err != nil {
		return fail(err)
	}

	sealed, err := kickoff.Load(changeRoot)
	if err != nil {
		return fail(err)
	}
	if parsed.HandOff {
		if err := requireHandOffStops(parsed, sealed, workspaceRoot); err != nil {
			return fail(err)
		}
	}
	if commitReview != "" {
		if err := requireCommitReviewStops(parsed, sealed, workspaceRoot); err != nil {
			return fail(err)
		}
		// Sin cesión, prepareHandOff no corre: la puerta aún pendiente se comprueba aquí antes de
		// calcular nada más y otra vez bajo el cerrojo (RF-GAT-20).
		if !parsed.HandOff {
			ledger, err := kickoff.LoadGates(changeRoot)
			if err != nil {
				return fail(err)
			}
			if err := gateStillPendingCheck(parsed, workspaceRoot)(ledger); err != nil {
				return fail(err)
			}
		}
	}

	var roster []multirole.RoleAssignment
	if sealed != nil {
		roster = rosterFromKickoff(*sealed)
	}
	isRoleApply := isRoleApplyGate(parsed.Gate)
	if isRoleApply {
		if sealed == nil {
			return fail(gateCoded(GateCodeKickoffMissing, fmt.Errorf("no se puede registrar %q: el cambio %q no tiene un kickoff sellado con roster de roles; sella el kickoff primero, por ejemplo con `axiom sdd kickoff seal --cwd %s --change %s --execution-style checkpointed`",
				parsed.Gate, parsed.Change, pathquote.Quote(workspaceRoot), pathquote.Quote(parsed.Change))))
		}
		if !gateKeyInRoster(parsed.Gate, roster) {
			// refusal:by-design operator-knowledge: only the operator knows which sealed role they meant to target; the message already lists the complete valid roster, and no runnable command can pick a role for them -- the roster is fixed at kickoff time and this verb never re-seals it
			return fail(gateCoded(GateCodeRoleNotInRoster, fmt.Errorf("%q no pertenece al roster sellado de %q (roster: %s)", parsed.Gate, parsed.Change, strings.Join(rosterRoleNames(roster), ", "))))
		}
	}

	// Sin --actor, el actor es la identidad git de la raíz del workspace; sin identidad
	// completa se conserva "cli" y el aviso se emite solo si el registro llega a escribirse.
	actor, actorWarning := parsed.Actor, ""
	if actor == "" {
		actor, actorWarning = defaultActor(workspaceRoot, gateActorTarget)
	}
	digest, err := gateArtifactDigest(changeRoot, parsed.Gate, sealed)
	if err != nil {
		return fail(err)
	}
	if in.ExpectedDigest != "" && digest != in.ExpectedDigest {
		return fail(gateCoded(GateCodeDigestChanged, fmt.Errorf("el artefacto de la compuerta %q del cambio %q cambió desde que lo viste; vuelve a abrir el detalle o consulta `axiom sdd status %s --cwd %s` y decide sobre la versión actual",
			parsed.Gate, parsed.Change, pathquote.ShellWord(parsed.Change), pathquote.Quote(workspaceRoot))))
	}
	// Exit 2 of a stop: every check is a read and runs BEFORE gates.yaml is written (DD-02).
	var handOff *handOffPrepared
	if parsed.HandOff {
		handOff, err = prepareHandOff(parsed, loc, changeRoot, roster, digest)
		if err != nil {
			return fail(err)
		}
	}
	record := kickoff.GateRecord{
		Gate:           parsed.Gate,
		Decision:       parsed.Decision,
		Reason:         parsed.Reason,
		ArtifactDigest: digest,
		HandedOff:      handOff != nil,
		CommitReview:   commitReview,
		Actor:          actor,
		RecordedAt:     time.Now().UTC(),
	}
	if handOff != nil {
		// The phase handed over is persisted as computed NOW: the roster moves on afterwards.
		record.HandedOffTo = handOff.handedOffTo
	}
	// Evidence verification runs, and can refuse, BEFORE kickoff.AppendGate:
	// an unconfirmed pr_merged ancestor must append nothing to gates.yaml
	// (task 21.1's second case), which is only guaranteed by checking first.
	if parsed.EvidenceKind != "" {
		evidence, evidenceErr := verifyGateEvidence(workspaceRoot, parsed)
		if evidenceErr != nil {
			return fail(gateCoded(GateCodeEvidenceRefused, evidenceErr))
		}
		record.EvidenceKind = evidence.Kind
		record.EvidenceRef = evidence.Ref
		record.EvidenceBaseRef = evidence.BaseRef
		record.Verified = evidence.Verified
	}
	if handOff != nil || commitReview != "" || in.ExpectedDigest != "" {
		// The "still pending" check is repeated under the ledger lock: two recorders (exit 2 and
		// exit 3, two approvals with a commit review mode, or the dashboard with a seen digest) racing on the same pending gate must
		// leave a single approved record.
		err = kickoff.AppendGateChecked(changeRoot, record, gateStillPendingCheck(parsed, workspaceRoot))
	} else {
		err = kickoff.AppendGate(changeRoot, record)
	}
	if err != nil {
		return fail(err)
	}
	outcome.Record = record
	if actorWarning != "" {
		notice(gateNoticeActorIdentity)(actorWarning)
	}
	emitGateIntegrationEvent(loc, changeRoot, record, notice(gateNoticeIntegrationDispatch))

	// RF-GAT-15: aprobar publica, pero `gate record` solo registra. Calcula (sin ejecutar) la orden de
	// `axiom sdd publish` que sigue al registro, para que el orquestador la ejecute una vez.
	publish := resolveGatePublish(parsed, workspaceRoot, handOff)
	outcome.Publish = publish.target
	outcome.PublishUnavailable = publish.unavailable
	if handOff != nil {
		plan := handOff.plan
		outcome.HandedOffTo = handOff.handedOffTo
		outcome.Delivery = &plan
	}

	if isRoleApply && sealed != nil {
		closedValue := false
		outcome.LastRoleClosed = &closedValue
		if parsed.Decision == kickoff.DecisionApproved {
			closed, pending, evalErr := lastBlockingRoleJustClosed(changeRoot, roster, parsed.Gate)
			if evalErr != nil {
				return fail(evalErr)
			}
			closedValue = closed
			outcome.PendingNonBlocking = pending
		}
	}
	return outcome, nil
}
