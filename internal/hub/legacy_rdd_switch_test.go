package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLegacyHub escribe un workspaces.json crudo con la entrada que dejó una
// versión anterior y devuelve el gestor que lo lee.
func writeLegacyHub(t *testing.T, entryExtra string) *Manager {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workspaces.json")
	raw := `{"version":"1.0","active_workspace":"p","workspaces":[{"id":"p","name":"Proyecto","path":"/tmp/proyecto","is_configured":true,"worktree_mode":"worktree"` +
		entryExtra + `}]}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	return mgr
}

// Un rdd_mode que dejó una versión anterior se ignora, conserva el resto de la
// entrada y desaparece en la siguiente escritura, también con un tipo inesperado.
func TestLegacyRDDSwitchInTheHubIsIgnoredAndDroppedOnTheNextWrite(t *testing.T) {
	for name, extra := range map[string]string{
		"valor válido":      `,"rdd_mode":"on"`,
		"objeto inesperado": `,"rdd_mode":{"x":1}`,
		"número inesperado": `,"rdd_mode":7`,
	} {
		t.Run(name, func(t *testing.T) {
			mgr := writeLegacyHub(t, extra)

			cfg, err := mgr.Load()
			if err != nil {
				t.Fatalf("Load(): %v", err)
			}
			if len(cfg.Workspaces) != 1 || cfg.Workspaces[0].WorktreeMode != "worktree" || !cfg.Workspaces[0].IsConfigured {
				t.Fatalf("la entrada perdió el resto de campos: %+v", cfg.Workspaces)
			}
			if err := mgr.Save(cfg); err != nil {
				t.Fatalf("Save(): %v", err)
			}
			raw, err := os.ReadFile(mgr.GetConfigPath())
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "rdd_mode") {
				t.Errorf("la escritura debe retirar rdd_mode:\n%s", raw)
			}
			if !strings.Contains(string(raw), "worktree_mode") {
				t.Errorf("la escritura no debe perder worktree_mode:\n%s", raw)
			}
		})
	}
}

// Una entrada con solo los campos mínimos y un rdd_mode suelto también carga.
func TestLegacyRDDSwitchInAPartialHubEntryLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspaces.json")
	raw := `{"version":"1.0","workspaces":[{"id":"p","path":"/tmp/p","rdd_mode":"off"}]}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if len(cfg.Workspaces) != 1 || cfg.Workspaces[0].ID != "p" {
		t.Fatalf("entrada inesperada: %+v", cfg.Workspaces)
	}
	if err := mgr.Save(cfg); err != nil {
		t.Fatal(err)
	}
	out, _ := os.ReadFile(path)
	if strings.Contains(string(out), "rdd_mode") {
		t.Errorf("la escritura debe retirar rdd_mode:\n%s", out)
	}
}
