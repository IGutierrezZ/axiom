package dashboard

import (
	"fmt"
	"os"
	"testing"
)

// sandboxHome is the throwaway home directory every test in this package runs
// under. It is empty only if TestMain did not run.
var sandboxHome string

// TestMain keeps the dashboard test binary away from the machine it runs on.
//
// The service resolves os.UserHomeDir for the hub catalog, backups, the
// doctor, the skills index mirror and the install state, and several handlers
// reach tools that write outside the process (sync, upgrade, codegraph). Pointing
// HOME/USERPROFILE at a throwaway directory for the whole binary — the same
// pattern as internal/app, internal/cli and internal/reviewtransaction — makes
// those paths hermetic even for a test that forgets to isolate itself.
// DO_NOT_TRACK keeps telemetry offline for the same reason.
//
// Tests that run a side-effecting flow (sync, upgrade, reindex, archive sync)
// must also use a t.TempDir() project and, where the production code shells out,
// the package seams (runAppArgsFn, upgradeSequenceReportFn) or a stand-in binary
// on PATH, so nothing is written to the repository checkout either.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	home, err := os.MkdirTemp("", "axiom-dashboard-test-home-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "dashboard tests: create sandbox home: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(home) }()

	for key, value := range map[string]string{
		"HOME":         home,
		"USERPROFILE":  home,
		"DO_NOT_TRACK": "1",
	} {
		if err := os.Setenv(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "dashboard tests: set %s: %v\n", key, err)
			return 1
		}
	}
	sandboxHome = home

	return m.Run()
}

// TestSandboxHomeIsInForce pins the TestMain contract: if the sandbox is ever
// dropped, the package would silently go back to touching the real home.
func TestSandboxHomeIsInForce(t *testing.T) {
	if sandboxHome == "" {
		t.Fatal("sandboxHome está vacío: TestMain no aisló el HOME del paquete")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() falló: %v", err)
	}
	sandboxInfo, err := os.Stat(sandboxHome)
	if err != nil {
		t.Fatalf("Stat(%s): %v", sandboxHome, err)
	}
	homeInfo, err := os.Stat(home)
	if err != nil {
		t.Fatalf("Stat(%s): %v", home, err)
	}
	if !os.SameFile(sandboxInfo, homeInfo) {
		t.Errorf("os.UserHomeDir() = %s, want the sandbox home %s", home, sandboxHome)
	}
}
