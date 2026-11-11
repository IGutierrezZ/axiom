package uninstall

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/components/engram"
	"github.com/IGutierrezZ/axiom/v3/internal/components/mcp"
	"github.com/IGutierrezZ/axiom/v3/internal/components/persona"
	"github.com/IGutierrezZ/axiom/v3/internal/components/sdd"
	"github.com/IGutierrezZ/axiom/v3/internal/components/skills"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/state"
)

// copilotHome aísla un home que es además el home real del proceso: COPILOT_HOME solo
// se respeta en ese caso. Confina también las rutas de VS Code.
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

// installCopilot instala en home lo que Axiom escribe para el agente: persona, SDD
// (prompt y skills), skills generales, Engram y Context7.
func installCopilot(t *testing.T, home string, id model.AgentID) agents.Adapter {
	t.Helper()
	adapter, err := agents.NewAdapter(id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := persona.Inject(home, adapter, model.PersonaNeutral); err != nil {
		t.Fatalf("persona.Inject(%s): %v", id, err)
	}
	if _, err := sdd.Inject(home, adapter, ""); err != nil {
		t.Fatalf("sdd.Inject(%s): %v", id, err)
	}
	if _, err := skills.Inject(home, adapter, []model.SkillID{model.SkillSDDInit}); err != nil {
		t.Fatalf("skills.Inject(%s): %v", id, err)
	}
	if _, err := engram.Inject(home, adapter); err != nil {
		t.Fatalf("engram.Inject(%s): %v", id, err)
	}
	if _, err := mcp.Inject(home, home, adapter); err != nil {
		t.Fatalf("mcp.Inject(%s): %v", id, err)
	}
	return adapter
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func newCopilotService(t *testing.T, home string) *Service {
	t.Helper()
	svc, err := NewService(home, t.TempDir(), "dev")
	if err != nil {
		t.Fatal(err)
	}
	svc.snapshotter = stubSnapshotter{}
	return svc
}

// Desinstalar Copilot CLI retira todo lo que Axiom le escribió: el prompt, el MCP de
// Engram y de Context7 y las skills. Lo que es del usuario se conserva.
func TestUninstallCopilotCLIRemovesWhatAxiomInstalled(t *testing.T) {
	home := copilotHome(t)
	if err := state.Write(home, state.InstallState{InstalledAgents: []string{string(model.AgentGitHubCopilotCLI)}}); err != nil {
		t.Fatal(err)
	}
	cli := installCopilot(t, home, model.AgentGitHubCopilotCLI)

	prompt := cli.SystemPromptFile(home)
	mcpPath := cli.MCPConfigPath(home, "engram")
	if !exists(prompt) || !exists(mcpPath) || !exists(filepath.Join(cli.SkillsDir(home), "sdd-init", "SKILL.md")) {
		t.Fatal("la instalación previa no dejó los ficheros esperados")
	}

	// Un servidor del usuario junto a los de Axiom.
	merged, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(merged), "engram") || !strings.Contains(string(merged), "context7") {
		t.Fatalf("mcp-config.json sin las entradas de Axiom:\n%s", merged)
	}
	if err := os.WriteFile(mcpPath, []byte(strings.Replace(string(merged), `"mcpServers": {`, `"mcpServers": {"mine": {"type":"http","url":"https://example.test/mcp","tools":["a"]},`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := newCopilotService(t, home)
	if _, err := svc.PartialUninstall([]model.AgentID{model.AgentGitHubCopilotCLI}, allManagedComponents); err != nil {
		t.Fatalf("PartialUninstall: %v", err)
	}

	if exists(prompt) {
		got, _ := os.ReadFile(prompt)
		t.Errorf("copilot-instructions.md sigue en pie con:\n%s", got)
	}
	if exists(cli.SkillsDir(home)) {
		t.Errorf("la carpeta de skills %s sigue en pie", cli.SkillsDir(home))
	}
	raw, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("mcp-config.json desapareció con el servidor del usuario dentro: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, "engram") || strings.Contains(text, "context7") {
		t.Errorf("mcp-config.json conserva entradas de Axiom:\n%s", text)
	}
	if !strings.Contains(text, "example.test") {
		t.Errorf("mcp-config.json perdió el servidor del usuario:\n%s", text)
	}
	got, err := state.Read(home)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(got.InstalledAgents, string(model.AgentGitHubCopilotCLI)) {
		t.Fatalf("el estado aún lista Copilot CLI: %v", got.InstalledAgents)
	}
}

// El contenido propio del usuario en copilot-instructions.md se conserva.
func TestUninstallCopilotCLIKeepsUserInstructions(t *testing.T) {
	home := copilotHome(t)
	if err := state.Write(home, state.InstallState{InstalledAgents: []string{string(model.AgentGitHubCopilotCLI)}}); err != nil {
		t.Fatal(err)
	}
	cli := installCopilot(t, home, model.AgentGitHubCopilotCLI)
	prompt := cli.SystemPromptFile(home)
	existing, err := os.ReadFile(prompt)
	if err != nil {
		t.Fatal(err)
	}
	const mine = "\n\n# Mis reglas\n\nResponde siempre en castellano.\n"
	if err := os.WriteFile(prompt, append(existing, []byte(mine)...), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := newCopilotService(t, home)
	if _, err := svc.PartialUninstall([]model.AgentID{model.AgentGitHubCopilotCLI}, allManagedComponents); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(prompt)
	if err != nil {
		t.Fatalf("copilot-instructions.md desapareció con contenido del usuario: %v", err)
	}
	if !strings.Contains(string(got), "Responde siempre en castellano.") {
		t.Fatalf("se perdió el contenido del usuario:\n%s", got)
	}
	if strings.Contains(string(got), "Spec-Driven Development") {
		t.Fatalf("el orquestador de Axiom sigue en el prompt:\n%s", got)
	}
}

// Las skills de ~/.copilot/skills son de los dos agentes. Desinstalar uno no retira
// las del otro mientras siga instalado; desinstalar el último sí las retira.
func TestUninstallOneCopilotAgentKeepsTheSharedSkillsOfTheOther(t *testing.T) {
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
			if remaining.SkillsDir(home) != removing.SkillsDir(home) {
				t.Fatalf("las carpetas de skills deberían coincidir: %q / %q", remaining.SkillsDir(home), removing.SkillsDir(home))
			}
			skillsDir := remaining.SkillsDir(home)
			sddSkill := filepath.Join(skillsDir, "sdd-init", "SKILL.md")
			shared := filepath.Join(skillsDir, "_shared", "engram-convention.md")
			if !exists(sddSkill) || !exists(shared) {
				t.Fatal("la instalación previa no dejó las skills compartidas")
			}

			svc := newCopilotService(t, home)
			if _, err := svc.PartialUninstall([]model.AgentID{tc.remove}, allManagedComponents); err != nil {
				t.Fatalf("PartialUninstall(%s): %v", tc.remove, err)
			}
			if !exists(sddSkill) || !exists(shared) {
				t.Fatalf("desinstalar %s borró las skills que %s sigue usando", tc.remove, tc.remains)
			}
			if exists(removing.SystemPromptFile(home)) {
				t.Errorf("el prompt de %s sigue instalado", tc.remove)
			}
			if !exists(remaining.SystemPromptFile(home)) {
				t.Errorf("desinstalar %s borró el prompt de %s", tc.remove, tc.remains)
			}

			// Al retirar el último agente, las skills compartidas ya no las usa nadie.
			svc = newCopilotService(t, home)
			if _, err := svc.PartialUninstall([]model.AgentID{tc.remains}, allManagedComponents); err != nil {
				t.Fatalf("PartialUninstall(%s): %v", tc.remains, err)
			}
			if exists(sddSkill) || exists(shared) {
				t.Fatalf("al retirar el último agente quedaron skills compartidas en %s", skillsDir)
			}
		})
	}
}

// Con COPILOT_HOME las carpetas de skills son distintas: cada agente retira las suyas.
func TestUninstallCopilotCLIWithCopilotHomeRemovesOnlyItsOwnSkills(t *testing.T) {
	home := copilotHome(t)
	custom := filepath.Join(t.TempDir(), "custom-copilot")
	t.Setenv("COPILOT_HOME", custom)
	if err := state.Write(home, state.InstallState{InstalledAgents: []string{string(model.AgentVSCodeCopilot), string(model.AgentGitHubCopilotCLI)}}); err != nil {
		t.Fatal(err)
	}
	installCopilot(t, home, model.AgentVSCodeCopilot)
	cli := installCopilot(t, home, model.AgentGitHubCopilotCLI)
	vscodeSkill := filepath.Join(home, ".copilot", "skills", "sdd-init", "SKILL.md")
	cliSkill := filepath.Join(custom, "skills", "sdd-init", "SKILL.md")
	if !exists(vscodeSkill) || !exists(cliSkill) {
		t.Fatal("la instalación previa no dejó las skills de cada agente")
	}

	svc := newCopilotService(t, home)
	if _, err := svc.PartialUninstall([]model.AgentID{model.AgentGitHubCopilotCLI}, allManagedComponents); err != nil {
		t.Fatal(err)
	}
	if exists(cliSkill) || exists(cli.SystemPromptFile(home)) {
		t.Error("Copilot CLI no retiró sus propios ficheros en COPILOT_HOME")
	}
	if !exists(vscodeSkill) {
		t.Error("retirar Copilot CLI borró las skills de VS Code Copilot")
	}
}
