package adoptcheck

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/adoption"
)

const requisitosDoc = "# Requisitos\n\nIntroducción.\n\n## RF-001 Alta de usuario\n\nEl sistema permite el alta.\n\n## RF-002 Baja de usuario\n\nEl sistema permite la baja.\n"

// writeTree escribe ficheros (ruta con `/` → contenido) bajo root.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func mustScan(t *testing.T, root string) adoption.Inventory {
	t.Helper()
	inv, err := adoption.Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return inv
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			out[rel] = "<dir>"
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
