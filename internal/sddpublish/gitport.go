package sddpublish

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/gitcommon"
	"github.com/IGutierrezZ/axiom/v3/internal/gitref"
)

// pushTimeout es el tiempo máximo de `git push` (DD-08): la red necesita más margen que una
// lectura local. Es una variable de paquete para que las pruebas lo reduzcan; ningún código
// de producción la cambia.
var pushTimeout = 120 * time.Second

// GitCLI es el adaptador del puerto Git sobre gitcommon.Run y gitcommon.RunWithTimeout, el
// único punto que ejecuta git en el producto (REQ-27.11). No tiene estado: cada método ejecuta
// git desde el directorio que recibe, que es siempre la raíz git del destino.
type GitCLI struct{}

// NewGit devuelve el adaptador real del puerto Git.
func NewGit() *GitCLI { return &GitCLI{} }

var _ Git = (*GitCLI)(nil)

var (
	hookName = regexp.MustCompile(`^[a-z][a-z-]*$`)
	shaName  = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
)

// literalPathspec cita una ruta relativa a la raíz como literal desde la raíz del repositorio
// (`:(top,literal)<ruta>`), de modo que ningún carácter de la ruta se interprete como patrón ni
// como opción. Rechaza la ruta vacía, la raíz, las absolutas, las que suben con `..` y las que
// llevan caracteres de control.
func literalPathspec(rel string) (string, error) {
	if err := checkRelativePath(rel); err != nil {
		return "", err
	}
	return ":(top,literal)" + rel, nil
}

func excludePathspec(glob string) (string, error) {
	if err := checkRelativePath(glob); err != nil {
		return "", err
	}
	return ":(top,exclude,glob)" + glob, nil
}

func checkRelativePath(rel string) error {
	invalid := func(reason string) error {
		return fmt.Errorf("la ruta %s no es válida para git: %s", strconv.Quote(rel), reason)
	}
	switch {
	case rel == "" || rel == ".":
		return invalid("debe nombrar algo distinto de la raíz del repositorio")
	case strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) || filepath.IsAbs(rel):
		return invalid("debe ser relativa a la raíz del repositorio")
	case strings.ContainsFunc(rel, func(r rune) bool { return r < 0x20 || r == 0x7f }):
		return invalid("no puede llevar caracteres de control")
	}
	for _, segment := range strings.Split(rel, "/") {
		if segment == ".." {
			return invalid("no puede llevar segmentos «..»")
		}
	}
	return nil
}

func literalPathspecs(paths []string) ([]string, error) {
	specs := make([]string, 0, len(paths))
	for _, rel := range paths {
		spec, err := literalPathspec(rel)
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

// checkRevision valida una referencia que se pasa a git como revisión: un nombre de referencia
// válido (que además no empieza por «-»).
func checkRevision(ref string) error {
	if err := gitref.ValidRefName(ref); err != nil {
		return fmt.Errorf("la referencia %s no es válida para git: %w", strconv.Quote(ref), err)
	}
	return nil
}

// RemoteConfigured implementa Git.
func (*GitCLI) RemoteConfigured(ctx context.Context, dir, remote string) (bool, error) {
	if err := checkRevision(remote); err != nil {
		return false, err
	}
	result, err := gitcommon.Run(ctx, dir, "config", "--get", "remote."+remote+".url")
	if err != nil {
		return false, err
	}
	switch result.ExitCode {
	case 0:
		return strings.TrimSpace(string(result.Stdout)) != "", nil
	case 1:
		return false, nil
	default:
		return false, commandFailed(dir, result, "config", "--get", "remote."+remote+".url")
	}
}

// ResolveCommit implementa Git.
func (*GitCLI) ResolveCommit(ctx context.Context, dir, ref string) (string, bool, error) {
	if err := checkRevision(ref); err != nil {
		return "", false, err
	}
	result, err := gitcommon.Run(ctx, dir, "rev-parse", "--verify", "--quiet", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return "", false, err
	}
	switch result.ExitCode {
	case 0:
		sha := strings.TrimSpace(string(result.Stdout))
		return sha, sha != "", nil
	case 1:
		return "", false, nil
	default:
		return "", false, commandFailed(dir, result, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	}
}

// DirtyPaths implementa Git.
func (*GitCLI) DirtyPaths(ctx context.Context, dir string, paths []string) ([]string, error) {
	specs, err := literalPathspecs(paths)
	if err != nil {
		return nil, err
	}
	args := append([]string{"status", "--porcelain=v1", "-z", "--untracked-files=all", "--no-renames", "--"}, specs...)
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, commandFailed(dir, result, args...)
	}
	var dirty []string
	for _, entry := range strings.Split(string(result.Stdout), "\x00") {
		// «XY ruta»: dos columnas de estado, un espacio y la ruta, relativa a la raíz git.
		if len(entry) > 3 {
			dirty = append(dirty, entry[3:])
		}
	}
	return dirty, nil
}

// IdentityConfigured implementa Git.
func (*GitCLI) IdentityConfigured(ctx context.Context, dir string) (bool, error) {
	for _, variable := range []string{"GIT_AUTHOR_IDENT", "GIT_COMMITTER_IDENT"} {
		result, err := gitcommon.Run(ctx, dir, "var", variable)
		if err != nil {
			return false, err
		}
		if result.ExitCode != 0 || strings.TrimSpace(string(result.Stdout)) == "" {
			return false, nil
		}
	}
	return true, nil
}

// HookPresent implementa Git.
func (*GitCLI) HookPresent(ctx context.Context, dir, hook string) (bool, error) {
	if !hookName.MatchString(hook) {
		return false, fmt.Errorf("el nombre de hook %s no es válido", strconv.Quote(hook))
	}
	result, err := gitcommon.Run(ctx, dir, "rev-parse", "--git-path", "hooks/"+hook)
	if err != nil {
		return false, err
	}
	if result.ExitCode != 0 {
		return false, commandFailed(dir, result, "rev-parse", "--git-path", "hooks/"+hook)
	}
	line, _, _ := strings.Cut(string(result.Stdout), "\n")
	line = strings.TrimRight(line, "\r")
	if line == "" {
		return false, nil
	}
	hookPath := filepath.FromSlash(line)
	if !filepath.IsAbs(hookPath) {
		hookPath = filepath.Join(dir, hookPath)
	}
	info, err := os.Stat(hookPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("comprobar el hook %s: %w", hookPath, err)
	case !info.Mode().IsRegular():
		return false, nil
	case runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0:
		// Git ignora un hook sin permiso de ejecución.
		return false, nil
	}
	return true, nil
}

// UnpublishedCommits implementa Git.
func (*GitCLI) UnpublishedCommits(ctx context.Context, dir, head string, exclude []string, limit int) ([]string, error) {
	if limit < 1 {
		return nil, errors.New("el límite de commits debe ser positivo")
	}
	if err := checkRevision(head); err != nil {
		return nil, err
	}
	args := []string{"rev-list", "--reverse", "--max-count=" + strconv.Itoa(limit+1), "--end-of-options", head}
	for _, ref := range exclude {
		if err := checkRevision(ref); err != nil {
			return nil, err
		}
		args = append(args, "^"+ref)
	}
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, commandFailed(dir, result, args...)
	}
	return splitLines(string(result.Stdout)), nil
}

// CommitPaths implementa Git.
func (*GitCLI) CommitPaths(ctx context.Context, dir, sha string) ([]string, error) {
	if !shaName.MatchString(sha) {
		return nil, fmt.Errorf("el commit %s no es un SHA válido", strconv.Quote(sha))
	}
	args := []string{"diff-tree", "-r", "--no-commit-id", "--name-only", "-z", "--no-renames", "--root", "--cc", sha}
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, commandFailed(dir, result, args...)
	}
	var paths []string
	for _, entry := range strings.Split(string(result.Stdout), "\x00") {
		if entry != "" {
			paths = append(paths, entry)
		}
	}
	return paths, nil
}

// Commit implementa Git. Son dos pasos: `git add --all -- <rutas>`, que solo toca el índice de
// esas rutas e incluye los borrados, y `git commit --only -m <mensaje> -- <rutas>`, que confirma
// solo esas rutas y deja preparado lo demás. Nunca `-a`, `--no-verify` ni `--amend`.
func (*GitCLI) Commit(ctx context.Context, dir, message string, paths, exclude []string) (string, error) {
	specs, err := literalPathspecs(paths)
	if err != nil {
		return "", err
	}
	if len(specs) == 0 {
		return "", errors.New("no hay rutas que confirmar")
	}
	for _, glob := range exclude {
		spec, err := excludePathspec(glob)
		if err != nil {
			return "", err
		}
		specs = append(specs, spec)
	}
	add := append([]string{"add", "--all", "--"}, specs...)
	if err := runStep(ctx, dir, "add", add); err != nil {
		return "", err
	}
	commit := append([]string{"commit", "--only", "-m", message, "--"}, specs...)
	if err := runStep(ctx, dir, "commit", commit); err != nil {
		return "", err
	}
	head, err := gitcommon.Run(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	if head.ExitCode != 0 {
		return "", commandFailed(dir, head, "rev-parse", "HEAD")
	}
	return strings.TrimSpace(string(head.Stdout)), nil
}

func runStep(ctx context.Context, dir, step string, args []string) error {
	result, err := gitcommon.Run(ctx, dir, args...)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &CommandError{Step: step, ExitCode: result.ExitCode, Stderr: commandOutput(result)}
	}
	return nil
}

// commandOutput es lo que git dijo al fallar: `stderr` y, si git escribió el motivo en la salida
// estándar (`git commit` lo hace con «nothing to commit»), esta también.
func commandOutput(result gitcommon.Result) string {
	stdout := strings.TrimSpace(string(result.Stdout))
	switch {
	case result.Stderr == "":
		return stdout
	case stdout == "":
		return result.Stderr
	default:
		return stdout + "\n" + result.Stderr
	}
}

// Push implementa Git.
func (*GitCLI) Push(ctx context.Context, dir, remote, branch string, setUpstream bool) (PushOutcome, error) {
	if err := checkRevision(remote); err != nil {
		return PushOutcome{}, err
	}
	if err := checkRevision(branch); err != nil {
		return PushOutcome{}, err
	}
	args := []string{"push", "--porcelain"}
	if setUpstream {
		args = append(args, "--set-upstream")
	}
	args = append(args, remote, "refs/heads/"+branch+":refs/heads/"+branch)
	result, err := gitcommon.RunWithTimeout(ctx, dir, pushTimeout, args...)
	switch {
	case errors.Is(err, gitcommon.ErrTimeout):
		return PushOutcome{TimedOut: true, Stdout: string(result.Stdout), Stderr: result.Stderr}, nil
	case err != nil:
		return PushOutcome{}, err
	}
	return PushOutcome{ExitCode: result.ExitCode, Stdout: string(result.Stdout), Stderr: result.Stderr}, nil
}

func splitLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimRight(line, "\r"); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func commandFailed(dir string, result gitcommon.Result, args ...string) error {
	return fmt.Errorf("git %s en %s terminó con el código %d: %s", strings.Join(args, " "), dir, result.ExitCode, result.Stderr)
}
