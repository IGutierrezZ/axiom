package main

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func fakeEnv(env map[string]string) func(string) string {
	return func(k string) string { return env[k] }
}

func existsIn(paths ...string) func(string) bool {
	return func(p string) bool { return slices.Contains(paths, p) }
}

func noLookPath(string) (string, error) { return "", errors.New("no encontrado") }

func TestResolveAppBrowserWindows(t *testing.T) {
	env := fakeEnv(map[string]string{
		"ProgramFiles(x86)": `X86`,
		"ProgramFiles":      `PF`,
		"LOCALAPPDATA":      `LA`,
	})
	edge := filepath.Join("X86", `Microsoft\Edge\Application\msedge.exe`)
	chrome := filepath.Join("LA", `Google\Chrome\Application\chrome.exe`)

	got, ok := resolveAppBrowser("windows", env, existsIn(edge, chrome), noLookPath)
	if !ok || got != edge {
		t.Fatalf("con Edge y Chrome: got %q, %v; want %q", got, ok, edge)
	}
	got, ok = resolveAppBrowser("windows", env, existsIn(chrome), noLookPath)
	if !ok || got != chrome {
		t.Fatalf("solo Chrome: got %q, %v; want %q", got, ok, chrome)
	}
	if got, ok = resolveAppBrowser("windows", env, existsIn(), noLookPath); ok {
		t.Fatalf("sin navegadores: got %q, want ok=false", got)
	}
}

func TestResolveAppBrowserUnix(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "chromium" {
			return "/usr/bin/chromium", nil
		}
		return "", errors.New("no encontrado")
	}
	got, ok := resolveAppBrowser("linux", fakeEnv(nil), existsIn(), lookPath)
	if !ok || got != "/usr/bin/chromium" {
		t.Fatalf("got %q, %v; want /usr/bin/chromium", got, ok)
	}
	if got, ok = resolveAppBrowser("linux", fakeEnv(nil), existsIn(), noLookPath); ok {
		t.Fatalf("sin navegadores: got %q, want ok=false", got)
	}
}

func TestDetachedChildArgs(t *testing.T) {
	args := detachedChildArgs(47700, "/tmp/ws")
	for _, want := range []string{"ui", "--no-browser", "--port", "47700", "--path", "/tmp/ws"} {
		if !slices.Contains(args, want) {
			t.Errorf("faltan %q en %v", want, args)
		}
	}
	for _, banned := range []string{"--detach", "--app"} {
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
