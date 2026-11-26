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
// (RF-PLH-09).
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
		return fmt.Errorf("el cambio %q ya esta archivado; gestiona la correccion mediante un ticket de bug o un nuevo incremento en vez de reabrirlo (RF-ARC-03)", rel)
	}
	return nil
}

// IntegrationsOutboxFile es el único fichero que Axiom puede escribir en una carpeta archivada
// (MODIFIED RF-ARC-03): la cola local de operaciones de integración. internal/integrations usa esta
// constante; no hay un segundo literal.
const IntegrationsOutboxFile = ".integrations-outbox.yaml"

const (
	integrationsOutboxTempPrefix = ".integrations-outbox-"
	integrationsOutboxTempSuffix = ".yaml"
)

// archivedFolderPattern reconoce la raíz exacta de una carpeta archivada respecto a workspaceRoot:
// openspec/changes/archive/AAAA-MM-DD-<cambio>, sin anidar.
var archivedFolderPattern = regexp.MustCompile(`^` + regexp.QuoteMeta(archivedChangesPrefix) + `/\d{4}-\d{2}-\d{2}-[^/]+$`)

// RefuseArchivedWriteExceptLocalState rechaza escribir name en changeRoot salvo cuando changeRoot
// NO está archivada (delega en RefuseArchivedRoot) o cuando es exactamente la raíz de una carpeta
// archivada (<workspace>/openspec/changes/archive/<AAAA-MM-DD>-<cambio>, sin anidar) y name es
// IntegrationsOutboxFile o un temporal ".integrations-outbox-*.yaml" de su escritura atómica.
// Cualquier otro nombre (integrations.yaml, gates.yaml, otro fichero con punto) o ruta con
// separador se rechaza; en una carpeta archivada con el mismo mensaje que RefuseArchivedRoot
// (RF-ARC-03). RefuseArchivedRoot no cambia y sigue siendo la guarda de todos los demás escritores.
func RefuseArchivedWriteExceptLocalState(workspaceRoot, changeRoot, name string) error {
	rel, err := filepath.Rel(workspaceRoot, changeRoot)
	if err != nil {
		return fmt.Errorf("resolver la raiz del cambio %q respecto al workspace %q: %w", changeRoot, workspaceRoot, err)
	}
	rel = filepath.ToSlash(rel)
	archived := rel == archivedChangesPrefix || strings.HasPrefix(rel, archivedChangesPrefix+"/")
	validName := name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`)

	if !archived {
		if guard := RefuseArchivedRoot(workspaceRoot, changeRoot); guard != nil {
			return guard // fuera del workspace
		}
		if !validName {
			return fmt.Errorf("el nombre de fichero %q no es válido: debe ser un nombre simple, sin separadores", name)
		}
		return nil
	}
	if validName && archivedFolderPattern.MatchString(rel) && isIntegrationsOutboxName(name) {
		return nil
	}
	return RefuseArchivedRoot(workspaceRoot, changeRoot)
}

// isIntegrationsOutboxName dice si name es la cola de integraciones o un temporal de su escritura
// atómica (.integrations-outbox-<relleno>.yaml con relleno no vacío y sin separadores).
func isIntegrationsOutboxName(name string) bool {
	if name == IntegrationsOutboxFile {
		return true
	}
	if !strings.HasPrefix(name, integrationsOutboxTempPrefix) || !strings.HasSuffix(name, integrationsOutboxTempSuffix) {
		return false
	}
	fill := strings.TrimSuffix(strings.TrimPrefix(name, integrationsOutboxTempPrefix), integrationsOutboxTempSuffix)
	return fill != "" && !strings.ContainsAny(fill, `/\`)
}
