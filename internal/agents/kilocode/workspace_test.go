package kilocode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithWorkspaceUsesProjectLayout(t *testing.T) {
	root := t.TempDir()
	a := NewAdapter().WithWorkspace(root)

	for name, tc := range map[string]struct{ got, want string }{
		"GlobalConfigDir":  {a.GlobalConfigDir(root), filepath.Join(root, ".kilo")},
		"SystemPromptDir":  {a.SystemPromptDir(root), filepath.Join(root, ".kilo", "rules")},
		"SystemPromptFile": {a.SystemPromptFile(root), filepath.Join(root, ".kilo", "rules", "axiom.md")},
		"SkillsDir":        {a.SkillsDir(root), filepath.Join(root, ".kilo", "skills")},
		"CommandsDir":      {a.CommandsDir(root), filepath.Join(root, ".kilo", "commands")},
		"SettingsPath":     {a.SettingsPath(root), filepath.Join(root, ".kilo", "kilo.json")},
		"MCPConfigPath":    {a.MCPConfigPath(root, "engram"), filepath.Join(root, ".kilo", "kilo.json")},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, quiero %q", name, tc.got, tc.want)
		}
	}
}

func TestWithWorkspaceKeepsGlobalLayoutElsewhere(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	bound := NewAdapter().WithWorkspace(root)

	if got, want := bound.SkillsDir(home), filepath.Join(home, ".config", "kilo", "skills"); got != want {
		t.Errorf("SkillsDir(home) = %q, quiero %q", got, want)
	}
	if got, want := bound.SettingsPath(home), filepath.Join(home, ".config", "kilo", "opencode.json"); got != want {
		t.Errorf("SettingsPath(home) = %q, quiero %q", got, want)
	}
	if got, want := bound.SystemPromptFile(home), filepath.Join(home, ".config", "kilo", "AGENTS.md"); got != want {
		t.Errorf("SystemPromptFile(home) = %q, quiero %q", got, want)
	}
	if got, want := NewAdapter().SkillsDir(root), filepath.Join(root, ".config", "kilo", "skills"); got != want {
		t.Errorf("el adaptador sin ligar cambió: %q", got)
	}
}

// El fichero de configuración que ya existe se respeta en el orden de precedencia
// de Kilo, para no crear un segundo fichero junto al del usuario.
func TestWithWorkspaceReusesExistingProjectConfig(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files []string
		want  string
	}{
		{"ninguno", nil, filepath.Join(".kilo", "kilo.json")},
		{"kilo.json en .kilo", []string{filepath.Join(".kilo", "kilo.json")}, filepath.Join(".kilo", "kilo.json")},
		{"kilo.jsonc en la raíz", []string{"kilo.jsonc"}, "kilo.jsonc"},
		{"gana .kilo", []string{"kilo.jsonc", filepath.Join(".kilo", "kilo.json")}, filepath.Join(".kilo", "kilo.json")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, rel := range tc.files {
				path := filepath.Join(root, rel)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			a := NewAdapter().WithWorkspace(root)
			if got, want := a.SettingsPath(root), filepath.Join(root, tc.want); got != want {
				t.Errorf("SettingsPath = %q, quiero %q", got, want)
			}
		})
	}
}

func TestWithWorkspaceInstructionsAndPrompts(t *testing.T) {
	root := t.TempDir()
	a := NewAdapter().WithWorkspace(root)

	settings, entry, ok := a.WorkspaceInstructions(root)
	if !ok || settings != filepath.Join(root, ".kilo", "kilo.json") || entry != ".kilo/rules/axiom.md" {
		t.Errorf("WorkspaceInstructions = %q, %q, %v", settings, entry, ok)
	}
	if dir, ok := a.WorkspacePromptsDir(root); !ok || dir != filepath.Join(root, ".kilo", "prompts", "sdd") {
		t.Errorf("WorkspacePromptsDir = %q, %v", dir, ok)
	}
}
