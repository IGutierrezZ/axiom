package legacyhome

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func TestMigrateHomeMovesLegacyState(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", "")
	home := t.TempDir()
	legacy := filepath.Join(home, ".gentle-ai")
	write(t, filepath.Join(legacy, "state.json"), `{"a":1}`)
	write(t, filepath.Join(legacy, "pi-codegraph.json"), `{"b":2}`)
	write(t, filepath.Join(legacy, "cache", "x"), "x")

	if err := MigrateHome(home); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(home, ".axiom", "state.json")); got != `{"a":1}` {
		t.Errorf("state.json = %q", got)
	}
	if got := read(t, filepath.Join(home, ".axiom", "pi-codegraph.json")); got != `{"b":2}` {
		t.Errorf("pi-codegraph.json = %q", got)
	}
	if exists(legacy) {
		t.Errorf("~/.gentle-ai debía retirarse al quedar vacío")
	}

	// Idempotente: una segunda ejecución no cambia nada ni falla.
	if err := MigrateHome(home); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(home, ".axiom", "state.json")); got != `{"a":1}` {
		t.Errorf("state.json tras repetir = %q", got)
	}
}

func TestMigrateHomeKeepsExistingNewState(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", "")
	home := t.TempDir()
	write(t, filepath.Join(home, ".gentle-ai", "state.json"), "viejo")
	write(t, filepath.Join(home, ".axiom", "state.json"), "nuevo")

	if err := MigrateHome(home); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(home, ".axiom", "state.json")); got != "nuevo" {
		t.Errorf("la ruta nueva se pisó: %q", got)
	}
	if got := read(t, filepath.Join(home, ".gentle-ai", "state.json")); got != "viejo" {
		t.Errorf("la ruta vieja se perdió: %q", got)
	}
}

func TestMigrateHomeLeavesBinAlone(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", "")
	home := t.TempDir()
	write(t, filepath.Join(home, ".gentle-ai", "bin", "launcher"), "l")
	if err := MigrateHome(home); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(home, ".gentle-ai", "bin", "launcher")) {
		t.Errorf("el lanzador heredado no debe tocarse")
	}
}

func TestMigrateHomeMovesBackupsWithoutOverwriting(t *testing.T) {
	t.Setenv("AXIOM_STATE_DIR", "")
	home := t.TempDir()
	oldRoot := filepath.Join(home, ".gentle-ai", "backups")
	newRoot := filepath.Join(home, ".axiom", "backups")
	manifest := func(root, id string) string {
		dir := filepath.Join(root, id)
		b, _ := json.Marshal(map[string]any{
			"id":       id,
			"root_dir": dir,
			"entries":  []any{map[string]any{"snapshot_path": filepath.Join(dir, "files", "f")}},
		})
		return string(b)
	}
	write(t, filepath.Join(oldRoot, "mover", "manifest.json"), manifest(oldRoot, "mover"))
	write(t, filepath.Join(oldRoot, "choque", "manifest.json"), manifest(oldRoot, "choque"))
	write(t, filepath.Join(newRoot, "choque", "manifest.json"), "nuevo")

	if err := MigrateHome(home); err != nil {
		t.Fatal(err)
	}
	var got struct {
		RootDir string `json:"root_dir"`
		Entries []struct {
			SnapshotPath string `json:"snapshot_path"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(read(t, filepath.Join(newRoot, "mover", "manifest.json"))), &got); err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(newRoot, "mover"); got.RootDir != want {
		t.Errorf("root_dir = %q, want %q", got.RootDir, want)
	}
	if want := filepath.Join(newRoot, "mover", "files", "f"); got.Entries[0].SnapshotPath != want {
		t.Errorf("snapshot_path = %q, want %q", got.Entries[0].SnapshotPath, want)
	}
	if read(t, filepath.Join(newRoot, "choque", "manifest.json")) != "nuevo" {
		t.Errorf("un id existente en la raíz nueva se pisó")
	}
	if !exists(filepath.Join(oldRoot, "choque", "manifest.json")) {
		t.Errorf("el id en conflicto debe quedar en la raíz vieja")
	}
}

func TestMoveIfAbsent(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "s", "f"), filepath.Join(dir, "d", "n", "f")
	write(t, src, "1")
	moved, err := MoveIfAbsent(src, dst)
	if err != nil || !moved {
		t.Fatalf("moved=%v err=%v", moved, err)
	}
	if moved, err := MoveIfAbsent(src, dst); err != nil || moved {
		t.Fatalf("repetir: moved=%v err=%v", moved, err)
	}
}
