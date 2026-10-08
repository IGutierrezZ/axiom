package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
	"github.com/IGutierrezZ/axiom/v3/internal/pathquote"
	"github.com/IGutierrezZ/axiom/v3/internal/workspace"
)

// resolveGovernanceChangeRoot resolves the workspace from cwd with
// workspace.Locate (falling back to the process working directory when cwd is
// empty) and returns the resolved Location together with the absolute path to
// an EXISTING change directory named change. Every governance verb therefore
// reads and writes the same planning home whatever directory it starts from
// (REQ-24.8): the master folder, the specs repository, or a code repository.
//
// It checks first openspec/changes/<change> under loc.WorkspaceRoot and then
// the archived change, which kickoff.ArchivedChangeRoot finds either under its
// bare name or as the newest YYYY-MM-DD-<change> entry sdd-archive produces.
// Checking the archived location too — rather than only the active one — is
// what lets a bare, separator-free --change value (already required by
// kickoff.validateChangeName's containment rule) still resolve to an archived
// change, so RefuseArchivedRoot below can reject it by D-14 instead of this
// function reporting a misleading "does not exist" (REQ-24.9, scenario 3).
//
// It never creates a directory: `axiom sdd kickoff`/`axiom sdd gate` write
// inside a change that already exists (task 8.2). It also refuses a change
// root already living under openspec/changes/archive/ (D-14) here, once, so
// RunSDDKickoff/RunSDDGate never need two independent copies of that check
// (task 10.4).
func resolveGovernanceChangeRoot(cwd, change string) (workspace.Location, string, error) {
	loc, err := workspace.Locate(cwd)
	if err != nil {
		return workspace.Location{}, "", err
	}

	changeRoot := filepath.Join(loc.WorkspaceRoot, "openspec", "changes", change)
	if !isDirectory(changeRoot) {
		archived, ok := kickoff.ArchivedChangeRoot(loc.WorkspaceRoot, change)
		if !ok {
			return workspace.Location{}, "", fmt.Errorf("el cambio %q no existe en %q ni en %q; estos verbos escriben dentro de un cambio ya creado, nunca crean uno nuevo; ejecuta `axiom sdd status --cwd %s` para ver los cambios activos",
				change, changeRoot, filepath.Join(loc.WorkspaceRoot, "openspec", "changes", "archive", change), pathquote.Quote(loc.WorkspaceRoot))
		}
		changeRoot = archived
	}

	if err := kickoff.RefuseArchivedRoot(loc.WorkspaceRoot, changeRoot); err != nil {
		return workspace.Location{}, "", err
	}
	return loc, changeRoot, nil
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// hasGovernanceHelpFlag reports whether args requests help, mirroring
// hasSDDArchiveComposeHelp's exact --help/-h check (sdd_archive_compose.go)
// for the two new kickoff/gate verbs.
func hasGovernanceHelpFlag(args []string) bool {
	for _, argument := range args {
		if argument == "--help" || argument == "-h" {
			return true
		}
	}
	return false
}
