package reviewtransaction

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
)

// reviewWorkspacePointerFilename is the multirepo workspace marker. It spells
// the same name as workspace.WorkspacePointerFilename; this package does not
// import internal/workspace, and a test pins the two spellings together.
const reviewWorkspacePointerFilename = ".axiom-workspace"

// reviewMultirepoListedRepositories bounds how many child repositories a
// multirepo refusal names, so a master folder with dozens of them still yields
// one readable sentence.
const reviewMultirepoListedRepositories = 5

// ReviewMultirepoWorkspaceError reports that PrepareReviewRepositoryRoot
// refused to run `git init` because the directory is the master folder of a
// multirepo workspace: it carries the .axiom-workspace marker (itself or in an
// ancestor) or holds repositories of its own. Nothing was written. Callers
// branch on it with errors.As to classify the refusal as an operator-fixable
// request instead of an unexpected fault.
type ReviewMultirepoWorkspaceError struct {
	// Workspace is the canonical directory the caller asked to prepare.
	Workspace string
	// Pointer is the .axiom-workspace entry that marked the workspace, or empty
	// when only child repositories identified it.
	Pointer string
	// Repositories lists every direct child directory that holds Git metadata.
	Repositories []string
	err          error
}

func (err *ReviewMultirepoWorkspaceError) Error() string { return err.err.Error() }
func (err *ReviewMultirepoWorkspaceError) Unwrap() error { return err.err }

// PrepareReviewRepositoryRoot is the review lifecycle entry boundary for a
// workspace that may not yet have local Git metadata. It reuses an existing
// containing worktree and initializes Git only after both hardened discovery
// and an isolated probe prove the workspace is genuinely unversioned.
//
// A multirepo master folder is never initialized: it is not a repository, and
// a repository created around its children would swallow them. Such a
// directory is refused with a *ReviewMultirepoWorkspaceError that names the
// child repositories review should be pointed at.
//
// SnapshotBuilder and every lower-level repository boundary remain read-only:
// callers that need lifecycle preparation must opt in at this boundary.
func PrepareReviewRepositoryRoot(ctx context.Context, workspace string) (string, error) {
	builder := SnapshotBuilder{Repo: workspace}
	root, resolveErr := builder.ResolveRepositoryRoot(ctx)
	if resolveErr == nil {
		return root, nil
	}
	if !reviewRootResolutionReportsNoRepository(resolveErr) {
		return "", resolveErr
	}

	workspace, pathErr := canonicalRepositoryPath(workspace)
	if pathErr != nil {
		return "", pathErr
	}
	info, statErr := os.Stat(workspace)
	if statErr != nil || !info.IsDir() {
		return "", resolveErr
	}
	metadataPresent, metadataErr := reviewWorkspaceHasGitMetadata(workspace)
	if metadataErr != nil || metadataPresent {
		return "", resolveErr
	}
	multirepo, multirepoErr := reviewDetectMultirepoWorkspace(workspace)
	if multirepoErr != nil {
		return "", resolveErr
	}
	if multirepo != nil {
		return "", multirepo
	}

	if _, probeErr := runGitIsolated(ctx, workspace, nil, nil, "rev-parse", "--is-inside-work-tree"); !reviewIsolatedProbeReportsNoRepository(probeErr) {
		return "", resolveErr
	}
	if _, initErr := runGitIsolated(ctx, workspace, nil, nil, "init", "--quiet"); initErr != nil {
		return "", initErr
	}
	return (SnapshotBuilder{Repo: workspace}).ResolveRepositoryRoot(ctx)
}

// ReviewRootResolutionReportsNoRepository reports the bounded Git discovery
// failure for a cwd outside any repository. It lets higher-level command
// surfaces decide whether repository identity was truly required instead of
// matching localized Git stderr text.
func ReviewRootResolutionReportsNoRepository(err error) bool {
	return reviewRootResolutionReportsNoRepository(err)
}

func reviewRootResolutionReportsNoRepository(err error) bool {
	return reviewGitCommandExited128(err, "rev-parse", "--show-toplevel")
}

func reviewIsolatedProbeReportsNoRepository(err error) bool {
	return reviewGitCommandExited128(err, "rev-parse", "--is-inside-work-tree")
}

func reviewGitCommandExited128(err error, args ...string) bool {
	var commandErr *GitCommandError
	if !errors.As(err, &commandErr) || commandErr.ExitCode != 128 || len(commandErr.Args) != len(args) {
		return false
	}
	for index, arg := range args {
		if commandErr.Args[index] != arg {
			return false
		}
	}
	return true
}

// reviewWorkspaceHasGitMetadata refuses initialization whenever a .git entry
// already exists in the requested directory or one of its parents. A valid
// containing worktree returned before this scan; reaching it here means the
// existing metadata is unusable and must be left untouched.
func reviewWorkspaceHasGitMetadata(workspace string) (bool, error) {
	for directory := workspace; ; directory = filepath.Dir(directory) {
		_, err := os.Lstat(filepath.Join(directory, ".git"))
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return false, err
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return false, nil
		}
	}
}

// reviewDetectMultirepoWorkspace reports whether workspace is the master folder
// of a multirepo workspace, which review must never turn into a repository. It
// is deliberately read-only. A nil result with a nil error means the directory
// is an ordinary unversioned one; an unreadable directory is returned as an
// error so the caller refuses to initialize what it could not inspect.
func reviewDetectMultirepoWorkspace(workspace string) (*ReviewMultirepoWorkspaceError, error) {
	pointer, err := reviewWorkspacePointerPath(workspace)
	if err != nil {
		return nil, err
	}
	repositories, err := reviewChildRepositories(workspace)
	if err != nil {
		return nil, err
	}
	if pointer == "" && len(repositories) == 0 {
		return nil, nil
	}
	return &ReviewMultirepoWorkspaceError{
		Workspace:    workspace,
		Pointer:      pointer,
		Repositories: repositories,
		err: fmt.Errorf("review does not initialize Git in %s: it is a multirepo workspace, not a repository (%s); rerun `axiom review status --cwd <repo>` with --cwd naming one of its repositories",
			pathquote.Quote(workspace), reviewMultirepoEvidence(pointer, repositories)),
	}, nil
}

// reviewWorkspacePointerPath returns the first .axiom-workspace entry found in
// workspace or one of its ancestors, or empty when none exists.
func reviewWorkspacePointerPath(workspace string) (string, error) {
	for directory := workspace; ; directory = filepath.Dir(directory) {
		candidate := filepath.Join(directory, reviewWorkspacePointerFilename)
		_, err := os.Lstat(candidate)
		if err == nil {
			return candidate, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		if parent := filepath.Dir(directory); parent == directory {
			return "", nil
		}
	}
}

// reviewChildRepositories returns the direct child directories of workspace
// that hold a .git entry, as a directory or a gitfile. It never descends
// further: a multirepo master lists its repositories one level down.
func reviewChildRepositories(workspace string) ([]string, error) {
	entries, err := os.ReadDir(workspace)
	if err != nil {
		return nil, err
	}
	var repositories []string
	for _, entry := range entries {
		if !reviewChildMayHoldRepository(workspace, entry) {
			continue
		}
		child := filepath.Join(workspace, entry.Name())
		_, err := os.Lstat(filepath.Join(child, ".git"))
		switch {
		case err == nil:
			repositories = append(repositories, child)
		case errors.Is(err, fs.ErrNotExist):
		default:
			return nil, err
		}
	}
	return repositories, nil
}

// reviewChildMayHoldRepository filters out entries that cannot be a directory,
// so a stray file or a link to a file never turns an ordinary unversioned
// folder into a refusal. A symlink or Windows junction counts when it resolves
// to a directory.
func reviewChildMayHoldRepository(workspace string, entry fs.DirEntry) bool {
	if entry.IsDir() {
		return true
	}
	if entry.Type()&(fs.ModeSymlink|fs.ModeIrregular) == 0 {
		return false
	}
	info, err := os.Stat(filepath.Join(workspace, entry.Name()))
	return err == nil && info.IsDir()
}

// reviewMultirepoEvidence renders why a directory was recognized as a
// multirepo master: the marker found, the child repositories (bounded), or both.
func reviewMultirepoEvidence(pointer string, repositories []string) string {
	var evidence []string
	if pointer != "" {
		evidence = append(evidence, "workspace marker "+pathquote.Quote(pointer))
	}
	if len(repositories) > 0 {
		listed := repositories
		if len(listed) > reviewMultirepoListedRepositories {
			listed = listed[:reviewMultirepoListedRepositories]
		}
		quoted := make([]string, 0, len(listed))
		for _, repository := range listed {
			quoted = append(quoted, pathquote.Quote(repository))
		}
		text := "repositories " + strings.Join(quoted, ", ")
		if extra := len(repositories) - len(listed); extra > 0 {
			text += fmt.Sprintf(" and %d more", extra)
		}
		evidence = append(evidence, text)
	}
	return strings.Join(evidence, "; ")
}
