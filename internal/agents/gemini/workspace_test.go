package gemini

import (
	"path/filepath"
	"slices"
	"testing"
)

// Ligado a la raíz de un proyecto, las instrucciones van a un fichero propio de
// .gemini; el resto de rutas (settings.json, skills) coinciden con las globales
// relativas a esa raíz.
func TestWithWorkspaceUsesOwnInstructionsFile(t *testing.T) {
	root := t.TempDir()
	a := NewAdapter().WithWorkspace(root)

	for name, tc := range map[string]struct{ got, want string }{
		"SystemPromptFile": {a.SystemPromptFile(root), filepath.Join(root, ".gemini", "AXIOM.md")},
		"SettingsPath":     {a.SettingsPath(root), filepath.Join(root, ".gemini", "settings.json")},
		"MCPConfigPath":    {a.MCPConfigPath(root, "engram"), filepath.Join(root, ".gemini", "settings.json")},
		"SkillsDir":        {a.SkillsDir(root), filepath.Join(root, ".agents", "skills")},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %q, quiero %q", name, tc.got, tc.want)
		}
	}
	for _, p := range a.LegacySystemPromptFiles(root) {
		if p != filepath.Join(root, ".gemini", "GEMINI.md") {
			t.Errorf("LegacySystemPromptFiles = %q", p)
		}
	}
}

// Cualquier otra carpeta, la de usuario por ejemplo, conserva las rutas globales y
// no ofrece ni registro de contexto ni ficheros antiguos que retirar.
func TestWithWorkspaceKeepsGlobalLayoutElsewhere(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	a := NewAdapter().WithWorkspace(root)

	if got, want := a.SystemPromptFile(home), filepath.Join(home, ".gemini", "GEMINI.md"); got != want {
		t.Errorf("SystemPromptFile(home) = %q, quiero %q", got, want)
	}
	if files := a.LegacySystemPromptFiles(home); len(files) != 0 {
		t.Errorf("LegacySystemPromptFiles(home) = %v, quiero ninguno", files)
	}
	if _, _, _, ok := a.WorkspaceContextFile(home); ok {
		t.Errorf("WorkspaceContextFile(home) no debe registrar nada fuera del proyecto")
	}
	if got, want := NewAdapter().SystemPromptFile(root), filepath.Join(root, ".gemini", "GEMINI.md"); got != want {
		t.Errorf("el adaptador sin ligar cambió: %q, quiero %q", got, want)
	}
}

// El adaptador global no ofrece las capacidades de proyecto: así la instalación
// global no añade pasos ni escribe context.fileName.
func TestGlobalAdapterHasNoWorkspaceCapabilities(t *testing.T) {
	var adapter any = NewAdapter()
	if _, ok := adapter.(interface {
		WorkspaceContextFile(string) (string, string, []string, bool)
	}); ok {
		t.Errorf("el adaptador global no debe registrar context.fileName")
	}
	if _, ok := adapter.(interface{ LegacySystemPromptFiles(string) []string }); ok {
		t.Errorf("el adaptador global no debe retirar ficheros antiguos")
	}
}

func TestWorkspaceContextFile(t *testing.T) {
	root := t.TempDir()
	settingsPath, entry, defaults, ok := NewAdapter().WithWorkspace(root).WorkspaceContextFile(root)
	if !ok {
		t.Fatal("WorkspaceContextFile(root) debe registrar el fichero en el proyecto")
	}
	if settingsPath != filepath.Join(root, ".gemini", "settings.json") {
		t.Errorf("settingsPath = %q", settingsPath)
	}
	if entry != ".gemini/AXIOM.md" {
		t.Errorf("entry = %q, quiero .gemini/AXIOM.md (relativo, con barras normales)", entry)
	}
	if !slices.Equal(defaults, []string{"GEMINI.md"}) {
		t.Errorf("defaults = %v, quiero [GEMINI.md]", defaults)
	}
}
