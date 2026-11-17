package sddstatus

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// ChangeArtifactPaths devuelve las specs de dominio ordenadas y, sin ellas, el spec.md plano.
func TestChangeArtifactPathsSpecsOrderedAndFlatFallback(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"specs/b/spec.md", "specs/a/spec.md", "tasks.md"} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("contenido\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := ChangeArtifactPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(root, "specs", "a", "spec.md"), filepath.Join(root, "specs", "b", "spec.md")}
	if !reflect.DeepEqual(paths.Specs, want) {
		t.Errorf("Specs = %v, se esperaba %v", paths.Specs, want)
	}
	if len(paths.Tasks) != 1 || filepath.Base(paths.Tasks[0]) != "tasks.md" {
		t.Errorf("Tasks = %v, se esperaba tasks.md", paths.Tasks)
	}

	flat := t.TempDir()
	if err := os.WriteFile(filepath.Join(flat, "spec.md"), []byte("plana\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths, err = ChangeArtifactPaths(flat)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths.Specs) != 1 || filepath.Base(paths.Specs[0]) != "spec.md" {
		t.Errorf("Specs planas = %v", paths.Specs)
	}
}
