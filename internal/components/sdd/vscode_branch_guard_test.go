package sdd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/vscode"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// El fichero de hooks de VS Code Copilot sigue el formato del arnés de Copilot: versión 1,
// `preToolUse` y `postToolUse` con la guarda, el matcher de las herramientas de escritura y
// una orden que termina siempre con 0 en bash y en PowerShell, porque en `preToolUse`
// cualquier otro código deniega la escritura.
func TestVSCodeBranchGuardHooksJSONFollowsTheCopilotHarnessFormat(t *testing.T) {
	var document struct {
		Version int                           `json:"version"`
		Hooks   map[string][]copilotHookEntry `json:"hooks"`
	}
	if err := json.Unmarshal(VSCodeBranchGuardHooksJSON(model.AgentVSCodeCopilot), &document); err != nil {
		t.Fatalf("el fichero no es JSON válido: %v", err)
	}
	if document.Version != 1 || len(document.Hooks) != 2 {
		t.Fatalf("documento = %+v; se esperaba version 1 con preToolUse y postToolUse", document)
	}
	for event, flag := range map[string]string{"preToolUse": "pre", "postToolUse": "post"} {
		entries := document.Hooks[event]
		if len(entries) != 1 {
			t.Fatalf("%s: %d entradas, se esperaba 1", event, len(entries))
		}
		want := copilotHookEntry{
			Type:       "command",
			Bash:       "axiom branch guard --agent vscode-copilot --event " + flag + " || true",
			PowerShell: "axiom branch guard --agent vscode-copilot --event " + flag + "; exit 0",
			Matcher:    "create|edit|str_replace_editor|apply_patch",
			TimeoutSec: 10,
		}
		if entries[0] != want {
			t.Errorf("%s = %+v, se esperaba %+v", event, entries[0], want)
		}
	}
	if !bytes.Equal(VSCodeBranchGuardHooksJSON(model.AgentVSCodeCopilot), VSCodeBranchGuardHooksJSON(model.AgentVSCodeCopilot)) {
		t.Error("el contenido debe ser determinista")
	}
}

// En el proyecto se instala .github/hooks/axiom-branch-guard.json; repetirlo no cambia nada y
// no toca los demás ficheros de .github/hooks. La instalación global no instala hooks.
func TestInstallSkillRegistryAutomationWritesTheVSCodeBranchGuardHooksFile(t *testing.T) {
	root := t.TempDir()
	teamHook := filepath.Join(root, ".github", "hooks", "equipo.json")
	if err := os.MkdirAll(filepath.Dir(teamHook), 0o755); err != nil {
		t.Fatal(err)
	}
	teamContent := []byte(`{"version":1,"hooks":{"sessionStart":[{"type":"command","command":"echo hola"}]}}`)
	if err := os.WriteFile(teamHook, teamContent, 0o644); err != nil {
		t.Fatal(err)
	}
	adapter := vscode.NewAdapter().WithWorkspace(root)
	hooksPath := filepath.Join(root, ".github", "hooks", "axiom-branch-guard.json")

	first, err := installSkillRegistryAutomation(root, adapter)
	if err != nil {
		t.Fatalf("installSkillRegistryAutomation() error = %v", err)
	}
	if !first.Changed || len(first.Files) != 1 || first.Files[0] != hooksPath {
		t.Errorf("primera instalación = %+v; se esperaba un cambio en %s", first, hooksPath)
	}
	if got, _ := os.ReadFile(hooksPath); !bytes.Equal(got, VSCodeBranchGuardHooksJSON(model.AgentVSCodeCopilot)) {
		t.Errorf("contenido = %s", got)
	}
	second, err := installSkillRegistryAutomation(root, adapter)
	if err != nil || second.Changed {
		t.Errorf("segunda instalación = %+v, %v; no debe cambiar nada", second, err)
	}
	if got, _ := os.ReadFile(teamHook); !bytes.Equal(got, teamContent) {
		t.Errorf("el hook del equipo cambió: %s", got)
	}

	home := t.TempDir()
	if result, err := installSkillRegistryAutomation(home, vscode.NewAdapter()); err != nil || result.Changed || len(result.Files) != 0 {
		t.Errorf("instalación global = %+v, %v; no debe instalar hooks", result, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".github")); !os.IsNotExist(err) {
		t.Errorf("la instalación global no debe crear .github: %v", err)
	}
}
