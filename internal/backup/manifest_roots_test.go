package backup

import (
	"path/filepath"
	"testing"
)

// TestBackupRootFor verifies that BackupRootFor resolves the canonical backup
// root (~/.axiom/backups) for a variety of home directory shapes, including a
// Windows-style path with spaces and an empty string.
func TestBackupRootFor(t *testing.T) {
	tests := []struct {
		name string
		home string
	}{
		{name: "unix-style home", home: "/home/user"},
		{name: "windows-style home with spaces", home: `C:\Users\Jane Doe`},
		{name: "empty home", home: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := filepath.Join(tt.home, ".axiom", "backups")
			got := BackupRootFor(tt.home)
			if got != want {
				t.Errorf("BackupRootFor(%q) = %q, want %q", tt.home, got, want)
			}
		})
	}
}

// TestBackupRootForFollowsAxiomStateDir verifica que AXIOM_STATE_DIR mueve los
// respaldos junto con el resto del estado de máquina y que, sin la variable, la ruta por defecto no cambia.
func TestBackupRootForFollowsAxiomStateDir(t *testing.T) {
	home := t.TempDir()
	stateDir := filepath.Join(t.TempDir(), "estado")

	t.Setenv("AXIOM_STATE_DIR", "")
	if got, want := BackupRootFor(home), filepath.Join(home, ".axiom", "backups"); got != want {
		t.Errorf("sin AXIOM_STATE_DIR, BackupRootFor() = %q, want %q", got, want)
	}

	t.Setenv("AXIOM_STATE_DIR", stateDir)
	if got, want := BackupRootFor(home), filepath.Join(stateDir, "backups"); got != want {
		t.Errorf("con AXIOM_STATE_DIR, BackupRootFor() = %q, want %q", got, want)
	}
	roots := BackupRoots(home)
	if len(roots) != 1 || roots[0] != BackupRootFor(home) {
		t.Errorf("BackupRoots() = %v, want solo la canónica", roots)
	}
	if got := BackupOriginForRoot(home, BackupRootFor(home)); got != BackupOriginAxiom {
		t.Errorf("origen de la raíz canónica = %v, want %v", got, BackupOriginAxiom)
	}
}

// TestBackupRoots verifies that BackupRoots returns exactly the canonical
// root: the legacy root was retired and is migrated by internal/legacyhome.
func TestBackupRoots(t *testing.T) {
	tests := []struct {
		name string
		home string
	}{
		{name: "unix-style home", home: "/home/user"},
		{name: "windows-style home with spaces", home: `C:\Users\Jane Doe`},
		{name: "empty home", home: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := []string{BackupRootFor(tt.home)}
			got := BackupRoots(tt.home)
			if len(got) != len(want) {
				t.Fatalf("BackupRoots(%q) = %v, want %v", tt.home, got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("BackupRoots(%q)[%d] = %q, want %q", tt.home, i, got[i], want[i])
				}
			}
		})
	}
}
