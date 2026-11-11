package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// copilotProjectState es el estado de un proyecto con SDD y una asignación de Copilot.
func copilotProjectState(agents ...string) state.InstallState {
	return state.InstallState{
		InstalledAgents:     agents,
		SelectionConfigured: true,
		Components:          []model.ComponentID{model.ComponentPersona, model.ComponentSDD, model.ComponentEngram},
		Persona:             "neutral",
		CopilotPhaseAssignments: map[string]state.CopilotPhaseAssignmentState{
			"sdd-apply": {Model: "Claude Sonnet 4.6 (copilot)", Effort: "high"},
		},
	}
}

func assertCopilotProjectAgents(t *testing.T, root string) string {
	t.Helper()
	agentsDir := filepath.Join(root, ".github", "agents")
	apply, err := os.ReadFile(filepath.Join(agentsDir, "sdd-apply.agent.md"))
	if err != nil {
		t.Fatalf("el sync de proyecto no escribió .github/agents/sdd-apply.agent.md: %v", err)
	}
	for _, line := range []string{"model: Claude Sonnet 4.6 (copilot)\n", "reasoning-effort: high\n"} {
		if !strings.Contains(string(apply), line) {
			t.Errorf("sdd-apply.agent.md no contiene %q:\n%s", line, apply)
		}
	}
	design, err := os.ReadFile(filepath.Join(agentsDir, "sdd-design.agent.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(design), "model:") || strings.Contains(string(design), "reasoning-effort:") {
		t.Errorf("sdd-design no tiene asignación y no debe llevar modelo:\n%s", design)
	}
	return string(apply)
}

// axiom sync global con Copilot CLI y VS Code Copilot ya no escribe los agentes por fase en
// ~/.copilot/agents, y retira los que dejó una versión anterior sin tocar los del usuario.
func TestRunSync_GlobalCopilotWritesNoPhaseAgentsAndRetiresLegacyOnes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("COPILOT_HOME", "")
	if err := state.Write(home, state.InstallState{InstalledAgents: []string{"vscode-copilot", "github-copilot-cli"}}); err != nil {
		t.Fatalf("state.Write() error = %v", err)
	}
	setSyncTestHome(t, home)
	agentsDir := filepath.Join(home, ".copilot", "agents")
	mustWriteFile(t, filepath.Join(agentsDir, "sdd-apply.agent.md"), []byte("---\nname: sdd-apply\n---\nde una versión anterior\n"))
	mustWriteFile(t, filepath.Join(agentsDir, "jd-judge-a.agent.md"), []byte("---\nname: jd-judge-a\n---\nde una versión anterior\n"))
	const own = "---\nname: mi-agente\n---\npropio\n"
	mustWriteFile(t, filepath.Join(agentsDir, "mi-agente.agent.md"), []byte(own))

	for i := 0; i < 2; i++ { // la segunda ejecución es idempotente
		if _, err := RunSync([]string{"--agents", "vscode-copilot,github-copilot-cli"}); err != nil {
			t.Fatalf("RunSync() #%d error = %v", i+1, err)
		}
	}

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if !slices.Equal(names, []string{"mi-agente.agent.md"}) {
		t.Errorf("en ~/.copilot/agents solo debe quedar el agente del usuario: %v", names)
	}
	if data, err := os.ReadFile(filepath.Join(agentsDir, "mi-agente.agent.md")); err != nil || string(data) != own {
		t.Errorf("el agente del usuario cambió: %v %q", err, data)
	}
}

// Con solo Copilot CLI (sin modo workspace), el sync del proyecto escribe igualmente
// .github/agents con las asignaciones del estado del proyecto, y no toca la carpeta de usuario.
func TestRunSyncWorkspace_CopilotCLIOnlyWritesProjectAgents(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	captureWorkspaceAgentsNotice(t)
	if err := state.ProjectStore(home, root).Write(copilotProjectState("github-copilot-cli")); err != nil {
		t.Fatal(err)
	}

	result, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	if result.NoOp || len(result.ChangedFiles) == 0 {
		t.Errorf("el primer sync debe informar de los agentes escritos: noop=%v ficheros=%v", result.NoOp, result.ChangedFiles)
	}
	first := assertCopilotProjectAgents(t, root)
	if _, err := os.Stat(filepath.Join(home, ".copilot", "agents")); !os.IsNotExist(err) {
		t.Errorf("no se debe escribir en ~/.copilot/agents: %v", err)
	}

	// Una segunda sincronización deja los mismos bytes y no escribe nada.
	again, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("segunda RunSync() error = %v", err)
	}
	if !again.NoOp {
		t.Errorf("la segunda sincronización no debe reescribir nada: %v", again.ChangedFiles)
	}
	if second := assertCopilotProjectAgents(t, root); second != first {
		t.Error("la segunda sincronización cambió sdd-apply.agent.md")
	}
}

// Con VS Code Copilot y Copilot CLI a la vez, el adaptador de VS Code es el único que
// escribe .github/agents: el resultado es el mismo y no se duplica nada.
func TestRunSyncWorkspace_VSCodeAndCopilotCLIWriteProjectAgentsOnce(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	captureWorkspaceAgentsNotice(t)
	if err := state.ProjectStore(home, root).Write(copilotProjectState("vscode-copilot", "github-copilot-cli")); err != nil {
		t.Fatal(err)
	}

	result, err := RunSync([]string{"--scope", "workspace"})
	if err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	if want := []model.AgentID{model.AgentVSCodeCopilot}; !slices.Equal(result.Agents, want) {
		t.Errorf("agentes sincronizados = %v, want %v", result.Agents, want)
	}
	assertCopilotProjectAgents(t, root)
	entries, err := os.ReadDir(filepath.Join(root, ".github", "agents"))
	if err != nil || len(entries) != len(model.CopilotPhaseNames()) {
		t.Errorf("debe haber un agente por fase: %d, %v", len(entries), err)
	}
	if _, err := os.Stat(filepath.Join(home, ".copilot", "agents")); !os.IsNotExist(err) {
		t.Errorf("no se debe escribir en ~/.copilot/agents: %v", err)
	}
}

// Un proyecto con otro adaptador (aquí Claude Code) también recibe .github/agents cuando el
// usuario tiene Copilot CLI configurado: lo lee de ahí.
func TestRunSyncWorkspace_CopilotCLIConfiguredForTheUserFeedsTheProject(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	captureWorkspaceAgentsNotice(t)
	t.Setenv("COPILOT_HOME", "")
	mustWriteFile(t, filepath.Join(home, ".copilot", "config.json"), []byte("{}\n"))
	if err := state.ProjectStore(home, root).Write(copilotProjectState("claude-code")); err != nil {
		t.Fatal(err)
	}

	if _, err := RunSync([]string{"--scope", "workspace"}); err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	assertCopilotProjectAgents(t, root)
}

// Sin Copilot en el proyecto ni en la configuración del usuario no aparece .github/agents.
func TestRunSyncWorkspace_WithoutCopilotWritesNoProjectAgents(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	captureWorkspaceAgentsNotice(t)
	t.Setenv("COPILOT_HOME", "")
	if err := state.ProjectStore(home, root).Write(copilotProjectState("claude-code")); err != nil {
		t.Fatal(err)
	}

	if _, err := RunSync([]string{"--scope", "workspace"}); err != nil {
		t.Fatalf("RunSync() error = %v", err)
	}
	assertNothingUnder(t, root, ".github")
}

// La instalación de proyecto sigue la misma regla que el sync: con Copilot CLI configurado
// para el usuario, el proyecto recibe .github/agents aunque el agente elegido sea otro.
func TestRunInstallWorkspace_CopilotCLIConfiguredForTheUserFeedsTheProject(t *testing.T) {
	root, _, home := workspaceAgentsFixture(t)
	t.Chdir(root)
	t.Setenv("COPILOT_HOME", "")
	mustWriteFile(t, filepath.Join(home, ".copilot", "config.json"), []byte("{}\n"))

	if _, err := RunInstall([]string{"--scope", "workspace", "--agent", "claude-code", "--preset", "custom", "--component", "sdd"}, macOSDetectionResult()); err != nil {
		t.Fatalf("RunInstall() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".github", "agents", "sdd-apply.agent.md")); err != nil {
		t.Errorf("la instalación de proyecto no escribió .github/agents: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".copilot", "agents")); !os.IsNotExist(err) {
		t.Errorf("no se debe escribir en ~/.copilot/agents: %v", err)
	}
}
