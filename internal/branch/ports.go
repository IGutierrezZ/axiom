package branch

import "context"

// Puertos del dominio (design §4.1). El motor los consume y los adaptadores los
// implementan: `branch/gitcli` implementa Git sobre gitcommon.Run, `branch/registry`
// implementa Registry, el hub aporta Prefs y la CLI aporta LineageProbe. Separarlos
// permite probar el motor con dobles y fija que ningún método publica: el puerto Git
// no tiene `push`, y Fetch es la única operación remota (REQ-25.1, REQ-25.10).

// Git son las operaciones de git que necesita el motor. dir es siempre el checkout
// principal del repositorio (nunca el directorio del proceso) y toda creación o
// borrado se ejecuta desde él.
type Git interface {
	TopLevel(ctx context.Context, dir string) (string, error)
	CommonDir(ctx context.Context, dir string) (string, error)
	ResolveCommit(ctx context.Context, dir, ref string) (sha string, ok bool, err error)
	RefKind(ctx context.Context, dir, ref string) (string, error) // "remote-tracking" | "local"
	BranchExists(ctx context.Context, dir, name string) (bool, error)
	Worktrees(ctx context.Context, dir string) ([]WorktreeEntry, error)
	DirtyPaths(ctx context.Context, dir string) ([]string, error)
	CreateBranch(ctx context.Context, dir, name, startPoint string) error
	CreateTrackingBranch(ctx context.Context, dir, name, remoteRef string) error // adopta refs/remotes/origin/<name>, sin red
	AddWorktree(ctx context.Context, dir, path, branch string) error
	Switch(ctx context.Context, dir, branch string) error
	RemoveWorktree(ctx context.Context, dir, path string, force bool) error
	RemoveEmptyDir(ctx context.Context, path string) error // solo si está vacío: gitcli.ErrDirNotEmpty en otro caso
	DeleteBranch(ctx context.Context, dir, name string) error
	IsAncestor(ctx context.Context, dir, ancestor, descendant string) (bool, error)
	MergeTreeEquivalent(ctx context.Context, dir, target, branch string) (equivalent, supported bool, err error)
	AheadBehind(ctx context.Context, dir, base, branch string) (ahead, behind int, err error)
	TreeDirs(ctx context.Context, dir, rev, relPath string) ([]string, bool, error)                     // dirs, exists, err
	FileAt(ctx context.Context, dir, rev, relPath string) ([]byte, bool, error)                         // contenido, exists, err
	RevList(ctx context.Context, dir string, firstParent bool, revisions ...string) ([]RevEntry, error) // rev-list --parents [--first-parent] <revs> --
	CommitFiles(ctx context.Context, dir string, revisions ...string) ([]CommitPaths, error)            // log -z --no-renames --name-only
	BranchCreation(ctx context.Context, dir, name string) (sha, from string, ok bool, err error)        // reflog show, entrada más antigua
	Fetch(ctx context.Context, dir, remote string) error                                                // la única operación remota
}

// Registry es el registro local de los planes materializados, en el git common dir
// del repositorio del planning home (REQ-25.14).
type Registry interface {
	Lock(ctx context.Context, commonDir string, lane Lane, change string) (unlock func(), err error)
	Load(commonDir string, lane Lane, change string) (*Record, error) // nil, nil cuando no existe
	Save(commonDir string, rec Record) error                          // no escribe si los bytes no cambian
	Delete(commonDir string, lane Lane, change string) error
	List(commonDir string, lane Lane) ([]Record, error) // los del carril, ordenados por cambio
}

// Prefs son las preferencias personales de worktree del hub (REQ-25.8).
type Prefs interface {
	WorktreePrefs(projectMaster string) (mode Mode, root string, source string, warning string)
}
