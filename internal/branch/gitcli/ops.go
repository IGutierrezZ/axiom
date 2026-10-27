// Package gitcli son las operaciones tipadas de git del motor de ramas (INC-25, design
// D-01): el adaptador de los puertos de `internal/branch` sobre gitcommon.Run, el único
// punto que ejecuta git (REQ-25.1, G-7). Las lecturas usan una lista cerrada de
// subcomandos (`rev-parse`, `worktree list`, `status`, `rev-list`, `ls-tree` y `merge-base
// --is-ancestor`) y las escrituras son solo `branch` (crear y eliminar), `worktree add`,
// `worktree remove` y `switch` (modo `in_place`); ninguna es remota. La única
// operación remota es Fetch (`git fetch origin`, `ensure --fetch` y `done --fetch`, REQ-25.10):
// el puerto no tiene ningún método que publique. Las operaciones de la capa de árbol equivalente
// de `done` llegan con el primer llamador que las usa.
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

// TreeDirs lista los subdirectorios inmediatos de relPath en el árbol de rev, sin leer el
// árbol de trabajo (`git ls-tree -z --full-tree <rev> -- <relPath>/`): solo cuentan las
// entradas de tipo `tree` (ni ficheros ni submódulos), en el orden de git. exists dice si
// relPath es una carpeta que tiene algo en rev; una ruta que no existe, o que es un
// fichero, no es un error. relPath es relativa a la raíz del repositorio, con «/», y se
// toma literalmente (ls-tree no expande comodines); dir es cualquier directorio del
// repositorio. Los nombres salen íntegros porque la salida es `-z`.
func (*Git) TreeDirs(ctx context.Context, dir, rev, relPath string) (dirs []string, exists bool, err error) {
	if err := plainArgument(rev); err != nil {
		return nil, false, err
	}
	if err := pathspecArgument(relPath); err != nil {
		return nil, false, err
	}
	folder := strings.TrimSuffix(relPath, "/") + "/"
	args := []string{"ls-tree", "-z", "--full-tree", rev, "--", folder}
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return nil, false, err
	}
	if result.ExitCode != 0 {
		return nil, false, commandFailed(dir, result, args...)
	}
	for _, record := range strings.Split(string(result.Stdout), "\x00") {
		if record == "" {
			continue
		}
		// `<modo> <tipo> <objeto>\t<ruta>`: la ruta va tras el único tabulador.
		meta, entry, found := strings.Cut(record, "\t")
		fields := strings.Fields(meta)
		name, inside := strings.CutPrefix(entry, folder)
		if !found || len(fields) != 3 || !inside || name == "" || strings.Contains(name, "/") {
			return nil, false, fmt.Errorf("git %s en %s devolvió una entrada que no se entiende: %q", strings.Join(args, " "), dir, record)
		}
		exists = true
		if fields[1] == "tree" {
			dirs = append(dirs, name)
		}
	}
	return dirs, exists, nil
}

// CreateBranch crea la rama local name apuntando a startPoint (`git branch --no-track
// <name> <startPoint>`). Nunca mueve ni reinicia una rama existente: si name ya existe,
// git falla y la rama queda donde estaba (REQ-25.7). `--no-track` evita que git configure
// un upstream cuando el punto de partida es una referencia remota ya traída a local:
// `axiom branch` nunca publica ni sigue una rama remota. dir es el checkout principal.
func (*Git) CreateBranch(ctx context.Context, dir, name, startPoint string) error {
	if err := plainArgument(name); err != nil {
		return err
	}
	if err := plainArgument(startPoint); err != nil {
		return err
	}
	return runWrite(ctx, dir, "branch", "--no-track", name, startPoint)
}

// AddWorktree despliega la rama local existente branch en un worktree nuevo en path
// (`git worktree add <path> <branch>`). Nunca crea la rama: si no existe, git falla y no
// deja nada. path debe ser absoluta (los worktrees nunca se colocan por una ruta relativa
// al directorio del proceso) y dir es el checkout principal, que no cambia de rama, de
// índice ni de árbol de trabajo (REQ-25.8).
func (*Git) AddWorktree(ctx context.Context, dir, path, branchName string) error {
	if !filepath.IsAbs(path) {
		return errors.New("la ruta " + strconv.Quote(path) + " del worktree no es válida para git: debe ser absoluta")
	}
	if err := plainArgument(branchName); err != nil {
		return err
	}
	return runWrite(ctx, dir, "worktree", "add", path, branchName)
}

// Switch despliega la rama local existente name en el propio repositorio (`git switch
// --no-guess <name>`): es el cambio de rama del modo `in_place` (REQ-25.8). Nunca crea la
// rama ni la adivina desde una referencia remota local del mismo nombre, que crearía una rama
// con seguimiento (`--no-guess`): si no existe, git falla y el repositorio queda como estaba.
// Tampoco sobrescribe cambios sin confirmar: ante un conflicto, git se niega. Quien llama
// comprueba antes, sin efectos, que el árbol está limpio (REQ-25.9). dir es el checkout
// principal del repositorio, que es el que cambia de rama.
func (*Git) Switch(ctx context.Context, dir, name string) error {
	if err := plainArgument(name); err != nil {
		return err
	}
	return runWrite(ctx, dir, "switch", "--no-guess", name)
}

// IsAncestor dice si ancestor es ancestro de descendant, o el mismo commit (`git merge-base
// --is-ancestor`): la capa 1 de la detección de fusión de `done` (`ancestry`, REQ-25.16). Es una
// lectura que usa solo referencias locales. Una referencia que git no resuelve no es «no es
// ancestro» sino un error: quien llama comprueba antes que existen.
func (*Git) IsAncestor(ctx context.Context, dir, ancestor, descendant string) (bool, error) {
	if err := plainArgument(ancestor); err != nil {
		return false, err
	}
	if err := plainArgument(descendant); err != nil {
		return false, err
	}
	args := []string{"merge-base", "--is-ancestor", ancestor, descendant}
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

// RemoveWorktree retira el worktree enlazado de path (`git worktree remove [--force] <path>`) y
// su registro en git: es el primer paso del desmontaje de `done` (REQ-25.17). Sin force, git se
// niega ante un worktree con cambios sin confirmar; con él los descarta. path debe ser absoluta y
// dir es el checkout principal, nunca el worktree que se retira. Nunca borra la rama.
func (*Git) RemoveWorktree(ctx context.Context, dir, path string, force bool) error {
	if !filepath.IsAbs(path) {
		return errors.New("la ruta " + strconv.Quote(path) + " del worktree no es válida para git: debe ser absoluta")
	}
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	return runWrite(ctx, dir, append(args, path)...)
}

// DeleteBranch elimina la rama local name aunque git no la considere fusionada
// (`git branch -D <name>`): tras un squash o un rebase la fusión ya está confirmada por las capas
// de `done`, y `-d` se negaría (REQ-25.17). Solo toca `refs/heads/<name>`: nunca una rama remota.
// git se niega si la rama está desplegada en algún worktree, de modo que se retira antes. dir es
// el checkout principal.
func (*Git) DeleteBranch(ctx context.Context, dir, name string) error {
	if err := plainArgument(name); err != nil {
		return err
	}
	return runWrite(ctx, dir, "branch", "-D", name)
}

// Fetch trae las referencias del remoto (`git fetch <remote>`, sin `--prune`): es la única
// operación remota del motor y solo la lanza `ensure --fetch` (REQ-25.10, design D-06).
// Actualiza las referencias remotas locales (`refs/remotes/<remote>/*`) y no mueve la rama
// actual ni toca el árbol de trabajo. dir es el checkout principal. Un fallo (remoto
// inexistente o inalcanzable) devuelve un error en castellano con el comando, el directorio
// y el resumen de git.
func (*Git) Fetch(ctx context.Context, dir, remote string) error {
	if err := plainArgument(remote); err != nil {
		return err
	}
	return runWrite(ctx, dir, "fetch", remote)
}

// runWrite ejecuta una escritura de git y devuelve un error en castellano, con el comando,
// el directorio y el resumen de git, si no termina con el código 0.
func runWrite(ctx context.Context, dir string, args ...string) error {
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return commandFailed(dir, result, args...)
	}
	return nil
}

// plainArgument rechaza un nombre o una referencia que git leería como una opción
// (empieza por «-») o que está vacío, antes de lanzar nada.
func plainArgument(arg string) error {
	if arg == "" || strings.HasPrefix(arg, "-") {
		return errors.New("la referencia " + strconv.Quote(arg) + " no es válida para git: no puede estar vacía ni empezar por «-»")
	}
	return nil
}

// pathspecArgument rechaza una ruta que git leería de otro modo que como una ruta
// literal relativa a la raíz del repositorio: vacía, la propia raíz, absoluta, con
// segmentos `..` o `.`, con caracteres de control, o que empieza por «-» (una opción) o
// «:» (un prefijo de patrón mágico).
func pathspecArgument(relPath string) error {
	invalid := func(reason string) error {
		return fmt.Errorf("la ruta %s no es válida para git: %s", strconv.Quote(relPath), reason)
	}
	switch {
	case relPath == "" || relPath == ".":
		return invalid("debe nombrar una carpeta distinta de la raíz del repositorio")
	case strings.HasPrefix(relPath, "/") || strings.HasPrefix(relPath, `\`) || filepath.IsAbs(relPath):
		return invalid("debe ser relativa a la raíz del repositorio")
	case strings.HasPrefix(relPath, "-") || strings.HasPrefix(relPath, ":"):
		return invalid("no puede empezar por «-» ni por «:»")
	case strings.ContainsFunc(relPath, func(r rune) bool { return r < 0x20 || r == 0x7f }):
		return invalid("no puede llevar caracteres de control")
	}
	for _, segment := range strings.Split(relPath, "/") {
		if segment == "." || segment == ".." {
			return invalid("no puede llevar segmentos «.» ni «..»")
		}
	}
	return nil
}

// commandFailed describe un comando de git que terminó con un código inesperado.
func commandFailed(dir string, result gitcommon.Result, args ...string) error {
	return fmt.Errorf("git %s en %s terminó con el código %d: %s", strings.Join(args, " "), dir, result.ExitCode, result.Stderr)
}
