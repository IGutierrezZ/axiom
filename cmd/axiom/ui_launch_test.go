package main

import (
	"context"
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
