package sdd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// copilotHome prepara un home aislado que además es el "home real" del proceso:
// COPILOT_HOME solo se respeta en ese caso. También confina las rutas de VS Code.
func copilotHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("COPILOT_HOME", "")
	return home
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	return string(data)
}

// Copilot CLI recibe el orquestador en su propio copilot-instructions.md y las
// skills en <raíz>/skills, y no toca la carpeta de prompts de VS Code.
func TestInjectCopilotCLIWritesSDDToItsOwnFiles(t *testing.T) {
	home := copilotHome(t)
	cli := mustAdapter(t, model.AgentGitHubCopilotCLI)

	result, err := Inject(home, cli, "")
	if err != nil {
		t.Fatalf("Inject(copilot-cli) error = %v", err)
	}
	if !result.Changed {
		t.Fatal("Inject(copilot-cli) changed = false")
	}

	prompt := filepath.Join(home, ".copilot", "copilot-instructions.md")
	if got := cli.SystemPromptFile(home); got != prompt {
		t.Fatalf("SystemPromptFile = %q, want %q", got, prompt)
	}
	text := readText(t, prompt)
	if !strings.Contains(text, "Spec-Driven Development") {
		t.Fatal("copilot-instructions.md no contiene el orquestador SDD")
	}
	if strings.Contains(text, "default `engram` when available") {
		t.Fatal("el orquestador de Copilot CLI conserva la cláusula previa a la preflight de sesión")
	}
	for _, rel := range []string{filepath.Join("sdd-init", "SKILL.md"), filepath.Join("_shared", "engram-convention.md")} {
		if _, err := os.Stat(filepath.Join(home, ".copilot", "skills", rel)); err != nil {
			t.Fatalf("falta la skill %s: %v", rel, err)
		}
	}
	vscodePrompt := mustAdapter(t, model.AgentVSCodeCopilot).SystemPromptFile(home)
	if _, err := os.Stat(vscodePrompt); !os.IsNotExist(err) {
		t.Fatalf("Copilot CLI escribió el prompt de VS Code %q (err=%v)", vscodePrompt, err)
	}

	again, err := Inject(home, cli, "")
	if err != nil || again.Changed {
		t.Fatalf("segundo Inject = %+v, %v; want sin cambios", again, err)
	}
}

// VS Code Copilot y Copilot CLI escriben las mismas skills en ~/.copilot/skills. El
// contenido no puede depender de quién escribe: con la identidad de runtime de cada
// uno se pisarían la copia en cada sync.
func TestCopilotCLIAndVSCodeShareSkillsWithoutOverwritingEachOther(t *testing.T) {
	home := copilotHome(t)
	vscode := mustAdapter(t, model.AgentVSCodeCopilot)
	cli := mustAdapter(t, model.AgentGitHubCopilotCLI)

	for _, adapter := range []struct {
		name string
		run  func() (InjectionResult, error)
	}{
		{"vscode", func() (InjectionResult, error) { return Inject(home, vscode, "") }},
		{"cli", func() (InjectionResult, error) { return Inject(home, cli, "") }},
	} {
		if _, err := adapter.run(); err != nil {
			t.Fatalf("primer Inject(%s): %v", adapter.name, err)
		}
	}

	ledger := filepath.Join(home, ".copilot", "skills", "_shared", "review-ledger-contract.md")
	first := readText(t, ledger)
	if strings.Contains(first, runtimeAgentIDPlaceholder) {
		t.Fatal("la skill compartida conserva el hueco de identidad sin resolver")
	}
	for _, identity := range []model.AgentID{model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI} {
		if strings.Contains(first, string(identity)) {
			t.Fatalf("la skill compartida lleva la identidad %q de uno solo de los dos agentes", identity)
		}
	}
	if !strings.Contains(first, "<runtime>") {
		t.Fatal("la skill compartida debe llevar el hueco genérico <runtime>")
	}

	// Un segundo sync de cada uno no cambia nada, en cualquier orden.
	for _, adapter := range []struct {
		name string
		run  func() (InjectionResult, error)
	}{
		{"cli", func() (InjectionResult, error) { return Inject(home, cli, "") }},
		{"vscode", func() (InjectionResult, error) { return Inject(home, vscode, "") }},
		{"cli", func() (InjectionResult, error) { return Inject(home, cli, "") }},
	} {
		result, err := adapter.run()
		if err != nil {
			t.Fatalf("segundo Inject(%s): %v", adapter.name, err)
		}
		if result.Changed {
			t.Fatalf("segundo Inject(%s) cambió ficheros: los dos agentes se pisan las skills compartidas", adapter.name)
		}
	}
	if second := readText(t, ledger); second != first {
		t.Fatal("la skill compartida cambió entre syncs")
	}
}

// Con COPILOT_HOME las carpetas dejan de coincidir: cada agente conserva su propia
// identidad de runtime y no se tocan entre sí.
func TestCopilotCLIWithCopilotHomeDoesNotShareSkillsWithVSCode(t *testing.T) {
	home := copilotHome(t)
	custom := filepath.Join(t.TempDir(), "custom-copilot")
	t.Setenv("COPILOT_HOME", custom)
	vscode := mustAdapter(t, model.AgentVSCodeCopilot)
	cli := mustAdapter(t, model.AgentGitHubCopilotCLI)

	if _, err := Inject(home, vscode, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := Inject(home, cli, ""); err != nil {
		t.Fatal(err)
	}

	vscodeLedger := readText(t, filepath.Join(home, ".copilot", "skills", "_shared", "review-ledger-contract.md"))
	cliLedger := readText(t, filepath.Join(custom, "skills", "_shared", "review-ledger-contract.md"))
	if !strings.Contains(vscodeLedger, string(model.AgentVSCodeCopilot)) {
		t.Fatal("VS Code Copilot debe conservar su identidad cuando la carpeta no se comparte")
	}
	if !strings.Contains(cliLedger, string(model.AgentGitHubCopilotCLI)) {
		t.Fatal("Copilot CLI debe conservar su identidad cuando la carpeta no se comparte")
	}
	if _, err := os.Stat(filepath.Join(custom, "copilot-instructions.md")); err != nil {
		t.Fatalf("el prompt de Copilot CLI no sigue COPILOT_HOME: %v", err)
	}
}
