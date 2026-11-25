//go:build !windows

package sddstatus

import "testing"

func windowsShortPath(t *testing.T, path string) string {
	t.Helper()
	return path
}
