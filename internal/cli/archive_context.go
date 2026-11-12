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
	"github.com/IGutierrezZ/axiom/v3/internal/techcontext"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RunArchiveContext es el punto de entrada de la CLI de
// `axiom archive context <check|prepare|verify|show>` (REQ-30.15 a REQ-30.20). Es un
// adaptador fino: resuelve el workspace con workspace.Locate, toma del status SDD la
// carpeta del cambio, las raíces de edición autorizadas y el almacén de artefactos
// efectivo, y entrega todo a techcontext.Check, Prepare, Verify y ReadTrace, que son
// dueños de cada decisión.
//
// Un rechazo de la integración (un delta con errores, una autorización que falta,
// una verificación fallida) va en el informe, que siempre se imprime; el comando sale
// después con un código distinto de cero. `check` no escribe nada y no exige git ni
// autorización; `prepare` y `verify` escriben solo <cambio>/integration.yaml y se
// consultan con la guarda A en el contexto `archive`; `show` solo lee, también de un
// cambio ya archivado.
func RunArchiveContext(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		return renderArchiveContextHelp(stdout)
	}
	if len(args) == 0 {
		return errors.New("archive context requiere un subcomando: check, prepare, verify o show; ejecuta `axiom archive context --help`")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "check":
		return runArchiveContextCheck(rest, stdout)
	case "prepare":
		return runArchiveContextPrepare(rest, stdout)
	case "verify":
		return runArchiveContextVerify(rest, stdout)
	case "show":
		return runArchiveContextShow(rest, stdout)
	default:
		return fmt.Errorf("subcomando %q no reconocido para archive context; opciones: check, prepare, verify, show; ejecuta `axiom archive context --help`", sub)
	}
}

func renderArchiveContextHelp(stdout io.Writer) error {
	lines := []string{
		"Uso: axiom archive context <check|prepare|verify|show> [argumentos]",
		"  check   --path <ruta> --change <nombre> [--json]",
		"          Valida el delta de contexto del cambio (formato, rutas mantenidas y de referencia, anclas).",
		"          No escribe nada, no exige git ni autorización de edición y sale con código distinto de cero si hay errores.",
		"  prepare --path <ruta> --change <nombre> [--json]",
		"          Valida, comprueba las precondiciones git y la autorización de edición, registra el HEAD de cada repositorio afectado",
		"          y escribe solo <cambio>/integration.yaml. Imprime los trabajos de edición del contexto. Solo en la rama de archivado.",
		"  verify  --path <ruta> --change <nombre> [--json]",
		"          Verificación nativa tras editar: traza, presencia, anclas y alcance por diff.",
		"          Con éxito marca las entradas como integradas y escribe solo <cambio>/integration.yaml; si falla, lista todas las violaciones,",
		"          sale con código 1 y no modifica nada. Solo en la rama de archivado.",
		"  show    --path <ruta> --change <nombre> [--format markdown|json] [--json]",
		"          Muestra la traza del cambio, también de uno ya archivado con prefijo de fecha; con markdown es la sección que se añade al informe de archive.",
		"          Sale con código 1 si el cambio no tiene traza.",
		"  --cwd es un alias de --path; si se dan los dos con valores distintos, el comando falla.",
	}
	_, err := io.WriteString(stdout, strings.Join(lines, "\n")+"\n")
	return err
}

// contextArgs son los argumentos de los cuatro verbos.
type contextArgs struct {
	path   string
	change string
	asJSON bool
	format string // solo show: markdown o json
}

func contextCommandLine(verb string) string {
	return fmt.Sprintf("axiom archive context %s --path <ruta> --change <cambio>", verb)
}

func parseContextArgs(verb string, args []string) (contextArgs, error) {
	flags := flag.NewFlagSet("archive context "+verb, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("path", "", "Directorio desde el que resolver el workspaceRoot (por defecto, el directorio actual)")
	cwd := flags.String("cwd", "", "Alias de --path")
	change := flags.String("change", "", "Nombre del cambio activo")
	asJSON := flags.Bool("json", false, "Emite el informe en JSON")
	var format *string
	if verb == "show" {
		format = flags.String("format", techcontext.TraceFormatMarkdown, "Formato de la traza: markdown o json")
	}
	if err := flags.Parse(args); err != nil {
		return contextArgs{}, fmt.Errorf("archive context %s: %w; ejecuta `axiom archive context --help`", verb, err)
	}
	if flags.NArg() != 0 {
		return contextArgs{}, fmt.Errorf("archive context %s: argumento inesperado %q; ejecuta `%s` sin argumentos posicionales", verb, flags.Arg(0), contextCommandLine(verb))
	}
	if strings.TrimSpace(*change) == "" {
		return contextArgs{}, fmt.Errorf("archive context %s requiere --change <cambio>; ejecuta `%s` (`axiom sdd status` lista los cambios activos)", verb, contextCommandLine(verb))
	}
	if trimmed := strings.TrimSpace(*change); trimmed != *change || *change == "." || *change == ".." || strings.ContainsAny(*change, `/\:`) {
		return contextArgs{}, fmt.Errorf("archive context %s: --change %q no es un nombre de cambio válido (sin rutas ni separadores); indica el nombre de la carpeta de <planning home>/changes y vuelve a ejecutar `%s`", verb, *change, contextCommandLine(verb))
	}
	parsed := contextArgs{path: *path, change: *change, asJSON: *asJSON}
	switch {
	case *path != "" && *cwd != "" && !sameContextPath(*path, *cwd):
		return contextArgs{}, fmt.Errorf("archive context %s: --path (%s) y --cwd (%s) apuntan a rutas distintas; --cwd es un alias de --path: usa solo --path en `%s`", verb, *path, *cwd, contextCommandLine(verb))
	case *path == "":
		parsed.path = *cwd
	}
	if format != nil {
		parsed.format = *format
		if *asJSON {
			parsed.format = techcontext.TraceFormatJSON
		}
		if parsed.format != techcontext.TraceFormatMarkdown && parsed.format != techcontext.TraceFormatJSON {
			return contextArgs{}, fmt.Errorf("archive context show: --format %q no es válido; usa markdown o json en `axiom archive context show --path <ruta> --change <cambio> --format markdown`", parsed.format)
		}
	}
	return parsed, nil
}

// sameContextPath dice si dos rutas dadas en la línea de comandos designan la misma.
func sameContextPath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return absA == absB
}

// resolveContextStatus es la costura con el status SDD: las pruebas la sustituyen
// para no consultar Engram.
var resolveContextStatus = sddstatus.Resolve

// locateContext resuelve el espacio con workspace.Locate. Con un `baseline:`
// residual se niega con el error tipado de su retirada (REQ-30.21).
func locateContext(verb, path, change string) (workspace.Location, error) {
	start := path
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	loc, err := workspace.Locate(start)
	if err != nil {
		return workspace.Location{}, err
	}
	if retired := loc.Declarations.BaselineRetiredErr; retired != nil {
		return workspace.Location{}, retired
	}
	return loc, nil
}

// resolveContextTarget toma del status SDD la carpeta del cambio, las raíces de
// edición autorizadas y el almacén de artefactos efectivo.
func resolveContextTarget(verb string, parsed contextArgs) (techcontext.Target, error) {
	loc, err := locateContext(verb, parsed.path, parsed.change)
	if err != nil {
		return techcontext.Target{}, err
	}
	start := parsed.path
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	prefix := "archive context " + verb
	status, err := resolveContextStatus(sddstatus.ResolveOptions{CWD: start, ChangeName: parsed.change})
	if err != nil {
		return techcontext.Target{}, fmt.Errorf("%s: no se pudo resolver el estado SDD del cambio %q: %w; ejecuta `axiom sdd status %s --cwd %s` para ver la causa", prefix, parsed.change, err, pathquote.ShellWord(parsed.change), pathquote.Quote(loc.WorkspaceRoot))
	}
	if status.Archived != nil {
		return techcontext.Target{}, fmt.Errorf("%s: el cambio %q ya está archivado en %s y no se integra de nuevo; consulta su traza con `axiom archive context show --path %s --change %s`", prefix, parsed.change, status.Archived.Path, pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(parsed.change))
	}

	store := string(status.ArtifactStore)
	changeRoot := ""
	switch {
	case status.ChangeRoot != nil && *status.ChangeRoot != "":
		changeRoot = *status.ChangeRoot
	case store == string(sddstatus.ArtifactStoreEngram) || store == string(sddstatus.ArtifactStoreNone):
		// Sin ficheros no hay carpeta del cambio: `check` y `prepare` informan de la
		// causa de REQ-24.62 si el contexto mantenido está declarado.
		changeRoot = filepath.Join(loc.PlanningHome, "changes", parsed.change)
	default:
		return techcontext.Target{}, fmt.Errorf("%s: no existe el cambio activo %q en %s; comprueba el nombre con `axiom sdd status --cwd %s` y vuelve a ejecutar el comando",
			prefix, parsed.change, filepath.Join(loc.PlanningHome, "changes"), pathquote.Quote(loc.WorkspaceRoot))
	}
	return techcontext.Target{
		Location:      loc,
		Change:        parsed.change,
		ChangeRoot:    changeRoot,
		AllowedRoots:  status.ActionContext.AllowedEditRoots,
		ArtifactStore: store,
	}, nil
}

func contextCommand(verb string, target techcontext.Target) string {
	return fmt.Sprintf("axiom archive context %s --path %s --change %s", verb, pathquote.Quote(target.Location.WorkspaceRoot), pathquote.ShellWord(target.Change))
}

func runArchiveContextCheck(args []string, stdout io.Writer) error {
	parsed, err := parseContextArgs("check", args)
	if err != nil {
		return err
	}
	target, err := resolveContextTarget("check", parsed)
	if err != nil {
		return err
	}
	report, err := techcontext.Check(context.Background(), target)
	if err != nil {
		return fmt.Errorf("archive context check: %w; vuelve a ejecutar `%s`", err, contextCommand("check", target))
	}
	if err := renderContextReport(stdout, parsed.asJSON, "check", report, ""); err != nil {
		return err
	}
	return contextReportError("check", target, report)
}

func runArchiveContextPrepare(args []string, stdout io.Writer) error {
	parsed, err := parseContextArgs("prepare", args)
	if err != nil {
		return err
	}
	target, err := resolveContextTarget("prepare", parsed)
	if err != nil {
		return err
	}
	if err := guardChangeWriter(target.ChangeRoot, target.Change, "archive context prepare", branch.ContextArchive); err != nil {
		return err
	}
	_, report, err := techcontext.Prepare(context.Background(), target)
	if err != nil {
		return fmt.Errorf("archive context prepare: %w; vuelve a ejecutar `%s`", err, contextCommand("prepare", target))
	}
	planPath := ""
	if len(report.Errors) == 0 {
		planPath = filepath.Join(target.ChangeRoot, techcontext.TraceFileName)
	} else {
		// Sin plan no hay nada que editar: los trabajos solo son válidos si prepare
		// dejó el plan, así que no se emiten con errores.
		report.ContextItems = nil
	}
	if err := renderContextReport(stdout, parsed.asJSON, "prepare", report, planPath); err != nil {
		return err
	}
	return contextReportError("prepare", target, report)
}

func runArchiveContextVerify(args []string, stdout io.Writer) error {
	parsed, err := parseContextArgs("verify", args)
	if err != nil {
		return err
	}
	target, err := resolveContextTarget("verify", parsed)
	if err != nil {
		return err
	}
	if err := guardChangeWriter(target.ChangeRoot, target.Change, "archive context verify", branch.ContextArchive); err != nil {
		return err
	}
	report, err := techcontext.Verify(context.Background(), target)
	if err != nil {
		return fmt.Errorf("archive context verify: %w; vuelve a ejecutar `%s`", err, contextCommand("verify", target))
	}
	if parsed.asJSON {
		if err := encodeJSON(stdout, report); err != nil {
			return err
		}
	} else if _, err := io.WriteString(stdout, contextVerifyText(target, report)); err != nil {
		return err
	}
	if report.Passed {
		return nil
	}
	return fmt.Errorf("archive context verify: la verificación de la integración del cambio %q no se superó (%d violación(es), listadas en el informe); el cambio sigue en openspec/changes sin archivar; corrígelas y vuelve a ejecutar `%s`", target.Change, len(report.Violations), contextCommand("verify", target))
}

// contextVerifyText es la forma legible del informe de `verify`.
func contextVerifyText(target techcontext.Target, report techcontext.VerifyReport) string {
	var b strings.Builder
	if report.Passed {
		fmt.Fprintf(&b, "Verificación del contexto técnico del cambio %s: superada\n", report.Change)
		fmt.Fprintf(&b, "Las entradas de contexto quedan integradas en %s.\n", filepath.Join(target.ChangeRoot, techcontext.TraceFileName))
		fmt.Fprintf(&b, "Siguiente paso: mueve el cambio a openspec/changes/archive y añade la traza al informe con `%s --format markdown`\n", contextCommand("show", target))
		return b.String()
	}
	fmt.Fprintf(&b, "Verificación del contexto técnico del cambio %s: no superada (%d violación(es))\n\nViolaciones:\n", report.Change, len(report.Violations))
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

func runArchiveContextShow(args []string, stdout io.Writer) error {
	parsed, err := parseContextArgs("show", args)
	if err != nil {
		return err
	}
	loc, changeRoot, err := resolveContextTraceRoot(parsed)
	if err != nil {
		return err
	}
	planPath := filepath.Join(changeRoot, techcontext.TraceFileName)
	trace, err := techcontext.ReadTrace(planPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("archive context show: el cambio %q no tiene traza de integración (no existe %s): la traza la escribe `axiom archive context prepare`; ejecútalo con `axiom archive context prepare --path %s --change %s` si el cambio sigue activo", parsed.change, planPath, pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(parsed.change))
	case err != nil:
		return fmt.Errorf("archive context show: la traza de integración del cambio %q no se puede leer: %w; comprueba %s o vuelve a preparar con `axiom archive context prepare --path %s --change %s`", parsed.change, err, planPath, pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(parsed.change))
	}
	data, err := techcontext.RenderTrace(trace, parsed.format)
	if err != nil {
		return fmt.Errorf("archive context show: %w; usa `axiom archive context show --path %s --change %s --format markdown`", err, pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(parsed.change))
	}
	_, err = stdout.Write(data)
	return err
}

// resolveContextTraceRoot resuelve el espacio con workspace.Locate y la carpeta del
// cambio para leer su traza: la activa o, si el cambio ya se archivó, la carpeta
// archivada que kickoff.ArchivedChangeRoot encuentra con su nombre exacto o con el
// prefijo de fecha YYYY-MM-DD-. A diferencia de resolveContextTarget, no rechaza un
// cambio archivado: `show` solo lee.
func resolveContextTraceRoot(parsed contextArgs) (workspace.Location, string, error) {
	loc, err := locateContext("show", parsed.path, parsed.change)
	if err != nil {
		return workspace.Location{}, "", err
	}
	if active := filepath.Join(loc.PlanningHome, "changes", parsed.change); isDirectory(active) {
		return loc, active, nil
	}
	if archived, ok := kickoff.ArchivedChangeRoot(loc.WorkspaceRoot, parsed.change); ok {
		return loc, archived, nil
	}
	return workspace.Location{}, "", fmt.Errorf("archive context show: no existe el cambio %q, ni activo en %s ni archivado en %s; comprueba el nombre con `axiom sdd status --cwd %s` y vuelve a ejecutar el comando",
		parsed.change, filepath.Join(loc.PlanningHome, "changes"), filepath.Join(loc.PlanningHome, "changes", "archive"), pathquote.Quote(loc.WorkspaceRoot))
}

// contextReportError convierte los errores del informe en la salida distinta de
// cero del comando. El informe ya se imprimió entero.
func contextReportError(verb string, target techcontext.Target, report techcontext.Report) error {
	if len(report.Errors) == 0 {
		return nil
	}
	return fmt.Errorf("archive context %s: el cambio %q tiene %d error(es) de integración, listados en el informe; corrígelos y vuelve a ejecutar `%s`", verb, target.Change, len(report.Errors), contextCommand(verb, target))
}

func renderContextReport(stdout io.Writer, asJSON bool, verb string, report techcontext.Report, planPath string) error {
	if asJSON {
		return encodeJSON(stdout, report)
	}
	_, err := io.WriteString(stdout, contextReportText(verb, report, planPath))
	return err
}

// contextReportText es la forma legible del informe de `check` y `prepare`.
func contextReportText(verb string, report techcontext.Report, planPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Contexto técnico del cambio %s: %s\n", report.Change, verb)
	if report.Context.Declared {
		fmt.Fprintf(&b, "\nContexto técnico mantenido: presente=%t, decisión=%s\n", report.Context.Present, report.Context.Decision)
		if report.Context.Reason != "" {
			fmt.Fprintf(&b, "  Motivo: %s\n", report.Context.Reason)
		}
		for _, entry := range report.Context.Entries {
			fmt.Fprintf(&b, "  %s %s: %s > %s (%s)\n", entry.ID, entry.Action, entry.Document, entry.Anchor, entry.Reason)
		}
	} else {
		b.WriteString("\nEl proyecto no declara contexto técnico mantenido: no hay nada que integrar.\n")
	}
	if len(report.ContextItems) > 0 {
		b.WriteString("\nEdición del contexto:\n")
		for _, item := range report.ContextItems {
			fmt.Fprintf(&b, "  - %s %s: %s > %s\n", item.ID, item.Action, item.Document, item.Anchor)
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
