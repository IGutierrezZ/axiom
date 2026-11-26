package sddstatus

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/kickoff"
)

func TestTasksGateArtifactPaths(t *testing.T) {
	dir := t.TempDir()
	tasks := filepath.Join(dir, "tasks.md")
	pruebas := filepath.Join(dir, "pruebas.md")

	if got := TasksGateArtifactPaths(dir, nil); !reflect.DeepEqual(got, []string{tasks}) {
		t.Fatalf("sin pruebas.md = %v, se esperaba solo tasks.md", got)
	}
	if err := os.WriteFile(pruebas, []byte(" \n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := TasksGateArtifactPaths(dir, nil); !reflect.DeepEqual(got, []string{tasks}) {
		t.Fatalf("pruebas.md en blanco = %v, se esperaba solo tasks.md", got)
	}
	if err := os.WriteFile(pruebas, []byte("# Pruebas\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := TasksGateArtifactPaths(dir, nil); !reflect.DeepEqual(got, []string{tasks, pruebas}) {
		t.Fatalf("con pruebas.md = %v", got)
	}
	roles := []kickoff.KickoffRole{{Role: "backend", TasksFile: "tasks.backend.md"}, {Role: "frontend", TasksFile: "tasks.frontend.md"}}
	want := []string{filepath.Join(dir, "tasks.backend.md"), filepath.Join(dir, "tasks.frontend.md"), pruebas}
	if got := TasksGateArtifactPaths(dir, roles); !reflect.DeepEqual(got, want) {
		t.Fatalf("por rol = %v, se esperaba %v", got, want)
	}
}
