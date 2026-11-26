package sddstatus

import (
	"path/filepath"

	"github.com/IGutierrezZ/axiom/v3/internal/acceptancetests"
	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// TasksGateArtifactPaths returns the single set of files the shared "tasks"
// gate digest covers. Both `axiom sdd gate record` (internal/cli) and the
// status governance read (governanceArtifactInputs) call it, so the two
// digests cannot drift.
//
// The set is tasks.md, or every role's own TasksFile when the sealed roster
// declares them (kickoff.RoleTaskFiles), plus pruebas.md only when it exists and
// is not blank. Without pruebas.md the list is exactly the previous one, so the
// digest of an existing change does not change. The per-role and
// role-apply:<role> digests never include pruebas.md.
func TasksGateArtifactPaths(changeRoot string, roles []kickoff.KickoffRole) []string {
	paths := roleTaskPaths(changeRoot, roles)
	if len(paths) == 0 {
		paths = []string{filepath.Join(changeRoot, "tasks.md")}
	}
	if acceptancePath := filepath.Join(changeRoot, acceptancetests.FileName); hasContent(acceptancePath) {
		paths = append(paths, acceptancePath)
	}
	return paths
}
