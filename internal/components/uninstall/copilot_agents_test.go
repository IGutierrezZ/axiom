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

// Desinstalar Copilot CLI retira los `.agent.md` que Axiom escribió y conserva los
// que son del usuario, y con ellos la carpeta.
func TestUninstallCopilotCLIRemovesManagedAgentsAndKeepsUserAgents(t *testing.T) {
	home := copilotHome(t)
	if err := state.Write(home, state.InstallState{InstalledAgents: []string{string(model.AgentGitHubCopilotCLI)}}); err != nil {
		t.Fatal(err)
	}
	cli := installCopilot(t, home, model.AgentGitHubCopilotCLI)
	agentsDir := cli.SubAgentsDir(home)
	if len(copilotManagedAgents(t, agentsDir)) == 0 {
		t.Fatalf("la instalación previa no dejó agentes en %s", agentsDir)
	}
	own := filepath.Join(agentsDir, "mi-agente.agent.md")
	if err := os.WriteFile(own, []byte("---\nname: mi-agente\n---\npropio\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := newCopilotService(t, home)
	if _, err := svc.PartialUninstall([]model.AgentID{model.AgentGitHubCopilotCLI}, allManagedComponents); err != nil {
		t.Fatalf("PartialUninstall: %v", err)
	}

	if left := copilotManagedAgents(t, agentsDir); len(left) != 0 {
		t.Errorf("quedan agentes de Axiom: %v", left)
	}
	if exists(filepath.Join(agentsDir, "jd-judge-a.agent.md")) {
		t.Error("jd-judge-a.agent.md sigue en pie")
	}
	if !exists(own) {
		t.Error("el agente del usuario debe sobrevivir a la desinstalación")
	}
}

// Sin agentes del usuario, desinstalar el último agente de Copilot deja la carpeta
// de agentes vacía y la retira.
func TestUninstallCopilotCLILeavesNoEmptyAgentsDir(t *testing.T) {
	home := copilotHome(t)
	if err := state.Write(home, state.InstallState{InstalledAgents: []string{string(model.AgentGitHubCopilotCLI)}}); err != nil {
		t.Fatal(err)
	}
	cli := installCopilot(t, home, model.AgentGitHubCopilotCLI)

	svc := newCopilotService(t, home)
	if _, err := svc.PartialUninstall([]model.AgentID{model.AgentGitHubCopilotCLI}, allManagedComponents); err != nil {
		t.Fatalf("PartialUninstall: %v", err)
	}
	if exists(cli.SubAgentsDir(home)) {
		t.Errorf("la carpeta %s sigue en pie", cli.SubAgentsDir(home))
	}
}

// Los agentes de ~/.copilot/agents son de los dos agentes. Desinstalar uno no retira
// los del otro mientras siga instalado; desinstalar el último sí los retira.
func TestUninstallOneCopilotAgentKeepsTheSharedAgentsOfTheOther(t *testing.T) {
	for _, tc := range []struct {
		name            string
		remove, remains model.AgentID
	}{
		{"se retira Copilot CLI y queda VS Code", model.AgentGitHubCopilotCLI, model.AgentVSCodeCopilot},
		{"se retira VS Code y queda Copilot CLI", model.AgentVSCodeCopilot, model.AgentGitHubCopilotCLI},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := copilotHome(t)
			if err := state.Write(home, state.InstallState{InstalledAgents: []string{string(model.AgentVSCodeCopilot), string(model.AgentGitHubCopilotCLI)}}); err != nil {
				t.Fatal(err)
			}
			installCopilot(t, home, model.AgentVSCodeCopilot)
			installCopilot(t, home, model.AgentGitHubCopilotCLI)

			remaining, err := agents.NewAdapter(tc.remains)
			if err != nil {
				t.Fatal(err)
			}
			removing, err := agents.NewAdapter(tc.remove)
			if err != nil {
				t.Fatal(err)
			}
			if remaining.SubAgentsDir(home) != removing.SubAgentsDir(home) {
				t.Fatalf("las carpetas de agentes deberían coincidir: %q / %q", remaining.SubAgentsDir(home), removing.SubAgentsDir(home))
			}
			agentsDir := remaining.SubAgentsDir(home)
			installed := len(copilotManagedAgents(t, agentsDir))
			if installed == 0 {
				t.Fatal("la instalación previa no dejó los agentes compartidos")
			}

			svc := newCopilotService(t, home)
			if _, err := svc.PartialUninstall([]model.AgentID{tc.remove}, allManagedComponents); err != nil {
				t.Fatalf("PartialUninstall(%s): %v", tc.remove, err)
			}
			if got := len(copilotManagedAgents(t, agentsDir)); got != installed {
				t.Errorf("tras retirar %s quedan %d agentes, había %d", tc.remove, got, installed)
			}

			svc = newCopilotService(t, home)
			if _, err := svc.PartialUninstall([]model.AgentID{tc.remains}, allManagedComponents); err != nil {
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
