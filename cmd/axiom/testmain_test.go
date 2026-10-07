package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/hub"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// sandboxHome is the throwaway home directory every test in this package runs
// under. It is empty only if TestMain did not run.
var sandboxHome string

// TestMain keeps the cmd/axiom test binary away from the machine it runs on.
//
// runInit registers the initialised project in the global hub catalog and marks
// it active (hub.Register + SetActive), and hub.NewManager("") resolves that
// catalog from os.UserHomeDir(): ~/.axiom/workspaces.json. Without a sandbox,
// every in-process runInit call made by the knowledge CLI tests leaves a stale
// temp-directory entry — and a changed active workspace — in the developer's real
// hub. Pointing HOME/USERPROFILE at a throwaway directory for the whole binary
// (the same pattern as internal/dashboard, internal/app and internal/cli) makes
// every home-relative path hermetic, including the subprocess tests, which
// inherit this process environment. AXIOM_NO_PERSISTENT_PATH keeps the real
// binary those subprocess tests run (which is not a Go test binary, so it
// bypasses the in-process guard) from writing the developer's real user PATH.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	home, err := os.MkdirTemp("", "axiom-cmd-test-home-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmd/axiom tests: create sandbox home: %v\n", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(home) }()

	for key, value := range map[string]string{
		"HOME":                        home,
		"USERPROFILE":                 home,
		"LOCALAPPDATA":                filepath.Join(home, "AppData", "Local"),
		"APPDATA":                     filepath.Join(home, "AppData", "Roaming"),
		system.NoPersistentPathEnvVar: "1",
	} {
		if err := os.Setenv(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "cmd/axiom tests: set %s: %v\n", key, err)
			return 1
		}
	}
	// A developer's own AXIOM_STATE_DIR is cleared, not pinned to one shared
	// directory: system.AxiomDir must keep following the sandbox home.
	if err := os.Unsetenv(system.EnvStateDirAxiom); err != nil {
		fmt.Fprintf(os.Stderr, "cmd/axiom tests: unset %s: %v\n", system.EnvStateDirAxiom, err)
		return 1
	}
	sandboxHome = home

	return m.Run()
}

// TestSandboxHomeIsInForce pins the TestMain contract: if the sandbox is ever
// dropped, the package would silently go back to touching the real home and hub.
func TestSandboxHomeIsInForce(t *testing.T) {
	if sandboxHome == "" {
		t.Fatal("sandboxHome está vacío: TestMain no aisló el HOME del paquete")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() falló: %v", err)
	}
	if home != sandboxHome {
		t.Errorf("os.UserHomeDir() = %s, want the sandbox home %s", home, sandboxHome)
	}
	if got := os.Getenv(system.NoPersistentPathEnvVar); got != "1" {
		t.Errorf("%s = %q, want %q", system.NoPersistentPathEnvVar, got, "1")
	}

	hubMgr, err := hub.NewManager("")
	if err != nil {
		t.Fatalf("hub.NewManager(\"\") falló: %v", err)
	}
	want := filepath.Join(sandboxHome, ".axiom", "workspaces.json")
	if got := hubMgr.GetConfigPath(); got != want {
		t.Errorf("hub.NewManager(\"\").GetConfigPath() = %s, want %s", got, want)
	}
}

// TestRunInitRegistersInSandboxHub reproduces the original leak end to end: an
// in-process runInit must land in the sandbox hub, never in the real one.
func TestRunInitRegistersInSandboxHub(t *testing.T) {
	project := t.TempDir()

	runInit([]string{"-path", project, "-name", "SandboxHubProbe"})

	hubMgr, err := hub.NewManager("")
	if err != nil {
		t.Fatalf("hub.NewManager(\"\") falló: %v", err)
	}
	if !strings.HasPrefix(hubMgr.GetConfigPath(), sandboxHome) {
		t.Fatalf("el hub apunta fuera del sandbox: %s (sandbox %s)", hubMgr.GetConfigPath(), sandboxHome)
	}
	cfg, err := hubMgr.Load()
	if err != nil {
		t.Fatalf("no se pudo cargar el hub del sandbox: %v", err)
	}

	want := filepath.Clean(project)
	for _, w := range cfg.Workspaces {
		if strings.EqualFold(filepath.Clean(w.Path), want) {
			return
		}
	}
	t.Errorf("runInit no registró %s en el hub del sandbox %s: %+v", want, hubMgr.GetConfigPath(), cfg.Workspaces)
}
