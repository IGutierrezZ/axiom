package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/backup"
)

// countBackups cuenta las copias de respaldo que hay en la raíz de respaldos de home.
func countBackups(t *testing.T, home string) int {
	t.Helper()
	entries, err := os.ReadDir(backup.BackupRootFor(home))
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatal(err)
	}
	return len(entries)
}

// Con AXIOM_STATE_DIR definida, install y sync en workspace dejan todo el estado de
// máquina (respaldos incluidos) en esa raíz y nada bajo ~/.axiom ni ~/.gentle-ai.
func TestInstallAndSyncKeepMachineStateUnderAxiomStateDir(t *testing.T) {
	root, _, home := workspaceRootFixture(t, false, true)
	stateDir := filepath.Join(t.TempDir(), "estado")
	t.Setenv("AXIOM_STATE_DIR", stateDir)
	t.Chdir(root)

	installInProject(t, root, "--component", "engram", "--persona", "neutral")
	if _, err := RunSync([]string{"--scope", "workspace", "--agents", "claude-code"}); err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}

	for _, legacy := range []string{".axiom", ".gentle-ai"} {
		if _, err := os.Stat(filepath.Join(home, legacy)); !os.IsNotExist(err) {
			t.Errorf("con AXIOM_STATE_DIR no debe crearse ~/%s; stat err=%v; contenido de HOME: %v", legacy, err, listFiles(t, home))
		}
	}
	if got, want := backup.BackupRootFor(home), filepath.Join(stateDir, "backups"); got != want {
		t.Fatalf("BackupRootFor() = %q, want %q", got, want)
	}
	if got := countBackups(t, home); got == 0 {
		t.Errorf("el install debía dejar su respaldo previo bajo %s", filepath.Join(stateDir, "backups"))
	}
}
