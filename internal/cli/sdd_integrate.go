package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/specbaseline"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RunSDDIntegrate is the CLI entry point for
// `axiom sdd integrate <check|prepare|verify|show>` (inc-24, REQ-24.42, REQ-24.51,
// REQ-24.52, REQ-24.54, D-13, design §4.4). It is a thin adapter: it resolves the
// workspace with workspace.Locate, takes the change root, the edit roots and the
// effective artifact store from the SDD status, and hands them to
// specbaseline.Check, Prepare and Verify, which own every decision.
//
// A rejection of the integration (a conflicting body, an undeclared domain, a
// missing authorization, a failed verification) is carried in the report, which is
// always printed; the command then exits with a non-zero code. `check` never
// writes anything and needs neither git nor an authorization; `prepare` and
// `verify` write only <change>/integration.yaml; `show` only reads it, also for a
// change already archived under a dated folder.
func RunSDDIntegrate(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		return renderSDDIntegrateHelp(stdout)
	}
	if len(args) == 0 {
		return errors.New("integrate requiere un subcomando: check, prepare, verify o show; ejecuta `axiom sdd integrate --help`")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "check":
		return runSDDIntegrateCheck(rest, stdout)
	case "prepare":
		return runSDDIntegratePrepare(rest, stdout)
	case "verify":
		return runSDDIntegrateVerify(rest, stdout)
	case "show":
		return runSDDIntegrateShow(rest, stdout)
	default:
		return fmt.Errorf("subcomando %q no reconocido para integrate; opciones: check, prepare, verify, show; ejecuta `axiom sdd integrate --help`", sub)
	}
}

func renderSDDIntegrateHelp(stdout io.Writer) error {
	_, _ = fmt.Fprintln(stdout, "Uso: axiom sdd integrate <check|prepare|verify|show> [argumentos]")
	_, _ = fmt.Fprintln(stdout, "  check   --cwd <ruta> --change <nombre> [--json]")
	_, _ = fmt.Fprintln(stdout, "          Valida los deltas del cambio contra la baseline declarada (dominios, cuerpos, correspondencia de IDs y delta de contexto).")
	_, _ = fmt.Fprintln(stdout, "          No escribe nada, no exige git ni autorización de edición y sale con código distinto de cero si hay errores.")
	_, _ = fmt.Fprintln(stdout, "  prepare --cwd <ruta> --change <nombre> [--supersede] [--superseded-requirements <req1,req2>] [--json]")
	_, _ = fmt.Fprintln(stdout, "          Valida y compone en memoria todo el cambio, comprueba las precondiciones git y la autorización de edición,")
	_, _ = fmt.Fprintln(stdout, "          registra el HEAD de cada repositorio afectado y escribe solo <cambio>/integration.yaml.")
	_, _ = fmt.Fprintln(stdout, "          Imprime el comando de composición (`axiom sdd archive-compose --change ... --in-place`) y los trabajos de edición dirigida.")
	_, _ = fmt.Fprintln(stdout, "  verify  --cwd <ruta> --change <nombre> [--json]")
	_, _ = fmt.Fprintln(stdout, "          Verificación nativa de la integración tras la edición dirigida: traza, composiciones, presencia, anclas, alcance por diff e IDs provisionales.")
	_, _ = fmt.Fprintln(stdout, "          Con éxito marca las entradas como integradas y escribe solo <cambio>/integration.yaml; si falla, lista todas las violaciones, sale con código 1 y no modifica nada.")
	_, _ = fmt.Fprintln(stdout, "  show    --cwd <ruta> --change <nombre> [--format markdown|json]")
	_, _ = fmt.Fprintln(stdout, "          Muestra la traza de integración del cambio, también de uno ya archivado con prefijo de fecha; con markdown es la sección que se añade al informe de archive.")
	_, _ = fmt.Fprintln(stdout, "          Sale con código 1 si el cambio no tiene traza.")
	return nil
}

// integrateArgs son los argumentos de `check`, `prepare`, `verify` y `show`.
type integrateArgs struct {
	cwd     string
	change  string
	asJSON  bool
	format  string // solo show: markdown o json
	options specbaseline.PlanOptions
}

func parseIntegrateArgs(verb string, args []string) (integrateArgs, error) {
	flags := flag.NewFlagSet("integrate "+verb, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	cwd := flags.String("cwd", "", "Directorio desde el que resolver el workspaceRoot (por defecto, el directorio actual)")
	change := flags.String("change", "", "Nombre del cambio activo")
	var asJSON *bool
	var format *string
	if verb == "show" {
		format = flags.String("format", specbaseline.TraceFormatMarkdown, "Formato de la traza: markdown o json")
	} else {
		asJSON = flags.Bool("json", false, "Emite el informe en JSON")
	}
	var supersede *bool
	var superseded *string
	if verb == "prepare" {
		supersede = flags.Bool("supersede", false, "Marca los requisitos de REMOVED como superados en vez de eliminarlos")
		superseded = flags.String("superseded-requirements", "", "Lista de requisitos adicionales separados por comas que se marcan como superados")
	}
	if err := flags.Parse(args); err != nil {
		return integrateArgs{}, fmt.Errorf("integrate %s: %w; ejecuta `axiom sdd integrate --help`", verb, err)
	}
	if flags.NArg() != 0 {
		return integrateArgs{}, fmt.Errorf("integrate %s: argumento inesperado %q; ejecuta `axiom sdd integrate %s --cwd <ruta> --change <cambio>` sin argumentos posicionales", verb, flags.Arg(0), verb)
	}
	if err := checkIntegrationChangeName(verb, *change); err != nil {
		return integrateArgs{}, err
	}
	parsed := integrateArgs{cwd: *cwd, change: *change}
	if asJSON != nil {
		parsed.asJSON = *asJSON
	}
	if format != nil {
		parsed.format = *format
		if parsed.format != specbaseline.TraceFormatMarkdown && parsed.format != specbaseline.TraceFormatJSON {
			return integrateArgs{}, fmt.Errorf("integrate show: --format %q no es válido; usa markdown o json en `axiom sdd integrate show --cwd <ruta> --change <cambio> --format markdown`", parsed.format)
		}
	}
	if verb == "prepare" {
		parsed.options = specbaseline.PlanOptions{Supersede: *supersede, SupersededRequirements: splitSupersededRequirements(*superseded)}
	}
	return parsed, nil
}

// checkIntegrationChangeName rechaza un --change ausente o que no sea un nombre
// simple de cambio: se usa como segmento de ruta de <planning home>/changes.
func checkIntegrationChangeName(verb, change string) error {
	if strings.TrimSpace(change) == "" {
		return fmt.Errorf("integrate %s requiere --change <cambio>; ejecuta `axiom sdd integrate %s --cwd <ruta> --change <cambio>` (`axiom sdd status` lista los cambios activos)", verb, verb)
	}
	if strings.TrimSpace(change) != change || change == "." || change == ".." || strings.ContainsAny(change, `/\:`) {
		return fmt.Errorf("integrate %s: --change %q no es un nombre de cambio válido (sin rutas ni separadores); indica el nombre de la carpeta de <planning home>/changes y vuelve a ejecutar `axiom sdd integrate %s --cwd <ruta> --change <cambio>`", verb, change, verb)
	}
	return nil
}

// resolveIntegrationStatus es la costura con el status SDD: las pruebas la
// sustituyen para no consultar Engram.
var resolveIntegrationStatus = sddstatus.Resolve

// resolveIntegrationTarget resuelve el espacio con workspace.Locate y toma del
// status SDD la carpeta del cambio, las raíces de edición autorizadas y el
// almacén de artefactos efectivo (m8).
func resolveIntegrationTarget(prefix, cwd, change string) (specbaseline.ChangeTarget, error) {
	start := cwd
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	loc, err := workspace.Locate(start)
	if err != nil {
		return specbaseline.ChangeTarget{}, err
	}
	status, err := resolveIntegrationStatus(sddstatus.ResolveOptions{CWD: start, ChangeName: change})
	if err != nil {
		return specbaseline.ChangeTarget{}, fmt.Errorf("%s: no se pudo resolver el estado SDD del cambio %q: %w; ejecuta `axiom sdd status %s --cwd %s` para ver la causa", prefix, change, err, pathquote.ShellWord(change), pathquote.Quote(loc.WorkspaceRoot))
	}
	if status.Archived != nil {
		return specbaseline.ChangeTarget{}, fmt.Errorf("%s: el cambio %q ya está archivado en %s y no se integra de nuevo; consulta `axiom sdd status %s --cwd %s`", prefix, change, status.Archived.Path, pathquote.ShellWord(change), pathquote.Quote(loc.WorkspaceRoot))
	}

	store := string(status.ArtifactStore)
	changeRoot := ""
	switch {
	case status.ChangeRoot != nil && *status.ChangeRoot != "":
		changeRoot = *status.ChangeRoot
	case store == string(sddstatus.ArtifactStoreEngram) || store == string(sddstatus.ArtifactStoreNone):
		// Sin ficheros no hay carpeta del cambio: `check` y `prepare` informan de la
		// causa de REQ-24.62 si la baseline o el contexto mantenido están declarados.
		changeRoot = filepath.Join(loc.PlanningHome, "changes", change)
	default:
		return specbaseline.ChangeTarget{}, fmt.Errorf("%s: no existe el cambio activo %q en %s; comprueba el nombre con `axiom sdd status --cwd %s` y vuelve a ejecutar el comando",
			prefix, change, filepath.Join(loc.PlanningHome, "changes"), pathquote.Quote(loc.WorkspaceRoot))
	}
	return specbaseline.ChangeTarget{
		Location:      loc,
		Change:        change,
		ChangeRoot:    changeRoot,
		AllowedRoots:  status.ActionContext.AllowedEditRoots,
		ArtifactStore: store,
	}, nil
}

func runSDDIntegrateCheck(args []string, stdout io.Writer) error {
	parsed, err := parseIntegrateArgs("check", args)
	if err != nil {
		return err
	}
	target, err := resolveIntegrationTarget("integrate check", parsed.cwd, parsed.change)
	if err != nil {
		return err
	}
	report, err := specbaseline.Check(context.Background(), target)
	if err != nil {
		return fmt.Errorf("integrate check: %w; vuelve a ejecutar `%s`", err, integrateCommand("check", target))
	}
	if err := renderIntegrateReport(stdout, parsed.asJSON, "check", report, ""); err != nil {
		return err
	}
	return integrateReportError("check", target, report)
}

func runSDDIntegratePrepare(args []string, stdout io.Writer) error {
	parsed, err := parseIntegrateArgs("prepare", args)
	if err != nil {
		return err
	}
	target, err := resolveIntegrationTarget("integrate prepare", parsed.cwd, parsed.change)
	if err != nil {
		return err
	}
	if err := guardChangeWriter(target.ChangeRoot, target.Change, "sdd integrate prepare", branch.ContextArchive); err != nil {
		return err
	}
	_, report, err := specbaseline.Prepare(context.Background(), target, parsed.options)
	if err != nil {
		return fmt.Errorf("integrate prepare: %w; vuelve a ejecutar `%s`", err, integrateCommand("prepare", target))
	}
	planPath := ""
	if len(report.Errors) == 0 {
		planPath = filepath.Join(target.ChangeRoot, specbaseline.PlanFileName)
	} else {
		// Sin plan no hay nada que componer ni que editar: el comando de composición
		// solo es válido si prepare dejó el plan, así que no se emite con errores.
		report.ComposeCommand, report.Directed, report.ContextItems, report.Register = "", nil, nil, nil
	}
	if err := renderIntegrateReport(stdout, parsed.asJSON, "prepare", report, planPath); err != nil {
		return err
	}
	return integrateReportError("prepare", target, report)
}

func runSDDIntegrateVerify(args []string, stdout io.Writer) error {
	parsed, err := parseIntegrateArgs("verify", args)
	if err != nil {
		return err
	}
	target, err := resolveIntegrationTarget("integrate verify", parsed.cwd, parsed.change)
	if err != nil {
		return err
	}
	if err := guardChangeWriter(target.ChangeRoot, target.Change, "sdd integrate verify", branch.ContextArchive); err != nil {
		return err
	}
	report, err := specbaseline.Verify(context.Background(), target)
	if err != nil {
		return fmt.Errorf("integrate verify: %w; vuelve a ejecutar `%s`", err, integrateCommand("verify", target))
	}
	if parsed.asJSON {
		if err := encodeJSON(stdout, report); err != nil {
			return err
		}
	} else if _, err := io.WriteString(stdout, integrateVerifyText(target, report)); err != nil {
		return err
	}
	if report.Passed {
		return nil
	}
	return fmt.Errorf("integrate verify: la verificación de la integración del cambio %q no se superó (%d violación(es), listadas en el informe); el cambio sigue en openspec/changes sin archivar; corrígelas y vuelve a ejecutar `%s`", target.Change, len(report.Violations), integrateCommand("verify", target))
}

// integrateVerifyText es la forma legible del informe de `verify`.
func integrateVerifyText(target specbaseline.ChangeTarget, report specbaseline.VerifyReport) string {
	var b strings.Builder
	if report.Passed {
		fmt.Fprintf(&b, "Verificación de la integración del cambio %s: superada\n", report.Change)
		fmt.Fprintf(&b, "Las entradas dirigidas y de contexto quedan integradas en %s.\n", filepath.Join(target.ChangeRoot, specbaseline.PlanFileName))
		fmt.Fprintf(&b, "Siguiente paso: mueve el cambio a openspec/changes/archive y añade la traza al informe con `axiom sdd integrate show --cwd %s --change %s --format markdown`\n", pathquote.Quote(target.Location.WorkspaceRoot), pathquote.ShellWord(target.Change))
		return b.String()
	}
	fmt.Fprintf(&b, "Verificación de la integración del cambio %s: no superada (%d violación(es))\n\nViolaciones:\n", report.Change, len(report.Violations))
	for _, violation := range report.Violations {
		fmt.Fprintf(&b, "  - [%s]", violation.Check)
		if violation.Entry != "" {
			fmt.Fprintf(&b, " entrada %s", violation.Entry)
		}
		if violation.File != "" {
			fmt.Fprintf(&b, " (%s)", violation.File)
		}
		fmt.Fprintf(&b, ": %s\n", violation.Detail)
	}
	return b.String()
}

func runSDDIntegrateShow(args []string, stdout io.Writer) error {
	parsed, err := parseIntegrateArgs("show", args)
	if err != nil {
		return err
	}
	loc, changeRoot, err := resolveTraceRoot("integrate show", parsed.cwd, parsed.change)
	if err != nil {
		return err
	}
	planPath := filepath.Join(changeRoot, specbaseline.PlanFileName)
	plan, err := specbaseline.ReadPlan(planPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("integrate show: el cambio %q no tiene traza de integración (no existe %s): la traza la escribe `axiom sdd integrate prepare`; ejecútalo con `axiom sdd integrate prepare --cwd %s --change %s` si el cambio sigue activo", parsed.change, planPath, pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(parsed.change))
	case err != nil:
		return fmt.Errorf("integrate show: la traza de integración del cambio %q no se puede leer: %w; comprueba %s o vuelve a preparar con `axiom sdd integrate prepare --cwd %s --change %s`", parsed.change, err, planPath, pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(parsed.change))
	}
	data, err := specbaseline.RenderTrace(plan, parsed.format)
	if err != nil {
		return fmt.Errorf("integrate show: %w; usa `axiom sdd integrate show --cwd %s --change %s --format markdown`", err, pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(parsed.change))
	}
	_, err = stdout.Write(data)
	return err
}

// resolveTraceRoot resuelve el espacio con workspace.Locate y la carpeta del
// cambio para leer su traza: la activa o, si el cambio ya se archivó, la carpeta
// archivada que kickoff.ArchivedChangeRoot encuentra con su nombre exacto o con el
// prefijo de fecha YYYY-MM-DD- (D-13 paso 6). A diferencia de resolveIntegrationTarget,
// no rechaza un cambio archivado: `show` solo lee.
func resolveTraceRoot(prefix, cwd, change string) (workspace.Location, string, error) {
	start := cwd
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	loc, err := workspace.Locate(start)
	if err != nil {
		return workspace.Location{}, "", err
	}
	if active := filepath.Join(loc.PlanningHome, "changes", change); isDirectory(active) {
		return loc, active, nil
	}
	if archived, ok := kickoff.ArchivedChangeRoot(loc.WorkspaceRoot, change); ok {
		return loc, archived, nil
	}
	return workspace.Location{}, "", fmt.Errorf("%s: no existe el cambio %q, ni activo en %s ni archivado en %s; comprueba el nombre con `axiom sdd status --cwd %s` y vuelve a ejecutar el comando",
		prefix, change, filepath.Join(loc.PlanningHome, "changes"), filepath.Join(loc.PlanningHome, "changes", "archive"), pathquote.Quote(loc.WorkspaceRoot))
}

func integrateCommand(verb string, target specbaseline.ChangeTarget) string {
	return fmt.Sprintf("axiom sdd integrate %s --cwd %s --change %s", verb, pathquote.Quote(target.Location.WorkspaceRoot), pathquote.ShellWord(target.Change))
}

// integrateReportError convierte los errores del informe en la salida distinta de
// cero del comando. El informe ya se imprimió entero.
func integrateReportError(verb string, target specbaseline.ChangeTarget, report specbaseline.Report) error {
	if len(report.Errors) == 0 {
		return nil
	}
	return fmt.Errorf("integrate %s: el cambio %q tiene %d error(es) de integración, listados en el informe; corrígelos y vuelve a ejecutar `%s`", verb, target.Change, len(report.Errors), integrateCommand(verb, target))
}

func renderIntegrateReport(stdout io.Writer, asJSON bool, verb string, report specbaseline.Report, planPath string) error {
	if asJSON {
		return encodeJSON(stdout, report)
	}
	_, err := io.WriteString(stdout, integrateReportText(verb, report, planPath))
	return err
}

// integrateReportText es la forma legible del informe de `check` y `prepare`.
func integrateReportText(verb string, report specbaseline.Report, planPath string) string {
	var b strings.Builder
	baseline := "baseline por defecto"
	if report.Baseline == specbaseline.BaselineKindProject {
		baseline = "baseline del proyecto"
	}
	fmt.Fprintf(&b, "Integración del cambio %s (%s): %s\n", report.Change, baseline, verb)

	if len(report.Documents) > 0 {
		b.WriteString("\nDocumentos:\n")
	}
	for _, document := range report.Documents {
		fmt.Fprintf(&b, "  - %s (%s): %s\n", document.Domain, document.Integration, document.Path)
		for _, operation := range document.Operations {
			fmt.Fprintf(&b, "      %s: %s\n", operation.Section, strings.Join(operation.IDs, ", "))
		}
		for _, family := range document.Families {
			fmt.Fprintf(&b, "      familia %s: máximo %s, definitivos %s..%s", family.Family, family.Max, family.FirstDefinitive, family.LastDefinitive)
			if family.SuggestedStart != "" {
				fmt.Fprintf(&b, ", inicio provisional sugerido %s", family.SuggestedStart)
			}
			b.WriteString("\n")
			if family.Note != "" {
				fmt.Fprintf(&b, "        %s\n", family.Note)
			}
		}
	}

	if len(report.IDMap) > 0 {
		b.WriteString("\nCorrespondencia de IDs:\n")
		for _, assignment := range report.IDMap {
			fmt.Fprintf(&b, "  %s -> %s (dominio %q)\n", assignment.Provisional, assignment.Definitive, assignment.Domain)
		}
	}

	if report.Context.Declared {
		fmt.Fprintf(&b, "\nContexto técnico mantenido: presente=%t, decisión=%s\n", report.Context.Present, report.Context.Decision)
		if report.Context.Reason != "" {
			fmt.Fprintf(&b, "  Motivo: %s\n", report.Context.Reason)
		}
		for _, entry := range report.Context.Entries {
			fmt.Fprintf(&b, "  %s %s: %s > %s (%s)\n", entry.ID, entry.Action, entry.Document, entry.Anchor, entry.Reason)
		}
	}

	if report.ComposeCommand != "" {
		fmt.Fprintf(&b, "\nComando de composición:\n  %s\n", report.ComposeCommand)
	}
	if len(report.Directed) > 0 {
		b.WriteString("\nEdición dirigida:\n")
		for _, item := range report.Directed {
			fmt.Fprintf(&b, "  - %s (%s): sección %q\n", item.Document, item.Domain, item.Section)
			for _, entry := range item.Entries {
				fmt.Fprintf(&b, "      %s %s\n", entry.Section, entry.Name)
			}
		}
	}
	if len(report.ContextItems) > 0 {
		b.WriteString("\nEdición del contexto:\n")
		for _, item := range report.ContextItems {
			fmt.Fprintf(&b, "  - %s %s: %s > %s\n", item.ID, item.Action, item.Document, item.Anchor)
		}
	}
	if len(report.Register) > 0 {
		b.WriteString("\nSolo registro:\n")
		for _, entry := range report.Register {
			fmt.Fprintf(&b, "  - %s: %s\n", entry.Document, entry.ID)
		}
	}
	if planPath != "" {
		fmt.Fprintf(&b, "\nPlan de integración: %s\n", planPath)
	}
	if len(report.Warnings) > 0 {
		b.WriteString("\nAvisos:\n")
		for _, warning := range report.Warnings {
			fmt.Fprintf(&b, "  - %s\n", warning)
		}
	}
	if len(report.Errors) > 0 {
		b.WriteString("\nErrores:\n")
		for _, problem := range report.Errors {
			fmt.Fprintf(&b, "  - %s\n", problem)
		}
	}
	return b.String()
}
