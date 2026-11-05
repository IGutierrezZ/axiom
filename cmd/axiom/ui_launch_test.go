package main

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
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

func TestDetachedServerCommandWorkDir(t *testing.T) {
	ws := t.TempDir()
	cmd := newDetachedServerCommand("axiom", 47700, ws)
	if cmd.Dir == "" || isWithinWorkspace(ws, cmd.Dir) {
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

func isWithinWorkspace(ws, dir string) bool {
	rel, err := filepath.Rel(filepath.Clean(ws), filepath.Clean(dir))
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))))
}
