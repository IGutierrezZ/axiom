package cli

import (
	"slices"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// El doctor comprueba el binario `copilot` de Copilot CLI y no el `code` de VS Code.
func TestDoctorChecksTheCopilotBinaryForCopilotCLI(t *testing.T) {
	got := requiredDoctorTools([]string{string(model.AgentGitHubCopilotCLI)})
	if !slices.Contains(got, "copilot") {
		t.Fatalf("requiredDoctorTools = %v, want it to include copilot", got)
	}
	if slices.Contains(got, "code") {
		t.Fatalf("requiredDoctorTools = %v: Copilot CLI no necesita `code`", got)
	}

	vscode := requiredDoctorTools([]string{string(model.AgentVSCodeCopilot)})
	if slices.Contains(vscode, "copilot") || !slices.Contains(vscode, "code") {
		t.Fatalf("requiredDoctorTools(vscode) = %v, want code y no copilot", vscode)
	}
}

// La detección elige Copilot CLI y VS Code Copilot por separado: cada uno aparece
// solo cuando su propia señal existe.
func TestDefaultAgentsFromDetectionKeepsCopilotCLIAndVSCodeApart(t *testing.T) {
	cli := defaultAgentsFromDetection(makeDetectionWithAgents("github-copilot-cli"))
	if !slices.Equal(cli, []model.AgentID{model.AgentGitHubCopilotCLI}) {
		t.Fatalf("defaultAgentsFromDetection(cli) = %v", cli)
	}
	vscode := defaultAgentsFromDetection(makeDetectionWithAgents("vscode-copilot"))
	if !slices.Equal(vscode, []model.AgentID{model.AgentVSCodeCopilot}) {
		t.Fatalf("defaultAgentsFromDetection(vscode) = %v", vscode)
	}
	both := defaultAgentsFromDetection(makeDetectionWithAgents("vscode-copilot", "github-copilot-cli"))
	if len(both) != 2 {
		t.Fatalf("defaultAgentsFromDetection(both) = %v, want los dos", both)
	}
}

// Copilot CLI es solo global: no hay una configuración de proyecto de Copilot CLI que
// Axiom pueda respaldar, y aplicarle las rutas de VS Code (.github, .vscode) o
// <proyecto>/.copilot escribiría ficheros que el CLI no lee. En workspace se omite
// entero, con el aviso común, y VS Code Copilot sigue teniendo su modo workspace.
func TestSplitAgentsForScopeOmitsCopilotCLIInWorkspace(t *testing.T) {
	selected := []model.AgentID{model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI, model.AgentCodex}

	kept, omitted := splitAgentsForScope(ScopeWorkspace, selected)
	if want := []model.AgentID{model.AgentVSCodeCopilot, model.AgentCodex}; !slices.Equal(kept, want) {
		t.Errorf("workspace conserva %v, want %v", kept, want)
	}
	if want := []model.AgentID{model.AgentGitHubCopilotCLI}; !slices.Equal(omitted, want) {
		t.Errorf("workspace omite %v, want %v", omitted, want)
	}
	if agentSupportsWorkspace(model.AgentGitHubCopilotCLI) {
		t.Error("Copilot CLI no debe declarar modo workspace")
	}

	kept, omitted = splitAgentsForScope(ScopeGlobal, selected)
	if !slices.Equal(kept, selected) || omitted != nil {
		t.Errorf("global no filtra: kept=%v omitted=%v", kept, omitted)
	}
}
