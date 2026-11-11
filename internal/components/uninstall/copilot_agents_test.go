package uninstall

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/sdd"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

func copilotManagedAgents(t *testing.T, dir string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "sdd-*.agent.md"))
	if err != nil {
		t.Fatal(err)
	}
	return paths
}

// Los agentes por fase de Copilot ya no se escriben en la carpeta de usuario, así que la
// desinstalación global de Copilot CLI y de VS Code Copilot no toca ~/.copilot/agents:
// lo que haya ahí es del usuario.
func TestUninstallCopilotGlobalLeavesTheUserAgentsFolderAlone(t *testing.T) {
	for _, id := range []model.AgentID{model.AgentGitHubCopilotCLI, model.AgentVSCodeCopilot} {
		t.Run(string(id), func(t *testing.T) {
			home := copilotHome(t)
			if err := state.Write(home, state.InstallState{InstalledAgents: []string{string(id)}}); err != nil {
				t.Fatal(err)
			}
			installCopilot(t, home, id)
			agentsDir := filepath.Join(home, ".copilot", "agents")
			if len(copilotManagedAgents(t, agentsDir)) != 0 {
				t.Fatalf("la instalación global no debe dejar agentes por fase en %s", agentsDir)
			}
			own := filepath.Join(agentsDir, "mi-agente.agent.md")
			if err := os.MkdirAll(agentsDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(own, []byte("---\nname: mi-agente\n---\npropio\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			svc := newCopilotService(t, home)
			if _, err := svc.PartialUninstall([]model.AgentID{id}, allManagedComponents); err != nil {
				t.Fatalf("PartialUninstall: %v", err)
			}
			if !exists(own) {
				t.Error("el agente del usuario debe sobrevivir a la desinstalación")
			}
		})
	}
}

func newCopilotWorkspaceService(t *testing.T, workspace string) *Service {
	t.Helper()
	svc, err := NewScopedService(t.TempDir(), workspace, "dev", Options{Scope: ScopeWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	svc.snapshotter = stubSnapshotter{}
	return svc
}

// Con solo Copilot CLI en el proyecto, el sync escribe .github/agents (sdd.InjectCopilotProjectAgents)
// y desinstalar Copilot CLI del proyecto los retira y deja los del equipo.
func TestWorkspaceUninstallCopilotCLIRemovesTheProjectAgents(t *testing.T) {
	copilotHome(t)
	workspace := t.TempDir()
	adapter := agents.BindWorkspace(mustAdapter(t, model.AgentVSCodeCopilot), workspace)
	if _, err := sdd.InjectCopilotProjectAgents(workspace, adapter, sdd.InjectOptions{}); err != nil {
		t.Fatalf("InjectCopilotProjectAgents: %v", err)
	}
	agentsDir := filepath.Join(workspace, ".github", "agents")
	team := filepath.Join(agentsDir, "equipo.agent.md")
	if err := os.WriteFile(team, []byte("---\nname: equipo\n---\ndel equipo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := state.Write(workspace, state.InstallState{InstalledAgents: []string{string(model.AgentGitHubCopilotCLI)}}); err != nil {
		t.Fatal(err)
	}

	if _, err := newCopilotWorkspaceService(t, workspace).PartialUninstall([]model.AgentID{model.AgentGitHubCopilotCLI}, allManagedComponents); err != nil {
		t.Fatalf("PartialUninstall: %v", err)
	}
	if left := copilotManagedAgents(t, agentsDir); len(left) != 0 {
		t.Errorf("quedan agentes de Axiom: %v", left)
	}
	if !exists(team) {
		t.Error("el agente del equipo debe sobrevivir")
	}
}

// VS Code Copilot y Copilot CLI comparten .github/agents en el proyecto: retirar uno no
// quita los agentes mientras el otro siga en el estado; retirar el último sí.
func TestWorkspaceUninstallOneCopilotAgentKeepsTheSharedProjectAgents(t *testing.T) {
	for _, tc := range []struct {
		name            string
		remove, remains model.AgentID
	}{
		{"se retira Copilot CLI y queda VS Code", model.AgentGitHubCopilotCLI, model.AgentVSCodeCopilot},
		{"se retira VS Code y queda Copilot CLI", model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copilotHome(t)
			workspace := t.TempDir()
			adapter := agents.BindWorkspace(mustAdapter(t, model.AgentVSCodeCopilot), workspace)
			if _, err := sdd.Inject(workspace, adapter, "", sdd.InjectOptions{WorkspaceDir: workspace}); err != nil {
				t.Fatalf("sdd.Inject: %v", err)
			}
			if err := state.Write(workspace, state.InstallState{InstalledAgents: []string{string(model.AgentVSCodeCopilot), string(model.AgentGitHubCopilotCLI)}}); err != nil {
				t.Fatal(err)
			}
			agentsDir := filepath.Join(workspace, ".github", "agents")
			installed := len(copilotManagedAgents(t, agentsDir))
			if installed == 0 {
				t.Fatal("la instalación previa no dejó los agentes compartidos")
			}

			if _, err := newCopilotWorkspaceService(t, workspace).PartialUninstall([]model.AgentID{tc.remove}, allManagedComponents); err != nil {
				t.Fatalf("PartialUninstall(%s): %v", tc.remove, err)
			}
			if got := len(copilotManagedAgents(t, agentsDir)); got != installed {
				t.Errorf("tras retirar %s quedan %d agentes, había %d", tc.remove, got, installed)
			}

			if _, err := newCopilotWorkspaceService(t, workspace).PartialUninstall([]model.AgentID{tc.remains}, allManagedComponents); err != nil {
				t.Fatalf("PartialUninstall(%s): %v", tc.remains, err)
			}
			if left := copilotManagedAgents(t, agentsDir); len(left) != 0 {
				t.Errorf("tras retirar el último agente quedan %v", left)
			}
		})
	}
}

// En el ámbito de proyecto, VS Code Copilot tiene sus agentes en .github/agents: la
// desinstalación retira los de Axiom y deja los del equipo, que son del repositorio.
func TestWorkspaceUninstallVSCodeRemovesManagedAgentsAndKeepsTeamAgents(t *testing.T) {
	copilotHome(t)
	workspace := t.TempDir()
	adapter := agents.BindWorkspace(mustAdapter(t, model.AgentVSCodeCopilot), workspace)
	if _, err := sdd.Inject(workspace, adapter, "", sdd.InjectOptions{WorkspaceDir: workspace}); err != nil {
		t.Fatalf("sdd.Inject: %v", err)
	}
	agentsDir := filepath.Join(workspace, ".github", "agents")
	if len(copilotManagedAgents(t, agentsDir)) == 0 {
		t.Fatalf("la instalación de proyecto no dejó agentes en %s", agentsDir)
	}
	team := filepath.Join(agentsDir, "equipo.agent.md")
	if err := os.WriteFile(team, []byte("---\nname: equipo\n---\ndel equipo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := state.Write(workspace, state.InstallState{InstalledAgents: []string{string(model.AgentVSCodeCopilot)}}); err != nil {
		t.Fatal(err)
	}

	svc, err := NewScopedService(t.TempDir(), workspace, "dev", Options{Scope: ScopeWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	svc.snapshotter = stubSnapshotter{}
	if _, err := svc.PartialUninstall([]model.AgentID{model.AgentVSCodeCopilot}, allManagedComponents); err != nil {
		t.Fatalf("PartialUninstall: %v", err)
	}
	if left := copilotManagedAgents(t, agentsDir); len(left) != 0 {
		t.Errorf("quedan agentes de Axiom: %v", left)
	}
	if !exists(team) {
		t.Error("el agente del equipo debe sobrevivir")
	}
}

func mustAdapter(t *testing.T, id model.AgentID) agents.Adapter {
	t.Helper()
	adapter, err := agents.NewAdapter(id)
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}
