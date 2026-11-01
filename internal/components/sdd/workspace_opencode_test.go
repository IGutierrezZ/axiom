package sdd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
)

// En workspace, el marcador de skills de los overlays de OpenCode y Kilo se
// resuelve a la carpeta de skills compartida del proyecto (.agents/skills), que
// leen los dos agentes y no la de cada uno (.opencode/skills, .kilo/skills),
// no a la carpeta global bajo .config.
func TestResolveOverlaySkillsDirUsesProjectSkillsOfBoundAdapter(t *testing.T) {
	const overlay = `{"agent":{"sdd-apply":{"prompt":"Read {{AXIOM_SKILLS_DIR}}/sdd-apply/SKILL.md"}}}`
	root := t.TempDir()

	for _, tc := range []struct {
		agent model.AgentID
		want  string
	}{
		{model.AgentOpenCode, ".agents/skills/sdd-apply/SKILL.md"},
		{model.AgentKilocode, ".agents/skills/sdd-apply/SKILL.md"},
	} {
		adapter, err := agents.NewAdapter(tc.agent)
		if err != nil {
			t.Fatal(err)
		}
		resolved, err := resolveOverlaySkillsDir(overlay, agents.BindWorkspace(adapter, root), root)
		if err != nil {
			t.Fatalf("%s: %v", tc.agent, err)
		}
		if !strings.Contains(resolved, tc.want) || strings.Contains(resolved, ".config") {
			t.Errorf("%s: overlay resuelto inesperado: %s", tc.agent, resolved)
		}
		// En global sigue siendo la ruta de usuario.
		global, err := resolveOverlaySkillsDir(overlay, adapter, "")
		if err != nil || !strings.Contains(global, "~/.config/") {
			t.Errorf("%s global: %s (%v)", tc.agent, global, err)
		}
	}
}

// En workspace los prompts compartidos del modo multi van a la carpeta de
// recursos del proyecto y los agentes los referencian relativos a la configuración
// del proyecto; en global siguen en .config/opencode/prompts/sdd.
func TestSharedPromptDirForBoundAdapter(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	root := t.TempDir()

	opencodeAdapter, _ := agents.NewAdapter(model.AgentOpenCode)
	bound := agents.BindWorkspace(opencodeAdapter, root)
	want := filepath.Join(root, ".opencode", "prompts", "sdd")
	if got := SharedPromptDirFor(bound, root); got != want {
		t.Errorf("SharedPromptDirFor(workspace) = %q, quiero %q", got, want)
	}
	if got, want := SharedPromptDirFor(bound, "/home/usuario"), SharedPromptDir("/home/usuario"); got != want {
		t.Errorf("SharedPromptDirFor(otra carpeta) = %q, quiero la global %q", got, want)
	}
	if got, want := SharedPromptDirFor(opencodeAdapter, root), SharedPromptDir(root); got != want {
		t.Errorf("SharedPromptDirFor(sin ligar) = %q, quiero %q", got, want)
	}

	ref, err := SharedPromptFileRef(bound.SettingsPath(root), want, "sdd-apply")
	if err != nil || ref != "{file:./.opencode/prompts/sdd/sdd-apply.md}" {
		t.Errorf("SharedPromptFileRef = %q, %v", ref, err)
	}
}

// Instalar SDD en multi-modo en workspace escribe los prompts compartidos en el
// proyecto y las referencias de los subagentes apuntan a ellos.
func TestInjectWorkspaceMultiWritesSharedPromptsInProject(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("AXIOM_STATE_DIR", t.TempDir())

	for _, tc := range []struct {
		agent     model.AgentID
		promptDir string
		ref       string
	}{
		{model.AgentOpenCode, filepath.Join(".opencode", "prompts", "sdd"), "{file:./.opencode/prompts/sdd/sdd-apply.md}"},
		{model.AgentKilocode, filepath.Join(".kilo", "prompts", "sdd"), "{file:./prompts/sdd/sdd-apply.md}"},
	} {
		t.Run(string(tc.agent), func(t *testing.T) {
			root := t.TempDir()
			base, err := agents.NewAdapter(tc.agent)
			if err != nil {
				t.Fatal(err)
			}
			adapter := agents.BindWorkspace(base, root)

			if _, err := Inject(root, adapter, model.SDDModeMulti, InjectOptions{WorkspaceDir: root}); err != nil {
				t.Fatalf("Inject: %v", err)
			}

			if _, err := os.Stat(filepath.Join(root, tc.promptDir, "sdd-apply.md")); err != nil {
				t.Errorf("falta el prompt compartido en el proyecto: %v", err)
			}
			if _, err := os.Stat(filepath.Join(root, ".config")); err == nil {
				t.Errorf("no debe crearse .config en el proyecto")
			}
			prompt := agentPrompt(t, readOpenCodeAgents(t, adapter.SettingsPath(root)), "sdd-apply")
			if prompt != tc.ref {
				t.Errorf("sdd-apply prompt = %q, quiero %q", prompt, tc.ref)
			}
		})
	}
}
