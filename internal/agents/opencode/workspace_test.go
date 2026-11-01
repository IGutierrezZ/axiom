package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

// Ligado a la raíz de un proyecto, el adaptador resuelve la distribución que
// OpenCode lee en el proyecto: opencode.json en la raíz y .opencode/ para el resto.
func TestWithWorkspaceUsesProjectLayout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	root := t.TempDir()
	a := NewAdapter().WithWorkspace(root)

	for name, tc := range map[string]struct{ got, want string }{
		"GlobalConfigDir":  {a.GlobalConfigDir(root), filepath.Join(root, ".opencode")},
		"SystemPromptDir":  {a.SystemPromptDir(root), filepath.Join(root, ".opencode")},
		"SystemPromptFile": {a.SystemPromptFile(root), filepath.Join(root, ".opencode", "axiom.md")},
		"SkillsDir":        {a.SkillsDir(root), filepath.Join(root, ".agents", "skills")},
		"CommandsDir":      {a.CommandsDir(root), filepath.Join(root, ".opencode", "commands")},
		"SettingsPath":     {a.SettingsPath(root), filepath.Join(root, "opencode.json")},
		"MCPConfigPath":    {a.MCPConfigPath(root, "engram"), filepath.Join(root, "opencode.json")},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, quiero %q", name, tc.got, tc.want)
		}
	}
}

// Cualquier otra carpeta, la de usuario por ejemplo, conserva las rutas globales,
// y un adaptador sin ligar no cambia.
func TestWithWorkspaceKeepsGlobalLayoutElsewhere(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	root, home := t.TempDir(), t.TempDir()
	bound := NewAdapter().WithWorkspace(root)
	plain := NewAdapter()

	if got, want := bound.SkillsDir(home), plain.SkillsDir(home); got != want {
		t.Errorf("SkillsDir(home) = %q, quiero la global %q", got, want)
	}
	if got, want := bound.SystemPromptFile(home), filepath.Join(home, ".config", "opencode", "AGENTS.md"); got != want {
		t.Errorf("SystemPromptFile(home) = %q, quiero %q", got, want)
	}
	if got, want := bound.SettingsPath(home), plain.SettingsPath(home); got != want {
		t.Errorf("SettingsPath(home) = %q, quiero %q", got, want)
	}
	if got, want := plain.SkillsDir(root), filepath.Join(root, ".config", "opencode", "skills"); got != want {
		t.Errorf("el adaptador sin ligar cambió: %q, quiero %q", got, want)
	}
	if _, ok := plain.WorkspacePromptsDir(root); ok {
		t.Errorf("un adaptador sin ligar no tiene carpeta de prompts de proyecto")
	}
	if _, _, ok := bound.WorkspaceInstructions(home); ok {
		t.Errorf("fuera del proyecto no hay registro de instrucciones")
	}
}

// Un opencode.jsonc existente manda sobre opencode.json, como en la configuración global.
func TestWithWorkspacePrefersExistingJSONC(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "opencode.jsonc"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewAdapter().WithWorkspace(root)

	if got, want := a.SettingsPath(root), filepath.Join(root, "opencode.jsonc"); got != want {
		t.Errorf("SettingsPath = %q, quiero %q", got, want)
	}
}

func TestWithWorkspaceInstructionsAndPrompts(t *testing.T) {
	root := t.TempDir()
	a := NewAdapter().WithWorkspace(root)

	settings, entry, ok := a.WorkspaceInstructions(root)
	if !ok || settings != filepath.Join(root, "opencode.json") || entry != ".opencode/axiom.md" {
		t.Errorf("WorkspaceInstructions = %q, %q, %v", settings, entry, ok)
	}
	if dir, ok := a.WorkspacePromptsDir(root); !ok || dir != filepath.Join(root, ".opencode", "prompts", "sdd") {
		t.Errorf("WorkspacePromptsDir = %q, %v", dir, ok)
	}
}
