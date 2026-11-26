package gitcli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/branch"
	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

// commitSentinel abre el encabezado de cada commit en la salida de CommitFiles. Lleva un byte de
// control que git no escribe en una ruta corriente, de modo que un nombre de fichero no se
// confunde con el inicio de un commit.
const commitSentinel = "\x01axiom-commit\x01"

// RevList lista los commits de las revisiones con sus padres (`git rev-list --parents
// [--first-parent] <revisiones> --`), del más reciente al más antiguo. Una revisión excluida va
// con el prefijo «^». Es la única forma de leer la cadena de primeros padres con sus padres
// (AheadBehind solo cuenta). Es una lectura local y toda revisión pasa por plainArgument.
func (*Git) RevList(ctx context.Context, dir string, firstParent bool, revisions ...string) ([]branch.RevEntry, error) {
	if err := plainRevisions(revisions); err != nil {
		return nil, err
	}
	args := []string{"rev-list", "--parents"}
	if firstParent {
		args = append(args, "--first-parent")
	}
	args = append(append(args, revisions...), "--")
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, commandFailed(dir, result, args...)
	}
	var entries []branch.RevEntry
	for _, line := range strings.Split(string(result.Stdout), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		entries = append(entries, branch.RevEntry{SHA: fields[0], Parents: fields[1:]})
	}
	return entries, nil
}

// CommitFiles devuelve los commits de las revisiones con las rutas que toca cada uno, en una sola
// lectura (`git log -z --no-renames --no-show-signature --format=<centinela>%H%x09%P --name-only
// <revisiones> --`). Con `--no-renames` un renombrado aporta sus dos rutas y la salida `-z` deja
// íntegros los nombres con espacios y caracteres no ASCII. Git no lista los ficheros de un commit
// de fusión: su entrada lleva sus padres y ninguna ruta. Es una lectura local.
func (*Git) CommitFiles(ctx context.Context, dir string, revisions ...string) ([]branch.CommitPaths, error) {
	if err := plainRevisions(revisions); err != nil {
		return nil, err
	}
	args := []string{"log", "-z", "--no-renames", "--no-show-signature", "--format=" + strings.ReplaceAll(commitSentinel, "\x01", "%x01") + "%H%x09%P", "--name-only"}
	args = append(append(args, revisions...), "--")
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, commandFailed(dir, result, args...)
	}
	var commits []branch.CommitPaths
	for _, chunk := range strings.Split(string(result.Stdout), commitSentinel) {
		if chunk == "" {
			continue
		}
		header, rest, found := strings.Cut(chunk, "\x00")
		sha, parents, tab := strings.Cut(header, "\t")
		if !found || !tab || sha == "" {
			return nil, fmt.Errorf("git %s en %s devolvió una entrada que no se entiende: %q", strings.Join(args, " "), dir, chunk)
		}
		commit := branch.CommitPaths{SHA: sha, Parents: strings.Fields(parents)}
		for _, path := range strings.Split(strings.TrimPrefix(rest, "\n"), "\x00") {
			if path != "" {
				commit.Paths = append(commit.Paths, path)
			}
		}
		commits = append(commits, commit)
	}
	return commits, nil
}

// MergeResolvedPaths implementa branch.Git: las rutas de un commit de fusión que la propia fusión
// resolvió (`git diff-tree -r --no-commit-id --name-only -z --no-renames --cc <sha>`), la misma
// lectura con la que `axiom sdd publish` clasifica los commits. En una fusión limpia no hay
// ninguna. Es una lectura local.
func (*Git) MergeResolvedPaths(ctx context.Context, dir, sha string) ([]string, error) {
	if err := plainArgument(sha); err != nil {
		return nil, err
	}
	args := []string{"diff-tree", "-r", "--no-commit-id", "--name-only", "-z", "--no-renames", "--cc", sha}
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, commandFailed(dir, result, args...)
	}
	var paths []string
	for _, path := range strings.Split(string(result.Stdout), "\x00") {
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

// remoteCopies son los nombres con los que git escribe la copia remota de una rama en el asunto
// del reflog cuando la rama se adopta de ella.
func remoteCopies(name string) []string {
	return []string{"refs/remotes/origin/" + name, "remotes/origin/" + name, "origin/" + name}
}

// BranchCreation lee la entrada más antigua del reflog de la rama local name (`git reflog show
// --format=%H%x09%gs refs/heads/<name> --`): sha es el commit en que se creó la rama y from, el
// origen que declara el asunto `branch: Created from <origen>`. ok es falso si no hay entrada (rama
// solo remota, reflog caducado), si el asunto no es de una creación con `git branch` o si la rama
// se adoptó de su propia copia remota (`origin/<rama>`, REQ-27.15): en esos casos el commit de
// creación no dice dónde empezó el trabajo propio. Es una lectura local.
func (*Git) BranchCreation(ctx context.Context, dir, name string) (sha, from string, ok bool, err error) {
	if err := plainArgument(name); err != nil {
		return "", "", false, err
	}
	args := []string{"reflog", "show", "--format=%H%x09%gs", "refs/heads/" + name, "--"}
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return "", "", false, err
	}
	if result.ExitCode != 0 {
		// Una rama sin reflog (o que no existe) no es un fallo: no hay creación que leer.
		return "", "", false, nil
	}
	lines := strings.Split(strings.TrimRight(string(result.Stdout), "\r\n"), "\n")
	oldest := strings.TrimRight(lines[len(lines)-1], "\r")
	if oldest == "" {
		return "", "", false, nil
	}
	sha, subject, found := strings.Cut(oldest, "\t")
	origin, created := strings.CutPrefix(subject, "branch: Created from ")
	if !found || sha == "" || !created || origin == "" {
		return "", "", false, nil
	}
	for _, copy := range remoteCopies(name) {
		if origin == copy {
			return "", "", false, nil
		}
	}
	return sha, origin, true, nil
}

// plainRevisions comprueba que hay al menos una revisión y que ninguna es una opción para git. El
// prefijo «^» de una exclusión es válido: lo que se rechaza es el «-» inicial.
func plainRevisions(revisions []string) error {
	if len(revisions) == 0 {
		return errors.New("falta al menos una revisión para git")
	}
	for _, revision := range revisions {
		if err := plainArgument(strings.TrimPrefix(revision, "^")); err != nil {
			return err
		}
	}
	return nil
}
