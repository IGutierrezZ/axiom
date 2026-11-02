package codex

import (
	"path/filepath"
	"testing"
)

// Ligado a la raíz de un proyecto, las instrucciones van a un fichero propio de
// .codex (fuente de developer_instructions), las skills a .agents/skills y el
// MCP, los hooks y la configuración siguen en .codex.
func TestWithWorkspaceUsesProjectLayout(t *testing.T) {
	root := t.TempDir()
	a := NewAdapter().WithWorkspace(root)

	for name, tc := range map[string]struct{ got, want string }{
		"SystemPromptFile": {a.SystemPromptFile(root), filepath.Join(root, ".codex", "axiom-instructions.md")},
		"SkillsDir":        {a.SkillsDir(root), filepath.Join(root, ".agents", "skills")},
		"MCPConfigPath":    {a.MCPConfigPath(root, "engram"), filepath.Join(root, ".codex", "config.toml")},
		"GlobalConfigDir":  {a.GlobalConfigDir(root), filepath.Join(root, ".codex")},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, quiero %q", name, tc.got, tc.want)
		}
	}
	if got := a.SystemPromptFile(root); filepath.Base(got) == "AGENTS.md" {
		t.Errorf("las instrucciones no pueden ir a un AGENTS.md: %s", got)
	}
	for _, p := range a.LegacySystemPromptFiles(root) {
		if p != filepath.Join(root, ".codex", "AGENTS.md") {
			t.Errorf("LegacySystemPromptFiles = %q", p)
		}
	}
}

// Cualquier otra carpeta, la de usuario por ejemplo, conserva las rutas globales y
// no ofrece ni registro de instrucciones ni ficheros antiguos que retirar.
func TestWithWorkspaceKeepsGlobalLayoutElsewhere(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	a := NewAdapter().WithWorkspace(root)

	if got, want := a.SystemPromptFile(home), filepath.Join(home, ".codex", "AGENTS.md"); got != want {
		t.Errorf("SystemPromptFile(home) = %q, quiero %q", got, want)
	}
	if got, want := a.SkillsDir(home), filepath.Join(home, ".codex", "skills"); got != want {
		t.Errorf("SkillsDir(home) = %q, quiero %q", got, want)
	}
	if got, want := a.SkillsDir(""), filepath.Join(".codex", "skills"); got != want {
		t.Errorf("SkillsDir(\"\") = %q, quiero %q (la ruta global relativa)", got, want)
	}
	if files := a.LegacySystemPromptFiles(home); len(files) != 0 {
		t.Errorf("LegacySystemPromptFiles(home) = %v, quiero ninguno", files)
	}
	if _, _, ok := a.WorkspaceInlineInstructions(home); ok {
		t.Errorf("WorkspaceInlineInstructions(home) no debe registrar nada fuera del proyecto")
	}
	if got, want := NewAdapter().SystemPromptFile(root), filepath.Join(root, ".codex", "AGENTS.md"); got != want {
		t.Errorf("el adaptador sin ligar cambió: %q, quiero %q", got, want)
	}
}

// El adaptador global no ofrece las capacidades de proyecto: así la instalación
// global no añade pasos ni escribe developer_instructions.
func TestGlobalAdapterHasNoWorkspaceCapabilities(t *testing.T) {
	var adapter any = NewAdapter()
	if _, ok := adapter.(interface {
		WorkspaceInlineInstructions(string) (string, string, bool)
	}); ok {
		t.Errorf("el adaptador global no debe registrar developer_instructions")
	}
	if _, ok := adapter.(interface{ LegacySystemPromptFiles(string) []string }); ok {
		t.Errorf("el adaptador global no debe retirar ficheros antiguos")
	}
}

func TestWorkspaceInlineInstructions(t *testing.T) {
	root := t.TempDir()
	configPath, key, ok := NewAdapter().WithWorkspace(root).WorkspaceInlineInstructions(root)
	if !ok {
		t.Fatal("WorkspaceInlineInstructions(root) debe registrar el texto en el proyecto")
	}
	if configPath != filepath.Join(root, ".codex", "config.toml") {
		t.Errorf("configPath = %q", configPath)
	}
	if key != "developer_instructions" {
		t.Errorf("key = %q, quiero developer_instructions", key)
	}
}

// Las capacidades que promueve el tipo embebido siguen disponibles en el ligado:
// la tabla de esfuerzos por fase la usa SDD.
func TestWorkspaceAdapterKeepsEmbeddedCapabilities(t *testing.T) {
	a := NewAdapter().WithWorkspace(t.TempDir())
	if !a.SupportsMCP() || !a.SupportsSkills() || !a.SupportsSystemPrompt() {
		t.Errorf("el adaptador ligado debe conservar las capacidades de Codex")
	}
	if a.Agent() != NewAdapter().Agent() {
		t.Errorf("Agent() cambió con el ligado")
	}
}
