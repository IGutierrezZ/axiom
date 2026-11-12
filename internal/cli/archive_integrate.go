package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
	"github.com/IGutierrezZ/axiom/v3/internal/livespec"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RunArchiveIntegrate es el punto de entrada de la CLI de
// `axiom archive integrate --odd <feature>` (REQ-30.2). Es un adaptador fino: resuelve
// el workspace con workspace.Locate (en multirepo, el repositorio de specs, REQ-30.8),
// consulta la guarda A en el carril ODD, lee odd/tasks/<feature>.md y entrega todo a
// livespec, que es dueño de cada decisión: analiza el documento, comprueba los
// identificadores, compone cada delta sobre openspec/specs/ en memoria y las escribe
// todas o ninguna. No tiene compuertas, no crea artefactos SDD y no modifica el
// documento ODD.
//
// El orden de rechazos es configuración, rama, documento, secciones, impacto,
// identificadores, composición y contexto técnico; la primera escritura ocurre tras la
// última comprobación.
func RunArchiveIntegrate(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		return renderArchiveIntegrateHelp(stdout)
	}
	parsed, err := parseODDIntegrateArgs(args)
	if err != nil {
		return err
	}

	start := parsed.path
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	loc, err := workspace.Locate(start)
	if err != nil {
		return err
	}
	// Mientras `axiom sdd integrate` exista, una baseline declarada la tiene el verbo
	// antiguo (solo U1–U5; la retirada de `baseline:` la hace U6).
	if loc.Declarations.Baseline != nil || loc.Declarations.BaselineErr != nil {
		return fmt.Errorf("archive integrate: %s declara `baseline:` y `axiom archive integrate --odd` solo integra en la spec viva de proyectos sin baseline; continúa con `axiom sdd integrate prepare --cwd %s --change <cambio>`",
			pathquote.Quote(loc.ConfigPath), pathquote.Quote(loc.WorkspaceRoot))
	}
	command := oddIntegrateCommandLine(parsed.feature, loc.WorkspaceRoot)

	if err := guardODDWriter(loc.WorkspaceRoot, parsed.feature, "archive integrate --odd"); err != nil {
		return err
	}

	docPath := filepath.Join(loc.WorkspaceRoot, "odd", "tasks", parsed.feature+".md")
	data, err := os.ReadFile(docPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("archive integrate: no existe el documento ODD %s; crea la feature con `axiom branch ensure --lane odd --change %s --cwd %s` y su documento odd/tasks/%s.md, o corrige el nombre de --odd; no se ha escrito nada",
			docPath, pathquote.ShellWord(parsed.feature), pathquote.Quote(loc.WorkspaceRoot), parsed.feature)
	case err != nil:
		return fmt.Errorf("archive integrate: no se puede leer %s: %w; comprueba los permisos del fichero y vuelve a ejecutar `axiom archive integrate --odd %s`; no se ha escrito nada", docPath, err, pathquote.ShellWord(parsed.feature))
	}

	doc, err := livespec.ParseFeatureDoc(parsed.feature, data)
	if err != nil {
		return fmt.Errorf("archive integrate: %w; corrige el documento y vuelve a ejecutar `%s`; no se ha escrito nada", err, command)
	}

	registry, err := livespec.LoadRegistry(loc.PlanningHome, livespec.RegistryOptions{})
	if err != nil {
		return fmt.Errorf("archive integrate: %w; vuelve a ejecutar `axiom archive integrate --odd %s` cuando se pueda leer; no se ha escrito nada", err, pathquote.ShellWord(parsed.feature))
	}
	plan, err := livespec.PlanIntegration(loc, doc, registry)
	if err != nil {
		return fmt.Errorf("archive integrate: %w; corrige el documento o la spec viva y vuelve a ejecutar `%s`; no se ha escrito nada", err, command)
	}

	written, err := plan.Apply()
	if err != nil {
		return fmt.Errorf("archive integrate: %w", err)
	}
	report := plan.Report(written)
	if parsed.asJSON {
		return encodeJSON(stdout, report)
	}
	_, err = io.WriteString(stdout, oddIntegrateReportText(report))
	return err
}

func renderArchiveIntegrateHelp(stdout io.Writer) error {
	lines := []string{
		"Uso: axiom archive integrate --odd <feature> [--path <ruta>] [--json]",
		"  Integra en la spec viva (openspec/specs/) la sección `## Spec viva` del documento odd/tasks/<feature>.md de la rama actual,",
		"  con la misma composición que el archivado SDD. Cada capacidad lleva su delta en un bloque ```openspec bajo `### Capacidad: <dominio>`.",
		"  Todo o nada: valida y compone todas las capacidades en memoria antes de escribir ninguna. Es idempotente: repetirla no cambia nada.",
		"  Un `### Requirement:` ADDED lleva su `(REQ-n.m)` en el título; si el identificador ya existe en openspec/specs/ o en la delta de un cambio activo, se rechaza con el siguiente número libre.",
		"  Un documento con `Tipo: bug` declara su impacto: una `## Spec viva` con delta, o una sección `## Impacto en la spec` con la línea exacta `Sin impacto en la spec` y una línea `Motivo: <una línea>`.",
		"  Con `technical_context` en axiom.yaml, rechaza que la rama modifique rutas de referencia (modo reference) o no se pueda determinar su base, e informa de las rutas mantenidas que tocó.",
		"  Escribe solo openspec/specs/ y, si ya existe, openspec/INDEX.md; no modifica el documento ODD ni crea artefactos SDD. Sale con código distinto de cero ante cualquier rechazo, sin escribir nada.",
		"  Solo en la rama de la feature (guarda de rama del carril ODD). En multirepo opera sobre el repositorio de specs.",
		"  --odd <feature>  Nombre de la feature: el documento es odd/tasks/<feature>.md (obligatorio).",
		"  --path <ruta>    Directorio desde el que resolver el workspaceRoot (por defecto, el directorio actual). --cwd es un alias.",
		"  --json           Emite el informe axiom.odd-integration/v1.",
	}
	_, err := io.WriteString(stdout, strings.Join(lines, "\n")+"\n")
	return err
}

type oddIntegrateArgs struct {
	feature string
	path    string
	asJSON  bool
}

func oddIntegrateCommandLine(feature, root string) string {
	return fmt.Sprintf("axiom archive integrate --odd %s --path %s", pathquote.ShellWord(feature), pathquote.Quote(root))
}

func parseODDIntegrateArgs(args []string) (oddIntegrateArgs, error) {
	flags := flag.NewFlagSet("archive integrate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	odd := flags.String("odd", "", "Feature ODD cuyo documento odd/tasks/<feature>.md se integra")
	path := flags.String("path", "", "Directorio desde el que resolver el workspaceRoot (por defecto, el directorio actual)")
	cwd := flags.String("cwd", "", "Alias de --path")
	asJSON := flags.Bool("json", false, "Emite el informe en JSON")
	if err := flags.Parse(args); err != nil {
		return oddIntegrateArgs{}, fmt.Errorf("archive integrate: %w; ejecuta `axiom archive integrate --help`", err)
	}
	if flags.NArg() != 0 {
		return oddIntegrateArgs{}, fmt.Errorf("archive integrate: argumento inesperado %q; ejecuta `axiom archive integrate --odd <feature>` sin argumentos posicionales", flags.Arg(0))
	}
	feature := *odd
	if strings.TrimSpace(feature) == "" {
		return oddIntegrateArgs{}, errors.New("archive integrate requiere --odd <feature>; ejecuta `axiom archive integrate --odd <feature>` con el nombre del documento odd/tasks/<feature>.md")
	}
	if strings.TrimSpace(feature) != feature || feature == "." || feature == ".." || strings.ContainsAny(feature, `/\:`) {
		return oddIntegrateArgs{}, fmt.Errorf("archive integrate: --odd %q no es un nombre de feature válido (sin rutas ni separadores); indica el nombre del fichero de odd/tasks sin la extensión y ejecuta `axiom archive integrate --odd <feature>`", feature)
	}
	parsed := oddIntegrateArgs{feature: feature, path: *path, asJSON: *asJSON}
	switch {
	case *path != "" && *cwd != "" && !sameContextPath(*path, *cwd):
		return oddIntegrateArgs{}, fmt.Errorf("archive integrate: --path (%s) y --cwd (%s) apuntan a rutas distintas; --cwd es un alias de --path: usa solo --path en `axiom archive integrate --odd %s --path <ruta>`", *path, *cwd, pathquote.ShellWord(feature))
	case *path == "":
		parsed.path = *cwd
	}
	return parsed, nil
}

// guardODDWriter aplica la guarda A al escritor de la spec viva de una feature ODD, que
// vive en la rama de la feature: la ruta que consulta es la raíz del workspace.
func guardODDWriter(workspaceRoot, feature, op string) error {
	warning, err := engine.Default().RequireChangeBranch(context.Background(), engine.WriteTarget{
		ChangeRoot: workspaceRoot,
		Change:     feature,
		Lane:       branch.LaneODD,
		Context:    branch.ContextOrdinary,
		Op:         op,
	})
	if warning != "" {
		_, _ = fmt.Fprintln(writerBranchWarnings, "Aviso: "+warning)
	}
	return err
}

// oddIntegrateReportText es la forma legible del informe.
func oddIntegrateReportText(report livespec.Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Integración ODD de la feature %s (tipo: %s)\n", report.Feature, report.Kind)
	switch report.Impact {
	case livespec.ImpactDelta:
		b.WriteString("Impacto en la spec: delta\n")
	case livespec.ImpactNone:
		fmt.Fprintf(&b, "Impacto en la spec: sin impacto en la spec (motivo: %s)\n", report.Reason)
	default:
		b.WriteString("Impacto en la spec: sin declarar\n")
	}
	if len(report.Capabilities) > 0 {
		b.WriteString("\nCapacidades:\n")
		for _, c := range report.Capabilities {
			fmt.Fprintf(&b, "  - %s: %s (%s)\n", c.Name, c.Outcome, c.Path)
		}
	}
	if len(report.Written) > 0 {
		b.WriteString("\nEscrito:\n")
		for _, path := range report.Written {
			fmt.Fprintf(&b, "  - %s\n", path)
		}
	}
	if context := report.Context; context != nil && context.Declared {
		if len(context.MaintainedTouched) == 0 {
			b.WriteString("\nContexto técnico: la rama no tocó ninguna ruta mantenida.\n")
		} else {
			b.WriteString("\nContexto técnico: la rama tocó rutas mantenidas:\n")
			for _, path := range context.MaintainedTouched {
				fmt.Fprintf(&b, "  - %s\n", path)
			}
		}
	}
	if report.NothingToIntegrate {
		b.WriteString("\nNo había nada que integrar: la spec viva no cambia.\n")
	}
	return b.String()
}
