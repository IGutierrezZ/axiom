package kickoff

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// archivedChangesPrefix is the workspace-relative, slash-normalized path
// segment that marks a change root as already archived (D-14). It is
// compared only against the slash-normalized result of filepath.Rel, never
// against a raw platform-separator string, so the check behaves the same
// on Windows and POSIX.
const archivedChangesPrefix = "openspec/changes/archive"

// archiveDatePrefixPattern matches the 11-character YYYY-MM-DD- prefix the
// archive phase stamps on every openspec/changes/archive/ entry. It repeats
// the rule of sddstatus.newestArchivedOpenSpecEntry on purpose: that file is
// shared with upstream and is not modified by inc-24.
var archiveDatePrefixPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-$`)

// ArchivedChangeRoot returns the archive folder of change under workspaceRoot.
// It looks first for openspec/changes/archive/<change> (the legacy form the
// existing tests use) and, when that does not exist, for the newest entry named
// YYYY-MM-DD-<change>, which is what sdd-archive produces. Only an exact
// <change> suffix after a full date prefix matches (change "foo-bar" never
// matches "2026-01-01-foo-bar-baz"); date prefixes sort lexicographically, so
// the greatest match is the newest. A change name that is not a bare name (it
// is empty or carries a separator) never resolves.
//
// Resolving the archive folder lets RefuseArchivedRoot reject an archived
// change as archived instead of the verbs answering that it does not exist
// (REQ-24.9).
func ArchivedChangeRoot(workspaceRoot, change string) (string, bool) {
	if change == "" || change == "." || change == ".." || strings.ContainsAny(change, `/\`) {
		return "", false
	}
	archiveDir := filepath.Join(workspaceRoot, "openspec", "changes", "archive")
	exact := filepath.Join(archiveDir, change)
	if info, err := os.Stat(exact); err == nil && info.IsDir() {
		return exact, true
	}
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		return "", false
	}
	newest := ""
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || len(name) != 11+len(change) ||
			!archiveDatePrefixPattern.MatchString(name[:11]) || name[11:] != change {
			continue
		}
		if name > newest {
			newest = name
		}
	}
	if newest == "" {
		return "", false
	}
	return filepath.Join(archiveDir, newest), true
}

// RefuseArchivedRoot rejects a write whose changeRoot has escaped
// workspaceRoot, or whose changeRoot already lives under
// openspec/changes/archive/. Post-archive immutability of ownership is
// declared, not filesystem-enforced (D-14): Axiom cannot stop an editor
// from touching files under archive/ directly, but every write path this
// package owns calls this guard first, so at least no Axiom-owned verb
// reopens an archived change on the caller's behalf.
func RefuseArchivedRoot(workspaceRoot, changeRoot string) error {
	rel, err := filepath.Rel(workspaceRoot, changeRoot)
	if err != nil {
		return fmt.Errorf("resolver la raiz del cambio %q respecto al workspace %q: %w", changeRoot, workspaceRoot, err)
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
		return fmt.Errorf("la raiz del cambio %q queda fuera del workspace %q", changeRoot, workspaceRoot)
	}
	if rel == archivedChangesPrefix || strings.HasPrefix(rel, archivedChangesPrefix+"/") {
		return fmt.Errorf("el cambio %q ya esta archivado; gestiona la correccion mediante un ticket de bug o un nuevo incremento en vez de reabrirlo (REQ-21.18)", rel)
	}
	return nil
}
