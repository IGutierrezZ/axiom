package hub

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewManagerDefaultPathHonorsAxiomStateDir comprueba que el Hub vive bajo
// AXIOM_STATE_DIR cuando está definida, igual que el resto del estado de Axiom:
// así un proceso que ejecuta `axiom` puede aislar su Hub sin tocar HOME.
func TestNewManagerDefaultPathHonorsAxiomStateDir(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", stateDir)

	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	want := filepath.Join(stateDir, "workspaces.json")
	if got := mgr.GetConfigPath(); got != want {
		t.Fatalf("ruta del Hub = %q; se esperaba %q", got, want)
	}

	if _, err := mgr.Register(t.TempDir(), "Aislado", ""); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("el Hub no se escribió bajo AXIOM_STATE_DIR: %v", err)
	}
}

// TestNewManagerDefaultPathFallsBackToHome comprueba la ruta histórica
// (~/.axiom/workspaces.json) cuando AXIOM_STATE_DIR no está definida.
func TestNewManagerDefaultPathFallsBackToHome(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	want := filepath.Join(home, ".axiom", "workspaces.json")
	if got := mgr.GetConfigPath(); got != want {
		t.Fatalf("ruta del Hub = %q; se esperaba %q", got, want)
	}
}
