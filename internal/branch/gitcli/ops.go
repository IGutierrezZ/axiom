// Package gitcli son las lecturas tipadas de git del motor de ramas (INC-25, design
// D-01): el adaptador de los puertos de `internal/branch` sobre gitcommon.Run, el único
// punto que ejecuta git (REQ-25.1, G-7). Usa una lista cerrada de subcomandos de
// lectura (`rev-parse`, `worktree list`, `status` y `rev-list`) y ninguno remoto; las
// escrituras de `ensure` y `done` y el resto de lecturas llegan con el primer
// llamador que las usa.
//
// Los textos de error están en castellano y nombran el comando de git y el directorio.
package gitcli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

// Git implementa las lecturas de git del motor. No tiene estado: cada método ejecuta
// git con gitcommon.Run desde el directorio que recibe, que es siempre un directorio
// del repositorio (nunca el del proceso).
type Git struct{}

// New devuelve las lecturas de git del motor.
func New() *Git { return &Git{} }

// TopLevel devuelve la raíz del árbol de trabajo que contiene dir. Fuera de un
// repositorio devuelve un error que envuelve gitcommon.ErrNotGitRepository.
func (*Git) TopLevel(ctx context.Context, dir string) (string, error) {
	return gitcommon.TopLevel(ctx, dir)
}

// CommonDir devuelve el git common dir del repositorio que contiene dir. Fuera de un
// repositorio devuelve un error que envuelve gitcommon.ErrNotGitRepository.
func (*Git) CommonDir(ctx context.Context, dir string) (string, error) {
	return gitcommon.CommonDir(ctx, dir)
}

// ResolveCommit resuelve ref al commit al que apunta (`git rev-parse --verify --quiet
// <ref>^{commit}`): una etiqueta anotada se pela hasta su commit. Una referencia que no
// existe, o un repositorio sin commits, no es un error: devuelve ok = false.
func (*Git) ResolveCommit(ctx context.Context, dir, ref string) (sha string, ok bool, err error) {
	if err := plainArgument(ref); err != nil {
		return "", false, err
	}
	result, err := gitcommon.Run(ctx, dir, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil {
		return "", false, err
	}
	switch result.ExitCode {
	case 0:
		return strings.TrimSpace(string(result.Stdout)), true, nil
	case 1:
		return "", false, nil
	}
	return "", false, commandFailed(dir, result, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
}

// BranchExists dice si existe la rama local name (`refs/heads/<name>`); una etiqueta o
// una referencia remota con el mismo nombre no cuentan.
func (*Git) BranchExists(ctx context.Context, dir, name string) (bool, error) {
	if err := plainArgument(name); err != nil {
		return false, err
	}
	args := []string{"rev-parse", "--verify", "--quiet", "refs/heads/" + name}
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return false, err
	}
	switch result.ExitCode {
	case 0:
		return true, nil
	case 1:
		return false, nil
	}
	return false, commandFailed(dir, result, args...)
}

// Worktrees lista los worktrees del repositorio (`git worktree list --porcelain`), el
// checkout principal primero. Las rutas son absolutas, limpias y con el separador del
// sistema. No admite rutas con saltos de línea.
func (*Git) Worktrees(ctx context.Context, dir string) ([]branch.WorktreeEntry, error) {
	args := []string{"worktree", "list", "--porcelain"}
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, commandFailed(dir, result, args...)
	}
	entries, err := parseWorktrees(string(result.Stdout))
	if err != nil {
		return nil, fmt.Errorf("git %s en %s devolvió una salida que no se entiende: %w", strings.Join(args, " "), dir, err)
	}
	return entries, nil
}

// parseWorktrees interpreta la salida de `git worktree list --porcelain`: bloques
// separados por una línea en blanco, cada uno con `worktree <ruta>`, `HEAD <commit>` y
// `branch refs/heads/<rama>` o `detached`, y atributos que aquí no se usan (`bare`,
// `locked`, `prunable`). Tolera CRLF y la falta de línea en blanco final.
func parseWorktrees(output string) ([]branch.WorktreeEntry, error) {
	var entries []branch.WorktreeEntry
	var current *branch.WorktreeEntry
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			current = nil
			continue
		}
		key, value, _ := strings.Cut(line, " ")
		if key == "worktree" {
			entries = append(entries, branch.WorktreeEntry{Path: filepath.Clean(filepath.FromSlash(value))})
			current = &entries[len(entries)-1]
			continue
		}
		if current == nil {
			return nil, fmt.Errorf("la línea %q no pertenece a ningún worktree", line)
		}
		switch key {
		case "HEAD":
			current.Head = value
		case "branch":
			current.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "detached":
			current.Detached = true
		}
	}
	return entries, nil
}

// DirtyPaths devuelve las rutas con cambios sin confirmar del árbol de trabajo de dir:
// modificadas, preparadas y sin seguimiento (`git status --porcelain=v1 -z
// --untracked-files=normal`). Un renombrado cuenta una vez, con su ruta nueva, y una
// carpeta sin seguimiento, una vez, con su «/» final. Las rutas salen tal cual, con
// «/» y sin comillas.
func (*Git) DirtyPaths(ctx context.Context, dir string) ([]string, error) {
	args := []string{"status", "--porcelain=v1", "-z", "--untracked-files=normal"}
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, commandFailed(dir, result, args...)
	}
	var paths []string
	records := strings.Split(string(result.Stdout), "\x00")
	for i := 0; i < len(records); i++ {
		record := records[i]
		if record == "" {
			continue
		}
		if len(record) < 4 || record[2] != ' ' {
			return nil, fmt.Errorf("git %s en %s devolvió una entrada que no se entiende: %q", strings.Join(args, " "), dir, record)
		}
		paths = append(paths, record[3:])
		// En un renombrado o una copia, el registro siguiente es la ruta de origen.
		if strings.ContainsAny(record[:2], "RC") {
			i++
		}
	}
	return paths, nil
}

// AheadBehind cuenta los commits de branch que no tiene base (adelanto) y los de base
// que no tiene branch (atraso), solo con referencias locales (`git rev-list
// --left-right --count <base>...<branch>`). base y branch son referencias completas o
// identificadores de commit.
func (*Git) AheadBehind(ctx context.Context, dir, base, branchRef string) (ahead, behind int, err error) {
	if err := plainArgument(base); err != nil {
		return 0, 0, err
	}
	if err := plainArgument(branchRef); err != nil {
		return 0, 0, err
	}
	args := []string{"rev-list", "--left-right", "--count", base + "..." + branchRef}
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return 0, 0, err
	}
	if result.ExitCode != 0 {
		return 0, 0, commandFailed(dir, result, args...)
	}
	counts := strings.Fields(string(result.Stdout))
	if len(counts) != 2 {
		return 0, 0, fmt.Errorf("git %s en %s devolvió %q, se esperaban dos números", strings.Join(args, " "), dir, strings.TrimSpace(string(result.Stdout)))
	}
	if behind, err = strconv.Atoi(counts[0]); err != nil {
		return 0, 0, fmt.Errorf("git %s en %s devolvió un atraso que no es un número: %w", strings.Join(args, " "), dir, err)
	}
	if ahead, err = strconv.Atoi(counts[1]); err != nil {
		return 0, 0, fmt.Errorf("git %s en %s devolvió un adelanto que no es un número: %w", strings.Join(args, " "), dir, err)
	}
	return ahead, behind, nil
}

// plainArgument rechaza un nombre o una referencia que git leería como una opción
// (empieza por «-») o que está vacío, antes de lanzar nada.
func plainArgument(arg string) error {
	if arg == "" || strings.HasPrefix(arg, "-") {
		return errors.New("la referencia " + strconv.Quote(arg) + " no es válida para git: no puede estar vacía ni empezar por «-»")
	}
	return nil
}

// commandFailed describe un comando de git que terminó con un código inesperado.
func commandFailed(dir string, result gitcommon.Result, args ...string) error {
	return fmt.Errorf("git %s en %s terminó con el código %d: %s", strings.Join(args, " "), dir, result.ExitCode, result.Stderr)
}
