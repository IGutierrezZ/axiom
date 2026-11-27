package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/multirole"
)

// lastRoleNoticeMarker is the stable phrase RunSDDGate's last-role notice
// always contains. The notice fires exactly once, when an approval closes the
// last active role (RF-RCL-01), and writes no handoff.md. Keeping this as a
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
	_, _ = fmt.Fprintln(stdout, "  record --cwd <ruta> --change <nombre> --gate spec|design|tasks|role-apply:<rol>|integration --decision approved|rejected [--hand-off] [--commit-review notify|stop|stop-before-commit] [--reason \"<texto>\"] [--evidence-kind pr_merged|deployment|attestation] [--commit <sha>] [--base-ref <ref>] [--evidence \"<texto>\"] [--actor <id>] [--json]")
	_, _ = fmt.Fprintln(stdout, "    --actor: si se omite, la identidad git configurada (\"user.name <user.email>\") en la raíz del workspace; sin ella, \"cli\"")
	_, _ = fmt.Fprintln(stdout, "    --commit-review: modo de revisión por commit; solo con --gate tasks --decision approved en un cambio checkpointed; sin la bandera equivale a notify")
	_, _ = fmt.Fprintln(stdout, "  show --cwd <ruta> --change <nombre> [--json]")
	return nil
}

func runSDDGateRecord(args []string, stdout io.Writer) error {
	// `--commit-review` se separa antes del análisis de internal/kickoff (D2): no toca args.go.
	commitReview, args, err := splitCommitReviewFlag(args)
	if err != nil {
		return err
	}
	parsed, err := kickoff.ParseGateRecordArgs(args)
	if err != nil {
		return err
	}
	outcome, err := RecordGate(context.Background(), GateRecordInput{
		CWD: parsed.CWD, Change: parsed.Change, Gate: parsed.Gate, Decision: parsed.Decision,
		Reason: parsed.Reason, Actor: parsed.Actor, HandOff: parsed.HandOff, CommitReview: commitReview,
		EvidenceKind: parsed.EvidenceKind, Commit: parsed.Commit, BaseRef: parsed.BaseRef, Evidence: parsed.Evidence,
	})
	// Los avisos salen antes de cualquier stdout y también cuando el registro falla.
	flushGateNotices(outcome.Notices)
	if err != nil {
		return err
	}

	if parsed.JSON {
		return writeGateRecordJSON(stdout, gateRecordDocument(outcome))
	}
	var handOff *handOffPrepared
	if outcome.Delivery != nil {
		handOff = &handOffPrepared{handedOffTo: outcome.HandedOffTo, plan: *outcome.Delivery}
	}
	if err := writeGateRecordHeader(stdout, parsed, commitReview, handOff); err != nil {
		return err
	}
	if handOff != nil {
		if err := writeHandOffDelivery(stdout, handOff); err != nil {
			return err
		}
	} else if err := writePublishNext(stdout, gatePublish{target: outcome.Publish, unavailable: outcome.PublishUnavailable}); err != nil {
		return err
	}
	// The notice is the only thing the approval of the last blocking role adds
	// (RF-RCL-01): no handoff.md is written any more, the ledger already says every
	// blocking role closed and `status` opens verify from it (RF-RCL-03). It
	// accompanies exit 2 and exit 3 alike, names the deferred and optional roles
	// still open, and is not repeated when one of them closes later.
	if outcome.LastRoleClosed != nil && *outcome.LastRoleClosed {
		if _, err := fmt.Fprintf(stdout, "Aviso: se ha concluido la implementacion del %s del cambio %q; queda abierta la verificacion global de la solucion (RF-RCL-01, RF-RCL-03).%s\n", lastRoleNoticeMarker, parsed.Change, pendingNonBlockingNotice(outcome.PendingNonBlocking)); err != nil {
			return err
		}
	}
	return nil
}

// flushGateNotices vuelca cada aviso recogido por RecordGate en su sumidero de siempre y con el
// mismo prefijo, en el orden en que se produjeron.
func flushGateNotices(notices []GateNotice) {
	for _, n := range notices {
		switch n.Source {
		case gateNoticeBranchGuard:
			_, _ = fmt.Fprintln(writerBranchWarnings, "Aviso: "+n.Text)
		case gateNoticeActorIdentity:
			emitActorIdentityWarning(n.Text)
		case gateNoticeIntegrationDispatch:
			_, _ = fmt.Fprintln(integrationWarnings, "Aviso: "+n.Text)
		}
	}
}

// pendingNonBlockingNotice is the sentence the last-role notice appends when roles
// with policy deferred or optional still have their role-apply gate open: each role
// with its policy and the state of its gate, and that none of them blocks the global
// verification (RF-RCL-01). Empty when there is none, so a change with only blocking
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
// last role with policy blocking (RF-RCL-01): gate belongs to a blocking role and,
// with it, every blocking role-apply gate of the roster is approved in the change's
// ledger. It reuses kickoff.EvaluateGates and kickoff.LastBlockingRoleClosed
// rather than re-deriving gate status by hand. It also returns the roles with
// policy deferred or optional whose gate is still open, which the notice names;
// they never block the global verification (RF-RCL-03).
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
