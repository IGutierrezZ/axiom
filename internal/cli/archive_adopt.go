package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
	"github.com/IGutierrezZ/axiom/v3/internal/adoption/adoptcheck"
	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// RunArchiveAdopt es el punto de entrada de la CLI de `axiom archive adopt <scan|check>`
// (REQ-30.9 a REQ-30.14). Es un adaptador fino: resuelve el workspaceRoot con
// workspace.Locate (en multirepo, el repositorio de specs, REQ-30.14) y entrega todo a
// el paquete adoption, que es dueño de cada decisión.
//
// `scan` inventaría los orígenes de requisitos previos y solo escribe con `--change`, y
// entonces únicamente <cambio>/adoption-inventory.yaml, tras consultar la guarda A en el
// contexto ordinario (REQ-25.19). `check` verifica el cambio de adopción sin escribir ni
// registrar ninguna decisión de compuerta (REQ-30.13).
func RunArchiveAdopt(args []string, stdout io.Writer) error {
	if hasGovernanceHelpFlag(args) {
		return renderArchiveAdoptHelp(stdout)
	}
	if len(args) == 0 {
		return errors.New("archive adopt requiere un subcomando: scan o check; ejecuta `axiom archive adopt --help`")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "scan":
		return runArchiveAdoptScan(rest, stdout)
	case "check":
		return runArchiveAdoptCheck(rest, stdout)
	default:
		return fmt.Errorf("subcomando %q no reconocido para archive adopt; opciones: scan, check; ejecuta `axiom archive adopt --help`", sub)
	}
}

func renderArchiveAdoptHelp(stdout io.Writer) error {
	lines := []string{
		"Uso: axiom archive adopt <scan|check> [argumentos]",
		"  scan  [--path <ruta>] [--change <cambio>] [--json]",
		"        Inventaría los orígenes de requisitos previos (ficheros .md, .markdown, .txt, .docx, .xlsx y .pdf fuera de openspec/), sus unidades",
		"        (encabezados H1–H3 con su ancla), los identificadores ajenos (RF-120…) y los candidatos a technical_context. Solo lee, es determinista",
		"        y no exige openspec/. Con --change escribe únicamente <cambio>/adoption-inventory.yaml, solo en la rama del cambio; sin orígenes informa y sale con 0.",
		"  check --change <cambio> [--path <ruta>] [--json]",
		"        Verifica el cambio de adopción sin escribir ni registrar compuertas: solo ADDED, (REQ-n.m) únicos de una serie nueva, escenarios, Origen: que resuelven",
		"        y cobertura del inventario (cada unidad migrada con Origen: o descartada con motivo en `## Descartado` de adoption.md). Lista todos los fallos y sale con 1 si hay alguno.",
		"        La fidelidad semántica de la migración la aprueba un humano en la compuerta `spec`; esta orden no la juzga.",
		"  --path <ruta>  Directorio desde el que resolver el workspaceRoot (por defecto, el directorio actual). --cwd es un alias.",
		"  --json         Emite el informe en JSON.",
		"  En multirepo opera sobre el repositorio de especificaciones; en monorepo-embedded, sobre el repositorio actual.",
	}
	_, err := io.WriteString(stdout, strings.Join(lines, "\n")+"\n")
	return err
}

type adoptArgs struct {
	path   string
	change string
	asJSON bool
}

func adoptCommandLine(verb string) string {
	if verb == "check" {
		return "axiom archive adopt check --change <cambio> --path <ruta>"
	}
	return "axiom archive adopt scan --path <ruta> [--change <cambio>]"
}

func parseAdoptArgs(verb string, args []string) (adoptArgs, error) {
	flags := flag.NewFlagSet("archive adopt "+verb, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	path := flags.String("path", "", "Directorio desde el que resolver el workspaceRoot (por defecto, el directorio actual)")
	cwd := flags.String("cwd", "", "Alias de --path")
	change := flags.String("change", "", "Nombre del cambio de adopción")
	asJSON := flags.Bool("json", false, "Emite el informe en JSON")
	if err := flags.Parse(args); err != nil {
		return adoptArgs{}, fmt.Errorf("archive adopt %s: %w; ejecuta `axiom archive adopt --help`", verb, err)
	}
	if flags.NArg() != 0 {
		return adoptArgs{}, fmt.Errorf("archive adopt %s: argumento inesperado %q; ejecuta `%s` sin argumentos posicionales", verb, flags.Arg(0), adoptCommandLine(verb))
	}
	if verb == "check" && strings.TrimSpace(*change) == "" {
		return adoptArgs{}, fmt.Errorf("archive adopt check requiere --change <cambio>; ejecuta `%s` (`axiom sdd status` lista los cambios activos)", adoptCommandLine(verb))
	}
	if *change != "" {
		if trimmed := strings.TrimSpace(*change); trimmed != *change || *change == "." || *change == ".." || strings.ContainsAny(*change, `/\:`) {
			return adoptArgs{}, fmt.Errorf("archive adopt %s: --change %q no es un nombre de cambio válido (sin rutas ni separadores); indica el nombre de la carpeta de <planning home>/changes y vuelve a ejecutar `%s`", verb, *change, adoptCommandLine(verb))
		}
	}
	parsed := adoptArgs{path: *path, change: *change, asJSON: *asJSON}
	switch {
	case *path != "" && *cwd != "" && !sameContextPath(*path, *cwd):
		return adoptArgs{}, fmt.Errorf("archive adopt %s: --path (%s) y --cwd (%s) apuntan a rutas distintas; --cwd es un alias de --path: usa solo --path en `%s`", verb, *path, *cwd, adoptCommandLine(verb))
	case *path == "":
		parsed.path = *cwd
	}
	return parsed, nil
}

// locateAdopt resuelve el workspaceRoot. No exige openspec/ ni axiom.yaml: sin marcador,
// el directorio de partida es la raíz.
func locateAdopt(verb string, parsed adoptArgs) (workspace.Location, error) {
	start := parsed.path
	if strings.TrimSpace(start) == "" {
		start = "."
	}
	loc, err := workspace.Locate(start)
	if err != nil {
		return workspace.Location{}, fmt.Errorf("archive adopt %s: %w; vuelve a ejecutar `%s`", verb, err, adoptCommandLine(verb))
	}
	return loc, nil
}

// adoptScanReport es la salida JSON de `scan`: el inventario y, con --change, el fichero
// escrito.
type adoptScanReport struct {
	adoption.Inventory
	Written string `json:"written,omitempty"`
}

func runArchiveAdoptScan(args []string, stdout io.Writer) error {
	parsed, err := parseAdoptArgs("scan", args)
	if err != nil {
		return err
	}
	loc, err := locateAdopt("scan", parsed)
	if err != nil {
		return err
	}
	inv, err := adoption.Scan(loc.WorkspaceRoot)
	if err != nil {
		return fmt.Errorf("archive adopt scan: %w; comprueba el directorio y vuelve a ejecutar `axiom archive adopt scan --path %s`", err, pathquote.Quote(loc.WorkspaceRoot))
	}

	written := ""
	if parsed.change != "" && len(inv.Sources) > 0 {
		changeRoot := filepath.Join(loc.PlanningHome, "changes", parsed.change)
		if err := guardChangeWriter(changeRoot, parsed.change, "archive adopt scan", branch.ContextOrdinary); err != nil {
			return err
		}
		data, err := inv.Marshal()
		if err != nil {
			return fmt.Errorf("archive adopt scan: %w; no se ha escrito nada; vuelve a ejecutar `axiom archive adopt scan --path %s --change %s`", err, pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(parsed.change))
		}
		target := filepath.Join(changeRoot, adoption.InventoryFileName)
		if err := writeFileAtomic(target, data); err != nil {
			return fmt.Errorf("archive adopt scan: %w; comprueba los permisos y vuelve a ejecutar `axiom archive adopt scan --path %s --change %s`", err, pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(parsed.change))
		}
		written = target
	}

	if parsed.asJSON {
		return encodeJSON(stdout, adoptScanReport{Inventory: inv, Written: written})
	}
	_, err = io.WriteString(stdout, adoptScanText(loc.WorkspaceRoot, inv, parsed.change, written))
	return err
}

// writeFileAtomic escribe data en target creando la carpeta si falta, con un temporal en
// la misma carpeta y un renombrado: nunca deja un inventario a medias.
func writeFileAtomic(target string, data []byte) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("no se puede crear %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".adoption-inventory-*.tmp")
	if err != nil {
		return fmt.Errorf("no se puede escribir en %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("no se puede escribir %s: %w", target, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("no se puede escribir %s: %w", target, err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("no se puede escribir %s: %w", target, err)
	}
	return nil
}

func adoptScanText(root string, inv adoption.Inventory, change, written string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Inventario de adopción de %s\n", root)
	if len(inv.Sources) == 0 {
		b.WriteString("\nNo hay orígenes candidatos de requisitos fuera de openspec/: no hay nada que migrar.\n")
	} else {
		fmt.Fprintf(&b, "\nOrígenes (%d):\n", len(inv.Sources))
		for _, src := range inv.Sources {
			fmt.Fprintf(&b, "  - %s (%s, %d unidad(es))\n", src.Path, src.Kind, len(src.Units))
			for _, unit := range src.Units {
				ref := "(fichero entero)"
				if unit.Anchor != "" {
					ref = "#" + unit.Anchor
				}
				fmt.Fprintf(&b, "      %s  %s", ref, unit.Title)
				if len(unit.ForeignIDs) > 0 {
					fmt.Fprintf(&b, "  [%s]", strings.Join(unit.ForeignIDs, ", "))
				}
				b.WriteString("\n")
			}
		}
	}
	if len(inv.ContextCandidates) > 0 {
		b.WriteString("\nCandidatos a technical_context (propuestos; axiom.yaml no se modifica):\n")
		for _, c := range inv.ContextCandidates {
			fmt.Fprintf(&b, "  - %s → %s (%s)\n", c.Path, c.Mode, c.Reason)
		}
	}
	if len(inv.Warnings) > 0 {
		b.WriteString("\nAvisos:\n")
		for _, w := range inv.Warnings {
			fmt.Fprintf(&b, "  - %s\n", w)
		}
	}
	switch {
	case written != "":
		fmt.Fprintf(&b, "\nInventario escrito: %s\n", written)
		fmt.Fprintf(&b, "Siguiente paso: redacta la delta de adopción y comprueba el cambio con `axiom archive adopt check --path %s --change %s`\n", pathquote.Quote(root), pathquote.ShellWord(change))
	case change != "":
		b.WriteString("\nNo se ha escrito ningún fichero: sin orígenes no hay inventario que guardar.\n")
	case len(inv.Sources) > 0:
		fmt.Fprintf(&b, "\nNo se ha escrito ningún fichero. Para guardar el inventario en un cambio: `axiom archive adopt scan --path %s --change <cambio>`\n", pathquote.Quote(root))
	}
	return b.String()
}

func runArchiveAdoptCheck(args []string, stdout io.Writer) error {
	parsed, err := parseAdoptArgs("check", args)
	if err != nil {
		return err
	}
	loc, err := locateAdopt("check", parsed)
	if err != nil {
		return err
	}
	report, err := adoptcheck.Check(loc, parsed.change)
	if err != nil {
		return fmt.Errorf("archive adopt check: %w; vuelve a ejecutar `axiom archive adopt check --path %s --change %s`", err, pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(parsed.change))
	}
	if parsed.asJSON {
		if err := encodeJSON(stdout, report); err != nil {
			return err
		}
	} else if _, err := io.WriteString(stdout, adoptCheckText(report)); err != nil {
		return err
	}
	if report.Passed {
		return nil
	}
	return fmt.Errorf("archive adopt check: el cambio %q no supera la verificación (%d fallo(s), listados en el informe); corrígelos y vuelve a ejecutar `axiom archive adopt check --path %s --change %s`",
		parsed.change, len(report.Failures), pathquote.Quote(loc.WorkspaceRoot), pathquote.ShellWord(parsed.change))
}

func adoptCheckText(report adoptcheck.CheckReport) string {
	var b strings.Builder
	if report.Passed {
		fmt.Fprintf(&b, "Verificación de la adopción del cambio %s: superada\n", report.Change)
		b.WriteString("Un humano debe aprobar en la compuerta `spec` la fidelidad de la migración; esta verificación solo comprueba forma y cobertura.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "Verificación de la adopción del cambio %s: no superada (%d fallo(s))\n\nFallos:\n", report.Change, len(report.Failures))
	for _, f := range report.Failures {
		fmt.Fprintf(&b, "  - [%s] %s: %s\n", f.Rule, f.Subject, f.Detail)
	}
	return b.String()
}
