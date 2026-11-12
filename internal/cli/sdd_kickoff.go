package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RunSDDKickoff is the CLI entry point for `axiom sdd kickoff <seal|show>`
// (design.md S5.7). It is a thin I/O and formatting adapter: every decision
// (retro-seal inference, single-write sealing, archived-root refusal, role
// file naming) lives in internal/kickoff (task 9.4).
func RunSDDKickoff(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		return renderSDDKickoffHelp(stdout)
	}
	if len(args) == 0 {
		return fmt.Errorf("kickoff requiere un subcomando: seal o show; ejecuta `axiom sdd kickoff --help`")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "seal":
		return runSDDKickoffSeal(rest, stdout)
	case "show":
		return runSDDKickoffShow(rest, stdout)
	default:
		return fmt.Errorf("subcomando %q no reconocido para kickoff; opciones: seal, show; ejecuta `axiom sdd kickoff --help`", sub)
	}
}

func renderSDDKickoffHelp(stdout io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "Uso: axiom sdd kickoff <seal|show> [argumentos]")
	_, _ = fmt.Fprintln(stdout, "  seal --cwd <ruta> --change <nombre> (--execution-style continuous|checkpointed | --from-session-pace interactive|auto) [--docs-publication auto (solo con continuous)] [--role <id>[:blocking|deferred|optional] ...] [--deployment-target local|staging|production] [--json]")
	_, _ = fmt.Fprintln(stdout, "    continuous solo con el rol fullstack: con roles especializados se sella --execution-style checkpointed")
	_, _ = fmt.Fprintln(stdout, "  seal --infer --cwd <ruta> --change <nombre> [--json]   (retro-sellado conservador, REQ-21.4)")
	_, _ = fmt.Fprintln(stdout, "  show --cwd <ruta> --change <nombre> [--json]")
	return nil
}

func runSDDKickoffSeal(args []string, stdout io.Writer) error {
	parsed, err := kickoff.ParseSealArgs(args)
	if err != nil {
		return err
	}
	loc, changeRoot, existed, err := resolveSealChangeRoot(parsed.CWD, parsed.Change)
	if err != nil {
		return err
	}

	var sealing kickoff.Kickoff
	if parsed.Infer {
		// El retrosellado es de un cambio con artefactos: nunca crea la carpeta.
		if !existed {
			return fmt.Errorf("el cambio %q no existe en %q; `seal --infer` retrosella un cambio que ya tiene artefactos y nunca crea la carpeta; para sellar un cambio nuevo ejecuta `axiom sdd kickoff seal` sin `--infer`",
				parsed.Change, changeRoot)
		}
		designPath := filepath.Join(changeRoot, "design.md")
		// loc.Config is the base configuration the resolver already loaded
		// (through the pointer when there is one); it is nil when there is no
		// axiom.yaml or it cannot be loaded, which InferKickoff accepts.
		inferred, note, inferErr := kickoff.InferKickoff(designPath, loc.Config)
		if inferErr != nil {
			if note != "" {
				return fmt.Errorf("%s: %w", note, inferErr)
			}
			return inferErr
		}
		sealing = inferred
	} else {
		sealing = parsed.ToKickoff(parsed.Change)
		// Una declaración `sdd:` inválida, o contradictoria con `workflow.lane`, se
		// rechaza antes de tocar nada (REQ-29.1, REQ-29.12).
		if err := loc.Declarations.SDDErr; err != nil {
			return fmt.Errorf("no se sella el cambio %q: la declaración `sdd:` de axiom.yaml no es válida: %w", parsed.Change, err)
		}
		// Guarda A (REQ-25.38) e instantánea de política git (REQ-25.36): sin --infer.
		if err := applySealBranchPolicy(loc, changeRoot, &sealing); err != nil {
			return err
		}
		// Instantánea de `sdd:` (REQ-29.13): solo un proyecto ISO la lleva, y nunca el
		// retrosellado.
		if sdd := loc.Declarations.SDD; sdd != nil && sdd.ISO {
			sealing.SDD = &kickoff.SDDSnapshot{ISO: true}
		}
	}
	sealing.Change = parsed.Change

	// Defensa en profundidad (DD-01): la regla de continuous también se
	// comprueba sobre el documento ensamblado; el retrosellado queda exento.
	if err := sealing.ValidateForSeal(); err != nil {
		return err
	}

	// La carpeta se crea después de todas las validaciones y justo antes de sellar;
	// si el sellado falla, se retira lo que se creó (REQ-29.12).
	rollback, err := kickoff.PrepareChangeRoot(changeRoot)
	if err != nil {
		return err
	}
	sealed, wrote, err := sealKickoffFile(changeRoot, sealing)
	if err != nil {
		rollback()
		return err
	}

	return renderKickoffSealResult(stdout, parsed.JSON, sealed, wrote)
}

// sealKickoffFile es la costura de prueba sobre kickoff.Seal.
var sealKickoffFile = kickoff.Seal

// resolveSealChangeRoot resuelve la carpeta de un cambio para `seal`, que a
// diferencia de gate y show puede sellar un cambio cuya carpeta aún no existe
// (REQ-29.12). Valida el nombre con gitref.ValidBranchComponent antes de construir
// ninguna ruta, reserva `archive` (colisionaría con changes/archive/), conserva la
// negativa de un cambio archivado y devuelve la ruta y si la carpeta ya existía,
// sin crearla.
func resolveSealChangeRoot(cwd, change string) (workspace.Location, string, bool, error) {
	if err := gitref.ValidBranchComponent(change); err != nil {
		return workspace.Location{}, "", false, fmt.Errorf("--change %q no es un nombre de cambio válido: %w", change, err)
	}
	if change == "archive" {
		return workspace.Location{}, "", false, fmt.Errorf("--change %q es un nombre reservado: openspec/changes/archive/ guarda los cambios archivados; elige otro nombre y ejecuta `axiom sdd kickoff seal --change <nombre>`", change)
	}
	loc, err := workspace.Locate(cwd)
	if err != nil {
		return workspace.Location{}, "", false, err
	}

	changeRoot := filepath.Join(loc.WorkspaceRoot, "openspec", "changes", change)
	existed := isDirectory(changeRoot)
	if !existed {
		if archived, ok := kickoff.ArchivedChangeRoot(loc.WorkspaceRoot, change); ok {
			changeRoot, existed = archived, true
		}
	}
	if err := kickoff.RefuseArchivedRoot(loc.WorkspaceRoot, changeRoot); err != nil {
		return workspace.Location{}, "", false, err
	}
	return loc, changeRoot, existed, nil
}

func runSDDKickoffShow(args []string, stdout io.Writer) error {
	parsed, err := kickoff.ParseShowArgs(args)
	if err != nil {
		return err
	}
	_, changeRoot, err := resolveGovernanceChangeRoot(parsed.CWD, parsed.Change)
	if err != nil {
		return err
	}

	sealed, err := kickoff.Load(changeRoot)
	if err != nil {
		return err
	}

	if parsed.JSON {
		result := kickoffShowResultJSON{Sealed: sealed != nil}
		if sealed != nil {
			payload := toKickoffJSON(*sealed)
			result.Kickoff = &payload
		}
		return encodeJSON(stdout, result)
	}
	if sealed == nil {
		_, err := fmt.Fprintf(stdout, "El cambio %q no tiene kickoff sellado (sin kickoff sellado).\n", parsed.Change)
		return err
	}
	_, err = fmt.Fprintln(stdout, renderKickoffSummaryText(*sealed, false))
	return err
}

func renderKickoffSealResult(stdout io.Writer, asJSON bool, sealed kickoff.Kickoff, wrote bool) error {
	if asJSON {
		return encodeJSON(stdout, kickoffSealResultJSON{Wrote: wrote, Kickoff: toKickoffJSON(sealed)})
	}
	_, err := fmt.Fprintln(stdout, renderKickoffSummaryText(sealed, wrote))
	return err
}

func renderKickoffSummaryText(k kickoff.Kickoff, wrote bool) string {
	prefix := "Kickoff ya estaba sellado"
	if wrote {
		prefix = "Kickoff sellado"
	}
	roles := make([]string, 0, len(k.Config.Roles))
	for _, r := range k.Config.Roles {
		roles = append(roles, fmt.Sprintf("%s:%s", r.Role, r.GatePolicy))
	}
	text := fmt.Sprintf("%s para %q: flow_mode=%s execution_style=%s roles=%v deployment_target=%s sealed_by=%s",
		prefix, k.Change, k.Config.FlowMode, k.Config.ExecutionStyle, roles, k.Lifecycle.DeploymentTarget, k.SealedBy)
	if k.Config.DocsPublication != "" {
		text += fmt.Sprintf(" docs_publication=%s", k.Config.DocsPublication)
	}
	if k.IsISO() {
		text += " iso=true"
	}
	return text
}

func encodeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// kickoffJSON, kickoffRoleJSON, kickoffSealResultJSON, and
// kickoffShowResultJSON are this verb's own --json shape: kickoff.Kickoff
// carries yaml tags, not json ones, so it is translated explicitly rather
// than encoded as-is (which would leak Go field-name casing into the CLI's
// JSON contract).
type kickoffRoleJSON struct {
	Role       string `json:"role"`
	GatePolicy string `json:"gatePolicy"`
	TasksFile  string `json:"tasksFile"`
	VerifyFile string `json:"verifyFile"`
}

type kickoffJSON struct {
	Schema           string            `json:"schema"`
	Change           string            `json:"change"`
	SealedAt         string            `json:"sealedAt"`
	SealedBy         string            `json:"sealedBy"`
	FlowMode         string            `json:"flowMode"`
	ExecutionStyle   string            `json:"executionStyle"`
	HandoffPolicy    string            `json:"handoffPolicy,omitempty"` // deprecated (REQ-26.9): only what the file carries
	Roles            []kickoffRoleJSON `json:"roles"`
	DeploymentTarget string            `json:"deploymentTarget"`
	DocsPublication  string            `json:"docsPublication,omitempty"`
	ISO              bool              `json:"iso,omitempty"` // REQ-29.13: solo un cambio sellado como ISO
}

func toKickoffJSON(k kickoff.Kickoff) kickoffJSON {
	roles := make([]kickoffRoleJSON, 0, len(k.Config.Roles))
	for _, r := range k.Config.Roles {
		roles = append(roles, kickoffRoleJSON{
			Role:       r.Role,
			GatePolicy: string(r.GatePolicy),
			TasksFile:  r.TasksFile,
			VerifyFile: r.VerifyFile,
		})
	}
	return kickoffJSON{
		Schema:           k.Schema,
		Change:           k.Change,
		SealedAt:         k.SealedAt.Format("2006-01-02T15:04:05Z07:00"),
		SealedBy:         k.SealedBy,
		FlowMode:         string(k.Config.FlowMode),
		ExecutionStyle:   string(k.Config.ExecutionStyle),
		HandoffPolicy:    string(k.Config.HandoffPolicy),
		Roles:            roles,
		DeploymentTarget: k.Lifecycle.DeploymentTarget,
		DocsPublication:  string(k.Config.DocsPublication),
		ISO:              k.IsISO(),
	}
}

type kickoffSealResultJSON struct {
	Wrote   bool        `json:"wrote"`
	Kickoff kickoffJSON `json:"kickoff"`
}

type kickoffShowResultJSON struct {
	Sealed  bool         `json:"sealed"`
	Kickoff *kickoffJSON `json:"kickoff,omitempty"`
}
