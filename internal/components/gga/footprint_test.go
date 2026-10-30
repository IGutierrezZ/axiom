package gga

import (
	"path/filepath"
	"testing"
)

func TestFootprintPaths(t *testing.T) {
	home := filepath.Join("home", "u")
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))

	linux := FootprintPaths(home, "linux")
	wantLinux := []string{
		filepath.Join(home, ".config", "gga"),
		filepath.Join(home, ".local", "share", "gga"),
	}
	if len(linux) != len(wantLinux) {
		t.Fatalf("linux = %v, want %v", linux, wantLinux)
	}
	for i := range wantLinux {
		if linux[i] != wantLinux[i] {
			t.Errorf("linux[%d] = %q, want %q", i, linux[i], wantLinux[i])
		}
	}

	windows := FootprintPaths(home, "windows")
	for _, want := range []string{
		filepath.Join(home, "AppData", "Roaming", "gga"),
		filepath.Join(home, ".local", "share", "gga"),
		filepath.Join(home, "bin", "gga.cmd"),
		filepath.Join(home, "bin", "gga.ps1"),
	} {
		found := false
		for _, path := range windows {
			found = found || path == want
		}
		if !found {
			t.Errorf("windows no lista %q: %v", want, windows)
		}
	}
}

func TestFootprintPathEntriesOnlyOnWindows(t *testing.T) {
	home := filepath.Join("home", "u")
	if got := FootprintPathEntries(home, "linux"); got != nil {
		t.Errorf("linux no persiste cambios de PATH, got %v", got)
	}
	got := FootprintPathEntries(home, "windows")
	if len(got) != 1 || got[0] != filepath.Join(home, "bin") {
		t.Errorf("windows debe añadir ~/bin, got %v", got)
	}
}
