package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// Esqueleto de `axiom branch` (INC-25, D-02): la familia de verbos que planifica y
// consulta las ramas y worktrees de un cambio. Esta rebanada solo trae `status` en
// su forma mínima, sobre el único punto que ejecuta git (gitcommon); el resultado
// completo, el dominio y el motor llegan con las rebanadas siguientes y la ayuda
// lista solo los verbos que existen (decisión T3). Ningún verbo ejecuta `push`, abre
// un PR ni hace operaciones remotas (REQ-25.1).

// Códigos de salida de `axiom branch` (design D-02): 0 cuando se alcanza el estado
// pedido, 1 ante un rechazo tipado o un error, 2 ante un error de uso.
const (
	branchExitReady   = 0
	branchExitRefused = 1
	branchExitUsage   = 2
)

// branchResultSchema es el contrato de salida común de la familia (design §4.3).
const branchResultSchema = "axiom.branch-result/v1"

// Códigos tipados que usa esta rebanada, tomados de la lista cerrada de la spec.
const (
	branchCodeNotAGitRepository = "not_a_git_repository"
	branchCodeGitFailed         = "git_failed"
)

const branchHelp = `Uso: axiom branch <subcomando> [argumentos]

Subcomandos:
  status   Muestra el repositorio git, su git common dir y la rama actual (solo lectura)

Ejecuta ` + "`axiom branch <subcomando> --help`" + ` para ver las banderas de cada subcomando.
`

const branchStatusHelp = `Uso: axiom branch status [--cwd <ruta>] [--json]

Muestra, solo con lecturas, la raíz del repositorio git que contiene el directorio de
partida, su git common dir y la rama actual. No ejecuta fetch, pull, push ni ls-remote.

Banderas:
  --cwd <ruta>   Directorio de partida (por defecto, el directorio de trabajo del proceso)
  --json         Imprime el resultado como axiom.branch-result/v1
`

// branchStatusResult es el `axiom.branch-result/v1` mínimo de esta rebanada. Las
// claves van en camelCase y cada valor enumerado lleva su token estable en inglés y,
// en un campo hermano, su etiqueta en castellano (REQ-25.2). Todas las rutas son
// absolutas. La serialización es estable y no lleva marcas de tiempo.
type branchStatusResult struct {
	Schema        string   `json:"schema"`
	Operation     string   `json:"operation"`
	Status        string   `json:"status"`
	StatusLabel   string   `json:"statusLabel"`
	Code          string   `json:"code,omitempty"`
	Message       string   `json:"message,omitempty"`
	Continuation  string   `json:"continuation,omitempty"`
	GitRoot       string   `json:"gitRoot,omitempty"`
	CommonDir     string   `json:"commonDir,omitempty"`
	CurrentBranch string   `json:"currentBranch,omitempty"`
	Detached      bool     `json:"detached,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

// RunBranch ejecuta `axiom branch <subcomando>` y devuelve el código de salida.
func RunBranch(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, branchHelp)
		return branchExitUsage
	}
	switch subcommand, rest := args[0], args[1:]; subcommand {
	case "--help", "-h", "help":
		fmt.Fprint(stdout, branchHelp)
		return branchExitReady
	case "status":
		return runBranchStatus(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: subcomando '%s' no reconocido para branch; el disponible es status. Ejecuta `axiom branch --help`.\n", subcommand)
		return branchExitUsage
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
			return branchExitReady
		}
		fmt.Fprintf(stderr, "Error: %v. Ejecuta `axiom branch status --help`.\n", err)
		return branchExitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "Error: argumento inesperado '%s' para branch status. Ejecuta `axiom branch status --help`.\n", flags.Arg(0))
		return branchExitUsage
	}
	dir, err := filepath.Abs(*cwd)
	if err != nil {
		fmt.Fprintf(stderr, "Error: no se pudo resolver --cwd '%s' (%v). Indica una ruta absoluta con `axiom branch status --cwd <ruta>`.\n", *cwd, err)
		return branchExitUsage
	}
	return writeBranchStatus(branchStatus(context.Background(), dir), *asJSON, stdout, stderr)
}

// branchStatus lee, sin escribir nada, el repositorio al que pertenece dir: su raíz y
// su git common dir con git (TopLevel, CommonDir) y la rama actual de los ficheros
// (ReadHead, REQ-25.21). dir debe ser absoluto.
func branchStatus(ctx context.Context, dir string) branchStatusResult {
	result := branchStatusResult{Schema: branchResultSchema, Operation: "status"}

	if info, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return branchRefusal(result, branchCodeNotAGitRepository,
				fmt.Sprintf("el directorio %s no existe", dir),
				"indica el directorio de un repositorio git con `axiom branch status --cwd <ruta>`")
		}
		return branchRefusal(result, branchCodeNotAGitRepository,
			fmt.Sprintf("no se pudo acceder al directorio %s: %v", dir, err),
			"indica el directorio de un repositorio git con `axiom branch status --cwd <ruta>`")
	} else if !info.IsDir() {
		return branchRefusal(result, branchCodeNotAGitRepository,
			fmt.Sprintf("%s no es un directorio", dir),
			"indica el directorio de un repositorio git con `axiom branch status --cwd <ruta>`")
	}

	gitRoot, err := gitcommon.TopLevel(ctx, dir)
	if err != nil {
		return branchGitFailure(result, dir, err)
	}
	commonDir, err := gitcommon.CommonDir(ctx, dir)
	if err != nil {
		return branchGitFailure(result, dir, err)
	}

	result.Status, result.StatusLabel = "ready", branchStatusLabel("ready")
	result.GitRoot, result.CommonDir = gitRoot, commonDir
	switch branch, detached, headErr := gitcommon.ReadHead(gitRoot); {
	case headErr != nil:
		result.Warnings = append(result.Warnings, fmt.Sprintf("no se pudo determinar la rama actual: %v", headErr))
	case detached:
		result.Detached = true
	default:
		result.CurrentBranch = branch
	}
	return result
}

// branchRefusal marca result como un rechazo tipado.
func branchRefusal(result branchStatusResult, code, message, continuation string) branchStatusResult {
	result.Status, result.StatusLabel = "refused", branchStatusLabel("refused")
	result.Code, result.Message, result.Continuation = code, message, continuation
	return result
}

// branchGitFailure traduce un fallo de gitcommon: «no es un repositorio» es un
// rechazo (`not_a_git_repository`); cualquier otro es `git_failed`, que nombra la
// operación y resume el error de git (REQ-25.2).
func branchGitFailure(result branchStatusResult, dir string, err error) branchStatusResult {
	if errors.Is(err, gitcommon.ErrNotGitRepository) {
		return branchRefusal(result, branchCodeNotAGitRepository,
			fmt.Sprintf("%s no pertenece a ningún repositorio git", dir),
			fmt.Sprintf("indica el directorio de un repositorio git con `axiom branch status --cwd <ruta>` o inicia uno aquí con `git -C %s init`", pathquote.Quote(dir)))
	}
	result.Status, result.StatusLabel = "error", branchStatusLabel("error")
	result.Code = branchCodeGitFailed
	result.Message = fmt.Sprintf("git falló: %v", err)
	result.Continuation = fmt.Sprintf("comprueba que git está instalado y que puedes leer %s, y repite `axiom branch status --cwd %s`", dir, pathquote.Quote(dir))
	return result
}

// branchStatusLabel es la etiqueta en castellano de cada token de `status`.
func branchStatusLabel(status string) string {
	switch status {
	case "ready":
		return "listo"
	case "refused":
		return "rechazado"
	default:
		return "error"
	}
}

// writeBranchStatus imprime el resultado, como JSON o como texto en castellano, y
// devuelve el código de salida: 0 solo cuando el estado pedido se alcanzó.
func writeBranchStatus(result branchStatusResult, asJSON bool, stdout, stderr io.Writer) int {
	exit := branchExitReady
	if result.Status != "ready" {
		exit = branchExitRefused
	}
	if asJSON {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(stderr, "Error: no se pudo escribir el resultado: %v. Repite `axiom branch status --json`.\n", err)
			return branchExitRefused
		}
		return exit
	}
	if result.Status != "ready" {
		fmt.Fprintf(stderr, "Error [%s]: %s\nContinuación: %s\n", result.Code, result.Message, result.Continuation)
		return exit
	}
	fmt.Fprintf(stdout, "Repositorio git: %s\n", result.GitRoot)
	fmt.Fprintf(stdout, "Directorio común de git: %s\n", result.CommonDir)
	switch {
	case result.Detached:
		fmt.Fprintln(stdout, "Rama actual: ninguna (HEAD desacoplado)")
	case result.CurrentBranch != "":
		fmt.Fprintf(stdout, "Rama actual: %s\n", result.CurrentBranch)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(stdout, "Aviso: %s\n", warning)
	}
	return exit
}
