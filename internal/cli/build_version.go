package cli

import (
	"runtime/debug"
	"strings"
)

// buildVersionAndCommit resolves the build's own version/commit the
// same way internal/app.ResolveVersion does, without importing internal/app
// (this package must not depend on the top-level command dispatcher).
// AppVersion is the build's stamped version, handed to this package by
// internal/app at startup. It is the same string `axiom --version` prints.
//
// Reading build info alone was wrong and a tester caught it: for any stamped
// build the module pseudo-version disagrees with what --version reports, so
// doctor announced "version 1.49.1-0.2026..." for a binary that calls itself
// 2.2.0-rc.1. Doctor exists precisely to say which build is running, so two
// answers for one binary is a defect in both.
func buildVersionAndCommit() (string, string) {
	version, commit := "dev", "unknown"
	if stamped := strings.TrimSpace(AppVersion); stamped != "" && stamped != "dev" {
		version = stamped
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		// Build info still answers when nothing stamped the binary, and always
		// supplies the commit, which the stamped version does not carry.
		if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = strings.TrimPrefix(info.Main.Version, "v")
		}
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				commit = setting.Value
			}
		}
	}
	return version, commit
}
