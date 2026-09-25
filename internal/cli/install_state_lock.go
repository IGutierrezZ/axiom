package cli

import "github.com/IGutierrezZ/axiom/v3/internal/statecoord"

func withInstallStateLock(homeDir string, operation func() error) error {
	return statecoord.WithLock(homeDir, operation)
}
