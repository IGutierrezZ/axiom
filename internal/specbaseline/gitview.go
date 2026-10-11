package specbaseline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// gitTimeout acota cada invocación de git, y gitOutputLimit la salida que se lee
// de ella (D-15).
const (
	gitTimeout     = 30 * time.Second
	gitOutputLimit = 64 << 20
	gitStderrLimit = 64 << 10
)

// gitRedirectingVariables son las variables con las que un proceso padre (un hook
// de git, otra herramienta) fija el repositorio sobre el que actúa git. Se retiran
// del entorno: la selección del repositorio es siempre la de la ruta que se
// pregunta, nunca la del proceso (design §8, «Selección de repositorio git»).
var gitRedirectingVariables = map[string]bool{
	"GIT_DIR":                          true,
	"GIT_WORK_TREE":                    true,
	"GIT_INDEX_FILE":                   true,
	"GIT_COMMON_DIR":                   true,
	"GIT_PREFIX":                       true,
	"GIT_NAMESPACE":                    true,
	"GIT_OBJECT_DIRECTORY":             true,
	"GIT_ALTERNATE_OBJECT_DIRECTORIES": true,
}

// NotGitRepositoryError es la parada (a) de REQ-24.49: el directorio de un
// documento afectado no está dentro de un repositorio git.
type NotGitRepositoryError struct {
	Path string
}

func (e *NotGitRepositoryError) Error() string {
	return fmt.Sprintf("%s no está dentro de un repositorio git: la verificación por diff de la integración requiere un repositorio git; versiona el directorio con `git init` y confirma su estado actual, o declara la ruta en un repositorio versionado", e.Path)
}

// gitRepository es un repositorio git de solo lectura. root es la raíz absoluta
// que devuelve `git rev-parse --show-toplevel` desde una ruta absoluta limpia, y
// es el directorio de trabajo de todas las invocaciones: nunca se usa un
// directorio relativo ni el del proceso.
type gitRepository struct {
	root string
}

// gitEnvironment es el entorno de git: el del proceso sin las variables que
// redirigen el repositorio, con `GIT_OPTIONAL_LOCKS=0` y sin preguntas
// interactivas. `GIT_OPTIONAL_LOCKS=0` evita que `git status` refresque el índice,
// pero no que lo haga `git diff` (git 2.41, comprobado): ese comando se ejecuta
// con `withScratchIndex`.
func gitEnvironment() []string {
	var env []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if gitRedirectingVariables[strings.ToUpper(name)] {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
}

// limitedBuffer acumula la salida de un proceso hasta un límite y descarta el
// resto, anotando que se excedió.
type limitedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.buffer.Len(); len(p) > room {
		b.exceeded = true
		if room > 0 {
			b.buffer.Write(p[:room])
		}
		return len(p), nil
	}
	return b.buffer.Write(p)
}

// gitResult es lo que devuelve una invocación de git.
type gitResult struct {
	stdout []byte
	stderr string
	exit   int
}

// execGit ejecuta git con cmd.Dir = dir, tiempo máximo y límite de salida. Un
// código de salida distinto de cero no es un error de ejecución: lo decide quien
// llama a partir de gitResult.exit.
func execGit(ctx context.Context, dir string, args ...string) (gitResult, error) {
	return execGitEnv(ctx, dir, nil, args...)
}

// execGitEnv es execGit con variables de entorno añadidas solo para esa
// invocación. Se añaden después de gitEnvironment, que retira las que redirigen el
// repositorio: así una variable como GIT_INDEX_FILE solo la fija quien llama, nunca
// el proceso.
func execGitEnv(ctx context.Context, dir string, extraEnv []string, args ...string) (gitResult, error) {
	binary, err := exec.LookPath("git")
	if err != nil {
		return gitResult{}, fmt.Errorf("git no está disponible en el PATH: la verificación por diff de la integración lo requiere: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = append(gitEnvironment(), extraEnv...)
	stdout := &limitedBuffer{limit: gitOutputLimit}
	stderr := &limitedBuffer{limit: gitStderrLimit}
	cmd.Stdout, cmd.Stderr = stdout, stderr

	runErr := cmd.Run()
	result := gitResult{stdout: stdout.buffer.Bytes(), stderr: strings.TrimSpace(stderr.buffer.String())}
	var exit *exec.ExitError
	switch {
	case ctx.Err() != nil:
		return result, fmt.Errorf("git %s en %s superó el tiempo máximo de %s", args[0], dir, gitTimeout)
	case errors.As(runErr, &exit):
		result.exit = exit.ExitCode()
	case runErr != nil:
		return result, fmt.Errorf("ejecutar git %s en %s: %w", args[0], dir, runErr)
	}
	if stdout.exceeded {
		return result, fmt.Errorf("la salida de git %s en %s supera el límite de %d bytes", args[0], dir, gitOutputLimit)
	}
	return result, nil
}

// withScratchIndex ejecuta fn con una ruta de índice temporal para GIT_INDEX_FILE,
// y la borra al terminar, también si fn falla. Con las marcas de tiempo del índice
// desfasadas (un `touch` de un fichero seguido tras el commit), `git diff` refresca
// el índice y lo reescribe aunque se invoque con GIT_OPTIONAL_LOCKS=0 (git 2.41,
// comprobado también con `--no-optional-locks`): el comando contradiría la promesa de
// solo lectura. Git refresca la copia y el índice real no se toca.
//
//   - La copia está en un directorio privado del temporal del sistema, fuera del
//     repositorio, y se crea con `os.MkdirTemp` en lugar de con `os.CreateTemp`: así el
//     `index.lock` que git crea junto al índice cuando lo escribe cae también en él y
//     se borra con el directorio.
//   - El índice real se localiza con `git rev-parse --git-path index`, resuelto contra
//     la raíz del repositorio: en un worktree enlazado o con GIT_DIR no es
//     `<raíz>/.git/index`. La copia conserva la fecha del original, de modo que git
//     decide igual qué entradas son «racy».
//   - Sin índice real (repositorio recién creado o sin índice) no se copia nada: la ruta
//     apunta a un fichero que no existe dentro del directorio privado, que git lee como
//     un índice vacío, igual que con el repositorio sin índice, y nada se crea en el
//     repositorio.
//   - Con `core.splitIndex` el índice enlaza con `sharedindex.*` en el directorio git;
//     git los resuelve desde ese directorio y no desde la ruta del índice, así que la
//     copia funciona (las pruebas comprueban que ni el índice ni los compartidos
//     cambian). Si falta un índice compartido, git falla y el error llega a quien
//     llama; no se ignora.
func (r gitRepository) withScratchIndex(ctx context.Context, fn func(indexFile string) error) error {
	located, err := execGit(ctx, r.root, "rev-parse", "--git-path", "index")
	if err != nil {
		return err
	}
	if located.exit != 0 {
		return fmt.Errorf("git rev-parse --git-path index en %s terminó con el código %d: %s", r.root, located.exit, located.stderr)
	}
	realIndex := filepath.FromSlash(strings.TrimRight(string(located.stdout), "\r\n"))
	if realIndex == "" {
		return fmt.Errorf("git rev-parse --git-path index en %s no devolvió ninguna ruta", r.root)
	}
	if !filepath.IsAbs(realIndex) {
		realIndex = filepath.Join(r.root, realIndex)
	}

	scratch, err := os.MkdirTemp("", "axiom-indice-*")
	if err != nil {
		return fmt.Errorf("crear la carpeta temporal del índice de git de %s: %w", r.root, err)
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	indexFile := filepath.Join(scratch, "index")
	if err := copyIndex(realIndex, indexFile); err != nil {
		return fmt.Errorf("copiar el índice de git de %s: %w", r.root, err)
	}
	return fn(indexFile)
}

// copyIndex copia el índice src en dst conservando su fecha de modificación. Un src
// que no existe no es un error: no se crea dst.
func copyIndex(src, dst string) error {
	in, err := os.Open(src)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chtimes(dst, info.ModTime(), info.ModTime())
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

// openRepository localiza el repositorio git que contiene la ruta: ejecuta
// `git rev-parse --show-toplevel` con cmd.Dir en el directorio existente más
// profundo de la ruta, que debe ser absoluta. Un repositorio anidado es el
// repositorio de sus ficheros. Sin repositorio devuelve *NotGitRepositoryError.
func openRepository(ctx context.Context, p string) (gitRepository, error) {
	if !filepath.IsAbs(p) {
		return gitRepository{}, fmt.Errorf("la ruta %q no es absoluta: el repositorio git se elige siempre desde una ruta absoluta y nunca desde el directorio de trabajo del proceso", p)
	}
	clean := filepath.Clean(p)
	result, err := execGit(ctx, deepestExistingDir(clean), "rev-parse", "--show-toplevel")
	if err != nil {
		return gitRepository{}, err
	}
	switch {
	case result.exit == 0:
	case strings.Contains(result.stderr, "dubious ownership"):
		return gitRepository{}, fmt.Errorf("git rechaza el repositorio de %s por su propietario (safe.directory): %s", clean, result.stderr)
	case result.exit == 128:
		return gitRepository{}, &NotGitRepositoryError{Path: clean}
	default:
		return gitRepository{}, fmt.Errorf("git rev-parse --show-toplevel en %s terminó con el código %d: %s", clean, result.exit, result.stderr)
	}
	top := strings.TrimSpace(string(result.stdout))
	if top == "" {
		return gitRepository{}, &NotGitRepositoryError{Path: clean}
	}
	return gitRepository{root: resolveDeepest(filepath.FromSlash(top))}, nil
}

// head devuelve el commit de HEAD, o la cadena vacía si el repositorio no tiene
// commits (en cuyo caso se compara con el árbol vacío).
func (r gitRepository) head(ctx context.Context) (string, error) {
	result, err := execGit(ctx, r.root, "rev-parse", "--verify", "--quiet", "HEAD^{commit}")
	if err != nil {
		return "", err
	}
	switch result.exit {
	case 0:
		return strings.TrimSpace(string(result.stdout)), nil
	case 1:
		return "", nil
	}
	return "", fmt.Errorf("git rev-parse HEAD en %s terminó con el código %d: %s", r.root, result.exit, result.stderr)
}

// statusEntry es una entrada de `git status --porcelain=v1`: el código XY de
// índice y árbol de trabajo (`??` sin seguimiento) y la ruta relativa a la raíz.
type statusEntry struct {
	Path string // relativa a la raíz del repositorio, con '/'
	Code string
}

// status devuelve el estado del repositorio con `git status --porcelain=v1 -z
// --untracked-files=all`: sin entrecomillado y con cada fichero sin seguimiento por
// separado. El origen de un renombrado no es una entrada.
func (r gitRepository) status(ctx context.Context) ([]statusEntry, error) {
	result, err := execGit(ctx, r.root, "-c", "core.fsmonitor=false", "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	if result.exit != 0 {
		return nil, fmt.Errorf("git status en %s terminó con el código %d: %s", r.root, result.exit, result.stderr)
	}
	var entries []statusEntry
	fields := bytes.Split(result.stdout, []byte{0})
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
		if strings.ContainsAny(code, "RC") {
			i++ // con -z, un renombrado o una copia añade el origen como otro campo
		}
	}
	return entries, nil
}

// upstreamState es el estado de una rama frente a su upstream, según las
// referencias locales. Notes explica por qué no se pudo comprobar.
type upstreamState struct {
	Behind int
	Notes  []string
}

// upstream cuenta con `git rev-list --count HEAD..@{upstream}` los commits que la
// rama lleva por detrás. Usa solo las referencias locales: Axiom nunca actualiza
// el remoto. Sin upstream, con HEAD desacoplado o sin commits devuelve una nota y
// no bloquea (design §8, «Estado del push»).
func (r gitRepository) upstream(ctx context.Context) upstreamState {
	if branch, err := execGit(ctx, r.root, "symbolic-ref", "--quiet", "HEAD"); err == nil && branch.exit == 1 {
		return upstreamState{Notes: []string{"HEAD está desacoplado: no hay rama con upstream con la que comparar y no se comprueba si va por detrás"}}
	}
	if head, err := r.head(ctx); err == nil && head == "" {
		return upstreamState{Notes: []string{"el repositorio no tiene commits: no hay upstream con el que comparar"}}
	}
	tracking, err := execGit(ctx, r.root, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil || tracking.exit != 0 {
		return upstreamState{Notes: []string{"la rama actual no tiene upstream configurado: no se comprueba si va por detrás"}}
	}
	counted, err := execGit(ctx, r.root, "rev-list", "--count", "HEAD..@{upstream}")
	if err != nil || counted.exit != 0 {
		return upstreamState{Notes: []string{"no se pudo contar los commits que la rama lleva por detrás de su upstream"}}
	}
	behind, err := strconv.Atoi(strings.TrimSpace(string(counted.stdout)))
	if err != nil {
		return upstreamState{Notes: []string{"no se pudo interpretar el número de commits que la rama lleva por detrás de su upstream"}}
	}
	return upstreamState{Behind: behind}
}
