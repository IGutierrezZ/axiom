package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// lastRoleNoticeMarker is the stable phrase RunSDDGate's last-role notice
// always contains. The notice fires exactly once, when an approval closes the
// last active role (REQ-21.13), and writes no handoff.md. Keeping this as a
// named constant, rather than duplicating a literal string at every call and
// assertion site, is what lets both this file and its test check "did the
// notice fire" without agreeing on prose by coincidence.
const lastRoleNoticeMarker = "ultimo rol activo"

// RunSDDGate is the CLI entry point for `axiom sdd gate <record|show>`
// (design.md S5.7). Like RunSDDKickoff, it is a thin I/O adapter: gate
// vocabulary, roster membership, and last-role detection all delegate to
// internal/kickoff.
func RunSDDGate(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		return renderSDDGateHelp(stdout)
	}
	if len(args) == 0 {
		return fmt.Errorf("gate requiere un subcomando: record o show; ejecuta `axiom sdd gate --help`")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "record":
		return runSDDGateRecord(rest, stdout)
	case "show":
		return runSDDGateShow(rest, stdout)
	default:
		return fmt.Errorf("subcomando %q no reconocido para gate; opciones: record, show; ejecuta `axiom sdd gate --help`", sub)
	}
}

func renderSDDGateHelp(stdout io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "Uso: axiom sdd gate <record|show> [argumentos]")
	_, _ = fmt.Fprintln(stdout, "  record --cwd <ruta> --change <nombre> --gate spec|design|tasks|role-apply:<rol>|integration --decision approved|rejected [--hand-off] [--reason \"<texto>\"] [--evidence-kind pr_merged|deployment|attestation] [--commit <sha>] [--base-ref <ref>] [--evidence \"<texto>\"] [--actor <id>] [--json]")
	_, _ = fmt.Fprintln(stdout, "  show --cwd <ruta> --change <nombre> [--json]")
	return nil
}

func runSDDGateRecord(args []string, stdout io.Writer) error {
	parsed, err := kickoff.ParseGateRecordArgs(args)
	if err != nil {
		return err
	}
	loc, changeRoot, err := resolveGovernanceChangeRoot(parsed.CWD, parsed.Change)
	if err != nil {
		return err
	}
	workspaceRoot := loc.WorkspaceRoot
	parsed = completeGateBaseRef(parsed, loc)
	// Guarda A (REQ-25.19): una sola consulta cubre AppendGate de esta invocación; la compuerta
	// de integración se registra en la rama de archivado (G-10).
	writerContext := branch.ContextOrdinary
	if parsed.Gate == kickoff.GateIntegration {
		writerContext = branch.ContextArchive
	}
	if err := guardChangeWriter(changeRoot, parsed.Change, "sdd gate record", writerContext); err != nil {
		return err
	}

	sealed, err := kickoff.Load(changeRoot)
	if err != nil {
		return err
	}
	if parsed.HandOff {
		if err := requireHandOffStops(parsed, sealed, workspaceRoot); err != nil {
			return err
		}
	}

	var roster []multirole.RoleAssignment
	if sealed != nil {
		roster = rosterFromKickoff(*sealed)
	}
	isRoleApply := isRoleApplyGate(parsed.Gate)
	if isRoleApply {
		if sealed == nil {
			return fmt.Errorf("no se puede registrar %q: el cambio %q no tiene un kickoff sellado con roster de roles; sella el kickoff primero, por ejemplo con `axiom sdd kickoff seal --cwd %s --change %s --execution-style checkpointed`",
				parsed.Gate, parsed.Change, pathquote.Quote(workspaceRoot), pathquote.Quote(parsed.Change))
		}
		if !gateKeyInRoster(parsed.Gate, roster) {
			// refusal:by-design operator-knowledge: only the operator knows which sealed role they meant to target; the message already lists the complete valid roster, and no runnable command can pick a role for them -- the roster is fixed at kickoff time and this verb never re-seals it
			return fmt.Errorf("%q no pertenece al roster sellado de %q (roster: %s)", parsed.Gate, parsed.Change, strings.Join(rosterRoleNames(roster), ", "))
		}
	}

	actor := parsed.Actor
	if actor == "" {
		actor = "cli"
	}
	digest, err := gateArtifactDigest(changeRoot, parsed.Gate, sealed)
	if err != nil {
		return err
	}
	// Exit 2 of a stop: every check is a read and runs BEFORE gates.yaml is written (DD-02).
	var handOff *handOffPrepared
	if parsed.HandOff {
		handOff, err = prepareHandOff(parsed, loc, changeRoot, roster, digest)
		if err != nil {
			return err
		}
	}
	record := kickoff.GateRecord{
		Gate:           parsed.Gate,
		Decision:       parsed.Decision,
		Reason:         parsed.Reason,
		ArtifactDigest: digest,
		HandedOff:      handOff != nil,
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
			return evidenceErr
		}
		record.EvidenceKind = evidence.Kind
		record.EvidenceRef = evidence.Ref
		record.EvidenceBaseRef = evidence.BaseRef
		record.Verified = evidence.Verified
	}
	if handOff != nil {
		// The "still pending" check is repeated under the ledger lock: two recorders (exit 2 and
		// exit 3) racing on the same pending gate must leave a single approved record.
		err = kickoff.AppendGateChecked(changeRoot, record, handOff.notApprovedCheck)
	} else {
		err = kickoff.AppendGate(changeRoot, record)
	}
	if err != nil {
		return err
	}

	// REQ-27.12: aprobar publica, pero `gate record` solo registra. Calcula (sin ejecutar) la orden de
	// `axiom sdd publish` que sigue al registro, para que el orquestador la ejecute una vez.
	publish := resolveGatePublish(parsed, workspaceRoot, handOff)

	if !parsed.JSON {
		if err := writeGateRecordHeader(stdout, parsed, handOff); err != nil {
			return err
		}
		if handOff != nil {
			if err := writeHandOffDelivery(stdout, handOff); err != nil {
				return err
			}
		} else if err := writePublishNext(stdout, publish); err != nil {
			return err
		}
	}

	var lastRoleClosed *bool
	var pendingNonBlocking []kickoff.PendingRole
	if isRoleApply && sealed != nil {
		closedValue := false
		lastRoleClosed = &closedValue
		if parsed.Decision == kickoff.DecisionApproved {
			closed, pending, evalErr := lastBlockingRoleJustClosed(changeRoot, roster, parsed.Gate)
			if evalErr != nil {
				return evalErr
			}
			closedValue = closed
			pendingNonBlocking = pending
			// The notice is the only thing the approval of the last blocking role adds
			// (REQ-21.13): no handoff.md is written any more, the ledger already says every
			// blocking role closed and `status` opens verify from it (REQ-21.15). It
			// accompanies exit 2 and exit 3 alike, names the deferred and optional roles
			// still open, and is not repeated when one of them closes later.
			if closed && !parsed.JSON {
				if _, err := fmt.Fprintf(stdout, "Aviso: se ha concluido la implementacion del %s del cambio %q; queda abierta la verificacion global de la solucion (REQ-21.13, REQ-21.15).%s\n", lastRoleNoticeMarker, parsed.Change, pendingNonBlockingNotice(pending)); err != nil {
					return err
				}
			}
		}
	}

	if parsed.JSON {
		return writeGateRecordJSON(stdout, parsed, handOff, publish, lastRoleClosed, pendingNonBlocking)
	}
	return nil
}

// pendingNonBlockingNotice is the sentence the last-role notice appends when roles
// with policy deferred or optional still have their role-apply gate open: each role
// with its policy and the state of its gate, and that none of them blocks the global
// verification (REQ-21.13). Empty when there is none, so a change with only blocking
// roles prints the notice exactly as before.
func pendingNonBlockingNotice(pending []kickoff.PendingRole) string {
	if len(pending) == 0 {
		return ""
	}
	items := make([]string, 0, len(pending))
	for _, role := range pending {
		state := "pendiente"
		if role.Status == "rejected" {
			state = "rechazado"
		}
		items = append(items, fmt.Sprintf("%s (politica %s, role-apply %s)", role.Role, role.Policy, state))
	}
	return " Roles no bloqueantes aun abiertos, que no bloquean la verificacion global: " + strings.Join(items, ", ") + "."
}

func runSDDGateShow(args []string, stdout io.Writer) error {
	parsed, err := kickoff.ParseShowArgs(args)
	if err != nil {
		return err
	}
	_, changeRoot, err := resolveGovernanceChangeRoot(parsed.CWD, parsed.Change)
	if err != nil {
		return err
	}

	ledger, err := kickoff.LoadGates(changeRoot)
	if err != nil {
		return err
	}
	if len(ledger.Records) == 0 {
		_, err := fmt.Fprintf(stdout, "El cambio %q no tiene ninguna decision de compuerta registrada todavia.\n", parsed.Change)
		return err
	}
	for _, rec := range ledger.Records {
		if _, err := fmt.Fprintf(stdout, "%s: %s (%s) — %s\n", rec.Gate, rec.Decision, rec.Actor, rec.Reason); err != nil {
			return err
		}
	}
	return nil
}

// isRoleApplyGate reports whether key is one of the four fixed gate keys
// (false) or a role-apply:<rol> key (true). It relies only on exported
// kickoff constants — not on a second "role-apply:" string literal — so
// this package never risks constructing or recognising that prefix
// differently than kickoff.RoleApplyGate does.
func isRoleApplyGate(key kickoff.GateKey) bool {
	switch key {
	case kickoff.GateSpec, kickoff.GateDesign, kickoff.GateTasks, kickoff.GateIntegration:
		return false
	default:
		return true
	}
}

func rosterFromKickoff(k kickoff.Kickoff) []multirole.RoleAssignment {
	roster := make([]multirole.RoleAssignment, 0, len(k.Config.Roles))
	for _, r := range k.Config.Roles {
		roster = append(roster, multirole.RoleAssignment{Role: r.Role, GatePolicy: r.GatePolicy})
	}
	return roster
}

func rosterRoleNames(roster []multirole.RoleAssignment) []string {
	names := make([]string, 0, len(roster))
	for _, r := range roster {
		names = append(names, r.Role)
	}
	return names
}

// gateKeyInRoster checks role-apply:<rol> membership by re-deriving each
// roster role's own canonical gate key via kickoff.RoleApplyGate and
// comparing, instead of splitting parsed.Gate's role segment back out by
// hand — the same drift kickoff.RoleApplyGate's own doc comment warns
// against, avoided here without needing a second exported helper in
// internal/kickoff.
func gateKeyInRoster(key kickoff.GateKey, roster []multirole.RoleAssignment) bool {
	for _, r := range roster {
		if kickoff.RoleApplyGate(r.Role) == key {
			return true
		}
	}
	return false
}

// lastBlockingRoleJustClosed reports whether the approval of gate just closed the
// last role with policy blocking (REQ-21.13): gate belongs to a blocking role and,
// with it, every blocking role-apply gate of the roster is approved in the change's
// ledger. It reuses kickoff.EvaluateGates and kickoff.LastBlockingRoleClosed
// rather than re-deriving gate status by hand. It also returns the roles with
// policy deferred or optional whose gate is still open, which the notice names;
// they never block the global verification (REQ-21.15).
//
// The approval of a deferred or optional role never closes anything, so it never
// repeats the notice. With every role blocking, or with the single fullstack role,
// the result is exactly the one LastRoleClosed gave before.
//
// It marks every roster role as already "reached" (RolePending: 0)
// because, at this call site, apply-progress tracking is not the question
// being asked — that belongs to sddstatus (wired in Phase 13), which this
// verb does not read. The question EvaluateGates answers here is narrower
// and fully answerable from the ledger alone: given the sealed roster, has
// every blocking role's role-apply:<role> gate ended up approved? A role whose
// last-recorded decision is a rejection is still reported "rejected" by
// resolveGateStatus regardless of this stand-in RolePending map, so an
// unresolved role still correctly keeps the result false.
func lastBlockingRoleJustClosed(changeRoot string, roster []multirole.RoleAssignment, gate kickoff.GateKey) (bool, []kickoff.PendingRole, error) {
	if len(roster) == 0 {
		return false, nil, nil
	}
	for _, r := range roster {
		if kickoff.RoleApplyGate(r.Role) == gate && !kickoff.IsBlockingRole(r) {
			return false, nil, nil
		}
	}
	ledger, err := kickoff.LoadGates(changeRoot)
	if err != nil {
		return false, nil, err
	}
	rolePending := make(map[string]int, len(roster))
	for _, r := range roster {
		rolePending[r.Role] = 0
	}
	states, err := kickoff.EvaluateGates(kickoff.Inputs{
		Execution:   kickoff.ExecutionCheckpointed,
		Roles:       roster,
		RolePending: rolePending,
		Ledger:      ledger,
	})
	if err != nil {
		return false, nil, err
	}
	if !kickoff.LastBlockingRoleClosed(roster, states) {
		return false, nil, nil
	}
	return true, kickoff.PendingNonBlockingRoles(roster, states), nil
}

// fileHasContent reports whether path exists, is a regular file, and is
// non-empty — the same "has this artifact actually been produced" test
// internal/sddstatus's own hasContent applies, duplicated narrowly here
// because internal/cli has no dependency on internal/sddstatus's
// unexported helpers.
func fileHasContent(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}
