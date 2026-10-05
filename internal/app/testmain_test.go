package app

import (
	"os"
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

	code := m.Run()
	_ = os.RemoveAll(testHome)
	os.Exit(code)
}
