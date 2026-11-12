package kickoff

import (
	"os"
	"path/filepath"
	"testing"
)

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// REQ-29.12: crea los padres ausentes y la retirada borra solo lo creado, en
// orden inverso.
func TestPrepareChangeRoot(t *testing.T) {
	t.Run("crea padres ausentes y la retirada los borra", func(t *testing.T) {
		base := t.TempDir()
		changeRoot := filepath.Join(base, "openspec", "changes", "x")
		rollback, err := PrepareChangeRoot(changeRoot)
		if err != nil {
			t.Fatal(err)
		}
		if !isDir(changeRoot) {
			t.Fatalf("la carpeta del cambio debe existir: %s", changeRoot)
		}
		rollback()
		if isDir(filepath.Join(base, "openspec")) {
			t.Error("la retirada debe borrar también los padres que creó")
		}
		if !isDir(base) {
			t.Error("la retirada nunca toca lo que ya existía")
		}
	})

	t.Run("solo retira lo creado: un padre previo se conserva", func(t *testing.T) {
		base := t.TempDir()
		changes := filepath.Join(base, "openspec", "changes")
		if err := os.MkdirAll(changes, 0o755); err != nil {
			t.Fatal(err)
		}
		changeRoot := filepath.Join(changes, "x")
		rollback, err := PrepareChangeRoot(changeRoot)
		if err != nil {
			t.Fatal(err)
		}
		rollback()
		if isDir(changeRoot) {
			t.Error("la carpeta creada debe retirarse")
		}
		if !isDir(changes) {
			t.Error("el padre preexistente debe conservarse")
		}
	})

	t.Run("no borra un directorio que ya no está vacío", func(t *testing.T) {
		base := t.TempDir()
		changeRoot := filepath.Join(base, "openspec", "changes", "x")
		rollback, err := PrepareChangeRoot(changeRoot)
		if err != nil {
			t.Fatal(err)
		}
		extra := filepath.Join(changeRoot, "ajeno.md")
		if err := os.WriteFile(extra, []byte("no tocar\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		rollback()
		if _, err := os.Stat(extra); err != nil {
			t.Errorf("un fichero ajeno no se borra: %v", err)
		}
		if !isDir(filepath.Join(base, "openspec")) {
			t.Error("los padres de un directorio no vacío tampoco se borran")
		}
	})

	t.Run("carpeta existente intacta y retirada sin efecto", func(t *testing.T) {
		changeRoot := filepath.Join(t.TempDir(), "x")
		if err := os.Mkdir(changeRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		marker := filepath.Join(changeRoot, "proposal.md")
		if err := os.WriteFile(marker, []byte("p\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		rollback, err := PrepareChangeRoot(changeRoot)
		if err != nil {
			t.Fatal(err)
		}
		rollback()
		if _, err := os.Stat(marker); err != nil || !isDir(changeRoot) {
			t.Errorf("una carpeta existente queda intacta: %v", err)
		}
	})

	t.Run("una carpeta vacía preexistente no se borra", func(t *testing.T) {
		changeRoot := filepath.Join(t.TempDir(), "x")
		if err := os.Mkdir(changeRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		rollback, err := PrepareChangeRoot(changeRoot)
		if err != nil {
			t.Fatal(err)
		}
		rollback()
		if !isDir(changeRoot) {
			t.Error("la retirada no borra una carpeta que no creó, aunque esté vacía")
		}
	})

	t.Run("un fichero en el lugar de la carpeta es un error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "x")
		if err := os.WriteFile(path, []byte("f"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := PrepareChangeRoot(path); err == nil {
			t.Error("debe rechazar un fichero donde iría la carpeta")
		}
	})

	t.Run("la retirada es idempotente", func(t *testing.T) {
		changeRoot := filepath.Join(t.TempDir(), "a", "x")
		rollback, err := PrepareChangeRoot(changeRoot)
		if err != nil {
			t.Fatal(err)
		}
		rollback()
		rollback()
	})
}
