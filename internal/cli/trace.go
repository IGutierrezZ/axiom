package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
	"github.com/IGutierrezZ/axiom/v3/internal/legacyid"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/reqid"
	"github.com/IGutierrezZ/axiom/v3/internal/reqtrace"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// shortCommitLen es la longitud con la que el texto abrevia un commit.
const shortCommitLen = 12

// RunTrace es `axiom trace <ID> [--json] [--integration-branch <rama>] [--cwd <dir>]`
// (C-03-d): responde, sin escribir nada, qué cambios SDD archivados y qué documentos ODD
// integrados crearon, modificaron, retiraron, renombraron o cubrieron el requisito, con su
// tipo, sus tareas y su commit en la línea `--first-parent` de la rama de integración.
//
// Es un adaptador fino sobre reqtrace.Collect: valida la entrada (sin git), resuelve el
// proyecto y el patrón de identificadores, construye una fuente por carril con su propia raíz
// git y pinta la salida. git solo se ejecuta a través de gitcommon.Run, con la raíz absoluta
// del repositorio de cada fuente, nunca con el directorio del proceso, y solo para leer.
func RunTrace(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		return renderRequirementTraceHelp(stdout)
	}
	// El ID es el posicional y puede ir antes, entre o después de las opciones (decisión 13):
	// flag.Parse se detiene en el primer posicional, así que se vuelve a analizar el resto.
	flags := flag.NewFlagSet("trace", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	cwd := flags.String("cwd", "", "Directorio desde el que resolver el proyecto")
	integrationBranch := flags.String("integration-branch", "", "Rama de integración; por defecto la de axiom.yaml o main")
	asJSON := flags.Bool("json", false, "Salida JSON (esquema "+reqtrace.Schema+")")
	id := ""
	for rest := args; ; {
		if err := flags.Parse(rest); err != nil {
			return fmt.Errorf("trace: %w; ejecuta `axiom trace --help`", err)
		}
		if flags.NArg() == 0 {
			break
		}
		if id != "" {
			return fmt.Errorf("trace: argumento inesperado %q; ejecuta `axiom trace --help`", flags.Arg(0))
		}
		id, rest = flags.Arg(0), flags.Args()[1:]
	}
	if id == "" {
		return errors.New("trace requiere un ID de requisito; ejecuta `axiom trace <ID> [--json] [--integration-branch <rama>] [--cwd <dir>]`")
	}
	branchGiven := false
	flags.Visit(func(f *flag.Flag) { branchGiven = branchGiven || f.Name == "integration-branch" })
	if branchGiven {
		if err := gitref.ValidRefName(*integrationBranch); err != nil {
			return fmt.Errorf("trace: --integration-branch %q no es una referencia válida: %w; ejecuta `axiom trace <ID> --integration-branch <rama>` con un nombre de rama", *integrationBranch, err)
		}
	}

	start := *cwd
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	if !isDirectory(start) {
		return fmt.Errorf("trace: --cwd %q no es un directorio existente; ejecuta `axiom trace <ID> --cwd <dir>` con la carpeta de un proyecto Axiom", start)
	}
	loc, err := workspace.Locate(start)
	if err != nil {
		return fmt.Errorf("trace: %w", err)
	}
	if loc.Origin == workspace.OriginStart {
		return fmt.Errorf("trace: %s no está dentro de un proyecto Axiom (no hay axiom.yaml ni openspec/ por encima); ejecuta `axiom trace <ID> --cwd <proyecto>` con la carpeta de un proyecto", start)
	}
	pattern, err := projectIDPattern("trace", loc)
	if err != nil {
		return err
	}
	// El mapa de IDs heredados se carga antes de ParseID y de cualquier git: un mapa que existe
	// y no es válido falla con su error tipado aunque el ID de consulta también lo sea
	// (RF-TRC-17); sin mapa no hay alias ni aviso.
	legacyMap, err := legacyid.Load(os.DirFS(loc.PlanningHome), legacyid.FileName, pattern)
	if err != nil {
		var mapErr *legacyid.MapError
		if errors.As(err, &mapErr) {
			mapErr.Path = filepath.Join(loc.PlanningHome, legacyid.FileName)
		}
		return fmt.Errorf("trace: %w", err)
	}
	// Sin interfaz con puntero nulo: un *Map nil no se asigna a las interfaces.
	var legacy reqtrace.LegacyResolver
	var aliases reqtrace.AliasResolver = reqtrace.NoAliases{}
	if legacyMap != nil {
		legacy, aliases = legacyMap, legacyMap
	}
	query, err := reqtrace.ParseID(id, pattern, legacy)
	if err != nil {
		return fmt.Errorf("trace: %w; escribe un ID con la forma %s o un provisional con la forma %s; ejecuta `axiom trace --help`",
			err, pattern.Template(), reqid.ProvisionalText("<serie>", "<n>"))
	}

	ref := traceIntegrationBranch(*integrationBranch, nil, loc.Declarations)
	if err := gitref.ValidRefName(ref); err != nil {
		return fmt.Errorf("trace: la rama de integración %q de axiom.yaml no es una referencia válida: %w; ejecuta `axiom trace %s --integration-branch <rama>`", ref, err, id)
	}

	ctx := context.Background()
	sdd, err := traceSDDSource(ctx, loc)
	if err != nil {
		return err
	}
	odd, err := traceODDSource(ctx, loc)
	if err != nil {
		return err
	}

	result, err := reqtrace.Collect(ctx, reqtrace.Input{
		Query: query, Pattern: pattern, Ref: ref, Aliases: aliases, Legacy: legacy, SDD: sdd, ODD: odd,
	})
	var refErr *reqtrace.RefError
	if errors.As(err, &refErr) {
		return fmt.Errorf("trace: la rama de integración %q no se puede leer en el repositorio %s (%v); indica otra con `axiom trace %s --integration-branch <rama>`; trace nunca ejecuta fetch",
			refErr.Ref, refErr.Repo, refErr.Err, id)
	}
	if err != nil {
		return fmt.Errorf("trace: %w", err)
	}
	if *asJSON {
		return encodeJSON(stdout, result)
	}
	return renderRequirementTrace(stdout, result)
}

// traceSDDSource construye la fuente SDD: el planning home, con la raíz git que lo contiene.
func traceSDDSource(ctx context.Context, loc workspace.Location) (reqtrace.Source, error) {
	gitDir := loc.PlanningHome
	if !isDirectory(gitDir) {
		gitDir = loc.WorkspaceRoot
	}
	root, err := gitcommon.TopLevel(ctx, gitDir)
	if err != nil {
		return reqtrace.Source{}, fmt.Errorf("trace: no se puede localizar el repositorio git de %s: %w; la traza se deriva del historial git del planning home: ejecuta `axiom trace <ID> --cwd <dir>` con un proyecto dentro de un repositorio git", pathquote.Quote(gitDir), err)
	}
	prefix, err := planningHomePrefix(root, loc.PlanningHome)
	if err != nil {
		return reqtrace.Source{}, fmt.Errorf("trace: %w", err)
	}
	return reqtrace.Source{Files: os.DirFS(loc.PlanningHome), GitPrefix: prefix, History: gitHistory{root: root}, Repo: root}, nil
}

// traceODDSource construye la fuente ODD: `odd/tasks` de la raíz del workspace, con su propia
// raíz git. Sin la carpeta la fuente está ausente; sin repositorio git se informa sin historia
// y el dominio omite sus documentos con una advertencia.
func traceODDSource(ctx context.Context, loc workspace.Location) (reqtrace.Source, error) {
	dir := filepath.Join(loc.WorkspaceRoot, "odd", "tasks")
	if !isDirectory(dir) {
		return reqtrace.Source{}, nil
	}
	source := reqtrace.Source{Files: os.DirFS(dir)}
	root, err := gitcommon.TopLevel(ctx, dir)
	if errors.Is(err, gitcommon.ErrNotGitRepository) {
		return source, nil
	}
	if err != nil {
		return reqtrace.Source{}, fmt.Errorf("trace: no se puede localizar el repositorio git de %s: %w; ejecuta `axiom trace <ID> --cwd <dir>` desde un proyecto cuyo odd/tasks esté en un repositorio legible", pathquote.Quote(dir), err)
	}
	prefix, err := planningHomePrefix(root, dir)
	if err != nil {
		return source, nil
	}
	source.GitPrefix, source.History, source.Repo = prefix, gitHistory{root: root}, root
	return source, nil
}

// renderRequirementTrace escribe el texto legible de la traza (no es un contrato estable).
func renderRequirementTrace(stdout io.Writer, result reqtrace.Result) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Traza de %s en %s\n", result.ID, result.IntegrationBranch)
	fmt.Fprintf(&b, "IDs resueltos: %s\n", strings.Join(result.ResolvedIDs, ", "))
	hasODD := false
	if len(result.Increments) == 0 {
		fmt.Fprintf(&b, "Ningún cambio SDD archivado ni documento ODD integrado cita %s.\n", result.ID)
	} else {
		fmt.Fprintf(&b, "Incrementos (%d):\n", len(result.Increments))
	}
	for i, inc := range result.Increments {
		hasODD = hasODD || inc.Lane == reqtrace.LaneODD
		commit := fmt.Sprintf("sin commit en la línea --first-parent de %s", result.IntegrationBranch)
		if inc.Integration.Found {
			commit = fmt.Sprintf("%s (%s)", abbreviateCommit(inc.Integration.Commit), anchorLabel(inc.Integration.Anchor))
		}
		fmt.Fprintf(&b, "  %d. %s %s [%s, %s] %s\n", i+1, inc.Lane, inc.Name, inc.Type, inc.Role, commit)
		fmt.Fprintf(&b, "     %s\n", inc.Path)
		if len(inc.Capabilities) > 0 {
			fmt.Fprintf(&b, "     Capacidades: %s\n", strings.Join(inc.Capabilities, ", "))
		}
		for _, task := range inc.Tasks {
			fmt.Fprintf(&b, "     Cubre: %s:%d\n", task.File, task.Line)
		}
		for _, a := range inc.Assignments {
			fmt.Fprintf(&b, "     Asignación: %s -> %s\n", a.Provisional, a.Definitive)
		}
	}
	if len(result.Warnings) > 0 {
		b.WriteString("Advertencias:\n")
		for _, w := range result.Warnings {
			if w.Path != "" {
				fmt.Fprintf(&b, "  - [%s] %s: %s\n", w.Code, w.Path, w.Message)
			} else {
				fmt.Fprintf(&b, "  - [%s] %s\n", w.Code, w.Message)
			}
		}
	}
	if hasODD {
		b.WriteString("Nota: en ODD el commit es el primero de la línea --first-parent que trae odd/tasks/<feature>.md, no necesariamente el que añadió su `## Spec viva`.\n")
	}
	_, err := io.WriteString(stdout, b.String())
	return err
}

func abbreviateCommit(commit string) string {
	if len(commit) > shortCommitLen {
		return commit[:shortCommitLen]
	}
	return commit
}

func anchorLabel(anchor reqtrace.Anchor) string {
	switch anchor {
	case reqtrace.AnchorCode:
		return "código"
	case reqtrace.AnchorArchive:
		return "archivado"
	case reqtrace.AnchorDocument:
		return "documento"
	}
	return string(anchor)
}

func renderRequirementTraceHelp(stdout io.Writer) error {
	template := reqid.Default().Template()
	lines := []string{
		"Uso: axiom trace <ID> [--json] [--integration-branch <rama>] [--cwd <dir>]",
		"Traza un requisito de punta a punta, sin escribir nada: los cambios SDD archivados y los documentos ODD integrados",
		"que lo crearon, modificaron, retiraron, renombraron o cubrieron, con su tipo (increment o bug), sus tareas (líneas",
		"Cubre:) y su commit de integración en la línea --first-parent de la rama de integración.",
		"  <ID>                  Identificador del requisito con la forma del patrón del proyecto (por defecto " + template + ")",
		"                        o un provisional (" + reqid.ProvisionalText("<serie>", "<n>") + "). Provisional y definitivo dan el mismo resultado.",
		"  --integration-branch  Rama de integración; por defecto la de axiom.yaml o main. Si no existe, indica otra; nunca se ejecuta fetch.",
		"  --cwd                 Directorio desde el que resolver el proyecto.",
		"  --json                Salida JSON (" + reqtrace.Schema + "); los errores van siempre por stderr como texto.",
		"Roles: created, modified, removed, renamed y covered (línea Cubre:). Un incremento por rol.",
		"En ODD el commit es el primero de la línea --first-parent que trae el documento, no necesariamente el que añadió su",
		"`## Spec viva`. Un bug cuyo requisito estaba mal especificado (con delta y sin Informe de bug ni Impacto en la spec)",
		"sale como increment. Se leen los ficheros del árbol de trabajo y el anclaje usa la rama de integración: lo archivado o",
		"integrado solo en otra rama no aparece.",
	}
	_, err := fmt.Fprintln(stdout, strings.Join(lines, "\n"))
	return err
}
