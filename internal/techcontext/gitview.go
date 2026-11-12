package techcontext

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
)

// repository es un repositorio git de solo lectura: root es la raíz absoluta que
// devuelve `git rev-parse --show-toplevel` desde la ruta preguntada. Todo git se
// ejecuta con ese directorio y con gitcommon.Run, nunca desde el directorio del
// proceso (amenaza «selección de repositorio git», D7).
type repository struct {
	root string
}

// openRepository localiza el repositorio git que contiene dir.
func openRepository(ctx context.Context, dir string) (repository, error) {
	if !filepath.IsAbs(dir) {
		return repository{}, fmt.Errorf("la ruta %q no es absoluta: el repositorio git se elige siempre desde una ruta absoluta y nunca desde el directorio de trabajo del proceso", dir)
	}
	top, err := gitcommon.TopLevel(ctx, deepestExistingDir(filepath.Clean(dir)))
	if err != nil {
		return repository{}, err
	}
	return repository{root: resolveDeepest(top)}, nil
}

// deepestExistingDir devuelve el directorio existente más profundo de una ruta:
// ella misma si es un directorio, o el primer ancestro que lo es. Un documento
// que aún no existe se atribuye así al repositorio de su directorio.
func deepestExistingDir(p string) string {
	for {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
		parent := filepath.Dir(p)
		if parent == p {
			return p
		}
		p = parent
	}
}

// git ejecuta git en la raíz del repositorio. Un código de salida distinto de
// cero es un error: ninguna consulta de este paquete lo interpreta.
func (r repository) git(ctx context.Context, args ...string) ([]byte, error) {
	result, err := gitcommon.Run(ctx, r.root, args...)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("git %s en %s terminó con el código %d: %s", strings.Join(args, " "), r.root, result.ExitCode, result.Stderr)
	}
	return result.Stdout, nil
}

// head devuelve el commit de HEAD, o la cadena vacía si el repositorio no tiene
// commits (en cuyo caso se compara con el árbol vacío).
func (r repository) head(ctx context.Context) (string, error) {
	result, err := gitcommon.Run(ctx, r.root, "rev-parse", "--verify", "--quiet", "HEAD^{commit}")
	if err != nil {
		return "", err
	}
	switch result.ExitCode {
	case 0:
		return strings.TrimSpace(string(result.Stdout)), nil
	case 1:
		return "", nil
	}
	return "", fmt.Errorf("git rev-parse HEAD en %s terminó con el código %d: %s", r.root, result.ExitCode, result.Stderr)
}

// commitExists dice si el commit existe en el repositorio.
func (r repository) commitExists(ctx context.Context, commit string) (bool, error) {
	result, err := gitcommon.Run(ctx, r.root, "cat-file", "-e", commit+"^{commit}")
	if err != nil {
		return false, err
	}
	return result.ExitCode == 0, nil
}

// ChangedSince devuelve, como rutas absolutas ordenadas, los ficheros del
// repositorio que contiene repo que difieren de base. Une los cuatro estados en
// los que puede estar una edición (D7): confirmada (`diff --name-status
// base...HEAD`), en el índice (`diff --cached`), sin confirmar en el árbol de
// trabajo y no seguida (`ls-files --others --exclude-standard`); en un
// renombrado cuentan ambas rutas. Con base vacía se compara con el árbol vacío:
// cuenta todo fichero seguido o no seguido que exista. Solo lee y nunca ejecuta
// fetch; una base que no se resuelve falla cerrada.
func ChangedSince(ctx context.Context, repo, base string) ([]string, error) {
	r, err := openRepository(ctx, repo)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	add := func(rel string) {
		rel = strings.TrimSuffix(rel, "/")
		if rel != "" {
			set[filepath.Join(r.root, filepath.FromSlash(rel))] = true
		}
	}

	if base == "" {
		files, err := r.git(ctx, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
		if err != nil {
			return nil, err
		}
		for _, rel := range splitZ(files) {
			abs := filepath.Join(r.root, filepath.FromSlash(strings.TrimSuffix(rel, "/")))
			if _, err := os.Lstat(abs); err == nil {
				add(rel)
			}
		}
		return sortedKeys(set), nil
	}

	resolved, err := gitcommon.Run(ctx, r.root, "rev-parse", "--verify", "--quiet", base+"^{commit}")
	if err != nil {
		return nil, err
	}
	if resolved.ExitCode != 0 {
		return nil, fmt.Errorf("la base %q no se resuelve a un commit del repositorio %s", base, r.root)
	}

	committed, err := r.git(ctx, "diff", "--name-status", "-z", "--no-ext-diff", "--no-textconv", base+"...HEAD", "--")
	if err != nil {
		return nil, err
	}
	paths, err := nameStatusPaths(committed)
	if err != nil {
		return nil, fmt.Errorf("git diff %s...HEAD en %s: %w", base, r.root, err)
	}
	for _, rel := range paths {
		add(rel)
	}

	staged, err := r.git(ctx, "diff", "--cached", "--name-status", "-z", "--no-ext-diff", "--no-textconv", "--")
	if err != nil {
		return nil, err
	}
	paths, err = nameStatusPaths(staged)
	if err != nil {
		return nil, fmt.Errorf("git diff --cached en %s: %w", r.root, err)
	}
	for _, rel := range paths {
		add(rel)
	}

	// El árbol de trabajo se lee con `status`, que con GIT_OPTIONAL_LOCKS=0 no
	// reescribe el índice; `git diff` sí puede hacerlo si sus marcas de tiempo
	// están desfasadas.
	entries, err := r.status(ctx)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.Code != "??" && entry.Code[1] != ' ' {
			add(entry.Path)
		}
	}

	untracked, err := r.git(ctx, "ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	for _, rel := range splitZ(untracked) {
		add(rel)
	}
	return sortedKeys(set), nil
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// splitZ divide una salida terminada en NUL, sin entradas vacías.
func splitZ(data []byte) []string {
	var out []string
	for _, field := range bytes.Split(data, []byte{0}) {
		if len(field) > 0 {
			out = append(out, string(field))
		}
	}
	return out
}

// nameStatusPaths lee la salida de `git diff --name-status -z`: un estado y una
// ruta por entrada, y dos rutas (origen y destino) en un renombrado o una copia.
func nameStatusPaths(data []byte) ([]string, error) {
	fields := splitZ(data)
	var paths []string
	for i := 0; i < len(fields); {
		status := fields[i]
		count := 1
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			count = 2
		}
		if i+1+count > len(fields) {
			return nil, fmt.Errorf("devolvió una entrada incompleta: %q", status)
		}
		paths = append(paths, fields[i+1:i+1+count]...)
		i += 1 + count
	}
	return paths, nil
}

// statusEntry es una entrada de `git status --porcelain=v1`: el código XY de
// índice y árbol de trabajo (`??` sin seguimiento) y la ruta relativa a la raíz.
type statusEntry struct {
	Path string // relativa a la raíz del repositorio, con '/'
	Code string
}

// status devuelve el estado del repositorio con `git status --porcelain=v1 -z
// --untracked-files=all`: sin entrecomillado y con cada fichero sin seguimiento
// por separado. En un renombrado o una copia el origen cuenta como otra entrada.
func (r repository) status(ctx context.Context) ([]statusEntry, error) {
	out, err := r.git(ctx, "-c", "core.fsmonitor=false", "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	var entries []statusEntry
	fields := bytes.Split(out, []byte{0})
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		if len(field) == 0 {
			continue
		}
		if len(field) < 4 || field[2] != ' ' {
			return nil, fmt.Errorf("git status en %s devolvió una entrada que no se reconoce: %q", r.root, field)
		}
		code := string(field[:2])
		entries = append(entries, statusEntry{Path: string(field[3:]), Code: code})
		if strings.ContainsAny(code, "RC") && i+1 < len(fields) {
			i++
			entries = append(entries, statusEntry{Path: string(fields[i]), Code: code})
		}
	}
	return entries, nil
}

// behind cuenta con `git rev-list --count HEAD..@{upstream}` los commits que la
// rama lleva por detrás de su upstream, según las referencias locales: Axiom
// nunca actualiza el remoto. Sin upstream, con HEAD desacoplado o sin commits
// devuelve una nota y no bloquea.
func (r repository) behind(ctx context.Context) (count int, note string) {
	if branch, err := gitcommon.Run(ctx, r.root, "symbolic-ref", "--quiet", "HEAD"); err == nil && branch.ExitCode == 1 {
		return 0, "HEAD está desacoplado: no hay rama con upstream con la que comparar y no se comprueba si va por detrás"
	}
	if head, err := r.head(ctx); err == nil && head == "" {
		return 0, "el repositorio no tiene commits: no hay upstream con el que comparar"
	}
	if _, err := r.git(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}"); err != nil {
		return 0, "la rama actual no tiene upstream configurado: no se comprueba si va por detrás"
	}
	counted, err := r.git(ctx, "rev-list", "--count", "HEAD..@{upstream}")
	if err != nil {
		return 0, "no se pudo contar los commits que la rama lleva por detrás de su upstream"
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(counted)))
	if err != nil {
		return 0, "no se pudo interpretar el número de commits que la rama lleva por detrás de su upstream"
	}
	return n, ""
}

// committedExists dice si el fichero existía en el commit. La existencia se
// consulta con `git ls-tree`, que no depende del idioma de los mensajes de git.
func (r repository) committedExists(ctx context.Context, commit, rel string) (bool, error) {
	tree, err := r.git(ctx, "ls-tree", "-z", "--name-only", commit, "--", rel)
	if err != nil {
		return false, err
	}
	return len(bytes.Trim(tree, "\x00")) > 0, nil
}

// RepositoryRoot devuelve la raíz absoluta, con los enlaces simbólicos resueltos, del
// repositorio git que contiene path: un directorio, un fichero o una ruta que aún no
// existe (se atribuye al repositorio de su directorio existente más profundo). path
// debe ser absoluta: el repositorio nunca se elige desde el directorio del proceso.
func RepositoryRoot(ctx context.Context, path string) (string, error) {
	r, err := openRepository(ctx, path)
	if err != nil {
		return "", err
	}
	return r.root, nil
}
