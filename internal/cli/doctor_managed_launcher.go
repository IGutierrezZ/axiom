package cli

import (
	"fmt"
	"strings"

	opencodeactivation "github.com/IGutierrezZ/axiom/v3/internal/opencode"
)

// doctorManagedLauncher is an Axiom-managed OpenCode launcher found among the
// PATH copies of a tool, together with the real executable it wraps.
type doctorManagedLauncher struct {
	path   string
	target string
}

// doctorExcludeManagedLaunchers drops from copies every Axiom-managed OpenCode
// launcher that wraps another, non-launcher copy. The launcher is written on
// purpose when the background-subagents policy is on and delegates to the real
// OpenCode, so counting it as a second install is a false duplicate and the
// "remove duplicates" remedy would break the feature.
//
// A launcher is excluded only when the executable it wraps is itself one of the
// other copies. One wrapping a path that is not on PATH is kept, and so is any
// further copy, so genuine duplicates still surface.
func doctorExcludeManagedLaunchers(tool string, copies []string) ([]string, []doctorManagedLauncher) {
	if tool != "opencode" || len(copies) < 2 {
		return copies, nil
	}
	targets := make([]string, len(copies))
	for i, candidate := range copies {
		targets[i], _ = opencodeactivation.ManagedLauncherTarget(candidate)
	}
	kept := make([]string, 0, len(copies))
	var launchers []doctorManagedLauncher
	for i, candidate := range copies {
		if targets[i] != "" && doctorWrapsPlainCopy(targets, copies, i) {
			launchers = append(launchers, doctorManagedLauncher{path: candidate, target: targets[i]})
			continue
		}
		kept = append(kept, candidate)
	}
	return kept, launchers
}

// doctorWrapsPlainCopy reports whether the launcher at copies[index] wraps one
// of the other copies that is not itself a launcher, so launchers can never
// cancel each other out and leave nothing behind.
func doctorWrapsPlainCopy(targets, copies []string, index int) bool {
	for j, other := range copies {
		if j != index && targets[j] == "" && doctorSameExecutable(targets[index], other) {
			return true
		}
	}
	return false
}

// doctorManagedLauncherNote describes the launchers that were not counted, or
// returns "" when there are none.
func doctorManagedLauncherNote(launchers []doctorManagedLauncher) string {
	notes := make([]string, 0, len(launchers))
	for _, launcher := range launchers {
		notes = append(notes, fmt.Sprintf("Axiom-managed launcher %s wraps %s", launcher.path, launcher.target))
	}
	return strings.Join(notes, "; ")
}
