package dashboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IGutierrezZ/axiom/v3/internal/backup"
)

// TestGetBackupsReportsSizeOnDisk comprueba que cada respaldo lleva su tamaño en
// disco (la suma de los ficheros de su carpeta), que la pantalla Ecosistema muestra
// junto a la fecha. El manifiesto no guarda el tamaño, así que se calcula al listar.
func TestGetBackupsReportsSizeOnDisk(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	dir := filepath.Join(backup.BackupRootFor(home), "20261029120000")
	snapshot := filepath.Join(dir, "files", "settings.json")
	if err := os.MkdirAll(filepath.Dir(snapshot), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(snapshot, []byte(strings.Repeat("x", 1234)), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	manifestPath := filepath.Join(dir, backup.ManifestFilename)
	if err := backup.WriteManifest(manifestPath, backup.Manifest{
		ID:        "20261029120000",
		CreatedAt: time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC),
		RootDir:   dir,
		Entries: []backup.ManifestEntry{
			{OriginalPath: filepath.Join(home, ".claude", "settings.json"), SnapshotPath: snapshot, Existed: true},
		},
		FileCount: 1,
	}); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	manifestInfo, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	items, err := NewService(t.TempDir()).GetBackups()
	if err != nil {
		t.Fatalf("GetBackups: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("GetBackups devolvió %d respaldos, se esperaba 1", len(items))
	}
	if want := 1234 + manifestInfo.Size(); items[0].SizeBytes != want {
		t.Errorf("SizeBytes = %d, se esperaba %d (fichero del respaldo + manifiesto)", items[0].SizeBytes, want)
	}
	if len(items[0].Files) != 1 {
		t.Errorf("Files = %v, se esperaba un fichero", items[0].Files)
	}
}

// TestBackupSizeBytesToleratesMissingRoot comprueba que un respaldo cuya carpeta
// ya no existe no rompe el listado: el tamaño queda a cero y se omite del JSON.
func TestBackupSizeBytesToleratesMissingRoot(t *testing.T) {
	if got := backupSizeBytes(""); got != 0 {
		t.Errorf("backupSizeBytes(\"\") = %d, se esperaba 0", got)
	}
	if got := backupSizeBytes(filepath.Join(t.TempDir(), "no-existe")); got != 0 {
		t.Errorf("backupSizeBytes(carpeta inexistente) = %d, se esperaba 0", got)
	}
}

// TestCreateBackupFollowsAxiomStateDir comprueba que el respaldo bajo demanda va a
// la raíz de estado de Axiom y que, con AXIOM_STATE_DIR, el listado lo encuentra
// ahí y no deja nada en ~/.axiom.
func TestCreateBackupFollowsAxiomStateDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	stateDir := filepath.Join(t.TempDir(), "estado")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "state.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AXIOM_STATE_DIR", stateDir)

	svc := NewService(t.TempDir())
	item, err := svc.CreateBackup("prueba")
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "backups", item.Name, backup.ManifestFilename)); err != nil {
		t.Errorf("el respaldo debe estar bajo AXIOM_STATE_DIR: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".axiom")); !os.IsNotExist(err) {
		t.Errorf("no debe crearse ~/.axiom con AXIOM_STATE_DIR definida; stat err=%v", err)
	}
	items, err := svc.GetBackups()
	if err != nil {
		t.Fatalf("GetBackups: %v", err)
	}
	if len(items) != 1 || items[0].Name != item.Name {
		t.Errorf("GetBackups() = %v, se esperaba el respaldo recién creado", items)
	}
}
