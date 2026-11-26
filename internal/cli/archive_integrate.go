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
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RunArchiveIntegrate es el punto de entrada de la CLI de
// `axiom archive integrate --odd <feature>` (RF-OLS-02). Es un adaptador fino: resuelve
// el workspace con workspace.Locate (en multirepo, el repositorio de specs, RF-OLS-08),
// consulta la guarda A en el carril ODD, lee odd/tasks/<feature>.md y entrega todo a
// livespec, que es dueño de cada decisión: analiza el documento, comprueba los
// identificadores, compone cada delta sobre openspec/specs/ en memoria y las escribe
// todas o ninguna. No tiene compuertas y no crea artefactos SDD. Cuando el documento lleva
// identificadores provisionales, asigna el número y reescribe el propio documento dentro
// del mismo conjunto atómico (RF-OLS-02); sin provisionales no lo modifica.
//
// El orden de rechazos es configuración (incluidos `baseline:` residual e ISO), rama, documento, secciones, impacto,
// identificadores, composición y contexto técnico; la primera escritura ocurre tras la
// última comprobación. Con `--check` (RF-OLS-10) recorre exactamente la misma secuencia
// salvo la guarda de rama y la escritura: no escribe nada, ni el índice.
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
	// Un `baseline:` residual (RF-TOP-12) y un proyecto ISO (RF-OLS-07) se rechazan
	// antes de leer el documento ODD y sin escribir nada.
	if retired := loc.Declarations.BaselineRetiredErr; retired != nil {
		return retired
	}
	if err := livespec.RefuseISOProject(loc.Declarations.SDD != nil && loc.Declarations.SDD.ISO, loc.ConfigPath, loc.WorkspaceRoot, parsed.feature); err != nil {
		return err
	}
	// RF-RID-08: el patrón de identificadores se lee en vivo de axiom.yaml; inaccesible (bloque
	// `sdd:` inválido, también por `iso`, o configuración ilegible), la integración se niega sin
	// escribir y con la continuación de `axiom workspace validate`.
	pattern, err := projectIDPattern("archive integrate", loc)
	if err != nil {
		return fmt.Errorf("%w; no se ha escrito nada", err)
	}
	command := oddIntegrateCommandLine(parsed.feature, loc.WorkspaceRoot)

	// `--check` solo lee: la guarda de rama existe para proteger al escritor.
	if !parsed.check {
		if err := guardODDWriter(loc.WorkspaceRoot, parsed.feature, "archive integrate --odd"); err != nil {
			return err
		}
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

	registry, err := livespec.LoadRegistry(loc.PlanningHome, livespec.RegistryOptions{Pattern: pattern})
	if err != nil {
		return fmt.Errorf("archive integrate: %w; vuelve a ejecutar `axiom archive integrate --odd %s` cuando se pueda leer; no se ha escrito nada", err, pathquote.ShellWord(parsed.feature))
	}
	plan, err := livespec.PlanIntegration(loc, doc, registry)
	if err != nil {
		return fmt.Errorf("archive integrate: %w; corrige el documento o la spec viva y vuelve a ejecutar `%s`; no se ha escrito nada", err, command)
	}

	plan.Check = parsed.check
	var written []string
	if !parsed.check {
		if written, err = plan.Apply(); err != nil {
			return fmt.Errorf("archive integrate: %w", err)
		}
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
		"Uso: axiom archive integrate --odd <feature> [--path <ruta>] [--check] [--json]",
		"  Integra en la spec viva (openspec/specs/) la sección `## Spec viva` del documento odd/tasks/<feature>.md de la rama actual,",
		"  con la misma composición que el archivado SDD. Cada capacidad lleva su delta en un bloque ```openspec bajo `### Capacidad: <dominio>`.",
		"  Todo o nada: valida y compone todas las capacidades en memoria antes de escribir ninguna. Es idempotente: repetirla no cambia nada.",
		fmt.Sprintf("  Un `### Requirement:` ADDED lleva su `(%s)` en el título; si el identificador ya existe en openspec/specs/ o en la delta de un cambio activo, se rechaza con el siguiente número libre.", reqid.Default().Compact()),
		"  Un documento con `Tipo: bug` declara su impacto: una `## Spec viva` con delta, o una sección `## Impacto en la spec` con la línea exacta `Sin impacto en la spec` y una línea `Motivo: <una línea>`.",
		"  Con `technical_context` en axiom.yaml, rechaza que la rama modifique rutas de referencia (modo reference) o no se pueda determinar su base, e informa de las rutas mantenidas que tocó.",
		"  Un ADDED puede llevar un identificador provisional `(" + reqid.ProvisionalText("<serie>", "<ordinal>") + ")` en lugar del número: la orden asigna el siguiente número libre de la serie y reescribe el propio documento ODD (la valla `openspec` de `## Spec viva` y sus líneas `Cubre:`) en el mismo conjunto atómico, y deja el mapa en la sección `## Identificadores asignados` y en el informe.",
		"  Escribe solo openspec/specs/, si ya existe openspec/INDEX.md y el documento ODD cuando asigna provisionales (sin provisionales no lo modifica); no crea artefactos SDD. Sale con código distinto de cero ante cualquier rechazo, sin escribir nada.",
		"  Solo en la rama de la feature (guarda de rama del carril ODD). En multirepo opera sobre el repositorio de specs.",
		"  --odd <feature>  Nombre de la feature: el documento es odd/tasks/<feature>.md (obligatorio).",
		"  --path <ruta>    Directorio desde el que resolver el workspaceRoot (por defecto, el directorio actual). --cwd es un alias.",
		"  --check          Solo lectura: valida y planifica como la integración real (contexto técnico incluido), sin guarda de rama y sin escribir nada; el informe añade `check`, `requirements` (modifies y creates) y `assignments` (assigned, kept o reassigned).",
		"  --json           Emite el informe axiom.odd-integration/v1.",
	}
	_, err := io.WriteString(stdout, strings.Join(lines, "\n")+"\n")
	return err
}

type oddIntegrateArgs struct {
	feature string
	path    string
	asJSON  bool
	check   bool
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
	check := flags.Bool("check", false, "Solo lectura: planifica sin escribir nada")
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
	parsed := oddIntegrateArgs{feature: feature, path: *path, asJSON: *asJSON, check: *check}
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
	if report.Check {
		b.WriteString("\nSolo lectura (--check): no se ha escrito nada.\n")
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
