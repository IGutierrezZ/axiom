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
// respaldos junto con el resto del estado de máquina, que la raíz heredada sigue
// siendo solo de lectura y que, sin la variable, la ruta por defecto no cambia.
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
	if got, want := LegacyBackupRootFor(home), filepath.Join(home, ".gentle-ai", "backups"); got != want {
		t.Errorf("la raíz heredada no debe moverse: LegacyBackupRootFor() = %q, want %q", got, want)
	}
	roots := BackupRoots(home)
	if len(roots) != 2 || roots[0] != BackupRootFor(home) || roots[1] != LegacyBackupRootFor(home) {
		t.Errorf("BackupRoots() = %v, want la canónica primero y la heredada después", roots)
	}
	if got := BackupOriginForRoot(home, BackupRootFor(home)); got != BackupOriginAxiom {
		t.Errorf("origen de la raíz canónica = %v, want %v", got, BackupOriginAxiom)
	}
}

// TestLegacyBackupRootFor verifies that LegacyBackupRootFor resolves the
// legacy backup root (~/.gentle-ai/backups) for the same variety of home
// directory shapes as TestBackupRootFor.
func TestLegacyBackupRootFor(t *testing.T) {
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
			want := filepath.Join(tt.home, ".gentle-ai", "backups")
			got := LegacyBackupRootFor(tt.home)
			if got != want {
				t.Errorf("LegacyBackupRootFor(%q) = %q, want %q", tt.home, got, want)
			}
		})
	}
}

// TestBackupRoots verifies that BackupRoots returns exactly the canonical
// root followed by the legacy root, in that order. Order is part of the
// contract, not an implementation detail: every reader that scans both roots
// must see the canonical one first.
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
			want := []string{BackupRootFor(tt.home), LegacyBackupRootFor(tt.home)}
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
