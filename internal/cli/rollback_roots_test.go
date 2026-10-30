package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/backup"
	opencodeactivation "github.com/IGutierrezZ/axiom/v3/internal/opencode"
)

func TestRollbackRootsAddTheAxiomLauncherDirOnlyWhenStateDirIsOutsideHome(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()

	t.Setenv("AXIOM_STATE_DIR", "")
	if got := rollbackRoots(home, workspace); !slices.Equal(got, []string{home, workspace}) {
		t.Fatalf("rollbackRoots() sin AXIOM_STATE_DIR = %v, want home y workspace", got)
	}

	stateDir := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", stateDir)
	want := []string{home, workspace, filepath.Join(stateDir, "bin")}
	if got := rollbackRoots(home, workspace); !slices.Equal(got, want) {
		t.Fatalf("rollbackRoots() con AXIOM_STATE_DIR fuera de home = %v, want %v", got, want)
	}
}

// TestRollbackRestoresTheOpenCodeLauncherUnderAnAxiomStateDirOutsideHome reproduce un
// install que falla después de escribir el lanzador de OpenCode cuando
// AXIOM_STATE_DIR apunta fuera de HOME: el rollback debe restaurarlo, y una ruta que
// no cuelga de ninguna raíz sigue rechazándose.
func TestRollbackRestoresTheOpenCodeLauncherUnderAnAxiomStateDirOutsideHome(t *testing.T) {
	home := t.TempDir()
	workspace := t.TempDir()
	stateDir := t.TempDir()
	t.Setenv("AXIOM_STATE_DIR", stateDir)

	launcher := opencodeactivation.LauncherPaths(home, runtime.GOOS)[0]
	if !strings.HasPrefix(launcher, stateDir) {
		t.Fatalf("el lanzador %q debe vivir bajo AXIOM_STATE_DIR %q", launcher, stateDir)
	}
	if err := os.MkdirAll(filepath.Dir(launcher), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(launcher, []byte("original\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest, err := backup.NewSnapshotter().Create(filepath.Join(t.TempDir(), "snapshot"), []string{launcher})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(launcher, []byte("escrito por el install fallido\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	step := rollbackRestoreStep{id: "apply:rollback-restore", state: &runtimeState{manifest: manifest}, homeDir: home, workspaceDir: workspace}
	if err := step.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if got, err := os.ReadFile(launcher); err != nil || string(got) != "original\n" {
		t.Fatalf("lanzador tras el rollback = %q, %v; want el original", got, err)
	}

	outside := filepath.Join(t.TempDir(), "fuera.json")
	rejected := rollbackRestoreStep{
		id:           "apply:rollback-restore",
		state:        &runtimeState{manifest: backup.Manifest{Entries: []backup.ManifestEntry{{OriginalPath: outside, Existed: false}}}},
		homeDir:      home,
		workspaceDir: workspace,
	}
	if err := rejected.Rollback(); err == nil || !strings.Contains(err.Error(), "allowed root") {
		t.Fatalf("Rollback() con una ruta fuera de las raíces = %v, want rechazo por raíz no permitida", err)
	}
}
