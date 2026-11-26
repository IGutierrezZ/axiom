package changestart

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func seedTree(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "work-item.md"), "# titulo\n")
	writeFile(t, filepath.Join(root, "attachments", "a.png"), "datos binarios")
	writeFile(t, filepath.Join(root, "attachments", "deep", "b.txt"), "b")
	if err := os.MkdirAll(filepath.Join(root, "vacio"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestCopyTreeRoundTripVerifies(t *testing.T) {
	src, dst := filepath.Join(t.TempDir(), "src"), filepath.Join(t.TempDir(), "dst")
	seedTree(t, src)
	if err := copyTree(fsSeams{}, src, dst); err != nil {
		t.Fatalf("copyTree: %v", err)
	}
	if err := verifyTree(src, dst); err != nil {
		t.Fatalf("verifyTree: %v", err)
	}
	if !exists(filepath.Join(dst, "vacio")) {
		t.Error("los directorios vacíos también se copian")
	}
	if got, _ := os.ReadFile(filepath.Join(dst, "attachments", "a.png")); string(got) != "datos binarios" {
		t.Errorf("contenido copiado = %q", got)
	}
}

func TestCopyTreeRefusesSymlinks(t *testing.T) {
	src, dst := filepath.Join(t.TempDir(), "src"), filepath.Join(t.TempDir(), "dst")
	seedTree(t, src)
	if err := os.Symlink(filepath.Join(src, "work-item.md"), filepath.Join(src, "enlace")); err != nil {
		t.Skipf("no se pueden crear enlaces simbólicos aquí: %v", err)
	}
	if err := copyTree(fsSeams{}, src, dst); err == nil {
		t.Fatal("un enlace en el staging es un error")
	}
}

func TestVerifyTreeDetectsDifferences(t *testing.T) {
	cases := map[string]func(t *testing.T, dst string){
		"falta":    func(t *testing.T, dst string) { _ = os.Remove(filepath.Join(dst, "work-item.md")) },
		"sobra":    func(t *testing.T, dst string) { writeFile(t, filepath.Join(dst, "extra.txt"), "x") },
		"distinto": func(t *testing.T, dst string) { writeFile(t, filepath.Join(dst, "work-item.md"), "# titulo!\n") },
		// Mismo tamaño y distinto contenido: solo lo detecta el SHA-256.
		"mismo_tamano": func(t *testing.T, dst string) { writeFile(t, filepath.Join(dst, "attachments", "deep", "b.txt"), "c") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			src, dst := filepath.Join(t.TempDir(), "src"), filepath.Join(t.TempDir(), "dst")
			seedTree(t, src)
			if err := copyTree(fsSeams{}, src, dst); err != nil {
				t.Fatal(err)
			}
			mutate(t, dst)
			if err := verifyTree(src, dst); !errors.Is(err, errTreeMismatch) {
				t.Fatalf("verifyTree = %v, quiero errTreeMismatch", err)
			}
		})
	}
}
