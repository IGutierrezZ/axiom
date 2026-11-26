package persona

import (
	"path/filepath"
)

// OutputStylePaths describes the output-style resources owned by the persona.
// Write is also the post-apply verification target; Backup lists every file the
// installation may overwrite.
type OutputStylePaths struct {
	Write  string
	Backup []string
}

// OutputStyle describes the managed output style.
type OutputStyle struct {
	Name      string
	File      string
	AssetPath string
}

// ResourcePlan is the ephemeral managed-resource definition of the persona.
// It derives write, backup and verification paths without storing state.
type ResourcePlan struct {
	outputStyle OutputStyle
}

var managedOutputStyles = []OutputStyle{
	{Name: "Axiom", File: "axiom.md", AssetPath: "claude/output-style-axiom.md"},
}

// ManagedOutputStyles lists every output style Axiom writes. Uninstall removes a
// copy only when its content is still the one embedded here.
func ManagedOutputStyles() []OutputStyle {
	return append([]OutputStyle(nil), managedOutputStyles...)
}

// ResourcePlanFor returns the managed resources of the Axiom persona.
func ResourcePlanFor() ResourcePlan {
	return ResourcePlan{outputStyle: managedOutputStyles[0]}
}

// OutputStyle returns the managed output-style definition.
func (p ResourcePlan) OutputStyle() OutputStyle {
	return p.outputStyle
}

// OutputStylePaths derives all output-style paths for a delivery directory.
func (p ResourcePlan) OutputStylePaths(dir string) OutputStylePaths {
	if dir == "" {
		return OutputStylePaths{}
	}

	paths := OutputStylePaths{
		Write:  filepath.Join(dir, p.outputStyle.File),
		Backup: make([]string, 0, len(managedOutputStyles)),
	}
	for _, style := range managedOutputStyles {
		paths.Backup = append(paths.Backup, filepath.Join(dir, style.File))
	}
	return paths
}
