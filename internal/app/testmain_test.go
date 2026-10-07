package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	// Pin the Go toolchain's own locations before LOCALAPPDATA/APPDATA move into
	// the sandbox: on Windows GOCACHE defaults under LOCALAPPDATA and GOENV (the
	// `go env -w` settings) under APPDATA, so the tests that `go build
	// ./cmd/axiom` would otherwise start from a cold build cache and lose the
	// developer's proxy settings.
	pinGoToolchainEnv("GOCACHE", "GOENV")
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

// pinGoToolchainEnv exports the value `go env` resolves for each unset key, so
// later changes to the home-like variables it derives from do not move it. A
// missing go binary leaves the key unset; the tests that need go fail on their
// own with a clearer message.
func pinGoToolchainEnv(keys ...string) {
	for _, key := range keys {
		if os.Getenv(key) != "" {
			continue
		}
		output, err := exec.Command("go", "env", key).Output()
		if err != nil {
			continue
		}
		if value := strings.TrimSpace(string(output)); value != "" {
			_ = os.Setenv(key, value)
		}
	}
}
