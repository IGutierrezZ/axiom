package copilotconfig

import (
	"os"
	"path/filepath"
	"testing"
)

// setRealHome hace que os.UserHomeDir devuelva dir en Windows y en Unix.
func setRealHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

func TestRootDefaultsToDotCopilot(t *testing.T) {
	home := t.TempDir()
	setRealHome(t, home)
	t.Setenv(EnvVar, "")

	if got, want := Root(home), filepath.Join(home, ".copilot"); got != want {
		t.Fatalf("Root() = %q, want %q", got, want)
	}
}

func TestRootHonoursCopilotHomeForTheRealUserHome(t *testing.T) {
	home := t.TempDir()
	setRealHome(t, home)
	custom := filepath.Join(t.TempDir(), "custom-copilot")
	t.Setenv(EnvVar, custom)

	if got := Root(home); got != custom {
		t.Fatalf("Root() = %q, want %q", got, custom)
	}
}

func TestRootIgnoresCopilotHomeOutsideTheRealUserHome(t *testing.T) {
	setRealHome(t, t.TempDir())
	t.Setenv(EnvVar, filepath.Join(t.TempDir(), "custom-copilot"))

	sandbox := t.TempDir()
	if got, want := Root(sandbox), filepath.Join(sandbox, ".copilot"); got != want {
		t.Fatalf("Root() = %q, want %q: una raíz aislada debe quedarse contenida", got, want)
	}
}

func TestRootIgnoresRelativeCopilotHome(t *testing.T) {
	home := t.TempDir()
	setRealHome(t, home)
	t.Setenv(EnvVar, "relative/dir")

	if got, want := Root(home), filepath.Join(home, ".copilot"); got != want {
		t.Fatalf("Root() = %q, want %q", got, want)
	}
}

func TestHasCLIStateIgnoresTheSharedSkillsFolder(t *testing.T) {
	root := t.TempDir()
	if HasCLIState(root) {
		t.Fatal("una raíz vacía no es estado de Copilot CLI")
	}
	if HasCLIState(filepath.Join(root, "no-existe")) {
		t.Fatal("una raíz inexistente no es estado de Copilot CLI")
	}
	if err := os.MkdirAll(filepath.Join(root, "skills", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if HasCLIState(root) {
		t.Fatal("solo skills es de VS Code Copilot y de Axiom, no prueba que haya CLI")
	}
	if err := os.WriteFile(filepath.Join(root, "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !HasCLIState(root) {
		t.Fatal("settings.json es estado propio de Copilot CLI")
	}
}
