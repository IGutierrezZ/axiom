package sddstatus

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

// A change sealed with more than one role (or with a role whose plan is not
// the plain tasks.md) keeps one plan per role: tasks.<role>.md. Every consumer
// of the "tasks" artifact in this package resolves it through
// resolveRoleTaskFiles, so status, edit authority and the gate digests agree
// on the same file set. An unsealed change and a sealed single "fullstack"
// role resolve to nil here and keep using the shared tasks.md untouched.

// resolveRoleTaskFiles returns the absolute path of every role task file the
// sealed kickoff of changeRoot names, in kickoff role order, or nil when the
// change has no sealed kickoff or keeps its plan in the shared tasks.md. A
// corrupt kickoff.yaml is a named error, exactly as loadGovernance reports it.
func resolveRoleTaskFiles(changeRoot string) ([]string, error) {
	sealed, err := kickoff.Load(changeRoot)
	if err != nil || sealed == nil {
		return nil, err
	}
	return roleTaskPaths(changeRoot, sealed.Config.Roles), nil
}

// roleTaskPaths joins changeRoot with the role task file names of roles, or
// returns nil when the plan is the shared tasks.md (kickoff.RoleTaskFiles).
func roleTaskPaths(changeRoot string, roles []kickoff.KickoffRole) []string {
	names := kickoff.RoleTaskFiles(roles)
	if len(names) == 0 {
		return nil
	}
	paths := make([]string, 0, len(names))
	for _, name := range names {
		paths = append(paths, filepath.Join(changeRoot, name))
	}
	return paths
}

// roleTasks is the resolved "tasks" artifact of a multi-file change: its
// state, the aggregated checkbox progress, and the file names the blocked
// reasons must name instead of the shared "tasks.md".
type roleTasks struct {
	state ArtifactState
	// progress sums every role file that exists. AllComplete holds only when
	// the sum has no pending task AND every role file carries at least one
	// checkbox: a role whose plan has no checkboxes has nothing observable to
	// call finished, so it must never be absorbed into another role's "done".
	progress TaskProgress
	// incomplete lists the role files that are missing or empty.
	incomplete []string
	// noCheckboxes lists the role files with content but no markdown checkbox.
	noCheckboxes []string
}

// resolveRoleTasks evaluates the expected role task files (absolute paths in
// kickoff order): done when every one has content, partial when some exist
// without all being usable, missing when none exists.
func resolveRoleTasks(files []string) (roleTasks, error) {
	var resolved roleTasks
	existing := 0
	allHaveCheckboxes := true
	for _, path := range files {
		name := filepath.Base(path)
		if _, err := os.Stat(path); err == nil {
			existing++
		}
		if !hasContent(path) {
			resolved.incomplete = append(resolved.incomplete, name)
			continue
		}
		progress, err := countTaskProgress(path)
		if err != nil {
			return roleTasks{}, err
		}
		resolved.progress.Total += progress.Total
		resolved.progress.Completed += progress.Completed
		resolved.progress.Pending += progress.Pending
		if progress.Total == 0 {
			allHaveCheckboxes = false
			resolved.noCheckboxes = append(resolved.noCheckboxes, name)
		}
	}
	resolved.progress.AllComplete = resolved.progress.Total > 0 && resolved.progress.Pending == 0 && allHaveCheckboxes && len(resolved.incomplete) == 0
	switch {
	case existing == 0:
		resolved.state = ArtifactMissing
	case len(resolved.incomplete) > 0:
		resolved.state = ArtifactPartial
	default:
		resolved.state = ArtifactDone
	}
	return resolved, nil
}

// fileList renders file names for a blocked reason: "a.md" or "a.md, b.md".
func fileList(names []string) string {
	return strings.Join(names, ", ")
}

// isAre picks the verb that agrees with the number of files a reason names.
func isAre(names []string) string {
	if len(names) > 1 {
		return "are"
	}
	return "is"
}

// planText is one task plan read for edit-authority detection: the file name a
// blocked reason must cite and its text.
type planText struct {
	name string
	text string
}

// roleTaskPlans reads the role task files (absolute paths) as plans, in order.
func roleTaskPlans(files []string) []planText {
	plans := make([]planText, 0, len(files))
	for _, path := range files {
		plans = append(plans, planText{name: filepath.Base(path), text: readText(path)})
	}
	return plans
}

// uniqueSorted returns values de-duplicated in ascending order, the order
// detectUnauthorizedEditRoots already reports a single plan in.
func uniqueSorted(values []string) []string {
	seen := make(map[string]bool, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			unique = append(unique, value)
		}
	}
	sort.Strings(unique)
	return unique
}

// missingTasksReason is the planning blocker for an incomplete role task set.
func missingTasksReason(names []string) string {
	return fmt.Sprintf("%s %s missing or partial.", fileList(names), isAre(names))
}
