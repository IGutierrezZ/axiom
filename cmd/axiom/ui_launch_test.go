package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestDetachedChildArgs(t *testing.T) {
	args := detachedChildArgs(47700, "/tmp/ws")
	for _, want := range []string{"ui", "--no-browser", "--port", "47700", "--path", "/tmp/ws"} {
		if !slices.Contains(args, want) {
			t.Errorf("faltan %q en %v", want, args)
		}
	}
	for _, banned := range []string{"--detach"} {
		if slices.Contains(args, banned) {
			t.Errorf("%q no debe estar en %v", banned, args)
		}
	}
}

func TestNeutralWorkDir(t *testing.T) {
	root := t.TempDir()
	ws := filepath.Join(root, "ws")
	inside := filepath.Join(ws, "sub")
	outside := filepath.Join(root, "otro")
	volumeRoot := filepath.VolumeName(ws) + string(filepath.Separator)

	cases := []struct {
		name       string
		candidates []string
		want       string
	}{
		{"primer candidato fuera", []string{outside, root}, outside},
		{"salta el propio workspace", []string{ws, outside}, outside},
		{"salta un subdirectorio del workspace", []string{inside, outside}, outside},
		{"salta candidatos vacíos", []string{"", outside}, outside},
		{"directorio padre del workspace vale", []string{root}, root},
		{"prefijo de nombre no es contención", []string{ws + "-bis"}, ws + "-bis"},
		{"sin candidatos válidos usa la raíz del volumen", []string{ws, inside}, volumeRoot},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := neutralWorkDir(ws, tc.candidates...); got != tc.want {
				t.Errorf("neutralWorkDir(%q, %v) = %q; quiero %q", ws, tc.candidates, got, tc.want)
			}
		})
	}
}

func TestServerWorkDirOutsideWorkspace(t *testing.T) {
	ws := t.TempDir()
	got := serverWorkDir(ws)
	if isWithinDir(ws, got) {
		t.Fatalf("serverWorkDir(%q) = %q, dentro del workspace", ws, got)
	}
	if home, err := os.UserHomeDir(); err == nil && !isWithinDir(ws, home) && got != home {
		t.Errorf("serverWorkDir(%q) = %q; quiero el home %q", ws, got, home)
	}
}

func TestDetachedServerCommandWorkDir(t *testing.T) {
	ws := t.TempDir()
	cmd := newDetachedServerCommand("axiom", 47700, ws)
	if cmd.Dir == "" || isWithinDir(ws, cmd.Dir) {
		t.Fatalf("cmd.Dir = %q; debe estar fuera del workspace %q", cmd.Dir, ws)
	}
	if !slices.Contains(cmd.Args, ws) {
		t.Errorf("el workspace debe llegar por --path: %v", cmd.Args)
	}
}

func TestWaitForAxiom(t *testing.T) {
	calls := 0
	probe := func(context.Context) bool { calls++; return calls >= 3 }
	if !waitForAxiom(context.Background(), probe, time.Millisecond, time.Second) {
		t.Fatal("debía tener éxito tras 3 sondeos")
	}
	if calls != 3 {
		t.Errorf("sondeos = %d, want 3", calls)
	}

	never := func(context.Context) bool { return false }
	start := time.Now()
	if waitForAxiom(context.Background(), never, time.Millisecond, 30*time.Millisecond) {
		t.Fatal("debía agotar el tiempo")
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("el timeout tardó demasiado: %s", time.Since(start))
	}
}
