package system

import (
	"os"
)

// Well-known environment variable names with Axiom canonical keys.
const (
	EnvChannelAxiom                     = "AXIOM_CHANNEL"
	EnvOpenCodeBackgroundSubagentsAxiom = "AXIOM_OPENCODE_BACKGROUND_SUBAGENTS"
	EnvStateDirAxiom                    = "AXIOM_STATE_DIR"
)

// LookupEnv returns the value of the first present environment variable from the provided keys.
// It checks strictly the requested keys without falling back to legacy counterparts.
func LookupEnv(keys ...string) (string, bool) {
	for _, key := range keys {
		if key != "" {
			if val, ok := os.LookupEnv(key); ok {
				return val, true
			}
		}
	}
	return "", false
}

// Getenv returns the first non-empty value of the environment variables from the provided keys.
// It checks strictly the requested keys without falling back to legacy counterparts.
// If none is set, it returns the empty string.
func Getenv(keys ...string) string {
	for _, key := range keys {
		if key != "" {
			if val := os.Getenv(key); val != "" {
				return val
			}
		}
	}
	return ""
}
