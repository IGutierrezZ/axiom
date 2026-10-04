package skillregistry

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// projectRelativePath reports whether path lies strictly inside the directory
// cwd and, when it does, returns its path relative to cwd with OS separators.
//
// It is the single containment rule behind ScopeForPath, the repo-relative Path
// cell of the versioned table, the git ignore lookup and the project-first name
// dedupe. They must agree: if one said "project" while another computed the
// relative path with a different rule, an AGENTS.md row would be wrong (for
// example a `..\..\` prefixed path) instead of merely classified differently.
//
// The lexical check is exact and case-sensitive, and it is the whole rule on
// every OS but Windows. On Windows two spellings of one directory are common:
// a drive letter or directory in another case (`c:` against `C:`) and 8.3 short
// names (`IGUTIE~1` against `igutierrezz`, which is also how %TEMP% is spelled).
// There the lexical miss falls through to windowsProjectRelativePath. Nothing
// here returns an error: a miss that cannot be resolved is simply "not inside",
// which is what the lexical check alone would have said.
func projectRelativePath(cwd, path string) (string, bool) {
	cleanCwd := filepath.Clean(cwd)
	cleanPath := filepath.Clean(path)
	prefix := cleanCwd + string(os.PathSeparator)
	if strings.HasPrefix(cleanPath, prefix) {
		return cleanPath[len(prefix):], true
	}
	if runtime.GOOS != "windows" {
		return "", false
	}
	return windowsProjectRelativePath(cleanCwd, cleanPath)
}

// windowsProjectRelativePath resolves a lexical miss between two absolute,
// already cleaned Windows paths in two steps, cheapest first:
//
//  1. filepath.Rel, which on Windows compares case-insensitively (drive letter
//     included). It settles every case-only difference without touching disk.
//  2. relativeByIdentity, which settles 8.3 short names by asking the
//     filesystem.
//
// Relative inputs keep the lexical answer: with no volume there is nothing to
// normalize, and treating `.` as an alias of the process directory would change
// what the existing relative-cwd callers get.
func windowsProjectRelativePath(cwd, path string) (string, bool) {
	if !filepath.IsAbs(cwd) || !filepath.IsAbs(path) {
		return "", false
	}
	if rel, err := filepath.Rel(cwd, path); err == nil && isDescendantRel(rel) {
		return rel, true
	}
	return relativeByIdentity(cwd, path)
}

// isDescendantRel reports whether a filepath.Rel result names something below
// the base: not the base itself (`.`) and not outside it (`..`, `..\x`).
func isDescendantRel(rel string) bool {
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// relativeByIdentity decides containment by directory identity instead of by
// spelling. It takes the ancestor of path that sits as deep as cwd and asks
// os.SameFile (volume serial plus file index) whether that ancestor and cwd are
// the same directory; if so, what lies below the ancestor is the relative path.
//
// Two choices shape it, and both are deliberate.
//
// Identity is stat'ed, not resolved. filepath.EvalSymlinks would also expand
// short names, but it follows symlinks and junctions in path, so a skill
// directory that is a junction to a shared location (the dotfiles layout
// findAllSkillFiles deliberately follows) would resolve OUT of the project and
// be reclassified as "user". Here only an ancestor above the skill is stat'ed,
// the returned components are the ones the caller discovered, and a link below
// the project never changes the answer.
//
// Only the ancestor at cwd's depth is checked. An 8.3 alias (or a case change)
// renames a component, it never adds or removes one, so that single ancestor is
// the only candidate for the same directory. This also bounds the cost to one
// stat per path, which matters: every user-scope skill under $HOME takes this
// path on every refresh and a walk over all ancestors cost about 90 ms for a
// real 95-skill home. The trade-off is that a cwd reached through a link of a
// different depth is not recognised, which is what the lexical check said before.
//
// Both inputs must be cleaned and of the same kind (both absolute). A path whose
// cwd or ancestor cannot be stat'ed is simply not inside.
func relativeByIdentity(cwd, path string) (string, bool) {
	root, err := os.Stat(cwd)
	if err != nil || !root.IsDir() {
		return "", false
	}
	depth := len(pathElements(cwd))
	elements := pathElements(path)
	if len(elements) <= depth {
		return "", false // as deep as cwd or shallower: never strictly inside
	}
	ancestor := path
	for i := len(elements); i > depth; i-- {
		ancestor = filepath.Dir(ancestor)
	}
	if info, err := os.Stat(ancestor); err != nil || !os.SameFile(info, root) {
		return "", false
	}
	return filepath.Join(elements[depth:]...), true
}

// pathElements splits a cleaned path into the elements below its volume (the
// drive or UNC share on Windows, nothing elsewhere) and its root separator.
func pathElements(path string) []string {
	rest := strings.Trim(path[len(filepath.VolumeName(path)):], string(os.PathSeparator))
	if rest == "" {
		return nil
	}
	return strings.Split(rest, string(os.PathSeparator))
}
