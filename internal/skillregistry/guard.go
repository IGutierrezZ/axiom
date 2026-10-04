package skillregistry

import (
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/pathidentity"
)

// SkipReason classifies why `skill-registry refresh` must not initialize a
// registry at the resolved working directory. Startup hooks (OpenCode plugin,
// Codex/Claude SessionStart hooks) run the refresh from whatever directory
// the host resolved — which for a brand-new non-project directory can be "/",
// the user's home directory, or a markerless scratch folder. Initializing
// there either fails loudly (mkdir /.atl on a read-only root) or pollutes a
// directory that is not a project. The guard mirrors the CodeGraph principle:
// never initialize in the filesystem root, $HOME, or non-project folders.
type SkipReason string

const (
	// SkipNone means the directory is a project root and refresh proceeds.
	SkipNone SkipReason = ""
	// SkipFilesystemRoot: cwd resolved to the filesystem root. Always refused,
	// even if markers somehow exist there.
	SkipFilesystemRoot SkipReason = "filesystem-root"
	// SkipHomeDirectory: cwd is the user's home directory. Always refused,
	// even with an existing .atl, so a stray home registry never keeps
	// startup hooks writing into $HOME.
	SkipHomeDirectory SkipReason = "home-directory"
	// SkipNoProjectMarker: cwd carries no project marker (no .git, no
	// existing .atl, no project skills workspace). Refresh skips without
	// creating anything.
	SkipNoProjectMarker SkipReason = "no-project-marker"
)

// CleanPathArg exposes the --cwd normalization used by RefreshSkip and
// Regenerate. Callers that resolve a user-supplied --cwd against the process
// working directory (filepath.Abs) must normalize it first: on a non-Windows
// host a backslash-separated argument is not absolute, so resolving it before
// folding the separators joins it to the process cwd as a single relative
// filename and the refresh silently skips a real project.
func CleanPathArg(path string) string {
	return cleanPathArg(path)
}

// RefreshSkip reports whether a refresh at cwd must be skipped and why.
// It never writes to the filesystem.
func RefreshSkip(cwd, home string) SkipReason {
	cwd = cleanPathArg(cwd)
	if reason := protectedDirectory(cwd, home); reason != SkipNone {
		return reason
	}
	if !dirExists(cwd) {
		return SkipNoProjectMarker
	}
	if hasProjectMarker(cwd) {
		return SkipNone
	}
	return SkipNoProjectMarker
}

// protectedDirectory classifies cwd as one of the two directories that must
// never hold a registry, however a startup hook spelled it: the filesystem root
// (SkipFilesystemRoot) or the user's home directory (SkipHomeDirectory). Any
// other directory yields SkipNone. It is the one rule behind both RefreshSkip
// and the guard inside Regenerate, so the two can never disagree.
//
// cwd must already be cleaned. Both checks try the cheap lexical comparison
// first and only then ask the filesystem, because a path string cannot tell
// that two spellings name one directory.
func protectedDirectory(cwd, home string) SkipReason {
	if isFilesystemRoot(cwd) {
		return SkipFilesystemRoot
	}
	if isHomeDirectory(cwd, home) {
		return SkipHomeDirectory
	}
	return SkipNone
}

// isFilesystemRoot reports whether cwd is the root of its volume.
//
// The lexical test (a path whose parent is itself) is structural, so it already
// holds for any spelling of a root: `c:\` and `C:\`, and the `\\?\C:\` prefix.
// What it cannot see is a directory that merely resolves to the root, such as a
// symlink or junction pointing at it; that is the identity comparison against
// the root of the volume cwd names. A relative cwd has no volume of its own, so
// only the lexical test applies to it.
func isFilesystemRoot(cwd string) bool {
	if cwd == filepath.Dir(cwd) {
		// "/" on POSIX, a volume root on Windows.
		return true
	}
	if !filepath.IsAbs(cwd) {
		return false
	}
	return pathidentity.SameDirectory(cwd, filepath.VolumeName(cwd)+string(os.PathSeparator))
}

// isHomeDirectory reports whether cwd is the user's home directory.
//
// A Windows home comes from %USERPROFILE%, which rarely shares a spelling with
// the --cwd a hook passes: another drive-letter or directory case, an 8.3 short
// name (`IGUTIE~1`, which is also how %TEMP% is spelled) or a `\\?\` prefix. APFS
// is case-insensitive too, and any OS can reach home through a symlink. All of
// those are the same directory to the filesystem and different strings to a
// comparison, so identity is decided by pathidentity once the exact comparison
// misses. A relative home is not an identity at all (it would alias the process
// directory), and an empty one means there is no home to protect.
func isHomeDirectory(cwd, home string) bool {
	if home == "" {
		return false
	}
	home = filepath.Clean(home)
	if cwd == home {
		return true
	}
	return filepath.IsAbs(home) && pathidentity.SameDirectory(cwd, home)
}

// hasProjectMarker reports whether cwd looks like a project root: a Git
// checkout (.git directory, or the .git file used by worktrees and
// submodules), an already-initialized registry (.atl), or any workspace
// skill source directory this registry indexes (ProjectSkillDirs).
func hasProjectMarker(cwd string) bool {
	if _, err := os.Lstat(filepath.Join(cwd, ".git")); err == nil {
		return true
	}
	if dirExists(filepath.Join(cwd, ".atl")) {
		return true
	}
	if fileExists(filepath.Join(cwd, "axiom.yaml")) || fileExists(filepath.Join(cwd, ".axiom-workspace")) {
		return true
	}
	for _, dir := range ProjectSkillDirs(cwd) {
		if dirExists(dir) {
			return true
		}
	}
	return false
}
