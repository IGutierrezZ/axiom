package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/sddstatus"
)

const workspaceContextHelp = `Uso: axiom workspace context [--cwd <dir>] [--json]

Proyecta el contexto técnico declarado en axiom.yaml (technical_context) sin
seleccionar ningún cambio SDD: rutas absolutas y entry de cada modo.

  --cwd <dir>   Carpeta desde la que se localiza el workspace (por defecto, la actual)
  --json        Imprime el documento axiom.workspace-context

Sin declaración, technicalContext es null y el código de salida es 0. Con una
declaración inválida falla cerrado: technicalContext es null, invalid lleva el
motivo y el código de salida es 1.
`

// RunWorkspaceContext ejecuta `axiom workspace context` y devuelve el código de salida.
func RunWorkspaceContext(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("workspace context", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	cwd := flags.String("cwd", ".", "carpeta desde la que se localiza el workspace")
	asJSON := flags.Bool("json", false, "imprime el resultado como JSON")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, workspaceContextHelp)
			return 0
		}
		fmt.Fprintf(stderr, "Error: %v. Ejecuta `axiom workspace context --help`.\n", err)
		return 2
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: argumento inesperado '%s' para workspace context. Ejecuta `axiom workspace context --help`.\n", flags.Arg(0))
		return 2
	}
	dir, err := filepath.Abs(*cwd)
	if err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo resolver --cwd '%s' (%v).\n", *cwd, err)
		return 2
	}
	result, err := sddstatus.ResolveWorkspaceTechnicalContext(dir)
	if err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo localizar el workspace desde '%s': %v\n", dir, err)
		return 1
	}
	if result.Invalid != "" {
		result.Invalid += ". Ejecuta `axiom workspace validate` y corrige la declaración"
	}
	exit := 0
	if result.Invalid != "" {
		exit = 1
	}
	if *asJSON {
		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "Error: no se pudo serializar el resultado: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(encoded))
		return exit
	}
	if result.Invalid != "" {
		fmt.Fprintf(stderr, "Error: la declaración de technical_context no es válida: %s.\n", result.Invalid)
		return exit
	}
	writeWorkspaceContextText(stdout, result)
	return exit
}

func writeWorkspaceContextText(stdout io.Writer, result sddstatus.WorkspaceTechnicalContext) {
	if result.TechnicalContext == nil {
		fmt.Fprintln(stdout, "Sin contexto técnico declarado en axiom.yaml.")
		return
	}
	fmt.Fprintf(stdout, "Workspace: %s\n", result.WorkspaceRoot)
	for _, mode := range []struct {
		label      string
		projection *sddstatus.ContextModeProjection
	}{
		{"mantenido", result.TechnicalContext.Maintained},
		{"referencia", result.TechnicalContext.Reference},
	} {
		if mode.projection == nil {
			continue
		}
		fmt.Fprintf(stdout, "Contexto técnico %s:\n", mode.label)
		if mode.projection.Entry != "" {
			fmt.Fprintf(stdout, "  entry: %s\n", mode.projection.Entry)
		}
		for _, path := range mode.projection.Paths {
			fmt.Fprintf(stdout, "  ruta: %s\n", path)
		}
	}
}
