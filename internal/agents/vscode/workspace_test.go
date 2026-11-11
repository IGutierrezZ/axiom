package vscode

import (
	"path/filepath"
	"testing"
)

// Ligado a la raíz de un proyecto, las instrucciones, las skills y el MCP van a las
// rutas de proyecto que VS Code lee.
func TestWithWorkspaceUsesProjectLayout(t *testing.T) {
	root := t.TempDir()
	a := NewAdapter().WithWorkspace(root)

	for name, tc := range map[string]struct{ got, want string }{
		"SystemPromptDir":  {a.SystemPromptDir(root), filepath.Join(root, ".github", "instructions")},
		"SystemPromptFile": {a.SystemPromptFile(root), filepath.Join(root, ".github", "instructions", "axiom.instructions.md")},
		"SkillsDir":        {a.SkillsDir(root), filepath.Join(root, ".agents", "skills")},
		"SubAgentsDir":     {a.SubAgentsDir(root), filepath.Join(root, ".github", "agents")},
		"MCPConfigPath":    {a.MCPConfigPath(root, "engram"), filepath.Join(root, ".vscode", "mcp.json")},
		"SettingsPath":     {a.SettingsPath(root), filepath.Join(root, ".vscode", "settings.json")},
		"GlobalConfigDir":  {a.GlobalConfigDir(root), filepath.Join(root, ".github")},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, quiero %q", name, tc.got, tc.want)
		}
	}
	if files := a.LegacySystemPromptFiles(root); len(files) != 0 {
		t.Errorf("en un proyecto no hay nombres antiguos que retirar: %v", files)
	}
	if got := a.WorkspaceUserConfigFiles(root); len(got) != 1 || got[0] != filepath.Join(root, ".vscode", "mcp.json") {
		t.Errorf("WorkspaceUserConfigFiles(root) = %v, quiero .vscode/mcp.json", got)
	}
	if !a.InWorkspace(root) {
		t.Errorf("InWorkspace(root) debe ser verdadero")
	}
}

// Cualquier otra carpeta, la de usuario por ejemplo, conserva las rutas globales y no
// ofrece ficheros de proyecto que preparar.
func TestWithWorkspaceKeepsGlobalLayoutElsewhere(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	global := NewAdapter()
	a := global.WithWorkspace(root)

	for name, tc := range map[string]struct{ got, want string }{
		"SystemPromptFile": {a.SystemPromptFile(home), global.SystemPromptFile(home)},
		"SkillsDir":        {a.SkillsDir(home), filepath.Join(home, ".copilot", "skills")},
		"SubAgentsDir":     {a.SubAgentsDir(home), ""},
		"MCPConfigPath":    {a.MCPConfigPath(home, "engram"), global.MCPConfigPath(home, "engram")},
		"SettingsPath":     {a.SettingsPath(home), global.SettingsPath(home)},
		"GlobalConfigDir":  {a.GlobalConfigDir(home), filepath.Join(home, ".copilot")},
	} {
		if tc.got != tc.want {
			t.Errorf("%s(home) = %q, quiero %q", name, tc.got, tc.want)
		}
	}
	if got, want := a.LegacySystemPromptFiles(home), global.LegacySystemPromptFiles(home); len(got) != len(want) || len(got) == 0 {
		t.Errorf("LegacySystemPromptFiles(home) = %v, quiero %v", got, want)
	}
	if files := a.WorkspaceUserConfigFiles(home); len(files) != 0 {
		t.Errorf("WorkspaceUserConfigFiles(home) = %v, quiero ninguno", files)
	}
	if a.InWorkspace(home) {
		t.Errorf("InWorkspace(home) debe ser falso")
	}
}

// El adaptador global no ofrece las capacidades de proyecto, y el ligado sí.
func TestGlobalAdapterHasNoWorkspaceCapabilities(t *testing.T) {
	var global any = NewAdapter()
	if _, ok := global.(interface{ WorkspaceUserConfigFiles(string) []string }); ok {
		t.Errorf("el adaptador global no debe ofrecer ficheros de proyecto")
	}
	if _, ok := global.(interface{ InWorkspace(string) bool }); ok {
		t.Errorf("el adaptador global no está ligado a ningún proyecto")
	}
	var bound any = NewAdapter().WithWorkspace(t.TempDir())
	if _, ok := bound.(interface{ WorkspaceUserConfigFiles(string) []string }); !ok {
		t.Errorf("el adaptador ligado debe ofrecer ficheros de proyecto")
	}
}
