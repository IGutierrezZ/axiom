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
	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
)

// `axiom branch scope` (C-02-b, RF-BRA-01, RF-BRA-02, RF-BRA-35 a RF-BRA-38, design D1 y D3): calcula,
// solo con lecturas y sin red, el alcance conjunto de un cambio (los commits propios de su rama por
// repositorio de código) y la oferta de verificación conjunta. El cálculo vive en el motor
// (internal/branch/engine); aquí solo están las banderas, la ayuda y la salida.

const branchScopeHelp = `Uso: axiom branch scope --change <nombre> [--cwd <ruta>] [--json]

Calcula, solo con lecturas, el alcance conjunto de un cambio: por repositorio de código, la rama
del cambio, el punto de partida estable, los commits propios, los commits excluidos (de otros
cambios que entraron en la rama o en la integración) y los ficheros que tocan, sin la carpeta
openspec/changes/<cambio>/ del repositorio de specs. Indica además si la rama ya está fusionada
en la rama de integración y resume la oferta de verificación conjunta: offerable,
pending_merge, unverifiable o nothing_to_verify.

Exige un cambio con kickoff sellado y con la instantánea de política git del sellado: sin
kickoff rechaza con kickoff_not_sealed; en el carril odd, o con un kickoff sin instantánea, con
scope_not_applicable. Un repositorio de código cuya rama ya no existe, o cuya base o integración
faltan, no se omite: se declara unverifiable (branch_missing, base_missing o
fork_point_undetermined) con el git fetch origin que la persona debe autorizar. El repositorio
de specs que se trabaja in situ es exento.

Solo lee commits: no ejecuta fetch, pull, push ni ls-remote, no refresca el índice, no toca el
árbol de trabajo y no escribe ningún fichero. Por eso no admite --fetch.

Banderas:
  --change <nombre>   Nombre del cambio (obligatorio)
  --cwd <ruta>        Directorio de partida (por defecto, el directorio de trabajo del proceso)
  --json              Imprime el resultado como axiom.branch-scope/v1

Código de salida: 0 si el alcance se calculó (aunque haya repositorios unverifiable o ramas sin
fusionar), 1 ante un rechazo tipado o un error de git, 2 ante un error de uso.
`

// runBranchScope ejecuta `axiom branch scope`: --change es obligatorio, no puede estar en blanco y
// debe ser un nombre de cambio válido; el FlagSet no declara --fetch ni --lane, de modo que ambas
// son un error de uso (2) antes de leer ningún repositorio.
func runBranchScope(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("branch scope", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	change := flags.String("change", "", "nombre del cambio")
	cwd := flags.String("cwd", ".", "directorio de partida")
	asJSON := flags.Bool("json", false, "imprime el resultado como axiom.branch-scope/v1")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, branchScopeHelp)
			return branch.ExitReady
		}
		fmt.Fprintf(stderr, "Error: %v. Ejecuta `axiom branch scope --help`.\n", err)
		return branch.ExitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: argumento inesperado '%s' para branch scope. Ejecuta `axiom branch scope --help`.\n", flags.Arg(0))
		return branch.ExitUsage
	}
	if strings.TrimSpace(*change) == "" {
		fmt.Fprintln(stderr, "Error: falta --change: indica el nombre del cambio, por ejemplo `axiom branch scope --change <nombre>`. Ejecuta `axiom branch scope --help`.")
		return branch.ExitUsage
	}
	if err := gitref.ValidBranchComponent(*change); err != nil {
		fmt.Fprintf(stderr, "Error: el nombre de cambio '%s' no es válido (%v). Ejecuta `axiom branch scope --help`.\n", *change, err)
		return branch.ExitUsage
	}
	dir, err := filepath.Abs(*cwd)
	if err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo resolver --cwd '%s' (%v). Indica una ruta absoluta con `axiom branch scope --change <nombre> --cwd <ruta>`.\n", *cwd, err)
		return branch.ExitUsage
	}
	return writeBranchScope(engine.Default().Scope(context.Background(), dir, *change), *asJSON, stdout, stderr)
}

// writeBranchScope imprime el resultado, como JSON o como texto en castellano, y devuelve el
// código de salida: 0 cuando el alcance se calculó.
func writeBranchScope(result branch.ScopeResult, asJSON bool, stdout, stderr io.Writer) int {
	if asJSON {
		if err := result.WriteJSON(stdout); err != nil {
			fmt.Fprintf(stderr, "Error: no se pudo escribir el resultado: %v. Repite `axiom branch scope --json`.\n", err)
			return branch.ExitRefused
		}
		return result.ExitCode()
	}
	result.WriteText(stdout, stderr)
	return result.ExitCode()
}
