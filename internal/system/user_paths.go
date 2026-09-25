package system

import (
	"path/filepath"
)

// AxiomDir returns the canonical user directory for Axiom.
// If AXIOM_STATE_DIR is set in the environment,
// it takes precedence. Otherwise, it defaults to filepath.Join(homeDir, ".axiom").
func AxiomDir(homeDir string) string {
	if custom := Getenv("AXIOM_STATE_DIR"); custom != "" {
		return custom
	}
	return filepath.Join(homeDir, ".axiom")
}

// StatePath returns the canonical path to state.json under AxiomDir(homeDir).
func StatePath(homeDir string) string {
	return filepath.Join(AxiomDir(homeDir), "state.json")
}

// CacheDir returns the canonical path to cache directory under AxiomDir(homeDir).
func CacheDir(homeDir string) string {
	return filepath.Join(AxiomDir(homeDir), "cache")
}

// BinDir returns the canonical path to bin directory under AxiomDir(homeDir).
func BinDir(homeDir string) string {
	return filepath.Join(AxiomDir(homeDir), "bin")
}
