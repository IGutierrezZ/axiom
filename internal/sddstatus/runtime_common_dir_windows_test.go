//go:build windows

package sddstatus

import (
	"syscall"
	"testing"
)

func windowsShortPath(t *testing.T, path string) string {
	t.Helper()
	long, err := syscall.UTF16FromString(path)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]uint16, syscall.MAX_PATH*4)
	n, err := syscall.GetShortPathName(&long[0], &buffer[0], uint32(len(buffer)))
	if err != nil || n == 0 {
		t.Skipf("8.3 short name unavailable: %v", err)
	}
	return syscall.UTF16ToString(buffer[:n])
}
