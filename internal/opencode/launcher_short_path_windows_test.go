//go:build windows

package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlatformShortPathNamePointsAtTheSameFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "café")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "opencode.cmd")
	if err := os.WriteFile(target, []byte("@echo off\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	short, err := platformShortPathName(target)
	if err != nil || !isASCII(short) {
		t.Skipf("volume without 8.3 short names for %q: %q, %v", target, short, err)
	}
	longInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	shortInfo, err := os.Stat(short)
	if err != nil {
		t.Fatalf("Stat(%q): %v", short, err)
	}
	if !os.SameFile(longInfo, shortInfo) {
		t.Fatalf("short path %q does not name %q", short, target)
	}
}
