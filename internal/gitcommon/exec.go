package gitcommon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Límites de cada invocación de git (design §2.1, D-01): el tiempo máximo y los
// bytes que se leen de la salida estándar y de la de error. Son variables de
// paquete solo para que las pruebas los reduzcan sin esperar 30 s ni producir
// 64 MiB; ningún código de producción las cambia.
var (
	runTimeout  = 30 * time.Second
	stdoutLimit = 64 << 20
	stderrLimit = 64 << 10
)

// Observe, cuando no es nula, recibe el directorio y los argumentos de cada
// invocación de git antes de ejecutarla (design §4.6, G-7). Es nula en producción:
// solo la instalan las pruebas, que verifican con ella que `axiom branch status`
// no lanza `fetch`, `pull`, `push` ni `ls-remote`. Run le entrega una copia de los
// argumentos, de modo que el observador no puede alterar lo que se ejecuta.
var Observe func(dir string, args []string)

// ErrNotGitRepository lo devuelven TopLevel y CommonDir cuando el directorio no
// pertenece a ningún repositorio git. `axiom branch` lo traduce al código tipado
// `not_a_git_repository`.
var ErrNotGitRepository = errors.New("no pertenece a ningún repositorio git")

// redirectingVariables son las variables con las que un proceso padre (un hook de
// git, otra herramienta) fija el repositorio sobre el que actúa git. Se retiran del
// entorno: la selección del repositorio es siempre la del directorio que se pide,
// nunca la del proceso (design §8, «Selección de repositorio git»). Es la misma
// lista que usa internal/specbaseline/gitview.go.
var redirectingVariables = map[string]bool{
	"GIT_DIR":                          true,
	"GIT_WORK_TREE":                    true,
	"GIT_INDEX_FILE":                   true,
	"GIT_COMMON_DIR":                   true,
	"GIT_PREFIX":                       true,
	"GIT_NAMESPACE":                    true,
	"GIT_OBJECT_DIRECTORY":             true,
	"GIT_ALTERNATE_OBJECT_DIRECTORIES": true,
}

// gitEnvironment es el entorno de git: el del proceso sin las variables que
// redirigen el repositorio, con `GIT_OPTIONAL_LOCKS=0` para que las lecturas no
// refresquen el índice, sin preguntas interactivas y con `LC_ALL=C` para que los
// mensajes de git que se clasifican («not a git repository») no dependan del
// idioma del usuario. Las variables fijadas van al final: con claves repetidas,
// exec.Cmd usa la última.
func gitEnvironment() []string {
	var env []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if redirectingVariables[strings.ToUpper(name)] {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
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

// Result es lo que devuelve una invocación de git. Un código de salida distinto de
// cero no es un error de ejecución: lo decide quien llama a partir de ExitCode.
type Result struct {
	Stdout   []byte
	Stderr   string
	ExitCode int
}

// Run ejecuta git con cmd.Dir = dir y es el único `exec.Command` permitido en
// `internal/branch/**` y en este paquete (REQ-25.1, G-7: una prueba de arquitectura
// lo fija). dir debe ser un directorio absoluto y existente: git nunca se ejecuta
// desde el directorio del proceso. Aplica tiempo máximo y límite de salida, retira
// las variables que redirigen el repositorio (gitEnvironment) y avisa al
// observador, si lo hay, antes de lanzar el proceso.
//
// Devuelve un error cuando no se pudo ejecutar git, se superó el tiempo o el límite
// de salida, o quien llama canceló el contexto; un código de salida distinto de cero
// llega en Result.ExitCode con el mensaje de git en Result.Stderr.
func Run(ctx context.Context, dir string, args ...string) (Result, error) {
	if len(args) == 0 {
		return Result{}, errors.New("git sin argumentos: indica el subcomando de git que se ejecuta")
	}
	if !filepath.IsAbs(dir) {
		return Result{}, fmt.Errorf("la ruta %q no es absoluta: git se ejecuta siempre desde un directorio absoluto y nunca desde el directorio de trabajo del proceso", dir)
	}
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return Result{}, fmt.Errorf("el directorio %s no existe: git se ejecuta siempre desde un directorio existente", dir)
	case err != nil:
		return Result{}, fmt.Errorf("no se pudo comprobar el directorio %s: %w", dir, err)
	case !info.IsDir():
		return Result{}, fmt.Errorf("%s no es un directorio: git se ejecuta siempre desde un directorio", dir)
	}
	operation := "git " + strings.Join(args, " ")

	if Observe != nil {
		Observe(dir, append([]string(nil), args...))
	}
	binary, err := exec.LookPath("git")
	if err != nil {
		return Result{}, fmt.Errorf("git no está disponible en el PATH: instala git para ejecutar %s: %w", operation, err)
	}

	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, binary, args...)
	cmd.Dir = dir
	cmd.Env = gitEnvironment()
	stdout := &limitedBuffer{limit: stdoutLimit}
	stderr := &limitedBuffer{limit: stderrLimit}
	cmd.Stdout, cmd.Stderr = stdout, stderr

	runErr := cmd.Run()
	result := Result{Stdout: stdout.buffer.Bytes(), Stderr: strings.TrimSpace(stderr.buffer.String())}
	var exit *exec.ExitError
	switch {
	case runErr != nil && ctx.Err() != nil:
		return result, fmt.Errorf("%s en %s se interrumpió: %w", operation, dir, ctx.Err())
	case runErr != nil && runCtx.Err() != nil:
		return result, fmt.Errorf("%s en %s superó el tiempo máximo de %s", operation, dir, runTimeout)
	case errors.As(runErr, &exit):
		result.ExitCode = exit.ExitCode()
	case runErr != nil:
		return result, fmt.Errorf("ejecutar %s en %s: %w", operation, dir, runErr)
	}
	if stdout.exceeded {
		return result, fmt.Errorf("la salida de %s en %s supera el límite de %d bytes", operation, dir, stdoutLimit)
	}
	return result, nil
}

// TopLevel devuelve la raíz absoluta del árbol de trabajo que contiene dir
// (`git rev-parse --show-toplevel`, REQ-25.6): en un worktree enlazado es la raíz de
// ese worktree. Fuera de un repositorio devuelve ErrNotGitRepository.
func TopLevel(ctx context.Context, dir string) (string, error) {
	return pathFromGit(ctx, dir, "rev-parse", "--show-toplevel")
}

// CommonDir devuelve el git common dir absoluto del repositorio que contiene dir
// (`git rev-parse --path-format=absolute --git-common-dir`, REQ-25.14): el mismo
// directorio desde el checkout principal y desde cualquier worktree enlazado, sin
// suponer que se llama `.git`. Fuera de un repositorio devuelve ErrNotGitRepository.
func CommonDir(ctx context.Context, dir string) (string, error) {
	return pathFromGit(ctx, dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
}

// pathFromGit ejecuta una consulta de git que responde con una ruta y la devuelve
// con el separador y la grafía del sistema (git imprime `C:/…` en Windows).
func pathFromGit(ctx context.Context, dir string, args ...string) (string, error) {
	result, err := Run(ctx, dir, args...)
	if err != nil {
		return "", err
	}
	operation := "git " + strings.Join(args, " ")
	if result.ExitCode != 0 {
		// Con LC_ALL=C el mensaje de git no depende del idioma. Otros fallos con el
		// mismo código 128 (repositorio desnudo, propietario dudoso) no son «no es un
		// repositorio»: conservan el resumen de git.
		if strings.Contains(result.Stderr, "not a git repository") {
			return "", fmt.Errorf("%s: %w", dir, ErrNotGitRepository)
		}
		return "", fmt.Errorf("%s en %s terminó con el código %d: %s", operation, dir, result.ExitCode, result.Stderr)
	}
	line, _, _ := strings.Cut(string(result.Stdout), "\n")
	line = strings.TrimRight(line, "\r")
	if line == "" {
		return "", fmt.Errorf("%s en %s no devolvió ninguna ruta", operation, dir)
	}
	return filepath.Clean(filepath.FromSlash(line)), nil
}
