package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RunSDDArchiveIntegrate es `axiom sdd archive-integrate --cwd <dir> --change <cambio>
// [--supersede] [--superseded-requirements "<a>, <cap>/<b>"] [--check] [--json]` (REQ-47.8, D4):
// el paso nativo del archivado que cubre el cambio entero. Planifica en un plan único la
// asignación de los identificadores provisionales de todas sus capacidades, compone cada una
// sobre la spec viva (las nuevas, copiadas con sus definitivos), reescribe las deltas, las
// líneas `Cubre:` de las tareas y la matriz de los informes de verify, y deja la traza
// `ids.yaml`, todo en un conjunto atómico. Es un adaptador fino: cada decisión es de livespec.
//
// El orden de rechazos es `baseline:` residual, patrón de identificadores accesible, negativa
// ISO y guarda A (REQ-25.19: el verbo escribe en la carpeta del cambio, contexto `archive`).
// `--check` ejecuta lo mismo salvo la guarda y la escritura, y sobre un cambio ya archivado
// comprueba que cada definitivo de su traza aparece una sola vez en su capacidad.
func RunSDDArchiveIntegrate(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		return renderSDDArchiveIntegrateHelp(stdout)
	}
	flags := flag.NewFlagSet("sdd archive-integrate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	cwd := flags.String("cwd", "", "Directorio desde el que resolver el workspaceRoot")
	change := flags.String("change", "", "Cambio activo cuyas capacidades se integran")
	supersede := flags.Bool("supersede", false, "Marca los REMOVED de todas las capacidades como superados")
	superseded := flags.String("superseded-requirements", "", "Requisitos adicionales a marcar como superados, separados por comas, con el prefijo opcional <capacidad>/")
	check := flags.Bool("check", false, "Solo lectura: planifica y comprueba sin escribir nada")
	asJSON := flags.Bool("json", false, "Salida JSON (esquema "+livespec.ChangeReportSchema+")")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("sdd archive-integrate: %w; ejecuta `axiom sdd archive-integrate --help`", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("sdd archive-integrate: argumento inesperado %q; ejecuta `axiom sdd archive-integrate --cwd <dir> --change <cambio> [--supersede] [--superseded-requirements <a,b>] [--check] [--json]`", flags.Arg(0))
	}
	if strings.TrimSpace(*change) == "" {
		return errors.New("sdd archive-integrate requiere --change; ejecuta `axiom sdd archive-integrate --cwd <dir> --change <cambio> [--supersede] [--superseded-requirements <a,b>] [--check] [--json]`")
	}
	if err := gitref.ValidBranchComponent(*change); err != nil || *change == "archive" {
		reason := "es un nombre reservado"
		if err != nil {
			reason = err.Error()
		}
		return fmt.Errorf("sdd archive-integrate: --change %q no es un nombre de cambio válido: %s; usa el nombre de la carpeta del cambio que proyecta `axiom sdd status --cwd <dir> --json`", *change, reason)
	}
	start := *cwd
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	loc, err := workspace.Locate(start)
	if err != nil {
		return fmt.Errorf("sdd archive-integrate: %w", err)
	}

	// 1. `baseline:` residual (REQ-30.21) y 2. patrón de identificadores accesible (REQ-44.9).
	if retired := loc.Declarations.BaselineRetiredErr; retired != nil {
		return retired
	}
	pattern, err := projectIDPattern("sdd archive-integrate", loc)
	if err != nil {
		return fmt.Errorf("%w; no se ha escrito nada", err)
	}
	retry := fmt.Sprintf("axiom sdd archive-integrate --cwd %s --change %s", pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(*change))

	// Un cambio ya archivado solo admite la comprobación de solo lectura (REQ-21.18).
	if *check && !isDirectory(filepath.Join(loc.PlanningHome, "changes", *change)) {
		if archived, ok := kickoff.ArchivedChangeRoot(loc.WorkspaceRoot, *change); ok {
			return checkArchivedChange(stdout, *asJSON, loc, archived, *change, pattern)
		}
	}

	loc, changeRoot, err := resolveGovernanceChangeRoot(start, *change)
	if err != nil {
		return fmt.Errorf("sdd archive-integrate: %w", err)
	}

	// 3. Negativa ISO (REQ-29.10): un cambio ISO que no cumple la regla del archivado no se integra.
	if findings := sddstatus.ISOArchiveRefusal(loc.WorkspaceRoot, changeRoot); len(findings) > 0 {
		reasons := make([]string, 0, len(findings))
		for _, finding := range findings {
			reasons = append(reasons, finding.Reason())
		}
		return fmt.Errorf("sdd archive-integrate: el cambio ISO %q no puede archivarse y no se integra ni se escribe nada: %s; corrígelo y comprueba con `axiom sdd archive-check --cwd %s --change %s`",
			*change, strings.Join(reasons, " | "), pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(*change))
	}

	// 4. Guarda A (REQ-25.19): el verbo escribe en la carpeta del cambio; `--check` no escribe.
	if !*check {
		if err := guardChangeWriter(changeRoot, *change, "sdd archive-integrate", branch.ContextArchive); err != nil {
			return err
		}
	}

	registry, err := livespec.LoadRegistry(loc.PlanningHome, livespec.RegistryOptions{Pattern: pattern, ExcludeChange: *change})
	if err != nil {
		return fmt.Errorf("sdd archive-integrate: %w; vuelve a ejecutar `%s` cuando se pueda leer; no se ha escrito nada", err, retry)
	}
	plan, err := livespec.PlanChange(loc, *change, livespec.ChangeOptions{Supersede: *supersede, Superseded: splitSupersededRequirements(*superseded), Check: *check}, registry)
	if err != nil {
		return rejectIntegration(stdout, *asJSON, *change, *check, err, fmt.Sprintf("sdd archive-integrate: %%w; corrige el cambio o la spec viva y vuelve a ejecutar `%s`; no se ha escrito nada", retry))
	}
	var written []string
	if !*check {
		if written, err = plan.Apply(); err != nil {
			return rejectIntegration(stdout, *asJSON, *change, *check, err, "sdd archive-integrate: %w")
		}
	}
	return renderChangeReport(stdout, *asJSON, plan.Report(written))
}

// checkArchivedChange es `--check` sobre un cambio archivado: el mapa guardado en su carpeta
// y la spec viva actual, sin escribir nada (D4).
func checkArchivedChange(stdout io.Writer, asJSON bool, loc workspace.Location, archived, change string, pattern reqid.Pattern) error {
	trace, err := livespec.ReadTraceFile(filepath.Join(archived, "ids.yaml"))
	if err != nil {
		return rejectIntegration(stdout, asJSON, change, true, err, "sdd archive-integrate: %w")
	}
	registry, err := livespec.LoadRegistry(loc.PlanningHome, livespec.RegistryOptions{Pattern: pattern, SpecsOnly: true})
	if err != nil {
		return rejectIntegration(stdout, asJSON, change, true, err, "sdd archive-integrate: %w")
	}
	if err := livespec.CheckArchivedTrace(registry, trace); err != nil {
		return rejectIntegration(stdout, asJSON, change, true, err, "sdd archive-integrate: %w")
	}
	return renderChangeReport(stdout, asJSON, livespec.ArchivedReport(change, trace))
}
func rejectIntegration(stdout io.Writer, asJSON bool, change string, check bool, err error, format string) error {
	if asJSON {
		if encodeErr := encodeJSON(stdout, livespec.ErrorReport(change, check, err)); encodeErr != nil {
			return encodeErr
		}
	}
	return fmt.Errorf(format, err)
}

func renderChangeReport(stdout io.Writer, asJSON bool, report livespec.ChangeReport) error {
	if asJSON {
		return encodeJSON(stdout, report)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Integración del cambio SDD %s\n", report.Change)
	if report.Check {
		b.WriteString("Solo lectura (--check): no se ha escrito nada.\n")
	}
	if len(report.Capabilities) > 0 {
		b.WriteString("\nCapacidades:\n")
		for _, c := range report.Capabilities {
			fmt.Fprintf(&b, "  - %s: %s (%s)\n", c.Name, c.Outcome, c.Path)
			for _, s := range c.Skipped {
				fmt.Fprintf(&b, "      omitido, ya integrado: %s %s\n", s.Section, s.Requirement)
			}
		}
	}
	if len(report.Assignments) > 0 {
		b.WriteString("\nIdentificadores asignados:\n")
		for _, a := range report.Assignments {
			fmt.Fprintf(&b, "  - %s → %s (%s, %s)", a.Provisional, a.Definitive, a.Capability, a.State)
			if len(a.Previous) > 0 {
				fmt.Fprintf(&b, ", antes %s", strings.Join(a.Previous, ", "))
			}
			b.WriteString("\n")
		}
	}
	for _, d := range report.Duplicates {
		kind := "informativo"
		if d.Blocking {
			kind = "bloqueante"
		}
		fmt.Fprintf(&b, "Duplicado %s: %s en %s\n", kind, d.ID, strings.Join(d.Capabilities, ", "))
	}
	if len(report.Written) > 0 {
		b.WriteString("\nEscrito:\n")
		for _, path := range report.Written {
			fmt.Fprintf(&b, "  - %s\n", path)
		}
	}
	if report.NothingToIntegrate {
		b.WriteString("\nNo había nada que integrar: la spec viva y el cambio no cambian.\n")
	}
	_, err := io.WriteString(stdout, b.String())
	return err
}

func renderSDDArchiveIntegrateHelp(stdout io.Writer) error {
	lines := []string{
		"Uso: axiom sdd archive-integrate --cwd <dir> --change <cambio> [--supersede] [--superseded-requirements \"<a>, <cap>/<b>\"] [--check] [--json]",
		"  Paso nativo del archivado SDD que cubre el cambio entero, antes de la instantánea y del movimiento de la carpeta.",
		"  Planifica en un plan único la asignación de los identificadores provisionales (`" + reqid.ProvisionalText("<serie>", "<ordinal>") + "`) de todas las capacidades,",
		"  compone cada una sobre la spec viva (una capacidad nueva se copia con sus definitivos; una parcialmente integrada se reanuda con lo pendiente),",
		"  reescribe las deltas, las líneas `Cubre:` de tasks*.md y la columna `Requisito` de la matriz de verify-report*.md, y escribe la traza ids.yaml.",
		"  Todo o nada: valida y compone todo en memoria antes de escribir. Es idempotente: repetirla no cambia nada. Un cambio sin provisionales produce lo mismo que `archive-compose` por dominio.",
		"  Rechaza, en este orden, un `baseline:` residual, un patrón de identificadores inaccesible, un cambio ISO que no puede archivarse y (sin --check) la rama que no es la de archivado.",
		"  Una delta plana `specs/spec.md` y una capacidad nueva con MODIFIED, REMOVED o RENAMED se rechazan con un error tipado.",
		"  --cwd <dir>                       Directorio desde el que resolver el workspaceRoot.",
		"  --change <cambio>                 Cambio activo (obligatorio).",
		"  --supersede                       Marca los REMOVED de todas las capacidades como [SUPERSEDED / DEPRECADO].",
		"  --superseded-requirements <lista> Requisitos adicionales a superar, separados por comas; el nombre completo, incluido el identificador entre paréntesis, de una sola capacidad o con el prefijo `<capacidad>/`.",
		"  --check                           Solo lectura: las mismas comprobaciones y el mismo informe sin escribir (omite solo la guarda de rama). Sobre un cambio archivado comprueba que cada definitivo de su traza aparece una sola vez en su capacidad.",
		"  --json                            Emite el informe " + livespec.ChangeReportSchema + " (capabilities, requirements, assignments, duplicates, written, nothingToIntegrate y error).",
	}
	_, err := io.WriteString(stdout, strings.Join(lines, "\n")+"\n")
	return err
}
