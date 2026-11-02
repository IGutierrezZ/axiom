package sdd

import (
	"strings"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/agents/claude"
	"github.com/IGutierrezZ/axiom/v3/internal/agents/codex"
)

// renderCodexOrchestrator devuelve el orquestador de Codex resuelto para un ámbito:
// con root vacío, el global.
func renderCodexOrchestrator(t *testing.T, root string) string {
	t.Helper()
	adapter := codex.NewAdapter()
	prompt, err := renderSessionPreflightPrompt(adapter, InjectOptions{})
	if err != nil {
		t.Fatalf("renderSessionPreflightPrompt() error = %v", err)
	}
	if root == "" {
		return resolveSkillsDirPlaceholders(prompt, adapter, "")
	}
	bound := adapter.WithWorkspace(root)
	return adaptCodexOrchestratorForWorkspace(resolveSkillsDirPlaceholders(prompt, bound, root), bound, root)
}

// Con el ámbito global el orquestador de Codex conserva sus rutas en ~/.codex y la
// descripción de los ficheros de perfil.
func TestCodexOrchestratorGlobalKeepsHomePaths(t *testing.T) {
	prompt := renderCodexOrchestrator(t, "")

	for _, want := range []string{
		"`~/.codex/config.toml`",
		"`~/.codex/skills/_shared/skill-resolver.md`",
		"`~/.codex/skills/_shared/` (global) or `.agent/skills/_shared/` (workspace)",
		"writes three SDD model-selection profile files into `~/.codex/`",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("el orquestador global debe seguir conteniendo %q", want)
		}
	}
}

// Con el ámbito workspace el orquestador apunta a la configuración y a las skills del
// proyecto, y no promete ficheros de perfil que no se escriben.
func TestCodexOrchestratorWorkspaceUsesProjectPaths(t *testing.T) {
	prompt := renderCodexOrchestrator(t, t.TempDir())

	if strings.Contains(prompt, "~/.codex") {
		idx := strings.Index(prompt, "~/.codex")
		t.Errorf("el orquestador de proyecto no debe referirse a ~/.codex: ...%s...", prompt[max(0, idx-60):min(len(prompt), idx+80)])
	}
	for _, unwanted := range []string{"{{AXIOM_SKILLS_DIR", "{{CODEX_", "<tier>.config.toml", "writes three SDD model-selection profile files", "(workspace)"} {
		if strings.Contains(prompt, unwanted) {
			t.Errorf("el orquestador de proyecto contiene %q", unwanted)
		}
	}
	for _, want := range []string{
		"`.codex/config.toml`",
		"`.agents/skills/_shared/skill-resolver.md`",
		"`.agents/skills/_shared/`",
		"does not write model-selection profile files",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("el orquestador de proyecto debe contener %q", want)
		}
	}
}

// La reescritura solo afecta a Codex en un proyecto.
func TestAdaptCodexOrchestratorForWorkspaceOnlyTouchesCodexInProject(t *testing.T) {
	root := t.TempDir()
	content := "usa `~/.codex/config.toml` y ~/.codex/skills/x"

	if got := adaptCodexOrchestratorForWorkspace(content, codex.NewAdapter().WithWorkspace(root), ""); got != content {
		t.Errorf("sin raíz de proyecto no cambia nada: %q", got)
	}
	if got := adaptCodexOrchestratorForWorkspace(content, claude.NewAdapter(), root); got != content {
		t.Errorf("otro agente no cambia: %q", got)
	}
	if got := adaptCodexOrchestratorForWorkspace(content, codex.NewAdapter().WithWorkspace(root), root); got != "usa `.codex/config.toml` y .agents/skills/x" {
		t.Errorf("got %q", got)
	}
}
