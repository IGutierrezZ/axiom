package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// `axiom branch` (INC-25, D-02): la familia de verbos que planifica y consulta las
// ramas y worktrees de un cambio. Esta rebanada trae `status` sobre el dominio
// internal/branch (resultado `axiom.branch-result/v1`, etiquetas y política de ramas)
// y sobre el único punto que ejecuta git (gitcommon); el motor y los demás verbos
// llegan con las rebanadas siguientes y la ayuda lista solo los verbos que existen
// (decisión T3). Ningún verbo ejecuta `push`, abre un PR ni hace operaciones remotas
// (REQ-25.1).

const branchHelp = `Uso: axiom branch <subcomando> [argumentos]

Subcomandos:
  status   Muestra el repositorio git, la rama actual y la rama de integración (solo lectura)

Ejecuta ` + "`axiom branch <subcomando> --help`" + ` para ver las banderas de cada subcomando.
`

const branchStatusHelp = `Uso: axiom branch status [--cwd <ruta>] [--json]

Muestra, solo con lecturas, la raíz del repositorio git que contiene el directorio de
partida, su git common dir, la rama actual y la rama de integración que resuelve la
política de ramas de axiom.yaml (bloques git: y repositories:; sin ellos, main). No
ejecuta fetch, pull, push ni ls-remote.

Banderas:
  --cwd <ruta>   Directorio de partida (por defecto, el directorio de trabajo del proceso)
  --json         Imprime el resultado como axiom.branch-result/v1
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
	dir, err := filepath.Abs(*cwd)
	if err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo resolver --cwd '%s' (%v). Indica una ruta absoluta con `axiom branch status --cwd <ruta>`.\n", *cwd, err)
		return branch.ExitUsage
	}
	return writeBranchResult(branchStatus(context.Background(), dir), *asJSON, stdout, stderr)
}

// branchStatus lee, sin escribir nada, el repositorio al que pertenece dir: su raíz y
// su git common dir con git (TopLevel, CommonDir), la rama actual de los ficheros
// (ReadHead, REQ-25.21) y la rama de integración que resuelve la política de ramas
// del proyecto (branch.ResolvePolicy, REQ-25.3). dir debe ser absoluto.
func branchStatus(ctx context.Context, dir string) branch.Result {
	result := branch.NewResult("status")

	if reason := branchDirectoryProblem(dir); reason != "" {
		return result.Refused(branch.UnusableDirectory("status", reason))
	}
	gitRoot, err := gitcommon.TopLevel(ctx, dir)
	if err != nil {
		return result.Refused(branchGitRefusal("status", dir, "localizar la raíz del repositorio", err))
	}
	commonDir, err := gitcommon.CommonDir(ctx, dir)
	if err != nil {
		return result.Refused(branchGitRefusal("status", dir, "localizar el directorio común de git", err))
	}

	location, err := workspace.Locate(dir)
	if err != nil {
		return result.Refused(branch.LocateRefusal(err))
	}
	if refusal := branch.ConfigRefusal(location); refusal != nil {
		return result.Refused(refusal)
	}

	// Un repositorio fuera de la carpeta maestra no puede tener entrada propia en
	// `repositories:` (sus rutas son relativas a ella): rige la política del workspace.
	policy := branch.ResolvePolicy(location.Config)
	repository := branch.RepositoryResult{GitRoot: gitRoot, CommonDir: commonDir, IntegrationBranch: policy.IntegrationBranch}
	if relative, inside := branchRelativePath(location.Master, gitRoot); inside {
		repository.IntegrationBranch = policy.For(relative).IntegrationBranch
	}
	switch name, detached, headErr := gitcommon.ReadHead(gitRoot); {
	case headErr != nil:
		result.Warnings = append(result.Warnings, fmt.Sprintf("no se pudo determinar la rama actual: %v", headErr))
	case detached:
		repository.Detached = true
	default:
		repository.CurrentBranch = name
	}
	result.Repositories = []branch.RepositoryResult{repository}
	return result
}

// branchDirectoryProblem explica por qué dir no sirve como directorio de partida, o
// devuelve la cadena vacía si es un directorio al que se puede acceder.
func branchDirectoryProblem(dir string) string {
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return fmt.Sprintf("el directorio %s no existe", dir)
	case err != nil:
		return fmt.Sprintf("no se pudo acceder al directorio %s: %v", dir, err)
	case !info.IsDir():
		return fmt.Sprintf("%s no es un directorio", dir)
	}
	return ""
}

// branchGitRefusal traduce un fallo de gitcommon: «no es un repositorio» es un
// rechazo (`not_a_git_repository`); cualquier otro es `git_failed`, que nombra la
// operación y resume el error de git (REQ-25.2).
func branchGitRefusal(operation, dir, gitOperation string, err error) *branch.Refusal {
	if errors.Is(err, gitcommon.ErrNotGitRepository) {
		return branch.NotAGitRepository(operation, dir)
	}
	return branch.GitFailed(operation, dir, gitOperation, err)
}

// branchRelativePath devuelve la ruta de gitRoot relativa a la carpeta maestra, con
// «/» (`.` si es la propia carpeta maestra), y false si gitRoot no está dentro de ella.
// Asciende desde gitRoot comparando cada directorio con master por identidad, no por
// texto: git devuelve la grafía larga de Windows y la carpeta maestra puede llevar el
// nombre corto 8.3 (tasks.md, regla 10).
func branchRelativePath(master, gitRoot string) (string, bool) {
	var segments []string
	for current := filepath.Clean(gitRoot); ; {
		if pathidentity.SameDirectory(current, master) {
			if len(segments) == 0 {
				return ".", true
			}
			slices.Reverse(segments)
			return path.Join(segments...), true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		segments = append(segments, filepath.Base(current))
		current = parent
	}
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
