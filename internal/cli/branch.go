package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/branch/engine"
)

// `axiom branch` (INC-25, D-02): la familia de verbos que planifica y consulta las
// ramas y worktrees de un cambio. Esta rebanada trae `status` sobre el motor
// internal/branch/engine (planificador puro, lecturas de git y resultado
// `axiom.branch-result/v1`); los demás verbos llegan con las rebanadas siguientes y la
// ayuda lista solo los verbos que existen (decisión T3). Ningún verbo ejecuta `push`,
// abre un PR ni hace operaciones remotas (REQ-25.1).

const branchHelp = `Uso: axiom branch <subcomando> [argumentos]

Subcomandos:
  status   Estado de las ramas y los worktrees de un cambio (solo lectura)

Ejecuta ` + "`axiom branch <subcomando> --help`" + ` para ver las banderas de cada subcomando.
`

const branchStatusHelp = `Uso: axiom branch status --change <nombre> [--lane sdd|odd] [--cwd <ruta>] [--json]

Muestra, solo con lecturas, el estado de las ramas y los worktrees de un cambio: por
repositorio, su papel, la rama actual, la rama de integración que resuelve la política
de ramas de axiom.yaml (bloques git: y repositories:; sin ellos, main) y la base; por
rama, si existe, su worktree, si tiene cambios sin confirmar, los commits de adelanto y
de atraso respecto de su base (solo con referencias locales) y la rama destino
propuesta para su PR. Un cambio sin ramas creadas se informa como «no iniciado», con
código de salida 0. No ejecuta fetch, pull, push ni ls-remote.

Banderas:
  --change <nombre>   Nombre del cambio (obligatorio)
  --lane sdd|odd      Carril del cambio (por defecto, sdd)
  --cwd <ruta>        Directorio de partida (por defecto, el directorio de trabajo del proceso)
  --json              Imprime el resultado como axiom.branch-result/v1
`

// RunBranch ejecuta `axiom branch <subcomando>` y devuelve el código de salida.
func RunBranch(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, branchHelp)
		return branch.ExitUsage
	}
	switch subcommand, rest := args[0], args[1:]; subcommand {
	case "--help", "-h", "help":
		fmt.Fprint(stdout, branchHelp)
		return branch.ExitReady
	case "status":
		return runBranchStatus(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: subcomando '%s' no reconocido para branch; el disponible es status. Ejecuta `axiom branch --help`.\n", subcommand)
		return branch.ExitUsage
	}
}

func runBranchStatus(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("branch status", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	change := flags.String("change", "", "nombre del cambio")
	lane := flags.String("lane", string(branch.LaneSDD), "carril del cambio: sdd u odd")
	cwd := flags.String("cwd", ".", "directorio de partida")
	asJSON := flags.Bool("json", false, "imprime el resultado como axiom.branch-result/v1")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, branchStatusHelp)
			return branch.ExitReady
		}
		fmt.Fprintf(stderr, "Error: %v. Ejecuta `axiom branch status --help`.\n", err)
		return branch.ExitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: argumento inesperado '%s' para branch status. Ejecuta `axiom branch status --help`.\n", flags.Arg(0))
		return branch.ExitUsage
	}
	if strings.TrimSpace(*change) == "" {
		fmt.Fprintln(stderr, "Error: falta --change <nombre>: branch status muestra el estado de un cambio. Ejecuta `axiom branch status --help`.")
		return branch.ExitUsage
	}
	if *lane != string(branch.LaneSDD) && *lane != string(branch.LaneODD) {
		fmt.Fprintf(stderr, "Error: --lane '%s' no es válido: usa sdd u odd. Ejecuta `axiom branch status --help`.\n", *lane)
		return branch.ExitUsage
	}
	dir, err := filepath.Abs(*cwd)
	if err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo resolver --cwd '%s' (%v). Indica una ruta absoluta con `axiom branch status --cwd <ruta>`.\n", *cwd, err)
		return branch.ExitUsage
	}
	return writeBranchResult(engine.Default().Status(context.Background(), dir, *change, branch.Lane(*lane)), *asJSON, stdout, stderr)
}

// writeBranchResult imprime el resultado, como JSON o como texto en castellano, y
// devuelve el código de salida: 0 solo cuando el estado pedido se alcanzó.
func writeBranchResult(result branch.Result, asJSON bool, stdout, stderr io.Writer) int {
	if asJSON {
		if err := result.WriteJSON(stdout); err != nil {
			fmt.Fprintf(stderr, "Error: no se pudo escribir el resultado: %v. Repite `axiom branch %s --json`.\n", err, result.Operation)
			return branch.ExitRefused
		}
		return result.ExitCode()
	}
	result.WriteText(stdout, stderr)
	return result.ExitCode()
}
