package copilotcli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/IGutierrezZ/axiom/v3/internal/copilotconfig"
	"github.com/IGutierrezZ/axiom/v3/internal/model"
	"github.com/IGutierrezZ/axiom/v3/internal/system"
)

// useRealHome hace que os.UserHomeDir devuelva home: solo entonces se respeta
// COPILOT_HOME.
func useRealHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func TestAdapterIdentity(t *testing.T) {
	a := NewAdapter()
	if a.Agent() != model.AgentGitHubCopilotCLI || a.Tier() != model.TierFull {
		t.Fatalf("identity = %s/%s", a.Agent(), a.Tier())
	}
	if !a.SupportsSkills() || !a.SupportsSystemPrompt() || !a.SupportsMCP() {
		t.Fatal("Copilot CLI debe declarar skills, prompt de sistema y MCP")
	}
	if a.SupportsOutputStyles() || a.SupportsSlashCommands() || a.SupportsSubAgents() {
		t.Fatal("Copilot CLI no recibe estilos, comandos ni agentes en su carpeta de usuario: los agentes por fase viven solo en el proyecto")
	}
	if a.SystemPromptStrategy() != model.StrategyFileReplace || a.MCPStrategy() != model.StrategyMCPConfigFile {
		t.Fatalf("strategies = %v/%v", a.SystemPromptStrategy(), a.MCPStrategy())
	}
}

func TestAdapterPathsWithoutCopilotHome(t *testing.T) {
	home := t.TempDir()
	useRealHome(t, home)
	t.Setenv(copilotconfig.EnvVar, "")
	a := NewAdapter()
	root := filepath.Join(home, ".copilot")

	if a.GlobalConfigDir(home) != root || a.SystemPromptDir(home) != root {
		t.Errorf("GlobalConfigDir/SystemPromptDir = %q/%q, want %q", a.GlobalConfigDir(home), a.SystemPromptDir(home), root)
	}
	if got, want := a.SystemPromptFile(home), filepath.Join(root, "copilot-instructions.md"); got != want {
		t.Errorf("SystemPromptFile = %q, want %q", got, want)
	}
	if got, want := a.SkillsDir(home), filepath.Join(root, "skills"); got != want {
		t.Errorf("SkillsDir = %q, want %q", got, want)
	}
	if got := a.SubAgentsDir(home); got != "" {
		t.Errorf("SubAgentsDir = %q, want vacío", got)
	}
	if got, want := a.SettingsPath(home), filepath.Join(root, "settings.json"); got != want {
		t.Errorf("SettingsPath = %q, want %q", got, want)
	}
	if got, want := a.MCPConfigPath(home, "engram"), filepath.Join(root, "mcp-config.json"); got != want {
		t.Errorf("MCPConfigPath = %q, want %q", got, want)
	}
}

func TestAdapterPathsFollowCopilotHome(t *testing.T) {
	home := t.TempDir()
	useRealHome(t, home)
	custom := filepath.Join(t.TempDir(), "custom-copilot")
	t.Setenv(copilotconfig.EnvVar, custom)
	a := NewAdapter()

	if a.GlobalConfigDir(home) != custom ||
		a.SystemPromptFile(home) != filepath.Join(custom, "copilot-instructions.md") ||
		a.SkillsDir(home) != filepath.Join(custom, "skills") ||
		a.SettingsPath(home) != filepath.Join(custom, "settings.json") ||
		a.MCPConfigPath(home, "") != filepath.Join(custom, "mcp-config.json") {
		t.Fatalf("las rutas no siguen COPILOT_HOME=%s", custom)
	}
}

func TestAdapterSandboxedHomeIgnoresCopilotHome(t *testing.T) {
	useRealHome(t, t.TempDir())
	t.Setenv(copilotconfig.EnvVar, filepath.Join(t.TempDir(), "elsewhere"))
	sandbox := t.TempDir()

	if got, want := NewAdapter().SkillsDir(sandbox), filepath.Join(sandbox, ".copilot", "skills"); got != want {
		t.Fatalf("SkillsDir = %q, want %q: una raíz aislada no puede escapar", got, want)
	}
}

func TestAdapterInstallCommand(t *testing.T) {
	a := NewAdapter()
	npm := []string{"npm", "install", "-g", "--ignore-scripts", "@github/copilot@latest"}
	tests := []struct {
		name    string
		profile system.PlatformProfile
		want    []string
	}{
		{"linux con npm del sistema", system.PlatformProfile{OS: "linux", NpmWritable: false}, append([]string{"sudo"}, npm...)},
		{"linux con npm del usuario", system.PlatformProfile{OS: "linux", NpmWritable: true}, npm},
		{"macos", system.PlatformProfile{OS: "darwin"}, npm},
		{"windows", system.PlatformProfile{OS: "windows"}, npm},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := a.InstallCommand(tt.profile)
			if err != nil || len(got) != 1 || len(got[0]) != len(tt.want) {
				t.Fatalf("InstallCommand = %v, %v", got, err)
			}
			for i := range tt.want {
				if got[0][i] != tt.want[i] {
					t.Fatalf("InstallCommand = %v, want %v", got[0], tt.want)
				}
			}
		})
	}
}

func TestAdapterDetect(t *testing.T) {
	boom := errors.New("boom")
	found := func(string) (string, error) { return "/bin/copilot", nil }
	missing := func(string) (string, error) { return "", exec.ErrNotFound }
	withState := func(string) bool { return true }
	withoutState := func(string) bool { return false }

	tests := []struct {
		name      string
		lookPath  func(string) (string, error)
		stat      statResult
		hasState  func(string) bool
		wantIns   bool
		wantBin   string
		wantFound bool
		wantErr   error
	}{
		{"binario y config propia", found, statResult{isDir: true}, withState, true, "/bin/copilot", true, nil},
		{"solo binario", found, statResult{err: os.ErrNotExist}, withState, true, "/bin/copilot", false, nil},
		{"solo config propia", missing, statResult{isDir: true}, withState, false, "", true, nil},
		{"carpeta con solo skills de VS Code", missing, statResult{isDir: true}, withoutState, false, "", false, nil},
		{"nada", missing, statResult{err: os.ErrNotExist}, withState, false, "", false, nil},
		{"error de stat", found, statResult{err: boom}, withState, false, "", false, boom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			useRealHome(t, home)
			t.Setenv(copilotconfig.EnvVar, "")
			wantCfg := filepath.Join(home, ".copilot")
			a := &Adapter{
				lookPath: tt.lookPath,
				statPath: func(path string) statResult {
					if path != wantCfg {
						t.Fatalf("stat path = %q, want %q", path, wantCfg)
					}
					return tt.stat
				},
				hasState: tt.hasState,
			}
			ins, bin, cfg, ok, err := a.Detect(context.Background(), home)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if ins != tt.wantIns || bin != tt.wantBin || cfg != wantCfg || ok != tt.wantFound {
				t.Fatalf("Detect = (%v, %q, %q, %v), want (%v, %q, %q, %v)", ins, bin, cfg, ok, tt.wantIns, tt.wantBin, wantCfg, tt.wantFound)
			}
		})
	}
}

func TestAdapterDetectUsesCopilotHome(t *testing.T) {
	home := t.TempDir()
	useRealHome(t, home)
	custom := filepath.Join(t.TempDir(), "custom-copilot")
	t.Setenv(copilotconfig.EnvVar, custom)
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(custom, "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := NewAdapter()
	a.lookPath = func(string) (string, error) { return "", exec.ErrNotFound }

	_, _, cfg, ok, err := a.Detect(context.Background(), home)
	if err != nil || cfg != custom || !ok {
		t.Fatalf("Detect = (%q, %v, %v), want (%q, true, nil)", cfg, ok, err, custom)
	}
}

// Caso real: ~/.copilot con solo las skills de VS Code Copilot no es Copilot CLI.
func TestAdapterDetectIgnoresVSCodeOnlySkillsFolder(t *testing.T) {
	home := t.TempDir()
	useRealHome(t, home)
	t.Setenv(copilotconfig.EnvVar, "")
	if err := os.MkdirAll(filepath.Join(home, ".copilot", "skills", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := NewAdapter()
	a.lookPath = func(string) (string, error) { return "", exec.ErrNotFound }

	installed, _, _, ok, err := a.Detect(context.Background(), home)
	if err != nil || installed || ok {
		t.Fatalf("Detect = (installed=%v, found=%v, err=%v), want all false", installed, ok, err)
	}
}
