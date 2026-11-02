package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/backup"
)

// Sin AXIOM_STATE_DIR las rutas por defecto son las de siempre, y un sync que no
// escribe nada no deja una copia de respaldo nueva; uno que sí escribe la conserva.
func TestSyncWithoutChangesLeavesNoNewBackup(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", "")
	root, _, home := workspaceRootFixture(t, false, true)
	t.Chdir(root)

	installInProject(t, root, "--component", "engram", "--persona", "neutral")
	if got, want := backup.BackupRootFor(home), filepath.Join(home, ".axiom", "backups"); got != want {
		t.Fatalf("BackupRootFor() = %q, want %q", got, want)
	}
	afterInstall := countBackups(t, home)
	if afterInstall != 1 {
		t.Fatalf("tras el install debe haber exactamente 1 respaldo, hay %d", afterInstall)
	}

	for i := 1; i <= 2; i++ {
		result, err := RunSync([]string{"--scope", "workspace", "--agents", "claude-code"})
		if err != nil {
			t.Fatalf("RunSync() #%d error = %v", i, err)
		}
		if !result.NoOp {
			t.Fatalf("el sync #%d no debía cambiar nada: %v", i, result.ChangedFiles)
		}
		if got := countBackups(t, home); got != afterInstall {
			t.Errorf("tras el sync sin cambios #%d hay %d respaldos, want %d", i, got, afterInstall)
		}
	}

	// Un sync que sí escribe conserva su copia previa.
	if err := os.Remove(filepath.Join(root, ".mcp.json")); err != nil {
		t.Fatal(err)
	}
	result, err := RunSync([]string{"--scope", "workspace", "--agents", "claude-code"})
	if err != nil {
		t.Fatalf("RunSync() con cambios error = %v", err)
	}
	if result.NoOp {
		t.Fatal("el sync debía restaurar .mcp.json")
	}
	if got := countBackups(t, home); got != afterInstall+1 {
		t.Errorf("un sync con cambios debe conservar su respaldo previo: hay %d, want %d", got, afterInstall+1)
	}
}
