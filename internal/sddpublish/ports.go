// Package sddpublish es la publicación ligera de la documentación de un cambio SDD:
// `axiom sdd publish` (REQ-27.5 a REQ-27.11, DD-08 a DD-10). Es el único código del
// producto que ejecuta `git commit` y `git push`, siempre a través de
// internal/gitcommon/exec.go, y solo lo alcanza `internal/cli/sdd_publish.go`: una
// prueba de arquitectura lo fija (DD-14, REQ-27.11). El puerto de escritura de
// `axiom branch` sigue sin ningún método que publique (REQ-25.1).
//
// Qué se publica. Solo la lista cerrada de rutas de documentación de internal/docpaths,
// confirmada con `git add --all` y `git commit --only` sobre esas raíces, de modo que el
// código preparado sigue preparado y el no preparado, intacto. Lo que sube es una sola
// rama, `refs/heads/<b>:refs/heads/<b>` a `origin`, sin forzar y sin saltarse hooks. Los
// commits de código que aún no están en el remoto nunca se suben: se informan como la
// negativa tipada `code_commits_unpublished` y los sube la persona.
package sddpublish

import (
	"context"
	"fmt"
)

// Git son las operaciones de git de la publicación. dir es siempre la raíz git del destino
// (PublicationTarget.GitRoot), nunca el directorio del proceso. Es el único puerto del
// producto con Commit y Push (REQ-27.11). Las rutas son relativas a la raíz git, con «/»,
// y el adaptador las cita como literales; las referencias y las ramas se validan antes de
// pasarlas a git.
type Git interface {
	// RemoteConfigured dice si el remoto tiene URL configurada (`git config --get remote.<r>.url`).
	RemoteConfigured(ctx context.Context, dir, remote string) (bool, error)
	// ResolveCommit resuelve ref a su commit (`git rev-parse --verify --quiet <ref>^{commit}`).
	// Una referencia inexistente no es un error: devuelve ok = false.
	ResolveCommit(ctx context.Context, dir, ref string) (sha string, ok bool, err error)
	// DirtyPaths devuelve las rutas con cambios (modificadas, borradas, preparadas o sin
	// seguimiento, una por fichero) bajo las raíces dadas (`git status --porcelain=v1 -z
	// --untracked-files=all --no-renames`).
	DirtyPaths(ctx context.Context, dir string, paths []string) ([]string, error)
	// IdentityConfigured dice si git puede componer la identidad del autor y del confirmante
	// (`git var GIT_AUTHOR_IDENT` y `GIT_COMMITTER_IDENT`).
	IdentityConfigured(ctx context.Context, dir string) (bool, error)
	// HookPresent dice si el hook existe y git lo ejecutaría, respetando `core.hooksPath`
	// (`git rev-parse --git-path hooks/<hook>`).
	HookPresent(ctx context.Context, dir, hook string) (bool, error)
	// UnpublishedCommits devuelve, del más antiguo al más reciente, los commits alcanzables desde
	// head que no lo son desde ninguna referencia de exclude. Devuelve como mucho limit+1 SHA: si
	// hay más de limit, quien llama sabe que el rango es demasiado grande.
	UnpublishedCommits(ctx context.Context, dir, head string, exclude []string, limit int) ([]string, error)
	// CommitPaths devuelve las rutas que toca un commit; en una fusión (`--cc`), solo las que la
	// propia fusión resolvió.
	CommitPaths(ctx context.Context, dir, sha string) ([]string, error)
	// Commit confirma solo las rutas dadas: `git add --all -- <paths>` y
	// `git commit --only -m <mensaje> -- <paths>`, con exclude como globs relativos a la raíz
	// que se dejan fuera. Devuelve el SHA del commit. Un fallo es un *CommandError.
	Commit(ctx context.Context, dir, message string, paths, exclude []string) (string, error)
	// Push ejecuta `git push --porcelain [--set-upstream] <remote> refs/heads/<b>:refs/heads/<b>`,
	// sin `+`, sin `--force*` y sin `--no-verify`. Un rechazo no es un error: llega en el
	// resultado, que clasifica quien llama.
	Push(ctx context.Context, dir, remote, branch string, setUpstream bool) (PushOutcome, error)
}

// PushOutcome es lo que devolvió `git push`: el código de salida, la salida `--porcelain` y
// el `stderr` (con `LC_ALL=C`), o TimedOut si se superó el tiempo máximo.
type PushOutcome struct {
	ExitCode int
	Stdout   string
	Stderr   string
	TimedOut bool
}

// CommandError es el fallo de un comando de git de la publicación, con lo que hace falta para
// clasificarlo. Step es `add` o `commit`.
type CommandError struct {
	Step     string
	ExitCode int
	Stderr   string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("git %s terminó con el código %d: %s", e.Step, e.ExitCode, e.Stderr)
}
