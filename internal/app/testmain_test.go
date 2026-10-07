package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// TestMain gives the whole internal/app test binary a safe, sandboxed HOME
// (mirroring internal/cli's own TestMain, which many of this package's tests
// indirectly exercise through cli.RunInstall/RunSync): runUpdate resolves the
// user's home directory directly (it has no cli-level override var to reuse),
// so without this, os.UserHomeDir() would read whatever HOME happens to be set
// to in the process running `go test` — the developer's real home on a machine
// that has not sandboxed it for this specific test.
func TestMain(m *testing.M) {
	testHome, err := os.MkdirTemp("", "gentle-ai-app-test-home-*")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("HOME", testHome); err != nil {
		panic(err)
	}
	if err := os.Setenv("USERPROFILE", testHome); err != nil {
		panic(err)
	}
	// Opt out of persistent user PATH writes for every subprocess this binary
	// starts: a real axiom binary is not a Go test binary, so it would not hit
	// the in-process guard and would write the real HKCU\Environment PATH.
	if err := os.Setenv(system.NoPersistentPathEnvVar, "1"); err != nil {
		panic(err)
	}
	// Keep every other home-like location inside the sandbox too: on Windows the
	// agent adapters and the update detector read LOCALAPPDATA/APPDATA. A
	// developer's own AXIOM_STATE_DIR must not leak in either, but it is cleared
	// rather than pointed at one shared directory: system.AxiomDir derives the
	// state location from the home it is given, and many tests inject a
	// per-test home (selfUpdateHomeDirFn and similar) that a single global
	// AXIOM_STATE_DIR would silently override, sharing state across tests.
	if err := os.Unsetenv(system.EnvStateDirAxiom); err != nil {
		panic(err)
	}
	for key, value := range map[string]string{
		"LOCALAPPDATA": filepath.Join(testHome, "AppData", "Local"),
		"APPDATA":      filepath.Join(testHome, "AppData", "Roaming"),
	} {
		if err := os.Setenv(key, value); err != nil {
			panic(err)
		}
	}

	code := m.Run()
	_ = os.RemoveAll(testHome)
	os.Exit(code)
}
