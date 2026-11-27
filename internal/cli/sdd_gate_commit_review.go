package cli

import (
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// commitReviewFlag es la bandera de `axiom sdd gate record` que persiste el modo de revisión por
// commit (RF-GAT-20). Se separa aquí de los argumentos, antes de kickoff.ParseGateRecordArgs, para
// no tocar internal/kickoff/args.go (decisión D2 del diseño).
const commitReviewFlag = "--commit-review"

// commitReviewVocabulary es el vocabulario cerrado, tal como lo muestran los mensajes.
func commitReviewVocabulary() string {
	modes := kickoff.CommitReviewModes()
	names := make([]string, 0, len(modes))
	for _, mode := range modes {
		names = append(names, string(mode))
	}
	return strings.Join(names, ", ")
}

// splitCommitReviewFlag quita `--commit-review <modo>` de args y devuelve el modo ("" si la bandera
// no está) y el resto de los argumentos sin cambios y en su orden. Rechaza la bandera repetida, sin
// valor, con un valor fuera del vocabulario (incluidas la cadena vacía y las variantes de
// mayúsculas, RF-GAT-19). No toca args.
//
// La separación previa no es ambigua: `requireFlagValue` rechaza cualquier valor que empiece por
// `-`, así que `--commit-review` nunca puede ser el valor de otra bandera.
func splitCommitReviewFlag(args []string) (kickoff.CommitReviewMode, []string, error) {
	var mode kickoff.CommitReviewMode
	seen := false
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] != commitReviewFlag {
			rest = append(rest, args[i])
			continue
		}
		if seen {
			return "", nil, fmt.Errorf("`%s` se indicó más de una vez; repite `axiom sdd gate record` con un único `%s <%s>`", commitReviewFlag, commitReviewFlag, strings.ReplaceAll(commitReviewVocabulary(), ", ", "|"))
		}
		seen = true
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
			return "", nil, fmt.Errorf("`%s` requiere un valor; repite `axiom sdd gate record` con uno de: %s", commitReviewFlag, commitReviewVocabulary())
		}
		value, err := parseCommitReviewMode(args[i+1])
		if err != nil {
			return "", nil, err
		}
		mode = value
		i++
	}
	return mode, rest, nil
}

// parseCommitReviewMode valida un valor de `--commit-review` contra el vocabulario cerrado
// (RF-GAT-19) con el mismo mensaje que la CLI; la comparten la CLI y el dashboard.
func parseCommitReviewMode(value string) (kickoff.CommitReviewMode, error) {
	mode := kickoff.CommitReviewMode(value)
	for _, candidate := range kickoff.CommitReviewModes() {
		if mode == candidate {
			return mode, nil
		}
	}
	return "", fmt.Errorf("`%s` desconocido %q; repite `axiom sdd gate record` con uno de: %s", commitReviewFlag, value, commitReviewVocabulary())
}

// validateCommitReviewShape es la comprobación de forma de `--commit-review` (RF-GAT-20, D3), antes
// de tocar el disco: solo con `--gate tasks`, solo con `--decision approved` y nunca con
// `--evidence-kind`. Cada rechazo nombra la orden que sigue siendo válida. Sin modo no hace nada.
func validateCommitReviewShape(parsed kickoff.GateRecordArgs, mode kickoff.CommitReviewMode) error {
	if mode == "" {
		return nil
	}
	if parsed.Gate != kickoff.GateTasks {
		return fmt.Errorf("`%s` solo se admite en la compuerta tasks, no en %q; repite la orden sin la bandera: `axiom sdd gate record --change %s --gate %s --decision %s`",
			commitReviewFlag, parsed.Gate, parsed.Change, parsed.Gate, parsed.Decision)
	}
	if parsed.Decision != kickoff.DecisionApproved {
		return fmt.Errorf("`%s` solo acompaña a `--decision approved`; para iterar usa `axiom sdd gate record --change %s --gate tasks --decision rejected --reason <motivo>` sin la bandera",
			commitReviewFlag, parsed.Change)
	}
	if parsed.EvidenceKind != "" {
		return fmt.Errorf("`%s` no admite `--evidence-kind`: la evidencia es solo de la compuerta integration; repite `axiom sdd gate record --change %s --gate tasks --decision approved %s <%s>` sin evidencia",
			commitReviewFlag, parsed.Change, commitReviewFlag, strings.ReplaceAll(commitReviewVocabulary(), ", ", "|"))
	}
	return nil
}

// requireCommitReviewStops es la comprobación de estado de `--commit-review` (RF-GAT-20, D3): el
// cambio debe estar sellado `checkpointed`. En uno continuo, o sin sellar, no hay puerta `tasks` que
// proyecte el modo, y se rechaza nombrando la aprobación sin la bandera.
func requireCommitReviewStops(parsed kickoff.GateRecordArgs, sealed *kickoff.Kickoff, workspaceRoot string) error {
	if sealed != nil && sealed.Config.ExecutionStyle == kickoff.ExecutionCheckpointed {
		return nil
	}
	return gateCoded(GateCodeNoStops, fmt.Errorf("el cambio %q no tiene paradas: `%s` solo existe en cambios sellados con `--execution-style checkpointed`, y este es continuo o no está sellado; registra la decisión sin la bandera con `%s`",
		parsed.Change, commitReviewFlag, gateApprovedInvocation(parsed, workspaceRoot)))
}
