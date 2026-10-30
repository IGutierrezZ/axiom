package sdd

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/vscode"
)

// En un proyecto, el orquestador de VS Code sitúa los ficheros de convención en la
// carpeta de skills compartida del proyecto (.agents/skills) y no en .agent/skills; con la instalación global, y
// con otro agente, el texto no cambia.
func TestAdaptVSCodeOrchestratorForWorkspace(t *testing.T) {
	root := t.TempDir()
	adapter := vscode.NewAdapter()
	prompt, err := renderSessionPreflightPrompt(adapter, InjectOptions{})
	if err != nil {
		t.Fatalf("renderSessionPreflightPrompt() error = %v", err)
	}
	if !strings.Contains(prompt, vscodeWorkspaceConventionsClause) {
		t.Fatalf("el orquestador genérico ya no contiene la frase que se adapta: %q", vscodeWorkspaceConventionsClause)
	}

	bound := adapter.WithWorkspace(root)
	got := adaptVSCodeOrchestratorForWorkspace(prompt, bound, root)
	if strings.Contains(got, ".agent/skills") || !strings.Contains(got, "Convention files under the project skills directory `.agents/skills/_shared/`:") {
		t.Errorf("la ruta de convenciones no apunta a .agents/skills/_shared")
	}

	if same := adaptVSCodeOrchestratorForWorkspace(prompt, bound, ""); same != prompt {
		t.Errorf("con la instalación global el texto no debe cambiar")
	}
	if same := adaptVSCodeOrchestratorForWorkspace(prompt, claude.NewAdapter(), root); same != prompt {
		t.Errorf("otro agente no debe cambiar")
	}
}
